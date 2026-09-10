package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

// TestReserveAttemptCheckRun_AdmitsExactlyOneInvoker is what makes a
// deterministic check safe to own: the reservation, not a lock inside one
// process, is what stops two reconcilers launching the same command.
func TestReserveAttemptCheckRun_AdmitsExactlyOneInvoker(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "checkrunreserve")
	attempt, err := s.CreateAttemptWithFence(ctx, admissionFor(outcomeID, plan, "rk-checkrun", domain.FenceSubjectForProject("checkrunreserve")))
	if err != nil {
		t.Fatalf("admit attempt: %v", err)
	}
	run := ports.AttemptCheckRun{
		ID: "chkrun-1", AttemptID: attempt.ID, CheckID: "chk-1",
		ArtifactVersion: "artifact-v1", ReservationEpoch: "daemon-epoch-1", ReservedAt: time.Unix(100, 0).UTC(),
	}

	if err := s.ReserveAttemptCheckRun(ctx, run); err != nil {
		t.Fatalf("first reservation: %v", err)
	}
	second := run
	second.ID = "chkrun-2"
	if err := s.ReserveAttemptCheckRun(ctx, second); !errors.Is(err, ports.ErrCheckRunAlreadyReserved) {
		t.Fatalf("second reservation = %v, want ErrCheckRunAlreadyReserved", err)
	}

	stored, found, err := s.GetAttemptCheckRun(ctx, attempt.ID, "chk-1", "artifact-v1")
	if err != nil || !found {
		t.Fatalf("read run: found=%v err=%v", found, err)
	}
	if stored.ID != "chkrun-1" || stored.State != ports.CheckRunReserved {
		t.Fatalf("stored run = %+v, want the first reservation still holding it", stored)
	}
	if stored.ReservationEpoch != run.ReservationEpoch {
		t.Fatalf("reservation epoch = %q, want %q", stored.ReservationEpoch, run.ReservationEpoch)
	}
	// A different artifact version is a different question and gets its own
	// run: re-checking new output is not a duplicate invocation.
	other := run
	other.ID, other.ArtifactVersion = "chkrun-3", "artifact-v2"
	if err := s.ReserveAttemptCheckRun(ctx, other); err != nil {
		t.Fatalf("reservation for a new artifact version was refused: %v", err)
	}
}

// TestRecordAttemptCheckObservation_IsWriteOnce protects what proof is
// rebuilt from. A changed observation would silently change what the recorded
// evidence said.
func TestRecordAttemptCheckObservation_IsWriteOnce(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "checkrunwriteonce")
	attempt, err := s.CreateAttemptWithFence(ctx, admissionFor(outcomeID, plan, "rk-checkrun", domain.FenceSubjectForProject("checkrunwriteonce")))
	if err != nil {
		t.Fatalf("admit attempt: %v", err)
	}
	run := ports.AttemptCheckRun{
		ID: "chkrun-1", AttemptID: attempt.ID, CheckID: "chk-1",
		ArtifactVersion: "artifact-v1", ReservedAt: time.Unix(100, 0).UTC(),
	}
	if err := s.ReserveAttemptCheckRun(ctx, run); err != nil {
		t.Fatalf("reserve: %v", err)
	}

	run.Observation = ports.AttemptCheckObservation{
		Ran: true, Passed: false, ExitCode: 1, EnforcedBy: "macos-seatbelt-workspace-write",
		Output: "FAIL\n", EndedAt: time.Unix(200, 0).UTC(),
	}
	if err := s.RecordAttemptCheckObservation(ctx, run); err != nil {
		t.Fatalf("record observation: %v", err)
	}

	// A second record must not move it, and must not error either: a retried
	// write after a crash is ordinary, and refusing it would strand the run.
	rewritten := run
	rewritten.Observation.Passed = true
	rewritten.Observation.ExitCode = 0
	rewritten.Observation.Output = "ok\n"
	if err := s.RecordAttemptCheckObservation(ctx, rewritten); err != nil {
		t.Fatalf("repeat record: %v", err)
	}

	stored, found, err := s.GetAttemptCheckRun(ctx, attempt.ID, "chk-1", "artifact-v1")
	if err != nil || !found {
		t.Fatalf("read run: found=%v err=%v", found, err)
	}
	if !stored.Complete() {
		t.Fatalf("state = %q, want observed", stored.State)
	}
	if stored.Observation.Passed || stored.Observation.ExitCode != 1 || stored.Observation.Output != "FAIL\n" {
		t.Fatalf("the recorded observation changed: %+v", stored.Observation)
	}
	if stored.Observation.EnforcedBy != "macos-seatbelt-workspace-write" {
		t.Fatalf("the enforcing mechanism was lost: %+v", stored.Observation)
	}
}

// TestMarkAttemptCheckRunUnknown_OnlyClosesAnIncompleteReservation keeps a
// completed observation from being downgraded to unknown by a late recovery
// pass.
func TestMarkAttemptCheckRunUnknown_OnlyClosesAnIncompleteReservation(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "checkrununknown")
	attempt, err := s.CreateAttemptWithFence(ctx, admissionFor(outcomeID, plan, "rk-checkrun", domain.FenceSubjectForProject("checkrununknown")))
	if err != nil {
		t.Fatalf("admit attempt: %v", err)
	}
	reserved := ports.AttemptCheckRun{
		ID: "chkrun-1", AttemptID: attempt.ID, CheckID: "chk-open",
		ArtifactVersion: "artifact-v1", ReservedAt: time.Unix(100, 0).UTC(),
	}
	observed := reserved
	observed.ID, observed.CheckID = "chkrun-2", "chk-done"
	for _, run := range []ports.AttemptCheckRun{reserved, observed} {
		if err := s.ReserveAttemptCheckRun(ctx, run); err != nil {
			t.Fatalf("reserve %s: %v", run.CheckID, err)
		}
	}
	observed.Observation = ports.AttemptCheckObservation{Ran: true, Passed: true, EndedAt: time.Unix(200, 0).UTC()}
	if err := s.RecordAttemptCheckObservation(ctx, observed); err != nil {
		t.Fatalf("record: %v", err)
	}

	at := time.Unix(300, 0).UTC()
	for _, checkID := range []domain.ApprovedCheckID{"chk-open", "chk-done"} {
		if err := s.MarkAttemptCheckRunUnknown(ctx, attempt.ID, checkID, "artifact-v1", at); err != nil {
			t.Fatalf("mark %s unknown: %v", checkID, err)
		}
	}

	open, _, err := s.GetAttemptCheckRun(ctx, attempt.ID, "chk-open", "artifact-v1")
	if err != nil {
		t.Fatalf("read open run: %v", err)
	}
	if open.State != ports.CheckRunUnknown {
		t.Fatalf("incomplete reservation state = %q, want unknown", open.State)
	}
	done, _, err := s.GetAttemptCheckRun(ctx, attempt.ID, "chk-done", "artifact-v1")
	if err != nil {
		t.Fatalf("read completed run: %v", err)
	}
	if !done.Complete() || !done.Observation.Passed {
		t.Fatalf("a completed observation was downgraded: %+v", done)
	}
}

package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite"
)

// seedApprovedPlan builds a full Decide & Authorize lineage for one project
// and returns its approved plan plus the outcome id.
func seedApprovedPlan(t *testing.T, s *sqlite.Store, projectID string) (domain.PlanRevision, domain.OutcomeID) {
	t.Helper()
	ctx := context.Background()
	seedProject(t, s, projectID)
	space, err := s.EnsureWorkResponsibilitySpace(ctx, domain.ProjectID(projectID))
	if err != nil {
		t.Fatalf("ensure space: %v", err)
	}
	outcome := domain.Outcome{
		ID:      domain.OutcomeID("out-" + projectID),
		SpaceID: space.ID,
		Title:   "Local Focus Ledger",
	}
	first := domain.ContractRevision{
		ID:              domain.ContractRevisionID("cr-" + projectID),
		OutcomeID:       outcome.ID,
		Number:          1,
		Goal:            "Record focus locally.",
		SuccessCriteria: []string{"Blocks are recorded."},
		Review:          "Deterministic checks.",
	}
	if err := s.CreateOutcomeWithContract(ctx, outcome, first, "rk-create-"+projectID); err != nil {
		t.Fatalf("create outcome: %v", err)
	}
	unit := domain.WorkUnit{
		ID:                      domain.WorkUnitID("wu-" + projectID),
		Kind:                    domain.WorkUnitDirect,
		Title:                   "Deliver Local Focus Ledger",
		ContractRevisionNumber:  1,
		OutputSummary:           "Working local feature in the isolated worktree.",
		EvidenceChecks:          []string{"checks pass"},
		VerificationRequirement: "Deterministic checks.",
		StopConditions:          []string{"stop before remote effects"},
	}
	grants := []domain.CapabilityGrant{
		{ID: domain.CapabilityGrantID("cg-read-" + projectID), Name: domain.CapabilityWorktreeRead, Scope: "worktree/*"},
		{ID: domain.CapabilityGrantID("cg-write-" + projectID), Name: domain.CapabilityWorktreeWrite, Scope: "worktree/*"},
		{ID: domain.CapabilityGrantID("cg-exec-" + projectID), Name: domain.CapabilityWorktreeExec, Scope: "worktree/*"},
	}
	digest, err := domain.ComputeRunBriefCoreDigest(first, unit, grants)
	if err != nil {
		t.Fatalf("compute digest: %v", err)
	}
	plan, err := s.AppendPlanRevision(ctx, outcome.ID, domain.PlanRevision{
		ID:                     domain.PlanRevisionID("plan-" + projectID),
		OutcomeID:              outcome.ID,
		ContractRevisionNumber: 1,
		Status:                 domain.PlanStatusProposed,
		Summary:                "One direct Work Unit",
		WorkUnits:              []domain.WorkUnit{unit},
		Grants:                 grants,
		RunBriefCoreDigest:     digest,
	})
	if err != nil {
		t.Fatalf("append plan: %v", err)
	}
	approved, found, err := s.ApprovePlanRevision(ctx, outcome.ID, plan.ID)
	if err != nil || !found {
		t.Fatalf("approve plan: found=%v err=%v", found, err)
	}
	return approved, outcome.ID
}

// TestAttemptStore_FencedAdmissionIsAtomicAndExclusive pins D3/D4 at the store
// layer: the winner gets a queued attempt plus the open fence; the loser gets
// a typed fence conflict with ZERO durable rows (fail-closed admission).
func TestAttemptStore_FencedAdmissionIsAtomicAndExclusive(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "mer")
	subject := domain.FenceSubjectForProject("mer")

	at, err := s.CreateAttemptWithFence(ctx, outcomeID, plan, "rk-att-1", subject, time.Now().UTC())
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	if at.Status != domain.AttemptQueued || at.Number != 1 {
		t.Fatalf("attempt = %+v, want queued #1", at)
	}

	held, ok, err := s.OpenFenceForSubject(ctx, subject)
	if err != nil || !ok {
		t.Fatalf("open fence ok=%v err=%v", ok, err)
	}
	if held.AttemptID != at.ID || !held.Open() {
		t.Fatalf("fence = %+v, want open custody by %s", held, at.ID)
	}

	before, err := s.ListAttempts(ctx, outcomeID)
	if err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	_, err = s.CreateAttemptWithFence(ctx, outcomeID, plan, "rk-att-2", subject, time.Now().UTC())
	var fenced *ports.AttemptFenceHeldError
	if !errors.As(err, &fenced) {
		t.Fatalf("second admission must fail with AttemptFenceHeldError, got %v", err)
	}
	if fenced.Holder != at.ID {
		t.Fatalf("conflict holder = %s, want %s", fenced.Holder, at.ID)
	}
	after, err := s.ListAttempts(ctx, outcomeID)
	if err != nil {
		t.Fatalf("re-list attempts: %v", err)
	}
	if len(before) != 1 || len(after) != 1 {
		t.Fatalf("failed admission must leave zero rows: before=%d after=%d", len(before), len(after))
	}

	// Replay through the request key resolves the ORIGINAL attempt without a
	// second write.
	replay, found, err := s.FindAttemptByIdempotencyKey(ctx, "rk-att-1")
	if err != nil || !found || replay.ID != at.ID {
		t.Fatalf("replay found=%v id=%s err=%v", found, replay.ID, err)
	}
	finalList, err := s.ListAttempts(ctx, outcomeID)
	if err != nil {
		t.Fatal(err)
	}
	if len(finalList) != 1 {
		t.Fatalf("replayed start must not stack attempts, got %d", len(finalList))
	}
}

// TestAttemptStore_GuardedTransitionsAndSessionRefs covers the trigger-backed
// lifecycle seam, FK-free session refs, and ordered observations.
func TestAttemptStore_GuardedTransitionsAndSessionRefs(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "mer")
	at, err := s.CreateAttemptWithFence(ctx, outcomeID, plan, "", domain.FenceSubjectForProject("mer"), time.Now().UTC())
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}

	// Wrong expectation mutates nothing.
	rows, err := s.TransitionAttemptStatus(ctx, outcomeID, at.ID, domain.AttemptRunning, domain.AttemptPaused, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Fatalf("stale-guard transition affected %d rows, want 0", rows)
	}

	now := time.Now().UTC()
	for _, hop := range []domain.AttemptStatus{domain.AttemptRunning, domain.AttemptPaused} {
		expected := domain.AttemptQueued
		if hop == domain.AttemptPaused {
			expected = domain.AttemptRunning
		}
		rows, err := s.TransitionAttemptStatus(ctx, outcomeID, at.ID, expected, hop, now)
		if err != nil || rows != 1 {
			t.Fatalf("transition to %s: rows=%d err=%v", hop, rows, err)
		}
	}

	// The database itself refuses an illegal transition even when the guard
	// matches: paused -> succeeded is not a legal edge.
	_, err = s.TransitionAttemptStatus(ctx, outcomeID, at.ID, domain.AttemptPaused, domain.AttemptSucceeded, now)
	if err == nil || !strings.Contains(err.Error(), "illegal attempt status transition") {
		t.Fatalf("paused -> succeeded must abort at the trigger, got %v", err)
	}

	ref, err := s.BindAttemptSession(ctx, domain.AttemptSessionRef{
		AttemptID:              at.ID,
		SessionID:              "provider-session-1",
		Harness:                domain.HarnessCodex,
		Mode:                   domain.SessionModeTUI,
		RunBriefCoreDigest:     strings.Repeat("ab", 32),
		RunBriefCompiledDigest: strings.Repeat("cd", 32),
		AdmissionSnapshot:      `{"snapshotVersion":1}`,
	})
	if err != nil {
		t.Fatalf("bind session: %v", err)
	}
	if ref.Seq != 1 {
		t.Fatalf("first ref seq = %d, want 1", ref.Seq)
	}
	latest, ok, err := s.LatestAttemptSessionRef(ctx, at.ID)
	if err != nil || !ok || latest.SessionID != "provider-session-1" {
		t.Fatalf("latest ref ok=%v err=%v", ok, err)
	}

	firstObs, err := s.AppendAttemptObservation(ctx, at.ID, domain.ObservationOwnerPause, `{"by":"owner"}`, now)
	if err != nil {
		t.Fatalf("append observation: %v", err)
	}
	secondObs, err := s.AppendAttemptObservation(ctx, at.ID, domain.ObservationProviderExit, `{}`, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if secondObs.Seq != firstObs.Seq+1 {
		t.Fatalf("observation seqs must be monotonic: %d then %d", firstObs.Seq, secondObs.Seq)
	}

	projectID, ok, err := s.GetOutcomeProjectID(ctx, outcomeID)
	if err != nil || !ok || projectID != "mer" {
		t.Fatalf("outcome project = (%q, %v) err=%v", projectID, ok, err)
	}
}

// TestAttemptStore_ReconcileReleasesCustodyForReplacement walks the recovery
// path: reconcile releases the old fence with a reason, records receipts, and
// only then may a replacement attempt acquire the subject.
func TestAttemptStore_ReconcileReleasesCustodyForReplacement(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "mer")
	subject := domain.FenceSubjectForProject("mer")
	at, err := s.CreateAttemptWithFence(ctx, outcomeID, plan, "rk-a1", subject, time.Now().UTC())
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}

	if _, err := s.ReleaseFenceForAttempt(ctx, at.ID, "", time.Now()); err == nil {
		t.Fatal("release without reason must be refused")
	}
	rows, err := s.ReleaseFenceForAttempt(ctx, at.ID, "replacement_attempt", time.Now())
	if err != nil || rows != 1 {
		t.Fatalf("release rows=%d err=%v", rows, err)
	}
	rows, err = s.ReleaseFenceForAttempt(ctx, at.ID, "again", time.Now())
	if err != nil || rows != 0 {
		t.Fatalf("second release rows=%d err=%v, want 0", rows, err)
	}

	if err := s.CreateRecoveryReceipt(ctx, domain.AttemptRecoveryReceipt{
		ID:         "rcpt-lost",
		AttemptID:  at.ID,
		Resolution: domain.RecoveryReplacement,
		Detail:     `{"evidence":"heartbeat missing"}`,
	}); err != nil {
		t.Fatalf("record receipt: %v", err)
	}
	receipts, err := s.ListRecoveryReceipts(ctx, at.ID)
	if err != nil || len(receipts) != 1 {
		t.Fatalf("receipts len=%d err=%v", len(receipts), err)
	}

	// Custody handover: the replacement is always a NEW attempt row.
	replacement, err := s.CreateAttemptWithFence(ctx, outcomeID, plan, "rk-a2", subject, time.Now().UTC())
	if err != nil {
		t.Fatalf("replacement admission: %v", err)
	}
	if replacement.Number != 2 || replacement.ID == at.ID {
		t.Fatalf("replacement = %+v, want new row #2", replacement)
	}
	observations, err := s.ListAttemptObservations(ctx, at.ID)
	if err != nil {
		t.Fatal(err)
	}
	_ = observations
	// Stale-attempt inertness (D5): observations remain INSERTABLE on the
	// terminal predecessor after replacement — inspectable history that never
	// touches current truth.
	if _, err := s.AppendAttemptObservation(ctx, at.ID, "late_stale_report", `{}`, time.Now()); err != nil {
		t.Fatalf("stale observation must stay insertable: %v", err)
	}
	all, err := s.ListAttempts(ctx, outcomeID)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("lineage must hold both attempts, got %d", len(all))
	}
}

// TestAttemptStore_FenceLeaseRenewal pins the renewable-lease facts: renewal
// refreshes only OPEN fences for the custodian, and a released fence freezes
// forever (trigger-refused).
func TestAttemptStore_FenceLeaseRenewal(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "mer")
	subject := domain.FenceSubjectForProject("mer")
	at, err := s.CreateAttemptWithFence(ctx, outcomeID, plan, "rk-lease", subject, time.Now().UTC())
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	t0 := time.Now().UTC().Add(-time.Hour)
	if rows, err := s.RenewFenceForAttempt(ctx, at.ID, t0); err != nil || rows != 1 {
		t.Fatalf("renew rows=%d err=%v", rows, err)
	}
	fence, ok, err := s.OpenFenceForSubject(ctx, subject)
	if err != nil || !ok {
		t.Fatalf("fence ok=%v err=%v", ok, err)
	}
	if !fence.LastRenewedAt.Equal(t0.Truncate(time.Second)) && fence.LastRenewedAt.Sub(t0).Abs() > time.Second {
		t.Fatalf("lastRenewedAt = %v, want ~%v", fence.LastRenewedAt, t0)
	}

	if _, err := s.ReleaseFenceForAttempt(ctx, at.ID, "replacement_attempt", time.Now()); err != nil {
		t.Fatal(err)
	}
	if rows, err := s.RenewFenceForAttempt(ctx, at.ID, time.Now()); err != nil || rows != 0 {
		t.Fatalf("post-release renewal rows=%d err=%v, want 0", rows, err)
	}
}

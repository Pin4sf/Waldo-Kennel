package outcome_test

import (
	"context"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/service/intelligence/intelligencetest"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

// checkedHarness is a classification harness whose WorkUnit carries one
// approved deterministic check, with the runner and durable run record wired.
type checkedHarness struct {
	*classificationHarness
	runner *countingCheckRunner
	runs   *checkRunFakeStore
	check  domain.ApprovedCheck
}

// reentrantAfterCommitCheckRunStore models a durable reservation store whose
// INSERT has committed but whose Reserve call has not returned to the caller.
// That is the publication window a process-local ownership marker must cover.
type reentrantAfterCommitCheckRunStore struct {
	*checkRunFakeStore
	reenter    func() error
	reentered  bool
	reentryErr error
}

func (s *reentrantAfterCommitCheckRunStore) ReserveAttemptCheckRun(ctx context.Context, run ports.AttemptCheckRun) error {
	err := s.checkRunFakeStore.ReserveAttemptCheckRun(ctx, run)
	if err == nil && !s.reentered {
		s.reentered = true
		s.reentryErr = s.reenter()
	}
	return err
}

func newCheckedHarness(t *testing.T, observe func(domain.ApprovedCheck) ports.AttemptCheckObservation) *checkedHarness {
	t.Helper()
	base := newClassificationHarness(t)
	runner := &countingCheckRunner{observe: observe}
	runs := newCheckRunFakeStore()

	check := domain.ApprovedCheck{
		ID: "chk-1", CriterionID: base.criterion(t), Argv: []string{"go", "test", "./..."}, TimeoutSeconds: 60,
	}
	base.store.attachApprovedCheck(t, base.outcomeID, check)

	// The service is rebuilt with checks wired; everything else about the
	// fixture — the ended Attempt and its retained receipt — is unchanged.
	svc := outcome.New(base.store, func() time.Time { return time.Unix(1_000, 0).UTC() }).
		WithPlanning(intelligencetest.New(), &routingInventoryFake{candidates: []domain.RoutingCandidate{executionCandidate(domain.HarnessCodex, "")}}).
		WithExecution(&fakeSpawner{readiness: ports.AgentProfileReadiness{Ready: true, Detail: "profile ok"}}, newFakeHeartbeats()).
		WithCheckRunner(runner, runs)
	base.svc = svc
	return &checkedHarness{classificationHarness: base, runner: runner, runs: runs, check: check}
}

func failingObservation(check domain.ApprovedCheck) ports.AttemptCheckObservation {
	return ports.AttemptCheckObservation{
		Check: check, Ran: true, Passed: false, ExitCode: 1,
		EnforcedBy: "test-enforcement", Output: "FAIL\n",
	}
}

// TestApprovedChecks_AFailingCheckIsNotRelaunchedOnEveryTick is R2's headline.
// A failing check leaves its Attempt reconciled, and reconciliation
// re-enumerates reconciled Attempts forever — so without durable run identity
// the command would relaunch on every tick.
func TestApprovedChecks_AFailingCheckIsNotRelaunchedOnEveryTick(t *testing.T) {
	h := newCheckedHarness(t, failingObservation)
	ctx := context.Background()

	for tick := 0; tick < 3; tick++ {
		if err := h.svc.ReconcileAttemptOutcomes(ctx); err != nil {
			t.Fatalf("tick %d: %v", tick, err)
		}
	}
	if got := h.runner.totalChecksInvoked(); got != 1 {
		t.Fatalf("the check command was invoked %d times across three ticks, want once", got)
	}
	// A failing check must also leave the Attempt unclassified: the point of
	// not re-running it is not to make it pass.
	if status := h.reload(t).Status; status == domain.AttemptSucceeded {
		t.Fatalf("status = %s, want the Attempt to stay unclassified after a failing check", status)
	}
}

// TestApprovedChecks_ARestartAfterEvidenceCompletesProofWithoutRerunning
// covers the crash window between the evidence write and its verification.
// The recorded observation is what both writes derive from, so finishing the
// proof must not need the command again.
func TestApprovedChecks_ARestartAfterEvidenceCompletesProofWithoutRerunning(t *testing.T) {
	// A failing check keeps the Attempt unclassified, which is what lets the
	// next tick revisit it — the same window a crash between the two proof
	// writes leaves behind.
	h := newCheckedHarness(t, failingObservation)
	ctx := context.Background()

	if err := h.svc.ReconcileAttemptOutcomes(ctx); err != nil {
		t.Fatalf("first reconcile: %v", err)
	}
	if got := h.runner.calls(); got != 1 {
		t.Fatalf("runner calls = %d, want 1", got)
	}
	// The evidence and the recorded observation survive; the verification
	// write did not happen.
	h.store.dropVerificationRuns()

	if err := h.svc.ReconcileAttemptOutcomes(ctx); err != nil {
		t.Fatalf("reconcile after restart: %v", err)
	}
	if got := h.runner.totalChecksInvoked(); got != 1 {
		t.Fatalf("the command was invoked %d times, want once: proof must be finished from the recorded observation", got)
	}
	runs, err := h.store.ListVerificationRuns(ctx, h.outcomeID)
	if err != nil {
		t.Fatalf("list verifications: %v", err)
	}
	if len(runs) == 0 {
		t.Fatal("the interrupted verification was never completed")
	}
}

// TestApprovedChecks_ConcurrentReconcilersLaunchTheCommandOnce proves the
// reservation is what serialises invocation, not a lock inside one process.
func TestApprovedChecks_ConcurrentReconcilersLaunchTheCommandOnce(t *testing.T) {
	h := newCheckedHarness(t, nil)
	ctx := context.Background()

	// Two reservations for the same run: the second caller is refused and
	// must not invoke the command.
	first := ports.AttemptCheckRun{
		ID: "chkrun-other", AttemptID: h.attempt.ID, CheckID: h.check.ID,
		ArtifactVersion: h.receipt.ArtifactVersion, ReservedAt: time.Unix(900, 0).UTC(),
	}
	if err := h.runs.ReserveAttemptCheckRun(ctx, first); err != nil {
		t.Fatalf("seed reservation: %v", err)
	}
	if err := h.svc.ReconcileAttemptOutcomes(ctx); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if got := h.runner.totalChecksInvoked(); got != 0 {
		t.Fatalf("a reserved check was invoked %d times by a second reconciler", got)
	}
}

// TestApprovedChecks_AnInterruptedRunIsUnknownAndNeverRetried is the recovery
// rule. A reservation that survives a restart means the command may already
// have run and had effects; re-running it is not safe, and calling it failed
// would blame the work for a crash.
func TestApprovedChecks_AnInterruptedRunIsUnknownAndNeverRetried(t *testing.T) {
	h := newCheckedHarness(t, failingObservation)
	ctx := context.Background()

	if err := h.svc.ReconcileAttemptOutcomes(ctx); err != nil {
		t.Fatalf("first reconcile: %v", err)
	}
	// Roll the record back to a bare reservation and drop the proof it
	// produced: the process died between reserving and recording.
	h.runs.forgetObservations()
	h.store.dropProof()

	if err := h.svc.ReconcileAttemptOutcomes(ctx); err != nil {
		t.Fatalf("reconcile after interruption: %v", err)
	}
	if got := h.runner.totalChecksInvoked(); got != 1 {
		t.Fatalf("an interrupted check was invoked %d times, want the original one only", got)
	}
	run, found, err := h.runs.GetAttemptCheckRun(ctx, h.attempt.ID, h.check.ID, h.receipt.ArtifactVersion)
	if err != nil || !found {
		t.Fatalf("read check run: found=%v err=%v", found, err)
	}
	if run.State != ports.CheckRunUnknown {
		t.Fatalf("interrupted run state = %q, want unknown", run.State)
	}
	// Unknown is not proof either way, so the Attempt must stay unclassified.
	if status := h.reload(t).Status; status == domain.AttemptSucceeded {
		t.Fatalf("status = %s, want unclassified while a check outcome is unknown", status)
	}
	verifications, err := h.store.ListVerificationRuns(ctx, h.outcomeID)
	if err != nil {
		t.Fatalf("list verifications: %v", err)
	}
	if len(verifications) != 1 || verifications[0].Result != domain.VerificationInconclusive {
		t.Fatalf("verifications = %#v, want one inconclusive result", verifications)
	}
}

// TestApprovedChecks_APassingCheckProvesTheCriterionOnce is the green half:
// the mechanism still produces proof, and repeating the tick neither reruns
// the command nor writes a second observation.
func TestApprovedChecks_APassingCheckProvesTheCriterionOnce(t *testing.T) {
	h := newCheckedHarness(t, nil)
	ctx := context.Background()

	if err := h.svc.ReconcileAttemptOutcomes(ctx); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if status := h.reload(t).Status; status != domain.AttemptSucceeded {
		t.Fatalf("status = %s, want succeeded once its check passed", status)
	}
	if err := h.svc.ReconcileAttemptOutcomes(ctx); err != nil {
		t.Fatalf("second reconcile: %v", err)
	}
	if got := h.runner.totalChecksInvoked(); got != 1 {
		t.Fatalf("the command was invoked %d times, want once", got)
	}
	evidence, err := h.store.ListEvidenceItems(ctx, h.outcomeID)
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	if len(evidence) != 1 {
		t.Fatalf("evidence items = %d, want exactly one observation recorded", len(evidence))
	}
}

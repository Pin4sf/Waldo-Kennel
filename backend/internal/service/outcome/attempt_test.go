package outcome_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	"github.com/aoagents/agent-orchestrator/backend/internal/service/outcome"
)

// newAttemptHarness builds a fully wired Act & Observe fixture: an in-memory
// store, an execution seam double, and a heartbeat table. The returned service
// has already created the Local Focus Ledger Outcome and gotten its v0 plan
// proposed AND approved.
func newAttemptHarness(t *testing.T) (*outcome.Service, *attemptFakeStore, *fakeSpawner, *fakeHeartbeats, domain.OutcomeID, domain.PlanRevisionID) {
	t.Helper()
	store := newAttemptFakeStore()
	spawner := &fakeSpawner{readiness: ports.AgentProfileReadiness{Ready: true, Detail: "profile ok"}}
	heartbeats := newFakeHeartbeats()
	svc := outcome.NewWithExecution(store, nil, spawner, heartbeats)

	ctx := context.Background()
	view, err := svc.Create(ctx, validCreateInput())
	if err != nil {
		t.Fatalf("create outcome: %v", err)
	}
	planView, err := svc.ProposePlan(ctx, view.Outcome.ID, 1)
	if err != nil {
		t.Fatalf("propose plan: %v", err)
	}
	if _, err := svc.ApprovePlan(ctx, view.Outcome.ID, outcome.ApprovePlanInput{
		PlanRevisionID:           planView.Plan.ID,
		ExpectedContractRevision: 1,
	}); err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	return svc, store, spawner, heartbeats, view.Outcome.ID, planView.Plan.ID
}

func startInput(planID domain.PlanRevisionID) outcome.StartAttemptInput {
	return outcome.StartAttemptInput{PlanRevisionID: planID, RequestKey: "req-start-1"}
}

func requireAPICode(t *testing.T, err error) string {
	t.Helper()
	var apiErr *apierr.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want apierr.Error", err)
	}
	return apiErr.Code
}

// TestStartAttemptFailClosedQuartetLeavesZeroRows maps the four pre-durable
// refusals: unapproved plan, invalidated brief, narrowed authority, and an
// unready provider profile. NONE may leave any durable row behind.
func TestStartAttemptFailClosedQuartetLeavesZeroRows(t *testing.T) {
	durableRows := func(store *attemptFakeStore, outcomeID domain.OutcomeID) int {
		attempts, err := store.ListAttempts(context.Background(), outcomeID)
		if err != nil {
			t.Fatal(err)
		}
		return len(attempts)
	}

	t.Run("plan not approved", func(t *testing.T) {
		svc, store, _, _, outcomeID, _ := newAttemptHarness(t)
		// Re-propose leaves an unapproved proposal; craft one by proposing on a
		// fresh revision instead: simplest is to use the approved plan but
		// demote it through a direct store append.
		_, err := store.AppendPlanRevision(context.Background(), outcomeID, mustApprovedShape(t, svc, outcomeID))
		if err != nil {
			t.Fatalf("append proposal: %v", err)
		}
		plans, _ := store.ListAttempts(context.Background(), outcomeID)
		_ = plans
		// The newest proposed plan for revision 1 replays; force a genuinely
		// unapproved target by revising the pointer away and back is complex —
		// assert through the service using the freshly appended proposal id.
		proposal, ok, err := store.LatestProposedPlanRevision(context.Background(), outcomeID, 1)
		if err != nil || !ok {
			t.Fatalf("expected replayed proposal, ok=%v err=%v", ok, err)
		}
		_, err = svc.StartAttempt(context.Background(), outcomeID, outcome.StartAttemptInput{
			PlanRevisionID: proposal.ID, RequestKey: "rk-unapproved",
		})
		if code := requireAPICode(t, err); code != outcome.CodePlanNotApproved {
			t.Fatalf("code = %s, want PLAN_NOT_APPROVED", code)
		}
		if n := durableRows(store, outcomeID); n != 0 {
			t.Fatalf("refused admission persisted %d attempts, want 0", n)
		}
	})

	t.Run("brief invalidated by material change", func(t *testing.T) {
		svc, store, _, _, outcomeID, planID := newAttemptHarness(t)
		if _, err := svc.ReviseContract(context.Background(), outcomeID, outcome.ReviseContractInput{
			ExpectedRevision: 1,
			Goal:             "A materially different goal invalidates the frozen brief.",
			SuccessCriteria:  []string{"Different evidence."},
			Review:           "Different review.",
		}); err != nil {
			t.Fatalf("revise: %v", err)
		}
		_, err := svc.StartAttempt(context.Background(), outcomeID, startInput(planID))
		if code := requireAPICode(t, err); code != outcome.CodePlanBriefInvalidated {
			t.Fatalf("code = %s, want PLAN_BRIEF_INVALIDATED", code)
		}
		if n := durableRows(store, outcomeID); n != 0 {
			t.Fatalf("refused admission persisted %d attempts, want 0", n)
		}
	})

	t.Run("capability unauthorized", func(t *testing.T) {
		store := newAttemptFakeStore()
		spawner := &fakeSpawner{readiness: ports.AgentProfileReadiness{Ready: true}}
		wide := outcome.NewWithExecution(store, nil, spawner, newFakeHeartbeats())
		narrow := outcome.NewWithExecution(store, nil, spawner, newFakeHeartbeats())
		narrow.PolicyLayers = [][]string{{domain.CapabilityWorktreeRead}}
		ctx := context.Background()
		view, err := wide.Create(ctx, validCreateInput())
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		planView, err := wide.ProposePlan(ctx, view.Outcome.ID, 1)
		if err != nil {
			t.Fatalf("propose: %v", err)
		}
		// Authority may narrow AFTER approval; admission must re-check it.
		if _, err := wide.ApprovePlan(ctx, view.Outcome.ID, outcome.ApprovePlanInput{PlanRevisionID: planView.Plan.ID}); err != nil {
			t.Fatalf("approve: %v", err)
		}
		_, err = narrow.StartAttempt(ctx, view.Outcome.ID, startInput(planView.Plan.ID))
		if code := requireAPICode(t, err); code != outcome.CodeAttemptCapabilityUnauthorized {
			t.Fatalf("code = %s, want ATTEMPT_CAPABILITY_UNAUTHORIZED", code)
		}
		if n := durableRows(store, view.Outcome.ID); n != 0 {
			t.Fatalf("refused admission persisted %d attempts, want 0", n)
		}
	})

	t.Run("provider not ready", func(t *testing.T) {
		svc, store, spawner, _, outcomeID, planID := newAttemptHarness(t)
		spawner.mu.Lock()
		spawner.readiness = ports.AgentProfileReadiness{Ready: false, Detail: "not logged in"}
		spawner.mu.Unlock()
		_, err := svc.StartAttempt(context.Background(), outcomeID, startInput(planID))
		if code := requireAPICode(t, err); code != outcome.CodeAgentProfileNotReady {
			t.Fatalf("code = %s, want AGENT_PROFILE_NOT_READY", code)
		}
		spawner.readinessErr = ports.ErrAgentBinaryNotFound
		spawner.readiness = ports.AgentProfileReadiness{}
		_, err = svc.StartAttempt(context.Background(), outcomeID, outcome.StartAttemptInput{PlanRevisionID: planID, RequestKey: "rk-2"})
		if code := requireAPICode(t, err); code != outcome.CodeAgentBinaryNotFound {
			t.Fatalf("code = %s, want AGENT_BINARY_NOT_FOUND", code)
		}
		if n := durableRows(store, outcomeID); n != 0 {
			t.Fatalf("refused admissions persisted %d attempts, want 0", n)
		}
	})
}

// currentRevisionForTest resolves the Outcome's current revision content.
func currentRevisionForTest(svc *outcome.Service, outcomeID domain.OutcomeID) (domain.ContractRevision, error) {
	view, err := svc.Get(context.Background(), outcomeID)
	if err != nil {
		return domain.ContractRevision{}, err
	}
	return view.Current, nil
}

// mustApprovedShape builds a minimal VALID proposed plan for the not-approved
// refusal test.
func mustApprovedShape(t *testing.T, svc *outcome.Service, outcomeID domain.OutcomeID) domain.PlanRevision {
	t.Helper()
	full, err := svc.GetLatestPlan(context.Background(), outcomeID)
	if err != nil {
		t.Fatalf("latest plan: %v", err)
	}
	revision, err := currentRevisionForTest(svc, outcomeID)
	if err != nil {
		t.Fatalf("current revision: %v", err)
	}
	unit := full.Plan.WorkUnits[0]
	grants := full.Plan.Grants
	digest, err := domain.ComputeRunBriefCoreDigest(revision, unit, grants)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	return domain.PlanRevision{
		ID:                     domain.PlanRevisionID("plan-proposal-2"),
		OutcomeID:              outcomeID,
		ContractRevisionNumber: 1,
		Status:                 domain.PlanStatusProposed,
		Summary:                "Duplicate proposal",
		WorkUnits:              []domain.WorkUnit{unit},
		Grants:                 grants,
		RunBriefCoreDigest:     digest,
	}
}

// TestStartAttemptAdmissionOrdering pins the happy path: queued row + fence,
// spawn through the narrow seam, session ref with digests + snapshot,
// running transition, and a deterministic brief prompt.
func TestStartAttemptAdmissionOrdering(t *testing.T) {
	svc, store, spawner, heartbeats, outcomeID, planID := newAttemptHarness(t)
	ctx := context.Background()

	view, err := svc.StartAttempt(ctx, outcomeID, startInput(planID))
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if view.Attempt.Status != domain.AttemptRunning || view.Attempt.Number != 1 {
		t.Fatalf("attempt = %+v, want running #1", view.Attempt)
	}
	if len(view.Sessions) != 1 || view.Sessions[0].Seq != 1 {
		t.Fatalf("session refs = %+v, want exactly one binding", view.Sessions)
	}
	ref := view.Sessions[0]
	if len(ref.RunBriefCoreDigest) != 64 || len(ref.RunBriefCompiledDigest) != 64 {
		t.Fatal("both digests must be recorded on the ref")
	}
	if !strings.Contains(ref.AdmissionSnapshot, `"snapshotVersion":1`) {
		t.Fatalf("admission snapshot missing version pin: %s", ref.AdmissionSnapshot)
	}
	if view.Fence == nil || !view.Fence.Open() || view.Fence.AttemptID != view.Attempt.ID {
		t.Fatalf("fence = %+v, want open custody held by this attempt", view.Fence)
	}
	if view.Presentation.Phase != domain.AttemptPhaseUnconfirmed || !view.Presentation.Unconfirmed {
		// The fake session was never signalled: missing heartbeat derives
		// UNCONFIRMED, never dead.
		t.Fatalf("presentation = %+v, want unconfirmed until first signal", view.Presentation)
	}
	if len(spawner.spawned) != 1 {
		t.Fatalf("spawn calls = %d, want 1", spawner.spawnCalls())
	}
	req := spawner.spawned[0]
	if req.Harness != domain.HarnessCodex {
		t.Fatalf("harness = %s, want the locked v0 default codex", req.Harness)
	}
	if !strings.Contains(req.Prompt, "Record focus locally") && !strings.Contains(req.Prompt, validCreateInput().Goal[:20]) {
		t.Fatalf("prompt must carry the frozen goal, got: %.80s", req.Prompt)
	}
	if !strings.Contains(req.Prompt, "Stop conditions:") {
		t.Fatal("prompt must carry the stop conditions")
	}

	// First signal arrives: derivation flips to executing.
	heartbeats.signal(domain.SessionID(ref.SessionID))
	reread, err := svc.GetAttempt(ctx, outcomeID, view.Attempt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reread.Presentation.Phase != domain.AttemptPhaseExecuting {
		t.Fatalf("phase = %s, want executing after first signal", reread.Presentation.Phase)
	}
	_ = store
}

// TestStartAttemptReplayIsIdempotent proves a delivered request key resolves
// to the original attempt without a second admission.
func TestStartAttemptReplayIsIdempotent(t *testing.T) {
	svc, store, spawner, _, outcomeID, planID := newAttemptHarness(t)

	first, err := svc.StartAttempt(context.Background(), outcomeID, startInput(planID))
	if err != nil {
		t.Fatalf("first start: %v", err)
	}
	replay, err := svc.StartAttempt(context.Background(), outcomeID, startInput(planID))
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if replay.Attempt.ID != first.Attempt.ID {
		t.Fatalf("replay produced %s, want original %s", replay.Attempt.ID, first.Attempt.ID)
	}
	if spawner.spawnCalls() != 1 {
		t.Fatalf("spawn calls = %d, want exactly 1 across the replay", spawner.spawnCalls())
	}
	attempts, _ := store.ListAttempts(context.Background(), outcomeID)
	if len(attempts) != 1 {
		t.Fatalf("attempts = %d, want 1", len(attempts))
	}
}

// TestSpawnCrashLeavesTruthfulFailedStateAndHeldFence covers D3's crash rule:
// after the durable row exists, a spawner crash records truthful failed +
// observation + receipt, keeps the fence HELD, and blocks replacement until
// reconcile releases custody.
func TestSpawnCrashLeavesTruthfulFailedStateAndHeldFence(t *testing.T) {
	svc, store, spawner, _, outcomeID, planID := newAttemptHarness(t)
	ctx := context.Background()
	spawner.failNextSpawn(errors.New("provider process vanished during spawn"))

	_, err := svc.StartAttempt(ctx, outcomeID, startInput(planID))
	var apiErr *apierr.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "ATTEMPT_ADMIT_FAILED" {
		t.Fatalf("err = %v, want ATTEMPT_ADMIT_FAILED", err)
	}
	attempts, listErr := store.ListAttempts(ctx, outcomeID)
	if listErr != nil || len(attempts) != 1 {
		t.Fatalf("failed attempt must exist durably once, len=%d err=%v", len(attempts), listErr)
	}
	stored := attempts[0]
	if stored.Status != domain.AttemptFailed {
		t.Fatalf("status = %s, want failed", stored.Status)
	}
	fence, held, err := store.OpenFenceForSubject(ctx, domain.FenceSubjectForProject("mer"))
	if err != nil || !held || fence.AttemptID != stored.ID {
		t.Fatalf("fence must stay HELD by the failed attempt, held=%v err=%v", held, err)
	}
	observations, _ := store.ListAttemptObservations(ctx, stored.ID)
	if len(observations) == 0 || observations[0].Kind != domain.ObservationAdmissionFailed {
		t.Fatalf("observations = %+v, want admission_failed first", observations)
	}
	receipts, _ := store.ListRecoveryReceipts(ctx, stored.ID)
	if len(receipts) != 1 || receipts[0].Resolution != domain.RecoveryNeedsAttention {
		t.Fatalf("receipts = %+v, wants_attention", receipts)
	}

	// Replacement cannot bypass reconcile: custody is still held.
	if _, err := svc.StartAttempt(ctx, outcomeID, outcome.StartAttemptInput{PlanRevisionID: planID, RequestKey: "rk-replace"}); err == nil {
		t.Fatal("replacement start must be refused while the failed attempt holds custody")
	} else if code := requireAPICode(t, err); code != outcome.CodeAttemptFenceHeld {
		t.Fatalf("code = %s, want ATTEMPT_FENCE_HELD", code)
	}
}

// TestContainReconcileReplacementFlow walks the injected-loss path from the
// v0 failure matrix end to end.
func TestContainReconcileReplacementFlow(t *testing.T) {
	svc, _, _, heartbeats, outcomeID, planID := newAttemptHarness(t)
	ctx := context.Background()

	first, err := svc.StartAttempt(ctx, outcomeID, startInput(planID))
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	heartbeats.signal(domain.SessionID(first.Sessions[0].SessionID))

	// Provider process disappears mid-work-unit: the session GCs away.
	heartbeats.forget(domain.SessionID(first.Sessions[0].SessionID))
	unconfirmed, err := svc.GetAttempt(ctx, outcomeID, first.Attempt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !unconfirmed.Presentation.Unconfirmed || unconfirmed.Presentation.Phase != domain.AttemptPhaseUnconfirmed {
		t.Fatalf("presentation = %+v, want unconfirmed", unconfirmed.Presentation)
	}

	// Containment marks suspicion without deciding anything.
	contained, err := svc.RecoverAttempt(ctx, outcomeID, first.Attempt.ID, outcome.RecoveryInput{Action: outcome.RecoveryActionContain})
	if err != nil {
		t.Fatalf("contain: %v", err)
	}
	if contained.Attempt.Attempt.Status != domain.AttemptRunning {
		t.Fatalf("containment mutated status to %s", contained.Attempt.Attempt.Status)
	}
	lastObs := contained.Attempt.Observations[len(contained.Attempt.Observations)-1]
	if lastObs.Kind != domain.ObservationAttemptContained {
		t.Fatalf("last observation = %s, want contained", lastObs.Kind)
	}

	// Resuming without proof of life refuses closed.
	_, err = svc.RecoverAttempt(ctx, outcomeID, first.Attempt.ID, outcome.RecoveryInput{Action: outcome.RecoveryActionResume})
	if code := requireAPICode(t, err); code != outcome.CodeAttemptLivenessUnproven {
		t.Fatalf("resume code = %s, want ATTEMPT_LIVENESS_UNPROVEN", code)
	}

	// Reconcile cannot prove liveness: lost verdict + released custody +
	// replacement receipt.
	reconciled, err := svc.RecoverAttempt(ctx, outcomeID, first.Attempt.ID, outcome.RecoveryInput{Action: outcome.RecoveryActionReconcile})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if reconciled.Attempt.Attempt.Status != domain.AttemptLost {
		t.Fatalf("status = %s, want lost", reconciled.Attempt.Attempt.Status)
	}
	if reconciled.Receipt == nil || reconciled.Receipt.Resolution != domain.RecoveryReplacement {
		t.Fatalf("receipt = %+v, want replacement_attempt", reconciled.Receipt)
	}
	if reconciled.Attempt.Fence != nil {
		t.Fatal("custody must be released after the lost verdict")
	}

	// Safe next action: a replacement attempt acquires the freed subject.
	replacement, err := svc.StartAttempt(ctx, outcomeID, outcome.StartAttemptInput{PlanRevisionID: planID, RequestKey: "rk-replacement"})
	if err != nil {
		t.Fatalf("replacement start: %v", err)
	}
	if replacement.Attempt.Number != 2 || replacement.Attempt.ID == first.Attempt.ID {
		t.Fatalf("replacement = %+v, want NEW attempt #2", replacement.Attempt)
	}

	// Stale events stay inspectable but inert (D5): appending an observation
	// to the LOST predecessor succeeds and touches nothing current.
	stale, err := svc.RecordObservation(ctx, outcomeID, first.Attempt.ID, outcome.RecordObservationInput{
		Kind: "stale_provider_report", Payload: `{"claims":"success"}`,
	})
	if err != nil {
		t.Fatalf("stale observation must remain insertable: %v", err)
	}
	if stale.Seq < 2 {
		t.Fatalf("stale observation seq = %d", stale.Seq)
	}
	current, err := svc.GetAttempt(ctx, outcomeID, replacement.Attempt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(current.Observations) != 0 {
		t.Fatalf("stale observation leaked into current truth: %+v", current.Observations)
	}
	if current.Attempt.Status != domain.AttemptRunning {
		t.Fatalf("current status = %s, want running", current.Attempt.Status)
	}
}

// TestReconcileProvenAliveRecordsResumed proves the resume-same branch:
// present + signalled + not terminated => resumed receipt, no status churn.
func TestReconcileProvenAliveRecordsResumed(t *testing.T) {
	svc, _, _, heartbeats, outcomeID, planID := newAttemptHarness(t)
	ctx := context.Background()
	view, err := svc.StartAttempt(ctx, outcomeID, startInput(planID))
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	heartbeats.signal(domain.SessionID(view.Sessions[0].SessionID))

	result, err := svc.RecoverAttempt(ctx, outcomeID, view.Attempt.ID, outcome.RecoveryInput{Action: outcome.RecoveryActionReconcile})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.Receipt == nil || result.Receipt.Resolution != domain.RecoveryResumed {
		t.Fatalf("receipt = %+v, want resumed", result.Receipt)
	}
	if result.Attempt.Attempt.Status != domain.AttemptRunning {
		t.Fatalf("status = %s, want unchanged running", result.Attempt.Attempt.Status)
	}
}

// TestLivenessLoopClassifiesTerminatedProviderSession drives the daemon hook:
// termination becomes reconciled + provider_exit observation — an ordered
// observation only, never success.
func TestLivenessLoopClassifiesTerminatedProviderSession(t *testing.T) {
	svc, _, _, heartbeats, outcomeID, planID := newAttemptHarness(t)
	ctx := context.Background()
	view, err := svc.StartAttempt(ctx, outcomeID, startInput(planID))
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	heartbeats.signal(domain.SessionID(view.Sessions[0].SessionID))
	heartbeats.terminate(domain.SessionID(view.Sessions[0].SessionID))

	if err := svc.EvaluateAttemptLiveness(ctx); err != nil {
		t.Fatalf("liveness evaluation: %v", err)
	}
	reread, err := svc.GetAttempt(ctx, outcomeID, view.Attempt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reread.Attempt.Status != domain.AttemptReconciled {
		t.Fatalf("status = %s, want reconciled", reread.Attempt.Status)
	}
	if !reread.Presentation.EndedUnclassified {
		t.Fatalf("presentation = %+v, want ended-unclassified", reread.Presentation)
	}
	lastObs := reread.Observations[len(reread.Observations)-1]
	if lastObs.Kind != domain.ObservationProviderExit {
		t.Fatalf("observation = %s, want provider_exit", lastObs.Kind)
	}
	// A second pass is a no-op: the attempt no longer runs.
	if err := svc.EvaluateAttemptLiveness(ctx); err != nil {
		t.Fatalf("second liveness pass: %v", err)
	}
}

// TestPauseResumeCancelGuards covers owner controls including resume
// enforcing the readiness contract.
func TestPauseResumeCancelGuards(t *testing.T) {
	svc, _, spawner, heartbeats, outcomeID, planID := newAttemptHarness(t)
	ctx := context.Background()
	view, err := svc.StartAttempt(ctx, outcomeID, startInput(planID))
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	heartbeats.signal(domain.SessionID(view.Sessions[0].SessionID))

	paused, err := svc.PauseAttempt(ctx, outcomeID, view.Attempt.ID)
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	if paused.Attempt.Status != domain.AttemptPaused || paused.Presentation.Phase != domain.AttemptPhaseSuspended {
		t.Fatalf("paused state = %+v / %+v", paused.Attempt, paused.Presentation)
	}

	// Resume enforces spawn's readiness contract: degrade the profile and the
	// resume refuses.
	spawner.mu.Lock()
	spawner.readiness = ports.AgentProfileReadiness{Ready: false, Detail: "logged out"}
	spawner.mu.Unlock()
	_, err = svc.ResumeAttempt(ctx, outcomeID, view.Attempt.ID)
	if code := requireAPICode(t, err); code != outcome.CodeAgentProfileNotReady {
		t.Fatalf("resume code = %s, want AGENT_PROFILE_NOT_READY", code)
	}
	spawner.mu.Lock()
	spawner.readiness = ports.AgentProfileReadiness{Ready: true}
	spawner.mu.Unlock()

	resumed, err := svc.ResumeAttempt(ctx, outcomeID, view.Attempt.ID)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if resumed.Attempt.Status != domain.AttemptRunning {
		t.Fatalf("status = %s, want running", resumed.Attempt.Status)
	}

	cancelled, err := svc.CancelAttempt(ctx, outcomeID, view.Attempt.ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if cancelled.Attempt.Status != domain.AttemptCancelled {
		t.Fatalf("status = %s, want cancelled", cancelled.Attempt.Status)
	}
	// Terminal states accept nothing further.
	_, err = svc.PauseAttempt(ctx, outcomeID, view.Attempt.ID)
	if code := requireAPICode(t, err); code != "ATTEMPT_NOT_RUNNING" {
		t.Fatalf("pause-after-end code = %s, want ATTEMPT_NOT_RUNNING", code)
	}
}

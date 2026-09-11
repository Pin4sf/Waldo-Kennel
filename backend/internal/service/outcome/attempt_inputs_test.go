package outcome_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// TestStartAttempt_RootWorkUnitIsAdmittedWithNoInputs pins the other half of
// the handoff contract: a unit with no dependencies has nothing to receive,
// and the launch request must say so rather than carrying a stale input set.
func TestStartAttempt_RootWorkUnitIsAdmittedWithNoInputs(t *testing.T) {
	svc, _, spawner, _, outcomeID, planID := newAttemptHarness(t)
	plan, err := svc.GetLatestPlan(context.Background(), outcomeID)
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	rememberFirstWorkUnit(plan.Plan)

	if _, err := svc.StartAttempt(context.Background(), outcomeID, startInput(planID)); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(spawner.spawned) != 1 {
		t.Fatalf("spawn calls = %d", len(spawner.spawned))
	}
	if got := spawner.spawned[0].Inputs; len(got) != 0 {
		t.Fatalf("root WorkUnit was admitted with inputs %#v", got)
	}
}

// TestStartAttempt_InputProvisioningFailureEndsTheAttemptWithoutLaunching is
// the fail-closed launch rule. Provisioning runs before any provider process
// exists, so the refusal must say a provider was NOT launched, record why
// durably, and end the Attempt instead of leaving the owner reconciling a run
// that never began.
func TestStartAttempt_InputProvisioningFailureEndsTheAttemptWithoutLaunching(t *testing.T) {
	svc, store, spawner, _, outcomeID, planID := newAttemptHarness(t)
	plan, err := svc.GetLatestPlan(context.Background(), outcomeID)
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	rememberFirstWorkUnit(plan.Plan)
	spawner.failNextSpawn(fmt.Errorf("%w: predecessor blob digest mismatch", ports.ErrAttemptInputProvisioning))

	_, startErr := svc.StartAttempt(context.Background(), outcomeID, startInput(planID))
	if code := requireAPICode(t, startErr); code != "UPSTREAM_MATERIALIZATION_FAILED" {
		t.Fatalf("refusal code = %q, want UPSTREAM_MATERIALIZATION_FAILED", code)
	}

	attempts, err := store.ListAttempts(context.Background(), outcomeID)
	if err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	if len(attempts) != 1 {
		t.Fatalf("attempts = %d, want the one admitted attempt", len(attempts))
	}
	// The Attempt is ended, not left queued holding custody: nothing ran, so
	// keeping the worktree fence would block the owner without protecting
	// anything.
	if attempts[0].Status != domain.AttemptFailed {
		t.Fatalf("attempt status = %s, want failed", attempts[0].Status)
	}
	observations, err := store.ListAttemptObservations(context.Background(), attempts[0].ID)
	if err != nil {
		t.Fatalf("list observations: %v", err)
	}
	var recorded bool
	for _, observation := range observations {
		if observation.Kind == domain.ObservationInputProvisioningFailed {
			recorded = true
		}
		// An ambiguous-start observation here would send the owner to
		// reconcile a provider that was never launched.
		if observation.Kind == domain.ObservationAdmissionAmbiguous || observation.Kind == domain.ObservationActivationAmbiguous {
			t.Fatalf("a known pre-launch failure was recorded as ambiguous: %s", observation.Kind)
		}
	}
	if !recorded {
		t.Fatalf("no provisioning-failure observation was recorded: %#v", observations)
	}
	if _, bound, err := store.LatestAttemptSessionRef(context.Background(), attempts[0].ID); err != nil || bound {
		t.Fatalf("a provider session was bound despite failed provisioning: bound=%v err=%v", bound, err)
	}
}

// TestStartAttempt_ReplayedRequestKeyDoesNotLaunchTwice keeps the handoff
// idempotent: repeating a start returns the same Attempt rather than resolving
// inputs again and admitting a second one.
func TestStartAttempt_ReplayedRequestKeyDoesNotLaunchTwice(t *testing.T) {
	svc, _, spawner, _, outcomeID, planID := newAttemptHarness(t)
	plan, err := svc.GetLatestPlan(context.Background(), outcomeID)
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	rememberFirstWorkUnit(plan.Plan)

	first, err := svc.StartAttempt(context.Background(), outcomeID, startInput(planID))
	if err != nil {
		t.Fatalf("first start: %v", err)
	}
	second, err := svc.StartAttempt(context.Background(), outcomeID, startInput(planID))
	if err != nil {
		t.Fatalf("replayed start: %v", err)
	}
	if first.Attempt.ID != second.Attempt.ID {
		t.Fatalf("replay produced a second Attempt: %s vs %s", first.Attempt.ID, second.Attempt.ID)
	}
	if calls := spawner.spawnCalls(); calls != 1 {
		t.Fatalf("spawn calls = %d, want exactly one launch", calls)
	}
}

func TestStartAttempt_WorkspaceFailureIsKnownAndReleasesCustody(t *testing.T) {
	svc, store, spawner, _, outcomeID, planID := newAttemptHarness(t)
	plan, err := svc.GetLatestPlan(context.Background(), outcomeID)
	if err != nil {
		t.Fatal(err)
	}
	rememberFirstWorkUnit(plan.Plan)
	spawner.failNextSpawn(fmt.Errorf("%w: branch checked out in another profile", ports.ErrAttemptWorkspacePreparation))
	_, err = svc.StartAttempt(context.Background(), outcomeID, startInput(planID))
	if requireAPICode(t, err) != "ATTEMPT_WORKSPACE_PREPARATION_FAILED" {
		t.Fatal(err)
	}
	attempts, err := store.ListAttempts(context.Background(), outcomeID)
	if err != nil || len(attempts) != 1 || attempts[0].Status != domain.AttemptFailed {
		t.Fatalf("failure not terminal: %+v %v", attempts, err)
	}
	observations, err := store.ListAttemptObservations(context.Background(), attempts[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, observation := range observations {
		if observation.Kind == domain.ObservationAdmissionAmbiguous {
			t.Fatal("known workspace failure treated as unknown launch")
		}
	}
	if _, err := svc.StartAttempt(context.Background(), outcomeID, startInput(planID)); err != nil {
		t.Fatalf("known failure held custody against retry: %v", err)
	}
}

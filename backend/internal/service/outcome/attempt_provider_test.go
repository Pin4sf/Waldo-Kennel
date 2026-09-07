package outcome_test

import (
	"context"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

type configuredAttemptStore struct {
	*attemptFakeStore
	project domain.ProjectRecord
}

func (f *configuredAttemptStore) GetProject(_ context.Context, id string) (domain.ProjectRecord, bool, error) {
	if f.project.ID != id {
		return domain.ProjectRecord{}, false, nil
	}
	return f.project, true, nil
}

func newConfiguredAttemptHarness(t *testing.T, provider domain.AgentHarness) (*outcome.Service, *configuredAttemptStore, *fakeSpawner, domain.OutcomeID, domain.PlanRevisionID) {
	t.Helper()
	base := newAttemptFakeStore()
	store := &configuredAttemptStore{
		attemptFakeStore: base,
		project: domain.ProjectRecord{
			ID:   "provider-attempt",
			Path: "/tmp/provider-attempt",
			Config: domain.ProjectConfig{
				Worker: domain.RoleOverride{Harness: provider},
			},
		},
	}
	spawner := &fakeSpawner{readiness: ports.AgentProfileReadiness{Ready: true, Detail: "ready"}}
	svc := outcome.NewWithExecution(store, nil, spawner, newFakeHeartbeats())
	view, err := svc.Create(context.Background(), outcome.CreateInput{
		ProjectID:       "provider-attempt",
		Title:           "Provider admission",
		Goal:            "Execute only the authorized provider.",
		SuccessCriteria: []string{"no provider substitution occurs"},
		Review:          "deterministic tests",
		RequestKey:      "req-provider-attempt-create",
	})
	if err != nil {
		t.Fatalf("create outcome: %v", err)
	}
	planView, err := svc.ProposePlan(context.Background(), view.Outcome.ID, 1)
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if _, err := svc.ApprovePlan(context.Background(), view.Outcome.ID, outcome.ApprovePlanInput{
		PlanRevisionID:           planView.Plan.ID,
		ExpectedContractRevision: 1,
	}); err != nil {
		t.Fatalf("approve: %v", err)
	}
	return svc, store, spawner, view.Outcome.ID, planView.Plan.ID
}

func TestStartAttemptUsesFrozenNonCodexProviderWhenRequestHarnessIsEmpty(t *testing.T) {
	svc, _, spawner, outcomeID, planID := newConfiguredAttemptHarness(t, domain.HarnessClaudeCode)
	if _, err := svc.StartAttempt(context.Background(), outcomeID, outcome.StartAttemptInput{
		PlanRevisionID: planID,
		RequestKey:     "req-provider-attempt-start",
	}); err != nil {
		t.Fatalf("start: %v", err)
	}
	spawner.mu.Lock()
	defer spawner.mu.Unlock()
	if len(spawner.spawned) != 1 {
		t.Fatalf("spawn calls = %d, want 1", len(spawner.spawned))
	}
	if got := spawner.spawned[0].Harness; got != domain.HarnessClaudeCode {
		t.Fatalf("spawned provider = %q, want frozen provider %q", got, domain.HarnessClaudeCode)
	}
}

func TestStartAttemptRejectsProviderMismatchBeforePersistenceOrSpawn(t *testing.T) {
	svc, store, spawner, outcomeID, planID := newConfiguredAttemptHarness(t, domain.HarnessClaudeCode)
	_, err := svc.StartAttempt(context.Background(), outcomeID, outcome.StartAttemptInput{
		PlanRevisionID: planID,
		Harness:        domain.HarnessCodex,
		RequestKey:     "req-provider-mismatch",
	})
	if code := requireAPICode(t, err); code != outcome.CodeAttemptProviderMismatch {
		t.Fatalf("code = %s, want %s", code, outcome.CodeAttemptProviderMismatch)
	}
	attempts, listErr := store.ListAttempts(context.Background(), outcomeID)
	if listErr != nil {
		t.Fatalf("list attempts: %v", listErr)
	}
	if len(attempts) != 0 {
		t.Fatalf("mismatch persisted %d attempts, want 0", len(attempts))
	}
	spawner.mu.Lock()
	spawned := len(spawner.spawned)
	spawner.mu.Unlock()
	if spawned != 0 {
		t.Fatalf("mismatch spawned %d provider sessions, want 0", spawned)
	}
}

func TestStartAttemptRejectsLegacyUnboundPlanBeforePersistenceOrSpawn(t *testing.T) {
	svc, store, spawner, outcomeID, planID := newConfiguredAttemptHarness(t, domain.HarnessClaudeCode)

	// Simulate a plan created before provider binding existed. Historical rows
	// remain readable, but admission must not infer a provider from current
	// Project settings or from a brand default.
	store.planFakeStore.mu.Lock()
	for i := range store.plans[outcomeID] {
		if store.plans[outcomeID][i].ID == planID {
			store.plans[outcomeID][i].WorkUnits[0].Provider = ""
		}
	}
	units := store.units[planID]
	if len(units) == 1 {
		units[0].Provider = ""
		store.units[planID] = units
	}
	store.planFakeStore.mu.Unlock()

	_, err := svc.StartAttempt(context.Background(), outcomeID, outcome.StartAttemptInput{
		PlanRevisionID: planID,
		RequestKey:     "req-provider-unbound",
	})
	if code := requireAPICode(t, err); code != outcome.CodePlanProviderUnbound {
		t.Fatalf("code = %s, want %s", code, outcome.CodePlanProviderUnbound)
	}
	attempts, listErr := store.ListAttempts(context.Background(), outcomeID)
	if listErr != nil {
		t.Fatalf("list attempts: %v", listErr)
	}
	if len(attempts) != 0 {
		t.Fatalf("unbound plan persisted %d attempts, want 0", len(attempts))
	}
	spawner.mu.Lock()
	spawned := len(spawner.spawned)
	spawner.mu.Unlock()
	if spawned != 0 {
		t.Fatalf("unbound plan spawned %d provider sessions, want 0", spawned)
	}
}

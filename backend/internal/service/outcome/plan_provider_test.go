package outcome_test

import (
	"context"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

// GetProject gives the shared plan/attempt fake an explicit non-Codex worker.
// Existing tests should therefore prove that no hidden Codex fixture remains.
func (f *planFakeStore) GetProject(_ context.Context, id string) (domain.ProjectRecord, bool, error) {
	if id != "mer" {
		return domain.ProjectRecord{}, false, nil
	}
	return domain.ProjectRecord{
		ID:   "mer",
		Path: "/tmp/mer",
		Config: domain.ProjectConfig{
			Worker: domain.RoleOverride{Harness: domain.HarnessClaudeCode},
		},
	}, true, nil
}

type configuredPlanStore struct {
	*planFakeStore
	project domain.ProjectRecord
}

func (f *configuredPlanStore) GetProject(_ context.Context, id string) (domain.ProjectRecord, bool, error) {
	if f.project.ID != id {
		return domain.ProjectRecord{}, false, nil
	}
	return f.project, true, nil
}

func seedPlanServiceWithProject(t *testing.T, project domain.ProjectRecord) (*outcome.Service, *configuredPlanStore, domain.OutcomeID) {
	t.Helper()
	base := newPlanFakeStore()
	store := &configuredPlanStore{planFakeStore: base, project: project}
	store.fakeStore.spaces[domain.ProjectID(project.ID)] = domain.ResponsibilitySpace{
		ID:        "rsp-provider-plan",
		Kind:      domain.ResponsibilitySpaceWorkProject,
		ProjectID: domain.ProjectID(project.ID),
	}
	svc := outcome.New(store, nil)
	view, err := svc.Create(context.Background(), outcome.CreateInput{
		ProjectID:       domain.ProjectID(project.ID),
		Title:           "Provider-bound work",
		Goal:            "Run the authorized provider only.",
		SuccessCriteria: []string{"the provider is frozen into the WorkUnit"},
		Review:          "deterministic tests",
		RequestKey:      "req-provider-plan-" + project.ID,
	})
	if err != nil {
		t.Fatalf("seed outcome: %v", err)
	}
	return svc, store, view.Outcome.ID
}

func TestProposePlanBindsExplicitProjectWorker(t *testing.T) {
	svc, _, outcomeID := newPlanTestService(t)
	view, err := svc.ProposePlan(context.Background(), outcomeID, 1)
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if got := view.Plan.WorkUnits[0].Provider; got != domain.HarnessClaudeCode {
		t.Fatalf("WorkUnit provider = %q, want explicit Project worker %q", got, domain.HarnessClaudeCode)
	}
}

func TestProposePlanFailsWhenProjectWorkerIsUnconfigured(t *testing.T) {
	project := domain.ProjectRecord{ID: "unconfigured", Path: "/tmp/unconfigured"}
	svc, store, outcomeID := seedPlanServiceWithProject(t, project)

	_, err := svc.ProposePlan(context.Background(), outcomeID, 1)
	if code := apiCode(t, err); code != outcome.CodePlanProviderUnbound {
		t.Fatalf("code = %s, want %s", code, outcome.CodePlanProviderUnbound)
	}
	store.mu.Lock()
	persisted := len(store.plans[outcomeID])
	store.mu.Unlock()
	if persisted != 0 {
		t.Fatalf("unbound proposal persisted %d plans, want 0", persisted)
	}
}

func TestChangingProjectWorkerCreatesFreshProviderBoundPlan(t *testing.T) {
	project := domain.ProjectRecord{
		ID:   "switch-worker",
		Path: "/tmp/switch-worker",
		Config: domain.ProjectConfig{
			Worker: domain.RoleOverride{Harness: domain.HarnessClaudeCode},
		},
	}
	svc, store, outcomeID := seedPlanServiceWithProject(t, project)
	first, err := svc.ProposePlan(context.Background(), outcomeID, 1)
	if err != nil {
		t.Fatalf("first proposal: %v", err)
	}
	store.project.Config.Worker.Harness = domain.HarnessPi
	second, err := svc.ProposePlan(context.Background(), outcomeID, 1)
	if err != nil {
		t.Fatalf("second proposal: %v", err)
	}
	if second.Plan.ID == first.Plan.ID {
		t.Fatal("changing the Project worker must create a fresh authorization artifact")
	}
	if second.Plan.WorkUnits[0].Provider != domain.HarnessPi {
		t.Fatalf("second provider = %q, want %q", second.Plan.WorkUnits[0].Provider, domain.HarnessPi)
	}
	if second.Plan.RunBriefCoreDigest == first.Plan.RunBriefCoreDigest {
		t.Fatal("changing the provider must change the RunBrief core digest")
	}
}

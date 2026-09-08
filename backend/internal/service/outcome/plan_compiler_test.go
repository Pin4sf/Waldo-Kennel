package outcome_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

type planningFakeStore struct {
	*attemptFakeStore
	project      domain.ProjectRecord
	projectReads int
	runs         map[domain.IntelligenceRunID]domain.IntelligenceRun
}

func newPlanningFakeStore() *planningFakeStore {
	return &planningFakeStore{
		attemptFakeStore: newAttemptFakeStore(),
		project: domain.ProjectRecord{
			ID: "mer",
			Config: domain.ProjectConfig{Worker: domain.RoleOverride{
				Harness:     domain.HarnessClaudeCode,
				AgentConfig: domain.AgentConfig{Model: "sonnet-test"},
			}},
		},
		runs: map[domain.IntelligenceRunID]domain.IntelligenceRun{},
	}
}

func (f *planningFakeStore) GetProject(_ context.Context, id string) (domain.ProjectRecord, bool, error) {
	f.projectReads++
	if id != f.project.ID {
		return domain.ProjectRecord{}, false, nil
	}
	return f.project, true, nil
}

func (f *planningFakeStore) CreateIntelligenceRun(_ context.Context, run domain.IntelligenceRun) error {
	if err := run.Validate(); err != nil {
		return err
	}
	if _, exists := f.runs[run.ID]; exists {
		return fmt.Errorf("duplicate intelligence run %s", run.ID)
	}
	f.runs[run.ID] = run
	return nil
}
func (f *planningFakeStore) GetIntelligenceRun(_ context.Context, id domain.IntelligenceRunID) (domain.IntelligenceRun, bool, error) {
	run, ok := f.runs[id]
	return run, ok, nil
}
func (f *planningFakeStore) ListNonTerminalIntelligenceRuns(_ context.Context) ([]domain.IntelligenceRun, error) {
	var out []domain.IntelligenceRun
	for _, run := range f.runs {
		if !run.Status.Terminal() {
			out = append(out, run)
		}
	}
	return out, nil
}
func (f *planningFakeStore) RecordIntelligenceRunEffectiveProvenance(_ context.Context, id domain.IntelligenceRunID, provider domain.IntelligenceProviderID, model, native string) error {
	run, ok := f.runs[id]
	if !ok {
		return fmt.Errorf("missing intelligence run %s", id)
	}
	run.EffectiveProvider, run.EffectiveModel, run.NativeSessionRef = provider, model, native
	f.runs[id] = run
	return nil
}
func (f *planningFakeStore) UpdateIntelligenceRunStatus(_ context.Context, id domain.IntelligenceRunID, status domain.IntelligenceRunStatus, output domain.SHA256Digest, code, detail string, completed *time.Time) error {
	run, ok := f.runs[id]
	if !ok {
		return fmt.Errorf("missing intelligence run %s", id)
	}
	run.Status, run.OutputDigest, run.FailureCode, run.FailureDetail, run.CompletedAt = status, output, code, detail, completed
	f.runs[id] = run
	return nil
}

type twoUnitPlanIntelligence struct{ calls int }

func (*twoUnitPlanIntelligence) ID() domain.IntelligenceProviderID { return "test-plan-intelligence" }
func (*twoUnitPlanIntelligence) AnalyzeContract(context.Context, ports.ContractIntelligenceRequest) (ports.ContractIntelligenceResponse, error) {
	return ports.ContractIntelligenceResponse{}, fmt.Errorf("contract intelligence not used")
}
func (p *twoUnitPlanIntelligence) DraftPlan(context.Context, ports.PlanIntelligenceRequest) (ports.PlanIntelligenceResponse, error) {
	p.calls++
	return ports.PlanIntelligenceResponse{
		Proposal: domain.PlanDraftProposal{
			Summary: "Edit, then verify the confirmed Contract.",
			WorkUnits: []domain.PlanDraftWorkUnit{
				{Key: "verify", Title: "Verify outcome", Intent: domain.WorkUnitIntentExecute, OutputSummary: "Verified result", CriteriaCovered: []string{"C2"}, DependsOn: []string{"edit"}, EvidenceIdeas: []string{"verification output"}},
				{Key: "edit", Title: "Implement outcome", Intent: domain.WorkUnitIntentModify, OutputSummary: "Implemented result", CriteriaCovered: []string{"C1"}, EvidenceIdeas: []string{"workspace diff"}},
			},
		},
		Provenance: ports.IntelligenceProvenance{EffectiveProvider: "test-plan-intelligence", EffectiveModel: "planner-test"},
	}, nil
}

type routingInventoryFake struct {
	calls      int
	preference *domain.RoutingPreference
	candidates []domain.RoutingCandidate
}
func (r *routingInventoryFake) RoutingSnapshot(_ context.Context, _ domain.ProjectID, preference *domain.RoutingPreference) (ports.RoutingInventorySnapshot, error) {
	r.calls++
	if preference != nil {
		copy := *preference
		r.preference = &copy
	}
	return ports.RoutingInventorySnapshot{SnapshotID: "snapshot-test", Candidates: r.candidates}, nil
}

func readyClaudeCandidate() domain.RoutingCandidate {
	return domain.RoutingCandidate{
		ID: "claude-code", Provider: "claude-code", ModelSelection: domain.ExecutionBindingModelProviderDefault,
		WorkerEligible: true, CoordinatorEligible: true, Readiness: domain.CapabilitySupported,
		Capabilities: map[string]domain.CapabilitySupport{
			domain.CapabilityWorktreeRead:  domain.CapabilitySupported,
			domain.CapabilityWorktreeWrite: domain.CapabilitySupported,
			domain.CapabilityWorktreeExec:  domain.CapabilitySupported,
		},
		Models: map[string]domain.CapabilitySupport{"sonnet-test": domain.CapabilitySupported},
	}
}

func fullLocalAuthority() domain.ProposedAuthority {
	return domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true, ExecuteLocal: true}
}

func newPlanningTestService(t *testing.T, router *routingInventoryFake) (*outcome.Service, *planningFakeStore, domain.OutcomeID, *twoUnitPlanIntelligence) {
	t.Helper()
	store := newPlanningFakeStore()
	provider := &twoUnitPlanIntelligence{}
	svc := outcome.New(store, nil).WithPlanning(provider, router)

	store.planFakeStore.mu.Lock()
	store.fakeStore.spaces["mer"] = domain.ResponsibilitySpace{ID: "rsp-plan-compiler", Kind: domain.ResponsibilitySpaceWorkProject, ProjectID: "mer"}
	store.planFakeStore.mu.Unlock()

	ctx := context.Background()
	view, err := svc.Create(ctx, outcome.CreateInput{
		ProjectID: "mer", Title: "Compiler outcome", Goal: "Ship and verify a bounded change.",
		SuccessCriteria: []string{"Implementation is present."}, Review: "Run deterministic verification.",
		AuthorityCeiling: fullLocalAuthority(), StopConditions: []string{"Stop before remote effects."},
		RequestKey: "req-plan-compiler",
	})
	if err != nil {
		t.Fatalf("seed outcome: %v", err)
	}
	view, err = svc.ReviseContract(ctx, view.Outcome.ID, outcome.ReviseContractInput{
		ExpectedRevision: 1, Goal: "Ship and verify a bounded change.",
		SuccessCriteria: []string{"Implementation is present.", "Verification proves the implementation behaves as required."},
		Review: "Run deterministic verification.", AuthorityCeiling: fullLocalAuthority(), StopConditions: []string{"Stop before remote effects."},
	})
	if err != nil {
		t.Fatalf("seed second criterion: %v", err)
	}
	return svc, store, view.Outcome.ID, provider
}

func TestProposePlanCompilesIntelligenceGraphAndRoutesEveryWorkUnit(t *testing.T) {
	router := &routingInventoryFake{candidates: []domain.RoutingCandidate{readyClaudeCandidate()}}
	svc, store, outcomeID, provider := newPlanningTestService(t, router)
	view, err := svc.ProposePlan(context.Background(), outcomeID, 2)
	if err != nil {
		t.Fatalf("propose plan: %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("plan intelligence calls = %d, want 1", provider.calls)
	}
	if len(view.Plan.WorkUnits) != 2 || len(view.Plan.RoutingDecisions) != 2 {
		t.Fatalf("compiled plan = %+v", view.Plan)
	}
	ordered, err := view.Plan.TopologicalWorkUnits()
	if err != nil {
		t.Fatalf("topological order: %v", err)
	}
	if ordered[0].Title != "Implement outcome" || ordered[1].Title != "Verify outcome" {
		t.Fatalf("topological titles = %q -> %q", ordered[0].Title, ordered[1].Title)
	}
	if !reflect.DeepEqual(ordered[0].RequiredCapabilities, []string{domain.CapabilityWorktreeRead, domain.CapabilityWorktreeWrite}) {
		t.Fatalf("edit capabilities = %v, want read+write only", ordered[0].RequiredCapabilities)
	}
	if !reflect.DeepEqual(ordered[1].RequiredCapabilities, []string{domain.CapabilityWorktreeRead, domain.CapabilityWorktreeExec}) {
		t.Fatalf("verify capabilities = %v, want read+exec only", ordered[1].RequiredCapabilities)
	}
	for _, unit := range view.Plan.WorkUnits {
		if unit.Provider != domain.HarnessClaudeCode || unit.ModelSelection != domain.ExecutionBindingModelExplicit || unit.Model != "sonnet-test" {
			t.Fatalf("work unit %s binding = %s/%s/%s", unit.ID, unit.Provider, unit.ModelSelection, unit.Model)
		}
		if len(unit.CriterionIDs) != 1 || unit.CriterionIDs[0].IsZero() {
			t.Fatalf("work unit %s criteria = %+v", unit.ID, unit.CriterionIDs)
		}
	}
	if err := domain.ValidateExactPlanCapabilityGrants(view.Plan.Grants, view.Plan.WorkUnits); err != nil {
		t.Fatalf("compiled grants: %v", err)
	}
	if router.preference == nil || router.preference.Provider != string(domain.HarnessClaudeCode) || router.preference.Model != "sonnet-test" {
		t.Fatalf("preference = %+v", router.preference)
	}
	if router.calls != 2 {
		t.Fatalf("routing calls = %d, want 2", router.calls)
	}
	var runs []domain.IntelligenceRun
	for _, run := range store.runs {
		if run.Kind == domain.IntelligenceRunPlanDraft {
			runs = append(runs, run)
		}
	}
	if len(runs) != 1 || runs[0].Status != domain.IntelligenceRunFulfilled || runs[0].OutcomeID != outcomeID || runs[0].ContractRevisionID.IsZero() {
		t.Fatalf("plan runs = %+v", runs)
	}
}

func TestPlanCompilerRejectsWorkUnitOutsideContractCeiling(t *testing.T) {
	router := &routingInventoryFake{candidates: []domain.RoutingCandidate{readyClaudeCandidate()}}
	svc, store, outcomeID, _ := newPlanningTestService(t, router)
	store.planFakeStore.mu.Lock()
	revs := store.revs[outcomeID]
	revs[len(revs)-1].AuthorityCeiling = domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true}
	store.revs[outcomeID] = revs
	store.planFakeStore.mu.Unlock()

	_, err := svc.ProposePlan(context.Background(), outcomeID, 2)
	if err == nil {
		t.Fatal("execute intent outside Contract ceiling must fail closed")
	}
	if code := apiCode(t, err); code != "PLAN_AUTHORITY_INSUFFICIENT" {
		t.Fatalf("code = %s, want PLAN_AUTHORITY_INSUFFICIENT", code)
	}
	if got := len(store.plans[outcomeID]); got != 0 {
		t.Fatalf("authority refusal persisted %d plans", got)
	}
}

func TestPlanCompilerDoesNotWidenMissingNewWorkAuthority(t *testing.T) {
	router := &routingInventoryFake{candidates: []domain.RoutingCandidate{readyClaudeCandidate()}}
	svc, store, outcomeID, _ := newPlanningTestService(t, router)
	store.planFakeStore.mu.Lock()
	revs := store.revs[outcomeID]
	revs[len(revs)-1].AuthorityCeiling = domain.ProposedAuthority{}
	store.revs[outcomeID] = revs
	store.planFakeStore.mu.Unlock()

	_, err := svc.ProposePlan(context.Background(), outcomeID, 2)
	if err == nil {
		t.Fatal("missing new-work authority must not become read+write+exec")
	}
	if code := apiCode(t, err); code != "PLAN_AUTHORITY_REQUIRED" {
		t.Fatalf("code = %s, want PLAN_AUTHORITY_REQUIRED", code)
	}
	if router.calls != 0 {
		t.Fatalf("routing ran %d times after authority refusal", router.calls)
	}
}

func TestApprovePlanDoesNotRereadMutableProjectPreference(t *testing.T) {
	router := &routingInventoryFake{candidates: []domain.RoutingCandidate{readyClaudeCandidate()}}
	svc, store, outcomeID, _ := newPlanningTestService(t, router)
	proposal, err := svc.ProposePlan(context.Background(), outcomeID, 2)
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	reads := store.projectReads
	store.project.Config.Worker = domain.RoleOverride{Harness: domain.HarnessCodex, AgentConfig: domain.AgentConfig{Model: "changed-after-proposal"}}
	approved, err := svc.ApprovePlan(context.Background(), outcomeID, outcome.ApprovePlanInput{PlanRevisionID: proposal.Plan.ID, ExpectedContractRevision: 2})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if store.projectReads != reads {
		t.Fatalf("approval reread Project: %d -> %d", reads, store.projectReads)
	}
	for _, unit := range approved.Plan.WorkUnits {
		if unit.Provider != domain.HarnessClaudeCode || unit.Model != "sonnet-test" {
			t.Fatalf("approved binding changed: %+v", unit)
		}
	}
}

func TestProposePlanNoValidRoutePersistsNothing(t *testing.T) {
	router := &routingInventoryFake{candidates: []domain.RoutingCandidate{{
		ID: "claude-code", Provider: "claude-code", WorkerEligible: true,
		ModelSelection: domain.ExecutionBindingModelProviderDefault, Readiness: domain.CapabilityUnsupported,
		Capabilities: map[string]domain.CapabilitySupport{}, Models: map[string]domain.CapabilitySupport{},
	}}}
	svc, store, outcomeID, _ := newPlanningTestService(t, router)
	_, err := svc.ProposePlan(context.Background(), outcomeID, 2)
	if err == nil {
		t.Fatal("no admissible route must fail closed")
	}
	if code := apiCode(t, err); code != "PLAN_NO_VALID_ROUTE" {
		t.Fatalf("code = %s", code)
	}
	if persisted := len(store.plans[outcomeID]); persisted != 0 {
		t.Fatalf("persisted %d plans", persisted)
	}
}

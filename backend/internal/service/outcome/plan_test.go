package outcome_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

// planFakeStore keeps immutable Plan revisions and their unit/grant payloads so
// service tests exercise numbering, replay and approval CAS semantics without
// depending on SQLite.
type planFakeStore struct {
	*fakeStore

	mu     sync.Mutex
	plans  map[domain.OutcomeID][]domain.PlanRevision
	units  map[domain.PlanRevisionID][]domain.WorkUnit
	grants map[domain.PlanRevisionID][]domain.CapabilityGrant
}

func newPlanFakeStore() *planFakeStore {
	return &planFakeStore{
		fakeStore: &fakeStore{
			spaces:   map[domain.ProjectID]domain.ResponsibilitySpace{},
			outcomes: map[domain.OutcomeID]domain.Outcome{},
			revs:     map[domain.OutcomeID][]domain.ContractRevision{},
			keys:     map[string]domain.OutcomeID{},
		},
		plans:  map[domain.OutcomeID][]domain.PlanRevision{},
		units:  map[domain.PlanRevisionID][]domain.WorkUnit{},
		grants: map[domain.PlanRevisionID][]domain.CapabilityGrant{},
	}
}

func (f *planFakeStore) AppendPlanRevision(_ context.Context, outcomeID domain.OutcomeID, plan domain.PlanRevision) (domain.PlanRevision, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	plan.Number = int64(len(f.plans[outcomeID]) + 1)
	if err := plan.Validate(); err != nil {
		return domain.PlanRevision{}, err
	}
	f.plans[outcomeID] = append(f.plans[outcomeID], plan)
	f.units[plan.ID] = append([]domain.WorkUnit(nil), plan.WorkUnits...)
	f.grants[plan.ID] = append([]domain.CapabilityGrant(nil), plan.Grants...)
	return plan, nil
}

func (f *planFakeStore) LatestProposedPlanRevision(_ context.Context, outcomeID domain.OutcomeID, contractRevision int64) (domain.PlanRevision, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var best domain.PlanRevision
	found := false
	for _, plan := range f.plans[outcomeID] {
		if plan.Status == domain.PlanStatusProposed && plan.ContractRevisionNumber == contractRevision {
			if !found || plan.Number > best.Number {
				best, found = plan, true
			}
		}
	}
	return best, found, nil
}

func (f *planFakeStore) GetPlanRevision(_ context.Context, outcomeID domain.OutcomeID, planID domain.PlanRevisionID) (domain.PlanRevision, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, plan := range f.plans[outcomeID] {
		if plan.ID == planID {
			out := plan
			out.WorkUnits, out.Grants = append([]domain.WorkUnit(nil), f.units[planID]...), append([]domain.CapabilityGrant(nil), f.grants[planID]...)
			return out, true, nil
		}
	}
	return domain.PlanRevision{}, false, nil
}

func (f *planFakeStore) GetLatestPlanRevision(_ context.Context, outcomeID domain.OutcomeID) (domain.PlanRevision, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := len(f.plans[outcomeID])
	if n == 0 {
		return domain.PlanRevision{}, false, nil
	}
	plan := f.plans[outcomeID][n-1]
	plan.WorkUnits, plan.Grants = append([]domain.WorkUnit(nil), f.units[plan.ID]...), append([]domain.CapabilityGrant(nil), f.grants[plan.ID]...)
	return plan, true, nil
}

func (f *planFakeStore) ApprovePlanRevision(_ context.Context, outcomeID domain.OutcomeID, planID domain.PlanRevisionID) (domain.PlanRevision, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, plan := range f.plans[outcomeID] {
		if plan.ID != planID {
			continue
		}
		if plan.Status == domain.PlanStatusApproved {
			out := plan
			out.WorkUnits, out.Grants = append([]domain.WorkUnit(nil), f.units[planID]...), append([]domain.CapabilityGrant(nil), f.grants[planID]...)
			return out, true, nil
		}
		f.plans[outcomeID][i].Status = domain.PlanStatusApproved
		out := f.plans[outcomeID][i]
		out.WorkUnits, out.Grants = append([]domain.WorkUnit(nil), f.units[planID]...), append([]domain.CapabilityGrant(nil), f.grants[planID]...)
		return out, true, nil
	}
	return domain.PlanRevision{}, false, nil
}

func apiCode(t *testing.T, err error) string {
	t.Helper()
	var apiErr *apierr.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierr, got %T: %v", err, err)
	}
	return apiErr.Code
}

func TestProposePlanReentryIsIdempotent(t *testing.T) {
	router := &routingInventoryFake{candidates: []domain.RoutingCandidate{readyClaudeCandidate()}}
	svc, store, outcomeID, provider := newPlanningTestService(t, router)

	first, err := svc.ProposePlan(context.Background(), outcomeID, 2)
	if err != nil {
		t.Fatalf("first propose: %v", err)
	}
	second, err := svc.ProposePlan(context.Background(), outcomeID, 2)
	if err != nil {
		t.Fatalf("re-entry propose: %v", err)
	}
	if second.Plan.ID != first.Plan.ID || second.Plan.RunBriefCoreDigest != first.Plan.RunBriefCoreDigest {
		t.Fatal("ordinary re-entry must return the immutable existing proposal")
	}
	if provider.calls != 1 {
		t.Fatalf("plan intelligence calls = %d, want 1", provider.calls)
	}
	if got := len(store.plans[outcomeID]); got != 1 {
		t.Fatalf("persisted plans = %d, want 1", got)
	}
}

func TestProposePlanRejectsStaleContractPointer(t *testing.T) {
	router := &routingInventoryFake{candidates: []domain.RoutingCandidate{readyClaudeCandidate()}}
	svc, _, outcomeID, _ := newPlanningTestService(t, router)
	if _, err := svc.ProposePlan(context.Background(), outcomeID, 99); err == nil {
		t.Fatal("stale pointer must be refused")
	} else if code := apiCode(t, err); code != "PLAN_CONTRACT_STALE" {
		t.Fatalf("code = %s, want PLAN_CONTRACT_STALE", code)
	}
}

func TestContractRevisionMovementInvalidatesEarlierPlan(t *testing.T) {
	router := &routingInventoryFake{candidates: []domain.RoutingCandidate{readyClaudeCandidate()}}
	svc, _, outcomeID, _ := newPlanningTestService(t, router)
	ctx := context.Background()

	first, err := svc.ProposePlan(ctx, outcomeID, 2)
	if err != nil {
		t.Fatalf("propose r2: %v", err)
	}
	if _, err := svc.ReviseContract(ctx, outcomeID, outcome.ReviseContractInput{
		ExpectedRevision: 2,
		Goal:             "Ship and verify the bounded change with an owner note.",
		SuccessCriteria: []string{
			"Implementation is present.",
			"Verification proves the implementation behaves as required.",
		},
		Review:           "Run deterministic verification and inspect the owner note.",
		AuthorityCeiling: fullLocalAuthority(),
		StopConditions:   []string{"Stop before remote effects."},
	}); err != nil {
		t.Fatalf("revise contract: %v", err)
	}

	if _, err := svc.ApprovePlan(ctx, outcomeID, outcome.ApprovePlanInput{PlanRevisionID: first.Plan.ID, ExpectedContractRevision: 3}); err == nil {
		t.Fatal("plan bound to superseded Contract must not approve")
	} else if code := apiCode(t, err); code != "PLAN_CONTRACT_STALE" {
		t.Fatalf("code = %s, want PLAN_CONTRACT_STALE", code)
	}

	next, err := svc.ProposePlan(ctx, outcomeID, 3)
	if err != nil {
		t.Fatalf("propose r3: %v", err)
	}
	if next.Plan.ContractRevisionNumber != 3 || next.Plan.RunBriefCoreDigest == first.Plan.RunBriefCoreDigest {
		t.Fatal("material Contract revision must produce a fresh frozen Plan")
	}
	approved, err := svc.ApprovePlan(ctx, outcomeID, outcome.ApprovePlanInput{PlanRevisionID: next.Plan.ID, ExpectedContractRevision: 3})
	if err != nil {
		t.Fatalf("approve r3: %v", err)
	}
	if approved.Plan.Status != domain.PlanStatusApproved {
		t.Fatalf("status = %s, want approved", approved.Plan.Status)
	}
}

func TestProposeFailsClosedWhenDaemonPolicyNarrows(t *testing.T) {
	router := &routingInventoryFake{candidates: []domain.RoutingCandidate{readyClaudeCandidate()}}
	svc, store, outcomeID, _ := newPlanningTestService(t, router)
	svc.PolicyLayers = [][]string{{domain.CapabilityWorktreeRead}}

	if _, err := svc.ProposePlan(context.Background(), outcomeID, 2); err == nil {
		t.Fatal("narrowed daemon authority must fail proposal closed")
	} else if code := apiCode(t, err); code != "PLAN_CAPABILITY_UNAUTHORIZED" {
		t.Fatalf("code = %s, want PLAN_CAPABILITY_UNAUTHORIZED", code)
	}
	if got := len(store.plans[outcomeID]); got != 0 {
		t.Fatalf("refused proposal persisted %d plans", got)
	}
}

func TestLowerPolicyLayerCannotWidenAuthority(t *testing.T) {
	router := &routingInventoryFake{candidates: []domain.RoutingCandidate{readyClaudeCandidate()}}
	svc, store, outcomeID, _ := newPlanningTestService(t, router)
	svc.PolicyLayers = [][]string{
		{domain.CapabilityWorktreeRead},
		{domain.CapabilityWorktreeRead, domain.CapabilityWorktreeWrite, domain.CapabilityWorktreeExec, "network.fetch"},
	}
	if _, err := svc.ProposePlan(context.Background(), outcomeID, 2); err == nil {
		t.Fatal("lower layer widening must fail closed")
	} else if code := apiCode(t, err); code != "PLAN_CAPABILITY_UNAUTHORIZED" {
		t.Fatalf("code = %s, want PLAN_CAPABILITY_UNAUTHORIZED", code)
	}
	if got := len(store.plans[outcomeID]); got != 0 {
		t.Fatalf("refused proposal persisted %d plans", got)
	}
}

func TestApproveRechecksCurrentPolicyWithoutRerouting(t *testing.T) {
	router := &routingInventoryFake{candidates: []domain.RoutingCandidate{readyClaudeCandidate()}}
	svc, _, outcomeID, _ := newPlanningTestService(t, router)
	view, err := svc.ProposePlan(context.Background(), outcomeID, 2)
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	svc.PolicyLayers = [][]string{{domain.CapabilityWorktreeRead}}
	if _, err := svc.ApprovePlan(context.Background(), outcomeID, outcome.ApprovePlanInput{PlanRevisionID: view.Plan.ID, ExpectedContractRevision: 2}); err == nil {
		t.Fatal("approval must recheck the current authority ceiling")
	} else if code := apiCode(t, err); code != "PLAN_CAPABILITY_UNAUTHORIZED" {
		t.Fatalf("code = %s, want PLAN_CAPABILITY_UNAUTHORIZED", code)
	}
}

func TestApproveUnknownPlanIsNotFound(t *testing.T) {
	router := &routingInventoryFake{candidates: []domain.RoutingCandidate{readyClaudeCandidate()}}
	svc, _, outcomeID, _ := newPlanningTestService(t, router)
	if _, err := svc.ProposePlan(context.Background(), outcomeID, 2); err != nil {
		t.Fatalf("propose: %v", err)
	}
	if _, err := svc.ApprovePlan(context.Background(), outcomeID, outcome.ApprovePlanInput{PlanRevisionID: "plan-missing", ExpectedContractRevision: 2}); err == nil {
		t.Fatal("unknown plan must 404")
	} else if code := apiCode(t, err); code != "PLAN_NOT_FOUND" {
		t.Fatalf("code = %s, want PLAN_NOT_FOUND", code)
	}
	if _, err := svc.GetLatestPlan(context.Background(), "out-ghost"); err == nil {
		t.Fatal("plans for unknown outcomes must 404")
	}
}

package outcome_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/service/outcome"
)

// planFakeStore extends the contract fake from contract_test.go with a real
// in-memory plan store so policy tests exercise actual numbering, replay
// lookup, and CAS semantics rather than stubs.
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
	f.units[plan.ID] = plan.WorkUnits
	f.grants[plan.ID] = plan.Grants
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
			out.WorkUnits, out.Grants = f.units[planID], f.grants[planID]
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
	plan.WorkUnits, plan.Grants = f.units[plan.ID], f.grants[plan.ID]
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
			out.WorkUnits, out.Grants = f.units[planID], f.grants[planID]
			return out, true, nil
		}
		f.plans[outcomeID][i].Status = domain.PlanStatusApproved
		out := f.plans[outcomeID][i]
		out.WorkUnits, out.Grants = f.units[planID], f.grants[planID]
		return out, true, nil
	}
	return domain.PlanRevision{}, false, nil
}

func newPlanTestService(t *testing.T) (*outcome.Service, *planFakeStore, domain.OutcomeID) {
	t.Helper()
	store := newPlanFakeStore()
	svc := outcome.New(store, nil)

	store.mu.Lock()
	store.fakeStore.spaces["mer"] = domain.ResponsibilitySpace{
		ID: "rsp-plan", Kind: domain.ResponsibilitySpaceWorkProject, ProjectID: "mer",
	}
	store.mu.Unlock()

	ctx := context.Background()
	view, err := svc.Create(ctx, outcome.CreateInput{
		ProjectID:       "mer",
		Title:           "Local Focus Ledger",
		Goal:            "Record today's protected focus time locally.",
		SuccessCriteria: []string{"Entering positive whole minutes creates one focus block."},
		Review:          "Deterministic checks plus owner walkthrough.",
		Constraints:     []string{"Local only."},
		RequestKey:      "req-plan-test",
	})
	if err != nil {
		t.Fatalf("seed outcome: %v", err)
	}
	return svc, store, view.Outcome.ID
}

func apiCode(t *testing.T, err error) string {
	t.Helper()
	var apiErr *apierr.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierr, got %T: %v", err, err)
	}
	return apiErr.Code
}

func TestProposePlanIsDeterministicAndReplays(t *testing.T) {
	svc, _, outcomeID := newPlanTestService(t)
	ctx := context.Background()

	first, err := svc.ProposePlan(ctx, outcomeID, 1)
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if first.Plan.Status != domain.PlanStatusProposed || first.Plan.Number != 1 {
		t.Fatalf("first proposal = #%d %s", first.Plan.Number, first.Plan.Status)
	}
	if len(first.Plan.WorkUnits) != 1 || first.Plan.WorkUnits[0].Kind != domain.WorkUnitDirect {
		t.Fatalf("v0 proposal must be exactly one direct unit: %+v", first.Plan.WorkUnits)
	}
	if len(first.Plan.Grants) != len(domain.V0RequiredCapabilities) {
		t.Fatalf("grants = %+v, want the v0 trio", first.Plan.Grants)
	}
	// Evidence binds criteria; verification mirrors review.
	if got := strings.Join(first.Plan.WorkUnits[0].EvidenceChecks, "|"); !strings.Contains(got, "focus block") {
		t.Fatalf("evidence checks must bind success criteria, got %q", got)
	}
	if first.Plan.WorkUnits[0].VerificationRequirement == "" {
		t.Fatal("verification requirement must mirror the contract review")
	}

	second, err := svc.ProposePlan(ctx, outcomeID, 1)
	if err != nil {
		t.Fatalf("re-propose: %v", err)
	}
	if second.Plan.ID != first.Plan.ID || second.Plan.RunBriefCoreDigest != first.Plan.RunBriefCoreDigest {
		t.Fatal("re-proposing one revision must replay, not stack duplicates")
	}
}

func TestProposePlanRejectsStaleContractPointer(t *testing.T) {
	svc, _, outcomeID := newPlanTestService(t)
	ctx := context.Background()

	view, err := svc.Get(ctx, outcomeID)
	if err != nil {
		t.Fatalf("get outcome: %v", err)
	}
	if _, err := svc.ProposePlan(ctx, outcomeID, view.Outcome.CurrentRevisionNumber+5); err == nil {
		t.Fatal("stale pointer must be refused")
	} else if code := apiCode(t, err); code != "PLAN_CONTRACT_STALE" {
		t.Fatalf("code = %s, want PLAN_CONTRACT_STALE", code)
	}
}

func TestMaterialChangeForcesFreshBriefBeforeApproval(t *testing.T) {
	svc, _, outcomeID := newPlanTestService(t)
	ctx := context.Background()

	firstView, err := svc.ProposePlan(ctx, outcomeID, 1)
	if err != nil {
		t.Fatalf("propose r1 plan: %v", err)
	}

	// Owner edits the contract: r2 supersedes every plan bound to r1.
	if _, err := svc.ReviseContract(ctx, outcomeID, outcome.ReviseContractInput{
		ExpectedRevision: 1,
		Goal:             "Record today's protected focus time locally, with notes.",
		SuccessCriteria: []string{
			"Entering positive whole minutes creates one focus block.",
			"Notes are retained.",
		},
		Review: "Deterministic checks plus owner walkthrough.",
	}); err != nil {
		t.Fatalf("revise contract: %v", err)
	}

	_, err = svc.ApprovePlan(ctx, outcomeID, outcome.ApprovePlanInput{
		PlanRevisionID:           firstView.Plan.ID,
		ExpectedContractRevision: 2,
	})
	if err == nil {
		t.Fatal("approving a plan bound to superseded r1 must be refused")
	}
	if code := apiCode(t, err); code != "PLAN_CONTRACT_STALE" {
		t.Fatalf("code = %s, want PLAN_CONTRACT_STALE", code)
	}

	next, err := svc.ProposePlan(ctx, outcomeID, 2)
	if err != nil {
		t.Fatalf("propose r2 plan: %v", err)
	}
	if next.Plan.ContractRevisionNumber != 2 || next.Plan.RunBriefCoreDigest == firstView.Plan.RunBriefCoreDigest {
		t.Fatal("the r2 proposal must carry a fresh RunBrief digest")
	}

	auth, err := svc.ApprovePlan(ctx, outcomeID, outcome.ApprovePlanInput{
		PlanRevisionID:           next.Plan.ID,
		ExpectedContractRevision: 2,
	})
	if err != nil {
		t.Fatalf("approve r2 plan: %v", err)
	}
	if auth.Plan.Status != domain.PlanStatusApproved {
		t.Fatalf("status = %s, want approved", auth.Plan.Status)
	}

	latest, err := svc.GetLatestPlan(ctx, outcomeID)
	if err != nil || latest.Plan.ID != next.Plan.ID {
		t.Fatalf("latest plan = %+v err=%v, want the approved r2 plan", latest, err)
	}
}

func TestProposeFailsClosedWhenAuthorityNarrows(t *testing.T) {
	svc, store, outcomeID := newPlanTestService(t)
	ctx := context.Background()

	// A constrained environment offers only read: the full v0 trio cannot be
	// granted, so proposing aborts instead of persisting a hollow plan.
	svc.PolicyLayers = [][]string{{domain.CapabilityWorktreeRead}}
	if _, err := svc.ProposePlan(ctx, outcomeID, 1); err == nil {
		t.Fatal("narrowed authority must fail proposal closed")
	} else if code := apiCode(t, err); code != "PLAN_CAPABILITY_UNAUTHORIZED" {
		t.Fatalf("code = %s, want PLAN_CAPABILITY_UNAUTHORIZED", code)
	}

	store.mu.Lock()
	persisted := len(store.plans[outcomeID])
	store.mu.Unlock()
	if persisted != 0 {
		t.Fatalf("a refused proposal persisted %d plans; want none", persisted)
	}
}

func TestLowerLayerCannotWidenAuthority(t *testing.T) {
	svc, store, outcomeID := newPlanTestService(t)
	ctx := context.Background()

	// The runtime layer advertises network.fetch, but the upper policy ceiling
	// does not. The intersection excludes everything beyond read, so the v0
	// trio fails closed and no widening survives.
	svc.PolicyLayers = [][]string{
		{domain.CapabilityWorktreeRead},
		{domain.CapabilityWorktreeRead, domain.CapabilityWorktreeWrite, domain.CapabilityWorktreeExec, "network.fetch"},
	}
	if _, err := svc.ProposePlan(ctx, outcomeID, 1); err == nil {
		t.Fatal("widening attempt must fail closed")
	} else if code := apiCode(t, err); code != "PLAN_CAPABILITY_UNAUTHORIZED" {
		t.Fatalf("code = %s, want PLAN_CAPABILITY_UNAUTHORIZED", code)
	}

	store.mu.Lock()
	persisted := len(store.plans[outcomeID])
	store.mu.Unlock()
	if persisted != 0 {
		t.Fatal("no plan may persist when authority is exceeded")
	}
}

func TestApproveRechecksAuthorityAtAuthorizationTime(t *testing.T) {
	svc, _, outcomeID := newPlanTestService(t)
	ctx := context.Background()

	view, err := svc.ProposePlan(ctx, outcomeID, 1)
	if err != nil {
		t.Fatalf("propose under full authority: %v", err)
	}

	// Policy narrows after the proposal was recorded.
	svc.PolicyLayers = [][]string{{domain.CapabilityWorktreeRead}}
	_, err = svc.ApprovePlan(ctx, outcomeID, outcome.ApprovePlanInput{
		PlanRevisionID:           view.Plan.ID,
		ExpectedContractRevision: 1,
	})
	if err == nil {
		t.Fatal("approval must re-run fail-closed checks against current authority")
	}
	if code := apiCode(t, err); code != "PLAN_CAPABILITY_UNAUTHORIZED" {
		t.Fatalf("code = %s, want PLAN_CAPABILITY_UNAUTHORIZED", code)
	}
}

func TestApproveUnknownPlanIsNotFound(t *testing.T) {
	svc, _, outcomeID := newPlanTestService(t)
	ctx := context.Background()

	if _, err := svc.ProposePlan(ctx, outcomeID, 1); err != nil {
		t.Fatalf("propose: %v", err)
	}
	if _, err := svc.ApprovePlan(ctx, outcomeID, outcome.ApprovePlanInput{
		PlanRevisionID:           "plan-missing",
		ExpectedContractRevision: 1,
	}); err == nil {
		t.Fatal("unknown plan must 404")
	} else if code := apiCode(t, err); code != "PLAN_NOT_FOUND" {
		t.Fatalf("code = %s, want PLAN_NOT_FOUND", code)
	}
	if _, err := svc.GetLatestPlan(ctx, "out-ghost"); err == nil {
		t.Fatal("plans for unknown outcomes must 404")
	}
}

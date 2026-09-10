package store_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

// seedPlanWithApprovedChecks builds an approved Plan whose single WorkUnit
// carries two deterministic checks, in a deliberate order.
func seedPlanWithApprovedChecks(t *testing.T, s *sqlite.Store, projectID string) (domain.PlanRevision, domain.OutcomeID) {
	t.Helper()
	ctx := context.Background()
	seedProject(t, s, projectID)
	space, err := s.EnsureWorkResponsibilitySpace(ctx, domain.ProjectID(projectID))
	if err != nil {
		t.Fatalf("ensure space: %v", err)
	}
	outcomeID := domain.OutcomeID("out-" + projectID)
	criterionID := domain.CriterionID("crit-" + projectID)
	contract := domain.ContractRevision{
		ID: domain.ContractRevisionID("cr-" + projectID), OutcomeID: outcomeID, Number: 1,
		Goal: "Record focus locally.", SuccessCriteria: []string{"Blocks are recorded."},
		Review:   "Deterministic checks.",
		Criteria: []domain.ContractCriterion{{ID: criterionID, ContractRevisionID: domain.ContractRevisionID("cr-" + projectID), Position: 1, Text: "Blocks are recorded."}},
	}
	if err := s.CreateOutcomeWithContract(ctx, domain.Outcome{
		ID: outcomeID, SpaceID: space.ID, Title: "Local Focus Ledger",
	}, contract, "rk-create-"+projectID); err != nil {
		t.Fatalf("create outcome: %v", err)
	}
	unit := domain.WorkUnit{
		ID: domain.WorkUnitID("wu-" + projectID), Kind: domain.WorkUnitDirect, Title: "Deliver it",
		ContractRevisionNumber: 1, OutputSummary: "Working local feature.",
		EvidenceChecks: []string{"checks pass"}, VerificationRequirement: "Deterministic checks.",
		StopConditions: []string{"stop before remote effects"},
		Provider:       domain.HarnessCodex, ModelSelection: domain.ExecutionBindingModelProviderDefault,
		CriterionIDs: []domain.CriterionID{criterionID},
		RequiredCapabilities: []string{
			domain.CapabilityWorktreeRead, domain.CapabilityWorktreeWrite, domain.CapabilityWorktreeExec,
		},
		Checks: []domain.ApprovedCheck{
			{ID: domain.ApprovedCheckID("chk-build-" + projectID), CriterionID: criterionID, Argv: []string{"go", "build", "./..."}, TimeoutSeconds: 120},
			{ID: domain.ApprovedCheckID("chk-test-" + projectID), CriterionID: criterionID, Argv: []string{"go", "test", "./..."}, TimeoutSeconds: 300},
		},
	}
	grants := []domain.CapabilityGrant{
		{ID: domain.CapabilityGrantID("cg-read-" + projectID), Name: domain.CapabilityWorktreeRead, Scope: "worktree/*"},
		{ID: domain.CapabilityGrantID("cg-write-" + projectID), Name: domain.CapabilityWorktreeWrite, Scope: "worktree/*"},
		{ID: domain.CapabilityGrantID("cg-exec-" + projectID), Name: domain.CapabilityWorktreeExec, Scope: "worktree/*"},
	}
	digest, err := domain.ComputeRunBriefCoreDigest(contract, unit, grants)
	if err != nil {
		t.Fatalf("compute digest: %v", err)
	}
	plan, err := s.AppendPlanRevision(ctx, outcomeID, domain.PlanRevision{
		ID: domain.PlanRevisionID("plan-" + projectID), OutcomeID: outcomeID, ContractRevisionNumber: 1,
		Status: domain.PlanStatusProposed, Summary: "One direct Work Unit",
		WorkUnits: []domain.WorkUnit{unit}, Grants: grants,
		RoutingDecisions: []domain.WorkUnitRoutingDecision{{
			WorkUnitID: unit.ID,
			Decision: domain.RoutingDecision{
				Status: domain.RoutingDecisionRecommended, PolicyVersion: domain.RoutingPolicyVersion,
				Role: domain.RoutingRoleWorker, RecommendedCandidateID: string(domain.HarnessCodex),
				RecommendedProvider: string(domain.HarnessCodex), RecommendedModelSelection: domain.ExecutionBindingModelProviderDefault,
			},
		}},
		RunBriefCoreDigest: digest,
	})
	if err != nil {
		t.Fatalf("append plan: %v", err)
	}
	approved, found, err := s.ApprovePlanRevision(ctx, outcomeID, plan.ID)
	if err != nil || !found {
		t.Fatalf("approve plan: found=%v err=%v", found, err)
	}
	return approved, outcomeID
}

// TestApprovedChecksRoundTripInApprovedOrder pins the two facts execution
// depends on: the commands survive readback exactly, and they come back in the
// order they were approved in. A build check that runs after the test that
// depends on it is a different check set than the one the owner reviewed.
func TestApprovedChecksRoundTripInApprovedOrder(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	plan, outcomeID := seedPlanWithApprovedChecks(t, s, "checksroundtrip")

	stored, found, err := s.GetPlanRevision(ctx, outcomeID, plan.ID)
	if err != nil || !found {
		t.Fatalf("read plan: found=%v err=%v", found, err)
	}
	checks := stored.WorkUnits[0].Checks
	if len(checks) != 2 {
		t.Fatalf("checks = %d, want 2", len(checks))
	}
	if got := strings.Join(checks[0].Argv, " "); got != "go build ./..." {
		t.Fatalf("first check = %q, want the approved order", got)
	}
	if got := strings.Join(checks[1].Argv, " "); got != "go test ./..." {
		t.Fatalf("second check = %q", got)
	}
	if checks[0].CriterionID != stored.WorkUnits[0].CriterionIDs[0] {
		t.Fatalf("criterion binding lost: %+v", checks[0])
	}
	if checks[0].TimeoutSeconds != 120 || checks[1].TimeoutSeconds != 300 {
		t.Fatalf("timeouts lost: %d/%d", checks[0].TimeoutSeconds, checks[1].TimeoutSeconds)
	}
}

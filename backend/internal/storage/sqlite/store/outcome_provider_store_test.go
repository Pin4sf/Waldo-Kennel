package store_test

import (
	"context"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
)

func providerBoundPlanFixture(t *testing.T, provider domain.AgentHarness) (*domain.PlanRevision, domain.ContractRevision, *domain.WorkUnit, []domain.CapabilityGrant) {
	t.Helper()
	revision := domain.ContractRevision{
		ID:              domain.ContractRevisionID("cr-provider-plan"),
		OutcomeID:       domain.OutcomeID("out-provider-plan"),
		Number:          1,
		Goal:            "Deliver the provider-bound plan.",
		SuccessCriteria: []string{"the selected provider is durable"},
		Review:          "deterministic tests",
	}
	unit := domain.WorkUnit{
		ID:                      domain.WorkUnitID("wu-provider-plan"),
		Kind:                    domain.WorkUnitDirect,
		Title:                   "Deliver provider-bound plan",
		ContractRevisionNumber:  1,
		Provider:                provider,
		OutputSummary:           "A tested provider-bound result.",
		EvidenceChecks:          []string{"provider binding round-trips"},
		VerificationRequirement: "deterministic tests",
		StopConditions:          []string{"stop before remote effects"},
	}
	grants := []domain.CapabilityGrant{
		{ID: domain.CapabilityGrantID("cg-provider-read"), Name: domain.CapabilityWorktreeRead, Scope: "worktree/*"},
		{ID: domain.CapabilityGrantID("cg-provider-write"), Name: domain.CapabilityWorktreeWrite, Scope: "worktree/*"},
		{ID: domain.CapabilityGrantID("cg-provider-exec"), Name: domain.CapabilityWorktreeExec, Scope: "worktree/*"},
	}
	digest, err := domain.ComputeRunBriefCoreDigest(revision, unit, grants)
	if err != nil {
		t.Fatalf("compute plan digest: %v", err)
	}
	plan := &domain.PlanRevision{
		ID:                     domain.PlanRevisionID("plan-provider-bound"),
		OutcomeID:              revision.OutcomeID,
		ContractRevisionNumber: 1,
		Status:                 domain.PlanStatusProposed,
		Summary:                "One provider-bound unit.",
		WorkUnits:              []domain.WorkUnit{unit},
		Grants:                 grants,
		RunBriefCoreDigest:     digest,
	}
	return plan, revision, &unit, grants
}

func seedProviderPlanOutcome(t *testing.T, s *sqlite.Store) {
	t.Helper()
	ctx := context.Background()
	seedProject(t, s, "provider-project")
	space, err := s.EnsureWorkResponsibilitySpace(ctx, "provider-project")
	if err != nil {
		t.Fatalf("ensure provider project space: %v", err)
	}
	outcome, revision := focusLedgerContract(space.ID, "provider-plan")
	outcome.ID = "out-provider-plan"
	revision.ID = "cr-provider-plan"
	revision.OutcomeID = outcome.ID
	if err := s.CreateOutcomeWithContract(ctx, outcome, revision, "req-provider-plan"); err != nil {
		t.Fatalf("create provider plan outcome: %v", err)
	}
}

func TestOutcomeStore_WorkUnitProviderBindingRoundTrips(t *testing.T) {
	s := newTestStore(t)
	seedProviderPlanOutcome(t, s)
	plan, _, _, _ := providerBoundPlanFixture(t, domain.HarnessClaudeCode)

	saved, err := s.AppendPlanRevisionWithProvider(context.Background(), plan.OutcomeID, *plan)
	if err != nil {
		t.Fatalf("append provider-bound plan: %v", err)
	}
	if saved.WorkUnits[0].Provider != domain.HarnessClaudeCode {
		t.Fatalf("saved provider = %q, want %q", saved.WorkUnits[0].Provider, domain.HarnessClaudeCode)
	}
	got, ok, err := s.GetWorkUnitProvider(context.Background(), saved.WorkUnits[0].ID)
	if err != nil || !ok {
		t.Fatalf("get work unit provider ok=%v err=%v", ok, err)
	}
	if got != domain.HarnessClaudeCode {
		t.Fatalf("provider binding = %q, want %q", got, domain.HarnessClaudeCode)
	}
}

func TestOutcomeStore_LegacyWorkUnitWithoutProviderHasNoInventedBinding(t *testing.T) {
	s := newTestStore(t)
	seedProviderPlanOutcome(t, s)
	plan, _, _, _ := providerBoundPlanFixture(t, "")
	plan.ID = "plan-provider-legacy"
	plan.WorkUnits[0].ID = "wu-provider-legacy"
	plan.Grants[0].ID = "cg-provider-legacy-read"
	plan.Grants[1].ID = "cg-provider-legacy-write"
	plan.Grants[2].ID = "cg-provider-legacy-exec"
	digest, err := domain.ComputeRunBriefCoreDigest(domain.ContractRevision{
		ID:              "cr-provider-plan",
		OutcomeID:       "out-provider-plan",
		Number:          1,
		Goal:            "Deliver the provider-bound plan.",
		SuccessCriteria: []string{"the selected provider is durable"},
		Review:          "deterministic tests",
	}, plan.WorkUnits[0], plan.Grants)
	if err != nil {
		t.Fatalf("compute legacy digest: %v", err)
	}
	plan.RunBriefCoreDigest = digest

	if _, err := s.AppendPlanRevision(context.Background(), plan.OutcomeID, *plan); err != nil {
		t.Fatalf("append legacy provider-less plan: %v", err)
	}
	got, ok, err := s.GetWorkUnitProvider(context.Background(), plan.WorkUnits[0].ID)
	if err != nil {
		t.Fatalf("get legacy binding: %v", err)
	}
	if ok || got != "" {
		t.Fatalf("legacy provider binding = %q ok=%v, want empty/false", got, ok)
	}
}

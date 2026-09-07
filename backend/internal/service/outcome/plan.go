package outcome

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
)

// PlanView is the read model over one plan revision together with the Outcome
// it plans for. Stage presentation stays with callers.
type PlanView struct {
	Outcome domain.Outcome
	Plan    domain.PlanRevision
}

// AuthorizedPlanView marks the plan returned after owner approval; the plan's
// status is approved and its capability grants are active.
type AuthorizedPlanView struct {
	Outcome domain.Outcome
	Plan    domain.PlanRevision
}

// ApprovePlanInput names the plan to authorize and the contract revision the
// approver was looking at, so a race between reading and approving surfaces
// as an explicit stale-conflict instead of a silent authority transfer.
type ApprovePlanInput struct {
	PlanRevisionID           domain.PlanRevisionID
	ExpectedContractRevision int64
}

func v0StopConditionList() []string {
	return []string{
		"Stop before an unapproved dependency",
		"Stop before any remote effect (network, push, PR, deploy)",
		"Stop before writes outside the isolated worktree",
		"Stop on contradictory Project policy",
	}
}

// ProposePlan builds the v0 deterministic proposal for the Outcome's current
// contract: one smallest-sufficient direct Work Unit whose evidence binds the
// contract's success criteria, the Project's explicit worker provider, plus the
// worktree-local capability trio, frozen by the RunBrief core digest.
// Re-proposing replays only when both contract revision and provider binding
// still match; changing the Project worker requires a fresh authorization.
func (s *Service) ProposePlan(ctx context.Context, outcomeID domain.OutcomeID, expectedContractRevision int64) (PlanView, error) {
	outcome, ok, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return PlanView{}, err
	}
	if !ok {
		return PlanView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	if expectedContractRevision < 1 {
		return PlanView{}, apierr.Invalid("EXPECTED_REVISION_REQUIRED", "State which contract revision the plan executes", nil)
	}
	if outcome.CurrentRevisionNumber != expectedContractRevision {
		return PlanView{}, apierr.New(apierr.KindConflict, "PLAN_CONTRACT_STALE",
			fmt.Sprintf("Plan must bind revision %s; reload the Outcome",
				formatI64(outcome.CurrentRevisionNumber)),
			map[string]any{
				"outcomeId":        string(outcomeID),
				"expectedRevision": expectedContractRevision,
				"currentRevision":  outcome.CurrentRevisionNumber,
			})
	}

	revision, err := s.currentRevision(ctx, outcome)
	if err != nil {
		return PlanView{}, err
	}
	provider, err := s.projectWorkerProvider(ctx, outcomeID)
	if err != nil {
		return PlanView{}, err
	}

	if existing, found, err := s.store.LatestProposedPlanRevision(ctx, outcomeID, revision.Number); err != nil {
		return PlanView{}, err
	} else if found {
		existing, err = s.hydratePlanProvider(ctx, existing)
		if err != nil {
			return PlanView{}, err
		}
		if len(existing.WorkUnits) == 1 && existing.WorkUnits[0].Provider == provider {
			return PlanView{Outcome: outcome, Plan: existing}, nil
		}
		// Legacy provider-less proposals and proposals for a previous Project
		// worker remain immutable history. Create a fresh proposal instead.
	}

	unit := s.v0WorkUnit(outcome, revision, provider)
	grants := s.v0Grants()
	if err := s.authorizeCapabilities(grants); err != nil {
		return PlanView{}, err
	}
	digest, err := domain.ComputeRunBriefCoreDigest(revision, unit, grants)
	if err != nil {
		return PlanView{}, err
	}

	proposal := domain.PlanRevision{
		ID:                     domain.PlanRevisionID("plan-" + uuid.NewString()),
		OutcomeID:              outcomeID,
		ContractRevisionNumber: revision.Number,
		Status:                 domain.PlanStatusProposed,
		Summary:                "One direct Work Unit executing this contract locally.",
		WorkUnits:              []domain.WorkUnit{unit},
		Grants:                 grants,
		RunBriefCoreDigest:     digest,
	}
	saved, err := s.appendProviderBoundPlan(ctx, outcomeID, proposal)
	if err != nil {
		return PlanView{}, err
	}
	return PlanView{Outcome: outcome, Plan: saved}, nil
}

// ApprovePlan authorizes a proposed plan. The plan must still bind the
// Outcome's current contract revision — any material change forces a fresh
// proposal and therefore a fresh RunBrief — and every grant must still be
// allowed by all authority layers at the moment of approval. A legacy plan
// without a provider binding cannot become newly executable.
func (s *Service) ApprovePlan(ctx context.Context, outcomeID domain.OutcomeID, in ApprovePlanInput) (AuthorizedPlanView, error) {
	outcome, ok, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return AuthorizedPlanView{}, err
	}
	if !ok {
		return AuthorizedPlanView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	if in.PlanRevisionID.IsZero() {
		return AuthorizedPlanView{}, apierr.Invalid("PLAN_ID_REQUIRED", "Name the plan to authorize", nil)
	}
	if in.ExpectedContractRevision >= 1 && in.ExpectedContractRevision != outcome.CurrentRevisionNumber {
		return AuthorizedPlanView{}, apierr.New(apierr.KindConflict, "PLAN_CONTRACT_STALE",
			fmt.Sprintf("Approval was prepared against revision %s; the Outcome is at %s",
				formatI64(in.ExpectedContractRevision), formatI64(outcome.CurrentRevisionNumber)),
			map[string]any{
				"outcomeId":        string(outcomeID),
				"expectedRevision": in.ExpectedContractRevision,
				"currentRevision":  outcome.CurrentRevisionNumber,
			})
	}

	plan, found, err := s.store.GetPlanRevision(ctx, outcomeID, in.PlanRevisionID)
	if err != nil {
		return AuthorizedPlanView{}, err
	}
	if !found {
		return AuthorizedPlanView{}, apierr.NotFound("PLAN_NOT_FOUND", "That plan does not exist")
	}
	plan, err = s.hydratePlanProvider(ctx, plan)
	if err != nil {
		return AuthorizedPlanView{}, err
	}
	if len(plan.WorkUnits) != 1 || plan.WorkUnits[0].Provider == "" {
		return AuthorizedPlanView{}, providerUnboundError(outcomeID)
	}
	if !plan.BindsCurrentContract(outcome.CurrentRevisionNumber) {
		return AuthorizedPlanView{}, apierr.New(apierr.KindConflict, "PLAN_CONTRACT_STALE",
			fmt.Sprintf("Plan binds contract revision %s; the Outcome is at %s — propose a new plan",
				formatI64(plan.ContractRevisionNumber), formatI64(outcome.CurrentRevisionNumber)),
			map[string]any{
				"outcomeId":           string(outcomeID),
				"planId":              string(plan.ID),
				"planRevisionBinding": plan.ContractRevisionNumber,
				"currentRevision":     outcome.CurrentRevisionNumber,
			})
	}
	if err := s.authorizeCapabilities(plan.Grants); err != nil {
		return AuthorizedPlanView{}, err
	}

	approved, found, err := s.store.ApprovePlanRevision(ctx, outcomeID, plan.ID)
	if err != nil {
		return AuthorizedPlanView{}, err
	}
	if !found {
		return AuthorizedPlanView{}, apierr.NotFound("PLAN_NOT_FOUND", "That plan does not exist")
	}
	approved, err = s.hydratePlanProvider(ctx, approved)
	if err != nil {
		return AuthorizedPlanView{}, err
	}
	return AuthorizedPlanView{Outcome: outcome, Plan: approved}, nil
}

// GetLatestPlan reads the newest plan of any status for re-entry surfaces.
func (s *Service) GetLatestPlan(ctx context.Context, outcomeID domain.OutcomeID) (PlanView, error) {
	outcome, ok, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return PlanView{}, err
	}
	if !ok {
		return PlanView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	plan, found, err := s.store.GetLatestPlanRevision(ctx, outcomeID)
	if err != nil {
		return PlanView{}, err
	}
	if !found {
		return PlanView{}, apierr.NotFound("PLAN_NOT_FOUND", "This Outcome has no plan yet")
	}
	plan, err = s.hydratePlanProvider(ctx, plan)
	if err != nil {
		return PlanView{}, err
	}
	return PlanView{Outcome: outcome, Plan: plan}, nil
}

// currentRevision resolves the content of the Outcome's current revision.
func (s *Service) currentRevision(ctx context.Context, outcome domain.Outcome) (domain.ContractRevision, error) {
	history, err := s.store.ListContractRevisions(ctx, outcome.ID)
	if err != nil {
		return domain.ContractRevision{}, err
	}
	for _, rev := range history {
		if rev.Number == outcome.CurrentRevisionNumber {
			return rev, nil
		}
	}
	return domain.ContractRevision{}, fmt.Errorf("outcome %s points at missing revision %d", outcome.ID, outcome.CurrentRevisionNumber)
}

// v0WorkUnit derives the smallest-sufficient direct unit from the frozen
// contract and explicit Project worker. Deterministic by construction apart
// from identity: identical contract/provider inputs yield identical RunBrief
// semantics even though WorkUnit IDs differ.
func (s *Service) v0WorkUnit(outcome domain.Outcome, revision domain.ContractRevision, provider domain.AgentHarness) domain.WorkUnit {
	checks := make([]string, len(revision.SuccessCriteria))
	copy(checks, revision.SuccessCriteria)
	return domain.WorkUnit{
		ID:                      domain.WorkUnitID("wu-" + uuid.NewString()),
		Kind:                    domain.WorkUnitDirect,
		Title:                   "Deliver \"" + outcome.Title + "\"",
		ContractRevisionNumber:  revision.Number,
		Provider:                provider,
		OutputSummary:           "The finished result, built and verified inside the isolated project worktree.",
		EvidenceChecks:          checks,
		VerificationRequirement: revision.Review,
		StopConditions:          v0StopConditionList(),
	}
}

// v0Grants is the worktree-local capability trio at the widest scope v0 ever
// grants: everything inside the isolated worktree, nothing outside it.
func (s *Service) v0Grants() []domain.CapabilityGrant {
	return []domain.CapabilityGrant{
		{ID: domain.CapabilityGrantID("cg-" + uuid.NewString()), Name: domain.CapabilityWorktreeRead, Scope: "worktree/*"},
		{ID: domain.CapabilityGrantID("cg-" + uuid.NewString()), Name: domain.CapabilityWorktreeWrite, Scope: "worktree/*"},
		{ID: domain.CapabilityGrantID("cg-" + uuid.NewString()), Name: domain.CapabilityWorktreeExec, Scope: "worktree/*"},
	}
}

// authoritativeCapabilities intersects every configured authority layer. The
// default is the v0 policy ceiling itself; narrower environments configure
// fewer capabilities and authorization fails closed against the result.
func (s *Service) authoritativeCapabilities() []string {
	if len(s.PolicyLayers) == 0 {
		return domain.V0RequiredCapabilities
	}
	return domain.AuthorityIntersection(s.PolicyLayers...)
}

// authorizeCapabilities applies both policy gates: every granted capability
// must survive the authority intersection, and every required capability
// must be present. Violations abort with the offender named — never dropped.
func (s *Service) authorizeCapabilities(grants []domain.CapabilityGrant) error {
	authoritative := s.authoritativeCapabilities()
	if err := domain.GrantsFailClosed(grants, authoritative); err != nil {
		var offenders []string
		for _, grant := range grants {
			offenders = append(offenders, grant.Name)
		}
		return apierr.New(apierr.KindConflict, "PLAN_CAPABILITY_UNAUTHORIZED",
			err.Error(),
			map[string]any{
				"granted":       offenders,
				"authoritative": authoritative,
			})
	}
	if missing := domain.MissingRequiredCapabilities(grants); len(missing) > 0 {
		return apierr.Invalid("PLAN_CAPABILITY_MISSING",
			"This environment cannot offer every capability the plan requires: "+strings.Join(missing, ", "),
			map[string]any{"missing": missing})
	}
	return nil
}

func formatI64(v int64) string {
	return strconv.FormatInt(v, 10)
}
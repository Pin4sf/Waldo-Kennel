package outcome

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
)

// PlanView is the service projection of a plan revision.
type PlanView struct {
	Outcome domain.Outcome
	Plan    domain.PlanRevision
}

// AuthorizedPlanView is the projection of an owner-approved plan.
type AuthorizedPlanView struct {
	Outcome domain.Outcome
	Plan    domain.PlanRevision
}

// ApprovePlanInput identifies the plan revision the owner approves.
type ApprovePlanInput struct {
	PlanRevisionID           domain.PlanRevisionID
	ExpectedContractRevision int64
}

func mandatoryPlanStopConditions() []string {
	return []string{
		"Stop before an unapproved dependency",
		"Stop before any remote effect (network, push, PR, deploy) unless the approved Plan explicitly authorizes it",
		"Stop before writes outside the isolated worktree",
		"Stop on contradictory Project policy",
	}
}

// ProposePlan turns non-authoritative planning output into one canonical,
// fully-routed proposal. Project provider/model state is consulted here only as
// preference. The persisted WorkUnit bindings are the authority used later.
func (s *Service) ProposePlan(ctx context.Context, outcomeID domain.OutcomeID, expectedContractRevision int64) (PlanView, error) {
	return s.proposePlan(ctx, outcomeID, expectedContractRevision, "")
}

// ReplanPlan is an explicit owner-directed revision request. Feedback is
// carried into a new immutable proposal; it never mutates or reuses an
// approved/proposed plan in place.
func (s *Service) ReplanPlan(ctx context.Context, outcomeID domain.OutcomeID, expectedContractRevision int64, feedback string) (PlanView, error) {
	if strings.TrimSpace(feedback) == "" {
		return PlanView{}, apierr.Invalid("PLAN_REPLAN_FEEDBACK_REQUIRED", "Explain what the next proposal must change", nil)
	}
	return s.proposePlan(ctx, outcomeID, expectedContractRevision, strings.TrimSpace(feedback))
}

func (s *Service) proposePlan(ctx context.Context, outcomeID domain.OutcomeID, expectedContractRevision int64, replanFeedback string) (PlanView, error) {
	outcomeRecord, ok, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return PlanView{}, err
	}
	if !ok {
		return PlanView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	if expectedContractRevision < 1 {
		return PlanView{}, apierr.Invalid("EXPECTED_REVISION_REQUIRED", "State which contract revision the plan executes", nil)
	}
	if outcomeRecord.CurrentRevisionNumber != expectedContractRevision {
		return PlanView{}, apierr.New(apierr.KindConflict, "PLAN_CONTRACT_STALE",
			fmt.Sprintf("Plan must bind revision %s; reload the Outcome", formatI64(outcomeRecord.CurrentRevisionNumber)),
			map[string]any{"outcomeId": string(outcomeID), "expectedRevision": expectedContractRevision, "currentRevision": outcomeRecord.CurrentRevisionNumber})
	}
	if s.planIntelligence == nil || s.intelligenceRuns == nil || s.routing == nil {
		return PlanView{}, apierr.Internal("PLAN_CONTROL_PLANE_UNWIRED", "Planning is not fully wired in this environment")
	}

	revision, err := s.currentRevision(ctx, outcomeRecord)
	if err != nil {
		return PlanView{}, err
	}
	projectID, project, err := s.projectForOutcome(ctx, outcomeID)
	if err != nil {
		return PlanView{}, err
	}
	preference, hasPreference, err := domain.ResolveEffectiveExecutionPreference(revision.ExecutionPreference, project.Config)
	if err != nil {
		return PlanView{}, apierr.Invalid("PLAN_PREFERENCE_INVALID", err.Error(), nil)
	}
	routingPreference := routingPreferenceFromExecution(preference, hasPreference)

	// Ordinary re-entry is idempotent. A different immutable Contract or
	// execution preference naturally produces a fresh proposal. Explicit
	// re-planning is a separate command surface rather than pretending current
	// runtime readiness is part of immutable Plan identity.
	if replanFeedback == "" {
		if existing, found, err := s.store.LatestProposedPlanRevision(ctx, outcomeID, revision.Number); err != nil {
			return PlanView{}, err
		} else if found && planUsesPreference(existing, routingPreference) {
			return PlanView{Outcome: outcomeRecord, Plan: existing}, nil
		}
	}

	aliases, err := criterionAliases(revision)
	if err != nil {
		return PlanView{}, err
	}
	draft, err := s.draftPlanWithProvenance(ctx, projectID, outcomeRecord, revision, aliases, replanFeedback)
	if err != nil {
		return PlanView{}, err
	}
	units, routingDecisions, err := s.compileAndRoutePlan(ctx, projectID, revision, draft, aliases, routingPreference)
	if err != nil {
		return PlanView{}, err
	}
	grants := grantsForUnits(units)
	if err := s.authorizeCapabilities(revision, grants, units); err != nil {
		return PlanView{}, err
	}
	digest, err := domain.ComputePlanRunBriefCoreDigest(revision, units, grants)
	if err != nil {
		return PlanView{}, err
	}
	proposal := domain.PlanRevision{
		ID:                     domain.PlanRevisionID("plan-" + uuid.NewString()),
		OutcomeID:              outcomeID,
		ContractRevisionNumber: revision.Number,
		Status:                 domain.PlanStatusProposed,
		Summary:                draft.Summary,
		Assumptions:            append([]string(nil), draft.Assumptions...),
		Blockers:               append([]string(nil), draft.Blockers...),
		WorkUnits:              units,
		Grants:                 grants,
		RoutingDecisions:       routingDecisions,
		RunBriefCoreDigest:     digest,
	}
	validation := proposal
	validation.Number = 1
	if err := validation.ValidateAgainstContract(revision); err != nil {
		return PlanView{}, apierr.Invalid("PLAN_DRAFT_CRITERIA_INVALID", err.Error(), nil)
	}
	saved, err := s.store.AppendPlanRevision(ctx, outcomeID, proposal)
	if err != nil {
		return PlanView{}, err
	}
	return PlanView{Outcome: outcomeRecord, Plan: saved}, nil
}

func routingPreferenceFromExecution(preference domain.ExecutionPreference, ok bool) *domain.RoutingPreference {
	if !ok {
		return nil
	}
	return &domain.RoutingPreference{
		Provider:       string(preference.Provider),
		ModelSelection: preference.ModelSelection,
		Model:          preference.Model,
	}
}

func planUsesPreference(plan domain.PlanRevision, preference *domain.RoutingPreference) bool {
	if len(plan.RoutingDecisions) == 0 {
		return false
	}
	for _, record := range plan.RoutingDecisions {
		got := record.Decision.EffectivePreference
		switch {
		case got == nil && preference == nil:
			continue
		case got == nil || preference == nil:
			return false
		case got.Provider != preference.Provider || got.ModelSelection != preference.ModelSelection || got.Model != preference.Model:
			return false
		}
	}
	return true
}

func (s *Service) compileAndRoutePlan(
	ctx context.Context,
	projectID domain.ProjectID,
	revision domain.ContractRevision,
	draft domain.PlanDraftProposal,
	aliases map[string]domain.CriterionID,
	preference *domain.RoutingPreference,
) ([]domain.WorkUnit, []domain.WorkUnitRoutingDecision, error) {
	if err := draft.Validate(); err != nil {
		return nil, nil, apierr.Invalid("PLAN_DRAFT_INVALID", err.Error(), nil)
	}

	ids := make(map[string]domain.WorkUnitID, len(draft.WorkUnits))
	for _, draftUnit := range draft.WorkUnits {
		ids[strings.TrimSpace(draftUnit.Key)] = domain.WorkUnitID("wu-" + uuid.NewString())
	}

	units := make([]domain.WorkUnit, 0, len(draft.WorkUnits))
	for _, draftUnit := range draft.WorkUnits {
		unitID := ids[strings.TrimSpace(draftUnit.Key)]
		criteria := make([]domain.CriterionID, 0, len(draftUnit.CriteriaCovered))
		checks := make([]string, 0, len(draftUnit.EvidenceIdeas))
		for _, alias := range draftUnit.CriteriaCovered {
			criterionID, ok := aliases[strings.TrimSpace(alias)]
			if !ok {
				return nil, nil, apierr.Invalid("PLAN_DRAFT_CRITERION_UNKNOWN", "Plan intelligence referenced an unknown Contract criterion alias", map[string]any{"alias": alias})
			}
			criteria = append(criteria, criterionID)
		}
		checks = append(checks, draftUnit.EvidenceIdeas...)
		if len(checks) == 0 {
			for _, criterionID := range criteria {
				for _, criterion := range revision.Criteria {
					if criterion.ID == criterionID {
						checks = append(checks, criterion.Text)
					}
				}
			}
		}
		dependencies := make([]domain.WorkUnitID, 0, len(draftUnit.DependsOn))
		for _, dependency := range draftUnit.DependsOn {
			dependencyID, ok := ids[strings.TrimSpace(dependency)]
			if !ok {
				return nil, nil, apierr.Invalid("PLAN_DRAFT_DEPENDENCY_UNKNOWN", "Plan intelligence referenced an unknown WorkUnit dependency", map[string]any{"dependency": dependency})
			}
			dependencies = append(dependencies, dependencyID)
		}
		requiredCapabilities, err := draftUnit.Intent.RequiredCapabilities()
		if err != nil {
			return nil, nil, apierr.Invalid("PLAN_DRAFT_INTENT_INVALID", err.Error(), map[string]any{"workUnitKey": draftUnit.Key})
		}
		unit := domain.WorkUnit{
			ID:                      unitID,
			Kind:                    domain.WorkUnitDirect,
			Title:                   strings.TrimSpace(draftUnit.Title),
			ContractRevisionNumber:  revision.Number,
			OutputSummary:           strings.TrimSpace(draftUnit.OutputSummary),
			EvidenceChecks:          uniqueNonBlank(checks),
			VerificationRequirement: revision.Review,
			StopConditions:          uniqueNonBlank(append(append([]string{}, revision.StopConditions...), mandatoryPlanStopConditions()...)),
			DependsOn:               dependencies,
			CriterionIDs:            criteria,
			RequiredCapabilities:    requiredCapabilities,
		}
		if err := validateWorkUnitWithinContractCeiling(revision, unit); err != nil {
			return nil, nil, err
		}
		units = append(units, unit)
	}

	decisions := make([]domain.WorkUnitRoutingDecision, 0, len(units))
	for i := range units {
		snapshot, err := s.routing.RoutingSnapshot(ctx, projectID, preference)
		if err != nil {
			return nil, nil, err
		}
		decision := domain.RouteExecution(domain.RoutingRequirements{
			Role:             domain.RoutingRoleWorker,
			HardCapabilities: append([]string(nil), units[i].RequiredCapabilities...),
			Preference:       preference,
		}, snapshot.Candidates, snapshot.SnapshotID)
		binding, ok := decision.RecommendedBinding()
		if !ok {
			return nil, nil, apierr.New(apierr.KindConflict, "PLAN_NO_VALID_ROUTE",
				"No installed and ready worker can satisfy this WorkUnit's approved requirements",
				map[string]any{"workUnitId": string(units[i].ID), "evaluations": decision.Evaluations})
		}
		if err := units[i].BindExecution(binding); err != nil {
			return nil, nil, err
		}
		decisions = append(decisions, domain.WorkUnitRoutingDecision{WorkUnitID: units[i].ID, Decision: decision})
	}
	return units, decisions, nil
}

// contractCapabilityCeiling converts the confirmed typed ceiling to capability
// names. It never invents authority for a missing/zero ceiling. Read is implied
// by write/execute because neither operation can be performed meaningfully on
// a workspace Kennel is forbidden to inspect.
func contractCapabilityCeiling(revision domain.ContractRevision) []string {
	ceiling := revision.AuthorityCeiling
	var allowed []string
	if ceiling.ReadWorkspace || ceiling.WriteWorkspace || ceiling.ExecuteLocal {
		allowed = append(allowed, domain.CapabilityWorktreeRead)
	}
	if ceiling.WriteWorkspace {
		allowed = append(allowed, domain.CapabilityWorktreeWrite)
	}
	if ceiling.ExecuteLocal {
		allowed = append(allowed, domain.CapabilityWorktreeExec)
	}
	return allowed
}

func validateWorkUnitWithinContractCeiling(revision domain.ContractRevision, unit domain.WorkUnit) error {
	allowed := contractCapabilityCeiling(revision)
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, capability := range allowed {
		allowedSet[capability] = struct{}{}
	}
	var missing []string
	for _, capability := range unit.RequiredCapabilities {
		if _, ok := allowedSet[capability]; !ok {
			missing = append(missing, capability)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	if len(allowed) == 0 {
		return apierr.New(apierr.KindConflict, "PLAN_AUTHORITY_REQUIRED",
			"The confirmed Contract does not grant workspace authority for new execution; revise or reconfirm the Contract before planning",
			map[string]any{"workUnitId": string(unit.ID), "required": missing, "contractCeiling": allowed})
	}
	return apierr.New(apierr.KindConflict, "PLAN_AUTHORITY_INSUFFICIENT",
		"This WorkUnit needs authority outside the confirmed Contract ceiling",
		map[string]any{"workUnitId": string(unit.ID), "required": unit.RequiredCapabilities, "missing": missing, "contractCeiling": allowed})
}

func grantsForUnits(units []domain.WorkUnit) []domain.CapabilityGrant {
	names := domain.RequiredPlanCapabilities(units)
	grants := make([]domain.CapabilityGrant, 0, len(names))
	for _, name := range names {
		grants = append(grants, domain.CapabilityGrant{
			ID: domain.CapabilityGrantID("cg-" + uuid.NewString()), Name: name, Scope: "worktree/*",
		})
	}
	return grants
}

func uniqueNonBlank(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// ApprovePlan converts an already-routed immutable proposal into authority.
// It deliberately does not read Project preferences or routing inventory.
func (s *Service) ApprovePlan(ctx context.Context, outcomeID domain.OutcomeID, in ApprovePlanInput) (AuthorizedPlanView, error) {
	outcomeRecord, ok, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return AuthorizedPlanView{}, err
	}
	if !ok {
		return AuthorizedPlanView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	if in.PlanRevisionID.IsZero() {
		return AuthorizedPlanView{}, apierr.Invalid("PLAN_ID_REQUIRED", "Name the plan to authorize", nil)
	}
	if in.ExpectedContractRevision >= 1 && in.ExpectedContractRevision != outcomeRecord.CurrentRevisionNumber {
		return AuthorizedPlanView{}, apierr.New(apierr.KindConflict, "PLAN_CONTRACT_STALE",
			fmt.Sprintf("Approval was prepared against revision %s; the Outcome is at %s", formatI64(in.ExpectedContractRevision), formatI64(outcomeRecord.CurrentRevisionNumber)),
			map[string]any{"outcomeId": string(outcomeID), "expectedRevision": in.ExpectedContractRevision, "currentRevision": outcomeRecord.CurrentRevisionNumber})
	}
	plan, found, err := s.store.GetPlanRevision(ctx, outcomeID, in.PlanRevisionID)
	if err != nil {
		return AuthorizedPlanView{}, err
	}
	if !found {
		return AuthorizedPlanView{}, apierr.NotFound("PLAN_NOT_FOUND", "That plan does not exist")
	}
	if !plan.BindsCurrentContract(outcomeRecord.CurrentRevisionNumber) {
		return AuthorizedPlanView{}, apierr.New(apierr.KindConflict, "PLAN_CONTRACT_STALE",
			fmt.Sprintf("Plan binds contract revision %s; the Outcome is at %s — propose a new plan", formatI64(plan.ContractRevisionNumber), formatI64(outcomeRecord.CurrentRevisionNumber)),
			map[string]any{"outcomeId": string(outcomeID), "planId": string(plan.ID), "planRevisionBinding": plan.ContractRevisionNumber, "currentRevision": outcomeRecord.CurrentRevisionNumber})
	}
	revision, err := s.currentRevision(ctx, outcomeRecord)
	if err != nil {
		return AuthorizedPlanView{}, err
	}
	if err := plan.ValidateForApproval(revision); err != nil {
		return AuthorizedPlanView{}, apierr.New(apierr.KindConflict, "PLAN_NOT_APPROVABLE", err.Error(), map[string]any{"planId": string(plan.ID)})
	}
	if err := s.authorizeCapabilities(revision, plan.Grants, plan.WorkUnits); err != nil {
		return AuthorizedPlanView{}, err
	}
	approved, found, err := s.store.ApprovePlanRevision(ctx, outcomeID, plan.ID)
	if err != nil {
		return AuthorizedPlanView{}, err
	}
	if !found {
		return AuthorizedPlanView{}, apierr.NotFound("PLAN_NOT_FOUND", "That plan does not exist")
	}
	return AuthorizedPlanView{Outcome: outcomeRecord, Plan: approved}, nil
}

// GetLatestPlan returns the latest plan revision for an Outcome.
func (s *Service) GetLatestPlan(ctx context.Context, outcomeID domain.OutcomeID) (PlanView, error) {
	outcomeRecord, ok, err := s.store.GetOutcome(ctx, outcomeID)
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
	return PlanView{Outcome: outcomeRecord, Plan: plan}, nil
}

func (s *Service) currentRevision(ctx context.Context, outcomeRecord domain.Outcome) (domain.ContractRevision, error) {
	history, err := s.store.ListContractRevisions(ctx, outcomeRecord.ID)
	if err != nil {
		return domain.ContractRevision{}, err
	}
	for _, revision := range history {
		if revision.Number == outcomeRecord.CurrentRevisionNumber {
			return revision, nil
		}
	}
	return domain.ContractRevision{}, fmt.Errorf("outcome %s points at missing revision %d", outcomeRecord.ID, outcomeRecord.CurrentRevisionNumber)
}

// authoritativeCapabilities is the daemon/runtime policy ceiling only. The
// Contract ceiling is checked separately and never defaults from this value.
func (s *Service) authoritativeCapabilities() []string {
	if len(s.PolicyLayers) == 0 {
		return []string{domain.CapabilityWorktreeRead, domain.CapabilityWorktreeWrite, domain.CapabilityWorktreeExec}
	}
	return domain.AuthorityIntersection(s.PolicyLayers...)
}

func (s *Service) authorizeCapabilities(revision domain.ContractRevision, grants []domain.CapabilityGrant, units []domain.WorkUnit) error {
	for _, unit := range units {
		if err := validateWorkUnitWithinContractCeiling(revision, unit); err != nil {
			return err
		}
	}
	if err := domain.ValidateExactPlanCapabilityGrants(grants, units); err != nil {
		return apierr.Invalid("PLAN_CAPABILITY_NOT_MINIMAL", err.Error(), nil)
	}
	authoritative := s.authoritativeCapabilities()
	if err := domain.GrantsFailClosed(grants, authoritative); err != nil {
		offenders := make([]string, 0, len(grants))
		for _, grant := range grants {
			offenders = append(offenders, grant.Name)
		}
		sort.Strings(offenders)
		return apierr.New(apierr.KindConflict, "PLAN_CAPABILITY_UNAUTHORIZED", err.Error(),
			map[string]any{"granted": offenders, "authoritative": authoritative})
	}
	return nil
}

func formatI64(v int64) string { return strconv.FormatInt(v, 10) }

package outcome

import (
	"context"
	"fmt"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
)

const (
	// CodePlanProviderUnbound means the immutable WorkUnit carries no execution
	// provider. Projects may be unconfigured; executable plans may not.
	CodePlanProviderUnbound = "PLAN_PROVIDER_UNBOUND"
	// CodeAttemptProviderMismatch means a compatibility request tried to name a
	// provider different from the one already frozen into the authorized plan.
	CodeAttemptProviderMismatch = "ATTEMPT_PROVIDER_MISMATCH"
)

// projectConfigSource is deliberately narrower than the Project service. The
// SQLite store already satisfies it; focused test stores can provide only the
// record needed to bind a future WorkUnit.
type projectConfigSource interface {
	GetProject(ctx context.Context, id string) (domain.ProjectRecord, bool, error)
}

// providerPlanStore is an additive SQLite capability. It lets production
// persist the WorkUnit + provider binding in one transaction without widening
// ports.OutcomeStore or forcing in-memory test stores to model a storage detail.
type providerPlanStore interface {
	AppendPlanRevisionWithProvider(ctx context.Context, outcomeID domain.OutcomeID, plan domain.PlanRevision) (domain.PlanRevision, error)
	GetWorkUnitProvider(ctx context.Context, workUnitID domain.WorkUnitID) (domain.AgentHarness, bool, error)
}

func (s *Service) projectWorkerProvider(ctx context.Context, outcomeID domain.OutcomeID) (domain.AgentHarness, error) {
	projectID, ok, err := s.store.GetOutcomeProjectID(ctx, outcomeID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	projects, ok := s.store.(projectConfigSource)
	if !ok {
		return "", apierr.Internal("PROJECT_CONFIG_UNWIRED", "Project configuration is unavailable in this environment")
	}
	record, found, err := projects.GetProject(ctx, string(projectID))
	if err != nil {
		return "", err
	}
	if !found {
		return "", apierr.NotFound("PROJECT_NOT_FOUND", "The Project for this Outcome is no longer registered")
	}

	provider := domain.AgentHarness(strings.TrimSpace(string(record.Config.Worker.Harness)))
	if provider == "" {
		return "", providerUnboundError(outcomeID)
	}
	if !provider.IsKnown() || !provider.IsSelectableForNewWork() {
		return "", apierr.New(apierr.KindConflict, "PLAN_PROVIDER_NOT_ADMITTED",
			fmt.Sprintf("Project worker %q is not admitted for new worker execution", provider),
			map[string]any{"outcomeId": string(outcomeID), "provider": string(provider)})
	}
	return provider, nil
}

func providerUnboundError(outcomeID domain.OutcomeID) error {
	return apierr.New(apierr.KindConflict, CodePlanProviderUnbound,
		"This work has no execution provider bound. Configure a Project worker, then propose and approve a fresh plan.",
		map[string]any{
			"outcomeId": string(outcomeID),
			"recovery":  "configure_project_worker_and_replan",
		})
}

// hydratePlanProvider overlays the storage-side provider binding onto the
// domain WorkUnit. Legacy plans have no binding row and remain empty/readable.
func (s *Service) hydratePlanProvider(ctx context.Context, plan domain.PlanRevision) (domain.PlanRevision, error) {
	if len(plan.WorkUnits) != 1 || plan.WorkUnits[0].Provider != "" {
		return plan, nil
	}
	bindings, ok := s.store.(providerPlanStore)
	if !ok {
		return plan, nil
	}
	provider, found, err := bindings.GetWorkUnitProvider(ctx, plan.WorkUnits[0].ID)
	if err != nil {
		return domain.PlanRevision{}, err
	}
	if found {
		plan.WorkUnits[0].Provider = provider
	}
	return plan, nil
}

func (s *Service) appendProviderBoundPlan(ctx context.Context, outcomeID domain.OutcomeID, plan domain.PlanRevision) (domain.PlanRevision, error) {
	if bindings, ok := s.store.(providerPlanStore); ok {
		return bindings.AppendPlanRevisionWithProvider(ctx, outcomeID, plan)
	}
	// Test/in-memory stores persist the domain value directly, including
	// Provider. Production SQLite always takes the atomic capability above.
	return s.store.AppendPlanRevision(ctx, outcomeID, plan)
}

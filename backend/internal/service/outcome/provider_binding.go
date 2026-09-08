package outcome

import (
	"context"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
)

const (
	// CodePlanProviderUnbound preserves the historical-plan recovery vocabulary.
	// New plans fail earlier when routing cannot produce an exact binding.
	CodePlanProviderUnbound = "PLAN_PROVIDER_UNBOUND"
	// CodeAttemptProviderMismatch means a compatibility request named a provider
	// different from the immutable WorkUnit binding.
	CodeAttemptProviderMismatch = "ATTEMPT_PROVIDER_MISMATCH"
)

type projectConfigSource interface {
	GetProject(ctx context.Context, id string) (domain.ProjectRecord, bool, error)
}

func (s *Service) projectForOutcome(ctx context.Context, outcomeID domain.OutcomeID) (domain.ProjectID, domain.ProjectRecord, error) {
	projectID, ok, err := s.store.GetOutcomeProjectID(ctx, outcomeID)
	if err != nil {
		return "", domain.ProjectRecord{}, err
	}
	if !ok {
		return "", domain.ProjectRecord{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	projects, ok := s.store.(projectConfigSource)
	if !ok {
		return "", domain.ProjectRecord{}, apierr.Internal("PROJECT_CONFIG_UNWIRED", "Project configuration is unavailable in this environment")
	}
	record, found, err := projects.GetProject(ctx, string(projectID))
	if err != nil {
		return "", domain.ProjectRecord{}, err
	}
	if !found {
		return "", domain.ProjectRecord{}, apierr.NotFound("PROJECT_NOT_FOUND", "The Project for this Outcome is no longer registered")
	}
	return projectID, record, nil
}

func providerUnboundError(outcomeID domain.OutcomeID) error {
	return apierr.New(apierr.KindConflict, CodePlanProviderUnbound,
		"This historical work has no exact execution provider/model binding. Propose and approve a fresh plan.",
		map[string]any{
			"outcomeId": string(outcomeID),
			"recovery":  "replan",
		})
}

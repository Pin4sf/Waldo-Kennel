package outcome

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

func criterionAliases(revision domain.ContractRevision) (map[string]domain.CriterionID, error) {
	if len(revision.Criteria) == 0 {
		return nil, fmt.Errorf("contract revision %s has no canonical criterion identities", revision.ID)
	}
	criteria := append([]domain.ContractCriterion(nil), revision.Criteria...)
	sort.Slice(criteria, func(i, j int) bool { return criteria[i].Position < criteria[j].Position })
	aliases := make(map[string]domain.CriterionID, len(criteria))
	for i, criterion := range criteria {
		if criterion.ID.IsZero() {
			return nil, fmt.Errorf("contract revision %s contains a blank criterion id", revision.ID)
		}
		aliases[fmt.Sprintf("C%d", i+1)] = criterion.ID
	}
	return aliases, nil
}

func (s *Service) draftPlanWithProvenance(
	ctx context.Context,
	projectID domain.ProjectID,
	outcome domain.Outcome,
	revision domain.ContractRevision,
	aliases map[string]domain.CriterionID,
) (domain.PlanDraftProposal, error) {
	if s.planIntelligence == nil || s.intelligenceRuns == nil {
		return domain.PlanDraftProposal{}, fmt.Errorf("plan intelligence is not wired")
	}
	request := ports.PlanIntelligenceRequest{Outcome: outcome, Contract: revision, CriterionAliases: aliases}
	encoded, err := json.Marshal(request)
	if err != nil {
		return domain.PlanDraftProposal{}, fmt.Errorf("encode plan intelligence input: %w", err)
	}
	now := s.clock().UTC()
	run := domain.IntelligenceRun{
		ID:                 domain.IntelligenceRunID("intel-" + uuid.NewString()),
		Kind:               domain.IntelligenceRunPlanDraft,
		ProjectID:          projectID,
		OutcomeID:          outcome.ID,
		ContractRevisionID: revision.ID,
		SourceRevision:     revision.Number,
		RequestedProvider:  s.planIntelligence.ID(),
		InputDigest:        domain.DigestSHA256(encoded),
		Status:             domain.IntelligenceRunRequested,
		CreatedAt:          now,
	}
	if err := s.intelligenceRuns.CreateIntelligenceRun(ctx, run); err != nil {
		return domain.PlanDraftProposal{}, err
	}
	if err := s.intelligenceRuns.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunRunning, "", "", "", nil); err != nil {
		return domain.PlanDraftProposal{}, err
	}

	response, err := s.planIntelligence.DraftPlan(ctx, request)
	if err != nil {
		completed := s.clock().UTC()
		_ = s.intelligenceRuns.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFailed, "", "INTELLIGENCE_PROVIDER_FAILED", "Plan intelligence provider failed", &completed)
		return domain.PlanDraftProposal{}, err
	}
	if err := response.Proposal.Validate(); err != nil {
		completed := s.clock().UTC()
		_ = s.intelligenceRuns.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFailed, "", "PLAN_DRAFT_INVALID", "Plan intelligence returned an invalid proposal", &completed)
		return domain.PlanDraftProposal{}, fmt.Errorf("plan intelligence proposal: %w", err)
	}
	if err := s.intelligenceRuns.RecordIntelligenceRunEffectiveProvenance(ctx, run.ID,
		response.Provenance.EffectiveProvider, response.Provenance.EffectiveModel, response.Provenance.NativeSessionRef); err != nil {
		return domain.PlanDraftProposal{}, err
	}
	output, err := json.Marshal(response.Proposal)
	if err != nil {
		return domain.PlanDraftProposal{}, fmt.Errorf("encode plan intelligence output: %w", err)
	}
	completed := s.clock().UTC()
	if err := s.intelligenceRuns.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFulfilled, domain.DigestSHA256(output), "", "", &completed); err != nil {
		return domain.PlanDraftProposal{}, err
	}
	return response.Proposal, nil
}

// keep time imported in generated/refactor-safe builds where the compiler may
// inline clock uses differently; this assertion also documents UTC completion.
var _ = time.Time{}

package outcome

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
)

// CodeContributionBlocked reports a contributing Outcome whose declared
// upstream siblings are not yet accepted.
const CodeContributionBlocked = "CONTRIBUTION_BLOCKED"

// WaiveDependencyInput is the owner's override of an ordering they authorized.
type WaiveDependencyInput struct {
	// FromRef must finish before ToRef under the authorized decomposition.
	FromRef string
	ToRef   string
	// Reason is required: a waiver nobody can explain later is
	// indistinguishable from a mistake.
	Reason string
}

// startGateFor resolves whether a contributing Outcome may begin work.
//
// A Project-level or standalone Outcome is never gated — gating is a property
// of contributing to something, not of Outcomes in general — and neither is a
// contributor whose parent decomposition is still only proposed. An offer must
// not be able to block real work.
func (s *Service) startGateFor(ctx context.Context, child domain.Outcome) (domain.ContributionStartGate, error) {
	if !child.IsContributing() {
		return domain.ContributionStartGate{}, nil
	}
	decomposition, found, err := s.store.LatestDecompositionRevision(ctx, child.ParentID)
	if err != nil {
		return domain.ContributionStartGate{}, err
	}
	if !found || decomposition.Status != domain.DecompositionAuthorized {
		return domain.ContributionStartGate{}, nil
	}
	waivers, err := s.store.ListContributionDependencyWaivers(ctx, decomposition.ID)
	if err != nil {
		return domain.ContributionStartGate{}, err
	}
	accepted, err := s.acceptedContributors(ctx, decomposition)
	if err != nil {
		return domain.ContributionStartGate{}, err
	}
	return domain.DeriveContributionStartGate(child.ID, decomposition, waivers, accepted), nil
}

// acceptedContributors reads which sibling Outcomes currently carry an owner
// acceptance. A missing proof store means acceptance cannot be proven, and an
// unprovable upstream must read as unaccepted rather than as done.
func (s *Service) acceptedContributors(ctx context.Context, decomposition domain.DecompositionRevision) (map[domain.OutcomeID]bool, error) {
	accepted := make(map[domain.OutcomeID]bool, len(decomposition.Contributors))
	if s.proof == nil {
		return accepted, nil
	}
	for _, contributor := range decomposition.Contributors {
		if contributor.ChildOutcomeID.IsZero() {
			continue
		}
		decisions, err := s.proof.ListAcceptanceDecisions(ctx, contributor.ChildOutcomeID)
		if err != nil {
			return nil, err
		}
		accepted[contributor.ChildOutcomeID] = domain.LatestDecisionAccepts(decisions)
	}
	return accepted, nil
}

// blockedError turns an unmet gate into a refusal the owner can act on: it
// names every upstream, why each is unmet, and the two ways forward.
func blockedError(child domain.OutcomeID, gate domain.ContributionStartGate) error {
	blockers := make([]map[string]any, 0, len(gate.Blocked))
	names := make([]string, 0, len(gate.Blocked))
	for _, block := range gate.Blocked {
		blockers = append(blockers, map[string]any{
			"ref": block.Ref, "outcomeId": string(block.OutcomeID),
			"title": block.Title, "reason": block.Reason,
		})
		label := block.Title
		if label == "" {
			label = block.Ref
		}
		names = append(names, label)
	}
	return apierr.Conflict(CodeContributionBlocked,
		"This contribution waits on "+strings.Join(names, ", ")+
			". Accept the upstream contribution, or waive the dependency with a reason.",
		map[string]any{"outcomeId": string(child), "blockedBy": blockers})
}

// WaiveContributionDependency records the owner's decision to start a
// contributing Outcome before a declared upstream is accepted.
//
// Only the owner may waive, the reason is durable, and the waiver never
// disappears: a waived dependency is overridden, not forgotten, and stays
// visible next to whatever the early start produced.
func (s *Service) WaiveContributionDependency(ctx context.Context, parentID domain.OutcomeID, in WaiveDependencyInput) (DecompositionView, error) {
	from, to := strings.TrimSpace(in.FromRef), strings.TrimSpace(in.ToRef)
	reason := strings.TrimSpace(in.Reason)
	switch {
	case from == "" || to == "":
		return DecompositionView{}, apierr.Invalid("WAIVER_REFS_REQUIRED", "Name both contributions the dependency orders", nil)
	case from == to:
		return DecompositionView{}, apierr.Invalid("WAIVER_REFS_REQUIRED", "A contribution cannot depend on itself", nil)
	case reason == "":
		return DecompositionView{}, apierr.Invalid("WAIVER_REASON_REQUIRED",
			"Explain why this ordering is safe to override; the reason is kept with the decision", nil)
	}

	if _, ok, err := s.store.GetOutcome(ctx, parentID); err != nil {
		return DecompositionView{}, err
	} else if !ok {
		return DecompositionView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome no longer exists")
	}
	decomposition, found, err := s.store.LatestDecompositionRevision(ctx, parentID)
	if err != nil {
		return DecompositionView{}, err
	}
	if !found || decomposition.Status != domain.DecompositionAuthorized {
		return DecompositionView{}, apierr.Conflict("DECOMPOSITION_NOT_AUTHORIZED",
			"Authorize the decomposition before waiving one of its dependencies", nil)
	}
	declared := false
	for _, dependency := range decomposition.Dependencies {
		if dependency.FromRef == from && dependency.ToRef == to {
			declared = true
			break
		}
	}
	if !declared {
		// Waiving an ordering nobody declared would record consent to nothing.
		return DecompositionView{}, apierr.Invalid("DEPENDENCY_NOT_DECLARED",
			"This decomposition declares no such ordering to waive",
			map[string]any{"fromRef": from, "toRef": to})
	}

	if err := s.store.AppendContributionDependencyWaiver(ctx, domain.ContributionDependencyWaiver{
		ID:              domain.ContributionWaiverID("cw-" + uuid.NewString()),
		DecompositionID: decomposition.ID,
		FromRef:         from,
		ToRef:           to,
		Reason:          reason,
		WaivedBy:        domain.AcceptanceActorUser,
		CreatedAt:       s.clock(),
	}); err != nil {
		return DecompositionView{}, err
	}
	return s.LatestDecomposition(ctx, parentID)
}

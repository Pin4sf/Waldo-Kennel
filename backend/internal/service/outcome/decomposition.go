package outcome

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// ProposeDecompositionInput states a decomposition for validation.
//
// Contributors may be omitted, in which case the daemon derives the
// deterministic default: one contributing Outcome per parent criterion. That
// default is a starting point the owner corrects, not a recommendation — it
// makes no claim to be the right topology, only a complete and inspectable one.
type ProposeDecompositionInput struct {
	// ExpectedContractRevision must name the parent's current revision. A
	// stale number means the decomposition answers a contract that has since
	// changed, and is refused rather than silently rebound.
	ExpectedContractRevision int64
	Rationale                string
	Contributors             []ProposedContributionInput
	// RetainedCriteria are parent criteria the owner will prove directly.
	RetainedCriteria []domain.CriterionID
	Dependencies     []ContributionDependencyInput
}

// ProposedContributionInput is one contributing Outcome as offered.
type ProposedContributionInput struct {
	Ref             string
	Title           string
	Goal            string
	SuccessCriteria []string
	Review          string
	Constraints     []string
	NonGoals        []string
	Authority       domain.ProposedAuthority
	ClaimedCriteria []domain.CriterionID
}

// ContributionDependencyInput declares that From must finish before To starts.
type ContributionDependencyInput struct {
	FromRef string
	ToRef   string
}

// DecompositionView is one decomposition and how it stands against the
// parent's current contract.
type DecompositionView struct {
	Decomposition domain.DecompositionRevision
	// Stale reports that the parent's contract has moved on since this
	// decomposition was proposed. A stale proposal cannot be authorized.
	Stale bool
}

// ProposeDecomposition validates a decomposition and records it as a proposal.
//
// Nothing it accepts is a responsibility yet: no Outcome, no contract, no
// binding. Authorization is the owner decision that creates them, so a refused
// proposal leaves nothing behind.
//
// The deterministic gates run here, before the owner ever reviews the offer,
// and each names its offender:
//
//  1. the parent must exist, be a root, and hold the expected contract;
//  2. every criterion referenced must exist in that contract;
//  3. every criterion must be claimed or retained — never neither;
//  4. no contributor may claim authority the parent does not hold;
//  5. dependencies must name proposed contributions and must not cycle.
//
// A model may author the contributors. It cannot author the verdict.
func (s *Service) ProposeDecomposition(ctx context.Context, parentID domain.OutcomeID, in ProposeDecompositionInput) (DecompositionView, error) {
	parent, current, err := s.decomposableParent(ctx, parentID, in.ExpectedContractRevision)
	if err != nil {
		return DecompositionView{}, err
	}

	contributors := in.Contributors
	rationale := strings.TrimSpace(in.Rationale)
	retained := in.RetainedCriteria
	if len(contributors) == 0 {
		contributors, rationale = defaultDecomposition(current, rationale)
		retained = nil
	}

	proposed, err := toProposedContributions(contributors)
	if err != nil {
		return DecompositionView{}, err
	}
	dependencies := make([]domain.ContributionDependency, 0, len(in.Dependencies))
	for _, dependency := range in.Dependencies {
		dependencies = append(dependencies, domain.ContributionDependency{
			ID:      "cdep-" + uuid.NewString(),
			FromRef: strings.TrimSpace(dependency.FromRef),
			ToRef:   strings.TrimSpace(dependency.ToRef),
		})
	}
	if err := s.validateDecomposition(current, proposed, retained, dependencies); err != nil {
		return DecompositionView{}, err
	}

	revision := domain.DecompositionRevision{
		ID:                 domain.DecompositionRevisionID("dec-" + uuid.NewString()),
		OutcomeID:          parent.ID,
		Number:             1, // storage assigns the real number in-transaction
		ContractRevisionID: current.ID,
		Status:             domain.DecompositionProposed,
		Rationale:          rationale,
		Contributors:       proposed,
		RetainedCriteria:   retained,
		Dependencies:       dependencies,
		CreatedAt:          s.clock(),
	}
	if err := revision.Validate(); err != nil {
		return DecompositionView{}, apierr.Invalid("DECOMPOSITION_INVALID", err.Error(), nil)
	}

	stored, err := s.store.AppendDecompositionRevision(ctx, revision)
	if err != nil {
		return DecompositionView{}, err
	}
	return DecompositionView{Decomposition: stored}, nil
}

// AuthorizeDecomposition is the owner decision that creates the contributing
// Outcomes a proposal described.
//
// It re-runs every gate against the parent's CURRENT contract rather than
// trusting what passed at propose time. A contract that moved in between makes
// the proposal stale, and a stale decomposition answers a question the owner
// is no longer asking.
func (s *Service) AuthorizeDecomposition(ctx context.Context, parentID domain.OutcomeID, decompositionID domain.DecompositionRevisionID) (DecompositionView, error) {
	if strings.TrimSpace(string(decompositionID)) == "" {
		return DecompositionView{}, apierr.Invalid("DECOMPOSITION_REQUIRED", "Name the decomposition to authorize", nil)
	}
	parent, ok, err := s.store.GetOutcome(ctx, parentID)
	if err != nil {
		return DecompositionView{}, err
	}
	if !ok {
		return DecompositionView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome no longer exists")
	}
	revision, found, err := s.store.GetDecompositionRevision(ctx, parentID, decompositionID)
	if err != nil {
		return DecompositionView{}, err
	}
	if !found {
		return DecompositionView{}, apierr.NotFound("DECOMPOSITION_NOT_FOUND", "That decomposition no longer exists")
	}
	if revision.Status == domain.DecompositionAuthorized {
		// Re-authorizing is a replay, not an error: the contributing Outcomes
		// already exist and a second attempt must not create them twice.
		return DecompositionView{Decomposition: revision}, nil
	}

	current, err := s.currentRevision(ctx, parent)
	if err != nil {
		return DecompositionView{}, err
	}
	if revision.ContractRevisionID != current.ID {
		return DecompositionView{}, apierr.Invalid("DECOMPOSITION_STALE",
			"The parent contract changed after this decomposition was proposed; propose it again against the current revision",
			map[string]any{"proposedAgainst": string(revision.ContractRevisionID), "current": string(current.ID)})
	}
	if err := s.validateDecomposition(current, revision.Contributors, revision.RetainedCriteria, revision.Dependencies); err != nil {
		return DecompositionView{}, err
	}

	now := s.clock()
	contributions := make([]ports.AuthorizedContribution, 0, len(revision.Contributors))
	for _, contributor := range revision.Contributors {
		child := domain.Outcome{
			ID:       domain.OutcomeID("out-" + uuid.NewString()),
			SpaceID:  parent.SpaceID,
			ParentID: parent.ID,
			Title:    contributor.Title,
		}
		first := domain.ContractRevision{
			ID:               domain.ContractRevisionID("cr-" + uuid.NewString()),
			OutcomeID:        child.ID,
			Goal:             contributor.Goal,
			SuccessCriteria:  contributor.SuccessCriteria,
			Review:           contributor.Review,
			Constraints:      contributor.Constraints,
			NonGoals:         contributor.NonGoals,
			AuthorityCeiling: contributor.Authority,
			CreatedAt:        now,
		}
		first.Criteria = stableCriteria(first.ID, first.SuccessCriteria)

		links := make([]domain.ContributionLink, 0, len(contributor.ClaimedCriteria))
		for _, criterion := range contributor.ClaimedCriteria {
			links = append(links, domain.ContributionLink{
				ID:                       domain.ContributionLinkID("cl-" + uuid.NewString()),
				ParentOutcomeID:          parent.ID,
				ChildOutcomeID:           child.ID,
				ParentContractRevisionID: current.ID,
				ParentCriterionID:        criterion,
				CreatedAt:                now,
			})
		}
		contributions = append(contributions, ports.AuthorizedContribution{
			Ref: contributor.Ref, Outcome: child, First: first, Links: links,
		})
	}

	if err := s.store.AuthorizeDecompositionRevision(ctx, parentID, decompositionID, contributions, now); err != nil {
		if errors.Is(err, ports.ErrDecompositionNotProposed) {
			// Lost the race to a concurrent authorization; serve the winner.
			if authorized, found, readErr := s.store.GetDecompositionRevision(ctx, parentID, decompositionID); readErr == nil && found {
				return DecompositionView{Decomposition: authorized}, nil
			}
		}
		return DecompositionView{}, err
	}

	authorized, _, err := s.store.GetDecompositionRevision(ctx, parentID, decompositionID)
	if err != nil {
		return DecompositionView{}, err
	}
	return DecompositionView{Decomposition: authorized}, nil
}

// LatestDecomposition reads the newest decomposition and whether the parent's
// contract has moved past it.
func (s *Service) LatestDecomposition(ctx context.Context, parentID domain.OutcomeID) (DecompositionView, error) {
	parent, ok, err := s.store.GetOutcome(ctx, parentID)
	if err != nil {
		return DecompositionView{}, err
	}
	if !ok {
		return DecompositionView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome no longer exists")
	}
	revision, found, err := s.store.LatestDecompositionRevision(ctx, parentID)
	if err != nil {
		return DecompositionView{}, err
	}
	if !found {
		return DecompositionView{}, apierr.NotFound("DECOMPOSITION_NOT_FOUND", "This Outcome has not been decomposed")
	}
	current, err := s.currentRevision(ctx, parent)
	if err != nil {
		return DecompositionView{}, err
	}
	return DecompositionView{Decomposition: revision, Stale: revision.ContractRevisionID != current.ID}, nil
}

// decomposableParent resolves a parent that may legally be decomposed against
// an exact contract revision.
func (s *Service) decomposableParent(ctx context.Context, parentID domain.OutcomeID, expected int64) (domain.Outcome, domain.ContractRevision, error) {
	if strings.TrimSpace(string(parentID)) == "" {
		return domain.Outcome{}, domain.ContractRevision{}, apierr.Invalid("OUTCOME_REQUIRED", "Name the Outcome to decompose", nil)
	}
	if expected < 1 {
		return domain.Outcome{}, domain.ContractRevision{}, apierr.Invalid("EXPECTED_REVISION_REQUIRED",
			"State which contract revision this decomposition answers", nil)
	}
	parent, ok, err := s.store.GetOutcome(ctx, parentID)
	if err != nil {
		return domain.Outcome{}, domain.ContractRevision{}, err
	}
	if !ok {
		return domain.Outcome{}, domain.ContractRevision{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome no longer exists")
	}
	if parent.IsContributing() {
		return domain.Outcome{}, domain.ContractRevision{}, apierr.Invalid("COMPOSITION_DEPTH_LIMIT",
			fmt.Sprintf("Composition is %d levels deep: a contributing Outcome cannot itself be decomposed", domain.CompositionDepthLimit),
			map[string]any{"outcomeId": string(parentID)})
	}
	current, err := s.currentRevision(ctx, parent)
	if err != nil {
		return domain.Outcome{}, domain.ContractRevision{}, err
	}
	if current.Number != expected {
		return domain.Outcome{}, domain.ContractRevision{}, apierr.Invalid("DECOMPOSITION_STALE",
			"That is no longer the current contract revision",
			map[string]any{"expected": expected, "current": current.Number})
	}
	return parent, current, nil
}

// validateDecomposition runs the deterministic gates. Order matters only in
// that each refusal names one class of problem rather than a merged blur.
func (s *Service) validateDecomposition(current domain.ContractRevision, contributors []domain.ProposedContribution, retained []domain.CriterionID, dependencies []domain.ContributionDependency) error {
	if unknown := domain.UnknownClaimedCriteria(current, contributors, retained); len(unknown) > 0 {
		ids := make([]string, 0, len(unknown))
		for _, id := range unknown {
			ids = append(ids, string(id))
		}
		return apierr.Invalid("CRITERION_NOT_IN_CURRENT_CONTRACT",
			"These criteria are not part of the parent's current contract revision: "+strings.Join(ids, ", "),
			map[string]any{"criterionIds": ids, "contractRevision": current.Number})
	}
	if uncovered := domain.UncoveredCriteria(current, contributors, retained); len(uncovered) > 0 {
		texts := make([]string, 0, len(uncovered))
		ids := make([]string, 0, len(uncovered))
		for _, criterion := range uncovered {
			texts = append(texts, criterion.Text)
			ids = append(ids, string(criterion.ID))
		}
		// The gate that makes a decomposition trustworthy. An unclassified
		// criterion is how a project reports itself done while missing
		// something material.
		return apierr.Invalid("DECOMPOSITION_INCOMPLETE",
			"Every criterion must be claimed by a contributing Outcome or explicitly retained. These are neither: "+strings.Join(texts, "; "),
			map[string]any{"criterionIds": ids})
	}
	if offenders := domain.OverClaimedAuthority(current.AuthorityCeiling, contributors); len(offenders) > 0 {
		refs := make([]string, 0, len(offenders))
		for ref := range offenders {
			refs = append(refs, ref)
		}
		sort.Strings(refs)
		parts := make([]string, 0, len(refs))
		for _, ref := range refs {
			parts = append(parts, ref+" ("+strings.Join(offenders[ref], ", ")+")")
		}
		return apierr.Invalid("AUTHORITY_WIDENED",
			"A contributing Outcome cannot claim authority its parent does not hold: "+strings.Join(parts, "; "),
			map[string]any{"contributions": offenders})
	}
	if err := domain.ValidateDependencyOrdering(contributors, dependencies); err != nil {
		return apierr.Invalid("DEPENDENCY_CYCLE", err.Error(), nil)
	}
	return nil
}

// defaultDecomposition derives the deterministic starting point: one
// contributing Outcome per parent criterion.
//
// It is deliberately mechanical rather than clever. A model may replace it
// with a better topology, but the daemon's own proposal must be reproducible
// and explainable — identical contracts yield identical decompositions.
func defaultDecomposition(current domain.ContractRevision, rationale string) ([]ProposedContributionInput, string) {
	contributors := make([]ProposedContributionInput, 0, len(current.Criteria))
	for i, criterion := range current.Criteria {
		contributors = append(contributors, ProposedContributionInput{
			Ref:             fmt.Sprintf("c%d", i+1),
			Title:           criterion.Text,
			Goal:            criterion.Text,
			SuccessCriteria: []string{criterion.Text},
			Review:          current.Review,
			Constraints:     current.Constraints,
			NonGoals:        current.NonGoals,
			Authority:       current.AuthorityCeiling,
			ClaimedCriteria: []domain.CriterionID{criterion.ID},
		})
	}
	if rationale == "" {
		rationale = fmt.Sprintf(
			"One contributing Outcome per criterion (%d). This is the daemon's mechanical starting point, not a recommended topology — correct it before authorizing.",
			len(contributors))
	}
	return contributors, rationale
}

func toProposedContributions(inputs []ProposedContributionInput) ([]domain.ProposedContribution, error) {
	proposed := make([]domain.ProposedContribution, 0, len(inputs))
	for i, in := range inputs {
		contributor := domain.ProposedContribution{
			Ref:             strings.TrimSpace(in.Ref),
			Position:        int64(i + 1),
			Title:           strings.TrimSpace(in.Title),
			Goal:            strings.TrimSpace(in.Goal),
			SuccessCriteria: trimAll(in.SuccessCriteria),
			Review:          strings.TrimSpace(in.Review),
			Constraints:     trimAll(in.Constraints),
			NonGoals:        trimAll(in.NonGoals),
			Authority:       in.Authority,
			ClaimedCriteria: in.ClaimedCriteria,
		}
		if contributor.Ref == "" {
			contributor.Ref = fmt.Sprintf("c%d", i+1)
		}
		if err := contributor.Validate(); err != nil {
			return nil, apierr.Invalid("CONTRIBUTION_INVALID", err.Error(), nil)
		}
		proposed = append(proposed, contributor)
	}
	return proposed, nil
}

func trimAll(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

package outcome

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
)

// CreateContributionInput states one contributing Outcome and the parent
// criteria it claims. The contract fields mirror CreateInput: a contributing
// Outcome is a full responsibility, not a task, so it states its own goal,
// criteria, and review just like a Project-level one.
type CreateContributionInput struct {
	RequestKey string
	Title      string
	Goal       string
	// Criteria are the child's own success criteria.
	SuccessCriteria []string
	Review          string
	Constraints     []string
	NonGoals        []string
	Clarification   string
	// ClaimedCriteria names the parent criterion identities this Outcome
	// contributes to. At least one is required: an Outcome that claims
	// nothing is not a contribution.
	ClaimedCriteria []domain.CriterionID
	// Authority is the child's proposed ceiling. It must be contained by the
	// parent's; omitted fields simply claim less.
	Authority domain.ProposedAuthority
}

// CompositionView projects a decomposed Outcome's contributors and the
// coverage of its current criteria. Shape is derived here, never stored.
type CompositionView struct {
	Shape        domain.OutcomeShape
	Parent       *domain.Outcome
	Contributors []ContributorView
	Coverage     []domain.CriterionClaim
}

// ContributorView is one contributing Outcome with the bindings that make it
// a contribution, plus whether those bindings still name the parent's current
// revision.
type ContributorView struct {
	Outcome domain.Outcome
	Links   []domain.ContributionLink
	// Stale reports that this Outcome is bound to a superseded parent
	// revision. It blocks new authorization; it does not kill running work.
	Stale bool
}

// Unclaimed returns the current parent criteria no contributor claims. A
// non-empty result is a truthful report of an incomplete decomposition, not
// an error: coverage becomes a gate at authorization and at acceptance.
func (v CompositionView) Unclaimed() []domain.CriterionClaim {
	unclaimed := make([]domain.CriterionClaim, 0)
	for _, claim := range v.Coverage {
		if !claim.Claimed() {
			unclaimed = append(unclaimed, claim)
		}
	}
	return unclaimed
}

// CreateContribution adds one contributing Outcome beneath parentID.
//
// Every refusal here fails closed and names its offender. The gates, in the
// order a caller hits them:
//
//  1. the parent must exist, hold a contract, and itself be a root — the
//     depth cap makes cycles impossible by construction;
//  2. every claimed criterion must exist in the parent's CURRENT revision, so
//     a contribution can never be bound to a superseded criterion;
//  3. the child's authority ceiling must be contained by the parent's.
//
// Storage then enforces link immutability and the single-parent-revision rule
// inside the same transaction that creates the child.
func (s *Service) CreateContribution(ctx context.Context, parentID domain.OutcomeID, in CreateContributionInput) (OutcomeView, error) {
	if strings.TrimSpace(string(parentID)) == "" {
		return OutcomeView{}, apierr.Invalid("PARENT_REQUIRED", "Name the Outcome this one contributes to", nil)
	}
	if strings.TrimSpace(in.RequestKey) == "" {
		return OutcomeView{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this create", nil)
	}
	content := normalizeContractContent(in.RequestKey, in.Title, in.Goal, in.SuccessCriteria, in.Review, in.Constraints, in.NonGoals, in.Clarification)
	if err := validateTitle(content.title); err != nil {
		return OutcomeView{}, err
	}
	if err := validateContractCore(content); err != nil {
		return OutcomeView{}, err
	}

	// Replay first: a delivered create never writes twice.
	if existing, ok, err := s.store.FindOutcomeByIdempotencyKey(ctx, content.requestKey); err != nil {
		return OutcomeView{}, err
	} else if ok {
		return s.Get(ctx, existing.ID)
	}

	parent, ok, err := s.store.GetOutcome(ctx, parentID)
	if err != nil {
		return OutcomeView{}, err
	}
	if !ok {
		return OutcomeView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome no longer exists")
	}
	if parent.IsContributing() {
		return OutcomeView{}, apierr.Invalid("COMPOSITION_DEPTH_LIMIT",
			fmt.Sprintf("Composition is %d levels deep: a contributing Outcome cannot itself be decomposed", domain.CompositionDepthLimit),
			map[string]any{"parentId": string(parentID), "parentOf": string(parent.ParentID)})
	}
	parentCurrent, err := s.currentRevision(ctx, parent)
	if err != nil {
		return OutcomeView{}, err
	}

	claimed, err := resolveClaimedCriteria(parentCurrent, in.ClaimedCriteria)
	if err != nil {
		return OutcomeView{}, err
	}
	if widened := domain.AuthorityWidenings(parentCurrent.AuthorityCeiling, in.Authority); len(widened) > 0 {
		return OutcomeView{}, apierr.Invalid("AUTHORITY_WIDENED",
			"A contributing Outcome cannot claim authority its parent does not hold: "+strings.Join(widened, ", "),
			map[string]any{"parentId": string(parentID), "widened": widened})
	}

	now := s.clock()
	child := domain.Outcome{
		ID:       domain.OutcomeID("out-" + uuid.NewString()),
		SpaceID:  parent.SpaceID,
		ParentID: parent.ID,
		Title:    content.title,
	}
	first := domain.ContractRevision{
		ID:               domain.ContractRevisionID("cr-" + uuid.NewString()),
		OutcomeID:        child.ID,
		Goal:             content.goal,
		SuccessCriteria:  content.criteria,
		Review:           content.review,
		Constraints:      content.constraints,
		NonGoals:         content.nonGoals,
		Clarification:    content.clarification,
		AuthorityCeiling: in.Authority,
		CreatedAt:        now,
	}
	first.Criteria = stableCriteria(first.ID, first.SuccessCriteria)

	links := make([]domain.ContributionLink, 0, len(claimed))
	for _, criterionID := range claimed {
		links = append(links, domain.ContributionLink{
			ID:                       domain.ContributionLinkID("cl-" + uuid.NewString()),
			ParentOutcomeID:          parent.ID,
			ChildOutcomeID:           child.ID,
			ParentContractRevisionID: parentCurrent.ID,
			ParentCriterionID:        criterionID,
			CreatedAt:                now,
		})
	}

	if err := s.store.CreateContributionWithContract(ctx, child, first, links, content.requestKey); err != nil {
		// Either a genuine failure or a lost replay race against an identical
		// request; resolve through the key so both paths serve the winner.
		if existing, ok, findErr := s.store.FindOutcomeByIdempotencyKey(ctx, content.requestKey); findErr == nil && ok {
			return s.Get(ctx, existing.ID)
		}
		return OutcomeView{}, err
	}
	return s.Get(ctx, child.ID)
}

// Composition reads one Outcome's shape, contributors, and criterion coverage.
// A direct Outcome returns shape "direct" with no contributors, which is what
// every Outcome created before composition existed reports.
func (s *Service) Composition(ctx context.Context, id domain.OutcomeID) (CompositionView, error) {
	record, ok, err := s.store.GetOutcome(ctx, id)
	if err != nil {
		return CompositionView{}, err
	}
	if !ok {
		return CompositionView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome no longer exists")
	}

	view := CompositionView{Coverage: []domain.CriterionClaim{}, Contributors: []ContributorView{}}
	if record.IsContributing() {
		parent, found, err := s.store.GetOutcome(ctx, record.ParentID)
		if err != nil {
			return CompositionView{}, err
		}
		if found {
			view.Parent = &parent
		}
	}

	children, err := s.store.ListContributingOutcomes(ctx, id)
	if err != nil {
		return CompositionView{}, err
	}
	view.Shape = domain.ShapeForChildCount(len(children))
	if len(children) == 0 {
		return view, nil
	}

	links, err := s.store.ListContributionLinksForParent(ctx, id)
	if err != nil {
		return CompositionView{}, err
	}
	current, err := s.currentRevision(ctx, record)
	if err != nil {
		return CompositionView{}, err
	}
	byChild := make(map[domain.OutcomeID][]domain.ContributionLink, len(children))
	for _, link := range links {
		byChild[link.ChildOutcomeID] = append(byChild[link.ChildOutcomeID], link)
	}
	for _, child := range children {
		childLinks := byChild[child.ID]
		if childLinks == nil {
			childLinks = []domain.ContributionLink{}
		}
		view.Contributors = append(view.Contributors, ContributorView{
			Outcome: child,
			Links:   childLinks,
			Stale:   domain.ContributionStale(current, childLinks),
		})
	}
	view.Coverage = domain.CriterionCoverage(current, links)
	return view, nil
}

// resolveClaimedCriteria checks every claimed identity against the parent's
// current revision and rejects duplicates. Binding to a criterion that no
// longer exists would let a contribution prove something the parent stopped
// asking for.
func resolveClaimedCriteria(parentCurrent domain.ContractRevision, claimed []domain.CriterionID) ([]domain.CriterionID, error) {
	if len(claimed) == 0 {
		return nil, apierr.Invalid("CONTRIBUTION_UNBOUND",
			"Name at least one parent criterion this Outcome contributes to", nil)
	}
	known := make(map[domain.CriterionID]struct{}, len(parentCurrent.Criteria))
	for _, criterion := range parentCurrent.Criteria {
		known[criterion.ID] = struct{}{}
	}
	seen := make(map[domain.CriterionID]struct{}, len(claimed))
	resolved := make([]domain.CriterionID, 0, len(claimed))
	for _, id := range claimed {
		trimmed := domain.CriterionID(strings.TrimSpace(string(id)))
		if trimmed.IsZero() {
			return nil, apierr.Invalid("CRITERION_REQUIRED", "A claimed criterion identity cannot be blank", nil)
		}
		if _, ok := known[trimmed]; !ok {
			return nil, apierr.Invalid("CRITERION_NOT_IN_CURRENT_CONTRACT",
				"That criterion is not part of the parent's current contract revision",
				map[string]any{"criterionId": string(trimmed), "parentRevision": parentCurrent.Number})
		}
		if _, dup := seen[trimmed]; dup {
			return nil, apierr.Invalid("CRITERION_CLAIMED_TWICE",
				"A contributing Outcome claims each parent criterion at most once",
				map[string]any{"criterionId": string(trimmed)})
		}
		seen[trimmed] = struct{}{}
		resolved = append(resolved, trimmed)
	}
	return resolved, nil
}

package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// CompositionDepthLimit is the number of Outcome levels composition admits: a
// Project-level Outcome and its contributing Outcomes, and no further.
//
// The cap is a governance decision, not a storage limit (ADR 0007). Each extra
// level multiplies authority intersection, staleness cascade, coverage
// validation, and attention roll-up. Raising it requires evidence that a third
// level is needed, not the observation that the schema would tolerate one.
const CompositionDepthLimit = 2

// OutcomeShape is the derived answer to "how is this Outcome pursued?". It is
// never stored: like every other Outcome status it is computed from durable
// facts at read time, so a row can never disagree with its own children.
type OutcomeShape string

const (
	// OutcomeShapeDirect owns PlanRevisions and Attempts. Every Outcome
	// created before composition existed is direct, and behaves unchanged.
	OutcomeShapeDirect OutcomeShape = "direct"
	// OutcomeShapeDecomposed owns contributing Outcomes instead of a plan.
	// Its decomposition is its plan; it never starts an Attempt of its own.
	OutcomeShapeDecomposed OutcomeShape = "decomposed"
)

// Valid reports whether s is a supported shape.
func (s OutcomeShape) Valid() bool {
	return s == OutcomeShapeDirect || s == OutcomeShapeDecomposed
}

// ShapeForChildCount derives an Outcome's shape. Callers pass the number of
// contributing Outcomes the store actually holds, never a stored flag.
func ShapeForChildCount(children int) OutcomeShape {
	if children > 0 {
		return OutcomeShapeDecomposed
	}
	return OutcomeShapeDirect
}

// ContributionLinkID identifies one immutable criterion binding.
type ContributionLinkID string

// IsZero reports whether the id is unset or blank.
func (id ContributionLinkID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// String returns the raw identifier.
func (id ContributionLinkID) String() string {
	return string(id)
}

// ContributionLink binds one contributing Outcome to one criterion of its
// parent's contract, at an exact parent ContractRevision.
//
// Contribution is criterion-bound or it is decorative: without this binding a
// parent's proof roll-up would be an assertion that some children exist, not
// evidence that its criteria became true. Links are append-only, and every
// link for one child names the same parent revision — that shared revision is
// the child's binding, and the sole fact staleness is derived from.
type ContributionLink struct {
	ID                       ContributionLinkID
	ParentOutcomeID          OutcomeID
	ChildOutcomeID           OutcomeID
	ParentContractRevisionID ContractRevisionID
	ParentCriterionID        CriterionID
	CreatedAt                time.Time
}

// Validate checks intrinsic link invariants. Whether the criterion belongs to
// the named revision, and the revision to the named parent, is a referential
// question storage answers.
func (l ContributionLink) Validate() error {
	switch {
	case l.ID.IsZero():
		return fmt.Errorf("contribution link id is required")
	case l.ParentOutcomeID.IsZero():
		return fmt.Errorf("contribution link parent outcome id is required")
	case l.ChildOutcomeID.IsZero():
		return fmt.Errorf("contribution link child outcome id is required")
	case l.ParentContractRevisionID.IsZero():
		return fmt.Errorf("contribution link parent contract revision id is required")
	case l.ParentCriterionID.IsZero():
		return fmt.Errorf("contribution link parent criterion id is required")
	case l.ParentOutcomeID == l.ChildOutcomeID:
		return fmt.Errorf("contribution link cannot bind an outcome to itself")
	}
	return nil
}

// ValidateContributionLinkSet checks the invariants that only hold across a
// whole child's bindings: at least one criterion, one consistent parent, one
// consistent parent revision, and no duplicate criterion.
//
// The single-revision rule is what makes staleness decidable. A child with
// links spanning two parent revisions would be simultaneously current and
// superseded, and nothing downstream could say which.
func ValidateContributionLinkSet(child OutcomeID, links []ContributionLink) error {
	if len(links) == 0 {
		return fmt.Errorf("a contributing outcome must claim at least one parent criterion")
	}
	seen := make(map[CriterionID]struct{}, len(links))
	for i, link := range links {
		if err := link.Validate(); err != nil {
			return fmt.Errorf("contribution link %d: %w", i+1, err)
		}
		if link.ChildOutcomeID != child {
			return fmt.Errorf("contribution link %d belongs to outcome %q, not %q", i+1, link.ChildOutcomeID, child)
		}
		if link.ParentOutcomeID != links[0].ParentOutcomeID {
			return fmt.Errorf("contribution links name two different parents (%q and %q)", links[0].ParentOutcomeID, link.ParentOutcomeID)
		}
		if link.ParentContractRevisionID != links[0].ParentContractRevisionID {
			return fmt.Errorf("contribution links span two parent contract revisions (%q and %q); a child binds to exactly one",
				links[0].ParentContractRevisionID, link.ParentContractRevisionID)
		}
		if _, dup := seen[link.ParentCriterionID]; dup {
			return fmt.Errorf("criterion %q is claimed twice by the same contributing outcome", link.ParentCriterionID)
		}
		seen[link.ParentCriterionID] = struct{}{}
	}
	return nil
}

// AuthorityWidenings names every authority a child claims that its parent does
// not hold. Effective authority is the intersection of every layer: a lower
// layer may narrow it and may never widen it, and composition adds one layer.
//
// The offenders are returned rather than a bare boolean so a refusal can say
// which authority was over-claimed instead of failing unexplainably.
func AuthorityWidenings(parent, child ProposedAuthority) []string {
	widened := make([]string, 0, 8)
	check := func(name string, parentHas, childWants bool) {
		if childWants && !parentHas {
			widened = append(widened, name)
		}
	}
	check("readWorkspace", parent.ReadWorkspace, child.ReadWorkspace)
	check("writeWorkspace", parent.WriteWorkspace, child.WriteWorkspace)
	check("executeLocal", parent.ExecuteLocal, child.ExecuteLocal)
	check("useNetwork", parent.UseNetwork, child.UseNetwork)
	check("commitLocal", parent.CommitLocal, child.CommitLocal)
	check("createPR", parent.CreatePR, child.CreatePR)
	check("deploy", parent.Deploy, child.Deploy)
	check("externalEffect", parent.ExternalEffect, child.ExternalEffect)
	return widened
}

// AuthorityContains fails closed when a contributing Outcome's ceiling exceeds
// its parent's, naming every over-claimed authority.
func AuthorityContains(parent, child ProposedAuthority) error {
	if widened := AuthorityWidenings(parent, child); len(widened) > 0 {
		return fmt.Errorf("contributing outcome widens parent authority: %s", strings.Join(widened, ", "))
	}
	return nil
}

// CriterionClaim is one parent criterion and the contributing Outcomes that
// claim it. Nothing claiming a criterion is a truthful report, not an error:
// coverage becomes a gate at decomposition authorization and again at parent
// acceptance, so a partly decomposed parent reads as incomplete, not invalid.
type CriterionClaim struct {
	CriterionID CriterionID
	Position    int64
	Text        string
	ClaimedBy   []OutcomeID
}

// Claimed reports whether any contributing Outcome claims this criterion.
func (c CriterionClaim) Claimed() bool { return len(c.ClaimedBy) > 0 }

// CriterionCoverage projects a decomposed Outcome's current criteria against
// the links its contributing Outcomes hold. It answers the only question that
// makes a decomposition trustworthy: is every criterion someone's job?
//
// Links bound to a superseded parent revision are ignored, so a contract
// revision truthfully drops coverage back to unclaimed rather than carrying
// stale claims forward as if they still applied.
func CriterionCoverage(current ContractRevision, links []ContributionLink) []CriterionClaim {
	claims := make([]CriterionClaim, 0, len(current.Criteria))
	byCriterion := make(map[CriterionID][]OutcomeID, len(current.Criteria))
	for _, link := range links {
		if link.ParentContractRevisionID != current.ID {
			continue
		}
		byCriterion[link.ParentCriterionID] = append(byCriterion[link.ParentCriterionID], link.ChildOutcomeID)
	}
	for _, criterion := range current.Criteria {
		children := byCriterion[criterion.ID]
		sort.Slice(children, func(i, j int) bool { return children[i] < children[j] })
		claims = append(claims, CriterionClaim{
			CriterionID: criterion.ID,
			Position:    criterion.Position,
			Text:        criterion.Text,
			ClaimedBy:   children,
		})
	}
	sort.Slice(claims, func(i, j int) bool { return claims[i].Position < claims[j].Position })
	return claims
}

// UnclaimedCriteria returns the criteria no contributing Outcome claims under
// the current parent revision.
func UnclaimedCriteria(current ContractRevision, links []ContributionLink) []CriterionClaim {
	unclaimed := make([]CriterionClaim, 0)
	for _, claim := range CriterionCoverage(current, links) {
		if !claim.Claimed() {
			unclaimed = append(unclaimed, claim)
		}
	}
	return unclaimed
}

// ContributionStale reports whether a contributing Outcome is bound to a
// superseded parent revision.
//
// Staleness blocks new authorization only. A running Attempt keeps its
// tactical freedom and reconciles at its own fence: a superseded parent
// contract is not proof that in-flight work is dead.
func ContributionStale(parentCurrent ContractRevision, links []ContributionLink) bool {
	for _, link := range links {
		if link.ParentContractRevisionID != parentCurrent.ID {
			return true
		}
	}
	return false
}

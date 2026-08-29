package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// DecompositionRevisionID identifies one immutable decomposition proposal.
type DecompositionRevisionID string

// IsZero reports whether the id is unset or blank.
func (id DecompositionRevisionID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// String returns the raw identifier.
func (id DecompositionRevisionID) String() string { return string(id) }

// DecompositionStatus mirrors PlanStatus: a decomposition is validated when
// proposed and becomes real only when the owner authorizes it.
type DecompositionStatus string

const (
	// DecompositionProposed marks a validated but unauthorized decomposition.
	// Nothing exists yet — no contributing Outcome, no contract, no binding.
	DecompositionProposed DecompositionStatus = "proposed"
	// DecompositionAuthorized marks the owner decision that created the
	// contributing Outcomes. Authorization is final for this revision;
	// supersession happens through a new parent contract revision.
	DecompositionAuthorized DecompositionStatus = "authorized"
)

// Valid reports whether s is a supported decomposition status.
func (s DecompositionStatus) Valid() bool {
	return s == DecompositionProposed || s == DecompositionAuthorized
}

// ProposedContribution is one contributing Outcome as proposed, before it
// exists. It carries the whole contract the child would own, because a
// contributing Outcome is a full responsibility rather than a task — the owner
// has to be able to correct its goal and criteria before agreeing to it.
//
// Ref is a caller-supplied handle unique within the proposal. Dependencies
// address it, because the real OutcomeID does not exist until authorization.
type ProposedContribution struct {
	Ref             string
	Position        int64
	Title           string
	Goal            string
	SuccessCriteria []string
	Review          string
	Constraints     []string
	NonGoals        []string
	Authority       ProposedAuthority
	// ClaimedCriteria names the parent criteria this contribution would prove.
	ClaimedCriteria []CriterionID
	// ChildOutcomeID is empty until authorization resolves the proposal into
	// a real Outcome.
	ChildOutcomeID OutcomeID
}

// Validate checks one proposed contribution in isolation.
func (c ProposedContribution) Validate() error {
	switch {
	case strings.TrimSpace(c.Ref) == "":
		return fmt.Errorf("proposed contribution ref is required")
	case c.Position < 1:
		return fmt.Errorf("proposed contribution %q position must be at least 1", c.Ref)
	case strings.TrimSpace(c.Title) == "":
		return fmt.Errorf("proposed contribution %q title is required", c.Ref)
	case strings.TrimSpace(c.Goal) == "":
		return fmt.Errorf("proposed contribution %q goal is required", c.Ref)
	case strings.TrimSpace(c.Review) == "":
		return fmt.Errorf("proposed contribution %q review method is required", c.Ref)
	case len(c.SuccessCriteria) == 0:
		return fmt.Errorf("proposed contribution %q requires at least one success criterion", c.Ref)
	case len(c.ClaimedCriteria) == 0:
		return fmt.Errorf("proposed contribution %q claims no parent criterion", c.Ref)
	}
	for i, criterion := range c.SuccessCriteria {
		if strings.TrimSpace(criterion) == "" {
			return fmt.Errorf("proposed contribution %q success criterion %d is blank", c.Ref, i+1)
		}
	}
	seen := make(map[CriterionID]struct{}, len(c.ClaimedCriteria))
	for _, id := range c.ClaimedCriteria {
		if id.IsZero() {
			return fmt.Errorf("proposed contribution %q claims a blank criterion", c.Ref)
		}
		if _, dup := seen[id]; dup {
			return fmt.Errorf("proposed contribution %q claims criterion %q twice", c.Ref, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

// ContributionDependency is a declared ordering between two sibling
// contributions inside one decomposition. It gates when a contributor's first
// Attempt may start; it never blocks reading, planning, or correcting.
type ContributionDependency struct {
	ID string
	// FromRef must finish before ToRef starts.
	FromRef string
	ToRef   string
}

// Validate checks intrinsic dependency invariants.
func (d ContributionDependency) Validate() error {
	switch {
	case strings.TrimSpace(d.ID) == "":
		return fmt.Errorf("contribution dependency id is required")
	case strings.TrimSpace(d.FromRef) == "":
		return fmt.Errorf("contribution dependency source ref is required")
	case strings.TrimSpace(d.ToRef) == "":
		return fmt.Errorf("contribution dependency target ref is required")
	case d.FromRef == d.ToRef:
		return fmt.Errorf("contribution %q cannot depend on itself", d.FromRef)
	}
	return nil
}

// DecompositionRevision is one immutable proposal-and-authorization record: the
// contributing Outcomes a parent would be pursued through, the parent criteria
// each would prove, the criteria the owner keeps, and the order between them —
// all frozen against one parent ContractRevision.
//
// It is the decomposed Outcome's plan. A decomposed Outcome has no
// PlanRevision and no Attempt of its own (ADR 0007).
type DecompositionRevision struct {
	ID        DecompositionRevisionID
	OutcomeID OutcomeID
	Number    int64
	// ContractRevisionID freezes which parent contract this decomposition
	// answers. A later parent revision supersedes it.
	ContractRevisionID ContractRevisionID
	Status             DecompositionStatus
	// Rationale is the plain-language explanation of the topology. It is
	// required: a decomposition the owner cannot evaluate is not reviewable.
	Rationale    string
	Contributors []ProposedContribution
	// RetainedCriteria are the parent criteria no contributor claims because
	// the owner proves them directly. Retention decides who proves a
	// criterion, never whether it is proved.
	RetainedCriteria []CriterionID
	Dependencies     []ContributionDependency
	CreatedAt        time.Time
	AuthorizedAt     *time.Time
}

// Validate checks intrinsic decomposition invariants. Coverage against the
// parent contract and criterion existence are checked separately, because they
// need the parent revision this type deliberately does not carry.
func (d DecompositionRevision) Validate() error {
	switch {
	case d.ID.IsZero():
		return fmt.Errorf("decomposition revision id is required")
	case d.OutcomeID.IsZero():
		return fmt.Errorf("decomposition revision outcome id is required")
	case d.Number < 1:
		return fmt.Errorf("decomposition revision number must be at least 1")
	case d.ContractRevisionID.IsZero():
		return fmt.Errorf("decomposition revision must bind a parent contract revision")
	case !d.Status.Valid():
		return fmt.Errorf("unsupported decomposition status %q", d.Status)
	case strings.TrimSpace(d.Rationale) == "":
		return fmt.Errorf("decomposition revision requires a plain-language rationale")
	case len(d.Contributors) == 0:
		return fmt.Errorf("decomposition revision requires at least one contributing outcome")
	}

	refs := make(map[string]struct{}, len(d.Contributors))
	claimed := make(map[CriterionID]string, len(d.Contributors))
	for _, contributor := range d.Contributors {
		if err := contributor.Validate(); err != nil {
			return err
		}
		if _, dup := refs[contributor.Ref]; dup {
			return fmt.Errorf("decomposition names contribution ref %q twice", contributor.Ref)
		}
		refs[contributor.Ref] = struct{}{}
		for _, criterion := range contributor.ClaimedCriteria {
			if owner, taken := claimed[criterion]; taken && owner != contributor.Ref {
				continue // two contributors may both prove one criterion
			}
			claimed[criterion] = contributor.Ref
		}
	}
	for _, criterion := range d.RetainedCriteria {
		if criterion.IsZero() {
			return fmt.Errorf("decomposition retains a blank criterion")
		}
		// Retaining a criterion a contributor already claims is contradictory:
		// it says the owner proves it AND delegates it.
		if ref, taken := claimed[criterion]; taken {
			return fmt.Errorf("criterion %q is both retained by the owner and claimed by contribution %q", criterion, ref)
		}
	}
	for _, dependency := range d.Dependencies {
		if err := dependency.Validate(); err != nil {
			return err
		}
		if _, ok := refs[dependency.FromRef]; !ok {
			return fmt.Errorf("dependency names unknown contribution %q", dependency.FromRef)
		}
		if _, ok := refs[dependency.ToRef]; !ok {
			return fmt.Errorf("dependency names unknown contribution %q", dependency.ToRef)
		}
	}
	return ValidateDependencyOrdering(d.Contributors, d.Dependencies)
}

// ValidateDependencyOrdering rejects dependency cycles.
//
// The composition depth cap makes parent/child cycles unreachable, but sibling
// dependencies are a genuine directed graph and can loop. A cycle would make
// every contributor in it permanently ungatable — each waiting on the next —
// so it is refused at authorization rather than discovered at Attempt start.
func ValidateDependencyOrdering(contributors []ProposedContribution, dependencies []ContributionDependency) error {
	if len(dependencies) == 0 {
		return nil
	}
	edges := make(map[string][]string, len(contributors))
	for _, dependency := range dependencies {
		edges[dependency.FromRef] = append(edges[dependency.FromRef], dependency.ToRef)
	}
	for ref := range edges {
		sort.Strings(edges[ref])
	}

	const (
		unvisited = 0
		onStack   = 1
		done      = 2
	)
	state := make(map[string]int, len(contributors))
	var path []string

	var walk func(ref string) error
	walk = func(ref string) error {
		state[ref] = onStack
		path = append(path, ref)
		for _, next := range edges[ref] {
			switch state[next] {
			case onStack:
				// Report the cycle itself, not just its existence: the owner
				// has to know which ordering to cut.
				start := 0
				for i, entry := range path {
					if entry == next {
						start = i
						break
					}
				}
				return fmt.Errorf("contribution dependencies form a cycle: %s -> %s",
					strings.Join(path[start:], " -> "), next)
			case unvisited:
				if err := walk(next); err != nil {
					return err
				}
			}
		}
		path = path[:len(path)-1]
		state[ref] = done
		return nil
	}

	refs := make([]string, 0, len(contributors))
	for _, contributor := range contributors {
		refs = append(refs, contributor.Ref)
	}
	sort.Strings(refs)
	for _, ref := range refs {
		if state[ref] == unvisited {
			if err := walk(ref); err != nil {
				return err
			}
		}
	}
	return nil
}

// UncoveredCriteria returns the parent criteria a decomposition neither
// delegates nor retains.
//
// This is the gate that makes a decomposition trustworthy. An unclassified
// criterion is exactly how a project would report itself done while missing
// something material, so authorization refuses on any non-empty result. There
// is no third state: a criterion is someone's job, or the owner's.
func UncoveredCriteria(current ContractRevision, contributors []ProposedContribution, retained []CriterionID) []ContractCriterion {
	accounted := make(map[CriterionID]struct{}, len(current.Criteria))
	for _, contributor := range contributors {
		for _, criterion := range contributor.ClaimedCriteria {
			accounted[criterion] = struct{}{}
		}
	}
	for _, criterion := range retained {
		accounted[criterion] = struct{}{}
	}
	uncovered := make([]ContractCriterion, 0)
	for _, criterion := range current.Criteria {
		if _, ok := accounted[criterion.ID]; !ok {
			uncovered = append(uncovered, criterion)
		}
	}
	return uncovered
}

// UnknownClaimedCriteria returns every criterion identity a decomposition
// references that the parent's current revision does not define. Binding to a
// criterion that no longer exists would let a contribution prove something the
// parent stopped asking for.
func UnknownClaimedCriteria(current ContractRevision, contributors []ProposedContribution, retained []CriterionID) []CriterionID {
	known := make(map[CriterionID]struct{}, len(current.Criteria))
	for _, criterion := range current.Criteria {
		known[criterion.ID] = struct{}{}
	}
	seen := make(map[CriterionID]struct{})
	unknown := make([]CriterionID, 0)
	consider := func(id CriterionID) {
		if _, ok := known[id]; ok {
			return
		}
		if _, dup := seen[id]; dup {
			return
		}
		seen[id] = struct{}{}
		unknown = append(unknown, id)
	}
	for _, contributor := range contributors {
		for _, criterion := range contributor.ClaimedCriteria {
			consider(criterion)
		}
	}
	for _, criterion := range retained {
		consider(criterion)
	}
	return unknown
}

// OverClaimedAuthority returns each contribution ref whose proposed ceiling
// exceeds the parent's, with the authorities it over-claims.
func OverClaimedAuthority(parent ProposedAuthority, contributors []ProposedContribution) map[string][]string {
	offenders := make(map[string][]string)
	for _, contributor := range contributors {
		if widened := AuthorityWidenings(parent, contributor.Authority); len(widened) > 0 {
			offenders[contributor.Ref] = widened
		}
	}
	return offenders
}

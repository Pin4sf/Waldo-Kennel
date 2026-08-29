package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ContributionWaiverID identifies one immutable dependency waiver.
type ContributionWaiverID string

// IsZero reports whether the id is unset or blank.
func (id ContributionWaiverID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// ContributionDependencyWaiver is the owner's explicit decision to start a
// contributing Outcome before a declared upstream sibling is accepted.
//
// It is durable and attributable on purpose. Overriding an ordering the owner
// themselves authorized is a real decision about risk, not a UI convenience,
// and it must remain readable afterwards next to whatever the early start
// produced. Waivers are append-only; withdrawing one is a new decomposition.
type ContributionDependencyWaiver struct {
	ID              ContributionWaiverID
	DecompositionID DecompositionRevisionID
	FromRef         string
	ToRef           string
	// Reason is required: a waiver nobody can explain later is indistinguishable
	// from a mistake.
	Reason    string
	WaivedBy  AcceptanceActorType
	CreatedAt time.Time
}

// Validate checks intrinsic waiver invariants.
func (w ContributionDependencyWaiver) Validate() error {
	switch {
	case w.ID.IsZero():
		return fmt.Errorf("contribution waiver id is required")
	case w.DecompositionID.IsZero():
		return fmt.Errorf("contribution waiver decomposition id is required")
	case strings.TrimSpace(w.FromRef) == "":
		return fmt.Errorf("contribution waiver source ref is required")
	case strings.TrimSpace(w.ToRef) == "":
		return fmt.Errorf("contribution waiver target ref is required")
	case strings.TrimSpace(w.Reason) == "":
		return fmt.Errorf("a dependency waiver requires a reason")
	case w.WaivedBy != AcceptanceActorUser:
		// Same rule as acceptance: only the owner may override their own
		// declared ordering. No automated actor may waive.
		return fmt.Errorf("only the user may waive a contribution dependency")
	}
	return nil
}

// UpstreamBlock names one unmet dependency standing between a contributing
// Outcome and its first Attempt.
type UpstreamBlock struct {
	// Ref is the upstream contribution's handle inside the decomposition.
	Ref string
	// OutcomeID is the upstream contributing Outcome, empty when the
	// decomposition was authorized but that contributor cannot be resolved.
	OutcomeID OutcomeID
	Title     string
	// Reason states why it does not clear the gate, in the owner's terms.
	Reason string
}

// ContributionStartGate is the answer to "may this contributing Outcome start
// work yet?", derived from durable facts alone.
type ContributionStartGate struct {
	// Applies is false for a Project-level or standalone Outcome: gating is a
	// property of contributing to something, not of Outcomes in general.
	Applies bool
	Blocked []UpstreamBlock
	// Waived lists upstreams that would block but for an explicit owner
	// waiver. They stay visible: a waived dependency is overridden, not gone.
	Waived []UpstreamBlock
}

// Clear reports whether execution may proceed.
func (g ContributionStartGate) Clear() bool { return len(g.Blocked) == 0 }

// BlockedRefs lists the unmet upstream refs in stable order.
func (g ContributionStartGate) BlockedRefs() []string {
	refs := make([]string, 0, len(g.Blocked))
	for _, block := range g.Blocked {
		refs = append(refs, block.Ref)
	}
	sort.Strings(refs)
	return refs
}

// AcceptedOutcomes reports which of the given Outcomes carry a current owner
// acceptance, from their decision history.
//
// The LAST decision wins: an Outcome accepted and later reopened is not
// accepted, and treating it as accepted would let downstream work start on a
// result the owner withdrew.
func AcceptedOutcomes(decisions map[OutcomeID][]AcceptanceDecision) map[OutcomeID]bool {
	accepted := make(map[OutcomeID]bool, len(decisions))
	for id, history := range decisions {
		accepted[id] = LatestDecisionAccepts(history)
	}
	return accepted
}

// LatestDecisionAccepts reports whether the newest decision in a history is an
// acceptance. An empty history is not an acceptance.
func LatestDecisionAccepts(history []AcceptanceDecision) bool {
	newest := AcceptanceDecision{}
	found := false
	for _, decision := range history {
		if !found || decision.CreatedAt.After(newest.CreatedAt) {
			newest, found = decision, true
		}
	}
	return found && newest.Kind == AcceptanceAccept
}

// DeriveContributionStartGate computes whether one contributing Outcome may
// begin, from the authorized decomposition, the owner's waivers, and which
// sibling Outcomes are currently accepted.
//
// Only an AUTHORIZED decomposition gates anything. A proposal is an offer, and
// an offer must not be able to block real work.
func DeriveContributionStartGate(
	child OutcomeID,
	decomposition DecompositionRevision,
	waivers []ContributionDependencyWaiver,
	accepted map[OutcomeID]bool,
) ContributionStartGate {
	if decomposition.Status != DecompositionAuthorized {
		return ContributionStartGate{}
	}
	ref, ok := contributionRefFor(child, decomposition)
	if !ok {
		return ContributionStartGate{}
	}

	byRef := make(map[string]ProposedContribution, len(decomposition.Contributors))
	for _, contributor := range decomposition.Contributors {
		byRef[contributor.Ref] = contributor
	}
	waived := make(map[string]struct{}, len(waivers))
	for _, waiver := range waivers {
		if waiver.ToRef == ref {
			waived[waiver.FromRef] = struct{}{}
		}
	}

	gate := ContributionStartGate{Applies: true}
	upstreams := make([]string, 0, len(decomposition.Dependencies))
	for _, dependency := range decomposition.Dependencies {
		if dependency.ToRef == ref {
			upstreams = append(upstreams, dependency.FromRef)
		}
	}
	sort.Strings(upstreams)

	for _, upstream := range upstreams {
		contributor := byRef[upstream]
		block := UpstreamBlock{Ref: upstream, OutcomeID: contributor.ChildOutcomeID, Title: contributor.Title}
		switch {
		case contributor.ChildOutcomeID.IsZero():
			block.Reason = "that contribution has no Outcome yet"
		case accepted[contributor.ChildOutcomeID]:
			continue // upstream is done; nothing to report
		default:
			block.Reason = "waiting for your acceptance"
		}
		if _, isWaived := waived[upstream]; isWaived {
			gate.Waived = append(gate.Waived, block)
			continue
		}
		gate.Blocked = append(gate.Blocked, block)
	}
	return gate
}

// contributionRefFor resolves a contributing Outcome to its handle inside an
// authorized decomposition.
func contributionRefFor(child OutcomeID, decomposition DecompositionRevision) (string, bool) {
	for _, contributor := range decomposition.Contributors {
		if contributor.ChildOutcomeID == child && !child.IsZero() {
			return contributor.Ref, true
		}
	}
	return "", false
}

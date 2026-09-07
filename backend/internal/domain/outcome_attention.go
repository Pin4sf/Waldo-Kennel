package domain

import "sort"

// AttentionKind classifies what a parent Outcome needs from its owner, using
// the launch attention vocabulary. The kinds are ordered by urgency so a
// roll-up can present the most demanding item first without inventing scores.
type AttentionKind string

const (
	// AttentionNeedsYou means a live session is blocked on its user.
	AttentionNeedsYou AttentionKind = "needs_you"
	// AttentionActionRequired means one exact human-only action is necessary —
	// unproven custody, an unclassified end, a failed start.
	AttentionActionRequired AttentionKind = "action_required"
	// AttentionReadyForAcceptance means a contributor's criteria have their
	// evidence and verification and await the owner's decision.
	AttentionReadyForAcceptance AttentionKind = "ready_for_acceptance"
	// AttentionWaiting means a dependency or condition is unresolved and no
	// immediate action helps.
	AttentionWaiting AttentionKind = "waiting"
	// AttentionRunning means work is proceeding and needs nothing.
	AttentionRunning AttentionKind = "running"
	// AttentionAccepted means the contributor is closed.
	AttentionAccepted AttentionKind = "accepted"
)

// attentionRank orders kinds most-demanding first. Presentation reads this;
// nothing durable does.
var attentionRank = map[AttentionKind]int{
	AttentionNeedsYou:           0,
	AttentionActionRequired:     1,
	AttentionReadyForAcceptance: 2,
	AttentionWaiting:            3,
	AttentionRunning:            4,
	AttentionAccepted:           5,
}

// Rank returns the urgency order of a kind; unknown kinds sort last.
func (k AttentionKind) Rank() int {
	if rank, ok := attentionRank[k]; ok {
		return rank
	}
	return len(attentionRank)
}

// ContributorAttention is one roll-up item. It always names the contributing
// Outcome it came from: a parent that merged its contributors' attention into
// an undifferentiated feed would be a second activity dashboard, which is
// exactly what the Outcome model exists to replace.
type ContributorAttention struct {
	OutcomeID OutcomeID     `json:"outcomeId"`
	Title     string        `json:"title"`
	Kind      AttentionKind `json:"kind"`
	// Reason states the situation in the owner's terms, without requiring a
	// transcript to interpret.
	Reason string `json:"reason"`
	// NextAction is the smallest safe thing to do, or empty when nothing is
	// useful yet.
	NextAction string `json:"nextAction,omitempty"`
}

// ContributorFacts is everything the roll-up needs about one contributing
// Outcome. It is assembled from durable facts by the service; the derivation
// below stays pure so it is testable without a store.
type ContributorFacts struct {
	OutcomeID OutcomeID
	Title     string
	// Accepted reports a current owner acceptance.
	Accepted bool
	// Stale reports a binding to a superseded parent contract revision.
	Stale bool
	// Gate reports unmet upstream dependencies.
	Gate ContributionStartGate
	// Attempts carries the derived presentation of every attempt, newest last.
	Attempts []AttemptPresentation
	// ReadyForAcceptance reports that proof is complete and the owner's
	// decision is the only thing left.
	ReadyForAcceptance bool
}

// DeriveContributorAttention classifies one contributor. Precedence is
// deliberate and fixed: a live session blocked on its user outranks everything,
// and "accepted" is only reported when nothing else is outstanding.
func DeriveContributorAttention(facts ContributorFacts) ContributorAttention {
	item := ContributorAttention{OutcomeID: facts.OutcomeID, Title: facts.Title}

	// A running attempt's own derived phase is the most truthful signal there
	// is, so it wins over anything inferred from surrounding state.
	for _, attempt := range facts.Attempts {
		switch attempt.Phase {
		case AttemptPhaseNeedsInput:
			item.Kind = AttentionNeedsYou
			item.Reason = "the running session is blocked on you (" + attempt.Attention + ")"
			item.NextAction = attempt.NextAction
			return item
		case AttemptPhaseUnconfirmed, AttemptPhaseSuspectLost, AttemptPhaseHaltedFailed, AttemptPhaseEndedUnclassified:
			item.Kind = AttentionActionRequired
			item.Reason = "this contribution's attempt is " + string(attempt.Phase)
			item.NextAction = attempt.NextAction
			return item
		}
	}

	switch {
	case facts.Accepted:
		item.Kind = AttentionAccepted
		item.Reason = "accepted"
	case facts.Stale:
		// Stale blocks new authorization, so it needs the owner before it
		// needs anything else.
		item.Kind = AttentionActionRequired
		item.Reason = "bound to a superseded parent contract revision"
		item.NextAction = "Re-bind this contribution to the current contract"
	case facts.ReadyForAcceptance:
		item.Kind = AttentionReadyForAcceptance
		item.Reason = "criteria are proved and await your decision"
		item.NextAction = "Review the evidence and accept or reopen"
	case !facts.Gate.Clear():
		item.Kind = AttentionWaiting
		item.Reason = "waiting on " + joinBlocks(facts.Gate.Blocked)
		item.NextAction = "Accept the upstream contribution, or waive the dependency"
	case hasRunningAttempt(facts.Attempts):
		item.Kind = AttentionRunning
		item.Reason = "work is running"
	default:
		item.Kind = AttentionWaiting
		item.Reason = "no attempt has started"
		item.NextAction = "Start work on this contribution"
	}
	return item
}

// RollUpContributorAttention orders contributors most-demanding first, keeping
// a stable order within a kind so the surface does not reshuffle on refresh.
func RollUpContributorAttention(facts []ContributorFacts) []ContributorAttention {
	items := make([]ContributorAttention, 0, len(facts))
	for _, contributor := range facts {
		items = append(items, DeriveContributorAttention(contributor))
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind.Rank() != items[j].Kind.Rank() {
			return items[i].Kind.Rank() < items[j].Kind.Rank()
		}
		return items[i].OutcomeID < items[j].OutcomeID
	})
	return items
}

// ParentAttention summarises a decomposed Outcome for its owner.
type ParentAttention struct {
	// Headline is the most demanding kind across contributors, or empty when
	// there are none.
	Headline     AttentionKind          `json:"headline,omitempty"`
	Items        []ContributorAttention `json:"items"`
	Counts       map[AttentionKind]int  `json:"counts"`
	AcceptedOf   int                    `json:"acceptedOf"`
	Contributors int                    `json:"contributors"`
}

// SummariseParentAttention builds the parent-level projection. It is derived
// at read time from durable facts and never stored.
func SummariseParentAttention(facts []ContributorFacts) ParentAttention {
	items := RollUpContributorAttention(facts)
	summary := ParentAttention{
		Items:        items,
		Counts:       make(map[AttentionKind]int, len(attentionRank)),
		Contributors: len(facts),
	}
	for _, item := range items {
		summary.Counts[item.Kind]++
		if item.Kind == AttentionAccepted {
			summary.AcceptedOf++
		}
	}
	if len(items) > 0 {
		summary.Headline = items[0].Kind
	}
	return summary
}

func hasRunningAttempt(attempts []AttemptPresentation) bool {
	for _, attempt := range attempts {
		if attempt.Phase == AttemptPhaseExecuting || attempt.Phase == AttemptPhaseAwaitingStart {
			return true
		}
	}
	return false
}

func joinBlocks(blocks []UpstreamBlock) string {
	if len(blocks) == 0 {
		return "nothing"
	}
	names := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block.Title != "" {
			names = append(names, block.Title)
			continue
		}
		names = append(names, block.Ref)
	}
	sort.Strings(names)
	out := names[0]
	for _, name := range names[1:] {
		out += ", " + name
	}
	return out
}

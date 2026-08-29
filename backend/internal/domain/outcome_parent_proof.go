package domain

import "sort"

// IndependenceSatisfies reports whether an actual verification class meets a
// required minimum.
//
// It deliberately refuses to impose a total order on the classes, because they
// are not comparable on one axis: a deterministic check and an owner
// walkthrough are strong in different ways, and ranking them would encode a
// judgment the architecture never made. Exactly one threshold is real —
// whether the verifier was independent of the producer — so:
//
//   - a minimum of producer_self_check accepts anything;
//   - a minimum of separate_session means "independent of the producer" and
//     accepts every class except producer_self_check;
//   - any other minimum is an EXACT requirement for that class.
func IndependenceSatisfies(actual, minimum VerificationIndependenceClass) bool {
	switch minimum {
	case VerificationProducerSelfCheck, "":
		return actual != ""
	case VerificationSeparateSession:
		return actual != "" && actual != VerificationProducerSelfCheck
	default:
		return actual == minimum
	}
}

// MinimumBatchIndependence is the bar a contributing Outcome's proof must clear
// to enter a batched acceptance: independent of whoever produced the work.
//
// Producer self-checks are useful Evidence but are not independent
// verification, and a batch is precisely where a weak check is most likely to
// pass unexamined among stronger neighbours.
const MinimumBatchIndependence = VerificationSeparateSession

// WeakestIndependence returns the weakest class in a set, where "weakest" means
// producer_self_check if present and otherwise the first class in stable order.
// An empty set returns the empty class, which satisfies no minimum.
func WeakestIndependence(classes []VerificationIndependenceClass) VerificationIndependenceClass {
	if len(classes) == 0 {
		return ""
	}
	ordered := make([]string, 0, len(classes))
	for _, class := range classes {
		if class == VerificationProducerSelfCheck {
			return VerificationProducerSelfCheck
		}
		ordered = append(ordered, string(class))
	}
	sort.Strings(ordered)
	return VerificationIndependenceClass(ordered[0])
}

// ContributorProofFacts is what a batched acceptance needs to know about one
// contributing Outcome. Every field is derived from durable proof facts.
type ContributorProofFacts struct {
	OutcomeID OutcomeID
	Title     string
	// Ready reports that every current criterion has supporting Evidence and
	// a passing Verification.
	Ready bool
	// Accepted reports a current owner acceptance; an accepted contributor is
	// already closed and does not re-enter a batch.
	Accepted bool
	// Contradicted reports that a criterion's latest Evidence contradicts it.
	Contradicted bool
	// Stale reports a binding to a superseded parent contract revision.
	Stale bool
	// BackingIndependence is the weakest class among the verifications that
	// actually make this contributor ready. Empty means nothing backs it.
	BackingIndependence VerificationIndependenceClass
}

// BatchEntryVerdict is the daemon's answer to "may this contributing Outcome
// enter the owner's batched acceptance?".
//
// The daemon's only power over a batch is exclusion. It may withhold a
// contributor whose proof does not hold up; it may never accept one.
type BatchEntryVerdict struct {
	OutcomeID OutcomeID `json:"outcomeId"`
	Title     string    `json:"title"`
	Eligible  bool      `json:"eligible"`
	// Reason explains an exclusion in the owner's terms, or states why the
	// contributor is ready when eligible.
	Reason string `json:"reason"`
	// Remedy names the smallest thing that would make it eligible.
	Remedy string `json:"remedy,omitempty"`
}

// EligibleForAcceptanceBatch decides whether one contributing Outcome may enter
// the owner's batched acceptance, and says why when it may not.
//
// Exclusion is not rejection: an excluded contributor is escalated on its own,
// with the smallest remedy named. Batching collapses the owner's keystrokes,
// never their authority, so the bar for entering the batch has to be exactly
// the bar for an individual acceptance.
func EligibleForAcceptanceBatch(facts ContributorProofFacts, minimum VerificationIndependenceClass) BatchEntryVerdict {
	verdict := BatchEntryVerdict{OutcomeID: facts.OutcomeID, Title: facts.Title}
	switch {
	case facts.Accepted:
		verdict.Reason = "already accepted"
	case facts.Contradicted:
		verdict.Reason = "its latest Evidence contradicts a criterion"
		verdict.Remedy = "Resolve the contradiction, then record fresh Evidence"
	case facts.Stale:
		verdict.Reason = "bound to a superseded parent contract revision"
		verdict.Remedy = "Re-bind this contribution to the current contract"
	case !facts.Ready:
		verdict.Reason = "its criteria are not fully proved yet"
		verdict.Remedy = "Complete the remaining Evidence and Verification"
	case !IndependenceSatisfies(facts.BackingIndependence, minimum):
		verdict.Reason = "verified only by " + independenceLabel(facts.BackingIndependence) +
			", weaker than the " + independenceLabel(minimum) + " this batch requires"
		verdict.Remedy = "Run an independent verifier for this contribution"
	default:
		verdict.Eligible = true
		verdict.Reason = "criteria are proved and independently verified"
	}
	return verdict
}

func independenceLabel(class VerificationIndependenceClass) string {
	switch class {
	case "":
		return "nothing"
	case VerificationProducerSelfCheck:
		return "the producer's own self-check"
	case VerificationSeparateSession:
		return "separate-session review"
	case VerificationCrossProvider:
		return "cross-provider review"
	case VerificationDeterministic:
		return "a deterministic check"
	case VerificationOwnerWalkthrough:
		return "an owner walkthrough"
	}
	return string(class)
}

// DelegatedCriterion reports how one parent criterion is satisfied by the
// contributing Outcomes that claim it.
type DelegatedCriterion struct {
	CriterionID CriterionID
	ClaimedBy   []OutcomeID
	// Proved reports that EVERY claiming contributor is accepted. All of them,
	// because a criterion two contributors share is only true when both are.
	Proved bool
	Gap    string
}

// DelegatedCriteria maps each claimed parent criterion to its delegation
// status, from the contribution links bound to the parent's current revision
// and which contributors are currently accepted.
//
// Retained criteria deliberately do not appear: the owner proves those
// directly, so they stay on the parent's own evidence path.
func DelegatedCriteria(
	current ContractRevision,
	links []ContributionLink,
	accepted map[OutcomeID]bool,
	titles map[OutcomeID]string,
) map[CriterionID]DelegatedCriterion {
	delegated := make(map[CriterionID]DelegatedCriterion)
	for _, link := range links {
		if link.ParentContractRevisionID != current.ID {
			continue
		}
		entry := delegated[link.ParentCriterionID]
		entry.CriterionID = link.ParentCriterionID
		entry.ClaimedBy = append(entry.ClaimedBy, link.ChildOutcomeID)
		delegated[link.ParentCriterionID] = entry
	}
	for id, entry := range delegated {
		sort.Slice(entry.ClaimedBy, func(i, j int) bool { return entry.ClaimedBy[i] < entry.ClaimedBy[j] })
		outstanding := make([]string, 0, len(entry.ClaimedBy))
		for _, child := range entry.ClaimedBy {
			if !accepted[child] {
				label := titles[child]
				if label == "" {
					label = string(child)
				}
				outstanding = append(outstanding, label)
			}
		}
		entry.Proved = len(outstanding) == 0
		if !entry.Proved {
			entry.Gap = "Accept " + joinNames(outstanding) + " to prove this criterion."
		}
		delegated[id] = entry
	}
	return delegated
}

func joinNames(names []string) string {
	if len(names) == 0 {
		return "nothing"
	}
	sort.Strings(names)
	out := names[0]
	for _, name := range names[1:] {
		out += ", " + name
	}
	return out
}

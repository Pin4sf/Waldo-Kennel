package domain

// ReviewerHarness identifies the provider used for a code-review run. Kennel's
// review surface is intentionally bounded to the same five first-class provider
// integrations as worker execution; readiness remains a runtime fact.
type ReviewerHarness string

const (
	ReviewerClaudeCode ReviewerHarness = "claude-code"
	ReviewerCodex      ReviewerHarness = "codex"
	ReviewerCursor     ReviewerHarness = "cursor"
	ReviewerOpenCode   ReviewerHarness = "opencode"
	ReviewerPi         ReviewerHarness = "pi"
)

// AllReviewerHarnesses is the complete reviewer provider vocabulary shipped by
// Kennel. Historical donor reviewer identities are intentionally not retained.
var AllReviewerHarnesses = []ReviewerHarness{
	ReviewerCodex,
	ReviewerClaudeCode,
	ReviewerOpenCode,
	ReviewerCursor,
	ReviewerPi,
}

func (h ReviewerHarness) IsRecognizedPersisted() bool {
	for _, candidate := range AllReviewerHarnesses {
		if h == candidate {
			return true
		}
	}
	return false
}

// IsSelectableForNewWork reports build support only. Local binary/auth/profile
// readiness and reviewer execution errors remain fail-closed runtime checks.
func (h ReviewerHarness) IsSelectableForNewWork() bool {
	return h.IsRecognizedPersisted()
}

func (h ReviewerHarness) IsKnown() bool {
	return h.IsRecognizedPersisted()
}

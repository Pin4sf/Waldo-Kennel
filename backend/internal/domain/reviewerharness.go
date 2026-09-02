package domain

// ReviewerHarness identifies the provider used for a review run. Reviewer
// support is intentionally limited to the same five product providers Kennel
// ships for execution.
type ReviewerHarness string

const (
	ReviewerClaudeCode ReviewerHarness = "claude-code"
	ReviewerCodex      ReviewerHarness = "codex"
	ReviewerCursor     ReviewerHarness = "cursor"
	ReviewerOpenCode   ReviewerHarness = "opencode"
	ReviewerPi         ReviewerHarness = "pi"
)

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

// IsSelectableForNewWork reports build support only. Runtime installation,
// authentication and reviewer capability are separate admission checks.
func (h ReviewerHarness) IsSelectableForNewWork() bool {
	return h.IsRecognizedPersisted()
}

func (h ReviewerHarness) IsKnown() bool {
	return h.IsRecognizedPersisted()
}

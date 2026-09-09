package domain

// ReviewerHarness identifies the provider used for a code-review run. Kennel's
// review surface is intentionally bounded to the same five first-class provider
// integrations as worker execution; readiness remains a runtime fact.
type ReviewerHarness string

const (
	// ReviewerClaudeCode identifies Claude Code reviewer runs.
	ReviewerClaudeCode ReviewerHarness = "claude-code"
	// ReviewerCodex identifies Codex reviewer runs.
	ReviewerCodex ReviewerHarness = "codex"
	// ReviewerCursor identifies Cursor reviewer runs.
	ReviewerCursor ReviewerHarness = "cursor"
	// ReviewerOpenCode identifies OpenCode reviewer runs.
	ReviewerOpenCode ReviewerHarness = "opencode"
	// ReviewerPi identifies Pi reviewer runs.
	ReviewerPi ReviewerHarness = "pi"
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

// IsRecognizedPersisted reports whether the harness is in the persisted vocabulary.
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

// IsKnown reports whether the harness is recognized by the domain.
func (h ReviewerHarness) IsKnown() bool {
	return h.IsRecognizedPersisted()
}

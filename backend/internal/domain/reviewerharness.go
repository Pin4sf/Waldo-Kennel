package domain

// ReviewerHarness identifies a code-review agent. It is a separate vocabulary
// from AgentHarness on purpose: a reviewer-only tool (e.g. the Greptile CLI)
// must not become a valid worker, and a worker harness does not automatically
// become a valid reviewer. The two sets are maintained independently and only
// happen to share ids where the same tool serves both roles.
type ReviewerHarness string

// Supported reviewer harnesses. Add a reviewer-only tool here (and register its
// adapter) without widening the worker AgentHarness set.
const (
	ReviewerClaudeCode ReviewerHarness = "claude-code"
	ReviewerCodex      ReviewerHarness = "codex"
	ReviewerCursor     ReviewerHarness = "cursor"
	ReviewerOpenCode   ReviewerHarness = "opencode"

	// Retired reviewer identities remain for historical-row decoding only. They
	// are not selectable because they are absent from AllReviewerHarnesses.
	ReviewerCopilot  ReviewerHarness = "copilot"
	ReviewerKiloCode ReviewerHarness = "kilocode"
	ReviewerKimchi   ReviewerHarness = "kimchi"
	ReviewerKiro     ReviewerHarness = "kiro"
	ReviewerPi       ReviewerHarness = "pi"
	ReviewerQwen     ReviewerHarness = "qwen"
	ReviewerAgy      ReviewerHarness = "agy"
	ReviewerContinue ReviewerHarness = "continue"
	ReviewerGoose    ReviewerHarness = "goose"
	ReviewerVibe     ReviewerHarness = "vibe"
	ReviewerDevin    ReviewerHarness = "devin"
	ReviewerDroid    ReviewerHarness = "droid"
	ReviewerKimi     ReviewerHarness = "kimi"
	ReviewerMuse     ReviewerHarness = "muse"
	ReviewerAmp      ReviewerHarness = "amp"
	ReviewerAider    ReviewerHarness = "aider"
	ReviewerGrok     ReviewerHarness = "grok"
	ReviewerCrush    ReviewerHarness = "crush"
	ReviewerAuggie   ReviewerHarness = "auggie"
	ReviewerCline    ReviewerHarness = "cline"
	ReviewerAutohand ReviewerHarness = "autohand"
)

// AllReviewerHarnesses is the canonical set used to validate a configured
// reviewer harness.
var AllReviewerHarnesses = []ReviewerHarness{
	ReviewerClaudeCode,
	ReviewerCodex,
	ReviewerCursor,
	ReviewerOpenCode,
}

// IsKnown reports whether h is one of the supported reviewer harnesses.
func (h ReviewerHarness) IsKnown() bool {
	for _, k := range AllReviewerHarnesses {
		if h == k {
			return true
		}
	}
	return false
}

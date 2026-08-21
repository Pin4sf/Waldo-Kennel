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
	ReviewerCopilot    ReviewerHarness = "copilot"
	ReviewerCursor     ReviewerHarness = "cursor"
	ReviewerKiloCode   ReviewerHarness = "kilocode"
	ReviewerKimchi     ReviewerHarness = "kimchi"
	ReviewerOpenCode   ReviewerHarness = "opencode"
	ReviewerKiro       ReviewerHarness = "kiro"
	ReviewerPi         ReviewerHarness = "pi"
	ReviewerQwen       ReviewerHarness = "qwen"
	ReviewerAgy        ReviewerHarness = "agy"
	ReviewerContinue   ReviewerHarness = "continue"
	ReviewerGoose      ReviewerHarness = "goose"
	ReviewerVibe       ReviewerHarness = "vibe"
	ReviewerDevin      ReviewerHarness = "devin"
	ReviewerDroid      ReviewerHarness = "droid"
	ReviewerKimi       ReviewerHarness = "kimi"
	ReviewerMuse       ReviewerHarness = "muse"
	ReviewerAmp        ReviewerHarness = "amp"
	ReviewerAider      ReviewerHarness = "aider"
	ReviewerGrok       ReviewerHarness = "grok"
	ReviewerCrush      ReviewerHarness = "crush"
	ReviewerAuggie     ReviewerHarness = "auggie"
	ReviewerCline      ReviewerHarness = "cline"
	ReviewerAutohand   ReviewerHarness = "autohand"
)

// AllReviewerHarnesses is the canonical set used to validate a configured
// reviewer harness.
var AllReviewerHarnesses = []ReviewerHarness{
	ReviewerClaudeCode,
	ReviewerCodex,
	ReviewerCopilot,
	ReviewerCursor,
	ReviewerKiloCode,
	ReviewerKimchi,
	ReviewerOpenCode,
	ReviewerKiro,
	ReviewerPi,
	ReviewerQwen,
	ReviewerAgy,
	ReviewerContinue,
	ReviewerGoose,
	ReviewerVibe,
	ReviewerDevin,
	ReviewerDroid,
	ReviewerKimi,
	ReviewerMuse,
	ReviewerAmp,
	ReviewerAider,
	ReviewerGrok,
	ReviewerCrush,
	ReviewerAuggie,
	ReviewerCline,
	ReviewerAutohand,
}

// IsRecognizedPersisted reports whether h is an identity that existing durable
// review rows may continue to read.
func (h ReviewerHarness) IsRecognizedPersisted() bool {
	for _, k := range AllReviewerHarnesses {
		if h == k {
			return true
		}
	}
	return false
}

// IsSelectableForNewWork reports whether h may start a new review in this build.
func (h ReviewerHarness) IsSelectableForNewWork() bool {
	return h == ReviewerCodex
}

// IsKnown is retained as a compatibility alias for persisted identity checks.
func (h ReviewerHarness) IsKnown() bool {
	return h.IsRecognizedPersisted()
}

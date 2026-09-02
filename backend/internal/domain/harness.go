package domain

// AgentHarness identifies which local coding harness executes an Attempt.
type AgentHarness string

const (
	HarnessClaudeCode AgentHarness = "claude-code"
	HarnessCodex      AgentHarness = "codex"
	HarnessOpenCode   AgentHarness = "opencode"
	HarnessCursor     AgentHarness = "cursor"
	HarnessPi         AgentHarness = "pi"

	// HarnessFake is test-only and is never exposed as a Kennel provider.
	HarnessFake AgentHarness = "fake"
)

// AllHarnesses is the complete product provider vocabulary shipped by Kennel.
// Provider availability on a particular machine is discovered at runtime.
var AllHarnesses = []AgentHarness{
	HarnessCodex,
	HarnessClaudeCode,
	HarnessOpenCode,
	HarnessCursor,
	HarnessPi,
}

// IsRecognizedPersisted reports whether h is an active Kennel provider identity.
// The fake harness remains recognized only so focused tests can construct rows.
func (h AgentHarness) IsRecognizedPersisted() bool {
	if h == HarnessFake {
		return true
	}
	for _, candidate := range AllHarnesses {
		if h == candidate {
			return true
		}
	}
	return false
}

// IsSelectableForNewWork reports whether Kennel supports this provider for new
// work. Installation, authentication and role capability are separate runtime
// admission checks; this method must never encode a preferred provider.
func (h AgentHarness) IsSelectableForNewWork() bool {
	for _, candidate := range AllHarnesses {
		if h == candidate {
			return true
		}
	}
	return false
}

func (h AgentHarness) IsKnown() bool {
	return h.IsRecognizedPersisted()
}

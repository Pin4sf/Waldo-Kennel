package domain

// AgentHarness identifies which local coding harness executes a Kennel session.
type AgentHarness string

// Active Kennel provider harnesses.
const (
	HarnessClaudeCode AgentHarness = "claude-code"
	HarnessCodex      AgentHarness = "codex"
	HarnessOpenCode   AgentHarness = "opencode"
	HarnessCursor     AgentHarness = "cursor"
	HarnessPi         AgentHarness = "pi"

	// HarnessFake is test-only. It is not part of the product provider surface.
	HarnessFake AgentHarness = "fake"
)

// AllHarnesses is the complete provider vocabulary shipped by Kennel.
// Installation, authentication, project configuration and per-role readiness
// are runtime facts and must not be encoded by adding dormant provider ids here.
var AllHarnesses = []AgentHarness{
	HarnessCodex,
	HarnessClaudeCode,
	HarnessOpenCode,
	HarnessCursor,
	HarnessPi,
}

// IsRecognizedPersisted reports whether h is an active Kennel provider identity.
// HarnessFake remains recognized only so focused tests can construct fixtures.
// Kennel intentionally carries no historical provider compatibility vocabulary.
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

// IsSelectableForNewWork reports build support for fresh worker execution.
// Local installation/auth/profile readiness is checked by the agent inventory
// and again authoritatively at spawn; this predicate never expresses provider
// preference or machine state.
func (h AgentHarness) IsSelectableForNewWork() bool {
	for _, candidate := range AllHarnesses {
		if h == candidate {
			return true
		}
	}
	return false
}

// IsSelectableAsCoordinator reports whether the provider has the structured
// interaction and recovery contract Kennel requires for coordinator-class
// roles. Codex, Claude Code and OpenCode have registered structured chat
// drivers; Cursor and Pi remain worker-only until equivalent support exists.
func (h AgentHarness) IsSelectableAsCoordinator() bool {
	switch h {
	case HarnessCodex, HarnessClaudeCode, HarnessOpenCode:
		return true
	default:
		return false
	}
}

// IsSelectableAsSwitchTarget reports whether the provider implements Kennel's
// complete continuation contract for replacing the provider beneath one stable
// logical session. The session manager still validates the concrete adapter at
// runtime before any source session is stopped.
func (h AgentHarness) IsSelectableAsSwitchTarget() bool {
	switch h {
	case HarnessCodex, HarnessClaudeCode, HarnessOpenCode:
		return true
	default:
		return false
	}
}

// IsKnown is the canonical provider-identity predicate used by persisted domain
// validation and project configuration.
func (h AgentHarness) IsKnown() bool {
	return h.IsRecognizedPersisted()
}

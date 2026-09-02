package activitydispatch

import (
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

func TestDeriverTokensAreActiveHarnesses(t *testing.T) {
	for token := range Derivers {
		if token == string(domain.HarnessFake) {
			continue
		}
		if !domain.AgentHarness(token).IsKnown() {
			t.Errorf("deriver token %q is not an active Kennel provider", token)
		}
	}
}

func TestSupportsHarnessMatchesActualHookCapability(t *testing.T) {
	for _, harness := range []domain.AgentHarness{
		domain.HarnessCodex,
		domain.HarnessClaudeCode,
		domain.HarnessOpenCode,
		domain.HarnessCursor,
	} {
		if !SupportsHarness(harness) {
			t.Errorf("SupportsHarness(%q) = false, want true", harness)
		}
	}
	if SupportsHarness(domain.HarnessPi) {
		t.Error("Pi has no Kennel hook integration yet; silence must not be interpreted as a lifecycle signal")
	}
	if SupportsHarness("") || SupportsHarness("aider") {
		t.Error("unknown/removed providers must not have a hook pipeline")
	}
}

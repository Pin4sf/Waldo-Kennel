package activitydispatch

import (
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

func TestDeriverTokensAreSelectableHarnessesOrFake(t *testing.T) {
	for token := range Derivers {
		if token != string(domain.HarnessFake) && !domain.AgentHarness(token).IsKnown() {
			t.Errorf("deriver token %q is not a selectable AgentHarness", token)
		}
	}
}

func TestSupportsHarness(t *testing.T) {
	for _, h := range []domain.AgentHarness{domain.HarnessCodex, domain.HarnessClaudeCode, domain.HarnessOpenCode, domain.HarnessCursor} {
		if !SupportsHarness(h) {
			t.Errorf("SupportsHarness(%q) = false, want true", h)
		}
	}
	if SupportsHarness(domain.HarnessGrok) || SupportsHarness(domain.AgentHarness("")) {
		t.Error("retired and empty harnesses must not have an activity pipeline")
	}
}

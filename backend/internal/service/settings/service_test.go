package settings

import (
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

type allChatHarnesses struct{}

func (allChatHarnesses) SupportsChat(domain.AgentHarness) bool { return true }

func TestChatHarnessesExcludesHistoricalCandidates(t *testing.T) {
	svc := New(nil, allChatHarnesses{}, nil)

	got := svc.ChatHarnesses([]domain.AgentHarness{domain.HarnessClaudeCode, domain.HarnessCodex})
	if len(got) != 1 || got[0] != domain.HarnessCodex {
		t.Fatalf("ChatHarnesses() = %#v, want only Codex", got)
	}
}

// Package activitydispatch maps Kennel hook callbacks onto normalized activity
// state. Only providers that actually install Kennel hooks belong here.
package activitydispatch

import (
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/activitystate"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/claudecode"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/codex"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/fake"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/opencode"
	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

type DeriveFunc func(event string, payload []byte) (domain.ActivityState, bool)

// Pi currently has no Kennel hook integration and therefore intentionally does
// not appear here. Cursor uses the standard hook event vocabulary.
var Derivers = map[string]DeriveFunc{
	"claude-code": claudecode.DeriveActivityState,
	"codex":       codex.DeriveActivityState,
	"opencode":    opencode.DeriveActivityState,
	"cursor":      activitystate.StandardDeriveActivityState,
	"fake":        fake.DeriveActivityState,
}

func Derive(agent, event string, payload []byte) (domain.ActivityState, bool) {
	derive, found := Derivers[agent]
	if !found {
		return "", false
	}
	return derive(event, payload)
}

func SupportsHarness(h domain.AgentHarness) bool {
	_, ok := Derivers[string(h)]
	return ok
}

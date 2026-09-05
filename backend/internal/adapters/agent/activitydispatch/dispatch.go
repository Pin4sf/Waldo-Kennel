// Package activitydispatch is the single source of truth mapping the agent
// token in `kennel hooks <agent> <event>` onto the function that interprets that
// agent's hook callbacks as an Kennel activity state.
//
// The hidden `kennel hooks` CLI command dispatches a live callback through it. Every
// adapter that installs `kennel hooks <tok>` callbacks must have a deriver
// registered here — otherwise the adapter writes callbacks that nothing on the
// receiving side understands, so its activity is silently never reported.
package activitydispatch

import (
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/activitystate"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/claudecode"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/codex"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/fake"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/opencode"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// DeriveFunc maps a native agent hook event and its raw stdin payload onto an Kennel
// activity state. ok=false means the event carries no activity signal.
type DeriveFunc func(event string, payload []byte) (domain.ActivityState, bool)

// Derivers maps the agent token in `kennel hooks <agent> <event>` to its deriver.
// Per-adapter PRs add their tokens here as they land.
var Derivers = map[string]DeriveFunc{
	// Adapters that parse hook payloads for finer-grained state keep their own
	// deriver; the rest share the name-only StandardDeriveActivityState.
	"claude-code": claudecode.DeriveActivityState,
	"codex":       codex.DeriveActivityState,
	"opencode":    opencode.DeriveActivityState,
	"cursor":      activitystate.StandardDeriveActivityState,
	// pi installs no hook callbacks (it exposes lifecycle only through in-process
	// extensions), so it has no deriver and SupportsHarness reports false for it.
	"fake": fake.DeriveActivityState,
}

// Derive looks up the deriver for an agent token and applies it. ok=false when
// the token has no registered deriver or the event carries no activity signal —
// the caller reports nothing in either case.
func Derive(agent, event string, payload []byte) (domain.ActivityState, bool) {
	derive, found := Derivers[agent]
	if !found {
		return "", false
	}
	return derive(event, payload)
}

// SupportsHarness reports whether a harness has an activity pipeline at all:
// a registered deriver here means its adapter installs `kennel hooks <harness>`
// callbacks that can reach the daemon. Status derivation uses this to decide
// whether prolonged silence is suspicious (no_signal) or simply all a hook-less
// harness can ever report (idle). Harness names and `kennel hooks` agent tokens are
// the same strings by convention.
func SupportsHarness(h domain.AgentHarness) bool {
	_, ok := Derivers[string(h)]
	return ok
}

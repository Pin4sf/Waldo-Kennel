package sessionmanager

import (
	"context"
	"fmt"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// ProfileReadinessForSpawn probes the profile gate EXACTLY the way Spawn
// enforces it: the same adapter capability, resolved against the config the
// launch would use (project role defaults merged with explicit request
// overrides), so "ready" means launchable rather than merely installed.
//
// Attempt admission (#31) runs this during its fail-closed ordering, before
// any durable row exists, and again before resuming a paused attempt. It is
// deliberately a thin wrapper over the same helpers Spawn calls — duplicating
// the merge rules elsewhere would let the two answers drift.
func ProfileReadinessForSpawn(
	ctx context.Context,
	agents ports.AgentResolver,
	projectCfg domain.ProjectConfig,
	kind domain.SessionKind,
	harness domain.AgentHarness,
	overrides ports.AgentConfig,
) (ports.AgentProfileReadiness, error) {
	agent, ok := agents.Agent(harness)
	if !ok {
		// An unregistered harness can never launch; say so instead of failing
		// the whole probe.
		return ports.AgentProfileReadiness{
			Ready:  false,
			Detail: fmt.Sprintf("no agent adapter registered for %q", harness),
		}, nil
	}
	checker, ok := agent.(ports.AgentProfileReadinessChecker)
	if !ok {
		// Adapters without a profile gate have no launch dependency beyond
		// binary presence, which Spawn itself validates authoritatively.
		return ports.AgentProfileReadiness{
			Ready:  true,
			Detail: "harness declares no profile gate; spawn remains the validation point",
		}, nil
	}
	probeConfig := applySpawnAgentConfig(freshAgentConfig(kind, harness, projectCfg), overrides)
	return checker.ProfileReadiness(ctx, probeConfig)
}

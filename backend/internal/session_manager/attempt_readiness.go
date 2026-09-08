package sessionmanager

import (
	"context"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// ProfileReadinessForSpawn probes the profile gate EXACTLY the way an ordinary
// Spawn resolves it: Project role defaults merged with explicit request
// overrides. Governed Attempts should use ProfileReadinessForExactSpawn so the
// approved WorkUnit binding, not mutable Project model state, owns the probe.
func ProfileReadinessForSpawn(
	ctx context.Context,
	agents ports.AgentResolver,
	projectCfg domain.ProjectConfig,
	kind domain.SessionKind,
	harness domain.AgentHarness,
	overrides ports.AgentConfig,
) (ports.AgentProfileReadiness, error) {
	resolvedHarness, config, err := spawnExecutionConfig(ports.SpawnConfig{
		Kind:        kind,
		Harness:     harness,
		AgentConfig: overrides,
	}, projectCfg)
	if err != nil {
		return ports.AgentProfileReadiness{}, err
	}
	return profileReadinessForConfig(ctx, agents, resolvedHarness, config)
}

// ProfileReadinessForExactSpawn probes the frozen execution binding through the
// same config resolver Manager.Spawn uses. provider_default therefore means an
// actually empty concrete model even when the Project currently names one.
func ProfileReadinessForExactSpawn(
	ctx context.Context,
	agents ports.AgentResolver,
	projectCfg domain.ProjectConfig,
	kind domain.SessionKind,
	binding domain.ExecutionBinding,
	overrides ports.AgentConfig,
) (ports.AgentProfileReadiness, error) {
	bindingCopy := binding
	harness, config, err := spawnExecutionConfig(ports.SpawnConfig{
		Kind:                  kind,
		Harness:               binding.Provider,
		AgentConfig:           overrides,
		ExactExecutionBinding: &bindingCopy,
	}, projectCfg)
	if err != nil {
		return ports.AgentProfileReadiness{}, err
	}
	return profileReadinessForConfig(ctx, agents, harness, config)
}

func profileReadinessForConfig(
	ctx context.Context,
	agents ports.AgentResolver,
	harness domain.AgentHarness,
	config ports.AgentConfig,
) (ports.AgentProfileReadiness, error) {
	agent, ok := agents.Agent(harness)
	if !ok {
		return ports.AgentProfileReadiness{
			Ready:  false,
			Detail: fmt.Sprintf("no agent adapter registered for %q", harness),
		}, nil
	}
	checker, ok := agent.(ports.AgentProfileReadinessChecker)
	if !ok {
		return ports.AgentProfileReadiness{
			Ready:  true,
			Detail: "harness declares no profile gate; spawn remains the validation point",
		}, nil
	}
	return checker.ProfileReadiness(ctx, config)
}

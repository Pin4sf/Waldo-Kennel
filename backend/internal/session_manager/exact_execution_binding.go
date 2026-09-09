package sessionmanager

import (
	"fmt"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// prepareSpawnExecution converts frozen WorkUnit authority into local launch
// inputs before Manager.Spawn performs readiness or chooses TUI versus Chat.
// The Project record is a request-local copy, so clearing its model preference
// here cannot mutate durable Project configuration.
func prepareSpawnExecution(cfg ports.SpawnConfig, projectCfg domain.ProjectConfig) (ports.SpawnConfig, domain.ProjectConfig, error) {
	harness, agentConfig, err := spawnExecutionConfig(cfg, projectCfg)
	if err != nil {
		return ports.SpawnConfig{}, domain.ProjectConfig{}, err
	}
	cfg.Harness = harness
	if cfg.ExactExecutionBinding == nil {
		// Ordinary ad-hoc session spawning keeps its existing request/project
		// merge semantics. Exact binding resolution is deliberately additive and
		// must not become a second compatibility path for legacy sessions.
		return cfg, projectCfg, nil
	}
	cfg.AgentConfig = agentConfig

	// Existing TUI and Chat launch plumbing both merge Project config again.
	// Remove only mutable Project model preference from this local copy; the exact
	// binding has already been validated and copied into cfg.AgentConfig above.
	// Non-model preferences remain available only where they do not widen an
	// attached governed ExecutionPolicy.
	projectCfg.AgentConfig.Model = ""
	if cfg.Kind == domain.KindOrchestrator {
		projectCfg.Orchestrator.AgentConfig.Model = ""
	} else {
		projectCfg.Worker.AgentConfig.Model = ""
	}
	return cfg, projectCfg, nil
}

// spawnExecutionConfig resolves the launch harness/config once for the whole
// Spawn operation. Project and request values remain ordinary preferences for
// ad-hoc sessions. A governed Attempt supplies ExactExecutionBinding; that
// frozen WorkUnit authority wins after preference resolution.
func spawnExecutionConfig(cfg ports.SpawnConfig, projectCfg domain.ProjectConfig) (domain.AgentHarness, ports.AgentConfig, error) {
	harness := effectiveHarness(cfg.Harness, cfg.Kind, projectCfg)
	agentConfig := applySpawnAgentConfig(freshAgentConfig(cfg.Kind, harness, projectCfg), cfg.AgentConfig)
	if cfg.ExactExecutionBinding == nil {
		return harness, agentConfig, nil
	}

	binding := *cfg.ExactExecutionBinding
	if err := binding.ValidateForNewWork(); err != nil {
		return "", ports.AgentConfig{}, fmt.Errorf("exact execution binding: %w", err)
	}
	harness = binding.Provider

	// Resolve non-model provider settings against the exact provider so a
	// Project role config for another harness cannot leak its mode/profile into
	// this launch. Then freeze only the model dimension from WorkUnit authority;
	// permissions are reduced to a safe provider-neutral posture when an
	// immutable ExecutionPolicy is attached; adapters perform the final mapping.
	agentConfig = applySpawnAgentConfig(freshAgentConfig(cfg.Kind, harness, projectCfg), cfg.AgentConfig)
	switch binding.ModelSelection {
	case domain.ExecutionBindingModelProviderDefault:
		agentConfig.Model = ""
	case domain.ExecutionBindingModelExplicit:
		agentConfig.Model = strings.TrimSpace(binding.Model)
	default:
		return "", ports.AgentConfig{}, fmt.Errorf("exact execution binding has unsupported model selection %q", binding.ModelSelection)
	}
	if cfg.ExecutionPolicy != nil {
		// A governed policy owns the permission posture too. Provider adapters
		// still map the normalized capabilities to their native sandbox, but a
		// mutable Project bypass setting must never reach that boundary as the
		// fallback posture.
		agentConfig.Permissions = domain.PermissionModeAcceptEdits
	}
	return harness, agentConfig, nil
}

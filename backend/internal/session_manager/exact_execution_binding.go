package sessionmanager

import (
	"fmt"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// spawnExecutionConfig resolves the launch harness/config once for the whole
// Spawn operation. Project and request values remain ordinary preferences for
// ad-hoc sessions. A governed Attempt supplies ExactExecutionBinding; that
// frozen WorkUnit authority wins after preference resolution so readiness, TUI,
// and Chat all observe exactly the same provider/model semantics.
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
	// permissions remain provider-neutral capability authority.
	agentConfig = applySpawnAgentConfig(freshAgentConfig(cfg.Kind, harness, projectCfg), cfg.AgentConfig)
	switch binding.ModelSelection {
	case domain.ExecutionBindingModelProviderDefault:
		agentConfig.Model = ""
	case domain.ExecutionBindingModelExplicit:
		agentConfig.Model = strings.TrimSpace(binding.Model)
	default:
		return "", ports.AgentConfig{}, fmt.Errorf("exact execution binding has unsupported model selection %q", binding.ModelSelection)
	}
	return harness, agentConfig, nil
}

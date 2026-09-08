package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// providerDefaultModelOverride is a transport-only non-empty override that
// clears an inherited Project model inside Session Manager's existing merge.
// Every selectable execution adapter trims model text before constructing its
// provider command, so this value becomes "no --model" at the provider
// boundary. It is never persisted or exposed as canonical model state.
const providerDefaultModelOverride = " "

// SpawnExactAttempt starts a subordinate worker for an approved WorkUnit.
// Provider/model semantics come only from binding; mutable Project provider or
// model preferences cannot reinterpret an approved Plan. Other Project session
// configuration (permissions, environment, workspace provisioning) remains
// ordinary runtime configuration rather than execution-routing authority.
func (s *Service) SpawnExactAttempt(ctx context.Context, cfg ports.SpawnConfig, binding domain.ExecutionBinding) (domain.Session, int, int, error) {
	if err := binding.ValidateForNewWork(); err != nil {
		return domain.Session{}, 0, 0, fmt.Errorf("exact attempt binding: %w", err)
	}
	cfg.Kind = domain.KindWorker
	cfg.Harness = binding.Provider
	bindingCopy := binding
	cfg.ExactExecutionBinding = &bindingCopy

	switch binding.ModelSelection {
	case domain.ExecutionBindingModelProviderDefault:
		cfg.AgentConfig.Model = providerDefaultModelOverride
	case domain.ExecutionBindingModelExplicit:
		model := strings.TrimSpace(binding.Model)
		if model == "" {
			return domain.Session{}, 0, 0, fmt.Errorf("exact attempt binding: explicit model is required")
		}
		cfg.AgentConfig.Model = model
	default:
		return domain.Session{}, 0, 0, fmt.Errorf("exact attempt binding: unsupported model selection %q", binding.ModelSelection)
	}
	return s.Spawn(ctx, cfg)
}

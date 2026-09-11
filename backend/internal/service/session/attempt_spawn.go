package session

import (
	"context"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// SpawnExactAttempt is the canonical governed worker launch boundary. The
// approved WorkUnit owns provider/model authority; mutable Project defaults are
// only preferences used before Plan approval and must not reinterpret it here.
func (s *Service) SpawnExactAttempt(ctx context.Context, cfg ports.SpawnConfig, binding domain.ExecutionBinding) (domain.Session, int, int, error) {
	if err := binding.ValidateForNewWork(); err != nil {
		return domain.Session{}, 0, 0, fmt.Errorf("spawn exact attempt: %w", err)
	}
	cfg.Kind = domain.KindWorker
	cfg.Harness = binding.Provider
	bindingCopy := binding
	cfg.ExactExecutionBinding = &bindingCopy
	// Session Manager applies model semantics from ExactExecutionBinding after
	// Project/request preferences are resolved. Do not encode provider_default
	// as a magic model value here: an empty concrete model is meaningful and
	// means the provider owns model selection for this exact binding.
	cfg.AgentConfig.Model = ""
	return s.Spawn(ctx, cfg)
}

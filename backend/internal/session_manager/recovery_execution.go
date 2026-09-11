package sessionmanager

import (
	"context"
	"fmt"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// attemptSessionEvidenceStore is deliberately optional so old ordinary
// sessions and small compatibility stores remain readable. Production SQLite
// implements it. A session marked as governed cannot recover without it.
type attemptSessionEvidenceStore interface {
	LatestAttemptSessionRefForSession(ctx context.Context, sessionID string) (domain.AttemptSessionRef, bool, error)
}

type recoveryExecution struct {
	binding domain.ExecutionBinding
	policy  domain.AttemptExecutionPolicy
}

func (m *Manager) loadRecoveryExecution(ctx context.Context, rec domain.SessionRecord) (*recoveryExecution, error) {
	marker := strings.TrimSpace(rec.Metadata.GovernedExecutionPolicyDigest)
	store, supported := m.store.(attemptSessionEvidenceStore)
	if !supported {
		if marker == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("governed recovery evidence unavailable")
	}

	ref, found, err := store.LatestAttemptSessionRefForSession(ctx, string(rec.ID))
	if err != nil {
		return nil, fmt.Errorf("load governed recovery evidence: %w", err)
	}
	if !found {
		if marker == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("governed recovery evidence is missing")
	}
	if marker == "" && !domain.LooksLikeGovernedAdmissionSnapshot(ref.AdmissionSnapshot) {
		// Older Attempt refs remain readable for inspection/recovery compatibility,
		// but they never acquire a synthesized policy on this path.
		return nil, nil
	}

	snapshot, err := domain.ParseAdmissionSnapshot(ref.AdmissionSnapshot)
	if err != nil {
		return nil, fmt.Errorf("governed recovery admission snapshot is invalid: %w", err)
	}
	if err := snapshot.ValidateSession(rec, ref, marker); err != nil {
		return nil, fmt.Errorf("governed recovery %w", err)
	}
	binding := domain.ExecutionBinding{
		Provider:       domain.AgentHarness(snapshot.Harness),
		ModelSelection: snapshot.ModelSelection,
		Model:          strings.TrimSpace(snapshot.RequestedModel),
	}
	if err := binding.ValidateForNewWork(); err != nil {
		return nil, fmt.Errorf("governed recovery execution binding is invalid: %w", err)
	}
	return &recoveryExecution{binding: binding, policy: snapshot.ExecutionPolicy}, nil
}

// sessionIsGoverned reports whether rec is bound to a governed Attempt's
// frozen ExecutionPolicy and provider binding, reusing the exact evidence
// resolution recovery already trusts (loadRecoveryExecution) rather than a
// second, possibly-diverging notion of "governed". Ambiguous or invalid
// governed evidence fails closed: it is treated as governed, the same way
// recovery itself refuses to proceed on evidence it cannot validate. An
// `unknown` verdict here must never read as "ordinary session".
func (m *Manager) sessionIsGoverned(ctx context.Context, rec domain.SessionRecord) bool {
	execution, err := m.loadRecoveryExecution(ctx, rec)
	if err != nil {
		return true
	}
	return execution != nil
}

func recoveryAgentConfig(rec domain.SessionRecord, project domain.ProjectRecord, execution *recoveryExecution) (ports.AgentConfig, error) {
	if execution == nil {
		return effectiveAgentConfig(rec.Kind, project.Config), nil
	}
	binding := execution.binding
	policy := execution.policy
	cfg, _, err := prepareSpawnExecution(ports.SpawnConfig{
		Kind:                  rec.Kind,
		Harness:               binding.Provider,
		ExactExecutionBinding: &binding,
		ExecutionPolicy:       &policy,
	}, project.Config)
	if err != nil {
		return ports.AgentConfig{}, fmt.Errorf("resolve governed recovery config: %w", err)
	}
	return cfg.AgentConfig, nil
}

package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	sessionmanager "github.com/Pin4sf/Waldo-Kennel/backend/internal/session_manager"
)

// attemptLivenessInterval is how often the daemon reconcile hook re-evaluates
// running attempts against their bound session's heartbeat facts.
const attemptLivenessInterval = 15 * time.Second

type projectConfigSource interface {
	GetProject(ctx context.Context, id string) (domain.ProjectRecord, bool, error)
}

// attemptSessionControl is the subordinate runtime surface used by Outcome
// execution. SpawnExactAttempt is intentionally distinct from ordinary Spawn:
// approved WorkUnits have frozen provider/model semantics that mutable Project
// preferences are not allowed to reinterpret.
type attemptSessionControl interface {
	SpawnExactAttempt(ctx context.Context, cfg ports.SpawnConfig, binding domain.ExecutionBinding) (domain.Session, int, int, error)
	Kill(ctx context.Context, id domain.SessionID) (bool, error)
	Get(ctx context.Context, id domain.SessionID) (domain.Session, error)
}

type attemptSpawner struct {
	sessions attemptSessionControl
	projects projectConfigSource
	agents   ports.AgentResolver
}

var _ ports.AttemptSessionSpawner = attemptSpawner{}

func (a attemptSpawner) ProfileReadiness(ctx context.Context, projectID domain.ProjectID, binding domain.ExecutionBinding, policy *domain.AttemptExecutionPolicy) (ports.AgentProfileReadiness, error) {
	if a.projects == nil || a.agents == nil {
		return ports.AgentProfileReadiness{}, fmt.Errorf("attempt spawner is not fully wired")
	}
	if err := binding.ValidateForNewWork(); err != nil {
		return profileNotReady(err.Error())
	}
	rec, ok, err := a.projects.GetProject(ctx, string(projectID))
	if err != nil {
		return ports.AgentProfileReadiness{}, err
	}
	if !ok {
		return ports.AgentProfileReadiness{Ready: false, Detail: "project is not registered"}, nil
	}

	// Readiness consumes the same exact execution binding as launch. Project
	// configuration may still contribute provider-neutral runtime settings, but
	// it cannot rewrite the frozen provider/model selection after approval.
	return sessionmanager.ProfileReadinessForExactSpawn(
		ctx,
		a.agents,
		rec.Config,
		domain.KindWorker,
		binding,
		ports.AgentConfig{},
		policy,
	)
}

func profileNotReady(detail string) (ports.AgentProfileReadiness, error) {
	return ports.AgentProfileReadiness{Ready: false, Detail: detail}, nil
}

func (a attemptSpawner) Spawn(ctx context.Context, req ports.AttemptSpawnRequest) (ports.AttemptSpawnResult, error) {
	binding := domain.ExecutionBinding{
		Provider:       req.Harness,
		ModelSelection: req.ModelSelection,
		Model:          strings.TrimSpace(req.Model),
	}
	if err := binding.ValidateForNewWork(); err != nil {
		return ports.AttemptSpawnResult{}, err
	}
	sess, _, _, err := a.sessions.SpawnExactAttempt(ctx, ports.SpawnConfig{
		ProjectID:       req.ProjectID,
		Kind:            domain.KindWorker,
		Harness:         binding.Provider,
		ExecutionPolicy: req.ExecutionPolicy,
		Prompt:          req.Prompt,
		DisplayName:     req.DisplayName,
	}, binding)
	if err != nil {
		return ports.AttemptSpawnResult{}, err
	}
	// The ordinary session read model does not currently expose a runtime-
	// reported effective model. Leave it unknown rather than fabricating one;
	// the immutable requested binding remains recorded separately by Attempt.
	return ports.AttemptSpawnResult{Session: sess}, nil
}

func (a attemptSpawner) Terminate(ctx context.Context, _ domain.ProjectID, sessionID string) (ports.TerminationResult, error) {
	freed, err := a.sessions.Kill(ctx, domain.SessionID(sessionID))
	if err != nil {
		return ports.TerminationResult{}, err
	}
	rec, err := a.sessions.Get(ctx, domain.SessionID(sessionID))
	if err != nil || !rec.IsTerminated {
		return ports.TerminationResult{}, fmt.Errorf("%w: durable record for %q does not show a terminated session", ports.ErrProviderStopUnproven, sessionID)
	}
	return ports.TerminationResult{ProviderStopped: true, WorkspaceFreed: freed}, nil
}

func runAttemptLivenessLoop(ctx context.Context, attempts attemptLivenessHook, log *slog.Logger) {
	reconcile(ctx, attempts, log, "on boot")
	ticker := time.NewTicker(attemptLivenessInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcile(ctx, attempts, log, "")
		}
	}
}

// reconcile runs both halves of the terminal-state sequence in order: decide
// whether execution has ended, then classify what ended.
//
// Liveness runs first on purpose. Classification only ever looks at attempts
// already recorded as reconciled, so running it after liveness lets an attempt
// that just ended be classified in the same tick instead of waiting for the
// next one. Both are pure functions of durable facts, so a failure in either
// leaves state untouched and the next tick retries.
func reconcile(ctx context.Context, attempts attemptLivenessHook, log *slog.Logger, when string) {
	suffix := ""
	if when != "" {
		suffix = " " + when
	}
	if err := attempts.EvaluateAttemptLiveness(ctx); err != nil {
		log.Warn("attempt liveness evaluation"+suffix, "err", err)
	}
	if err := attempts.ReconcileAttemptOutcomes(ctx); err != nil {
		log.Warn("attempt outcome reconciliation"+suffix, "err", err)
	}
}

type attemptLivenessHook interface {
	EvaluateAttemptLiveness(ctx context.Context) error
	// ReconcileAttemptOutcomes classifies attempts whose execution has ended.
	ReconcileAttemptOutcomes(ctx context.Context) error
}

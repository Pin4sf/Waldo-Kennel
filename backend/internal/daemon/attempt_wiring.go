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

func (a attemptSpawner) ProfileReadiness(ctx context.Context, projectID domain.ProjectID, binding domain.ExecutionBinding) (ports.AgentProfileReadiness, error) {
	if a.projects == nil || a.agents == nil {
		return ports.AgentProfileReadiness{}, fmt.Errorf("attempt spawner is not fully wired")
	}
	if err := binding.ValidateForNewWork(); err != nil {
		return ports.AgentProfileReadiness{Ready: false, Detail: err.Error()}, nil
	}
	rec, ok, err := a.projects.GetProject(ctx, string(projectID))
	if err != nil {
		return ports.AgentProfileReadiness{}, err
	}
	if !ok {
		return ports.AgentProfileReadiness{Ready: false, Detail: "project is not registered"}, nil
	}

	// Readiness may consume non-routing Project runtime configuration, but it
	// must not inherit mutable provider/model preference after Plan approval.
	cfg := rec.Config
	cfg.AgentConfig.Model = ""
	cfg.Worker.Harness = ""
	cfg.Worker.AgentConfig.Model = ""
	override := ports.AgentConfig{}
	switch binding.ModelSelection {
	case domain.ExecutionBindingModelProviderDefault:
		// Session Manager's merge treats non-empty override text as authoritative;
		// selectable adapters trim model text before launch, so whitespace means
		// an explicit provider-default selection rather than inherited Project model.
		override.Model = " "
	case domain.ExecutionBindingModelExplicit:
		override.Model = strings.TrimSpace(binding.Model)
	default:
		return ports.AgentProfileReadiness{Ready: false, Detail: "unsupported model selection"}, nil
	}
	return sessionmanager.ProfileReadinessForSpawn(ctx, a.agents, cfg, domain.KindWorker, binding.Provider, override)
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
		ProjectID:   req.ProjectID,
		Kind:        domain.KindWorker,
		Harness:     binding.Provider,
		Prompt:      req.Prompt,
		DisplayName: req.DisplayName,
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
	if err := attempts.EvaluateAttemptLiveness(ctx); err != nil {
		log.Warn("attempt liveness evaluation on boot", "err", err)
	}
	ticker := time.NewTicker(attemptLivenessInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := attempts.EvaluateAttemptLiveness(ctx); err != nil {
				log.Warn("attempt liveness evaluation", "err", err)
			}
		}
	}
}

type attemptLivenessHook interface {
	EvaluateAttemptLiveness(ctx context.Context) error
}

package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	sessionsvc "github.com/aoagents/agent-orchestrator/backend/internal/service/session"
	"github.com/aoagents/agent-orchestrator/backend/internal/session_manager"
)

// attemptLivenessInterval is how often the daemon reconcile hook re-evaluates
// running attempts against their bound session's heartbeat facts.
const attemptLivenessInterval = 15 * time.Second

// projectConfigSource is the narrow slice of the projects store the spawner
// needs: the registered config a worker launch would resolve its agent
// defaults from.
type projectConfigSource interface {
	GetProject(ctx context.Context, id string) (domain.ProjectRecord, bool, error)
}

// attemptSpawner adapts the EXISTING session spawn path onto the narrow
// ports.AttemptSessionSpawner seam. It adds no provider knowledge: readiness
// goes through session_manager.ProfileReadinessForSpawn (the same checker and
// config merge Spawn enforces), and spawning delegates to service/session.Spawn,
// which re-probes internally. There is no fallback provider anywhere.
type attemptSpawner struct {
	sessions *sessionsvc.Service
	projects projectConfigSource
	agents   ports.AgentResolver
}

var _ ports.AttemptSessionSpawner = attemptSpawner{}

// ProfileReadiness probes whether the named harness can launch for a worker
// on this project, using exactly the config a real spawn would resolve.
func (a attemptSpawner) ProfileReadiness(ctx context.Context, projectID domain.ProjectID, harness domain.AgentHarness) (ports.AgentProfileReadiness, error) {
	if a.projects == nil || a.agents == nil {
		return ports.AgentProfileReadiness{}, fmt.Errorf("attempt spawner is not fully wired")
	}
	rec, ok, err := a.projects.GetProject(ctx, string(projectID))
	if err != nil {
		return ports.AgentProfileReadiness{}, err
	}
	if !ok {
		return ports.AgentProfileReadiness{Ready: false, Detail: "project is not registered"}, nil
	}
	return sessionmanager.ProfileReadinessForSpawn(ctx, a.agents, rec.Config, domain.KindWorker, harness, ports.AgentConfig{})
}

// Spawn starts the worker session through the ordinary path.
func (a attemptSpawner) Spawn(ctx context.Context, req ports.AttemptSpawnRequest) (domain.Session, error) {
	sess, _, _, err := a.sessions.Spawn(ctx, ports.SpawnConfig{
		ProjectID:   req.ProjectID,
		Kind:        domain.KindWorker,
		Harness:     req.Harness,
		Prompt:      req.Prompt,
		DisplayName: req.DisplayName,
	})
	return sess, err
}

// runAttemptLivenessLoop drives the daemon-side reconcile hook for Act &
// Observe: terminated provider sessions become reconciled attempts with an
// ordered exit observation; silent heartbeats mutate nothing and stay derived
// as unconfirmed until contain/reconcile decides.
func runAttemptLivenessLoop(ctx context.Context, attempts attemptLivenessHook, log *slog.Logger) {
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

// attemptLivenessHook is the narrow surface the loop consumes so tests can
// inject a stub instead of the full outcome service.
type attemptLivenessHook interface {
	EvaluateAttemptLiveness(ctx context.Context) error
}

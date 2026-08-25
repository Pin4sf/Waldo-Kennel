package daemon

import (
	"context"
	"errors"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// fakeSessionControl models the durable session surface the attempt spawner
// consumes: Kill answers with the WORKSPACE-reclamation boolean exactly like
// session_manager.Kill, and Get serves the durable record that provider-stop
// proof must be derived from.
type fakeSessionControl struct {
	killFreed bool
	killErr   error
	killed    []domain.SessionID
	record    domain.Session
	recordErr error
}

func (f *fakeSessionControl) Spawn(_ context.Context, cfg ports.SpawnConfig) (domain.Session, int, int, error) {
	rec := domain.SessionRecord{ID: domain.SessionID("sess-" + string(cfg.Harness)), Mode: domain.SessionModeTUI}
	return domain.Session{SessionRecord: rec}, 0, 0, nil
}

func (f *fakeSessionControl) Kill(_ context.Context, id domain.SessionID) (bool, error) {
	if f.killErr != nil {
		return false, f.killErr
	}
	f.killed = append(f.killed, id)
	return f.killFreed, nil
}

func (f *fakeSessionControl) Get(_ context.Context, _ domain.SessionID) (domain.Session, error) {
	if f.recordErr != nil {
		return domain.Session{}, f.recordErr
	}
	return f.record, nil
}

func terminatedSession() domain.Session {
	rec := domain.SessionRecord{ID: "sess-codex", IsTerminated: true}
	return domain.Session{SessionRecord: rec}
}

// TestTerminateDerivesProviderStopFromDurableRecord pins the round-3 P1 fix:
// Kill's boolean reports WORKSPACE reclamation, never provider liveness. The
// adapter must derive ProviderStopped from the durable session fact and keep
// WorkspaceFreed a separate, honest answer.
func TestTerminateDerivesProviderStopFromDurableRecord(t *testing.T) {
	projectID := domain.ProjectID("mer")
	sessionID := "sess-codex"

	t.Run("durably terminated with preserved dirty work", func(t *testing.T) {
		sessions := &fakeSessionControl{killFreed: false, record: terminatedSession()}
		spawner := attemptSpawner{sessions: sessions}
		res, err := spawner.Terminate(context.Background(), projectID, sessionID)
		if err != nil {
			t.Fatalf("terminate: %v", err)
		}
		// The exact reviewer shape: freed=false because the dirty worktree was
		// preserved — while the provider is DURABLY TERMINATED.
		if !res.ProviderStopped || res.WorkspaceFreed {
			t.Fatalf("result = %+v, want stopped=true freed=false", res)
		}
		if len(sessions.killed) != 1 || string(sessions.killed[0]) != sessionID {
			t.Fatalf("kill calls = %v, want the bound session", sessions.killed)
		}
	})

	t.Run("clean stop reclaims the workspace too", func(t *testing.T) {
		sessions := &fakeSessionControl{killFreed: true, record: terminatedSession()}
		spawner := attemptSpawner{sessions: sessions}
		res, err := spawner.Terminate(context.Background(), projectID, sessionID)
		if err != nil {
			t.Fatalf("terminate: %v", err)
		}
		if !res.ProviderStopped || !res.WorkspaceFreed {
			t.Fatalf("result = %+v, want both facts true", res)
		}
	})

	t.Run("absent session row stays unproven", func(t *testing.T) {
		// Manager.Kill answers (false, nil) for an absent row ("benign race");
		// without a durable record there is NO proof of anything.
		sessions := &fakeSessionControl{
			killFreed: false,
			recordErr: apierr.NotFound("SESSION_NOT_FOUND", "Unknown session"),
		}
		spawner := attemptSpawner{sessions: sessions}
		_, err := spawner.Terminate(context.Background(), projectID, sessionID)
		if !errors.Is(err, ports.ErrProviderStopUnproven) {
			t.Fatalf("err = %v, want ErrProviderStopUnproven", err)
		}
	})

	t.Run("record without termination fact stays unproven", func(t *testing.T) {
		// Kill reported success but the durable record does not show the
		// termination yet: UNKNOWN, not stopped.
		sessions := &fakeSessionControl{killFreed: true}
		spawner := attemptSpawner{sessions: sessions}
		_, err := spawner.Terminate(context.Background(), projectID, sessionID)
		if !errors.Is(err, ports.ErrProviderStopUnproven) {
			t.Fatalf("err = %v, want ErrProviderStopUnproven", err)
		}
	})

	t.Run("kill errors propagate untouched", func(t *testing.T) {
		sentinel := errors.New("runtime refused")
		sessions := &fakeSessionControl{killErr: sentinel}
		spawner := attemptSpawner{sessions: sessions}
		if _, err := spawner.Terminate(context.Background(), projectID, sessionID); !errors.Is(err, sentinel) {
			t.Fatalf("err = %v, want the kill error", err)
		}
	})
}

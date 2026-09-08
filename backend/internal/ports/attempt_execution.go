package ports

import (
	"context"
	"errors"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// AttemptSpawnRequest is the complete immutable execution binding admission
// hands to the provider seam. Provider/model values come from the approved
// WorkUnit; the spawner must not reread mutable Project preferences.
type AttemptSpawnRequest struct {
	ProjectID      domain.ProjectID
	Harness        domain.AgentHarness
	ModelSelection domain.ExecutionBindingModelSelection
	Model          string
	Prompt         string
	DisplayName    string
}

// AttemptSpawnResult reports the spawned subordinate session and, when the
// provider/runtime can truthfully report it, the concrete effective model.
// Empty EffectiveModel is valid for provider-default runtimes that do not
// expose the selected model.
type AttemptSpawnResult struct {
	Session        domain.Session
	EffectiveModel string
}

// AttemptSessionSpawner is the narrow boundary between governed Attempt
// admission and the real session spawn path. Readiness and spawn both consume
// the same exact immutable binding; no Project provider/model fallback is
// permitted after Plan approval.
type AttemptSessionSpawner interface {
	ProfileReadiness(ctx context.Context, projectID domain.ProjectID, binding domain.ExecutionBinding) (AgentProfileReadiness, error)
	Spawn(ctx context.Context, req AttemptSpawnRequest) (AttemptSpawnResult, error)
	Terminate(ctx context.Context, projectID domain.ProjectID, sessionID string) (TerminationResult, error)
}

type TerminationResult struct {
	ProviderStopped bool
	WorkspaceFreed  bool
}

var ErrProviderStopUnproven = errors.New("provider stop could not be proven")

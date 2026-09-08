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
	ProjectID       domain.ProjectID
	Harness         domain.AgentHarness
	ModelSelection  domain.ExecutionBindingModelSelection
	Model           string
	Prompt          string
	DisplayName     string
}

// AttemptSessionSpawner is the narrow boundary between governed Attempt
// admission and the real session spawn path. There is no silent provider/model
// fallback: the exact requested binding is the only one probed or spawned.
type AttemptSessionSpawner interface {
	ProfileReadiness(ctx context.Context, projectID domain.ProjectID, harness domain.AgentHarness) (AgentProfileReadiness, error)
	Spawn(ctx context.Context, req AttemptSpawnRequest) (domain.Session, error)
	Terminate(ctx context.Context, projectID domain.ProjectID, sessionID string) (TerminationResult, error)
}

type TerminationResult struct {
	ProviderStopped bool
	WorkspaceFreed  bool
}

var ErrProviderStopUnproven = errors.New("provider stop could not be proven")

package ports

import (
	"context"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

// AttemptSpawnRequest is everything attempt admission (#31) hands the
// execution seam to start one governed worker session.
type AttemptSpawnRequest struct {
	ProjectID domain.ProjectID
	Harness   domain.AgentHarness
	// Prompt is the deterministic RunBrief text derived from the frozen plan —
	// never a transcript replay.
	Prompt string
	// DisplayName labels the session in read models; empty falls back to the id.
	DisplayName string
}

// AttemptSessionSpawner is the narrow execution seam between attempt
// admission (#31) and the real spawn path. The daemon adapts it over the
// existing service/session.Spawn boundary and the agent registry's readiness
// checker; tests inject fakes.
//
// There is NO silent provider fallback: the requested harness is the only
// harness probed or spawned, and an unready harness fails admission closed.
type AttemptSessionSpawner interface {
	// ProfileReadiness probes whether the named harness can launch for a
	// worker on the given project, using the same checker and config merge
	// session spawn consults. Adapters without a profile gate report ready
	// with an explanatory detail.
	ProfileReadiness(ctx context.Context, projectID domain.ProjectID, harness domain.AgentHarness) (AgentProfileReadiness, error)

	// Spawn starts the provider session through the ordinary session service
	// path, which re-probes readiness internally exactly like every other
	// worker spawn.
	Spawn(ctx context.Context, req AttemptSpawnRequest) (domain.Session, error)
}

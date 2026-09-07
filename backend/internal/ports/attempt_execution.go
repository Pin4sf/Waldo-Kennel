package ports

import (
	"context"
	"errors"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
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

	// Terminate stops a bound provider session through the same authority
	// that spawned it. It returns TWO independent facts: ProviderStopped is
	// proven from the durable session record after teardown, and
	// WorkspaceFreed only mirrors whether the workspace could be reclaimed.
	// A preserved dirty worktree does NOT make the stop unproven; an absent
	// or un-terminated durable record DOES — return ErrProviderStopUnproven
	// in that case and callers MUST treat custody as held.
	Terminate(ctx context.Context, projectID domain.ProjectID, sessionID string) (TerminationResult, error)
}

// TerminationResult separates the two facts one stop produces. They are
// deliberately distinct: "the provider is no longer running" and "the
// workspace was reclaimed" fail independently (a dirty worktree is preserved
// while its provider is durably terminated), and conflating them either
// fakes a live provider or hides kept evidence.
type TerminationResult struct {
	// ProviderStopped reports that the provider runtime is durably stopped,
	// derived from the durable session record — never from a
	// workspace-reclamation boolean.
	ProviderStopped bool
	// WorkspaceFreed reports whether the workspace was reclaimed. False means
	// it was preserved for inspection; that says nothing about liveness.
	WorkspaceFreed bool
}

// ErrProviderStopUnproven reports a terminate whose runtime outcome could not
// be proven (missing session row, record without a termination fact,
// ambiguous kill). Custody law treats this as NOT stopped.
var ErrProviderStopUnproven = errors.New("provider stop could not be proven")

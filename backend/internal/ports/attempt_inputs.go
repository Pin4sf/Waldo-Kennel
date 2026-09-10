package ports

import (
	"context"
	"errors"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// ErrAttemptInputProvisioning reports that a successor's predecessor outputs
// could not be placed in its workspace.
//
// It is deliberately distinct from an ambiguous start: provisioning happens
// before any provider process exists, so this error proves nothing was
// launched. Reporting it as an unknown activation would leave the owner
// reconciling a run that never began.
var ErrAttemptInputProvisioning = errors.New("attempt input provisioning failed")

// AttemptInputRef names one exact retained predecessor result admitted to a
// successor.
//
// The artifact version is part of the reference, not something resolved later:
// admission authorized *these bytes*, and a restart or retry that re-resolved
// "the latest result of that WorkUnit" could silently hand the successor work
// the owner never authorized.
type AttemptInputRef struct {
	AttemptID       domain.AttemptID
	WorkUnitID      domain.WorkUnitID
	ArtifactVersion string
}

// AttemptDocumentInputs names the approved supplied-document snapshot a
// staged Outcome runs against. Naming the revision and digest rather than
// paths is what stops execution re-reading the owner's originals.
type AttemptDocumentInputs struct {
	ContextID domain.DocumentContextID
	Revision  int64
	Digest    string
}

// AttemptInputProvisionRequest is the daemon-owned materialization boundary.
// Every field is derived from durable Attempt and workspace records; no client
// path or provider claim reaches it.
type AttemptInputProvisionRequest struct {
	Inputs []AttemptInputRef
	// Documents is the approved supplied-document snapshot, when this Outcome
	// is a document Outcome. Predecessor artifacts and approved documents
	// share one seam so both obey one fail-closed rule.
	Documents     *AttemptDocumentInputs
	WorkspacePath string
	WorkspaceKind domain.WorkspaceKind
	// BaseRevision is the successor workspace's own resolved base. It must
	// match the base the predecessors produced their changes against.
	BaseRevision string
}

// AttemptInputProvisioner materializes verified predecessor output into a
// successor's freshly created workspace, before any provider is launched.
//
// It is a port rather than a direct dependency because the session manager
// must not learn how artifacts are stored, and because a daemon without
// retention has to fail closed here rather than start a successor on an empty
// base and call that a handoff.
type AttemptInputProvisioner interface {
	ProvisionAttemptInputs(ctx context.Context, req AttemptInputProvisionRequest) error
}

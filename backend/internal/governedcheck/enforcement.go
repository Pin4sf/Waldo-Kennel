package governedcheck

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// ErrEnforcementUnavailable means no mechanism on this host can apply the
// approved filesystem, execution and network limits to this check, so nothing
// was run.
//
// This is a terminal, actionable state, not a reason to degrade. A check that
// runs without its limits is not a weaker check; it is an unbounded process
// wearing a governed check's evidence.
var ErrEnforcementUnavailable = errors.New("no mechanism can enforce the approved limits for this check")

// Enforcement is the boundary that actually applies an AttemptExecutionPolicy
// to a check process.
//
// Argument shape is not enforcement. Rejecting "sh", pinning cwd and filtering
// the environment stop an honest command from wandering; none of them stop a
// command that writes outside the workspace or opens a socket. Only the
// mechanism that owns the process -- in this codebase, the provider adapter
// sandbox the policy is mapped into (Codex TUI --sandbox workspace-write with
// sandbox_workspace_write.network_access=false and an empty writable_roots,
// or Codex Chat's turn-level sandboxPolicy) -- can do that.
//
// There is deliberately no implementation of this interface in production code
// yet. Until a mechanism exists that can be shown to deny an out-of-workspace
// write and a network effect, Run fails closed rather than executing checks
// through bare exec and labelling the result governed.
type Enforcement interface {
	// Name identifies the enforcing mechanism, and is recorded on the result
	// so evidence says what actually held the boundary.
	Name() string
	// Command returns the process to run, already constructed so the mechanism
	// applies the policy. It returns ErrEnforcementUnavailable if it cannot
	// enforce this particular request rather than running it unconfined.
	Command(ctx context.Context, req Request, root string) (*exec.Cmd, error)
}

// Available resolves the enforcement mechanism for an approved policy.
//
// It currently returns ErrEnforcementUnavailable on every host. The mapping
// from AttemptExecutionPolicy to a provider sandbox lives in the adapters and
// is not reachable as a standalone check runner yet; claiming otherwise would
// put a governed label on an unenforced process.
func Available(policy domain.AttemptExecutionPolicy) (Enforcement, error) {
	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("invalid execution policy: %w", err)
	}
	return nil, fmt.Errorf(
		"%w: no sandbox mechanism is wired for standalone checks on this host",
		ErrEnforcementUnavailable,
	)
}

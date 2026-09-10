// Package governedcheck is the deterministic check boundary for an admitted
// WorkUnit. It accepts a discrete executable/argument vector, never a shell
// string, and never inherits the daemon's environment.
//
// Those are hygiene, not enforcement. The approved filesystem, execution and
// network limits are applied by an Enforcement mechanism, and Run refuses to
// execute anything when none is available. See enforcement.go.
package governedcheck

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

var (
	// ErrCapabilityDenied means the approved Attempt lacks worktree execution.
	ErrCapabilityDenied = errors.New("governed check capability denied")
	// ErrInvalidCommand means the check is not a discrete allowed command.
	ErrInvalidCommand = errors.New("governed check command is invalid")
	// ErrOutputLimit means the bounded check output was exceeded.
	ErrOutputLimit = errors.New("governed check output limit exceeded")
)

// Request is the immutable execution boundary for one deterministic check.
type Request struct {
	Policy        domain.AttemptExecutionPolicy
	WorkspaceRoot string
	Argv          []string
	Environment   map[string]string
	Timeout       time.Duration
	// Enforcement applies the approved limits. When nil, Run resolves the
	// mechanism available on this host and fails closed if there is none.
	Enforcement    Enforcement
	MaxOutputBytes int
}

// Result records bounded check output and termination facts.
type Result struct {
	Output   string
	ExitCode int
	// EnforcedBy names the mechanism that held the boundary, so evidence can
	// never imply a confinement that did not exist.
	EnforcedBy      string
	TimedOut        bool
	Cancelled       bool
	OutputTruncated bool
	// TerminationUnknown means the check's process tree could not be confirmed
	// stopped within the bounded wait. Custody is retained: an unknown
	// termination is not a finished check and never releases the workspace.
	TerminationUnknown bool
}

const (
	// killGrace is how long the process group has after SIGTERM before the
	// group is killed outright.
	killGrace = 2 * time.Second
	// waitSlack bounds Wait itself, so a descendant holding the output pipe
	// open cannot keep Run from returning.
	waitSlack = 3 * time.Second
)

// Run executes one check under the supplied Attempt policy.
func Run(ctx context.Context, req Request) (Result, error) {
	if err := req.Policy.Validate(); err != nil {
		return Result{}, fmt.Errorf("invalid execution policy: %w", err)
	}
	if !req.Policy.Has(domain.CapabilityWorktreeExec) {
		return Result{}, ErrCapabilityDenied
	}
	if len(req.Argv) == 0 || strings.TrimSpace(req.Argv[0]) == "" {
		return Result{}, ErrInvalidCommand
	}
	if shellLike(req.Argv[0]) || strings.ContainsAny(req.Argv[0], `/\`) {
		return Result{}, fmt.Errorf("%w: executable must be a discrete allowlisted name", ErrInvalidCommand)
	}
	root, err := confinedRoot(req.WorkspaceRoot)
	if err != nil {
		return Result{}, err
	}
	for _, arg := range req.Argv {
		if strings.IndexByte(arg, 0) >= 0 {
			return Result{}, ErrInvalidCommand
		}
	}
	if req.MaxOutputBytes <= 0 {
		req.MaxOutputBytes = 1 << 20
	}
	if req.Timeout <= 0 {
		req.Timeout = 2 * time.Minute
	}
	// Resolve enforcement before doing anything with a process. If the approved
	// limits cannot be applied, no check runs at all.
	enforcement := req.Enforcement
	if enforcement == nil {
		if enforcement, err = Available(req.Policy); err != nil {
			return Result{}, err
		}
	}
	ctx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()
	cmd, err := enforcement.Command(ctx, req, root)
	if err != nil {
		return Result{}, err
	}
	if cmd == nil {
		return Result{}, fmt.Errorf("%w: %s returned no command", ErrEnforcementUnavailable, enforcement.Name())
	}
	configureProcessGroup(cmd)
	// Cancellation must reach the whole group the check created. The default
	// signals only the leader, so a check that spawned children would leave
	// them running -- and holding the output pipe -- after Run returned.
	cmd.Cancel = func() error { return terminateTree(cmd, killGrace) }
	cmd.WaitDelay = killGrace + waitSlack
	var output boundedBuffer
	output.limit = req.MaxOutputBytes
	cmd.Stdout, cmd.Stderr = &output, &output
	err = cmd.Run()
	result := Result{Output: output.String(), ExitCode: 0, EnforcedBy: enforcement.Name(), OutputTruncated: output.truncated}
	if errors.Is(err, exec.ErrWaitDelay) {
		// The process tree outlived the bounded wait. Report it rather than
		// letting a still-running descendant look like a completed check.
		result.TerminationUnknown = true
	}
	if output.truncated {
		return result, ErrOutputLimit
	}
	if ctx.Err() != nil {
		result.Cancelled = errors.Is(ctx.Err(), context.Canceled)
		result.TimedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
		return result, ctx.Err()
	}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	}
	return result, err
}

func shellLike(name string) bool {
	switch strings.ToLower(filepath.Base(name)) {
	case "sh", "bash", "zsh", "fish", "cmd", "powershell", "pwsh":
		return true
	}
	return false
}

func confinedRoot(raw string) (string, error) {
	if !filepath.IsAbs(raw) {
		return "", errors.New("governed check workspace must be absolute")
	}
	root, err := filepath.EvalSymlinks(filepath.Clean(raw))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", errors.New("governed check workspace is not a directory")
	}
	return root, nil
}

func safeEnvironment(extra map[string]string) []string {
	env := []string{"PATH=/usr/local/bin:/usr/bin:/bin", "LC_ALL=C", "LANG=C"}
	for key, value := range extra {
		if key == "PATH" || key == "HOME" || key == "SHELL" || strings.ContainsAny(key, "=\x00") {
			continue
		}
		if strings.IndexByte(value, 0) >= 0 {
			continue
		}
		env = append(env, key+"="+value)
	}
	return env
}

type boundedBuffer struct {
	bytes.Buffer
	limit     int
	truncated bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = b.Buffer.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	return b.Buffer.Write(p)
}

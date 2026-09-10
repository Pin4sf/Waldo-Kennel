// Package governedcheck is the deterministic check boundary for an admitted
// WorkUnit. It accepts a discrete executable/argument vector, never a shell
// string, and never inherits the daemon's environment.
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
	ErrCapabilityDenied = errors.New("governed check capability denied")
	ErrInvalidCommand   = errors.New("governed check command is invalid")
	ErrOutputLimit      = errors.New("governed check output limit exceeded")
)

type Request struct {
	Policy         domain.AttemptExecutionPolicy
	WorkspaceRoot  string
	Argv           []string
	Environment    map[string]string
	Timeout        time.Duration
	MaxOutputBytes int
}

type Result struct {
	Output          string
	ExitCode        int
	TimedOut        bool
	Cancelled       bool
	OutputTruncated bool
}

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
	ctx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, req.Argv[0], req.Argv[1:]...)
	cmd.Dir = root
	cmd.Env = safeEnvironment(req.Environment)
	configureProcessGroup(cmd)
	var output boundedBuffer
	output.limit = req.MaxOutputBytes
	cmd.Stdout, cmd.Stderr = &output, &output
	err = cmd.Run()
	result := Result{Output: output.String(), ExitCode: 0, OutputTruncated: output.truncated}
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

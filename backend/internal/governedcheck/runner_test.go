package governedcheck

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func checkPolicy() domain.AttemptExecutionPolicy {
	return domain.AttemptExecutionPolicy{OutcomeID: "o", PlanRevisionID: "p", WorkUnitID: "u", ContractRevisionNumber: 1, RunBriefCoreDigest: "brief", RequiredCapabilities: []string{domain.CapabilityWorktreeExec}, Grants: []domain.CapabilityGrant{{ID: "g", Name: domain.CapabilityWorktreeExec, Scope: "worktree/*"}}}
}

// unenforced runs a check with no confinement whatsoever. It exists only to
// exercise the runner's own mechanics -- bounded output, timeout, process-tree
// termination -- and lives in a _test.go file so production code cannot reach
// it even by mistake. Its Name says exactly what it is, so any evidence
// produced with it is labelled unenforced rather than governed.
type unenforced struct {
	// script, when set, replaces the request argv with a shell script. Only a
	// test needs this: real enforcement builds the command from the argv.
	script string
}

func (unenforced) Name() string { return "unenforced-test-runner" }

func (u unenforced) Command(ctx context.Context, req Request, root string) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	if u.script != "" {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", u.script)
	} else {
		cmd = exec.CommandContext(ctx, req.Argv[0], req.Argv[1:]...)
	}
	cmd.Dir = root
	cmd.Env = safeEnvironment(req.Environment)
	return cmd, nil
}

// SP1. The production path has no enforcement mechanism, so a check must fail
// closed and run nothing. The falsifier is the canary: argv and cwd assertions
// would pass either way, but a file outside the workspace proves whether the
// process ran.
func TestRunFailsClosedWithoutAnEnforcementMechanism(t *testing.T) {
	outside := t.TempDir()
	canary := filepath.Join(outside, "canary")
	if err := os.WriteFile(canary, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Run(context.Background(), Request{
		Policy: checkPolicy(), WorkspaceRoot: t.TempDir(),
		Argv: []string{"touch", canary}, Timeout: 2 * time.Second,
	})
	if !errors.Is(err, ErrEnforcementUnavailable) {
		t.Fatalf("err = %v, want ErrEnforcementUnavailable", err)
	}
	if result.EnforcedBy != "" || result.ExitCode != 0 || result.Output != "" {
		t.Fatalf("a refused check produced a result: %#v", result)
	}
	body, readErr := os.ReadFile(canary)
	if readErr != nil {
		t.Fatalf("out-of-workspace canary was removed: %v", readErr)
	}
	if string(body) != "untouched" {
		t.Fatal("out-of-workspace canary was modified: the check ran despite having no enforcement")
	}
}

// Available must never quietly hand back a permissive mechanism.
func TestAvailableReportsNoMechanism(t *testing.T) {
	mechanism, err := Available(checkPolicy())
	if mechanism != nil {
		t.Fatalf("resolved an enforcement mechanism %q that cannot be shown to confine anything", mechanism.Name())
	}
	if !errors.Is(err, ErrEnforcementUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

// The unenforced test runner must be honest about what it is, so no evidence
// produced with it can read as governed.
func TestUnenforcedRunnerIsLabelledOnTheResult(t *testing.T) {
	workspace := t.TempDir()
	result, err := Run(context.Background(), Request{
		Policy: checkPolicy(), WorkspaceRoot: workspace, Enforcement: unenforced{},
		Argv: []string{"touch", "check-output"}, Timeout: 5 * time.Second,
	})
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("run = %#v, %v", result, err)
	}
	if result.EnforcedBy != "unenforced-test-runner" {
		t.Fatalf("EnforcedBy = %q", result.EnforcedBy)
	}
	if _, err := os.Stat(filepath.Join(workspace, "check-output")); err != nil {
		t.Fatal(err)
	}
}

// ST2. Cancellation must reach the descendants, not just the leader. The
// script backgrounds a child and records its pid; after Run returns, that pid
// must be gone.
func TestRunStopsTheWholeProcessTreeOnTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process groups are configured only on unix")
	}
	workspace := t.TempDir()
	pidFile := filepath.Join(workspace, "child.pid")
	script := "sleep 45 & echo $! > " + pidFile + "; wait"

	started := time.Now()
	result, err := Run(context.Background(), Request{
		Policy: checkPolicy(), WorkspaceRoot: workspace,
		Enforcement: unenforced{script: script},
		Argv:        []string{"spawn-detached-child"}, Timeout: 400 * time.Millisecond,
	})
	elapsed := time.Since(started)
	if err == nil {
		t.Fatal("a timed-out check reported success")
	}
	if !result.TimedOut {
		t.Fatalf("result = %#v, want TimedOut", result)
	}
	if elapsed > killGrace+waitSlack+5*time.Second {
		t.Fatalf("Run blocked for %s: the bounded wait did not hold", elapsed)
	}

	pid := readChildPID(t, pidFile)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	t.Fatalf("child %d outlived the check: cancellation reached only the group leader", pid)
}

func readChildPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		body, err := os.ReadFile(path)
		if err == nil && strings.TrimSpace(string(body)) != "" {
			pid, convErr := strconv.Atoi(strings.TrimSpace(string(body)))
			if convErr == nil {
				return pid
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("child never recorded its pid at %s", path)
	return 0
}

func TestRunRejectsShellAndCapabilityWidening(t *testing.T) {
	if _, err := Run(context.Background(), Request{Policy: checkPolicy(), Enforcement: unenforced{}, WorkspaceRoot: t.TempDir(), Argv: []string{"sh", "-c", "touch outside"}}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("shell err = %v", err)
	}
	policy := checkPolicy()
	policy.RequiredCapabilities = []string{domain.CapabilityWorktreeRead}
	policy.Grants[0].Name = domain.CapabilityWorktreeRead
	if _, err := Run(context.Background(), Request{Policy: policy, Enforcement: unenforced{}, WorkspaceRoot: t.TempDir(), Argv: []string{"true"}}); !errors.Is(err, ErrCapabilityDenied) {
		t.Fatalf("capability err = %v", err)
	}
}

// Capability and command validation must be refused before any process starts,
// not after.
func TestDeniedCheckStartsNoProcess(t *testing.T) {
	workspace := t.TempDir()
	policy := checkPolicy()
	policy.RequiredCapabilities = []string{domain.CapabilityWorktreeRead}
	policy.Grants[0].Name = domain.CapabilityWorktreeRead
	if _, err := Run(context.Background(), Request{
		Policy: policy, Enforcement: unenforced{}, WorkspaceRoot: workspace,
		Argv: []string{"touch", "should-not-exist"}, Timeout: time.Second,
	}); !errors.Is(err, ErrCapabilityDenied) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "should-not-exist")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a capability-denied check still ran its command")
	}
}

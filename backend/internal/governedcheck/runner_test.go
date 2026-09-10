package governedcheck

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
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

// SP1, falsifier one. A check that tries to write outside its workspace must be
// denied and the out-of-workspace canary must be byte-identical afterwards.
// argv and cwd assertions pass whether or not the process was confined; only
// the canary distinguishes them.
func TestEnforcedCheckCannotWriteOutsideItsWorkspace(t *testing.T) {
	requireEnforcement(t)
	outside := t.TempDir()
	canary := filepath.Join(outside, "canary")
	if err := os.WriteFile(canary, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	escape := filepath.Join(outside, "escaped")

	result, err := Run(context.Background(), Request{
		Policy: writePolicy(), WorkspaceRoot: t.TempDir(),
		Argv: []string{"touch", escape}, Timeout: 20 * time.Second,
	})
	if err == nil {
		t.Fatalf("an out-of-workspace write succeeded: %#v", result)
	}
	if _, statErr := os.Stat(escape); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("the denied check created %s anyway", escape)
	}
	body, readErr := os.ReadFile(canary)
	if readErr != nil || string(body) != "untouched" {
		t.Fatalf("out-of-workspace canary changed: %q err=%v", body, readErr)
	}
	if result.EnforcedBy == "" {
		t.Fatal("a denied check must still record which mechanism denied it")
	}
}

// SP1, falsifier two. A check that attempts a network effect against a
// controlled local endpoint must produce no request at that endpoint. Asserting
// only that the command failed would pass for any unrelated error.
func TestEnforcedCheckCannotReachTheNetwork(t *testing.T) {
	requireEnforcement(t)
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Prove the endpoint counts a real request, so a zero below means denial
	// rather than a broken fixture.
	if resp, err := http.Get(server.URL); err == nil {
		_ = resp.Body.Close()
	}
	if requests.Load() != 1 {
		t.Fatalf("controlled endpoint did not observe its own probe: %d", requests.Load())
	}

	result, err := Run(context.Background(), Request{
		Policy: writePolicy(), WorkspaceRoot: t.TempDir(),
		Argv: []string{"curl", "-s", "-m", "5", server.URL}, Timeout: 25 * time.Second,
	})
	if err == nil {
		t.Fatalf("a network call from inside a governed check succeeded: %#v", result)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("the endpoint observed %d requests; the check reached the network", got-1)
	}
}

// Enforcement must not be indiscriminate: the check still has to be able to do
// its job inside the workspace it was granted.
func TestEnforcedCheckCanWriteInsideItsWorkspace(t *testing.T) {
	requireEnforcement(t)
	workspace := t.TempDir()
	result, err := Run(context.Background(), Request{
		Policy: writePolicy(), WorkspaceRoot: workspace,
		Argv: []string{"touch", "check-output"}, Timeout: 20 * time.Second,
	})
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("an authorized in-workspace write was denied: %#v %v", result, err)
	}
	root, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "check-output")); err != nil {
		t.Fatalf("the check did not produce its output: %v", err)
	}
}

// Without worktree.write the check gets no filesystem write at all, not a
// narrower one.
func TestReadOnlyPolicyDeniesEvenTheWorkspace(t *testing.T) {
	requireEnforcement(t)
	workspace := t.TempDir()
	policy := checkPolicy() // exec + read, no write
	if _, err := Run(context.Background(), Request{
		Policy: policy, WorkspaceRoot: workspace,
		Argv: []string{"touch", "should-not-exist"}, Timeout: 20 * time.Second,
	}); err == nil {
		t.Fatal("a read-only policy allowed a workspace write")
	}
	root, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "should-not-exist")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a read-only check wrote to its workspace")
	}
}

// A capability the platform cannot translate is refused, never approximated.
// Running with an unmapped capability would grant more than was approved.
func TestUnmappableCapabilityFailsClosed(t *testing.T) {
	outside := t.TempDir()
	canary := filepath.Join(outside, "canary")
	if err := os.WriteFile(canary, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	policy := checkPolicy()
	// Capabilities are stored sorted; "network.egress" sorts first.
	policy.RequiredCapabilities = []string{"network.egress", domain.CapabilityWorktreeExec}
	policy.Grants = []domain.CapabilityGrant{
		{ID: "g-net", Name: "network.egress", Scope: "*"},
		{ID: "g", Name: domain.CapabilityWorktreeExec, Scope: "worktree/*"},
	}

	result, err := Run(context.Background(), Request{
		Policy: policy, WorkspaceRoot: t.TempDir(),
		Argv: []string{"touch", canary}, Timeout: 5 * time.Second,
	})
	if !errors.Is(err, ErrEnforcementUnavailable) {
		t.Fatalf("err = %v, want ErrEnforcementUnavailable", err)
	}
	if result.EnforcedBy != "" || result.Output != "" {
		t.Fatalf("a refused check produced a result: %#v", result)
	}
	body, readErr := os.ReadFile(canary)
	if readErr != nil || string(body) != "untouched" {
		t.Fatal("the out-of-workspace canary changed: the check ran despite having no enforcement")
	}
}

// requireEnforcement skips where no mechanism exists, so an unenforceable host
// records these rows as skipped rather than silently passing them.
func requireEnforcement(t *testing.T) {
	t.Helper()
	mechanism, err := Available(writePolicy())
	if errors.Is(err, ErrEnforcementUnavailable) {
		t.Skipf("no enforcement mechanism on this host: %v", err)
	}
	if err != nil || mechanism == nil {
		t.Fatalf("resolve enforcement: %v", err)
	}
}

func writePolicy() domain.AttemptExecutionPolicy {
	policy := checkPolicy()
	policy.RequiredCapabilities = []string{domain.CapabilityWorktreeExec, domain.CapabilityWorktreeWrite}
	policy.Grants = []domain.CapabilityGrant{
		{ID: "g", Name: domain.CapabilityWorktreeExec, Scope: "worktree/*"},
		{ID: "g-write", Name: domain.CapabilityWorktreeWrite, Scope: "worktree/*"},
	}
	return policy
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

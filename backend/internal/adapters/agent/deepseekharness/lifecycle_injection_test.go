package deepseekharness

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/deepseekharness/dshtest"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// The lifecycle scaffold exercises the seams a real dsh integration will
// replace, using throwaway fake binaries on PATH instead of the real CLI:
//
//   - a missing binary fails closed with ErrAgentBinaryNotFound before any
//     runtime is created (the truthful "not ready" state);
//   - a fake binary on PATH is resolved and the adapter's launch argv actually
//     executes it, proving the after-start delivery contract has something to
//     talk to;
//   - a fake binary that exits immediately still resolves: binary presence
//     never claims a healthy run, which is exactly why Kennel owns run-state
//     truth in the session manager rather than in resolution.

// writeFakeDSH delegates script emission to the exported dshtest package so
// the in-package scaffold and every cross-package consumer share one fixture
// implementation (issue #60: importable failure-injection fixtures).
func writeFakeDSH(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	dshtest.WriteScriptAt(t, path, body)
	return path
}

func fakePATH(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("PATH", dir)
	restore := narrowBinarySpecForTest(t)
	t.Cleanup(restore)
}

// A missing dsh binary is the truthful not-ready admission: the adapter
// surfaces ErrAgentBinaryNotFound so the session manager reports Action
// Required instead of launching into an empty pane.
func TestLifecycleMissingBinaryFailsClosedBeforeRuntime(t *testing.T) {
	fakePATH(t, t.TempDir())

	plugin := &Plugin{}
	if _, err := plugin.GetLaunchCommand(context.Background(), ports.LaunchConfig{
		Prompt: "hello",
		Config: ports.AgentConfig{Profile: "tui"},
	}); !errors.Is(err, ports.ErrAgentBinaryNotFound) {
		t.Fatalf("GetLaunchCommand err = %v, want ErrAgentBinaryNotFound", err)
	}
	// The readiness probe surfaces the same truth rather than a verdict.
	if _, err := plugin.ProfileReadiness(context.Background(), ports.AgentConfig{Profile: "tui"}); !errors.Is(err, ports.ErrAgentBinaryNotFound) {
		t.Fatalf("ProfileReadiness err = %v, want ErrAgentBinaryNotFound", err)
	}
}

// With any executable named dsh on PATH, the adapter resolves it and its
// launch argv runs that exact binary — the seam a real install replaces.
func TestLifecycleFakeBinaryIsResolvedAndLaunched(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "launched")
	writeFakeDSH(t, dir, "dsh", "echo started > "+marker)
	fakePATH(t, dir)

	plugin := &Plugin{}
	cmd, err := plugin.GetLaunchCommand(context.Background(), ports.LaunchConfig{
		Prompt:      "record one focus block",
		Permissions: ports.PermissionModeBypassPermissions,
		Config:      ports.AgentConfig{Profile: "tui"},
	})
	if err != nil {
		t.Fatalf("GetLaunchCommand err: %v", err)
	}
	want := []string{cmd[0], "--profile", "tui"}
	if !reflect.DeepEqual(cmd, want) {
		t.Fatalf("cmd = %#v, want %#v (resolved fake + profile flag only)", cmd, want)
	}
	if output, err := execCommandOutput(cmd[0]); err != nil || output != "" {
		t.Fatalf("launch argv run = (%q, %v)", output, err)
	}
	if data, readErr := os.ReadFile(marker); readErr != nil || string(data) == "" {
		t.Fatalf("fake dsh did not run (marker=%q err=%v)", marker, readErr)
	}
}

// A binary that dies immediately still resolves. Resolution truthfully says
// nothing about run health; unknown-outcome handling belongs to Attempt
// observation and recovery, never to binary lookup.
func TestLifecycleCrashingBinaryStillResolves(t *testing.T) {
	dir := t.TempDir()
	writeFakeDSH(t, dir, "dsh", "exit 3")
	fakePATH(t, dir)

	plugin := &Plugin{}
	binary, err := plugin.ResolveBinary(context.Background())
	if err != nil {
		t.Fatalf("ResolveBinary err: %v", err)
	}
	if filepath.Base(binary) != "dsh" {
		t.Fatalf("resolved %q, want the crashing fake dsh", binary)
	}
}

// Profile readiness composes the configured profile via the verified
// `--dump-config` contract: exit 0 means launchable.
func TestReadinessComposesConfiguredProfile(t *testing.T) {
	dir := t.TempDir()
	writeFakeDSH(t, dir, "dsh", `echo composed`)
	fakePATH(t, dir)

	plugin := &Plugin{}
	readiness, err := plugin.ProfileReadiness(context.Background(), ports.AgentConfig{Profile: "tui"})
	if err != nil {
		t.Fatalf("ProfileReadiness err: %v", err)
	}
	if !readiness.Ready {
		t.Fatalf("readiness = %+v, want ready", readiness)
	}
}

func TestReadinessReportsInvalidProfileTruthfully(t *testing.T) {
	dir := t.TempDir()
	writeFakeDSH(t, dir, "dsh", `echo 'profile "broken" does not exist' >&2; exit 1`)
	fakePATH(t, dir)

	plugin := &Plugin{}
	readiness, err := plugin.ProfileReadiness(context.Background(), ports.AgentConfig{Profile: "broken"})
	if err != nil {
		t.Fatalf("ProfileReadiness err: %v", err)
	}
	if readiness.Ready {
		t.Fatalf("readiness = %+v, want not ready", readiness)
	}
	if !strings.Contains(readiness.Detail, "does not exist") {
		t.Fatalf("detail = %q, want the CLI's own failure text", readiness.Detail)
	}
}

// An unselected profile is not-ready with guidance, never a silent pass —
// bare `dsh` cannot launch on the published contract anyway.
func TestReadinessWithoutProfileIsNotReady(t *testing.T) {
	dir := t.TempDir()
	writeFakeDSH(t, dir, "dsh", "exit 0")
	fakePATH(t, dir)

	plugin := &Plugin{}
	readiness, err := plugin.ProfileReadiness(context.Background(), ports.AgentConfig{})
	if err != nil {
		t.Fatalf("ProfileReadiness err: %v", err)
	}
	if readiness.Ready {
		t.Fatal("readiness = ready, want not ready without a selected profile")
	}
	if !strings.Contains(readiness.Detail, "no dsh profile selected") {
		t.Fatalf("detail = %q, want selection guidance", readiness.Detail)
	}
}

// A binary that disappears entirely between resolution and probing surfaces
// the missing-binary truth rather than a fabricated readiness verdict.
func TestReadinessMissingBinaryFailsClosed(t *testing.T) {
	fakePATH(t, t.TempDir())

	plugin := &Plugin{}
	if _, err := plugin.ProfileReadiness(context.Background(), ports.AgentConfig{Profile: "tui"}); !errors.Is(err, ports.ErrAgentBinaryNotFound) {
		t.Fatalf("err = %v, want ErrAgentBinaryNotFound", err)
	}
}

// A probe that outlives its own budget must answer "not ready" with truthful
// timeout wording: a stuck `dsh --dump-config` can never be claimed launchable,
// and the kill is not dressed up as an ordinary exit-status failure.
func TestReadinessProbeTimeoutReportsNotReady(t *testing.T) {
	dir := t.TempDir()
	writeFakeDSH(t, dir, "dsh", "sleep 30")
	fakePATH(t, dir)

	original := profileProbeTimeout
	profileProbeTimeout = 50 * time.Millisecond
	defer func() { profileProbeTimeout = original }()

	plugin := &Plugin{}
	readiness, err := plugin.ProfileReadiness(context.Background(), ports.AgentConfig{Profile: "stuck"})
	if err != nil {
		t.Fatalf("ProfileReadiness err: %v", err)
	}
	if readiness.Ready {
		t.Fatalf("readiness = %+v, want not ready when the probe exceeds its budget", readiness)
	}
	if !strings.Contains(readiness.Detail, "timed out") {
		t.Fatalf("detail = %q, want truthful timeout wording", readiness.Detail)
	}
}

// Binary presence never implied run success: the crashing fake still resolves
// and GetLaunchCommand still produces argv, but executing that argv returns an
// exec error carrying the binary's own exit status. Run-state truth stays with
// Attempt observation in the session manager, never with resolution.
func TestLifecycleResolvedLaunchArgvStillSurfacesExecFailure(t *testing.T) {
	dir := t.TempDir()
	writeFakeDSH(t, dir, "dsh", "echo boom >&2\nexit 7")
	fakePATH(t, dir)

	plugin := &Plugin{}
	cmd, err := plugin.GetLaunchCommand(context.Background(), ports.LaunchConfig{
		Prompt: "record one focus block",
		Config: ports.AgentConfig{Profile: "tui"},
	})
	if err != nil {
		t.Fatalf("GetLaunchCommand err: %v", err)
	}

	_, runErr := execCommandOutput(cmd[0])
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) {
		t.Fatalf("launch argv run err = %v, want an exec error from the exiting binary", runErr)
	}
	if code := exitErr.ExitCode(); code != 7 {
		t.Fatalf("exit code = %d, want the fake's own status 7", code)
	}
}

func execCommandOutput(path string) (string, error) {
	out, err := exec.Command(path).Output() //nolint:gosec // test-controlled fake binary
	return string(out), err
}

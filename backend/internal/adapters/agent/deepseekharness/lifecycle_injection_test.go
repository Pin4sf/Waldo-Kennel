package deepseekharness

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
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

func writeFakeDSH(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	script := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake dsh: %v", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("fake shell binaries are unix-only")
	}
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
	if _, err := plugin.GetLaunchCommand(context.Background(), ports.LaunchConfig{Prompt: "hello"}); !errors.Is(err, ports.ErrAgentBinaryNotFound) {
		t.Fatalf("GetLaunchCommand err = %v, want ErrAgentBinaryNotFound", err)
	}
	status, err := plugin.AuthStatus(context.Background())
	if err == nil || status != ports.AgentAuthStatusUnknown {
		t.Fatalf("AuthStatus = (%q, %v), want unknown with error", status, err)
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
	})
	if err != nil {
		t.Fatalf("GetLaunchCommand err: %v", err)
	}
	if len(cmd) != 1 || filepath.Base(cmd[0]) != "dsh" {
		t.Fatalf("cmd = %#v, want the resolved fake dsh only", cmd)
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

func execCommandOutput(path string) (string, error) {
	out, err := exec.Command(path).Output() //nolint:gosec // test-controlled fake binary
	return string(out), err
}

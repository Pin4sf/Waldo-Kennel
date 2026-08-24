package deepseekharness

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/adapters"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/binaryutil"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

func TestManifest(t *testing.T) {
	m := (&Plugin{}).Manifest()
	if m.ID != "deepseek-harness" {
		t.Fatalf("ID = %q, want deepseek-harness", m.ID)
	}
	if m.Name != "DeepSeek Harness" {
		t.Fatalf("Name = %q", m.Name)
	}
	hasAgent := false
	for _, c := range m.Capabilities {
		if c == adapters.CapabilityAgent {
			hasAgent = true
		}
	}
	if !hasAgent {
		t.Fatal("missing CapabilityAgent")
	}
}

func TestGetConfigSpecAdvertisesProfileMode(t *testing.T) {
	spec, err := (&Plugin{}).GetConfigSpec(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(spec.Fields) != 1 || spec.Fields[0].Key != "mode" {
		t.Fatalf("fields = %#v, want exactly the dsh profile mode", spec.Fields)
	}
	if spec.Fields[0].Type != ports.ConfigFieldString {
		t.Fatalf("type = %q, want free text: dsh profiles are user-built, not a closed enum", spec.Fields[0].Type)
	}
}

func TestGetPromptDeliveryStrategy(t *testing.T) {
	s, err := (&Plugin{}).GetPromptDeliveryStrategy(context.Background(), ports.LaunchConfig{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if s != ports.PromptDeliveryAfterStart {
		t.Fatalf("strategy = %q, want after_start", s)
	}
}

func TestPromptReadinessHintsDeclareNoPatterns(t *testing.T) {
	hints, err := (&Plugin{}).PromptReadinessHints(context.Background(), ports.LaunchConfig{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if hints.InitialDelay <= 0 {
		t.Fatalf("InitialDelay = %v, want a short settle delay", hints.InitialDelay)
	}
	if len(hints.Patterns) != 0 {
		t.Fatalf("patterns = %#v, want none: no dsh banner is verified", hints.Patterns)
	}
}

// GetLaunchCommand must stay free of invented flags: bare binary only, even
// when an explicit permission mode is requested, because sessions run under
// dsh's native approval settings until a flag mapping is verified.
func TestGetLaunchCommandBareArgvUnderAnyPermissionMode(t *testing.T) {
	for _, mode := range []ports.PermissionMode{
		ports.PermissionModeDefault,
		ports.PermissionModeAcceptEdits,
		ports.PermissionModeAuto,
		ports.PermissionModeBypassPermissions,
	} {
		plugin := &Plugin{resolvedBinary: "dsh"}
		cmd, err := plugin.GetLaunchCommand(context.Background(), ports.LaunchConfig{
			Prompt:      "do the thing",
			Permissions: mode,
		})
		if err != nil {
			t.Fatalf("mode %v err: %v", mode, err)
		}
		if want := []string{"dsh"}; !reflect.DeepEqual(cmd, want) {
			t.Fatalf("mode %v cmd = %#v, want %#v", mode, cmd, want)
		}
	}
}

// A stored model override must fail closed rather than launch with an
// unverified flag that could select something else entirely.
func TestGetLaunchCommandRejectsModelOverride(t *testing.T) {
	plugin := &Plugin{resolvedBinary: "dsh"}
	_, err := plugin.GetLaunchCommand(context.Background(), ports.LaunchConfig{
		Config: ports.AgentConfig{Model: "deepseek-chat"},
	})
	if err == nil {
		t.Fatal("expected model override to fail closed")
	}
	if !strings.Contains(err.Error(), "model override") {
		t.Fatalf("err = %v, want a model-override explanation", err)
	}
}

// A configured mode selects the dsh profile to boot via the one verified
// launcher flag; empty mode defers to dsh's own default.
func TestGetLaunchCommandAppendsProfileFlag(t *testing.T) {
	plugin := &Plugin{resolvedBinary: "dsh"}
	cmd, err := plugin.GetLaunchCommand(context.Background(), ports.LaunchConfig{
		Config: ports.AgentConfig{Mode: " tui "},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if want := []string{"dsh", "--profile", "tui"}; !reflect.DeepEqual(cmd, want) {
		t.Fatalf("cmd = %#v, want %#v", cmd, want)
	}
}

// Native resume stays unavailable until the dsh CLI contract lands; Kennel
// falls back to a fresh launch with standing instructions.
func TestGetRestoreCommandFallsBackUntilCLIContract(t *testing.T) {
	plugin := &Plugin{resolvedBinary: "dsh"}
	for _, metadata := range []map[string]string{nil, {ports.MetadataKeyAgentSessionID: "native-123"}} {
		cmd, ok, err := plugin.GetRestoreCommand(context.Background(), ports.RestoreConfig{
			Session: ports.SessionRef{Metadata: metadata},
		})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if ok || cmd != nil {
			t.Fatalf("ok = %v cmd = %#v, want fallback (false, nil)", ok, cmd)
		}
	}
}

func TestSessionInfoReadsHookMetadata(t *testing.T) {
	plugin := &Plugin{}
	info, ok, err := plugin.SessionInfo(context.Background(), ports.SessionRef{
		ID: "session-1",
		Metadata: map[string]string{
			ports.MetadataKeyAgentSessionID: "native-123",
			ports.MetadataKeyTitle:          "Fix the flake",
		},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !ok {
		t.Fatal("ok = false, want true when hook metadata exists")
	}
	if info.AgentSessionID != "native-123" || info.Title != "Fix the flake" {
		t.Fatalf("info = %#v", info)
	}
}

func TestSessionInfoFalseWhenNoHookMetadata(t *testing.T) {
	plugin := &Plugin{}
	_, ok, err := plugin.SessionInfo(context.Background(), ports.SessionRef{ID: "session-1"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ok {
		t.Fatal("ok = true, want false without hook metadata")
	}
}

// The adapter writes nothing into user worktrees: hooks await the real dsh
// contract, so install/uninstall/status are documented no-ops.
func TestGetAgentHooksInstallsNothing(t *testing.T) {
	plugin := &Plugin{}
	ws := t.TempDir()
	if err := plugin.GetAgentHooks(context.Background(), ports.WorkspaceHookConfig{WorkspacePath: ws}); err != nil {
		t.Fatalf("GetAgentHooks err: %v", err)
	}
	installed, err := plugin.AreHooksInstalled(context.Background(), ws)
	if err != nil {
		t.Fatalf("AreHooksInstalled err: %v", err)
	}
	if installed {
		t.Fatal("AreHooksInstalled = true, want false for a pristine workspace")
	}
	if err := plugin.UninstallHooks(context.Background(), ws); err != nil {
		t.Fatalf("UninstallHooks err: %v", err)
	}
}

func TestAuthStatusEnvKeyIsAdvisoryAuthorized(t *testing.T) {
	plugin := &Plugin{resolvedBinary: "dsh"}
	t.Setenv("DEEPSEEK_API_KEY", "test-key")
	status, err := plugin.AuthStatus(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if status != ports.AgentAuthStatusAuthorized {
		t.Fatalf("status = %q, want authorized", status)
	}
}

func TestResolveBinaryMissingFailsClosed(t *testing.T) {
	restore := narrowBinarySpecForTest(t)
	defer restore()
	// Empty PATH and HOME: nothing is resolvable even on machines where the
	// real dsh CLI (or a pnpm shim) happens to be installed.
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	plugin := &Plugin{}
	_, err := plugin.ResolveBinary(context.Background())
	if !errors.Is(err, ports.ErrAgentBinaryNotFound) {
		t.Fatalf("err = %v, want ErrAgentBinaryNotFound", err)
	}
}

// narrowBinarySpecForTest strips well-known machine install paths while
// keeping executable names, so resolution only sees the test-controlled
// environment and failure cases stay hermetic.
func narrowBinarySpecForTest(t *testing.T) func() {
	t.Helper()
	orig := dshBinarySpec
	dshBinarySpec = binaryutil.BinarySpec{
		Label:    "dsh",
		Names:    []string{"dsh", "deepseek-harness"},
		WinNames: []string{"dsh.cmd", "dsh.exe", "dsh"},
	}
	return func() { dshBinarySpec = orig }
}

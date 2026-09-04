package registry

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/hookutil"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// Any hook file written into a worktree must be covered by a Kennel-managed
// sibling .gitignore, otherwise worktree cleanup can be permanently blocked.
func TestGetAgentHooksFootprintIsGitignored(t *testing.T) {
	for _, ha := range Harnessed() {
		t.Run(string(ha.Harness), func(t *testing.T) {
			ws := t.TempDir()
			cfg := ports.WorkspaceHookConfig{
				SessionID:     "proj-1",
				WorkspacePath: ws,
				DataDir:       t.TempDir(),
			}
			if err := ha.Agent.GetAgentHooks(context.Background(), cfg); err != nil {
				t.Fatalf("GetAgentHooks: %v", err)
			}
			for _, rel := range workspaceFiles(t, ws) {
				gitignorePath := filepath.Join(ws, filepath.Dir(rel), ".gitignore")
				data, err := os.ReadFile(gitignorePath) //nolint:gosec // test-owned temp dir
				if err != nil {
					t.Errorf("hook file %q has no sibling .gitignore (%v)", rel, err)
					continue
				}
				content := string(data)
				if !strings.Contains(content, hookutil.GitignoreSentinel) {
					t.Errorf(".gitignore next to %q is not Kennel-managed", rel)
					continue
				}
				if entry := "/" + filepath.Base(rel); !hasLine(content, entry) {
					t.Errorf(".gitignore next to %q does not list %q", rel, entry)
				}
			}
		})
	}
}

// Auth is provider-specific. Some providers have a truthful non-interactive
// probe; others deliberately report unknown and let spawn be authoritative.
// What the registry must guarantee is that every active provider can resolve a
// binary and expose usable project configuration without a historical adapter.
func TestEveryProductionHarnessReportsModelOrModeConfig(t *testing.T) {
	for _, ha := range Harnessed() {
		t.Run(string(ha.Harness), func(t *testing.T) {
			spec, err := ha.Agent.GetConfigSpec(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range spec.Fields {
				if field.Key == "model" || field.Key == "mode" {
					return
				}
			}
			t.Fatalf("%s exposes neither model nor mode configuration: %#v", ha.Harness, spec.Fields)
		})
	}
}

func TestHarnessedExcludesFixtureHarness(t *testing.T) {
	for _, ha := range Harnessed() {
		if ha.Harness == domain.HarnessFake {
			t.Fatal("fake harness must never be returned as a shipped provider")
		}
	}
}

// A harness is advertised as a switch target only when its real adapter proves
// the continuation contract required by the switch saga.
func TestSwitchTargetAdmittedHarnessesDeclareContinuation(t *testing.T) {
	admitted := 0
	for _, ha := range Harnessed() {
		if !ha.Harness.IsSelectableAsSwitchTarget() {
			continue
		}
		admitted++
		t.Run(string(ha.Harness), func(t *testing.T) {
			provider, ok := ha.Agent.(ports.AgentContinuationCapabilityProvider)
			if !ok {
				t.Fatalf("%s is switch-target-admitted without continuation capabilities", ha.Harness)
			}
			caps := provider.ContinuationCapabilities()
			switch caps.FreshNativeSessionID {
			case ports.FreshNativeSessionIDProviderAssigned:
			case ports.FreshNativeSessionIDCallerAssigned:
				if _, ok := ha.Agent.(ports.AgentFreshNativeSessionIDProvider); !ok {
					t.Fatalf("%s declares caller-assigned ids without an allocator", ha.Harness)
				}
			default:
				t.Fatalf("%s declares no verified fresh native-session identity mode", ha.Harness)
			}
			if _, ok := ha.Agent.(ports.AgentNativeSessionConfigProvider); !ok {
				t.Fatalf("%s is switch-target-admitted without a native session config dir", ha.Harness)
			}
		})
	}
	if admitted != 3 {
		t.Fatalf("switch-target-admitted harness count = %d, want 3", admitted)
	}
}

func workspaceFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk workspace: %v", err)
	}
	return files
}

func hasLine(content, line string) bool {
	for _, value := range strings.Split(content, "\n") {
		if strings.TrimSpace(value) == line {
			return true
		}
	}
	return false
}

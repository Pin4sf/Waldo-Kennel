package registry

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/hookutil"
	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// TestGetAgentHooksFootprintIsGitignored enforces a contract every shipped
// (and future) adapter must hold: any file GetAgentHooks writes into a session
// worktree must be covered by a sibling Kennel-managed self-ignoring .gitignore
// (hookutil.EnsureWorkspaceGitignore). Hook files are untracked, and
// `git worktree remove` (without --force) refuses on any untracked file — an
// uncovered hook file makes every one of that adapter's session workspaces
// permanently undeletable (kill/cleanup can never free them).
func TestGetAgentHooksFootprintIsGitignored(t *testing.T) {
	for _, ha := range Harnessed() {
		t.Run(string(ha.Harness), func(t *testing.T) {
			ws := t.TempDir()
			if ha.Harness == "autohand" {
				t.Setenv("AUTOHAND_CONFIG", filepath.Join(t.TempDir(), "config.json"))
			}
			cfg := ports.WorkspaceHookConfig{
				SessionID:     "proj-1",
				WorkspacePath: ws,
				DataDir:       t.TempDir(),
			}
			if ha.Harness == "kimi" {
				cfg.Env = map[string]string{"KIMI_CODE_HOME": filepath.Join(cfg.DataDir, "kimi")}
			}
			if err := ha.Agent.GetAgentHooks(context.Background(), cfg); err != nil {
				t.Fatalf("GetAgentHooks: %v", err)
			}
			files := workspaceFiles(t, ws)
			for _, rel := range files {
				gitignorePath := filepath.Join(ws, filepath.Dir(rel), ".gitignore")
				data, err := os.ReadFile(gitignorePath) //nolint:gosec // test-owned temp dir
				if err != nil {
					t.Errorf("hook file %q has no sibling .gitignore (%v); it will keep the session worktree permanently dirty", rel, err)
					continue
				}
				content := string(data)
				if !strings.Contains(content, hookutil.GitignoreSentinel) {
					t.Errorf(".gitignore next to %q is not Kennel-managed (missing sentinel)", rel)
					continue
				}
				if entry := "/" + filepath.Base(rel); !hasLine(content, entry) {
					t.Errorf(".gitignore next to %q does not list %q", rel, entry)
				}
			}
		})
	}
}

func TestEveryHarnessReportsAuthStatus(t *testing.T) {
	authCheckerExempt := map[string]string{
		"continue":    "Continue auth probes require sending a model prompt, so catalog refresh must not run them",
		"prime-agent": "Prime Agent has no documented non-interactive local auth probe; spawn remains authoritative",
		// An API key cannot establish that an arbitrary dsh profile — possibly
		// pointed at a different provider — is launchable, so credential-based
		// probes would overclaim. ProfileReadiness is the verified probe.
		"deepseek-harness": "dsh auth depends on the selected profile; profile readiness is the truthful signal",
	}
	for _, ha := range Harnessed() {
		if reason, exempt := authCheckerExempt[string(ha.Harness)]; exempt {
			if _, ok := ha.Agent.(ports.AgentAuthChecker); ok {
				t.Errorf("%s implements ports.AgentAuthChecker but is exempt: %s", ha.Harness, reason)
			}
			continue
		}
		if _, ok := ha.Agent.(ports.AgentAuthChecker); !ok {
			t.Errorf("%s does not implement ports.AgentAuthChecker", ha.Harness)
		}
	}
}

// TestDeepSeekHarnessAdmitsThroughProfileReadiness pins the replacement
// admission capability: DeepSeek Harness must implement the readiness probe
// its exemption above relies on.
func TestDeepSeekHarnessAdmitsThroughProfileReadiness(t *testing.T) {
	reg, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	adapter, ok := reg.Get("deepseek-harness")
	if !ok {
		t.Fatal("registry does not contain deepseek-harness")
	}
	if _, ok := adapter.(ports.AgentProfileReadinessChecker); !ok {
		t.Fatal("deepseek-harness does not implement ports.AgentProfileReadinessChecker")
	}
}

func TestRegistryIncludesPrimeAgent(t *testing.T) {
	reg, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	adapter, ok := reg.Get("prime-agent")
	if !ok {
		t.Fatal("registry does not contain prime-agent")
	}
	manifest := adapter.Manifest()
	if manifest.Name != "Prime Agent" {
		t.Fatalf("prime-agent manifest name = %q, want Prime Agent", manifest.Name)
	}

	for _, item := range Harnessed() {
		if item.Harness == "prime-agent" {
			return
		}
	}
	t.Fatal("Harnessed does not contain prime-agent")
}

func TestRegistryIncludesOMP(t *testing.T) {
	reg, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	adapter, ok := reg.Get("omp")
	if !ok {
		t.Fatal("registry does not contain omp")
	}
	manifest := adapter.Manifest()
	if manifest.Name != "OMP" {
		t.Fatalf("omp manifest name = %q, want OMP", manifest.Name)
	}

	for _, item := range Harnessed() {
		if item.Harness == domain.HarnessOMP {
			return
		}
	}
	t.Fatal("Harnessed does not contain omp")
}

func TestHarnessedExcludesFakeHarness(t *testing.T) {
	for _, ha := range Harnessed() {
		if ha.Harness == domain.HarnessFake {
			t.Fatal("fake harness must not be returned as a shipped selectable agent")
		}
	}
}

// A shipped harness must give the owner some say in what actually runs. Most
// CLIs spell that as a model or a mode. A few spell it as a profile: DeepSeek
// Harness boots `dsh --profile <name>`, where the profile is the user-built
// plugin stack that pins the model, and its adapter deliberately exposes no
// model field because no dsh model flag has been verified (GetLaunchCommand
// refuses a stored override rather than guessing one). Accepting `profile`
// records that seam instead of pressuring an adapter into advertising a flag
// its CLI does not have. What stays forbidden is a harness that exposes no
// selection seam at all.
func TestEveryProductionHarnessReportsASelectionSeam(t *testing.T) {
	for _, ha := range Harnessed() {
		t.Run(string(ha.Harness), func(t *testing.T) {
			spec, err := ha.Agent.GetConfigSpec(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range spec.Fields {
				switch field.Key {
				case "model", "mode", "profile":
					return
				}
			}
			t.Fatalf("%s exposes no model, mode, or profile configuration: %#v", ha.Harness, spec.Fields)
		})
	}
}

// workspaceFiles returns every regular file under root, relative to root.
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
	for _, l := range strings.Split(content, "\n") {
		if strings.TrimSpace(l) == line {
			return true
		}
	}
	return false
}

// TestSwitchTargetAdmittedHarnessesDeclareContinuation enforces the invariant
// the switch path depends on: a harness may only be switch-target-admitted if
// its shipped adapter actually satisfies session_manager.validateContinuationAgent.
// Widening domain admission without the adapter support would advertise
// switches that can only fail at activation, which is the exact trap the
// DeepSeek Harness gating comment describes. Asserting it here, over the real
// registry, keeps the domain predicate and the adapter capability from drifting.
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
				t.Fatalf("%s is switch-target-admitted without declaring continuation capabilities", ha.Harness)
			}
			caps := provider.ContinuationCapabilities()
			switch caps.FreshNativeSessionID {
			case ports.FreshNativeSessionIDProviderAssigned:
			case ports.FreshNativeSessionIDCallerAssigned:
				// Caller-assigned ids are only usable with an allocator.
				if _, ok := ha.Agent.(ports.AgentFreshNativeSessionIDProvider); !ok {
					t.Fatalf("%s declares caller-assigned ids without an allocator", ha.Harness)
				}
			default:
				t.Fatalf("%s declares no verified fresh native-session identity mode", ha.Harness)
			}
			// Preserving the outgoing conversation resolves the adapter's native
			// state root, so a switch-admitted adapter must expose one.
			if _, ok := ha.Agent.(ports.AgentNativeSessionConfigProvider); !ok {
				t.Fatalf("%s is switch-target-admitted without a native session config dir", ha.Harness)
			}
		})
	}
	if admitted == 0 {
		t.Fatal("no switch-target-admitted harnesses found: the invariant would vacuously pass")
	}
}

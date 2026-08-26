// Package deepseekharness implements the DeepSeek Harness agent adapter.
//
// DeepSeek Harness ("dsh") boots named profiles — ordered stacks of
// plugin-bundle patch layers (verified against `dsh --help`, v0.1.1-rc.2,
// whose contract requires `--profile <name>`: bare `dsh` exits with an
// error). This cut is deliberately narrow and truthful about what is verified
// versus pending:
//
//   - Launch builds `dsh --profile <name>` and nothing more. A profile is
//     REQUIRED and validated ahead of spawn by ProfileReadiness; Kennel
//     delivers the worker's initial task through the terminal after startup
//     (after_start delivery), so no prompt-flag mapping is required.
//   - The adapter exposes one config key, "profile" (AgentConfig.Profile):
//     the dsh profile to boot. Profiles are user-built
//     (`dsh plugin --profile <name> add <package>`), so the field is free
//     text rather than a fixed enum. The legacy mode field stays Amp-only.
//   - Permission modes are not mapped onto dsh flags (no such mapping is
//     documented): sessions run under dsh's own native approval settings,
//     mirroring how the Amp adapter treats undocumented permission flags.
//   - Model overrides fail closed. The adapter rejects a configured model
//     rather than forwarding an unverified flag, so a stored override can
//     never silently launch something else.
//   - Native resume and hook-based session-id capture await the dsh CLI's
//     hook contract. GetRestoreCommand therefore reports ok=false (Kennel
//     falls back to a fresh launch with standing instructions), and
//     GetAgentHooks installs nothing, leaving user worktrees pristine.
//
// As the real dsh CLI contract lands, each pending seam gains its verified
// mapping without changing the adapter's outward contract.
package deepseekharness

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/adapters"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/agentbase"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/binaryutil"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

var dshBinarySpec = binaryutil.BinarySpec{
	Label:         "dsh",
	Names:         []string{"dsh", "deepseek-harness"},
	WinNames:      []string{"dsh.cmd", "dsh.exe", "dsh"},
	UnixPaths:     []string{"/usr/local/bin/dsh", "/opt/homebrew/bin/dsh"},
	UnixHomePaths: binaryutil.NodeManagedUnixHomePaths("dsh", []string{".dsh", "bin", "dsh"}),
	NodeManaged:   true,
}

// Plugin is the DeepSeek Harness agent adapter.
type Plugin struct {
	agentbase.Base
	binaryMu       sync.Mutex
	resolvedBinary string
}

// New returns a ready-to-register DeepSeek Harness adapter.
func New() *Plugin {
	return &Plugin{}
}

var _ adapters.Adapter = (*Plugin)(nil)
var _ ports.Agent = (*Plugin)(nil)

// Manifest returns the adapter's static self-description.
func (p *Plugin) Manifest() adapters.Manifest {
	return adapters.Manifest{
		ID:          "deepseek-harness",
		Name:        "DeepSeek Harness",
		Description: "Run DeepSeek Harness worker sessions.",
		Version:     "0.0.1",
		Capabilities: []adapters.Capability{
			adapters.CapabilityAgent,
		},
	}
}

// GetConfigSpec reports dsh's one verified configuration seam: the profile
// to boot. Profiles are user-built plugin stacks (see `dsh plugin --profile
// <name> add <package>`), so the field is free text rather than a closed
// enum. A profile is required for launch. There is no model field because no
// model flag has been verified; a stored model override fails closed in
// GetLaunchCommand instead.
func (p *Plugin) GetConfigSpec(ctx context.Context) (ports.ConfigSpec, error) {
	if err := ctx.Err(); err != nil {
		return ports.ConfigSpec{}, err
	}
	return ports.ConfigSpec{Fields: []ports.ConfigField{{
		Key:         "profile",
		Type:        ports.ConfigFieldString,
		Description: "Required dsh profile to boot (created via `dsh plugin --profile <name> add <package>`); maps to AgentConfig.Profile.",
	}}}, nil
}

// GetLaunchCommand builds `dsh --profile <name>` and nothing else. A profile
// is REQUIRED — bare `dsh` exits with "--profile <name> is required" on the
// published contract — so an unselected profile fails closed here with an
// actionable message instead of surfacing as a spawn failure. The prompt and
// any standing instructions arrive through after-start terminal delivery. A
// configured model override is refused rather than guessed at; permission
// modes stay untranslated because no dsh permission flag is documented (same
// reasoning as the Amp adapter).
func (p *Plugin) GetLaunchCommand(ctx context.Context, cfg ports.LaunchConfig) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if model := strings.TrimSpace(cfg.Config.Model); model != "" {
		return nil, fmt.Errorf("deepseek-harness: model override %q is not supported until the dsh CLI flag contract is pinned", model)
	}
	profile := strings.TrimSpace(cfg.Config.Profile)
	if profile == "" {
		return nil, fmt.Errorf("deepseek-harness: no dsh profile configured — create one ('dsh plugin --profile <name> add <package>') and select it as this agent's profile")
	}

	binary, err := p.dshBinary(ctx)
	if err != nil {
		return nil, err
	}
	return []string{binary, "--profile", profile}, nil
}

// GetPromptDeliveryStrategy reports that Kennel injects prompted tasks into
// the interactive terminal after startup.
func (p *Plugin) GetPromptDeliveryStrategy(ctx context.Context, _ ports.LaunchConfig) (ports.PromptDeliveryStrategy, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return ports.PromptDeliveryAfterStart, nil
}

// PromptReadinessHints waits a short fixed delay before delivering the first
// task. No banner patterns are declared because none are verified; empty
// patterns make the manager deliver immediately after the delay instead of
// guessing at dsh's startup UI.
func (p *Plugin) PromptReadinessHints(ctx context.Context, _ ports.LaunchConfig) (ports.PromptReadinessHints, error) {
	if err := ctx.Err(); err != nil {
		return ports.PromptReadinessHints{}, err
	}
	return ports.PromptReadinessHints{
		InitialDelay: 750 * time.Millisecond,
	}, nil
}

// GetAgentHooks installs nothing. The dsh CLI does not yet expose a workspace
// hook contract Kennel can write into, and writing Claude-shaped files that
// nothing reads would only pollute user worktrees. Session metadata capture
// arrives with the real contract.
func (p *Plugin) GetAgentHooks(_ context.Context, _ ports.WorkspaceHookConfig) error {
	return nil
}

// UninstallHooks mirrors the no-op install so teardown stays symmetric.
func (p *Plugin) UninstallHooks(_ context.Context, _ string) error {
	return nil
}

// AreHooksInstalled always reports false because this adapter never writes
// hooks.
func (p *Plugin) AreHooksInstalled(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// GetRestoreCommand reports ok=false until the dsh CLI exposes a verified
// native-resume identity. Even when hook-captured metadata exists, building a
// resume argv with an unverified flag would risk launching a different
// conversation, so Kennel falls back to a fresh launch with standing
// instructions — the documented recovery path.
func (p *Plugin) GetRestoreCommand(ctx context.Context, cfg ports.RestoreConfig) ([]string, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	return nil, false, nil
}

// SessionInfo reads hook-derived metadata under Kennel's normalized keys.
// With hooks not yet installed this reports ok=false, which is truthful:
// no native session id is known.
func (p *Plugin) SessionInfo(ctx context.Context, session ports.SessionRef) (ports.SessionInfo, bool, error) {
	if err := ctx.Err(); err != nil {
		return ports.SessionInfo{}, false, err
	}
	info, ok := agentbase.StandardSessionInfo(session)
	return info, ok, nil
}

// ResolveDSHBinary finds the `dsh` binary.
func ResolveDSHBinary(ctx context.Context) (string, error) {
	return binaryutil.ResolveBinary(ctx, dshBinarySpec)
}

func (p *Plugin) dshBinary(ctx context.Context) (string, error) {
	p.binaryMu.Lock()
	defer p.binaryMu.Unlock()

	if p.resolvedBinary != "" {
		return p.resolvedBinary, nil
	}

	binary, err := ResolveDSHBinary(ctx)
	if err != nil {
		return "", err
	}
	p.resolvedBinary = binary
	return binary, nil
}

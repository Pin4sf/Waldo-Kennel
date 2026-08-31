package kimchi

import (
	"context"
	"path/filepath"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/hooksjson"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

const (
	settingsDirName  = ".kimchi"
	settingsFileName = "hooks.local.json"

	hookCommandPrefix = "kennel hooks kimchi "
	hookTimeout       = 30
)

// startupMatcher is referenced by pointer so SessionStart serializes with
// its required "startup" matcher.
var startupMatcher = "startup"

// managedHooks is the source of truth for the hooks Kennel installs into
// .kimchi/hooks.local.json: SessionStart (under the "startup" matcher),
// UserPromptSubmit, Stop, Notification, SessionEnd, the tool-use trio
// (PreToolUse, PostToolUse, PostToolUseFail), and the post-tool-use-failure
// Kennel sub-command.
//
// PostToolUseFail is Kimchi's native event name, not Claude Code's
// PostToolUseFailure — the wrong name silently fails to fire.
// The Kennel sub-command post-tool-use-failure is used (not post-tool-use-fail)
// to match the lifecycle manager's isToolUseEvent recognition.
var managedHooks = []hooksjson.HookSpec{
	{Event: "SessionStart", Matcher: &startupMatcher, Command: hookCommandPrefix + "session-start"},
	{Event: "UserPromptSubmit", Command: hookCommandPrefix + "user-prompt-submit"},
	{Event: "Stop", Command: hookCommandPrefix + "stop"},
	{Event: "Notification", Command: hookCommandPrefix + "notification"},
	{Event: "SessionEnd", Command: hookCommandPrefix + "session-end"},
	{Event: "PreToolUse", Command: hookCommandPrefix + "pre-tool-use"},
	{Event: "PostToolUse", Command: hookCommandPrefix + "post-tool-use"},
	{Event: "PostToolUseFail", Command: hookCommandPrefix + "post-tool-use-failure"},
}

// kimchiHooks manages Kennel's hooks in the workspace-local
// .kimchi/hooks.local.json file.
var kimchiHooks = hooksjson.Manager{
	Label:         "kimchi",
	CommandPrefix: hookCommandPrefix,
	Timeout:       hookTimeout,
	Path:          settingsPath,
	Managed:       managedHooks,
}

func settingsPath(workspacePath string) string {
	return filepath.Join(workspacePath, settingsDirName, settingsFileName)
}

// GetAgentHooks installs Kennel's Kimchi hooks, preserving user-defined hooks and
// unrelated settings.
func (p *Plugin) GetAgentHooks(ctx context.Context, cfg ports.WorkspaceHookConfig) error {
	return kimchiHooks.Install(ctx, cfg.WorkspacePath)
}

// UninstallHooks removes Kennel's Kimchi hooks, leaving user-defined hooks untouched.
func (p *Plugin) UninstallHooks(ctx context.Context, workspacePath string) error {
	return kimchiHooks.Uninstall(ctx, workspacePath)
}

// AreHooksInstalled reports whether any Kennel Kimchi hook is present.
func (p *Plugin) AreHooksInstalled(ctx context.Context, workspacePath string) (bool, error) {
	return kimchiHooks.AreInstalled(ctx, workspacePath)
}

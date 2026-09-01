package devin

import (
	"context"
	"path/filepath"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/hooksjson"
)

const (
	devinConfigDirName     = ".devin"
	devinConfigFileName    = "config.local.json"
	devinHookCommandPrefix = "kennel hooks devin "
	devinHookTimeout       = 30
)

var devinManagedHooks = []hooksjson.HookSpec{
	{Event: "SessionStart", Command: devinHookCommandPrefix + "session-start"},
	{Event: "UserPromptSubmit", Command: devinHookCommandPrefix + "user-prompt-submit"},
	{Event: "Stop", Command: devinHookCommandPrefix + "stop"},
	{Event: "SessionEnd", Command: devinHookCommandPrefix + "session-end"},
}

var devinHooks = hooksjson.Manager{
	Label:         "devin",
	CommandPrefix: devinHookCommandPrefix,
	Timeout:       devinHookTimeout,
	Path:          devinConfigPath,
	Managed:       devinManagedHooks,
}

func devinConfigPath(workspacePath string) string {
	return filepath.Join(workspacePath, devinConfigDirName, devinConfigFileName)
}

// UninstallHooks removes Kennel's Devin hooks, leaving user-defined hooks untouched.
func (p *Plugin) UninstallHooks(ctx context.Context, workspacePath string) error {
	return devinHooks.Uninstall(ctx, workspacePath)
}

// AreHooksInstalled reports whether any Kennel Devin hook is present.
func (p *Plugin) AreHooksInstalled(ctx context.Context, workspacePath string) (bool, error) {
	return devinHooks.AreInstalled(ctx, workspacePath)
}

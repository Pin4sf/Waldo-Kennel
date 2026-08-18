package telemetrymeta

import "strings"

// NormalizeCommandPath canonicalizes command paths received from current CLIs
// and best-effort legacy loopback callers before cost-control classification.
func NormalizeCommandPath(commandPath string) string {
	return strings.ToLower(strings.Join(strings.Fields(commandPath), " "))
}

// IsRoutineInternalCLICommand reports whether a successful CLI invocation is
// routine desktop/agent plumbing rather than product usage.
func IsRoutineInternalCLICommand(commandPath string) bool {
	normalized := NormalizeCommandPath(commandPath)
	for _, routine := range routineInternalCLICommands {
		if normalized == routine || strings.HasPrefix(normalized, routine+" ") {
			return true
		}
	}
	return false
}

var routineInternalCLICommands = []string{
	"kennel status",
	"kennel session ls",
	"kennel session get",
	"kennel session agent-switch ls",
	"kennel session handoff",
	"kennel project ls",
	"kennel project get",
	"kennel orchestrator ls",
	"kennel hooks",
	"kennel pty-host",
}

// CLIActorType infers the actor for legacy loopback CLI telemetry requests that
// predate the explicit actor_type field. Unknown actor-less commands are treated
// as system activity so foreign/local automation cannot inflate DAU by default.
func CLIActorType(actorType, commandPath string) string {
	normalized := NormalizeCommandPath(commandPath)
	if _, ok := legacyActorlessSystemCLICommands[normalized]; ok {
		return "system"
	}

	switch actorType {
	case "agent", "user":
		return actorType
	case "system":
		return "system"
	}

	if _, ok := legacyActorlessUserCLICommands[normalized]; ok {
		return "user"
	}
	switch normalized {
	case "kennel session agent-switch", "kennel session agent-switch ls", "kennel session switch-agent":
		return "user"
	}
	if normalized == "kennel hooks" {
		return "agent"
	}
	return "system"
}

var legacyActorlessSystemCLICommands = map[string]struct{}{
	"kennel agent-process":           {},
	"kennel agent-process supervise": {},
	"kennel completion":              {},
	"kennel daemon":                  {},
	"kennel help":                    {},
	"kennel pty-host":                {},
	"kennel start":                   {},
}

var legacyActorlessUserCLICommands = map[string]struct{}{
	"kennel agent":                  {},
	"kennel agent ls":               {},
	"kennel browser":                {},
	"kennel browser check":          {},
	"kennel browser click":          {},
	"kennel browser console":        {},
	"kennel browser dblclick":       {},
	"kennel browser devtools":       {},
	"kennel browser devtools close": {},
	"kennel browser devtools open":  {},
	"kennel browser dialog":         {},
	"kennel browser dialog accept":  {},
	"kennel browser dialog dismiss": {},
	"kennel browser dialog status":  {},
	"kennel browser drag":           {},
	"kennel browser errors":         {},
	"kennel browser fill":           {},
	"kennel browser focus":          {},
	"kennel browser frame":          {},
	"kennel browser get":            {},
	"kennel browser highlight":      {},
	"kennel browser hover":          {},
	"kennel browser network":        {},
	"kennel browser network clear":  {},
	"kennel browser network list":   {},
	"kennel browser network start":  {},
	"kennel browser network status": {},
	"kennel browser network stop":   {},
	"kennel browser open":           {},
	"kennel browser press":          {},
	"kennel browser screenshot":     {},
	"kennel browser scroll":         {},
	"kennel browser scrollintoview": {},
	"kennel browser select":         {},
	"kennel browser snapshot":       {},
	"kennel browser tab":            {},
	"kennel browser tab close":      {},
	"kennel browser tab new":        {},
	"kennel browser tab select":     {},
	"kennel browser status":         {},
	"kennel browser tabs":           {},
	"kennel browser type":           {},
	"kennel browser uncheck":        {},
	"kennel browser unhighlight":    {},
	"kennel browser wait":           {},
	"kennel dev":                    {},
	"kennel dev import-projects":    {},
	"kennel doctor":                 {},
	"kennel import":                 {},
	"kennel launch":                 {},
	"kennel orchestrator":           {},
	"kennel orchestrator done":      {},
	"kennel pr":                     {},
	"kennel pr merge":               {},
	"kennel pr resolve-comments":    {},
	"kennel preview":                {},
	"kennel preview clear":          {},
	"kennel preview start":          {},
	"kennel preview status":         {},
	"kennel preview stop":           {},
	"kennel project":                {},
	"kennel project add":            {},
	"kennel project rm":             {},
	"kennel project set-config":     {},
	"kennel review":                 {},
	"kennel review cancel":          {},
	"kennel review ls":              {},
	"kennel review submit":          {},
	"kennel review trigger":         {},
	"kennel send":                   {},
	"kennel session":                {},
	"kennel session claim-pr":       {},
	"kennel session cleanup":        {},
	"kennel session kill":           {},
	"kennel session rename":         {},
	"kennel session restore":        {},
	"kennel spawn":                  {},
	"kennel stop":                   {},
	"kennel version":                {},

	// Legacy commands observed in PostHog's current billing-period data.
	"kennel handoff":                   {},
	"kennel project orchestration get": {},
	"kennel project orchestration set": {},
	"kennel smoke list":                {},
	"kennel smoke set":                 {},
}

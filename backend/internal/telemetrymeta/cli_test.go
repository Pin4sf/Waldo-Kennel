package telemetrymeta

import "testing"

func TestNormalizeCommandPath(t *testing.T) {
	if got := NormalizeCommandPath("  KENNEL   Hooks  claude-code  post-tool-use "); got != "kennel hooks claude-code post-tool-use" {
		t.Fatalf("NormalizeCommandPath = %q, want normalized lowercase fields", got)
	}
}

func TestIsRoutineInternalCLICommandNormalizesLegacyShapes(t *testing.T) {
	for _, commandPath := range []string{
		"kennel hooks",
		"kennel  hooks",
		"KENNEL HOOKS",
		"kennel hooks claude-code post-tool-use",
		"kennel session get sess-123",
		"kennel session agent-switch ls sess-123",
		"kennel session handoff submit --switch switch-1",
		"kennel project ls",
		"kennel pty-host session-1",
	} {
		if !IsRoutineInternalCLICommand(commandPath) {
			t.Errorf("IsRoutineInternalCLICommand(%q) = false, want true", commandPath)
		}
	}
}

func TestCLIActorTypeKeepsKnownLegacyUserCommands(t *testing.T) {
	for _, commandPath := range []string{
		"kennel agent ls",
		"kennel session claim-pr",
		"kennel session switch-agent",
		"kennel session agent-switch",
		"kennel session agent-switch ls",
		"kennel dev import-projects",
		"kennel project orchestration get",
		"kennel project orchestration set",
		"kennel handoff",
		"kennel smoke list",
		"kennel smoke set",
	} {
		if got := CLIActorType("", commandPath); got != "user" {
			t.Errorf("CLIActorType(%q) = %q, want user", commandPath, got)
		}
	}
}

func TestCLIActorTypeTreatsInternalAgentHandoffAsSystemByDefault(t *testing.T) {
	for _, commandPath := range []string{
		"kennel session handoff",
		"kennel session handoff submit",
	} {
		if got := CLIActorType("", commandPath); got != "system" {
			t.Errorf("CLIActorType(%q) = %q, want system", commandPath, got)
		}
	}
}

func TestCLIActorTypeSystemCommandsOverrideExplicitActor(t *testing.T) {
	for _, tc := range []struct {
		actorType   string
		commandPath string
	}{
		{actorType: "user", commandPath: "kennel daemon"},
		{actorType: "agent", commandPath: "kennel start"},
		{actorType: "user", commandPath: "KENNEL  AGENT-PROCESS  SUPERVISE"},
	} {
		if got := CLIActorType(tc.actorType, tc.commandPath); got != "system" {
			t.Errorf("CLIActorType(%q, %q) = %q, want system", tc.actorType, tc.commandPath, got)
		}
	}
}

func TestCLIActorTypeKeepsConservativeFallback(t *testing.T) {
	for _, tc := range []struct {
		actorType   string
		commandPath string
		want        string
	}{
		{actorType: "agent", commandPath: "kennel surprise", want: "agent"},
		{actorType: "user", commandPath: "kennel surprise", want: "user"},
		{actorType: "system", commandPath: "kennel spawn", want: "system"},
		{commandPath: "kennel daemon", want: "system"},
		{commandPath: "kennel spawn", want: "user"},
		{commandPath: "kennel surprise", want: "system"},
	} {
		if got := CLIActorType(tc.actorType, tc.commandPath); got != tc.want {
			t.Errorf("CLIActorType(%q, %q) = %q, want %q", tc.actorType, tc.commandPath, got, tc.want)
		}
	}
}

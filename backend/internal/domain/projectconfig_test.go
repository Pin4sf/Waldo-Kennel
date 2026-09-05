package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProjectConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ProjectConfig
		wantErr bool
	}{
		{"empty ok", ProjectConfig{}, false},
		{"good agent config", ProjectConfig{AgentConfig: AgentConfig{Model: "m", Permissions: PermissionModeAuto}}, false},
		{"good agent mode", ProjectConfig{AgentConfig: AgentConfig{Mode: "ultra"}}, false},
		{"bad agent mode", ProjectConfig{AgentConfig: AgentConfig{Mode: "turbo"}}, true},
		{"good agent profile", ProjectConfig{AgentConfig: AgentConfig{Profile: "waldo-profile"}}, false},
		{"agent profile with whitespace", ProjectConfig{AgentConfig: AgentConfig{Profile: " waldo "}}, true},
		{"blank agent profile", ProjectConfig{AgentConfig: AgentConfig{Profile: "   "}}, true},
		{"agent profile with carriage return", ProjectConfig{AgentConfig: AgentConfig{Profile: "waldo\r"}}, true},
		{"agent profile with NBSP", ProjectConfig{AgentConfig: AgentConfig{Profile: "waldo\u00A0profile"}}, true},
		{"bad permission", ProjectConfig{AgentConfig: AgentConfig{Permissions: "yolo"}}, true},
		{"good session prefix", ProjectConfig{SessionPrefix: "ao"}, false},
		{"session prefix with slash", ProjectConfig{SessionPrefix: "ao/project"}, true},
		{"session prefix with backslash", ProjectConfig{SessionPrefix: `ao\project`}, true},
		{"session prefix traversal component", ProjectConfig{SessionPrefix: ".."}, true},
		{"good role override", ProjectConfig{Worker: RoleOverride{Harness: HarnessCodex}}, false},
		{"shipped role override is selectable", ProjectConfig{Worker: RoleOverride{Harness: HarnessClaudeCode}}, false},
		{"retired role override is not selectable for new work", ProjectConfig{Worker: RoleOverride{Harness: AgentHarness("aider")}}, true},
		{"unknown role harness", ProjectConfig{Orchestrator: RoleOverride{Harness: "nope"}}, true},
		{"bad role agent config", ProjectConfig{Worker: RoleOverride{AgentConfig: AgentConfig{Permissions: "nope"}}}, true},
		{"good symlinks", ProjectConfig{Symlinks: []string{".env", "configs/dev.toml"}}, false},
		{"symlink absolute path", ProjectConfig{Symlinks: []string{"/etc/passwd"}}, true},
		{"symlink parent escape", ProjectConfig{Symlinks: []string{"../escape"}}, true},
		{"symlink embedded parent", ProjectConfig{Symlinks: []string{"a/../../b"}}, true},
		{"symlink bare ..", ProjectConfig{Symlinks: []string{".."}}, true},
		{"good prompt rules", ProjectConfig{AgentRules: "Run tests.", AgentRulesFile: "docs/agent-rules.md", OrchestratorRules: "Delegate work."}, false},
		{"agent rules file absolute path", ProjectConfig{AgentRulesFile: "/etc/passwd"}, true},
		{"agent rules file parent escape", ProjectConfig{AgentRulesFile: "../rules.md"}, true},
		{"agent rules file cleans to dot", ProjectConfig{AgentRulesFile: "docs/.."}, true},
		{"agent rules file bare dot", ProjectConfig{AgentRulesFile: "."}, true},
		{"good codex reviewer", ProjectConfig{Reviewers: []ReviewerConfig{{Harness: ReviewerCodex}}}, false},
		// Every shipped provider has a reviewer adapter and is admitted as one.
		{"claude-code reviewer is admitted", ProjectConfig{Reviewers: []ReviewerConfig{{Harness: ReviewerClaudeCode}}}, false},
		{"opencode reviewer is admitted", ProjectConfig{Reviewers: []ReviewerConfig{{Harness: ReviewerOpenCode}}}, false},
		{"cursor reviewer is admitted", ProjectConfig{Reviewers: []ReviewerConfig{{Harness: ReviewerCursor}}}, false},
		{"pi reviewer is admitted", ProjectConfig{Reviewers: []ReviewerConfig{{Harness: ReviewerPi}}}, false},
		{"retired reviewer identity is refused", ProjectConfig{Reviewers: []ReviewerConfig{{Harness: ReviewerHarness("muse")}}}, true},
		{"unknown reviewer harness", ProjectConfig{Reviewers: []ReviewerConfig{{Harness: "nope"}}}, true},
		{"empty reviewer harness", ProjectConfig{Reviewers: []ReviewerConfig{{Harness: ""}}}, true},
		{"tracker intake assignee rule", ProjectConfig{TrackerIntake: TrackerIntakeConfig{Enabled: true, Assignee: "alice"}}, false},
		{"tracker intake explicit github", ProjectConfig{TrackerIntake: TrackerIntakeConfig{Enabled: true, Provider: TrackerProviderGitHub, Assignee: "alice"}}, false},
		{"tracker intake no rule", ProjectConfig{TrackerIntake: TrackerIntakeConfig{Enabled: true}}, true},
		{"tracker intake unknown provider", ProjectConfig{TrackerIntake: TrackerIntakeConfig{Enabled: true, Provider: "linear", Assignee: "alice"}}, true},
		{"tracker intake repo with whitespace", ProjectConfig{TrackerIntake: TrackerIntakeConfig{Enabled: true, Repo: " acme/demo", Assignee: "alice"}}, true},
		{"tracker intake assignee with whitespace", ProjectConfig{TrackerIntake: TrackerIntakeConfig{Enabled: true, Assignee: " alice"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultProjectConfig(t *testing.T) {
	def := DefaultProjectConfig()

	// The one documented non-empty default.
	if def.DefaultBranch != DefaultBranchAuto {
		t.Fatalf("default DefaultBranch = %q, want %q", def.DefaultBranch, DefaultBranchAuto)
	}

	// Every other field defaults to its zero value: clearing the documented
	// default must leave the config completely empty.
	def.DefaultBranch = ""
	if !def.IsZero() {
		t.Fatalf("default config has unexpected non-zero fields: %#v", def)
	}
}

func TestProjectConfigWithDefaults(t *testing.T) {
	// An unset config gets the documented defaults.
	got := (ProjectConfig{}).WithDefaults()
	if got.DefaultBranch != DefaultBranchAuto {
		t.Fatalf("WithDefaults = %#v, want branch=%s", got, DefaultBranchAuto)
	}

	// Set fields are preserved, not overwritten.
	got = (ProjectConfig{
		DefaultBranch: "develop",
		AgentConfig:   AgentConfig{Model: "m"},
	}).WithDefaults()
	if got.DefaultBranch != "develop" {
		t.Fatalf("WithDefaults overwrote set fields: %#v", got)
	}
	if got.AgentConfig.Model != "m" {
		t.Fatalf("WithDefaults dropped a set field: %#v", got.AgentConfig)
	}
	if got.WorktreeBaseBranch() != "develop" {
		t.Fatalf("WorktreeBaseBranch = %q, want develop", got.WorktreeBaseBranch())
	}
	if got := (ProjectConfig{}).WorktreeBaseBranch(); got != "" {
		t.Fatalf("automatic WorktreeBaseBranch = %q, want empty for adapter inference", got)
	}
	if got := (ProjectConfig{DefaultBranch: DefaultBranchAuto}).WorktreeBaseBranch(); got != "" {
		t.Fatalf("explicit auto WorktreeBaseBranch = %q, want empty for adapter inference", got)
	}

	got = (ProjectConfig{TrackerIntake: TrackerIntakeConfig{Enabled: true, Assignee: "alice"}}).WithDefaults()
	if got.TrackerIntake.Provider != TrackerProviderGitHub {
		t.Fatalf("TrackerIntake.Provider = %q, want %q", got.TrackerIntake.Provider, TrackerProviderGitHub)
	}

	got = (ProjectConfig{}).WithDefaults()
	if got.TrackerIntake.Provider != "" {
		t.Fatalf("disabled TrackerIntake.Provider = %q, want empty", got.TrackerIntake.Provider)
	}
}

func TestResolveReviewerHarness(t *testing.T) {
	// A configured reviewer always wins, regardless of the worker harness.
	cfg := ProjectConfig{Reviewers: []ReviewerConfig{{Harness: ReviewerClaudeCode}}}
	if got := cfg.ResolveReviewerHarness(HarnessCursor); got != ReviewerClaudeCode {
		t.Fatalf("configured reviewer = %q, want claude-code", got)
	}

	// No reviewer configured: every SHIPPED worker reviews with its own provider,
	// so resolution never routes work from one live brand onto another.
	for _, tc := range []struct {
		worker AgentHarness
		want   ReviewerHarness
	}{
		{HarnessClaudeCode, ReviewerClaudeCode},
		{HarnessCodex, ReviewerCodex},
		{HarnessOpenCode, ReviewerOpenCode},
		{HarnessCursor, ReviewerCursor},
		{HarnessPi, ReviewerPi},
	} {
		if got := (ProjectConfig{}).ResolveReviewerHarness(tc.worker); got != tc.want {
			t.Errorf("%s worker = %q, want reviewer %q", tc.worker, got, tc.want)
		}
	}

	// The fallback is now reachable only for a persisted worker identity this
	// build no longer ships — not as routing between two live providers.
	if got := (ProjectConfig{}).ResolveReviewerHarness(AgentHarness("crush")); got != FallbackReviewerHarness {
		t.Fatalf("retired worker identity = %q, want %q", got, FallbackReviewerHarness)
	}
}

func TestProjectConfigIsZero(t *testing.T) {
	if !(ProjectConfig{}).IsZero() {
		t.Fatal("empty config should be zero")
	}
	if (ProjectConfig{DefaultBranch: "main"}).IsZero() {
		t.Fatal("populated config should not be zero")
	}
	if (ProjectConfig{Env: map[string]string{"A": "b"}}).IsZero() {
		t.Fatal("config with env should not be zero")
	}
}

func TestProjectAgentPreferencesValidateRejectsUnsupportedRoles(t *testing.T) {
	cases := []struct {
		name    string
		prefs   ProjectAgentPreferences
		wantErr string
	}{
		{
			name:    "deepseek cannot be preferred coordinator yet",
			prefs:   ProjectAgentPreferences{Coordinator: "deepseek-harness"},
			wantErr: "coordinator",
		},
		{
			name:    "cursor ships but is not admitted as a coordinator",
			prefs:   ProjectAgentPreferences{Coordinator: "cursor"},
			wantErr: "coordinator",
		},
		{
			name:    "unknown harness names are rejected",
			prefs:   ProjectAgentPreferences{Verifier: "not-a-harness"},
			wantErr: "verifier",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.prefs.Validate()
			if err == nil {
				t.Fatalf("want error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not mention %q", err, tc.wantErr)
			}
		})
	}
}

func TestProjectAgentPreferencesValidateAcceptsEligibleCombinations(t *testing.T) {
	prefs := ProjectAgentPreferences{
		DefaultWorker: "opencode",
		Coordinator:   "codex",
		Verifier:      "codex",
	}
	if err := prefs.Validate(); err != nil {
		t.Fatalf("eligible preferences must validate, got %v", err)
	}
	if err := (ProjectAgentPreferences{}).Validate(); err != nil {
		t.Fatalf("empty preferences mean no preference and must validate, got %v", err)
	}
}

func TestResolveMissionRolesDistinguishesPreferenceFromDefault(t *testing.T) {
	roles := ResolveMissionRoles(ProjectConfig{AgentPreferences: ProjectAgentPreferences{DefaultWorker: "opencode"}})
	if roles.Worker.Harness != HarnessOpenCode || roles.Worker.Source != RoleSourcePreference {
		t.Fatalf("worker should honor the preference: %+v", roles.Worker)
	}
	if roles.Coordinator.Harness != HarnessCodex || roles.Coordinator.Source != RoleSourceDefault {
		t.Fatalf("unset roles fall back to the recommended default: %+v", roles.Coordinator)
	}

	defaults := ResolveMissionRoles(ProjectConfig{AgentPreferences: ProjectAgentPreferences{}})
	for _, role := range []ResolvedAgentRole{defaults.Analyzer, defaults.Coordinator, defaults.Worker, defaults.Verifier} {
		if role.Harness != HarnessCodex || role.Source != RoleSourceDefault {
			t.Fatalf("empty preferences resolve every role to codex-by-default: %+v", role)
		}
	}
}

func TestProjectConfigRoundTripsAgentPreferences(t *testing.T) {
	cfg := ProjectConfig{}
	cfg.AgentPreferences = ProjectAgentPreferences{DefaultWorker: "opencode"}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back ProjectConfig
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.AgentPreferences.DefaultWorker != string(HarnessOpenCode) {
		t.Fatalf("preferences lost across JSON round-trip: %+v", back.AgentPreferences)
	}
	if err := back.Validate(); err != nil {
		t.Fatalf("round-tripped config must validate: %v", err)
	}
}

// The project selects its worker and orchestrator harness in Settings and in
// the Outcome composer. A Mission role with no separate preference recorded
// has to follow that selection: before this, a project that displayed
// "opencode" everywhere still spawned Codex for analysis and decomposition,
// because those roles read only agentPreferences and fell back to a constant.
func TestMissionRolesFollowTheHarnessTheProjectSelected(t *testing.T) {
	cfg := ProjectConfig{
		Worker:       RoleOverride{Harness: HarnessOpenCode},
		Orchestrator: RoleOverride{Harness: HarnessOpenCode},
	}
	roles := ResolveMissionRoles(cfg)
	if roles.Worker.Harness != HarnessOpenCode {
		t.Errorf("worker = %q, want the selected harness", roles.Worker.Harness)
	}
	for name, role := range map[string]ResolvedAgentRole{
		"analyzer": roles.Analyzer, "coordinator": roles.Coordinator, "verifier": roles.Verifier,
	} {
		if role.Harness != HarnessOpenCode {
			t.Errorf("%s = %q, want the selected orchestrator harness", name, role.Harness)
		}
	}
}

// An explicit Mission-role preference is still the stronger statement: it is
// set for that role specifically, while the selection covers a whole class.
func TestAnExplicitRolePreferenceOutranksTheSelection(t *testing.T) {
	roles := ResolveMissionRoles(ProjectConfig{
		AgentPreferences: ProjectAgentPreferences{Analyzer: string(HarnessCodex)},
		Orchestrator:     RoleOverride{Harness: HarnessOpenCode},
	})
	if roles.Analyzer.Harness != HarnessCodex {
		t.Errorf("analyzer = %q, want the explicit preference", roles.Analyzer.Harness)
	}
	// The roles that carry no preference of their own still follow selection.
	if roles.Coordinator.Harness != HarnessOpenCode {
		t.Errorf("coordinator = %q, want the selected harness", roles.Coordinator.Harness)
	}
}

// A selection the role cannot admit is not silently honored; it falls through
// to the capability default rather than proposing something spawn would refuse.
func TestAnIneligibleSelectionFallsThroughToTheDefault(t *testing.T) {
	roles := ResolveMissionRoles(ProjectConfig{
		Orchestrator: RoleOverride{Harness: AgentHarness("deepseek-harness")},
	})
	if roles.Analyzer.Harness != HarnessCodex {
		t.Errorf("analyzer = %q, want the capability default for an ineligible selection", roles.Analyzer.Harness)
	}
}

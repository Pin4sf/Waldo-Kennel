package domain

import "testing"

func TestProjectConfigValidate(t *testing.T) {
	for _, harness := range AllHarnesses {
		t.Run("worker_"+string(harness), func(t *testing.T) {
			cfg := ProjectConfig{Worker: RoleOverride{Harness: harness}}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("active worker provider %q should validate: %v", harness, err)
			}
		})
		t.Run("orchestrator_"+string(harness), func(t *testing.T) {
			cfg := ProjectConfig{Orchestrator: RoleOverride{Harness: harness}}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("active orchestrator provider %q should validate: %v", harness, err)
			}
		})
	}

	for _, reviewer := range AllReviewerHarnesses {
		t.Run("reviewer_"+string(reviewer), func(t *testing.T) {
			cfg := ProjectConfig{Reviewers: []ReviewerConfig{{Harness: reviewer}}}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("active reviewer provider %q should validate: %v", reviewer, err)
			}
		})
	}

	tests := []struct {
		name    string
		cfg     ProjectConfig
		wantErr bool
	}{
		{"empty ok", ProjectConfig{}, false},
		{"good agent config", ProjectConfig{AgentConfig: AgentConfig{Model: "m", Permissions: PermissionModeAuto}}, false},
		{"bad permission", ProjectConfig{AgentConfig: AgentConfig{Permissions: "yolo"}}, true},
		{"unknown role harness", ProjectConfig{Orchestrator: RoleOverride{Harness: "nope"}}, true},
		{"unknown reviewer harness", ProjectConfig{Reviewers: []ReviewerConfig{{Harness: "nope"}}}, true},
		{"empty reviewer harness", ProjectConfig{Reviewers: []ReviewerConfig{{Harness: ""}}}, true},
		{"good session prefix", ProjectConfig{SessionPrefix: "kennel"}, false},
		{"session prefix traversal", ProjectConfig{SessionPrefix: ".."}, true},
		{"good symlink", ProjectConfig{Symlinks: []string{"configs/dev.toml"}}, false},
		{"symlink escape", ProjectConfig{Symlinks: []string{"../escape"}}, true},
		{"good rules file", ProjectConfig{AgentRulesFile: "docs/agent-rules.md"}, false},
		{"rules file escape", ProjectConfig{AgentRulesFile: "../rules.md"}, true},
		{"tracker intake assignee", ProjectConfig{TrackerIntake: TrackerIntakeConfig{Enabled: true, Assignee: "alice"}}, false},
		{"tracker intake no rule", ProjectConfig{TrackerIntake: TrackerIntakeConfig{Enabled: true}}, true},
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
	if def.DefaultBranch != DefaultBranchAuto {
		t.Fatalf("default DefaultBranch = %q, want %q", def.DefaultBranch, DefaultBranchAuto)
	}
	def.DefaultBranch = ""
	if !def.IsZero() {
		t.Fatalf("default config has unexpected non-zero fields: %#v", def)
	}
}

func TestProjectConfigWithDefaults(t *testing.T) {
	got := (ProjectConfig{}).WithDefaults()
	if got.DefaultBranch != DefaultBranchAuto {
		t.Fatalf("WithDefaults = %#v, want branch=%s", got, DefaultBranchAuto)
	}
	got = (ProjectConfig{DefaultBranch: "develop", AgentConfig: AgentConfig{Model: "m"}}).WithDefaults()
	if got.DefaultBranch != "develop" || got.AgentConfig.Model != "m" {
		t.Fatalf("WithDefaults overwrote set fields: %#v", got)
	}
	if got.WorktreeBaseBranch() != "develop" {
		t.Fatalf("WorktreeBaseBranch = %q, want develop", got.WorktreeBaseBranch())
	}
	if got := (ProjectConfig{}).WorktreeBaseBranch(); got != "" {
		t.Fatalf("automatic WorktreeBaseBranch = %q, want empty", got)
	}
}

func TestResolveReviewerHarness(t *testing.T) {
	configured := ProjectConfig{Reviewers: []ReviewerConfig{{Harness: ReviewerClaudeCode}}}
	if got := configured.ResolveReviewerHarness(HarnessPi); got != ReviewerClaudeCode {
		t.Fatalf("configured reviewer = %q, want claude-code", got)
	}

	cases := map[AgentHarness]ReviewerHarness{
		HarnessCodex:      ReviewerCodex,
		HarnessClaudeCode: ReviewerClaudeCode,
		HarnessOpenCode:   ReviewerOpenCode,
		HarnessCursor:     ReviewerCursor,
		HarnessPi:         ReviewerPi,
	}
	for worker, want := range cases {
		if got := (ProjectConfig{}).ResolveReviewerHarness(worker); got != want {
			t.Errorf("worker %q reviewer = %q, want %q", worker, got, want)
		}
	}
	if got := (ProjectConfig{}).ResolveReviewerHarness("unknown"); got != "" {
		t.Fatalf("unknown worker reviewer = %q, want empty", got)
	}
}

func TestProjectConfigIsZero(t *testing.T) {
	if !(ProjectConfig{}).IsZero() {
		t.Fatal("empty config should be zero")
	}
	if (ProjectConfig{DefaultBranch: "main"}).IsZero() {
		t.Fatal("populated config should not be zero")
	}
}

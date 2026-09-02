package domain

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
)

// ProjectConfig is Kennel's typed per-project configuration. It is persisted as
// one JSON blob per project and resolved when work starts; there is no free-form
// provider policy map in the renderer.
type ProjectConfig struct {
	// DefaultBranch is the base branch new session worktrees are created from.
	// Empty and DefaultBranchAuto both mean infer each repository's Git default.
	DefaultBranch string `json:"defaultBranch,omitempty"`
	// SessionPrefix overrides the displayed session-id prefix.
	SessionPrefix string `json:"sessionPrefix,omitempty"`

	// Env are extra environment variables forwarded into managed provider
	// runtimes. Kennel-owned variables always win.
	Env map[string]string `json:"env,omitempty"`
	// Symlinks are repo-relative paths symlinked into each session workspace.
	Symlinks []string `json:"symlinks,omitempty"`
	// PostCreate are shell commands run in the workspace after it is created.
	PostCreate []string `json:"postCreate,omitempty"`

	AgentRules       string `json:"agentRules,omitempty"`
	AgentRulesFile   string `json:"agentRulesFile,omitempty"`
	OrchestratorRules string `json:"orchestratorRules,omitempty"`

	// AgentConfig is the shared default agent config for the project.
	AgentConfig AgentConfig `json:"agentConfig,omitempty"`
	// Worker and Orchestrator are explicit role-specific provider/config choices.
	// An empty Harness means the role has not been selected yet; it never means
	// "use Codex".
	Worker       RoleOverride `json:"worker,omitempty"`
	Orchestrator RoleOverride `json:"orchestrator,omitempty"`

	// Reviewers names the provider(s) used for explicit code review. When unset,
	// Kennel reuses the worker provider only when that provider has a matching
	// reviewer adapter; all five shipped providers currently do.
	Reviewers []ReviewerConfig `json:"reviewers,omitempty"`

	TrackerIntake TrackerIntakeConfig `json:"trackerIntake,omitempty"`
	ContainerReap ContainerReapConfig `json:"containerReap,omitempty"`
}

type ContainerReapConfig struct {
	Disabled bool `json:"disabled,omitempty"`
}

type ReviewerConfig struct {
	Harness ReviewerHarness `json:"harness"`
}

// ResolveReviewerHarness picks the explicit reviewer when configured; otherwise
// it reuses the active worker provider. There is deliberately no cross-provider
// fallback because Kennel must not silently substitute one provider for another.
func (c ProjectConfig) ResolveReviewerHarness(worker AgentHarness) ReviewerHarness {
	if len(c.Reviewers) > 0 {
		return c.Reviewers[0].Harness
	}
	switch worker {
	case HarnessCodex:
		return ReviewerCodex
	case HarnessClaudeCode:
		return ReviewerClaudeCode
	case HarnessOpenCode:
		return ReviewerOpenCode
	case HarnessCursor:
		return ReviewerCursor
	case HarnessPi:
		return ReviewerPi
	default:
		return ""
	}
}

type RoleOverride struct {
	Harness     AgentHarness `json:"agent,omitempty"`
	AgentConfig AgentConfig  `json:"agentConfig,omitempty"`
}

const (
	DefaultBranchAuto = "auto"
	DefaultBranchName = "main"
)

func DefaultProjectConfig() ProjectConfig {
	return ProjectConfig{DefaultBranch: DefaultBranchAuto}
}

func (c ProjectConfig) WithDefaults() ProjectConfig {
	def := DefaultProjectConfig()
	if c.DefaultBranch == "" {
		c.DefaultBranch = def.DefaultBranch
	}
	c.TrackerIntake = c.TrackerIntake.WithDefaults()
	return c
}

func (c ProjectConfig) WorktreeBaseBranch() string {
	branch := c.WithDefaults().DefaultBranch
	if branch == DefaultBranchAuto {
		return ""
	}
	return branch
}

func (c ProjectConfig) IsZero() bool {
	return reflect.DeepEqual(c, ProjectConfig{})
}

// Validate checks durable configuration vocabulary only. Machine-specific
// installation/auth readiness is intentionally checked at admission/spawn time,
// so a project can remain configured when a provider is temporarily unavailable.
func (c ProjectConfig) Validate() error {
	if err := c.AgentConfig.Validate(); err != nil {
		return err
	}
	if err := validateNameComponent("sessionPrefix", c.SessionPrefix); err != nil {
		return err
	}
	for role, override := range map[string]RoleOverride{"worker": c.Worker, "orchestrator": c.Orchestrator} {
		if override.Harness != "" && !override.Harness.IsSelectableForNewWork() {
			return fmt.Errorf("%s.agent: provider %q is not supported by Kennel", role, override.Harness)
		}
		if err := override.AgentConfig.Validate(); err != nil {
			return fmt.Errorf("%s.%w", role, err)
		}
	}
	for _, symlink := range c.Symlinks {
		if err := validateRepoRelative(symlink); err != nil {
			return fmt.Errorf("symlink %q: %w", symlink, err)
		}
	}
	if err := validateRepoRelative(c.AgentRulesFile); err != nil {
		return fmt.Errorf("agentRulesFile %q: %w", c.AgentRulesFile, err)
	}
	for i, reviewer := range c.Reviewers {
		if !reviewer.Harness.IsSelectableForNewWork() {
			return fmt.Errorf("reviewers[%d].harness: provider %q is not supported by Kennel", i, reviewer.Harness)
		}
	}
	if err := c.TrackerIntake.Validate(); err != nil {
		return err
	}
	return nil
}

func validateNoWhitespaceField(name, value string) error {
	if value == "" {
		return nil
	}
	if strings.TrimSpace(value) != value {
		return fmt.Errorf("%s: must not have leading or trailing whitespace", name)
	}
	return nil
}

func validateNameComponent(name, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if strings.ContainsAny(trimmed, `/\`) || trimmed == "." || trimmed == ".." {
		return fmt.Errorf("%s: must not contain path separators or traversal components", name)
	}
	return nil
}

func validateRepoRelative(p string) error {
	trimmed := strings.TrimSpace(p)
	if trimmed == "" {
		return nil
	}
	if filepath.IsAbs(trimmed) || strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, `\`) {
		return fmt.Errorf("path must be repo-relative and must not escape the project root")
	}
	clean := filepath.Clean(trimmed)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path must be repo-relative and must not escape the project root")
	}
	for _, segment := range strings.Split(filepath.ToSlash(clean), "/") {
		if segment == ".." {
			return fmt.Errorf("path must be repo-relative and must not escape the project root")
		}
	}
	return nil
}

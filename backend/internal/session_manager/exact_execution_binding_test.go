package sessionmanager

import (
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

func TestSpawnExecutionConfigProviderDefaultClearsMutableProjectModel(t *testing.T) {
	project := domain.ProjectConfig{
		AgentConfig: domain.AgentConfig{Model: "project-base-model"},
		Worker: domain.RoleOverride{
			Harness:     domain.HarnessCodex,
			AgentConfig: domain.AgentConfig{Model: "project-worker-model", Profile: "project-profile"},
		},
	}
	binding := domain.ExecutionBinding{
		Provider:       domain.HarnessCodex,
		ModelSelection: domain.ExecutionBindingModelProviderDefault,
	}

	harness, config, err := spawnExecutionConfig(ports.SpawnConfig{
		Kind:                  domain.KindWorker,
		Harness:               domain.HarnessCodex,
		AgentConfig:           ports.AgentConfig{Model: "request-model", Permissions: domain.PermissionModeAuto},
		ExactExecutionBinding: &binding,
	}, project)
	if err != nil {
		t.Fatalf("spawnExecutionConfig: %v", err)
	}
	if harness != domain.HarnessCodex {
		t.Fatalf("harness = %q, want %q", harness, domain.HarnessCodex)
	}
	if config.Model != "" {
		t.Fatalf("provider_default inherited concrete model %q", config.Model)
	}
	if config.Profile != "project-profile" {
		t.Fatalf("profile = %q, want provider-local project profile", config.Profile)
	}
	if config.Permissions != domain.PermissionModeAuto {
		t.Fatalf("permissions = %q, want request capability permissions", config.Permissions)
	}
}

func TestSpawnExecutionConfigExplicitBindingOverridesProjectAndRequestModel(t *testing.T) {
	project := domain.ProjectConfig{
		Worker: domain.RoleOverride{
			Harness:     domain.HarnessCodex,
			AgentConfig: domain.AgentConfig{Model: "mutable-project-model"},
		},
	}
	binding := domain.ExecutionBinding{
		Provider:       domain.HarnessClaudeCode,
		ModelSelection: domain.ExecutionBindingModelExplicit,
		Model:          " claude-exact-model ",
	}

	harness, config, err := spawnExecutionConfig(ports.SpawnConfig{
		Kind:                  domain.KindWorker,
		Harness:               domain.HarnessCodex,
		AgentConfig:           ports.AgentConfig{Model: "mutable-request-model"},
		ExactExecutionBinding: &binding,
	}, project)
	if err != nil {
		t.Fatalf("spawnExecutionConfig: %v", err)
	}
	if harness != domain.HarnessClaudeCode {
		t.Fatalf("harness = %q, want exact provider %q", harness, domain.HarnessClaudeCode)
	}
	if config.Model != "claude-exact-model" {
		t.Fatalf("model = %q, want exact model", config.Model)
	}
}

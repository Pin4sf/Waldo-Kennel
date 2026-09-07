package domain

import (
	"strings"
	"testing"
)

func TestExecutionPreferenceValidation(t *testing.T) {
	tests := []struct {
		name    string
		pref    ExecutionPreference
		wantErr string
	}{
		{
			name: "explicit model is valid",
			pref: ExecutionPreference{
				Provider:       HarnessClaudeCode,
				ModelSelection: ExecutionPreferenceModelExplicit,
				Model:          "sonnet",
			},
		},
		{
			name: "provider default is explicit preference semantics",
			pref: ExecutionPreference{
				Provider:       HarnessCodex,
				ModelSelection: ExecutionPreferenceModelProviderDefault,
			},
		},
		{
			name: "model never implies provider",
			pref: ExecutionPreference{
				ModelSelection: ExecutionPreferenceModelExplicit,
				Model:          "model-x",
			},
			wantErr: "execution preference provider is required",
		},
		{
			name: "explicit selection requires a model",
			pref: ExecutionPreference{
				Provider:       HarnessCodex,
				ModelSelection: ExecutionPreferenceModelExplicit,
			},
			wantErr: "explicit execution preference model is required",
		},
		{
			name: "provider default cannot carry a model",
			pref: ExecutionPreference{
				Provider:       HarnessCodex,
				ModelSelection: ExecutionPreferenceModelProviderDefault,
				Model:          "model-x",
			},
			wantErr: "provider-default execution preference must not name a model",
		},
		{
			name: "unknown model selection is rejected",
			pref: ExecutionPreference{
				Provider:       HarnessCodex,
				ModelSelection: ExecutionPreferenceModelSelection("mystery"),
			},
			wantErr: "unsupported execution preference model selection",
		},
		{
			name: "unknown provider is rejected",
			pref: ExecutionPreference{
				Provider:       AgentHarness("other"),
				ModelSelection: ExecutionPreferenceModelProviderDefault,
			},
			wantErr: "unsupported execution preference provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pref.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestResolveEffectiveExecutionPreferenceUsesProjectWorkerWhenOutcomeHasNoOverride(t *testing.T) {
	project := ProjectConfig{
		Worker: RoleOverride{
			Harness: HarnessClaudeCode,
			AgentConfig: AgentConfig{
				Model: "sonnet",
			},
		},
	}

	got, ok, err := ResolveEffectiveExecutionPreference(nil, project)
	if err != nil {
		t.Fatalf("ResolveEffectiveExecutionPreference() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveEffectiveExecutionPreference() ok = false, want true")
	}
	if got.Provider != HarnessClaudeCode || got.ModelSelection != ExecutionPreferenceModelExplicit || got.Model != "sonnet" {
		t.Fatalf("effective preference = %+v, want claude-code/sonnet explicit", got)
	}
}

func TestResolveEffectiveExecutionPreferenceOutcomeProviderDefaultDoesNotLeakProjectModel(t *testing.T) {
	project := ProjectConfig{
		Worker: RoleOverride{
			Harness: HarnessClaudeCode,
			AgentConfig: AgentConfig{
				Model: "sonnet",
			},
		},
	}
	outcome := &ExecutionPreference{
		Provider:       HarnessCodex,
		ModelSelection: ExecutionPreferenceModelProviderDefault,
	}

	got, ok, err := ResolveEffectiveExecutionPreference(outcome, project)
	if err != nil {
		t.Fatalf("ResolveEffectiveExecutionPreference() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveEffectiveExecutionPreference() ok = false, want true")
	}
	if got.Provider != HarnessCodex || got.ModelSelection != ExecutionPreferenceModelProviderDefault || got.Model != "" {
		t.Fatalf("effective preference = %+v, want codex/provider-default with no inherited model", got)
	}
}

func TestResolveEffectiveExecutionPreferenceProjectProviderWithoutModelMeansProviderDefault(t *testing.T) {
	project := ProjectConfig{Worker: RoleOverride{Harness: HarnessOpenCode}}

	got, ok, err := ResolveEffectiveExecutionPreference(nil, project)
	if err != nil {
		t.Fatalf("ResolveEffectiveExecutionPreference() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveEffectiveExecutionPreference() ok = false, want true")
	}
	if got.Provider != HarnessOpenCode || got.ModelSelection != ExecutionPreferenceModelProviderDefault || got.Model != "" {
		t.Fatalf("effective preference = %+v, want opencode/provider-default", got)
	}
}

func TestResolveEffectiveExecutionPreferenceNoOutcomeAndNoProjectPreferenceReturnsNone(t *testing.T) {
	got, ok, err := ResolveEffectiveExecutionPreference(nil, ProjectConfig{})
	if err != nil {
		t.Fatalf("ResolveEffectiveExecutionPreference() error = %v", err)
	}
	if ok {
		t.Fatalf("ResolveEffectiveExecutionPreference() = %+v, true; want zero, false", got)
	}
	if got.Provider != "" {
		t.Fatalf("provider = %q, want empty; no implicit provider is allowed", got.Provider)
	}
}

func TestResolveEffectiveExecutionPreferenceRejectsProjectModelWithoutProvider(t *testing.T) {
	project := ProjectConfig{
		Worker: RoleOverride{AgentConfig: AgentConfig{Model: "orphan-model"}},
	}

	_, ok, err := ResolveEffectiveExecutionPreference(nil, project)
	if err == nil || !strings.Contains(err.Error(), "project worker model requires a provider") {
		t.Fatalf("ResolveEffectiveExecutionPreference() error = %v, want provider-required validation error", err)
	}
	if ok {
		t.Fatal("ResolveEffectiveExecutionPreference() ok = true, want false")
	}
}

func TestContractRevisionValidatesExecutionPreference(t *testing.T) {
	revision := ContractRevision{
		ID:              "cr_01",
		OutcomeID:       "out_01",
		Number:          1,
		Goal:            "Ship the requested change.",
		SuccessCriteria: []string{"The requested behavior is verified."},
		Review:          "Owner review.",
		ExecutionPreference: &ExecutionPreference{
			Provider:       HarnessCodex,
			ModelSelection: ExecutionPreferenceModelExplicit,
			Model:          "",
		},
	}

	err := revision.Validate()
	if err == nil || !strings.Contains(err.Error(), "explicit execution preference model is required") {
		t.Fatalf("ContractRevision.Validate() = %v, want invalid execution preference", err)
	}
}

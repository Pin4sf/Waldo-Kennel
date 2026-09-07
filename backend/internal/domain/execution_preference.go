package domain

import (
	"fmt"
	"strings"
)

// ExecutionPreferenceModelSelection names how a preferred provider should
// select its model. Preference is planning input only; it never authorizes an
// Attempt directly.
type ExecutionPreferenceModelSelection string

const (
	// ExecutionPreferenceModelProviderDefault deliberately delegates model
	// choice to the selected provider. It is explicit semantics, not an omitted
	// field that may inherit a model from another provider.
	ExecutionPreferenceModelProviderDefault ExecutionPreferenceModelSelection = "provider_default"
	// ExecutionPreferenceModelExplicit freezes one concrete model identity as
	// the user's planning preference.
	ExecutionPreferenceModelExplicit ExecutionPreferenceModelSelection = "explicit"
)

// ExecutionPreference is an optional provider/model preference carried by an
// immutable ContractRevision. When present it owns both provider and model
// semantics so a Project model can never leak across an Outcome provider
// override.
type ExecutionPreference struct {
	Provider       AgentHarness                      `json:"provider"`
	ModelSelection ExecutionPreferenceModelSelection `json:"modelSelection"`
	Model          string                            `json:"model,omitempty"`
}

// Validate checks that the preference is complete and refers only to the
// active product provider surface. Runtime installation/auth readiness belongs
// to routing admission, not immutable contract validation.
func (p ExecutionPreference) Validate() error {
	if strings.TrimSpace(string(p.Provider)) == "" {
		return fmt.Errorf("execution preference provider is required")
	}
	if !p.Provider.IsSelectableForNewWork() {
		return fmt.Errorf("unsupported execution preference provider %q", p.Provider)
	}
	switch p.ModelSelection {
	case ExecutionPreferenceModelExplicit:
		if strings.TrimSpace(p.Model) == "" {
			return fmt.Errorf("explicit execution preference model is required")
		}
	case ExecutionPreferenceModelProviderDefault:
		if strings.TrimSpace(p.Model) != "" {
			return fmt.Errorf("provider-default execution preference must not name a model")
		}
	default:
		return fmt.Errorf("unsupported execution preference model selection %q", p.ModelSelection)
	}
	return nil
}

// ResolveEffectiveExecutionPreference resolves immutable Outcome preference
// over the mutable Project worker baseline. Absence at both layers is a real
// no-preference state; Kennel must not manufacture a provider.
func ResolveEffectiveExecutionPreference(outcomePreference *ExecutionPreference, project ProjectConfig) (ExecutionPreference, bool, error) {
	if outcomePreference != nil {
		pref := *outcomePreference
		if err := pref.Validate(); err != nil {
			return ExecutionPreference{}, false, err
		}
		return pref, true, nil
	}

	provider := AgentHarness(strings.TrimSpace(string(project.Worker.Harness)))
	model := strings.TrimSpace(project.Worker.AgentConfig.Model)
	if provider == "" {
		if model != "" {
			return ExecutionPreference{}, false, fmt.Errorf("project worker model requires a provider")
		}
		return ExecutionPreference{}, false, nil
	}

	pref := ExecutionPreference{Provider: provider}
	if model == "" {
		pref.ModelSelection = ExecutionPreferenceModelProviderDefault
	} else {
		pref.ModelSelection = ExecutionPreferenceModelExplicit
		pref.Model = model
	}
	if err := pref.Validate(); err != nil {
		return ExecutionPreference{}, false, fmt.Errorf("project worker execution preference: %w", err)
	}
	return pref, true, nil
}

package domain

import (
	"fmt"
	"strings"
)

// ExecutionBindingModelSelection records the model semantics frozen by an
// approved WorkUnit. It is separate from preference because approval turns a
// recommendation into execution authority.
type ExecutionBindingModelSelection string

const (
	// ExecutionBindingModelProviderDefault delegates model selection to the provider.
	ExecutionBindingModelProviderDefault ExecutionBindingModelSelection = "provider_default"
	// ExecutionBindingModelExplicit freezes a concrete model selection.
	ExecutionBindingModelExplicit ExecutionBindingModelSelection = "explicit"
	// ExecutionBindingModelHistoricalUnbound is read compatibility for plans
	// created before WT3. New plans must never produce it and Attempts must not
	// execute it.
	ExecutionBindingModelHistoricalUnbound ExecutionBindingModelSelection = "historical_unbound"
)

// ExecutionBinding is the exact provider/model authority carried by a WorkUnit.
type ExecutionBinding struct {
	Provider       AgentHarness                   `json:"provider"`
	ModelSelection ExecutionBindingModelSelection `json:"modelSelection"`
	Model          string                         `json:"model,omitempty"`
}

// ValidateReadable admits historical model-unbound rows so old Plans stay
// inspectable while still validating every concrete WT3 binding.
func (b ExecutionBinding) ValidateReadable() error {
	if strings.TrimSpace(string(b.Provider)) == "" {
		return fmt.Errorf("execution binding provider is required")
	}
	if !b.Provider.IsRecognizedPersisted() {
		return fmt.Errorf("unsupported execution binding provider %q", b.Provider)
	}
	switch b.ModelSelection {
	case ExecutionBindingModelExplicit:
		if strings.TrimSpace(b.Model) == "" {
			return fmt.Errorf("explicit execution binding model is required")
		}
	case ExecutionBindingModelProviderDefault:
		if strings.TrimSpace(b.Model) != "" {
			return fmt.Errorf("provider-default execution binding must not name a model")
		}
	case ExecutionBindingModelHistoricalUnbound:
		if strings.TrimSpace(b.Model) != "" {
			return fmt.Errorf("historical-unbound execution binding must not name a model")
		}
	default:
		return fmt.Errorf("unsupported execution binding model selection %q", b.ModelSelection)
	}
	return nil
}

// ValidateForNewWork rejects compatibility-only state and guarantees an
// Attempt can execute the binding without consulting mutable preferences.
func (b ExecutionBinding) ValidateForNewWork() error {
	if err := b.ValidateReadable(); err != nil {
		return err
	}
	if !b.Provider.IsSelectableForNewWork() {
		return fmt.Errorf("execution binding provider %q is not selectable for new work", b.Provider)
	}
	if b.ModelSelection == ExecutionBindingModelHistoricalUnbound {
		return fmt.Errorf("historical-unbound execution binding cannot authorize new work")
	}
	return nil
}

// ExecutionBindingFromPreference turns a provider/model preference into the
// same model semantics used by an approved binding. Routing may choose a
// different candidate; this helper only performs the representation change.
func ExecutionBindingFromPreference(p ExecutionPreference) ExecutionBinding {
	selection := ExecutionBindingModelProviderDefault
	if p.ModelSelection == ExecutionPreferenceModelExplicit {
		selection = ExecutionBindingModelExplicit
	}
	return ExecutionBinding{Provider: p.Provider, ModelSelection: selection, Model: p.Model}
}

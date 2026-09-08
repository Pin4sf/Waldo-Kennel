package domain

import (
	"fmt"
	"strings"
)

// PlanDraftProposal is non-authoritative planning output. It is validated and
// compiled into an immutable PlanRevision before the owner may approve it.
// The MVP permits multiple WorkUnits only when their dependencies can be
// executed truthfully in stable serial order.
type PlanDraftProposal struct {
	Summary     string
	WorkUnits   []PlanDraftWorkUnit
	Assumptions []string
	Blockers    []string
}

// PlanDraftWorkUnit is provider-neutral proposal material. Routing happens
// after the proposal is validated, so a model cannot smuggle execution
// authority into its output by naming a provider or provider-native session.
type PlanDraftWorkUnit struct {
	Key                     string
	Title                   string
	OutputSummary           string
	EvidenceChecks          []string
	VerificationRequirement string
	StopConditions          []string
	DependsOn               []string
	HardCapabilities        []string
	SoftCapabilities        []string
	ExecutionConstraints    []string
}

func (p PlanDraftProposal) Validate() error {
	if strings.TrimSpace(p.Summary) == "" {
		return fmt.Errorf("plan draft summary is required")
	}
	if len(p.WorkUnits) == 0 {
		return fmt.Errorf("plan draft requires at least one work unit")
	}
	if len(p.WorkUnits) > 16 {
		return fmt.Errorf("plan draft has %d work units; MVP limit is 16", len(p.WorkUnits))
	}
	seen := make(map[string]struct{}, len(p.WorkUnits))
	for i, unit := range p.WorkUnits {
		if err := unit.validate(i, seen); err != nil {
			return err
		}
		seen[strings.TrimSpace(unit.Key)] = struct{}{}
	}
	if err := validateNonBlankPlanDraftList("assumption", p.Assumptions); err != nil {
		return err
	}
	if err := validateNonBlankPlanDraftList("blocker", p.Blockers); err != nil {
		return err
	}
	return nil
}

func (w PlanDraftWorkUnit) validate(index int, earlier map[string]struct{}) error {
	key := strings.TrimSpace(w.Key)
	if key == "" {
		return fmt.Errorf("plan draft work unit %d key is required", index+1)
	}
	if _, duplicate := earlier[key]; duplicate {
		return fmt.Errorf("plan draft work unit key %q is duplicated", key)
	}
	if strings.TrimSpace(w.Title) == "" {
		return fmt.Errorf("plan draft work unit %q title is required", key)
	}
	if strings.TrimSpace(w.OutputSummary) == "" {
		return fmt.Errorf("plan draft work unit %q output summary is required", key)
	}
	if len(w.EvidenceChecks) == 0 {
		return fmt.Errorf("plan draft work unit %q requires evidence checks", key)
	}
	if err := validateNonBlankPlanDraftList("evidence check", w.EvidenceChecks); err != nil {
		return fmt.Errorf("plan draft work unit %q: %w", key, err)
	}
	if strings.TrimSpace(w.VerificationRequirement) == "" {
		return fmt.Errorf("plan draft work unit %q verification requirement is required", key)
	}
	if err := validateNonBlankPlanDraftList("stop condition", w.StopConditions); err != nil {
		return fmt.Errorf("plan draft work unit %q: %w", key, err)
	}
	if len(w.HardCapabilities) == 0 {
		return fmt.Errorf("plan draft work unit %q requires at least one hard capability", key)
	}
	if err := validateNonBlankPlanDraftList("hard capability", w.HardCapabilities); err != nil {
		return fmt.Errorf("plan draft work unit %q: %w", key, err)
	}
	if err := validateNonBlankPlanDraftList("soft capability", w.SoftCapabilities); err != nil {
		return fmt.Errorf("plan draft work unit %q: %w", key, err)
	}
	if err := validateNonBlankPlanDraftList("execution constraint", w.ExecutionConstraints); err != nil {
		return fmt.Errorf("plan draft work unit %q: %w", key, err)
	}
	depSeen := make(map[string]struct{}, len(w.DependsOn))
	for _, raw := range w.DependsOn {
		dependency := strings.TrimSpace(raw)
		if dependency == "" {
			return fmt.Errorf("plan draft work unit %q has a blank dependency", key)
		}
		if dependency == key {
			return fmt.Errorf("plan draft work unit %q cannot depend on itself", key)
		}
		if _, duplicate := depSeen[dependency]; duplicate {
			return fmt.Errorf("plan draft work unit %q repeats dependency %q", key, dependency)
		}
		depSeen[dependency] = struct{}{}
		if _, existsEarlier := earlier[dependency]; !existsEarlier {
			return fmt.Errorf("plan draft work unit %q depends on %q before it appears; MVP plans must be serializable in stable order", key, dependency)
		}
	}
	return nil
}

func validateNonBlankPlanDraftList(kind string, values []string) error {
	for i, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s %d is blank", kind, i+1)
		}
	}
	return nil
}

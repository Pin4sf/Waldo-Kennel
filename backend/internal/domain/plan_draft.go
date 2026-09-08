package domain

import (
	"fmt"
	"sort"
	"strings"
)

// MaxPlanDraftWorkUnits is an operational bound on untrusted planning output,
// not a law of Outcomes or the scheduler.
const MaxPlanDraftWorkUnits = 16

// PlanDraftProposal is non-authoritative intelligence output. It describes what
// work probably needs doing; deterministic Kennel compilation derives authority,
// routing requirements, mandatory stops, and verification obligations.
type PlanDraftProposal struct {
	Summary     string
	WorkUnits   []PlanDraftWorkUnit
	Assumptions []string
	Blockers    []string
}

// PlanDraftWorkUnit deliberately contains no provider, model, capability grant,
// stop policy, or canonical verification authority. CriteriaCovered uses stable
// model-facing aliases (for example C1/C2) that the control plane maps to the
// Contract's canonical CriterionIDs.
type PlanDraftWorkUnit struct {
	Key             string
	Title           string
	OutputSummary   string
	CriteriaCovered []string
	DependsOn       []string
	EvidenceIdeas   []string
}

func (p PlanDraftProposal) Validate() error {
	if strings.TrimSpace(p.Summary) == "" {
		return fmt.Errorf("plan draft summary is required")
	}
	if len(p.WorkUnits) == 0 {
		return fmt.Errorf("plan draft requires at least one work unit")
	}
	if len(p.WorkUnits) > MaxPlanDraftWorkUnits {
		return fmt.Errorf("plan draft has %d work units; maximum planning output is %d", len(p.WorkUnits), MaxPlanDraftWorkUnits)
	}

	units := make(map[string]PlanDraftWorkUnit, len(p.WorkUnits))
	for i, unit := range p.WorkUnits {
		key := strings.TrimSpace(unit.Key)
		if key == "" {
			return fmt.Errorf("plan draft work unit %d key is required", i+1)
		}
		if _, duplicate := units[key]; duplicate {
			return fmt.Errorf("plan draft work unit key %q is duplicated", key)
		}
		if strings.TrimSpace(unit.Title) == "" {
			return fmt.Errorf("plan draft work unit %q title is required", key)
		}
		if strings.TrimSpace(unit.OutputSummary) == "" {
			return fmt.Errorf("plan draft work unit %q output summary is required", key)
		}
		if len(unit.CriteriaCovered) == 0 {
			return fmt.Errorf("plan draft work unit %q must cover at least one contract criterion", key)
		}
		if err := validateUniqueNonBlankPlanDraftList("criterion alias", unit.CriteriaCovered); err != nil {
			return fmt.Errorf("plan draft work unit %q: %w", key, err)
		}
		if err := validateUniqueNonBlankPlanDraftList("evidence idea", unit.EvidenceIdeas); err != nil {
			return fmt.Errorf("plan draft work unit %q: %w", key, err)
		}
		units[key] = unit
	}

	for key, unit := range units {
		seenDependencies := map[string]struct{}{}
		for _, raw := range unit.DependsOn {
			dependency := strings.TrimSpace(raw)
			if dependency == "" {
				return fmt.Errorf("plan draft work unit %q has a blank dependency", key)
			}
			if dependency == key {
				return fmt.Errorf("plan draft work unit %q cannot depend on itself", key)
			}
			if _, exists := units[dependency]; !exists {
				return fmt.Errorf("plan draft work unit %q depends on unknown work unit %q", key, dependency)
			}
			if _, duplicate := seenDependencies[dependency]; duplicate {
				return fmt.Errorf("plan draft work unit %q repeats dependency %q", key, dependency)
			}
			seenDependencies[dependency] = struct{}{}
		}
	}
	if _, err := p.TopologicalOrder(); err != nil {
		return err
	}
	if err := validateUniqueNonBlankPlanDraftList("assumption", p.Assumptions); err != nil {
		return err
	}
	if err := validateUniqueNonBlankPlanDraftList("blocker", p.Blockers); err != nil {
		return err
	}
	return nil
}

// TopologicalOrder derives deterministic serial execution order from dependency
// truth. Array serialization order is deliberately irrelevant.
func (p PlanDraftProposal) TopologicalOrder() ([]string, error) {
	units := make(map[string]PlanDraftWorkUnit, len(p.WorkUnits))
	indegree := make(map[string]int, len(p.WorkUnits))
	dependents := make(map[string][]string, len(p.WorkUnits))
	for _, unit := range p.WorkUnits {
		key := strings.TrimSpace(unit.Key)
		if key == "" {
			return nil, fmt.Errorf("plan draft contains a blank work unit key")
		}
		if _, duplicate := units[key]; duplicate {
			return nil, fmt.Errorf("plan draft work unit key %q is duplicated", key)
		}
		units[key] = unit
		indegree[key] = 0
	}
	for key, unit := range units {
		seen := map[string]struct{}{}
		for _, raw := range unit.DependsOn {
			dependency := strings.TrimSpace(raw)
			if dependency == "" || dependency == key {
				return nil, fmt.Errorf("plan draft contains an invalid dependency for %q", key)
			}
			if _, exists := units[dependency]; !exists {
				return nil, fmt.Errorf("plan draft work unit %q depends on unknown work unit %q", key, dependency)
			}
			if _, duplicate := seen[dependency]; duplicate {
				return nil, fmt.Errorf("plan draft work unit %q repeats dependency %q", key, dependency)
			}
			seen[dependency] = struct{}{}
			indegree[key]++
			dependents[dependency] = append(dependents[dependency], key)
		}
	}

	var ready []string
	for key, degree := range indegree {
		if degree == 0 {
			ready = append(ready, key)
		}
	}
	sort.Strings(ready)
	order := make([]string, 0, len(units))
	for len(ready) > 0 {
		key := ready[0]
		ready = ready[1:]
		order = append(order, key)
		next := append([]string(nil), dependents[key]...)
		sort.Strings(next)
		for _, dependent := range next {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				ready = append(ready, dependent)
				sort.Strings(ready)
			}
		}
	}
	if len(order) != len(units) {
		return nil, fmt.Errorf("plan draft work unit dependencies contain a cycle")
	}
	return order, nil
}

func validateUniqueNonBlankPlanDraftList(kind string, values []string) error {
	seen := map[string]struct{}{}
	for i, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("%s %d is blank", kind, i+1)
		}
		if _, duplicate := seen[trimmed]; duplicate {
			return fmt.Errorf("%s %q is duplicated", kind, trimmed)
		}
		seen[trimmed] = struct{}{}
	}
	return nil
}

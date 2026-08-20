// Package outcome validates an orchestrator-produced outcome plan before it
// can create workers. It is intentionally deterministic: the model proposes
// tasks, but AO remains the authority that rejects cycles, missing dependencies,
// and unavailable harness assignments.
package outcome

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

// Plan is the validated worker graph that may be handed to AO's existing
// session delegation service.
type Plan struct {
	Outcome domain.Outcome
	Tasks   []domain.OutcomeTask
}

// BuildPlan validates an agent-proposed graph and resolves every task to a
// harness known to be available for this machine. An explicit request is never
// silently changed: choosing a specific harness is a user/model constraint,
// while an empty request asks AO to choose deterministically.
func BuildPlan(outcome domain.Outcome, tasks []domain.OutcomeTask, available []domain.AgentHarness) (Plan, error) {
	if err := outcome.Validate(); err != nil {
		return Plan{}, err
	}
	if len(tasks) == 0 {
		return Plan{}, fmt.Errorf("outcome plan requires at least one task")
	}
	availableSet := make(map[domain.AgentHarness]struct{}, len(available))
	for _, harness := range available {
		if harness.IsKnown() {
			availableSet[harness] = struct{}{}
		}
	}
	if len(availableSet) == 0 {
		return Plan{}, fmt.Errorf("outcome plan requires at least one available harness")
	}

	byID := make(map[string]domain.OutcomeTask, len(tasks))
	for i := range tasks {
		task := tasks[i]
		task.ID = strings.TrimSpace(task.ID)
		task.OutcomeID = outcome.ID
		task.Title = strings.TrimSpace(task.Title)
		task.Brief = strings.TrimSpace(task.Brief)
		if task.ID == "" || task.Title == "" || task.Brief == "" {
			return Plan{}, fmt.Errorf("outcome task %d requires id, title, and brief", i+1)
		}
		if _, exists := byID[task.ID]; exists {
			return Plan{}, fmt.Errorf("outcome plan has duplicate task id %q", task.ID)
		}
		if task.RequestedHarness != "" {
			if !task.RequestedHarness.IsKnown() {
				return Plan{}, fmt.Errorf("task %q requests unknown harness %q", task.ID, task.RequestedHarness)
			}
			if _, ok := availableSet[task.RequestedHarness]; !ok {
				return Plan{}, fmt.Errorf("task %q requests unavailable harness %q", task.ID, task.RequestedHarness)
			}
			task.AssignedHarness = task.RequestedHarness
		} else {
			task.AssignedHarness = chooseHarness(availableSet)
		}
		task.Status = domain.OutcomeTaskStatusPlanned
		byID[task.ID] = task
	}

	for _, task := range byID {
		for _, dependency := range task.DependsOn {
			if dependency == task.ID {
				return Plan{}, fmt.Errorf("task %q cannot depend on itself", task.ID)
			}
			if _, ok := byID[dependency]; !ok {
				return Plan{}, fmt.Errorf("task %q depends on unknown task %q", task.ID, dependency)
			}
		}
	}
	if hasCycle(byID) {
		return Plan{}, fmt.Errorf("outcome plan contains a dependency cycle")
	}

	resolved := make([]domain.OutcomeTask, 0, len(tasks))
	for _, task := range tasks {
		resolved = append(resolved, byID[strings.TrimSpace(task.ID)])
	}
	return Plan{Outcome: outcome, Tasks: resolved}, nil
}

func chooseHarness(available map[domain.AgentHarness]struct{}) domain.AgentHarness {
	// This is deliberately policy, not a hidden model preference. The stable
	// order makes a reviewed plan reproducible until a project-level routing
	// policy is added in a later slice.
	preferred := []domain.AgentHarness{domain.HarnessCodex, domain.HarnessClaudeCode, domain.HarnessOpenCode, domain.HarnessCursor}
	for _, harness := range preferred {
		if _, ok := available[harness]; ok {
			return harness
		}
	}
	choices := make([]string, 0, len(available))
	for harness := range available {
		choices = append(choices, string(harness))
	}
	sort.Strings(choices)
	return domain.AgentHarness(choices[0])
}

func hasCycle(tasks map[string]domain.OutcomeTask) bool {
	visiting := make(map[string]bool, len(tasks))
	visited := make(map[string]bool, len(tasks))
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return true
		}
		if visited[id] {
			return false
		}
		visiting[id] = true
		for _, dependency := range tasks[id].DependsOn {
			if visit(dependency) {
				return true
			}
		}
		visiting[id] = false
		visited[id] = true
		return false
	}
	for id := range tasks {
		if visit(id) {
			return true
		}
	}
	return false
}

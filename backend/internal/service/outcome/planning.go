// Package outcome validates an orchestrator-produced outcome plan before it
// can create workers. The model may propose work, but Kennel remains the
// deterministic authority for graph validity and provider admission.
package outcome

import (
	"fmt"
	"strings"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

// Plan is the validated worker graph handed to the execution layer.
type Plan struct {
	Outcome domain.Outcome
	Tasks   []domain.OutcomeTask
}

// BuildPlan validates an agent-proposed graph and resolves task providers from
// the already-admitted providers supplied by the caller. Explicit provider
// requests are preserved. When a task has no provider request Kennel only
// chooses automatically if exactly one provider is available; otherwise the
// ambiguity is surfaced instead of hidden behind a provider brand priority.
func BuildPlan(outcome domain.Outcome, tasks []domain.OutcomeTask, available []domain.AgentHarness) (Plan, error) {
	if err := outcome.Validate(); err != nil {
		return Plan{}, err
	}
	if len(tasks) == 0 {
		return Plan{}, fmt.Errorf("outcome plan requires at least one task")
	}
	availableSet := make(map[domain.AgentHarness]struct{}, len(available))
	for _, harness := range available {
		if harness.IsSelectableForNewWork() {
			availableSet[harness] = struct{}{}
		}
	}
	if len(availableSet) == 0 {
		return Plan{}, fmt.Errorf("outcome plan requires at least one ready provider")
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
		assigned, err := resolveHarness(task, availableSet)
		if err != nil {
			return Plan{}, err
		}
		task.AssignedHarness = assigned
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

func resolveHarness(task domain.OutcomeTask, available map[domain.AgentHarness]struct{}) (domain.AgentHarness, error) {
	if task.RequestedHarness != "" {
		if !task.RequestedHarness.IsSelectableForNewWork() {
			return "", fmt.Errorf("task %q requests provider %q that Kennel does not support", task.ID, task.RequestedHarness)
		}
		if _, ok := available[task.RequestedHarness]; !ok {
			return "", fmt.Errorf("task %q requests provider %q that is not ready", task.ID, task.RequestedHarness)
		}
		return task.RequestedHarness, nil
	}
	if len(available) != 1 {
		return "", fmt.Errorf("task %q requires an explicit provider because %d providers are ready", task.ID, len(available))
	}
	for harness := range available {
		return harness, nil
	}
	return "", fmt.Errorf("task %q has no ready provider", task.ID)
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

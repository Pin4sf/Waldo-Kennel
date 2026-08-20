package outcome

import (
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

func validOutcome() domain.Outcome {
	return domain.Outcome{ID: "out-1", ProjectID: "kennel", Title: "Outcome", Definition: "Ship the requested result.", Status: domain.OutcomeStatusPlanning}
}

func TestBuildPlanAssignsAvailableHarnessAndPreservesDependencies(t *testing.T) {
	plan, err := BuildPlan(validOutcome(), []domain.OutcomeTask{
		{ID: "design", Title: "Design", Brief: "Define the contract.", RequestedHarness: domain.HarnessClaudeCode},
		{ID: "build", Title: "Build", Brief: "Implement the contract.", DependsOn: []string{"design"}},
	}, []domain.AgentHarness{domain.HarnessOpenCode, domain.HarnessCodex, domain.HarnessClaudeCode})
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if got := plan.Tasks[0].AssignedHarness; got != domain.HarnessClaudeCode {
		t.Fatalf("explicit harness = %q", got)
	}
	if got := plan.Tasks[1].AssignedHarness; got != domain.HarnessCodex {
		t.Fatalf("default harness = %q, want codex", got)
	}
	if got := plan.Tasks[1].DependsOn; len(got) != 1 || got[0] != "design" {
		t.Fatalf("dependencies = %#v", got)
	}
}

func TestBuildPlanFailsClosedForUnavailableHarnessAndCycles(t *testing.T) {
	_, err := BuildPlan(validOutcome(), []domain.OutcomeTask{{ID: "one", Title: "One", Brief: "Do one.", RequestedHarness: domain.HarnessCodex}}, []domain.AgentHarness{domain.HarnessClaudeCode})
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("err = %v, want unavailable harness", err)
	}
	_, err = BuildPlan(validOutcome(), []domain.OutcomeTask{{ID: "one", Title: "One", Brief: "Do one.", DependsOn: []string{"two"}}, {ID: "two", Title: "Two", Brief: "Do two.", DependsOn: []string{"one"}}}, []domain.AgentHarness{domain.HarnessCodex})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("err = %v, want cycle", err)
	}
}

func TestTaskLifecycleRejectsMarkerStyleShortcuts(t *testing.T) {
	if domain.OutcomeTaskStatusWorking.CanTransitionTo(domain.OutcomeTaskStatusReadyToMerge) {
		t.Fatal("working task must not bypass evidence review")
	}
	if !domain.OutcomeTaskStatusWorking.CanTransitionTo(domain.OutcomeTaskStatusInReview) {
		t.Fatal("working task should be eligible for daemon-validated review")
	}
	if !domain.OutcomeTaskStatusInReview.CanTransitionTo(domain.OutcomeTaskStatusReadyToMerge) {
		t.Fatal("reviewed task should be eligible for ready_to_merge")
	}
	if domain.OutcomeTaskStatusOnHold.CanTransitionTo(domain.OutcomeTaskStatusReadyToMerge) {
		t.Fatal("dependency hold must not imply completion")
	}
}

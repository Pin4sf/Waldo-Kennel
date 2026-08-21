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
		{ID: "design", Title: "Design", Brief: "Define the contract.", RequestedHarness: domain.HarnessCodex},
		{ID: "build", Title: "Build", Brief: "Implement the contract.", DependsOn: []string{"design"}},
	}, []domain.AgentHarness{domain.HarnessCodex})
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if got := plan.Tasks[0].AssignedHarness; got != domain.HarnessCodex {
		t.Fatalf("explicit harness = %q", got)
	}
	if got := plan.Tasks[1].AssignedHarness; got != domain.HarnessCodex {
		t.Fatalf("default harness = %q, want codex", got)
	}
	if got := plan.Tasks[1].DependsOn; len(got) != 1 || got[0] != "design" {
		t.Fatalf("dependencies = %#v", got)
	}
}

func TestBuildPlanRejectsHistoricalRequestedAndAvailableHarnesses(t *testing.T) {
	_, err := BuildPlan(validOutcome(), []domain.OutcomeTask{{
		ID: "one", Title: "One", Brief: "Do one.", RequestedHarness: domain.HarnessClaudeCode,
	}}, []domain.AgentHarness{domain.HarnessCodex, domain.HarnessClaudeCode})
	if err == nil || !strings.Contains(err.Error(), "not selectable for new work") {
		t.Fatalf("historical requested harness error = %v, want non-selectable rejection", err)
	}

	_, err = BuildPlan(validOutcome(), []domain.OutcomeTask{{
		ID: "one", Title: "One", Brief: "Do one.",
	}}, []domain.AgentHarness{domain.HarnessClaudeCode})
	if err == nil || !strings.Contains(err.Error(), "at least one available") {
		t.Fatalf("historical available harness error = %v, want no selectable harness rejection", err)
	}
}

func TestBuildPlanFailsClosedForUnavailableHarnessAndCycles(t *testing.T) {
	_, err := BuildPlan(validOutcome(), []domain.OutcomeTask{{ID: "one", Title: "One", Brief: "Do one.", RequestedHarness: domain.HarnessCodex}}, []domain.AgentHarness{domain.HarnessClaudeCode})
	if err == nil || !strings.Contains(err.Error(), "at least one available") {
		t.Fatalf("err = %v, want no selectable harness", err)
	}
	_, err = BuildPlan(validOutcome(), []domain.OutcomeTask{{ID: "one", Title: "One", Brief: "Do one.", DependsOn: []string{"two"}}, {ID: "two", Title: "Two", Brief: "Do two.", DependsOn: []string{"one"}}}, []domain.AgentHarness{domain.HarnessCodex})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("err = %v, want cycle", err)
	}
}

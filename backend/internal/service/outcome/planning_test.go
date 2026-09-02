package outcome

import (
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

func validOutcome() domain.Outcome {
	return domain.Outcome{ID: "out-1", ProjectID: "kennel", Title: "Outcome", Definition: "Ship the requested result.", Status: domain.OutcomeStatusPlanning}
}

func TestBuildPlanPreservesExplicitProviderAndDependencies(t *testing.T) {
	plan, err := BuildPlan(validOutcome(), []domain.OutcomeTask{
		{ID: "design", Title: "Design", Brief: "Define the contract.", RequestedHarness: domain.HarnessClaudeCode},
		{ID: "build", Title: "Build", Brief: "Implement the contract.", DependsOn: []string{"design"}, RequestedHarness: domain.HarnessPi},
	}, []domain.AgentHarness{domain.HarnessCodex, domain.HarnessClaudeCode, domain.HarnessPi})
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if got := plan.Tasks[0].AssignedHarness; got != domain.HarnessClaudeCode {
		t.Fatalf("design provider = %q, want claude-code", got)
	}
	if got := plan.Tasks[1].AssignedHarness; got != domain.HarnessPi {
		t.Fatalf("build provider = %q, want pi", got)
	}
	if got := plan.Tasks[1].DependsOn; len(got) != 1 || got[0] != "design" {
		t.Fatalf("dependencies = %#v", got)
	}
}

func TestBuildPlanAutomaticallyUsesOnlyReadyProvider(t *testing.T) {
	plan, err := BuildPlan(validOutcome(), []domain.OutcomeTask{{ID: "one", Title: "One", Brief: "Do one."}}, []domain.AgentHarness{domain.HarnessOpenCode})
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if got := plan.Tasks[0].AssignedHarness; got != domain.HarnessOpenCode {
		t.Fatalf("provider = %q, want opencode", got)
	}
}

func TestBuildPlanRequiresExplicitProviderWhenMultipleAreReady(t *testing.T) {
	_, err := BuildPlan(validOutcome(), []domain.OutcomeTask{{ID: "one", Title: "One", Brief: "Do one."}}, []domain.AgentHarness{domain.HarnessCodex, domain.HarnessClaudeCode})
	if err == nil || !strings.Contains(err.Error(), "requires an explicit provider") {
		t.Fatalf("err = %v, want ambiguity error", err)
	}
}

func TestBuildPlanRejectsUnsupportedOrUnreadyProvider(t *testing.T) {
	_, err := BuildPlan(validOutcome(), []domain.OutcomeTask{{ID: "one", Title: "One", Brief: "Do one.", RequestedHarness: "aider"}}, []domain.AgentHarness{domain.HarnessCodex})
	if err == nil || !strings.Contains(err.Error(), "does not support") {
		t.Fatalf("unsupported provider error = %v", err)
	}

	_, err = BuildPlan(validOutcome(), []domain.OutcomeTask{{ID: "one", Title: "One", Brief: "Do one.", RequestedHarness: domain.HarnessCursor}}, []domain.AgentHarness{domain.HarnessCodex})
	if err == nil || !strings.Contains(err.Error(), "not ready") {
		t.Fatalf("unready provider error = %v", err)
	}
}

func TestBuildPlanRejectsCycles(t *testing.T) {
	_, err := BuildPlan(validOutcome(), []domain.OutcomeTask{
		{ID: "one", Title: "One", Brief: "Do one.", DependsOn: []string{"two"}, RequestedHarness: domain.HarnessCodex},
		{ID: "two", Title: "Two", Brief: "Do two.", DependsOn: []string{"one"}, RequestedHarness: domain.HarnessCodex},
	}, []domain.AgentHarness{domain.HarnessCodex})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("err = %v, want cycle", err)
	}
}

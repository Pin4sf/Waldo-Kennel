package outcome_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

type interactivePlanningFake struct {
	repositoryObserved bool
}

func (*interactivePlanningFake) ID() domain.IntelligenceProviderID { return "openai" }
func (*interactivePlanningFake) AnalyzeContract(context.Context, ports.ContractIntelligenceRequest) (ports.ContractIntelligenceResponse, error) {
	return ports.ContractIntelligenceResponse{}, fmt.Errorf("not used")
}
func (*interactivePlanningFake) DraftPlan(context.Context, ports.PlanIntelligenceRequest) (ports.PlanIntelligenceResponse, error) {
	return ports.PlanIntelligenceResponse{}, fmt.Errorf("not used")
}
func (*interactivePlanningFake) PlanningCandidates(context.Context) ([]ports.PlanningCandidate, error) {
	return []ports.PlanningCandidate{{
		ID: "direct-openai-planner", Ready: true,
		Binding: domain.PlanningBinding{Mode: domain.PlanningModeDirectAPI, Provider: "openai", ModelSelection: domain.PlanningModelExplicit, Model: "planner-test"},
	}}, nil
}
func (f *interactivePlanningFake) DiscussPlan(_ context.Context, request ports.PlanningDiscussionRequest) (ports.PlanningDiscussionResponse, error) {
	for _, file := range request.RepositoryContext.Files {
		if file.Path == "README.md" && strings.Contains(file.Content, "planning fixture") {
			f.repositoryObserved = true
		}
	}
	provenance := ports.IntelligenceProvenance{EffectiveProvider: "openai", EffectiveModel: "planner-test"}
	latest := request.Turns[len(request.Turns)-1].Text
	if request.Finalize {
		return ports.PlanningDiscussionResponse{Provenance: provenance, Result: ports.PlanningResult{
			Kind: ports.PlanningResultPlanProposal, Message: "A bounded implementation Plan is ready for review.",
			PlanProposal: &domain.PlanDraftProposal{Summary: "Make the bounded local change.", WorkUnits: []domain.PlanDraftWorkUnit{{
				Key: "implement", Title: "Implement the confirmed Outcome", Intent: domain.WorkUnitIntentModify,
				OutputSummary: "The requested local change is ready for review.", CriteriaCovered: []string{"C1"}, EvidenceIdeas: []string{"inspect the retained diff"},
			}}},
		}}, nil
	}
	if strings.Contains(latest, "Contract") {
		return ports.PlanningDiscussionResponse{Provenance: provenance, Result: ports.PlanningResult{
			Kind: ports.PlanningResultContractChange, Message: "The success criterion could be narrower.",
			ContractChange: &ports.PlanContractChangeProposal{Summary: "Narrow the criterion before approval if this distinction matters.", ChangedFields: []string{"successCriteria"}},
		}}, nil
	}
	return ports.PlanningDiscussionResponse{Provenance: provenance, Result: ports.PlanningResult{
		Kind: ports.PlanningResultClarification, Message: "One choice will keep the Plan small.",
		Clarification: &ports.PlanClarification{Question: "Should the first slice stay local only?", Reason: "Remote effects need separate authority.", Recommendation: "Keep it local.", Alternatives: []string{"Include remote delivery later"}},
	}}, nil
}

func TestInteractivePlanning_RepositoryDiscussionProducesOnlyAProposedPlan(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("planning fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit := func(args ...string) {
		t.Helper()
		command := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	runGit("init", "-q")
	runGit("config", "user.name", "Planning Test")
	runGit("config", "user.email", "planning@example.invalid")
	runGit("add", "README.md")
	runGit("commit", "-qm", "fixture")

	store := sqlitetest.MustOpen(t)
	project := domain.ProjectRecord{
		ID: "interactive-planning-project", Path: repo, DisplayName: "Planning fixture", RegisteredAt: time.Now().UTC(),
		Config: domain.ProjectConfig{Worker: domain.RoleOverride{Harness: domain.HarnessClaudeCode, AgentConfig: domain.AgentConfig{Model: "sonnet-test"}}},
	}
	if err := store.UpsertProject(ctx, project); err != nil {
		t.Fatalf("register project: %v", err)
	}
	provider := &interactivePlanningFake{}
	router := &routingInventoryFake{candidates: []domain.RoutingCandidate{readyClaudeCandidate()}}
	svc := outcome.New(store, nil).WithPlanning(provider, router)
	created, err := svc.Create(ctx, outcome.CreateInput{
		ProjectID: domain.ProjectID(project.ID), Title: "Interactive planning", Goal: "Make one bounded local change.",
		SuccessCriteria: []string{"The local change is reviewable."}, Review: "Owner reviews the retained diff.",
		AuthorityCeiling: domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true},
		StopConditions:   []string{"Stop before commands or external effects."}, RequestKey: "interactive-outcome",
	})
	if err != nil {
		t.Fatalf("create Outcome: %v", err)
	}
	candidates, err := svc.PlanningCandidates(ctx, created.Outcome.ID, 1)
	if err != nil || len(candidates) != 1 || !candidates[0].Ready {
		t.Fatalf("planning candidates = %+v err=%v", candidates, err)
	}
	view, err := svc.StartPlanning(ctx, created.Outcome.ID, outcome.StartPlanningInput{
		ExpectedContractRevision: 1, CandidateID: candidates[0].ID, RequestKey: "start-interactive-planning",
	})
	if err != nil {
		t.Fatalf("start planning: %v", err)
	}
	if view.Session.ContextMode != domain.PlanningContextRepositoryRead || view.Session.WaitingOn != domain.PlanningWaitingOwner || len(view.Turns) != 0 {
		t.Fatalf("initial planning view = %+v", view)
	}

	view, err = svc.ContinuePlanning(ctx, created.Outcome.ID, view.Session.ID, outcome.PlanningMessageInput{
		ExpectedSessionRevision: view.Session.Revision, Text: "Inspect it and ask what is ambiguous.", RequestKey: "planning-message-1",
	})
	if err != nil {
		t.Fatalf("continue planning: %v", err)
	}
	if !provider.repositoryObserved || len(view.Turns) != 2 || view.Turns[1].Kind != domain.PlanningTurnClarification || view.ProposedPlan != nil {
		t.Fatalf("clarification planning view = %+v repositoryObserved=%v", view, provider.repositoryObserved)
	}

	view, err = svc.ContinuePlanning(ctx, created.Outcome.ID, view.Session.ID, outcome.PlanningMessageInput{
		ExpectedSessionRevision: view.Session.Revision, Text: "Suggest any Contract change, but do not make it.", RequestKey: "planning-message-2",
	})
	if err != nil {
		t.Fatalf("request Contract suggestion: %v", err)
	}
	if view.Turns[len(view.Turns)-1].Kind != domain.PlanningTurnContractChangeProposal {
		t.Fatalf("last turn = %+v", view.Turns[len(view.Turns)-1])
	}
	unchanged, err := svc.Get(ctx, created.Outcome.ID)
	if err != nil || unchanged.Outcome.CurrentRevisionNumber != 1 || len(unchanged.History) != 1 {
		t.Fatalf("Contract changed through suggestion: revision=%d history=%d err=%v", unchanged.Outcome.CurrentRevisionNumber, len(unchanged.History), err)
	}

	view, err = svc.FinalizePlanning(ctx, created.Outcome.ID, view.Session.ID, outcome.PlanningFinalizeInput{
		ExpectedSessionRevision: view.Session.Revision, RequestKey: "planning-finalize",
	})
	if err != nil {
		t.Fatalf("finalize planning: %v", err)
	}
	if view.Session.Status != domain.PlanningSessionProposalReady || view.ProposedPlan == nil || view.ProposedPlan.Status != domain.PlanStatusProposed {
		t.Fatalf("final planning view = %+v", view)
	}
	if view.ProposedPlan.PlanningSessionID != view.Session.ID || view.ProposedPlan.SourceIntelligenceRunID.IsZero() {
		t.Fatalf("Plan lost planning provenance: %+v", view.ProposedPlan)
	}
	attempts, err := store.ListAttempts(ctx, created.Outcome.ID)
	if err != nil || len(attempts) != 0 {
		t.Fatalf("planning created execution attempts: attempts=%+v err=%v", attempts, err)
	}
}

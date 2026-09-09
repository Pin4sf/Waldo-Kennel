package intelligence

import (
	"context"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

type captureLLMClient struct {
	request ports.LLMRequest
	result  string
}

func (c *captureLLMClient) ID() string { return "test-provider" }
func (c *captureLLMClient) Complete(_ context.Context, request ports.LLMRequest) (ports.LLMResponse, error) {
	c.request = request
	return ports.LLMResponse{JSON: []byte(c.result), EffectiveModel: "test-model"}, nil
}

func TestAnalyzeContractPreservesClarificationAsContextNotTemporalSemantics(t *testing.T) {
	client := &captureLLMClient{result: `{"decision":"propose","proposal":{"title":"Login continuity","desiredState":"Email login remains available","criteria":[{"text":"Email login continues to work","evidenceExpected":["login test"]}],"reviewMethod":"Run the login test","constraints":[],"nonGoals":[],"authorityCeiling":{"readWorkspace":true,"writeWorkspace":false,"executeLocal":true,"useNetwork":false,"commitLocal":false,"createPR":false,"deploy":false,"externalEffect":false},"stopConditions":["Stop on auth data changes"],"assumptions":["Existing auth provider is retained"],"facet":"software"}}`}
	provider := NewLLMProvider(client)
	answer := "preserve email login"
	result, err := provider.AnalyzeContract(context.Background(), ports.ContractIntelligenceRequest{
		Session:           domain.IntakeSession{Statement: "Keep the login flow stable"},
		Clarification:     &domain.ClarificationRequest{Question: "What must remain stable?"},
		ClarificationText: answer,
		PreviousProposal:  &domain.OutcomeContractProposal{Title: "Old login proposal"},
		RepositoryContext: ports.RepositoryContextSnapshot{Revision: "abc123", Files: []ports.RepositoryContextFile{{Path: "auth/login.go", Content: "email login"}}},
	})
	if err != nil {
		t.Fatalf("AnalyzeContract() error = %v", err)
	}
	if result.Result.Proposal == nil || result.Result.Proposal.TemporalCondition != nil {
		t.Fatalf("proposal temporal condition = %#v", result.Result.Proposal)
	}
	for _, want := range []string{"preserve email login", "Old login proposal", "auth/login.go", "abc123"} {
		if !strings.Contains(client.request.User, want) {
			t.Errorf("request omitted grounded context %q: %s", want, client.request.User)
		}
	}
}

func TestDraftPlanCarriesExplicitReplanFeedbackAndChecks(t *testing.T) {
	client := &captureLLMClient{result: `{"summary":"Use the inspected check","workUnits":[{"key":"W1","title":"Implement","intent":"modify_and_execute","outputSummary":"Changed code and verified it","criteriaCovered":["C1"],"dependsOn":[],"evidenceIdeas":["test output"]}],"assumptions":[],"blockers":[]}`}
	provider := NewLLMProvider(client)
	_, err := provider.DraftPlan(context.Background(), ports.PlanIntelligenceRequest{
		Outcome:          domain.Outcome{Title: "Grounded work"},
		Contract:         domain.ContractRevision{Goal: "Make the repo change", Criteria: []domain.ContractCriterion{{ID: "criterion-1", Text: "It works"}}},
		CriterionAliases: map[string]domain.CriterionID{"C1": "criterion-1"},
		ReplanFeedback:   "Use the distinctive repository test command",
		RepositoryContext: ports.RepositoryContextSnapshot{
			Revision:      "abc123",
			CheckCommands: []string{"npm run test:distinctive -> go test ./internal/feature"},
		},
	})
	if err != nil {
		t.Fatalf("DraftPlan() error = %v", err)
	}
	for _, want := range []string{"distinctive repository test command", "npm run test:distinctive", "abc123"} {
		if !strings.Contains(client.request.User, want) {
			t.Errorf("request omitted replan/context %q: %s", want, client.request.User)
		}
	}
}

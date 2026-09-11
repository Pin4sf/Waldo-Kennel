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

func TestPlanSchemaAllowsExecutableCriterionChecks(t *testing.T) {
	properties := planSchema([]string{"C1"})["properties"].(map[string]any)
	unit := properties["workUnits"].(map[string]any)["items"].(map[string]any)
	checks, ok := unit["properties"].(map[string]any)["checkCommands"].(map[string]any)
	if !ok {
		t.Fatal("structured plan schema forbids executable checkCommands accepted by the compiler")
	}
	item := checks["items"].(map[string]any)
	for _, name := range []string{"criterionAlias", "argv", "timeoutSeconds"} {
		if _, ok := item["properties"].(map[string]any)[name]; !ok {
			t.Errorf("check schema missing %s", name)
		}
	}
}

func TestDraftPlanPreservesExecutableCheckArguments(t *testing.T) {
	client := &captureLLMClient{result: `{"summary":"Verify greeting","workUnits":[{"key":"W1","title":"Change and check","intent":"modify_and_execute","outputSummary":"Correct greeting","criteriaCovered":["C1"],"checkCommands":[{"criterionAlias":"C1","argv":["python3","-c","import subprocess; assert subprocess.check_output(['python3', 'greet.py']) == b'Hello Kennel\\n'\n"],"timeoutSeconds":12}]}]}`}
	result, err := NewLLMProvider(client).DraftPlan(context.Background(), ports.PlanIntelligenceRequest{CriterionAliases: map[string]domain.CriterionID{"C1": "criterion-1"}})
	if err != nil {
		t.Fatal(err)
	}
	checks := result.Proposal.WorkUnits[0].CheckCommands
	if len(checks) != 1 || checks[0].CriterionAlias != "C1" || checks[0].TimeoutSeconds != 12 || len(checks[0].Argv) != 3 || checks[0].Argv[2] != "import subprocess; assert subprocess.check_output(['python3', 'greet.py']) == b'Hello Kennel\\n'\n" {
		t.Fatalf("check arguments or criterion binding changed: %#v", checks)
	}
}

func TestDraftPlanReceivesFrozenPermissions(t *testing.T) {
	for _, execute := range []bool{false, true} {
		client := &captureLLMClient{result: `{"workUnits":[]}`}
		_, err := NewLLMProvider(client).DraftPlan(context.Background(), ports.PlanIntelligenceRequest{
			Contract: domain.ContractRevision{AuthorityCeiling: domain.ProposedAuthority{ReadWorkspace: true, ExecuteLocal: execute}},
		})
		if err != nil {
			t.Fatal(err)
		}
		want := "executeLocal=false"
		if execute {
			want = "executeLocal=true"
		}
		for _, field := range []string{"readWorkspace=true", "writeWorkspace=false", want, "useNetwork=false", "commitLocal=false", "createPR=false", "deploy=false", "externalEffect=false"} {
			if !strings.Contains(client.request.User, field) {
				t.Errorf("missing frozen permission %s", field)
			}
		}
		unit := client.request.Schema["properties"].(map[string]any)["workUnits"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
		wantType := "null"
		if execute {
			wantType = "array"
		}
		if unit["checkCommands"].(map[string]any)["type"] != wantType {
			t.Fatal("schema permits checks outside the frozen command boundary")
		}
		if !execute && !strings.Contains(client.request.User, "checkCommands must be empty") {
			t.Error("read-only planning omitted the command-check restriction")
		}
	}
}

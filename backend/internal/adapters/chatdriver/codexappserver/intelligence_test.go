package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

func testIntelligenceRequest() ports.LLMRequest {
	return ports.LLMRequest{
		System:     "You are a bounded planner.",
		User:       "Draft the smallest plan for the approved outcome.",
		SchemaName: "plan_draft",
		Schema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []any{"summary"},
			"properties":           map[string]any{"summary": map[string]any{"type": "string"}},
		},
	}
}

func TestIntelligenceClientPinsBoundedStructuredTurn(t *testing.T) {
	d, srv := newTestDriver(t)
	client := NewIntelligenceClient(d, IntelligenceConfig{Model: "approved-model", Effort: "high", Timeout: time.Second})

	result := make(chan struct {
		response ports.LLMResponse
		err      error
	}, 1)
	go func() {
		response, err := client.Complete(context.Background(), testIntelligenceRequest())
		result <- struct {
			response ports.LLMResponse
			err      error
		}{response, err}
	}()

	start := srv.awaitFrame(func(f frame) bool { return f.Method == "thread/start" })
	var startParams struct {
		ApprovalPolicy string         `json:"approvalPolicy"`
		Sandbox        string         `json:"sandbox"`
		Ephemeral      bool           `json:"ephemeral"`
		Model          string         `json:"model"`
		Config         map[string]any `json:"config"`
	}
	if err := json.Unmarshal(start.Params, &startParams); err != nil {
		t.Fatalf("thread/start params: %v", err)
	}
	if startParams.ApprovalPolicy != "never" || startParams.Sandbox != "read-only" || !startParams.Ephemeral {
		t.Fatalf("unsafe intelligence thread posture: %+v", startParams)
	}
	if startParams.Model != "approved-model" {
		t.Fatalf("thread model = %q, want approved-model", startParams.Model)
	}
	if got, ok := startParams.Config["mcp_servers"].(map[string]any); !ok || len(got) != 0 {
		t.Fatalf("ambient MCP config = %#v, want an explicit empty map", startParams.Config["mcp_servers"])
	}
	features, ok := startParams.Config["features"].(map[string]any)
	if !ok || features["plugins"] != false || features["apps"] != false {
		t.Fatalf("ambient plugin/app config = %#v", startParams.Config["features"])
	}

	turn := srv.awaitFrame(func(f frame) bool { return f.Method == "turn/start" })
	var turnParams struct {
		Model          string          `json:"model"`
		Effort         string          `json:"effort"`
		ApprovalPolicy string          `json:"approvalPolicy"`
		SandboxPolicy  map[string]any  `json:"sandboxPolicy"`
		OutputSchema   json.RawMessage `json:"outputSchema"`
		ClientMessage  string          `json:"clientUserMessageId"`
	}
	if err := json.Unmarshal(turn.Params, &turnParams); err != nil {
		t.Fatalf("turn/start params: %v", err)
	}
	if turnParams.Model != "approved-model" || turnParams.Effort != "high" || turnParams.ApprovalPolicy != "never" {
		t.Fatalf("turn selection/posture = %+v", turnParams)
	}
	if turnParams.SandboxPolicy["type"] != "readOnly" || turnParams.SandboxPolicy["networkAccess"] != false {
		t.Fatalf("turn sandbox policy = %#v", turnParams.SandboxPolicy)
	}
	if !json.Valid(turnParams.OutputSchema) || turnParams.ClientMessage == "" {
		t.Fatalf("structured output or idempotency key missing: schema=%s id=%q", turnParams.OutputSchema, turnParams.ClientMessage)
	}

	// A stale response for another turn is ignored. The settled response is
	// accepted only for the provider turn the adapter started.
	srv.push(`{"method":"item/completed","params":{"threadId":"thread-1","turnId":"stale-turn","item":{"id":"stale","type":"agentMessage","text":"not this request"}}}`)
	srv.push(`{"method":"item/completed","params":{"threadId":"thread-1","turnId":"turn-1","item":{"id":"msg-1","type":"agentMessage","text":"{\"summary\":\"bounded\"}"}}}`)
	srv.push(`{"method":"turn/completed","params":{"threadId":"thread-1","turn":{"id":"turn-1","status":"completed","items":[]}}}`)

	got := <-result
	if got.err != nil {
		t.Fatalf("Complete: %v", got.err)
	}
	if string(got.response.JSON) != `{"summary":"bounded"}` {
		t.Fatalf("JSON = %s", got.response.JSON)
	}
	if got.response.EffectiveModel != "gpt-test" || got.response.NativeSessionRef != "thread-1" {
		t.Fatalf("provenance = %+v", got.response)
	}
}

func TestIntelligenceClientRejectsMalformedSettledReply(t *testing.T) {
	d, srv := newTestDriver(t)
	client := NewIntelligenceClient(d, IntelligenceConfig{Timeout: time.Second})
	result := make(chan error, 1)
	go func() {
		_, err := client.Complete(context.Background(), testIntelligenceRequest())
		result <- err
	}()
	srv.awaitFrame(func(f frame) bool { return f.Method == "turn/start" })
	srv.push(`{"method":"item/completed","params":{"threadId":"thread-1","turnId":"turn-1","item":{"id":"msg-1","type":"agentMessage","text":"not json"}}}`)
	srv.push(`{"method":"turn/completed","params":{"threadId":"thread-1","turn":{"id":"turn-1","status":"completed","items":[]}}}`)
	err := <-result
	var failure *ports.ReasoningFailure
	if !errors.As(err, &failure) || failure.Kind != ports.ReasoningInvalidOutput {
		t.Fatalf("err = %v, want invalid structured output", err)
	}
}

func TestIntelligenceClientCancellationInterruptsNamedTurn(t *testing.T) {
	d, srv := newTestDriver(t)
	client := NewIntelligenceClient(d, IntelligenceConfig{Timeout: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := client.Complete(ctx, testIntelligenceRequest())
		result <- err
	}()
	srv.awaitFrame(func(f frame) bool { return f.Method == "turn/start" })
	srv.awaitResponse("turn/start")
	cancel()
	interrupt := srv.awaitFrame(func(f frame) bool { return f.Method == "turn/interrupt" })
	var params struct {
		TurnID string `json:"turnId"`
	}
	if err := json.Unmarshal(interrupt.Params, &params); err != nil || params.TurnID != "turn-1" {
		t.Fatalf("interrupt params = %s (%v)", interrupt.Params, err)
	}
	err := <-result
	var failure *ports.ReasoningFailure
	if !errors.As(err, &failure) || failure.Kind != ports.ReasoningCancelled {
		t.Fatalf("err = %v, want cancelled reasoning failure", err)
	}
}

func TestIntelligenceClientRequiresCodexAuth(t *testing.T) {
	d, _ := newTestDriver(t)
	d.plugin = fakePlugin{bin: "codex", authStatus: ports.AgentAuthStatusUnauthorized}
	client := NewIntelligenceClient(d, IntelligenceConfig{})
	_, err := client.Complete(context.Background(), testIntelligenceRequest())
	var failure *ports.ReasoningFailure
	if !errors.As(err, &failure) || failure.Kind != ports.ReasoningUnauthorized {
		t.Fatalf("err = %v, want unauthorized reasoning failure", err)
	}
	if !strings.Contains(failure.Error(), "sign in") {
		t.Fatalf("auth error = %q, want actionable sign-in guidance", failure.Error())
	}
}

package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
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

func TestIntelligenceCapabilityFailsClosedBeforePermissionProfileProtocol(t *testing.T) {
	d, _ := newTestDriver(t)
	d.versionProbe = func(context.Context, string) (string, error) { return "codex-cli 0.153.3", nil }
	if err := d.ProbeIntelligence(context.Background()); !errors.Is(err, ports.ErrChatDriverIncompatible) {
		t.Fatalf("ProbeIntelligence error = %v, want incompatible permission-profile protocol", err)
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
		Permissions    string         `json:"permissions"`
		RuntimeRoots   []string       `json:"runtimeWorkspaceRoots"`
		Environments   []any          `json:"environments"`
		Ephemeral      bool           `json:"ephemeral"`
		Model          string         `json:"model"`
		Config         map[string]any `json:"config"`
	}
	if err := json.Unmarshal(start.Params, &startParams); err != nil {
		t.Fatalf("thread/start params: %v", err)
	}
	if startParams.ApprovalPolicy != "never" || startParams.Permissions != intelligencePermissionProfile || !startParams.Ephemeral {
		t.Fatalf("unsafe intelligence thread posture: %+v", startParams)
	}
	if len(startParams.RuntimeRoots) != 1 || len(startParams.Environments) != 0 {
		t.Fatalf("intelligence runtime scope = roots:%v environments:%v", startParams.RuntimeRoots, startParams.Environments)
	}
	if startParams.Model != "approved-model" {
		t.Fatalf("thread model = %q, want approved-model", startParams.Model)
	}
	if got, ok := startParams.Config["mcp_servers"].(map[string]any); !ok || len(got) != 0 {
		t.Fatalf("ambient MCP config = %#v, want an explicit empty map", startParams.Config["mcp_servers"])
	}
	profiles, ok := startParams.Config["permissions"].(map[string]any)
	if !ok {
		t.Fatalf("permission profiles = %#v", startParams.Config["permissions"])
	}
	profile, ok := profiles[intelligencePermissionProfile].(map[string]any)
	if !ok {
		t.Fatalf("reasoning permission profile = %#v", profiles[intelligencePermissionProfile])
	}
	filesystem, ok := profile["filesystem"].(map[string]any)
	if !ok || filesystem[":minimal"] != "read" || filesystem[startParams.RuntimeRoots[0]] != "read" || len(filesystem) != 2 {
		t.Fatalf("reasoning filesystem scope = %#v", profile["filesystem"])
	}
	network, ok := profile["network"].(map[string]any)
	if !ok || network["enabled"] != false {
		t.Fatalf("reasoning network scope = %#v", profile["network"])
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
		Permissions    string          `json:"permissions"`
		RuntimeRoots   []string        `json:"runtimeWorkspaceRoots"`
		OutputSchema   json.RawMessage `json:"outputSchema"`
		ClientMessage  string          `json:"clientUserMessageId"`
		Input          []struct {
			Text string `json:"text"`
		} `json:"input"`
	}
	if err := json.Unmarshal(turn.Params, &turnParams); err != nil {
		t.Fatalf("turn/start params: %v", err)
	}
	if turnParams.Model != "approved-model" || turnParams.Effort != "high" || turnParams.ApprovalPolicy != "never" || turnParams.Permissions != intelligencePermissionProfile {
		t.Fatalf("turn selection/posture = %+v", turnParams)
	}
	if len(turnParams.RuntimeRoots) != 1 || turnParams.RuntimeRoots[0] != startParams.RuntimeRoots[0] {
		t.Fatalf("turn runtime roots = %#v", turnParams.RuntimeRoots)
	}
	if len(turnParams.Input) != 1 || !strings.Contains(turnParams.Input[0].Text, "Do not use tools") {
		t.Fatalf("packet-only prompt did not retain no-tool instruction: %#v", turnParams.Input)
	}
	if !json.Valid(turnParams.OutputSchema) || turnParams.ClientMessage == "" {
		t.Fatalf("structured output or idempotency key missing: schema=%s id=%q", turnParams.OutputSchema, turnParams.ClientMessage)
	}

	// Identity-less and stale responses are ignored. The settled response is
	// accepted only for the provider turn the adapter started.
	srv.push(`{"method":"item/completed","params":{"threadId":"thread-1","item":{"id":"missing-turn","type":"agentMessage","text":"not this request"}}}`)
	srv.push(`{"method":"turn/completed","params":{"threadId":"thread-1","turn":{"status":"completed","items":[]}}}`)
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

func TestIntelligenceClientPreservesProviderDefaultSelection(t *testing.T) {
	d, srv := newTestDriver(t)
	client := NewIntelligenceClient(d, IntelligenceConfig{Timeout: time.Second})
	result := make(chan struct {
		response ports.LLMResponse
		err      error
	}, 1)
	go func() {
		response, err := client.Complete(context.Background(), testIntelligenceRequest())
		result <- struct {
			response ports.LLMResponse
			err      error
		}{response: response, err: err}
	}()

	start := srv.awaitFrame(func(f frame) bool { return f.Method == "thread/start" })
	var startParams map[string]any
	if err := json.Unmarshal(start.Params, &startParams); err != nil {
		t.Fatal(err)
	}
	if _, found := startParams["model"]; found {
		t.Fatalf("provider-default thread named a model: %#v", startParams["model"])
	}
	turn := srv.awaitFrame(func(f frame) bool { return f.Method == "turn/start" })
	var turnParams map[string]any
	if err := json.Unmarshal(turn.Params, &turnParams); err != nil {
		t.Fatal(err)
	}
	if _, found := turnParams["model"]; found {
		t.Fatalf("provider-default turn named a model: %#v", turnParams["model"])
	}

	srv.push(`{"method":"item/completed","params":{"threadId":"thread-1","turnId":"turn-1","item":{"id":"msg-1","type":"agentMessage","text":"{\"summary\":\"default\"}"}}}`)
	srv.push(`{"method":"turn/completed","params":{"threadId":"thread-1","turn":{"id":"turn-1","status":"completed","items":[]}}}`)
	got := <-result
	if got.err != nil || got.response.EffectiveModel != "gpt-test" {
		t.Fatalf("provider-default result = %+v err=%v", got.response, got.err)
	}
}

func TestIntelligenceClientPinsAuthorizedRepositoryReadWithoutWidening(t *testing.T) {
	d, srv := newTestDriver(t)
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	client := NewIntelligenceClient(d, IntelligenceConfig{Timeout: time.Second})
	request := testIntelligenceRequest()
	request.ContextAccess = ports.ReasoningContextAccess{Mode: ports.ReasoningContextRepositoryRead, Root: root}

	result := make(chan error, 1)
	go func() {
		_, err := client.Complete(context.Background(), request)
		result <- err
	}()

	start := srv.awaitFrame(func(f frame) bool { return f.Method == "thread/start" })
	var startParams struct {
		Cwd          string         `json:"cwd"`
		Permissions  string         `json:"permissions"`
		Sandbox      string         `json:"sandbox"`
		RuntimeRoots []string       `json:"runtimeWorkspaceRoots"`
		Config       map[string]any `json:"config"`
	}
	if err := json.Unmarshal(start.Params, &startParams); err != nil {
		t.Fatalf("thread/start params: %v", err)
	}
	if startParams.Cwd != root || startParams.Permissions != intelligencePermissionProfile || startParams.Sandbox != "" {
		t.Fatalf("repository reasoning posture = %+v", startParams)
	}
	if len(startParams.RuntimeRoots) != 1 || startParams.RuntimeRoots[0] != root {
		t.Fatalf("repository runtime roots = %v", startParams.RuntimeRoots)
	}
	profiles := startParams.Config["permissions"].(map[string]any)
	profile := profiles[intelligencePermissionProfile].(map[string]any)
	filesystem := profile["filesystem"].(map[string]any)
	if filesystem[root] != "read" || filesystem[":minimal"] != "read" || len(filesystem) != 2 {
		t.Fatalf("repository filesystem scope = %#v", filesystem)
	}

	turn := srv.awaitFrame(func(f frame) bool { return f.Method == "turn/start" })
	var turnParams struct {
		Input []struct {
			Text string `json:"text"`
		} `json:"input"`
	}
	if err := json.Unmarshal(turn.Params, &turnParams); err != nil {
		t.Fatalf("turn/start params: %v", err)
	}
	if len(turnParams.Input) != 1 || !strings.Contains(turnParams.Input[0].Text, root) || strings.Contains(turnParams.Input[0].Text, "Do not use tools") {
		t.Fatalf("repository tool instruction = %#v", turnParams.Input)
	}
	srv.push(`{"method":"item/completed","params":{"threadId":"thread-1","turnId":"turn-1","item":{"id":"msg-1","type":"agentMessage","text":"{\"summary\":\"inspected\"}"}}}`)
	srv.push(`{"method":"turn/completed","params":{"threadId":"thread-1","turn":{"id":"turn-1","status":"completed","items":[]}}}`)
	if err := <-result; err != nil {
		t.Fatalf("Complete: %v", err)
	}
}

func TestIntelligenceClientFailsClosedOnReportedFileChange(t *testing.T) {
	d, srv := newTestDriver(t)
	client := NewIntelligenceClient(d, IntelligenceConfig{Timeout: time.Second})
	result := make(chan error, 1)
	go func() {
		_, err := client.Complete(context.Background(), testIntelligenceRequest())
		result <- err
	}()
	srv.awaitFrame(func(f frame) bool { return f.Method == "turn/start" })
	srv.push(`{"method":"item/started","params":{"threadId":"thread-1","turnId":"turn-1","item":{"id":"change-1","type":"fileChange","changes":[]}}}`)
	err := <-result
	var failure *ports.ReasoningFailure
	if !errors.As(err, &failure) || failure.Err == nil || !strings.Contains(failure.Err.Error(), "forbidden file change") {
		t.Fatalf("Complete error = %v, want forbidden file change", err)
	}
}

func TestSendTurnRejectsMissingProviderTurnID(t *testing.T) {
	d, srv := newTestDriver(t)
	conv, err := d.connect(context.Background(), "/tmp/ws", nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conv.Close()
	conv.start("thread-1", "gpt-test", "high", nil)
	srv.respondTo("turn/start", `{"turn":{"status":"inProgress","items":[]}}`)
	_, err = conv.sendTurn(context.Background(), ports.ChatUserMessage{Text: "hello"}, nil)
	if err == nil || !strings.Contains(err.Error(), "no turn id") {
		t.Fatalf("sendTurn error = %v, want missing turn id", err)
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

func TestIntelligenceClientDoesNotAcceptQueuedCompletionAfterCancellation(t *testing.T) {
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
	// Make the success frames available at the same boundary as cancellation.
	// waitForIntelligenceTurn must re-check ctx after receiving either frame.
	srv.push(`{"method":"item/completed","params":{"threadId":"thread-1","turnId":"turn-1","item":{"id":"msg-1","type":"agentMessage","text":"late"}}}`)
	srv.push(`{"method":"turn/completed","params":{"threadId":"thread-1","turn":{"id":"turn-1","status":"completed","items":[]}}}`)
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

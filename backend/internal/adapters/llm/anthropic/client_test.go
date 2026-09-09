package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

func anthropicRequest() ports.LLMRequest {
	return ports.LLMRequest{System: "system", User: "user", SchemaName: "answer", Schema: map[string]any{"type": "object", "properties": map[string]any{"ok": map[string]any{"type": "boolean"}}, "required": []any{"ok"}, "additionalProperties": false}}
}

func TestCompleteSendsStructuredRequestAndParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" || r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("request = %s %s headers=%v", r.Method, r.URL.Path, r.Header)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "claude-test" || body["max_tokens"] != float64(16000) {
			t.Fatalf("body = %#v", body)
		}
		if _, ok := body["output_config"]; !ok {
			t.Fatalf("body lacks structured output config: %#v", body)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg-1","type":"message","role":"assistant","model":"claude-effective","content":[{"type":"text","text":"{\"ok\":true}"}],"stop_reason":"end_turn","usage":{"input_tokens":3,"output_tokens":2}}`))
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "test-key", Model: "claude-test", BaseURL: server.URL, HTTPClient: server.Client(), MaxRetries: 0})
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.Complete(context.Background(), anthropicRequest())
	if err != nil {
		t.Fatal(err)
	}
	if string(got.JSON) != `{"ok":true}` || got.EffectiveModel != "claude-effective" || *got.InputTokens != 3 || *got.OutputTokens != 2 {
		t.Fatalf("response = %+v", got)
	}
}

func TestCompleteRejectsRefusalMalformedAndHTTPFailuresWithoutRetry(t *testing.T) {
	for _, tc := range []struct {
		name, response string
		status         int
		want           ports.ReasoningFailureKind
	}{
		{"refusal", `{"id":"m","type":"message","role":"assistant","model":"m","content":[],"stop_reason":"refusal","stop_details":{"category":"safety"}}`, http.StatusOK, ports.ReasoningDeclined},
		{"malformed", `{"id":"m","type":"message","role":"assistant","model":"m","content":[{"type":"text","text":"not-json"}],"stop_reason":"end_turn"}`, http.StatusOK, ports.ReasoningInvalidOutput},
		{"unauthorized", `{"error":{"type":"authentication_error","message":"bad key"}}`, http.StatusUnauthorized, ports.ReasoningUnauthorized},
		{"rate-limited", `{"error":{"type":"rate_limit_error","message":"slow down"}}`, http.StatusTooManyRequests, ports.ReasoningRateLimited},
		{"server-fault", `{"error":{"type":"api_error","message":"boom"}}`, http.StatusBadGateway, ports.ReasoningUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				// The SDK only parses a JSON body when it is announced as
				// JSON. Without this header it fails at the transport layer,
				// which is what previously let the refusal/malformed rows
				// "pass" while never reaching the code they name.
				w.Header().Set("content-type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.response))
			}))
			defer server.Close()
			client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, HTTPClient: server.Client(), MaxRetries: 0})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Complete(context.Background(), anthropicRequest())
			if got := reasoningKind(t, err); got != tc.want {
				t.Fatalf("kind = %q, want %q (err = %v)", got, tc.want, err)
			}
			if calls != 1 {
				t.Fatalf("calls = %d, want one with retries disabled", calls)
			}
		})
	}
}

// Asserting the classification rather than the error message is what keeps
// this deterministic; see the matching note in the openai adapter's tests.
func TestCompleteClassifiesClientDeadlineAsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, HTTPClient: &http.Client{Transport: server.Client().Transport, Timeout: 20 * time.Millisecond}, MaxRetries: 0})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Complete(context.Background(), anthropicRequest())
	if got := reasoningKind(t, err); got != ports.ReasoningTimedOut {
		t.Fatalf("kind = %q, want %q (err = %v)", got, ports.ReasoningTimedOut, err)
	}
}

func TestCompleteClassifiesCallerCancellationAsCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, HTTPClient: server.Client(), MaxRetries: 0})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	defer cancel()
	_, err = client.Complete(ctx, anthropicRequest())
	if got := reasoningKind(t, err); got != ports.ReasoningCancelled {
		t.Fatalf("kind = %q, want %q (err = %v)", got, ports.ReasoningCancelled, err)
	}
}

func TestNewWithoutKeyReportsMissingSetupNotAnOpaqueFailure(t *testing.T) {
	_, err := New(Config{})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
	if got := reasoningKind(t, err); got != ports.ReasoningNotConfigured {
		t.Fatalf("kind = %q, want %q", got, ports.ReasoningNotConfigured)
	}
}

// reasoningKind fails the test unless err carries a classification, so an
// unclassified reasoning failure can never pass silently.
func reasoningKind(t *testing.T, err error) ports.ReasoningFailureKind {
	t.Helper()
	if err == nil {
		t.Fatal("expected a reasoning failure, got nil")
	}
	var failure *ports.ReasoningFailure
	if !errors.As(err, &failure) {
		t.Fatalf("error is not classified: %v", err)
	}
	return failure.Kind
}

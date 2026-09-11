package openai

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

func openAIRequest() ports.LLMRequest {
	return ports.LLMRequest{System: "system", User: "user", SchemaName: "answer", Schema: map[string]any{"type": "object", "properties": map[string]any{"ok": map[string]any{"type": "boolean"}}, "required": []any{"ok"}, "additionalProperties": false}}
}

func TestCompleteSendsStructuredRequestAndParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" || r.Header.Get("authorization") != "Bearer test-key" {
			t.Errorf("request = %s %s authorization=%q", r.Method, r.URL.Path, r.Header.Get("authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-test" || body["max_output_tokens"] != float64(32000) {
			t.Fatalf("body = %#v", body)
		}
		if _, ok := body["text"]; !ok {
			t.Fatalf("body lacks structured output config: %#v", body)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp-1","object":"response","created_at":1,"status":"completed","model":"gpt-effective","output":[{"type":"message","id":"msg-1","status":"completed","role":"assistant","content":[{"type":"output_text","text":"{\"ok\":true}","annotations":[]}]}],"usage":{"input_tokens":3,"output_tokens":2,"total_tokens":5}}`))
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "test-key", Model: "gpt-test", BaseURL: server.URL + "/v1", HTTPClient: server.Client(), MaxRetries: 0})
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.Complete(context.Background(), openAIRequest())
	if err != nil {
		t.Fatal(err)
	}
	if string(got.JSON) != `{"ok":true}` || got.EffectiveModel != "gpt-effective" || *got.InputTokens != 3 || *got.OutputTokens != 2 {
		t.Fatalf("response = %+v", got)
	}
}

func TestCompleteRejectsRefusalMalformedIncompleteAndHTTPFailuresWithoutRetry(t *testing.T) {
	for _, tc := range []struct {
		name, response string
		status         int
		want           ports.ReasoningFailureKind
	}{
		{"refusal", `{"id":"r","object":"response","status":"completed","model":"m","output":[{"type":"message","content":[{"type":"refusal","refusal":"no"}]}]}`, http.StatusOK, ports.ReasoningDeclined},
		{"malformed", `{"id":"r","object":"response","status":"completed","model":"m","output":[{"type":"message","content":[{"type":"output_text","text":"not-json"}]}]}`, http.StatusOK, ports.ReasoningInvalidOutput},
		{"incomplete", `{"id":"r","object":"response","status":"incomplete","model":"m","incomplete_details":{"reason":"max_output_tokens"}}`, http.StatusOK, ports.ReasoningIncomplete},
		{"unauthorized", `{"error":{"message":"bad key"}}`, http.StatusUnauthorized, ports.ReasoningUnauthorized},
		{"rate-limited", `{"error":{"message":"slow down"}}`, http.StatusTooManyRequests, ports.ReasoningRateLimited},
		{"server-fault", `{"error":{"message":"boom"}}`, http.StatusBadGateway, ports.ReasoningUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				// The SDK only parses a JSON body when it is announced as
				// JSON. Without this header it fails at the transport layer,
				// which is what previously let the refusal/malformed/
				// incomplete rows "pass" while never reaching the code they
				// name.
				w.Header().Set("content-type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.response))
			}))
			defer server.Close()
			client, err := New(Config{APIKey: "test-key", BaseURL: server.URL + "/v1", HTTPClient: server.Client(), MaxRetries: 0})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Complete(context.Background(), openAIRequest())
			if got := reasoningKind(t, err); got != tc.want {
				t.Fatalf("kind = %q, want %q (err = %v)", got, tc.want, err)
			}
			if calls != 1 {
				t.Fatalf("calls = %d, want one with retries disabled", calls)
			}
		})
	}
}

// A client-side deadline and an owner cancellation both surface through the
// same transport error, and Go's wording for them depends on which timeout
// mechanism fires first. Asserting the classification rather than the message
// is what makes these deterministic: the previous string match on "deadline
// exceeded" failed 5 of 8 runs on this adapter because http.Client.Timeout and
// the SDK's own request deadline race to produce different text.
func TestCompleteClassifiesClientDeadlineAsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL + "/v1", HTTPClient: &http.Client{Transport: server.Client().Transport, Timeout: 20 * time.Millisecond}, MaxRetries: 0})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Complete(context.Background(), openAIRequest())
	if got := reasoningKind(t, err); got != ports.ReasoningTimedOut {
		t.Fatalf("kind = %q, want %q (err = %v)", got, ports.ReasoningTimedOut, err)
	}
	if !ports.ReasoningTimedOut.Retryable() {
		t.Fatal("a timeout must remain offerable as a deliberate retry")
	}
}

// Cancellation stays distinct from a timeout: the owner abandoning a call is
// their decision and must never be offered as an automatic retry of a billed
// request.
func TestCompleteClassifiesCallerCancellationAsCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL + "/v1", HTTPClient: server.Client(), MaxRetries: 0})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	defer cancel()
	_, err = client.Complete(ctx, openAIRequest())
	if got := reasoningKind(t, err); got != ports.ReasoningCancelled {
		t.Fatalf("kind = %q, want %q (err = %v)", got, ports.ReasoningCancelled, err)
	}
	if ports.ReasoningCancelled.Retryable() {
		t.Fatal("cancellation must not be automatically retryable")
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

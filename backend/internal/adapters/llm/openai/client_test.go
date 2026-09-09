package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	}{
		{"refusal", `{"id":"r","object":"response","status":"completed","model":"m","output":[{"type":"message","content":[{"type":"refusal","refusal":"no"}]}]}`, http.StatusOK},
		{"malformed", `{"id":"r","object":"response","status":"completed","model":"m","output":[{"type":"message","content":[{"type":"output_text","text":"not-json"}]}]}`, http.StatusOK},
		{"incomplete", `{"id":"r","object":"response","status":"incomplete","model":"m","incomplete_details":{"reason":"max_output_tokens"}}`, http.StatusOK},
		{"unauthorized", `{"error":{"message":"bad key"}}`, http.StatusUnauthorized},
		{"rate-limited", `{"error":{"message":"slow down"}}`, http.StatusTooManyRequests},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.response))
			}))
			defer server.Close()
			client, err := New(Config{APIKey: "test-key", BaseURL: server.URL + "/v1", HTTPClient: server.Client(), MaxRetries: 0})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Complete(context.Background(), openAIRequest()); err == nil {
				t.Fatal("expected error")
			}
			if calls != 1 {
				t.Fatalf("calls = %d, want one with retries disabled", calls)
			}
		})
	}
}

func TestCompleteHonorsCancellationAndTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(200 * time.Millisecond):
		}
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL + "/v1", HTTPClient: &http.Client{Transport: server.Client().Transport, Timeout: 20 * time.Millisecond}, MaxRetries: 0})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Complete(context.Background(), openAIRequest()); err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("error = %v, want deadline", err)
	}
}

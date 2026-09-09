package anthropic

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
	}{
		{"refusal", `{"id":"m","type":"message","role":"assistant","model":"m","content":[],"stop_reason":"refusal","stop_details":{"category":"safety"}}`, http.StatusOK},
		{"malformed", `{"id":"m","type":"message","role":"assistant","model":"m","content":[{"type":"text","text":"not-json"}],"stop_reason":"end_turn"}`, http.StatusOK},
		{"unauthorized", `{"error":{"type":"authentication_error","message":"bad key"}}`, http.StatusUnauthorized},
		{"rate-limited", `{"error":{"type":"rate_limit_error","message":"slow down"}}`, http.StatusTooManyRequests},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.response))
			}))
			defer server.Close()
			client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, HTTPClient: server.Client(), MaxRetries: 0})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Complete(context.Background(), anthropicRequest()); err == nil {
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
	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, HTTPClient: &http.Client{Transport: server.Client().Transport, Timeout: 20 * time.Millisecond}, MaxRetries: 0})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Complete(context.Background(), anthropicRequest()); err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("error = %v, want deadline", err)
	}
}

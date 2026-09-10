package daemon

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	settingssvc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/settings"
)

// The probe is what separates "a credential is stored" from "reasoning works".
// These exercise the real probe against a controlled local endpoint, so the
// whole path — credential header, structured request, reply parsing and
// failure classification — is proved without a live billed provider.
//
// The stand-in is reached only through KENNEL_WALDO_BASE_URL, which is never
// persisted, so no packaged install can be pointed at it.
func TestProbeReasoningAcceptsAWorkingProvider(t *testing.T) {
	var seenAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("authorization")
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp-1","object":"response","created_at":1,"status":"completed","model":"gpt-probe","output":[{"type":"message","id":"m","status":"completed","role":"assistant","content":[{"type":"output_text","text":"{\"ok\":true}","annotations":[]}]}],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	err := probeReasoning(context.Background(), settingssvc.ReasoningConfig{
		Provider: providerOpenAI, APIKey: "probe-key", BaseURL: server.URL + "/v1",
	})
	if err != nil {
		t.Fatalf("probe error = %v, want success", err)
	}
	if seenAuth != "Bearer probe-key" {
		t.Fatalf("authorization = %q, want the configured credential", seenAuth)
	}
}

// A revoked or mistyped key is exactly the case a non-empty-string readiness
// check could never catch, and it must come back classified so the owner is
// told to fix the credential rather than to retry.
func TestProbeReasoningClassifiesARejectedCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	defer server.Close()

	err := probeReasoning(context.Background(), settingssvc.ReasoningConfig{
		Provider: providerOpenAI, APIKey: "revoked", BaseURL: server.URL + "/v1",
	})
	var failure *ports.ReasoningFailure
	if !errors.As(err, &failure) {
		t.Fatalf("probe error is not classified: %v", err)
	}
	if failure.Kind != ports.ReasoningUnauthorized {
		t.Fatalf("kind = %q, want %q", failure.Kind, ports.ReasoningUnauthorized)
	}
	if failure.Kind.Retryable() {
		t.Fatal("a rejected credential must not be offered as a plain retry")
	}
}

// A reply that is not usable structured material is a failed probe, not a
// success: the point is to prove the whole path, including parsing.
func TestProbeReasoningRejectsAnUnusableReply(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id":"r","object":"response","status":"completed","model":"m","output":[{"type":"message","content":[{"type":"output_text","text":"not-json"}]}]}`))
	}))
	defer server.Close()

	err := probeReasoning(context.Background(), settingssvc.ReasoningConfig{
		Provider: providerOpenAI, APIKey: "k", BaseURL: server.URL + "/v1",
	})
	var failure *ports.ReasoningFailure
	if !errors.As(err, &failure) || failure.Kind != ports.ReasoningInvalidOutput {
		t.Fatalf("error = %v, want a classified invalid-output failure", err)
	}
}

// Without a credential there is nothing to probe, and the probe must say so
// rather than reporting a transport problem.
func TestProbeReasoningReportsMissingSetup(t *testing.T) {
	err := probeReasoning(context.Background(), settingssvc.ReasoningConfig{Provider: providerOpenAI})
	var failure *ports.ReasoningFailure
	if !errors.As(err, &failure) || failure.Kind != ports.ReasoningNotConfigured {
		t.Fatalf("error = %v, want a classified not-configured failure", err)
	}
}

// The redirect must be env-only. If it were persistable, a stored setting could
// silently send the owner's credential to another host.
func TestReasoningBaseURLIsEnvironmentOnly(t *testing.T) {
	svc := settingssvc.New(nil, nil, nil).WithReasoningEnvLookup(func(key string) string {
		if key == "KENNEL_WALDO_BASE_URL" {
			return "http://127.0.0.1:9/v1"
		}
		return ""
	})
	if got := svc.ReasoningBaseURL(); got != "http://127.0.0.1:9/v1" {
		t.Fatalf("base URL = %q, want the environment value", got)
	}
	if got := settingssvc.New(nil, nil, nil).ReasoningBaseURL(); got != "" {
		t.Fatalf("base URL = %q, want empty without the environment override", got)
	}
}

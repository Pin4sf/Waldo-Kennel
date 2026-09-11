package settings

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

type allChatHarnesses struct{}

func (allChatHarnesses) SupportsChat(domain.AgentHarness) bool { return true }

func TestChatHarnessesExcludesHistoricalCandidates(t *testing.T) {
	svc := New(nil, allChatHarnesses{}, nil)

	got := svc.ChatHarnesses([]domain.AgentHarness{domain.AgentHarness("aider"), domain.HarnessCodex})
	if len(got) != 1 || got[0] != domain.HarnessCodex {
		t.Fatalf("ChatHarnesses() = %#v, want only Codex", got)
	}
}

type reasoningSettingsStore struct{ snapshot Snapshot }

func (s *reasoningSettingsStore) GetAppSettings(context.Context) (Snapshot, error) {
	return s.snapshot, nil
}
func (s *reasoningSettingsStore) SetDefaultSessionMode(context.Context, domain.SessionMode, time.Time) error {
	return nil
}
func (s *reasoningSettingsStore) SetReasoningSettings(_ context.Context, provider, model, effort string, _ time.Time) error {
	s.snapshot.ReasoningProvider, s.snapshot.ReasoningModel, s.snapshot.ReasoningEffort = provider, model, effort
	s.snapshot.ReasoningGeneration++
	return nil
}

func (s *reasoningSettingsStore) SetReasoningVerification(_ context.Context, verifiedAt *time.Time, provider, model string, _ time.Time) error {
	s.snapshot.ReasoningVerifiedAt = verifiedAt
	s.snapshot.ReasoningVerifiedProvider, s.snapshot.ReasoningVerifiedModel = provider, model
	return nil
}

func (s *reasoningSettingsStore) SetReasoningVerificationForGeneration(_ context.Context, verifiedAt *time.Time, provider, model string, generation int64, fingerprint string, _ time.Time) (bool, error) {
	if s.snapshot.ReasoningGeneration != generation {
		return false, nil
	}
	s.snapshot.ReasoningVerifiedAt = verifiedAt
	s.snapshot.ReasoningVerifiedProvider, s.snapshot.ReasoningVerifiedModel = provider, model
	s.snapshot.ReasoningVerifiedGeneration, s.snapshot.ReasoningVerificationFingerprint = generation, fingerprint
	return true, nil
}

type reasoningSecret struct{ value string }

func (s *reasoningSecret) Get(context.Context) (string, error) { return s.value, nil }
func (s *reasoningSecret) Set(_ context.Context, value string) error {
	s.value = value
	return nil
}
func (s *reasoningSecret) Clear(context.Context) error { s.value = ""; return nil }

type providerReasoningSecret struct{ values map[string]string }

func (s *providerReasoningSecret) Get(context.Context) (string, error) {
	return "", nil
}
func (s *providerReasoningSecret) Set(_ context.Context, value string) error {
	s.values["legacy"] = value
	return nil
}
func (s *providerReasoningSecret) Clear(context.Context) error {
	delete(s.values, "legacy")
	return nil
}
func (s *providerReasoningSecret) GetForProvider(_ context.Context, provider string) (string, error) {
	return s.values[provider], nil
}
func (s *providerReasoningSecret) SetForProvider(_ context.Context, provider, value string) error {
	s.values[provider] = value
	return nil
}
func (s *providerReasoningSecret) ClearForProvider(_ context.Context, provider string) error {
	delete(s.values, provider)
	return nil
}

func TestResolveReasoningUsesPersistedSelectionAndDaemonSecret(t *testing.T) {
	store := &reasoningSettingsStore{snapshot: Snapshot{ReasoningProvider: "openai", ReasoningModel: "gpt-test", ReasoningEffort: "medium"}}
	secret := &reasoningSecret{value: "local-secret"}
	svc := New(store, nil, nil).WithReasoningSecrets(secret).WithReasoningEnvLookup(func(string) string { return "" })

	cfg, err := svc.ResolveReasoning(context.Background())
	if err != nil {
		t.Fatalf("ResolveReasoning() error = %v", err)
	}
	if cfg.Provider != "openai" || cfg.Model != "gpt-test" || cfg.Effort != "medium" || cfg.APIKey != "local-secret" {
		t.Fatalf("resolved config = %#v", cfg)
	}
	status, err := svc.GetReasoning(context.Background())
	if err != nil || !status.Ready || !status.KeyConfigured {
		t.Fatalf("reasoning status = %#v, err=%v", status, err)
	}
}

func TestReasoningEnvironmentOverridesDoNotLeakIntoStatus(t *testing.T) {
	store := &reasoningSettingsStore{}
	secret := &reasoningSecret{}
	values := map[string]string{
		"KENNEL_WALDO_PROVIDER": "anthropic",
		"KENNEL_WALDO_MODEL":    "claude-test",
		"KENNEL_WALDO_EFFORT":   "high",
		"KENNEL_WALDO_API_KEY":  "env-secret-canary",
	}
	svc := New(store, nil, nil).WithReasoningSecrets(secret).WithReasoningEnvLookup(func(name string) string { return values[name] })
	status, err := svc.GetReasoning(context.Background())
	if err != nil || !status.Ready || status.Provider != "anthropic" || status.Model != "claude-test" || status.Effort != "high" {
		t.Fatalf("reasoning status = %#v, err=%v", status, err)
	}
	if status.Error != "" || status.KeyConfigured != true {
		t.Fatalf("unexpected status error or key state = %#v", status)
	}
	encoded, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal reasoning status: %v", err)
	}
	if strings.Contains(string(encoded), "env-secret-canary") {
		t.Fatal("reasoning status leaked the API key")
	}
}

func TestGetReasoningReportsActionableMissingCredential(t *testing.T) {
	store := &reasoningSettingsStore{snapshot: Snapshot{ReasoningProvider: "anthropic"}}
	svc := New(store, nil, nil).WithReasoningSecrets(&reasoningSecret{}).WithReasoningEnvLookup(func(string) string { return "" })
	status, err := svc.GetReasoning(context.Background())
	if err != nil || status.Ready || status.ErrorCode != "MISSING_CREDENTIAL" || status.KeyConfigured {
		t.Fatalf("missing credential status = %#v, err=%v", status, err)
	}
}

func TestSetReasoningDoesNotReuseCredentialWhenProviderChanges(t *testing.T) {
	store := &reasoningSettingsStore{snapshot: Snapshot{ReasoningProvider: "anthropic"}}
	secret := &providerReasoningSecret{values: map[string]string{}}
	svc := New(store, nil, nil).WithReasoningSecrets(secret).WithReasoningEnvLookup(func(string) string { return "" })

	if _, err := svc.SetReasoning(context.Background(), ReasoningInput{Provider: "anthropic", APIKey: "anthropic-canary"}); err != nil {
		t.Fatalf("set Anthropic reasoning: %v", err)
	}
	status, err := svc.SetReasoning(context.Background(), ReasoningInput{Provider: "openai"})
	if err != nil {
		t.Fatalf("switch provider: %v", err)
	}
	if status.Ready || status.KeyConfigured || status.ErrorCode != "MISSING_CREDENTIAL" {
		t.Fatalf("provider switch status = %#v, want actionable missing OpenAI credential", status)
	}
	if got := secret.values["openai"]; got != "" {
		t.Fatalf("OpenAI credential unexpectedly populated from Anthropic key: %q", got)
	}
}

func TestCodexReasoningUsesHarnessReadinessInsteadOfAnAPIKey(t *testing.T) {
	store := &reasoningSettingsStore{snapshot: Snapshot{ReasoningProvider: "codex", ReasoningModel: "gpt-test", ReasoningEffort: "high"}}
	available := false
	svc := New(store, nil, nil).
		WithReasoningEnvLookup(func(name string) string {
			if name == "KENNEL_WALDO_API_KEY" {
				return "ambient-api-key-must-not-enter-codex-mode"
			}
			return ""
		}).
		WithReasoningAvailability(func(context.Context, ReasoningConfig) error {
			if available {
				return nil
			}
			return ports.NewReasoningFailure(ports.ReasoningUnavailable, "Codex app-server is unavailable", nil)
		})

	status, err := svc.GetReasoning(context.Background())
	if err != nil || status.Mode != "codex_harness" || status.Provider != "codex" || !status.Configured || status.Ready || status.KeyConfigured || status.ErrorCode != "REASONING_NOT_READY" {
		t.Fatalf("unavailable Codex status = %#v, err=%v", status, err)
	}
	cfg, err := svc.ResolveReasoning(context.Background())
	if err != nil || cfg.APIKey != "" || cfg.KeySource != "codex-app-server-sign-in" {
		t.Fatalf("Codex reasoning resolved an API key: cfg=%#v err=%v", cfg, err)
	}
	available = true
	status, err = svc.GetReasoning(context.Background())
	if err != nil || !status.Ready || status.KeyConfigured || status.Mode != "codex_harness" {
		t.Fatalf("available Codex status = %#v, err=%v", status, err)
	}
}

func TestSetReasoningCodexDoesNotRequireOrAcceptAnAPIKey(t *testing.T) {
	store := &reasoningSettingsStore{}
	svc := New(store, nil, nil).WithReasoningEnvLookup(func(string) string { return "" })
	status, err := svc.SetReasoning(context.Background(), ReasoningInput{Provider: "codex", Model: "provider-default"})
	if err != nil {
		t.Fatalf("set Codex reasoning: %v", err)
	}
	if status.Provider != "codex" || status.Mode != "codex_harness" || status.KeyConfigured {
		t.Fatalf("Codex selection = %#v", status)
	}
	if _, err := svc.SetReasoning(context.Background(), ReasoningInput{Provider: "codex", APIKey: "must-not-be-used"}); err == nil {
		t.Fatal("Codex harness unexpectedly accepted an API key")
	}
}

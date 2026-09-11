package daemon

import (
	"strings"
	"testing"

	llmanthropic "github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/llm/anthropic"
	llmopenai "github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/llm/openai"
)

// env builds a lookup over a fixed set of variables. Reading through a
// function rather than the process environment keeps these cases independent
// of whatever keys the developer running the suite happens to have exported.
func env(pairs map[string]string) func(string) string {
	return func(name string) string { return pairs[name] }
}

func TestResolveReasoningConfigChoosesTheProviderTheOwnerHasAKeyFor(t *testing.T) {
	for _, tc := range []struct {
		name         string
		vars         map[string]string
		wantProvider string
		wantKey      string
		wantSource   string
	}{
		{
			// The pre-existing contract from ADR 0012, unchanged by the second
			// adapter: a lone KENNEL_WALDO_API_KEY still means Anthropic.
			name:         "shared key alone stays on anthropic",
			vars:         map[string]string{"KENNEL_WALDO_API_KEY": "k-shared"},
			wantProvider: providerAnthropic,
			wantKey:      "k-shared",
			wantSource:   "KENNEL_WALDO_API_KEY",
		},
		{
			name:         "anthropic key alone",
			vars:         map[string]string{"ANTHROPIC_API_KEY": "k-ant"},
			wantProvider: providerAnthropic,
			wantKey:      "k-ant",
			wantSource:   "ANTHROPIC_API_KEY",
		},
		{
			name:         "openai key alone",
			vars:         map[string]string{"OPENAI_API_KEY": "k-oai"},
			wantProvider: providerOpenAI,
			wantKey:      "k-oai",
			wantSource:   "OPENAI_API_KEY",
		},
		{
			// Plenty of developers have ANTHROPIC_API_KEY exported for other
			// tools. Without a stated preference it wins, so nothing changes
			// under someone who adds an OpenAI key for an unrelated reason.
			name:         "both keys, no preference, anthropic wins",
			vars:         map[string]string{"ANTHROPIC_API_KEY": "k-ant", "OPENAI_API_KEY": "k-oai"},
			wantProvider: providerAnthropic,
			wantKey:      "k-ant",
			wantSource:   "ANTHROPIC_API_KEY",
		},
		{
			name: "explicit openai overrides an ambient anthropic key",
			vars: map[string]string{
				"KENNEL_WALDO_PROVIDER": "openai",
				"ANTHROPIC_API_KEY":     "k-ant",
				"OPENAI_API_KEY":        "k-oai",
			},
			wantProvider: providerOpenAI,
			wantKey:      "k-oai",
			wantSource:   "OPENAI_API_KEY",
		},
		{
			name: "explicit anthropic overrides an ambient openai key",
			vars: map[string]string{
				"KENNEL_WALDO_PROVIDER": "anthropic",
				"OPENAI_API_KEY":        "k-oai",
				"ANTHROPIC_API_KEY":     "k-ant",
			},
			wantProvider: providerAnthropic,
			wantKey:      "k-ant",
			wantSource:   "ANTHROPIC_API_KEY",
		},
		{
			// The point of the dedicated variable: keep Waldo's reasoning
			// credential separate from the one the coding agents use.
			name: "shared key is preferred over the provider's own",
			vars: map[string]string{
				"KENNEL_WALDO_PROVIDER": "openai",
				"KENNEL_WALDO_API_KEY":  "k-waldo",
				"OPENAI_API_KEY":        "k-oai",
			},
			wantProvider: providerOpenAI,
			wantKey:      "k-waldo",
			wantSource:   "KENNEL_WALDO_API_KEY",
		},
		{
			name:         "provider name is case and space insensitive",
			vars:         map[string]string{"KENNEL_WALDO_PROVIDER": "  OpenAI ", "OPENAI_API_KEY": "k-oai"},
			wantProvider: providerOpenAI,
			wantKey:      "k-oai",
			wantSource:   "OPENAI_API_KEY",
		},
		{
			name:         "explicit Codex harness does not require an API key",
			vars:         map[string]string{"KENNEL_WALDO_PROVIDER": " Codex ", "KENNEL_WALDO_MODEL": "gpt-test"},
			wantProvider: providerCodex,
			wantSource:   "codex-app-server-sign-in",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := resolveReasoningConfig(env(tc.vars))
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if cfg.Provider != tc.wantProvider {
				t.Errorf("provider = %q, want %q", cfg.Provider, tc.wantProvider)
			}
			if cfg.APIKey != tc.wantKey {
				t.Errorf("key = %q, want %q", cfg.APIKey, tc.wantKey)
			}
			if cfg.KeySource != tc.wantSource {
				t.Errorf("key source = %q, want %q", cfg.KeySource, tc.wantSource)
			}
		})
	}
}

func TestResolveReasoningConfigCarriesModelAndEffort(t *testing.T) {
	cfg, err := resolveReasoningConfig(env(map[string]string{
		"KENNEL_WALDO_PROVIDER": "openai",
		"OPENAI_API_KEY":        "k-oai",
		"KENNEL_WALDO_MODEL":    " gpt-5.5 ",
		"KENNEL_WALDO_EFFORT":   " high ",
	}))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.Model != "gpt-5.5" {
		t.Errorf("model = %q, want %q", cfg.Model, "gpt-5.5")
	}
	if cfg.Effort != "high" {
		t.Errorf("effort = %q, want %q", cfg.Effort, "high")
	}
}

func TestResolveReasoningConfigRejectsAnUnknownProvider(t *testing.T) {
	// Silently falling back would send the owner's key somewhere they did not
	// name. Naming the two supported values is the whole value of the error.
	_, err := resolveReasoningConfig(env(map[string]string{
		"KENNEL_WALDO_PROVIDER": "gemini",
		"KENNEL_WALDO_API_KEY":  "k-shared",
	}))
	if err == nil {
		t.Fatal("expected an error for an unsupported provider")
	}
	for _, want := range []string{"gemini", providerAnthropic, providerOpenAI, providerCodex} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q: %v", want, err)
		}
	}
}

func TestResolveReasoningConfigSaysWhichKeyIsMissing(t *testing.T) {
	// ADR 0012: with no credential Waldo fails and says why, rather than
	// degrading to a canned proposal. The message has to name the variable the
	// owner should set for the provider they asked for.
	_, err := resolveReasoningConfig(env(map[string]string{"KENNEL_WALDO_PROVIDER": "openai"}))
	if err == nil {
		t.Fatal("expected an error when no key is configured")
	}
	if !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Errorf("error should name OPENAI_API_KEY: %v", err)
	}
	if strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("error should not send an OpenAI owner to the Anthropic variable: %v", err)
	}

	_, err = resolveReasoningConfig(env(nil))
	if err == nil {
		t.Fatal("expected an error when nothing at all is configured")
	}
	if !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("unconfigured install should name the default provider's variable: %v", err)
	}
}

func TestNewReasonerBuildsTheChosenAdapter(t *testing.T) {
	openaiClient, err := newReasoner(reasoningConfig{Provider: providerOpenAI, APIKey: "k-oai"})
	if err != nil {
		t.Fatalf("openai: %v", err)
	}
	if openaiClient.ID() != llmopenai.ProviderID {
		t.Errorf("provider id = %q, want %q", openaiClient.ID(), llmopenai.ProviderID)
	}

	anthropicClient, err := newReasoner(reasoningConfig{Provider: providerAnthropic, APIKey: "k-ant"})
	if err != nil {
		t.Fatalf("anthropic: %v", err)
	}
	if anthropicClient.ID() != llmanthropic.ProviderID {
		t.Errorf("provider id = %q, want %q", anthropicClient.ID(), llmanthropic.ProviderID)
	}
}

func TestNewReasonerReturnsATrulyNilClientWhenUnconfigured(t *testing.T) {
	// A typed nil pointer in an interface is non-nil, which would make the
	// daemon's `reasoner != nil` guard wire an unusable provider instead of
	// warning that reasoning is unconfigured.
	for _, provider := range []string{providerAnthropic, providerOpenAI} {
		client, err := newReasoner(reasoningConfig{Provider: provider})
		if err == nil {
			t.Fatalf("%s: expected an error with no key", provider)
		}
		if client != nil {
			t.Errorf("%s: client must be nil, got %#v", provider, client)
		}
	}
}

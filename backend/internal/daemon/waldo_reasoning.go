package daemon

import (
	"context"
	"fmt"
	"strings"
	"time"

	llmanthropic "github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/llm/anthropic"
	llmopenai "github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/llm/openai"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	intelligencesvc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/intelligence"
	settingssvc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/settings"
)

// Waldo reasons with the owner's own model (ADR 0012). Which provider that is
// belongs to the owner too: the reasoning port is provider-neutral, so
// supporting a second one is a wiring decision, not an architectural one.
const (
	providerAnthropic = "anthropic"
	providerOpenAI    = "openai"
)

// reasoningConfig is the resolved answer to "whose model, and with what key".
type reasoningConfig struct {
	Provider string
	APIKey   string
	Model    string
	Effort   string
	// KeySource names the environment variable the key came from, for a log
	// line that tells the owner which of several keys Waldo actually picked up.
	KeySource string
	// BaseURL redirects reasoning at a local stand-in. It is a development and
	// test seam only: it comes from KENNEL_WALDO_BASE_URL and is never
	// persisted in settings, so a packaged install always talks to the real
	// provider. It exists so the reasoning path — credential handling,
	// classification, readiness — can be exercised against a controlled local
	// endpoint instead of a live billed provider.
	BaseURL string
}

// resolveReasoningConfig decides which provider Waldo thinks with.
//
// An explicit KENNEL_WALDO_PROVIDER always wins. Otherwise the choice follows
// whichever credential is present, and Anthropic keeps precedence so an
// install that worked before this adapter existed keeps behaving identically.
func resolveReasoningConfig(lookup func(string) string) (reasoningConfig, error) {
	cfg := reasoningConfig{
		Model:   strings.TrimSpace(lookup("KENNEL_WALDO_MODEL")),
		Effort:  strings.TrimSpace(lookup("KENNEL_WALDO_EFFORT")),
		BaseURL: strings.TrimSpace(lookup("KENNEL_WALDO_BASE_URL")),
	}

	sharedKey := strings.TrimSpace(lookup("KENNEL_WALDO_API_KEY"))
	anthropicKey := strings.TrimSpace(lookup("ANTHROPIC_API_KEY"))
	openaiKey := strings.TrimSpace(lookup("OPENAI_API_KEY"))

	switch requested := strings.ToLower(strings.TrimSpace(lookup("KENNEL_WALDO_PROVIDER"))); requested {
	case providerAnthropic:
		cfg.Provider = providerAnthropic
	case providerOpenAI:
		cfg.Provider = providerOpenAI
	case "":
		// No stated preference: follow the key the owner actually has.
		switch {
		case sharedKey != "", anthropicKey != "":
			cfg.Provider = providerAnthropic
		case openaiKey != "":
			cfg.Provider = providerOpenAI
		default:
			// Nothing is configured. Name the historical default so the
			// unconfigured-key error below reads as one clear problem.
			cfg.Provider = providerAnthropic
		}
	default:
		return reasoningConfig{}, fmt.Errorf(
			"KENNEL_WALDO_PROVIDER is %q; Waldo reasons with %q or %q", requested, providerAnthropic, providerOpenAI)
	}

	// KENNEL_WALDO_API_KEY is preferred for both providers so an owner can keep
	// Waldo's reasoning credential separate from the key their coding agents
	// already use. The provider's own variable is the familiar fallback.
	fallback := "ANTHROPIC_API_KEY"
	fallbackKey := anthropicKey
	if cfg.Provider == providerOpenAI {
		fallback, fallbackKey = "OPENAI_API_KEY", openaiKey
	}
	switch {
	case sharedKey != "":
		cfg.APIKey, cfg.KeySource = sharedKey, "KENNEL_WALDO_API_KEY"
	case fallbackKey != "":
		cfg.APIKey, cfg.KeySource = fallbackKey, fallback
	default:
		return cfg, fmt.Errorf("no reasoning key for %s: set KENNEL_WALDO_API_KEY or %s", cfg.Provider, fallback)
	}
	return cfg, nil
}

// newReasoner builds Waldo's reasoning client for the resolved provider.
//
// The key belongs to the user and stays on this machine: ADR 0003 rules out a
// Waldo-operated LLM API and Waldo-funded inference, not Waldo reasoning with
// the owner's own authenticated provider.
// Each branch returns an explicit nil interface on failure. Handing back the
// adapter's typed nil pointer instead would produce a non-nil interface value
// holding nil, and every `reasoner != nil` check downstream would be wrong.
func newReasoner(cfg reasoningConfig) (ports.LLMClient, error) {
	switch cfg.Provider {
	case providerOpenAI:
		client, err := llmopenai.New(llmopenai.Config{APIKey: cfg.APIKey, Model: cfg.Model, Effort: cfg.Effort, BaseURL: cfg.BaseURL, MaxRetries: 0})
		if err != nil {
			return nil, err
		}
		return client, nil
	default:
		client, err := llmanthropic.New(llmanthropic.Config{APIKey: cfg.APIKey, Model: cfg.Model, Effort: cfg.Effort, BaseURL: cfg.BaseURL, MaxRetries: 0})
		if err != nil {
			return nil, err
		}
		return client, nil
	}
}

// configuredIntelligenceProvider resolves settings at call time. A fresh
// profile can therefore configure a credential and retry without restarting
// the daemon; the renderer never receives the credential.
type configuredIntelligenceProvider struct{ settings *settingssvc.Service }

var _ ports.IntelligenceProvider = (*configuredIntelligenceProvider)(nil)

func newConfiguredIntelligenceProvider(settings *settingssvc.Service) *configuredIntelligenceProvider {
	return &configuredIntelligenceProvider{settings: settings}
}

func (*configuredIntelligenceProvider) ID() domain.IntelligenceProviderID {
	return intelligencesvc.LLMProviderID
}

func (p *configuredIntelligenceProvider) client(ctx context.Context) (ports.LLMClient, error) {
	if p == nil || p.settings == nil {
		return nil, fmt.Errorf("reasoning settings are unavailable")
	}
	cfg, err := p.settings.ResolveReasoning(ctx)
	if err != nil {
		return nil, err
	}
	return newReasoner(reasoningConfig{
		Provider: cfg.Provider, APIKey: cfg.APIKey, Model: cfg.Model, Effort: cfg.Effort,
		BaseURL: p.settings.ReasoningBaseURL(),
	})
}

func (p *configuredIntelligenceProvider) AnalyzeContract(ctx context.Context, request ports.ContractIntelligenceRequest) (ports.ContractIntelligenceResponse, error) {
	client, err := p.client(ctx)
	if err != nil {
		return ports.ContractIntelligenceResponse{}, err
	}
	return intelligencesvc.NewLLMProvider(client).AnalyzeContract(ctx, request)
}

func (p *configuredIntelligenceProvider) DraftPlan(ctx context.Context, request ports.PlanIntelligenceRequest) (ports.PlanIntelligenceResponse, error) {
	client, err := p.client(ctx)
	if err != nil {
		return ports.PlanIntelligenceResponse{}, err
	}
	return intelligencesvc.NewLLMProvider(client).DraftPlan(ctx, request)
}

// probeReasoningBudget bounds one readiness probe. Named operational policy: a
// probe that hangs must not hold the settings request open.
const probeReasoningBudget = 30 * time.Second

// probeReasoning performs the smallest real reasoning call that still proves
// the whole path works: credential accepted, model reachable, and a structured
// reply that parses.
//
// It exists because "a credential is stored" is not evidence that reasoning
// works — a key can be revoked, mistyped, or lack access to the selected model,
// and reporting that as ready sends the owner into a Plan proposal that then
// fails at the provider. This is the only thing that may set verified state,
// and it runs only when the owner asks, because the call may be billed.
func probeReasoning(ctx context.Context, cfg settingssvc.ReasoningConfig) error {
	client, err := newReasoner(reasoningConfig{
		Provider: cfg.Provider, APIKey: cfg.APIKey, Model: cfg.Model, Effort: cfg.Effort,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, probeReasoningBudget)
	defer cancel()
	// A trivial fixed schema, so the probe measures the provider path and not
	// the model's ability to handle a hard request.
	_, err = client.Complete(ctx, ports.LLMRequest{
		System:     "Reply with the requested JSON object and nothing else.",
		User:       "Return {\"ok\": true}.",
		SchemaName: "readiness_probe",
		MaxTokens:  256,
		Schema: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{"ok": map[string]any{"type": "boolean"}},
			"required":             []any{"ok"},
			"additionalProperties": false,
		},
	})
	return err
}

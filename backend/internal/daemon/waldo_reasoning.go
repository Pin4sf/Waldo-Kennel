package daemon

import (
	"fmt"
	"os"
	"strings"

	llmanthropic "github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/llm/anthropic"
	llmopenai "github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/llm/openai"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
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
}

// resolveReasoningConfig decides which provider Waldo thinks with.
//
// An explicit KENNEL_WALDO_PROVIDER always wins. Otherwise the choice follows
// whichever credential is present, and Anthropic keeps precedence so an
// install that worked before this adapter existed keeps behaving identically.
func resolveReasoningConfig(lookup func(string) string) (reasoningConfig, error) {
	cfg := reasoningConfig{
		Model:  strings.TrimSpace(lookup("KENNEL_WALDO_MODEL")),
		Effort: strings.TrimSpace(lookup("KENNEL_WALDO_EFFORT")),
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
		client, err := llmopenai.New(llmopenai.Config{APIKey: cfg.APIKey, Model: cfg.Model, Effort: cfg.Effort})
		if err != nil {
			return nil, err
		}
		return client, nil
	default:
		client, err := llmanthropic.New(llmanthropic.Config{APIKey: cfg.APIKey, Model: cfg.Model, Effort: cfg.Effort})
		if err != nil {
			return nil, err
		}
		return client, nil
	}
}

// waldoReasoner resolves the owner's provider and builds its client. A nil
// client is returned with an explanatory error when nothing is configured;
// intake then fails retryably and says why, rather than serving a canned
// proposal (ADR 0012).
func waldoReasoner() (ports.LLMClient, reasoningConfig, error) {
	cfg, err := resolveReasoningConfig(os.Getenv)
	if err != nil {
		return nil, cfg, err
	}
	client, err := newReasoner(cfg)
	if err != nil {
		return nil, cfg, err
	}
	return client, cfg, nil
}

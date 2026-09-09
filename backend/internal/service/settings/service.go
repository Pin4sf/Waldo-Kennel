// Package settings owns Kennel's daemon-side user preferences.
//
// It exists so every spawn surface — desktop, mobile, `kennel spawn`, headless —
// resolves one value. A renderer-held preference would look correct in Settings
// while disagreeing with the CLI, which is worse than having no control.
package settings

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// Store is the durable preference surface.
type Store interface {
	GetAppSettings(ctx context.Context) (Snapshot, error)
	SetDefaultSessionMode(ctx context.Context, mode domain.SessionMode, now time.Time) error
	SetReasoningSettings(ctx context.Context, provider, model, effort string, now time.Time) error
}

// Snapshot is the current preference set.
type Snapshot struct {
	DefaultSessionMode domain.SessionMode
	ReasoningProvider  string
	ReasoningModel     string
	ReasoningEffort    string
	UpdatedAt          time.Time
}

// SecretStore is intentionally narrower than a general credential manager.
// Settings can report whether a credential exists, but never return its value.
type SecretStore interface {
	Get(context.Context) (string, error)
	Set(context.Context, string) error
	Clear(context.Context) error
}

type ReasoningInput struct {
	Provider string
	Model    string
	Effort   string
	APIKey   string
	ClearKey bool
}

type ReasoningStatus struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	Effort        string `json:"effort"`
	Configured    bool   `json:"configured"`
	Ready         bool   `json:"ready"`
	KeyConfigured bool   `json:"keyConfigured"`
	ErrorCode     string `json:"errorCode,omitempty"`
	Error         string `json:"error,omitempty"`
}

type ReasoningConfig struct {
	Provider  string
	APIKey    string
	Model     string
	Effort    string
	KeySource string
}

// ChatCapability reports which harnesses can run in chat mode, so the UI can warn
// that choosing chat narrows the agents available rather than letting the user
// discover it at spawn time.
type ChatCapability interface {
	SupportsChat(harness domain.AgentHarness) bool
}

// Service reads and writes preferences.
type Service struct {
	store   Store
	chat    ChatCapability
	now     func() time.Time
	secrets SecretStore
	lookup  func(string) string
}

// New builds the service.
func New(store Store, chat ChatCapability, now func() time.Time) *Service {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, chat: chat, now: now, lookup: os.Getenv}
}

// WithReasoningSecrets adds the daemon-owned secret boundary used by Waldo's
// reasoning configuration. Without it, settings remain readable but cannot
// claim that reasoning is configured.
func (s *Service) WithReasoningSecrets(secrets SecretStore) *Service { s.secrets = secrets; return s }

// WithReasoningEnvLookup makes precedence testable without inheriting a shell
// credential into a packaged launch.
func (s *Service) WithReasoningEnvLookup(lookup func(string) string) *Service {
	if lookup != nil {
		s.lookup = lookup
	}
	return s
}

// Get returns the current preferences.
func (s *Service) Get(ctx context.Context) (Snapshot, error) {
	return s.store.GetAppSettings(ctx)
}

func (s *Service) GetReasoning(ctx context.Context) (ReasoningStatus, error) {
	cfg, err := s.ResolveReasoning(ctx)
	status := ReasoningStatus{}
	if err != nil {
		if cfg.Provider != "" {
			status.Provider = cfg.Provider
		}
		status.Model, status.Effort = cfg.Model, cfg.Effort
		status.KeyConfigured = cfg.APIKey != ""
		status.Configured = status.Provider != "" && status.KeyConfigured
		status.ErrorCode, status.Error = reasoningError(err)
		return status, nil
	}
	status = ReasoningStatus{Provider: cfg.Provider, Model: cfg.Model, Effort: cfg.Effort, KeyConfigured: cfg.APIKey != "", Configured: true, Ready: true}
	return status, nil
}

func (s *Service) SetReasoning(ctx context.Context, input ReasoningInput) (ReasoningStatus, error) {
	provider := strings.ToLower(strings.TrimSpace(input.Provider))
	if provider != "anthropic" && provider != "openai" {
		return ReasoningStatus{}, fmt.Errorf("reasoning provider must be anthropic or openai")
	}
	if s.secrets == nil {
		return ReasoningStatus{}, fmt.Errorf("local reasoning secret store is unavailable")
	}
	if input.ClearKey {
		if err := s.secrets.Clear(ctx); err != nil {
			return ReasoningStatus{}, err
		}
	} else if strings.TrimSpace(input.APIKey) != "" {
		if err := s.secrets.Set(ctx, input.APIKey); err != nil {
			return ReasoningStatus{}, err
		}
	}
	if err := s.store.SetReasoningSettings(ctx, provider, strings.TrimSpace(input.Model), strings.TrimSpace(input.Effort), s.now()); err != nil {
		return ReasoningStatus{}, err
	}
	return s.GetReasoning(ctx)
}

// ResolveReasoning applies explicit development environment overrides over the
// persisted non-secret selection and daemon-owned local credential.
func (s *Service) ResolveReasoning(ctx context.Context) (ReasoningConfig, error) {
	snapshot, err := s.store.GetAppSettings(ctx)
	if err != nil {
		return ReasoningConfig{}, err
	}
	key := ""
	if s.secrets != nil {
		key, err = s.secrets.Get(ctx)
		if err != nil {
			return ReasoningConfig{}, err
		}
	}
	lookup := s.lookup
	provider := strings.ToLower(strings.TrimSpace(lookup("KENNEL_WALDO_PROVIDER")))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(snapshot.ReasoningProvider))
	}
	model := strings.TrimSpace(lookup("KENNEL_WALDO_MODEL"))
	if model == "" {
		model = strings.TrimSpace(snapshot.ReasoningModel)
	}
	effort := strings.TrimSpace(lookup("KENNEL_WALDO_EFFORT"))
	if effort == "" {
		effort = strings.TrimSpace(snapshot.ReasoningEffort)
	}
	shared := strings.TrimSpace(lookup("KENNEL_WALDO_API_KEY"))
	ant := strings.TrimSpace(lookup("ANTHROPIC_API_KEY"))
	oai := strings.TrimSpace(lookup("OPENAI_API_KEY"))
	if provider == "" {
		switch {
		case shared != "", ant != "":
			provider = "anthropic"
		case oai != "":
			provider = "openai"
		}
	}
	keySource := "local-secret-store"
	if shared != "" {
		key, keySource = shared, "KENNEL_WALDO_API_KEY"
	} else if provider == "anthropic" && ant != "" {
		key, keySource = ant, "ANTHROPIC_API_KEY"
	} else if provider == "openai" && oai != "" {
		key, keySource = oai, "OPENAI_API_KEY"
	}
	cfg := ReasoningConfig{Provider: provider, APIKey: key, Model: model, Effort: effort, KeySource: keySource}
	if provider == "" {
		return cfg, fmt.Errorf("reasoning provider is not configured; choose anthropic or openai")
	}
	if provider != "anthropic" && provider != "openai" {
		return cfg, fmt.Errorf("unsupported reasoning provider %q", provider)
	}
	if key == "" {
		return cfg, fmt.Errorf("reasoning credential is not configured for %s", provider)
	}
	return cfg, nil
}

func reasoningError(err error) (string, string) {
	message := err.Error()
	switch {
	case strings.Contains(message, "credential"):
		return "MISSING_CREDENTIAL", message
	case strings.Contains(message, "provider"):
		return "PROVIDER_NOT_READY", message
	default:
		return "REASONING_NOT_READY", message
	}
}

// DefaultSessionMode resolves the default for a spawn that named no mode. A read
// failure falls back to the compatibility default rather than failing the spawn:
// an unreadable preference should not stop work.
func (s *Service) DefaultSessionMode(ctx context.Context) domain.SessionMode {
	snapshot, err := s.store.GetAppSettings(ctx)
	if err != nil {
		return domain.DefaultSessionMode
	}
	return domain.NormalizeSessionMode(snapshot.DefaultSessionMode)
}

// SetDefaultSessionMode changes the default for sessions created afterwards.
//
// It deliberately does not touch existing sessions or their controllers. This
// is a preference for future sessions, not an implicit migration of the present.
func (s *Service) SetDefaultSessionMode(ctx context.Context, mode domain.SessionMode) (Snapshot, error) {
	if !mode.Valid() {
		return Snapshot{}, fmt.Errorf("%w: %q", ports.ErrChatUnsupported, mode)
	}
	if err := s.store.SetDefaultSessionMode(ctx, mode, s.now()); err != nil {
		return Snapshot{}, err
	}
	return s.store.GetAppSettings(ctx)
}

// ChatHarnesses lists the harnesses that can run in chat mode today.
func (s *Service) ChatHarnesses(candidates []domain.AgentHarness) []domain.AgentHarness {
	if s.chat == nil {
		return nil
	}
	var out []domain.AgentHarness
	for _, harness := range candidates {
		if harness.IsSelectableForNewWork() && s.chat.SupportsChat(harness) {
			out = append(out, harness)
		}
	}
	return out
}

// Package settings owns Kennel's daemon-side user preferences.
//
// It exists so every spawn surface — desktop, mobile, `kennel spawn`, headless —
// resolves one value. A renderer-held preference would look correct in Settings
// while disagreeing with the CLI, which is worse than having no control.
package settings

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	SetReasoningVerification(ctx context.Context, verifiedAt *time.Time, provider, model string, now time.Time) error
}

// VerificationGenerationStore is implemented by durable stores that can
// conditionally apply a probe result. The optional seam preserves small test
// stores while production settings reject results from an older generation.
type VerificationGenerationStore interface {
	SetReasoningVerificationForGeneration(context.Context, *time.Time, string, string, int64, string, time.Time) (bool, error)
}

// Snapshot is the current preference set.
type Snapshot struct {
	DefaultSessionMode domain.SessionMode
	ReasoningProvider  string
	ReasoningModel     string
	ReasoningEffort    string
	// ReasoningVerifiedAt is when a probe last actually succeeded, for the
	// provider/model pair it succeeded for. Nil means never verified.
	ReasoningVerifiedAt              *time.Time
	ReasoningVerifiedProvider        string
	ReasoningVerifiedModel           string
	ReasoningGeneration              int64
	ReasoningVerifiedGeneration      int64
	ReasoningVerificationFingerprint string
	UpdatedAt                        time.Time
}

// SecretStore is intentionally narrower than a general credential manager.
// Settings can report whether a credential exists, but never return its value.
type SecretStore interface {
	Get(context.Context) (string, error)
	Set(context.Context, string) error
	Clear(context.Context) error
}

// ProviderSecretStore keeps credentials bound to the provider selected by the
// owner. SecretStore remains embedded for compatibility with older stores, but
// a provider-aware store must never hand one provider's key to another.
type ProviderSecretStore interface {
	SecretStore
	GetForProvider(context.Context, string) (string, error)
	SetForProvider(context.Context, string, string) error
	ClearForProvider(context.Context, string) error
}

// ReasoningInput is the owner-authored reasoning selection and optional secret update.
type ReasoningInput struct {
	Provider string
	Model    string
	Effort   string
	APIKey   string
	ClearKey bool
}

// ReasoningStatus reports configured and ready state without exposing the secret.
type ReasoningStatus struct {
	// Mode distinguishes direct API credentials from the signed-in Codex
	// harness while Provider remains the selected reasoning identity.
	Mode       string `json:"mode"`
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Effort     string `json:"effort"`
	Configured bool   `json:"configured"`
	// Ready reports only that a call can be attempted: a provider is selected
	// and a matching credential is present. It is deliberately NOT a claim that
	// reasoning works.
	Ready bool `json:"ready"`
	// KeyConfigured reports that a credential exists for the selected
	// provider. A present key can still be revoked or mistyped.
	KeyConfigured bool `json:"keyConfigured"`
	// Verified reports that an actual probe succeeded for exactly the
	// currently selected provider and model. Switching either one drops back to
	// false rather than inheriting the previous pair's proof.
	Verified   bool       `json:"verified"`
	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`
	ErrorCode  string     `json:"errorCode,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// ReasoningConfig is the resolved daemon-internal reasoning configuration.
type ReasoningConfig struct {
	Mode      string
	Provider  string
	APIKey    string
	Model     string
	Effort    string
	KeySource string
	// BaseURL is a development/test redirect to a local reasoning stand-in.
	// It comes only from KENNEL_WALDO_BASE_URL and is never persisted.
	BaseURL string
}

// ChatCapability reports which harnesses can run in chat mode, so the UI can warn
// that choosing chat narrows the agents available rather than letting the user
// discover it at spawn time.
type ChatCapability interface {
	SupportsChat(harness domain.AgentHarness) bool
}

// Service reads and writes preferences.
type Service struct {
	store        Store
	chat         ChatCapability
	now          func() time.Time
	secrets      SecretStore
	lookup       func(string) string
	probe        ReasoningProbe
	availability ReasoningAvailability
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

// GetReasoning reports the current reasoning readiness state.
func (s *Service) GetReasoning(ctx context.Context) (ReasoningStatus, error) {
	cfg, err := s.ResolveReasoning(ctx)
	status := ReasoningStatus{}
	if err != nil {
		if cfg.Provider != "" {
			status.Mode = cfg.Mode
			status.Provider = cfg.Provider
		}
		status.Model, status.Effort = cfg.Model, cfg.Effort
		status.KeyConfigured = cfg.APIKey != ""
		status.Configured = status.Provider != "" && status.KeyConfigured
		status.ErrorCode, status.Error = reasoningError(err)
		return status, nil
	}
	status = ReasoningStatus{
		Mode: cfg.Mode, Provider: cfg.Provider, Model: cfg.Model, Effort: cfg.Effort,
		KeyConfigured: cfg.APIKey != "", Configured: true, Ready: true,
	}
	if cfg.Provider == "codex" {
		status.KeyConfigured = false
		if s.availability == nil {
			status.Ready = false
			status.ErrorCode, status.Error = reasoningError(ports.NewReasoningFailure(
				ports.ReasoningUnavailable, "Codex harness availability has not been checked", nil))
		} else if availabilityErr := s.availability(ctx, cfg); availabilityErr != nil {
			status.Ready = false
			status.ErrorCode, status.Error = reasoningError(availabilityErr)
		}
	}
	if snapshot, snapErr := s.store.GetAppSettings(ctx); snapErr == nil {
		status.Verified, status.VerifiedAt = verificationFor(snapshot, cfg)
	}
	return status, nil
}

// verificationFor reports a stored verification only when it belongs to the
// provider and model in force now.
//
// A verification is proof about one exact pair. Carrying it across a provider
// switch would be the same mistake as reusing another provider's credential:
// the owner would be told reasoning is verified for a combination that was
// never probed.
func verificationFor(snapshot Snapshot, cfg ReasoningConfig) (bool, *time.Time) {
	if snapshot.ReasoningVerifiedAt == nil {
		return false, nil
	}
	if snapshot.ReasoningVerifiedProvider != cfg.Provider {
		return false, nil
	}
	// An empty verified model means the probe ran against the provider default,
	// which only still holds if no explicit model is selected now.
	if snapshot.ReasoningVerifiedModel != cfg.Model {
		return false, nil
	}
	if snapshot.ReasoningGeneration > 0 && (snapshot.ReasoningVerifiedGeneration != snapshot.ReasoningGeneration || snapshot.ReasoningVerificationFingerprint != reasoningFingerprint(cfg)) {
		return false, nil
	}
	return true, snapshot.ReasoningVerifiedAt
}

func reasoningFingerprint(cfg ReasoningConfig) string {
	// A credential digest binds the probe without storing or returning the
	// credential itself. Endpoint/model/provider overrides are included because
	// they change what the probe actually tested.
	h := sha256.New()
	for _, value := range []string{cfg.Mode, cfg.Provider, cfg.Model, cfg.Effort, cfg.BaseURL, cfg.APIKey} {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(value))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyReasoning probes the configured provider with one minimal real call and
// records whether it worked.
//
// This is the only thing that can set Verified, and it is owner-triggered: a
// reasoning call may be billed, so the daemon never probes on its own schedule.
// A failure clears any stored verification instead of leaving a stale success
// in place, and returns the classified reason so the UI can say what to fix.
func (s *Service) VerifyReasoning(ctx context.Context) (ReasoningStatus, error) {
	return s.verifyReasoningWith(ctx, s.probe)
}

// WithReasoningProbe supplies the probe used by VerifyReasoning. The daemon
// wires the real one; keeping it injectable is what lets the probe's own
// classification be tested against a local HTTP stand-in rather than a
// live provider.
func (s *Service) WithReasoningProbe(probe ReasoningProbe) *Service {
	s.probe = probe
	return s
}

// ReasoningAvailability checks local/provider harness readiness without making
// a model call. Direct API modes retain their credential-presence semantics;
// the Codex harness uses this to avoid claiming readiness when the binary,
// protocol, or sign-in state is unavailable.
type ReasoningAvailability func(context.Context, ReasoningConfig) error

func (s *Service) WithReasoningAvailability(check ReasoningAvailability) *Service {
	s.availability = check
	return s
}

// ReasoningProbe performs one minimal real reasoning call for the resolved
// configuration and reports whether it worked.
type ReasoningProbe func(context.Context, ReasoningConfig) error

func (s *Service) verifyReasoningWith(ctx context.Context, probe ReasoningProbe) (ReasoningStatus, error) {
	cfg, err := s.ResolveReasoning(ctx)
	if err != nil {
		// Not configured at all: nothing to probe, and the existing readiness
		// path already describes it accurately.
		return s.GetReasoning(ctx)
	}
	if probe == nil {
		return ReasoningStatus{}, fmt.Errorf("no reasoning probe is available")
	}
	start, err := s.store.GetAppSettings(ctx)
	if err != nil {
		return ReasoningStatus{}, err
	}
	fingerprint := reasoningFingerprint(cfg)
	if probeErr := probe(ctx, cfg); probeErr != nil {
		current, currentErr := s.currentVerificationConfig(ctx)
		if currentErr != nil {
			return ReasoningStatus{}, currentErr
		}
		if current != fingerprint || !s.applyVerification(ctx, nil, "", "", start.ReasoningGeneration, fingerprint) {
			status, _ := s.GetReasoning(ctx)
			status.ErrorCode, status.Error = "VERIFICATION_STALE", "Verification finished for an older reasoning configuration; verify the current settings"
			return status, nil
		}
		if clearErr := s.clearVerification(ctx, start.ReasoningGeneration, fingerprint); clearErr != nil {
			return ReasoningStatus{}, clearErr
		}
		status, statusErr := s.GetReasoning(ctx)
		if statusErr != nil {
			return ReasoningStatus{}, statusErr
		}
		status.Verified, status.VerifiedAt = false, nil
		status.ErrorCode, status.Error = reasoningError(probeErr)
		return status, nil
	}
	verified := s.now().UTC()
	current, currentErr := s.currentVerificationConfig(ctx)
	if currentErr != nil {
		return ReasoningStatus{}, currentErr
	}
	if current != fingerprint {
		status, _ := s.GetReasoning(ctx)
		status.ErrorCode, status.Error = "VERIFICATION_STALE", "Verification finished for an older reasoning configuration; verify the current settings"
		return status, nil
	}
	if applied := s.applyVerification(ctx, &verified, cfg.Provider, cfg.Model, start.ReasoningGeneration, fingerprint); !applied {
		status, _ := s.GetReasoning(ctx)
		status.ErrorCode, status.Error = "VERIFICATION_STALE", "Verification finished for an older reasoning configuration; verify the current settings"
		return status, nil
	}
	if err := s.setVerification(ctx, &verified, cfg.Provider, cfg.Model); err != nil {
		return ReasoningStatus{}, err
	}
	return s.GetReasoning(ctx)
}

func (s *Service) currentVerificationConfig(ctx context.Context) (string, error) {
	cfg, err := s.ResolveReasoning(ctx)
	if err != nil {
		return "", err
	}
	return reasoningFingerprint(cfg), nil
}

func (s *Service) applyVerification(ctx context.Context, at *time.Time, provider, model string, generation int64, fingerprint string) bool {
	store, ok := s.store.(VerificationGenerationStore)
	if !ok {
		return true
	}
	applied, err := store.SetReasoningVerificationForGeneration(ctx, at, provider, model, generation, fingerprint, s.now())
	return err == nil && applied
}

func (s *Service) clearVerification(ctx context.Context, generation int64, fingerprint string) error {
	if store, ok := s.store.(VerificationGenerationStore); ok {
		applied, err := store.SetReasoningVerificationForGeneration(ctx, nil, "", "", generation, fingerprint, s.now())
		if err != nil {
			return err
		}
		if !applied {
			return nil
		}
		return nil
	}
	return s.store.SetReasoningVerification(ctx, nil, "", "", s.now())
}

func (s *Service) setVerification(ctx context.Context, at *time.Time, provider, model string) error {
	if _, ok := s.store.(VerificationGenerationStore); ok {
		return nil
	}
	return s.store.SetReasoningVerification(ctx, at, provider, model, s.now())
}

// SetReasoning persists the owner-selected provider/model/effort and secret.
func (s *Service) SetReasoning(ctx context.Context, input ReasoningInput) (ReasoningStatus, error) {
	provider := strings.ToLower(strings.TrimSpace(input.Provider))
	if provider != "anthropic" && provider != "openai" && provider != "codex" {
		return ReasoningStatus{}, fmt.Errorf("reasoning provider must be anthropic, openai, or codex")
	}
	if provider == "codex" && strings.TrimSpace(input.APIKey) != "" {
		return ReasoningStatus{}, fmt.Errorf("Codex harness mode uses Codex sign-in; do not provide an API key")
	}
	if provider != "codex" && s.secrets == nil {
		return ReasoningStatus{}, fmt.Errorf("local reasoning secret store is unavailable")
	}
	if provider != "codex" && input.ClearKey {
		if err := s.clearReasoningSecret(ctx, provider); err != nil {
			return ReasoningStatus{}, err
		}
	} else if strings.TrimSpace(input.APIKey) != "" {
		if err := s.setReasoningSecret(ctx, provider, input.APIKey); err != nil {
			return ReasoningStatus{}, err
		}
	}
	if err := s.store.SetReasoningSettings(ctx, provider, strings.TrimSpace(input.Model), strings.TrimSpace(input.Effort), s.now()); err != nil {
		return ReasoningStatus{}, err
	}
	// Any change to the selection or the credential invalidates the previous
	// probe. Keeping the old stamp would report a combination as verified that
	// was never tried with these settings.
	if err := s.store.SetReasoningVerification(ctx, nil, "", "", s.now()); err != nil {
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
	key := ""
	if provider != "codex" && s.secrets != nil {
		if providerSecrets, ok := s.secrets.(ProviderSecretStore); ok && (provider == "anthropic" || provider == "openai") {
			key, err = providerSecrets.GetForProvider(ctx, provider)
		} else {
			key, err = s.secrets.Get(ctx)
		}
		if err != nil {
			return ReasoningConfig{}, err
		}
	}
	keySource := "local-secret-store"
	if provider == "codex" {
		keySource = "codex-app-server-sign-in"
	}
	if provider != "codex" {
		if shared != "" {
			key, keySource = shared, "KENNEL_WALDO_API_KEY"
		} else if provider == "anthropic" && ant != "" {
			key, keySource = ant, "ANTHROPIC_API_KEY"
		} else if provider == "openai" && oai != "" {
			key, keySource = oai, "OPENAI_API_KEY"
		}
	}
	cfg := ReasoningConfig{
		Mode: reasoningMode(provider), Provider: provider, APIKey: key, Model: model, Effort: effort, KeySource: keySource,
		BaseURL: strings.TrimSpace(lookup("KENNEL_WALDO_BASE_URL")),
	}
	if provider == "" {
		return cfg, fmt.Errorf("%w; choose anthropic, openai, or codex", errProviderNotSelected)
	}
	if provider != "anthropic" && provider != "openai" && provider != "codex" {
		return cfg, fmt.Errorf("%w: %q", errProviderUnsupported, provider)
	}
	if provider != "codex" && key == "" {
		return cfg, fmt.Errorf("%w for %s", errMissingCredential, provider)
	}
	return cfg, nil
}

// Local setup sentinels. They exist so reasoningError can identify a setup
// state without matching on message text.
var (
	errProviderNotSelected = errors.New("reasoning provider is not configured")
	errProviderUnsupported = errors.New("unsupported reasoning provider")
	errMissingCredential   = errors.New("reasoning credential is not configured")
)

func reasoningMode(provider string) string {
	if provider == "codex" {
		return "codex_harness"
	}
	if provider == "anthropic" || provider == "openai" {
		return "direct_api"
	}
	return ""
}

func (s *Service) setReasoningSecret(ctx context.Context, provider, value string) error {
	if providerSecrets, ok := s.secrets.(ProviderSecretStore); ok {
		return providerSecrets.SetForProvider(ctx, provider, value)
	}
	return s.secrets.Set(ctx, value)
}

func (s *Service) clearReasoningSecret(ctx context.Context, provider string) error {
	if providerSecrets, ok := s.secrets.(ProviderSecretStore); ok {
		return providerSecrets.ClearForProvider(ctx, provider)
	}
	return s.secrets.Clear(ctx)
}

// reasoningError maps a setup or probe failure to a stable code.
//
// It used to derive the code by substring-matching its own error prose, so
// rewording a message silently changed the machine code the UI switches on.
// Classified reasoning failures now answer for themselves; the remaining
// string cases are this package's own local setup errors, matched on the
// sentinels it raises rather than on free text.
func reasoningError(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	var failure *ports.ReasoningFailure
	if errors.As(err, &failure) {
		switch failure.Kind {
		case ports.ReasoningNotConfigured:
			return "MISSING_CREDENTIAL", failure.Error()
		case ports.ReasoningUnauthorized:
			return "CREDENTIAL_REJECTED", failure.Error()
		default:
			return "REASONING_NOT_READY", failure.Error()
		}
	}
	switch {
	case errors.Is(err, errMissingCredential):
		return "MISSING_CREDENTIAL", err.Error()
	case errors.Is(err, errProviderNotSelected), errors.Is(err, errProviderUnsupported):
		return "PROVIDER_NOT_READY", err.Error()
	default:
		return "REASONING_NOT_READY", err.Error()
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

// ReasoningBaseURL reports the development/test reasoning redirect, if any.
// Empty in every packaged install, because it is env-only and never persisted.
func (s *Service) ReasoningBaseURL() string {
	if s == nil || s.lookup == nil {
		return ""
	}
	return strings.TrimSpace(s.lookup("KENNEL_WALDO_BASE_URL"))
}

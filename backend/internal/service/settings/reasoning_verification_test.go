package settings

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

func verifiableService(provider, model, key string) (*Service, *reasoningSettingsStore) {
	store := &reasoningSettingsStore{snapshot: Snapshot{ReasoningProvider: provider, ReasoningModel: model}}
	secret := &providerReasoningSecret{values: map[string]string{provider: key}}
	svc := New(store, nil, nil).
		WithReasoningSecrets(secret).
		WithReasoningEnvLookup(func(string) string { return "" })
	return svc, store
}

// A stored credential proves nothing about whether reasoning works: it can be
// revoked or mistyped. Readiness previously set Ready=true purely because the
// key string was non-empty and reported nothing else, so the owner was sent
// into a Plan proposal that then failed at the provider.
func TestGetReasoningSeparatesKeyPresentFromVerified(t *testing.T) {
	svc, _ := verifiableService("openai", "gpt-test", "present-but-unproven")

	status, err := svc.GetReasoning(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !status.KeyConfigured || !status.Configured || !status.Ready {
		t.Fatalf("a present credential should still report attemptable: %#v", status)
	}
	if status.Verified || status.VerifiedAt != nil {
		t.Fatalf("an unprobed credential must never report verified: %#v", status)
	}
}

func TestVerifyReasoningRecordsAnActualProbe(t *testing.T) {
	svc, store := verifiableService("openai", "gpt-test", "good-key")
	probed := 0

	status, err := svc.WithReasoningProbe(func(_ context.Context, cfg ReasoningConfig) error {
		probed++
		if cfg.APIKey != "good-key" || cfg.Provider != "openai" {
			t.Fatalf("probe received %#v", cfg)
		}
		return nil
	}).VerifyReasoning(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if probed != 1 {
		t.Fatalf("probe calls = %d, want exactly one owner-triggered call", probed)
	}
	if !status.Verified || status.VerifiedAt == nil {
		t.Fatalf("status = %#v, want verified", status)
	}
	if store.snapshot.ReasoningVerifiedProvider != "openai" || store.snapshot.ReasoningVerifiedModel != "gpt-test" {
		t.Fatalf("verification was not bound to the probed pair: %#v", store.snapshot)
	}
}

// A failed probe must clear the record rather than leave a stale success, and
// must say what to fix.
func TestVerifyReasoningClearsVerificationOnFailure(t *testing.T) {
	svc, store := verifiableService("openai", "gpt-test", "revoked-key")
	earlier := time.Now().UTC().Add(-time.Hour)
	store.snapshot.ReasoningVerifiedAt = &earlier
	store.snapshot.ReasoningVerifiedProvider, store.snapshot.ReasoningVerifiedModel = "openai", "gpt-test"

	status, err := svc.WithReasoningProbe(func(context.Context, ReasoningConfig) error {
		return ports.NewReasoningFailure(ports.ReasoningUnauthorized, "provider rejected the credential", nil)
	}).VerifyReasoning(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Verified || status.VerifiedAt != nil {
		t.Fatalf("a failed probe must not report verified: %#v", status)
	}
	if store.snapshot.ReasoningVerifiedAt != nil {
		t.Fatal("a failed probe must clear the stored verification, not keep a stale success")
	}
	if status.ErrorCode != "CREDENTIAL_REJECTED" {
		t.Fatalf("errorCode = %q, want CREDENTIAL_REJECTED", status.ErrorCode)
	}
}

// Verification belongs to the exact provider/model pair that was probed.
// Inheriting it across a switch is the same class of bug as reusing another
// provider's credential.
func TestVerificationDoesNotSurviveAProviderOrModelChange(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		verifiedProvider, nowProv string
		verifiedModel, nowModel   string
	}{
		{"provider switched", "openai", "anthropic", "m", "m"},
		{"model switched", "openai", "openai", "gpt-test", "gpt-other"},
		{"explicit model added after a default probe", "openai", "openai", "", "gpt-test"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stamp := time.Now().UTC()
			snapshot := Snapshot{
				ReasoningVerifiedAt:       &stamp,
				ReasoningVerifiedProvider: tc.verifiedProvider,
				ReasoningVerifiedModel:    tc.verifiedModel,
			}
			verified, at := verificationFor(snapshot, ReasoningConfig{Provider: tc.nowProv, Model: tc.nowModel})
			if verified || at != nil {
				t.Fatalf("verification leaked across a change: verified=%v at=%v", verified, at)
			}
		})
	}
}

// Changing the selection or the credential must invalidate the old probe.
func TestSetReasoningInvalidatesAnEarlierVerification(t *testing.T) {
	svc, store := verifiableService("openai", "gpt-test", "good-key")
	stamp := time.Now().UTC()
	store.snapshot.ReasoningVerifiedAt = &stamp
	store.snapshot.ReasoningVerifiedProvider, store.snapshot.ReasoningVerifiedModel = "openai", "gpt-test"

	if _, err := svc.SetReasoning(context.Background(), ReasoningInput{
		Provider: "anthropic", Model: "claude-test", APIKey: "another-key",
	}); err != nil {
		t.Fatal(err)
	}
	if store.snapshot.ReasoningVerifiedAt != nil {
		t.Fatal("a settings change must clear the previous verification")
	}
}

// The machine code the UI switches on must not depend on error prose. It used
// to be derived with strings.Contains over this package's own messages, so a
// reworded message silently changed the code.
func TestReasoningErrorCodesComeFromSentinelsNotMessageText(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"missing credential", errMissingCredential, "MISSING_CREDENTIAL"},
		{"no provider selected", errProviderNotSelected, "PROVIDER_NOT_READY"},
		{"unsupported provider", errProviderUnsupported, "PROVIDER_NOT_READY"},
		// These two are the red-green: their messages contain neither
		// "credential" nor "provider", so the old substring matcher answered
		// REASONING_NOT_READY for both and the UI could not tell a missing key
		// from a rejected one.
		{"classified missing setup", ports.NewReasoningFailure(ports.ReasoningNotConfigured, "x", nil), "MISSING_CREDENTIAL"},
		{"classified rejection", ports.NewReasoningFailure(ports.ReasoningUnauthorized, "x", nil), "CREDENTIAL_REJECTED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, message := reasoningError(tc.err)
			if code == "" || message == "" {
				t.Fatalf("code=%q message=%q, both are required", code, message)
			}
			if code != tc.want {
				t.Fatalf("code = %q, want %q", code, tc.want)
			}
			// Wrapping must not change the code. Classification travels
			// through errors.As/Is; a substring matcher would answer on
			// whatever words the outer layer happened to add.
			wrapped := fmt.Errorf("proposing a plan: %w", tc.err)
			if got, _ := reasoningError(wrapped); got != code {
				t.Fatalf("wrapped code = %q, want %q", got, code)
			}
		})
	}
}

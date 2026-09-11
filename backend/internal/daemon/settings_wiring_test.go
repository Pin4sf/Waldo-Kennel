package daemon

import (
	"context"
	"testing"
	"time"

	settingssvc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/settings"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
)

func TestSettingsWiringVerificationSurvivesReopen(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := sqlite.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := db.SetReasoningSettings(ctx, "openai", "gpt-5-nano", "minimal", time.Now()); err != nil {
		t.Fatal(err)
	}
	makeService := func() *settingssvc.Service {
		return settingssvc.New(settingsStore{store: db}, nil, nil).WithReasoningEnvLookup(func(key string) string {
			if key == "OPENAI_API_KEY" {
				return "test-only-key"
			}
			return ""
		})
	}
	svc := makeService().WithReasoningProbe(func(context.Context, settingssvc.ReasoningConfig) error { return nil })
	status, err := svc.VerifyReasoning(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Verified {
		t.Fatalf("successful probe lost verification through daemon wiring: %#v", status)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sqlite.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	status, err = makeService().GetReasoning(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Verified {
		t.Fatal("verification did not survive SQLite reopen")
	}
	if err := db.SetReasoningSettings(ctx, "openai", "different-model", "minimal", time.Now()); err != nil {
		t.Fatal(err)
	}
	status, err = makeService().GetReasoning(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.Verified {
		t.Fatal("changed model inherited verification")
	}
}

func TestSettingsWiringRejectsVerificationAfterSelectionChanges(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := db.SetReasoningSettings(ctx, "openai", "gpt-5-nano", "minimal", time.Now()); err != nil {
		t.Fatal(err)
	}
	svc := settingssvc.New(settingsStore{store: db}, nil, nil).
		WithReasoningEnvLookup(func(key string) string {
			if key == "OPENAI_API_KEY" {
				return "test-only-key"
			}
			return ""
		}).
		WithReasoningProbe(func(context.Context, settingssvc.ReasoningConfig) error {
			// Even an identical selection is a new generation and must invalidate
			// the outstanding probe rather than accepting its earlier authorization.
			return db.SetReasoningSettings(ctx, "openai", "gpt-5-nano", "minimal", time.Now())
		})
	status, err := svc.VerifyReasoning(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.Verified || status.ErrorCode != "VERIFICATION_STALE" {
		t.Fatalf("superseded probe was not rejected: %#v", status)
	}
}

func TestNativePlanningCandidateRequiresSuccessfulPacketVerification(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := db.SetReasoningSettings(ctx, providerCodex, "", "medium", time.Now()); err != nil {
		t.Fatal(err)
	}
	svc := settingssvc.New(settingsStore{store: db}, nil, nil).
		WithReasoningAvailability(func(context.Context, settingssvc.ReasoningConfig) error { return nil }).
		WithReasoningProbe(func(context.Context, settingssvc.ReasoningConfig) error { return nil })
	provider := newConfiguredIntelligenceProvider(svc, nil)

	candidates, err := provider.PlanningCandidates(ctx)
	if err != nil || len(candidates) != 1 {
		t.Fatalf("unverified native candidates = %+v, err=%v", candidates, err)
	}
	if candidates[0].Ready || candidates[0].UnavailableCode != "REASONING_NOT_VERIFIED" {
		t.Fatalf("unverified native candidate must be unavailable: %+v", candidates[0])
	}
	if candidates[0].Binding.Mode != "native_harness" {
		t.Fatalf("candidate binding = %+v", candidates[0].Binding)
	}

	status, err := svc.VerifyReasoning(ctx)
	if err != nil || !status.Verified {
		t.Fatalf("verify native packet status=%+v err=%v", status, err)
	}
	candidates, err = provider.PlanningCandidates(ctx)
	if err != nil || len(candidates) != 1 || !candidates[0].Ready || candidates[0].UnavailableCode != "" {
		t.Fatalf("verified native candidates = %+v, err=%v", candidates, err)
	}
}

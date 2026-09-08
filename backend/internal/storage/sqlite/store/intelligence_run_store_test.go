package store_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func TestIntelligenceRunStoreRoundTripAndTerminalImmutability(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	seedProject(t, s, "project-intel")
	seedAnalyzingIntakeForProject(t, s, "project-intel", "intake-intel", "key-intel", now)

	run := domain.IntelligenceRun{
		ID:                "intel-store-1",
		Kind:              domain.IntelligenceRunContractAnalysis,
		ProjectID:         "project-intel",
		IntakeID:          "intake-intel",
		SourceRevision:    0,
		RequestedProvider: "waldo-reasoner",
		RequestedModel:    "planner-v2",
		EffectiveProvider: "direct-api.example/v1",
		EffectiveModel:    "planner-2026-09",
		NativeSessionRef:  "native-request-1",
		InputDigest:       domain.DigestSHA256([]byte("input")),
		Status:            domain.IntelligenceRunRequested,
		CreatedAt:         now,
	}
	if err := s.CreateIntelligenceRun(ctx, run); err != nil {
		t.Fatalf("create intelligence run: %v", err)
	}
	got, found, err := s.GetIntelligenceRun(ctx, run.ID)
	if err != nil || !found {
		t.Fatalf("get intelligence run: found=%v err=%v", found, err)
	}
	if got.RequestedProvider != run.RequestedProvider || got.EffectiveProvider != run.EffectiveProvider || got.EffectiveModel != run.EffectiveModel || got.InputDigest != run.InputDigest {
		t.Fatalf("round trip changed provenance: %#v", got)
	}

	if err := s.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunRunning, "", "", "", nil); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	completedAt := now.Add(time.Minute)
	output := domain.DigestSHA256([]byte("output-a"))
	if err := s.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFulfilled, output, "", "", &completedAt); err != nil {
		t.Fatalf("fulfill intelligence run: %v", err)
	}
	if err := s.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFulfilled, output, "", "", &completedAt); err != nil {
		t.Fatalf("identical terminal replay should be idempotent: %v", err)
	}
	if err := s.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFulfilled, domain.DigestSHA256([]byte("output-b")), "", "", &completedAt); err == nil {
		t.Fatal("terminal replay replaced output digest")
	}

	got, found, err = s.GetIntelligenceRun(ctx, run.ID)
	if err != nil || !found {
		t.Fatalf("get fulfilled run: found=%v err=%v", found, err)
	}
	if got.Status != domain.IntelligenceRunFulfilled || got.OutputDigest != output || got.CompletedAt == nil || !got.CompletedAt.Equal(completedAt) {
		t.Fatalf("fulfilled state changed after rejected replay: %#v", got)
	}
}

func TestIntelligenceRunStoreEffectiveProvenanceIsMonotonic(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	seedProject(t, s, "project-provenance")
	seedAnalyzingIntakeForProject(t, s, "project-provenance", "intake-provenance", "key-provenance", now)

	run := domain.IntelligenceRun{
		ID: "intel-provenance", Kind: domain.IntelligenceRunContractAnalysis,
		ProjectID: "project-provenance", IntakeID: "intake-provenance",
		InputDigest: domain.DigestSHA256([]byte("input")), Status: domain.IntelligenceRunRunning, CreatedAt: now,
	}
	if err := s.CreateIntelligenceRun(ctx, run); err != nil {
		t.Fatalf("create intelligence run: %v", err)
	}
	if err := s.RecordIntelligenceRunEffectiveProvenance(ctx, run.ID, "provider-a", "", "native-1"); err != nil {
		t.Fatalf("record provider provenance: %v", err)
	}
	if err := s.RecordIntelligenceRunEffectiveProvenance(ctx, run.ID, "provider-a", "model-a", "native-1"); err != nil {
		t.Fatalf("fill effective model: %v", err)
	}
	if err := s.RecordIntelligenceRunEffectiveProvenance(ctx, run.ID, "provider-b", "model-b", "native-2"); err == nil {
		t.Fatal("known effective provenance was replaced")
	}
	got, _, err := s.GetIntelligenceRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("get intelligence run: %v", err)
	}
	if got.EffectiveProvider != "provider-a" || got.EffectiveModel != "model-a" || got.NativeSessionRef != "native-1" {
		t.Fatalf("effective provenance changed: %#v", got)
	}
}

func TestIntelligenceRunStoreListsOnlyNonTerminalRuns(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	seedProject(t, s, "project-list")

	statuses := []domain.IntelligenceRunStatus{
		domain.IntelligenceRunRequested,
		domain.IntelligenceRunRunning,
		domain.IntelligenceRunFailed,
	}
	for i, status := range statuses {
		intakeID := domain.IntakeSessionID("intake-list-" + string(rune('a'+i)))
		seedAnalyzingIntakeForProject(t, s, "project-list", intakeID, "key-"+intakeID.String(), now.Add(time.Duration(i)*time.Second))
		run := domain.IntelligenceRun{
			ID: domain.IntelligenceRunID("intel-list-" + string(rune('a'+i))), Kind: domain.IntelligenceRunContractAnalysis,
			ProjectID: "project-list", IntakeID: intakeID, SourceRevision: 0,
			InputDigest: domain.DigestSHA256([]byte("list input")), Status: status,
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		}
		if status.Terminal() {
			completedAt := now.Add(time.Minute)
			run.CompletedAt = &completedAt
		}
		if err := s.CreateIntelligenceRun(ctx, run); err != nil {
			t.Fatalf("create %s: %v", run.ID, err)
		}
	}

	got, err := s.ListNonTerminalIntelligenceRuns(ctx)
	if err != nil {
		t.Fatalf("list non-terminal runs: %v", err)
	}
	if len(got) != 2 || got[0].Status != domain.IntelligenceRunRequested || got[1].Status != domain.IntelligenceRunRunning {
		t.Fatalf("non-terminal runs = %#v", got)
	}
}

func TestIntelligenceRunPersistenceStoresDigestNotSensitiveInput(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	seedProject(t, s, "project-secret")
	seedAnalyzingIntakeForProject(t, s, "project-secret", "intake-secret", "key-secret", now)

	const canary = "sk-canary-must-never-be-persisted-as-intelligence-input"
	run := domain.IntelligenceRun{
		ID: "intel-secret", Kind: domain.IntelligenceRunContractAnalysis,
		ProjectID: "project-secret", IntakeID: "intake-secret",
		InputDigest: domain.DigestSHA256([]byte(canary)), Status: domain.IntelligenceRunRequested, CreatedAt: now,
	}
	if err := s.CreateIntelligenceRun(ctx, run); err != nil {
		t.Fatalf("create intelligence run: %v", err)
	}
	got, found, err := s.GetIntelligenceRun(ctx, run.ID)
	if err != nil || !found {
		t.Fatalf("get intelligence run: found=%v err=%v", found, err)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal persisted run: %v", err)
	}
	if strings.Contains(string(encoded), canary) {
		t.Fatal("sensitive source input appeared in persisted IntelligenceRun state")
	}
	if got.InputDigest != domain.DigestSHA256([]byte(canary)) {
		t.Fatalf("persisted digest = %q", got.InputDigest)
	}
}

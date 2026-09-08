package store_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func TestIntelligenceRunStoreRoundTripAndStatus(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	run := domain.IntelligenceRun{
		ID:             domain.IntelligenceRunID("intel-store-1"),
		Kind:           domain.IntelligenceRunContractAnalysis,
		ProjectID:      domain.ProjectID("project-intel"),
		IntakeID:       domain.IntakeSessionID("intake-intel"),
		SourceRevision: 3,
		Provider:       domain.HarnessCodex,
		ModelSelection: domain.IntelligenceRunModelExplicit,
		Model:          "model-x",
		InputDigest:    strings.Repeat("a", 64),
		Status:         domain.IntelligenceRunRequested,
		CreatedAt:      now,
	}
	if err := s.CreateIntelligenceRun(ctx, run); err != nil {
		t.Fatalf("create intelligence run: %v", err)
	}
	got, found, err := s.GetIntelligenceRun(ctx, run.ID)
	if err != nil || !found {
		t.Fatalf("get intelligence run: found=%v err=%v", found, err)
	}
	if got.SourceRevision != 3 || got.Provider != domain.HarnessCodex || got.Model != "model-x" || got.InputDigest != run.InputDigest {
		t.Fatalf("round trip changed provenance: %#v", got)
	}

	if err := s.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunRunning, "", "", "", nil); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	completedAt := now.Add(time.Minute)
	if err := s.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFulfilled, strings.Repeat("b", 64), "", "", &completedAt); err != nil {
		t.Fatalf("fulfill intelligence run: %v", err)
	}
	got, found, err = s.GetIntelligenceRun(ctx, run.ID)
	if err != nil || !found {
		t.Fatalf("get fulfilled run: found=%v err=%v", found, err)
	}
	if got.Status != domain.IntelligenceRunFulfilled || got.OutputDigest != strings.Repeat("b", 64) || got.CompletedAt == nil {
		t.Fatalf("fulfilled state not persisted: %#v", got)
	}
}

func TestIntelligenceRunStoreListsOnlyNonTerminalRuns(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	for i, status := range []domain.IntelligenceRunStatus{
		domain.IntelligenceRunRequested,
		domain.IntelligenceRunRunning,
		domain.IntelligenceRunFailed,
	} {
		run := domain.IntelligenceRun{
			ID:             domain.IntelligenceRunID("intel-list-" + string(rune('a'+i))),
			Kind:           domain.IntelligenceRunContractAnalysis,
			ProjectID:      domain.ProjectID("project-intel"),
			IntakeID:       domain.IntakeSessionID("intake-intel"),
			SourceRevision: int64(i),
			InputDigest:    strings.Repeat("c", 64),
			Status:         status,
			CreatedAt:      now.Add(time.Duration(i) * time.Second),
		}
		if status == domain.IntelligenceRunFailed {
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

func TestIntelligenceRunCarriesNoSecretOrCallbackTokenField(t *testing.T) {
	typeOfRun := reflect.TypeOf(domain.IntelligenceRun{})
	for i := 0; i < typeOfRun.NumField(); i++ {
		name := strings.ToLower(typeOfRun.Field(i).Name)
		if strings.Contains(name, "secret") || strings.Contains(name, "token") || strings.Contains(name, "apikey") || strings.Contains(name, "api_key") {
			t.Fatalf("IntelligenceRun persists secret-bearing field %q", typeOfRun.Field(i).Name)
		}
	}
}

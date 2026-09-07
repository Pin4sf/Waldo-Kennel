package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

// Expiry is a terminal intake fact, not just a provider-request fact. If the
// daemon stops after closing an expired request, reopening the database must
// not reconstruct the intake as still analyzing, and a late provider callback
// must not be able to resurrect the closed ask.
func TestExpiredIntakeAnalysisIsDurableAcrossRestart(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	store, err := sqlitetest.Open(dataDir)
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}

	seedProject(t, store, "req-project")
	now := time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)
	seedAnalyzingIntake(t, store, "intake-expired", "key-expired", now)

	request := domain.IntakeAnalysisRequest{
		ID: "ireq-expired", IntakeID: "intake-expired", ExpectedProposalRevision: 0,
		Status: domain.IntakeAnalysisRequested, CallbackTokenDigest: domain.HashCallbackToken("expired-token"),
		ExpiresAt: now.Add(domain.DefaultIntakeAnalysisRequestTTL), CreatedAt: now,
	}
	if err := store.CreateIntakeAnalysisRequest(ctx, request); err != nil {
		t.Fatalf("create request: %v", err)
	}

	expiredAt := request.ExpiresAt.Add(time.Second)
	if err := store.AnswerIntakeAnalysisRequest(ctx, ports.IntakeAnalysisRequestAnswer{
		RequestID: request.ID,
		Status: domain.IntakeAnalysisExpired,
		RefusalReason: "No proposal arrived before the request expired",
		At: expiredAt,
	}); err != nil {
		t.Fatalf("expire request: %v", err)
	}

	assertExpiredIntakeTruth(t, ctx, store, request.ID, request.IntakeID)

	if err := store.Close(); err != nil {
		t.Fatalf("close store before restart: %v", err)
	}

	reopened, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer func() {
		if err := reopened.Close(); err != nil {
			t.Errorf("close reopened store: %v", err)
		}
	}()

	assertExpiredIntakeTruth(t, ctx, reopened, request.ID, request.IntakeID)

	late := ports.IntakeAnalysisRequestAnswer{
		RequestID: request.ID,
		Status: domain.IntakeAnalysisFulfilled,
		RawProposal: `{"late":true}`,
		At: expiredAt.Add(time.Minute),
	}
	if err := reopened.AnswerIntakeAnalysisRequest(ctx, late); !errors.Is(err, ports.ErrIntakeAnalysisRequestClosed) {
		t.Fatalf("late callback error = %v, want ErrIntakeAnalysisRequestClosed", err)
	}
}

func assertExpiredIntakeTruth(t *testing.T, ctx context.Context, store interface {
	GetIntake(context.Context, domain.IntakeSessionID) (ports.IntakeSnapshot, bool, error)
	GetIntakeAnalysisRequest(context.Context, domain.IntakeAnalysisRequestID) (domain.IntakeAnalysisRequest, bool, error)
}, requestID domain.IntakeAnalysisRequestID, intakeID domain.IntakeSessionID) {
	t.Helper()

	request, found, err := store.GetIntakeAnalysisRequest(ctx, requestID)
	if err != nil || !found {
		t.Fatalf("get expired request: found=%v err=%v", found, err)
	}
	if request.Status != domain.IntakeAnalysisExpired {
		t.Fatalf("request status = %q, want %q", request.Status, domain.IntakeAnalysisExpired)
	}
	if request.AnsweredAt == nil {
		t.Fatal("expired request lost its terminal timestamp")
	}

	intake, found, err := store.GetIntake(ctx, intakeID)
	if err != nil || !found {
		t.Fatalf("get expired intake: found=%v err=%v", found, err)
	}
	if intake.Session.Status != domain.IntakeStatusAnalysisFailed {
		t.Fatalf("intake status = %q, want %q", intake.Session.Status, domain.IntakeStatusAnalysisFailed)
	}
	if intake.Session.FailureCode != "INTAKE_ANALYSIS_EXPIRED" {
		t.Fatalf("intake failure code = %q, want INTAKE_ANALYSIS_EXPIRED", intake.Session.FailureCode)
	}
}

package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	sqlitestore "github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/store"
)

func TestMigration0115PreservesExistingIntakeAnalysisRequest(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "kennel.db")+pragmas)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	upTo(t, db, 114)
	store := sqlitestore.NewStore(db, db)
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC)
	if err := store.UpsertProject(ctx, domain.ProjectRecord{
		ID: "intel-migrate", Path: "/tmp/intel-migrate", RegisteredAt: now,
	}); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	intake := domain.IntakeSession{
		ID: "intake-before-0115", SourceSurface: domain.IntakeSourceWork,
		Purpose: domain.IntakePurposeOutcome, ProjectID: "intel-migrate",
		Statement: "preserve this pre-0115 request", Status: domain.IntakeStatusCaptured,
		CreatedAt: now, UpdatedAt: now,
	}
	if _, err := store.CreateIntake(ctx, intake, nil, ports.IntakeIdempotency{Key: "pre-0115", Fingerprint: "pre-0115"}); err != nil {
		t.Fatalf("seed intake: %v", err)
	}
	if _, err := store.BeginIntakeAnalysis(ctx, intake.ID, 0, now); err != nil {
		t.Fatalf("begin intake analysis: %v", err)
	}
	request := domain.IntakeAnalysisRequest{
		ID: "ireq-before-0115", IntakeID: intake.ID, ExpectedProposalRevision: 0,
		Status: domain.IntakeAnalysisRequested, CallbackTokenDigest: domain.HashCallbackToken("pre-0115-token"),
		SessionID: "legacy-analysis-session", Harness: domain.HarnessCodex,
		ExpiresAt: now.Add(domain.DefaultIntakeAnalysisRequestTTL), CreatedAt: now,
	}
	if err := store.CreateIntakeAnalysisRequest(ctx, request); err != nil {
		t.Fatalf("seed analysis request: %v", err)
	}

	if err := migrate(db); err != nil {
		t.Fatalf("migrate through 0115: %v", err)
	}
	stored, found, err := store.GetIntakeAnalysisRequest(ctx, request.ID)
	if err != nil || !found {
		t.Fatalf("read historical request after migration: found=%v err=%v", found, err)
	}
	if stored.SessionID != request.SessionID || stored.Harness != request.Harness || stored.CallbackTokenDigest != request.CallbackTokenDigest {
		t.Fatalf("historical request changed across migration: %+v", stored)
	}
	var linked string
	if err := db.QueryRow(`SELECT intelligence_run_id FROM intake_analysis_requests WHERE id = ?`, request.ID).Scan(&linked); err != nil {
		t.Fatalf("read new intelligence link column: %v", err)
	}
	if linked != "" {
		t.Fatalf("historical request was assigned synthetic intelligence run %q", linked)
	}
	var tableCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='intelligence_runs'`).Scan(&tableCount); err != nil {
		t.Fatalf("inspect intelligence_runs table: %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("intelligence_runs table count=%d, want 1", tableCount)
	}
}

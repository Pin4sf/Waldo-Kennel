package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func TestIntakeAnalysisRequestMayLinkExactlyOneIntelligenceRun(t *testing.T) {
	s, err := sqlitetest.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	ctx := context.Background()
	seedProject(t, s, "req-project")
	now := time.Date(2026, 9, 8, 8, 0, 0, 0, time.UTC)
	seedAnalyzingIntake(t, s, "intake-intel-link", "key-intel-link", now)

	request := domain.IntakeAnalysisRequest{
		ID: "ireq-intel-link", IntakeID: "intake-intel-link", ExpectedProposalRevision: 0,
		Status: domain.IntakeAnalysisRequested, CallbackTokenDigest: domain.HashCallbackToken("link-token"),
		ExpiresAt: now.Add(domain.DefaultIntakeAnalysisRequestTTL), CreatedAt: now,
	}
	if err := s.CreateIntakeAnalysisRequest(ctx, request); err != nil {
		t.Fatalf("create request: %v", err)
	}
	if err := s.BindIntakeAnalysisRequestIntelligenceRun(ctx, request.ID, "intel-link-1"); err != nil {
		t.Fatalf("bind intelligence run: %v", err)
	}
	stored, found, err := s.GetIntakeAnalysisRequest(ctx, request.ID)
	if err != nil || !found {
		t.Fatalf("get linked request: found=%v err=%v", found, err)
	}
	if stored.IntelligenceRunID != "intel-link-1" {
		t.Fatalf("intelligence run link = %q", stored.IntelligenceRunID)
	}
	if err := s.BindIntakeAnalysisRequestIntelligenceRun(ctx, request.ID, "intel-link-2"); err == nil {
		t.Fatal("request accepted a second intelligence run link")
	}
}

func TestHistoricalIntakeAnalysisRequestWithoutIntelligenceRunRemainsReadable(t *testing.T) {
	s, err := sqlitetest.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	ctx := context.Background()
	seedProject(t, s, "req-project")
	now := time.Date(2026, 9, 8, 8, 0, 0, 0, time.UTC)
	seedAnalyzingIntake(t, s, "intake-historical", "key-historical", now)

	request := domain.IntakeAnalysisRequest{
		ID: "ireq-historical", IntakeID: "intake-historical", ExpectedProposalRevision: 0,
		Status: domain.IntakeAnalysisRequested, CallbackTokenDigest: domain.HashCallbackToken("historical-token"),
		SessionID: "legacy-session", Harness: domain.HarnessCodex,
		ExpiresAt: now.Add(domain.DefaultIntakeAnalysisRequestTTL), CreatedAt: now,
	}
	if err := s.CreateIntakeAnalysisRequest(ctx, request); err != nil {
		t.Fatalf("create historical-compatible request: %v", err)
	}
	stored, found, err := s.GetIntakeAnalysisRequest(ctx, request.ID)
	if err != nil || !found {
		t.Fatalf("get historical request: found=%v err=%v", found, err)
	}
	if stored.IntelligenceRunID != "" || stored.SessionID != "legacy-session" || stored.Harness != domain.HarnessCodex {
		t.Fatalf("historical provenance changed: %+v", stored)
	}
}

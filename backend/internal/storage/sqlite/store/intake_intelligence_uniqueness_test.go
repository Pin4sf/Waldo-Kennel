package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func TestBindIntakeAnalysisRequestIntelligenceRun_RejectsRunReuse(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 12, 30, 0, 0, time.UTC)
	seedProject(t, s, "reuse-project")
	seedAnalyzingIntakeForProject(t, s, "reuse-project", "reuse-intake", "reuse-intake-key", now)

	makeRequest := func(id domain.IntakeAnalysisRequestID, token string, at time.Time) domain.IntakeAnalysisRequest {
		return domain.IntakeAnalysisRequest{
			ID: id, IntakeID: "reuse-intake", ExpectedProposalRevision: 0,
			Status: domain.IntakeAnalysisRequested, CallbackTokenDigest: domain.HashCallbackToken(token),
			ExpiresAt: at.Add(domain.DefaultIntakeAnalysisRequestTTL), CreatedAt: at,
		}
	}
	requestA := makeRequest("ireq-reuse-a", "reuse-token-a", now)
	requestB := makeRequest("ireq-reuse-b", "reuse-token-b", now.Add(time.Second))
	for _, request := range []domain.IntakeAnalysisRequest{requestA, requestB} {
		if err := s.CreateIntakeAnalysisRequest(ctx, request); err != nil {
			t.Fatalf("create request %s: %v", request.ID, err)
		}
	}

	run := contractIntelligenceRun("intel-reuse", "reuse-project", "reuse-intake", now)
	if err := s.CreateIntelligenceRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	if err := s.BindIntakeAnalysisRequestIntelligenceRun(ctx, requestA.ID, run.ID); err != nil {
		t.Fatalf("bind first request: %v", err)
	}
	if err := s.BindIntakeAnalysisRequestIntelligenceRun(ctx, requestB.ID, run.ID); !errors.Is(err, ports.ErrIntakeAnalysisIntelligenceRunUsed) {
		t.Fatalf("reuse error=%v, want one-run-one-callback error", err)
	}
}

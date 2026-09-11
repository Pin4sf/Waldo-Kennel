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

func seedAnalyzingIntakeForProject(t *testing.T, s *sqlite.Store, projectID string, id domain.IntakeSessionID, key string, now time.Time) {
	t.Helper()
	session := domain.IntakeSession{
		ID: id, SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome,
		ProjectID: domain.ProjectID(projectID), Statement: "Let intelligence propose a Contract",
		Status: domain.IntakeStatusCaptured, CreatedAt: now, UpdatedAt: now,
	}
	if _, err := s.CreateIntake(context.Background(), session, nil, ports.IntakeIdempotency{Key: key, Fingerprint: key}); err != nil {
		t.Fatalf("create intake %s: %v", id, err)
	}
	if _, err := s.BeginIntakeAnalysis(context.Background(), id, 0, now); err != nil {
		t.Fatalf("begin intake analysis %s: %v", id, err)
	}
}

func contractIntelligenceRun(id, projectID string, intakeID domain.IntakeSessionID, now time.Time) domain.IntelligenceRun {
	return domain.IntelligenceRun{
		ID: domain.IntelligenceRunID(id), Kind: domain.IntelligenceRunContractAnalysis,
		ProjectID: domain.ProjectID(projectID), IntakeID: intakeID, SourceRevision: 0,
		InputDigest: domain.DigestSHA256([]byte(id + ":input")), Status: domain.IntelligenceRunRequested, CreatedAt: now,
	}
}

func TestBindIntakeAnalysisRequestIntelligenceRun(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 8, 0, 0, 0, time.UTC)

	setup := func(t *testing.T) (*sqlite.Store, domain.IntakeAnalysisRequest) {
		t.Helper()
		s := sqlitetest.MustOpen(t)
		seedProject(t, s, "req-project")
		seedAnalyzingIntakeForProject(t, s, "req-project", "intake-link", "key-link", now)
		request := domain.IntakeAnalysisRequest{
			ID: "ireq-link", IntakeID: "intake-link", ExpectedProposalRevision: 0,
			Status: domain.IntakeAnalysisRequested, CallbackTokenDigest: domain.HashCallbackToken("link-token"),
			ExpiresAt: now.Add(domain.DefaultIntakeAnalysisRequestTTL), CreatedAt: now,
		}
		if err := s.CreateIntakeAnalysisRequest(ctx, request); err != nil {
			t.Fatalf("create request: %v", err)
		}
		return s, request
	}

	t.Run("accepts matching contract analysis", func(t *testing.T) {
		s, request := setup(t)
		run := contractIntelligenceRun("intel-match", "req-project", request.IntakeID, now)
		if err := s.CreateIntelligenceRun(ctx, run); err != nil {
			t.Fatalf("create run: %v", err)
		}
		if err := s.BindIntakeAnalysisRequestIntelligenceRun(ctx, request.ID, run.ID); err != nil {
			t.Fatalf("bind matching run: %v", err)
		}
		linkedID, found, err := s.GetIntakeAnalysisRequestIntelligenceRun(ctx, request.ID)
		if err != nil || !found || linkedID != run.ID {
			t.Fatalf("linked id=%q found=%v err=%v", linkedID, found, err)
		}
	})

	t.Run("rejects rebind", func(t *testing.T) {
		s, request := setup(t)
		runA := contractIntelligenceRun("intel-a", "req-project", request.IntakeID, now)
		runB := contractIntelligenceRun("intel-b", "req-project", request.IntakeID, now.Add(time.Second))
		for _, run := range []domain.IntelligenceRun{runA, runB} {
			if err := s.CreateIntelligenceRun(ctx, run); err != nil {
				t.Fatalf("create %s: %v", run.ID, err)
			}
		}
		if err := s.BindIntakeAnalysisRequestIntelligenceRun(ctx, request.ID, runA.ID); err != nil {
			t.Fatalf("bind A: %v", err)
		}
		if err := s.BindIntakeAnalysisRequestIntelligenceRun(ctx, request.ID, runB.ID); !errors.Is(err, ports.ErrIntakeAnalysisIntelligenceRunBound) {
			t.Fatalf("rebind error=%v, want write-once provenance error", err)
		}
	})

	t.Run("rejects mismatched intake", func(t *testing.T) {
		s, request := setup(t)
		seedAnalyzingIntakeForProject(t, s, "req-project", "other-intake", "key-other-intake", now)
		run := contractIntelligenceRun("intel-other-intake", "req-project", "other-intake", now)
		if err := s.CreateIntelligenceRun(ctx, run); err != nil {
			t.Fatalf("create run: %v", err)
		}
		if err := s.BindIntakeAnalysisRequestIntelligenceRun(ctx, request.ID, run.ID); !errors.Is(err, ports.ErrIntakeAnalysisIntelligenceLineage) {
			t.Fatalf("mismatched-intake error=%v", err)
		}
	})

	t.Run("rejects wrong project", func(t *testing.T) {
		s, request := setup(t)
		seedProject(t, s, "other-project")
		run := contractIntelligenceRun("intel-other-project", "other-project", request.IntakeID, now)
		if err := s.CreateIntelligenceRun(ctx, run); err != nil {
			t.Fatalf("create run: %v", err)
		}
		if err := s.BindIntakeAnalysisRequestIntelligenceRun(ctx, request.ID, run.ID); !errors.Is(err, ports.ErrIntakeAnalysisIntelligenceLineage) {
			t.Fatalf("wrong-project error=%v", err)
		}
	})

	t.Run("rejects plan draft run", func(t *testing.T) {
		s, request := setup(t)
		seedProviderPlanOutcome(t, s)
		run := domain.IntelligenceRun{
			ID: "intel-plan", Kind: domain.IntelligenceRunPlanDraft,
			ProjectID: "provider-project", OutcomeID: "out-provider-plan", ContractRevisionID: "cr-provider-plan", SourceRevision: 1,
			InputDigest: domain.DigestSHA256([]byte("plan input")), Status: domain.IntelligenceRunRequested, CreatedAt: now,
		}
		if err := s.CreateIntelligenceRun(ctx, run); err != nil {
			t.Fatalf("create plan run: %v", err)
		}
		if err := s.BindIntakeAnalysisRequestIntelligenceRun(ctx, request.ID, run.ID); !errors.Is(err, ports.ErrIntakeAnalysisIntelligenceLineage) {
			t.Fatalf("plan-draft error=%v", err)
		}
	})

	t.Run("rejects closed request", func(t *testing.T) {
		s, request := setup(t)
		run := contractIntelligenceRun("intel-closed", "req-project", request.IntakeID, now)
		if err := s.CreateIntelligenceRun(ctx, run); err != nil {
			t.Fatalf("create run: %v", err)
		}
		if err := s.AnswerIntakeAnalysisRequest(ctx, ports.IntakeAnalysisRequestAnswer{
			RequestID: request.ID, Status: domain.IntakeAnalysisRejected, RawProposal: `{}`, RefusalReason: "closed for test", At: now.Add(time.Minute),
		}); err != nil {
			t.Fatalf("close request: %v", err)
		}
		if err := s.BindIntakeAnalysisRequestIntelligenceRun(ctx, request.ID, run.ID); !errors.Is(err, ports.ErrIntakeAnalysisRequestClosed) {
			t.Fatalf("closed-request error=%v", err)
		}
	})
}

func TestHistoricalIntakeAnalysisRequestWithoutIntelligenceRunRemainsReadable(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 8, 0, 0, 0, time.UTC)
	seedProject(t, s, "req-project")
	seedAnalyzingIntakeForProject(t, s, "req-project", "intake-historical", "key-historical", now)

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
	if stored.SessionID != "legacy-session" || stored.Harness != domain.HarnessCodex {
		t.Fatalf("historical provenance changed: %+v", stored)
	}
	if _, linked, err := s.GetIntakeAnalysisRequestIntelligenceRun(ctx, request.ID); err != nil || linked {
		t.Fatalf("historical row gained a run link: linked=%v err=%v", linked, err)
	}
}

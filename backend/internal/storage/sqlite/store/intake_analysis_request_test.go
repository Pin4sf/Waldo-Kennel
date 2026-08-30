package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/sqlitetest"
)

func seedAnalyzingIntake(t *testing.T, s interface {
	CreateIntake(context.Context, domain.IntakeSession, []domain.IntakeConversationRef, ports.IntakeIdempotency) (ports.IntakeSnapshot, error)
	BeginIntakeAnalysis(context.Context, domain.IntakeSessionID, int64, time.Time) (ports.IntakeSnapshot, error)
}, id domain.IntakeSessionID, key string, now time.Time) {
	t.Helper()
	session := domain.IntakeSession{
		ID: id, SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome,
		ProjectID: "req-project", Statement: "Let an agent draft this",
		Status: domain.IntakeStatusCaptured, CreatedAt: now, UpdatedAt: now,
	}
	if _, err := s.CreateIntake(context.Background(), session, nil, ports.IntakeIdempotency{Key: key, Fingerprint: key}); err != nil {
		t.Fatalf("create intake: %v", err)
	}
	if _, err := s.BeginIntakeAnalysis(context.Background(), id, 0, now); err != nil {
		t.Fatalf("begin analysis: %v", err)
	}
}

// The hazard this whole relation exists to avoid: RecoverInterruptedAnalyses
// turns every `analyzing` intake into a failure at daemon start, which is
// right for an in-process call that cannot outlive the process and fatal for
// an agent that is supposed to. The open request row is what tells them apart.
func TestStartupSweepSparesIntakesAnAgentIsStillAnsweringFor(t *testing.T) {
	s, err := sqlitetest.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	ctx := context.Background()
	seedProject(t, s, "req-project")
	now := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)

	seedAnalyzingIntake(t, s, "intake-agent", "key-agent", now)
	seedAnalyzingIntake(t, s, "intake-inprocess", "key-inprocess", now)

	request := domain.IntakeAnalysisRequest{
		ID: "ireq-1", IntakeID: "intake-agent", ExpectedProposalRevision: 0,
		Status: domain.IntakeAnalysisRequested, CallbackTokenDigest: domain.HashCallbackToken("a-token"),
		ExpiresAt: now.Add(domain.DefaultIntakeAnalysisRequestTTL), CreatedAt: now,
	}
	if err := s.CreateIntakeAnalysisRequest(ctx, request); err != nil {
		t.Fatalf("create request: %v", err)
	}

	swept, err := s.RecoverInterruptedIntakeAnalyses(ctx, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("recover interrupted analyses: %v", err)
	}
	if swept != 1 {
		t.Fatalf("sweep touched %d intakes, want only the in-process one", swept)
	}
	agentIntake, _, err := s.GetIntake(ctx, "intake-agent")
	if err != nil {
		t.Fatalf("get agent intake: %v", err)
	}
	if agentIntake.Session.Status != domain.IntakeStatusAnalyzing {
		t.Fatalf("the sweep reaped an intake an agent is answering for: %q", agentIntake.Session.Status)
	}
	inProcess, _, err := s.GetIntake(ctx, "intake-inprocess")
	if err != nil {
		t.Fatalf("get in-process intake: %v", err)
	}
	if inProcess.Session.Status != domain.IntakeStatusAnalysisFailed {
		t.Fatalf("an interrupted in-process analysis was not failed: %q", inProcess.Session.Status)
	}
}

func TestIntakeAnalysisRequestCallbackIsSingleUseAndTokenIsNeverStored(t *testing.T) {
	s, err := sqlitetest.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	ctx := context.Background()
	seedProject(t, s, "req-project")
	now := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	seedAnalyzingIntake(t, s, "intake-single", "key-single", now)

	const token = "a-token-only-the-agent-holds"
	request := domain.IntakeAnalysisRequest{
		ID: "ireq-single", IntakeID: "intake-single", ExpectedProposalRevision: 0,
		Status: domain.IntakeAnalysisRequested, CallbackTokenDigest: domain.HashCallbackToken(token),
		ExpiresAt: now.Add(domain.DefaultIntakeAnalysisRequestTTL), CreatedAt: now,
	}
	if err := s.CreateIntakeAnalysisRequest(ctx, request); err != nil {
		t.Fatalf("create request: %v", err)
	}
	if err := s.BindIntakeAnalysisRequestSession(ctx, "ireq-single", "sess-1", "codex"); err != nil {
		t.Fatalf("bind session: %v", err)
	}

	stored, found, err := s.GetIntakeAnalysisRequest(ctx, "ireq-single")
	if err != nil || !found {
		t.Fatalf("read back request: found=%v err=%v", found, err)
	}
	if stored.CallbackTokenDigest == token {
		t.Fatal("the raw callback token was stored")
	}
	if !stored.CallbackTokenMatches(token) || stored.CallbackTokenMatches("another-token") {
		t.Fatal("stored digest does not address exactly the minted token")
	}
	if stored.SessionID != "sess-1" || stored.Harness != "codex" {
		t.Fatalf("session binding lost: %+v", stored)
	}

	answer := ports.IntakeAnalysisRequestAnswer{
		RequestID: "ireq-single", Status: domain.IntakeAnalysisFulfilled,
		RawProposal: `{"draft":true}`, At: now.Add(time.Minute),
	}
	if err := s.AnswerIntakeAnalysisRequest(ctx, answer); err != nil {
		t.Fatalf("first answer: %v", err)
	}
	// Single-use: a retrying agent must not overwrite the first answer.
	if err := s.AnswerIntakeAnalysisRequest(ctx, answer); !errors.Is(err, ports.ErrIntakeAnalysisRequestClosed) {
		t.Fatalf("second answer error = %v, want ErrIntakeAnalysisRequestClosed", err)
	}
	// And a closed ask can no longer be re-bound to a different session.
	if err := s.BindIntakeAnalysisRequestSession(ctx, "ireq-single", "sess-2", "opencode"); !errors.Is(err, ports.ErrIntakeAnalysisRequestClosed) {
		t.Fatalf("rebinding a closed request error = %v, want ErrIntakeAnalysisRequestClosed", err)
	}

	closed, _, err := s.GetIntakeAnalysisRequest(ctx, "ireq-single")
	if err != nil {
		t.Fatalf("read back closed request: %v", err)
	}
	if closed.Status != domain.IntakeAnalysisFulfilled || closed.RawProposal != `{"draft":true}` || closed.AnsweredAt == nil {
		t.Fatalf("closed request did not retain its verdict and draft: %+v", closed)
	}
}

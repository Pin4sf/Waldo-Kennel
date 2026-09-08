package intake

import (
	"context"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

func TestAnswerClarificationOfflineNeverInvokesLiveAnalyzer(t *testing.T) {
	now := time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC)
	store := &memoryIntakeStore{}
	analyzer := &scriptedAnalyzer{results: []ports.IntakeAnalysisResult{
		{Clarification: &domain.ClarificationRequest{
			Question: "What does today mean for this Outcome?",
			Reason:   "The date boundary changes which records count toward success.",
		}},
	}}
	service := New(store, analyzer, func() time.Time { return now })

	captured, err := service.Capture(context.Background(), CaptureInput{
		SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome,
		ProjectID: "project-1", Statement: "Show my total focus time today", RequestKey: "capture-offline-answer",
	})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	needsUser, err := service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if needsUser.Session.Status != domain.IntakeStatusNeedsUser {
		t.Fatalf("Analyze() status = %q, want needs_user", needsUser.Session.Status)
	}
	liveCalls := analyzer.calls

	ready, err := service.AnswerClarification(context.Background(), captured.Session.ID, AnswerClarificationInput{
		ExpectedProposalRevision: 0,
		Answer:                   "The Mac's local calendar day.",
		Offline:                  true,
	})
	if err != nil {
		t.Fatalf("AnswerClarification(offline) error = %v", err)
	}
	if ready.Session.Status != domain.IntakeStatusReady || ready.Proposal == nil {
		t.Fatalf("offline answer left status=%q proposal=%v, want ready proposal", ready.Session.Status, ready.Proposal != nil)
	}
	if analyzer.calls != liveCalls {
		t.Fatalf("offline clarification invoked live analyzer: calls %d -> %d", liveCalls, analyzer.calls)
	}
	if len(store.requests) != 0 {
		t.Fatalf("offline clarification opened %d durable callback request(s)", len(store.requests))
	}
}

func TestAnalyzeWithoutLiveAnalyzerUsesDeterministicFloor(t *testing.T) {
	now := time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC)
	store := &memoryIntakeStore{}
	service := New(store, nil, func() time.Time { return now })

	captured, err := service.Capture(context.Background(), CaptureInput{
		SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome,
		ProjectID: "project-1", Statement: "Add a visible README section", RequestKey: "capture-no-live-analyzer",
	})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	ready, err := service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0})
	if err != nil {
		t.Fatalf("Analyze() with no live analyzer error = %v", err)
	}
	if ready.Session.Status != domain.IntakeStatusReady || ready.Proposal == nil {
		t.Fatalf("deterministic floor status=%q proposal=%v", ready.Session.Status, ready.Proposal != nil)
	}
	if len(store.requests) != 0 {
		t.Fatalf("deterministic floor opened %d durable callback request(s)", len(store.requests))
	}
}

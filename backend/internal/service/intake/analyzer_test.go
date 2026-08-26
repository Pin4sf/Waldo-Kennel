package intake

import (
	"context"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

func TestRuleBasedAnalyzerAdvancesSimpleOutcomeAndBoundsMaterialQuestion(t *testing.T) {
	analyzer := NewRuleBasedAnalyzer()
	simple, err := analyzer.Analyze(context.Background(), ports.IntakeAnalysisInput{Session: domain.IntakeSession{Statement: "Add keyboard navigation to settings"}})
	if err != nil || simple.Proposal == nil || simple.Clarification != nil {
		t.Fatalf("simple analysis = %+v err=%v", simple, err)
	}
	ambiguous, err := analyzer.Analyze(context.Background(), ports.IntakeAnalysisInput{Session: domain.IntakeSession{Statement: "Show focus time today"}})
	if err != nil || ambiguous.Clarification == nil || ambiguous.Proposal != nil {
		t.Fatalf("ambiguous analysis = %+v err=%v", ambiguous, err)
	}
	answered, err := analyzer.Analyze(context.Background(), ports.IntakeAnalysisInput{Session: domain.IntakeSession{Statement: "Show focus time today"}, ClarificationText: "Use the Mac local calendar day"})
	if err != nil || answered.Proposal == nil || answered.Clarification != nil || answered.Proposal.TemporalCondition == nil {
		t.Fatalf("answered analysis = %+v err=%v", answered, err)
	}
}

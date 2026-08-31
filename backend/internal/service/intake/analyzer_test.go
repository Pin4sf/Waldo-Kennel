package intake

import (
	"context"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// inlineResult asserts the offline baseline answered at once. It is the one
// analyzer that may: it performs no model call and no I/O, so a deferred
// ticket from it would mean something started work that should not have.
func inlineResult(t *testing.T, ticket ports.IntakeAnalysisTicket, err error, label string) ports.IntakeAnalysisResult {
	t.Helper()
	if err != nil {
		t.Fatalf("%s analysis err=%v", label, err)
	}
	if ticket.Inline == nil {
		t.Fatalf("%s analysis deferred; the rule-based baseline must answer inline: %+v", label, ticket)
	}
	if ticket.SessionID != "" {
		t.Fatalf("%s analysis started session %q; the rule-based baseline spawns nothing", label, ticket.SessionID)
	}
	return *ticket.Inline
}

func TestRuleBasedAnalyzerAdvancesSimpleOutcomeAndBoundsMaterialQuestion(t *testing.T) {
	analyzer := NewRuleBasedAnalyzer()
	ticket, err := analyzer.Analyze(context.Background(), ports.IntakeAnalysisInput{Session: domain.IntakeSession{Statement: "Add keyboard navigation to settings"}})
	simple := inlineResult(t, ticket, err, "simple")
	if simple.Proposal == nil || simple.Clarification != nil {
		t.Fatalf("simple analysis = %+v", simple)
	}
	ticket, err = analyzer.Analyze(context.Background(), ports.IntakeAnalysisInput{Session: domain.IntakeSession{Statement: "Show focus time today"}})
	ambiguous := inlineResult(t, ticket, err, "ambiguous")
	if ambiguous.Clarification == nil || ambiguous.Proposal != nil {
		t.Fatalf("ambiguous analysis = %+v", ambiguous)
	}
	ticket, err = analyzer.Analyze(context.Background(), ports.IntakeAnalysisInput{Session: domain.IntakeSession{Statement: "Show focus time today"}, ClarificationText: "Use the Mac local calendar day"})
	answered := inlineResult(t, ticket, err, "answered")
	if answered.Proposal == nil || answered.Clarification != nil || answered.Proposal.TemporalCondition == nil {
		t.Fatalf("answered analysis = %+v", answered)
	}
}

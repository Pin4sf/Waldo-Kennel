package intake

import (
	"context"
	"regexp"
	"strings"
	"unicode"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// RuleBasedAnalyzer is the offline-truth baseline for shared intake. It makes
// a conservative editable proposal without claiming model analysis. A future
// analyzer may enrich the same typed contract behind the port.
//
// It answers synchronously, so it fills the ticket's Inline field. That is not
// a shortcut around the no-synchronous-model-call rule — it performs no model
// call and no I/O at all, which is exactly why it can answer at once, and why
// it remains the floor an agent-backed analyzer degrades to.
type RuleBasedAnalyzer struct{}

// NewRuleBasedAnalyzer constructs the deterministic offline baseline.
func NewRuleBasedAnalyzer() *RuleBasedAnalyzer { return &RuleBasedAnalyzer{} }

var ambiguousLocalDay = regexp.MustCompile(`(?i)\b(today|tonight|this morning|this evening)\b`)

var _ ports.IntakeAnalyzer = (*RuleBasedAnalyzer)(nil)

// Analyze returns one bounded question or an editable typed proposal, always
// inline.
func (*RuleBasedAnalyzer) Analyze(_ context.Context, input ports.IntakeAnalysisInput) (ports.IntakeAnalysisTicket, error) {
	statement := strings.TrimSpace(input.Session.Statement)
	if ambiguousLocalDay.MatchString(statement) && strings.TrimSpace(input.ClarificationText) == "" {
		return inline(ports.IntakeAnalysisResult{Clarification: &domain.ClarificationRequest{
			Question:            "Which time boundary should this Outcome use?",
			Reason:              "The boundary changes what counts toward the result.",
			Recommendation:      "Use this Mac's local calendar day.",
			Alternatives:        []string{"Mac local calendar day", "Rolling 24 hours"},
			DeferralConsequence: "The proposal will use the Mac's local calendar day.",
		}}), nil
	}
	title := outcomeTitle(statement)
	facet := domain.ContractFacet{Kind: inferFacet(statement), Summary: title}
	proposal := &domain.OutcomeContractProposal{
		Title: title, DesiredState: statement,
		Criteria:         []domain.ProposedCriterion{{Text: "The requested result is observable and matches the confirmed desired state.", EvidenceExpected: []string{"A deterministic check or owner walkthrough demonstrates the result."}}},
		ReviewMethod:     "Run the relevant deterministic checks, then complete an owner walkthrough.",
		AuthorityCeiling: domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true, ExecuteLocal: true},
		StopConditions:   []string{"Stop before any remote, publishing, deployment, or external effect that the owner has not explicitly authorized."},
		Facets:           []domain.ContractFacet{facet},
	}
	if answer := strings.TrimSpace(input.ClarificationText); answer != "" {
		proposal.TemporalCondition = &answer
	}
	return inline(ports.IntakeAnalysisResult{Proposal: proposal}), nil
}

func inline(result ports.IntakeAnalysisResult) ports.IntakeAnalysisTicket {
	return ports.IntakeAnalysisTicket{Inline: &result}
}

func outcomeTitle(statement string) string {
	words := strings.Fields(strings.TrimSpace(statement))
	if len(words) > 8 {
		words = words[:8]
	}
	title := strings.TrimRight(strings.Join(words, " "), ".!? ")
	if title == "" {
		return "New Outcome"
	}
	runes := []rune(title)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func inferFacet(statement string) domain.ContractFacetKind {
	lower := strings.ToLower(statement)
	for _, token := range []string{"research", "investigate", "compare", "study"} {
		if strings.Contains(lower, token) {
			return domain.ContractFacetResearch
		}
	}
	for _, token := range []string{"design", "screen", "interface", "layout"} {
		if strings.Contains(lower, token) {
			return domain.ContractFacetDesign
		}
	}
	for _, token := range []string{"document", "write", "guide"} {
		if strings.Contains(lower, token) {
			return domain.ContractFacetDocumentation
		}
	}
	return domain.ContractFacetSoftware
}

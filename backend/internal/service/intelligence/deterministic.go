// Package intelligence contains non-authoritative reasoning providers and the
// small adapters that connect them to Outcome/Intake services.
package intelligence

import (
	"context"
	"fmt"
	"sort"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	intakevc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/intake"
)

const DeterministicProviderID domain.IntelligenceProviderID = "deterministic-local"

// DeterministicProvider is the always-available intelligence floor. It performs
// no provider/tool I/O, creates no execution session, and returns structured
// proposal material synchronously behind the same port a future local/API
// reasoner implements.
type DeterministicProvider struct {
	contract *intakevc.RuleBasedAnalyzer
}

func NewDeterministicProvider() *DeterministicProvider {
	return &DeterministicProvider{contract: intakevc.NewRuleBasedAnalyzer()}
}

func (*DeterministicProvider) ID() domain.IntelligenceProviderID { return DeterministicProviderID }

func (p *DeterministicProvider) AnalyzeContract(ctx context.Context, request ports.ContractIntelligenceRequest) (ports.ContractIntelligenceResponse, error) {
	if p == nil || p.contract == nil {
		return ports.ContractIntelligenceResponse{}, fmt.Errorf("deterministic contract intelligence is unavailable")
	}
	ticket, err := p.contract.Analyze(ctx, ports.IntakeAnalysisInput{
		Session: request.Session,
		ConversationRefs: request.ConversationRefs,
		PreviousProposal: request.PreviousProposal,
		Clarification: request.Clarification,
		ClarificationText: request.ClarificationText,
		Defer: func(context.Context) (ports.IntakeCallback, error) {
			return ports.IntakeCallback{}, fmt.Errorf("deterministic intelligence cannot defer")
		},
	})
	if err != nil {
		return ports.ContractIntelligenceResponse{}, err
	}
	if ticket.Inline == nil {
		return ports.ContractIntelligenceResponse{}, fmt.Errorf("deterministic contract intelligence returned no inline result")
	}
	return ports.ContractIntelligenceResponse{
		Result: *ticket.Inline,
		Provenance: ports.IntelligenceProvenance{EffectiveProvider: DeterministicProviderID},
	}, nil
}

func (*DeterministicProvider) DraftPlan(_ context.Context, request ports.PlanIntelligenceRequest) (ports.PlanIntelligenceResponse, error) {
	if err := request.Contract.Validate(); err != nil {
		return ports.PlanIntelligenceResponse{}, fmt.Errorf("draft plan contract: %w", err)
	}
	if request.Outcome.ID != request.Contract.OutcomeID {
		return ports.PlanIntelligenceResponse{}, fmt.Errorf("draft plan outcome does not match contract")
	}
	if len(request.CriterionAliases) == 0 {
		return ports.PlanIntelligenceResponse{}, fmt.Errorf("draft plan requires criterion aliases")
	}

	aliases := make([]string, 0, len(request.CriterionAliases))
	for alias := range request.CriterionAliases {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	evidence := make([]string, 0)
	for _, expectation := range request.Contract.EvidenceExpectations {
		evidence = append(evidence, expectation.Descriptions...)
	}
	if len(evidence) == 0 {
		// The deterministic floor should still be criterion-specific rather
		// than emitting a generic proof placeholder. These are evidence ideas,
		// not proof or acceptance.
		for _, criterion := range request.Contract.Criteria {
			evidence = append(evidence, criterion.Text)
		}
	}
	if len(evidence) == 0 {
		evidence = append(evidence, request.Contract.SuccessCriteria...)
	}

	proposal := domain.PlanDraftProposal{
		Summary: "Execute the confirmed Contract as one bounded reviewable unit.",
		WorkUnits: []domain.PlanDraftWorkUnit{{
			Key: "deliver",
			Title: request.Outcome.Title,
			OutputSummary: "A reviewable result that satisfies the confirmed Contract.",
			CriteriaCovered: aliases,
			EvidenceIdeas: evidence,
		}},
	}
	if err := proposal.Validate(); err != nil {
		return ports.PlanIntelligenceResponse{}, err
	}
	return ports.PlanIntelligenceResponse{
		Proposal: proposal,
		Provenance: ports.IntelligenceProvenance{EffectiveProvider: DeterministicProviderID},
	}, nil
}

var _ ports.IntelligenceProvider = (*DeterministicProvider)(nil)

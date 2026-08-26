package ports

import (
	"context"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

// IntakeAnalysisInput is the bounded, provenance-bearing analyzer packet. It
// contains conversation identifiers only; transcript bodies remain in their
// owning conversation aggregate.
type IntakeAnalysisInput struct {
	Session           domain.IntakeSession
	ConversationRefs  []domain.IntakeConversationRef
	PreviousProposal  *domain.OutcomeContractProposal
	Clarification     *domain.ClarificationRequest
	ClarificationText string
}

// IntakeAnalysisResult is structured proposal output. Exactly one of Proposal
// or Clarification may be present.
type IntakeAnalysisResult struct {
	Proposal      *domain.OutcomeContractProposal
	Clarification *domain.ClarificationRequest
}

// IntakeAnalyzer proposes understanding but owns no canonical responsibility.
type IntakeAnalyzer interface {
	Analyze(context.Context, IntakeAnalysisInput) (IntakeAnalysisResult, error)
}

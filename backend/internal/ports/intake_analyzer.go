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

// IntakeAnalysisTicket is what an analyzer returns immediately.
//
// It mirrors DecompositionProposalTicket, as decomposition_proposer.go's own
// comment instructs, because the two seams answer the same question: the
// daemon makes NO synchronous model call, so an agent-backed analyzer cannot
// return a proposal from Analyze — it starts work that answers later over the
// API.
//
// Inline is the exception that keeps a deterministic analyzer a first-class
// implementation of this port rather than a special case the service branches
// around: the offline rule-based baseline genuinely has its answer already, so
// it returns one. An agent-backed analyzer leaves Inline nil and answers on
// its callback.
//
// Exactly one of Inline or SessionID is meaningful. Both empty means the
// analyzer started nothing and produced nothing, which is a programming error
// rather than a state an intake can sit in.
type IntakeAnalysisTicket struct {
	// Inline is the answer when the analyzer had one synchronously.
	Inline *IntakeAnalysisResult
	// SessionID names the bounded session started to answer, when one was.
	SessionID string
	// Detail explains what was started, in the owner's terms.
	Detail string
}

// IntakeAnalyzer proposes understanding but owns no canonical responsibility.
// Whatever it produces passes exactly the same validation as a hand-authored
// proposal, and only the owner confirms.
type IntakeAnalyzer interface {
	Analyze(context.Context, IntakeAnalysisInput) (IntakeAnalysisTicket, error)
}

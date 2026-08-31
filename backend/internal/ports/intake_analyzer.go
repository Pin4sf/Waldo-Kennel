package ports

import (
	"context"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

// IntakeCallback is where a deferred answer goes: the durable request it
// answers, the scoping token that addresses it, and when it stops being
// accepted.
//
// It carries no URL. The service that mints it owns durable state, not HTTP;
// the daemon layer that spawns the agent is what knows the loopback origin and
// renders the endpoint into the brief — the same split decomposition uses.
type IntakeCallback struct {
	RequestID domain.IntakeAnalysisRequestID
	// Token is SCOPING, NOT AUTHENTICATION. See IntakeAnalysisRequest.
	Token     string
	ExpiresAt time.Time
}

// IntakeAnalysisInput is the bounded, provenance-bearing analyzer packet. It
// contains conversation identifiers only; transcript bodies remain in their
// owning conversation aggregate.
type IntakeAnalysisInput struct {
	Session           domain.IntakeSession
	ConversationRefs  []domain.IntakeConversationRef
	PreviousProposal  *domain.OutcomeContractProposal
	Clarification     *domain.ClarificationRequest
	ClarificationText string

	// Defer opens the durable request this analysis will answer on and returns
	// where to answer. It is a capability rather than a field because minting
	// is a durable write with consequences: an analyzer that answers inline
	// never calls it, and no request row is written for an analysis no agent
	// was ever asked to do.
	//
	// The request is persisted before this returns, so an agent spawned
	// afterwards always has somewhere to answer. Call it exactly once, and
	// only when about to defer.
	Defer func(context.Context) (IntakeCallback, error)
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
	// Harness names which agent was asked, so a waiting state can say who is
	// working rather than showing an anonymous spinner.
	Harness domain.AgentHarness
	// Detail explains what was started, in the owner's terms.
	Detail string
}

// IntakeAnalyzer proposes understanding but owns no canonical responsibility.
// Whatever it produces passes exactly the same validation as a hand-authored
// proposal, and only the owner confirms.
type IntakeAnalyzer interface {
	Analyze(context.Context, IntakeAnalysisInput) (IntakeAnalysisTicket, error)
}

// AnalystSessionReaper ends the bounded session spawned to answer an ask, once
// that ask is closed.
//
// A proposing session has exactly one job and is finished the moment its
// answer lands — or the moment the ask is refused, expired, or cancelled.
// Leaving it running is not merely untidy: it holds a worktree and a runtime
// name, and the next spawn for the same project collides with it.
//
// Reaping is best-effort by construction. The ask is already durably closed
// before this runs, so a failure to kill leaves a stray process, never an
// inconsistent record.
type AnalystSessionReaper interface {
	Kill(ctx context.Context, sessionID string) error
}

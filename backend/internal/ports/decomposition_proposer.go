package ports

import (
	"context"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

// DecompositionProposalInput is the bounded packet a proposer is given. It
// carries the parent's contract and its STABLE criterion identities, because
// contribution is criterion-bound: a proposer that had to invent identities
// could not produce a bindable proposal.
type DecompositionProposalInput struct {
	RequestID        domain.DecompositionRequestID
	ProjectID        domain.ProjectID
	OutcomeID        domain.OutcomeID
	OutcomeTitle     string
	Contract         domain.ContractRevision
	CallbackURL      string
	CallbackToken    string
	MaxContributions int
	ParentAuthority  domain.ProposedAuthority
}

// DecompositionProposalTicket is what a proposer returns immediately. A
// proposal is NOT part of it: the daemon has no synchronous model call, so an
// agent-backed proposer starts work that answers later over the API.
type DecompositionProposalTicket struct {
	// SessionID names the bounded session started to answer, when one was.
	SessionID string
	// Detail explains what was started, in the owner's terms.
	Detail string
}

// DecompositionProposer starts work that proposes a decomposition. It owns no
// canonical responsibility: whatever it produces is validated by exactly the
// same gates as a hand-authored proposal, and only the owner authorizes.
//
// This mirrors IntakeAnalyzer deliberately. When intake's rule-based analyzer
// gains a model-backed implementation it should follow this same shape rather
// than inventing a second seam.
type DecompositionProposer interface {
	Propose(context.Context, DecompositionProposalInput) (DecompositionProposalTicket, error)
}

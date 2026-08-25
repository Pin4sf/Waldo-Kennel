package ports

import (
	"context"
	"fmt"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

// IntakeIdempotency binds a request key to a normalized input fingerprint.
type IntakeIdempotency struct {
	Key         string
	Fingerprint string
}

// IntakeSnapshot is the complete durable read model for one intake.
type IntakeSnapshot struct {
	Session           domain.IntakeSession
	ConversationRefs  []domain.IntakeConversationRef
	Proposal          *domain.OutcomeContractProposal
	Clarification     *domain.ClarificationRequest
	ConfirmedOutcome  *domain.Outcome
	ConfirmedContract *domain.ContractRevision
}

// IntakeRevisionConflictError reports an optimistic concurrency mismatch.
type IntakeRevisionConflictError struct {
	IntakeID         domain.IntakeSessionID
	ExpectedRevision int64
	CurrentRevision  int64
}

func (err *IntakeRevisionConflictError) Error() string {
	return fmt.Sprintf("intake %s proposal revision conflict: expected %d, current %d", err.IntakeID, err.ExpectedRevision, err.CurrentRevision)
}

// IntakeIdempotencyConflictError reports reuse with different input.
type IntakeIdempotencyConflictError struct{ Key string }

func (err *IntakeIdempotencyConflictError) Error() string {
	return fmt.Sprintf("intake idempotency key %q is already bound to different input", err.Key)
}

// IntakeStore persists the shared state machine and immutable proposal
// revisions. Every mutation is guarded by the expected proposal revision.
type IntakeStore interface {
	GetProject(context.Context, string) (domain.ProjectRecord, bool, error)
	CreateIntake(context.Context, domain.IntakeSession, []domain.IntakeConversationRef, IntakeIdempotency) (IntakeSnapshot, error)
	GetIntake(context.Context, domain.IntakeSessionID) (IntakeSnapshot, bool, error)
	BeginIntakeAnalysis(context.Context, domain.IntakeSessionID, int64, time.Time) (IntakeSnapshot, error)
	CompleteIntakeWithProposal(context.Context, domain.IntakeSessionID, int64, domain.OutcomeContractProposal, time.Time) (IntakeSnapshot, error)
	CompleteIntakeWithClarification(context.Context, domain.IntakeSessionID, int64, domain.ClarificationRequest, time.Time) (IntakeSnapshot, error)
	AnswerIntakeClarification(context.Context, domain.IntakeSessionID, int64, string, time.Time) (IntakeSnapshot, error)
	AppendIntakeProposalRevision(context.Context, domain.IntakeSessionID, int64, domain.OutcomeContractProposal, time.Time) (IntakeSnapshot, error)
	EnsureWorkResponsibilitySpace(context.Context, domain.ProjectID) (domain.ResponsibilitySpace, error)
	ConfirmIntakeWithOutcome(context.Context, domain.IntakeSessionID, int64, domain.Outcome, domain.ContractRevision, IntakeIdempotency, time.Time) (IntakeSnapshot, error)
	FailIntakeAnalysis(context.Context, domain.IntakeSessionID, int64, string, time.Time) (IntakeSnapshot, error)
}

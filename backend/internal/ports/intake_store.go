package ports

import (
	"context"
	"errors"
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
	CancelIntake(context.Context, domain.IntakeSessionID, int64, string, time.Time) (IntakeSnapshot, error)
	RecoverInterruptedIntakeAnalyses(context.Context, time.Time) (int64, error)

	// CreateIntakeAnalysisRequest opens one durable ask for an agent-authored
	// Contract proposal. It is written BEFORE the agent is spawned.
	CreateIntakeAnalysisRequest(context.Context, domain.IntakeAnalysisRequest) error

	// GetIntakeAnalysisRequest reads one ask; ok=false when absent.
	GetIntakeAnalysisRequest(context.Context, domain.IntakeAnalysisRequestID) (domain.IntakeAnalysisRequest, bool, error)

	// LatestIntakeAnalysisRequest returns an intake's newest ask of any status,
	// which is what the waiting state and the refused-draft view read.
	LatestIntakeAnalysisRequest(context.Context, domain.IntakeSessionID) (domain.IntakeAnalysisRequest, bool, error)

	// AnswerIntakeAnalysisRequest closes an open ask one way, retaining the
	// draft whatever the verdict. Answering a closed ask changes nothing and
	// returns ErrIntakeAnalysisRequestClosed — that guard is what makes the
	// callback single-use.
	AnswerIntakeAnalysisRequest(context.Context, IntakeAnalysisRequestAnswer) error

	// ListOpenIntakeAnalysisRequests returns every unanswered ask so a
	// durable deadline can be enforced at startup and on a timer.
	ListOpenIntakeAnalysisRequests(context.Context) ([]domain.IntakeAnalysisRequest, error)

	// BindIntakeAnalysisRequestSession records which spawned session and
	// harness are answering, so a restart can tell what was working on this
	// and the waiting state can name who.
	BindIntakeAnalysisRequestSession(context.Context, domain.IntakeAnalysisRequestID, string, domain.AgentHarness) error
}

// IntakeAnalysisRequestAnswer closes one ask. RawProposal is retained whatever
// the verdict, so a refused draft stays inspectable.
type IntakeAnalysisRequestAnswer struct {
	RequestID     domain.IntakeAnalysisRequestID
	Status        domain.IntakeAnalysisRequestStatus
	RawProposal   string
	RefusalReason string
	At            time.Time
}

// ErrIntakeAnalysisRequestClosed reports an answer to an ask that is no longer
// open. It is the single-use guard, not an unexpected failure.
var ErrIntakeAnalysisRequestClosed = errors.New("intake analysis request is not open")

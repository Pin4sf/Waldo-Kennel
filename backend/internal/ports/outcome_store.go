package ports

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// OutcomeConflictError reports an optimistic-concurrency failure on the
// Outcome's immutable Contract pointer.
type OutcomeConflictError struct {
	OutcomeID           domain.OutcomeID
	ExpectedRevisionNum int64
	CurrentRevisionNum  int64
}

func (e *OutcomeConflictError) Error() string {
	return fmt.Sprintf("outcome %s moved past contract revision %s (current %s)",
		e.OutcomeID, strconv.FormatInt(e.ExpectedRevisionNum, 10), strconv.FormatInt(e.CurrentRevisionNum, 10))
}

// AttemptAdmission is the exact canonical identity storage needs to create one
// queued Attempt. The scheduler/service must choose the approved WorkUnit
// before this boundary; storage never indexes into a Plan to guess what runs.
type AttemptAdmission struct {
	OutcomeID              domain.OutcomeID
	PlanRevisionID         domain.PlanRevisionID
	WorkUnitID             domain.WorkUnitID
	ContractRevisionNumber int64
	RequestKey             string
	FenceSubject           string
	At                     time.Time
}

// OutcomeStore is the canonical durable boundary for Outcome control-plane
// state. Implementations own atomic writes; services own policy and authority.
type OutcomeStore interface {
	EnsureWorkResponsibilitySpace(context.Context, domain.ProjectID) (domain.ResponsibilitySpace, error)
	FindOutcomeByIdempotencyKey(context.Context, string) (domain.Outcome, bool, error)
	CreateOutcomeWithContract(context.Context, domain.Outcome, domain.ContractRevision, string) error
	GetOutcome(context.Context, domain.OutcomeID) (domain.Outcome, bool, error)
	ListOutcomesByProject(context.Context, domain.ProjectID) ([]domain.Outcome, error)
	AppendContractRevision(context.Context, domain.OutcomeID, int64, domain.ContractRevision) (int64, error)
	ListContractRevisions(context.Context, domain.OutcomeID) ([]domain.ContractRevision, error)

	CreateContributionWithContract(context.Context, domain.Outcome, domain.ContractRevision, []domain.ContributionLink, string) error
	ListContributingOutcomes(context.Context, domain.OutcomeID) ([]domain.Outcome, error)
	ListContributionLinksForParent(context.Context, domain.OutcomeID) ([]domain.ContributionLink, error)
	ListContributionLinksForChild(context.Context, domain.OutcomeID) ([]domain.ContributionLink, error)

	AppendDecompositionRevision(context.Context, domain.DecompositionRevision) (domain.DecompositionRevision, error)
	AuthorizeDecompositionRevision(context.Context, domain.OutcomeID, domain.DecompositionRevisionID, []AuthorizedContribution, time.Time) error
	GetDecompositionRevision(context.Context, domain.OutcomeID, domain.DecompositionRevisionID) (domain.DecompositionRevision, bool, error)
	LatestDecompositionRevision(context.Context, domain.OutcomeID) (domain.DecompositionRevision, bool, error)
	AppendContributionDependencyWaiver(context.Context, domain.ContributionDependencyWaiver) error
	ListContributionDependencyWaivers(context.Context, domain.DecompositionRevisionID) ([]domain.ContributionDependencyWaiver, error)
	CreateDecompositionRequest(context.Context, domain.DecompositionRequest) error
	GetDecompositionRequest(context.Context, domain.DecompositionRequestID) (domain.DecompositionRequest, bool, error)
	LatestDecompositionRequest(context.Context, domain.OutcomeID) (domain.DecompositionRequest, bool, error)
	AnswerDecompositionRequest(context.Context, DecompositionRequestAnswer) error
	ListOpenDecompositionRequests(context.Context) ([]domain.DecompositionRequest, error)
	BindDecompositionRequestSession(context.Context, domain.DecompositionRequestID, string) error

	AppendPlanRevision(context.Context, domain.OutcomeID, domain.PlanRevision) (domain.PlanRevision, error)
	LatestProposedPlanRevision(context.Context, domain.OutcomeID, int64) (domain.PlanRevision, bool, error)
	GetPlanRevision(context.Context, domain.OutcomeID, domain.PlanRevisionID) (domain.PlanRevision, bool, error)
	GetLatestPlanRevision(context.Context, domain.OutcomeID) (domain.PlanRevision, bool, error)
	ApprovePlanRevision(context.Context, domain.OutcomeID, domain.PlanRevisionID) (domain.PlanRevision, bool, error)

	GetOutcomeProjectID(context.Context, domain.OutcomeID) (domain.ProjectID, bool, error)
	FindAttemptByIdempotencyKey(context.Context, string) (domain.Attempt, bool, error)
	CreateAttemptWithFence(context.Context, AttemptAdmission) (domain.Attempt, error)
	GetAttempt(context.Context, domain.OutcomeID, domain.AttemptID) (domain.Attempt, bool, error)
	ListAttempts(context.Context, domain.OutcomeID) ([]domain.Attempt, error)
	TransitionAttemptStatus(context.Context, domain.OutcomeID, domain.AttemptID, domain.AttemptStatus, domain.AttemptStatus, time.Time) (int64, error)
	ListAttemptsByStatus(context.Context, domain.AttemptStatus) ([]domain.Attempt, error)
	BindAttemptSession(context.Context, domain.AttemptSessionRef) (domain.AttemptSessionRef, error)
	LatestAttemptSessionRef(context.Context, domain.AttemptID) (domain.AttemptSessionRef, bool, error)
	ListAttemptSessionRefs(context.Context, domain.AttemptID) ([]domain.AttemptSessionRef, error)
	AppendAttemptObservation(context.Context, domain.AttemptID, string, string, time.Time) (domain.AttemptObservation, error)
	ListAttemptObservations(context.Context, domain.AttemptID) ([]domain.AttemptObservation, error)
	OpenFenceForSubject(context.Context, string) (domain.AttemptFence, bool, error)
	ReleaseFenceForAttempt(context.Context, domain.AttemptID, string, time.Time) (int64, error)
	RenewFenceForAttempt(context.Context, domain.AttemptID, time.Time) (int64, error)
	CreateRecoveryReceipt(context.Context, domain.AttemptRecoveryReceipt) error
	ListRecoveryReceipts(context.Context, domain.AttemptID) ([]domain.AttemptRecoveryReceipt, error)
}

// AttemptFenceHeldError reports exclusive worktree custody held by another
// Attempt. Admission must leave zero new rows when this is returned.
type AttemptFenceHeldError struct {
	Subject   string
	Holder    domain.AttemptID
	OutcomeID domain.OutcomeID
}

func (e *AttemptFenceHeldError) Error() string {
	return fmt.Sprintf("worktree subject %s is fenced by attempt %s", e.Subject, e.Holder)
}

// AttemptReplayError carries the canonical Attempt for an already-delivered
// idempotency key; callers must serve it and must not spawn again.
type AttemptReplayError struct{ Attempt domain.Attempt }

func (e *AttemptReplayError) Error() string {
	return fmt.Sprintf("attempt %s was already admitted for this request key", e.Attempt.ID)
}

// AuthorizedContribution is the atomic persistence payload created from one
// owner-authorized decomposition proposal.
type AuthorizedContribution struct {
	Ref     string
	Outcome domain.Outcome
	First   domain.ContractRevision
	Links   []domain.ContributionLink
}

var ErrDecompositionNotProposed = errors.New("decomposition is not an open proposal")

type DecompositionRequestAnswer struct {
	RequestID       domain.DecompositionRequestID
	Status          domain.DecompositionRequestStatus
	RawProposal     string
	RefusalReason   string
	DecompositionID domain.DecompositionRevisionID
	At              time.Time
}

var ErrDecompositionRequestClosed = errors.New("decomposition request is not open")

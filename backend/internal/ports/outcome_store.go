package ports

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

// OutcomeConflictError reports an optimistic-concurrency failure: the caller
// addressed the Outcome through a revision pointer that is no longer current.
// Services pass it through; controllers map it to the shared conflict envelope
// (HTTP 409).
type OutcomeConflictError struct {
	OutcomeID           domain.OutcomeID
	ExpectedRevisionNum int64
	CurrentRevisionNum  int64
}

func (e *OutcomeConflictError) Error() string {
	return fmt.Sprintf("outcome %s moved past contract revision %s (current %s)",
		e.OutcomeID, strconv.FormatInt(e.ExpectedRevisionNum, 10), strconv.FormatInt(e.CurrentRevisionNum, 10))
}

// OutcomeStore is the durable boundary for canonical responsibility contracts.
// Implementations live in storage; services and controllers never touch SQLite.
//
// Writes are atomic multi-step transactions so a crash can never strand an
// Outcome without its contract revision or advance a revision pointer past a
// row that was never written. Contract revisions are append-only history:
// implementations assign revision numbers inside the write transaction, so
// concurrent revisers serialize and a loser sees *OutcomeConflictError with
// nothing persisted.
type OutcomeStore interface {
	// EnsureWorkResponsibilitySpace resolves the project-backed Work space,
	// creating it on first use. The same space is returned for every call for
	// the lifetime of the project.
	EnsureWorkResponsibilitySpace(ctx context.Context, projectID domain.ProjectID) (domain.ResponsibilitySpace, error)

	// FindOutcomeByIdempotencyKey resolves a previously delivered create
	// request by its client-supplied key; ok=false when unknown.
	FindOutcomeByIdempotencyKey(ctx context.Context, key string) (domain.Outcome, bool, error)

	// CreateOutcomeWithContract atomically persists the Outcome together with
	// its ContractRevision 1 and advances the current-revision pointer to it.
	// A duplicate request key surfaces as a unique-constraint error; callers
	// resolve replays through FindOutcomeByIdempotencyKey beforehand.
	CreateOutcomeWithContract(ctx context.Context, outcome domain.Outcome, first domain.ContractRevision, requestKey string) error

	// GetOutcome reads one Outcome by id; ok=false when absent.
	GetOutcome(ctx context.Context, id domain.OutcomeID) (domain.Outcome, bool, error)

	// ListOutcomesByProject returns canonical Outcomes belonging to the
	// project's Work responsibility space in stable creation order. A project
	// without a Work space or Outcomes returns an empty list.
	ListOutcomesByProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Outcome, error)

	// AppendContractRevision atomically appends one immutable revision and
	// swaps the current-revision pointer from expected to the newly assigned
	// number, which it returns. When expected no longer names the current
	// revision it returns *OutcomeConflictError and persists nothing.
	AppendContractRevision(ctx context.Context, id domain.OutcomeID, expectedCurrent int64, revision domain.ContractRevision) (int64, error)

	// ListContractRevisions returns the full immutable history ordered by
	// ascending revision number.
	ListContractRevisions(ctx context.Context, id domain.OutcomeID) ([]domain.ContractRevision, error)

	// CreateContributionWithContract atomically persists one contributing
	// Outcome, its ContractRevision 1, and every criterion binding, so a child
	// can never exist claiming nothing (ADR 0007). Storage enforces the depth
	// cap, the single-parent-revision rule, and link immutability; the service
	// enforces authority containment and criterion identity beforehand.
	CreateContributionWithContract(ctx context.Context, child domain.Outcome, first domain.ContractRevision, links []domain.ContributionLink, requestKey string) error

	// ListContributingOutcomes returns the Outcomes contributing to parent in
	// stable creation order. An empty list means the Outcome is direct: shape
	// is derived from this, never stored.
	ListContributingOutcomes(ctx context.Context, parent domain.OutcomeID) ([]domain.Outcome, error)

	// ListContributionLinksForParent returns every binding held against
	// parent, including those bound to superseded revisions. Superseded links
	// stay readable so a stale contribution can be explained rather than
	// silently disappearing.
	ListContributionLinksForParent(ctx context.Context, parent domain.OutcomeID) ([]domain.ContributionLink, error)

	// ListContributionLinksForChild returns one contributing Outcome's
	// bindings. Their shared parent revision is that Outcome's binding.
	ListContributionLinksForChild(ctx context.Context, child domain.OutcomeID) ([]domain.ContributionLink, error)

	// AppendDecompositionRevision persists one PROPOSED decomposition with its
	// contributors, retained criteria, and dependencies, assigning the number
	// inside the transaction. It creates no Outcome and no binding: a proposal
	// is a reviewable offer, and a refused one leaves nothing behind.
	AppendDecompositionRevision(ctx context.Context, revision domain.DecompositionRevision) (domain.DecompositionRevision, error)

	// AuthorizeDecompositionRevision is the owner decision that turns a
	// proposal into responsibilities. Every contributing Outcome, its first
	// contract, its criterion bindings, the proposal's resolution, and the
	// one-way status move land in one transaction — a half-decomposed parent
	// would be a decomposition nobody authorized. Authorizing anything that is
	// not an open proposal returns ErrDecompositionNotProposed.
	AuthorizeDecompositionRevision(ctx context.Context, outcomeID domain.OutcomeID, decompositionID domain.DecompositionRevisionID, contributions []AuthorizedContribution, at time.Time) error

	// GetDecompositionRevision reads one decomposition; ok=false when absent
	// for this Outcome.
	GetDecompositionRevision(ctx context.Context, outcomeID domain.OutcomeID, id domain.DecompositionRevisionID) (domain.DecompositionRevision, bool, error)

	// LatestDecompositionRevision returns the newest decomposition of any
	// status; ok=false when the Outcome has never been decomposed.
	LatestDecompositionRevision(ctx context.Context, outcomeID domain.OutcomeID) (domain.DecompositionRevision, bool, error)

	// AppendContributionDependencyWaiver records the owner's override of a
	// declared ordering. Storage refuses a waiver for a dependency nobody
	// declared: consenting to nothing is not consent.
	AppendContributionDependencyWaiver(ctx context.Context, waiver domain.ContributionDependencyWaiver) error

	// ListContributionDependencyWaivers returns every waiver recorded against
	// one decomposition. Waivers are append-only, so this is the full history.
	ListContributionDependencyWaivers(ctx context.Context, decompositionID domain.DecompositionRevisionID) ([]domain.ContributionDependencyWaiver, error)

	// AppendPlanRevision atomically persists one proposed plan together with
	// its single Work Unit and capability grants, assigning the plan number
	// inside the transaction and returning the plan with that number.
	AppendPlanRevision(ctx context.Context, outcomeID domain.OutcomeID, plan domain.PlanRevision) (domain.PlanRevision, error)

	// LatestProposedPlanRevision resolves the highest-numbered proposed plan
	// bound to the named contract revision; ok=false when none exists. Create
	// replays resolve through it so re-proposing never stacks duplicates.
	LatestProposedPlanRevision(ctx context.Context, outcomeID domain.OutcomeID, contractRevision int64) (domain.PlanRevision, bool, error)

	// GetPlanRevision reads one plan with its work unit and grants; ok=false
	// when absent for this Outcome.
	GetPlanRevision(ctx context.Context, outcomeID domain.OutcomeID, planID domain.PlanRevisionID) (domain.PlanRevision, bool, error)

	// GetLatestPlanRevision returns the newest plan of any status; ok=false
	// when the Outcome has no plans yet.
	GetLatestPlanRevision(ctx context.Context, outcomeID domain.OutcomeID) (domain.PlanRevision, bool, error)

	// ApprovePlanRevision moves a plan from proposed to approved under an
	// optimistic guard and activates its grants. Re-approving an already
	// approved plan is idempotent; found=false means no such plan.
	ApprovePlanRevision(ctx context.Context, outcomeID domain.OutcomeID, planID domain.PlanRevisionID) (domain.PlanRevision, bool, error)

	// GetOutcomeProjectID resolves the project backing an Outcome through its
	// responsibility space; ok=false when the Outcome does not exist. Act &
	// Observe needs it to admit worker sessions onto the right project.
	GetOutcomeProjectID(ctx context.Context, outcomeID domain.OutcomeID) (domain.ProjectID, bool, error)

	// FindAttemptByIdempotencyKey resolves a previously delivered start
	// request by its client-supplied key; ok=false when unknown.
	FindAttemptByIdempotencyKey(ctx context.Context, key string) (domain.Attempt, bool, error)

	// CreateAttemptWithFence atomically persists one queued attempt together
	// with the open custody fence over the named worktree subject. The
	// attempt number is assigned inside the transaction. When another open
	// fence already holds the subject it returns *AttemptFenceHeldError with
	// NOTHING persisted: replacement inherits custody only through reconcile
	// releasing the old fence first.
	CreateAttemptWithFence(ctx context.Context, outcomeID domain.OutcomeID, plan domain.PlanRevision, requestKey string, subject string, at time.Time) (domain.Attempt, error)

	// GetAttempt reads one attempt of an Outcome; ok=false when absent.
	GetAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (domain.Attempt, bool, error)

	// ListAttempts returns the Outcome's attempts in ascending attempt order.
	ListAttempts(ctx context.Context, outcomeID domain.OutcomeID) ([]domain.Attempt, error)

	// TransitionAttemptStatus applies one stored-status transition under an
	// optimistic guard on the expected current status. rows=0 means the
	// attempt was absent or no longer held the expected status.
	TransitionAttemptStatus(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID, expected, next domain.AttemptStatus, at time.Time) (int64, error)

	// ListAttemptsByStatus returns every attempt currently in the given
	// stored status; the liveness reconcile loop walks running attempts.
	ListAttemptsByStatus(ctx context.Context, status domain.AttemptStatus) ([]domain.Attempt, error)

	// BindAttemptSession appends one immutable provider-session ref to the
	// attempt, assigning seq inside the transaction.
	BindAttemptSession(ctx context.Context, ref domain.AttemptSessionRef) (domain.AttemptSessionRef, error)

	// LatestAttemptSessionRef resolves the most recent session binding;
	// ok=false when the attempt has none.
	LatestAttemptSessionRef(ctx context.Context, attemptID domain.AttemptID) (domain.AttemptSessionRef, bool, error)

	// ListAttemptSessionRefs returns every binding in ascending seq order.
	ListAttemptSessionRefs(ctx context.Context, attemptID domain.AttemptID) ([]domain.AttemptSessionRef, error)

	// AppendAttemptObservation appends one ordered observation to the
	// attempt. Insertable for ANY attempt state — stale observations stay
	// inspectable after replacement — but they never mutate current truth.
	AppendAttemptObservation(ctx context.Context, attemptID domain.AttemptID, kind string, payload string, at time.Time) (domain.AttemptObservation, error)

	// ListAttemptObservations returns the attempt's ordered observation log.
	ListAttemptObservations(ctx context.Context, attemptID domain.AttemptID) ([]domain.AttemptObservation, error)

	// OpenFenceForSubject resolves the open custody fence over a worktree
	// subject; ok=false when the subject is free.
	OpenFenceForSubject(ctx context.Context, subject string) (domain.AttemptFence, bool, error)

	// ReleaseFenceForAttempt releases the attempt's open fence with a reason.
	// rows=0 means no open fence was held by this attempt.
	ReleaseFenceForAttempt(ctx context.Context, attemptID domain.AttemptID, reason string, at time.Time) (int64, error)

	// RenewFenceForAttempt refreshes the open fence's lease timestamp so a
	// stale renewal exposes custody that may outlive its provider. rows=0
	// means no open fence is held by this attempt.
	RenewFenceForAttempt(ctx context.Context, attemptID domain.AttemptID, at time.Time) (int64, error)

	// CreateRecoveryReceipt appends one immutable recovery receipt.
	CreateRecoveryReceipt(ctx context.Context, receipt domain.AttemptRecoveryReceipt) error

	// ListRecoveryReceipts returns the attempt's receipts in emission order.
	ListRecoveryReceipts(ctx context.Context, attemptID domain.AttemptID) ([]domain.AttemptRecoveryReceipt, error)
}

// AttemptFenceHeldError reports that a worktree subject is already under the
// custody of another open attempt fence. Start admission fails closed with
// nothing persisted; the safe path is reconcile -> release(old) -> new
// attempt.
type AttemptFenceHeldError struct {
	Subject   string
	Holder    domain.AttemptID
	OutcomeID domain.OutcomeID
}

func (e *AttemptFenceHeldError) Error() string {
	return fmt.Sprintf("worktree subject %s is fenced by attempt %s", e.Subject, e.Holder)
}

// AttemptReplayError reports that a start request key was already delivered:
// the named attempt IS the canonical answer, so the caller must serve it and
// MUST NOT spawn again. It mirrors OutcomeConflictError as an error carrying
// its resolution.
type AttemptReplayError struct {
	Attempt domain.Attempt
}

func (e *AttemptReplayError) Error() string {
	return fmt.Sprintf("attempt %s was already admitted for this request key", e.Attempt.ID)
}

// AuthorizedContribution pairs one proposal ref with the Outcome, contract,
// and bindings authorization creates for it. The service resolves proposals
// into these; storage writes them atomically.
type AuthorizedContribution struct {
	Ref     string
	Outcome domain.Outcome
	First   domain.ContractRevision
	Links   []domain.ContributionLink
}

// ErrDecompositionNotProposed reports an authorization attempt against a
// decomposition that is not an open proposal — already authorized or absent.
var ErrDecompositionNotProposed = errors.New("decomposition is not an open proposal")

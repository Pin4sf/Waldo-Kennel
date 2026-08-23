package ports

import (
	"context"
	"fmt"
	"strconv"

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

	// AppendContractRevision atomically appends one immutable revision and
	// swaps the current-revision pointer from expected to the newly assigned
	// number, which it returns. When expected no longer names the current
	// revision it returns *OutcomeConflictError and persists nothing.
	AppendContractRevision(ctx context.Context, id domain.OutcomeID, expectedCurrent int64, revision domain.ContractRevision) (int64, error)

	// ListContractRevisions returns the full immutable history ordered by
	// ascending revision number.
	ListContractRevisions(ctx context.Context, id domain.OutcomeID) ([]domain.ContractRevision, error)
}

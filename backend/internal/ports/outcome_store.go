package ports

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

// OutcomeConflictError reports an optimistic-concurrency failure: the caller
// addressed the Outcome through a revision pointer that is no longer current.
// Services map it to the shared conflict envelope (HTTP 409).
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
// Create semantics are idempotent by client request key: replaying a delivered
// create resolves to the original Outcome instead of writing a duplicate, and
// reports created=false so callers neither double-emit effects nor claim fresh
// authority. Contract revisions are append-only history; AdvanceOutcomeContract
// is a compare-and-swap on the outcome's current-revision pointer and fails
// with *OutcomeConflictError when another writer moved first.
type OutcomeStore interface {
	// EnsureWorkResponsibilitySpace resolves the project-backed Work space,
	// creating it on first use. The same space is returned for every call for
	// the lifetime of the project.
	EnsureWorkResponsibilitySpace(ctx context.Context, projectID domain.ProjectID) (domain.ResponsibilitySpace, error)

	// FindOutcomeByIdempotencyKey resolves a previously delivered create.
	FindOutcomeByIdempotencyKey(ctx context.Context, key string) (domain.Outcome, bool, error)

	// CreateOutcome persists a new Outcome in its space. A duplicate
	// idempotency key surfaces as a unique-constraint error; callers must
	// resolve replays through FindOutcomeByIdempotencyKey first.
	CreateOutcome(ctx context.Context, outcome domain.Outcome, idempotencyKey string) error

	// GetOutcome reads one Outcome by id; ok=false when absent.
	GetOutcome(ctx context.Context, id domain.OutcomeID) (domain.Outcome, bool, error)

	// NextContractRevisionNumber returns the next append-only revision number
	// for the Outcome (1 for the first revision).
	NextContractRevisionNumber(ctx context.Context, id domain.OutcomeID) (int64, error)

	// CreateContractRevision appends one immutable revision. The number must
	// equal the value NextContractRevisionNumber just handed out; duplicates
	// surface as unique-constraint errors.
	CreateContractRevision(ctx context.Context, revision domain.ContractRevision) error

	// AdvanceOutcomeContract swaps the Outcome's current-revision pointer from
	// expected to next. It returns *OutcomeConflictError when expected no
	// longer names the current revision.
	AdvanceOutcomeContract(ctx context.Context, id domain.OutcomeID, expected, next int64, at time.Time) error

	// ListContractRevisions returns the full immutable history ordered by
	// ascending revision number.
	ListContractRevisions(ctx context.Context, id domain.OutcomeID) ([]domain.ContractRevision, error)
}

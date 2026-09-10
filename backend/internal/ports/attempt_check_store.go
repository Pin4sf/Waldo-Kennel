package ports

import (
	"context"
	"errors"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// ErrCheckRunAlreadyReserved means another reconciler already holds the
// reservation for this exact check run, so this caller must not invoke the
// command. It is the durable equivalent of losing a race, not an error the
// owner needs to see.
var ErrCheckRunAlreadyReserved = errors.New("attempt check run is already reserved")

// CheckRunState is how far one durable check run has got.
type CheckRunState string

// The three states a check run can be in.
const (
	// CheckRunReserved means invocation was about to happen and no outcome
	// has been recorded. After a restart this is UNKNOWN, not "not yet run":
	// the command may have run and had effects.
	CheckRunReserved CheckRunState = "reserved"
	// CheckRunObserved means the immutable observation is complete.
	CheckRunObserved CheckRunState = "observed"
	// CheckRunUnknown is a reservation that did not complete. It is never
	// retried automatically.
	CheckRunUnknown CheckRunState = "unknown"
)

// AttemptCheckRun is the durable identity and result of running one approved
// check against one exact retained artifact version.
type AttemptCheckRun struct {
	ID              string
	AttemptID       domain.AttemptID
	CheckID         domain.ApprovedCheckID
	ArtifactVersion string
	State           CheckRunState
	// ReservationEpoch identifies the daemon invocation that owns the live
	// reservation. A reservation alone is not proof that its invoker died.
	ReservationEpoch string

	Observation AttemptCheckObservation
	// ArtifactChanged and ObservedArtifactVersion record whether the run
	// changed the result it was checking.
	ArtifactChanged         bool
	ObservedArtifactVersion string

	ReservedAt time.Time
	ObservedAt *time.Time
}

// Complete reports an observation that can be turned into proof.
func (r AttemptCheckRun) Complete() bool { return r.State == CheckRunObserved }

// AttemptCheckRunStore owns durable check-run identity.
//
// It exists so "did this command already run?" is a fact rather than an
// inference. Reconciliation re-enumerates every ended Attempt on every tick,
// so without it a failing check would relaunch its command indefinitely and a
// crash between writing evidence and writing its verification would relaunch
// it too.
type AttemptCheckRunStore interface {
	// ReserveAttemptCheckRun claims the right to invoke this exact check,
	// before invocation. It returns ErrCheckRunAlreadyReserved when a record
	// already exists, so the caller reuses that record instead of running.
	ReserveAttemptCheckRun(context.Context, AttemptCheckRun) error
	GetAttemptCheckRun(context.Context, domain.AttemptID, domain.ApprovedCheckID, string) (AttemptCheckRun, bool, error)
	ListAttemptCheckRuns(context.Context, domain.AttemptID, string) ([]AttemptCheckRun, error)
	// RecordAttemptCheckObservation completes a reservation. Write-once.
	RecordAttemptCheckObservation(context.Context, AttemptCheckRun) error
	// MarkAttemptCheckRunUnknown records a reservation that did not complete,
	// so it is visible and inspectable rather than silently retried.
	MarkAttemptCheckRunUnknown(context.Context, domain.AttemptID, domain.ApprovedCheckID, string, time.Time) error
}

package ports

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// QuickCaptureID identifies one explicit Home capture.
type QuickCaptureID string

// QuickCaptureKind records whether explicit text is preserved as a note or as
// a non-canonical Open Loop candidate.
type QuickCaptureKind string

const (
	QuickCaptureKindNote              QuickCaptureKind = "note"
	QuickCaptureKindOpenLoopCandidate QuickCaptureKind = "open_loop_candidate"
)

// Valid reports whether kind can be persisted by the Home store.
func (kind QuickCaptureKind) Valid() bool {
	return kind == QuickCaptureKindNote || kind == QuickCaptureKindOpenLoopCandidate
}

// QuickCapture is preserved explicit input. It is not canonical responsibility.
type QuickCapture struct {
	ID        QuickCaptureID
	SpaceID   domain.ResponsibilitySpaceID
	Text      string
	Kind      QuickCaptureKind
	CreatedAt time.Time
}

// HomeIdempotency binds a caller key to the service-normalized request intent.
type HomeIdempotency struct {
	Key         string
	Fingerprint string
}

// HomeIdempotencyConflictError means a key was already bound to other input.
type HomeIdempotencyConflictError struct {
	Key string
}

func (e *HomeIdempotencyConflictError) Error() string {
	return fmt.Sprintf("home idempotency key %q is already bound to different input", e.Key)
}

// OpenLoopRevisionConflictError reports an optimistic concurrency failure.
type OpenLoopRevisionConflictError struct {
	OpenLoopID       domain.OpenLoopID
	ExpectedRevision int64
	CurrentRevision  int64
}

func (e *OpenLoopRevisionConflictError) Error() string {
	return fmt.Sprintf("open loop %s moved past revision %s (current %s)", e.OpenLoopID,
		strconv.FormatInt(e.ExpectedRevision, 10), strconv.FormatInt(e.CurrentRevision, 10))
}

// OpenLoopSnapshot is canonical Open Loop truth plus append-only disposition
// history and its optimistic-concurrency revision.
type OpenLoopSnapshot struct {
	OpenLoop     domain.OpenLoop
	Dispositions []domain.LoopDisposition
	Revision     int64
}

// HomeStore persists Home facts without deciding what may become canonical.
// Implementations bind idempotency inside the same transaction as each write.
type HomeStore interface {
	// SaveQuickCapture atomically binds request to one preserved note or
	// candidate. Replays with the same fingerprint return the first result;
	// different fingerprints return *HomeIdempotencyConflictError.
	SaveQuickCapture(ctx context.Context, capture QuickCapture, request HomeIdempotency) (QuickCapture, error)

	// CreateOpenLoopWithConfirmation persists both facts in one transaction.
	// It must never commit one without the other. Idempotency is resolved in
	// that same transaction.
	CreateOpenLoopWithConfirmation(ctx context.Context, loop domain.OpenLoop, confirmation domain.LoopDisposition, request HomeIdempotency) (OpenLoopSnapshot, error)

	// AppendLoopDisposition appends one immutable disposition when
	// expectedRevision is current and advances the revision atomically.
	AppendLoopDisposition(ctx context.Context, loopID domain.OpenLoopID, expectedRevision int64, disposition domain.LoopDisposition, request HomeIdempotency) (OpenLoopSnapshot, error)
}

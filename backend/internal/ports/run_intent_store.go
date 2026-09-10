package ports

import (
	"context"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// RunIntentStore owns the durable record of what the owner has authorized the
// daemon to keep doing with an approved Plan.
//
// It is a separate port from OutcomeStore because a daemon can perfectly well
// read and schedule Outcomes without it; absent, run intent is simply
// unavailable and only per-Attempt Start remains, which is a truthful reduced
// capability rather than an assumed authorization.
type RunIntentStore interface {
	// AppendRunIntent records one new generation. Implementations choose the
	// generation number inside the write, and return the existing generation
	// unchanged when the request key has already been served.
	AppendRunIntent(context.Context, domain.OutcomeRunIntent) (domain.OutcomeRunIntent, error)
	CurrentRunIntent(context.Context, domain.OutcomeID) (domain.OutcomeRunIntent, bool, error)
	// FindRunIntentByRequestKey resolves a command's replay identity. It is
	// consulted before the transition is decided: a repeated key is the same
	// command arriving twice, not a new one to validate against state the
	// first copy already changed.
	FindRunIntentByRequestKey(context.Context, string) (domain.OutcomeRunIntent, bool, error)
	ListRunIntents(context.Context, domain.OutcomeID) ([]domain.OutcomeRunIntent, error)
	// ListOutcomesWithRunIntent returns Outcomes whose CURRENT generation
	// holds this desired state. An older generation that once said "running"
	// must never restart work the owner has since paused.
	ListOutcomesWithRunIntent(context.Context, domain.RunIntentDesired) ([]domain.OutcomeRunIntent, error)
	// AcknowledgeRunIntent marks a generation as having taken effect.
	// Write-once and idempotent.
	AcknowledgeRunIntent(context.Context, domain.OutcomeID, int64, time.Time) error
}

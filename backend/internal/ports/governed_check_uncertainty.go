package ports

import (
	"context"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// GovernedCheckUncertainty is trusted control-plane evidence that a provider-
// invoked approved check did not have a confirmed terminal process tree.
// It is not check proof; its presence blocks completion and custody release.
type GovernedCheckUncertainty struct {
	SessionID          domain.SessionID       `json:"sessionId"`
	CheckID            domain.ApprovedCheckID `json:"checkId"`
	TerminationUnknown bool                   `json:"terminationUnknown"`
	TimedOut           bool                   `json:"timedOut"`
	Cancelled          bool                   `json:"cancelled"`
	EnforcedBy         string                 `json:"enforcedBy,omitempty"`
	ObservedAt         time.Time              `json:"observedAt"`
}

// GovernedCheckUncertaintySink durably records uncertainty before the private
// tool server returns control to the provider.
type GovernedCheckUncertaintySink interface {
	RecordGovernedCheckUncertainty(context.Context, GovernedCheckUncertainty) error
	ClearGovernedCheckUncertainty(context.Context, domain.SessionID) error
}

// GovernedCheckUncertaintySource lets Outcome reconciliation import the
// trusted marker into canonical Attempt observations.
type GovernedCheckUncertaintySource interface {
	GovernedCheckUncertainty(context.Context, domain.SessionID) (GovernedCheckUncertainty, bool, error)
}

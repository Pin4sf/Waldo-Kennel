package ports

import (
	"context"
	"fmt"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// DeliveryReplayConflictError means a request key was reused for a different
// immutable delivery request.
type DeliveryReplayConflictError struct {
	Existing domain.OutcomeDelivery
	Request  domain.OutcomeDelivery
}

func (e *DeliveryReplayConflictError) Error() string {
	return fmt.Sprintf("delivery request key %q already names a different delivery", e.Request.RequestKey)
}

// DeliveryStore is the durable delivery state boundary. The filesystem
// transfer is separate because a process can crash between bytes and state.
type DeliveryStore interface {
	CreateOutcomeDelivery(context.Context, domain.OutcomeDelivery) error
	FindOutcomeDeliveryByRequestKey(context.Context, string) (domain.OutcomeDelivery, bool, error)
	GetOutcomeDelivery(context.Context, domain.OutcomeID, domain.DeliveryID) (domain.OutcomeDelivery, bool, error)
	ListOutcomeDeliveries(context.Context, domain.OutcomeID) ([]domain.OutcomeDelivery, error)
	CompleteOutcomeDelivery(context.Context, domain.OutcomeDelivery) (bool, error)
	FailPendingOutcomeDeliveries(context.Context, time.Time, string, string) (int64, error)
}

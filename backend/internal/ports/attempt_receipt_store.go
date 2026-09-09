package ports

import (
	"context"
	"errors"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// ErrAttemptReceiptFrozen reports a refused overwrite of a receipt that has
// already served as review evidence.
//
// Freezing is what makes "what the owner reviewed" a stable thing. Without it,
// a later retention pass over the same workspace could replace the manifest an
// acceptance decision was made against.
var ErrAttemptReceiptFrozen = errors.New("attempt receipt is frozen and cannot be replaced")

// AttemptReceiptStore owns the durable record of what each Attempt produced.
//
// Provider claims about output are claims. This store holds Kennel's own
// observation of the workspace, which is what a downstream WorkUnit consumes
// and what a delivery manifest is built from.
type AttemptReceiptStore interface {
	// SaveAttemptReceipt writes the receipt and its file manifest atomically,
	// refusing with ErrAttemptReceiptFrozen once the receipt is frozen.
	SaveAttemptReceipt(context.Context, domain.AttemptReceipt) error
	GetAttemptReceipt(context.Context, domain.AttemptID) (domain.AttemptReceipt, bool, error)
	// FreezeAttemptReceipt marks the receipt as review evidence. Idempotent.
	FreezeAttemptReceipt(context.Context, domain.AttemptID, time.Time) error
}

package domain

import (
	"fmt"
	"strings"
	"time"
)

// DeliveryID identifies one durable owner-triggered transfer.
type DeliveryID string

// DeliveryDisposition says whether the transferred result is accepted or a
// clearly labelled draft. It never changes the Outcome's acceptance state.
type DeliveryDisposition string

const (
	// DeliveryAccepted records a transfer of an artifact with current owner acceptance.
	DeliveryAccepted DeliveryDisposition = "accepted"
	// DeliveryDraft records a transfer that is explicitly not accepted.
	DeliveryDraft DeliveryDisposition = "draft"
)

// DeliveryState is the durable transfer lifecycle. Pending is recoverable
// evidence of a request, never evidence that bytes reached the destination.
type DeliveryState string

const (
	// DeliveryPending records a durable request whose filesystem effect is unresolved.
	DeliveryPending DeliveryState = "pending"
	// DeliverySucceeded records a fully staged and committed destination.
	DeliverySucceeded DeliveryState = "succeeded"
	// DeliveryFailed records a delivery that did not produce a successful destination.
	DeliveryFailed DeliveryState = "failed"
	// DeliveryCancelled records a delivery cancelled before it could be claimed successful.
	DeliveryCancelled DeliveryState = "cancelled"
)

// OutcomeDelivery records one exact artifact transfer and its observed result.
type OutcomeDelivery struct {
	ID                   DeliveryID
	OutcomeID            OutcomeID
	AttemptID            AttemptID
	WorkUnitID           WorkUnitID
	ArtifactVersion      string
	Disposition          DeliveryDisposition
	Destination          string
	AcceptanceDecisionID AcceptanceDecisionID
	RequestKey           string
	RequestFingerprint   string
	State                DeliveryState
	ManifestPath         string
	FileCount            int
	ByteCount            int64
	FailureCode          string
	FailureDetail        string
	RequestedAt          time.Time
	CompletedAt          *time.Time
}

// Validate checks the identity and terminal-state invariants of a delivery.
func (d OutcomeDelivery) Validate() error {
	switch {
	case strings.TrimSpace(string(d.ID)) == "":
		return fmt.Errorf("delivery id is required")
	case d.OutcomeID.IsZero() || d.AttemptID.IsZero() || d.WorkUnitID.IsZero():
		return fmt.Errorf("delivery lineage is required")
	case strings.TrimSpace(d.ArtifactVersion) == "":
		return fmt.Errorf("delivery artifact version is required")
	case d.Disposition != DeliveryAccepted && d.Disposition != DeliveryDraft:
		return fmt.Errorf("delivery disposition is invalid")
	case strings.TrimSpace(d.Destination) == "":
		return fmt.Errorf("delivery destination is required")
	case strings.TrimSpace(d.RequestKey) == "" || strings.TrimSpace(d.RequestFingerprint) == "":
		return fmt.Errorf("delivery idempotency identity is required")
	case d.State != DeliveryPending && d.State != DeliverySucceeded && d.State != DeliveryFailed && d.State != DeliveryCancelled:
		return fmt.Errorf("delivery state is invalid")
	case d.FileCount < 0 || d.ByteCount < 0:
		return fmt.Errorf("delivery size cannot be negative")
	case d.RequestedAt.IsZero():
		return fmt.Errorf("delivery requested timestamp is required")
	}
	if d.State == DeliveryPending && d.CompletedAt != nil {
		return fmt.Errorf("pending delivery cannot have a completion timestamp")
	}
	if d.State != DeliveryPending && d.CompletedAt == nil {
		return fmt.Errorf("terminal delivery requires a completion timestamp")
	}
	if d.State == DeliverySucceeded && d.ManifestPath == "" {
		return fmt.Errorf("succeeded delivery requires a manifest path")
	}
	return nil
}

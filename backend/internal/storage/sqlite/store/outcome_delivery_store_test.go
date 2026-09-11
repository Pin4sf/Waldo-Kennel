package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func TestOutcomeDeliveryStore_ReplayCompletionAndRestartFailureAreDurable(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "delivery-store")
	attempt, err := s.CreateAttemptWithFence(ctx, admissionFor(outcomeID, plan, "delivery-attempt", domain.FenceSubjectForProject("delivery-store")))
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	now := time.Unix(100, 0).UTC()
	delivery := domain.OutcomeDelivery{
		ID: "dlv-delivery-store", OutcomeID: outcomeID, AttemptID: attempt.ID, WorkUnitID: attempt.WorkUnitID,
		ArtifactVersion: "artifact-v1", Disposition: domain.DeliveryDraft, Destination: "/tmp/delivery-store",
		RequestKey: "delivery-key", RequestFingerprint: "delivery-fingerprint", State: domain.DeliveryPending,
		RequestedAt: now,
	}
	if err := s.CreateOutcomeDelivery(ctx, delivery); err != nil {
		t.Fatalf("create delivery: %v", err)
	}
	if err := s.CreateOutcomeDelivery(ctx, delivery); err != nil {
		t.Fatalf("same delivery replay: %v", err)
	}
	changed := delivery
	changed.RequestFingerprint = "different-fingerprint"
	var replayConflict *ports.DeliveryReplayConflictError
	if err := s.CreateOutcomeDelivery(ctx, changed); !errors.As(err, &replayConflict) {
		t.Fatalf("changed replay = %v, want DeliveryReplayConflictError", err)
	}

	completedAt := now.Add(time.Minute)
	terminal := delivery
	terminal.State = domain.DeliverySucceeded
	terminal.ManifestPath = "/tmp/delivery-store/KENNEL-EXPORT.json"
	terminal.FileCount = 2
	terminal.ByteCount = 12
	terminal.CompletedAt = &completedAt
	terminal.CompletionSource = domain.DeliveryObserved
	updated, err := s.CompleteOutcomeDelivery(ctx, terminal)
	if err != nil || !updated {
		t.Fatalf("complete delivery: updated=%v err=%v", updated, err)
	}
	got, found, err := s.GetOutcomeDelivery(ctx, outcomeID, delivery.ID)
	if err != nil || !found {
		t.Fatalf("read completed delivery: found=%v err=%v", found, err)
	}
	if got.CompletionSource != domain.DeliveryObserved {
		t.Fatalf("completion source = %q, want observed", got.CompletionSource)
	}
	if got.State != domain.DeliverySucceeded || got.ManifestPath != terminal.ManifestPath || got.CompletedAt == nil {
		t.Fatalf("completed delivery = %+v", got)
	}

	pending := delivery
	pending.ID = "dlv-delivery-pending"
	pending.RequestKey = "delivery-pending-key"
	pending.RequestFingerprint = "delivery-pending-fingerprint"
	if err := s.CreateOutcomeDelivery(ctx, pending); err != nil {
		t.Fatalf("create pending delivery: %v", err)
	}
	// Recovery reads pending rows one at a time, because each destination has
	// to be inspected before anything is concluded about it.
	stillPending, err := s.ListPendingOutcomeDeliveries(ctx)
	if err != nil {
		t.Fatalf("list pending deliveries: %v", err)
	}
	if len(stillPending) != 1 || stillPending[0].ID != pending.ID {
		t.Fatalf("pending deliveries = %+v, want only the unresolved one", stillPending)
	}
	if stillPending[0].CompletionSource != "" {
		t.Fatalf("pending delivery named completion source %q", stillPending[0].CompletionSource)
	}

	interruptedAt := now.Add(2 * time.Minute)
	failed := pending
	failed.State, failed.FailureCode, failed.FailureDetail = domain.DeliveryFailed, "DELIVERY_INTERRUPTED", "daemon restarted"
	failed.CompletionSource, failed.CompletedAt = domain.DeliveryRecovered, &interruptedAt
	if changed, err := s.CompleteOutcomeDelivery(ctx, failed); err != nil || !changed {
		t.Fatalf("close pending delivery: changed=%v err=%v", changed, err)
	}
	pendingGot, found, err := s.GetOutcomeDelivery(ctx, outcomeID, pending.ID)
	if err != nil || !found {
		t.Fatalf("read interrupted delivery: found=%v err=%v", found, err)
	}
	if pendingGot.State != domain.DeliveryFailed || pendingGot.FailureCode != "DELIVERY_INTERRUPTED" || pendingGot.CompletedAt == nil {
		t.Fatalf("interrupted delivery = %+v", pendingGot)
	}
	// How a terminal result was decided is durable, and a recovered verdict is
	// not the same evidence as one this daemon watched.
	if pendingGot.CompletionSource != domain.DeliveryRecovered {
		t.Fatalf("completion source = %q, want recovered", pendingGot.CompletionSource)
	}
	if remaining, err := s.ListPendingOutcomeDeliveries(ctx); err != nil || len(remaining) != 0 {
		t.Fatalf("pending after closing = %+v err=%v", remaining, err)
	}
}

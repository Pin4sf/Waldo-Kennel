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
	updated, err := s.CompleteOutcomeDelivery(ctx, terminal)
	if err != nil || !updated {
		t.Fatalf("complete delivery: updated=%v err=%v", updated, err)
	}
	got, found, err := s.GetOutcomeDelivery(ctx, outcomeID, delivery.ID)
	if err != nil || !found {
		t.Fatalf("read completed delivery: found=%v err=%v", found, err)
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
	interruptedAt := now.Add(2 * time.Minute)
	count, err := s.FailPendingOutcomeDeliveries(ctx, interruptedAt, "DELIVERY_INTERRUPTED", "daemon restarted")
	if err != nil || count != 1 {
		t.Fatalf("fail pending deliveries: count=%d err=%v", count, err)
	}
	pendingGot, found, err := s.GetOutcomeDelivery(ctx, outcomeID, pending.ID)
	if err != nil || !found {
		t.Fatalf("read interrupted delivery: found=%v err=%v", found, err)
	}
	if pendingGot.State != domain.DeliveryFailed || pendingGot.FailureCode != "DELIVERY_INTERRUPTED" || pendingGot.CompletedAt == nil {
		t.Fatalf("interrupted delivery = %+v", pendingGot)
	}
}

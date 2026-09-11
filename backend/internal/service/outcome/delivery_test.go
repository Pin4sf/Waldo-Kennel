package outcome_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/artifactstore"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	outcomevc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

type deliveryFixture struct {
	ports.OutcomeStore
	ports.OutcomeProofStore
	ports.AttemptReceiptStore
	outcome   domain.Outcome
	attempt   domain.Attempt
	revision  domain.ContractRevision
	receipt   domain.AttemptReceipt
	decisions []domain.AcceptanceDecision
}

func (f *deliveryFixture) GetOutcome(context.Context, domain.OutcomeID) (domain.Outcome, bool, error) {
	return f.outcome, true, nil
}

func (f *deliveryFixture) GetAttempt(context.Context, domain.OutcomeID, domain.AttemptID) (domain.Attempt, bool, error) {
	return f.attempt, true, nil
}

func (f *deliveryFixture) ListContractRevisions(context.Context, domain.OutcomeID) ([]domain.ContractRevision, error) {
	return []domain.ContractRevision{f.revision}, nil
}

func (f *deliveryFixture) GetAttemptReceipt(context.Context, domain.AttemptID) (domain.AttemptReceipt, bool, error) {
	return f.receipt, true, nil
}

func (f *deliveryFixture) ListAcceptanceDecisions(context.Context, domain.OutcomeID) ([]domain.AcceptanceDecision, error) {
	return append([]domain.AcceptanceDecision(nil), f.decisions...), nil
}

type memoryDeliveryStore struct {
	byKey map[string]domain.OutcomeDelivery
	byID  map[domain.DeliveryID]domain.OutcomeDelivery
}

func (s *memoryDeliveryStore) CreateOutcomeDelivery(_ context.Context, delivery domain.OutcomeDelivery) error {
	if existing, ok := s.byKey[delivery.RequestKey]; ok {
		if existing.RequestFingerprint != delivery.RequestFingerprint {
			return &ports.DeliveryReplayConflictError{Existing: existing, Request: delivery}
		}
		return nil
	}
	s.byKey[delivery.RequestKey], s.byID[delivery.ID] = delivery, delivery
	return nil
}

func (s *memoryDeliveryStore) FindOutcomeDeliveryByRequestKey(_ context.Context, key string) (domain.OutcomeDelivery, bool, error) {
	delivery, ok := s.byKey[key]
	return delivery, ok, nil
}

func (s *memoryDeliveryStore) GetOutcomeDelivery(_ context.Context, _ domain.OutcomeID, id domain.DeliveryID) (domain.OutcomeDelivery, bool, error) {
	delivery, ok := s.byID[id]
	return delivery, ok, nil
}

func (s *memoryDeliveryStore) ListOutcomeDeliveries(_ context.Context, _ domain.OutcomeID) ([]domain.OutcomeDelivery, error) {
	items := make([]domain.OutcomeDelivery, 0, len(s.byID))
	for _, delivery := range s.byID {
		items = append(items, delivery)
	}
	return items, nil
}

func (s *memoryDeliveryStore) CompleteOutcomeDelivery(_ context.Context, delivery domain.OutcomeDelivery) (bool, error) {
	existing, ok := s.byID[delivery.ID]
	if !ok || existing.State != domain.DeliveryPending {
		return false, nil
	}
	s.byID[delivery.ID], s.byKey[delivery.RequestKey] = delivery, delivery
	return true, nil
}

// ListPendingOutcomeDeliveries mirrors the SQLite query recovery reads, in the
// same requested-at order.
func (s *memoryDeliveryStore) ListPendingOutcomeDeliveries(_ context.Context) ([]domain.OutcomeDelivery, error) {
	var pending []domain.OutcomeDelivery
	for _, delivery := range s.byID {
		if delivery.State == domain.DeliveryPending {
			pending = append(pending, delivery)
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		if !pending[i].RequestedAt.Equal(pending[j].RequestedAt) {
			return pending[i].RequestedAt.Before(pending[j].RequestedAt)
		}
		return pending[i].ID < pending[j].ID
	})
	return pending, nil
}

func TestRequestDeliveryBindsCurrentAcceptanceAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	at := time.Unix(500, 0).UTC()
	artifacts, err := artifactstore.New(artifactstore.Config{Root: filepath.Join(t.TempDir(), "artifacts")})
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "answer.txt"), []byte("accepted\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	attempt := domain.Attempt{ID: "attempt-delivery", OutcomeID: "outcome-delivery", WorkUnitID: "unit-delivery", PlanRevisionID: "plan-delivery", ContractRevisionNumber: 1}
	retained, err := artifacts.Retain(ctx, artifactstore.Input{
		AttemptID: attempt.ID, OutcomeID: attempt.OutcomeID, PlanRevisionID: attempt.PlanRevisionID,
		WorkUnitID: attempt.WorkUnitID, ContractRevisionNumber: 1, WorkspaceKind: domain.WorkspaceStagedFolder,
		WorkspacePath: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	decision := domain.AcceptanceDecision{
		ID: "accept-delivery", OutcomeID: attempt.OutcomeID, ContractRevisionID: "contract-delivery",
		Kind: domain.AcceptanceAccept, ActorType: domain.AcceptanceActorUser, Summary: "Reviewed exact result",
		ResourceDisposition: domain.ResourceDispositionRetain, RequestKey: "accept-key",
		RequestFingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", CreatedAt: at,
	}
	fixture := &deliveryFixture{
		outcome: domain.Outcome{ID: attempt.OutcomeID}, attempt: attempt,
		revision: domain.ContractRevision{ID: decision.ContractRevisionID, OutcomeID: attempt.OutcomeID, Number: 1},
		receipt:  retained.Receipt, decisions: []domain.AcceptanceDecision{decision},
	}
	deliveries := &memoryDeliveryStore{byKey: map[string]domain.OutcomeDelivery{}, byID: map[domain.DeliveryID]domain.OutcomeDelivery{}}
	service := outcomevc.New(fixture, func() time.Time { return at }).WithProofStore(fixture).WithDelivery(deliveries, artifacts)
	destination := filepath.Join(t.TempDir(), "accepted-bundle")
	in := outcomevc.RequestDeliveryInput{
		AttemptID: attempt.ID, ArtifactVersion: retained.Receipt.ArtifactVersion, Destination: destination,
		Disposition: domain.DeliveryAccepted, AcceptanceDecisionID: decision.ID, RequestKey: "delivery-request",
	}
	first, err := service.RequestDelivery(ctx, attempt.OutcomeID, in)
	if err != nil {
		t.Fatalf("request delivery: %v", err)
	}
	if first.State != domain.DeliverySucceeded || first.ManifestPath != filepath.Join(destination, artifactstore.ManifestName) {
		t.Fatalf("delivery = %+v", first)
	}
	body, err := os.ReadFile(filepath.Join(destination, "answer.txt"))
	if err != nil || string(body) != "accepted\n" {
		t.Fatalf("delivered body = %q err=%v", body, err)
	}
	second, err := service.RequestDelivery(ctx, attempt.OutcomeID, in)
	if err != nil || second.ID != first.ID {
		t.Fatalf("replay = %+v err=%v, want same durable result", second, err)
	}

	rework := decision
	rework.ID = "rework-delivery"
	rework.Kind = domain.AcceptanceRequestRework
	rework.RequestKey = "rework-key"
	fixture.decisions = append(fixture.decisions, rework)
	in.RequestKey = "delivery-after-rework"
	if _, err := service.RequestDelivery(ctx, attempt.OutcomeID, in); err == nil {
		t.Fatal("delivery after request-rework succeeded")
	} else {
		var apiError *apierr.Error
		if !errors.As(err, &apiError) || apiError.Code != "DELIVERY_NOT_ACCEPTED" {
			t.Fatalf("delivery after request-rework = %v, want DELIVERY_NOT_ACCEPTED", err)
		}
	}
}

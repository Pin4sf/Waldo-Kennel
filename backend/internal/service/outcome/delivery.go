package outcome

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/artifactstore"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

const (
	deliveryRequestInvalid   = "DELIVERY_REQUEST_INVALID"
	deliveryArtifactMismatch = "DELIVERY_ARTIFACT_MISMATCH"
	deliveryArtifactMissing  = "DELIVERY_ARTIFACT_MISSING"
	deliveryNotAccepted      = "DELIVERY_NOT_ACCEPTED"
	deliveryInterrupted      = "DELIVERY_INTERRUPTED"
	deliveryFailed           = "DELIVERY_FAILED"
	deliveryCancelled        = "DELIVERY_CANCELLED"
)

// RequestDeliveryInput is the complete owner request. The artifact version is
// intentionally required: delivery never means "the newest result".
type RequestDeliveryInput struct {
	AttemptID            domain.AttemptID
	ArtifactVersion      string
	Destination          string
	Disposition          domain.DeliveryDisposition
	AcceptanceDecisionID domain.AcceptanceDecisionID
	RequestKey           string
}

// DeliveryManager is separate from Manager so existing reduced-capability
// fakes do not gain a new mandatory method. The controller reports 501 until
// both durable delivery and the retained artifact store are wired.
type DeliveryManager interface {
	DeliveriesEnabled() bool
	ListDeliveries(context.Context, domain.OutcomeID) ([]domain.OutcomeDelivery, error)
	GetDelivery(context.Context, domain.OutcomeID, domain.DeliveryID) (domain.OutcomeDelivery, error)
	RequestDelivery(context.Context, domain.OutcomeID, RequestDeliveryInput) (domain.OutcomeDelivery, error)
	ReconcileDeliveries(context.Context) (int64, error)
}

// WithDelivery wires the durable delivery ledger and retained artifact store.
func (s *Service) WithDelivery(deliveries ports.DeliveryStore, artifacts *artifactstore.Store) *Service {
	s.deliveryStore = deliveries
	s.deliveryArtifacts = artifacts
	return s
}

// DeliveriesEnabled reports whether this daemon can perform durable delivery.
func (s *Service) DeliveriesEnabled() bool {
	return s != nil && s.deliveryStore != nil && s.deliveryArtifacts != nil && s.receipts != nil
}

// ListDeliveries reads delivery history for one Outcome.
func (s *Service) ListDeliveries(ctx context.Context, outcomeID domain.OutcomeID) ([]domain.OutcomeDelivery, error) {
	if !s.DeliveriesEnabled() {
		return nil, apierr.Internal("DELIVERY_UNAVAILABLE", "Durable delivery is not wired in this daemon")
	}
	if _, ok, err := s.store.GetOutcome(ctx, outcomeID); err != nil {
		return nil, err
	} else if !ok {
		return nil, apierr.NotFound("OUTCOME_NOT_FOUND", "Outcome was not found")
	}
	return s.deliveryStore.ListOutcomeDeliveries(ctx, outcomeID)
}

// GetDelivery reads one delivery inside its Outcome lineage.
func (s *Service) GetDelivery(ctx context.Context, outcomeID domain.OutcomeID, deliveryID domain.DeliveryID) (domain.OutcomeDelivery, error) {
	if !s.DeliveriesEnabled() {
		return domain.OutcomeDelivery{}, apierr.Internal("DELIVERY_UNAVAILABLE", "Durable delivery is not wired in this daemon")
	}
	delivery, ok, err := s.deliveryStore.GetOutcomeDelivery(ctx, outcomeID, deliveryID)
	if err != nil {
		return domain.OutcomeDelivery{}, err
	}
	if !ok {
		return domain.OutcomeDelivery{}, apierr.NotFound("DELIVERY_NOT_FOUND", "Delivery was not found")
	}
	return delivery, nil
}

// RequestDelivery validates and executes one owner-triggered exact-artifact transfer.
func (s *Service) RequestDelivery(ctx context.Context, outcomeID domain.OutcomeID, in RequestDeliveryInput) (domain.OutcomeDelivery, error) {
	if !s.DeliveriesEnabled() {
		return domain.OutcomeDelivery{}, apierr.Internal("DELIVERY_UNAVAILABLE", "Durable delivery is not wired in this daemon")
	}
	destination, err := validateDeliveryInput(outcomeID, in)
	if err != nil {
		return domain.OutcomeDelivery{}, err
	}
	fingerprint := deliveryFingerprint(outcomeID, in, destination)
	if existing, ok, err := s.deliveryStore.FindOutcomeDeliveryByRequestKey(ctx, in.RequestKey); err != nil {
		return domain.OutcomeDelivery{}, err
	} else if ok {
		if existing.OutcomeID != outcomeID || existing.RequestFingerprint != fingerprint {
			return domain.OutcomeDelivery{}, apierr.Conflict("DELIVERY_REQUEST_CONFLICT", "Request key is already bound to a different delivery", map[string]any{"requestKey": in.RequestKey})
		}
		return existing, nil
	}

	if _, ok, err := s.store.GetOutcome(ctx, outcomeID); err != nil {
		return domain.OutcomeDelivery{}, err
	} else if !ok {
		return domain.OutcomeDelivery{}, apierr.NotFound("OUTCOME_NOT_FOUND", "Outcome was not found")
	}
	attempt, ok, err := s.store.GetAttempt(ctx, outcomeID, in.AttemptID)
	if err != nil {
		return domain.OutcomeDelivery{}, err
	}
	if !ok {
		return domain.OutcomeDelivery{}, apierr.NotFound("ATTEMPT_NOT_FOUND", "Attempt was not found for this Outcome")
	}
	receipt, ok, err := s.receipts.GetAttemptReceipt(ctx, in.AttemptID)
	if err != nil {
		return domain.OutcomeDelivery{}, err
	}
	if !ok {
		return domain.OutcomeDelivery{}, apierr.NotFound(deliveryArtifactMissing, "The Attempt has no retained artifact")
	}
	if receipt.OutcomeID != outcomeID || receipt.AttemptID != in.AttemptID || receipt.WorkUnitID != attempt.WorkUnitID {
		return domain.OutcomeDelivery{}, apierr.Conflict(deliveryArtifactMismatch, "The retained artifact does not belong to this Attempt lineage", nil)
	}
	if receipt.ArtifactVersion != in.ArtifactVersion {
		return domain.OutcomeDelivery{}, apierr.Conflict(deliveryArtifactMismatch, "The requested artifact version is not the retained Attempt result", map[string]any{"retainedArtifactVersion": receipt.ArtifactVersion})
	}
	if !receipt.RetentionState.Complete() {
		return domain.OutcomeDelivery{}, apierr.Conflict(deliveryArtifactMissing, "Only a complete retained artifact can be delivered", map[string]any{"retentionState": string(receipt.RetentionState)})
	}

	contractRevisionID, err := s.contractRevisionForReceipt(ctx, outcomeID, receipt.ContractRevisionNumber)
	if err != nil {
		return domain.OutcomeDelivery{}, err
	}
	var decision *domain.AcceptanceDecision
	if in.Disposition == domain.DeliveryAccepted {
		decision, err = s.currentAcceptance(ctx, outcomeID, contractRevisionID, in.AcceptanceDecisionID)
		if err != nil {
			return domain.OutcomeDelivery{}, err
		}
	}

	now := s.clock().UTC()
	delivery := domain.OutcomeDelivery{
		ID: domain.DeliveryID("dlv-" + uuid.NewString()), OutcomeID: outcomeID,
		AttemptID: attempt.ID, WorkUnitID: attempt.WorkUnitID, ArtifactVersion: receipt.ArtifactVersion,
		Disposition: in.Disposition, Destination: destination,
		RequestKey: in.RequestKey, RequestFingerprint: fingerprint, State: domain.DeliveryPending, RequestedAt: now,
	}
	if decision != nil {
		delivery.AcceptanceDecisionID = decision.ID
	}
	if err := s.deliveryStore.CreateOutcomeDelivery(ctx, delivery); err != nil {
		var conflict *ports.DeliveryReplayConflictError
		if errors.As(err, &conflict) {
			return domain.OutcomeDelivery{}, apierr.Conflict("DELIVERY_REQUEST_CONFLICT", "Request key is already bound to a different delivery", nil)
		}
		return domain.OutcomeDelivery{}, err
	}
	// The store may have resolved a cross-process unique-key race as an
	// idempotent replay. Re-read the durable row before touching the
	// destination so only the request that actually owns the pending row can
	// perform the filesystem effect.
	if persisted, ok, err := s.deliveryStore.FindOutcomeDeliveryByRequestKey(ctx, delivery.RequestKey); err != nil {
		return domain.OutcomeDelivery{}, err
	} else if !ok {
		return domain.OutcomeDelivery{}, fmt.Errorf("delivery %s was not readable after creation", delivery.ID)
	} else if persisted.ID != delivery.ID {
		return persisted, nil
	}

	_, exportErr := s.deliveryArtifacts.Export(ctx, artifactstore.ExportRequest{
		Receipt: receipt, Decision: decision, ContractRevisionID: contractRevisionID,
		AcceptedArtifactVersion: receipt.ArtifactVersion, Draft: in.Disposition == domain.DeliveryDraft,
		Destination: destination,
	})
	terminal := delivery
	completed := s.clock().UTC()
	terminal.CompletedAt = &completed
	if exportErr != nil || ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			terminal.State = domain.DeliveryCancelled
			terminal.FailureCode = deliveryCancelled
			terminal.FailureDetail = "delivery was cancelled; no successful delivery is claimed"
		} else {
			terminal.State = domain.DeliveryFailed
			terminal.FailureCode, terminal.FailureDetail = deliveryFailure(exportErr)
		}
		if err := s.completeDelivery(context.WithoutCancel(ctx), terminal); err != nil {
			return domain.OutcomeDelivery{}, err
		}
		return terminal, nil
	}
	terminal.State = domain.DeliverySucceeded
	terminal.ManifestPath = filepath.Join(destination, artifactstore.ManifestName)
	terminal.FileCount = len(receipt.Files)
	for _, file := range receipt.Files {
		if file.SizeBytes != nil && file.ChangeKind != domain.ArtifactDeleted {
			terminal.ByteCount += *file.SizeBytes
		}
	}
	if err := s.completeDelivery(context.WithoutCancel(ctx), terminal); err != nil {
		return domain.OutcomeDelivery{}, err
	}
	return terminal, nil
}

// ReconcileDeliveries closes requests left pending across a daemon restart.
func (s *Service) ReconcileDeliveries(ctx context.Context) (int64, error) {
	if !s.DeliveriesEnabled() {
		return 0, nil
	}
	return s.deliveryStore.FailPendingOutcomeDeliveries(context.WithoutCancel(ctx), s.clock().UTC(), deliveryInterrupted, "daemon restarted before delivery reached a terminal state")
}

func (s *Service) completeDelivery(ctx context.Context, delivery domain.OutcomeDelivery) error {
	changed, err := s.deliveryStore.CompleteOutcomeDelivery(ctx, delivery)
	if err != nil {
		return err
	}
	if changed {
		return nil
	}
	existing, ok, err := s.deliveryStore.GetOutcomeDelivery(ctx, delivery.OutcomeID, delivery.ID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("delivery %s disappeared before terminalization", delivery.ID)
	}
	if existing.State == delivery.State && existing.RequestFingerprint == delivery.RequestFingerprint {
		return nil
	}
	return fmt.Errorf("delivery %s became stale before terminalization", delivery.ID)
}

func (s *Service) contractRevisionForReceipt(ctx context.Context, outcomeID domain.OutcomeID, number int64) (domain.ContractRevisionID, error) {
	revisions, err := s.store.ListContractRevisions(ctx, outcomeID)
	if err != nil {
		return "", err
	}
	for _, revision := range revisions {
		if revision.Number == number {
			return revision.ID, nil
		}
	}
	return "", apierr.Conflict(deliveryArtifactMismatch, "The retained artifact refers to a missing Contract revision", map[string]any{"contractRevisionNumber": number})
}

func (s *Service) currentAcceptance(ctx context.Context, outcomeID domain.OutcomeID, revisionID domain.ContractRevisionID, decisionID domain.AcceptanceDecisionID) (*domain.AcceptanceDecision, error) {
	if s.proof == nil {
		return nil, apierr.Internal("OUTCOME_PROOF_UNAVAILABLE", "Outcome acceptance storage is unavailable")
	}
	if strings.TrimSpace(string(decisionID)) == "" {
		return nil, apierr.Conflict(deliveryNotAccepted, "Accepted delivery requires the owner's AcceptanceDecision", nil)
	}
	decisions, err := s.proof.ListAcceptanceDecisions(ctx, outcomeID)
	if err != nil {
		return nil, err
	}
	var found *domain.AcceptanceDecision
	for i := range decisions {
		if decisions[i].ID == decisionID {
			decision := decisions[i]
			found = &decision
			break
		}
	}
	if found == nil || found.OutcomeID != outcomeID || found.ContractRevisionID != revisionID || found.Kind != domain.AcceptanceAccept || found.ActorType != domain.AcceptanceActorUser {
		return nil, apierr.Conflict(deliveryNotAccepted, "The requested AcceptanceDecision does not authorize this artifact", nil)
	}
	if len(decisions) == 0 || decisions[len(decisions)-1].ID != decisionID {
		return nil, apierr.Conflict(deliveryNotAccepted, "The requested artifact is no longer the current accepted result", nil)
	}
	return found, nil
}

func validateDeliveryInput(outcomeID domain.OutcomeID, in RequestDeliveryInput) (string, error) {
	if outcomeID.IsZero() || in.AttemptID.IsZero() || strings.TrimSpace(in.ArtifactVersion) == "" || strings.TrimSpace(in.RequestKey) == "" {
		return "", apierr.Invalid(deliveryRequestInvalid, "Outcome, Attempt, artifact version, and request key are required", nil)
	}
	if in.Disposition != domain.DeliveryAccepted && in.Disposition != domain.DeliveryDraft {
		return "", apierr.Invalid(deliveryRequestInvalid, "Disposition must be accepted or draft", nil)
	}
	if in.Disposition == domain.DeliveryDraft && strings.TrimSpace(string(in.AcceptanceDecisionID)) != "" {
		return "", apierr.Invalid(deliveryRequestInvalid, "Draft delivery cannot carry an AcceptanceDecision", nil)
	}
	if in.Disposition == domain.DeliveryAccepted && strings.TrimSpace(string(in.AcceptanceDecisionID)) == "" {
		return "", apierr.Invalid(deliveryRequestInvalid, "Accepted delivery requires an AcceptanceDecision", nil)
	}
	if !filepath.IsAbs(in.Destination) || filepath.Clean(in.Destination) == string(filepath.Separator) {
		return "", apierr.Invalid(deliveryRequestInvalid, "Destination must be an absolute non-root path", nil)
	}
	destination := filepath.Clean(in.Destination)
	if info, err := os.Lstat(destination); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", apierr.Conflict("DELIVERY_DESTINATION_UNSAFE", "Destination cannot be a symlink", nil)
	}
	return destination, nil
}

func deliveryFingerprint(outcomeID domain.OutcomeID, in RequestDeliveryInput, destination string) string {
	payload, _ := json.Marshal(struct {
		OutcomeID            string `json:"outcomeId"`
		AttemptID            string `json:"attemptId"`
		ArtifactVersion      string `json:"artifactVersion"`
		Destination          string `json:"destination"`
		Disposition          string `json:"disposition"`
		AcceptanceDecisionID string `json:"acceptanceDecisionId"`
	}{string(outcomeID), string(in.AttemptID), in.ArtifactVersion, destination, string(in.Disposition), string(in.AcceptanceDecisionID)})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func deliveryFailure(err error) (string, string) {
	if err == nil {
		return deliveryFailed, "delivery did not complete"
	}
	switch {
	case errors.Is(err, artifactstore.ErrExportDestinationConflict):
		return "DELIVERY_DESTINATION_CONFLICT", err.Error()
	case errors.Is(err, artifactstore.ErrExportDestinationUnsafe):
		return "DELIVERY_DESTINATION_UNSAFE", err.Error()
	case errors.Is(err, artifactstore.ErrExportManifestCollision):
		return "DELIVERY_MANIFEST_COLLISION", err.Error()
	case errors.Is(err, artifactstore.ErrExportArtifactMissing):
		return deliveryArtifactMissing, err.Error()
	default:
		return deliveryFailed, err.Error()
	}
}

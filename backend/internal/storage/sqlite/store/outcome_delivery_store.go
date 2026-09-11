package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/gen"
)

var _ ports.DeliveryStore = (*Store)(nil)

// CreateOutcomeDelivery persists the request exactly once. A retry with the
// same request key and fingerprint is a replay; reusing the key for another
// request is a durable conflict, even when the first request is still pending.
func (s *Store) CreateOutcomeDelivery(ctx context.Context, delivery domain.OutcomeDelivery) error {
	if err := delivery.Validate(); err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.inTx(ctx, fmt.Sprintf("create delivery %s", delivery.ID), func(q *gen.Queries) error {
		row, err := q.FindOutcomeDeliveryByRequestKey(ctx, delivery.RequestKey)
		if err == nil {
			existing := deliveryFromRow(row)
			if existing.OutcomeID != delivery.OutcomeID || existing.RequestFingerprint != delivery.RequestFingerprint {
				return &ports.DeliveryReplayConflictError{Existing: existing, Request: delivery}
			}
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("find delivery request key: %w", err)
		}
		if err := q.CreateOutcomeDelivery(ctx, gen.CreateOutcomeDeliveryParams{
			ID: string(delivery.ID), OutcomeID: string(delivery.OutcomeID), AttemptID: string(delivery.AttemptID),
			WorkUnitID: string(delivery.WorkUnitID), ArtifactVersion: delivery.ArtifactVersion,
			Disposition: string(delivery.Disposition), Destination: delivery.Destination,
			AcceptanceDecisionID: string(delivery.AcceptanceDecisionID), RequestKey: delivery.RequestKey,
			RequestFingerprint: delivery.RequestFingerprint, State: string(delivery.State),
			ManifestPath: delivery.ManifestPath, FileCount: int64(delivery.FileCount), ByteCount: delivery.ByteCount,
			FailureCode: delivery.FailureCode, FailureDetail: delivery.FailureDetail,
			RequestedAt: delivery.RequestedAt, CompletedAt: nullableTime(delivery.CompletedAt),
		}); err != nil {
			// A separate daemon may have won the unique request-key race after
			// our read. Resolve that race to the same replay semantics instead
			// of leaking a raw SQLite constraint error.
			if raced, findErr := q.FindOutcomeDeliveryByRequestKey(ctx, delivery.RequestKey); findErr == nil {
				existing := deliveryFromRow(raced)
				if existing.OutcomeID != delivery.OutcomeID || existing.RequestFingerprint != delivery.RequestFingerprint {
					return &ports.DeliveryReplayConflictError{Existing: existing, Request: delivery}
				}
				return nil
			}
			return fmt.Errorf("insert delivery: %w", err)
		}
		return nil
	})
}

// FindOutcomeDeliveryByRequestKey resolves a delivery replay identity.
func (s *Store) FindOutcomeDeliveryByRequestKey(ctx context.Context, key string) (domain.OutcomeDelivery, bool, error) {
	row, err := s.qr.FindOutcomeDeliveryByRequestKey(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.OutcomeDelivery{}, false, nil
	}
	if err != nil {
		return domain.OutcomeDelivery{}, false, fmt.Errorf("find delivery by request key: %w", err)
	}
	return deliveryFromRow(row), true, nil
}

// GetOutcomeDelivery reads one delivery inside its Outcome lineage.
func (s *Store) GetOutcomeDelivery(ctx context.Context, outcomeID domain.OutcomeID, id domain.DeliveryID) (domain.OutcomeDelivery, bool, error) {
	row, err := s.qr.GetOutcomeDelivery(ctx, gen.GetOutcomeDeliveryParams{OutcomeID: string(outcomeID), ID: string(id)})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.OutcomeDelivery{}, false, nil
	}
	if err != nil {
		return domain.OutcomeDelivery{}, false, fmt.Errorf("get delivery %s: %w", id, err)
	}
	return deliveryFromRow(row), true, nil
}

// ListOutcomeDeliveries returns an Outcome's durable delivery history.
func (s *Store) ListOutcomeDeliveries(ctx context.Context, outcomeID domain.OutcomeID) ([]domain.OutcomeDelivery, error) {
	rows, err := s.qr.ListOutcomeDeliveries(ctx, string(outcomeID))
	if err != nil {
		return nil, fmt.Errorf("list deliveries for %s: %w", outcomeID, err)
	}
	out := make([]domain.OutcomeDelivery, 0, len(rows))
	for _, row := range rows {
		out = append(out, deliveryFromRow(row))
	}
	return out, nil
}

// CompleteOutcomeDelivery records exactly one pending delivery result.
func (s *Store) CompleteOutcomeDelivery(ctx context.Context, delivery domain.OutcomeDelivery) (bool, error) {
	if err := delivery.Validate(); err != nil {
		return false, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	var changed int64
	err := s.inTx(ctx, fmt.Sprintf("complete delivery %s", delivery.ID), func(q *gen.Queries) error {
		var err error
		changed, err = q.CompleteOutcomeDelivery(ctx, gen.CompleteOutcomeDeliveryParams{
			State: string(delivery.State), ManifestPath: delivery.ManifestPath,
			FileCount: int64(delivery.FileCount), ByteCount: delivery.ByteCount,
			FailureCode: delivery.FailureCode, FailureDetail: delivery.FailureDetail,
			CompletedAt:      nullableTime(delivery.CompletedAt),
			CompletionSource: string(delivery.CompletionSource), ID: string(delivery.ID),
		})
		return err
	})
	if err != nil {
		return false, fmt.Errorf("complete delivery %s: %w", delivery.ID, err)
	}
	return changed > 0, nil
}

// ListPendingOutcomeDeliveries reads every request whose filesystem effect is
// still unresolved, so recovery can inspect each destination on its own.
func (s *Store) ListPendingOutcomeDeliveries(ctx context.Context) ([]domain.OutcomeDelivery, error) {
	rows, err := s.qr.ListPendingOutcomeDeliveries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pending deliveries: %w", err)
	}
	out := make([]domain.OutcomeDelivery, 0, len(rows))
	for _, row := range rows {
		out = append(out, deliveryFromRow(row))
	}
	return out, nil
}

func deliveryFromRow(row gen.OutcomeDelivery) domain.OutcomeDelivery {
	delivery := domain.OutcomeDelivery{
		ID: domain.DeliveryID(row.ID), OutcomeID: domain.OutcomeID(row.OutcomeID),
		AttemptID: domain.AttemptID(row.AttemptID), WorkUnitID: domain.WorkUnitID(row.WorkUnitID),
		ArtifactVersion: row.ArtifactVersion, Disposition: domain.DeliveryDisposition(row.Disposition),
		Destination: row.Destination, AcceptanceDecisionID: domain.AcceptanceDecisionID(row.AcceptanceDecisionID),
		RequestKey: row.RequestKey, RequestFingerprint: row.RequestFingerprint,
		State: domain.DeliveryState(row.State), ManifestPath: row.ManifestPath,
		FileCount: int(row.FileCount), ByteCount: row.ByteCount, FailureCode: row.FailureCode,
		FailureDetail: row.FailureDetail, RequestedAt: row.RequestedAt,
	}
	if row.State != string(domain.DeliveryPending) {
		// Historical rows predate recovery and default to 'observed'; a
		// pending row has nothing to report yet.
		delivery.CompletionSource = domain.DeliveryCompletionSource(row.CompletionSource)
	}
	if row.CompletedAt.Valid {
		at := row.CompletedAt.Time
		delivery.CompletedAt = &at
	}
	return delivery
}

func nullableTime(at *time.Time) sql.NullTime {
	if at == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *at, Valid: true}
}

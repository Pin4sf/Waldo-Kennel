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

// AppendRunIntent records one new authorized generation.
//
// The generation is chosen inside the write transaction, not by the caller:
// two concurrent commands that both read generation 3 must not both write 4,
// and the unique (outcome_id, generation) index is what makes the loser fail
// rather than silently overwrite the winner's authorization.
func (s *Store) AppendRunIntent(ctx context.Context, intent domain.OutcomeRunIntent) (domain.OutcomeRunIntent, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return domain.OutcomeRunIntent{}, fmt.Errorf("begin append run intent: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	// A replayed command must return the generation it already produced,
	// never authorize a second one.
	if existing, err := txq.FindOutcomeRunIntentByRequestKey(ctx, intent.RequestKey); err == nil {
		return runIntentFromRow(existing), nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.OutcomeRunIntent{}, fmt.Errorf("find run intent by request key: %w", err)
	}

	current, err := txq.CurrentOutcomeRunIntent(ctx, string(intent.OutcomeID))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		intent.Generation = 1
	case err != nil:
		return domain.OutcomeRunIntent{}, fmt.Errorf("read current run intent: %w", err)
	default:
		intent.Generation = current.Generation + 1
	}
	// Validation runs after the store assigns the generation: the caller does
	// not supply one, and a guessed value must never be able to overwrite
	// somebody else's authorization.
	if err := intent.Validate(); err != nil {
		return domain.OutcomeRunIntent{}, err
	}

	if err := txq.CreateOutcomeRunIntent(ctx, gen.CreateOutcomeRunIntentParams{
		ID: string(intent.ID), OutcomeID: string(intent.OutcomeID), Generation: intent.Generation,
		Desired: string(intent.Desired), PlanRevisionID: string(intent.PlanRevisionID),
		ContractRevisionNumber: intent.ContractRevisionNumber,
		RequestKey:             intent.RequestKey, RequestedAt: intent.RequestedAt,
	}); err != nil {
		return domain.OutcomeRunIntent{}, fmt.Errorf("create run intent: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.OutcomeRunIntent{}, fmt.Errorf("commit run intent: %w", err)
	}
	return intent, nil
}

// CurrentRunIntent reads the latest authorized generation for an Outcome.
// Absent is a real answer: nothing has ever been authorized.
func (s *Store) CurrentRunIntent(ctx context.Context, outcomeID domain.OutcomeID) (domain.OutcomeRunIntent, bool, error) {
	row, err := s.qr.CurrentOutcomeRunIntent(ctx, string(outcomeID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.OutcomeRunIntent{}, false, nil
	}
	if err != nil {
		return domain.OutcomeRunIntent{}, false, fmt.Errorf("current run intent for %s: %w", outcomeID, err)
	}
	return runIntentFromRow(row), true, nil
}

// FindRunIntentByRequestKey resolves a command's replay identity.
func (s *Store) FindRunIntentByRequestKey(ctx context.Context, requestKey string) (domain.OutcomeRunIntent, bool, error) {
	row, err := s.qr.FindOutcomeRunIntentByRequestKey(ctx, requestKey)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.OutcomeRunIntent{}, false, nil
	}
	if err != nil {
		return domain.OutcomeRunIntent{}, false, fmt.Errorf("find run intent by request key: %w", err)
	}
	return runIntentFromRow(row), true, nil
}

// ListRunIntents returns an Outcome's full authorization history, oldest
// first. It is decision history, so nothing is ever filtered out of it.
func (s *Store) ListRunIntents(ctx context.Context, outcomeID domain.OutcomeID) ([]domain.OutcomeRunIntent, error) {
	rows, err := s.qr.ListOutcomeRunIntents(ctx, string(outcomeID))
	if err != nil {
		return nil, fmt.Errorf("list run intents for %s: %w", outcomeID, err)
	}
	intents := make([]domain.OutcomeRunIntent, 0, len(rows))
	for _, row := range rows {
		intents = append(intents, runIntentFromRow(row))
	}
	return intents, nil
}

// ListOutcomesWithRunIntent returns every Outcome whose CURRENT generation
// holds the given desired state. Continuation reads this; an older generation
// that once said "running" must never restart work the owner has since
// paused.
func (s *Store) ListOutcomesWithRunIntent(ctx context.Context, desired domain.RunIntentDesired) ([]domain.OutcomeRunIntent, error) {
	rows, err := s.qr.ListCurrentRunIntentsByDesired(ctx, string(desired))
	if err != nil {
		return nil, fmt.Errorf("list run intents desiring %s: %w", desired, err)
	}
	intents := make([]domain.OutcomeRunIntent, 0, len(rows))
	for _, row := range rows {
		intents = append(intents, runIntentFromRow(row))
	}
	return intents, nil
}

// AcknowledgeRunIntent marks a generation as having taken effect. It is
// write-once and idempotent: a second acknowledgement changes nothing rather
// than moving the timestamp.
func (s *Store) AcknowledgeRunIntent(ctx context.Context, outcomeID domain.OutcomeID, generation int64, at time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, err := s.qw.AcknowledgeOutcomeRunIntent(ctx, gen.AcknowledgeOutcomeRunIntentParams{
		AcknowledgedAt: sql.NullTime{Time: at, Valid: true},
		OutcomeID:      string(outcomeID), Generation: generation,
	}); err != nil {
		return fmt.Errorf("acknowledge run intent %s/%d: %w", outcomeID, generation, err)
	}
	return nil
}

var _ ports.RunIntentStore = (*Store)(nil)

func runIntentFromRow(row gen.OutcomeRunIntent) domain.OutcomeRunIntent {
	intent := domain.OutcomeRunIntent{
		ID: domain.RunIntentID(row.ID), OutcomeID: domain.OutcomeID(row.OutcomeID),
		Generation: row.Generation, Desired: domain.RunIntentDesired(row.Desired),
		PlanRevisionID:         domain.PlanRevisionID(row.PlanRevisionID),
		ContractRevisionNumber: row.ContractRevisionNumber,
		RequestKey:             row.RequestKey, RequestedAt: row.RequestedAt,
	}
	if row.AcknowledgedAt.Valid {
		at := row.AcknowledgedAt.Time
		intent.AcknowledgedAt = &at
	}
	return intent
}

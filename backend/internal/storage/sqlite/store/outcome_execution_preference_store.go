package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/gen"
)

// CreateOutcomeWithContractExecution atomically persists an Outcome and its
// first immutable ContractRevision including WT3 execution preference state.
func (s *Store) CreateOutcomeWithContractExecution(ctx context.Context, outcome domain.Outcome, first domain.ContractRevision, requestKey string) error {
	if err := outcome.Validate(); err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create outcome %s: %w", outcome.ID, err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	var key sql.NullString
	if requestKey != "" {
		key = sql.NullString{String: requestKey, Valid: true}
	}
	if err := txq.CreateOutcome(ctx, gen.CreateOutcomeParams{
		ID:              outcome.ID,
		SpaceID:         outcome.SpaceID,
		Title:           outcome.Title,
		IdempotencyKey:  key,
		ParentOutcomeID: nullOutcomeID(outcome.ParentID),
	}); err != nil {
		return fmt.Errorf("create outcome %s: %w", outcome.ID, err)
	}

	number, err := nextRevisionNumber(ctx, txq, outcome.ID)
	if err != nil {
		return err
	}
	first.Number = number
	if err := insertContractRevisionExecution(ctx, txq, tx, first); err != nil {
		return err
	}
	rows, err := txq.AdvanceOutcomeCurrentRevision(ctx, gen.AdvanceOutcomeCurrentRevisionParams{
		CurrentRevisionNumber:   number,
		UpdatedAt:               first.CreatedAt,
		ID:                      outcome.ID,
		CurrentRevisionNumber_2: 0,
	})
	if err != nil {
		return fmt.Errorf("point outcome %s at revision 1: %w", outcome.ID, err)
	}
	if rows != 1 {
		return fmt.Errorf("point outcome %s at revision 1: pointer moved concurrently", outcome.ID)
	}
	return tx.Commit()
}

// AppendContractRevisionExecution atomically appends an immutable revision
// including WT3 preference state and advances the optimistic current pointer.
func (s *Store) AppendContractRevisionExecution(ctx context.Context, id domain.OutcomeID, expectedCurrent int64, revision domain.ContractRevision) (int64, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin append revision for %s: %w", id, err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)
	number, err := nextRevisionNumber(ctx, txq, id)
	if err != nil {
		return 0, err
	}
	revision.Number = number
	if err := insertContractRevisionExecution(ctx, txq, tx, revision); err != nil {
		return 0, err
	}
	rows, err := txq.AdvanceOutcomeCurrentRevision(ctx, gen.AdvanceOutcomeCurrentRevisionParams{
		CurrentRevisionNumber:   number,
		UpdatedAt:               time.Now().UTC(),
		ID:                      id,
		CurrentRevisionNumber_2: expectedCurrent,
	})
	if err != nil {
		return 0, fmt.Errorf("advance outcome %s to revision %d: %w", id, number, err)
	}
	if rows == 0 {
		currentNum := int64(-1)
		row, getErr := txq.GetOutcome(ctx, id)
		if getErr == nil {
			currentNum = row.CurrentRevisionNumber
		}
		return 0, &ports.OutcomeConflictError{OutcomeID: id, ExpectedRevisionNum: expectedCurrent, CurrentRevisionNum: currentNum}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit append revision for %s: %w", id, err)
	}
	return number, nil
}

func insertContractRevisionExecution(ctx context.Context, q *gen.Queries, tx *sql.Tx, revision domain.ContractRevision) error {
	if len(revision.Criteria) == 0 {
		revision.Criteria = make([]domain.ContractCriterion, 0, len(revision.SuccessCriteria))
		for i, text := range revision.SuccessCriteria {
			revision.Criteria = append(revision.Criteria, domain.ContractCriterion{
				ID:                 domain.CriterionID(fmt.Sprintf("crit-%s-%04d", revision.ID, i+1)),
				ContractRevisionID: revision.ID,
				Position:           int64(i + 1),
				Text:               text,
			})
		}
	}
	criteria, err := marshalJSONStrings(revision.SuccessCriteria)
	if err != nil { return fmt.Errorf("revision criteria: %w", err) }
	constraints, err := marshalJSONStrings(revision.Constraints)
	if err != nil { return fmt.Errorf("revision constraints: %w", err) }
	nonGoals, err := marshalJSONStrings(revision.NonGoals)
	if err != nil { return fmt.Errorf("revision non-goals: %w", err) }
	if err := revision.Validate(); err != nil { return err }

	var preference any
	if revision.ExecutionPreference != nil {
		encoded, err := json.Marshal(revision.ExecutionPreference)
		if err != nil { return fmt.Errorf("encode revision execution preference: %w", err) }
		preference = string(encoded)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO contract_revisions
    (id, outcome_id, number, goal, success_criteria, review, constraints, non_goals, clarification, execution_preference_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		revision.ID, revision.OutcomeID, revision.Number, revision.Goal, criteria, revision.Review,
		constraints, nonGoals, revision.Clarification, preference,
	)
	if err != nil { return fmt.Errorf("create contract revision %s: %w", revision.ID, err) }
	for _, criterion := range revision.Criteria {
		if err := q.CreateContractCriterion(ctx, gen.CreateContractCriterionParams{
			ID: string(criterion.ID), ContractRevisionID: string(criterion.ContractRevisionID),
			Position: criterion.Position, Text: criterion.Text,
		}); err != nil {
			return fmt.Errorf("create contract criterion %s: %w", criterion.ID, err)
		}
	}
	if err := insertContractIntakeCore(ctx, q, revision); err != nil {
		return fmt.Errorf("create contract revision core %s: %w", revision.ID, err)
	}
	return nil
}

// GetContractExecutionPreference reads the immutable preference projection for
// one revision. NULL is a truthful no-preference state for old and new rows.
func (s *Store) GetContractExecutionPreference(ctx context.Context, id domain.ContractRevisionID) (*domain.ExecutionPreference, error) {
	var raw sql.NullString
	err := s.readDB.QueryRowContext(ctx,
		`SELECT execution_preference_json FROM contract_revisions WHERE id = ?`, id,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) { return nil, nil }
	if err != nil { return nil, fmt.Errorf("read execution preference for %s: %w", id, err) }
	if !raw.Valid || raw.String == "" { return nil, nil }
	var preference domain.ExecutionPreference
	if err := json.Unmarshal([]byte(raw.String), &preference); err != nil {
		return nil, fmt.Errorf("decode execution preference for %s: %w", id, err)
	}
	if err := preference.Validate(); err != nil {
		return nil, fmt.Errorf("invalid execution preference for %s: %w", id, err)
	}
	return &preference, nil
}

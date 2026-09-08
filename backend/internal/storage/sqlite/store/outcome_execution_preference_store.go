package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// writeContractExecutionPreference is part of the canonical Contract write
// transaction. The generated base INSERT predates 0114, so the additive column
// is populated immediately in the same transaction rather than by a parallel
// Create/Append writer or by hand-editing sqlc output.
func writeContractExecutionPreference(ctx context.Context, tx *sql.Tx, revision domain.ContractRevision) error {
	if revision.ExecutionPreference == nil {
		return nil
	}
	if err := revision.ExecutionPreference.Validate(); err != nil {
		return err
	}
	encoded, err := json.Marshal(revision.ExecutionPreference)
	if err != nil {
		return fmt.Errorf("encode execution preference for contract revision %s: %w", revision.ID, err)
	}
	result, err := tx.ExecContext(ctx,
		`UPDATE contract_revisions SET execution_preference_json = ? WHERE id = ?`,
		string(encoded), revision.ID,
	)
	if err != nil {
		return fmt.Errorf("persist execution preference for contract revision %s: %w", revision.ID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count execution preference write for contract revision %s: %w", revision.ID, err)
	}
	if rows != 1 {
		return fmt.Errorf("persist execution preference for contract revision %s: revision row missing", revision.ID)
	}
	return nil
}

// GetContractExecutionPreference reads immutable planning preference. NULL is
// truthful no-preference state for historical and new Contracts.
func (s *Store) GetContractExecutionPreference(ctx context.Context, id domain.ContractRevisionID) (*domain.ExecutionPreference, error) {
	var raw sql.NullString
	err := s.readDB.QueryRowContext(ctx,
		`SELECT execution_preference_json FROM contract_revisions WHERE id = ?`, id,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read execution preference for %s: %w", id, err)
	}
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	var preference domain.ExecutionPreference
	if err := json.Unmarshal([]byte(raw.String), &preference); err != nil {
		return nil, fmt.Errorf("decode execution preference for %s: %w", id, err)
	}
	if err := preference.Validate(); err != nil {
		return nil, fmt.Errorf("invalid execution preference for %s: %w", id, err)
	}
	return &preference, nil
}

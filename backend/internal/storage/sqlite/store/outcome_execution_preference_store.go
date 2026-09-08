package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// encodeContractExecutionPreference renders the immutable planning preference
// for the canonical Contract INSERT.
//
// It has to be written by the insert itself: contract_revisions is append-only
// and trigger-guarded against UPDATE, so a follow-up write in the same
// transaction aborts with "contract revisions are immutable" and no Contract
// carrying a preference could ever be created. A nil preference is truthful
// no-preference state and stays NULL.
func encodeContractExecutionPreference(revision domain.ContractRevision) (sql.NullString, error) {
	if revision.ExecutionPreference == nil {
		return sql.NullString{}, nil
	}
	if err := revision.ExecutionPreference.Validate(); err != nil {
		return sql.NullString{}, err
	}
	encoded, err := json.Marshal(revision.ExecutionPreference)
	if err != nil {
		return sql.NullString{}, fmt.Errorf("encode execution preference for contract revision %s: %w", revision.ID, err)
	}
	return sql.NullString{String: string(encoded), Valid: true}, nil
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

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/gen"
	"github.com/google/uuid"
)

// EnsureWorkResponsibilitySpace resolves the project-backed Work space,
// creating it on first use. The daemon serialises writes, so check-then-insert
// under writeMu is race-free; the partial unique index backstops it.
func (s *Store) EnsureWorkResponsibilitySpace(ctx context.Context, projectID domain.ProjectID) (domain.ResponsibilitySpace, error) {
	row, err := s.qr.FindWorkResponsibilitySpaceByProject(ctx, projectID)
	if err == nil {
		return responsibilitySpaceFromRow(row), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.ResponsibilitySpace{}, fmt.Errorf("find work space for %s: %w", projectID, err)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	space := domain.ResponsibilitySpace{
		ID:        domain.ResponsibilitySpaceID("rsp-" + uuid.NewString()),
		Kind:      domain.ResponsibilitySpaceWorkProject,
		ProjectID: projectID,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.qw.CreateResponsibilitySpace(ctx, gen.CreateResponsibilitySpaceParams{
		ID:        space.ID,
		ProjectID: space.ProjectID,
	}); err != nil {
		if isSQLiteUnique(err) {
			// Lost a concurrent first-use race; return the winner.
			row, err := s.qw.FindWorkResponsibilitySpaceByProject(ctx, projectID)
			if err != nil {
				return domain.ResponsibilitySpace{}, fmt.Errorf("re-find work space for %s: %w", projectID, err)
			}
			return responsibilitySpaceFromRow(row), nil
		}
		return domain.ResponsibilitySpace{}, fmt.Errorf("create work space for %s: %w", projectID, err)
	}
	return space, nil
}

// FindOutcomeByIdempotencyKey resolves a previously delivered create request.
func (s *Store) FindOutcomeByIdempotencyKey(ctx context.Context, key string) (domain.Outcome, bool, error) {
	row, err := s.qr.FindOutcomeByIdempotencyKey(ctx, sql.NullString{String: key, Valid: key != ""})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Outcome{}, false, nil
	}
	if err != nil {
		return domain.Outcome{}, false, fmt.Errorf("find outcome by idempotency key: %w", err)
	}
	return outcomeFromRow(row), true, nil
}

// CreateOutcomeWithContract atomically persists the Outcome with its first
// contract revision and points current at it.
func (s *Store) CreateOutcomeWithContract(ctx context.Context, outcome domain.Outcome, first domain.ContractRevision, requestKey string) error {
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
		ID:             outcome.ID,
		SpaceID:        outcome.SpaceID,
		Title:          outcome.Title,
		IdempotencyKey: key,
	}); err != nil {
		return fmt.Errorf("create outcome %s: %w", outcome.ID, err)
	}

	number, err := nextRevisionNumber(ctx, txq, outcome.ID)
	if err != nil {
		return err
	}
	first.Number = number
	if err := insertContractRevision(ctx, txq, first); err != nil {
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

// AppendContractRevision atomically appends one immutable revision and swings
// the current-revision pointer from expected to it. A stale expected rolls the
// whole transaction back and reports the conflict.
func (s *Store) AppendContractRevision(ctx context.Context, id domain.OutcomeID, expectedCurrent int64, revision domain.ContractRevision) (int64, error) {
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
	if err := insertContractRevision(ctx, txq, revision); err != nil {
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

// GetOutcome reads one Outcome by id; ok=false when absent.
func (s *Store) GetOutcome(ctx context.Context, id domain.OutcomeID) (domain.Outcome, bool, error) {
	row, err := s.qr.GetOutcome(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Outcome{}, false, nil
	}
	if err != nil {
		return domain.Outcome{}, false, fmt.Errorf("get outcome %s: %w", id, err)
	}
	return outcomeFromRow(row), true, nil
}

// ListContractRevisions returns full immutable history ordered by number.
func (s *Store) ListContractRevisions(ctx context.Context, id domain.OutcomeID) ([]domain.ContractRevision, error) {
	rows, err := s.qr.ListContractRevisions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list revisions for %s: %w", id, err)
	}
	out := make([]domain.ContractRevision, 0, len(rows))
	for _, row := range rows {
		rev, err := contractRevisionFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, rev)
	}
	return out, nil
}

func nextRevisionNumber(ctx context.Context, q *gen.Queries, id domain.OutcomeID) (int64, error) {
	maxNum, err := q.MaxContractRevisionNumber(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("max revision number for %s: %w", id, err)
	}
	switch v := maxNum.(type) {
	case int64:
		return v + 1, nil
	default:
		return 0, fmt.Errorf("max revision number for %s: unexpected type %T", id, maxNum)
	}
}

func insertContractRevision(ctx context.Context, q *gen.Queries, revision domain.ContractRevision) error {
	criteria, err := marshalJSONStrings(revision.SuccessCriteria)
	if err != nil {
		return fmt.Errorf("revision criteria: %w", err)
	}
	constraints, err := marshalJSONStrings(revision.Constraints)
	if err != nil {
		return fmt.Errorf("revision constraints: %w", err)
	}
	nonGoals, err := marshalJSONStrings(revision.NonGoals)
	if err != nil {
		return fmt.Errorf("revision non-goals: %w", err)
	}
	if err := revision.Validate(); err != nil {
		return err
	}
	if err := q.CreateContractRevision(ctx, gen.CreateContractRevisionParams{
		ID:              revision.ID,
		OutcomeID:       revision.OutcomeID,
		Number:          revision.Number,
		Goal:            revision.Goal,
		SuccessCriteria: criteria,
		Review:          revision.Review,
		Constraints:     constraints,
		NonGoals:        nonGoals,
		Clarification:   revision.Clarification,
	}); err != nil {
		return fmt.Errorf("create contract revision %s: %w", revision.ID, err)
	}
	return nil
}

func outcomeFromRow(row gen.Outcome) domain.Outcome {
	return domain.Outcome{
		ID:                    row.ID,
		SpaceID:               row.SpaceID,
		Title:                 row.Title,
		CurrentRevisionNumber: row.CurrentRevisionNumber,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
}

func responsibilitySpaceFromRow(row gen.ResponsibilitySpace) domain.ResponsibilitySpace {
	return domain.ResponsibilitySpace{
		ID:        row.ID,
		Kind:      row.Kind,
		ProjectID: row.ProjectID,
		CreatedAt: row.CreatedAt,
	}
}

func contractRevisionFromRow(row gen.ContractRevision) (domain.ContractRevision, error) {
	criteria, err := unmarshalJSONStrings(row.SuccessCriteria)
	if err != nil {
		return domain.ContractRevision{}, fmt.Errorf("revision %s criteria: %w", row.ID, err)
	}
	constraints, err := unmarshalJSONStrings(row.Constraints)
	if err != nil {
		return domain.ContractRevision{}, fmt.Errorf("revision %s constraints: %w", row.ID, err)
	}
	nonGoals, err := unmarshalJSONStrings(row.NonGoals)
	if err != nil {
		return domain.ContractRevision{}, fmt.Errorf("revision %s non-goals: %w", row.ID, err)
	}
	return domain.ContractRevision{
		ID:              row.ID,
		OutcomeID:       row.OutcomeID,
		Number:          row.Number,
		Goal:            row.Goal,
		SuccessCriteria: criteria,
		Review:          row.Review,
		Constraints:     constraints,
		NonGoals:        nonGoals,
		Clarification:   row.Clarification,
		CreatedAt:       row.CreatedAt,
	}, nil
}

func marshalJSONStrings(in []string) (string, error) {
	if in == nil {
		in = []string{}
	}
	data, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalJSONStrings(data string) ([]string, error) {
	if data == "" {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}

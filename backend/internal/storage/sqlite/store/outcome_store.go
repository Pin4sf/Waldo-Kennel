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

// CreateOutcome persists a new Outcome row. Duplicate idempotency keys surface
// as unique-constraint errors; callers resolve replays beforehand.
func (s *Store) CreateOutcome(ctx context.Context, outcome domain.Outcome, idempotencyKey string) error {
	if err := outcome.Validate(); err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	var key sql.NullString
	if idempotencyKey != "" {
		key = sql.NullString{String: idempotencyKey, Valid: true}
	}
	if err := s.qw.CreateOutcome(ctx, gen.CreateOutcomeParams{
		ID:                    outcome.ID,
		SpaceID:               outcome.SpaceID,
		Title:                 outcome.Title,
		CurrentRevisionNumber: outcome.CurrentRevisionNumber,
		IdempotencyKey:        key,
	}); err != nil {
		return fmt.Errorf("create outcome %s: %w", outcome.ID, err)
	}
	return nil
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

// NextContractRevisionNumber hands out the next append-only revision number.
func (s *Store) NextContractRevisionNumber(ctx context.Context, id domain.OutcomeID) (int64, error) {
	maxNum, err := s.qr.MaxContractRevisionNumber(ctx, id)
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

// CreateContractRevision appends one immutable revision row.
func (s *Store) CreateContractRevision(ctx context.Context, revision domain.ContractRevision) error {
	if err := revision.Validate(); err != nil {
		return err
	}
	criteria, err := marshalJSONStrings(revision.SuccessCriteria)
	if err != nil {
		return fmt.Errorf("revision %s criteria: %w", revision.ID, err)
	}
	constraints, err := marshalJSONStrings(revision.Constraints)
	if err != nil {
		return fmt.Errorf("revision %s constraints: %w", revision.ID, err)
	}
	nonGoals, err := marshalJSONStrings(revision.NonGoals)
	if err != nil {
		return fmt.Errorf("revision %s non-goals: %w", revision.ID, err)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.qw.CreateContractRevision(ctx, gen.CreateContractRevisionParams{
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

// AdvanceOutcomeContract compare-and-swaps the current-revision pointer.
// rows-affected 0 means the expected pointer is stale — either another writer
// advanced first or the pointer never matched.
func (s *Store) AdvanceOutcomeContract(ctx context.Context, id domain.OutcomeID, expected, next int64, at time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	rows, err := s.qw.AdvanceOutcomeCurrentRevision(ctx, gen.AdvanceOutcomeCurrentRevisionParams{
		CurrentRevisionNumber:   next,
		UpdatedAt:               at,
		ID:                      id,
		CurrentRevisionNumber_2: expected,
	})
	if err != nil {
		return fmt.Errorf("advance outcome %s contract: %w", id, err)
	}
	if rows == 0 {
		current, ok, err := s.unlockedOutcome(ctx, id)
		if err != nil {
			return err
		}
		currentNum := int64(-1)
		if ok {
			currentNum = current.CurrentRevisionNumber
		}
		return &ports.OutcomeConflictError{OutcomeID: id, ExpectedRevisionNum: expected, CurrentRevisionNum: currentNum}
	}
	return nil
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

// unlockedOutcome reads through the writer connection while holding writeMu.
func (s *Store) unlockedOutcome(ctx context.Context, id domain.OutcomeID) (domain.Outcome, bool, error) {
	row, err := s.qw.GetOutcome(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Outcome{}, false, nil
	}
	if err != nil {
		return domain.Outcome{}, false, fmt.Errorf("get outcome %s: %w", id, err)
	}
	return outcomeFromRow(row), true, nil
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

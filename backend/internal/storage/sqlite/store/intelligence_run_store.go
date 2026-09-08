package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

const intelligenceRunColumns = `
id, kind, project_id, intake_id, outcome_id, contract_revision_id,
source_revision, provider, model_selection, model, input_digest, output_digest,
native_session_ref, status, failure_code, failure_detail, created_at, completed_at`

// CreateIntelligenceRun persists one bounded reasoning run. The schema has no
// Attempt, WorkUnit, execution session, capability-grant, or acceptance column:
// persistence preserves the authority boundary expressed by domain.IntelligenceRun.
func (s *Store) CreateIntelligenceRun(ctx context.Context, run domain.IntelligenceRun) error {
	if err := run.Validate(); err != nil {
		return err
	}
	var completedAt any
	if run.CompletedAt != nil {
		completedAt = run.CompletedAt.UTC()
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.writeDB.ExecContext(ctx, `
INSERT INTO intelligence_runs (`+intelligenceRunColumns+`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.Kind, run.ProjectID, run.IntakeID, run.OutcomeID, run.ContractRevisionID,
		run.SourceRevision, run.Provider, run.ModelSelection, run.Model, run.InputDigest, run.OutputDigest,
		run.NativeSessionRef, run.Status, run.FailureCode, run.FailureDetail, run.CreatedAt.UTC(), completedAt,
	)
	if err != nil {
		return fmt.Errorf("create intelligence run %s: %w", run.ID, err)
	}
	return nil
}

// GetIntelligenceRun returns one durable intelligence run by id.
func (s *Store) GetIntelligenceRun(ctx context.Context, id domain.IntelligenceRunID) (domain.IntelligenceRun, bool, error) {
	row := s.readDB.QueryRowContext(ctx, `SELECT `+intelligenceRunColumns+` FROM intelligence_runs WHERE id = ?`, id)
	run, err := scanIntelligenceRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.IntelligenceRun{}, false, nil
	}
	if err != nil {
		return domain.IntelligenceRun{}, false, fmt.Errorf("get intelligence run %s: %w", id, err)
	}
	return run, true, nil
}

// ListNonTerminalIntelligenceRuns is the restart/reconciliation input. Unknown
// provider-native runtime state is resolved by the adapter/control plane; this
// query never infers completion from process absence.
func (s *Store) ListNonTerminalIntelligenceRuns(ctx context.Context) ([]domain.IntelligenceRun, error) {
	rows, err := s.readDB.QueryContext(ctx, `
SELECT `+intelligenceRunColumns+`
FROM intelligence_runs
WHERE status IN ('requested','running')
ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list non-terminal intelligence runs: %w", err)
	}
	defer rows.Close()

	var out []domain.IntelligenceRun
	for rows.Next() {
		run, err := scanIntelligenceRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan non-terminal intelligence run: %w", err)
		}
		out = append(out, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate non-terminal intelligence runs: %w", err)
	}
	return out, nil
}

// UpdateIntelligenceRunStatus performs a one-way state transition with an
// optimistic status fence. It cannot revive terminal work or manufacture an
// execution Attempt as a side effect.
func (s *Store) UpdateIntelligenceRunStatus(
	ctx context.Context,
	id domain.IntelligenceRunID,
	next domain.IntelligenceRunStatus,
	outputDigest string,
	failureCode string,
	failureDetail string,
	completedAt *time.Time,
) error {
	current, found, err := s.GetIntelligenceRun(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("intelligence run %s does not exist", id)
	}
	previous := current.Status
	if err := current.TransitionTo(next); err != nil {
		return err
	}
	current.OutputDigest = outputDigest
	current.FailureCode = failureCode
	current.FailureDetail = failureDetail
	if completedAt != nil {
		t := completedAt.UTC()
		current.CompletedAt = &t
	}
	if next.Terminal() && current.CompletedAt == nil {
		now := time.Now().UTC()
		current.CompletedAt = &now
	}
	if err := current.Validate(); err != nil {
		return err
	}

	var completed any
	if current.CompletedAt != nil {
		completed = current.CompletedAt.UTC()
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	result, err := s.writeDB.ExecContext(ctx, `
UPDATE intelligence_runs
SET status = ?, output_digest = ?, failure_code = ?, failure_detail = ?, completed_at = ?
WHERE id = ? AND status = ?`,
		current.Status, current.OutputDigest, current.FailureCode, current.FailureDetail, completed, current.ID, previous,
	)
	if err != nil {
		return fmt.Errorf("update intelligence run %s: %w", id, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update intelligence run %s rows affected: %w", id, err)
	}
	if changed != 1 {
		return fmt.Errorf("intelligence run %s changed concurrently", id)
	}
	return nil
}

type intelligenceRunScanner interface {
	Scan(dest ...any) error
}

func scanIntelligenceRun(scanner intelligenceRunScanner) (domain.IntelligenceRun, error) {
	var run domain.IntelligenceRun
	var completedAt sql.NullTime
	if err := scanner.Scan(
		&run.ID, &run.Kind, &run.ProjectID, &run.IntakeID, &run.OutcomeID, &run.ContractRevisionID,
		&run.SourceRevision, &run.Provider, &run.ModelSelection, &run.Model, &run.InputDigest, &run.OutputDigest,
		&run.NativeSessionRef, &run.Status, &run.FailureCode, &run.FailureDetail, &run.CreatedAt, &completedAt,
	); err != nil {
		return domain.IntelligenceRun{}, err
	}
	if completedAt.Valid {
		t := completedAt.Time
		run.CompletedAt = &t
	}
	if err := run.Validate(); err != nil {
		return domain.IntelligenceRun{}, fmt.Errorf("invalid persisted intelligence run %s: %w", run.ID, err)
	}
	return run, nil
}

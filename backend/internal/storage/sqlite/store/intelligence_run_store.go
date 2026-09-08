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
source_revision, requested_provider, requested_model, effective_provider,
effective_model, native_session_ref, input_digest, output_digest, status,
failure_code, failure_detail, created_at, completed_at`

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
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.Kind, run.ProjectID, nullString(string(run.IntakeID)), nullString(string(run.OutcomeID)), nullString(run.ContractRevisionID.String()),
		run.SourceRevision, run.RequestedProvider, run.RequestedModel, run.EffectiveProvider,
		run.EffectiveModel, run.NativeSessionRef, run.InputDigest, run.OutputDigest, run.Status,
		run.FailureCode, run.FailureDetail, run.CreatedAt.UTC(), completedAt,
	)
	if err != nil {
		return fmt.Errorf("create intelligence run %s: %w", run.ID, err)
	}
	return nil
}

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

// ListNonTerminalIntelligenceRuns is restart/reconciliation input. It never
// infers completion from provider silence or process absence.
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

// RecordIntelligenceRunEffectiveProvenance fills provider-reported provenance
// monotonically while a run is non-terminal. Unknown values may stay empty;
// a known value can never be replaced by a different one.
func (s *Store) RecordIntelligenceRunEffectiveProvenance(
	ctx context.Context,
	id domain.IntelligenceRunID,
	provider domain.IntelligenceProviderID,
	model string,
	nativeSessionRef string,
) error {
	if provider.IsZero() {
		return fmt.Errorf("effective intelligence provider is required")
	}
	current, found, err := s.GetIntelligenceRun(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("intelligence run %s does not exist", id)
	}
	if current.Status.Terminal() {
		if current.EffectiveProvider == provider && current.EffectiveModel == model && current.NativeSessionRef == nativeSessionRef {
			return nil
		}
		return fmt.Errorf("terminal intelligence run %s provenance is immutable", id)
	}
	if !current.EffectiveProvider.IsZero() && current.EffectiveProvider != provider {
		return fmt.Errorf("intelligence run %s effective provider is already %q", id, current.EffectiveProvider)
	}
	if current.EffectiveModel != "" && current.EffectiveModel != model {
		return fmt.Errorf("intelligence run %s effective model is already %q", id, current.EffectiveModel)
	}
	if current.NativeSessionRef != "" && current.NativeSessionRef != nativeSessionRef {
		return fmt.Errorf("intelligence run %s native session reference is already bound", id)
	}

	next := current
	next.EffectiveProvider = provider
	if model != "" {
		next.EffectiveModel = model
	}
	if nativeSessionRef != "" {
		next.NativeSessionRef = nativeSessionRef
	}
	if err := next.Validate(); err != nil {
		return err
	}
	if next.EffectiveProvider == current.EffectiveProvider && next.EffectiveModel == current.EffectiveModel && next.NativeSessionRef == current.NativeSessionRef {
		return nil
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	result, err := s.writeDB.ExecContext(ctx, `
UPDATE intelligence_runs
SET effective_provider = ?, effective_model = ?, native_session_ref = ?
WHERE id = ? AND status = ?
  AND effective_provider = ? AND effective_model = ? AND native_session_ref = ?`,
		next.EffectiveProvider, next.EffectiveModel, next.NativeSessionRef,
		current.ID, current.Status, current.EffectiveProvider, current.EffectiveModel, current.NativeSessionRef,
	)
	if err != nil {
		return fmt.Errorf("record intelligence run %s effective provenance: %w", id, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("record intelligence run %s provenance rows affected: %w", id, err)
	}
	if changed != 1 {
		return fmt.Errorf("intelligence run %s changed concurrently", id)
	}
	return nil
}

// UpdateIntelligenceRunStatus performs a one-way state transition. Replaying
// an identical terminal result is idempotent; changing terminal provenance is
// rejected rather than overwritten.
func (s *Store) UpdateIntelligenceRunStatus(
	ctx context.Context,
	id domain.IntelligenceRunID,
	next domain.IntelligenceRunStatus,
	outputDigest domain.SHA256Digest,
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

	normalizedCompletedAt := normalizeCompletionTime(completedAt)
	if current.Status == next {
		if current.OutputDigest == outputDigest && current.FailureCode == failureCode && current.FailureDetail == failureDetail && sameTimePtr(current.CompletedAt, normalizedCompletedAt) {
			return nil
		}
		if current.Status.Terminal() {
			return fmt.Errorf("terminal intelligence run %s result is immutable", id)
		}
		return fmt.Errorf("intelligence run %s status replay changed durable result fields", id)
	}

	previous := current.Status
	if err := current.TransitionTo(next, normalizedCompletedAt); err != nil {
		return err
	}
	current.OutputDigest = outputDigest
	current.FailureCode = failureCode
	current.FailureDetail = failureDetail
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

func normalizeCompletionTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	t := value.UTC()
	return &t
}

func sameTimePtr(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

type intelligenceRunScanner interface {
	Scan(dest ...any) error
}

func scanIntelligenceRun(scanner intelligenceRunScanner) (domain.IntelligenceRun, error) {
	var run domain.IntelligenceRun
	var intakeID, outcomeID, contractRevisionID sql.NullString
	var completedAt sql.NullTime
	if err := scanner.Scan(
		&run.ID, &run.Kind, &run.ProjectID, &intakeID, &outcomeID, &contractRevisionID,
		&run.SourceRevision, &run.RequestedProvider, &run.RequestedModel, &run.EffectiveProvider,
		&run.EffectiveModel, &run.NativeSessionRef, &run.InputDigest, &run.OutputDigest, &run.Status,
		&run.FailureCode, &run.FailureDetail, &run.CreatedAt, &completedAt,
	); err != nil {
		return domain.IntelligenceRun{}, err
	}
	if intakeID.Valid {
		run.IntakeID = domain.IntakeSessionID(intakeID.String)
	}
	if outcomeID.Valid {
		run.OutcomeID = domain.OutcomeID(outcomeID.String)
	}
	if contractRevisionID.Valid {
		run.ContractRevisionID = domain.ContractRevisionID(contractRevisionID.String)
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

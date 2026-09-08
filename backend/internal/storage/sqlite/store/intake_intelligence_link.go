package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// BindIntakeAnalysisRequestIntelligenceRun binds the callback envelope to one
// matching Contract-analysis run. It rejects wrong lineage before the write and
// the SQL/schema fences keep the association write-once under concurrency.
func (s *Store) BindIntakeAnalysisRequestIntelligenceRun(ctx context.Context, requestID domain.IntakeAnalysisRequestID, runID domain.IntelligenceRunID) error {
	if runID.IsZero() {
		return fmt.Errorf("intelligence run id is required")
	}

	var requestIntakeID string
	var requestStatus domain.IntakeAnalysisRequestStatus
	var sessionID string
	var harness domain.AgentHarness
	var linkedRunID sql.NullString
	var projectID sql.NullString
	err := s.readDB.QueryRowContext(ctx, `
SELECT r.intake_id, r.status, r.session_id, r.harness, r.intelligence_run_id, i.project_id
FROM intake_analysis_requests r
JOIN intake_sessions i ON i.id = r.intake_id
WHERE r.id = ?`, requestID).Scan(
		&requestIntakeID, &requestStatus, &sessionID, &harness, &linkedRunID, &projectID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("intake analysis request %s does not exist", requestID)
	}
	if err != nil {
		return fmt.Errorf("read intake analysis request %s lineage: %w", requestID, err)
	}
	if !requestStatus.Open() {
		return ports.ErrIntakeAnalysisRequestClosed
	}
	if linkedRunID.Valid {
		return ports.ErrIntakeAnalysisIntelligenceRunBound
	}
	if sessionID != "" || harness != "" {
		return fmt.Errorf("%w: request already has legacy session provenance", ports.ErrIntakeAnalysisIntelligenceLineage)
	}

	run, found, err := s.GetIntelligenceRun(ctx, runID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("intelligence run %s does not exist", runID)
	}
	if run.Kind != domain.IntelligenceRunContractAnalysis {
		return fmt.Errorf("%w: run %s has kind %q", ports.ErrIntakeAnalysisIntelligenceLineage, runID, run.Kind)
	}
	if run.IntakeID != domain.IntakeSessionID(requestIntakeID) {
		return fmt.Errorf("%w: run %s belongs to intake %s", ports.ErrIntakeAnalysisIntelligenceLineage, runID, run.IntakeID)
	}
	if !projectID.Valid || run.ProjectID != domain.ProjectID(projectID.String) {
		return fmt.Errorf("%w: run %s belongs to project %s", ports.ErrIntakeAnalysisIntelligenceLineage, runID, run.ProjectID)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	result, err := s.writeDB.ExecContext(ctx, `
UPDATE intake_analysis_requests
SET intelligence_run_id = ?
WHERE id = ? AND status = 'requested'
  AND intelligence_run_id IS NULL AND session_id = '' AND harness = ''`, runID, requestID)
	if err != nil {
		return fmt.Errorf("bind intake analysis request %s to intelligence run %s: %w", requestID, runID, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("bind intake analysis request %s rows affected: %w", requestID, err)
	}
	if changed != 1 {
		return fmt.Errorf("bind intake analysis request %s: request changed concurrently", requestID)
	}
	return nil
}

// GetIntakeAnalysisRequestIntelligenceRun returns the optional canonical run
// link. NULL is the valid historical/unmigrated representation.
func (s *Store) GetIntakeAnalysisRequestIntelligenceRun(ctx context.Context, requestID domain.IntakeAnalysisRequestID) (domain.IntelligenceRunID, bool, error) {
	var runID sql.NullString
	err := s.readDB.QueryRowContext(ctx, `
SELECT intelligence_run_id
FROM intake_analysis_requests
WHERE id = ?`, requestID).Scan(&runID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get intelligence run link for intake analysis request %s: %w", requestID, err)
	}
	if !runID.Valid {
		return "", false, nil
	}
	return domain.IntelligenceRunID(runID.String), true, nil
}

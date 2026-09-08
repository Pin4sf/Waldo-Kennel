package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// BindIntakeAnalysisRequestIntelligenceRun associates the existing single-use
// callback envelope with its canonical non-authoritative reasoning run. The
// migration guard makes the link write-once while the request is open.
func (s *Store) BindIntakeAnalysisRequestIntelligenceRun(ctx context.Context, requestID domain.IntakeAnalysisRequestID, runID domain.IntelligenceRunID) error {
	if runID.IsZero() {
		return fmt.Errorf("intelligence run id is required")
	}
	if _, found, err := s.GetIntelligenceRun(ctx, runID); err != nil {
		return err
	} else if !found {
		return fmt.Errorf("intelligence run %s does not exist", runID)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	result, err := s.writeDB.ExecContext(ctx, `
UPDATE intake_analysis_requests
SET intelligence_run_id = ?
WHERE id = ? AND status = 'requested' AND intelligence_run_id = ''`, runID, requestID)
	if err != nil {
		return fmt.Errorf("bind intake analysis request %s to intelligence run %s: %w", requestID, runID, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("bind intake analysis request %s rows affected: %w", requestID, err)
	}
	if changed != 1 {
		return ports.ErrIntakeAnalysisRequestClosed
	}
	return nil
}

// GetIntakeAnalysisRequestIntelligenceRun returns the optional new canonical
// run link. Empty historical rows are valid and reported as found=false.
func (s *Store) GetIntakeAnalysisRequestIntelligenceRun(ctx context.Context, requestID domain.IntakeAnalysisRequestID) (domain.IntelligenceRunID, bool, error) {
	var runID string
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
	id := domain.IntelligenceRunID(runID)
	if id.IsZero() {
		return "", false, nil
	}
	return id, true, nil
}

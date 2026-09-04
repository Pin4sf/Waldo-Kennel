package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// AppendProjectBriefRevision appends one immutable Project Brief revision and
// advances the explicit current projection in the same transaction. conflict is
// true when expectedCurrent no longer matches the durable head; in that case
// the transaction is rolled back and no orphan revision remains.
func (s *Store) AppendProjectBriefRevision(
	ctx context.Context,
	projectID domain.ProjectID,
	expectedCurrent int64,
	revision domain.ProjectBriefRevision,
) (domain.ProjectBriefRevision, bool, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf("begin project brief revision for %s: %w", projectID, err)
	}
	defer func() { _ = tx.Rollback() }()

	current, err := currentProjectBriefNumber(ctx, tx, projectID)
	if err != nil {
		return domain.ProjectBriefRevision{}, false, err
	}
	if current != expectedCurrent {
		return domain.ProjectBriefRevision{}, true, nil
	}

	maxRevision, err := maxProjectBriefNumber(ctx, tx, projectID)
	if err != nil {
		return domain.ProjectBriefRevision{}, false, err
	}
	if maxRevision != current {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf(
			"project brief history/head divergence for %s: max revision %d, current %d",
			projectID, maxRevision, current,
		)
	}

	revision.ProjectID = projectID
	revision.Number = current + 1
	if err := revision.Validate(); err != nil {
		return domain.ProjectBriefRevision{}, false, err
	}
	conventions, err := json.Marshal(revision.Conventions)
	if err != nil {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf("marshal project brief conventions: %w", err)
	}
	constraints, err := json.Marshal(revision.Constraints)
	if err != nil {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf("marshal project brief constraints: %w", err)
	}
	provenance, err := json.Marshal(revision.Provenance)
	if err != nil {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf("marshal project brief provenance: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO project_brief_revisions (
		id, project_id, revision_number, purpose, product_context, technical_context,
		architecture_summary, conventions_json, constraints_json, setup_expectations,
		run_expectations, test_expectations, provenance_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		revision.ID,
		revision.ProjectID,
		revision.Number,
		revision.Purpose,
		revision.ProductContext,
		revision.TechnicalContext,
		revision.ArchitectureSummary,
		string(conventions),
		string(constraints),
		revision.SetupExpectations,
		revision.RunExpectations,
		revision.TestExpectations,
		string(provenance),
		revision.CreatedAt,
	); err != nil {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf("insert project brief revision %s: %w", revision.ID, err)
	}

	if current == 0 {
		if _, err := tx.ExecContext(ctx, `INSERT INTO project_brief_heads (
			project_id, current_revision_number, updated_at
		) VALUES (?, ?, ?)`, projectID, revision.Number, revision.CreatedAt); err != nil {
			return domain.ProjectBriefRevision{}, false, fmt.Errorf("create project brief head for %s: %w", projectID, err)
		}
	} else {
		result, err := tx.ExecContext(ctx, `UPDATE project_brief_heads
			SET current_revision_number = ?, updated_at = ?
			WHERE project_id = ? AND current_revision_number = ?`,
			revision.Number, revision.CreatedAt, projectID, current,
		)
		if err != nil {
			return domain.ProjectBriefRevision{}, false, fmt.Errorf("advance project brief head for %s: %w", projectID, err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return domain.ProjectBriefRevision{}, false, fmt.Errorf("read project brief head update for %s: %w", projectID, err)
		}
		if rows != 1 {
			return domain.ProjectBriefRevision{}, true, nil
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf("commit project brief revision for %s: %w", projectID, err)
	}
	return revision, false, nil
}

// GetCurrentProjectBriefRevision reads the current immutable Brief snapshot.
func (s *Store) GetCurrentProjectBriefRevision(ctx context.Context, projectID domain.ProjectID) (domain.ProjectBriefRevision, bool, error) {
	row := s.readDB.QueryRowContext(ctx, `SELECT
		r.id, r.project_id, r.revision_number, r.purpose, r.product_context,
		r.technical_context, r.architecture_summary, r.conventions_json,
		r.constraints_json, r.setup_expectations, r.run_expectations,
		r.test_expectations, r.provenance_json, r.created_at
	FROM project_brief_heads h
	JOIN project_brief_revisions r
	  ON r.project_id = h.project_id
	 AND r.revision_number = h.current_revision_number
	WHERE h.project_id = ?`, projectID)
	revision, err := scanProjectBriefRevision(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProjectBriefRevision{}, false, nil
	}
	if err != nil {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf("get current project brief for %s: %w", projectID, err)
	}
	return revision, true, nil
}

// ListProjectBriefRevisions returns immutable history ordered oldest to newest.
func (s *Store) ListProjectBriefRevisions(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectBriefRevision, error) {
	rows, err := s.readDB.QueryContext(ctx, `SELECT
		id, project_id, revision_number, purpose, product_context, technical_context,
		architecture_summary, conventions_json, constraints_json, setup_expectations,
		run_expectations, test_expectations, provenance_json, created_at
	FROM project_brief_revisions
	WHERE project_id = ?
	ORDER BY revision_number ASC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project brief revisions for %s: %w", projectID, err)
	}
	defer rows.Close()

	revisions := make([]domain.ProjectBriefRevision, 0)
	for rows.Next() {
		revision, err := scanProjectBriefRevision(rows)
		if err != nil {
			return nil, fmt.Errorf("scan project brief revision for %s: %w", projectID, err)
		}
		revisions = append(revisions, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project brief revisions for %s: %w", projectID, err)
	}
	return revisions, nil
}

func currentProjectBriefNumber(ctx context.Context, tx *sql.Tx, projectID domain.ProjectID) (int64, error) {
	var current int64
	err := tx.QueryRowContext(ctx, `SELECT current_revision_number FROM project_brief_heads WHERE project_id = ?`, projectID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read project brief head for %s: %w", projectID, err)
	}
	return current, nil
}

func maxProjectBriefNumber(ctx context.Context, tx *sql.Tx, projectID domain.ProjectID) (int64, error) {
	var max int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(revision_number), 0)
		FROM project_brief_revisions WHERE project_id = ?`, projectID).Scan(&max); err != nil {
		return 0, fmt.Errorf("read max project brief revision for %s: %w", projectID, err)
	}
	return max, nil
}

type projectBriefScanner interface {
	Scan(dest ...any) error
}

func scanProjectBriefRevision(scanner projectBriefScanner) (domain.ProjectBriefRevision, error) {
	var revision domain.ProjectBriefRevision
	var conventionsJSON, constraintsJSON, provenanceJSON string
	if err := scanner.Scan(
		&revision.ID,
		&revision.ProjectID,
		&revision.Number,
		&revision.Purpose,
		&revision.ProductContext,
		&revision.TechnicalContext,
		&revision.ArchitectureSummary,
		&conventionsJSON,
		&constraintsJSON,
		&revision.SetupExpectations,
		&revision.RunExpectations,
		&revision.TestExpectations,
		&provenanceJSON,
		&revision.CreatedAt,
	); err != nil {
		return domain.ProjectBriefRevision{}, err
	}
	if err := json.Unmarshal([]byte(conventionsJSON), &revision.Conventions); err != nil {
		return domain.ProjectBriefRevision{}, fmt.Errorf("decode project brief conventions: %w", err)
	}
	if err := json.Unmarshal([]byte(constraintsJSON), &revision.Constraints); err != nil {
		return domain.ProjectBriefRevision{}, fmt.Errorf("decode project brief constraints: %w", err)
	}
	if err := json.Unmarshal([]byte(provenanceJSON), &revision.Provenance); err != nil {
		return domain.ProjectBriefRevision{}, fmt.Errorf("decode project brief provenance: %w", err)
	}
	return revision, nil
}

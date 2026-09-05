package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/gen"
)

// AppendProjectBriefRevision appends one immutable Project Brief revision and
// advances the explicit current projection in the same transaction. conflict is
// true when expectedCurrent no longer matches the durable head; in that case
// the transaction is rolled back and no orphan revision remains.
//
// The head advance carries the expected revision in its own WHERE clause, so a
// racing writer loses at the database rather than in a read-then-compare window
// this transaction would have to hold open.
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
	txq := s.qw.WithTx(tx)

	current, err := currentProjectBriefNumber(ctx, txq, projectID)
	if err != nil {
		return domain.ProjectBriefRevision{}, false, err
	}
	if current != expectedCurrent {
		return domain.ProjectBriefRevision{}, true, nil
	}

	maxRevision, err := txq.MaxProjectBriefRevisionNumber(ctx, string(projectID))
	if err != nil {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf("read max project brief revision for %s: %w", projectID, err)
	}
	maxNumber, ok := maxRevision.(int64)
	if !ok {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf("read max project brief revision for %s: unexpected type %T", projectID, maxRevision)
	}
	if maxNumber != current {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf(
			"project brief history/head divergence for %s: max revision %d, current %d",
			projectID, maxNumber, current,
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

	if err := txq.CreateProjectBriefRevision(ctx, gen.CreateProjectBriefRevisionParams{
		ID:                  string(revision.ID),
		ProjectID:           string(projectID),
		RevisionNumber:      revision.Number,
		Purpose:             revision.Purpose,
		ProductContext:      revision.ProductContext,
		TechnicalContext:    revision.TechnicalContext,
		ArchitectureSummary: revision.ArchitectureSummary,
		ConventionsJson:     string(conventions),
		ConstraintsJson:     string(constraints),
		SetupExpectations:   revision.SetupExpectations,
		RunExpectations:     revision.RunExpectations,
		TestExpectations:    revision.TestExpectations,
		ProvenanceJson:      string(provenance),
		CreatedAt:           revision.CreatedAt,
	}); err != nil {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf("insert project brief revision for %s: %w", projectID, err)
	}

	rows, err := txq.AdvanceProjectBriefHead(ctx, gen.AdvanceProjectBriefHeadParams{
		ProjectID:             string(projectID),
		CurrentRevisionNumber: revision.Number,
		UpdatedAt:             revision.CreatedAt,
	})
	if err != nil {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf("advance project brief head for %s: %w", projectID, err)
	}
	if rows != 1 {
		// The head moved between the read above and this write. Roll back rather
		// than leave a revision whose head never pointed at it.
		return domain.ProjectBriefRevision{}, true, nil
	}

	if err := tx.Commit(); err != nil {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf("commit project brief revision for %s: %w", projectID, err)
	}
	return revision, false, nil
}

// GetCurrentProjectBriefRevision reads the current immutable Brief snapshot.
func (s *Store) GetCurrentProjectBriefRevision(ctx context.Context, projectID domain.ProjectID) (domain.ProjectBriefRevision, bool, error) {
	row, err := s.qr.GetCurrentProjectBriefRevision(ctx, string(projectID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProjectBriefRevision{}, false, nil
	}
	if err != nil {
		return domain.ProjectBriefRevision{}, false, fmt.Errorf("get current project brief for %s: %w", projectID, err)
	}
	revision, err := projectBriefRevisionFromRow(row)
	if err != nil {
		return domain.ProjectBriefRevision{}, false, err
	}
	return revision, true, nil
}

// ListProjectBriefRevisions returns immutable history ordered oldest to newest.
func (s *Store) ListProjectBriefRevisions(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectBriefRevision, error) {
	rows, err := s.qr.ListProjectBriefRevisions(ctx, string(projectID))
	if err != nil {
		return nil, fmt.Errorf("list project brief revisions for %s: %w", projectID, err)
	}
	revisions := make([]domain.ProjectBriefRevision, 0, len(rows))
	for _, row := range rows {
		revision, err := projectBriefRevisionFromRow(row)
		if err != nil {
			return nil, fmt.Errorf("scan project brief revision for %s: %w", projectID, err)
		}
		revisions = append(revisions, revision)
	}
	return revisions, nil
}

// currentProjectBriefNumber reads the durable head, treating "no head yet" as
// revision 0 so the first append is an ordinary transition from nothing.
func currentProjectBriefNumber(ctx context.Context, q *gen.Queries, projectID domain.ProjectID) (int64, error) {
	current, err := q.GetProjectBriefHead(ctx, string(projectID))
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read project brief head for %s: %w", projectID, err)
	}
	return current, nil
}

func projectBriefRevisionFromRow(row gen.ProjectBriefRevision) (domain.ProjectBriefRevision, error) {
	revision := domain.ProjectBriefRevision{
		ID:                  domain.ProjectBriefRevisionID(row.ID),
		ProjectID:           domain.ProjectID(row.ProjectID),
		Number:              row.RevisionNumber,
		Purpose:             row.Purpose,
		ProductContext:      row.ProductContext,
		TechnicalContext:    row.TechnicalContext,
		ArchitectureSummary: row.ArchitectureSummary,
		SetupExpectations:   row.SetupExpectations,
		RunExpectations:     row.RunExpectations,
		TestExpectations:    row.TestExpectations,
		CreatedAt:           row.CreatedAt,
	}
	if err := json.Unmarshal([]byte(row.ConventionsJson), &revision.Conventions); err != nil {
		return domain.ProjectBriefRevision{}, fmt.Errorf("decode project brief conventions: %w", err)
	}
	if err := json.Unmarshal([]byte(row.ConstraintsJson), &revision.Constraints); err != nil {
		return domain.ProjectBriefRevision{}, fmt.Errorf("decode project brief constraints: %w", err)
	}
	if err := json.Unmarshal([]byte(row.ProvenanceJson), &revision.Provenance); err != nil {
		return domain.ProjectBriefRevision{}, fmt.Errorf("decode project brief provenance: %w", err)
	}
	return revision, nil
}

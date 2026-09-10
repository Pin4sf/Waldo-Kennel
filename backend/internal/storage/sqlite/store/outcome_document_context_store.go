package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/gen"
)

var _ ports.DocumentContextStore = (*Store)(nil)

// AppendDocumentContext records one new selection revision and its sources in
// one transaction.
//
// The revision is chosen inside the write, not by the caller: two concurrent
// selections must not both become revision 2, and the unique index is what
// makes the loser fail rather than overwrite.
func (s *Store) AppendDocumentContext(ctx context.Context, selection domain.OutcomeDocumentContext) (domain.OutcomeDocumentContext, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return domain.OutcomeDocumentContext{}, fmt.Errorf("begin append document context: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	maximum, err := txq.MaxOutcomeDocumentContextRevision(ctx, string(selection.OutcomeID))
	if err != nil {
		return domain.OutcomeDocumentContext{}, fmt.Errorf("read current document context revision: %w", err)
	}
	selection.Revision = maximum + 1
	selection.State = domain.DocumentContextSelected
	selection.ApprovedAt = nil
	if err := selection.Validate(); err != nil {
		return domain.OutcomeDocumentContext{}, err
	}

	if err := txq.CreateOutcomeDocumentContext(ctx, gen.CreateOutcomeDocumentContextParams{
		ID: string(selection.ID), OutcomeID: string(selection.OutcomeID), Revision: selection.Revision,
		Digest: selection.Digest, State: string(selection.State), SelectedAt: selection.SelectedAt.UTC(),
	}); err != nil {
		return domain.OutcomeDocumentContext{}, fmt.Errorf("create document context: %w", err)
	}
	for _, source := range selection.Sources {
		if err := txq.CreateOutcomeDocumentSource(ctx, gen.CreateOutcomeDocumentSourceParams{
			ID: source.ID, ContextID: string(selection.ID), Position: int64(source.Position),
			SourcePath: source.SourcePath, Name: source.Name,
			ContentDigest: source.ContentDigest, SizeBytes: source.SizeBytes,
		}); err != nil {
			return domain.OutcomeDocumentContext{}, fmt.Errorf("record selected document %q: %w", source.Name, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.OutcomeDocumentContext{}, fmt.Errorf("commit document context: %w", err)
	}
	return selection, nil
}

// CurrentDocumentContext reads the latest selection revision for an Outcome.
func (s *Store) CurrentDocumentContext(ctx context.Context, outcomeID domain.OutcomeID) (domain.OutcomeDocumentContext, bool, error) {
	row, err := s.qr.CurrentOutcomeDocumentContext(ctx, string(outcomeID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.OutcomeDocumentContext{}, false, nil
	}
	if err != nil {
		return domain.OutcomeDocumentContext{}, false, fmt.Errorf("current document context for %s: %w", outcomeID, err)
	}
	return s.hydrateDocumentContext(ctx, row)
}

// GetDocumentContext reads one selection revision by identity.
func (s *Store) GetDocumentContext(ctx context.Context, id domain.DocumentContextID) (domain.OutcomeDocumentContext, bool, error) {
	row, err := s.qr.GetOutcomeDocumentContext(ctx, string(id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.OutcomeDocumentContext{}, false, nil
	}
	if err != nil {
		return domain.OutcomeDocumentContext{}, false, fmt.Errorf("document context %s: %w", id, err)
	}
	return s.hydrateDocumentContext(ctx, row)
}

// ApproveDocumentContext records the owner's approval of a selection. It is
// write-once and one-way; a second approval changes nothing.
func (s *Store) ApproveDocumentContext(ctx context.Context, id domain.DocumentContextID, at time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, err := s.qw.ApproveOutcomeDocumentContext(ctx, gen.ApproveOutcomeDocumentContextParams{
		ApprovedAt: sql.NullTime{Time: at.UTC(), Valid: true}, ID: string(id),
	}); err != nil {
		return fmt.Errorf("approve document context %s: %w", id, err)
	}
	return nil
}

func (s *Store) hydrateDocumentContext(ctx context.Context, row gen.OutcomeDocumentContext) (domain.OutcomeDocumentContext, bool, error) {
	selection := domain.OutcomeDocumentContext{
		ID: domain.DocumentContextID(row.ID), OutcomeID: domain.OutcomeID(row.OutcomeID),
		Revision: row.Revision, Digest: row.Digest,
		State: domain.DocumentContextState(row.State), SelectedAt: row.SelectedAt,
	}
	if row.ApprovedAt.Valid {
		at := row.ApprovedAt.Time
		selection.ApprovedAt = &at
	}
	sources, err := s.qr.ListOutcomeDocumentSources(ctx, row.ID)
	if err != nil {
		return domain.OutcomeDocumentContext{}, false, fmt.Errorf("list documents for %s: %w", row.ID, err)
	}
	for _, source := range sources {
		selection.Sources = append(selection.Sources, domain.DocumentSource{
			ID: source.ID, Position: int(source.Position), SourcePath: source.SourcePath,
			Name: source.Name, ContentDigest: source.ContentDigest, SizeBytes: source.SizeBytes,
		})
	}
	if err := selection.Validate(); err != nil {
		return domain.OutcomeDocumentContext{}, true, fmt.Errorf("document context %s failed readback validation: %w", row.ID, err)
	}
	return selection, true, nil
}

package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/gen"
)

// Durable record of what one Attempt produced. See migration 0119.

// SaveAttemptReceipt writes a receipt and its file manifest as one unit.
//
// Receipt and manifest must move together: a receipt whose artifact version
// describes a manifest that failed to write would claim provenance it does not
// have, and a downstream handoff verifies against exactly that version.
//
// A frozen receipt is never replaced. The service refuses first, but this
// returns the typed refusal even if it does not, because "later work cannot
// silently overwrite a reviewed artifact" has to hold at the write path.
func (s *Store) SaveAttemptReceipt(ctx context.Context, receipt domain.AttemptReceipt) error {
	if err := receipt.Validate(); err != nil {
		return err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save attempt receipt %s: %w", receipt.AttemptID, err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	existing, err := txq.GetAttemptReceipt(ctx, string(receipt.AttemptID))
	switch {
	case err == nil && existing.FrozenAt.Valid:
		return ports.ErrAttemptReceiptFrozen
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("read attempt receipt %s: %w", receipt.AttemptID, err)
	}

	now := receipt.UpdatedAt
	if now.IsZero() {
		now = receipt.ObservedAt
	}
	created := receipt.CreatedAt
	if created.IsZero() {
		created = now
	}

	if err := txq.UpsertAttemptReceipt(ctx, gen.UpsertAttemptReceiptParams{
		AttemptID:              string(receipt.AttemptID),
		OutcomeID:              string(receipt.OutcomeID),
		PlanRevisionID:         string(receipt.PlanRevisionID),
		WorkUnitID:             string(receipt.WorkUnitID),
		ContractRevisionNumber: receipt.ContractRevisionNumber,
		ArtifactVersion:        receipt.ArtifactVersion,
		WorkspaceKind:          string(receipt.WorkspaceKind),
		WorkspacePath:          receipt.WorkspacePath,
		RepositoryPath:         receipt.RepositoryPath,
		RepositoryIdentity:     receipt.RepositoryIdentity,
		BaseRevision:           receipt.BaseRevision,
		ResultRevision:         receipt.ResultRevision,
		WorkspaceDirty:         boolToInt(receipt.WorkspaceDirty),
		RetentionState:         string(receipt.RetentionState),
		RetentionDetail:        receipt.RetentionDetail,
		TerminationReason:      receipt.TerminationReason,
		ObservedAt:             receipt.ObservedAt.UTC(),
		CreatedAt:              created.UTC(),
		UpdatedAt:              now.UTC(),
	}); err != nil {
		return fmt.Errorf("save attempt receipt %s: %w", receipt.AttemptID, err)
	}

	// Replace the manifest wholesale: a re-run of retention describes the
	// workspace as it is now, and merging with a previous partial read would
	// produce a manifest matching no actual state.
	if err := txq.DeleteAttemptArtifactFiles(ctx, string(receipt.AttemptID)); err != nil {
		return fmt.Errorf("clear artifact manifest for %s: %w", receipt.AttemptID, err)
	}
	for _, file := range receipt.Files {
		id := file.ID
		if id == "" {
			id = "artifact-" + uuid.NewString()
		}
		if err := txq.InsertAttemptArtifactFile(ctx, gen.InsertAttemptArtifactFileParams{
			ID:                id,
			AttemptID:         string(receipt.AttemptID),
			RelativePath:      file.RelativePath,
			ChangeKind:        string(file.ChangeKind),
			ContentDigest:     file.ContentDigest,
			SizeBytes:         nullInt64(file.SizeBytes),
			FileMode:          nullInt64(file.FileMode),
			IsBinary:          boolToInt(file.IsBinary),
			UnsupportedReason: file.UnsupportedReason,
		}); err != nil {
			return fmt.Errorf("save artifact file %s for %s: %w", file.RelativePath, receipt.AttemptID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit attempt receipt %s: %w", receipt.AttemptID, err)
	}
	return nil
}

// GetAttemptReceipt reads one receipt with its manifest.
func (s *Store) GetAttemptReceipt(ctx context.Context, attemptID domain.AttemptID) (domain.AttemptReceipt, bool, error) {
	row, err := s.qr.GetAttemptReceipt(ctx, string(attemptID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AttemptReceipt{}, false, nil
	}
	if err != nil {
		return domain.AttemptReceipt{}, false, fmt.Errorf("read attempt receipt %s: %w", attemptID, err)
	}
	files, err := s.qr.ListAttemptArtifactFiles(ctx, string(attemptID))
	if err != nil {
		return domain.AttemptReceipt{}, false, fmt.Errorf("read artifact manifest %s: %w", attemptID, err)
	}
	return attemptReceiptFromRow(row, files), true, nil
}

// FreezeAttemptReceipt marks a receipt as review evidence, after which it is
// never replaced. Freezing twice is a no-op so the caller stays idempotent.
func (s *Store) FreezeAttemptReceipt(ctx context.Context, attemptID domain.AttemptID, at time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.qw.FreezeAttemptReceipt(ctx, gen.FreezeAttemptReceiptParams{
		FrozenAt:  sql.NullTime{Time: at.UTC(), Valid: true},
		UpdatedAt: at.UTC(),
		AttemptID: string(attemptID),
	}); err != nil {
		return fmt.Errorf("freeze attempt receipt %s: %w", attemptID, err)
	}
	return nil
}

func attemptReceiptFromRow(row gen.AttemptReceipt, files []gen.AttemptArtifactFile) domain.AttemptReceipt {
	receipt := domain.AttemptReceipt{
		AttemptID:              domain.AttemptID(row.AttemptID),
		OutcomeID:              domain.OutcomeID(row.OutcomeID),
		PlanRevisionID:         domain.PlanRevisionID(row.PlanRevisionID),
		WorkUnitID:             domain.WorkUnitID(row.WorkUnitID),
		ContractRevisionNumber: row.ContractRevisionNumber,
		ArtifactVersion:        row.ArtifactVersion,
		WorkspaceKind:          domain.WorkspaceKind(row.WorkspaceKind),
		WorkspacePath:          row.WorkspacePath,
		RepositoryPath:         row.RepositoryPath,
		RepositoryIdentity:     row.RepositoryIdentity,
		BaseRevision:           row.BaseRevision,
		ResultRevision:         row.ResultRevision,
		WorkspaceDirty:         row.WorkspaceDirty == 1,
		RetentionState:         domain.RetentionState(row.RetentionState),
		RetentionDetail:        row.RetentionDetail,
		TerminationReason:      row.TerminationReason,
		ObservedAt:             row.ObservedAt,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
	if row.FrozenAt.Valid {
		frozen := row.FrozenAt.Time
		receipt.FrozenAt = &frozen
	}
	for _, file := range files {
		receipt.Files = append(receipt.Files, domain.ArtifactFile{
			ID:                file.ID,
			AttemptID:         domain.AttemptID(file.AttemptID),
			RelativePath:      file.RelativePath,
			ChangeKind:        domain.ArtifactChangeKind(file.ChangeKind),
			ContentDigest:     file.ContentDigest,
			SizeBytes:         int64Ptr(file.SizeBytes),
			FileMode:          int64Ptr(file.FileMode),
			IsBinary:          file.IsBinary == 1,
			UnsupportedReason: file.UnsupportedReason,
		})
	}
	return receipt
}

func boolToInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func nullInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func int64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	out := value.Int64
	return &out
}

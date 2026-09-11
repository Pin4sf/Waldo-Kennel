package outcome

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

func (s *Service) deletionStore() (ports.OutcomeDeletionStore, error) {
	store, ok := s.store.(ports.OutcomeDeletionStore)
	if !ok {
		return nil, apierr.Invalid("OUTCOME_DELETION_UNAVAILABLE", "Outcome deletion is unavailable in this daemon", nil)
	}
	return store, nil
}

// PreviewOutcomeDeletion returns the current cleanup scope and blockers.
func (s *Service) PreviewOutcomeDeletion(ctx context.Context, id domain.OutcomeID) (ports.OutcomeDeletionPreview, error) {
	store, err := s.deletionStore()
	if err != nil {
		return ports.OutcomeDeletionPreview{}, err
	}
	return store.PreviewOutcomeDeletion(ctx, id)
}

// ListTrashedOutcomes lists recoverable roots for one project.
func (s *Service) ListTrashedOutcomes(ctx context.Context, project domain.ProjectID) ([]ports.OutcomeTrashEntry, error) {
	store, err := s.deletionStore()
	if err != nil {
		return nil, err
	}
	return store.ListTrashedOutcomes(ctx, project)
}

// ChangeOutcomeDeletion applies the owner-requested Trash, Restore or permanent cleanup action.
func (s *Service) ChangeOutcomeDeletion(ctx context.Context, id domain.OutcomeID, revision int64, action, confirmation string) error {
	store, err := s.deletionStore()
	if err != nil {
		return err
	}
	switch action {
	case "trash":
		return store.ChangeOutcomeTrash(ctx, id, revision, true)
	case "restore":
		return store.ChangeOutcomeTrash(ctx, id, revision, false)
	case "permanent":
	default:
		return apierr.Invalid("OUTCOME_DELETION_ACTION_INVALID", "Choose trash, restore or permanent", nil)
	}
	p, err := store.PreviewOutcomeDeletion(ctx, id)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(confirmation), "confirm") {
		return apierr.Invalid("OUTCOME_DELETION_CONFIRMATION_REQUIRED", "Type confirm to permanently delete this Outcome", nil)
	}
	if p.Revision != revision {
		return apierr.Invalid("OUTCOME_DELETION_STALE", "The Contract changed; refresh the deletion preview", nil)
	}
	if len(p.Blockers) > 0 {
		return apierr.Invalid("OUTCOME_DELETION_BLOCKED", strings.Join(p.Blockers, " "), nil)
	}
	if s.deliveryArtifacts == nil && (len(p.AttemptIDs) > 0 || len(p.DocumentContextIDs) > 0) {
		return apierr.Invalid("OUTCOME_CLEANUP_UNAVAILABLE", "The retained artifact store is unavailable", nil)
	}
	if len(p.SessionIDs) > 0 && s.spawner == nil {
		return apierr.Invalid("OUTCOME_CLEANUP_UNAVAILABLE", "Session cleanup is unavailable", nil)
	}
	if !p.Trashed {
		if err := store.ChangeOutcomeTrash(ctx, id, revision, true); err != nil {
			return err
		}
	}
	// This durable marker prevents Restore once irreversible cleanup begins.
	// A failure keeps the records in Trash, where the owner can retry cleanup.
	if err := store.BeginOutcomePurge(ctx, id, revision); err != nil {
		return err
	}
	p, err = store.PreviewOutcomeDeletion(ctx, id)
	if err != nil {
		return err
	}
	for _, sid := range p.SessionIDs {
		result, err := s.spawner.Terminate(ctx, domain.ProjectID(p.ProjectID), sid)
		if err != nil {
			return fmt.Errorf("session cleanup %s: %w", sid, err)
		}
		if !result.ProviderStopped {
			return apierr.Invalid("OUTCOME_CLEANUP_BLOCKED", "Session cleanup is incomplete; inspect its workspace before retrying: "+sid, nil)
		}
	}
	for _, path := range p.WorkspacePaths {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			return apierr.Invalid("OUTCOME_CLEANUP_BLOCKED", "Workspace cleanup is incomplete; inspect before retrying: "+path, nil)
		}
	}
	if s.deliveryArtifacts != nil {
		if err := s.deliveryArtifacts.RemoveOutcomeContent(ctx, p.AttemptIDs, p.DocumentContextIDs); err != nil {
			return err
		}
	}
	return store.PurgeOutcomeRecords(ctx, id, revision)
}

package ports

import (
	"context"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// OutcomeDeletionPreview names the scope of an owner-directed lifecycle action.
type OutcomeDeletionPreview struct {
	ProjectID          string   `json:"projectId"`
	Erasing            bool     `json:"erasing"`
	OutcomeID          string   `json:"outcomeId"`
	Title              string   `json:"title"`
	Revision           int64    `json:"revision"`
	Trashed            bool     `json:"trashed"`
	OutcomeCount       int      `json:"outcomeCount"`
	RecordCount        int      `json:"recordCount"`
	SessionIDs         []string `json:"sessionIds"`
	AttemptIDs         []string `json:"attemptIds"`
	WorkspacePaths     []string `json:"workspacePaths"`
	DocumentContextIDs []string `json:"documentContextIds"`
	Blockers           []string `json:"blockers"`
}

// OutcomeDeletionStore owns atomic Trash and erasure transitions.
type OutcomeDeletionStore interface {
	PreviewOutcomeDeletion(context.Context, domain.OutcomeID) (OutcomeDeletionPreview, error)
	ChangeOutcomeTrash(context.Context, domain.OutcomeID, int64, bool) error
	BeginOutcomePurge(context.Context, domain.OutcomeID, int64) error
	PurgeOutcomeRecords(context.Context, domain.OutcomeID, int64) error
	ListTrashedOutcomes(context.Context, domain.ProjectID) ([]OutcomeTrashEntry, error)
}

// OutcomeTrashEntry is list metadata; detailed scope is loaded explicitly.
type OutcomeTrashEntry struct {
	OutcomeID string `json:"outcomeId"`
	Title     string `json:"title"`
	Revision  int64  `json:"revision"`
	Trashed   bool   `json:"trashed"`
	Erasing   bool   `json:"erasing"`
}

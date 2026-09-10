package ports

import (
	"context"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// DocumentContextStore owns which local documents an Outcome is about.
//
// It is separate from OutcomeStore because supplied-document work is one
// supported shape of Outcome, not a property of all of them: a daemon without
// it simply cannot run document Outcomes, which is a truthful reduced
// capability rather than a half-built one.
type DocumentContextStore interface {
	// AppendDocumentContext records a new selection revision. Implementations
	// choose the revision inside the write and always store it unapproved:
	// selecting is not approving.
	AppendDocumentContext(context.Context, domain.OutcomeDocumentContext) (domain.OutcomeDocumentContext, error)
	CurrentDocumentContext(context.Context, domain.OutcomeID) (domain.OutcomeDocumentContext, bool, error)
	GetDocumentContext(context.Context, domain.DocumentContextID) (domain.OutcomeDocumentContext, bool, error)
	// ApproveDocumentContext is write-once and one-way.
	ApproveDocumentContext(context.Context, domain.DocumentContextID, time.Time) error
}

// DocumentSnapshotStore holds the approved bytes themselves.
//
// Execution and reasoning both read these, never the owner's originals, so an
// Outcome cannot come to mean something different because a file was edited
// after it was approved.
type DocumentSnapshotStore interface {
	SnapshotDocuments(context.Context, domain.DocumentContextID, []string) ([]domain.DocumentSource, error)
	ReadDocument(domain.DocumentContextID, domain.DocumentSource) ([]byte, error)
	// SourcesChangedSince reports selected documents whose bytes on disk no
	// longer match what was approved. It never re-reads them into the run.
	SourcesChangedSince([]domain.DocumentSource) []string
}

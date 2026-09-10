package outcome

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// Stable refusals for the supplied-document path.
const (
	// CodeDocumentContextUnavailable means this daemon cannot hold supplied
	// documents at all.
	CodeDocumentContextUnavailable = "DOCUMENT_CONTEXT_UNAVAILABLE"
	// CodeDocumentSelectionInvalid means the selection itself is not usable.
	CodeDocumentSelectionInvalid = "DOCUMENT_SELECTION_INVALID"
	// CodeDocumentContextNotApproved means execution was requested against a
	// selection the owner has not reviewed.
	CodeDocumentContextNotApproved = "DOCUMENT_CONTEXT_NOT_APPROVED"
	// CodeDocumentContextStale means the approval names a revision that is no
	// longer this Outcome's current selection.
	CodeDocumentContextStale = "DOCUMENT_CONTEXT_STALE"
	// CodeDocumentSourcesChanged means the owner's files have been edited
	// since the approved snapshot was taken.
	CodeDocumentSourcesChanged = "DOCUMENT_SOURCES_CHANGED"
)

// DocumentContextView is the owner-facing projection of a selection.
//
// ChangedSources is derived on read: the run itself always uses the approved
// snapshot, and this only tells the owner their material has moved on so they
// can deliberately re-select.
type DocumentContextView struct {
	Context        domain.OutcomeDocumentContext
	ChangedSources []string
}

// DocumentsEnabled reports whether this daemon can hold supplied documents.
func (s *Service) DocumentsEnabled() bool {
	return s != nil && s.documents != nil && s.documentBytes != nil
}

// SelectDocuments records a new selection revision and snapshots its bytes.
//
// Selecting is deliberately not approving. The snapshot is taken now so the
// scope the owner reviews is fixed material rather than whatever the files
// happen to contain when they get around to it.
func (s *Service) SelectDocuments(ctx context.Context, outcomeID domain.OutcomeID, paths []string) (DocumentContextView, error) {
	if !s.DocumentsEnabled() {
		return DocumentContextView{}, apierr.Internal(CodeDocumentContextUnavailable, "Supplied-document context is not wired in this daemon")
	}
	if _, ok, err := s.store.GetOutcome(ctx, outcomeID); err != nil {
		return DocumentContextView{}, err
	} else if !ok {
		return DocumentContextView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	cleaned := make([]string, 0, len(paths))
	for _, path := range paths {
		if trimmed := strings.TrimSpace(path); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	if len(cleaned) == 0 {
		return DocumentContextView{}, apierr.Invalid(CodeDocumentSelectionInvalid, "Select at least one document", nil)
	}

	contextID := domain.DocumentContextID("dctx-" + uuid.NewString())
	sources, err := s.documentBytes.SnapshotDocuments(ctx, contextID, cleaned)
	if err != nil {
		return DocumentContextView{}, apierr.Invalid(CodeDocumentSelectionInvalid, err.Error(), nil)
	}
	stored, err := s.documents.AppendDocumentContext(ctx, domain.OutcomeDocumentContext{
		ID: contextID, OutcomeID: outcomeID, Digest: domain.DocumentContextDigest(sources),
		SelectedAt: s.clock(), Sources: sources,
	})
	if err != nil {
		return DocumentContextView{}, err
	}
	return DocumentContextView{Context: stored}, nil
}

// ApproveDocuments records that the owner reviewed the selected scope.
//
// The expected digest is required: approving "the current selection" without
// naming it would let a selection made after the owner looked away be
// approved by a click aimed at the one they read.
func (s *Service) ApproveDocuments(ctx context.Context, outcomeID domain.OutcomeID, expectedDigest string) (DocumentContextView, error) {
	if !s.DocumentsEnabled() {
		return DocumentContextView{}, apierr.Internal(CodeDocumentContextUnavailable, "Supplied-document context is not wired in this daemon")
	}
	current, found, err := s.documents.CurrentDocumentContext(ctx, outcomeID)
	if err != nil {
		return DocumentContextView{}, err
	}
	if !found {
		return DocumentContextView{}, apierr.Conflict(CodeDocumentContextStale, "Select documents before approving them", nil)
	}
	if expected := strings.TrimSpace(expectedDigest); expected != "" && expected != current.Digest {
		return DocumentContextView{}, apierr.Conflict(CodeDocumentContextStale,
			"The selection changed since you reviewed it; reload and approve the current one",
			map[string]any{"expectedDigest": expected, "currentDigest": current.Digest})
	}
	if !current.Approved() {
		if err := s.documents.ApproveDocumentContext(ctx, current.ID, s.clock()); err != nil {
			return DocumentContextView{}, err
		}
	}
	return s.GetDocumentContext(ctx, outcomeID)
}

// GetDocumentContext projects the current selection plus whether the owner's
// files have been edited since it was approved.
func (s *Service) GetDocumentContext(ctx context.Context, outcomeID domain.OutcomeID) (DocumentContextView, error) {
	if !s.DocumentsEnabled() {
		return DocumentContextView{}, apierr.Internal(CodeDocumentContextUnavailable, "Supplied-document context is not wired in this daemon")
	}
	current, found, err := s.documents.CurrentDocumentContext(ctx, outcomeID)
	if err != nil {
		return DocumentContextView{}, err
	}
	if !found {
		return DocumentContextView{}, apierr.NotFound(CodeDocumentContextStale, "This Outcome has no selected documents")
	}
	return DocumentContextView{
		Context:        current,
		ChangedSources: s.documentBytes.SourcesChangedSince(current.Sources),
	}, nil
}

// approvedDocumentsForAdmission resolves the selection an Attempt may be
// staged from, refusing anything the owner has not reviewed.
//
// An Outcome with no selection is not a document Outcome and passes through:
// this path adds a requirement to supplied-document work, it does not impose
// one on repository work.
func (s *Service) approvedDocumentsForAdmission(ctx context.Context, outcomeID domain.OutcomeID) (domain.OutcomeDocumentContext, bool, error) {
	if !s.DocumentsEnabled() {
		return domain.OutcomeDocumentContext{}, false, nil
	}
	current, found, err := s.documents.CurrentDocumentContext(ctx, outcomeID)
	if err != nil {
		return domain.OutcomeDocumentContext{}, false, err
	}
	if !found {
		return domain.OutcomeDocumentContext{}, false, nil
	}
	if !current.Approved() {
		return domain.OutcomeDocumentContext{}, false, apierr.Conflict(CodeDocumentContextNotApproved,
			"Review and approve the selected documents before starting work on them",
			map[string]any{"outcomeId": string(outcomeID), "revision": current.Revision})
	}
	// The run uses the approved snapshot either way; changed sources are
	// refused so the owner decides whether the newer material is what they
	// meant, rather than getting a result about bytes they have replaced.
	if changed := s.documentBytes.SourcesChangedSince(current.Sources); len(changed) > 0 {
		return domain.OutcomeDocumentContext{}, false, apierr.Conflict(CodeDocumentSourcesChanged,
			"Selected documents have been edited since you approved them; re-select and approve to use the new content",
			map[string]any{"outcomeId": string(outcomeID), "changed": changed})
	}
	return current, true, nil
}

// groundInSelectedDocuments replaces the reasoning packet's file set with the
// Outcome's selected document snapshot.
//
// Reading the snapshot rather than the owner's originals is what makes the
// grounding digest mean something: the proposal is bound to bytes that cannot
// change under it. An Outcome with no selection is untouched, so repository
// work is unaffected.
func (s *Service) groundInSelectedDocuments(ctx context.Context, outcomeID domain.OutcomeID, snapshot *ports.RepositoryContextSnapshot) error {
	if !s.DocumentsEnabled() || snapshot == nil {
		return nil
	}
	current, found, err := s.documents.CurrentDocumentContext(ctx, outcomeID)
	if err != nil || !found {
		return err
	}
	files := make([]ports.RepositoryContextFile, 0, len(current.Sources))
	for _, source := range current.Sources {
		body, err := s.documentBytes.ReadDocument(current.ID, source)
		if err != nil {
			return apierr.Internal(CodeDocumentContextUnavailable,
				"A selected document could not be read from its approved snapshot")
		}
		files = append(files, ports.RepositoryContextFile{Path: source.Name, Content: string(body)})
	}
	snapshot.Root = "supplied-documents"
	snapshot.Revision = current.Digest
	snapshot.Dirty = false
	snapshot.Files = files
	// Instructions and check commands are repository facts. A document
	// Outcome has neither, and carrying them over would describe material the
	// owner did not select.
	snapshot.Instructions = nil
	snapshot.CheckCommands = nil
	return nil
}

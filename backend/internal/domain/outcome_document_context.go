package domain

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// DocumentContextID identifies one selected set of supplied documents.
type DocumentContextID string

// IsZero reports an unset document context id.
func (id DocumentContextID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// MaxSuppliedDocuments bounds one selection. Named operational policy, not a
// law of Outcomes.
const MaxSuppliedDocuments = 32

// DocumentSource is one owner-selected local document, as it was when it was
// selected.
//
// SourcePath is where it came from and is kept for provenance only: execution
// reads the snapshot, never the original, so an Outcome cannot quietly pick up
// an edit the owner never approved.
type DocumentSource struct {
	ID            string
	Position      int
	SourcePath    string
	Name          string
	ContentDigest string
	SizeBytes     int64
}

// Validate checks one selected document.
func (d DocumentSource) Validate() error {
	if strings.TrimSpace(d.ID) == "" {
		return fmt.Errorf("document source id is required")
	}
	if !filepath.IsAbs(d.SourcePath) {
		return fmt.Errorf("document source path %q must be absolute", d.SourcePath)
	}
	if err := validateArtifactPath(d.Name); err != nil {
		return fmt.Errorf("document name %q must stay inside the staged workspace", d.Name)
	}
	if len(d.ContentDigest) != 64 {
		return fmt.Errorf("document %q is missing content identity", d.Name)
	}
	if d.SizeBytes < 0 {
		return fmt.Errorf("document %q has a negative size", d.Name)
	}
	return nil
}

// DocumentContextState is how far a selection has got.
type DocumentContextState string

// The two states a selection can be in.
const (
	// DocumentContextSelected means the owner chose these documents and their
	// bytes are snapshotted, but nothing may execute against them yet.
	DocumentContextSelected DocumentContextState = "selected"
	// DocumentContextApproved means the owner reviewed the scope. Only an
	// approved context may be staged for execution.
	DocumentContextApproved DocumentContextState = "approved"
)

// OutcomeDocumentContext is one immutable revision of "which local documents
// this Outcome is about".
//
// Revisions are append-only. Re-selecting after an edit produces a new
// revision rather than changing what was already approved, so a Plan
// grounded in one set of bytes can never silently come to mean another.
type OutcomeDocumentContext struct {
	ID        DocumentContextID
	OutcomeID OutcomeID
	Revision  int64
	// Digest identifies the selected bytes as a set. It is what binds a
	// proposal to the material it was grounded in.
	Digest     string
	State      DocumentContextState
	SelectedAt time.Time
	ApprovedAt *time.Time
	Sources    []DocumentSource
}

// Approved reports whether execution may be staged from this context.
func (c OutcomeDocumentContext) Approved() bool { return c.State == DocumentContextApproved }

// Validate checks one selection revision.
func (c OutcomeDocumentContext) Validate() error {
	if c.ID.IsZero() {
		return fmt.Errorf("document context id is required")
	}
	if c.OutcomeID.IsZero() {
		return fmt.Errorf("document context outcome id is required")
	}
	if c.Revision < 1 {
		return fmt.Errorf("document context revision must be at least 1")
	}
	if len(c.Sources) == 0 {
		return fmt.Errorf("a document context must select at least one document")
	}
	if len(c.Sources) > MaxSuppliedDocuments {
		return fmt.Errorf("document context selects %d documents; the limit is %d", len(c.Sources), MaxSuppliedDocuments)
	}
	names := make(map[string]bool, len(c.Sources))
	for _, source := range c.Sources {
		if err := source.Validate(); err != nil {
			return err
		}
		if names[source.Name] {
			return fmt.Errorf("two selected documents would stage as %q", source.Name)
		}
		names[source.Name] = true
	}
	if c.Digest != DocumentContextDigest(c.Sources) {
		return fmt.Errorf("document context digest does not match its selection")
	}
	if c.SelectedAt.IsZero() {
		return fmt.Errorf("document context selection timestamp is required")
	}
	if c.State != DocumentContextSelected && c.State != DocumentContextApproved {
		return fmt.Errorf("document context state %q is invalid", c.State)
	}
	if c.Approved() && c.ApprovedAt == nil {
		return fmt.Errorf("an approved document context must record when it was approved")
	}
	return nil
}

// DocumentContextDigest is the canonical identity of a selection.
//
// It covers the staged name and content of every document in position order.
// Reordering the same files is a different selection, because the order is
// what the model was shown.
func DocumentContextDigest(sources []DocumentSource) string {
	var encoded bytes.Buffer
	write := func(value string) {
		_ = binary.Write(&encoded, binary.BigEndian, uint64(len(value)))
		_, _ = encoded.WriteString(value)
	}
	for _, source := range sources {
		write(source.Name)
		write(source.ContentDigest)
	}
	return string(DigestSHA256(encoded.Bytes()))
}

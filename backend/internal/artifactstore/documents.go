package artifactstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// documentRoot is where snapshotted supplied documents live, beside retained
// artifacts and under the same isolated application-state root.
const documentRoot = ".documents"

// SnapshotDocuments copies the owner's selected documents into durable
// storage and returns them as canonical sources.
//
// The snapshot is the point of the exercise. Execution and reasoning both
// read these bytes, never the originals, so an Outcome cannot come to mean
// something different because a file on disk was edited after it was
// approved — and the originals stay untouched, since supplied documents are
// input, never writable output.
func (s *Store) SnapshotDocuments(ctx context.Context, contextID domain.DocumentContextID, paths []string) ([]domain.DocumentSource, error) {
	if contextID.IsZero() {
		return nil, errors.New("document snapshot requires a context id")
	}
	if len(paths) == 0 {
		return nil, errors.New("a document context must select at least one document")
	}
	if len(paths) > domain.MaxSuppliedDocuments {
		return nil, fmt.Errorf("document context selects %d documents; the limit is %d", len(paths), domain.MaxSuppliedDocuments)
	}

	stage := filepath.Join(s.root, documentRoot, ".staging", string(contextID))
	if err := os.MkdirAll(stage, 0o750); err != nil {
		return nil, fmt.Errorf("create document staging: %w", err)
	}
	defer func() { _ = os.RemoveAll(stage) }()

	sources := make([]domain.DocumentSource, 0, len(paths))
	names := map[string]bool{}
	var total int64
	for position, raw := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		source, body, err := s.readDocument(raw, position, total)
		if err != nil {
			return nil, err
		}
		if names[source.Name] {
			return nil, fmt.Errorf("two selected documents would stage as %q", source.Name)
		}
		names[source.Name] = true
		total += source.SizeBytes

		staged := filepath.Join(stage, source.Name)
		if err := os.MkdirAll(filepath.Dir(staged), 0o750); err != nil {
			return nil, err
		}
		if err := writeFileSynced(staged, body, 0o640); err != nil {
			return nil, fmt.Errorf("stage document %q: %w", source.Name, err)
		}
		sources = append(sources, source)
	}

	final := filepath.Join(s.root, documentRoot, string(contextID))
	if err := s.publishDocuments(stage, final, sources); err != nil {
		return nil, err
	}
	return sources, nil
}

// readDocument reads one selected document under the same bounds the
// reasoning packet uses, refusing anything it cannot represent honestly.
func (s *Store) readDocument(raw string, position int, alreadyRead int64) (domain.DocumentSource, []byte, error) {
	if !filepath.IsAbs(raw) {
		return domain.DocumentSource{}, nil, fmt.Errorf("supplied document path must be absolute: %q", raw)
	}
	path := filepath.Clean(raw)
	info, err := os.Lstat(path)
	if err != nil {
		return domain.DocumentSource{}, nil, fmt.Errorf("read supplied document %q: %w", filepath.Base(path), err)
	}
	// A symlink or special file is refused rather than followed: the owner
	// selected a document, and what it points at may not be one.
	if !info.Mode().IsRegular() {
		return domain.DocumentSource{}, nil, fmt.Errorf("supplied document %q is not a regular file", filepath.Base(path))
	}
	if secretPath(path) {
		return domain.DocumentSource{}, nil, fmt.Errorf("supplied document %q is secret-like and cannot be selected", filepath.Base(path))
	}
	remaining := s.maxBytes - alreadyRead
	if remaining <= 0 || info.Size() > remaining {
		return domain.DocumentSource{}, nil, fmt.Errorf("supplied document bounds exceeded at %q", filepath.Base(path))
	}
	body, err := readStable(context.Background(), path, info, remaining)
	if err != nil {
		return domain.DocumentSource{}, nil, fmt.Errorf("supplied document %q: %w", filepath.Base(path), err)
	}
	digest := sha256.Sum256(body)
	return domain.DocumentSource{
		ID:            "doc-" + hex.EncodeToString(digest[:8]),
		Position:      position,
		SourcePath:    path,
		Name:          filepath.Base(path),
		ContentDigest: hex.EncodeToString(digest[:]),
		SizeBytes:     int64(len(body)),
	}, body, nil
}

// publishDocuments atomically moves a verified staging directory into place.
// A repeat selection of identical bytes is idempotent.
func (s *Store) publishDocuments(stage, final string, sources []domain.DocumentSource) error {
	if err := verifyStagedDocuments(stage, sources); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(final), 0o750); err != nil {
		return err
	}
	if _, err := os.Stat(final); err == nil {
		// Already published. The context id is derived from the selection, so
		// the existing directory holds the same bytes.
		return verifyStagedDocuments(final, sources)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := syncTree(stage); err != nil {
		return err
	}
	if err := os.Rename(stage, final); err != nil {
		return fmt.Errorf("publish documents: %w", err)
	}
	return syncDir(filepath.Dir(final))
}

// verifyStagedDocuments re-reads what was written. Bytes observed before a
// write are not evidence that they were stored.
func verifyStagedDocuments(root string, sources []domain.DocumentSource) error {
	for _, source := range sources {
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(source.Name)))
		if err != nil {
			return fmt.Errorf("verify document %q: %w", source.Name, err)
		}
		digest := sha256.Sum256(body)
		if hex.EncodeToString(digest[:]) != source.ContentDigest {
			return fmt.Errorf("document %q was not stored as selected", source.Name)
		}
	}
	return nil
}

// ReadDocument returns one snapshotted document's bytes, refusing content
// that no longer matches what was selected.
func (s *Store) ReadDocument(contextID domain.DocumentContextID, source domain.DocumentSource) ([]byte, error) {
	body, err := os.ReadFile(filepath.Join(s.root, documentRoot, string(contextID), filepath.FromSlash(source.Name)))
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(body)
	if hex.EncodeToString(digest[:]) != source.ContentDigest {
		return nil, fmt.Errorf("snapshotted document %q no longer matches its recorded content", source.Name)
	}
	return body, nil
}

// DocumentHandoff turns an approved selection into the same immutable input
// set a predecessor's retained result produces, so staging documents into a
// workspace reuses one materialization path rather than growing a second.
func (s *Store) DocumentHandoff(contextID domain.DocumentContextID, sources []domain.DocumentSource) (Handoff, error) {
	handoff := Handoff{WorkspaceKind: domain.WorkspaceStagedFolder, Files: make([]HandoffFile, 0, len(sources))}
	for _, source := range sources {
		body, err := s.ReadDocument(contextID, source)
		if err != nil {
			return Handoff{}, err
		}
		handoff.Files = append(handoff.Files, HandoffFile{
			RelativePath: source.Name, ChangeKind: domain.ArtifactAdded,
			// Supplied documents are input, not the Attempt's output. They
			// are staged read-write because a scratch workspace has no other
			// copy, but nothing about them claims the Attempt produced them.
			Content: body, FileMode: 0o640, ContentDigest: source.ContentDigest,
		})
	}
	return handoff, nil
}

// SourcesChangedSince reports selected documents whose current bytes on disk
// differ from what was approved.
//
// It is not a reason to re-read them: the approved snapshot is what runs.
// It is what lets the owner be told their material moved on, so they can
// deliberately re-select rather than discover it in the result.
func (s *Store) SourcesChangedSince(sources []domain.DocumentSource) []string {
	var changed []string
	for _, source := range sources {
		body, err := os.ReadFile(source.SourcePath)
		if err != nil {
			changed = append(changed, source.Name)
			continue
		}
		digest := sha256.Sum256(body)
		if hex.EncodeToString(digest[:]) != source.ContentDigest {
			changed = append(changed, source.Name)
		}
	}
	return changed
}

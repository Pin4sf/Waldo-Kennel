package artifactstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// Handoff is the immutable input set admitted to a successor. InputVersions
// belongs in the successor admission snapshot; callers must not resolve the
// latest upstream receipt again during restart.
type Handoff struct {
	InputVersions []string
	Files         []HandoffFile
}

type HandoffFile struct {
	RelativePath  string
	ChangeKind    domain.ArtifactChangeKind
	Content       []byte
	FileMode      os.FileMode
	ContentDigest string
}

// Compose deterministically combines retained predecessor results. A path may
// be repeated only when its semantic result is byte/mode identical. A
// deletion conflicts with a write, and distinct bases/repositories are never
// guessed into a last-writer-wins result.
func (s *Store) Compose(ctx context.Context, receipts []domain.AttemptReceipt) (Handoff, error) {
	if len(receipts) == 0 {
		return Handoff{}, errors.New("handoff requires at least one retained predecessor")
	}
	seenVersion := map[string]bool{}
	var kind domain.WorkspaceKind
	var base, identity string
	paths := map[string]HandoffFile{}
	for i, receipt := range receipts {
		if err := ctx.Err(); err != nil {
			return Handoff{}, err
		}
		if err := receipt.Validate(); err != nil {
			return Handoff{}, fmt.Errorf("predecessor %s: %w", receipt.AttemptID, err)
		}
		if !receipt.RetentionState.Complete() {
			return Handoff{}, fmt.Errorf("predecessor %s is %s, not retained", receipt.AttemptID, receipt.RetentionState)
		}
		if seenVersion[receipt.ArtifactVersion] {
			return Handoff{}, fmt.Errorf("predecessor artifact %s is repeated", receipt.ArtifactVersion)
		}
		seenVersion[receipt.ArtifactVersion] = true
		if i == 0 {
			kind, base, identity = receipt.WorkspaceKind, receipt.BaseRevision, receipt.RepositoryIdentity
		} else if kind != receipt.WorkspaceKind || base != receipt.BaseRevision || identity != receipt.RepositoryIdentity {
			return Handoff{}, errors.New("predecessor artifacts have incompatible workspace bases")
		}
		for _, file := range receipt.Files {
			entry := HandoffFile{RelativePath: file.RelativePath, ChangeKind: file.ChangeKind, ContentDigest: file.ContentDigest}
			if file.ChangeKind != domain.ArtifactDeleted {
				content, mode, err := s.Read(ctx, receipt, file)
				if err != nil {
					return Handoff{}, fmt.Errorf("read predecessor %s/%s: %w", receipt.AttemptID, file.RelativePath, err)
				}
				entry.Content, entry.FileMode = content, mode
			}
			if previous, ok := paths[file.RelativePath]; ok {
				if previous.ChangeKind != entry.ChangeKind || previous.ContentDigest != entry.ContentDigest || previous.FileMode.Perm() != entry.FileMode.Perm() {
					return Handoff{}, fmt.Errorf("predecessors conflict on %q", file.RelativePath)
				}
				continue
			}
			paths[file.RelativePath] = entry
		}
	}
	keys := make([]string, 0, len(paths))
	for path := range paths {
		keys = append(keys, path)
	}
	sort.Strings(keys)
	out := Handoff{InputVersions: make([]string, 0, len(receipts)), Files: make([]HandoffFile, 0, len(keys))}
	for _, receipt := range receipts {
		out.InputVersions = append(out.InputVersions, receipt.ArtifactVersion)
	}
	for _, path := range keys {
		out.Files = append(out.Files, paths[path])
	}
	return out, nil
}

// Apply materializes a handoff into a new, empty destination. It refuses an
// existing non-empty path so owner or scheduler code cannot overwrite a
// workspace that is not attributable to this successor.
func (s *Store) Apply(ctx context.Context, handoff Handoff, destination string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !filepath.IsAbs(destination) {
		return errors.New("handoff destination must be absolute")
	}
	if info, err := os.Stat(destination); err == nil {
		if !info.IsDir() {
			return errors.New("handoff destination is not a directory")
		}
		entries, readErr := os.ReadDir(destination)
		if readErr != nil {
			return readErr
		}
		if len(entries) != 0 {
			return errors.New("handoff destination is not empty")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	} else if err := os.MkdirAll(destination, 0o750); err != nil {
		return err
	}
	for _, file := range handoff.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		full, err := confinedPath(destination, file.RelativePath)
		if err != nil {
			return err
		}
		if file.ChangeKind == domain.ArtifactDeleted {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(full, file.Content, file.FileMode.Perm()); err != nil {
			return err
		}
		body, err := os.ReadFile(full)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(body)
		if hex.EncodeToString(digest[:]) != file.ContentDigest {
			return errors.New("materialized handoff digest mismatch")
		}
		info, err := os.Lstat(full)
		if err != nil || info.Mode().Perm() != file.FileMode.Perm() {
			return errors.New("materialized handoff mode mismatch")
		}
	}
	return nil
}

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
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// Handoff is the immutable input set admitted to a successor. InputVersions
// belongs in the successor admission snapshot; callers must not resolve the
// latest upstream receipt again during restart.
type Handoff struct {
	InputVersions []string
	// WorkspaceKind, BaseRevision and RepositoryIdentity are the custody shape
	// every predecessor agreed on. The successor's own workspace must match
	// before these changes may be written into it: the same diff applied to a
	// different base is a different result.
	WorkspaceKind      domain.WorkspaceKind
	BaseRevision       string
	RepositoryIdentity string
	Files              []HandoffFile
}

// HandoffFile is one deterministic, verified predecessor change.
type HandoffFile struct {
	RelativePath  string
	ChangeKind    domain.ArtifactChangeKind
	Content       []byte
	FileMode      os.FileMode
	ContentDigest string
}

// ErrHandoffBaseMismatch reports a successor workspace whose base is not the
// base the predecessors' changes were produced against.
var ErrHandoffBaseMismatch = errors.New("handoff base does not match the successor workspace")

// Compose deterministically combines retained predecessor results. A path may
// be repeated only when its semantic result is byte/mode identical. A
// deletion conflicts with a write, and distinct bases/repositories are never
// guessed into a last-writer-wins result.
func (s *Store) Compose(ctx context.Context, receipts []domain.AttemptReceipt) (Handoff, error) {
	if len(receipts) == 0 {
		return Handoff{}, errors.New("handoff requires at least one retained predecessor")
	}
	// Deduplicate on the producing Attempt, not the artifact version. The
	// version is a digest of the manifest, so two different WorkUnits that
	// legitimately produce identical output share one — that is shared
	// ancestry, which must join, not a predecessor listed twice.
	seenProducer := map[domain.AttemptID]bool{}
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
		if seenProducer[receipt.AttemptID] {
			return Handoff{}, fmt.Errorf("predecessor attempt %s is repeated", receipt.AttemptID)
		}
		seenProducer[receipt.AttemptID] = true
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
	out := Handoff{
		InputVersions: make([]string, 0, len(receipts)), Files: make([]HandoffFile, 0, len(keys)),
		WorkspaceKind: kind, BaseRevision: base, RepositoryIdentity: identity,
	}
	for _, receipt := range receipts {
		out.InputVersions = append(out.InputVersions, receipt.ArtifactVersion)
	}
	for _, path := range keys {
		out.Files = append(out.Files, paths[path])
	}
	return out, nil
}

// Materialize writes a composed handoff into a successor's already-provisioned
// workspace.
//
// The workspace normally already holds the approved base checkout, and that is
// the point: the predecessors' manifest describes only what *changed*, so
// every unchanged base file has to arrive by already being there. Writing the
// changes on top is what makes the successor's tree equal to what the
// predecessors actually left behind, deletions included.
//
// baseRevision is the successor workspace's own resolved base. It must equal
// the base the predecessors worked from, because the same set of changes
// applied to a different base is a different result and nobody would be told.
func (s *Store) Materialize(ctx context.Context, handoff Handoff, destination, baseRevision string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if handoff.WorkspaceKind == domain.WorkspaceGitWorktree {
		if strings.TrimSpace(handoff.BaseRevision) == "" || strings.TrimSpace(baseRevision) == "" {
			return fmt.Errorf("%w: base revision is unknown", ErrHandoffBaseMismatch)
		}
		if handoff.BaseRevision != baseRevision {
			return fmt.Errorf("%w: predecessors used %s, successor is at %s", ErrHandoffBaseMismatch, handoff.BaseRevision, baseRevision)
		}
	}
	root, err := physicalDirectory(destination)
	if err != nil {
		return fmt.Errorf("successor workspace: %w", err)
	}
	for _, file := range handoff.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		full, err := confinedPhysicalPath(root, file.RelativePath)
		if err != nil {
			return err
		}
		if file.ChangeKind == domain.ArtifactDeleted {
			if err := removeMaterialized(full); err != nil {
				return fmt.Errorf("apply deletion %q: %w", file.RelativePath, err)
			}
			continue
		}
		if err := writeMaterialized(full, file); err != nil {
			return fmt.Errorf("apply %q: %w", file.RelativePath, err)
		}
	}
	return verifyMaterialized(root, handoff)
}

// removeMaterialized applies an upstream deletion. A downstream unit that
// still sees a file its predecessor removed has not received that work.
//
// Only a regular file is removed. Finding a directory or a symlink where a
// deleted regular file should be means the workspace is not what the manifest
// describes, and quietly deleting it could destroy unrelated content.
func removeMaterialized(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("existing path is not a regular file")
	}
	return os.Remove(path)
}

func writeMaterialized(path string, file HandoffFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	// Replace rather than truncate in place: the base checkout may hold this
	// path as a symlink or with different permissions, and opening it for
	// write would follow the link out of custody.
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return errors.New("existing path is not a regular file")
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writeFileSynced(path, file.Content, file.FileMode.Perm())
}

// verifyMaterialized re-reads what was written. Bytes observed before a write
// are not evidence that the successor holds them, and a provider must never be
// launched against a workspace nobody checked.
func verifyMaterialized(root string, handoff Handoff) error {
	for _, file := range handoff.Files {
		full, err := confinedPhysicalPath(root, file.RelativePath)
		if err != nil {
			return err
		}
		if file.ChangeKind == domain.ArtifactDeleted {
			if _, err := os.Lstat(full); !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("materialized handoff still holds deleted path %q", file.RelativePath)
			}
			continue
		}
		body, err := os.ReadFile(full)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(body)
		if hex.EncodeToString(digest[:]) != file.ContentDigest {
			return fmt.Errorf("materialized handoff digest mismatch for %q", file.RelativePath)
		}
		info, err := os.Lstat(full)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Mode().Perm() != file.FileMode.Perm() {
			return fmt.Errorf("materialized handoff mode mismatch for %q", file.RelativePath)
		}
	}
	return nil
}

// confinedPhysicalPath resolves a manifest path inside root and refuses an
// escape through an existing symlink as well as through the name itself.
//
// A string check alone is not enough here: unlike a capture, materialization
// writes into a checkout that already has content, and a base repository is
// free to contain a symlink pointing anywhere on the machine.
func confinedPhysicalPath(root, name string) (string, error) {
	full, err := confinedPath(root, name)
	if err != nil {
		return "", err
	}
	// Walk down from root so the deepest existing ancestor is resolved
	// physically; anything below it does not exist yet and cannot be a link.
	resolved := root
	parts := strings.Split(filepath.ToSlash(name), "/")
	for i, part := range parts {
		candidate := filepath.Join(resolved, part)
		info, statErr := os.Lstat(candidate)
		if errors.Is(statErr, os.ErrNotExist) {
			return full, nil
		}
		if statErr != nil {
			return "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			// The final component being a symlink is handled by the writer,
			// which replaces it. An intermediate symlink is a directory the
			// manifest does not own and is refused outright.
			if i == len(parts)-1 {
				return full, nil
			}
			return "", fmt.Errorf("path %q crosses a symlink and leaves custody", name)
		}
		physical, evalErr := filepath.EvalSymlinks(candidate)
		if evalErr != nil {
			return "", evalErr
		}
		rel, relErr := filepath.Rel(root, filepath.Clean(physical))
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("path %q resolves outside the successor workspace", name)
		}
		resolved = candidate
	}
	return full, nil
}

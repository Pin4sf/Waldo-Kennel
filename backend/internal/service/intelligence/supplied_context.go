package intelligence

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

const suppliedContextMaxFiles = 32
const suppliedContextMaxBytes = 96 * 1024
const suppliedContextMaxFile = 12 * 1024

// BuildSuppliedDocumentContext reads an explicit, owner-selected set of local
// text documents. It is deliberately narrower than a general ingestion
// system: no PDF/OCR, indexing, network fetch, Git initialization, or folder
// traversal is implied by this API.
func BuildSuppliedDocumentContext(ctx context.Context, paths []string) (ports.RepositoryContextSnapshot, error) {
	if len(paths) == 0 {
		return ports.RepositoryContextSnapshot{}, errors.New("at least one supplied document is required")
	}
	if len(paths) > suppliedContextMaxFiles {
		return ports.RepositoryContextSnapshot{}, fmt.Errorf("supplied document limit exceeded: %d files", len(paths))
	}
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	snapshot := ports.RepositoryContextSnapshot{Root: "supplied", Files: make([]ports.RepositoryContextFile, 0, len(sorted))}
	seen := map[string]bool{}
	total := 0
	for _, raw := range sorted {
		if err := ctx.Err(); err != nil {
			return ports.RepositoryContextSnapshot{}, err
		}
		if !filepath.IsAbs(raw) {
			return ports.RepositoryContextSnapshot{}, fmt.Errorf("supplied document path must be absolute: %q", raw)
		}
		path := filepath.Clean(raw)
		if seen[path] {
			return ports.RepositoryContextSnapshot{}, fmt.Errorf("supplied document repeated: %q", path)
		}
		seen[path] = true
		if !supportedDocument(path) {
			return ports.RepositoryContextSnapshot{}, fmt.Errorf("unsupported supplied document format %q; supported formats are plain text, Markdown, CSV, JSON, and YAML", filepath.Ext(path))
		}
		info, err := os.Lstat(path)
		if err != nil {
			return ports.RepositoryContextSnapshot{}, fmt.Errorf("read supplied document %q: %w", filepath.Base(path), err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return ports.RepositoryContextSnapshot{}, fmt.Errorf("supplied document %q is not a regular file", filepath.Base(path))
		}
		if sensitiveContextFile(filepath.Base(path)) || sensitiveSuppliedDocument(path) {
			return ports.RepositoryContextSnapshot{}, fmt.Errorf("supplied document %q is secret-like and cannot be selected", filepath.Base(path))
		}
		if info.Size() > suppliedContextMaxFile || total+int(info.Size()) > suppliedContextMaxBytes {
			return ports.RepositoryContextSnapshot{}, fmt.Errorf("supplied document bounds exceeded at %q", filepath.Base(path))
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return ports.RepositoryContextSnapshot{}, err
		}
		after, err := os.Lstat(path)
		if err != nil || after.Size() != info.Size() || after.ModTime() != info.ModTime() || after.Mode().Perm() != info.Mode().Perm() {
			return ports.RepositoryContextSnapshot{}, fmt.Errorf("supplied document %q changed during read", filepath.Base(path))
		}
		if strings.IndexByte(string(body), 0) >= 0 {
			return ports.RepositoryContextSnapshot{}, fmt.Errorf("supplied document %q is binary", filepath.Base(path))
		}
		total += len(body)
		snapshot.Files = append(snapshot.Files, ports.RepositoryContextFile{Path: filepath.Base(path), Content: string(body)})
	}
	snapshot.Dirty = false
	return finalizeContext(snapshot), nil
}

func supportedDocument(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".md", ".markdown", ".csv", ".json", ".yaml", ".yml", ".rst":
		return true
	default:
		return false
	}
}

func sensitiveSuppliedDocument(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(base, "secret") || strings.Contains(base, "credential") || strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key")
}

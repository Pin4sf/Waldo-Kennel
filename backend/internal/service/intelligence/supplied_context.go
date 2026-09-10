package intelligence

import (
	"context"
	"errors"
	"fmt"
	"io"
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
		remaining := suppliedContextMaxBytes - total
		if remaining > suppliedContextMaxFile {
			remaining = suppliedContextMaxFile
		}
		if info.Size() > int64(remaining) {
			return ports.RepositoryContextSnapshot{}, fmt.Errorf("supplied document bounds exceeded at %q", filepath.Base(path))
		}
		body, err := readBoundedDocument(ctx, path, info, remaining)
		if err != nil {
			return ports.RepositoryContextSnapshot{}, fmt.Errorf("supplied document %q: %w", filepath.Base(path), err)
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

// afterLstat is a test seam for the window between deciding a path is a safe,
// in-bounds regular file and actually opening it. It is nil in production.
var afterLstat func(path string)

// readBoundedDocument reads at most limit bytes from the file identified by
// info, and refuses anything that is no longer that exact file.
//
// Checking the size and then calling os.ReadFile is not a bound: the path can
// grow, or be replaced by a symlink to something much larger, between the two
// calls, and the read would allocate whatever is there now. The bound has to be
// applied at the operation that reads, and identity has to be checked against
// the open descriptor rather than the path.
func readBoundedDocument(ctx context.Context, path string, info os.FileInfo, limit int) ([]byte, error) {
	if hook := afterLstat; hook != nil {
		hook(path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	// os.SameFile compares device and inode, so a path swapped for a symlink to
	// another file is caught here even though Open followed it.
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, errors.New("changed between selection and read")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// limit+1 so an over-bound file is detected without ever buffering it.
	body, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > limit {
		return nil, fmt.Errorf("exceeds the %d byte bound", limit)
	}
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, after) || after.Size() != info.Size() || after.ModTime() != info.ModTime() || after.Mode().Perm() != info.Mode().Perm() {
		return nil, errors.New("changed during read")
	}
	return body, nil
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

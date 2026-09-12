package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

const (
	contextMaxFile    = 12 * 1024
	contextMaxRuntime = 2 * time.Second
)

// RepositoryContextLimits bounds one BuildRepositoryContext call. In every
// field, zero or negative means uncapped — this package interprets that
// directly, so a caller (including a zero-value struct) never silently
// collapses to "allow nothing".
//
// MaxFiles and MaxBytes cap content exposure (the real token-management
// lever); MaxVisited only bounds directory-walk discovery. contextMaxFile
// (per-file truncation) and contextMaxRuntime (wall-clock backstop) remain
// fixed: they are safety bounds, not a token budget an owner would tune.
type RepositoryContextLimits struct {
	MaxFiles   int
	MaxBytes   int
	MaxVisited int
}

// DefaultRepositoryContextLimits are Kennel's built-in bounds, used whenever
// the owner has not configured an override. A too-low MaxVisited previously
// gave up directory-walk discovery before it ever reached the files that
// mattered on an ordinary real repository — observed directly against a
// ~3,500-entry repository during the launch-stabilization end-to-end
// exercise — which is why it is now a configurable, owner-raisable value
// rather than a fixed constant.
var DefaultRepositoryContextLimits = RepositoryContextLimits{
	MaxFiles:   32,
	MaxBytes:   96 * 1024,
	MaxVisited: 20000,
}

// RepositoryContextLimitsSource resolves the owner's configured bounds
// (already normalized: unset -> DefaultRepositoryContextLimits, uncapped ->
// zero). It is intentionally satisfied structurally: the
// settings service implements this shape without either package importing
// the other.
type RepositoryContextLimitsSource interface {
	RepositoryContextLimits(ctx context.Context) (maxFiles, maxBytes, maxVisited int, err error)
}

// atCap reports whether count has reached limit. limit <= 0 means uncapped:
// no count ever reaches it.
func atCap(count, limit int) bool {
	return limit > 0 && count >= limit
}

// overCap reports whether count has exceeded limit. limit <= 0 means
// uncapped: no count ever exceeds it.
func overCap(count, limit int) bool {
	return limit > 0 && count > limit
}

var errContextEntryLimit = errors.New("repository inspection reached its entry limit; context is partial")

// ProjectSource is the read-only registry seam needed to locate a registered
// repository. It deliberately exposes no writer or command-execution port.
type ProjectSource interface {
	GetProject(context.Context, string) (domain.ProjectRecord, bool, error)
}

type briefSource interface {
	GetCurrentProjectBriefRevision(context.Context, domain.ProjectID) (domain.ProjectBriefRevision, bool, error)
}

// BuildRepositoryContext inspects only bounded, text-oriented repository facts.
// It never runs package scripts, follows symlinks, or reads ignored files.
func BuildRepositoryContext(ctx context.Context, project domain.ProjectRecord, brief briefSource, limits RepositoryContextLimits) (ports.RepositoryContextSnapshot, error) {
	snapshot := ports.RepositoryContextSnapshot{ProjectID: domain.ProjectID(project.ID), Root: filepath.Clean(project.Path)}
	if brief != nil {
		current, ok, err := brief.GetCurrentProjectBriefRevision(ctx, snapshot.ProjectID)
		if err != nil {
			return snapshot, fmt.Errorf("load Project Brief: %w", err)
		}
		if ok {
			snapshot.ProjectBrief = &current
		}
	}
	rootInfo, err := os.Stat(snapshot.Root)
	if err != nil || rootInfo == nil || !rootInfo.IsDir() {
		snapshot.UnavailableReason = "registered repository root is unavailable for bounded inspection"
		return finalizeContext(snapshot), nil
	}

	inspectCtx, cancel := context.WithTimeout(ctx, contextMaxRuntime)
	defer cancel()
	snapshot.Revision = readGitFact(inspectCtx, snapshot.Root, "rev-parse", "HEAD")
	status := readGitFact(inspectCtx, snapshot.Root, "status", "--porcelain", "--untracked-files=normal")
	snapshot.Dirty = strings.TrimSpace(status) != ""
	if snapshot.Revision == "" {
		snapshot.UnavailableReason = "repository revision could not be inspected"
	}

	files, instructions, checks, collectErr := boundedFiles(inspectCtx, snapshot.Root, limits)
	snapshot.Files, snapshot.Instructions, snapshot.CheckCommands = files, instructions, checks
	if collectErr != nil {
		if errors.Is(collectErr, errContextEntryLimit) {
			snapshot.UnavailableReason = errContextEntryLimit.Error()
		} else if errors.Is(collectErr, context.Canceled) || errors.Is(collectErr, context.DeadlineExceeded) {
			snapshot.UnavailableReason = "repository inspection stopped before its bounded context was complete"
		} else if snapshot.UnavailableReason == "" {
			snapshot.UnavailableReason = "repository files could not be inspected safely"
		}
	}
	return finalizeContext(snapshot), nil
}

func readGitFact(ctx context.Context, root string, args ...string) string {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	output, err := command.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func boundedFiles(ctx context.Context, root string, limits RepositoryContextLimits) (files, instructions []ports.RepositoryContextFile, checks []string, collectErr error) {
	// candidateLimit bounds path discovery, scaled off MaxFiles (uncapped when
	// MaxFiles is): it only needs to comfortably outnumber MaxFiles so the
	// later content pass has enough priority candidates to choose from.
	candidateLimit := 0
	if limits.MaxFiles > 0 {
		candidateLimit = limits.MaxFiles * 3
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	priority := map[string]bool{
		"AGENTS.md": true, "README": true, "README.md": true, "README.txt": true,
		"package.json": true, "go.mod": true, "Makefile": true,
	}
	var candidates []string
	// Main project context must not lose its budget to a large source tree or
	// a directory of workflow files encountered first in lexical traversal.
	first := []string{"AGENTS.md", "README.md", "README", "README.txt", "package.json", "go.mod", "Makefile", "docs/STATUS.md", "docs/architecture.md"}
	seen := map[string]bool{}
	rank := map[string]int{}
	for index, rel := range first {
		rank[rel] = index + 1
		if !regularContextPath(root, rel) {
			continue
		}
		ignored, err := ignoredByGit(ctx, root, rel)
		if err != nil {
			return nil, nil, nil, err
		}
		if !ignored {
			candidates = append(candidates, rel)
			seen[rel] = true
		}
	}
	visited := 0
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		visited++
		if overCap(visited, limits.MaxVisited) {
			// Reaching a bounded discovery limit does not invalidate files found
			// before it. Stop discovery and inspect those candidates normally.
			collectErr = errContextEntryLimit
			return filepath.SkipAll
		}
		if path != root && entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return filepath.SkipDir
		}
		if rel == "." {
			return nil
		}
		if entry.IsDir() {
			if excludedContextDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if seen[rel] || sensitiveContextFile(rel) || (!priorityContextFile(rel, priority) && !shallowTextCandidate(rel)) {
			return nil
		}
		ignored, err := ignoredByGit(ctx, root, rel)
		if err != nil {
			return err
		}
		if ignored {
			return nil
		}
		candidates = append(candidates, rel)
		if atCap(len(candidates), candidateLimit) {
			return filepath.SkipAll
		}
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, filepath.SkipAll) {
		return nil, nil, nil, walkErr
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := rank[candidates[i]], rank[candidates[j]]
		if a != b {
			if a == 0 {
				return false
			}
			if b == 0 {
				return true
			}
			return a < b
		}
		return candidates[i] < candidates[j]
	})
	seenBytes := 0
	for _, rel := range candidates {
		if err := ctx.Err(); err != nil {
			return files, instructions, checks, err
		}
		if atCap(len(files)+len(instructions), limits.MaxFiles) || atCap(seenBytes, limits.MaxBytes) {
			break
		}
		path := filepath.Join(root, rel)
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		content, err := io.ReadAll(io.LimitReader(file, contextMaxFile+1))
		_ = file.Close()
		if err != nil || strings.IndexByte(string(content), 0) >= 0 {
			continue
		}
		truncated := len(content) > contextMaxFile
		if truncated {
			content = content[:contextMaxFile]
		}
		if limits.MaxBytes > 0 {
			remaining := limits.MaxBytes - seenBytes
			if len(content) > remaining {
				content = content[:remaining]
				truncated = true
			}
		}
		item := ports.RepositoryContextFile{Path: filepath.ToSlash(rel), Content: string(content), Truncated: truncated}
		seenBytes += len(content)
		if filepath.Base(rel) == "AGENTS.md" || strings.HasPrefix(filepath.ToSlash(rel), "docs/AGENTS") {
			instructions = append(instructions, item)
		} else {
			files = append(files, item)
		}
		if filepath.Base(rel) == "package.json" {
			checks = append(checks, packageCheckCommands(content)...)
		}
	}
	sort.Strings(checks)
	return files, instructions, uniqueStrings(checks), collectErr
}

// Explicit priority paths receive the same no-symlink rule as discovery.
func regularContextPath(root, rel string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	path := root
	for index, part := range parts {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return false
		}
		if index == len(parts)-1 {
			return info.Mode().IsRegular()
		}
		if !info.IsDir() {
			return false
		}
	}
	return false
}

func excludedContextDir(rel string) bool {
	for _, segment := range strings.Split(filepath.ToSlash(rel), "/") {
		switch segment {
		case ".git", ".kennel", "node_modules", "vendor", "dist", "build", "coverage", ".next", "tmp", "target":
			return true
		default:
			if strings.HasPrefix(segment, ".") && segment != ".github" {
				return true
			}
		}
	}
	return false
}

func sensitiveContextFile(rel string) bool {
	path := filepath.ToSlash(rel)
	for _, segment := range strings.Split(path, "/") {
		lower := strings.ToLower(segment)
		switch lower {
		case ".aws", ".ssh", ".gnupg", "secrets", "secret", "credentials", "credential", "private", ".npmrc", ".pypirc", ".netrc", ".git-credentials":
			return true
		}
	}
	base := strings.ToLower(filepath.Base(rel))
	if strings.HasPrefix(base, ".env") {
		return true
	}
	switch base {
	case "credentials.json", "secrets.json", "service-account.json", "dockerconfigjson":
		return true
	}
	for _, suffix := range []string{".pem", ".key", ".p12", ".pfx", ".pkcs12", ".jks", ".kdbx", ".age"} {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	return base == "id_rsa" || strings.HasPrefix(base, "id_rsa.") || base == "id_ed25519" || strings.HasPrefix(base, "id_ed25519.") || base == "known_hosts"
}

func priorityContextFile(rel string, priority map[string]bool) bool {
	base := filepath.Base(rel)
	return priority[base] || strings.HasPrefix(filepath.ToSlash(rel), ".github/")
}

func shallowTextCandidate(rel string) bool {
	slash := filepath.ToSlash(rel)
	return !strings.Contains(slash, "/") || strings.Count(slash, "/") == 1 && strings.HasPrefix(slash, "docs/")
}

func ignoredByGit(ctx context.Context, root, rel string) (bool, error) {
	command := exec.CommandContext(ctx, "git", "-C", root, "check-ignore", "--quiet", "--", rel)
	err := command.Run()
	if err == nil {
		return true, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return false, ctxErr
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("git check-ignore %q: %w", rel, err)
}

func packageCheckCommands(content []byte) []string {
	var parsed struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(content, &parsed) != nil {
		return nil
	}
	var checks []string
	for name, script := range parsed.Scripts {
		lower := strings.ToLower(name)
		if strings.Contains(lower, "test") || strings.Contains(lower, "lint") || strings.Contains(lower, "check") || strings.Contains(lower, "type") {
			checks = append(checks, "npm run "+name+" -> "+strings.TrimSpace(script))
		}
	}
	return checks
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func finalizeContext(snapshot ports.RepositoryContextSnapshot) ports.RepositoryContextSnapshot {
	withoutDigest := snapshot
	withoutDigest.Digest = ""
	encoded, err := json.Marshal(withoutDigest)
	if err == nil {
		snapshot.Digest = domain.DigestSHA256(encoded)
	}
	return snapshot
}

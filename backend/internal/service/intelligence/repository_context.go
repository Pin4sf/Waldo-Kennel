package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	contextMaxFiles      = 32
	contextMaxBytes      = 96 * 1024
	contextMaxFile       = 12 * 1024
	contextMaxRuntime    = 2 * time.Second
	contextMaxVisited    = 512
	contextMaxCandidates = contextMaxFiles * 3
)

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
func BuildRepositoryContext(ctx context.Context, project domain.ProjectRecord, brief briefSource) (ports.RepositoryContextSnapshot, error) {
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

	files, instructions, checks, collectErr := boundedFiles(inspectCtx, snapshot.Root)
	snapshot.Files, snapshot.Instructions, snapshot.CheckCommands = files, instructions, checks
	if collectErr != nil {
		if errors.Is(collectErr, context.Canceled) || errors.Is(collectErr, context.DeadlineExceeded) {
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

func boundedFiles(ctx context.Context, root string) (files, instructions []ports.RepositoryContextFile, checks []string, collectErr error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	priority := map[string]bool{
		"AGENTS.md": true, "README": true, "README.md": true, "README.txt": true,
		"package.json": true, "go.mod": true, "Makefile": true,
	}
	var candidates []string
	visited := 0
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		visited++
		if visited > contextMaxVisited {
			return fmt.Errorf("repository inspection exceeded %d filesystem entries", contextMaxVisited)
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
		if sensitiveContextFile(rel) || (!priorityContextFile(rel, priority) && !shallowTextCandidate(rel)) {
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
		if len(candidates) >= contextMaxCandidates {
			return filepath.SkipAll
		}
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, filepath.SkipAll) {
		return nil, nil, nil, walkErr
	}
	sort.Strings(candidates)
	seenBytes := 0
	for _, rel := range candidates {
		if err := ctx.Err(); err != nil {
			return files, instructions, checks, err
		}
		if len(files)+len(instructions) >= contextMaxFiles || seenBytes >= contextMaxBytes {
			break
		}
		path := filepath.Join(root, rel)
		info, err := os.Stat(path)
		if err != nil || info.Size() > contextMaxFile {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil || strings.IndexByte(string(content), 0) >= 0 {
			continue
		}
		remaining := contextMaxBytes - seenBytes
		if len(content) > remaining {
			content = content[:remaining]
		}
		item := ports.RepositoryContextFile{Path: filepath.ToSlash(rel), Content: string(content)}
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
	return files, instructions, uniqueStrings(checks), nil
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

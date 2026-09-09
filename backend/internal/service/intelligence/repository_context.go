package intelligence

import (
	"context"
	"encoding/json"
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
	contextMaxFiles   = 32
	contextMaxBytes   = 96 * 1024
	contextMaxFile    = 12 * 1024
	contextMaxRuntime = 2 * time.Second
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

	files, instructions, checks := boundedFiles(inspectCtx, snapshot.Root)
	snapshot.Files, snapshot.Instructions, snapshot.CheckCommands = files, instructions, checks
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

func boundedFiles(ctx context.Context, root string) (files, instructions []ports.RepositoryContextFile, checks []string) {
	priority := map[string]bool{
		"AGENTS.md": true, "README": true, "README.md": true, "README.txt": true,
		"package.json": true, "go.mod": true, "Makefile": true,
	}
	var candidates []string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return filepath.SkipDir
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
		if len(candidates) >= contextMaxFiles*3 || !priorityContextFile(rel, priority) && !shallowTextCandidate(rel) {
			return nil
		}
		if ignoredByGit(ctx, root, rel) {
			return nil
		}
		candidates = append(candidates, rel)
		return nil
	})
	sort.Strings(candidates)
	seenBytes := 0
	for _, rel := range candidates {
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
	return files, instructions, uniqueStrings(checks)
}

func excludedContextDir(rel string) bool {
	first := strings.Split(filepath.ToSlash(rel), "/")[0]
	switch first {
	case ".git", ".kennel", "node_modules", "vendor", "dist", "build", "coverage", ".next", "tmp", "target":
		return true
	default:
		return strings.HasPrefix(first, ".") && first != ".github"
	}
}

func priorityContextFile(rel string, priority map[string]bool) bool {
	base := filepath.Base(rel)
	return priority[base] || strings.HasPrefix(filepath.ToSlash(rel), ".github/")
}

func shallowTextCandidate(rel string) bool {
	slash := filepath.ToSlash(rel)
	return !strings.Contains(slash, "/") || strings.Count(slash, "/") == 1 && strings.HasPrefix(slash, "docs/")
}

func ignoredByGit(ctx context.Context, root, rel string) bool {
	command := exec.CommandContext(ctx, "git", "-C", root, "check-ignore", "--quiet", "--", rel)
	return command.Run() == nil
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

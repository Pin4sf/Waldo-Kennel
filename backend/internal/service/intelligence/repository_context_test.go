package intelligence

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func TestBuildRepositoryContextBoundsFilesAndExcludesIgnoredSymlinkedSecrets(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored.txt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"test:distinctive":"go test ./internal/feature","lint":"golangci-lint run"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("repository facts"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=unignored-secret-canary"), 0o600); err != nil {
		t.Fatal(err)
	}
	canary := filepath.Join(root, "ignored.txt")
	if err := os.WriteFile(canary, []byte("ignored-secret-canary"), 0o600); err != nil {
		t.Fatal(err)
	}
	escape := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(escape, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(escape, filepath.Join(root, "outside-link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := runGit(root, "init", "-q"); err != nil {
		t.Fatal(err)
	}

	snapshot, err := BuildRepositoryContext(context.Background(), domain.ProjectRecord{ID: "project-context", Path: root}, nil)
	if err != nil {
		t.Fatalf("BuildRepositoryContext() error = %v", err)
	}
	encoded := snapshot.Root + snapshot.Revision + snapshot.UnavailableReason
	for _, file := range append(snapshot.Instructions, snapshot.Files...) {
		encoded += file.Path + file.Content
	}
	if strings.Contains(encoded, "ignored-secret-canary") || strings.Contains(encoded, "unignored-secret-canary") || strings.Contains(encoded, "outside") {
		t.Fatalf("context included excluded content: %s", encoded)
	}
	if len(snapshot.CheckCommands) != 2 || !strings.Contains(strings.Join(snapshot.CheckCommands, "\n"), "test:distinctive") {
		t.Fatalf("check commands = %#v", snapshot.CheckCommands)
	}
	if snapshot.Digest == "" {
		t.Fatal("context digest is empty")
	}
}

func TestBuildRepositoryContextStopsWhenCancelled(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("must-not-be-collected"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	snapshot, err := BuildRepositoryContext(ctx, domain.ProjectRecord{ID: "cancelled", Path: root}, nil)
	if err != nil {
		t.Fatalf("BuildRepositoryContext() error = %v", err)
	}
	if len(snapshot.Files) != 0 || len(snapshot.Instructions) != 0 {
		t.Fatalf("cancelled inspection collected files: %#v %#v", snapshot.Files, snapshot.Instructions)
	}
}

func TestExcludedContextDirChecksEveryPathSegment(t *testing.T) {
	for _, rel := range []string{"src/node_modules", "src/.git", "src/vendor", "docs/build/output"} {
		if !excludedContextDir(rel) {
			t.Errorf("excludedContextDir(%q) = false, want true", rel)
		}
	}
}

func TestIgnoredByGitDistinguishesNotIgnoredFromGitFailure(t *testing.T) {
	root := t.TempDir()
	if ignored, err := ignoredByGit(context.Background(), root, "README.md"); err == nil || ignored {
		t.Fatalf("ignoredByGit() = ignored=%v, err=%v; want a non-ignored Git failure", ignored, err)
	}

	if err := runGit(root, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	if ignored, err := ignoredByGit(context.Background(), root, "README.md"); err != nil || ignored {
		t.Fatalf("ignoredByGit() = ignored=%v, err=%v; want clean not-ignored result", ignored, err)
	}
}

func runGit(root string, args ...string) error {
	// Keep the helper local to the test so the production context builder remains
	// the only filesystem-inspection implementation.
	command := append([]string{"-C", root}, args...)
	return exec.Command("git", command...).Run()
}

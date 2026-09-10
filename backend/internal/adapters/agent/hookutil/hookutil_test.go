package hookutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnsureWorkspaceGitignoreWritesSelfIgnoringFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".codex")
	if err := EnsureWorkspaceGitignore(dir, "hooks.json", "config.toml"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, GitignoreSentinel) {
		t.Fatalf("content missing sentinel: %q", content)
	}
	// Entries are anchored so only Kennel's files in THIS directory are ignored —
	// an agent's own files (even in the same dir) must keep counting as dirt.
	for _, want := range []string{"/.gitignore\n", "/hooks.json\n", "/config.toml\n"} {
		if !strings.Contains(content, want) {
			t.Errorf("content missing entry %q: %q", want, content)
		}
	}
}

func TestEnsureWorkspaceGitignoreIsIdempotent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".codex")
	if err := EnsureWorkspaceGitignore(dir, "hooks.json"); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	first, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := EnsureWorkspaceGitignore(dir, "hooks.json"); err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	second, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("rewrite changed content:\nfirst:  %q\nsecond: %q", first, second)
	}
}

func TestFileExistsIgnoresExecBit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-binary")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	// FileExists must return true for a non-executable regular file — Cline
	// uses it as a do-not-clobber guard for existing user hook files.
	if !FileExists(path) {
		t.Fatalf("FileExists returned false for non-executable regular file")
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if !FileExists(path) {
		t.Fatalf("FileExists returned false for executable regular file")
	}
}

func TestIsExecutableFileChecksExecBit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("exec-bit check is skipped on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-binary")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	// IsExecutableFile must return false for a non-executable regular file.
	if IsExecutableFile(path) {
		t.Fatalf("IsExecutableFile returned true for non-executable file")
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if !IsExecutableFile(path) {
		t.Fatalf("IsExecutableFile returned false for executable file")
	}
}

func TestEnsureWorkspaceGitignoreLeavesForeignFileUntouched(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".codex")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	foreign := "# user rules\n*.log\n"
	path := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(path, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := EnsureWorkspaceGitignore(dir, "hooks.json"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != foreign {
		t.Fatalf("foreign .gitignore was modified: %q", data)
	}
}

// The sentinel was renamed away from the donor product's name. A workspace that
// still carries the old one is Kennel's own file and must keep being rewritten;
// treating it as somebody else's .gitignore would leave that worktree
// permanently undeletable.
func TestLegacySentinelIsStillRecognisedAsKennelManaged(t *testing.T) {
	dir := t.TempDir()
	legacy := legacyGitignoreSentinel + "\n/old-hook-file\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureWorkspaceGitignore(dir, "hook.json"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), GitignoreSentinel) {
		t.Fatalf("legacy-managed .gitignore was not rewritten with the current sentinel:\n%s", content)
	}
	if strings.Contains(string(content), "/old-hook-file") {
		t.Fatal("rewrite kept stale entries from the legacy file")
	}
}

// A .gitignore Kennel did not write is still never touched.
func TestForeignGitignoreIsLeftAlone(t *testing.T) {
	dir := t.TempDir()
	foreign := "# the repository's own ignore file\nnode_modules/\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(foreign), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureWorkspaceGitignore(dir, "hook.json"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != foreign {
		t.Fatalf("a foreign .gitignore was modified:\n%s", content)
	}
}

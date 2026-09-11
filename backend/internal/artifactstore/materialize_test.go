package artifactstore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// predecessorBinary is deliberately non-UTF8 with an embedded NUL so a
// transfer that silently went through text handling would not survive it.
var predecessorBinary = []byte{0x00, 0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0xff, 0xfe, 0x00, 0x42}

// newBaseRepo creates the Project repository at the approved base, plus a
// checkout function for the isolated worktrees an Attempt actually runs in.
//
// Checkouts are clones of one repository rather than separately built ones:
// production successors branch from the same base commit as their
// predecessors, and two independently created repositories only share a commit
// id by the accident of being made in the same second.
func newBaseRepo(t *testing.T) (checkout func(name string) string, base string) {
	t.Helper()
	origin := t.TempDir()
	git(t, origin, "init", "-q")
	git(t, origin, "config", "user.email", "test@example.com")
	git(t, origin, "config", "user.name", "Kennel Test")
	for name, body := range map[string]string{
		"untouched.txt": "base content nobody changed\n",
		"changed.txt":   "the base version\n",
		"removed.txt":   "the predecessor deletes this\n",
	} {
		if err := os.WriteFile(filepath.Join(origin, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(t, origin, "add", ".")
	git(t, origin, "commit", "-qm", "base")
	base = git(t, origin, "rev-parse", "HEAD")
	return func(name string) string {
		path := filepath.Join(t.TempDir(), name)
		if out, err := exec.Command("git", "clone", "-q", origin, path).CombinedOutput(); err != nil {
			t.Fatalf("clone %s: %v (%s)", name, err, out)
		}
		return path
	}, base
}

// TestMaterialize_GivesTheSuccessorTheExactPredecessorTreeOnTopOfItsBase is
// the artifact-continuity mechanism: a successor must end up holding what the
// predecessor left behind, including the base files the predecessor never
// touched, an executable bit, exact binary bytes, and a deletion.
func TestMaterialize_GivesTheSuccessorTheExactPredecessorTreeOnTopOfItsBase(t *testing.T) {
	checkout, base := newBaseRepo(t)
	producer := checkout("producer")
	if err := os.WriteFile(filepath.Join(producer, "changed.txt"), []byte("the predecessor's version\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(producer, "build.sh"), []byte("#!/bin/sh\necho produced\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(producer, "asset.bin"), predecessorBinary, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(producer, "removed.txt")); err != nil {
		t.Fatal(err)
	}

	store, err := New(Config{Root: filepath.Join(t.TempDir(), "artifacts")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Retain(context.Background(), Input{
		AttemptID: "att-a", OutcomeID: "out-1", PlanRevisionID: "plan-1", WorkUnitID: "wu-a",
		ContractRevisionNumber: 1, WorkspaceKind: domain.WorkspaceGitWorktree, WorkspacePath: producer, BaseRevision: base,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Receipt.RetentionState.Complete() {
		t.Fatalf("retention = %s (%s)", result.Receipt.RetentionState, result.Receipt.RetentionDetail)
	}

	// The producer's workspace is gone before the successor is provisioned,
	// which is the normal case: retained bytes have to stand on their own.
	if err := os.RemoveAll(producer); err != nil {
		t.Fatal(err)
	}

	successor := checkout("successor")
	handoff, err := store.Compose(context.Background(), []domain.AttemptReceipt{result.Receipt})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Materialize(context.Background(), handoff, successor, base); err != nil {
		t.Fatal(err)
	}

	assertFile := func(name string, want []byte, mode os.FileMode) {
		t.Helper()
		body, readErr := os.ReadFile(filepath.Join(successor, name))
		if readErr != nil {
			t.Fatalf("%s: %v", name, readErr)
		}
		if !bytes.Equal(body, want) {
			t.Fatalf("%s body = %q, want %q", name, body, want)
		}
		info, statErr := os.Lstat(filepath.Join(successor, name))
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Mode().Perm() != mode {
			t.Fatalf("%s mode = %o, want %o", name, info.Mode().Perm(), mode)
		}
	}
	assertFile("changed.txt", []byte("the predecessor's version\n"), 0o644)
	assertFile("build.sh", []byte("#!/bin/sh\necho produced\n"), 0o755)
	assertFile("asset.bin", predecessorBinary, 0o644)
	// A base file the predecessor never touched must survive: the manifest
	// describes changes, so unchanged content arrives by already being there.
	assertFile("untouched.txt", []byte("base content nobody changed\n"), 0o644)
	// A deletion is output too. A successor that still sees the file has not
	// received its predecessor's work.
	if _, err := os.Lstat(filepath.Join(successor, "removed.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted path survived materialization: %v", err)
	}
}

func TestMaterialize_RefusesAWorkspaceOnADifferentBase(t *testing.T) {
	checkout, base := newBaseRepo(t)
	producer := checkout("producer")
	if err := os.WriteFile(filepath.Join(producer, "changed.txt"), []byte("produced\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := New(Config{Root: filepath.Join(t.TempDir(), "artifacts")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Retain(context.Background(), Input{
		AttemptID: "att-a", OutcomeID: "out-1", PlanRevisionID: "plan-1", WorkUnitID: "wu-a",
		ContractRevisionNumber: 1, WorkspaceKind: domain.WorkspaceGitWorktree, WorkspacePath: producer, BaseRevision: base,
	})
	if err != nil {
		t.Fatal(err)
	}
	handoff, err := store.Compose(context.Background(), []domain.AttemptReceipt{result.Receipt})
	if err != nil {
		t.Fatal(err)
	}
	successor := checkout("successor")

	for _, successorBase := range []string{"0000000000000000000000000000000000000000", ""} {
		err := store.Materialize(context.Background(), handoff, successor, successorBase)
		if !errors.Is(err, ErrHandoffBaseMismatch) {
			t.Fatalf("materialize onto base %q = %v, want a base mismatch", successorBase, err)
		}
	}
	// Nothing may have been written before the refusal.
	if body, readErr := os.ReadFile(filepath.Join(successor, "changed.txt")); readErr != nil || string(body) != "the base version\n" {
		t.Fatalf("refused materialization still wrote: %q / %v", body, readErr)
	}
}

func TestMaterialize_RefusesPathsThatLeaveTheSuccessorWorkspace(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("untouched\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := New(Config{Root: filepath.Join(t.TempDir(), "artifacts")})
	if err != nil {
		t.Fatal(err)
	}
	successor := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(successor, "escape")); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		path string
	}{
		{"parent traversal", "../escaped.txt"},
		{"absolute path", "/etc/escaped.txt"},
		{"through a symlinked directory", "escape/secret.txt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handoff := Handoff{
				WorkspaceKind: domain.WorkspaceStagedFolder,
				Files: []HandoffFile{{
					RelativePath: tc.path, ChangeKind: domain.ArtifactAdded,
					Content: []byte("written\n"), FileMode: 0o644, ContentDigest: "unused",
				}},
			}
			if err := store.Materialize(context.Background(), handoff, successor, ""); err == nil {
				t.Fatalf("materialized escaping path %q", tc.path)
			}
		})
	}
	if body, readErr := os.ReadFile(filepath.Join(outside, "secret.txt")); readErr != nil || string(body) != "untouched\n" {
		t.Fatalf("content outside custody changed: %q / %v", body, readErr)
	}
}

func TestCompose_RefusesACorruptOrMissingRetainedBlob(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "result.txt"), []byte("produced\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "artifacts")
	store, err := New(Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Retain(context.Background(), Input{
		AttemptID: "att-a", OutcomeID: "out-1", PlanRevisionID: "plan-1", WorkUnitID: "wu-a",
		ContractRevisionNumber: 1, WorkspaceKind: domain.WorkspaceStagedFolder, WorkspacePath: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	blob := filepath.Join(root, "att-a", result.Receipt.ArtifactVersion, "result.txt")

	if err := os.WriteFile(blob, []byte("tampered\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Compose(context.Background(), []domain.AttemptReceipt{result.Receipt}); err == nil {
		t.Fatal("composed a handoff from a tampered blob")
	}
	if err := os.Remove(blob); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Compose(context.Background(), []domain.AttemptReceipt{result.Receipt}); err == nil {
		t.Fatal("composed a handoff from a missing blob")
	}
}

// TestCompose_RefusesPredecessorsThatCannotBeJoined covers the multi-parent
// cases a serial scheduler can still produce: two branches of the DAG both
// reaching the same successor.
func TestCompose_RefusesPredecessorsThatCannotBeJoined(t *testing.T) {
	store, err := New(Config{Root: filepath.Join(t.TempDir(), "artifacts")})
	if err != nil {
		t.Fatal(err)
	}
	retain := func(attempt, unit, name string, body []byte, mode os.FileMode, kind domain.WorkspaceKind, base string) domain.AttemptReceipt {
		t.Helper()
		workspace := t.TempDir()
		if body != nil {
			if err := os.WriteFile(filepath.Join(workspace, name), body, mode); err != nil {
				t.Fatal(err)
			}
		}
		result, retainErr := store.Retain(context.Background(), Input{
			AttemptID: domain.AttemptID(attempt), OutcomeID: "out-1", PlanRevisionID: "plan-1",
			WorkUnitID: domain.WorkUnitID(unit), ContractRevisionNumber: 1,
			WorkspaceKind: kind, WorkspacePath: workspace, BaseRevision: base,
		})
		if retainErr != nil {
			t.Fatal(retainErr)
		}
		return result.Receipt
	}

	left := retain("att-left", "wu-left", "shared.txt", []byte("left wins\n"), 0o644, domain.WorkspaceStagedFolder, "")
	sameBytes := retain("att-same", "wu-same", "shared.txt", []byte("left wins\n"), 0o644, domain.WorkspaceStagedFolder, "")
	otherBytes := retain("att-right", "wu-right", "shared.txt", []byte("right wins\n"), 0o644, domain.WorkspaceStagedFolder, "")
	otherMode := retain("att-mode", "wu-mode", "shared.txt", []byte("left wins\n"), 0o755, domain.WorkspaceStagedFolder, "")

	// Identical results for the same path are a legitimate shared ancestry,
	// not a conflict, so the join must succeed rather than refuse everything.
	if _, err := store.Compose(context.Background(), []domain.AttemptReceipt{left, sameBytes}); err != nil {
		t.Fatalf("byte-identical predecessors refused: %v", err)
	}
	if _, err := store.Compose(context.Background(), []domain.AttemptReceipt{left, otherBytes}); err == nil {
		t.Fatal("conflicting writes to the same path composed into a last-writer-wins result")
	}
	if _, err := store.Compose(context.Background(), []domain.AttemptReceipt{left, otherMode}); err == nil {
		t.Fatal("conflicting file modes for the same path composed")
	}
	if _, err := store.Compose(context.Background(), []domain.AttemptReceipt{left, left}); err == nil {
		t.Fatal("the same artifact version composed with itself")
	}
}

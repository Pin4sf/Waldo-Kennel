package artifactstore

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func documentStore(t *testing.T) *Store {
	t.Helper()
	store, err := New(Config{Root: filepath.Join(t.TempDir(), "artifacts")})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func writeDocument(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestSnapshotDocuments_KeepsTheApprovedBytesAndLeavesOriginalsAlone is the
// whole point of snapshotting: what runs is what was approved, and the
// owner's files are input, never writable output.
func TestSnapshotDocuments_KeepsTheApprovedBytesAndLeavesOriginalsAlone(t *testing.T) {
	store := documentStore(t)
	dir := t.TempDir()
	brief := writeDocument(t, dir, "brief.md", "# Original brief\n")
	notes := writeDocument(t, dir, "notes.txt", "first note\n")

	sources, err := store.SnapshotDocuments(context.Background(), "dctx-1", []string{brief, notes})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(sources) != 2 {
		t.Fatalf("sources = %d, want 2", len(sources))
	}

	// The owner edits their file after approving it.
	if err := os.WriteFile(brief, []byte("# Rewritten after approval\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	body, err := store.ReadDocument("dctx-1", sources[0])
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(body) != "# Original brief\n" {
		t.Fatalf("snapshot = %q, want the bytes that were approved", body)
	}
	// And the edit is reported, so the owner can deliberately re-select.
	changed := store.SourcesChangedSince(sources)
	if len(changed) != 1 || changed[0] != "brief.md" {
		t.Fatalf("changed sources = %v, want just brief.md", changed)
	}
}

// TestSnapshotDocuments_RefusesWhatItCannotRepresentHonestly keeps the
// selection bounded and secret-safe.
func TestSnapshotDocuments_RefusesWhatItCannotRepresentHonestly(t *testing.T) {
	store := documentStore(t)
	dir := t.TempDir()
	ordinary := writeDocument(t, dir, "notes.txt", "fine\n")

	secret := writeDocument(t, dir, ".env", "TOKEN=do-not-copy\n")
	if _, err := store.SnapshotDocuments(context.Background(), "dctx-secret", []string{secret}); err == nil {
		t.Fatal("a secret-like document was snapshotted")
	}

	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(ordinary, link); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SnapshotDocuments(context.Background(), "dctx-link", []string{link}); err == nil {
		t.Fatal("a symlink was snapshotted instead of being refused")
	}

	if _, err := store.SnapshotDocuments(context.Background(), "dctx-rel", []string{"notes.txt"}); err == nil {
		t.Fatal("a relative path was accepted")
	}

	other := t.TempDir()
	clash := writeDocument(t, other, "notes.txt", "different\n")
	if _, err := store.SnapshotDocuments(context.Background(), "dctx-clash", []string{ordinary, clash}); err == nil {
		t.Fatal("two documents staging under the same name were accepted")
	}
}

// TestDocumentHandoff_StagesExactlyTheApprovedSnapshot proves documents reach
// a workspace through the same verified materialization path as a
// predecessor's retained result.
func TestDocumentHandoff_StagesExactlyTheApprovedSnapshot(t *testing.T) {
	store := documentStore(t)
	dir := t.TempDir()
	binaryish := []byte{0x41, 0x00, 0x42}
	if err := os.WriteFile(filepath.Join(dir, "data.csv"), []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "raw.txt"), binaryish, 0o644); err != nil {
		t.Fatal(err)
	}
	sources, err := store.SnapshotDocuments(context.Background(),
		"dctx-stage", []string{filepath.Join(dir, "data.csv"), filepath.Join(dir, "raw.txt")})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	handoff, err := store.DocumentHandoff("dctx-stage", sources)
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}
	if handoff.WorkspaceKind != domain.WorkspaceStagedFolder {
		t.Fatalf("custody = %q, want a staged folder; documents have no revisions", handoff.WorkspaceKind)
	}
	if handoff.BaseRevision != "" {
		t.Fatalf("base revision = %q, want none claimed for a staged folder", handoff.BaseRevision)
	}

	workspace := t.TempDir()
	if err := store.Materialize(context.Background(), handoff, workspace, ""); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(workspace, "data.csv"))
	if err != nil || string(got) != "a,b\n1,2\n" {
		t.Fatalf("staged csv = %q / %v", got, err)
	}
	raw, err := os.ReadFile(filepath.Join(workspace, "raw.txt"))
	if err != nil || !bytes.Equal(raw, binaryish) {
		t.Fatalf("staged bytes = %v / %v", raw, err)
	}
}

// TestDocumentContextDigest_IdentifiesTheSelectionNotItsOrigin keeps the
// grounding digest meaningful: the same material selected from a different
// directory is the same context, and different content is not.
func TestDocumentContextDigest_IdentifiesTheSelectionNotItsOrigin(t *testing.T) {
	store := documentStore(t)
	first, second := t.TempDir(), t.TempDir()
	writeDocument(t, first, "brief.md", "same\n")
	writeDocument(t, second, "brief.md", "same\n")

	one, err := store.SnapshotDocuments(context.Background(), "dctx-a", []string{filepath.Join(first, "brief.md")})
	if err != nil {
		t.Fatalf("snapshot a: %v", err)
	}
	two, err := store.SnapshotDocuments(context.Background(), "dctx-b", []string{filepath.Join(second, "brief.md")})
	if err != nil {
		t.Fatalf("snapshot b: %v", err)
	}
	if domain.DocumentContextDigest(one) != domain.DocumentContextDigest(two) {
		t.Fatal("identical material from two directories produced different context digests")
	}

	writeDocument(t, second, "brief.md", "different\n")
	three, err := store.SnapshotDocuments(context.Background(), "dctx-c", []string{filepath.Join(second, "brief.md")})
	if err != nil {
		t.Fatalf("snapshot c: %v", err)
	}
	if domain.DocumentContextDigest(one) == domain.DocumentContextDigest(three) {
		t.Fatal("changed content did not change the context digest")
	}
}

// TestReadDocument_RefusesASnapshotThatNoLongerMatches keeps a tampered
// snapshot from being staged as approved material.
func TestReadDocument_RefusesASnapshotThatNoLongerMatches(t *testing.T) {
	store := documentStore(t)
	dir := t.TempDir()
	sources, err := store.SnapshotDocuments(context.Background(),
		"dctx-tamper", []string{writeDocument(t, dir, "brief.md", "approved\n")})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	staged := filepath.Join(store.root, documentRoot, "dctx-tamper", "brief.md")
	if err := os.WriteFile(staged, []byte("tampered\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadDocument("dctx-tamper", sources[0]); err == nil ||
		!strings.Contains(err.Error(), "no longer matches") {
		t.Fatalf("tampered snapshot read = %v, want a refusal", err)
	}
}

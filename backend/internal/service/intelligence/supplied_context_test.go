package intelligence

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSuppliedDocumentContextIsBoundedAndDigestable(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "brief.md")
	second := filepath.Join(root, "data.csv")
	if err := os.WriteFile(first, []byte("## brief\nA bounded brief."), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("name,value\nanswer,yes\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	one, err := BuildSuppliedDocumentContext(context.Background(), []string{second, first})
	if err != nil {
		t.Fatal(err)
	}
	two, err := BuildSuppliedDocumentContext(context.Background(), []string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if one.Digest != two.Digest || len(one.Files) != 2 {
		t.Fatalf("snapshots = %#v / %#v", one, two)
	}
}

func TestBuildSuppliedDocumentContextRefusesUnsupportedSecretAndSymlink(t *testing.T) {
	root := t.TempDir()
	secret := filepath.Join(root, "credentials.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildSuppliedDocumentContext(context.Background(), []string{secret}); err == nil {
		t.Fatal("secret-like supplied file accepted")
	}
	pdf := filepath.Join(root, "report.pdf")
	if err := os.WriteFile(pdf, []byte("pdf"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildSuppliedDocumentContext(context.Background(), []string{pdf}); err == nil {
		t.Fatal("PDF silently accepted")
	}
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildSuppliedDocumentContext(context.Background(), []string{link}); err == nil {
		t.Fatal("symlink supplied file accepted")
	}
}

// SP6. The window between deciding a path is small enough and reading it is a
// real window. Growth or replacement in that window must be refused at the
// read, not discovered after the bytes are already in memory.
func TestSuppliedDocumentRefusesGrowthBetweenSelectionAndRead(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(doc, []byte("small\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	swapped := false
	afterLstat = func(path string) {
		if swapped || path != doc {
			return
		}
		swapped = true
		if err := os.WriteFile(doc, bytes.Repeat([]byte("x"), suppliedContextMaxFile*4), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { afterLstat = nil })

	snapshot, err := BuildSuppliedDocumentContext(context.Background(), []string{doc})
	if err == nil {
		t.Fatal("a document that grew past its bound was accepted")
	}
	// The specific error matters. Detecting the change *after* reading the
	// whole file reports "changed during read"; stopping at the bound reports
	// the bound, which is the only one that proves nothing oversized was read.
	if !strings.Contains(err.Error(), "byte bound") {
		t.Fatalf("err = %v, want the bound to fire at the read", err)
	}
	if len(snapshot.Files) != 0 {
		t.Fatalf("refused build returned %d files", len(snapshot.Files))
	}
}

func TestSuppliedDocumentRefusesReplacementWithASymlink(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "brief.md")
	target := filepath.Join(dir, "elsewhere.md")
	if err := os.WriteFile(doc, []byte("brief\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("someone else's document\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	swapped := false
	afterLstat = func(path string) {
		if swapped || path != doc {
			return
		}
		swapped = true
		if err := os.Remove(doc); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, doc); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { afterLstat = nil })

	snapshot, err := BuildSuppliedDocumentContext(context.Background(), []string{doc})
	if err == nil {
		t.Fatal("a path replaced by a symlink was still read")
	}
	// Identity is checked against the open descriptor, so the swap is caught
	// before any of the replacement is read.
	if !strings.Contains(err.Error(), "changed between selection and read") {
		t.Fatalf("err = %v, want the identity check to reject before reading", err)
	}
	for _, file := range snapshot.Files {
		if strings.Contains(file.Content, "someone else") {
			t.Fatal("content came from the replacement target")
		}
	}
}

func TestSuppliedDocumentRespectsCancellation(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(doc, []byte("content\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := BuildSuppliedDocumentContext(ctx, []string{doc}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

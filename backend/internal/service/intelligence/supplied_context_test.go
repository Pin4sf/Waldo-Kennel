package intelligence

import (
	"context"
	"os"
	"path/filepath"
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

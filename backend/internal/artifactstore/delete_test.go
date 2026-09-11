package artifactstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteRetainedContentPreservesOutsideFiles(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "keep")
	if err := os.WriteFile(target, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(outside, filepath.Join(root, "att-owned")); err != nil {
		t.Fatal(err)
	}
	if err = s.RemoveOutcomeContent(context.Background(), []string{"att-owned"}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(target); err != nil {
		t.Fatal("followed a symlink outside retention root", err)
	}
	if err = s.RemoveOutcomeContent(context.Background(), []string{"../escape"}, nil); err == nil {
		t.Fatal("accepted traversal")
	}
	if err = os.Symlink(outside, filepath.Join(root, documentRoot)); err != nil {
		t.Fatal(err)
	}
	if err = s.RemoveOutcomeContent(context.Background(), nil, []string{"keep"}); err == nil {
		t.Fatal("accepted replaced document root")
	}
	if _, err = os.Stat(target); err != nil {
		t.Fatal("outside file removed", err)
	}
}

package secretstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreKeepsSecretOutOfSQLiteAndRestrictsPermissions(t *testing.T) {
	root := t.TempDir()
	store := NewFileStore(root)
	const canary = "reasoning-canary-do-not-log"
	if err := store.Set(context.Background(), canary); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(context.Background())
	if err != nil || got != canary {
		t.Fatalf("Get() = %q, %v; want canary", got, err)
	}
	path := filepath.Join(root, "secrets", "waldo-reasoning-api-key")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("secret permissions = %o, want 600", info.Mode().Perm())
	}
	dir, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if dir.Mode().Perm() != 0o700 {
		t.Fatalf("secret directory permissions = %o, want 700", dir.Mode().Perm())
	}
	if err := store.Clear(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err = store.Get(context.Background())
	if err != nil || got != "" {
		t.Fatalf("cleared Get() = %q, %v; want empty", got, err)
	}
}

func TestFileStoreBindsCredentialsToProvider(t *testing.T) {
	store := NewFileStore(t.TempDir())
	if err := store.SetForProvider(context.Background(), "anthropic", "anthropic-canary"); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetForProvider(context.Background(), "openai")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("OpenAI lookup reused Anthropic credential %q", got)
	}
	got, err = store.GetForProvider(context.Background(), "anthropic")
	if err != nil || got != "anthropic-canary" {
		t.Fatalf("Anthropic lookup = %q, %v; want stored credential", got, err)
	}
}

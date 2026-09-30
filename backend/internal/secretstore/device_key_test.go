package secretstore

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeviceKeyStoreCustody(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	store := NewDeviceKeyStore(root)
	ref, err := store.Put(context.Background(), privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ref, base64.RawURLEncoding.EncodeToString(privateKey)) {
		t.Fatal("custody reference contains key material")
	}
	got, err := store.Get(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, privateKey) {
		t.Fatal("retrieved key differs from generated key")
	}
	path, err := store.path(ref)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("key file mode = %o", info.Mode().Perm())
	}
	dir, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if dir.Mode().Perm() != 0o700 {
		t.Fatalf("key directory mode = %o", dir.Mode().Perm())
	}
	if err := store.Clear(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(context.Background(), ref); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Get after clear error = %v", err)
	}
}

func TestDeviceKeyStoreRejectsInvalidReferenceAndMaterial(t *testing.T) {
	store := NewDeviceKeyStore(t.TempDir())
	if _, err := store.Put(context.Background(), make(ed25519.PrivateKey, ed25519.PrivateKeySize)); !errors.Is(err, ErrDeviceKeyInvalid) {
		t.Fatalf("Put malformed key error = %v", err)
	}
	if _, err := store.Get(context.Background(), "device-key:../escape"); !errors.Is(err, ErrDeviceKeyInvalid) {
		t.Fatalf("Get path traversal error = %v", err)
	}
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	path, _ := store.path(ref)
	if err := os.WriteFile(path, []byte("corrupt-key-material"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(context.Background(), ref); !errors.Is(err, ErrDeviceKeyInvalid) || strings.Contains(err.Error(), "corrupt-key-material") {
		t.Fatalf("Get corrupted key error = %v", err)
	}
}

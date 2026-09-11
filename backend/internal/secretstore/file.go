// Package secretstore contains the small local secret boundary used by the
// daemon for credentials that must not become SQLite/domain data.
package secretstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store persists one local secret without returning it through API settings.
type Store interface {
	Get(context.Context) (string, error)
	Set(context.Context, string) error
	Clear(context.Context) error
}

// FileStore keeps a secret under the daemon data directory with restrictive
// directory/file modes. This is the repository's local fallback abstraction
// when an OS keychain is not available; the path remains outside SQLite and
// the application state policy owns its backup/permissions.
type FileStore struct{ path string }

// NewFileStore creates a daemon-owned secret store rooted at dataDir.
func NewFileStore(dataDir string) *FileStore {
	return &FileStore{path: filepath.Join(dataDir, "secrets", "waldo-reasoning-api-key")}
}

// Get reads the secret without exposing it through the settings API.
func (s *FileStore) Get(ctx context.Context) (string, error) {
	return s.getAt(ctx, s.path)
}

// GetForProvider reads only the credential explicitly stored for provider.
// The legacy unqualified file is intentionally not a fallback: its provider
// identity is unknowable and reusing it could send a credential to the wrong
// provider after a settings switch.
func (s *FileStore) GetForProvider(ctx context.Context, provider string) (string, error) {
	path, err := s.providerPath(provider)
	if err != nil {
		return "", err
	}
	return s.getAt(ctx, path)
}

func (s *FileStore) getAt(ctx context.Context, path string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read local reasoning secret: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

// Set atomically replaces the local secret with restrictive file permissions.
func (s *FileStore) Set(ctx context.Context, value string) error {
	return s.setAt(ctx, s.path, value)
}

// SetForProvider stores a credential under an explicit provider identity.
func (s *FileStore) SetForProvider(ctx context.Context, provider, value string) error {
	path, err := s.providerPath(provider)
	if err != nil {
		return err
	}
	return s.setAt(ctx, path, value)
}

func (s *FileStore) setAt(ctx context.Context, path, value string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return s.Clear(ctx)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create local secret directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil { //nolint:gosec // private secret directory is intentionally owner-only.
		return fmt.Errorf("restrict local secret directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".waldo-reasoning-*")
	if err != nil {
		return fmt.Errorf("create local secret file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("restrict local secret file: %w", err)
	}
	if _, err := tmp.WriteString(value + "\n"); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write local reasoning secret: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("flush local reasoning secret: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close local reasoning secret: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("install local reasoning secret: %w", err)
	}
	return nil
}

// Clear removes the local secret if it exists.
func (s *FileStore) Clear(ctx context.Context) error {
	return s.clearAt(ctx, s.path)
}

// ClearForProvider removes only the selected provider's credential.
func (s *FileStore) ClearForProvider(ctx context.Context, provider string) error {
	path, err := s.providerPath(provider)
	if err != nil {
		return err
	}
	return s.clearAt(ctx, path)
}

func (s *FileStore) clearAt(ctx context.Context, path string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear local reasoning secret: %w", err)
	}
	return nil
}

func (s *FileStore) providerPath(provider string) (string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider != "anthropic" && provider != "openai" {
		return "", fmt.Errorf("unsupported reasoning provider %q", provider)
	}
	return filepath.Join(filepath.Dir(s.path), "waldo-reasoning-api-key-"+provider), nil
}

func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

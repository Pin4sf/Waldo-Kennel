package governedtools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

const uncertaintyDirectory = "governed-check-uncertainty"

// UncertaintyStore is the crash-safe handoff between the private MCP process
// and the daemon. The marker lives in Kennel application state, not in the
// repository result; the daemon imports it into append-only Attempt history.
type UncertaintyStore struct {
	root string
}

// NewUncertaintyStore binds the marker store to one Kennel data directory.
func NewUncertaintyStore(dataDir string) (*UncertaintyStore, error) {
	root := filepath.Clean(strings.TrimSpace(dataDir))
	if root == "." || !filepath.IsAbs(root) {
		return nil, errors.New("governed check uncertainty data directory must be absolute")
	}
	return &UncertaintyStore{root: filepath.Join(root, uncertaintyDirectory)}, nil
}

func (s *UncertaintyStore) markerPath(id domain.SessionID) (string, error) {
	if s == nil || strings.TrimSpace(string(id)) == "" {
		return "", errors.New("governed check uncertainty session id is required")
	}
	digest := sha256.Sum256([]byte(id))
	return filepath.Join(s.root, hex.EncodeToString(digest[:])+".json"), nil
}

// RecordGovernedCheckUncertainty atomically publishes the first unknown
// termination. Rewriting the same session marker is harmless and remains a
// denial-only operation; it can never grant execution authority.
func (s *UncertaintyStore) RecordGovernedCheckUncertainty(ctx context.Context, fact ports.GovernedCheckUncertainty) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !fact.TerminationUnknown {
		return errors.New("only unknown check termination is durable custody evidence")
	}
	path, err := s.markerPath(fact.SessionID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return fmt.Errorf("create governed uncertainty directory: %w", err)
	}
	raw, err := json.Marshal(fact)
	if err != nil {
		return fmt.Errorf("encode governed check uncertainty: %w", err)
	}
	tmp := filepath.Join(s.root, ".uncertainty-"+uuid.NewString())
	file, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create governed check uncertainty: %w", err)
	}
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmp)
		}
	}()
	if _, err = file.Write(raw); err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write governed check uncertainty: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("publish governed check uncertainty: %w", err)
	}
	removeTmp = false
	return syncDirectory(s.root)
}

// ClearGovernedCheckUncertainty removes the conservative marker only after
// the check runner has confirmed that the process tree terminated.
func (s *UncertaintyStore) ClearGovernedCheckUncertainty(ctx context.Context, id domain.SessionID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.markerPath(id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear governed check uncertainty: %w", err)
	}
	return syncDirectory(s.root)
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open governed uncertainty directory for sync: %w", err)
	}
	defer func() { _ = dir.Close() }()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync governed uncertainty directory: %w", err)
	}
	return nil
}

// GovernedCheckUncertainty reads one durable session marker.
func (s *UncertaintyStore) GovernedCheckUncertainty(ctx context.Context, id domain.SessionID) (ports.GovernedCheckUncertainty, bool, error) {
	if err := ctx.Err(); err != nil {
		return ports.GovernedCheckUncertainty{}, false, err
	}
	path, err := s.markerPath(id)
	if err != nil {
		return ports.GovernedCheckUncertainty{}, false, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ports.GovernedCheckUncertainty{}, false, nil
	}
	if err != nil {
		return ports.GovernedCheckUncertainty{}, false, fmt.Errorf("read governed check uncertainty: %w", err)
	}
	var fact ports.GovernedCheckUncertainty
	if err := json.Unmarshal(raw, &fact); err != nil {
		return ports.GovernedCheckUncertainty{}, false, fmt.Errorf("decode governed check uncertainty: %w", err)
	}
	if fact.SessionID != id || !fact.TerminationUnknown || fact.CheckID.IsZero() || fact.ObservedAt.IsZero() {
		return ports.GovernedCheckUncertainty{}, false, errors.New("governed check uncertainty marker is invalid")
	}
	return fact, true, nil
}

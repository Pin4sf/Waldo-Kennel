package secretstore

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const deviceKeyRefPrefix = "device-key:"

var ErrDeviceKeyInvalid = errors.New("invalid device key")

// DeviceKeyStore gives bridge keys a per-device custody reference while using
// the same restrictive, atomic file-secret boundary as other daemon secrets.
// References are random locators, never key material or filesystem paths.
type DeviceKeyStore struct {
	dataDir string
	files   FileStore
}

func NewDeviceKeyStore(dataDir string) *DeviceKeyStore {
	return &DeviceKeyStore{dataDir: dataDir}
}

// Put stores an ed25519 private key and returns its opaque custody reference.
// Neither the key nor its encoded form is included in errors.
func (s *DeviceKeyStore) Put(ctx context.Context, key ed25519.PrivateKey) (string, error) {
	if s == nil || !validDevicePrivateKey(key) {
		return "", ErrDeviceKeyInvalid
	}
	for attempt := 0; attempt < 3; attempt++ {
		var random [16]byte
		if _, err := io.ReadFull(rand.Reader, random[:]); err != nil {
			return "", fmt.Errorf("generate device key reference: %w", err)
		}
		ref := deviceKeyRefPrefix + hex.EncodeToString(random[:])
		path, _ := s.path(ref)
		if _, err := os.Lstat(path); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("check device key reference: %w", err)
		}
		if err := s.files.setAt(ctx, path, base64.RawURLEncoding.EncodeToString(key)); err != nil {
			return "", fmt.Errorf("store device key: %w", err)
		}
		return ref, nil
	}
	return "", errors.New("allocate device key reference")
}

func (s *DeviceKeyStore) Get(ctx context.Context, ref string) (ed25519.PrivateKey, error) {
	path, err := s.path(ref)
	if err != nil {
		return nil, err
	}
	encoded, err := s.files.getAt(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("read device key: %w", err)
	}
	if encoded == "" {
		return nil, os.ErrNotExist
	}
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || !validDevicePrivateKey(decoded) {
		return nil, ErrDeviceKeyInvalid
	}
	return ed25519.PrivateKey(decoded), nil
}

func (s *DeviceKeyStore) Clear(ctx context.Context, ref string) error {
	path, err := s.path(ref)
	if err != nil {
		return err
	}
	if err := s.files.clearAt(ctx, path); err != nil {
		return fmt.Errorf("clear device key: %w", err)
	}
	return nil
}

func (s *DeviceKeyStore) path(ref string) (string, error) {
	if s == nil || !filepath.IsAbs(s.dataDir) || !strings.HasPrefix(ref, deviceKeyRefPrefix) || len(ref) != len(deviceKeyRefPrefix)+32 {
		return "", ErrDeviceKeyInvalid
	}
	id := strings.TrimPrefix(ref, deviceKeyRefPrefix)
	if _, err := hex.DecodeString(id); err != nil || strings.ToLower(id) != id {
		return "", ErrDeviceKeyInvalid
	}
	return filepath.Join(s.dataDir, "secrets", "device-keys", id), nil
}

func validDevicePrivateKey(key []byte) bool {
	if len(key) != ed25519.PrivateKeySize {
		return false
	}
	derived := ed25519.NewKeyFromSeed(key[:ed25519.SeedSize])
	return subtle.ConstantTimeCompare(derived, key) == 1
}

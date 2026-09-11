package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SHA256Digest is the canonical persisted content digest used where Kennel
// needs immutable provenance without retaining the source payload.
type SHA256Digest string

// IsZero reports whether the digest is unset.
func (d SHA256Digest) IsZero() bool   { return strings.TrimSpace(string(d)) == "" }
func (d SHA256Digest) String() string { return string(d) }

// Valid reports whether the digest has the expected SHA-256 encoding.
func (d SHA256Digest) Valid() bool { return isSHA256Hex(string(d)) }

// DigestSHA256 hashes source bytes without persisting the source itself.
func DigestSHA256(source []byte) SHA256Digest {
	sum := sha256.Sum256(source)
	return SHA256Digest(hex.EncodeToString(sum[:]))
}

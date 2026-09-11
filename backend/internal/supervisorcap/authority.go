package supervisorcap

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// Authority issues launch-bound capabilities for Kennel's private process
// supervisor. It is stateless so a replacement daemon can validate a report
// from a worker generation that survived the restart.
const (
	EnvCapability    = "KENNEL_SUPERVISOR_CAPABILITY"
	HeaderCapability = "X-Kennel-Supervisor-Capability"
)

type Authority struct{}

func NewAuthority() *Authority { return &Authority{} }

func (a *Authority) Issue(sessionID domain.SessionID, launchID string) (token, verifier string, err error) {
	if a == nil || sessionID == "" || launchID == "" {
		return "", "", fmt.Errorf("issue supervisor capability: session id and launch id are required")
	}
	raw := make([]byte, sha256.Size)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate supervisor capability: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, capabilityVerifier(sessionID, launchID, token), nil
}

func (a *Authority) Valid(sessionID domain.SessionID, launchID, token, verifier string) bool {
	if a == nil || sessionID == "" || launchID == "" || token == "" || verifier == "" {
		return false
	}
	expected := capabilityVerifier(sessionID, launchID, token)
	return hmac.Equal([]byte(expected), []byte(verifier))
}

func capabilityVerifier(sessionID domain.SessionID, launchID, token string) string {
	h := sha256.New()
	_, _ = h.Write([]byte("kennel-supervisor-capability-verifier-v1\x00"))
	_, _ = h.Write([]byte(sessionID))
	_, _ = h.Write([]byte{'\x00'})
	_, _ = h.Write([]byte(launchID))
	_, _ = h.Write([]byte{'\x00'})
	_, _ = h.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

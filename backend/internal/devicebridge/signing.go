// Package devicebridge implements the outbound Waldo device transport. This
// file contains the v0.2.3 signing primitives shared by redeem and connect.
package devicebridge

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidSigningInput = errors.New("invalid device signing input")

// NewNonce returns the v0.2 16-byte CSPRNG nonce. Callers create a new nonce
// for every signed request or frame; no secret material appears in errors.
func NewNonce(random io.Reader) ([NonceBytes]byte, error) {
	var nonce [NonceBytes]byte
	if random == nil {
		random = rand.Reader
	}
	if _, err := io.ReadFull(random, nonce[:]); err != nil {
		return [NonceBytes]byte{}, err
	}
	return nonce, nil
}

// HTTPBaseString is the exact v0.2.3 HTTP signing string. The nonce is part
// of the signed bytes, and the path includes the exact connect query.
func HTTPBaseString(timestamp time.Time, nonce [NonceBytes]byte, method, path string, body []byte) (string, error) {
	if !validTimestamp(timestamp.Unix()) || !validHTTPShape(method, path, body) {
		return "", ErrInvalidSigningInput
	}
	hash := sha256.Sum256(body)
	return strings.Join([]string{
		formatTimestamp(timestamp), base64.RawURLEncoding.EncodeToString(nonce[:]), method, path, hex.EncodeToString(hash[:]),
	}, "\n"), nil
}

func validHTTPShape(method, path string, body []byte) bool {
	if method == http.MethodPost && path == RedeemPath {
		return len(body) <= RedeemBodyMaxBytes
	}
	if method != http.MethodGet || len(body) != 0 || !strings.HasPrefix(path, ConnectPath+ConnectVersionQueryPrefix+ContractVersion+ConnectCapabilitiesQuery) {
		return false
	}
	declaration := strings.TrimPrefix(path, ConnectPath+ConnectVersionQueryPrefix+ContractVersion+ConnectCapabilitiesQuery)
	for _, capabilities := range [][]string{{ClassMachineStateQuery}, {ClassNotifyLocal}, {ClassMachineStateQuery, ClassNotifyLocal}} {
		target, _ := ConnectRequestTarget(capabilities)
		if target == path && declaration != "" {
			return true
		}
	}
	return false
}

// SignHTTPHeaders returns the v0.2 authentication headers for a request whose
// body bytes are already fixed. Redeem omits deviceID; connect supplies it.
func SignHTTPHeaders(key ed25519.PrivateKey, timestamp time.Time, method, path string, body []byte, nonce [NonceBytes]byte, deviceID string) (http.Header, error) {
	if len(key) != ed25519.PrivateKeySize ||
		(method == http.MethodPost && deviceID != "") ||
		(method == http.MethodGet && !validIdentifier(deviceID)) {
		return nil, ErrInvalidSigningInput
	}
	base, err := HTTPBaseString(timestamp, nonce, method, path, body)
	if err != nil {
		return nil, err
	}
	header := make(http.Header)
	header.Set("X-Waldo-Timestamp", formatTimestamp(timestamp))
	header.Set("X-Waldo-Nonce", base64.RawURLEncoding.EncodeToString(nonce[:]))
	header.Set("X-Waldo-Signature", base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, []byte(base))))
	if deviceID != "" {
		header.Set("X-Waldo-Device-Id", deviceID)
	}
	return header, nil
}

func formatTimestamp(timestamp time.Time) string {
	return strconv.FormatInt(timestamp.Unix(), 10)
}

func validTimestamp(seconds int64) bool {
	return seconds > 0 && len(strconv.FormatInt(seconds, 10)) <= MaxTimestampDigits
}

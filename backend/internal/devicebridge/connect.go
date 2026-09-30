package devicebridge

import (
	"crypto/ed25519"
	"net/http"
	"net/url"
	"time"
)

// SignedConnect prepares the exact WSS target and signed HTTP upgrade headers
// for one pinned HTTPS origin. The WebSocket dialer must use this target
// verbatim, verify TLS chain/hostname, and forbid redirects.
func SignedConnect(origin string, capabilities []string, key ed25519.PrivateKey, timestamp time.Time, nonce [NonceBytes]byte, deviceID string) (string, http.Header, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return "", nil, ErrInvalidSigningInput
	}
	target, err := ConnectRequestTarget(capabilities)
	if err != nil {
		return "", nil, err
	}
	headers, err := SignHTTPHeaders(key, timestamp, http.MethodGet, target, nil, nonce, deviceID)
	if err != nil {
		return "", nil, err
	}
	return "wss://" + u.Host + target, headers, nil
}

package devicebridge

import (
	"context"
	"errors"
	"github.com/coder/websocket"
	"net/http"
	"net/url"
)

var ErrAuthenticationRejected = errors.New("device bridge authentication rejected")

// Only a standard transport with certificate and hostname validation enabled
// is admitted. A custom RoundTripper or TLS dial callback could bypass TLS.
func secureHTTPClient(client *http.Client, hostname string) (*http.Client, error) {
	if client == nil {
		client = http.DefaultClient
	}
	copyClient := *client
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	t, ok := transport.(*http.Transport)
	if !ok || t.DialTLS != nil || t.DialTLSContext != nil {
		return nil, ErrPairingInvalid
	}
	t = t.Clone()
	if t.TLSClientConfig != nil {
		if t.TLSClientConfig.InsecureSkipVerify || (t.TLSClientConfig.ServerName != "" && t.TLSClientConfig.ServerName != hostname) {
			return nil, ErrPairingInvalid
		}
		t.TLSClientConfig = t.TLSClientConfig.Clone()
	}
	copyClient.Transport = t
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &copyClient, nil
}

type Socket interface {
	Read(context.Context) ([]byte, error)
	Write(context.Context, []byte) error
	Close() error
}

type WebSocketDialer struct{ Client *http.Client }

func (d WebSocketDialer) Dial(ctx context.Context, target string, headers http.Header) (Socket, error) {
	u, err := url.Parse(target)
	if err != nil || u.Scheme != "wss" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return nil, ErrInvalidSigningInput
	}
	client, err := secureHTTPClient(d.Client, u.Hostname())
	if err != nil {
		return nil, err
	}
	conn, response, err := websocket.Dial(ctx, target, &websocket.DialOptions{HTTPClient: client, HTTPHeader: headers, CompressionMode: websocket.CompressionDisabled})
	if err != nil {
		if response != nil && response.StatusCode == http.StatusUnauthorized {
			return nil, ErrAuthenticationRejected
		}
		return nil, err
	}
	conn.SetReadLimit(FrameMaxBytes)
	return socketConn{conn}, nil
}

type socketConn struct{ conn *websocket.Conn }

func (s socketConn) Read(ctx context.Context) ([]byte, error) {
	kind, raw, err := s.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	if kind != websocket.MessageText {
		return nil, ErrInvalidFrameShape
	}
	return raw, nil
}
func (s socketConn) Write(ctx context.Context, raw []byte) error {
	return s.conn.Write(ctx, websocket.MessageText, raw)
}
func (s socketConn) Close() error { return s.conn.CloseNow() }

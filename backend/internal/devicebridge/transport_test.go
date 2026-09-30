package devicebridge

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestOutboundWebSocketExactSignedTargetAndHeartbeat(t *testing.T) {
	c, _ := sessionFixture(t)
	received := make(chan error, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target, _ := ConnectRequestTarget(c.Capabilities)
		if r.RequestURI != target {
			received <- errors.New("changed connect target")
			w.WriteHeader(401)
			return
		}
		stamp, _ := strconv.ParseInt(r.Header.Get("X-Waldo-Timestamp"), 10, 64)
		n, _ := base64.RawURLEncoding.DecodeString(r.Header.Get("X-Waldo-Nonce"))
		var nonce [NonceBytes]byte
		copy(nonce[:], n)
		base, _ := HTTPBaseString(time.Unix(stamp, 0), nonce, http.MethodGet, r.RequestURI, nil)
		signature, _ := base64.RawURLEncoding.DecodeString(r.Header.Get("X-Waldo-Signature"))
		if r.Header.Get("X-Waldo-Device-Id") != c.Scope.DeviceID || !ed25519.Verify(c.Key.Public().(ed25519.PublicKey), []byte(base), signature) {
			received <- errors.New("invalid connect signature")
			w.WriteHeader(401)
			return
		}
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{CompressionMode: websocket.CompressionDisabled})
		if err != nil {
			received <- err
			return
		}
		defer conn.CloseNow()
		kind, raw, err := conn.Read(r.Context())
		if err == nil && kind != websocket.MessageText {
			err = errors.New("nontext device frame")
		}
		if err == nil {
			frame, e := VerifyFrame(raw, c.Key.Public().(ed25519.PublicKey), c.clock())
			err = e
			if e == nil && frame["type"] != TypeHeartbeat {
				err = errors.New("first frame not heartbeat")
			}
		}
		received <- err
	}))
	defer server.Close()
	c.Origin = server.URL
	c.Dialer = WebSocketDialer{Client: server.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = c.Connect(ctx)
	if err := <-received; err != nil {
		t.Fatal(err)
	}
}
func TestTransportNoRedirectAndTLSFailClosed(t *testing.T) {
	var redirected bool
	destination := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected = true }))
	defer destination.Close()
	source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, http.StatusFound) }))
	defer source.Close()
	_, err := (WebSocketDialer{Client: source.Client()}).Dial(context.Background(), strings.Replace(source.URL, "https://", "wss://", 1)+ConnectPath, http.Header{})
	if err == nil || redirected {
		t.Fatal("redirect followed")
	}
	_, err = (WebSocketDialer{}).Dial(context.Background(), strings.Replace(source.URL, "https://", "wss://", 1)+ConnectPath, http.Header{})
	if err == nil {
		t.Fatal("untrusted certificate accepted")
	}
	insecure := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	if _, err := secureHTTPClient(insecure, "example.com"); err == nil {
		t.Fatal("TLS bypass admitted")
	}
}

type failingDialer struct{ err error }

func (d failingDialer) Dial(context.Context, string, http.Header) (Socket, error) { return nil, d.err }
func TestAuthenticationRejectionUnpairsButTransientFailureRetainsIdentity(t *testing.T) {
	for _, auth := range []bool{true, false} {
		c, store := sessionFixture(t)
		failure := errors.New("transient transport")
		if auth {
			failure = ErrAuthenticationRejected
		}
		c.Dialer = failingDialer{failure}
		var states []ConnectionState
		c.State = func(state ConnectionState) { states = append(states, state) }
		if err := c.Connect(context.Background()); !errors.Is(err, failure) {
			t.Fatal(err)
		}
		if store.cleared != auth {
			t.Fatal("wrong paired state cleared")
		}
		if auth && states[len(states)-1] != ConnectionUnpaired {
			t.Fatal("wrong revoked projection")
		}
		if !auth && states[len(states)-1] != ConnectionOffline {
			t.Fatal("wrong transient projection")
		}
	}
}

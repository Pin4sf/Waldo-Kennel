package httpd

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/bridgeactivation"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/config"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ownercommand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const bridgeTestToken = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
const bridgeOwnerToken = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE"

type bridgeRepoSpy struct {
	reserves, pairs int
	blocked         bool
}

func (r *bridgeRepoSpy) Load(context.Context, string) (bridgeactivation.Record, error) {
	return bridgeactivation.Record{OwnerID: "owner"}, nil
}
func (r *bridgeRepoSpy) Reserve(context.Context, string) (bridgeactivation.Record, error) {
	r.reserves++
	if r.blocked {
		return bridgeactivation.Record{}, bridgeactivation.ErrBlocked
	}
	return bridgeactivation.Record{OwnerID: "owner", Attempt: "attempt"}, nil
}
func (r *bridgeRepoSpy) Complete(context.Context, string, string, domain.DeviceBridgeDevice) error {
	return nil
}
func (r *bridgeRepoSpy) Block(context.Context, string, string, *devicebridge.PairingRecoveryError) error {
	return nil
}
func (r *bridgeRepoSpy) Revoke(context.Context, string, domain.DeviceBridgeDeviceID) error {
	return nil
}
func (r *bridgeRepoSpy) Checkpoint(context.Context, string, string, devicebridge.PairingRecoveryError) error {
	return nil
}
func (r *bridgeRepoSpy) Pair(context.Context, devicebridge.PairRequest) (domain.DeviceBridgeDevice, error) {
	r.pairs++
	return domain.DeviceBridgeDevice{}, errors.New("synthetic unavailable")
}

type bridgeFactory struct{ ready bool }

func (f bridgeFactory) Ready(context.Context) error {
	if f.ready {
		return nil
	}
	return bridgeactivation.ErrNotReady
}
func (f bridgeFactory) New(context.Context, string, domain.DeviceBridgeDevice, func(devicebridge.ConnectionState)) (bridgeactivation.Session, error) {
	return nil, bridgeactivation.ErrNotReady
}
func bridgeRouter(t *testing.T, ready bool) (http.Handler, *bridgeRepoSpy) {
	t.Helper()
	repo := &bridgeRepoSpy{}
	c, e := bridgeactivation.New("owner", "https://example.test", repo, repo, bridgeFactory{ready})
	if e != nil {
		t.Fatal(e)
	}
	authority := ownercommand.NewBridgeAuthority(bridgeTestToken)
	h := bridgeactivation.Handler(c, func(r *http.Request) bool { return authority.Authenticate(r.Header.Get("Authorization")) })
	return NewRouterWithControl(config.Config{}, discardLogger(), nil, APIDeps{}, ControlDeps{BridgeHandler: h, BridgeAuthority: authority, OwnerAuthority: ownercommand.NewAuthority(bridgeOwnerToken, "apprun-test"), ReplacementDecisions: &decisionStore{}}), repo
}
func bridgeRequest(h http.Handler, method, path, body, token string, edit func(*http.Request)) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://127.0.0.1"+path, strings.NewReader(body))
	r.RemoteAddr = "127.0.0.1:42"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", token)
	if edit != nil {
		edit(r)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestBridgeRoutesNotMountedWhenFlagOff(t *testing.T) {
	for _, control := range []ControlDeps{{}, {BridgeHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("nil authority mounted") })}, {BridgeAuthority: ownercommand.NewBridgeAuthority(bridgeTestToken)}} {
		h := NewRouterWithControl(config.Config{}, discardLogger(), nil, APIDeps{}, control)
		for _, v := range []struct{ method, path string }{{"GET", "/internal/bridge/status"}, {"POST", "/internal/bridge/pair"}} {
			w := bridgeRequest(h, v.method, v.path, "{}", "KennelBridge "+bridgeTestToken, nil)
			if w.Code != 404 || !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
		}
	}
}
func TestBridgeRejectsAnyOriginHeader(t *testing.T) {
	h, _ := bridgeRouter(t, true)
	for _, origin := range []string{"null", "https://evil.test", "http://localhost:9999", ""} {
		w := bridgeRequest(h, "GET", "/internal/bridge/status", "", "KennelBridge "+bridgeTestToken, func(r *http.Request) { r.Header.Set("Origin", origin) })
		if w.Code != 404 {
			t.Fatalf("origin gate %d", w.Code)
		}
	}
}
func TestBridgeRejectsNonLoopbackHost(t *testing.T) {
	h, _ := bridgeRouter(t, true)
	for _, host := range []string{"example.com", "192.168.1.2", "127.0.0.1", "[::1]", "localhost"} {
		w := bridgeRequest(h, "GET", "/internal/bridge/status", "", "KennelBridge "+bridgeTestToken, func(r *http.Request) { r.Host = host })
		want := 404
		if host == "127.0.0.1" || host == "[::1]" || host == "localhost" {
			want = 200
		}
		if w.Code != want {
			t.Fatalf("host=%s code=%d", host, w.Code)
		}
	}
	w := bridgeRequest(h, "GET", "/internal/bridge/status", "", "KennelBridge "+bridgeTestToken, func(r *http.Request) { r.RemoteAddr = "192.168.1.2:42"; r.Header.Set("X-Forwarded-For", "127.0.0.1") })
	if w.Code != 404 {
		t.Fatal("proxy header bypassed peer gate")
	}
}
func TestBridgeRequiresBridgeToken(t *testing.T) {
	h, _ := bridgeRouter(t, true)
	for _, token := range []string{"", "KennelBridge wrong", "KennelOwner " + bridgeOwnerToken, "KennelOwner " + bridgeTestToken} {
		w := bridgeRequest(h, "GET", "/internal/bridge/status", "", token, nil)
		if w.Code != 401 {
			t.Fatal(w.Code)
		}
	}
	for _, token := range []string{"KennelBridge " + bridgeTestToken, "KennelOwner " + bridgeTestToken} {
		w := bridgeRequest(h, "POST", "/internal/owner-commands/attempt-replacement-decisions", "{}", token, nil)
		if w.Code != 401 {
			t.Fatal("bridge capability accepted by owner route")
		}
	}
}
func TestBridgeStatusStrictAndNoStore(t *testing.T) {
	h, _ := bridgeRouter(t, true)
	w := bridgeRequest(h, "GET", "/internal/bridge/status?x=1", "", "KennelBridge "+bridgeTestToken, nil)
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
	w = bridgeRequest(h, "GET", "/internal/bridge/status", "", "KennelBridge "+bridgeTestToken, nil)
	var fields map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &fields); e != nil {
		t.Fatal(e)
	}
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || len(fields) != 2 || fields["state"] != "unpaired" || fields["ready"] != true {
		t.Fatal(w.Body.String())
	}
}
func TestBridgePairStrictJSON(t *testing.T) {
	h, repo := bridgeRouter(t, true)
	good := `{"code":"` + bridgeTestToken + `","label":"Mac","capabilities":["notify_local"]}`
	for _, body := range []string{`{"code":"x","code":"y","label":"Mac","capabilities":[]}`, `{"code":"x","label":"Mac","capabilities":[],"owner":"other"}`, `{}`, `{"code":1,"label":"Mac","capabilities":[]}`, strings.Repeat("x", 2049), "\xff", good + `{}`} {
		w := bridgeRequest(h, "POST", "/internal/bridge/pair", body, "KennelBridge "+bridgeTestToken, nil)
		if w.Code != 400 {
			t.Fatalf("invalid shape: %d", w.Code)
		}
	}
	for _, kind := range []string{"media", "query"} {
		path := "/internal/bridge/pair"
		if kind == "query" {
			path += "?x=1"
		}
		w := bridgeRequest(h, "POST", path, good, "KennelBridge "+bridgeTestToken, func(r *http.Request) {
			if kind == "media" {
				r.Header.Set("Content-Type", "text/plain")
			}
		})
		if w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
	if repo.reserves != 0 || repo.pairs != 0 {
		t.Fatal("invalid request reached storage")
	}
}
func TestBridgePairNotReadyIs503BeforeReserve(t *testing.T) {
	h, repo := bridgeRouter(t, false)
	w := bridgeRequest(h, "POST", "/internal/bridge/pair", `{"code":"`+bridgeTestToken+`","label":"Mac","capabilities":["notify_local"]}`, "KennelBridge "+bridgeTestToken, nil)
	if w.Code != 503 || repo.reserves != 0 || repo.pairs != 0 {
		t.Fatal("not ready request reserved")
	}
}
func TestBridgePairConflictIs409(t *testing.T) {
	h, repo := bridgeRouter(t, true)
	repo.blocked = true
	w := bridgeRequest(h, "POST", "/internal/bridge/pair", `{"code":"`+bridgeTestToken+`","label":"Mac","capabilities":["notify_local"]}`, "KennelBridge "+bridgeTestToken, nil)
	if w.Code != 409 || repo.pairs != 0 {
		t.Fatal(w.Code)
	}
}

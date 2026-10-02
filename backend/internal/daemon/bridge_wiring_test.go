package daemon

import (
	"database/sql"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ownercommand"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/activationrepo"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const wiringToken = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

func wiringEnv(flag string) func(string) string {
	return func(k string) string {
		return map[string]string{"KENNEL_WALDO_BRIDGE_ENABLED": flag, "KENNEL_WALDO_ORIGIN": "https://example.test", "KENNEL_WALDO_OWNER_ID": "owner"}[k]
	}
}
func TestDaemonBridgeWiringOffOpensNothing(t *testing.T) {
	var opens int
	open := func(string) (*sql.DB, error) { opens++; return nil, nil }
	for _, flag := range []string{"", "0", "true"} {
		b := wireBridge(t.TempDir(), wiringEnv(flag), ownercommand.NewBridgeAuthority(wiringToken), slog.New(slog.NewTextHandler(io.Discard, nil)), open)
		if b != nil || opens != 0 {
			t.Fatal("off opened storage")
		}
	}
}
func TestDaemonBridgeWiringOnMountsNotReady(t *testing.T) {
	dir := t.TempDir()
	s, e := sqlite.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	b := wireBridge(dir, wiringEnv("1"), ownercommand.NewBridgeAuthority(wiringToken), slog.New(slog.NewTextHandler(io.Discard, nil)), activationrepo.OpenDedicated)
	if b == nil || b.controller == nil || len(b.dbs) != 2 {
		t.Fatal("not mounted")
	}
	for _, tc := range []struct {
		method, path, body string
		want               int
	}{{"GET", "/status", "", 200}, {"POST", "/pair", `{"code":"` + wiringToken + `","label":"Mac","capabilities":["notify_local"]}`, 503}} {
		r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		r.RemoteAddr = "127.0.0.1:42"
		r.Header.Set("Authorization", "KennelBridge "+wiringToken)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		b.handler.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatal(w.Code)
		}
		if tc.method == http.MethodGet && !strings.Contains(w.Body.String(), `"ready":false`) {
			t.Fatal("premature readiness")
		}
	}
	var reservations int
	if e = b.dbs[0].QueryRow(`SELECT count(*) FROM device_bridge_activations`).Scan(&reservations); e != nil || reservations != 0 {
		t.Fatal("not ready reserved")
	}
	b.close()
	for _, db := range b.dbs {
		if db.Ping() == nil {
			t.Fatal("dedicated DB not closed")
		}
	}
}

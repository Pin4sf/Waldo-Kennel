package bridgepersist

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"fmt"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
	"path/filepath"
	"testing"
	"time"
)

func TestGateBReceiptRollbackCapacityAndAudit(t *testing.T) {
	dir := t.TempDir()
	s, e := sqlite.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	_ = s.Close()
	db, e := sql.Open("sqlite", "file:"+filepath.Join(dir, "kennel.db")+"?_pragma=foreign_keys(ON)")
	if e != nil {
		t.Fatal(e)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	b := New(db)
	ctx := context.Background()
	scope := devicebridge.Scope{DeviceID: "dev", OwnerID: "owner"}
	now := time.Now().UTC().Truncate(time.Second)
	_, e = db.Exec(`INSERT INTO device_bridge_devices VALUES('dev','owner','Mac','public','custody','0.2.3','[]','paired',?,NULL,NULL,?,?)`, now, now, now)
	if e != nil {
		t.Fatal(e)
	}
	// Capacity rejects NEW acceptance without evicting live tombstones.
	_, e = db.Exec(`WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<100000) INSERT INTO device_bridge_receipts SELECT 'dev',printf('r%d',x),printf('c%d',x),1,printf('i%d',x),'fp',?,? FROM n`, now.Unix()+600, now.Unix())
	if e != nil {
		t.Fatal(e)
	}
	c := commandFixture(now)
	a, e := b.Admit(ctx, scope, c, now)
	if e != nil || a.Reason != devicebridge.ReasonInvalidShape || a.New {
		t.Fatalf("capacity %+v %v", a, e)
	}
	var n int
	_ = db.QueryRow(`SELECT count(*) FROM device_bridge_receipts`).Scan(&n)
	if n != 100000 {
		t.Fatal("live tombstones evicted")
	}
	// After expiry horizon pruning, the new command can be journaled.
	_, e = db.Exec(`UPDATE device_bridge_receipts SET expires_at=?`, now.Unix()-301)
	if e != nil {
		t.Fatal(e)
	}
	a, e = b.Admit(ctx, scope, c, now)
	if e != nil || !a.New {
		t.Fatalf("after horizon %+v %v", a, e)
	}
	// A failed outbox INSERT must roll back the fingerprint and journal outcome.
	_, e = db.Exec(`CREATE TRIGGER gateb_injected_outbox_failure BEFORE INSERT ON device_bridge_outbox BEGIN SELECT RAISE(ABORT,'injected'); END`)
	if e != nil {
		t.Fatal(e)
	}
	r := resultFixture(t, now)
	if e = b.CommitResult(ctx, scope, c, r); e == nil {
		t.Fatal("injected result insert succeeded")
	}
	var state string
	_ = db.QueryRow(`SELECT state FROM device_bridge_inbox_journal WHERE device_id='dev' AND command_id='cmd'`).Scan(&state)
	if state != "acked" {
		t.Fatal("journal partial commit")
	}
	_ = db.QueryRow(`SELECT count(*) FROM device_bridge_fingerprints WHERE message_id='01ARZ3NDEKTSV4RRFFQ69G5FAX'`).Scan(&n)
	if n != 0 {
		t.Fatal("fingerprint partial commit")
	}
	_, _ = db.Exec(`DROP TRIGGER gateb_injected_outbox_failure`)
	if e = b.CommitResult(ctx, scope, c, r); e != nil {
		t.Fatal(e)
	}
	// A failed delete rolls back both receipt tombstone and its dedup identity.
	_, e = db.Exec(`CREATE TRIGGER gateb_injected_delete_failure BEFORE DELETE ON device_bridge_outbox BEGIN SELECT RAISE(ABORT,'injected'); END`)
	if e != nil {
		t.Fatal(e)
	}
	receipt := []byte(`{"command_id":"cmd","contract_version":"0.2.3","device_id":"dev","idempotency_key":"cmd","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAY","owner_id":"owner","payload":{"received_at":1,"result_message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAX"},"revision":1,"type":"receipt"}`)
	if e = b.ApplyReceipt(ctx, scope, receipt, now); e == nil {
		t.Fatal("injected receipt delete succeeded")
	}
	_ = db.QueryRow(`SELECT count(*) FROM device_bridge_receipts`).Scan(&n)
	if n != 0 {
		t.Fatal("receipt partial commit")
	}
	if n, e := b.Depth(ctx, scope); e != nil || n != 1 {
		t.Fatal("outbox lost", n, e)
	}
	_, _ = db.Exec(`DROP TRIGGER gateb_injected_delete_failure`)
	if e = b.ApplyReceipt(ctx, scope, receipt, now); e != nil {
		t.Fatal(e)
	}
	if e = b.ClearPaired(ctx, scope); e != nil {
		t.Fatal(e)
	}
	_ = db.QueryRow(`SELECT count(*) FROM device_bridge_inbox_journal`).Scan(&n)
	if n != 1 {
		t.Fatal("audit lost")
	}
	if _, e = b.Depth(ctx, scope); e != domain.ErrDeviceBridgeInvalid {
		t.Fatal("revoked scope usable", e)
	}
}

func commandFixture(now time.Time) []byte {
	return []byte(fmt.Sprintf(`{"class":"machine_state_query","command_id":"cmd","contract_version":"0.2.3","device_id":"dev","expires_at":%d,"idempotency_key":"cmd","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAV","owner_id":"owner","payload":{"query_id":"q","query_kind":"session_status"},"revision":1,"type":"command"}`, now.Unix()+600))
}
func resultFixture(t *testing.T, now time.Time) []byte {
	t.Helper()
	r, e := devicebridge.SignFrame([]byte(`{"command_id":"cmd","contract_version":"0.2.3","device_id":"dev","idempotency_key":"cmd","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAX","owner_id":"owner","payload":{"answer":{"query_id":"q","query_kind":"session_status","state":"unknown"},"status":"answered"},"revision":1,"type":"result"}`), ed25519.NewKeyFromSeed(make([]byte, 32)), now, [devicebridge.NonceBytes]byte{})
	if e != nil {
		t.Fatal(e)
	}
	return r
}

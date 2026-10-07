package sqlite

import (
	"database/sql"
	"github.com/pressly/goose/v3"
	"path/filepath"
	"testing"
)

// A 0165 database whose owner row was revoked (the blocked acceptance state)
// keeps that row verbatim as immutable audit and can admit a new identity.
func TestMigration0166RevokedOwnerCanRepair(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, e := sql.Open("sqlite", "file:"+path+pragmas)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	upTo(t, db, 165)
	stamp := "2026-10-07 00:00:00"
	mustExec(t, db, `INSERT INTO device_bridge_devices VALUES('dev','owner','Mac','public','custody','0.2.3','["machine_state_query"]','revoked',?,?,NULL,?,?)`, stamp, stamp, stamp, stamp)
	mustExec(t, db, `INSERT INTO device_bridge_activations(owner_id,attempt,phase,checkpoint_device_id,device_id,revoked,claim_generation,created_at,updated_at) VALUES('owner','','revoked','audit','dev',1,3,?,?)`, stamp, stamp)
	mustExec(t, db, `INSERT INTO device_bridge_activations(owner_id,attempt,phase,created_at,updated_at) VALUES('pending-owner','token','pending',?,?)`, stamp, stamp)
	upTo(t, db, 166)
	var n int
	if e = db.QueryRow(`SELECT count(*) FROM device_bridge_activations WHERE owner_id='owner' AND attempt='' AND phase='revoked' AND checkpoint_device_id='audit' AND device_id='dev' AND revoked=1 AND claim_generation=3 AND created_at=? AND updated_at=?`, stamp, stamp).Scan(&n); e != nil || n != 1 {
		t.Fatalf("revoked row not copied: %d %v", n, e)
	}
	if e = db.QueryRow(`SELECT count(*) FROM device_bridge_activations WHERE owner_id='pending-owner' AND attempt='token' AND phase='pending'`).Scan(&n); e != nil || n != 1 {
		t.Fatalf("pending row not copied: %d %v", n, e)
	}
	for _, q := range []string{
		`UPDATE device_bridge_activations SET phase='pending',attempt='x',revoked=0,device_id=NULL WHERE owner_id='owner'`,
		`DELETE FROM device_bridge_activations WHERE owner_id='owner'`,
		`INSERT INTO device_bridge_activations(owner_id,attempt,phase,created_at,updated_at) VALUES('pending-owner','second','pending','2026-10-07','2026-10-07')`,
		`INSERT INTO device_bridge_activations(owner_id,phase,device_id,created_at,updated_at) VALUES('owner','paired','dev','2026-10-07','2026-10-07')`,
	} {
		if _, e = db.Exec(q); e == nil {
			t.Fatalf("accepted %s", q)
		}
	}
	mustExec(t, db, `INSERT INTO device_bridge_activations(owner_id,attempt,phase,created_at,updated_at) VALUES('owner','fresh','pending',?,?)`, stamp, stamp)
	gooseMu.Lock()
	goose.SetBaseFS(migrationsFS)
	_ = goose.SetDialect("sqlite3")
	e = goose.Down(db, "migrations")
	gooseMu.Unlock()
	if e == nil {
		t.Fatal("used database rolled back")
	}
	if e = db.QueryRow(`SELECT count(*) FROM device_bridge_activations WHERE owner_id='owner'`).Scan(&n); e != nil || n != 2 {
		t.Fatalf("rollback changed rows: %d %v", n, e)
	}
}

func TestMigration0166EmptyRollbackRestoresPerOwnerShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, e := sql.Open("sqlite", "file:"+path+pragmas)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	upTo(t, db, 166)
	gooseMu.Lock()
	goose.SetBaseFS(migrationsFS)
	_ = goose.SetDialect("sqlite3")
	e = goose.Down(db, "migrations")
	gooseMu.Unlock()
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, db, `INSERT INTO device_bridge_activations(owner_id,attempt,phase,created_at,updated_at) VALUES('owner','token','pending','2026-10-07','2026-10-07')`)
	if _, e = db.Exec(`DELETE FROM device_bridge_activations`); e == nil {
		t.Fatal("0165 delete trigger missing")
	}
	mustExec(t, db, `UPDATE device_bridge_activations SET attempt='other' WHERE owner_id='owner'`)
	if _, e = db.Exec(`INSERT INTO device_bridge_activations(owner_id,attempt,phase,created_at,updated_at) VALUES('owner','again','pending','2026-10-07','2026-10-07')`); e == nil {
		t.Fatal("0165 owner key missing")
	}
}

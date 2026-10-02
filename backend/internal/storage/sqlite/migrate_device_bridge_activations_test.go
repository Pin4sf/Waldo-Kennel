package sqlite

import (
	"database/sql"
	"github.com/pressly/goose/v3"
	"path/filepath"
	"testing"
)

func TestMigration0165CopyRollbackRestart(t *testing.T) {
	for _, used := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "used"}[used], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "test.db")
			db, e := sql.Open("sqlite", "file:"+path+pragmas)
			if e != nil {
				t.Fatal(e)
			}
			defer db.Close()
			upTo(t, db, 164)
			stamp := "2026-09-30 00:00:00"
			mustExec(t, db, `INSERT INTO device_bridge_devices VALUES('dev','owner','Mac','public','custody','0.2.3','["machine_state_query"]','paired',?,NULL,NULL,?,?)`, stamp, stamp, stamp)
			upTo(t, db, 165)
			var n int
			if e = db.QueryRow(`SELECT count(*) FROM device_bridge_devices WHERE device_id='dev' AND owner_id='owner' AND label='Mac' AND public_key='public' AND key_custody_ref='custody' AND contract_version='0.2.3' AND capability_classes='["machine_state_query"]' AND state='paired' AND paired_at=? AND created_at=? AND updated_at=? AND revoked_at IS NULL AND last_seen_at IS NULL`, stamp, stamp, stamp).Scan(&n); e != nil || n != 1 {
				t.Fatalf("seed changed: %d %v", n, e)
			}
			if used {
				mustExec(t, db, `INSERT INTO device_bridge_activations(owner_id,attempt,phase,created_at,updated_at) VALUES('pending','token','pending',?,?)`, stamp, stamp)
			}
			gooseMu.Lock()
			goose.SetBaseFS(migrationsFS)
			_ = goose.SetDialect("sqlite3")
			e = goose.Down(db, "migrations")
			gooseMu.Unlock()
			if (e == nil) == used {
				t.Fatalf("rollback used=%v: %v", used, e)
			}
			if !used {
				upTo(t, db, 165)
			}
			if e = db.Close(); e != nil {
				t.Fatal(e)
			}
			db, e = sql.Open("sqlite", "file:"+path+pragmas)
			if e != nil {
				t.Fatal(e)
			}
			defer db.Close()
			if e = db.QueryRow(`SELECT count(*) FROM device_bridge_activations`).Scan(&n); e != nil {
				t.Fatal(e)
			}
			if used && n != 1 || !used && n != 0 {
				t.Fatalf("restart count %d", n)
			}
		})
	}
}

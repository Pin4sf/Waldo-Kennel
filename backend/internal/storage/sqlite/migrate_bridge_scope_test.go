package sqlite

import (
	"database/sql"
	"github.com/pressly/goose/v3"
	"path/filepath"
	"testing"
)

func TestGateBMigrationCopyRollbackRestart(t *testing.T) {
	for _, kind := range []string{"empty", "bound", "unbound", "unknown", "malformed", "bound_wrong_owner", "collision", "deleted_highwater"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "test.db")
			db, e := sql.Open("sqlite", "file:"+path+pragmas)
			if e != nil {
				t.Fatal(e)
			}
			defer db.Close()
			upTo(t, db, 163)
			stamp := "2026-09-30 00:00:00"
			mustExec(t, db, `INSERT INTO device_bridge_devices VALUES('dev','owner','Mac','public','custody','0.2.3','[]','paired',?,NULL,NULL,?,?)`, stamp, stamp, stamp)
			payload := `{"device_id":"dev"}`
			if kind == "unbound" {
				payload = `{"status":"delivered"}`
			}
			if kind == "unknown" {
				payload = `{"device_id":"wrong"}`
			}
			if kind == "bound_wrong_owner" {
				payload = `{"device_id":"dev","owner_id":"other"}`
			}
			if kind == "malformed" {
				payload = `oops`
			}
			if kind != "empty" && kind != "deleted_highwater" {
				mustExec(t, db, `INSERT INTO device_bridge_outbox VALUES(42,'msg','cmd',1,'result',?,'pending',?,NULL,NULL)`, payload, stamp)
				mustExec(t, db, `INSERT INTO device_bridge_inbox_journal VALUES('cmd',1,'idem','fingerprint','notify_local',?,'acked','',?,?)`, payload, stamp, stamp)
			}
			if kind == "collision" {
				mustExec(t, db, `DROP INDEX device_bridge_outbox_message_id`)
				mustExec(t, db, `DROP INDEX device_bridge_outbox_command_revision`)
				mustExec(t, db, `INSERT INTO device_bridge_outbox VALUES(43,'msg','cmd',1,'result',?,'pending',?,NULL,NULL)`, payload, stamp)
			}

			if kind == "deleted_highwater" {
				mustExec(t, db, `INSERT INTO device_bridge_outbox VALUES(99,'deleted','deleted',1,'result','{}','pending',?,NULL,NULL)`, stamp)
				mustExec(t, db, `DELETE FROM device_bridge_outbox`)
			}
			gooseMu.Lock()
			goose.SetBaseFS(migrationsFS)
			goose.SetLogger(goose.NopLogger())
			_ = goose.SetDialect("sqlite3")
			e = goose.UpTo(db, "migrations", 164)
			gooseMu.Unlock()
			success := kind == "empty" || kind == "bound" || kind == "deleted_highwater"
			t.Logf("fixture=%s migration_succeeded=%v legacy_outbox_rows_rejected=%d legacy_journal_rows_rejected=%d", kind, e == nil, func() int {
				if success {
					return 0
				}
				if kind == "collision" {
					return 2
				}
				return 1
			}(), func() int {
				if success {
					return 0
				}
				return 1
			}())
			if (e == nil) != success {
				t.Fatalf("migration success=%v error=%v", success, e)
			}
			var n int
			if success {
				if kind == "deleted_highwater" {
					var cursor int
					if e = db.QueryRow(`SELECT seq FROM sqlite_sequence WHERE name='device_bridge_outbox'`).Scan(&cursor); e != nil || cursor != 99 {
						t.Fatalf("high-water lost: %d %v", cursor, e)
					}
				}

				if kind == "bound" {
					var id string
					var seq int
					var p string
					if e = db.QueryRow(`SELECT device_id,seq,payload FROM device_bridge_outbox`).Scan(&id, &seq, &p); e != nil || id != "dev" || seq != 42 || p != payload {
						t.Fatalf("copy: %s %d %s %v", id, seq, p, e)
					}
					var journalPayload, oldFP, cls, state, reason, created, updated string
					var revision int
					if e = db.QueryRow(`SELECT payload,payload_fingerprint,class,state,reject_reason,CAST(created_at AS TEXT),CAST(updated_at AS TEXT),revision FROM device_bridge_inbox_journal WHERE device_id='dev'`).Scan(&journalPayload, &oldFP, &cls, &state, &reason, &created, &updated, &revision); e != nil {
						t.Fatal(e)
					}
					if journalPayload != payload || oldFP != "fingerprint" || cls != "notify_local" || state != "acked" || reason != "" || created != stamp || updated != stamp || revision != 1 {
						t.Fatal("journal columns changed")
					}
					var message, command, outclass, outstate, outcreated string
					var outrev int
					var sent, receipted sql.NullString
					if e = db.QueryRow(`SELECT message_id,command_id,revision,class,state,CAST(created_at AS TEXT),CAST(sent_at AS TEXT),CAST(receipted_at AS TEXT) FROM device_bridge_outbox WHERE seq=42`).Scan(&message, &command, &outrev, &outclass, &outstate, &outcreated, &sent, &receipted); e != nil {
						t.Fatal(e)
					}
					if message != "msg" || command != "cmd" || outrev != 1 || outclass != "result" || outstate != "pending" || outcreated != stamp || sent.Valid || receipted.Valid {
						t.Fatal("outbox columns changed")
					}
					// Preserve high-water cursor even if the highest historical row was deleted.
					// Same identities on a different device are valid; same scope collides.
					mustExec(t, db, `INSERT INTO device_bridge_devices VALUES('dev2','owner2','Mac','public','custody','0.2.3','[]','paired',?,NULL,NULL,?,?)`, stamp, stamp, stamp)
					mustExec(t, db, `INSERT INTO device_bridge_outbox(device_id,message_id,command_id,revision,class,payload,state,created_at) VALUES('dev2','msg','cmd',1,'result','{}','pending',?)`, stamp)
					if _, e = db.Exec(`INSERT INTO device_bridge_outbox(device_id,message_id,command_id,revision,class,payload,state,created_at) VALUES('dev','msg','cmd',1,'result','{}','pending',?)`, stamp); e == nil {
						t.Fatal("same scope collision accepted")
					}
				}
			} else {
				if e = db.QueryRow(`SELECT count(*) FROM device_bridge_outbox WHERE seq=42 AND payload=?`, payload).Scan(&n); e != nil || n != 1 {
					t.Fatalf("rollback lost outbox: n=%d %v", n, e)
				}
				if e = db.QueryRow(`SELECT count(*) FROM pragma_table_info('device_bridge_outbox') WHERE name='device_id'`).Scan(&n); e != nil || n != 0 {
					t.Fatal("failed migration left scoped schema")
				}
			}
			if e = db.Close(); e != nil {
				t.Fatal(e)
			}
			db, e = sql.Open("sqlite", "file:"+path+pragmas)
			if e != nil {
				t.Fatal(e)
			}
			defer db.Close()
			if e = db.QueryRow(`SELECT count(*) FROM device_bridge_outbox`).Scan(&n); e != nil {
				t.Fatal(e)
			}
			want := 1
			if kind == "collision" {
				want = 2
			}
			if kind == "empty" || kind == "deleted_highwater" {
				want = 0
			}
			if kind == "bound" {
				want = 2
			}
			if n != want {
				t.Fatalf("restart count=%d want=%d", n, want)
			}
			if success {
				gooseMu.Lock()
				goose.SetBaseFS(migrationsFS)
				_ = goose.SetDialect("sqlite3")
				e = goose.Down(db, "migrations")
				gooseMu.Unlock()
				if (e == nil) != (kind == "empty" || kind == "deleted_highwater") {
					t.Fatalf("downgrade data gate: %v", e)
				}
			}

		})
	}
}

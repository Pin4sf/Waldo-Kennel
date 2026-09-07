package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// TestMigrateRepairsProjectChatProjectionAfterUpstreamVersionCollision models
// an Kennel-derived database that applied Kennel's own migration 0098 before Kennel
// opened it. Goose sees version 98 in the ledger and therefore skips Kennel's
// different 0098 migration; startup must reconcile the physical trigger seam
// instead of silently losing project-chat assistant projections.
func TestMigrateRepairsProjectChatProjectionAfterUpstreamVersionCollision(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "kennel-derived.db")+pragmas)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	upTo(t, db, 97)
	// Kennel's upstream 0098 adds this column and records the same Goose version
	// Kennel historically used for its project-chat projection triggers.
	mustExec(t, db, `ALTER TABLE sessions ADD COLUMN agent_session_id_launch_id TEXT NOT NULL DEFAULT ''`)
	mustExec(t, db, `INSERT INTO goose_db_version (version_id, is_applied) VALUES (98, 1)`)
	// Kennel's next migration may also be present when Kennel imports a newer
	// chassis database. Keep that out-of-order ledger fact intact as well.
	mustExec(t, db, `ALTER TABLE session_interface_transitions ADD COLUMN notice_acknowledged_at TIMESTAMP`)
	mustExec(t, db, `INSERT INTO goose_db_version (version_id, is_applied) VALUES (99, 1)`)

	if err := migrate(db); err != nil {
		t.Fatalf("migrate Kennel-derived database: %v", err)
	}

	for _, name := range []string{
		"conversation_assistant_insert_session_projection",
		"conversation_assistant_settle_session_projection",
	} {
		var count int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'trigger' AND name = ?`, name,
		).Scan(&count); err != nil {
			t.Fatalf("inspect trigger %s: %v", name, err)
		}
		if count != 1 {
			t.Fatalf("trigger %s count = %d, want 1", name, count)
		}
	}

	// Reconciliation is safe on every subsequent startup.
	if err := migrate(db); err != nil {
		t.Fatalf("repeat migrate Kennel-derived database: %v", err)
	}
}

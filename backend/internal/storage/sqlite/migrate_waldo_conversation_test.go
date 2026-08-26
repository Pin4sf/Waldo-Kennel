package sqlite

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestMigration0105CreatesProjectConversationReferencesWithoutProviderTranscriptCopies(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 105)
	seedContractProject(t, db)

	for _, table := range []string{
		"waldo_conversations", "waldo_conversation_episodes", "waldo_conversation_turns",
		"waldo_context_attachments", "waldo_turn_context_refs", "waldo_continuation_receipts",
	} {
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("table %s count=%d err=%v", table, count, err)
		}
	}

	rows, err := db.Query(`PRAGMA table_info(waldo_continuation_receipts)`)
	if err != nil {
		t.Fatalf("continuation receipt schema: %v", err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, kind string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan schema: %v", err)
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate receipt schema: %v", err)
	}
	joined := strings.Join(columns, ",")
	for _, forbidden := range []string{"transcript_content", "transcript_body", "provider_message", "provider_payload"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("continuation receipts copy provider transcript data through %q in %s", forbidden, joined)
		}
	}
}

func TestMigration0105DownRestoresPriorCDCVocabulary(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 105)
	if err := restoreChangeLogWriters(db); err != nil {
		t.Fatalf("restore CDC: %v", err)
	}
	seedContractProject(t, db)
	if _, err := db.Exec(`INSERT INTO waldo_conversations
		(id, project_id, revision, latest_turn_sequence, created_at, updated_at)
		VALUES ('conversation-down', 'p1', 0, 0, datetime('now'), datetime('now'))`); err != nil {
		t.Fatalf("seed conversation: %v", err)
	}

	gooseMu.Lock()
	defer gooseMu.Unlock()
	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set dialect: %v", err)
	}
	if err := goose.DownTo(db, "migrations", 104); err != nil {
		t.Fatalf("down to 104: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master
		WHERE type = 'table' AND name LIKE 'waldo_%'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("Waldo tables after down count=%d err=%v", count, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM change_log
		WHERE event_type LIKE 'waldo_conversation_%'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("Waldo CDC after down count=%d err=%v", count, err)
	}
	if _, err := db.Exec(`INSERT INTO change_log (project_id, event_type, payload)
		VALUES ('p1', 'intake_captured', json('{}'))`); err != nil {
		t.Fatalf("0104 CDC event rejected after down: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO change_log (project_id, event_type, payload)
		VALUES ('p1', 'waldo_conversation_created', json('{}'))`); err == nil {
		t.Fatal("0105 CDC event remained accepted after down")
	}
}

func TestMigration0105EnforcesOrderingBindingAppendOnlyHistoryAndTriggerCDC(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 105)
	if err := restoreChangeLogWriters(db); err != nil {
		t.Fatalf("restore CDC: %v", err)
	}
	seedContractProject(t, db)
	now := "2026-08-26T10:00:00Z"

	if _, err := db.Exec(`INSERT INTO waldo_conversations
		(id, project_id, revision, latest_turn_sequence, created_at, updated_at)
		VALUES ('conversation-1', 'p1', 0, 0, ?, ?)`, now, now); err != nil {
		t.Fatalf("insert conversation: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO waldo_conversation_episodes
		(id, conversation_id, project_id, ordinal, state, request_key, request_fingerprint, created_at)
		VALUES ('episode-1', 'conversation-1', 'p1', 1, 'active', 'episode-key', 'episode-fingerprint', ?)`, now); err != nil {
		t.Fatalf("insert episode: %v", err)
	}
	if _, err := db.Exec(`UPDATE waldo_conversations SET revision = 1, updated_at = ? WHERE id = 'conversation-1'`, now); err != nil {
		t.Fatalf("advance conversation for episode: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO waldo_conversation_turns
		(id, conversation_id, episode_id, project_id, sequence, role, message, request_key, request_fingerprint, created_at)
		VALUES ('turn-1', 'conversation-1', 'episode-1', 'p1', 1, 'user', 'Current intent', 'turn-key-1', 'turn-fingerprint-1', ?)`, now); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	if _, err := db.Exec(`UPDATE waldo_conversations
		SET revision = 2, latest_turn_sequence = 1, updated_at = ? WHERE id = 'conversation-1'`, now); err != nil {
		t.Fatalf("advance conversation for turn: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO waldo_conversation_turns
		(id, conversation_id, episode_id, project_id, sequence, role, message, request_key, request_fingerprint, created_at)
		VALUES ('turn-duplicate', 'conversation-1', 'episode-1', 'p1', 1, 'user', 'Duplicate order', 'turn-key-2', 'turn-fingerprint-2', ?)`, now); err == nil {
		t.Fatal("duplicate conversation sequence must fail")
	}
	if _, err := db.Exec(`INSERT INTO waldo_conversation_turns
		(id, conversation_id, episode_id, project_id, sequence, role, message, request_key, request_fingerprint, created_at)
		VALUES ('turn-skipped', 'conversation-1', 'episode-1', 'p1', 3, 'user', 'Skipped order', 'turn-key-3', 'turn-fingerprint-3', ?)`, now); err == nil {
		t.Fatal("skipped conversation sequence must fail")
	}
	if _, err := db.Exec(`UPDATE waldo_conversation_turns SET message = 'rewritten' WHERE id = 'turn-1'`); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("turn rewrite error = %v", err)
	}
	if _, err := db.Exec(`INSERT INTO waldo_conversation_episodes
		(id, conversation_id, project_id, ordinal, state, request_key, request_fingerprint, created_at)
		VALUES ('episode-wrong-project', 'conversation-1', 'missing-project', 2, 'active', 'episode-key-2', 'episode-fingerprint-2', ?)`, now); err == nil {
		t.Fatal("episode with mismatched Project binding must fail")
	}

	if _, err := db.Exec(`INSERT INTO waldo_context_attachments
		(id, conversation_id, project_id, kind, object_id, object_revision, provenance_kind,
		 provenance_ref, attached_revision, attach_request_key, attach_request_fingerprint, created_at)
		VALUES ('attachment-1', 'conversation-1', 'p1', 'project', 'p1', '', 'user',
		 'turn-1', 3, 'attach-key', 'attach-fingerprint', ?)`, now); err != nil {
		t.Fatalf("attach context: %v", err)
	}
	if _, err := db.Exec(`UPDATE waldo_conversations SET revision = 3, updated_at = ? WHERE id = 'conversation-1'`, now); err != nil {
		t.Fatalf("advance conversation for context attach: %v", err)
	}
	if _, err := db.Exec(`UPDATE waldo_context_attachments
		SET detached_revision = 4, detached_at = ?, detach_reason = 'source revoked',
		    detach_request_key = 'detach-key', detach_request_fingerprint = 'detach-fingerprint'
		WHERE id = 'attachment-1'`, now); err != nil {
		t.Fatalf("detach context: %v", err)
	}
	if _, err := db.Exec(`UPDATE waldo_conversations SET revision = 4, updated_at = ? WHERE id = 'conversation-1'`, now); err != nil {
		t.Fatalf("advance conversation for context detach: %v", err)
	}
	if _, err := db.Exec(`UPDATE waldo_context_attachments SET detach_reason = 'rewritten' WHERE id = 'attachment-1'`); err == nil || !strings.Contains(err.Error(), "one-time") {
		t.Fatalf("second detach error = %v", err)
	}

	for _, eventType := range []string{
		"waldo_conversation_created", "waldo_conversation_episode_opened",
		"waldo_conversation_turn_appended", "waldo_conversation_context_attached",
		"waldo_conversation_context_detached",
	} {
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM change_log WHERE event_type = ?`, eventType).Scan(&count); err != nil && err != sql.ErrNoRows {
			t.Fatalf("count %s: %v", eventType, err)
		}
		if count != 1 {
			t.Fatalf("%s count = %d, want 1", eventType, count)
		}
	}
}

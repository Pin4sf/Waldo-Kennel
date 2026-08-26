package sqlite

import (
	"database/sql"
	"strings"
	"testing"
)

func TestMigration0104CreatesSharedIntakeWithoutTranscriptCopies(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 104)
	seedContractProject(t, db)

	rows, err := db.Query(`PRAGMA table_info(intake_conversation_refs)`)
	if err != nil {
		t.Fatalf("conversation ref schema: %v", err)
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
		t.Fatalf("iterate schema: %v", err)
	}
	joined := strings.Join(columns, ",")
	for _, forbidden := range []string{"content", "message", "transcript", "body"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("conversation refs copy %q in columns %s", forbidden, joined)
		}
	}

	if _, err := db.Exec(`INSERT INTO intake_sessions
		(id, source_surface, purpose, project_id, statement, status, request_key, request_fingerprint)
		VALUES ('intake_1', 'work', 'outcome', 'p1', 'Make restart durable', 'captured', 'capture-1', 'fingerprint-1')`); err != nil {
		t.Fatalf("insert intake: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO intake_conversation_refs (intake_id, episode_id, turn_id, position)
		VALUES ('intake_1', 'episode-1', 'turn-7', 1)`); err != nil {
		t.Fatalf("insert ref: %v", err)
	}
	if _, err := db.Exec(`UPDATE intake_sessions SET statement = 'rewritten', updated_at = datetime('now') WHERE id = 'intake_1'`); err == nil || !strings.Contains(err.Error(), "intent is immutable") {
		t.Fatalf("statement rewrite error = %v", err)
	}
	if _, err := db.Exec(`DELETE FROM intake_conversation_refs WHERE intake_id = 'intake_1'`); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("ref delete error = %v", err)
	}
}

func TestMigration0104ProposalLinkAndCDCGuards(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 104)
	if err := restoreChangeLogWriters(db); err != nil {
		t.Fatalf("restore CDC: %v", err)
	}
	var confirmationTrigger string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'trigger' AND name = 'intake_confirmations_cdc_insert'`).Scan(&confirmationTrigger); err != nil {
		t.Fatalf("read intake confirmation CDC trigger: %v", err)
	}
	if !strings.Contains(confirmationTrigger, "project_id") || !strings.Contains(strings.ToUpper(confirmationTrigger), "WHEN") {
		t.Fatalf("intake confirmation CDC trigger lacks project guard: %s", confirmationTrigger)
	}
	seedContractProject(t, db)
	spaceID := seedWorkSpace(t, db)
	if _, err := db.Exec(`INSERT INTO outcomes (id, space_id, title, current_revision_number) VALUES ('out_link', ?, 'Outcome', 0)`, spaceID); err != nil {
		t.Fatalf("seed outcome: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO intake_sessions
		(id, source_surface, purpose, project_id, source_open_loop_id, statement, status, request_key, request_fingerprint)
		VALUES ('intake_link', 'home', 'outcome', 'p1', 'loop-1', 'Handle this in Work', 'captured', 'capture-link', 'fingerprint-link')`); err != nil {
		t.Fatalf("insert intake: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO intake_proposal_revisions
		(id, intake_id, revision, title, desired_state, criteria, review_method, constraints, non_goals, authority_ceiling, stop_conditions, clarification_notes, facets)
		VALUES ('proposal-1', 'intake_link', 1, 'Outcome', 'Desired state', json('[{"id":"pc-1","text":"Criterion","evidenceExpected":["Check"]}]'), 'Review', json('[]'), json('[]'), json('{}'), json('["Stop"]'), json('[]'), json('[{"kind":"software","summary":"UI"}]'))`); err != nil {
		t.Fatalf("insert proposal: %v", err)
	}
	if _, err := db.Exec(`UPDATE intake_proposal_revisions SET desired_state = 'rewritten' WHERE id = 'proposal-1'`); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("proposal update error = %v", err)
	}

	if _, err := db.Exec(`INSERT INTO responsibility_links
		(id, project_id, source_open_loop_id, destination_outcome_id, creator, reason, request_key, request_fingerprint)
		VALUES ('link-1', 'p1', 'loop-1', 'out_link', 'owner', 'Preserve lineage', 'link-key-1', 'fingerprint-link-1')`); err != nil {
		t.Fatalf("insert link: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO responsibility_links
		(id, project_id, source_open_loop_id, destination_outcome_id, creator, reason, request_key, request_fingerprint)
		VALUES ('link-2', 'p1', 'loop-1', 'out_link', 'owner', 'Duplicate', 'link-key-2', 'fingerprint-link-2')`); err == nil {
		t.Fatal("duplicate active lineage must fail")
	}
	if _, err := db.Exec(`UPDATE responsibility_links SET ended_at = datetime('now'), ended_by = 'owner', ended_reason = 'No longer needed' WHERE id = 'link-1'`); err != nil {
		t.Fatalf("end link: %v", err)
	}
	if _, err := db.Exec(`UPDATE responsibility_links SET reason = 'rewritten' WHERE id = 'link-1'`); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("link rewrite error = %v", err)
	}

	for _, eventType := range []string{"intake_captured", "intake_proposal_revised", "responsibility_link_created", "responsibility_link_ended"} {
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM change_log WHERE event_type = ?`, eventType).Scan(&count); err != nil && err != sql.ErrNoRows {
			t.Fatalf("count %s: %v", eventType, err)
		}
		if count != 1 {
			t.Fatalf("%s count = %d, want 1", eventType, count)
		}
	}
}

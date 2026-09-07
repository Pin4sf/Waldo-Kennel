package sqlite

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
)

// openContractTestDB opens a database with production parity: WAL and foreign
// keys enabled, unlike the bare openTestDB harness used by older migrations.
func openContractTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "kennel.db") +
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func downFrom99(t *testing.T, db *sql.DB) {
	t.Helper()
	gooseMu.Lock()
	defer gooseMu.Unlock()
	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set dialect: %v", err)
	}
	if err := goose.DownTo(db, "migrations", 98); err != nil {
		t.Fatalf("down to 98: %v", err)
	}
}

func seedContractProject(t *testing.T, db *sql.DB) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO projects (id, path, display_name, registered_at)
		VALUES ('p1', '/tmp/p1', 'proj', ?)`, now); err != nil {
		t.Fatalf("seed project: %v", err)
	}
}

func seedWorkSpace(t *testing.T, db *sql.DB) string {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO responsibility_spaces (id, kind, project_id)
		VALUES ('rsp_1', 'WorkProject', 'p1')`); err != nil {
		t.Fatalf("seed space: %v", err)
	}
	return "rsp_1"
}

func TestMigration0099OneWorkSpacePerProject(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 99)
	seedContractProject(t, db)

	spaceID := seedWorkSpace(t, db)
	if _, err := db.Exec(`INSERT INTO responsibility_spaces (id, kind, project_id)
		VALUES ('rsp_2', 'WorkProject', 'p1')`); err == nil {
		t.Fatal("second WorkProject space for one project must be rejected")
	}
	if spaceID == "" {
		t.Fatal("space seed failed")
	}
}

func TestMigration0099OutcomeRequiresExistingSpace(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 99)
	seedContractProject(t, db)
	seedWorkSpace(t, db)

	if _, err := db.Exec(`INSERT INTO outcomes (id, space_id, title)
		VALUES ('out_1', 'rsp_missing', 'No space')`); err == nil {
		t.Fatal("outcome referencing a missing space must violate the foreign key")
	}
}

func TestMigration0099ContractRevisionsAreImmutable(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 99)
	seedContractProject(t, db)
	seedWorkSpace(t, db)

	if _, err := db.Exec(`INSERT INTO outcomes (id, space_id, title)
		VALUES ('out_1', 'rsp_1', 'Local Focus Ledger')`); err != nil {
		t.Fatalf("seed outcome: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO contract_revisions (id, outcome_id, number, goal, success_criteria, review)
		VALUES ('cr_1', 'out_1', 1, 'Goal.', json('["c1"]'), 'Checks.')`); err != nil {
		t.Fatalf("seed revision: %v", err)
	}

	if _, err := db.Exec(`UPDATE contract_revisions SET goal = 'Rewritten' WHERE id = 'cr_1'`); err == nil ||
		!strings.Contains(err.Error(), "contract revisions are immutable") {
		t.Fatalf("revision UPDATE = %v, want immutability abort", err)
	}
	if _, err := db.Exec(`DELETE FROM contract_revisions WHERE id = 'cr_1'`); err == nil ||
		!strings.Contains(err.Error(), "contract revisions are immutable") {
		t.Fatalf("revision DELETE = %v, want immutability abort", err)
	}
}

func TestMigration0099RevisionNumbersUniquePerOutcome(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 99)
	seedContractProject(t, db)
	seedWorkSpace(t, db)
	if _, err := db.Exec(`INSERT INTO outcomes (id, space_id, title)
		VALUES ('out_1', 'rsp_1', 'Local Focus Ledger')`); err != nil {
		t.Fatalf("seed outcome: %v", err)
	}

	insertRevision := func(id string, number int64) error {
		_, err := db.Exec(`INSERT INTO contract_revisions (id, outcome_id, number, goal, success_criteria, review)
			VALUES (?, 'out_1', ?, 'Goal.', json('["c1"]'), 'Checks.')`, id, number)
		return err
	}
	if err := insertRevision("cr_1", 1); err != nil {
		t.Fatalf("insert revision 1: %v", err)
	}
	if err := insertRevision("cr_dup", 1); err == nil {
		t.Fatal("duplicate revision number for one outcome must be rejected")
	}
	if err := insertRevision("cr_other", 2); err != nil {
		t.Fatalf("next revision number must be accepted: %v", err)
	}
}

func TestMigration0099OutcomeCDCTriggers(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 99)
	seedContractProject(t, db)
	spaceID := seedWorkSpace(t, db)

	countEvents := func(eventType string) int {
		t.Helper()
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM change_log WHERE event_type = ?`, eventType).Scan(&n); err != nil {
			t.Fatalf("count %s events: %v", eventType, err)
		}
		return n
	}

	if _, err := db.Exec(`INSERT INTO outcomes (id, space_id, title, current_revision_number)
		VALUES ('out_1', ?, 'Local Focus Ledger', 0)`, spaceID); err != nil {
		t.Fatalf("insert outcome: %v", err)
	}
	if countEvents("outcome_created") != 1 {
		t.Fatalf("outcome_created events = %d, want 1", countEvents("outcome_created"))
	}
	var projectID string
	var rawPayload string
	if err := db.QueryRow(`SELECT project_id, payload FROM change_log WHERE event_type = 'outcome_created'`).
		Scan(&projectID, &rawPayload); err != nil {
		t.Fatalf("read outcome_created event: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
		t.Fatalf("decode payload %q: %v", rawPayload, err)
	}
	if projectID != "p1" {
		t.Fatalf("event project_id = %q, want p1 (resolved through the space)", projectID)
	}
	if payload["id"] != "out_1" || payload["title"] != "Local Focus Ledger" {
		t.Fatalf("unexpected payload %#v", payload)
	}

	// Material update emits exactly one outcome_updated.
	if _, err := db.Exec(`UPDATE outcomes SET title = 'Renamed Ledger', updated_at = datetime('now')
		WHERE id = 'out_1'`); err != nil {
		t.Fatalf("rename outcome: %v", err)
	}
	if countEvents("outcome_updated") != 1 {
		t.Fatalf("outcome_updated events = %d, want 1", countEvents("outcome_updated"))
	}

	// Non-material timestamp touch emits nothing.
	before := countEvents("outcome_updated")
	if _, err := db.Exec(`UPDATE outcomes SET updated_at = datetime('now') WHERE id = 'out_1'`); err != nil {
		t.Fatalf("touch outcome: %v", err)
	}
	if countEvents("outcome_updated") != before {
		t.Fatal("timestamp-only update must not emit CDC")
	}

	// Contract revision insert emits the revision event.
	rawCriteria, err := json.Marshal([]string{"Entering a duration creates one focus block."})
	if err != nil {
		t.Fatalf("marshal criteria: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO contract_revisions (id, outcome_id, number, goal, success_criteria, review, constraints, non_goals)
		VALUES ('cr_1', 'out_1', 1, 'Record focus locally.', ?, 'Deterministic checks.', json('[]'), json('[]'))`,
		string(rawCriteria)); err != nil {
		t.Fatalf("insert revision: %v", err)
	}
	if countEvents("outcome_contract_revised") != 1 {
		t.Fatalf("outcome_contract_revised events = %d, want 1", countEvents("outcome_contract_revised"))
	}
}

func TestMigration0099PreservesInheritedChangeEventTypes(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 99)
	seedContractProject(t, db)

	// Inherited event types survive the CHECK rebuild.
	if _, err := db.Exec(`INSERT INTO change_log (project_id, event_type, payload)
		VALUES ('p1', 'session_created', json('{"id":"kennel-1"}'))`); err != nil {
		t.Fatalf("inherited event type rejected after rebuild: %v", err)
	}
	// Unknown types stay rejected.
	if _, err := db.Exec(`INSERT INTO change_log (project_id, event_type, payload)
		VALUES ('p1', 'made_up_event', json('{}'))`); err == nil {
		t.Fatal("unknown event type must be rejected by CHECK")
	}
}

func TestMigration0099DownRestoresPriorChangeLogShape(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 99)
	seedContractProject(t, db)
	spaceID := seedWorkSpace(t, db)
	if _, err := db.Exec(`INSERT INTO outcomes (id, space_id, title)
		VALUES ('out_1', ?, 'Ledger')`, spaceID); err != nil {
		t.Fatalf("insert outcome: %v", err)
	}

	downFrom99(t, db)

	var n int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name IN ('outcomes','contract_revisions','responsibility_spaces')`).Scan(&n); err != nil {
		t.Fatalf("count contract tables: %v", err)
	}
	if n != 0 {
		t.Fatalf("contract tables remain after Down: %d", n)
	}
	// Outcome rows are gone from the log; inherited types still accepted.
	if err := db.QueryRow(`SELECT count(*) FROM change_log WHERE event_type LIKE 'outcome%'`).Scan(&n); err != nil {
		t.Fatalf("count outcome events: %v", err)
	}
	if n != 0 {
		t.Fatalf("outcome events survive Down: %d", n)
	}
	if _, err := db.Exec(`INSERT INTO change_log (project_id, event_type, payload)
		VALUES ('p1', 'session_created', json('{"id":"kennel-1"}'))`); err != nil {
		t.Fatalf("inherited event type rejected after Down: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO change_log (project_id, event_type, payload)
		VALUES ('p1', 'outcome_created', json('{}'))`); err == nil {
		t.Fatal("outcome event type must be rejected after Down")
	}
}

package sqlite

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Migration 0106 must upgrade a populated pre-composition database in place.
// Existing Outcomes become the direct case — parent NULL — and nothing about
// their behavior changes.
func TestMigration0106UpgradesExistingOutcomesToTheDirectCase(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 105)
	seedContractProject(t, db)
	spaceID := seedWorkSpace(t, db)
	if _, err := db.Exec(`INSERT INTO outcomes (id, space_id, title, current_revision_number)
		VALUES ('out_pre', ?, 'Predates composition', 1)`, spaceID); err != nil {
		t.Fatalf("seed pre-composition outcome: %v", err)
	}

	upTo(t, db, 106)
	if err := reconcileComposedOutcomesSchema(db); err != nil {
		t.Fatalf("reconcile composed outcomes schema: %v", err)
	}
	if err := restoreChangeLogWriters(db); err != nil {
		t.Fatalf("restore change log writers: %v", err)
	}

	var parent *string
	if err := db.QueryRow(`SELECT parent_outcome_id FROM outcomes WHERE id = 'out_pre'`).Scan(&parent); err != nil {
		t.Fatalf("read migrated parent: %v", err)
	}
	if parent != nil {
		t.Fatalf("an outcome that predates composition must have no parent, got %q", *parent)
	}

	// The new event type is admitted by the rebuilt vocabulary...
	if _, err := db.Exec(`INSERT INTO change_log (project_id, event_type, payload)
		VALUES ('p1', 'outcome_contribution_bound', '{}')`); err != nil {
		t.Fatalf("rebuilt change_log must admit the binding event: %v", err)
	}
	// ...and the rebuild did not widen it into accepting anything at all.
	if _, err := db.Exec(`INSERT INTO change_log (project_id, event_type, payload)
		VALUES ('p1', 'not_a_real_event', '{}')`); err == nil {
		t.Fatal("rebuilt change_log must still reject unknown event types")
	}
}

// The depth cap and the binding guards are storage-level invariants, so they
// must hold against raw SQL that never passes through the service.
func TestMigration0106EnforcesCompositionInvariantsInSQL(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 106)
	if err := reconcileComposedOutcomesSchema(db); err != nil {
		t.Fatalf("reconcile composed outcomes schema: %v", err)
	}
	if err := restoreChangeLogWriters(db); err != nil {
		t.Fatalf("restore change log writers: %v", err)
	}
	seedContractProject(t, db)
	spaceID := seedWorkSpace(t, db)

	for _, id := range []string{"out_parent", "out_child"} {
		if _, err := db.Exec(`INSERT INTO outcomes (id, space_id, title, current_revision_number)
			VALUES (?, ?, ?, 1)`, id, spaceID, id); err != nil {
			t.Fatalf("seed %s: %v", id, err)
		}
	}
	if _, err := db.Exec(`UPDATE outcomes SET parent_outcome_id = 'out_parent' WHERE id = 'out_child'`); err != nil {
		t.Fatalf("bind child to parent: %v", err)
	}

	// A third level is refused.
	_, err := db.Exec(`INSERT INTO outcomes (id, space_id, title, current_revision_number, parent_outcome_id)
		VALUES ('out_grandchild', ?, 'third level', 1, 'out_child')`, spaceID)
	if err == nil {
		t.Fatal("a third composition level must be refused by SQL")
	}
	if !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("refusal must name the depth limit, got %v", err)
	}

	// And the same cap read from the other direction: a parent cannot become
	// a child.
	if _, err := db.Exec(`UPDATE outcomes SET parent_outcome_id = 'out_parent' WHERE id = 'out_parent'`); err == nil {
		t.Fatal("an outcome with contributors must not become a contributor")
	}
}

// A burned 0099 ledger entry marks the version applied while leaving
// `outcomes` physically absent. The seam must defer rather than wedge daemon
// startup, and must heal once the table arrives through a repair.
func TestComposedOutcomesSchemaDefersOnDegradedProfile(t *testing.T) {
	db := openContractTestDB(t)
	if _, err := db.Exec(`CREATE TABLE goose_db_version (
		id INTEGER PRIMARY KEY AUTOINCREMENT, version_id INTEGER NOT NULL,
		is_applied INTEGER NOT NULL, tstamp TIMESTAMP DEFAULT (datetime('now')))`); err != nil {
		t.Fatalf("seed goose table: %v", err)
	}

	// No outcomes table at all: the seam must be a no-op, not an error.
	if err := reconcileComposedOutcomesSchema(db); err != nil {
		t.Fatalf("degraded profile must defer, not fail: %v", err)
	}
	var present int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='contribution_links'`).Scan(&present); err != nil {
		t.Fatalf("inspect schema: %v", err)
	}
	if present != 0 {
		t.Fatal("a degraded profile must not get composition tables it cannot reference")
	}

	// Running it twice on a complete profile is also a no-op, so a repaired
	// database heals on the next start without duplicate-object errors.
	full := openContractTestDB(t)
	upTo(t, full, 106)
	if err := reconcileComposedOutcomesSchema(full); err != nil {
		t.Fatalf("first reconcile: %v", err)
	}
	if err := reconcileComposedOutcomesSchema(full); err != nil {
		t.Fatalf("reconcile must be idempotent: %v", err)
	}
}

// The composition DDL is shared verbatim between sqlc and the startup seam, so
// generated code cannot describe relations the database does not have. This
// asserts the sharing actually holds: every relation the schema file declares
// exists after a real migrate, and the sqlc-only column file matches what the
// seam adds in Go.
func TestComposedOutcomesSchemaMatchesTheSeam(t *testing.T) {
	declared := declaredRelations(t)
	if len(declared) < 5 {
		t.Fatalf("expected the schema file to declare the composition relations, found %v", declared)
	}

	db := openContractTestDB(t)
	upTo(t, db, 107)
	if err := reconcileComposedOutcomesSchema(db); err != nil {
		t.Fatalf("reconcile composed outcomes schema: %v", err)
	}
	for _, name := range declared {
		var present int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, name,
		).Scan(&present); err != nil {
			t.Fatalf("inspect %s: %v", name, err)
		}
		if present == 0 {
			t.Fatalf("schema declares %q for sqlc but the seam never creates it", name)
		}
	}

	// The ALTER lives in a sqlc-only file because SQLite cannot guard it; the
	// seam performs it in Go. Both must name the same column.
	columns, err := os.ReadFile(filepath.Join("schema", "composed_outcomes_columns.sql"))
	if err != nil {
		t.Fatalf("read sqlc-only column file: %v", err)
	}
	if !strings.Contains(string(columns), "parent_outcome_id") {
		t.Fatal("the sqlc-only column file must declare parent_outcome_id")
	}
	var hasParent int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('outcomes') WHERE name = 'parent_outcome_id'`,
	).Scan(&hasParent); err != nil {
		t.Fatalf("inspect outcomes columns: %v", err)
	}
	if hasParent != 1 {
		t.Fatal("the seam must add the column the sqlc-only file declares")
	}
}

// declaredRelations extracts every table name the shared schema file creates.
func declaredRelations(t *testing.T) []string {
	t.Helper()
	matches := regexp.MustCompile(`(?i)CREATE TABLE IF NOT EXISTS\s+(\w+)`).FindAllStringSubmatch(composedOutcomesDDL, -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, match[1])
	}
	return names
}

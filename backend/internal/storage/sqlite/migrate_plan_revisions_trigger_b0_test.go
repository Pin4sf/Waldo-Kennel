package sqlite

// Connect to Waldo, Slice B0: regression test for the plan_revisions
// immutability trigger across migration 0164.
//
// Background: migration 0157 recreates plan_revisions_immutable_update with a
// body that names routing_decisions_json, a column that is NOT added by any
// goose migration. It is added by reconcileExecutionRoutingSchema, which runs
// after goose.Up inside Open(). So a database that has only been migrated with
// goose (version 163 or 164) carries a trigger that references a missing
// column until Open() heals it.
//
// This test pins three things, using seeded plan_revisions rows:
//  1. the premise (latent bug at goose 163/164, kept in its own test so the
//     heal test cannot be confused by it);
//  2. the real Open() path heals the trigger and keeps every seeded row intact;
//  3. healing is idempotent across a second Open().
//
// All databases live in t.TempDir(). Nothing touches a real Kennel database.

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

const b0Digest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

type b0PlanRow struct {
	ID, OutcomeID, Status, Summary, Digest, CreatedAt string
	Number, ContractRev                               int
}

// b0RawDB opens <dir>/kennel.db the same way Open() does, without migrating.
func b0RawDB(t *testing.T, dir string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "kennel.db")+pragmas)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	return db
}

// b0SeedAt163 builds a scratch database at goose version 163 holding two
// plan_revisions rows (one proposed, one approved) under one Outcome.
func b0SeedAt163(t *testing.T, dir string) {
	t.Helper()
	db := b0RawDB(t, dir)
	defer db.Close()
	upTo(t, db, 163)
	stamp := "2026-09-30T00:00:00Z"
	for _, s := range []string{
		`INSERT INTO projects(id,path,display_name,registered_at,archived_at,kind) VALUES('p','/tmp/b0','B0','` + stamp + `',NULL,'single_repo')`,
		`INSERT INTO responsibility_spaces(id,kind,project_id,created_at) VALUES('rsp','WorkProject','p','` + stamp + `')`,
		`INSERT INTO outcomes(id,space_id,title,current_revision_number,created_at,updated_at) VALUES('out','rsp','Outcome',1,'` + stamp + `','` + stamp + `')`,
		`INSERT INTO contract_revisions(id,outcome_id,number,goal,success_criteria,review,constraints,non_goals,clarification,created_at) VALUES('cr','out',1,'g','["c"]','r','[]','[]','','` + stamp + `')`,
		`INSERT INTO plan_revisions(id,outcome_id,number,contract_revision_number,status,summary,run_brief_core_digest,created_at) VALUES('plan-1','out',1,1,'proposed','first','` + b0Digest + `','` + stamp + `')`,
		`INSERT INTO plan_revisions(id,outcome_id,number,contract_revision_number,status,summary,run_brief_core_digest,created_at) VALUES('plan-2','out',2,1,'approved','second','` + b0Digest + `','` + stamp + `')`,
	} {
		mustExec(t, db, s)
	}
}

func b0GooseUpTo164(t *testing.T, db *sql.DB) {
	t.Helper()
	gooseMu.Lock()
	defer gooseMu.Unlock()
	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(db, "migrations", 164); err != nil {
		t.Fatalf("goose.UpTo(164) with seeded plan_revisions rows failed: %v", err)
	}
}

func b0Rows(t *testing.T, db *sql.DB) []b0PlanRow {
	t.Helper()
	rows, err := db.Query(`SELECT id,outcome_id,number,contract_revision_number,status,summary,run_brief_core_digest,created_at FROM plan_revisions ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []b0PlanRow
	for rows.Next() {
		var r b0PlanRow
		if err := rows.Scan(&r.ID, &r.OutcomeID, &r.Number, &r.ContractRev, &r.Status, &r.Summary, &r.Digest, &r.CreatedAt); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func b0HasRoutingColumn(t *testing.T, db *sql.DB) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('plan_revisions') WHERE name='routing_decisions_json'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n > 0
}

func b0TriggerSQL(t *testing.T, db *sql.DB) string {
	t.Helper()
	var s string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='trigger' AND name='plan_revisions_immutable_update'`).Scan(&s); err != nil {
		t.Fatalf("plan_revisions_immutable_update trigger is missing: %v", err)
	}
	return s
}

// TestB0PremiseLatentTriggerBugAtGoose164 records the latent state: after
// goose reaches 164 and before Open() runs its reconcile steps, the trigger
// names a column plan_revisions does not have, so a legitimate UPDATE fails.
// If this test starts failing, the bug was fixed upstream or the premise is
// wrong on this machine; report that, do not "fix" the test silently.
func TestB0PremiseLatentTriggerBugAtGoose164(t *testing.T) {
	dir := t.TempDir()
	b0SeedAt163(t, dir)
	db := b0RawDB(t, dir)
	defer db.Close()

	_, err163 := db.Exec(`UPDATE plan_revisions SET status='approved' WHERE id='plan-1'`)
	t.Logf("goose=163 UPDATE plan_revisions err=%v routing_column_present=%v", err163, b0HasRoutingColumn(t, db))

	b0GooseUpTo164(t, db)
	if b0HasRoutingColumn(t, db) {
		t.Fatalf("premise changed: goose alone now installs routing_decisions_json")
	}
	if !strings.Contains(b0TriggerSQL(t, db), "routing_decisions_json") {
		t.Fatalf("premise changed: trigger no longer references routing_decisions_json at goose 164")
	}
	_, err164 := db.Exec(`UPDATE plan_revisions SET status='approved' WHERE id='plan-1'`)
	t.Logf("goose=164 UPDATE plan_revisions err=%v", err164)
	if err164 == nil || !strings.Contains(err164.Error(), "routing_decisions_json") {
		t.Fatalf("premise changed: expected 'no such column ... routing_decisions_json' at goose 164, got %v", err164)
	}
}

// TestB0OpenHealsPlanRevisionsTriggerWithSeededRows is the regression test.
func TestB0OpenHealsPlanRevisionsTriggerWithSeededRows(t *testing.T) {
	dir := t.TempDir()
	b0SeedAt163(t, dir)

	raw := b0RawDB(t, dir)
	before := b0Rows(t, raw)
	if len(before) != 2 {
		t.Fatalf("seed rows = %d, want 2", len(before))
	}
	b0GooseUpTo164(t, raw)
	if got := b0Rows(t, raw); len(got) != 2 {
		t.Fatalf("goose 164 changed plan_revisions row count: %d", len(got))
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	// First real Open(): runs migrate() including the post-goose reconcile.
	st, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() on a goose-164 database with seeded plan_revisions failed: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	db := b0RawDB(t, dir)
	defer db.Close()
	healedSQL := b0assertHealed(t, db, before, "after first Open()")

	// Second Open(): healing must be idempotent.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(dir)
	if err != nil {
		t.Fatalf("second Open() failed: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	db2 := b0RawDB(t, dir)
	defer db2.Close()
	secondSQL := b0assertHealed(t, db2, b0Rows(t, db2), "after second Open()")
	if healedSQL != secondSQL {
		t.Fatalf("trigger body changed between Opens:\nfirst:  %s\nsecond: %s", healedSQL, secondSQL)
	}
	var triggers int
	if err := db2.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name='plan_revisions_immutable_update'`).Scan(&triggers); err != nil || triggers != 1 {
		t.Fatalf("expected exactly one immutability trigger, got %d err=%v", triggers, err)
	}
}

// b0assertHealed fails loudly if the heal path is not leaving a working
// trigger, and returns the trigger SQL. It mutates seeded rows only through
// statements that must be rejected, plus one legitimate status change that is
// rolled back so the row snapshot stays comparable.
func b0assertHealed(t *testing.T, db *sql.DB, want []b0PlanRow, when string) string {
	t.Helper()
	if !b0HasRoutingColumn(t, db) {
		t.Fatalf("%s: routing_decisions_json column missing - heal path stopped installing it", when)
	}
	body := b0TriggerSQL(t, db)
	if !strings.Contains(body, "routing_decisions_json") {
		t.Fatalf("%s: trigger no longer guards routing_decisions_json", when)
	}
	got := b0Rows(t, db)
	if len(got) != len(want) {
		t.Fatalf("%s: row count %d, want %d", when, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: seeded row changed: got %+v want %+v", when, got[i], want[i])
		}
	}

	// Legitimate UPDATE (the one that failed before the heal) must succeed.
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`UPDATE plan_revisions SET status='approved' WHERE id='plan-1'`); err != nil {
		_ = tx.Rollback()
		t.Fatalf("%s: legitimate UPDATE still fails (trigger not healed): %v", when, err)
	}
	_ = tx.Rollback()

	// Immutable-column UPDATEs must be rejected by the trigger itself, not by
	// a missing-column error.
	for _, stmt := range []string{
		`UPDATE plan_revisions SET summary='tampered' WHERE id='plan-1'`,
		`UPDATE plan_revisions SET routing_decisions_json='[]' WHERE id='plan-1'`,
	} {
		_, err := db.Exec(stmt)
		if err == nil {
			t.Fatalf("%s: immutable UPDATE was allowed: %s", when, stmt)
		}
		if !strings.Contains(err.Error(), "plan revisions are immutable") {
			t.Fatalf("%s: %s rejected for the wrong reason: %v", when, stmt, err)
		}
	}

	// Rows are still intact after the rejected attempts.
	after := b0Rows(t, db)
	for i := range want {
		if after[i] != want[i] {
			t.Fatalf("%s: row mutated by rejected UPDATE: %+v", when, after[i])
		}
	}
	return body
}

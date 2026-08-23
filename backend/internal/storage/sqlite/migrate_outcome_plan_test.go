package sqlite

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestMigration0100PlanAuthorityLifecycle exercises the Decide & Authorize
// seam at the SQL layer: proposal and approval emit canonical CDC events,
// plan content is frozen outside the single approved-status transition, and
// every change_log writer detached by the rebuild — inherited and Outcome
// alike — is back in place afterwards through cdc_restore.
func TestMigration0100PlanAuthorityLifecycle(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 100)
	// Raw handles skip migrate(); restore the writers the way daemon startup
	// does immediately after goose completes.
	if err := restoreChangeLogWriters(db); err != nil {
		t.Fatalf("restore change log writers: %v", err)
	}
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
		VALUES ('out_plan', ?, 'Local Focus Ledger', 1)`, spaceID); err != nil {
		t.Fatalf("insert outcome: %v", err)
	}
	rawCriteria, err := json.Marshal([]string{"Entering a duration creates one focus block."})
	if err != nil {
		t.Fatalf("marshal criteria: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO contract_revisions (id, outcome_id, number, goal, success_criteria, review)
		VALUES ('cr_plan_1', 'out_plan', 1, 'Record focus locally.', ?, 'Deterministic checks.')`,
		string(rawCriteria)); err != nil {
		t.Fatalf("insert revision: %v", err)
	}
	digest := strings.Repeat("ab", 32)

	if _, err := db.Exec(`INSERT INTO plan_revisions
		(id, outcome_id, number, contract_revision_number, status, summary, run_brief_core_digest)
		VALUES ('plan_1', 'out_plan', 1, 1, 'proposed', 'One direct Work Unit', ?)`, digest); err != nil {
		t.Fatalf("insert plan: %v", err)
	}
	if countEvents("outcome_plan_proposed") != 1 {
		t.Fatalf("outcome_plan_proposed events = %d, want 1", countEvents("outcome_plan_proposed"))
	}

	var projectID string
	var rawPayload string
	if err := db.QueryRow(`SELECT project_id, payload FROM change_log WHERE event_type = 'outcome_plan_proposed'`).
		Scan(&projectID, &rawPayload); err != nil {
		t.Fatalf("read plan proposal event: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
		t.Fatalf("decode proposal payload %q: %v", rawPayload, err)
	}
	if projectID != "p1" {
		t.Fatalf("event project_id = %q, want p1 resolved through outcome and space", projectID)
	}
	if payload["planId"] != "plan_1" || payload["contractRevisionNumber"] != float64(1) {
		t.Fatalf("unexpected proposal payload %#v", payload)
	}

	if _, err := db.Exec(`INSERT INTO work_units
		(id, plan_revision_id, kind, title, contract_revision_number, output_summary,
		 evidence_checks, verification_requirement, stop_conditions)
		VALUES ('wu_1', 'plan_1', 'direct', 'Build and prove Local Focus Ledger', 1,
			'Working local feature in the isolated worktree', json('["checks pass"]'),
			'Deterministic verification plus owner walkthrough', json('["stop before remote effects"]'))`); err != nil {
		t.Fatalf("insert work unit: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO capability_grants (id, plan_revision_id, name, scope)
		VALUES ('cg_read', 'plan_1', 'worktree.read', 'worktree/*')`); err != nil {
		t.Fatalf("insert grant: %v", err)
	}

	// Content mutations are refused by the database itself.
	for name, stmt := range map[string]string{
		"plan rebinding":       `UPDATE plan_revisions SET contract_revision_number = 2 WHERE id = 'plan_1'`,
		"digest rewrite":       `UPDATE plan_revisions SET run_brief_core_digest = '` + strings.Repeat("cd", 32) + `' WHERE id = 'plan_1'`,
		"work unit edit":       `UPDATE work_units SET title = 'Rewritten' WHERE id = 'wu_1'`,
		"grant widening":       `UPDATE capability_grants SET scope = 'repo/*' WHERE id = 'cg_read'`,
		"proposal deletion":    `DELETE FROM plan_revisions WHERE id = 'plan_1'`,
		"capability deletion":  `DELETE FROM capability_grants WHERE id = 'cg_read'`,
	} {
		if _, err := db.Exec(stmt); err == nil {
			t.Fatalf("%s must be rejected by an immutability trigger", name)
		} else if !strings.Contains(err.Error(), "immutable") {
			t.Fatalf("%s rejected with unexpected error: %v", name, err)
		}
	}

	// Approval is the only legal transition and emits exactly one event.
	if _, err := db.Exec(`UPDATE plan_revisions SET status = 'approved' WHERE id = 'plan_1'`); err != nil {
		t.Fatalf("approve plan: %v", err)
	}
	if countEvents("outcome_plan_approved") != 1 {
		t.Fatalf("outcome_plan_approved events = %d, want 1", countEvents("outcome_plan_approved"))
	}
	if _, err := db.Exec(`UPDATE plan_revisions SET status = 'approved' WHERE id = 'plan_1'`); err != nil {
		t.Fatalf("redundant approve: %v", err)
	}
	if countEvents("outcome_plan_approved") != 1 {
		t.Fatal("no-op status write must not emit CDC")
	}

	var payloadRaw string
	if err := db.QueryRow(`SELECT payload FROM change_log WHERE event_type = 'outcome_plan_approved'`).
		Scan(&payloadRaw); err != nil {
		t.Fatalf("read approval event: %v", err)
	}
	var approval map[string]any
	if err := json.Unmarshal([]byte(payloadRaw), &approval); err != nil {
		t.Fatalf("decode approval payload %q: %v", payloadRaw, err)
	}
	if approval["previousStatus"] != "proposed" || approval["status"] != "approved" {
		t.Fatalf("unexpected approval payload %#v", approval)
	}

	// Every writer detached by the rebuild is restored by the seam, including
	// the 0099-shape Outcome writers on this complete profile.
	for _, trigger := range []string{
		"sessions_cdc_insert",
		"responsibility_outcomes_cdc_insert",
		"responsibility_contract_revisions_cdc_insert",
		"outcome_plans_cdc_insert",
		"outcome_plans_cdc_update",
	} {
		var n int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name = ?`, trigger,
		).Scan(&n); err != nil {
			t.Fatalf("inspect trigger %s: %v", trigger, err)
		}
		if n != 1 {
			t.Fatalf("trigger %s count = %d, want 1 after migrate()", trigger, n)
		}
	}
}

// TestMigration0100SurvivesSkippedOutcomeMigration pins the degraded-profile
// rule that motivated seam-only restoration: when a burned ledger entry makes
// 0100 run without 0099 having created the Outcome tables, the migration must
// still apply cleanly and leave no half-built writers behind.
func TestMigration0100SurvivesSkippedOutcomeMigration(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 98)
	// Burn version 99 the way upstream-ledger collisions do.
	mustExec(t, db, `INSERT INTO goose_db_version (version_id, is_applied) VALUES (99, 1)`)

	if err := migrate(db); err != nil {
		t.Fatalf("migrate with burned 0099: %v", err)
	}

	for _, table := range []string{"plan_revisions", "work_units", "capability_grants"} {
		var n int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?`, table,
		).Scan(&n); err != nil || n != 1 {
			t.Fatalf("table %s present=%d err=%v, want created", table, n, err)
		}
	}
	// No Outcome/Plan writer may exist against absent subject tables; startup
	// reconciliation creates them only once their migrations actually land.
	for _, trigger := range []string{
		"responsibility_outcomes_cdc_insert",
		"outcome_plans_cdc_insert",
		"outcome_plans_cdc_update",
	} {
		var n int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name = ?`, trigger,
		).Scan(&n); err != nil {
			t.Fatalf("inspect trigger %s: %v", trigger, err)
		}
		if n != 0 {
			t.Fatalf("trigger %s exists without its subject tables; want deferred to cdc_restore", trigger)
		}
	}
}

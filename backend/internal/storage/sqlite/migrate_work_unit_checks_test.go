package sqlite

import (
	"database/sql"
	"strings"
	"testing"
)

// seedWorkUnitForChecks builds the minimum plan lineage a check row needs.
func seedWorkUnitForChecks(t *testing.T, db *sql.DB) {
	t.Helper()
	seedContractProject(t, db)
	spaceID := seedWorkSpace(t, db)
	if _, err := db.Exec(`INSERT INTO outcomes (id, space_id, title, current_revision_number)
		VALUES ('out_chk', ?, 'Local Focus Ledger', 1)`, spaceID); err != nil {
		t.Fatalf("insert outcome: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO contract_revisions (id, outcome_id, number, goal, success_criteria, review)
		VALUES ('cr_chk_1', 'out_chk', 1, 'Record focus locally.', '["Blocks are recorded."]', 'Deterministic checks.')`); err != nil {
		t.Fatalf("insert revision: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO plan_revisions
		(id, outcome_id, number, contract_revision_number, status, summary, run_brief_core_digest)
		VALUES ('plan_chk', 'out_chk', 1, 1, 'proposed', 'One unit', ?)`, strings.Repeat("ab", 32)); err != nil {
		t.Fatalf("insert plan: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO work_units
		(id, plan_revision_id, kind, title, contract_revision_number, output_summary, evidence_checks, verification_requirement, stop_conditions)
		VALUES ('wu_chk', 'plan_chk', 'direct', 'Deliver it', 1, 'Working feature', '["checks pass"]', 'Deterministic checks.', '["stop"]')`); err != nil {
		t.Fatalf("insert work unit: %v", err)
	}
}

// TestMigration0122ApprovedChecksAreFrozenAuthority pins the guarantee that
// makes an approved check meaningful at all: the command cannot be edited,
// re-bound, widened or removed after approval. Without it, an Attempt already
// running under a Plan could be judged against a command the owner never saw.
func TestMigration0122ApprovedChecksAreFrozenAuthority(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 122)
	seedWorkUnitForChecks(t, db)

	if _, err := db.Exec(`INSERT INTO work_unit_checks (id, work_unit_id, criterion_id, position, argv, timeout_seconds)
		VALUES ('chk_1', 'wu_chk', 'crit_1', 0, '["go","test","./..."]', 300)`); err != nil {
		t.Fatalf("insert approved check: %v", err)
	}

	forbidden := map[string]string{
		"rewriting the command":   `UPDATE work_unit_checks SET argv = '["rm","-rf","."]' WHERE id = 'chk_1'`,
		"widening the timeout":    `UPDATE work_unit_checks SET timeout_seconds = 800 WHERE id = 'chk_1'`,
		"rebinding the criterion": `UPDATE work_unit_checks SET criterion_id = 'crit_other' WHERE id = 'chk_1'`,
		"reordering the check":    `UPDATE work_unit_checks SET position = 5 WHERE id = 'chk_1'`,
		"deleting the check":      `DELETE FROM work_unit_checks WHERE id = 'chk_1'`,
	}
	for name, query := range forbidden {
		t.Run(name, func(t *testing.T) {
			_, err := db.Exec(query)
			if err == nil {
				t.Fatalf("%s was permitted on an approved check", name)
			}
			if !strings.Contains(err.Error(), "immutable") {
				t.Fatalf("%s failed for the wrong reason: %v", name, err)
			}
		})
	}

	var argv string
	var timeout int64
	if err := db.QueryRow(`SELECT argv, timeout_seconds FROM work_unit_checks WHERE id = 'chk_1'`).Scan(&argv, &timeout); err != nil {
		t.Fatalf("re-read check: %v", err)
	}
	if argv != `["go","test","./..."]` || timeout != 300 {
		t.Fatalf("approved check changed: %s / %d", argv, timeout)
	}
}

// TestMigration0122RefusesUnboundedAndEmptyChecks keeps the schema itself from
// storing a check that could never be honoured: a command with no arguments,
// or a timeout outside the operational bound.
func TestMigration0122RefusesUnboundedAndEmptyChecks(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 122)
	seedWorkUnitForChecks(t, db)

	cases := map[string]string{
		"an empty command": `INSERT INTO work_unit_checks (id, work_unit_id, criterion_id, position, argv, timeout_seconds)
			VALUES ('chk_empty', 'wu_chk', 'crit_1', 0, '[]', 300)`,
		"a zero timeout": `INSERT INTO work_unit_checks (id, work_unit_id, criterion_id, position, argv, timeout_seconds)
			VALUES ('chk_zero', 'wu_chk', 'crit_1', 1, '["go","test"]', 0)`,
		"a timeout past the bound": `INSERT INTO work_unit_checks (id, work_unit_id, criterion_id, position, argv, timeout_seconds)
			VALUES ('chk_long', 'wu_chk', 'crit_1', 2, '["go","test"]', 901)`,
	}
	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := db.Exec(query); err == nil {
				t.Fatalf("%s was accepted", name)
			}
		})
	}
}

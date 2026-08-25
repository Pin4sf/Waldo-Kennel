package sqlite

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
)

func TestMigration0103BackfillsStableCriteriaAndGuardsProofLineage(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 102)
	seedContractProject(t, db)
	spaceID := seedWorkSpace(t, db)
	if _, err := db.Exec(`INSERT INTO outcomes (id, space_id, title, current_revision_number)
		VALUES ('out_proof', ?, 'Local Focus Ledger', 1)`, spaceID); err != nil {
		t.Fatalf("seed outcome: %v", err)
	}
	criteria, _ := json.Marshal([]string{"Record one block.", "Survive restart."})
	if _, err := db.Exec(`INSERT INTO contract_revisions
		(id, outcome_id, number, goal, success_criteria, review)
		VALUES ('cr_proof_1', 'out_proof', 1, 'Record focus.', ?, 'Checks and walkthrough.')`, string(criteria)); err != nil {
		t.Fatalf("seed revision: %v", err)
	}

	upTo(t, db, 103)
	if err := reconcileOutcomeProofSchema(db); err != nil {
		t.Fatalf("reconcile proof schema: %v", err)
	}
	if err := restoreChangeLogWriters(db); err != nil {
		t.Fatalf("restore change log writers: %v", err)
	}

	rows, err := db.Query(`SELECT id, position, text FROM contract_criteria
		WHERE contract_revision_id = 'cr_proof_1' ORDER BY position`)
	if err != nil {
		t.Fatalf("read backfilled criteria: %v", err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var id, text string
		var position int
		if err := rows.Scan(&id, &position, &text); err != nil {
			t.Fatalf("scan criterion: %v", err)
		}
		got = append(got, id+":"+text)
		if position != len(got) || strings.TrimSpace(id) == "" {
			t.Fatalf("criterion identity/position id=%q position=%d", id, position)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate criteria: %v", err)
	}
	if len(got) != 2 || !strings.HasSuffix(got[0], ":Record one block.") || !strings.HasSuffix(got[1], ":Survive restart.") {
		t.Fatalf("backfilled criteria = %#v", got)
	}

	criterionID := strings.SplitN(got[0], ":", 2)[0]
	digest := strings.Repeat("a", 64)
	fingerprint := strings.Repeat("b", 64)
	if _, err := db.Exec(`INSERT INTO evidence_items
		(id, outcome_id, contract_revision_id, criterion_id, subject_type, subject_id, subject_revision,
		 kind, source_type, source_ref, producer_type, producer_ref, summary, content_digest,
		 request_key, request_fingerprint)
		VALUES ('ev_1', 'out_proof', 'cr_proof_1', ?, 'outcome', 'out_proof', 'cr_proof_1',
		 'supporting', 'owner_walkthrough', 'walkthrough-1', 'user', 'owner', 'Visible flow works.', ?,
		 'evidence-key', ?)`, criterionID, digest, fingerprint); err != nil {
		t.Fatalf("insert evidence: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO evidence_items
		(id, outcome_id, contract_revision_id, criterion_id, subject_type, subject_id, subject_revision,
		 kind, source_type, source_ref, producer_type, producer_ref, summary, content_digest,
		 request_key, request_fingerprint)
		VALUES ('ev_bad', 'out_proof', 'cr_proof_1', 'criterion-missing', 'outcome', 'out_proof', 'cr_proof_1',
		 'supporting', 'owner_walkthrough', 'walkthrough-2', 'user', 'owner', 'Bad binding.', ?,
		 'evidence-bad-key', ?)`, digest, fingerprint); err == nil {
		t.Fatal("evidence bound to a criterion outside the revision must violate the foreign key")
	}

	for name, stmt := range map[string]string{
		"evidence update": `UPDATE evidence_items SET summary = 'rewritten' WHERE id = 'ev_1'`,
		"evidence delete": `DELETE FROM evidence_items WHERE id = 'ev_1'`,
	} {
		if _, err := db.Exec(stmt); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Fatalf("%s error = %v", name, err)
		}
	}

	var n int
	if err := db.QueryRow(`SELECT count(*) FROM change_log WHERE event_type = 'outcome_evidence_recorded'`).Scan(&n); err != nil {
		t.Fatalf("count evidence CDC: %v", err)
	}
	if n != 1 {
		t.Fatalf("outcome_evidence_recorded events = %d, want 1", n)
	}
}

func TestMigration0103AcceptanceAndCorrectionAreAppendOnly(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 103)
	if err := restoreChangeLogWriters(db); err != nil {
		t.Fatalf("restore change log writers: %v", err)
	}
	seedContractProject(t, db)
	spaceID := seedWorkSpace(t, db)
	if _, err := db.Exec(`INSERT INTO outcomes (id, space_id, title, current_revision_number)
		VALUES ('out_decision', ?, 'Outcome', 1)`, spaceID); err != nil {
		t.Fatalf("seed outcome: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO contract_revisions
		(id, outcome_id, number, goal, success_criteria, review)
		VALUES ('cr_decision_1', 'out_decision', 1, 'Goal.', json('["Criterion."]'), 'Review.')`); err != nil {
		t.Fatalf("seed revision: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO contract_criteria (id, contract_revision_id, position, text)
		VALUES ('crit_decision_1', 'cr_decision_1', 1, 'Criterion.')`); err != nil {
		t.Fatalf("seed criterion: %v", err)
	}

	fingerprint := strings.Repeat("c", 64)
	if _, err := db.Exec(`INSERT INTO acceptance_decisions
		(id, outcome_id, contract_revision_id, kind, actor_type, summary, resource_disposition,
		 request_key, request_fingerprint)
		VALUES ('acc_1', 'out_decision', 'cr_decision_1', 'request_rework', 'user', 'Restart is incomplete.',
		 'retain', 'accept-key', ?)`, fingerprint); err != nil {
		t.Fatalf("insert decision: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO outcome_corrections
		(id, decision_id, outcome_id, contract_revision_id, feedback, target_type, target_id)
		VALUES ('corr_1', 'acc_1', 'out_decision', 'cr_decision_1', 'Restart is incomplete.', 'attempt', 'att-next')`); err != nil {
		t.Fatalf("insert correction: %v", err)
	}

	for name, stmt := range map[string]string{
		"decision update":   `UPDATE acceptance_decisions SET summary = 'rewritten' WHERE id = 'acc_1'`,
		"decision delete":   `DELETE FROM acceptance_decisions WHERE id = 'acc_1'`,
		"correction update": `UPDATE outcome_corrections SET feedback = 'rewritten' WHERE id = 'corr_1'`,
		"correction delete": `DELETE FROM outcome_corrections WHERE id = 'corr_1'`,
	} {
		if _, err := db.Exec(stmt); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Fatalf("%s error = %v", name, err)
		}
	}

	for _, eventType := range []string{"outcome_acceptance_decided", "outcome_correction_recorded"} {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM change_log WHERE event_type = ?`, eventType).Scan(&n); err != nil && err != sql.ErrNoRows {
			t.Fatalf("count %s: %v", eventType, err)
		}
		if n != 1 {
			t.Fatalf("%s events = %d, want 1", eventType, n)
		}
	}
}

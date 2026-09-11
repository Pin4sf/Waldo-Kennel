package sqlite

import (
	"strings"
	"testing"
)

// TestMigration0124RunIntentGenerationsAreAppendOnly protects the decision
// history. A generation that could be edited or deleted would let the record
// of what the owner authorized be rewritten after the fact, and a cleared
// acknowledgement would make a pause that already took effect look like one
// still waiting.
func TestMigration0124RunIntentGenerationsAreAppendOnly(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 124)
	if err := restoreChangeLogWriters(db); err != nil {
		t.Fatalf("restore change log writers: %v", err)
	}
	seedContractProject(t, db)
	spaceID := seedWorkSpace(t, db)
	if _, err := db.Exec(`INSERT INTO outcomes (id, space_id, title, current_revision_number)
		VALUES ('out_ri', ?, 'Local Focus Ledger', 1)`, spaceID); err != nil {
		t.Fatalf("insert outcome: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO outcome_run_intents
		(id, outcome_id, generation, desired, plan_revision_id, contract_revision_number, request_key, requested_at)
		VALUES ('ri_1', 'out_ri', 1, 'running', 'plan_1', 1, 'rk-start', '2026-09-10T00:00:00Z')`); err != nil {
		t.Fatalf("insert intent: %v", err)
	}

	forbidden := map[string]string{
		"changing the desired state": `UPDATE outcome_run_intents SET desired = 'paused' WHERE id = 'ri_1'`,
		"renumbering a generation":   `UPDATE outcome_run_intents SET generation = 7 WHERE id = 'ri_1'`,
		"rebinding the Plan":         `UPDATE outcome_run_intents SET plan_revision_id = 'plan_other' WHERE id = 'ri_1'`,
		"rewriting the request key":  `UPDATE outcome_run_intents SET request_key = 'rk-other' WHERE id = 'ri_1'`,
		"deleting a generation":      `DELETE FROM outcome_run_intents WHERE id = 'ri_1'`,
	}
	for name, query := range forbidden {
		t.Run(name, func(t *testing.T) {
			_, err := db.Exec(query)
			if err == nil {
				t.Fatalf("%s was permitted", name)
			}
			if !strings.Contains(err.Error(), "immutable") {
				t.Fatalf("%s failed for the wrong reason: %v", name, err)
			}
		})
	}

	// Acknowledgement is the one permitted write, and only once.
	if _, err := db.Exec(`UPDATE outcome_run_intents SET acknowledged_at = '2026-09-10T00:01:00Z' WHERE id = 'ri_1'`); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	if _, err := db.Exec(`UPDATE outcome_run_intents SET acknowledged_at = NULL WHERE id = 'ri_1'`); err == nil {
		t.Fatal("clearing an acknowledgement was permitted")
	}
	if _, err := db.Exec(`UPDATE outcome_run_intents SET acknowledged_at = '2026-09-10T00:09:00Z' WHERE id = 'ri_1'`); err == nil {
		t.Fatal("moving an acknowledgement was permitted")
	}

	var acknowledged string
	if err := db.QueryRow(`SELECT acknowledged_at FROM outcome_run_intents WHERE id = 'ri_1'`).Scan(&acknowledged); err != nil {
		t.Fatalf("read acknowledgement: %v", err)
	}
	if !strings.HasPrefix(acknowledged, "2026-09-10T00:01") {
		t.Fatalf("acknowledgement = %q, want the first one to stand", acknowledged)
	}
}

// TestMigration0124EmitsRunIntentChangeEvents keeps the Mission's refresh on
// the canonical trigger-backed feed rather than a parallel status authority.
func TestMigration0124EmitsRunIntentChangeEvents(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 124)
	if err := restoreChangeLogWriters(db); err != nil {
		t.Fatalf("restore change log writers: %v", err)
	}
	seedContractProject(t, db)
	spaceID := seedWorkSpace(t, db)
	if _, err := db.Exec(`INSERT INTO outcomes (id, space_id, title, current_revision_number)
		VALUES ('out_ri', ?, 'Local Focus Ledger', 1)`, spaceID); err != nil {
		t.Fatalf("insert outcome: %v", err)
	}

	count := func() int {
		t.Helper()
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM change_log WHERE event_type = 'outcome_run_intent_changed'`).Scan(&n); err != nil {
			t.Fatalf("count events: %v", err)
		}
		return n
	}
	if got := count(); got != 0 {
		t.Fatalf("events before any intent = %d", got)
	}
	if _, err := db.Exec(`INSERT INTO outcome_run_intents
		(id, outcome_id, generation, desired, plan_revision_id, contract_revision_number, request_key, requested_at)
		VALUES ('ri_1', 'out_ri', 1, 'running', 'plan_1', 1, 'rk-start', '2026-09-10T00:00:00Z')`); err != nil {
		t.Fatalf("insert intent: %v", err)
	}
	if got := count(); got != 1 {
		t.Fatalf("events after authorizing = %d, want 1", got)
	}
	if _, err := db.Exec(`UPDATE outcome_run_intents SET acknowledged_at = '2026-09-10T00:01:00Z' WHERE id = 'ri_1'`); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	// Acknowledgement is a separate fact the Mission has to see: it is what
	// turns "pausing…" into "paused".
	if got := count(); got != 2 {
		t.Fatalf("events after acknowledgement = %d, want 2", got)
	}
}

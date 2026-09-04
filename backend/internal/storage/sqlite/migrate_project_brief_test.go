package sqlite

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMigration0111ProjectBriefRevisionsAreImmutableAndCurrentIsExplicit(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 111)
	seedContractProject(t, db)

	insertRevision := func(id string, number int64, purpose string) error {
		_, err := db.Exec(`INSERT INTO project_brief_revisions (
			id, project_id, revision_number, purpose, product_context, technical_context,
			architecture_summary, conventions_json, constraints_json, setup_expectations,
			run_expectations, test_expectations, provenance_json, created_at
		) VALUES (?, 'p1', ?, ?, '', '', '', json('[]'), json('[]'), '', '', '', json('["user"]'), datetime('now'))`,
			id, number, purpose)
		return err
	}

	if err := insertRevision("pbr_1", 1, "Purpose v1"); err != nil {
		t.Fatalf("insert revision 1: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO project_brief_heads (project_id, current_revision_number, updated_at)
		VALUES ('p1', 1, datetime('now'))`); err != nil {
		t.Fatalf("create current projection: %v", err)
	}

	if _, err := db.Exec(`UPDATE project_brief_revisions SET purpose = 'rewritten' WHERE id = 'pbr_1'`); err == nil ||
		!strings.Contains(err.Error(), "project brief revisions are immutable") {
		t.Fatalf("revision UPDATE = %v, want immutability abort", err)
	}
	if _, err := db.Exec(`DELETE FROM project_brief_revisions WHERE id = 'pbr_1'`); err == nil ||
		!strings.Contains(err.Error(), "project brief revisions are immutable") {
		t.Fatalf("revision DELETE = %v, want immutability abort", err)
	}

	if err := insertRevision("pbr_2", 2, "Purpose v2"); err != nil {
		t.Fatalf("insert revision 2: %v", err)
	}
	if _, err := db.Exec(`UPDATE project_brief_heads
		SET current_revision_number = 2, updated_at = datetime('now') WHERE project_id = 'p1'`); err != nil {
		t.Fatalf("advance current projection: %v", err)
	}

	var currentID, currentPurpose string
	if err := db.QueryRow(`SELECT r.id, r.purpose
		FROM project_brief_heads h
		JOIN project_brief_revisions r
		  ON r.project_id = h.project_id AND r.revision_number = h.current_revision_number
		WHERE h.project_id = 'p1'`).Scan(&currentID, &currentPurpose); err != nil {
		t.Fatalf("read current brief: %v", err)
	}
	if currentID != "pbr_2" || currentPurpose != "Purpose v2" {
		t.Fatalf("current brief = (%q, %q), want pbr_2/Purpose v2", currentID, currentPurpose)
	}

	var oldPurpose string
	if err := db.QueryRow(`SELECT purpose FROM project_brief_revisions WHERE id = 'pbr_1'`).Scan(&oldPurpose); err != nil {
		t.Fatalf("read historical revision: %v", err)
	}
	if oldPurpose != "Purpose v1" {
		t.Fatalf("historical purpose = %q, want Purpose v1", oldPurpose)
	}

	if err := insertRevision("pbr_dup", 2, "duplicate"); err == nil {
		t.Fatal("duplicate revision number for one project must be rejected")
	}
}

func TestMigration0111ProjectBriefCDCContainsIdentifiersOnly(t *testing.T) {
	db := openContractTestDB(t)
	upTo(t, db, 111)
	seedContractProject(t, db)

	if _, err := db.Exec(`INSERT INTO project_brief_revisions (
		id, project_id, revision_number, purpose, product_context, technical_context,
		architecture_summary, conventions_json, constraints_json, setup_expectations,
		run_expectations, test_expectations, provenance_json, created_at
	) VALUES ('pbr_1', 'p1', 1, 'Private purpose text', '', '', '', json('[]'), json('[]'), '', '', '', json('[]'), datetime('now'))`); err != nil {
		t.Fatalf("insert project brief: %v", err)
	}

	var raw string
	if err := db.QueryRow(`SELECT payload FROM change_log WHERE event_type = 'project_brief_revised'`).Scan(&raw); err != nil {
		t.Fatalf("read project brief CDC event: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["revisionId"] != "pbr_1" || payload["revision"] != float64(1) {
		t.Fatalf("unexpected payload %#v", payload)
	}
	if strings.Contains(raw, "Private purpose text") {
		t.Fatal("Project Brief content must not leak into CDC payloads")
	}

	if _, err := db.Exec(`INSERT INTO change_log (project_id, event_type, payload)
		VALUES ('p1', 'not_a_real_event', json('{}'))`); err == nil {
		t.Fatal("rebuilt change_log must still reject unknown event types")
	}
}

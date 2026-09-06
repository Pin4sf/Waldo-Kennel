package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestIntakeExpiryMigrationRepairsSplitStateWithoutKillingNewerRetry(t *testing.T) {
	dataDir := t.TempDir()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dataDir, "kennel.db")+pragmas)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Reproduce a profile that has every beta migration except the stacked WT1
	// provider binding (0112) and this intake durability migration (0113). This
	// is the state an affected install is in immediately before upgrading.
	upTo(t, db, 111)
	if _, err := db.Exec(`
INSERT INTO projects (
    id, path, repo_origin_url, display_name, registered_at, config, kind
) VALUES (?, ?, ?, ?, ?, ?, ?)
`,
		"expiry-project",
		"/tmp/expiry-project",
		"https://example.com/expiry-project.git",
		"Expiry project",
		"2026-09-06T09:00:00Z",
		`{"worker":{"agent":"codex"}}`,
		"single_repo",
	); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	seedMigrationIntake(t, db, "split", "split-key", "2026-09-06T09:00:00Z")
	seedMigrationRequest(t, db, "r-split", "split", "expired", "2026-09-06T09:00:00Z", "2026-09-06T09:11:00Z")

	// An older expiry must not fail an intake that already has a newer live
	// retry at the same proposal revision.
	seedMigrationIntake(t, db, "retry", "retry-key", "2026-09-06T09:20:00Z")
	seedMigrationRequest(t, db, "r-retry-old", "retry", "expired", "2026-09-06T09:00:00Z", "2026-09-06T09:11:00Z")
	seedMigrationRequest(t, db, "r-retry-new", "retry", "requested", "2026-09-06T09:20:00Z", "")

	if err := migrate(db); err != nil {
		t.Fatalf("migrate profile through 0113: %v", err)
	}

	assertMigrationIntakeState(t, db, "split", "analysis_failed", "INTAKE_ANALYSIS_EXPIRED")
	assertMigrationIntakeState(t, db, "retry", "analyzing", "")

	// The same invariant must hold for expiries that happen after migration,
	// not only for the one-time repair.
	seedMigrationIntake(t, db, "future", "future-key", "2026-09-06T10:00:00Z")
	seedMigrationRequest(t, db, "r-future", "future", "requested", "2026-09-06T10:00:00Z", "")
	if _, err := db.Exec(`
UPDATE intake_analysis_requests
SET status = 'expired', refusal_reason = 'No proposal arrived before the request expired', answered_at = ?
WHERE id = ? AND status = 'requested'
`, "2026-09-06T10:11:00Z", "r-future"); err != nil {
		t.Fatalf("expire future request: %v", err)
	}
	assertMigrationIntakeState(t, db, "future", "analysis_failed", "INTAKE_ANALYSIS_EXPIRED")
}

func seedMigrationIntake(t *testing.T, db *sql.DB, id, requestKey, createdAt string) {
	t.Helper()
	if _, err := db.Exec(`
INSERT INTO intake_sessions (
    id, source_surface, purpose, project_id, statement, status,
    current_proposal_revision, request_key, request_fingerprint, created_at, updated_at
) VALUES (?, 'work', 'outcome', 'expiry-project', ?, 'analyzing', 0, ?, ?, ?, ?)
`, id, "Understand and recover "+id, requestKey, requestKey, createdAt, createdAt); err != nil {
		t.Fatalf("seed intake %s: %v", id, err)
	}
}

func seedMigrationRequest(t *testing.T, db *sql.DB, id, intakeID, status, createdAt, answeredAt string) {
	t.Helper()
	var answered any
	if answeredAt != "" {
		answered = answeredAt
	}
	if _, err := db.Exec(`
INSERT INTO intake_analysis_requests (
    id, intake_id, expected_proposal_revision, status, callback_token_digest,
    expires_at, created_at, answered_at
) VALUES (?, ?, 0, ?, ?, ?, ?, ?)
`, id, intakeID, status, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "2026-09-06T10:10:00Z", createdAt, answered); err != nil {
		t.Fatalf("seed analysis request %s: %v", id, err)
	}
}

func assertMigrationIntakeState(t *testing.T, db *sql.DB, id, wantStatus, wantFailure string) {
	t.Helper()
	var status, failure string
	if err := db.QueryRow(`SELECT status, failure_code FROM intake_sessions WHERE id = ?`, id).Scan(&status, &failure); err != nil {
		t.Fatalf("read intake %s: %v", id, err)
	}
	if status != wantStatus || failure != wantFailure {
		t.Fatalf("intake %s = (%q, %q), want (%q, %q)", id, status, failure, wantStatus, wantFailure)
	}
}

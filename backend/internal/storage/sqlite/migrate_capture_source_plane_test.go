package sqlite

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func TestCaptureSourcePlaneMigrationConstraintsAndRollback(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "kennel.db")+pragmas)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(db, "migrations", 163, goose.WithAllowMissing()); err != nil {
		t.Fatal(err)
	}
	stamp := "2026-09-28 12:00:00"
	grant := `INSERT INTO capture_grants(grant_id,owner_id,device_id,modality,os_permission_kind,os_permission_state,scope_json,purpose,allowed_space_ids_json,processing_route,disclosure_policy,raw_retention_seconds,derived_retention_seconds,sensitivity,bystander_policy,export_behavior,delete_behavior,state,policy_generation,created_at,updated_at) VALUES(?,?,?,?,?,'granted','{}','local context','[]','local','local_only',3600,86400,'ordinary','exclude','owner_request_only','cascade_local_custody','active',1,?,?)`
	if _, err := db.Exec(grant, "screen", "owner", "device", "screen_frame", "screen_recording", stamp, stamp); err != nil {
		t.Fatalf("screen grant: %v", err)
	}
	if _, err := db.Exec(grant, "ax", "owner", "device", "ax_tree", "accessibility", stamp, stamp); err != nil {
		t.Fatalf("AX grant: %v", err)
	}
	if _, err := db.Exec(grant, "bad", "owner", "device", "ax_tree", "screen_recording", stamp, stamp); err == nil {
		t.Fatal("AX grant accepted screen permission identity")
	}
	if _, err := db.Exec(`UPDATE capture_grants SET processing_route='provider' WHERE grant_id='screen'`); err == nil {
		t.Fatal("provider route accepted")
	}
	artifact := `INSERT INTO source_artifacts(artifact_id,owner_id,grant_id,grant_generation,modality,content_kind,mime_type,hash_algorithm,content_hash,blob_custody_ref,provenance_kind,provenance_ref,captured_at,lifecycle_state,created_at) VALUES(?,?,?,?,?,?,?,'sha256',?,?,?,?,?,'available',?)`
	if _, err := db.Exec(artifact, "frame", "owner", "screen", 1, "screen_frame", "image", "image/jpeg", strings.Repeat("a", 64), "local:frame", "screen_capture", "device/frame", stamp, stamp); err != nil {
		t.Fatalf("frame: %v", err)
	}
	if _, err := db.Exec(artifact, "tree", "owner", "ax", 1, "ax_tree", "structured", "application/json", strings.Repeat("b", 64), "local:tree", "accessibility_snapshot", "device/tree", stamp, stamp); err != nil {
		t.Fatalf("AX tree: %v", err)
	}
	if _, err := db.Exec(artifact, "same-hash", "owner", "screen", 1, "screen_frame", "image", "image/jpeg", strings.Repeat("a", 64), "local:other", "screen_capture", "device/other", stamp, stamp); err == nil {
		t.Fatal("duplicate content address within grant accepted")
	}
	if _, err := db.Exec(artifact, "same-custody", "owner", "screen", 1, "screen_frame", "image", "image/jpeg", strings.Repeat("e", 64), "local:frame", "screen_capture", "device/other", stamp, stamp); err == nil {
		t.Fatal("duplicate custody reference accepted")
	}
	if _, err := db.Exec(artifact, "wrong-owner", "other", "screen", 1, "screen_frame", "image", "image/jpeg", strings.Repeat("c", 64), "local:wrong", "screen_capture", "device/wrong", stamp, stamp); err == nil {
		t.Fatal("cross-owner artifact lineage accepted")
	}
	if _, err := db.Exec(artifact, "wrong-kind", "owner", "screen", 1, "screen_frame", "structured", "application/json", strings.Repeat("d", 64), "local:wrongkind", "screen_capture", "device/wrong", stamp, stamp); err == nil {
		t.Fatal("mismatched modality/content kind accepted")
	}
	if _, err := db.Exec(artifact, "wrong-mime", "owner", "ax", 1, "ax_tree", "structured", "image/jpeg", strings.Repeat("f", 64), "local:wrongmime", "accessibility_snapshot", "device/wrong", stamp, stamp); err == nil {
		t.Fatal("AX tree accepted image MIME")
	}
	segment := `INSERT INTO source_segments(segment_id,owner_id,grant_id,artifact_id,grant_generation,modality,ordinal,start_offset,end_offset,provenance_ref,lifecycle_state,created_at) VALUES(?,?,?,?,?,?,0,0,100,'device/segment','available',?)`
	if _, err := db.Exec(segment, "seg-frame", "owner", "screen", "frame", 1, "screen_frame", stamp); err != nil {
		t.Fatalf("segment: %v", err)
	}
	if _, err := db.Exec(segment, "seg-wrong", "owner", "ax", "frame", 1, "ax_tree", stamp); err == nil {
		t.Fatal("cross-grant/modality segment lineage accepted")
	}
	for _, index := range []string{"capture_grants_owner_device_state", "source_artifacts_grant_generation", "source_artifacts_custody_ref", "source_segments_lineage"} {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='index' AND name=?`, index).Scan(&n); err != nil || n != 1 {
			t.Fatalf("index %s: count=%d err=%v", index, n, err)
		}
	}
	var captureTriggers int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='trigger' AND tbl_name IN ('capture_grants','source_artifacts','source_segments')`).Scan(&captureTriggers); err != nil || captureTriggers != 0 {
		t.Fatalf("K4-1 must have no CDC triggers: count=%d err=%v", captureTriggers, err)
	}
	if err := goose.DownTo(db, "migrations", 162); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"capture_grants", "source_artifacts", "source_segments"} {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("rollback %s: count=%d err=%v", table, n, err)
		}
	}
}

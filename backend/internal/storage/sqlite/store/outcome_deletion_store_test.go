package store_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func TestOutcomeTrashRestoreAndPermanentDeletion(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedProject(t, s, "mer")
	space, err := s.EnsureWorkResponsibilitySpace(ctx, "mer")
	if err != nil {
		t.Fatal(err)
	}
	first, contract := focusLedgerContract(space.ID, "erase-me")
	other, otherContract := focusLedgerContract(space.ID, "keep-me")
	if err = s.CreateOutcomeWithContract(ctx, first, contract, "delete-fixture"); err != nil {
		t.Fatal(err)
	}
	if err = s.CreateOutcomeWithContract(ctx, other, otherContract, "keep-fixture"); err != nil {
		t.Fatal(err)
	}
	p, err := s.PreviewOutcomeDeletion(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.OutcomeCount != 1 || p.RecordCount < 2 || len(p.Blockers) > 0 {
		t.Fatalf("preview: %+v", p)
	}
	if err = s.PurgeOutcomeRecords(ctx, first.ID, p.Revision); err == nil {
		t.Fatal("purged without Trash")
	}
	if err = s.ChangeOutcomeTrash(ctx, first.ID, p.Revision+1, true); err == nil {
		t.Fatal("accepted stale revision")
	}
	if err = s.ChangeOutcomeTrash(ctx, first.ID, p.Revision, true); err != nil {
		t.Fatal(err)
	}
	if _, found, err := s.GetOutcome(ctx, first.ID); err != nil || found {
		t.Fatal("Trash visible in active get")
	}
	trash, err := s.ListTrashedOutcomes(ctx, "mer")
	if err != nil || len(trash) != 1 {
		t.Fatalf("Trash: %+v %v", trash, err)
	}
	if err = s.ChangeOutcomeTrash(ctx, first.ID, p.Revision, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.GetOutcome(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.ChangeOutcomeTrash(ctx, first.ID, p.Revision, true); err != nil {
		t.Fatal(err)
	}
	if err = s.BeginOutcomePurge(ctx, first.ID, p.Revision); err != nil {
		t.Fatal(err)
	}
	if err = s.PurgeOutcomeRecords(ctx, first.ID, p.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err = s.PreviewOutcomeDeletion(ctx, first.ID); err == nil {
		t.Fatal("purged Outcome retained")
	}
	if _, _, err = s.GetOutcome(ctx, other.ID); err != nil {
		t.Fatalf("unrelated Outcome lost: %v", err)
	}
	if _, err = s.EnsureWorkResponsibilitySpace(ctx, "mer"); err != nil {
		t.Fatalf("project lost: %v", err)
	}
}

func TestOutcomeDeletionKeepsActiveFenceAndErasesOwnedSession(t *testing.T) {
	dir := t.TempDir()
	s := sqlitetest.MustOpenAt(t, dir)
	ctx := context.Background()
	plan, id := seedApprovedPlan(t, s, "mer")
	attempt, err := s.CreateAttemptWithFence(ctx, admissionFor(id, plan, "delete-active", domain.FenceSubjectForProject("mer")))
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.PreviewOutcomeDeletion(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Blockers) == 0 {
		t.Fatal("active Attempt not blocked")
	}
	if err = s.ChangeOutcomeTrash(ctx, id, p.Revision, true); err == nil {
		t.Fatal("active Attempt moved to Trash")
	}
	rec := sampleRecord("mer")
	rec.IsTerminated = true
	rec.Metadata.WorkspacePath = ""
	session, err := s.CreateSession(ctx, rec)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.BindAttemptSession(ctx, domain.AttemptSessionRef{AttemptID: attempt.ID, SessionID: string(session.ID), Harness: domain.HarnessCodex, Mode: domain.SessionModeTUI, RunBriefCoreDigest: strings.Repeat("ab", 32), RunBriefCompiledDigest: strings.Repeat("cd", 32), AdmissionSnapshot: `{"snapshotVersion":1}`})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.TransitionAttemptStatus(ctx, id, attempt.ID, domain.AttemptQueued, domain.AttemptFailed, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReleaseFenceForAttempt(ctx, attempt.ID, "test-provider-stopped", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	session.IsTerminated = true
	if err = s.UpdateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	p, err = s.PreviewOutcomeDeletion(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Blockers) != 0 {
		t.Fatalf("terminal Attempt remained blocked: %+v", p.Blockers)
	}
	if err = s.ChangeOutcomeTrash(ctx, id, p.Revision, true); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "kennel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var auditSeq int64
	var auditPayload string
	if err = db.QueryRow("SELECT seq,payload FROM change_log WHERE session_id=? ORDER BY seq LIMIT 1", session.ID).Scan(&auditSeq, &auditPayload); err != nil {
		t.Fatal(err)
	}
	if err = s.BeginOutcomePurge(ctx, id, p.Revision); err != nil {
		t.Fatal(err)
	}
	if err = s.ChangeOutcomeTrash(ctx, id, p.Revision, false); err == nil {
		t.Fatal("restored after irreversible cleanup started")
	}
	if err = s.PurgeOutcomeRecords(ctx, id, p.Revision); err != nil {
		t.Fatal(err)
	}
	if _, found, err := s.GetSession(ctx, session.ID); err != nil || found {
		t.Fatalf("owned session retained: %v %v", found, err)
	}
	var retainedPayload string
	var detachedSession sql.NullString
	if err = db.QueryRow("SELECT payload,session_id FROM change_log WHERE seq=?", auditSeq).Scan(&retainedPayload, &detachedSession); err != nil {
		t.Fatal(err)
	}
	if retainedPayload != auditPayload || detachedSession.Valid {
		t.Fatal("audit payload changed or deleted session FK retained")
	}

}

func TestOutcomeDeletionPreservesImmutableGuardsAndFencesLateWrites(t *testing.T) {
	dir := t.TempDir()
	s := sqlitetest.MustOpenAt(t, dir)
	ctx := context.Background()
	_, id := seedApprovedPlan(t, s, "mer")
	db, err := sql.Open("sqlite", filepath.Join(dir, "kennel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("DELETE FROM contract_revisions WHERE outcome_id=?", id); err == nil {
		t.Fatal("ordinary immutable Contract deletion allowed")
	}
	p, err := s.PreviewOutcomeDeletion(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.ChangeOutcomeTrash(ctx, id, p.Revision, true); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"INSERT INTO outcome_document_contexts(id,outcome_id,revision,digest,state,selected_at) VALUES('late-doc',?,1,'digest','selected',datetime('now'))",
		"INSERT INTO outcome_deliveries(id,outcome_id,attempt_id,work_unit_id,artifact_version,disposition,destination,request_key,request_fingerprint,state,requested_at) VALUES('late-delivery',?,'a','w','v','draft','/tmp/d','r','f','pending',datetime('now'))",
	} {
		if _, err = db.Exec(q, id); err == nil || !strings.Contains(err.Error(), "Outcome is in Trash") {
			t.Fatalf("late admission not fenced: %v", err)
		}
	}
	var count int
	if err = db.QueryRow("SELECT COUNT(*) FROM outcome_purge_scope").Scan(&count); err != nil || count != 0 {
		t.Fatalf("durable erasure authority leaked: %d %v", count, err)
	}
}

func TestOutcomeDeletionPurgesOwnedPlanningHistoryThroughGovernedScope(t *testing.T) {
	dir := t.TempDir()
	s := sqlitetest.MustOpenAt(t, dir)
	ctx := context.Background()
	revision := seedProviderPlanOutcome(t, s)
	now := time.Date(2026, 9, 11, 13, 0, 0, 0, time.UTC)
	session := planningSessionFixture(revision, now)
	if _, _, err := s.CreatePlanningSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	owner := domain.PlanningTurn{
		ID: "planning-purge-owner", Role: domain.PlanningTurnOwner, Kind: domain.PlanningTurnMessage,
		Text: "Retain this planning history until governed erasure.", RequestKey: "planning-purge-turn",
		RequestFingerprint: domain.DigestSHA256([]byte("planning-purge-turn")), CreatedAt: now.Add(time.Second),
	}
	waiting, _, _, err := s.AppendPlanningOwnerTurn(ctx, session.ID, session.Revision, owner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ClosePlanningSession(ctx, session.ID, waiting.Revision, domain.PlanningSessionCancelled); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(dir, "kennel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("DELETE FROM planning_turns WHERE id=?", owner.ID); err == nil {
		t.Fatal("ordinary delete bypassed planning-turn immutability")
	}

	preview, err := s.PreviewOutcomeDeletion(ctx, revision.OutcomeID)
	if err != nil || len(preview.Blockers) != 0 {
		t.Fatalf("preview with planning history=%+v err=%v", preview, err)
	}
	if err = s.ChangeOutcomeTrash(ctx, revision.OutcomeID, preview.Revision, true); err != nil {
		t.Fatal(err)
	}
	if err = s.BeginOutcomePurge(ctx, revision.OutcomeID, preview.Revision); err != nil {
		t.Fatal(err)
	}
	if err = s.PurgeOutcomeRecords(ctx, revision.OutcomeID, preview.Revision); err != nil {
		t.Fatalf("purge planning history: %v", err)
	}
	for _, table := range []string{"planning_turns", "planning_sessions"} {
		var count int
		if err = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s retained %d rows after governed purge: %v", table, count, err)
		}
	}
}

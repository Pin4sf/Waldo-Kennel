package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func receiptFixture(attemptID domain.AttemptID, outcomeID domain.OutcomeID, plan domain.PlanRevision, at time.Time) domain.AttemptReceipt {
	files := []domain.ArtifactFile{
		{
			ID: "artifact-1", AttemptID: attemptID, RelativePath: "internal/parser/parse.go",
			ChangeKind: domain.ArtifactModified, ContentDigest: string(domain.DigestSHA256([]byte("parse"))),
		},
		{
			ID: "artifact-2", AttemptID: attemptID, RelativePath: "internal/parser/legacy.go",
			ChangeKind: domain.ArtifactDeleted,
		},
	}
	return domain.AttemptReceipt{
		AttemptID: attemptID, OutcomeID: outcomeID, PlanRevisionID: plan.ID,
		WorkUnitID: plan.WorkUnits[0].ID, ContractRevisionNumber: plan.ContractRevisionNumber,
		ArtifactVersion: string(domain.ArtifactManifestDigest(files)),
		WorkspaceKind:   domain.WorkspaceGitWorktree,
		WorkspacePath:   "/tmp/kennel-workspace",
		RepositoryPath:  "/tmp/repo", BaseRevision: "base-sha", ResultRevision: "result-sha",
		WorkspaceDirty: true,
		RetentionState: domain.RetentionRetained,
		ObservedAt:     at, CreatedAt: at, UpdatedAt: at,
		Files: files,
	}
}

func TestAttemptReceiptRoundTripsProducingLineageAndManifest(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	at := time.Date(2026, 9, 9, 10, 0, 0, 0, time.UTC)
	plan, outcomeID := seedApprovedPlan(t, s, "receipt-round-trip")
	attempt, err := s.CreateAttemptWithFence(ctx, admissionFor(outcomeID, plan, "rk-receipt-1", domain.FenceSubjectForProject("receipt-round-trip")))
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}

	want := receiptFixture(attempt.ID, outcomeID, plan, at)
	if err := s.SaveAttemptReceipt(ctx, want); err != nil {
		t.Fatalf("save receipt: %v", err)
	}

	got, ok, err := s.GetAttemptReceipt(ctx, attempt.ID)
	if err != nil || !ok {
		t.Fatalf("read receipt: ok=%v err=%v", ok, err)
	}
	if got.ArtifactVersion != want.ArtifactVersion {
		t.Fatalf("artifact version = %q, want %q", got.ArtifactVersion, want.ArtifactVersion)
	}
	if got.OutcomeID != outcomeID || got.PlanRevisionID != plan.ID || got.WorkUnitID != plan.WorkUnits[0].ID {
		t.Fatalf("producing lineage lost: %+v", got)
	}
	if got.BaseRevision != "base-sha" || got.ResultRevision != "result-sha" || !got.WorkspaceDirty {
		t.Fatalf("revision facts lost: %+v", got)
	}
	if len(got.Files) != 2 {
		t.Fatalf("files = %d, want both", len(got.Files))
	}
	// A deletion is retained as output: a successor that re-creates a file the
	// predecessor removed has not received its work.
	var deleted *domain.ArtifactFile
	for i := range got.Files {
		if got.Files[i].ChangeKind == domain.ArtifactDeleted {
			deleted = &got.Files[i]
		}
	}
	if deleted == nil || deleted.RelativePath != "internal/parser/legacy.go" {
		t.Fatalf("deleted path not retained: %+v", got.Files)
	}
	if deleted.ContentDigest != "" {
		t.Fatal("a deletion must not carry a content digest")
	}
}

// The manifest identity must change when the produced content changes, because
// a downstream handoff asserts it received this exact artifact version.
func TestArtifactVersionChangesWithContentAndIgnoresOrder(t *testing.T) {
	base := []domain.ArtifactFile{
		{RelativePath: "a.txt", ChangeKind: domain.ArtifactModified, ContentDigest: "digest-a"},
		{RelativePath: "b.txt", ChangeKind: domain.ArtifactAdded, ContentDigest: "digest-b"},
	}
	reordered := []domain.ArtifactFile{base[1], base[0]}
	if domain.ArtifactManifestDigest(base) != domain.ArtifactManifestDigest(reordered) {
		t.Fatal("artifact version must not depend on manifest order")
	}

	changed := []domain.ArtifactFile{
		{RelativePath: "a.txt", ChangeKind: domain.ArtifactModified, ContentDigest: "digest-a-prime"},
		base[1],
	}
	if domain.ArtifactManifestDigest(base) == domain.ArtifactManifestDigest(changed) {
		t.Fatal("changed content must change the artifact version")
	}

	// A deletion is part of the identity too.
	withDeletion := append(append([]domain.ArtifactFile{}, base...),
		domain.ArtifactFile{RelativePath: "c.txt", ChangeKind: domain.ArtifactDeleted})
	if domain.ArtifactManifestDigest(base) == domain.ArtifactManifestDigest(withDeletion) {
		t.Fatal("a deletion must change the artifact version")
	}
}

// Freezing is what makes "what the owner reviewed" stable. A later retention
// pass over the same workspace must not be able to replace it.
func TestFrozenAttemptReceiptRefusesReplacement(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	at := time.Date(2026, 9, 9, 10, 0, 0, 0, time.UTC)
	plan, outcomeID := seedApprovedPlan(t, s, "receipt-freeze")
	attempt, err := s.CreateAttemptWithFence(ctx, admissionFor(outcomeID, plan, "rk-receipt-freeze", domain.FenceSubjectForProject("receipt-freeze")))
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	reviewed := receiptFixture(attempt.ID, outcomeID, plan, at)
	if err := s.SaveAttemptReceipt(ctx, reviewed); err != nil {
		t.Fatalf("save receipt: %v", err)
	}
	if err := s.FreezeAttemptReceipt(ctx, attempt.ID, at.Add(time.Minute)); err != nil {
		t.Fatalf("freeze receipt: %v", err)
	}

	replacement := reviewed
	replacement.ArtifactVersion = "a-different-version"
	replacement.Files = []domain.ArtifactFile{{
		ID: "artifact-9", AttemptID: attempt.ID, RelativePath: "rewritten.go",
		ChangeKind: domain.ArtifactModified, ContentDigest: "other",
	}}
	if err := s.SaveAttemptReceipt(ctx, replacement); !errors.Is(err, ports.ErrAttemptReceiptFrozen) {
		t.Fatalf("save over frozen receipt = %v, want ErrAttemptReceiptFrozen", err)
	}

	// The reviewed manifest survives intact, not partially replaced.
	got, _, err := s.GetAttemptReceipt(ctx, attempt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ArtifactVersion != reviewed.ArtifactVersion || len(got.Files) != 2 {
		t.Fatalf("frozen receipt was modified: %+v", got)
	}
	if !got.Frozen() {
		t.Fatal("receipt should report itself frozen")
	}
	// Freezing twice is a no-op so callers stay idempotent after a restart.
	if err := s.FreezeAttemptReceipt(ctx, attempt.ID, at.Add(2*time.Minute)); err != nil {
		t.Fatalf("re-freeze: %v", err)
	}
}

// Retention that ran into a bound is recorded as incomplete, and incomplete is
// never treated as a usable artifact.
func TestIncompleteRetentionIsRecordedAsIncomplete(t *testing.T) {
	s := sqlitetest.MustOpen(t)
	ctx := context.Background()
	at := time.Date(2026, 9, 9, 10, 0, 0, 0, time.UTC)
	plan, outcomeID := seedApprovedPlan(t, s, "receipt-partial")
	attempt, err := s.CreateAttemptWithFence(ctx, admissionFor(outcomeID, plan, "rk-receipt-partial", domain.FenceSubjectForProject("receipt-partial")))
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	partial := receiptFixture(attempt.ID, outcomeID, plan, at)
	partial.RetentionState = domain.RetentionIncomplete
	partial.RetentionDetail = "stopped at the file-count bound"
	if err := s.SaveAttemptReceipt(ctx, partial); err != nil {
		t.Fatalf("save receipt: %v", err)
	}
	got, _, err := s.GetAttemptReceipt(ctx, attempt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RetentionState.Complete() {
		t.Fatal("incomplete retention must never satisfy a downstream handoff")
	}
	if got.RetentionDetail == "" {
		t.Fatal("an incomplete retention must say why")
	}
}

// A staged folder is not a worktree. Reporting a revision for one would
// misdescribe custody, so the domain refuses it before it can be persisted.
func TestStagedFolderReceiptCannotClaimRevisions(t *testing.T) {
	receipt := domain.AttemptReceipt{
		AttemptID: "att-1", OutcomeID: "out-1", PlanRevisionID: "plan-1", WorkUnitID: "wu-1",
		ContractRevisionNumber: 1, ArtifactVersion: "v", WorkspaceKind: domain.WorkspaceStagedFolder,
		RetentionState: domain.RetentionRetained, ObservedAt: time.Now(), BaseRevision: "sha",
	}
	if err := receipt.Validate(); err == nil {
		t.Fatal("a staged folder must not report a revision")
	}
}

// A receipt path must stay inside the workspace, so a later export cannot be
// pointed outside custody.
func TestArtifactPathCannotEscapeTheWorkspace(t *testing.T) {
	for _, path := range []string{"../outside.txt", "/etc/passwd", "nested/../../escape"} {
		file := domain.ArtifactFile{
			ID: "a", AttemptID: "att-1", RelativePath: path, ChangeKind: domain.ArtifactModified,
		}
		if err := file.Validate(); err == nil {
			t.Fatalf("path %q must be rejected", path)
		}
	}
}

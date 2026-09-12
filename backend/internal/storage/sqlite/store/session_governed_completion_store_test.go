package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func TestSessionGovernedCompletionMetadataSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	store, err := sqlitetest.Open(dataDir)
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	seedProject(t, store, "mer")

	exitCode := 0
	rec := sampleRecord("mer")
	rec.Metadata.GovernedExecutionPolicyDigest = "policy-digest"
	rec.Metadata.SupervisorCapabilityVerifier = "supervisor-verifier"
	rec.Metadata.SupervisedProcessExitCode = &exitCode
	rec.Metadata.SupervisedProcessExitReason = "exited"
	created, err := store.CreateSession(ctx, rec)
	if err != nil {
		t.Fatalf("create governed session: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store before restart: %v", err)
	}

	reopened, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	got, found, err := reopened.GetSession(ctx, created.ID)
	if err != nil || !found {
		t.Fatalf("get governed session after restart: found=%v err=%v", found, err)
	}
	if got.Metadata.GovernedExecutionPolicyDigest != "policy-digest" {
		t.Fatalf("policy digest = %q", got.Metadata.GovernedExecutionPolicyDigest)
	}
	if got.Metadata.SupervisorCapabilityVerifier != "supervisor-verifier" {
		t.Fatalf("supervisor verifier = %q", got.Metadata.SupervisorCapabilityVerifier)
	}
	if got.Metadata.SupervisedProcessExitCode == nil || *got.Metadata.SupervisedProcessExitCode != 0 {
		t.Fatalf("exit code = %v", got.Metadata.SupervisedProcessExitCode)
	}
	if got.Metadata.SupervisedProcessExitReason != "exited" {
		t.Fatalf("exit reason = %q", got.Metadata.SupervisedProcessExitReason)
	}

	updatedCode := 17
	got.Metadata.SupervisedProcessExitCode = &updatedCode
	got.Metadata.SupervisedProcessExitReason = "failed"
	got.UpdatedAt = time.Now().UTC()
	if err := reopened.UpdateSession(ctx, got); err != nil {
		t.Fatalf("update governed session: %v", err)
	}
	updated, found, err := reopened.GetSession(ctx, created.ID)
	if err != nil || !found {
		t.Fatalf("get updated governed session: found=%v err=%v", found, err)
	}
	if updated.Metadata.SupervisedProcessExitCode == nil || *updated.Metadata.SupervisedProcessExitCode != 17 {
		t.Fatalf("updated exit code = %v", updated.Metadata.SupervisedProcessExitCode)
	}
	if updated.Metadata.SupervisedProcessExitReason != "failed" {
		t.Fatalf("updated exit reason = %q", updated.Metadata.SupervisedProcessExitReason)
	}
}

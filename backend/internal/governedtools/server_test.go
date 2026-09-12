package governedtools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/governedcheck"
)

func testPolicy(write, exec bool) domain.AttemptExecutionPolicy {
	names := []string{domain.CapabilityWorktreeRead}
	grants := []domain.CapabilityGrant{{ID: "read", Name: domain.CapabilityWorktreeRead, Scope: "worktree/*"}}
	var checks []domain.ApprovedCheck
	if write {
		names = append(names, domain.CapabilityWorktreeWrite)
		grants = append(grants, domain.CapabilityGrant{ID: "write", Name: domain.CapabilityWorktreeWrite, Scope: "worktree/*"})
	}
	if exec {
		names = []string{domain.CapabilityWorktreeExec, domain.CapabilityWorktreeRead, domain.CapabilityWorktreeWrite}
		grants = []domain.CapabilityGrant{{ID: "exec", Name: domain.CapabilityWorktreeExec, Scope: "worktree/*"}, {ID: "read", Name: domain.CapabilityWorktreeRead, Scope: "worktree/*"}, {ID: "write", Name: domain.CapabilityWorktreeWrite, Scope: "worktree/*"}}
		checks = []domain.ApprovedCheck{{ID: "check-1", CriterionID: "criterion-1", Argv: []string{"go", "test", "./..."}, TimeoutSeconds: 30}}
	}
	return domain.AttemptExecutionPolicy{OutcomeID: "out-1", PlanRevisionID: "plan-1", WorkUnitID: "wu-1", ContractRevisionNumber: 1, RunBriefCoreDigest: "brief", RequiredCapabilities: names, Grants: grants, ApprovedChecks: checks}
}

func TestRepositoryToolsEnforceReadWriteAndLeaseBoundary(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("evidence"), 0o644); err != nil {
		t.Fatal(err)
	}
	readOnly := Server{Policy: testPolicy(false, false), WorkspaceRoot: root}
	if got, err := readOnly.call(context.Background(), "read_text_file", map[string]interface{}{"path": "source.txt"}); err != nil || got != "evidence" {
		t.Fatalf("read = %q, %v", got, err)
	}
	if _, err := readOnly.call(context.Background(), "write_text_file", map[string]interface{}{"path": "report.md", "content": "no"}); err == nil {
		t.Fatal("read-only policy allowed write")
	}
	if _, err := readOnly.call(context.Background(), "read_text_file", map[string]interface{}{"path": "../outside"}); err == nil {
		t.Fatal("reader escaped workspace")
	}
	writer := Server{Policy: testPolicy(true, false), WorkspaceRoot: root}
	if _, err := writer.call(context.Background(), "write_text_file", map[string]interface{}{"path": "report.md", "content": "bounded"}); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(root, "report.md")); err != nil || string(b) != "bounded" {
		t.Fatalf("report = %q, %v", b, err)
	}
}

func TestApprovedCheckToolExecutesOnlyFrozenVector(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	server := Server{Policy: testPolicy(true, true), WorkspaceRoot: root, RunCheck: func(_ context.Context, req governedcheck.Request) (governedcheck.Result, error) {
		got = append([]string(nil), req.Argv...)
		return governedcheck.Result{ExitCode: 0, EnforcedBy: "test-fence", Output: "PASS"}, nil
	}}
	text, err := server.call(context.Background(), "run_approved_check", map[string]interface{}{"check_id": "check-1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "go test ./..." || !strings.Contains(text, "enforced_by=test-fence") {
		t.Fatalf("vector=%q output=%q", got, text)
	}
	if _, err := server.call(context.Background(), "run_approved_check", map[string]interface{}{"check_id": "other"}); err == nil {
		t.Fatal("unapproved check executed")
	}
}

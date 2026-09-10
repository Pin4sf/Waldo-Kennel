package governedcheck

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func checkPolicy() domain.AttemptExecutionPolicy {
	return domain.AttemptExecutionPolicy{OutcomeID: "o", PlanRevisionID: "p", WorkUnitID: "u", ContractRevisionNumber: 1, RunBriefCoreDigest: "brief", RequiredCapabilities: []string{domain.CapabilityWorktreeExec}, Grants: []domain.CapabilityGrant{{ID: "g", Name: domain.CapabilityWorktreeExec, Scope: "worktree/*"}}}
}

func TestRunPinsWorkspaceAndAllowsAuthorizedWrite(t *testing.T) {
	workspace := t.TempDir()
	result, err := Run(context.Background(), Request{Policy: checkPolicy(), WorkspaceRoot: workspace, Argv: []string{"touch", "check-output"}, Timeout: time.Second})
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("run = %#v, %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "check-output")); err != nil {
		t.Fatal(err)
	}
}

func TestRunRejectsShellAndCapabilityWidening(t *testing.T) {
	if _, err := Run(context.Background(), Request{Policy: checkPolicy(), WorkspaceRoot: t.TempDir(), Argv: []string{"sh", "-c", "touch outside"}}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("shell err = %v", err)
	}
	policy := checkPolicy()
	policy.RequiredCapabilities = []string{domain.CapabilityWorktreeRead}
	policy.Grants[0].Name = domain.CapabilityWorktreeRead
	if _, err := Run(context.Background(), Request{Policy: policy, WorkspaceRoot: t.TempDir(), Argv: []string{"true"}}); !errors.Is(err, ErrCapabilityDenied) {
		t.Fatalf("capability err = %v", err)
	}
}

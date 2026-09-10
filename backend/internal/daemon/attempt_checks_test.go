package daemon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/artifactstore"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/governedcheck"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// checkSessionSource returns one daemon-owned session pointing at workspace.
// The runner must resolve the path from this record and never from a caller.
type checkSessionSource struct {
	workspace string
}

func (c checkSessionSource) SpawnExactAttempt(context.Context, ports.SpawnConfig, domain.ExecutionBinding) (domain.Session, int, int, error) {
	return domain.Session{}, 0, 0, errors.New("not used")
}

func (c checkSessionSource) Kill(context.Context, domain.SessionID) (bool, error) {
	return false, errors.New("not used")
}

func (c checkSessionSource) Get(context.Context, domain.SessionID) (domain.Session, error) {
	return domain.Session{SessionRecord: domain.SessionRecord{
		ID: "sess-check", Metadata: domain.SessionMetadata{WorkspacePath: c.workspace},
	}}, nil
}

// checkRefSource supplies the session binding and receipt reads the runner
// needs, without a database.
type checkRefSource struct{ receipt domain.AttemptReceipt }

func (c *checkRefSource) SaveAttemptReceipt(context.Context, domain.AttemptReceipt) error { return nil }

func (c *checkRefSource) GetAttemptReceipt(_ context.Context, id domain.AttemptID) (domain.AttemptReceipt, bool, error) {
	if c.receipt.AttemptID != id {
		return domain.AttemptReceipt{}, false, nil
	}
	return c.receipt, true, nil
}

func (c *checkRefSource) FreezeAttemptReceipt(context.Context, domain.AttemptID, time.Time) error {
	return nil
}

func (c *checkRefSource) LatestAttemptSessionRef(context.Context, domain.AttemptID) (domain.AttemptSessionRef, bool, error) {
	return domain.AttemptSessionRef{AttemptID: "att-check", SessionID: "sess-check"}, true, nil
}

func checkPolicy(t *testing.T) domain.AttemptExecutionPolicy {
	t.Helper()
	return domain.AttemptExecutionPolicy{
		OutcomeID: "out-1", PlanRevisionID: "plan-1", WorkUnitID: "wu-1",
		ContractRevisionNumber: 1, RunBriefCoreDigest: "digest",
		// Sorted and unique: the policy validator requires it, so an
		// out-of-order fixture would skip these tests instead of running them.
		RequiredCapabilities: []string{
			domain.CapabilityWorktreeExec, domain.CapabilityWorktreeRead, domain.CapabilityWorktreeWrite,
		},
		// Grants line up positionally with the required capabilities above.
		Grants: []domain.CapabilityGrant{
			{ID: "cg-exec", Name: domain.CapabilityWorktreeExec, Scope: "worktree/*"},
			{ID: "cg-read", Name: domain.CapabilityWorktreeRead, Scope: "worktree/*"},
			{ID: "cg-write", Name: domain.CapabilityWorktreeWrite, Scope: "worktree/*"},
		},
	}
}

// newCheckRunner wires the adapter over a real workspace and blob store, and
// retains that workspace so the receipt describes the bytes being checked.
func newCheckRunner(t *testing.T, workspace string) (*attemptCheckRunner, domain.Attempt, domain.AttemptReceipt) {
	t.Helper()
	artifacts, err := artifactstore.New(artifactstore.Config{Root: filepath.Join(t.TempDir(), "artifacts")})
	if err != nil {
		t.Fatalf("artifact store: %v", err)
	}
	attempt := domain.Attempt{
		ID: "att-check", OutcomeID: "out-1", PlanRevisionID: "plan-1", WorkUnitID: "wu-1",
		ContractRevisionNumber: 1, Status: domain.AttemptReconciled,
	}
	result, err := artifacts.Retain(context.Background(), artifactstore.Input{
		AttemptID: attempt.ID, OutcomeID: attempt.OutcomeID, PlanRevisionID: attempt.PlanRevisionID,
		WorkUnitID: attempt.WorkUnitID, ContractRevisionNumber: attempt.ContractRevisionNumber,
		WorkspaceKind: domain.WorkspaceStagedFolder, WorkspacePath: workspace,
	})
	if err != nil {
		t.Fatalf("retain: %v", err)
	}
	refs := &checkRefSource{receipt: result.Receipt}
	return &attemptCheckRunner{
		sessions: checkSessionSource{workspace: workspace}, refs: refs, artifacts: artifacts,
	}, attempt, result.Receipt
}

func requireEnforcement(t *testing.T, policy domain.AttemptExecutionPolicy) {
	t.Helper()
	if _, err := governedcheck.Available(policy); err != nil {
		t.Skipf("no enforcement mechanism on this host: %v", err)
	}
}

// TestRunAttemptChecks_ObservesPassAndFailureInTheProducingWorkspace is the
// production integration the standalone runner never had: real approved
// commands, run under the Attempt's own policy, in the workspace it produced.
func TestRunAttemptChecks_ObservesPassAndFailureInTheProducingWorkspace(t *testing.T) {
	policy := checkPolicy(t)
	requireEnforcement(t, policy)

	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "result.txt"), []byte("produced\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner, attempt, receipt := newCheckRunner(t, workspace)

	result, err := runner.RunAttemptChecks(context.Background(), ports.AttemptCheckRequest{
		Attempt: attempt, Receipt: receipt, Policy: policy,
		Checks: []domain.ApprovedCheck{
			{ID: "chk-pass", CriterionID: "crit-a", Argv: []string{"true"}, TimeoutSeconds: 30},
			{ID: "chk-fail", CriterionID: "crit-a", Argv: []string{"false"}, TimeoutSeconds: 30},
		},
	})
	if err != nil {
		t.Fatalf("run checks: %v", err)
	}
	if len(result.Observations) != 2 {
		t.Fatalf("observations = %d, want 2", len(result.Observations))
	}
	pass, fail := result.Observations[0], result.Observations[1]
	if !pass.Ran || !pass.Passed {
		t.Fatalf("passing check = %+v", pass)
	}
	// Evidence must never imply a confinement that did not exist.
	if pass.EnforcedBy == "" {
		t.Fatal("a check reported no enforcing mechanism")
	}
	if !fail.Ran || fail.Passed {
		t.Fatalf("failing check = %+v", fail)
	}
	// Nothing touched the result, so the checked bytes are still the retained
	// ones and a pass may bind to them.
	if result.ArtifactChanged(receipt.ArtifactVersion) {
		t.Fatalf("observed version %q differs from retained %q with no writes",
			result.ObservedArtifactVersion, receipt.ArtifactVersion)
	}
}

// TestRunAttemptChecks_DetectsACheckThatRewritesTheResultItChecks is what
// stops a pass from pre-change bytes being attached to post-change output.
func TestRunAttemptChecks_DetectsACheckThatRewritesTheResultItChecks(t *testing.T) {
	policy := checkPolicy(t)
	requireEnforcement(t, policy)

	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "result.txt"), []byte("produced\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner, attempt, receipt := newCheckRunner(t, workspace)

	result, err := runner.RunAttemptChecks(context.Background(), ports.AttemptCheckRequest{
		Attempt: attempt, Receipt: receipt, Policy: policy,
		Checks: []domain.ApprovedCheck{
			// A formatter is the ordinary version of this: it exits zero and
			// rewrites the files it just judged.
			{ID: "chk-rewrite", CriterionID: "crit-a", Argv: []string{"cp", "result.txt", "generated.txt"}, TimeoutSeconds: 30},
		},
	})
	if err != nil {
		t.Fatalf("run checks: %v", err)
	}
	if len(result.Observations) != 1 || !result.Observations[0].Passed {
		t.Fatalf("observation = %+v", result.Observations)
	}
	if !result.ArtifactChanged(receipt.ArtifactVersion) {
		t.Fatalf("a check that wrote a new file was not detected: observed %q, retained %q",
			result.ObservedArtifactVersion, receipt.ArtifactVersion)
	}
}

// TestRunAttemptChecks_ReportsAnUnenforceablePolicyAsNotRun keeps the two
// states apart. A host that cannot confine the check has a setup problem; a
// red check has a work problem, and the owner fixes them differently.
func TestRunAttemptChecks_ReportsAnUnenforceablePolicyAsNotRun(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "result.txt"), []byte("produced\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner, attempt, receipt := newCheckRunner(t, workspace)

	// A capability with no enforcement mapping cannot be translated into a
	// confinement rule, so the check must not run at all.
	policy := checkPolicy(t)
	policy.RequiredCapabilities = append([]string{"network.egress"}, policy.RequiredCapabilities...)
	policy.Grants = append([]domain.CapabilityGrant{{ID: "cg-net", Name: "network.egress", Scope: "worktree/*"}}, policy.Grants...)

	result, err := runner.RunAttemptChecks(context.Background(), ports.AttemptCheckRequest{
		Attempt: attempt, Receipt: receipt, Policy: policy,
		Checks: []domain.ApprovedCheck{{ID: "chk-1", CriterionID: "crit-a", Argv: []string{"true"}, TimeoutSeconds: 30}},
	})
	if err != nil {
		t.Fatalf("run checks: %v", err)
	}
	observation := result.Observations[0]
	if observation.Ran || observation.Passed {
		t.Fatalf("an unenforceable check reported as run: %+v", observation)
	}
	if observation.Unavailable == "" {
		t.Fatal("an unenforceable check gave no reason")
	}
}

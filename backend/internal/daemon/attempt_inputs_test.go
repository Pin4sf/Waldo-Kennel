package daemon

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/artifactstore"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

// distinctiveBytes is deliberately non-UTF8 with an embedded NUL: a transfer
// that quietly went through text handling would not reproduce it.
var distinctiveBytes = []byte{0x00, 0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0xff, 0xfe, 0x42}

func handoffGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// newHandoffRepo creates the Project repository at the approved base, plus a
// checkout function for the isolated worktrees Attempts run in.
//
// Both Attempts check out clones of one repository, as production successors
// do: two independently created repositories share a commit id only by the
// accident of being built in the same second.
func newHandoffRepo(t *testing.T) (checkout func(name string) string, base string) {
	t.Helper()
	origin := t.TempDir()
	handoffGit(t, origin, "init", "-q")
	handoffGit(t, origin, "config", "user.email", "test@example.com")
	handoffGit(t, origin, "config", "user.name", "Kennel Test")
	for name, body := range map[string]string{
		"untouched.txt": "base content nobody changed\n",
		"notes.md":      "the base version\n",
		"legacy.txt":    "the predecessor removes this\n",
	} {
		if err := os.WriteFile(filepath.Join(origin, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	handoffGit(t, origin, "add", ".")
	handoffGit(t, origin, "commit", "-qm", "base")
	base = handoffGit(t, origin, "rev-parse", "HEAD")
	return func(name string) string {
		path := filepath.Join(t.TempDir(), name)
		if out, err := exec.Command("git", "clone", "-q", origin, path).CombinedOutput(); err != nil {
			t.Fatalf("clone %s: %v (%s)", name, err, out)
		}
		return path
	}, base
}

// seedAttemptRow creates the minimum durable lineage a receipt may reference:
// a Project, an Outcome with its first Contract, an approved Plan, and one
// admitted Attempt.
func seedAttemptRow(t *testing.T, s *sqlite.Store, projectID string) (domain.Attempt, domain.PlanRevision, domain.OutcomeID) {
	t.Helper()
	ctx := context.Background()
	if err := s.UpsertProject(ctx, domain.ProjectRecord{
		ID: projectID, Path: filepath.Join(t.TempDir(), projectID), RegisteredAt: time.Now().UTC().Truncate(time.Second),
	}); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	space, err := s.EnsureWorkResponsibilitySpace(ctx, domain.ProjectID(projectID))
	if err != nil {
		t.Fatalf("ensure space: %v", err)
	}
	outcomeID := domain.OutcomeID("out-" + projectID)
	contract := domain.ContractRevision{
		ID: domain.ContractRevisionID("cr-" + projectID), OutcomeID: outcomeID, Number: 1,
		Goal: "Hand work down.", SuccessCriteria: []string{"The successor holds the predecessor's result."},
		Review: "Deterministic checks.",
	}
	if err := s.CreateOutcomeWithContract(ctx, domain.Outcome{
		ID: outcomeID, SpaceID: space.ID, Title: "Artifact continuity",
	}, contract, "rk-create-"+projectID); err != nil {
		t.Fatalf("create outcome: %v", err)
	}
	unit := domain.WorkUnit{
		ID: domain.WorkUnitID("wu-" + projectID), Kind: domain.WorkUnitDirect, Title: "Produce a result",
		ContractRevisionNumber: 1, OutputSummary: "Files in the isolated worktree.",
		EvidenceChecks: []string{"checks pass"}, VerificationRequirement: "Deterministic checks.",
		StopConditions: []string{"stop before remote effects"},
		Provider:       domain.HarnessCodex, ModelSelection: domain.ExecutionBindingModelProviderDefault,
		RequiredCapabilities: []string{domain.CapabilityWorktreeRead, domain.CapabilityWorktreeWrite},
	}
	grants := []domain.CapabilityGrant{
		{ID: domain.CapabilityGrantID("cg-read-" + projectID), Name: domain.CapabilityWorktreeRead, Scope: "worktree/*"},
		{ID: domain.CapabilityGrantID("cg-write-" + projectID), Name: domain.CapabilityWorktreeWrite, Scope: "worktree/*"},
	}
	digest, err := domain.ComputeRunBriefCoreDigest(contract, unit, grants)
	if err != nil {
		t.Fatalf("compute digest: %v", err)
	}
	plan, err := s.AppendPlanRevision(ctx, outcomeID, domain.PlanRevision{
		ID: domain.PlanRevisionID("plan-" + projectID), OutcomeID: outcomeID, ContractRevisionNumber: 1,
		Status: domain.PlanStatusProposed, Summary: "One direct WorkUnit",
		WorkUnits: []domain.WorkUnit{unit}, Grants: grants,
		RoutingDecisions: []domain.WorkUnitRoutingDecision{{
			WorkUnitID: unit.ID,
			Decision: domain.RoutingDecision{
				Status: domain.RoutingDecisionRecommended, PolicyVersion: domain.RoutingPolicyVersion,
				Role: domain.RoutingRoleWorker, RecommendedCandidateID: string(domain.HarnessCodex),
				RecommendedProvider: string(domain.HarnessCodex), RecommendedModelSelection: domain.ExecutionBindingModelProviderDefault,
			},
		}},
		RunBriefCoreDigest: digest,
	})
	if err != nil {
		t.Fatalf("append plan: %v", err)
	}
	approved, found, err := s.ApprovePlanRevision(ctx, outcomeID, plan.ID)
	if err != nil || !found {
		t.Fatalf("approve plan: found=%v err=%v", found, err)
	}
	attempt, err := s.CreateAttemptWithFence(ctx, ports.AttemptAdmission{
		OutcomeID: outcomeID, PlanRevisionID: approved.ID, WorkUnitID: unit.ID,
		ContractRevisionNumber: approved.ContractRevisionNumber, RequestKey: "rk-attempt-" + projectID,
		FenceSubject: domain.FenceSubjectForProject(domain.ProjectID(projectID)), At: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("admit attempt: %v", err)
	}
	return attempt, approved, outcomeID
}

// TestProvisionAttemptInputs_HandsTheExactRetainedResultToASuccessorAfterRestart
// is the artifact-continuity requirement end to end at the daemon seam: A
// writes distinctive bytes, the daemon's durable state is closed and reopened,
// A's workspace is gone, and B is provisioned with exactly those bytes —
// including an executable mode, binary content and a deletion.
func TestProvisionAttemptInputs_HandsTheExactRetainedResultToASuccessorAfterRestart(t *testing.T) {
	dataDir := t.TempDir()
	artifactRoot := filepath.Join(dataDir, "artifacts")
	ctx := context.Background()

	store, err := sqlitetest.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	attempt, _, _ := seedAttemptRow(t, store, "continuity")

	checkout, base := newHandoffRepo(t)
	producer := checkout("producer")
	if err := os.WriteFile(filepath.Join(producer, "notes.md"), []byte("the predecessor's version\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(producer, "build.sh"), []byte("#!/bin/sh\necho produced\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(producer, "asset.bin"), distinctiveBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(producer, "legacy.txt")); err != nil {
		t.Fatal(err)
	}

	artifacts, err := artifactstore.New(artifactstore.Config{Root: artifactRoot})
	if err != nil {
		t.Fatalf("artifact store: %v", err)
	}
	result, err := artifacts.Retain(ctx, artifactstore.Input{
		AttemptID: attempt.ID, OutcomeID: attempt.OutcomeID, PlanRevisionID: attempt.PlanRevisionID,
		WorkUnitID: attempt.WorkUnitID, ContractRevisionNumber: attempt.ContractRevisionNumber,
		WorkspaceKind: domain.WorkspaceGitWorktree, WorkspacePath: producer, BaseRevision: base,
	})
	if err != nil {
		t.Fatalf("retain: %v", err)
	}
	if !result.Receipt.RetentionState.Complete() {
		t.Fatalf("retention = %s (%s)", result.Receipt.RetentionState, result.Receipt.RetentionDetail)
	}
	if err := store.SaveAttemptReceipt(ctx, result.Receipt); err != nil {
		t.Fatalf("save receipt: %v", err)
	}
	if err := store.FreezeAttemptReceipt(ctx, attempt.ID, time.Now().UTC()); err != nil {
		t.Fatalf("freeze receipt: %v", err)
	}
	admitted := ports.AttemptInputRef{
		AttemptID: attempt.ID, WorkUnitID: attempt.WorkUnitID, ArtifactVersion: result.Receipt.ArtifactVersion,
	}

	// Restart: close the daemon's durable state, drop the producer's
	// workspace, and rebuild both halves over the same on-disk roots.
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}
	if err := os.RemoveAll(producer); err != nil {
		t.Fatal(err)
	}
	restarted, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	restartedArtifacts, err := artifactstore.New(artifactstore.Config{Root: artifactRoot})
	if err != nil {
		t.Fatalf("reopen artifact store: %v", err)
	}
	provisioner := &attemptInputProvisioner{receipts: restarted, artifacts: restartedArtifacts}

	successor := checkout("successor")
	if err := provisioner.ProvisionAttemptInputs(ctx, ports.AttemptInputProvisionRequest{
		Inputs: []ports.AttemptInputRef{admitted}, WorkspacePath: successor,
		WorkspaceKind: domain.WorkspaceGitWorktree, BaseRevision: base,
	}); err != nil {
		t.Fatalf("provision after restart: %v", err)
	}

	assertFile := func(name string, want []byte, mode os.FileMode) {
		t.Helper()
		body, readErr := os.ReadFile(filepath.Join(successor, name))
		if readErr != nil {
			t.Fatalf("%s: %v", name, readErr)
		}
		if !bytes.Equal(body, want) {
			t.Fatalf("%s body = %q, want %q", name, body, want)
		}
		info, statErr := os.Lstat(filepath.Join(successor, name))
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Mode().Perm() != mode {
			t.Fatalf("%s mode = %o, want %o", name, info.Mode().Perm(), mode)
		}
	}
	assertFile("notes.md", []byte("the predecessor's version\n"), 0o644)
	assertFile("build.sh", []byte("#!/bin/sh\necho produced\n"), 0o755)
	assertFile("asset.bin", distinctiveBytes, 0o644)
	assertFile("untouched.txt", []byte("base content nobody changed\n"), 0o644)
	if _, err := os.Lstat(filepath.Join(successor, "legacy.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the predecessor's deletion did not reach the successor: %v", err)
	}
}

// TestProvisionAttemptInputs_RefusesAnythingButTheAdmittedArtifact proves the
// fail-closed half: a corrupt, replaced, unfrozen or missing upstream result
// stops provisioning, which is what stops a provider being launched on inputs
// nobody authorized.
func TestProvisionAttemptInputs_RefusesAnythingButTheAdmittedArtifact(t *testing.T) {
	dataDir := t.TempDir()
	artifactRoot := filepath.Join(dataDir, "artifacts")
	ctx := context.Background()

	store, err := sqlitetest.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	attempt, _, _ := seedAttemptRow(t, store, "refusals")

	producer := t.TempDir()
	if err := os.WriteFile(filepath.Join(producer, "result.txt"), []byte("produced\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	artifacts, err := artifactstore.New(artifactstore.Config{Root: artifactRoot})
	if err != nil {
		t.Fatalf("artifact store: %v", err)
	}
	result, err := artifacts.Retain(ctx, artifactstore.Input{
		AttemptID: attempt.ID, OutcomeID: attempt.OutcomeID, PlanRevisionID: attempt.PlanRevisionID,
		WorkUnitID: attempt.WorkUnitID, ContractRevisionNumber: attempt.ContractRevisionNumber,
		WorkspaceKind: domain.WorkspaceStagedFolder, WorkspacePath: producer,
	})
	if err != nil {
		t.Fatalf("retain: %v", err)
	}
	if err := store.SaveAttemptReceipt(ctx, result.Receipt); err != nil {
		t.Fatalf("save receipt: %v", err)
	}
	provisioner := &attemptInputProvisioner{receipts: store, artifacts: artifacts}
	admitted := ports.AttemptInputRef{
		AttemptID: attempt.ID, WorkUnitID: attempt.WorkUnitID, ArtifactVersion: result.Receipt.ArtifactVersion,
	}
	provision := func(inputs []ports.AttemptInputRef, destination string) error {
		return provisioner.ProvisionAttemptInputs(ctx, ports.AttemptInputProvisionRequest{
			Inputs: inputs, WorkspacePath: destination, WorkspaceKind: domain.WorkspaceStagedFolder,
		})
	}

	// An unfrozen result can still be replaced by a later retention pass, so
	// the successor could be built on bytes that change underneath it.
	if err := provision([]ports.AttemptInputRef{admitted}, t.TempDir()); !errors.Is(err, ports.ErrAttemptInputProvisioning) {
		t.Fatalf("unfrozen result provisioned: %v", err)
	}
	if err := store.FreezeAttemptReceipt(ctx, attempt.ID, time.Now().UTC()); err != nil {
		t.Fatalf("freeze receipt: %v", err)
	}
	destination := t.TempDir()
	if err := provision([]ports.AttemptInputRef{admitted}, destination); err != nil {
		t.Fatalf("frozen result refused: %v", err)
	}
	if body, readErr := os.ReadFile(filepath.Join(destination, "result.txt")); readErr != nil || string(body) != "produced\n" {
		t.Fatalf("provisioned content = %q / %v", body, readErr)
	}

	cases := []struct {
		name  string
		input ports.AttemptInputRef
	}{
		{
			// The retained result was replaced since admission. Substituting
			// the newer bytes would hand the successor work nobody authorized.
			name:  "a different artifact version than the one admitted",
			input: ports.AttemptInputRef{AttemptID: attempt.ID, WorkUnitID: attempt.WorkUnitID, ArtifactVersion: strings.Repeat("f", 64)},
		},
		{
			name:  "a result belonging to a different WorkUnit",
			input: ports.AttemptInputRef{AttemptID: attempt.ID, WorkUnitID: "wu-somebody-else", ArtifactVersion: result.Receipt.ArtifactVersion},
		},
		{
			name:  "an Attempt that retained nothing",
			input: ports.AttemptInputRef{AttemptID: "att-never-retained", WorkUnitID: attempt.WorkUnitID, ArtifactVersion: result.Receipt.ArtifactVersion},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := t.TempDir()
			if err := provision([]ports.AttemptInputRef{tc.input}, target); !errors.Is(err, ports.ErrAttemptInputProvisioning) {
				t.Fatalf("provisioned %s: %v", tc.name, err)
			}
			entries, readErr := os.ReadDir(target)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != 0 {
				t.Fatalf("refused provisioning still wrote %d entries", len(entries))
			}
		})
	}

	t.Run("a corrupt retained blob", func(t *testing.T) {
		blob := filepath.Join(artifactRoot, string(attempt.ID), result.Receipt.ArtifactVersion, "result.txt")
		if err := os.WriteFile(blob, []byte("tampered\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		target := t.TempDir()
		if err := provision([]ports.AttemptInputRef{admitted}, target); !errors.Is(err, ports.ErrAttemptInputProvisioning) {
			t.Fatalf("provisioned a tampered blob: %v", err)
		}
	})
}

// TestProvisionAttemptInputs_RefusesWhenRetentionIsUnwired keeps the seam
// fail-closed: a daemon that cannot resolve inputs must refuse, never start a
// successor on a workspace that is missing its predecessor's work.
func TestProvisionAttemptInputs_RefusesWhenRetentionIsUnwired(t *testing.T) {
	var unwired *attemptInputProvisioner
	err := unwired.ProvisionAttemptInputs(context.Background(), ports.AttemptInputProvisionRequest{
		Inputs: []ports.AttemptInputRef{{AttemptID: "att-a", WorkUnitID: "wu-a", ArtifactVersion: "v1"}},
	})
	if !errors.Is(err, ports.ErrAttemptInputProvisioning) {
		t.Fatalf("unwired provisioner = %v, want a provisioning refusal", err)
	}
}

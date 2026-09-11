package sessionmanager

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

type recordingInputProvisioner struct {
	seen ports.AttemptInputProvisionRequest
	err  error
}

func (r *recordingInputProvisioner) ProvisionAttemptInputs(_ context.Context, req ports.AttemptInputProvisionRequest) error {
	r.seen = req
	return r.err
}

var admittedInputs = []ports.AttemptInputRef{
	{AttemptID: "att-a", WorkUnitID: "wu-a", ArtifactVersion: "v-a"},
}

// TestProvisionAttemptInputs_RefusesWhenHandoffIsUnwired keeps the launch path
// fail-closed. A successor whose inputs cannot be placed must not be started
// on a workspace missing its predecessor's work: an absent provisioner is a
// refusal, not a silent skip.
func TestProvisionAttemptInputs_RefusesWhenHandoffIsUnwired(t *testing.T) {
	manager := &Manager{}
	err := manager.provisionAttemptInputs(context.Background(),
		ports.SpawnConfig{AttemptInputs: admittedInputs},
		domain.ProjectRecord{Kind: domain.ProjectKindSingleRepo},
		ports.WorkspaceInfo{Path: t.TempDir()})
	if !errors.Is(err, ports.ErrAttemptInputProvisioning) {
		t.Fatalf("unwired handoff = %v, want a provisioning refusal", err)
	}
}

func TestProvisionAttemptInputs_PassesTheAdmittedInputsAndCustodyShape(t *testing.T) {
	cases := []struct {
		name     string
		kind     domain.ProjectKind
		wantKind domain.WorkspaceKind
	}{
		{name: "a repository successor gets worktree custody", kind: domain.ProjectKindSingleRepo, wantKind: domain.WorkspaceGitWorktree},
		// A plain folder is not a worktree and must never be described as one,
		// so no base revision is claimed for it.
		{name: "a scratch successor gets staged-folder custody", kind: domain.ProjectKindScratch, wantKind: domain.WorkspaceStagedFolder},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provisioner := &recordingInputProvisioner{}
			manager := &Manager{}
			manager.SetAttemptInputProvisioner(provisioner)
			workspace := t.TempDir()

			if err := manager.provisionAttemptInputs(context.Background(),
				ports.SpawnConfig{AttemptInputs: admittedInputs},
				domain.ProjectRecord{Kind: tc.kind},
				ports.WorkspaceInfo{Path: workspace}); err != nil {
				t.Fatalf("provision: %v", err)
			}
			if len(provisioner.seen.Inputs) != 1 || provisioner.seen.Inputs[0] != admittedInputs[0] {
				t.Fatalf("inputs = %#v, want exactly the admitted reference", provisioner.seen.Inputs)
			}
			if provisioner.seen.WorkspacePath != workspace {
				t.Fatalf("workspace = %q, want the daemon-owned path %q", provisioner.seen.WorkspacePath, workspace)
			}
			if provisioner.seen.WorkspaceKind != tc.wantKind {
				t.Fatalf("custody = %q, want %q", provisioner.seen.WorkspaceKind, tc.wantKind)
			}
			if tc.wantKind == domain.WorkspaceStagedFolder && provisioner.seen.BaseRevision != "" {
				t.Fatalf("a staged folder reported base revision %q", provisioner.seen.BaseRevision)
			}
		})
	}
}

// TestProvisionAttemptInputs_PropagatesTheRefusalSoLaunchIsAbandoned proves
// the error stays recognisable through the spawn path, which is what lets the
// Outcome service report a known pre-launch failure rather than an unknown
// start.
func TestProvisionAttemptInputs_PropagatesTheRefusalSoLaunchIsAbandoned(t *testing.T) {
	provisioner := &recordingInputProvisioner{
		err: fmt.Errorf("%w: corrupt blob", ports.ErrAttemptInputProvisioning),
	}
	manager := &Manager{}
	manager.SetAttemptInputProvisioner(provisioner)

	err := manager.provisionAttemptInputs(context.Background(),
		ports.SpawnConfig{AttemptInputs: admittedInputs},
		domain.ProjectRecord{Kind: domain.ProjectKindScratch},
		ports.WorkspaceInfo{Path: t.TempDir()})
	if !errors.Is(err, ports.ErrAttemptInputProvisioning) {
		t.Fatalf("err = %v, want a recognisable provisioning refusal", err)
	}
}

package sessionmanager

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

type recoveryEvidenceFakeStore struct {
	*fakeStore
	ref   domain.AttemptSessionRef
	found bool
	err   error
}

func (s *recoveryEvidenceFakeStore) LatestAttemptSessionRefForSession(context.Context, string) (domain.AttemptSessionRef, bool, error) {
	return s.ref, s.found, s.err
}

func recoveryPolicy(t *testing.T) (domain.AttemptExecutionPolicy, string) {
	t.Helper()
	policy := domain.AttemptExecutionPolicy{
		OutcomeID:              "out-1",
		PlanRevisionID:         "plan-1",
		WorkUnitID:             "wu-1",
		ContractRevisionNumber: 1,
		RunBriefCoreDigest:     strings.Repeat("a", 64),
		RequiredCapabilities:   []string{domain.CapabilityWorktreeRead},
		Grants: []domain.CapabilityGrant{{
			ID: "grant-read", Name: domain.CapabilityWorktreeRead, Scope: "worktree/*",
		}},
	}
	digest, err := policy.Digest()
	if err != nil {
		t.Fatal(err)
	}
	return policy, digest
}

func recoveryRef(t *testing.T, sessionID domain.SessionID, mode domain.SessionMode, policy domain.AttemptExecutionPolicy, policyDigest string) domain.AttemptSessionRef {
	t.Helper()
	snapshot, err := json.Marshal(map[string]any{
		"snapshotVersion":        domain.AdmissionSnapshotVersion,
		"harness":                string(domain.HarnessCodex),
		"modelSelection":         string(domain.ExecutionBindingModelExplicit),
		"requestedModel":         "approved-model",
		"effectiveModel":         "approved-model",
		"workUnitId":             string(policy.WorkUnitID),
		"mode":                   string(mode),
		"runBriefCoreDigest":     policy.RunBriefCoreDigest,
		"runBriefCompiledDigest": strings.Repeat("b", 64),
		"executionPolicy":        policy,
		"executionPolicyDigest":  policyDigest,
		"sessionId":              string(sessionID),
	})
	if err != nil {
		t.Fatal(err)
	}
	return domain.AttemptSessionRef{
		ID: "asr-1", AttemptID: "att-1", Seq: 1, SessionID: string(sessionID),
		Harness: domain.HarnessCodex, Mode: mode,
		RunBriefCoreDigest: policy.RunBriefCoreDigest, RunBriefCompiledDigest: strings.Repeat("b", 64),
		AdmissionSnapshot: string(snapshot),
	}
}

func TestLoadRecoveryExecutionRejectsMissingGovernedEvidence(t *testing.T) {
	policy, digest := recoveryPolicy(t)
	_ = policy
	store := newFakeStore()
	store.sessions["mer-1"] = domain.SessionRecord{
		ID: "mer-1", ProjectID: "mer", Harness: domain.HarnessCodex,
		Metadata: domain.SessionMetadata{GovernedExecutionPolicyDigest: digest},
	}
	mgr := &Manager{store: store}

	if _, err := mgr.loadRecoveryExecution(context.Background(), store.sessions["mer-1"]); err == nil || !strings.Contains(err.Error(), "evidence unavailable") {
		t.Fatalf("loadRecoveryExecution error = %v, want missing evidence to block", err)
	}
}

func TestLoadRecoveryExecutionRejectsInvalidGovernedEvidence(t *testing.T) {
	policy, digest := recoveryPolicy(t)
	base := newFakeStore()
	base.sessions["mer-1"] = domain.SessionRecord{
		ID: "mer-1", ProjectID: "mer", Harness: domain.HarnessCodex,
		Metadata: domain.SessionMetadata{GovernedExecutionPolicyDigest: digest},
	}
	store := &recoveryEvidenceFakeStore{
		fakeStore: base,
		ref:       recoveryRef(t, "mer-1", domain.SessionModeTUI, policy, "not-the-policy-digest"),
		found:     true,
	}
	mgr := &Manager{store: store}

	if _, err := mgr.loadRecoveryExecution(context.Background(), base.sessions["mer-1"]); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("loadRecoveryExecution error = %v, want invalid evidence to block", err)
	}
}

func TestLoadRecoveryExecutionLeavesHistoricalSnapshotReadable(t *testing.T) {
	base := newFakeStore()
	rec := domain.SessionRecord{ID: "mer-1", ProjectID: "mer", Harness: domain.HarnessCodex}
	base.sessions[rec.ID] = rec
	store := &recoveryEvidenceFakeStore{
		fakeStore: base,
		ref: domain.AttemptSessionRef{
			ID: "asr-legacy", AttemptID: "att-legacy", Seq: 1, SessionID: string(rec.ID),
			Harness: domain.HarnessCodex, Mode: domain.SessionModeTUI,
			RunBriefCoreDigest: strings.Repeat("a", 64), RunBriefCompiledDigest: strings.Repeat("b", 64),
			AdmissionSnapshot: `{"snapshotVersion":1,"harness":"codex"}`,
		},
		found: true,
	}
	mgr := &Manager{store: store}

	if execution, err := mgr.loadRecoveryExecution(context.Background(), rec); err != nil || execution != nil {
		t.Fatalf("loadRecoveryExecution = (%v, %v), want legacy snapshot readable without synthesized policy", execution, err)
	}
}

func TestResumeGovernedTUIUsesAdmissionBindingAfterProjectChange(t *testing.T) {
	policy, digest := recoveryPolicy(t)
	base := newFakeStore()
	base.projects["mer"] = domain.ProjectRecord{ID: "mer", Config: domain.ProjectConfig{
		Worker: domain.RoleOverride{
			Harness:     domain.HarnessCodex,
			AgentConfig: domain.AgentConfig{Model: "broader-project-model", Permissions: domain.PermissionModeBypassPermissions},
		},
	}}
	base.sessions["mer-1"] = domain.SessionRecord{
		ID: "mer-1", ProjectID: "mer", Kind: domain.KindWorker, Harness: domain.HarnessCodex,
		Activity: domain.Activity{State: domain.ActivityExited},
		Metadata: domain.SessionMetadata{
			WorkspacePath: "/ws/mer-1", Branch: "kennel/mer-1", RuntimeHandleID: "h1", AgentSessionID: "native-1",
			GovernedExecutionPolicyDigest: digest,
		},
	}
	store := &recoveryEvidenceFakeStore{fakeStore: base, ref: recoveryRef(t, "mer-1", domain.SessionModeTUI, policy, digest), found: true}
	agent := &recordingAgent{}
	rt := &fakeRuntime{aliveByHandle: map[string]bool{"h1": true}}
	mgr := New(Deps{Runtime: rt, Agents: singleAgent{agent: agent}, Workspace: &fakeWorkspace{}, Store: store, Messenger: &fakeMessenger{}, Lifecycle: &fakeLCM{store: base}, LookPath: func(string) (string, error) { return "/bin/true", nil }})

	if _, err := mgr.ResumeAgentWithMode(context.Background(), "mer-1"); err != nil {
		t.Fatalf("ResumeAgentWithMode: %v", err)
	}
	if agent.lastRestore.ExecutionPolicy == nil {
		t.Fatalf("restore policy = nil, want durable Attempt policy")
	}
	gotDigest, err := agent.lastRestore.ExecutionPolicy.Digest()
	if err != nil || gotDigest != digest {
		t.Fatalf("restore policy digest = %q, want %q (err=%v)", gotDigest, digest, err)
	}
	if agent.lastRestore.Config.Model != "approved-model" {
		t.Fatalf("restore model = %q, want approved-model", agent.lastRestore.Config.Model)
	}
	if agent.lastRestore.Config.Permissions == domain.PermissionModeBypassPermissions {
		t.Fatal("restore inherited broader Project bypass permissions")
	}
}

func TestResumeGovernedChatUsesAdmissionBindingAfterProjectChange(t *testing.T) {
	policy, digest := recoveryPolicy(t)
	base := newFakeStore()
	base.projects["mer"] = domain.ProjectRecord{ID: "mer", Config: domain.ProjectConfig{
		Worker: domain.RoleOverride{
			Harness:     domain.HarnessCodex,
			AgentConfig: domain.AgentConfig{Model: "broader-project-model", Permissions: domain.PermissionModeBypassPermissions},
		},
	}}
	base.sessions["mer-1"] = domain.SessionRecord{
		ID: "mer-1", ProjectID: "mer", Kind: domain.KindWorker, Harness: domain.HarnessCodex,
		Mode: domain.SessionModeChat, Activity: domain.Activity{State: domain.ActivityExited},
		Metadata: domain.SessionMetadata{
			WorkspacePath: "/ws/mer-1", Branch: "kennel/mer-1", ProviderConversationID: "thread-1",
			GovernedExecutionPolicyDigest: digest,
		},
	}
	store := &recoveryEvidenceFakeStore{fakeStore: base, ref: recoveryRef(t, "mer-1", domain.SessionModeChat, policy, digest), found: true}
	launcher := &recordingLauncher{}
	mgr := New(Deps{Runtime: &fakeRuntime{}, Agents: fakeAgents{}, Workspace: &fakeWorkspace{}, Store: store, Messenger: &fakeMessenger{}, Chat: launcher, Lifecycle: &fakeLCM{store: base}, DataDir: "/kennel-test-data"})

	if _, err := mgr.ResumeAgentWithMode(context.Background(), "mer-1"); err != nil {
		t.Fatalf("ResumeAgentWithMode: %v", err)
	}
	if len(launcher.started) != 1 {
		t.Fatalf("started %d chat controllers, want 1", len(launcher.started))
	}
	started := launcher.started[0]
	if started.Model != "approved-model" {
		t.Fatalf("chat resume model = %q, want approved-model", started.Model)
	}
	if started.Permissions == domain.PermissionModeBypassPermissions {
		t.Fatal("chat resume inherited broader Project bypass permissions")
	}
	if started.ExecutionPolicy == nil {
		t.Fatal("chat resume policy = nil, want durable Attempt policy")
	}
	gotDigest, err := started.ExecutionPolicy.Digest()
	if err != nil || gotDigest != digest {
		t.Fatalf("chat resume policy digest = %q, want %q (err=%v)", gotDigest, digest, err)
	}
}

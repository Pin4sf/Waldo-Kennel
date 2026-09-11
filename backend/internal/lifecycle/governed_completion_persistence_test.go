package lifecycle

import (
	"context"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func TestMarkSpawnedPersistsGovernedCompletionFactsAndClearsPriorExit(t *testing.T) {
	ctx := context.Background()
	priorExitCode := 17
	st := newFakeStore()
	st.sessions["mer-1"] = domain.SessionRecord{
		ID:        "mer-1",
		ProjectID: "mer",
		Metadata: domain.SessionMetadata{
			RuntimeLaunchID:              "launch-1",
			SupervisorCapabilityVerifier: "supervisor-verifier-1",
			SupervisedProcessExitCode:    &priorExitCode,
			SupervisedProcessExitReason:  "failed",
		},
	}
	m := New(st, nil)

	if err := m.MarkSpawned(ctx, "mer-1", domain.SessionMetadata{
		RuntimeLaunchID:               "launch-2",
		GovernedExecutionPolicyDigest: "policy-digest",
		SupervisorCapabilityVerifier:  "supervisor-verifier-2",
	}); err != nil {
		t.Fatalf("MarkSpawned: %v", err)
	}

	got, _, err := st.GetSession(ctx, "mer-1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.Metadata.GovernedExecutionPolicyDigest != "policy-digest" {
		t.Fatalf("policy digest = %q", got.Metadata.GovernedExecutionPolicyDigest)
	}
	if got.Metadata.SupervisorCapabilityVerifier != "supervisor-verifier-2" {
		t.Fatalf("supervisor verifier = %q", got.Metadata.SupervisorCapabilityVerifier)
	}
	if got.Metadata.SupervisedProcessExitCode != nil || got.Metadata.SupervisedProcessExitReason != "" {
		t.Fatalf("prior exit survived relaunch: code=%v reason=%q", got.Metadata.SupervisedProcessExitCode, got.Metadata.SupervisedProcessExitReason)
	}
}

func TestMarkSpawnedNewGenerationClearsOmittedSupervisorVerifier(t *testing.T) {
	ctx := context.Background()
	st := newFakeStore()
	st.sessions["mer-1"] = domain.SessionRecord{
		ID:        "mer-1",
		ProjectID: "mer",
		Metadata: domain.SessionMetadata{
			RuntimeLaunchID:              "launch-1",
			SupervisorCapabilityVerifier: "supervisor-verifier-1",
		},
	}
	m := New(st, nil)

	if err := m.MarkSpawned(ctx, "mer-1", domain.SessionMetadata{RuntimeLaunchID: "launch-2"}); err != nil {
		t.Fatalf("MarkSpawned: %v", err)
	}
	got, _, err := st.GetSession(ctx, "mer-1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.Metadata.SupervisorCapabilityVerifier != "" {
		t.Fatalf("old supervisor verifier survived new generation: %q", got.Metadata.SupervisorCapabilityVerifier)
	}
}

func TestMarkSpawnedSameGenerationPreservesAuthenticatedCompletion(t *testing.T) {
	ctx := context.Background()
	authenticatedExitCode := 0
	injectedExitCode := 17
	st := newFakeStore()
	st.sessions["mer-1"] = domain.SessionRecord{
		ID:        "mer-1",
		ProjectID: "mer",
		Metadata: domain.SessionMetadata{
			RuntimeLaunchID:              "launch-1",
			SupervisorCapabilityVerifier: "supervisor-verifier-1",
			SupervisedProcessExitCode:    &authenticatedExitCode,
			SupervisedProcessExitReason:  "exited",
		},
	}
	m := New(st, nil)

	if err := m.MarkSpawned(ctx, "mer-1", domain.SessionMetadata{
		RuntimeLaunchID:              "launch-1",
		SupervisorCapabilityVerifier: "injected-verifier",
		SupervisedProcessExitCode:    &injectedExitCode,
		SupervisedProcessExitReason:  "injected",
	}); err != nil {
		t.Fatalf("MarkSpawned: %v", err)
	}
	got, _, err := st.GetSession(ctx, "mer-1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.Metadata.SupervisorCapabilityVerifier != "supervisor-verifier-1" {
		t.Fatalf("same generation replaced verifier: %q", got.Metadata.SupervisorCapabilityVerifier)
	}
	if got.Metadata.SupervisedProcessExitCode == nil || *got.Metadata.SupervisedProcessExitCode != 0 {
		t.Fatalf("same generation lost authenticated exit code: %v", got.Metadata.SupervisedProcessExitCode)
	}
	if got.Metadata.SupervisedProcessExitReason != "exited" {
		t.Fatalf("same generation replaced exit reason: %q", got.Metadata.SupervisedProcessExitReason)
	}
}

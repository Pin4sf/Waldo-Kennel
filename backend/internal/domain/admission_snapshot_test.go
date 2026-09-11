package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAdmissionSnapshotValidatesProcessExitCompletionBoundary(t *testing.T) {
	policy := AttemptExecutionPolicy{
		OutcomeID: "out-1", PlanRevisionID: "plan-1", WorkUnitID: "wu-1", ContractRevisionNumber: 1,
		RunBriefCoreDigest: strings.Repeat("a", 64), RequiredCapabilities: []string{CapabilityWorktreeRead},
		Grants: []CapabilityGrant{{ID: "read", Name: CapabilityWorktreeRead, Scope: "worktree/*"}},
	}
	digest, err := policy.Digest()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(AdmissionSnapshot{
		SnapshotVersion: AdmissionSnapshotVersion, Harness: string(HarnessCodex),
		ModelSelection: ExecutionBindingModelProviderDefault, WorkUnitID: "wu-1", Mode: SessionModeTUI,
		RunBriefCoreDigest: strings.Repeat("a", 64), RunBriefCompiledDigest: "compiled", ExecutionPolicy: policy,
		ExecutionPolicyDigest: digest, SessionID: "session-1", CompletionBoundary: AttemptCompletionProcessExit,
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := ParseAdmissionSnapshot(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	ref := AttemptSessionRef{
		ID: "ref-1", AttemptID: "att-1", Seq: 1, SessionID: "session-1", Harness: HarnessCodex, Mode: SessionModeTUI,
		RunBriefCoreDigest: strings.Repeat("a", 64), RunBriefCompiledDigest: "compiled", AdmissionSnapshot: string(raw),
	}
	rec := SessionRecord{ID: "session-1", Harness: HarnessCodex, Mode: SessionModeTUI}
	if err := snapshot.ValidateSession(rec, ref, digest); err != nil {
		t.Fatal(err)
	}
	if snapshot.CompletionBoundary != AttemptCompletionProcessExit {
		t.Fatalf("completion boundary = %q", snapshot.CompletionBoundary)
	}
}

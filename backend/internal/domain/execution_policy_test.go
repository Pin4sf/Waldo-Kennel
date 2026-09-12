package domain

import "testing"

func TestBuildAttemptExecutionPolicySelectsUnitGrantsAndIsStable(t *testing.T) {
	plan := PlanRevision{
		ID:                     "plan-1",
		OutcomeID:              "out-1",
		ContractRevisionNumber: 4,
		Grants: []CapabilityGrant{
			{ID: "write", Name: CapabilityWorktreeWrite, Scope: "worktree/*"},
			{ID: "read", Name: CapabilityWorktreeRead, Scope: "worktree/*"},
			{ID: "exec", Name: CapabilityWorktreeExec, Scope: "worktree/*"},
		},
	}
	unit := WorkUnit{ID: "wu-1", RequiredCapabilities: []string{CapabilityWorktreeExec, CapabilityWorktreeRead, CapabilityWorktreeWrite}}

	policy, err := BuildAttemptExecutionPolicy("out-1", plan, unit, "brief-digest")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{CapabilityWorktreeExec, CapabilityWorktreeRead, CapabilityWorktreeWrite}
	for i, got := range policy.RequiredCapabilities {
		if got != want[i] {
			t.Fatalf("required capabilities = %v, want %v", policy.RequiredCapabilities, want)
		}
	}
	if len(policy.Grants) != 3 || policy.Grants[0].Name != CapabilityWorktreeExec {
		t.Fatalf("normalized grants = %+v", policy.Grants)
	}
	first, err := policy.Digest()
	if err != nil {
		t.Fatal(err)
	}
	second, err := policy.Digest()
	if err != nil || first != second {
		t.Fatalf("policy digest is not stable: %q %q %v", first, second, err)
	}
}

func TestBuildAttemptExecutionPolicyFreezesApprovedChecksInDigest(t *testing.T) {
	plan := PlanRevision{ID: "plan-1", OutcomeID: "out-1", ContractRevisionNumber: 1, Grants: []CapabilityGrant{{ID: "exec", Name: CapabilityWorktreeExec, Scope: "worktree/*"}, {ID: "read", Name: CapabilityWorktreeRead, Scope: "worktree/*"}, {ID: "write", Name: CapabilityWorktreeWrite, Scope: "worktree/*"}}}
	unit := WorkUnit{ID: "wu-1", RequiredCapabilities: []string{CapabilityWorktreeExec, CapabilityWorktreeRead, CapabilityWorktreeWrite}, Checks: []ApprovedCheck{{ID: "check-b", CriterionID: "criterion-1", Argv: []string{"go", "test", "./..."}, TimeoutSeconds: 60}}}
	policy, err := BuildAttemptExecutionPolicy("out-1", plan, unit, "brief")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := policy.Digest()
	unit.Checks[0].Argv[1] = "vet"
	after, _ := policy.Digest()
	if before != after || policy.ApprovedChecks[0].Argv[1] != "test" {
		t.Fatalf("policy did not deep-freeze checks: %+v", policy.ApprovedChecks)
	}
	policy.ApprovedChecks[0].Argv[1] = "vet"
	changed, _ := policy.Digest()
	if changed == before {
		t.Fatal("check vector was absent from policy digest")
	}
}

func TestAttemptExecutionPolicyFreezesWorkspaceForRecovery(t *testing.T) {
	policy := AttemptExecutionPolicy{
		OutcomeID: "out-1", PlanRevisionID: "plan-1", WorkUnitID: "wu-1", ContractRevisionNumber: 1, RunBriefCoreDigest: "brief",
		RequiredCapabilities: []string{CapabilityWorktreeRead},
		Grants:               []CapabilityGrant{{ID: "read", Name: CapabilityWorktreeRead, Scope: "worktree/*"}},
	}
	bound, err := policy.BindWorkspaceRoot("/tmp/kennel-workspace-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := bound.ValidateWorkspaceRoot("/tmp/kennel-workspace-b"); err == nil {
		t.Fatal("workspace-bound policy accepted a different recovery root")
	}
	if _, err := bound.BindWorkspaceRoot("/tmp/kennel-workspace-b"); err == nil {
		t.Fatal("workspace-bound policy was rebound")
	}
	unboundDigest, _ := policy.Digest()
	boundDigest, _ := bound.Digest()
	if unboundDigest == boundDigest {
		t.Fatal("workspace root was absent from policy digest")
	}
}

func TestBuildAttemptExecutionPolicyDoesNotUseUnrequestedPlanGrant(t *testing.T) {
	plan := PlanRevision{
		ID:                     "plan-1",
		OutcomeID:              "out-1",
		ContractRevisionNumber: 1,
		Grants: []CapabilityGrant{
			{ID: "read", Name: CapabilityWorktreeRead, Scope: "worktree/*"},
			{ID: "write", Name: CapabilityWorktreeWrite, Scope: "worktree/*"},
		},
	}
	policy, err := BuildAttemptExecutionPolicy("out-1", plan, WorkUnit{ID: "wu-1", RequiredCapabilities: []string{CapabilityWorktreeRead}}, "brief")
	if err != nil {
		t.Fatal(err)
	}
	if len(policy.Grants) != 1 || policy.Grants[0].Name != CapabilityWorktreeRead {
		t.Fatalf("policy widened to plan grants: %+v", policy.Grants)
	}
}

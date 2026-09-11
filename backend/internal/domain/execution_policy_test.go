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

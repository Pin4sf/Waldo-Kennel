package domain

import (
	"strings"
	"testing"
)

func validWorkUnit() WorkUnit {
	return WorkUnit{
		ID:                      WorkUnitID("wu-test"),
		Kind:                    WorkUnitDirect,
		Title:                   "Build and prove the feature",
		ContractRevisionNumber:  1,
		OutputSummary:           "Working local feature in the isolated worktree",
		EvidenceChecks:          []string{"deterministic test suite passes"},
		VerificationRequirement: "verification runs outside the producer session",
		StopConditions:          []string{"stop before any remote effect or unapproved dependency"},
	}
}

func validGrants() []CapabilityGrant {
	return []CapabilityGrant{
		{ID: CapabilityGrantID("cg-read"), Name: CapabilityWorktreeRead, Scope: "worktree/*"},
		{ID: CapabilityGrantID("cg-write"), Name: CapabilityWorktreeWrite, Scope: "worktree/*"},
		{ID: CapabilityGrantID("cg-exec"), Name: CapabilityWorktreeExec, Scope: "worktree/*"},
	}
}

func validPlanRevision() PlanRevision {
	return PlanRevision{
		ID:                     PlanRevisionID("plan-test"),
		OutcomeID:              OutcomeID("out-test"),
		Number:                 1,
		ContractRevisionNumber: 1,
		Status:                 PlanStatusProposed,
		Summary:                "One direct unit",
		WorkUnits:              []WorkUnit{validWorkUnit()},
		Grants:                 validGrants(),
		RunBriefCoreDigest:     strings.Repeat("a", 64),
	}
}

func TestPlanRevisionValidation(t *testing.T) {
	t.Run("accepts a well-formed v0 plan", func(t *testing.T) {
		plan := validPlanRevision()
		if err := plan.Validate(); err != nil {
			t.Fatalf("Validate() = %v, want nil", err)
		}
	})

	cases := []struct {
		name    string
		mutate  func(*PlanRevision)
		wantSub string
	}{
		{"missing id", func(p *PlanRevision) { p.ID = "" }, "plan revision id is required"},
		{"missing outcome", func(p *PlanRevision) { p.OutcomeID = "" }, "outcome id is required"},
		{"zero number", func(p *PlanRevision) { p.Number = 0 }, "number must be at least 1"},
		{
			"unbound contract",
			func(p *PlanRevision) { p.ContractRevisionNumber = 0 },
			"bind a contract revision of at least 1",
		},
		{"bad status", func(p *PlanRevision) { p.Status = "completed" }, "unsupported plan status"},
		{
			"two work units",
			func(p *PlanRevision) {
				p.WorkUnits = append(p.WorkUnits, validWorkUnit())
			},
			"exactly one work unit",
		},
		{
			"non-direct unit",
			func(p *PlanRevision) { p.WorkUnits[0].Kind = "delegate" },
			`must be "direct"`,
		},
		{
			"no grants",
			func(p *PlanRevision) { p.Grants = nil },
			"at least one capability grant",
		},
		{
			"duplicate grant",
			func(p *PlanRevision) {
				p.Grants = append(p.Grants, CapabilityGrant{ID: CapabilityGrantID("cg-dup"), Name: CapabilityWorktreeRead, Scope: "worktree/*"})
			},
			`duplicate capability grant`,
		},
		{
			"digest absent",
			func(p *PlanRevision) { p.RunBriefCoreDigest = "" },
			"run brief core digest",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := validPlanRevision()
			tc.mutate(&plan)
			err := plan.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("Validate() = %v, want substring %q", err, tc.wantSub)
			}
		})
	}
}

func TestBindsCurrentContract(t *testing.T) {
	plan := validPlanRevision()
	if !plan.BindsCurrentContract(1) {
		t.Fatal("plan bound to r1 must bind current r1")
	}
	if plan.BindsCurrentContract(2) {
		t.Fatal("plan bound to r1 must not bind current r2: material change forces a new brief")
	}
}

func TestComputeRunBriefCoreDigest(t *testing.T) {
	digestFor := func(mutate func(*ContractRevision, *WorkUnit, *[]CapabilityGrant)) string {
		revision := ContractRevision{
			ID:              ContractRevisionID("cr-1"),
			OutcomeID:       OutcomeID("out-1"),
			Number:          1,
			Goal:            "Record today's protected focus time locally.",
			SuccessCriteria: []string{"one block can be recorded", "restart preserves it"},
			Review:          "deterministic checks plus owner walkthrough",
			Clarification:   "today means local calendar day",
		}
		unit := validWorkUnit()
		grants := validGrants()
		if mutate != nil {
			mutate(&revision, &unit, &grants)
		}
		digest, err := ComputeRunBriefCoreDigest(revision, unit, grants)
		if err != nil {
			t.Fatalf("ComputeRunBriefCoreDigest() = %v", err)
		}
		return digest
	}

	t.Run("is deterministic across construction order", func(t *testing.T) {
		first := digestFor(nil)
		second := digestFor(func(_ *ContractRevision, _ *WorkUnit, grants *[]CapabilityGrant) {
			g := *grants
			g[0], g[2] = g[2], g[0]
			*grants = g
		})
		if first != second {
			t.Fatalf("digest changed with grant order: %s vs %s", first, second)
		}
	})

	t.Run("changes when the contract changes materially", func(t *testing.T) {
		baseline := digestFor(nil)
		altered := digestFor(func(revision *ContractRevision, _ *WorkUnit, _ *[]CapabilityGrant) {
			revision.SuccessCriteria[0] = "one block can be recorded with a note"
		})
		if baseline == altered {
			t.Fatal("material contract change must yield a different run brief core digest")
		}
	})

	t.Run("changes when authority changes", func(t *testing.T) {
		baseline := digestFor(nil)
		altered := digestFor(func(_ *ContractRevision, _ *WorkUnit, grants *[]CapabilityGrant) {
			*grants = (*grants)[:2] // drop exec: narrowed authority is a different brief
		})
		if baseline == altered {
			t.Fatal("narrowed capability set must yield a different run brief core digest")
		}
	})

	t.Run("rejects invalid inputs instead of hashing them", func(t *testing.T) {
		revision := ContractRevision{ID: ContractRevisionID("cr-bad"), OutcomeID: OutcomeID("out-1"), Number: 1}
		if _, err := ComputeRunBriefCoreDigest(revision, validWorkUnit(), validGrants()); err == nil {
			t.Fatal("expected error for contract without goal/criteria/review")
		}
	})
}

func TestAuthorityIntersection(t *testing.T) {
	contractLayer := []string{CapabilityWorktreeRead, CapabilityWorktreeWrite, CapabilityWorktreeExec, CapabilityWorktreeRead}
	policyCeiling := []string{CapabilityWorktreeRead, CapabilityWorktreeWrite, CapabilityWorktreeExec}
	admissionLayer := []string{CapabilityWorktreeRead, CapabilityWorktreeExec} // runtime cannot offer write

	got := AuthorityIntersection(contractLayer, policyCeiling, admissionLayer)
	want := []string{CapabilityWorktreeExec, CapabilityWorktreeRead}
	if len(got) != len(want) {
		t.Fatalf("AuthorityIntersection() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("AuthorityIntersection() = %v, want %v", got, want)
		}
	}

	t.Run("lower layer cannot widen an upper layer", func(t *testing.T) {
		upper := []string{CapabilityWorktreeRead}
		lower := []string{CapabilityWorktreeRead, CapabilityWorktreeWrite, CapabilityWorktreeExec}
		got := AuthorityIntersection(upper, lower)
		if len(got) != 1 || got[0] != CapabilityWorktreeRead {
			t.Fatalf("AuthorityIntersection(upper, lower) = %v, want read only", got)
		}
	})

	t.Run("empty layer fails closed to nothing", func(t *testing.T) {
		got := AuthorityIntersection([]string{CapabilityWorktreeRead}, nil)
		if len(got) != 0 {
			t.Fatalf("AuthorityIntersection with empty layer = %v, want none", got)
		}
	})
}

func TestGrantsFailClosed(t *testing.T) {
	authoritative := []string{CapabilityWorktreeRead, CapabilityWorktreeWrite}
	grants := []CapabilityGrant{
		{ID: CapabilityGrantID("cg-1"), Name: CapabilityWorktreeRead, Scope: "worktree/*"},
		{ID: CapabilityGrantID("cg-2"), Name: "network.fetch", Scope: "worktree/*"},
	}
	err := GrantsFailClosed(grants, authoritative)
	if err == nil || !strings.Contains(err.Error(), `"network.fetch"`) {
		t.Fatalf("GrantsFailClosed() = %v, want offender named", err)
	}

	if err := GrantsFailClosed(validGrants(), V0RequiredCapabilities); err != nil {
		t.Fatalf("GrantsFailClosed(v0 trio) = %v, want nil", err)
	}
}

func TestMissingRequiredCapabilities(t *testing.T) {
	partial := []CapabilityGrant{
		{ID: CapabilityGrantID("cg-1"), Name: CapabilityWorktreeRead, Scope: "worktree/*"},
	}
	missing := MissingRequiredCapabilities(partial)
	if len(missing) != 2 {
		t.Fatalf("MissingRequiredCapabilities() = %v, want write+exec missing", missing)
	}
	if missing[0] != CapabilityWorktreeWrite || missing[1] != CapabilityWorktreeExec {
		t.Fatalf("MissingRequiredCapabilities() = %v, want deterministic required order", missing)
	}
}

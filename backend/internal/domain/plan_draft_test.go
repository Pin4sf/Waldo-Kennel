package domain

import "testing"

func validPlanDraftWorkUnit(key string) PlanDraftWorkUnit {
	return PlanDraftWorkUnit{
		Key:             key,
		Title:           "Do " + key,
		OutputSummary:   "A reviewable result for " + key,
		CriteriaCovered: []string{"C1"},
		EvidenceIdeas:   []string{"inspect the resulting repository state"},
	}
}

func TestPlanDraftProposalDependenciesAreGraphTruthNotListOrder(t *testing.T) {
	change := validPlanDraftWorkUnit("change")
	change.DependsOn = []string{"inspect"}
	inspect := validPlanDraftWorkUnit("inspect")
	proposal := PlanDraftProposal{
		Summary:   "Inspect first, then make the bounded change.",
		WorkUnits: []PlanDraftWorkUnit{change, inspect}, // deliberately reverse serialization order
	}
	if err := proposal.Validate(); err != nil {
		t.Fatalf("valid graph should not depend on list order: %v", err)
	}
	order, err := proposal.TopologicalOrder()
	if err != nil {
		t.Fatalf("topological order: %v", err)
	}
	if len(order) != 2 || order[0] != "inspect" || order[1] != "change" {
		t.Fatalf("topological order = %v", order)
	}
}

func TestPlanDraftProposalRejectsCycle(t *testing.T) {
	first := validPlanDraftWorkUnit("inspect")
	second := validPlanDraftWorkUnit("change")
	first.DependsOn = []string{"change"}
	second.DependsOn = []string{"inspect"}
	proposal := PlanDraftProposal{Summary: "cycle", WorkUnits: []PlanDraftWorkUnit{first, second}}
	if err := proposal.Validate(); err == nil {
		t.Fatal("cyclic dependencies should be rejected")
	}
}

func TestPlanDraftProposalRejectsUnknownDependency(t *testing.T) {
	unit := validPlanDraftWorkUnit("inspect")
	unit.DependsOn = []string{"missing"}
	proposal := PlanDraftProposal{Summary: "unknown dependency", WorkUnits: []PlanDraftWorkUnit{unit}}
	if err := proposal.Validate(); err == nil {
		t.Fatal("unknown dependency should be rejected")
	}
}

func TestPlanDraftProposalRejectsDuplicateDependency(t *testing.T) {
	first := validPlanDraftWorkUnit("inspect")
	second := validPlanDraftWorkUnit("change")
	second.DependsOn = []string{"inspect", "inspect"}
	proposal := PlanDraftProposal{Summary: "duplicate edge", WorkUnits: []PlanDraftWorkUnit{first, second}}
	if err := proposal.Validate(); err == nil {
		t.Fatal("duplicate dependency should be rejected")
	}
}

func TestPlanDraftProposalRejectsDuplicateKeys(t *testing.T) {
	proposal := PlanDraftProposal{
		Summary:   "duplicate keys",
		WorkUnits: []PlanDraftWorkUnit{validPlanDraftWorkUnit("same"), validPlanDraftWorkUnit("same")},
	}
	if err := proposal.Validate(); err == nil {
		t.Fatal("duplicate work-unit keys should be rejected")
	}
}

func TestPlanDraftProposalRequiresCriterionCoverage(t *testing.T) {
	unit := validPlanDraftWorkUnit("inspect")
	unit.CriteriaCovered = nil
	proposal := PlanDraftProposal{Summary: "missing coverage", WorkUnits: []PlanDraftWorkUnit{unit}}
	if err := proposal.Validate(); err == nil {
		t.Fatal("work unit without criterion aliases should be rejected")
	}
}

package domain

import "testing"

func validPlanDraftWorkUnit(key string) PlanDraftWorkUnit {
	return PlanDraftWorkUnit{
		Key:                     key,
		Title:                   "Do " + key,
		OutputSummary:           "A reviewable result for " + key,
		EvidenceChecks:          []string{"result exists"},
		VerificationRequirement: "owner reviews the result",
		StopConditions:          []string{"stop before an unapproved external effect"},
		HardCapabilities:        []string{CapabilityWorktreeRead},
	}
}

func TestPlanDraftProposalAcceptsStableSerializableOrder(t *testing.T) {
	first := validPlanDraftWorkUnit("inspect")
	second := validPlanDraftWorkUnit("change")
	second.DependsOn = []string{"inspect"}
	second.HardCapabilities = []string{CapabilityWorktreeRead, CapabilityWorktreeWrite}

	proposal := PlanDraftProposal{
		Summary:     "Inspect first, then make the bounded change.",
		WorkUnits:   []PlanDraftWorkUnit{first, second},
		Assumptions: []string{"the repository is locally available"},
	}
	if err := proposal.Validate(); err != nil {
		t.Fatalf("serializable proposal should be valid: %v", err)
	}
}

func TestPlanDraftProposalRejectsForwardDependency(t *testing.T) {
	first := validPlanDraftWorkUnit("inspect")
	first.DependsOn = []string{"change"}
	second := validPlanDraftWorkUnit("change")
	proposal := PlanDraftProposal{Summary: "invalid order", WorkUnits: []PlanDraftWorkUnit{first, second}}
	if err := proposal.Validate(); err == nil {
		t.Fatal("forward dependency should be rejected for serialized MVP scheduling")
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

func TestPlanDraftProposalRejectsBlankCapability(t *testing.T) {
	unit := validPlanDraftWorkUnit("inspect")
	unit.HardCapabilities = []string{""}
	proposal := PlanDraftProposal{Summary: "bad capability", WorkUnits: []PlanDraftWorkUnit{unit}}
	if err := proposal.Validate(); err == nil {
		t.Fatal("blank hard capability should be rejected")
	}
}

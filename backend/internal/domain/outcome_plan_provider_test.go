package domain

import "testing"

func TestRunBriefDigestIncludesWorkUnitProvider(t *testing.T) {
	revision := ContractRevision{
		ID:              ContractRevisionID("cr-provider"),
		OutcomeID:       OutcomeID("out-provider"),
		Number:          1,
		Goal:            "Implement the provider-bound work unit.",
		SuccessCriteria: []string{"provider identity is frozen into the plan"},
		Review:          "deterministic unit tests",
	}
	unit := validWorkUnit()
	unit.Provider = HarnessClaudeCode
	grants := validGrants()

	claudeDigest, err := ComputeRunBriefCoreDigest(revision, unit, grants)
	if err != nil {
		t.Fatalf("ComputeRunBriefCoreDigest(claude) = %v", err)
	}
	unit.Provider = HarnessCodex
	codexDigest, err := ComputeRunBriefCoreDigest(revision, unit, grants)
	if err != nil {
		t.Fatalf("ComputeRunBriefCoreDigest(codex) = %v", err)
	}
	if claudeDigest == codexDigest {
		t.Fatal("changing only the WorkUnit provider must change the frozen RunBrief digest")
	}
}

func TestLegacyWorkUnitWithoutProviderRemainsStructurallyReadable(t *testing.T) {
	unit := validWorkUnit()
	unit.Provider = ""
	if err := unit.Validate(); err != nil {
		t.Fatalf("legacy provider-less WorkUnit must remain readable, Validate() = %v", err)
	}
}

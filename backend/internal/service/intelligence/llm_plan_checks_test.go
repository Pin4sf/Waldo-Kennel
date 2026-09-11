package intelligence

import (
	"reflect"
	"testing"
)

type proposedCheck = struct {
	CriterionAlias string   `json:"criterionAlias"`
	Argv           []string `json:"argv"`
	TimeoutSeconds int64    `json:"timeoutSeconds"`
}

// TestPlanDraftChecks_PreservesTheArgumentVectorExactly is the boundary this
// normalization must not cross. Program arguments are not labels: trimming
// one changes what the command does, and dropping an empty one shifts every
// argument after it, so the command that runs is not the command reviewed.
func TestPlanDraftChecks_PreservesTheArgumentVectorExactly(t *testing.T) {
	cases := []struct {
		name string
		argv []string
	}{
		{name: "an argument whose whitespace is meaningful", argv: []string{"grep", "-F", " needle "}},
		{name: "an argument that is only whitespace", argv: []string{"printf", "%s", " "}},
		{name: "an empty positional argument", argv: []string{"printf", "%s", "", "tail"}},
		{name: "an argument that looks like a flag with padding", argv: []string{"go", "test", " -run=X"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			checks := planDraftChecks([]proposedCheck{{CriterionAlias: " C1 ", Argv: tc.argv, TimeoutSeconds: 30}})
			if len(checks) != 1 {
				t.Fatalf("checks = %d, want the proposal preserved for validation", len(checks))
			}
			if !reflect.DeepEqual(checks[0].Argv, tc.argv) {
				t.Fatalf("argv = %#v, want %#v exactly", checks[0].Argv, tc.argv)
			}
			// The alias is a label, not an argument, so trimming it is safe.
			if checks[0].CriterionAlias != "C1" {
				t.Fatalf("criterion alias = %q, want the trimmed label", checks[0].CriterionAlias)
			}
		})
	}
}

// TestPlanDraftChecks_DoesNotCopyTheProposalsBackingArray keeps a later
// mutation of the decoded reply from reaching an approved command.
func TestPlanDraftChecks_DoesNotCopyTheProposalsBackingArray(t *testing.T) {
	argv := []string{"go", "test", "./..."}
	checks := planDraftChecks([]proposedCheck{{CriterionAlias: "C1", Argv: argv, TimeoutSeconds: 30}})
	argv[2] = "./internal/..."
	if checks[0].Argv[2] != "./..." {
		t.Fatalf("argv followed a later mutation of the decoded reply: %#v", checks[0].Argv)
	}
}

// TestPlanDraftChecks_KeepsAPartialProposalForExplicitRefusal makes sure a
// malformed suggestion reaches validation with a reason instead of vanishing.
func TestPlanDraftChecks_KeepsAPartialProposalForExplicitRefusal(t *testing.T) {
	checks := planDraftChecks([]proposedCheck{
		{CriterionAlias: "C1", Argv: nil, TimeoutSeconds: 30},
		{CriterionAlias: "", Argv: []string{"go", "test"}, TimeoutSeconds: 30},
		{CriterionAlias: "", Argv: nil, TimeoutSeconds: 30},
	})
	// The first two are wrong in a way the owner should see stated; only the
	// wholly empty third proposed nothing at all.
	if len(checks) != 2 {
		t.Fatalf("checks = %d, want the two partial proposals kept for refusal", len(checks))
	}
}

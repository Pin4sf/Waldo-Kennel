package planquality_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/planquality"
)

// greetingFixtures builds the pair the committed live planning evidence turned
// on: a criterion about the exact greeting, a workspace that prints the wrong
// one, and a workspace that prints the right one.
func greetingFixtures(t *testing.T) (knownWrong, correct string) {
	t.Helper()
	knownWrong, correct = filepath.Join(t.TempDir(), "wrong"), filepath.Join(t.TempDir(), "right")
	for dir, greeting := range map[string]string{knownWrong: "Hello World\n", correct: "Hello Kennel\n"} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "greeting.txt"), []byte(greeting), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	return knownWrong, correct
}

func check(argv ...string) domain.ApprovedCheck {
	return domain.ApprovedCheck{ID: "chk-eval", CriterionID: "crit-greeting", Argv: argv, TimeoutSeconds: 30}
}

// TestEvaluate_GradesAProposedCheckByWhetherItSeparatesTheFixtures is the
// falsifier the planning review asked for. The vacuous row is the exact shape
// of the recorded live failure: a command that exits zero whatever the work did.
func TestEvaluate_GradesAProposedCheckByWhetherItSeparatesTheFixtures(t *testing.T) {
	knownWrong, correct := greetingFixtures(t)
	const criterion = "greeting.txt contains the greeting Hello Kennel"

	cases := []struct {
		name  string
		check domain.ApprovedCheck
		want  planquality.Verdict
	}{
		{
			// The committed live evidence: a check that prints the desired
			// string instead of testing for it. Structurally valid, and proof
			// of nothing.
			name:  "printing the greeting proves nothing",
			check: check("echo", "Hello Kennel"),
			want:  planquality.Vacuous,
		},
		{
			name:  "searching the artifact for the greeting separates them",
			check: check("grep", "-q", "Hello Kennel", "greeting.txt"),
			want:  planquality.Discriminating,
		},
		{
			name:  "searching for the wrong greeting measures it backwards",
			check: check("grep", "-q", "Hello World", "greeting.txt"),
			want:  planquality.Inverted,
		},
		{
			name:  "searching a file that is not there can never pass",
			check: check("grep", "-q", "Hello Kennel", "absent.txt"),
			want:  planquality.Broken,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := planquality.Evaluate(context.Background(), planquality.Case{
				Name: tc.name, Criterion: criterion, Check: tc.check,
				KnownWrongDir: knownWrong, CorrectDir: correct,
			})
			if result.Verdict != tc.want {
				t.Fatalf("verdict = %q, want %q\nknown-wrong: %+v\ncorrect: %+v\n%s",
					result.Verdict, tc.want, result.KnownWrong, result.Correct, result.Explanation)
			}
			if result.Explanation == "" {
				t.Fatal("a grade without an explanation cannot improve a proposal")
			}
			// Only a demonstrated separation may be relied on, and the three
			// other grades are each a different repair.
			if result.Verdict.Trustworthy() != (tc.want == planquality.Discriminating) {
				t.Fatalf("%q reported trustworthy = %v", result.Verdict, result.Verdict.Trustworthy())
			}
		})
	}
}

// TestEvaluate_RefusesToGradeWhatTheControlPlaneWouldRefuseToApprove keeps the
// evaluator from blessing a command the Plan compiler would reject anyway. A
// grade on an unapprovable check would be a measurement of something that can
// never run.
func TestEvaluate_RefusesToGradeWhatTheControlPlaneWouldRefuseToApprove(t *testing.T) {
	knownWrong, correct := greetingFixtures(t)
	// A shell would turn one reviewed command into an unreviewable program, so
	// the domain refuses it and so must the evaluator.
	result := planquality.Evaluate(context.Background(), planquality.Case{
		Name: "shell", Criterion: "anything", Check: check("sh", "-c", "grep -q 'Hello Kennel' greeting.txt"),
		KnownWrongDir: knownWrong, CorrectDir: correct,
	})
	if result.Verdict != planquality.Unusable {
		t.Fatalf("verdict = %q, want unusable", result.Verdict)
	}
	if result.Correct.Ran || result.KnownWrong.Ran {
		t.Fatal("an unapprovable check was executed against a fixture")
	}
}

// TestEvaluate_AnUnresolvableCommandIsUngradableRatherThanFailing keeps a host
// problem from reading as a verdict about the check. "This machine has no such
// command" is not "this check is broken".
func TestEvaluate_AnUnresolvableCommandIsUngradableRatherThanFailing(t *testing.T) {
	knownWrong, correct := greetingFixtures(t)
	result := planquality.Evaluate(context.Background(), planquality.Case{
		Name: "absent tool", Criterion: "anything",
		Check:         check("kennel-no-such-command-exists"),
		KnownWrongDir: knownWrong, CorrectDir: correct,
	})
	if result.Verdict != planquality.Unusable {
		t.Fatalf("verdict = %q, want unusable", result.Verdict)
	}
}

// TestEvaluate_ATimeoutIsNotAPass keeps a check that hangs from being graded as
// having separated anything. It ran, it did not pass, and on both fixtures that
// is broken rather than discriminating.
func TestEvaluate_ATimeoutIsNotAPass(t *testing.T) {
	knownWrong, correct := greetingFixtures(t)
	hang := domain.ApprovedCheck{
		ID: "chk-hang", CriterionID: "crit-greeting", Argv: []string{"sleep", "5"}, TimeoutSeconds: 1,
	}
	result := planquality.Evaluate(context.Background(), planquality.Case{
		Name: "hanging check", Criterion: "anything", Check: hang,
		KnownWrongDir: knownWrong, CorrectDir: correct,
	})
	if result.Verdict != planquality.Broken {
		t.Fatalf("verdict = %q, want broken", result.Verdict)
	}
	if !result.KnownWrong.TimedOut || result.KnownWrong.Passed {
		t.Fatalf("known-wrong observation = %+v, want a timed-out non-pass", result.KnownWrong)
	}
}

// TestGrade_ReportsEveryCheckThatMustNotBeTrusted is what a planning-quality
// gate consumes: the subset that cannot support a criterion, with the reason.
func TestGrade_ReportsEveryCheckThatMustNotBeTrusted(t *testing.T) {
	knownWrong, correct := greetingFixtures(t)
	const criterion = "greeting.txt contains the greeting Hello Kennel"
	report := planquality.Grade(context.Background(), []planquality.Case{
		{Name: "vacuous", Criterion: criterion, Check: check("echo", "Hello Kennel"),
			KnownWrongDir: knownWrong, CorrectDir: correct},
		{Name: "good", Criterion: criterion, Check: check("grep", "-q", "Hello Kennel", "greeting.txt"),
			KnownWrongDir: knownWrong, CorrectDir: correct},
	})

	untrustworthy := report.Untrustworthy()
	if len(untrustworthy) != 1 || untrustworthy[0].Case.Name != "vacuous" {
		t.Fatalf("untrustworthy = %+v, want only the vacuous check", untrustworthy)
	}
	// The rendered report has to name the command, or an owner cannot act on it.
	rendered := report.String()
	for _, want := range []string{"vacuous", "echo Hello Kennel", "discriminating"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("report is missing %q:\n%s", want, rendered)
		}
	}
}

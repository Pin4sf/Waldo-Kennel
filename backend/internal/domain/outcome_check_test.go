package domain

import (
	"strings"
	"testing"
)

func checkWithArgv(argv []string) ApprovedCheck {
	return ApprovedCheck{ID: "chk-1", CriterionID: "crit-a", Argv: argv, TimeoutSeconds: 30}
}

// TestApprovedCheckValidate_AcceptsMeaningfulWhitespaceInsideArguments keeps
// the vector executable as written. A pattern with real spaces is an ordinary
// argument, not a formatting accident.
func TestApprovedCheckValidate_AcceptsMeaningfulWhitespaceInsideArguments(t *testing.T) {
	for _, argv := range [][]string{
		{"grep", "-F", " needle "},
		{"go", "test", "-run", "Test A|Test B"},
		{"printf", "%s\n", "two words"},
	} {
		if err := checkWithArgv(argv).Validate(); err != nil {
			t.Fatalf("argv %#v was refused: %v", argv, err)
		}
	}
}

// TestApprovedCheckValidate_RefusesArgumentsItWillNotSilentlyRewrite is the
// other half of preserving the vector: anything that cannot be executed as
// written is refused with a reason, never trimmed or dropped, because
// dropping an argument shifts every one after it.
func TestApprovedCheckValidate_RefusesArgumentsItWillNotSilentlyRewrite(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want string
	}{
		{name: "an empty positional argument", argv: []string{"printf", "%s", "", "tail"}, want: "empty or whitespace only"},
		{name: "a whitespace-only argument", argv: []string{"printf", "%s", "   "}, want: "empty or whitespace only"},
		{name: "an argument containing NUL", argv: []string{"printf", "a\x00b"}, want: "contains NUL"},
		{name: "an executable with surrounding whitespace", argv: []string{" go ", "test"}, want: "surrounding whitespace"},
		{name: "an executable given as a path", argv: []string{"/usr/bin/go", "test"}, want: "not a path"},
		{name: "a shell", argv: []string{"bash", "-c", "true"}, want: "may not run a shell"},
		{name: "no command at all", argv: nil, want: "has no command"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkWithArgv(tc.argv).Validate()
			if err == nil {
				t.Fatalf("argv %#v was accepted", tc.argv)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("refusal %q does not say why (want %q)", err, tc.want)
			}
		})
	}
}

// TestApprovedCheckValidate_ReportsTheArgumentPositionItRefused so an owner
// can find the offending argument in a long command.
func TestApprovedCheckValidate_ReportsTheArgumentPositionItRefused(t *testing.T) {
	err := checkWithArgv([]string{"printf", "%s", "ok", "", "tail"}).Validate()
	if err == nil {
		t.Fatal("an empty argument was accepted")
	}
	if !strings.Contains(err.Error(), "argument 3") {
		t.Fatalf("refusal %q does not name the offending position", err)
	}
}

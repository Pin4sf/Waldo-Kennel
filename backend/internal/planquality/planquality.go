// Package planquality grades a proposed deterministic check by asking the one
// question a zero exit code cannot answer on its own: would this command have
// noticed if the work had not been done?
//
// The control plane already re-decides everything structural about a proposed
// check — the criterion it names, its identity, its bound, and that it is an
// argument vector rather than a shell. None of that establishes that the
// command's result depends on the criterion. A check that prints a greeting and
// exits zero satisfies every structural rule and proves nothing, which is
// exactly the failure recorded against the committed live planning evidence.
//
// The falsifier is a known-wrong baseline. Run the same command against a
// workspace that does not satisfy the criterion and against one that does. A
// check that passes both is not evidence; a check that fails both is not
// working; only a check that separates them can support the criterion.
//
// This is evaluation, not runtime policy. It executes owner-supplied fixtures,
// never a real workspace, and it decides nothing about acceptance. Its output is
// a measurement to improve proposals with — the deliberate alternative to
// encoding a denylist of commands that "do not count", which would be folklore
// and would be wrong for the next command nobody listed.
package planquality

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// Verdict grades one check against one known-wrong/correct fixture pair.
type Verdict string

const (
	// Discriminating means the check failed the known-wrong baseline and
	// passed the correct one. Only this verdict supports trusting the check.
	Discriminating Verdict = "discriminating"
	// Vacuous means the check passed both. Its exit status is independent of
	// whether the criterion holds, so a zero exit from it is not proof.
	Vacuous Verdict = "vacuous"
	// Broken means the check failed both, so it can never report success —
	// usually a command that cannot run where it is pointed.
	Broken Verdict = "broken"
	// Inverted means the check passed the known-wrong baseline and failed the
	// correct one. It measures something, but backwards.
	Inverted Verdict = "inverted"
	// Unusable means the check could not be evaluated at all: the control
	// plane would refuse it, or the fixture could not be run.
	Unusable Verdict = "unusable"
)

// Trustworthy reports whether this verdict permits treating the check as
// criterion evidence. Only a demonstrated separation does.
func (v Verdict) Trustworthy() bool { return v == Discriminating }

// Case is one grading: a criterion, the check proposed for it, and the two
// fixture workspaces that decide whether the check can tell them apart.
//
// The criterion text is carried for reporting only. Nothing here interprets it;
// the fixtures are what encode what "wrong" and "correct" mean, which is why an
// owner writes them and a model does not.
type Case struct {
	Name      string
	Criterion string
	Check     domain.ApprovedCheck
	// KnownWrongDir is a workspace where the criterion is false.
	KnownWrongDir string
	// CorrectDir is a workspace where the criterion is true.
	CorrectDir string
}

// Observation is what one fixture run produced.
type Observation struct {
	Ran      bool
	Passed   bool
	ExitCode int
	TimedOut bool
	Output   string
	Err      string
}

// Result is one graded Case.
type Result struct {
	Case        Case
	Verdict     Verdict
	KnownWrong  Observation
	Correct     Observation
	Explanation string
}

// maxCapturedOutput bounds what a fixture run contributes to a report. The
// grade comes from exit status; output is for a human reading the failure.
const maxCapturedOutput = 4 << 10

// Evaluate grades one Case by running the check in both fixtures.
//
// The check is validated first with the same domain rules the Plan compiler
// applies, so this can never grade — and so never bless — a command the control
// plane would refuse to approve.
func Evaluate(ctx context.Context, c Case) Result {
	result := Result{Case: c}
	if err := c.Check.Validate(); err != nil {
		result.Verdict = Unusable
		result.Explanation = "the control plane would refuse this check: " + err.Error()
		return result
	}
	if strings.TrimSpace(c.KnownWrongDir) == "" || strings.TrimSpace(c.CorrectDir) == "" {
		result.Verdict = Unusable
		result.Explanation = "grading needs both a known-wrong and a correct fixture"
		return result
	}
	result.KnownWrong = run(ctx, c.Check, c.KnownWrongDir)
	result.Correct = run(ctx, c.Check, c.CorrectDir)
	result.Verdict, result.Explanation = grade(result.KnownWrong, result.Correct)
	return result
}

func grade(knownWrong, correct Observation) (Verdict, string) {
	if !knownWrong.Ran || !correct.Ran {
		return Unusable, "the check could not be executed against both fixtures: " +
			strings.TrimSpace(knownWrong.Err+" "+correct.Err)
	}
	switch {
	case !knownWrong.Passed && correct.Passed:
		return Discriminating, fmt.Sprintf(
			"the check failed the known-wrong baseline (exit %d) and passed the correct fixture, so its result depends on the criterion",
			knownWrong.ExitCode)
	case knownWrong.Passed && correct.Passed:
		return Vacuous, "the check passed the known-wrong baseline too, so a zero exit from it says nothing about the criterion"
	case !knownWrong.Passed && !correct.Passed:
		return Broken, fmt.Sprintf(
			"the check failed the correct fixture as well (exit %d), so it can never report the criterion as met",
			correct.ExitCode)
	default:
		return Inverted, "the check passed the known-wrong baseline and failed the correct one, so it measures the criterion backwards"
	}
}

// run executes one check in one fixture directory.
//
// The command is the approved argv with no shell, the working directory is the
// fixture, and the timeout is the check's own bound. The environment is left as
// inherited: a check is graded as it will actually be run, and a fixture that
// only passes under a scrubbed environment would be a misleading grade.
func run(ctx context.Context, check domain.ApprovedCheck, dir string) Observation {
	timeout := time.Duration(check.TimeoutSeconds) * time.Second
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, check.Argv[0], check.Argv[1:]...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if len(output) > maxCapturedOutput {
		output = append(output[:maxCapturedOutput], "…"...)
	}
	observation := Observation{Output: string(output)}
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		observation.Ran, observation.TimedOut = true, true
		observation.Err = fmt.Sprintf("timed out after %ds", check.TimeoutSeconds)
		return observation
	}
	var exit *exec.ExitError
	switch {
	case err == nil:
		observation.Ran, observation.Passed = true, true
	case errors.As(err, &exit):
		observation.Ran, observation.ExitCode = true, exit.ExitCode()
	default:
		// The command never started — an unresolvable executable, most often.
		// That is not a failing check; it is an ungradable one.
		observation.Err = err.Error()
	}
	return observation
}

// Report grades several cases and summarises how many can be trusted.
type Report struct {
	Results []Result
}

// Grade evaluates every case in order.
func Grade(ctx context.Context, cases []Case) Report {
	report := Report{Results: make([]Result, 0, len(cases))}
	for _, c := range cases {
		report.Results = append(report.Results, Evaluate(ctx, c))
	}
	return report
}

// Untrustworthy returns the graded cases that must not be relied on as
// criterion proof, which is the whole point of running the report.
func (r Report) Untrustworthy() []Result {
	var out []Result
	for _, result := range r.Results {
		if !result.Verdict.Trustworthy() {
			out = append(out, result)
		}
	}
	return out
}

// String renders one line per graded case.
func (r Report) String() string {
	var b strings.Builder
	for _, result := range r.Results {
		fmt.Fprintf(&b, "%-14s %s\n    criterion: %s\n    command:   %s\n    %s\n",
			result.Verdict, result.Case.Name, result.Case.Criterion,
			strings.Join(result.Case.Check.Argv, " "), result.Explanation)
	}
	return b.String()
}

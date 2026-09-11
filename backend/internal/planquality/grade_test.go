package planquality

import "testing"

// TestGrade_RejectsAnInterruptedRunBeforeReadingItAsPassOrFail is the trap an
// earlier version of this grader fell into, tested where it can be made
// deterministic.
//
// A timeout is not a failure. A command that times out on the known-wrong
// fixture and exits zero on the correct one looks exactly like discrimination —
// one side did not pass, the other did — and that asymmetric shape is the
// dangerous one. The symmetric case only ever looked broken, which is why
// covering it alone missed this. Nobody knows what a timed-out run would have
// concluded, so grading it trustworthy would bless a check on the strength of a
// hang.
//
// Building the asymmetry through real commands is not portable: BSD and GNU
// xargs disagree, and a fifo is unix-only. The observations are constructed
// instead, which is also the only way to cover the interrupted-but-not-timed-out
// case at all.
func TestGrade_RejectsAnInterruptedRunBeforeReadingItAsPassOrFail(t *testing.T) {
	finished := func(passed bool, exit int) Observation {
		return Observation{Ran: true, Conclusive: true, Passed: passed, ExitCode: exit}
	}
	timedOut := Observation{Ran: true, TimedOut: true, Err: "timed out after 1s"}
	cancelled := Observation{Ran: true, Err: "interrupted: context canceled"}

	cases := []struct {
		name                string
		knownWrong, correct Observation
		want                Verdict
	}{
		{
			name: "the asymmetric timeout that used to read as discrimination",
			// This is the exact shape: known-wrong did not pass, correct did.
			knownWrong: timedOut, correct: finished(true, 0),
			want: Unusable,
		},
		{
			name:       "a timeout on the correct side is equally ungradable",
			knownWrong: finished(false, 1), correct: timedOut,
			want: Unusable,
		},
		{
			name:       "so is a cancelled run that never timed out",
			knownWrong: cancelled, correct: finished(true, 0),
			want: Unusable,
		},
		{
			name:       "a command that never started is ungradable",
			knownWrong: Observation{Err: "exec: not found"}, correct: finished(true, 0),
			want: Unusable,
		},
		// The four real grades, so the interruption guard cannot have swallowed
		// them on the way past.
		{
			name:       "failing wrong and passing correct is discrimination",
			knownWrong: finished(false, 1), correct: finished(true, 0),
			want: Discriminating,
		},
		{
			name:       "passing both is vacuous",
			knownWrong: finished(true, 0), correct: finished(true, 0),
			want: Vacuous,
		},
		{
			name:       "failing both is broken",
			knownWrong: finished(false, 2), correct: finished(false, 2),
			want: Broken,
		},
		{
			name:       "passing wrong and failing correct is inverted",
			knownWrong: finished(true, 0), correct: finished(false, 1),
			want: Inverted,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verdict, explanation := grade(tc.knownWrong, tc.correct)
			if verdict != tc.want {
				t.Fatalf("verdict = %q, want %q (%s)", verdict, tc.want, explanation)
			}
			if explanation == "" {
				t.Fatal("a grade without an explanation cannot improve a proposal")
			}
			if verdict.Trustworthy() != (tc.want == Discriminating) {
				t.Fatalf("%q reported trustworthy = %v", verdict, verdict.Trustworthy())
			}
		})
	}
}

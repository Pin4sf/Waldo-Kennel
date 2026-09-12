package domain

import "testing"

func TestSupervisedExitSucceeded(t *testing.T) {
	zero, nonzero := 0, 17

	cases := []struct {
		name     string
		exitCode *int
		reason   string
		want     bool
	}{
		{"legitimate zero exit", &zero, "exited", true},
		{"nonzero exit", &nonzero, "failed", false},
		{"missing code with exited reason", nil, "exited", false},
		{"contradictory zero exit plus failed reason", &zero, "failed", false},
		{"cancelled with no code", nil, "cancelled", false},
		{"start_failed with no code", nil, "start_failed", false},
		{"unknown with no code", nil, "unknown", false},
		{"nonzero exit mislabeled exited", &nonzero, "exited", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SupervisedExitSucceeded(c.exitCode, c.reason); got != c.want {
				t.Fatalf("SupervisedExitSucceeded(%v, %q) = %v, want %v", c.exitCode, c.reason, got, c.want)
			}
		})
	}
}

func TestSupervisedExitFactsConsistent(t *testing.T) {
	zero, nonzero := 0, 17

	cases := []struct {
		name     string
		exitCode *int
		reason   string
		want     bool
	}{
		{"zero code with exited reason", &zero, "exited", true},
		{"nonzero code with failed reason", &nonzero, "failed", true},
		{"nil code with cancelled reason", nil, "cancelled", true},
		{"zero code with failed reason is contradictory", &zero, "failed", false},
		{"nonzero code with exited reason is contradictory", &nonzero, "exited", false},
		{"nil code with exited reason is contradictory", nil, "exited", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SupervisedExitFactsConsistent(c.exitCode, c.reason); got != c.want {
				t.Fatalf("SupervisedExitFactsConsistent(%v, %q) = %v, want %v", c.exitCode, c.reason, got, c.want)
			}
		})
	}
}

package domain

import (
	"fmt"
	"strings"
)

// ApprovedCheckID identifies one deterministic check frozen into a Plan.
type ApprovedCheckID string

// IsZero reports an unset check id.
func (id ApprovedCheckID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// String renders the identifier.
func (id ApprovedCheckID) String() string { return string(id) }

// ApprovedCheckMaxTimeoutSeconds is named operational policy, not a law of
// Outcomes: a check has to be bounded, and fifteen minutes is long enough for
// a real suite while short enough that a hung check is noticed.
const ApprovedCheckMaxTimeoutSeconds int64 = 900

// ApprovedCheck is one deterministic command the owner authorized when they
// approved a Plan, bound to the criterion it is evidence for.
//
// It is deliberately an argument vector, never a command line. A shell string
// would make the approved authority unreadable — nobody can tell what
// `sh -c "..."` will do — and would let a Plan smuggle arbitrary execution
// past a review that only saw one command. Models may propose these; approval
// is what makes them authority, and the Plan digest freezes them so an
// approved check cannot change under the Attempt that runs it.
type ApprovedCheck struct {
	ID          ApprovedCheckID `json:"id"`
	CriterionID CriterionID     `json:"criterionId"`
	// Argv[0] is a discrete executable name resolved from the check runner's
	// own PATH; it is not a path and not a shell.
	Argv           []string `json:"argv"`
	TimeoutSeconds int64    `json:"timeoutSeconds"`
}

// Validate checks one approved check's structural invariants.
func (c ApprovedCheck) Validate() error {
	if c.ID.IsZero() {
		return fmt.Errorf("approved check id is required")
	}
	if strings.TrimSpace(string(c.CriterionID)) == "" {
		return fmt.Errorf("approved check %s must name the criterion it proves", c.ID)
	}
	if len(c.Argv) == 0 {
		return fmt.Errorf("approved check %s has no command", c.ID)
	}
	if err := validateCheckExecutable(c.ID, c.Argv[0]); err != nil {
		return err
	}
	// Arguments are kept exactly as proposed, so an empty or whitespace-only
	// one is refused here rather than trimmed away. Dropping it would shift
	// every later argument into a different position and run a command nobody
	// reviewed; accepting it would let an approved check carry an argument
	// whose intent cannot be read.
	for position, argument := range c.Argv[1:] {
		if strings.IndexByte(argument, 0) >= 0 {
			return fmt.Errorf("approved check %s argument %d contains NUL", c.ID, position+1)
		}
		if strings.TrimSpace(argument) == "" {
			return fmt.Errorf("approved check %s argument %d is empty or whitespace only; state the argument explicitly", c.ID, position+1)
		}
	}
	if c.TimeoutSeconds <= 0 || c.TimeoutSeconds > ApprovedCheckMaxTimeoutSeconds {
		return fmt.Errorf("approved check %s timeout must be between 1 and %d seconds", c.ID, ApprovedCheckMaxTimeoutSeconds)
	}
	return nil
}

// validateCheckExecutable checks argv[0] on its own terms.
//
// It is not an ordinary argument: it decides what program runs, so it must be
// a bare name the check runner resolves from its own fixed PATH. Surrounding
// whitespace is refused rather than trimmed, because the stored vector is
// what will be executed and a name that only works after trimming is not the
// name that was approved.
func validateCheckExecutable(id ApprovedCheckID, executable string) error {
	if strings.IndexByte(executable, 0) >= 0 {
		return fmt.Errorf("approved check %s executable contains NUL", id)
	}
	if executable == "" || strings.TrimSpace(executable) == "" {
		return fmt.Errorf("approved check %s names no executable", id)
	}
	if executable != strings.TrimSpace(executable) {
		return fmt.Errorf("approved check %s executable %q has surrounding whitespace", id, executable)
	}
	if strings.ContainsAny(executable, `/\`) {
		return fmt.Errorf("approved check %s must name a discrete executable, not a path", id)
	}
	switch strings.ToLower(executable) {
	case "sh", "bash", "zsh", "fish", "cmd", "powershell", "pwsh", "env", "eval", "exec":
		// A shell turns one reviewed command into an unreviewable program.
		return fmt.Errorf("approved check %s may not run a shell", id)
	}
	return nil
}

// ValidateApprovedChecks checks a WorkUnit's whole check set: unique ids, and
// every check bound to a criterion that WorkUnit actually owns.
//
// A check bound to somebody else's criterion would let one WorkUnit's result
// silently satisfy another's obligation.
func ValidateApprovedChecks(checks []ApprovedCheck, ownedCriteria []CriterionID) error {
	owned := make(map[CriterionID]bool, len(ownedCriteria))
	for _, id := range ownedCriteria {
		owned[id] = true
	}
	seen := make(map[ApprovedCheckID]bool, len(checks))
	for _, check := range checks {
		if err := check.Validate(); err != nil {
			return err
		}
		if seen[check.ID] {
			return fmt.Errorf("approved check %s is declared twice", check.ID)
		}
		seen[check.ID] = true
		if !owned[check.CriterionID] {
			return fmt.Errorf("approved check %s names criterion %s, which this WorkUnit does not own", check.ID, check.CriterionID)
		}
	}
	return nil
}

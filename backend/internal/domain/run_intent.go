package domain

// RunIntentDesired is what the owner has authorized the daemon to keep doing
// with an approved Plan.
//
// It is deliberately separate from AttemptStatus. An Attempt describes one
// launch; run intent describes whether serial continuation across the Plan is
// authorized at all. Conflating them is how a restarted daemon ends up either
// abandoning authorized work or resuming work the owner stopped.
type RunIntentDesired string

// The four durable run intents.
const (
	// RunIntentIdle means no continuation is authorized. It is the state
	// before the first Start and after a run completes.
	RunIntentIdle RunIntentDesired = "idle"
	// RunIntentRunning means the daemon may admit each eligible WorkUnit.
	RunIntentRunning RunIntentDesired = "running"
	// RunIntentPaused means no further admission, while work already running
	// is left alone. Pause is not a stop.
	RunIntentPaused RunIntentDesired = "paused"
	// RunIntentCancelled means no further admission and the active Attempt
	// should be terminated. Reaching it again requires a fresh Start.
	RunIntentCancelled RunIntentDesired = "cancelled"
)

// Valid reports whether d is a known desired run state.
func (d RunIntentDesired) Valid() bool {
	switch d {
	case RunIntentIdle, RunIntentRunning, RunIntentPaused, RunIntentCancelled:
		return true
	}
	return false
}

// AdmitsWork reports whether this intent permits admitting a new Attempt.
// Only an explicitly running intent does; unknown or stopped never does.
func (d RunIntentDesired) AdmitsWork() bool { return d == RunIntentRunning }

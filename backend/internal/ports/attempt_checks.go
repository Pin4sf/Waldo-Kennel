package ports

import (
	"context"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// AttemptCheckRequest asks for the approved deterministic checks of one ended
// Attempt to be executed under exactly that Attempt's frozen authority.
//
// The policy is the Attempt's own, not a wider one built for checking: a check
// that can do more than the work it is checking is a hole in the same fence.
type AttemptCheckRequest struct {
	Attempt domain.Attempt
	// Receipt names the retained result the checks are about. Its artifact
	// version is what any resulting proof is bound to.
	Receipt domain.AttemptReceipt
	Policy  domain.AttemptExecutionPolicy
	Checks  []domain.ApprovedCheck
}

// AttemptCheckObservation is what Kennel itself observed one check do.
//
// It is an observation, not a verdict handed in by anything that ran inside
// the workspace: Passed is derived from the exit status of a process this
// daemon launched under an enforcement mechanism it named.
type AttemptCheckObservation struct {
	Check domain.ApprovedCheck
	// ArtifactVersion is the retained result the check examined.
	ArtifactVersion string
	// EnforcedBy names the mechanism that held the boundary, so evidence can
	// never imply a confinement that did not exist. Empty means the check did
	// not run.
	EnforcedBy string
	// Ran is false when the check never launched — an unavailable enforcement
	// mechanism, an invalid command. That is different from a check that ran
	// and failed, and the two must not be reported the same way.
	Ran      bool
	ExitCode int
	Passed   bool

	TimedOut  bool
	Cancelled bool
	// TerminationUnknown means the check's process tree could not be confirmed
	// stopped. Custody is retained; an unknown termination is not a result.
	TerminationUnknown bool
	OutputTruncated    bool
	// Output is bounded and redacted. It is a log excerpt, never the content
	// integrity of the result.
	Output string
	// Unavailable explains a check that could not be run at all.
	Unavailable string

	StartedAt time.Time
	EndedAt   time.Time
}

// AttemptCheckResult reports every observation plus whether the checked bytes
// survived the checking.
type AttemptCheckResult struct {
	Observations []AttemptCheckObservation
	// ObservedArtifactVersion is the workspace's manifest version measured
	// after the checks ran. When it differs from the receipt's, the checks
	// changed the very result they were examining, and a pass taken from the
	// pre-change bytes cannot be attached to the post-change output.
	ObservedArtifactVersion string
	// MeasurementError explains a workspace that could not be re-measured.
	// Not knowing whether the result changed is itself a reason not to bind a
	// pass to it, so this counts as changed.
	MeasurementError string
}

// ArtifactChanged reports checks that mutated the result they were checking,
// or a workspace that could no longer be measured at all.
func (r AttemptCheckResult) ArtifactChanged(retained string) bool {
	if r.MeasurementError != "" {
		return true
	}
	return r.ObservedArtifactVersion != "" && r.ObservedArtifactVersion != retained
}

// AttemptCheckRunner executes an Attempt's approved checks under the enforced
// boundary and reports what was observed.
//
// It is a port because enforcement is host-specific and must fail closed: a
// daemon with no mechanism that can apply the approved limits reports the
// checks as unavailable rather than running them unconfined.
type AttemptCheckRunner interface {
	RunAttemptChecks(context.Context, AttemptCheckRequest) (AttemptCheckResult, error)
}

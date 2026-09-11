package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RunAdmissionFailure is the write-once reason one running authorization could
// not admit its next WorkUnit. A later owner command creates a new generation;
// it never clears or rewrites this history.
type RunAdmissionFailure struct {
	Code       string     `json:"code"`
	Message    string     `json:"message"`
	DetailJSON string     `json:"detailJson"`
	WorkUnitID WorkUnitID `json:"workUnitId"`
	OccurredAt time.Time  `json:"occurredAt"`
}

// Validate checks the required admission failure fields and detail encoding.
func (f RunAdmissionFailure) Validate() error {
	if strings.TrimSpace(f.Code) == "" || strings.TrimSpace(f.Message) == "" || f.WorkUnitID.IsZero() || f.OccurredAt.IsZero() {
		return fmt.Errorf("run admission failure requires code, message, work unit, and timestamp")
	}
	if !json.Valid([]byte(f.DetailJSON)) {
		return fmt.Errorf("run admission failure detail must be valid JSON")
	}
	return nil
}

// RunIntentID identifies one generation of an Outcome's run intent.
type RunIntentID string

// IsZero reports an unset run intent id.
func (id RunIntentID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// RunCommand is an owner command against an Outcome's run intent. It is
// deliberately narrower than the Mission action vocabulary the UI renders:
// those are things an owner can do to an Outcome, these are the four that
// change what the daemon is authorized to keep doing.
type RunCommand string

// The four owner commands.
const (
	RunCommandStart  RunCommand = "start"
	RunCommandPause  RunCommand = "pause"
	RunCommandResume RunCommand = "resume"
	RunCommandCancel RunCommand = "cancel"
)

// Valid reports whether c is a known command.
func (c RunCommand) Valid() bool {
	switch c {
	case RunCommandStart, RunCommandPause, RunCommandResume, RunCommandCancel:
		return true
	}
	return false
}

// OutcomeRunIntent is one durable generation of "what the owner has
// authorized the daemon to keep doing with this approved Plan".
//
// Generations are append-only. A pause is not an edit of the running
// generation, it is a new one — which is what lets a restarted daemon read the
// latest row and know exactly what it may do, and lets a client's stale
// command be refused by generation rather than silently applied.
type OutcomeRunIntent struct {
	ID         RunIntentID
	OutcomeID  OutcomeID
	Generation int64
	Desired    RunIntentDesired
	// Command and the expected-generation fields are admission inputs. They
	// are retained on the value passed to the store so the storage transaction
	// can validate the transition that the service composed; only the request
	// fingerprint is persisted for replay.
	Command                RunCommand
	ExpectedGeneration     int64
	ExpectedGenerationSet  bool
	RequestFingerprint     string
	PlanRevisionID         PlanRevisionID
	ContractRevisionNumber int64
	// RequestKey is the owner command's replay identity. Repeating a key
	// returns this same generation instead of creating another.
	RequestKey  string
	RequestedAt time.Time
	// AcknowledgedAt separates a request from its effect. A pause is
	// acknowledged once no further admission can follow; a cancel once the
	// provider stop is proven. Nil means the request has not taken effect yet.
	AcknowledgedAt   *time.Time
	AdmissionFailure *RunAdmissionFailure
}

// Acknowledged reports whether this intent has taken effect.
func (i OutcomeRunIntent) Acknowledged() bool { return i.AcknowledgedAt != nil }

// AdmitsWork reports whether this intent permits admitting a new Attempt.
func (i OutcomeRunIntent) AdmitsWork() bool { return i.Desired.AdmitsWork() }

// Validate checks one intent generation's invariants.
func (i OutcomeRunIntent) Validate() error {
	if i.ID.IsZero() {
		return fmt.Errorf("run intent id is required")
	}
	if i.OutcomeID.IsZero() {
		return fmt.Errorf("run intent outcome id is required")
	}
	if i.Generation < 1 {
		return fmt.Errorf("run intent generation must be at least 1")
	}
	if !i.Desired.Valid() {
		return fmt.Errorf("run intent desired state %q is invalid", i.Desired)
	}
	// Idle is the absence of authorization and names no Plan. Every other
	// state authorizes something about a specific approved Plan.
	if i.Desired != RunIntentIdle {
		if i.PlanRevisionID.IsZero() {
			return fmt.Errorf("run intent %q must name the approved Plan it applies to", i.Desired)
		}
		if i.ContractRevisionNumber < 1 {
			return fmt.Errorf("run intent %q must name the Contract revision it was authorized under", i.Desired)
		}
	}
	if strings.TrimSpace(i.RequestKey) == "" {
		return fmt.Errorf("run intent request key is required")
	}
	if i.RequestedAt.IsZero() {
		return fmt.Errorf("run intent requested timestamp is required")
	}
	if i.AdmissionFailure != nil {
		if i.Desired != RunIntentRunning {
			return fmt.Errorf("only a running intent may carry an admission failure")
		}
		if err := i.AdmissionFailure.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// NextRunIntent decides the generation an owner command produces.
//
// The transitions are deliberately narrow. Resume is refused from cancelled
// because a cancelled run has had its work stopped: continuing it would resume
// something the owner ended, and a fresh Start is the honest way to ask for
// more work. Pausing an already-paused run, or cancelling a cancelled one, is
// a no-op rather than a new generation, so a double click does not accumulate
// history that says nothing happened.
func NextRunIntent(current RunIntentDesired, command RunCommand) (RunIntentDesired, bool) {
	if !command.Valid() {
		return "", false
	}
	if current == "" {
		current = RunIntentIdle
	}
	switch command {
	case RunCommandStart:
		// Start is allowed from anywhere that is not already running: it is
		// how an owner asks for work after a pause they no longer want, or
		// after a cancellation.
		if current == RunIntentRunning {
			return "", false
		}
		return RunIntentRunning, true
	case RunCommandPause:
		if current != RunIntentRunning {
			return "", false
		}
		return RunIntentPaused, true
	case RunCommandResume:
		if current != RunIntentPaused {
			return "", false
		}
		return RunIntentRunning, true
	case RunCommandCancel:
		if current != RunIntentRunning && current != RunIntentPaused {
			return "", false
		}
		return RunIntentCancelled, true
	}
	return "", false
}

package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// AttemptID identifies one governed execution attempt of an Outcome's plan.
type AttemptID string

// IsZero reports whether the id is unset or blank.
func (id AttemptID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// String returns the raw identifier value.
func (id AttemptID) String() string {
	return string(id)
}

// AttemptStatus is the DURABLE status stored on an attempt row. Only the
// trigger-guarded legal transitions below may change it, and every mutation
// goes through the daemon's canonical writer.
//
// `unconfirmed` is deliberately ABSENT from this set: it is never stored. It
// is derived at read time from the bound session's heartbeat facts
// (activity_state, first_signal_at, is_terminated), because a missing
// heartbeat is unconfirmed — not dead — and suspicion must never be written
// as fact.
type AttemptStatus string

const (
	// AttemptQueued marks an admitted-but-not-yet-running attempt: its row and
	// fence exist, provider admission is in flight.
	AttemptQueued AttemptStatus = "queued"
	// AttemptRunning marks an attempt whose provider session was admitted and
	// bound. It stays running until a legal transition says otherwise; a
	// silent heartbeat downgrades only the DERIVED presentation.
	AttemptRunning AttemptStatus = "running"
	// AttemptPaused marks an owner-suspended attempt.
	AttemptPaused AttemptStatus = "paused"
	// AttemptSucceeded is reserved for #35's Verification binding. No #31 path
	// writes it: provider completion is never success.
	AttemptSucceeded AttemptStatus = "succeeded"
	// AttemptFailed marks an attempt that truthfully could not run (e.g. the
	// spawner crashed after the durable row existed).
	AttemptFailed AttemptStatus = "failed"
	// AttemptCancelled marks an owner-cancelled attempt.
	AttemptCancelled AttemptStatus = "cancelled"
	// AttemptLost marks an attempt whose custody reconcile could not prove
	// alive. Its effects stay suspect; replacement happens on a NEW row.
	AttemptLost AttemptStatus = "lost"
	// AttemptReconciled marks an ended attempt that reconcile has accounted
	// for WITHOUT classifying its result: the provider session exited and the
	// observation is recorded, but result classification awaits Verification
	// (#35). Provider completion ≠ done.
	AttemptReconciled AttemptStatus = "reconciled"
)

// Valid reports whether s is a supported attempt status.
func (s AttemptStatus) Valid() bool {
	switch s {
	case AttemptQueued, AttemptRunning, AttemptPaused, AttemptSucceeded,
		AttemptFailed, AttemptCancelled, AttemptLost, AttemptReconciled:
		return true
	}
	return false
}

// Terminal reports whether s closes the attempt's lifecycle.
func (s AttemptStatus) Terminal() bool {
	switch s {
	case AttemptSucceeded, AttemptFailed, AttemptCancelled, AttemptLost, AttemptReconciled:
		return true
	}
	return false
}

// LegalAttemptTransitions enumerates every stored-status transition the
// database triggers accept. Anything outside this map aborts the write at the
// SQL layer, so no service bug can invent a lifecycle.
var LegalAttemptTransitions = map[AttemptStatus][]AttemptStatus{
	// Queued attempts may be declared lost when reconcile cannot account for
	// an admission whose outcome is unknown (vanished start): custody must
	// resolve without dressing the ambiguity up as a clean failure.
	AttemptQueued: {AttemptRunning, AttemptFailed, AttemptCancelled, AttemptLost},
	AttemptPaused: {AttemptRunning, AttemptCancelled, AttemptLost},
	// Running attempts end through reconcile (lost/reconciled), owner action
	// (paused/cancelled), or truthful spawn/runner failure. There is no
	// running -> succeeded transition anywhere in #31: success arrives only
	// with #35's Verification binding.
	AttemptRunning: {AttemptPaused, AttemptFailed, AttemptCancelled, AttemptLost, AttemptReconciled},
}

// AttemptTransitionLegal reports whether from -> to is a legal stored
// transition.
func AttemptTransitionLegal(from, to AttemptStatus) bool {
	if from == to {
		return true // no-op writes are permitted and emit nothing
	}
	for _, next := range LegalAttemptTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// Attempt is one governed execution of an Outcome's approved plan. Attempts
// are append-only lineage: a replacement is always a NEW attempt row, never an
// update of the old one, so restart replays the exact custody history.
type Attempt struct {
	ID             AttemptID
	OutcomeID      OutcomeID
	PlanRevisionID PlanRevisionID
	WorkUnitID     WorkUnitID
	// Number is the attempt's 1-based position in the Outcome's lineage.
	Number int64
	Status AttemptStatus
	// ContractRevisionNumber snapshots which contract revision the executing
	// plan bound at admission; immutable for the attempt's life.
	ContractRevisionNumber int64
	RequestKey             string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// Validate checks intrinsic attempt invariants. Number uniqueness per Outcome,
// status transitions, and immutability are enforced by storage.
func (a Attempt) Validate() error {
	if a.ID.IsZero() {
		return fmt.Errorf("attempt id is required")
	}
	if a.OutcomeID.IsZero() {
		return fmt.Errorf("attempt outcome id is required")
	}
	if a.PlanRevisionID.IsZero() {
		return fmt.Errorf("attempt plan revision id is required")
	}
	if a.WorkUnitID.IsZero() {
		return fmt.Errorf("attempt work unit id is required")
	}
	if a.Number < 1 {
		return fmt.Errorf("attempt number must be at least 1")
	}
	if a.ContractRevisionNumber < 1 {
		return fmt.Errorf("attempt contract revision number must be at least 1")
	}
	if !a.Status.Valid() {
		return fmt.Errorf("unsupported attempt status %q", a.Status)
	}
	return nil
}

// AttemptSessionRef binds one attempt to one provider session identity.
//
// Per locked ruling D6, SessionID is plain TEXT with NO foreign key into
// sessions(id): a spawn rollback deletes seed session rows, but the ref
// history must outlive session-row GC so lineage survives. Harness, mode,
// digests, and the admission snapshot live on the ref.
type AttemptSessionRef struct {
	ID                     AttemptSessionRefID
	AttemptID              AttemptID
	Seq                    int64
	SessionID              string
	Harness                AgentHarness
	Mode                   SessionMode
	RunBriefCoreDigest     string
	RunBriefCompiledDigest string
	AdmissionSnapshot      string
	BoundAt                time.Time
}

// AttemptSessionRefID identifies one session binding inside an attempt.
type AttemptSessionRefID string

// IsZero reports whether the id is unset or blank.
func (id AttemptSessionRefID) IsZero() bool {
	return strings.TrimSpace(string(id)) == ""
}

// String returns the raw identifier value.
func (id AttemptSessionRefID) String() string {
	return string(id)
}

// Validate checks intrinsic session-ref invariants.
func (r AttemptSessionRef) Validate() error {
	if r.ID.IsZero() {
		return fmt.Errorf("attempt session ref id is required")
	}
	if r.AttemptID.IsZero() {
		return fmt.Errorf("attempt session ref attempt id is required")
	}
	if r.Seq < 1 {
		return fmt.Errorf("attempt session ref seq must be at least 1")
	}
	if strings.TrimSpace(r.SessionID) == "" {
		return fmt.Errorf("attempt session ref session id is required")
	}
	if len(r.RunBriefCoreDigest) != 64 {
		return fmt.Errorf("attempt session ref requires the run brief core digest")
	}
	return nil
}

// AdmissionSnapshotVersion pins the shape of the recorded admission snapshot
// so later slices can evolve the payload without reinterpreting old rows.
const AdmissionSnapshotVersion = 1

// AttemptObservation is one append-only, ordered fact observed about an
// attempt. Observations are insertable ALWAYS — including for stale attempts
// — so inspection stays possible after replacement, but no observation ever
// mutates current truth by itself.
type AttemptObservation struct {
	ID        string
	AttemptID AttemptID
	Seq       int64
	Kind      string
	Payload   string
	CreatedAt time.Time
}

// Validate checks intrinsic observation invariants.
func (o AttemptObservation) Validate() error {
	if strings.TrimSpace(o.ID) == "" {
		return fmt.Errorf("attempt observation id is required")
	}
	if o.AttemptID.IsZero() {
		return fmt.Errorf("attempt observation attempt id is required")
	}
	if o.Seq < 1 {
		return fmt.Errorf("attempt observation seq must be at least 1")
	}
	if strings.TrimSpace(o.Kind) == "" {
		return fmt.Errorf("attempt observation kind is required")
	}
	if o.Payload != "" && !json.Valid([]byte(o.Payload)) {
		return fmt.Errorf("attempt observation payload must be valid JSON")
	}
	return nil
}

// Observation kinds recorded by the v0 Act & Observe paths. Free-form kinds
// remain insertable; these are the canonical ones the service emits.
const (
	ObservationAttemptContained = "contained"
	ObservationAttemptResumed   = "resumed"
	ObservationProviderExit     = "provider_exit"
	ObservationAdmissionFailed  = "admission_failed"
	// ObservationAdmissionAmbiguous marks a start whose outcome is UNKNOWN:
	// the request may or may not have reached the provider. The attempt stays
	// queued and derives as unconfirmed until reconcile decides.
	ObservationAdmissionAmbiguous = "admission_ambiguous"
	ObservationOwnerCancel        = "owner_cancelled"
	ObservationOwnerPause         = "owner_paused"
	ObservationOwnerResume        = "owner_resumed"
	ObservationRecoveryAttention  = "needs_attention"
)

// AttemptFence is the custody lock over one worktree subject. At most ONE
// open fence per subject may exist (partial unique index on released_at IS
// NULL); replacement inherits custody only through reconcile releasing the
// old fence before the new attempt issues its own.
type AttemptFence struct {
	ID            string
	Subject       string
	AttemptID     AttemptID
	IssuedAt      time.Time
	ReleasedAt    time.Time
	ReleaseReason string
}

// Open reports whether the fence currently holds custody.
func (f AttemptFence) Open() bool {
	return f.ReleasedAt.IsZero()
}

// Validate checks intrinsic fence invariants.
func (f AttemptFence) Validate() error {
	if strings.TrimSpace(f.ID) == "" {
		return fmt.Errorf("attempt fence id is required")
	}
	if strings.TrimSpace(f.Subject) == "" {
		return fmt.Errorf("attempt fence subject is required")
	}
	if f.AttemptID.IsZero() {
		return fmt.Errorf("attempt fence attempt id is required")
	}
	if !f.Open() && f.ReleaseReason == "" {
		return fmt.Errorf("a released attempt fence must record why")
	}
	return nil
}

// FenceSubjectForProject names the v0 worktree fence subject: one governed
// isolated worktree per project, so two Outcomes can never hold simultaneous
// writers against the same tree.
func FenceSubjectForProject(projectID ProjectID) string {
	return "project:" + string(projectID)
}

// RecoveryResolution is the verdict a recovery receipt records.
type RecoveryResolution string

const (
	// RecoveryResumed proves the SAME attempt stayed (or returned) alive; no
	// replacement is warranted.
	RecoveryResumed RecoveryResolution = "resumed"
	// RecoveryReplacement declares the attempt unrecoverable and hands custody
	// to a future replacement attempt row.
	RecoveryReplacement RecoveryResolution = "replacement_attempt"
	// RecoveryNeedsAttention escalates to the owner without deciding custody.
	RecoveryNeedsAttention RecoveryResolution = "needs_attention"
)

// Valid reports whether r is a supported recovery resolution.
func (r RecoveryResolution) Valid() bool {
	switch r {
	case RecoveryResumed, RecoveryReplacement, RecoveryNeedsAttention:
		return true
	}
	return false
}

// AttemptRecoveryReceipt is the immutable record of one containment/reconcile
// decision. Receipts are append-only evidence; they never rewrite the
// attempt's own history.
type AttemptRecoveryReceipt struct {
	ID                   string
	AttemptID            AttemptID
	Resolution           RecoveryResolution
	ReplacementAttemptID AttemptID
	Detail               string
	CreatedAt            time.Time
}

// Validate checks intrinsic receipt invariants. ReplacementAttemptID is
// intentionally NOT foreign-keyed: it may name an attempt that does not exist
// yet when the receipt is written before the replacement starts.
func (r AttemptRecoveryReceipt) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("recovery receipt id is required")
	}
	if r.AttemptID.IsZero() {
		return fmt.Errorf("recovery receipt attempt id is required")
	}
	if !r.Resolution.Valid() {
		return fmt.Errorf("unsupported recovery resolution %q", r.Resolution)
	}
	return nil
}

// SessionHeartbeatFacts are the read-time heartbeat facts presentation is
// derived from. They mirror the durable session columns and nothing else —
// no transcript parsing, no derived-status reads.
type SessionHeartbeatFacts struct {
	// Present reports whether a session row exists for the bound ref. A
	// missing row (GC'd or rolled back) is unconfirmed, not dead.
	Present bool
	// ActivityState is the session's durable activity state, empty when absent.
	ActivityState ActivityState
	// FirstSignalAt is when the first agent hook signal arrived; zero means
	// the session has never signalled.
	FirstSignalAt time.Time
	// IsTerminated mirrors sessions.is_terminated.
	IsTerminated bool
}

// Derived attempt phases. These are presentation vocabulary computed at read
// time by DeriveAttemptPresentation — never stored.
const (
	AttemptPhaseAwaitingStart     = "awaiting_start"
	AttemptPhaseExecuting         = "executing"
	AttemptPhaseSuspended         = "suspended"
	AttemptPhaseUnconfirmed       = "unconfirmed"
	AttemptPhaseEndedUnclassified = "ended_unclassified"
	AttemptPhaseHaltedFailed      = "halted_failed"
	AttemptPhaseHaltedCancelled   = "halted_cancelled"
	AttemptPhaseSuspectLost       = "suspect_lost"
	AttemptPhaseSucceeded         = "succeeded"
)

// AttemptPresentation is the derived read-time truth about an attempt: what
// phase it is really in, whether its liveness is unproven, whether it ended
// without a result classification, and the safe next action for the owner.
type AttemptPresentation struct {
	Phase             string `json:"phase"`
	Unconfirmed       bool   `json:"unconfirmed"`
	EndedUnclassified bool   `json:"endedUnclassified"`
	NextAction        string `json:"nextAction"`
}

// DeriveAttemptPresentation computes the read model from the STORED status
// plus the bound session's heartbeat facts. Rules pinned here:
//
//   - missing heartbeat facts are UNCONFIRMED, never dead;
//   - provider/session termination is an END, never a success — ended
//     attempts present "ended, result unclassified";
//   - an admission whose outcome is UNKNOWN (unresolvedAdmission: the start
//     request may or may not have reached the provider) presents as
//     unconfirmed too — ambiguous startup is never shown as running, and
//     never dressed up as a clean failure;
//   - only stored statuses speak for themselves; nothing here mutates.
func DeriveAttemptPresentation(status AttemptStatus, facts SessionHeartbeatFacts, unresolvedAdmission bool) AttemptPresentation {
	switch status {
	case AttemptQueued:
		if unresolvedAdmission {
			return AttemptPresentation{
				Phase:       AttemptPhaseUnconfirmed,
				Unconfirmed: true,
				NextAction:  "Start outcome is unknown — reconcile before restarting so duplicate writers stay impossible.",
			}
		}
		return AttemptPresentation{
			Phase:      AttemptPhaseAwaitingStart,
			NextAction: "Admitting the authorized plan onto a provider session.",
		}
	case AttemptPaused:
		return AttemptPresentation{
			Phase:      AttemptPhaseSuspended,
			NextAction: "Resume the attempt or cancel it.",
		}
	case AttemptRunning:
		switch {
		case !facts.Present || facts.FirstSignalAt.IsZero():
			// Missing heartbeat = unconfirmed, not dead. Contain before any
			// replacement so duplicate effects stay impossible.
			return AttemptPresentation{
				Phase:       AttemptPhaseUnconfirmed,
				Unconfirmed: true,
				NextAction:  "Liveness is unproven — contain and reconcile before replacing.",
			}
		case facts.IsTerminated:
			return AttemptPresentation{
				Phase:             AttemptPhaseEndedUnclassified,
				EndedUnclassified: true,
				NextAction:        "The provider session ended. Completion is not done — classify through Verification.",
			}
		default:
			return AttemptPresentation{
				Phase:      AttemptPhaseExecuting,
				NextAction: "Waiting — observe.",
			}
		}
	case AttemptFailed:
		return AttemptPresentation{
			Phase:      AttemptPhaseHaltedFailed,
			NextAction: "Reconcile custody, then start a replacement attempt if needed.",
		}
	case AttemptCancelled:
		return AttemptPresentation{
			Phase:      AttemptPhaseHaltedCancelled,
			NextAction: "Start a replacement attempt if the Outcome stays active.",
		}
	case AttemptLost:
		return AttemptPresentation{
			Phase:      AttemptPhaseSuspectLost,
			NextAction: "Custody is unresolved — replace the attempt or mark it needing attention.",
		}
	case AttemptReconciled:
		return AttemptPresentation{
			Phase:             AttemptPhaseEndedUnclassified,
			EndedUnclassified: true,
			NextAction:        "Ended and accounted for. Result classification awaits Verification.",
		}
	case AttemptSucceeded:
		return AttemptPresentation{
			Phase:      AttemptPhaseSucceeded,
			NextAction: "Proceed to Prove & Close.",
		}
	default:
		return AttemptPresentation{
			Phase:       AttemptPhaseUnconfirmed,
			Unconfirmed: true,
			NextAction:  "Unknown stored state — treat as unconfirmed and reconcile.",
		}
	}
}

package domain

import (
	"fmt"
	"strings"
	"time"
)

// IntakeAnalysisRequestID identifies one durable ask for an agent-authored
// Contract proposal.
type IntakeAnalysisRequestID string

// IsZero reports whether the id is unset or blank.
func (id IntakeAnalysisRequestID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// String returns the raw identifier.
func (id IntakeAnalysisRequestID) String() string { return string(id) }

// IntakeAnalysisRequestStatus is the lifecycle of one ask.
type IntakeAnalysisRequestStatus string

const (
	// IntakeAnalysisRequested marks a spawned, unanswered ask.
	IntakeAnalysisRequested IntakeAnalysisRequestStatus = "requested"
	// IntakeAnalysisFulfilled marks an ask whose proposal passed validation
	// and became the intake's current editable proposal.
	IntakeAnalysisFulfilled IntakeAnalysisRequestStatus = "fulfilled"
	// IntakeAnalysisRejected marks an ask the agent answered with something
	// the daemon refused. The draft is retained so the owner can see it.
	IntakeAnalysisRejected IntakeAnalysisRequestStatus = "rejected"
	// IntakeAnalysisExpired marks an ask the agent never answered in time.
	IntakeAnalysisExpired IntakeAnalysisRequestStatus = "expired"
	// IntakeAnalysisRequestCancelled marks an owner-cancelled ask — the
	// escape hatch behind "use the offline proposal instead".
	IntakeAnalysisRequestCancelled IntakeAnalysisRequestStatus = "cancelled"
)

// Valid reports whether s is a supported request status.
func (s IntakeAnalysisRequestStatus) Valid() bool {
	switch s {
	case IntakeAnalysisRequested, IntakeAnalysisFulfilled, IntakeAnalysisRejected,
		IntakeAnalysisExpired, IntakeAnalysisRequestCancelled:
		return true
	}
	return false
}

// Open reports whether the request may still be answered.
func (s IntakeAnalysisRequestStatus) Open() bool { return s == IntakeAnalysisRequested }

// DefaultIntakeAnalysisRequestTTL bounds how long an unanswered ask stays open.
//
// It is longer than the decomposition equivalent on purpose: a decomposing
// agent is handed the contract it must split, while an analyzing agent is
// handed a sentence and has to read a repository it has never seen before it
// can propose criteria grounded in that repository. Reading is the entire
// point of asking an agent rather than the offline baseline, so the bound has
// to leave room for it.
//
// The bound is durable rather than an in-memory timer so a daemon restart does
// not leave a request open forever.
const DefaultIntakeAnalysisRequestTTL = 15 * time.Minute

// IntakeAnalysisRequest is one durable ask for an agent-authored Contract
// proposal.
//
// It exists because the daemon has no synchronous model call: the proposal
// arrives later, over the API, from a session the daemon spawned. The request
// is what makes that callback addressable, bounded, and single-use.
//
// It deliberately mirrors DecompositionRequest rather than inventing a second
// vocabulary for the same mechanism.
type IntakeAnalysisRequest struct {
	ID       IntakeAnalysisRequestID
	IntakeID IntakeSessionID
	// ExpectedProposalRevision freezes what the agent was asked about. It is
	// the intake counterpart of the decomposition request's frozen contract
	// revision: an answer that arrives after the owner has already revised the
	// proposal is refused rather than silently rebound to the newer one.
	ExpectedProposalRevision int64
	Status                   IntakeAnalysisRequestStatus
	// CallbackTokenDigest is the SHA-256 of the token handed to the spawned
	// session. The token itself is never stored.
	//
	// The token is SCOPING, NOT AUTHENTICATION. The loopback listener is
	// unauthenticated by deliberate decision, so any local process can already
	// reach this endpoint; the digest cannot change that. What it prevents is
	// the failure this mechanism introduces — a confused, retrying, or
	// misrouted agent answering for the wrong intake, against a superseded
	// revision, or twice.
	CallbackTokenDigest string
	// SessionID is the bounded session spawned to answer, when one started.
	SessionID string
	// Harness names which agent was asked, so the waiting state can say who is
	// working rather than showing an anonymous spinner.
	Harness   AgentHarness
	ExpiresAt time.Time
	// RawProposal is the agent's draft, retained even when refused so the
	// owner can see what was proposed and why it was not accepted.
	RawProposal string
	// RefusalReason is the daemon's own words when it refused the draft.
	RefusalReason string
	CreatedAt     time.Time
	AnsweredAt    *time.Time
}

// Validate checks intrinsic request invariants.
func (r IntakeAnalysisRequest) Validate() error {
	switch {
	case r.ID.IsZero():
		return fmt.Errorf("intake analysis request id is required")
	case r.IntakeID.IsZero():
		return fmt.Errorf("intake analysis request intake id is required")
	case r.ExpectedProposalRevision < 0:
		return fmt.Errorf("intake analysis request must freeze a proposal revision")
	case !r.Status.Valid():
		return fmt.Errorf("unsupported intake analysis request status %q", r.Status)
	case !isSHA256Hex(r.CallbackTokenDigest):
		return fmt.Errorf("intake analysis request callback token digest must be 64 hexadecimal characters")
	case r.ExpiresAt.IsZero():
		return fmt.Errorf("intake analysis request must carry a durable expiry")
	}
	return nil
}

// Expired reports whether an open request has passed its durable expiry.
// Expiry is derived from the clock at read time rather than stored as a state
// nothing advances, so a daemon that was not running still sees the truth.
func (r IntakeAnalysisRequest) Expired(now time.Time) bool {
	return r.Status.Open() && !now.Before(r.ExpiresAt)
}

// CallbackTokenMatches reports whether a presented token addresses this
// request. It compares digests, so a mismatch never reveals the real token.
func (r IntakeAnalysisRequest) CallbackTokenMatches(token string) bool {
	presented := strings.TrimSpace(token)
	if presented == "" || r.CallbackTokenDigest == "" {
		return false
	}
	return HashCallbackToken(presented) == r.CallbackTokenDigest
}

// AdmitProposalAnswer reports whether this request may accept an answer right
// now, and why not when it may not.
//
// Every refusal here happens before the proposal is even parsed: an answer to
// a closed, expired, or wrongly-addressed request is not a validation problem,
// it is a routing problem, and treating it as one keeps the two apart.
func (r IntakeAnalysisRequest) AdmitProposalAnswer(token string, now time.Time) error {
	if !r.Status.Open() {
		return fmt.Errorf("this intake analysis request is already %s", r.Status)
	}
	if r.Expired(now) {
		return fmt.Errorf("this intake analysis request expired")
	}
	if !r.CallbackTokenMatches(token) {
		return fmt.Errorf("callback token does not address this intake analysis request")
	}
	return nil
}

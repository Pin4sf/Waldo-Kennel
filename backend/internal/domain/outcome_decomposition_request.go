package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// DecompositionRequestID identifies one durable ask for an agent-authored
// decomposition.
type DecompositionRequestID string

// IsZero reports whether the id is unset or blank.
func (id DecompositionRequestID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// String returns the raw identifier.
func (id DecompositionRequestID) String() string { return string(id) }

// DecompositionRequestStatus is the lifecycle of one ask.
type DecompositionRequestStatus string

const (
	// DecompositionRequested marks a spawned, unanswered ask.
	DecompositionRequested DecompositionRequestStatus = "requested"
	// DecompositionFulfilled marks an ask whose proposal passed validation and
	// became a proposed DecompositionRevision.
	DecompositionFulfilled DecompositionRequestStatus = "fulfilled"
	// DecompositionRejected marks an ask the agent answered with something the
	// daemon refused. The draft is retained so the owner can correct it.
	DecompositionRejected DecompositionRequestStatus = "rejected"
	// DecompositionExpired marks an ask the agent never answered in time.
	DecompositionExpired DecompositionRequestStatus = "expired"
	// DecompositionRequestCancelled marks an owner-cancelled ask.
	DecompositionRequestCancelled DecompositionRequestStatus = "cancelled"
)

// Valid reports whether s is a supported request status.
func (s DecompositionRequestStatus) Valid() bool {
	switch s {
	case DecompositionRequested, DecompositionFulfilled, DecompositionRejected,
		DecompositionExpired, DecompositionRequestCancelled:
		return true
	}
	return false
}

// Open reports whether the request may still be answered.
func (s DecompositionRequestStatus) Open() bool { return s == DecompositionRequested }

// DefaultDecompositionRequestTTL bounds how long an unanswered ask stays open.
// The bound is durable rather than an in-memory timer so a daemon restart does
// not leave a request open forever.
const DefaultDecompositionRequestTTL = 10 * time.Minute

// MaxProposedContributions caps how many contributing Outcomes one proposal
// may contain. It is a sanity bound on a model's output, not a product limit:
// a proposal of thirty contributors is far likelier to be a runaway generation
// than a decomposition anyone wants to review.
const MaxProposedContributions = 12

// DecompositionRequest is one durable ask for an agent-authored decomposition.
//
// It exists because the daemon has no synchronous model call: the proposal
// arrives later, over the API, from a session the daemon spawned. The request
// is what makes that callback addressable, bounded, and single-use.
type DecompositionRequest struct {
	ID        DecompositionRequestID
	OutcomeID OutcomeID
	// ContractRevisionID freezes which contract the agent was asked about. A
	// proposal answering a superseded contract is refused rather than rebound.
	ContractRevisionID ContractRevisionID
	Status             DecompositionRequestStatus
	// CallbackTokenDigest is the SHA-256 of the token handed to the spawned
	// session. The token itself is never stored.
	//
	// The token is SCOPING, NOT AUTHENTICATION. The loopback listener is
	// unauthenticated by deliberate decision, so any local process can already
	// reach this endpoint; the digest cannot change that. What it prevents is
	// the failure this mechanism introduces — a confused, retrying, or
	// misrouted agent answering for the wrong Outcome, against a superseded
	// revision, or twice.
	CallbackTokenDigest string
	// SessionID is the bounded session spawned to answer, when one started.
	SessionID string
	ExpiresAt time.Time
	// RawProposal is the agent's draft, retained even when refused so the
	// owner can correct one field instead of regenerating. It is a draft, and
	// deliberately not a DecompositionRevision.
	RawProposal string
	// RefusalReason is the daemon's own words when it refused the draft.
	RefusalReason string
	// DecompositionID names the proposal this ask produced, once fulfilled.
	DecompositionID DecompositionRevisionID
	CreatedAt       time.Time
	AnsweredAt      *time.Time
}

// Validate checks intrinsic request invariants.
func (r DecompositionRequest) Validate() error {
	switch {
	case r.ID.IsZero():
		return fmt.Errorf("decomposition request id is required")
	case r.OutcomeID.IsZero():
		return fmt.Errorf("decomposition request outcome id is required")
	case r.ContractRevisionID.IsZero():
		return fmt.Errorf("decomposition request must freeze a contract revision")
	case !r.Status.Valid():
		return fmt.Errorf("unsupported decomposition request status %q", r.Status)
	case !isSHA256Hex(r.CallbackTokenDigest):
		return fmt.Errorf("decomposition request callback token digest must be 64 hexadecimal characters")
	case r.ExpiresAt.IsZero():
		return fmt.Errorf("decomposition request must carry a durable expiry")
	}
	return nil
}

// Expired reports whether an open request has passed its durable expiry.
// Expiry is derived from the clock at read time rather than stored as a state
// nothing advances, so a daemon that was not running still sees the truth.
func (r DecompositionRequest) Expired(now time.Time) bool {
	return r.Status.Open() && !now.Before(r.ExpiresAt)
}

// HashCallbackToken derives the stored digest from a callback token.
func HashCallbackToken(token string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(digest[:])
}

// CallbackTokenMatches reports whether a presented token addresses this
// request. It compares digests, so a mismatch never reveals the real token.
func (r DecompositionRequest) CallbackTokenMatches(token string) bool {
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
func (r DecompositionRequest) AdmitProposalAnswer(token string, now time.Time) error {
	if !r.Status.Open() {
		return fmt.Errorf("this decomposition request is already %s", r.Status)
	}
	if r.Expired(now) {
		return fmt.Errorf("this decomposition request expired")
	}
	if !r.CallbackTokenMatches(token) {
		return fmt.Errorf("callback token does not address this decomposition request")
	}
	return nil
}

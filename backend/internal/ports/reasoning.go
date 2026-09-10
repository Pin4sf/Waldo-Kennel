package ports

import (
	"context"
	"errors"
	"net"
	"os"
)

// ReasoningFailureKind classifies why one reasoning call did not yield a usable
// proposal.
//
// It exists because "the model call failed" is not an actionable fact. The
// owner's next move differs completely between "configure a credential",
// "wait and retry" and "this request cannot be satisfied" — and the daemon has
// to record which one happened, durably, so a restarted daemon and the Work UI
// agree on why a run is not running.
//
// The vocabulary is provider-neutral on purpose. Adapters map vendor errors
// onto it at the edge; no control-plane code branches on a vendor error type or
// an error string.
type ReasoningFailureKind string

const (
	// ReasoningNotConfigured means no reasoning provider/credential is
	// available to call at all. This is a setup state, not a failed call, and
	// no provider was contacted.
	ReasoningNotConfigured ReasoningFailureKind = "not_configured"
	// ReasoningUnauthorized means the provider rejected the credential.
	ReasoningUnauthorized ReasoningFailureKind = "unauthorized"
	// ReasoningRateLimited means the provider throttled the call.
	ReasoningRateLimited ReasoningFailureKind = "rate_limited"
	// ReasoningTimedOut means the call exceeded its own deadline. The owner did
	// not ask to stop; the call ran out of time.
	ReasoningTimedOut ReasoningFailureKind = "timed_out"
	// ReasoningCancelled means the caller abandoned the call. Kept distinct
	// from a timeout because only one of the two is the owner's own decision,
	// and only one of them is worth retrying automatically.
	ReasoningCancelled ReasoningFailureKind = "cancelled"
	// ReasoningDeclined means the model refused the request. A refusal is a
	// completed call with an answer Kennel cannot use, not a transport fault.
	ReasoningDeclined ReasoningFailureKind = "declined"
	// ReasoningIncomplete means the reply stopped before a whole payload,
	// typically against the token budget.
	ReasoningIncomplete ReasoningFailureKind = "incomplete"
	// ReasoningInvalidOutput means the reply arrived but was not usable
	// structured material.
	ReasoningInvalidOutput ReasoningFailureKind = "invalid_output"
	// ReasoningUnavailable means the provider could not be reached or returned
	// a server-side fault.
	ReasoningUnavailable ReasoningFailureKind = "unavailable"
)

// Retryable reports whether repeating the identical call could plausibly
// succeed without the owner changing anything.
//
// This governs whether a *deliberate* retry is offered. It never authorizes an
// automatic retry: a reasoning call may be billed, so repeating one after an
// ambiguous result is a decision the owner makes, not a loop the daemon runs.
func (k ReasoningFailureKind) Retryable() bool {
	switch k {
	case ReasoningRateLimited, ReasoningTimedOut, ReasoningUnavailable:
		return true
	default:
		return false
	}
}

// ReasoningFailure is a classified reasoning failure.
//
// Detail is human-facing and must stay free of secrets: vendor error payloads
// can echo prompt content and headers, so adapters pass a summary they have
// chosen, never the raw provider error text.
type ReasoningFailure struct {
	Kind   ReasoningFailureKind
	Detail string
	// Err is the underlying cause, kept for logs and errors.Is/As. It is never
	// persisted or returned over the API.
	Err error
}

func (f *ReasoningFailure) Error() string {
	if f == nil {
		return ""
	}
	if f.Detail != "" {
		return f.Detail
	}
	return string(f.Kind)
}

func (f *ReasoningFailure) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.Err
}

// NewReasoningFailure builds a classified failure.
func NewReasoningFailure(kind ReasoningFailureKind, detail string, err error) *ReasoningFailure {
	return &ReasoningFailure{Kind: kind, Detail: detail, Err: err}
}

// ClassifyReasoningTransport classifies a failed provider call from the two
// signals every adapter can supply: the caller's context and an HTTP status
// where the vendor SDK exposed one (zero when it did not).
//
// Adapters extract the status with their own SDK's error type and then share
// this rule, so "429 means rate limited" is stated once rather than per vendor.
//
// The context is checked first and deliberately: a client-side deadline and a
// caller cancellation surface through the same transport error, and Go's own
// wording for them varies by which timeout mechanism fires first — an
// http.Client.Timeout reports "Client.Timeout exceeded while awaiting headers"
// while a request context deadline reports "context deadline exceeded". Asking
// the context, rather than reading the message, is what makes the
// classification stable instead of a race.
func ClassifyReasoningTransport(ctx context.Context, status int, err error) *ReasoningFailure {
	// The caller's own context is the only thing that can tell a deadline
	// apart from a cancellation, because the transport error cannot.
	if ctx != nil {
		switch {
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return NewReasoningFailure(ReasoningTimedOut, "Reasoning call ran out of time", err)
		case errors.Is(ctx.Err(), context.Canceled):
			return NewReasoningFailure(ReasoningCancelled, "Reasoning call was cancelled", err)
		}
	}
	// No caller deadline fired, so any remaining timeout is the adapter's own
	// client timeout rather than the owner abandoning the request.
	if isTimeout(err) {
		return NewReasoningFailure(ReasoningTimedOut, "Reasoning call ran out of time", err)
	}
	if errors.Is(err, context.Canceled) {
		return NewReasoningFailure(ReasoningCancelled, "Reasoning call was cancelled", err)
	}

	switch {
	case status == 401 || status == 403:
		return NewReasoningFailure(ReasoningUnauthorized, "The reasoning provider rejected the configured credential", err)
	case status == 429:
		return NewReasoningFailure(ReasoningRateLimited, "The reasoning provider is rate limiting this credential", err)
	case status >= 500:
		return NewReasoningFailure(ReasoningUnavailable, "The reasoning provider reported a server-side failure", err)
	case status == 400 || status == 404 || status == 422:
		// A rejected request shape is not retryable and not a credential
		// problem; saying "unavailable" would invite a pointless retry.
		return NewReasoningFailure(ReasoningInvalidOutput, "The reasoning provider rejected the request", err)
	}
	return NewReasoningFailure(ReasoningUnavailable, "The reasoning provider could not be reached", err)
}

// isTimeout reports a transport-level deadline. net.Error covers dial/read
// deadlines and http.Client.Timeout; os.ErrDeadlineExceeded covers the
// deadline sentinel the net package returns directly.
func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

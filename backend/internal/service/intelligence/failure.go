package intelligence

import (
	"context"
	"errors"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// Stable machine codes for a reasoning failure. They are the contract the Work
// UI switches on to choose a next action — a missing credential sends the owner
// to settings, a throttle offers a deliberate retry — and the same codes are
// persisted as an IntelligenceRun's FailureCode so a restarted daemon still
// knows why a run stopped.
//
// They are provider-neutral: no code names a vendor, and none can be inferred
// from model text.
const (
	CodeReasoningNotConfigured = "REASONING_NOT_CONFIGURED"
	CodeReasoningUnauthorized  = "REASONING_CREDENTIAL_REJECTED"
	CodeReasoningRateLimited   = "REASONING_RATE_LIMITED"
	CodeReasoningTimedOut      = "REASONING_TIMED_OUT"
	CodeReasoningCancelled     = "REASONING_CANCELLED"
	CodeReasoningDeclined      = "REASONING_DECLINED"
	CodeReasoningIncomplete    = "REASONING_INCOMPLETE"
	CodeReasoningInvalidOutput = "REASONING_INVALID_OUTPUT"
	CodeReasoningUnavailable   = "REASONING_UNAVAILABLE"
)

// codeFor maps a classification to its stable code. Kept in one place so the
// API surface and the durable failure record can never disagree about what a
// given failure is called.
func codeFor(kind ports.ReasoningFailureKind) string {
	switch kind {
	case ports.ReasoningNotConfigured:
		return CodeReasoningNotConfigured
	case ports.ReasoningUnauthorized:
		return CodeReasoningUnauthorized
	case ports.ReasoningRateLimited:
		return CodeReasoningRateLimited
	case ports.ReasoningTimedOut:
		return CodeReasoningTimedOut
	case ports.ReasoningCancelled:
		return CodeReasoningCancelled
	case ports.ReasoningDeclined:
		return CodeReasoningDeclined
	case ports.ReasoningIncomplete:
		return CodeReasoningIncomplete
	case ports.ReasoningInvalidOutput:
		return CodeReasoningInvalidOutput
	default:
		return CodeReasoningUnavailable
	}
}

// APIError renders a reasoning failure as a typed API error.
//
// Before this existed, a propose call with no configured credential reached the
// owner as a bare 500 INTERNAL_ERROR with no way to act on it. Anything that is
// not a classified reasoning failure is returned untouched, so a genuine
// internal bug still surfaces as a 500 rather than being dressed up as an
// upstream problem.
func APIError(err error) error {
	if err == nil {
		return nil
	}
	var failure *ports.ReasoningFailure
	if !errors.As(err, &failure) {
		return err
	}
	code := codeFor(failure.Kind)
	message := failure.Detail
	if message == "" {
		message = "Waldo reasoning could not complete"
	}
	details := map[string]any{"retryable": failure.Kind.Retryable()}
	switch failure.Kind {
	case ports.ReasoningNotConfigured, ports.ReasoningUnauthorized:
		// Setup states: the owner has to change something before a retry can
		// possibly work, which is a state conflict rather than an outage.
		return apierr.Conflict(code, message, details)
	case ports.ReasoningRateLimited, ports.ReasoningTimedOut, ports.ReasoningUnavailable:
		// Kennel and the request are both fine; the upstream is not available
		// right now, and 503 is the status that says exactly that.
		return apierr.Unavailable(code, message, details)
	case ports.ReasoningCancelled:
		return apierr.Conflict(code, message, details)
	default:
		// Declined, incomplete and invalid output are completed calls whose
		// answer Kennel cannot use. They are not the caller's input error and
		// not an outage, so they stay a state conflict.
		return apierr.Conflict(code, message, details)
	}
}

// TerminalReason is the durable (code, detail) pair recorded on an
// IntelligenceRun when reasoning stops without a usable proposal.
//
// Detail comes from the adapter's own chosen summary, never from a raw provider
// error: vendor payloads can echo prompt content and request headers, and this
// value is persisted in canonical provenance.
func TerminalReason(err error) (string, string) {
	var failure *ports.ReasoningFailure
	if errors.As(err, &failure) {
		detail := failure.Detail
		if detail == "" {
			detail = "Waldo reasoning could not complete"
		}
		return codeFor(failure.Kind), detail
	}
	// An unclassified failure is recorded as exactly that. Guessing a specific
	// reason here would put a fabricated cause into durable provenance.
	return "INTELLIGENCE_PROVIDER_FAILED", "Reasoning provider failed without a classified reason"
}

// TerminalizationContext returns a context that can still persist a terminal
// reasoning state after the caller's own context is done.
//
// It deliberately does not inherit cancellation: the whole point is to survive
// it. It stays bounded so a wedged store cannot hold a request goroutine open
// forever, and it carries no request values because the only work it does is
// one small write against the local database.
func TerminalizationContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent != nil && parent.Err() == nil {
		// The caller is still alive, so its context is the honest one to use:
		// it keeps request tracing intact and still cancels if the caller goes
		// away mid-write.
		return context.WithTimeout(parent, terminalizationBudget)
	}
	return context.WithTimeout(context.WithoutCancel(orBackground(parent)), terminalizationBudget)
}

// terminalizationBudget bounds the post-cancellation write. It is a named
// operational policy, not a magic number: one local SQLite status update.
const terminalizationBudget = 5 * time.Second

func orBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

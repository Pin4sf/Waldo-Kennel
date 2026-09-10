package intelligence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// The recorded UX defect this closes: a propose call with no configured
// reasoning credential reached the owner as a bare 500 INTERNAL_ERROR, which
// gives the UI nothing to act on. Each classification must carry a stable code
// and a status that says what kind of problem it is.
func TestAPIErrorGivesEveryReasoningFailureAnActionableCode(t *testing.T) {
	for _, tc := range []struct {
		kind      ports.ReasoningFailureKind
		wantCode  string
		wantKind  apierr.Kind
		retryable bool
	}{
		{ports.ReasoningNotConfigured, CodeReasoningNotConfigured, apierr.KindConflict, false},
		{ports.ReasoningUnauthorized, CodeReasoningUnauthorized, apierr.KindConflict, false},
		{ports.ReasoningRateLimited, CodeReasoningRateLimited, apierr.KindUnavailable, true},
		{ports.ReasoningTimedOut, CodeReasoningTimedOut, apierr.KindUnavailable, true},
		{ports.ReasoningUnavailable, CodeReasoningUnavailable, apierr.KindUnavailable, true},
		{ports.ReasoningCancelled, CodeReasoningCancelled, apierr.KindConflict, false},
		{ports.ReasoningDeclined, CodeReasoningDeclined, apierr.KindConflict, false},
		{ports.ReasoningIncomplete, CodeReasoningIncomplete, apierr.KindConflict, false},
		{ports.ReasoningInvalidOutput, CodeReasoningInvalidOutput, apierr.KindConflict, false},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			err := APIError(ports.NewReasoningFailure(tc.kind, "detail", nil))
			var api *apierr.Error
			if !errors.As(err, &api) {
				t.Fatalf("error is not typed for the API: %v", err)
			}
			if api.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q", api.Code, tc.wantCode)
			}
			if api.Kind != tc.wantKind {
				t.Fatalf("kind = %v, want %v", api.Kind, tc.wantKind)
			}
			if api.Code == "INTERNAL_ERROR" {
				t.Fatal("a classified reasoning failure must never render as INTERNAL_ERROR")
			}
			if got := api.Details["retryable"]; got != tc.retryable {
				t.Fatalf("retryable = %v, want %v", got, tc.retryable)
			}
		})
	}
}

// A real bug must keep surfacing as a 500. Dressing every failure up as an
// upstream problem would hide Kennel's own defects.
func TestAPIErrorLeavesUnclassifiedErrorsAlone(t *testing.T) {
	sentinel := errors.New("nil map write")
	if got := APIError(sentinel); !errors.Is(got, sentinel) {
		t.Fatalf("err = %v, want the original error", got)
	}
	var api *apierr.Error
	if errors.As(APIError(sentinel), &api) {
		t.Fatal("an unclassified error must not be given an API code")
	}
	if APIError(nil) != nil {
		t.Fatal("nil must stay nil")
	}
}

// The durable record has to name the real reason. Recording every provider
// failure as one flat code is what previously made a missing credential
// indistinguishable from a refusal after a restart.
func TestTerminalReasonRecordsTheClassifiedReason(t *testing.T) {
	code, detail := TerminalReason(ports.NewReasoningFailure(
		ports.ReasoningRateLimited, "The reasoning provider is rate limiting this credential", nil))
	if code != CodeReasoningRateLimited {
		t.Fatalf("code = %q, want %q", code, CodeReasoningRateLimited)
	}
	if detail == "" {
		t.Fatal("detail must be recorded")
	}
}

// An unclassified failure is recorded as unclassified. Guessing a specific
// cause would write a fabricated reason into canonical provenance.
func TestTerminalReasonDoesNotInventAReason(t *testing.T) {
	code, detail := TerminalReason(errors.New("boom"))
	if code != "INTELLIGENCE_PROVIDER_FAILED" {
		t.Fatalf("code = %q, want the unclassified code", code)
	}
	if detail == "boom" {
		t.Fatal("raw provider text must not become the durable detail")
	}
}

// Terminalization must survive the caller giving up: that is the whole reason
// this context exists. A cancelled parent previously took the terminal write
// down with it and left the run displayed as still thinking.
func TestTerminalizationContextSurvivesACancelledCaller(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	ctx, done := TerminalizationContext(parent)
	defer done()
	if err := ctx.Err(); err != nil {
		t.Fatalf("cleanup context is already done: %v", err)
	}
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("cleanup context must stay bounded")
	}
}

// While the caller is alive its context is the honest one to use, so tracing
// survives and an abandoned request still stops the write.
func TestTerminalizationContextKeepsALiveCallerContext(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx, done := TerminalizationContext(parent)
	defer done()
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > terminalizationBudget+time.Second {
		t.Fatalf("deadline = %v, want within the terminalization budget", deadline)
	}
	cancel()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("a live caller's cancellation should still reach the cleanup context")
	}
}

package controllers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/service/intelligence"
	outcomevc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

// The recorded launch defect: proposing a Plan with no reasoning credential
// configured answered 500 INTERNAL_ERROR, which tells the owner nothing and
// gives the UI nothing to offer. Each classified reasoning failure must reach
// the wire as its own code and an honest status.
//
// This exercises the real router and the real envelope, so it proves the wire
// contract rather than the mapping function in isolation.
func TestProposePlanSurfacesReasoningFailuresAsTypedAPIErrors(t *testing.T) {
	for _, tc := range []struct {
		name       string
		kind       ports.ReasoningFailureKind
		wantStatus int
		wantCode   string
		wantRetry  bool
	}{
		{"missing setup", ports.ReasoningNotConfigured, http.StatusConflict, intelligence.CodeReasoningNotConfigured, false},
		{"rejected credential", ports.ReasoningUnauthorized, http.StatusConflict, intelligence.CodeReasoningUnauthorized, false},
		{"throttled", ports.ReasoningRateLimited, http.StatusServiceUnavailable, intelligence.CodeReasoningRateLimited, true},
		{"timed out", ports.ReasoningTimedOut, http.StatusServiceUnavailable, intelligence.CodeReasoningTimedOut, true},
		{"declined", ports.ReasoningDeclined, http.StatusConflict, intelligence.CodeReasoningDeclined, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeOutcomeService{
				proposePlan: func(context.Context, domain.OutcomeID, int64) (outcomevc.PlanView, error) {
					return outcomevc.PlanView{}, intelligence.APIError(
						ports.NewReasoningFailure(tc.kind, "reasoning could not complete", nil))
				},
			}
			srv := newOutcomesTestServer(t, svc)
			body, status, _ := doRequest(t, srv, http.MethodPost,
				"/api/v1/outcomes/out-1/plans", `{"expectedContractRevision":1}`)
			if status != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", status, tc.wantStatus, body)
			}
			var got struct {
				Code    string         `json:"code"`
				Message string         `json:"message"`
				Details map[string]any `json:"details"`
			}
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("decode envelope: %v (body %s)", err, body)
			}
			if got.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q", got.Code, tc.wantCode)
			}
			if got.Code == "INTERNAL_ERROR" {
				t.Fatal("a classified reasoning failure must not reach the owner as INTERNAL_ERROR")
			}
			if got.Message == "" {
				t.Fatal("the owner needs a message to act on")
			}
			if retry, _ := got.Details["retryable"].(bool); retry != tc.wantRetry {
				t.Fatalf("retryable = %v, want %v", retry, tc.wantRetry)
			}
		})
	}
}

// A genuine internal defect must still be a 500 with no invented upstream
// cause, so this typing cannot be used to hide Kennel's own bugs.
func TestProposePlanKeepsUnclassifiedFailuresInternal(t *testing.T) {
	svc := &fakeOutcomeService{
		proposePlan: func(context.Context, domain.OutcomeID, int64) (outcomevc.PlanView, error) {
			return outcomevc.PlanView{}, context.DeadlineExceeded
		},
	}
	srv := newOutcomesTestServer(t, svc)
	body, status, _ := doRequest(t, srv, http.MethodPost,
		"/api/v1/outcomes/out-1/plans", `{"expectedContractRevision":1}`)
	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body %s)", status, body)
	}
}

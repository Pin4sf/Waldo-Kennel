package controllers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/controllers"
	outcomevc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

var askedAt = time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)

func requestView(status domain.DecompositionRequestStatus) outcomevc.DecompositionRequestView {
	return outcomevc.DecompositionRequestView{
		Request: domain.DecompositionRequest{
			ID: "dreq-1", OutcomeID: "out-fix", ContractRevisionID: "cr-1",
			Status: status, ExpiresAt: askedAt.Add(10 * time.Minute), CreatedAt: askedAt,
		},
	}
}

func post(t *testing.T, url string, body any, header map[string]string) (*http.Response, map[string]any) {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range header {
		req.Header.Set(key, value)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	decoded := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&decoded)
	return res, decoded
}

// The ask returns 202: an agent has been started, and the proposal is not here
// yet. Answering 201 would claim a proposal exists.
func TestAskForDecompositionRouteAcceptsAndReturnsTheRequest(t *testing.T) {
	svc := &fakeOutcomeService{}
	svc.askForDecomposition = func(outcomeID domain.OutcomeID, expected int64) (outcomevc.DecompositionRequestView, error) {
		if outcomeID != "out-fix" || expected != 1 {
			t.Fatalf("ask received outcome=%q revision=%d", outcomeID, expected)
		}
		return requestView(domain.DecompositionRequested), nil
	}
	server := newOutcomesTestServer(t, svc)
	defer server.Close()

	res, body := post(t, server.URL+"/api/v1/outcomes/out-fix/decomposition-requests",
		map[string]any{"expectedContractRevision": 1}, nil)
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 Accepted", res.StatusCode)
	}
	request := body["request"].(map[string]any)
	if request["status"] != "requested" || request["id"] != "dreq-1" {
		t.Fatalf("request = %+v", request)
	}
	// The token is never echoed back to the caller that opened the ask.
	for key := range request {
		if strings.Contains(strings.ToLower(key), "token") {
			t.Fatalf("the callback token must not appear in the ask response, found %q", key)
		}
	}
}

// The callback carries its token in a header and is addressed by REQUEST, not
// by Outcome — the answering agent knows only the request it was handed.
func TestSubmitProposalRouteForwardsTokenAndRawBody(t *testing.T) {
	svc := &fakeOutcomeService{}
	svc.submitAgentProposal = func(id domain.DecompositionRequestID, token string, in outcomevc.ProposeDecompositionInput, raw string) (outcomevc.DecompositionRequestView, error) {
		if id != "dreq-1" || token != "tok-abc" {
			t.Fatalf("callback received id=%q token=%q", id, token)
		}
		if len(in.Contributors) != 1 || in.Contributors[0].Ref != "c1" {
			t.Fatalf("contributors = %+v", in.Contributors)
		}
		// The agent's own bytes are retained verbatim, not a re-render.
		if !strings.Contains(raw, "\"ref\":\"c1\"") {
			t.Fatalf("raw body was not preserved: %q", raw)
		}
		return requestView(domain.DecompositionFulfilled), nil
	}
	server := newOutcomesTestServer(t, svc)
	defer server.Close()

	res, body := post(t, server.URL+"/api/v1/decomposition-requests/dreq-1/proposal",
		map[string]any{
			"rationale": "Two slices.",
			"contributors": []map[string]any{{
				"ref": "c1", "title": "One", "goal": "g",
				"successCriteria": []string{"s"}, "review": "checks",
				"claimedCriteria": []string{"crit-1"},
			}},
		},
		map[string]string{controllers.DecompositionCallbackTokenHeader: "tok-abc"})

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if body["request"].(map[string]any)["status"] != "fulfilled" {
		t.Fatalf("request = %+v", body["request"])
	}
}

// A refused proposal is NOT a transport error: the agent did its job and the
// daemon disagreed. It answers 200 with a rejected request carrying the reason
// and the draft, because the owner is the one who acts on that.
func TestSubmitProposalRouteReportsARefusalAsA200(t *testing.T) {
	svc := &fakeOutcomeService{}
	svc.submitAgentProposal = func(domain.DecompositionRequestID, string, outcomevc.ProposeDecompositionInput, string) (outcomevc.DecompositionRequestView, error) {
		view := requestView(domain.DecompositionRejected)
		view.Request.RefusalReason = "Every criterion must be claimed or retained."
		view.Request.RawProposal = `{"contributors":[{"ref":"c1"}]}`
		return view, nil
	}
	server := newOutcomesTestServer(t, svc)
	defer server.Close()

	res, body := post(t, server.URL+"/api/v1/decomposition-requests/dreq-1/proposal",
		map[string]any{"rationale": "x"}, map[string]string{controllers.DecompositionCallbackTokenHeader: "tok-abc"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 — a refusal is a recorded verdict, not a transport failure", res.StatusCode)
	}
	request := body["request"].(map[string]any)
	if request["status"] != "rejected" {
		t.Fatalf("status = %v, want rejected", request["status"])
	}
	// Both the reason and the draft reach the owner.
	if request["refusalReason"] == nil || request["rawProposal"] == nil {
		t.Fatalf("a refusal must carry its reason and the draft: %+v", request)
	}
}

// A routing refusal IS an error, and reads identically whatever was wrong, so
// a caller probing tokens learns nothing from the difference.
func TestSubmitProposalRouteRefusesBadRouting(t *testing.T) {
	svc := &fakeOutcomeService{}
	svc.submitAgentProposal = func(domain.DecompositionRequestID, string, outcomevc.ProposeDecompositionInput, string) (outcomevc.DecompositionRequestView, error) {
		return outcomevc.DecompositionRequestView{}, apierr.Conflict("DECOMPOSITION_REQUEST_NOT_ADMITTED",
			"callback token does not address this decomposition request", nil)
	}
	server := newOutcomesTestServer(t, svc)
	defer server.Close()

	res, body := post(t, server.URL+"/api/v1/decomposition-requests/dreq-1/proposal",
		map[string]any{"rationale": "x"}, map[string]string{controllers.DecompositionCallbackTokenHeader: "wrong"})
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", res.StatusCode)
	}
	if body["code"] != "DECOMPOSITION_REQUEST_NOT_ADMITTED" {
		t.Fatalf("code = %v", body["code"])
	}
}

// A missing header must not read as an empty-but-valid token.
func TestSubmitProposalRouteForwardsAnAbsentTokenAsEmpty(t *testing.T) {
	svc := &fakeOutcomeService{}
	seen := "unset"
	svc.submitAgentProposal = func(_ domain.DecompositionRequestID, token string, _ outcomevc.ProposeDecompositionInput, _ string) (outcomevc.DecompositionRequestView, error) {
		seen = token
		return requestView(domain.DecompositionRejected), nil
	}
	server := newOutcomesTestServer(t, svc)
	defer server.Close()

	post(t, server.URL+"/api/v1/decomposition-requests/dreq-1/proposal", map[string]any{"rationale": "x"}, nil)
	if seen != "" {
		t.Fatalf("absent header forwarded as %q, want the empty string the domain refuses", seen)
	}
}

func TestLatestDecompositionRequestRoute(t *testing.T) {
	svc := &fakeOutcomeService{}
	svc.latestDecompositionRequest = func(domain.OutcomeID) (outcomevc.DecompositionRequestView, error) {
		view := requestView(domain.DecompositionRequested)
		view.Expired = true
		return view, nil
	}
	server := newOutcomesTestServer(t, svc)
	defer server.Close()

	res, err := http.Get(server.URL + "/api/v1/outcomes/out-fix/decomposition-request")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	body := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	// Expiry is derived and surfaced, so a stale ask does not read as pending.
	if body["request"].(map[string]any)["expired"] != true {
		t.Fatalf("expired must reach the caller: %+v", body["request"])
	}
}

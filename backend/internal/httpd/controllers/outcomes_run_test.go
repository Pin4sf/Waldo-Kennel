package controllers_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/config"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/controllers"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	outcomevc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

// runStateFake adds the Mission supervision reads to the ordinary Outcome
// service fake. It is a separate type so the plain fake keeps proving that a
// daemon without this capability answers 501 rather than a fabricated state.
type runStateFake struct {
	*fakeOutcomeService
	runState   func(context.Context, domain.OutcomeID) (outcomevc.RunStateView, error)
	listStates func(context.Context, domain.ProjectID, bool) ([]outcomevc.RunStateView, error)

	lastTopLevelOnly bool
}

func (f *runStateFake) GetRunState(ctx context.Context, id domain.OutcomeID) (outcomevc.RunStateView, error) {
	return f.runState(ctx, id)
}

func (f *runStateFake) ListProjectRunStates(ctx context.Context, id domain.ProjectID, topLevelOnly bool) ([]outcomevc.RunStateView, error) {
	f.lastTopLevelOnly = topLevelOnly
	return f.listStates(ctx, id, topLevelOnly)
}

// newRunStateTestServer mounts the Outcome routes against a service that also
// implements the Mission supervision reads.
func newRunStateTestServer(t *testing.T, svc controllers.OutcomeService) *httptest.Server {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httptest.NewServer(httpd.NewRouterWithControl(config.Config{}, log, nil, httpd.APIDeps{
		Outcomes: svc,
	}, httpd.ControlDeps{}))
}

func sampleRunState() outcomevc.RunStateView {
	return outcomevc.RunStateView{
		OutcomeID: "out-1", ProjectID: "mer", Title: "Ledger",
		State: outcomevc.MissionNeedsYou, AttentionReason: outcomevc.ReasonStartRequired,
		EligibleActions: []outcomevc.RunActionEligibility{
			{Action: outcomevc.RunActionStart, Available: true},
			{Action: outcomevc.RunActionExport, Reason: outcomevc.ReasonDeliveryUnavailable},
		},
		Blocker:    &outcomevc.RunBlocker{Code: outcomevc.ReasonStartRequired, Message: "The next WorkUnit is ready to start"},
		Freshness:  outcomevc.RunFreshness{ObservedAt: time.Now().UTC(), ContractRevisionNumber: 1, PlanRevisionID: "plan-1", ProofGeneration: 7},
		PlanStatus: domain.PlanStatusApproved, PlanBindsCurrentContract: true,
		ProvenCriteria: 1, RequiredCriteria: 2,
	}
}

func TestGetOutcomeRunStateRoute_ServesTheDerivedMissionProjection(t *testing.T) {
	fake := &runStateFake{fakeOutcomeService: &fakeOutcomeService{}}
	fake.runState = func(_ context.Context, id domain.OutcomeID) (outcomevc.RunStateView, error) {
		if id != "out-1" {
			t.Fatalf("run state requested for %q", id)
		}
		return sampleRunState(), nil
	}
	srv := newRunStateTestServer(t, fake)

	body, status, _ := doRequest(t, srv, http.MethodGet, "/api/v1/outcomes/out-1/run", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	var decoded struct {
		RunState struct {
			State           string `json:"state"`
			AttentionReason string `json:"attentionReason"`
			EligibleActions []struct {
				Action    string `json:"action"`
				Available bool   `json:"available"`
				Reason    string `json:"reason"`
			} `json:"eligibleActions"`
			Blocker *struct {
				Code string `json:"code"`
			} `json:"blocker"`
			Intent    *json.RawMessage `json:"intent"`
			Freshness struct {
				ProofGeneration int64 `json:"proofGeneration"`
			} `json:"freshness"`
		} `json:"runState"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	if decoded.RunState.State != "needs_you" || decoded.RunState.AttentionReason != outcomevc.ReasonStartRequired {
		t.Fatalf("state = %q/%q", decoded.RunState.State, decoded.RunState.AttentionReason)
	}
	if decoded.RunState.Blocker == nil || decoded.RunState.Blocker.Code != outcomevc.ReasonStartRequired {
		t.Fatalf("blocker = %+v", decoded.RunState.Blocker)
	}
	// A nil intent must serialize as absent, not as an idle run: the two mean
	// different things and the renderer branches on it.
	if decoded.RunState.Intent != nil {
		t.Fatalf("intent = %s, want absent while run intent is unbuilt", *decoded.RunState.Intent)
	}
	if decoded.RunState.Freshness.ProofGeneration != 7 {
		t.Fatalf("proof generation = %d, want 7", decoded.RunState.Freshness.ProofGeneration)
	}
	var exportSeen bool
	for _, action := range decoded.RunState.EligibleActions {
		if action.Action != "export" {
			continue
		}
		exportSeen = true
		if action.Available || action.Reason != outcomevc.ReasonDeliveryUnavailable {
			t.Fatalf("export = %+v, want unavailable with a reason", action)
		}
	}
	if !exportSeen {
		t.Fatal("export was omitted rather than reported unavailable")
	}
}

// TestGetOutcomeRunStateRoute_ServesTheAuthorizationTheDaemonWillActuponUs
// keeps the wire contract able to express an authorized run between WorkUnits:
// in_progress with Start refused, and an intent that says which Plan it binds.
func TestGetOutcomeRunStateRoute_ServesTheAuthorizationTheDaemonWillActUpon(t *testing.T) {
	fake := &runStateFake{fakeOutcomeService: &fakeOutcomeService{}}
	fake.runState = func(context.Context, domain.OutcomeID) (outcomevc.RunStateView, error) {
		view := sampleRunState()
		view.State, view.AttentionReason, view.Blocker = outcomevc.MissionInProgress, "", nil
		view.Intent = &outcomevc.RunIntentView{
			Generation: 2, Desired: string(domain.RunIntentRunning),
			PlanRevisionID: "plan-1", BindsCurrentPlan: true, RequestedAt: time.Now().UTC(),
		}
		view.EligibleActions = []outcomevc.RunActionEligibility{
			{Action: outcomevc.RunActionStart, Reason: outcomevc.ReasonRunAlreadyAuthorized},
			{Action: outcomevc.RunActionCancel, Available: true},
		}
		return view, nil
	}
	srv := newRunStateTestServer(t, fake)

	body, status, _ := doRequest(t, srv, http.MethodGet, "/api/v1/outcomes/out-1/run", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	var decoded struct {
		RunState struct {
			State  string `json:"state"`
			Intent *struct {
				Desired          string `json:"desired"`
				PlanRevisionID   string `json:"planRevisionId"`
				BindsCurrentPlan bool   `json:"bindsCurrentPlan"`
			} `json:"intent"`
			EligibleActions []struct {
				Action    string `json:"action"`
				Available bool   `json:"available"`
				Reason    string `json:"reason"`
			} `json:"eligibleActions"`
		} `json:"runState"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	if decoded.RunState.State != "in_progress" {
		t.Fatalf("state = %q, want in_progress", decoded.RunState.State)
	}
	intent := decoded.RunState.Intent
	if intent == nil || intent.Desired != "running" || intent.PlanRevisionID != "plan-1" || !intent.BindsCurrentPlan {
		t.Fatalf("intent = %+v, want a running authorization bound to plan-1", intent)
	}
	for _, action := range decoded.RunState.EligibleActions {
		if action.Action == "start" && (action.Available || action.Reason != outcomevc.ReasonRunAlreadyAuthorized) {
			t.Fatalf("start = %+v, want refused with %s", action, outcomevc.ReasonRunAlreadyAuthorized)
		}
		if action.Action == "cancel" && !action.Available {
			t.Fatalf("cancel = %+v, want an authorized run to be stoppable", action)
		}
	}
}

func TestListProjectOutcomeRunStatesRoute_DefaultsToTopLevelOutcomes(t *testing.T) {
	fake := &runStateFake{fakeOutcomeService: &fakeOutcomeService{}}
	fake.listStates = func(_ context.Context, id domain.ProjectID, _ bool) ([]outcomevc.RunStateView, error) {
		if id != "mer" {
			t.Fatalf("run states requested for %q", id)
		}
		return []outcomevc.RunStateView{sampleRunState()}, nil
	}
	srv := newRunStateTestServer(t, fake)

	if _, status, _ := doRequest(t, srv, http.MethodGet, "/api/v1/projects/mer/outcome-run-states", ""); status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if !fake.lastTopLevelOnly {
		t.Fatal("default listing included contributing Outcomes; they belong in their parent Mission")
	}
	if _, status, _ := doRequest(t, srv, http.MethodGet, "/api/v1/projects/mer/outcome-run-states?scope=all", ""); status != http.StatusOK {
		t.Fatalf("scoped status = %d", status)
	}
	if fake.lastTopLevelOnly {
		t.Fatal("scope=all did not widen the listing")
	}
}

func TestOutcomeRunStateRoute_IsNotImplementedWithoutTheCapability(t *testing.T) {
	// The plain Outcome fake cannot derive run state. The route must say so
	// with the spec-backed 501 rather than inventing an empty Mission.
	srv := newOutcomesTestServer(t, &fakeOutcomeService{})
	body, status, _ := doRequest(t, srv, http.MethodGet, "/api/v1/outcomes/out-1/run", "")
	if status != http.StatusNotImplemented {
		t.Fatalf("status = %d, body = %s", status, body)
	}
}

// TestUnbuiltRunOperationsRefuseWithTheirOwnCode pins the contract the
// renderer task builds against: each unbuilt operation answers 501 with a
// stable, distinguishable code, never a generic NOT_IMPLEMENTED and never a
// fabricated success.
func TestUnbuiltRunOperationsRefuseWithTheirOwnCode(t *testing.T) {
	srv := newOutcomesTestServer(t, &fakeOutcomeService{})

	cases := []struct {
		method, path, body, wantCode string
	}{
		{http.MethodPost, "/api/v1/outcomes/out-1/run", `{"action":"start","requestKey":"rk-1"}`, "RUN_INTENT_UNAVAILABLE"},
		{http.MethodGet, "/api/v1/outcomes/out-1/deliveries", "", "DELIVERY_UNAVAILABLE"},
		{http.MethodPost, "/api/v1/outcomes/out-1/deliveries", `{"attemptId":"att-1"}`, "DELIVERY_UNAVAILABLE"},
		{http.MethodGet, "/api/v1/outcomes/out-1/deliveries/dlv-1", "", "DELIVERY_UNAVAILABLE"},
		{http.MethodGet, "/api/v1/outcomes/out-1/usage", "", "USAGE_ATTRIBUTION_UNAVAILABLE"},
	}
	for _, tc := range cases {
		t.Run(tc.wantCode+" "+tc.path, func(t *testing.T) {
			body, status, _ := doRequest(t, srv, tc.method, tc.path, tc.body)
			if status != http.StatusNotImplemented {
				t.Fatalf("status = %d, body = %s", status, body)
			}
			var decoded struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("decode: %v (%s)", err, body)
			}
			if decoded.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q", decoded.Code, tc.wantCode)
			}
		})
	}
}

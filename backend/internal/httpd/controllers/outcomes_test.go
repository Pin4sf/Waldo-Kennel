package controllers_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/config"
	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/sqlitetest"
	outcomevc "github.com/aoagents/agent-orchestrator/backend/internal/service/outcome"
)

type fakeOutcomeService struct {
	create func(context.Context, outcomevc.CreateInput) (outcomevc.OutcomeView, error)
	revise func(context.Context, domain.OutcomeID, outcomevc.ReviseContractInput) (outcomevc.OutcomeView, error)
	get    func(context.Context, domain.OutcomeID) (outcomevc.OutcomeView, error)

	lastInput outcomevc.CreateInput
}

func (f *fakeOutcomeService) Create(ctx context.Context, in outcomevc.CreateInput) (outcomevc.OutcomeView, error) {
	f.lastInput = in
	return f.create(ctx, in)
}

func (f *fakeOutcomeService) ReviseContract(ctx context.Context, id domain.OutcomeID, in outcomevc.ReviseContractInput) (outcomevc.OutcomeView, error) {
	return f.revise(ctx, id, in)
}

func (f *fakeOutcomeService) Get(ctx context.Context, id domain.OutcomeID) (outcomevc.OutcomeView, error) {
	return f.get(ctx, id)
}

func sampleOutcomeView() outcomevc.OutcomeView {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	rev := domain.ContractRevision{
		ID: "cr-1", OutcomeID: "out-fix", Number: 1,
		Goal: "g", SuccessCriteria: []string{"c1"}, Review: "checks", CreatedAt: now,
	}
	return outcomevc.OutcomeView{
		Outcome: domain.Outcome{
			ID: "out-fix", SpaceID: "rsp-1", Title: "Ledger",
			CurrentRevisionNumber: 1, CreatedAt: now, UpdatedAt: now,
		},
		Current: rev,
		History: []domain.ContractRevision{rev},
	}
}

func newOutcomesTestServer(t *testing.T, svc *fakeOutcomeService) *httptest.Server {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httptest.NewServer(httpd.NewRouterWithControl(config.Config{}, log, nil, httpd.APIDeps{
		Outcomes: svc,
	}, httpd.ControlDeps{}))
}

func TestCreateOutcomeRoute(t *testing.T) {
	svc := &fakeOutcomeService{}
	svc.create = func(_ context.Context, in outcomevc.CreateInput) (outcomevc.OutcomeView, error) {
		if in.ProjectID != "mer" || in.RequestKey == "" || len(in.SuccessCriteria) == 0 {
			return outcomevc.OutcomeView{}, apierr.Invalid("OUTCOME_CRITERIA_REQUIRED", "Name at least one success criterion", nil)
		}
		return sampleOutcomeView(), nil
	}
	srv := newOutcomesTestServer(t, svc)

	respBytes, status, _ := doRequest(t, srv, http.MethodPost, "/api/v1/projects/mer/outcomes",
		`{"title":"Ledger","goal":"g","successCriteria":["c1"],"review":"r","requestKey":"rk-1"}`)
	if status != http.StatusCreated {
		t.Fatalf("create = %d, want 201: %s", status, respBytes)
	}
	for _, want := range []string{`"spaceId":"rsp-1"`, `"currentRevisionNumber":1`, `"currentRevision":`, `"history":[`, `"number":1`} {
		if !strings.Contains(string(respBytes), want) {
			t.Fatalf("create body missing %s: %s", want, respBytes)
		}
	}
	if svc.lastInput.ProjectID != "mer" {
		t.Fatalf("project id not propagated: %+v", svc.lastInput)
	}
}

func TestReviseOutcomeConflictIs409Envelope(t *testing.T) {
	svc := &fakeOutcomeService{}
	svc.revise = func(_ context.Context, _ domain.OutcomeID, _ outcomevc.ReviseContractInput) (outcomevc.OutcomeView, error) {
		return outcomevc.OutcomeView{}, apierr.New(apierr.KindConflict, "OUTCOME_CONTRACT_CONFLICT",
			"Contract moved to revision 2; reload and retry against it",
			map[string]any{"expectedRevision": 1, "currentRevision": 2})
	}
	srv := newOutcomesTestServer(t, svc)

	respBytes, status, _ := doRequest(t, srv, http.MethodPost, "/api/v1/outcomes/out-fix/revisions",
		`{"expectedRevision":1,"goal":"g","successCriteria":["c"],"review":"r"}`)
	if status != http.StatusConflict {
		t.Fatalf("conflict status = %d, want 409: %s", status, respBytes)
	}
	for _, want := range []string{`"code":"OUTCOME_CONTRACT_CONFLICT"`, `"currentRevision":2`} {
		if !strings.Contains(string(respBytes), want) {
			t.Fatalf("conflict envelope missing %s: %s", want, respBytes)
		}
	}
}

func TestGetUnknownOutcomeIs404Envelope(t *testing.T) {
	svc := &fakeOutcomeService{}
	svc.get = func(_ context.Context, _ domain.OutcomeID) (outcomevc.OutcomeView, error) {
		return outcomevc.OutcomeView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	srv := newOutcomesTestServer(t, svc)
	respBytes, status, _ := doRequest(t, srv, http.MethodGet, "/api/v1/outcomes/out-nope", "")
	if status != http.StatusNotFound || !strings.Contains(string(respBytes), `"code":"OUTCOME_NOT_FOUND"`) {
		t.Fatalf("get = %d %s, want 404 OUTCOME_NOT_FOUND", status, respBytes)
	}
}

// TestOutcomeRoutesFunctionalThroughRealStore is the end-to-end proof for #21:
// real SQLite (migrations + triggers), the real store, the real service, and
// the real router. Create → replay → revise → stale-conflict must behave
// exactly as the contract promises over HTTP.
func TestOutcomeRoutesFunctionalThroughRealStore(t *testing.T) {
	storeHandle := sqlitetest.MustOpen(t)
	ctx := context.Background()
	if err := storeHandle.UpsertProject(ctx, domain.ProjectRecord{
		ID: "mer", Path: "/tmp/kennel-outcome-fixture", RegisteredAt: time.Now().UTC(),
		Kind: domain.ProjectKindSingleRepo,
	}); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	svc := outcomevc.New(storeHandle, nil)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httptest.NewServer(httpd.NewRouterWithControl(config.Config{}, log, nil, httpd.APIDeps{
		Outcomes: svc,
	}, httpd.ControlDeps{}))
	defer srv.Close()

	const createBody = `{"title":"Local Focus Ledger","goal":"Record focus locally.","successCriteria":["Positive minutes create one block."],"review":"Deterministic checks.","requestKey":"req-e2e-1"}`

	// 1) Create lands ContractRevision 1.
	respBytes, status, _ := doRequest(t, srv, http.MethodPost, "/api/v1/projects/mer/outcomes", createBody)
	if status != http.StatusCreated {
		t.Fatalf("create = %d: %s", status, respBytes)
	}
	if !strings.Contains(string(respBytes), `"currentRevisionNumber":1`) ||
		!strings.Contains(string(respBytes), `"id":"out-`) {
		t.Fatalf("create payload unexpected: %s", respBytes)
	}
	id := extractOutcomeID(t, respBytes)

	// 2) Replay with the same key returns the SAME canonical object.
	replayBytes, replayStatus, _ := doRequest(t, srv, http.MethodPost, "/api/v1/projects/mer/outcomes", createBody)
	if replayStatus != http.StatusCreated || extractOutcomeID(t, replayBytes) != id {
		t.Fatalf("replay = %d ids %s vs %s", replayStatus, id, extractOutcomeID(t, replayBytes))
	}

	// 3) Revise against revision 1 appends immutable revision 2.
	revBody := `{"expectedRevision":1,"goal":"Revised goal.","successCriteria":["A","B"],"review":"checks"}`
	revBytes, revStatus, _ := doRequest(t, srv, http.MethodPost, "/api/v1/outcomes/"+id+"/revisions", revBody)
	if revStatus != http.StatusOK || !strings.Contains(string(revBytes), `"currentRevisionNumber":2`) {
		t.Fatalf("revise = %d: %s", revStatus, revBytes)
	}
	if !strings.Contains(string(revBytes), `"number":1`) || !strings.Contains(string(revBytes), `"number":2`) {
		t.Fatalf("revision history missing both entries: %s", revBytes)
	}

	// 4) A stale expected revision is a 409 that changes nothing.
	staleBytes, staleStatus, _ := doRequest(t, srv, http.MethodPost, "/api/v1/outcomes/"+id+"/revisions", revBody)
	if staleStatus != http.StatusConflict || !strings.Contains(string(staleBytes), `"code":"OUTCOME_CONTRACT_CONFLICT"`) {
		t.Fatalf("stale revise = %d: %s", staleStatus, staleBytes)
	}
	readBytes, readStatus, _ := doRequest(t, srv, http.MethodGet, "/api/v1/outcomes/"+id, "")
	if readStatus != http.StatusOK ||
		!strings.Contains(string(readBytes), `"currentRevisionNumber":2`) ||
		strings.Contains(string(readBytes), `"number":3`) {
		t.Fatalf("post-conflict read = %d: %s", readStatus, readBytes)
	}
}

func extractOutcomeID(t *testing.T, body []byte) string {
	t.Helper()
	s := string(body)
	marker := `"id":"out-`
	i := strings.Index(s, marker)
	if i < 0 {
		t.Fatalf("no outcome id in body: %s", s)
	}
	rest := s[i+len(marker):]
	j := strings.Index(rest, `"`)
	if j < 0 {
		t.Fatalf("malformed id in body: %s", s)
	}
	return "out-" + rest[:j]
}

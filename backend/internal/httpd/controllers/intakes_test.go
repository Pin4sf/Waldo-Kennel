package controllers_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/config"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd"
	intakevc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/intake"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func TestSharedIntakeRoutesStartWithSimplePromptAndPreserveConflictEnvelope(t *testing.T) {
	store := sqlitetest.MustOpen(t)
	if err := store.UpsertProject(t.Context(), domain.ProjectRecord{ID: "intake-api", Path: "/tmp/intake-api", RegisteredAt: time.Now().UTC()}); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	service := intakevc.New(store, intakevc.NewRuleBasedAnalyzer(), nil)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(httpd.NewRouterWithControl(config.Config{}, log, nil, httpd.APIDeps{Intakes: service}, httpd.ControlDeps{}))
	t.Cleanup(server.Close)

	body, status, _ := doRequest(t, server, http.MethodPost, "/api/v1/projects/intake-api/intakes", `{"sourceSurface":"work","statement":"Add keyboard navigation to settings","requestKey":"capture-api-1"}`)
	if status != http.StatusCreated {
		t.Fatalf("capture = %d: %s", status, body)
	}
	var captured struct {
		Intake struct {
			Session struct {
				ID, Status, Statement   string
				CurrentProposalRevision int64 `json:"currentProposalRevision"`
			} `json:"session"`
			Proposal any `json:"proposal"`
		} `json:"intake"`
	}
	if err := json.Unmarshal(body, &captured); err != nil {
		t.Fatalf("decode capture: %v", err)
	}
	if captured.Intake.Session.Status != "captured" || captured.Intake.Session.Statement == "" || captured.Intake.Proposal != nil {
		t.Fatalf("first response is not simple captured intent: %s", body)
	}

	path := "/api/v1/intakes/" + captured.Intake.Session.ID + "/analysis"
	body, status, _ = doRequest(t, server, http.MethodPost, path, `{"expectedProposalRevision":0}`)
	if status != http.StatusOK || !json.Valid(body) {
		t.Fatalf("analysis = %d: %s", status, body)
	}
	if !containsJSON(body, `"status":"ready"`) || !containsJSON(body, `"revision":1`) {
		t.Fatalf("analysis did not produce proposal: %s", body)
	}

	body, status, _ = doRequest(t, server, http.MethodPost, path, `{"expectedProposalRevision":0}`)
	if status != http.StatusConflict {
		t.Fatalf("stale = %d: %s", status, body)
	}
	var failure struct{ Code, RequestID string }
	if err := json.Unmarshal(body, &failure); err != nil {
		t.Fatalf("decode conflict: %v", err)
	}
	if failure.Code != "INTAKE_REVISION_CONFLICT" || failure.RequestID == "" {
		t.Fatalf("conflict envelope = %+v body=%s", failure, body)
	}
}

func TestCreateIntakeMissingProjectIs404Envelope(t *testing.T) {
	store := sqlitetest.MustOpen(t)
	service := intakevc.New(store, intakevc.NewRuleBasedAnalyzer(), nil)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(httpd.NewRouterWithControl(config.Config{}, log, nil, httpd.APIDeps{Intakes: service}, httpd.ControlDeps{}))
	t.Cleanup(server.Close)
	body, status, _ := doRequest(t, server, http.MethodPost, "/api/v1/projects/missing/intakes", `{"sourceSurface":"work","statement":"Do work","requestKey":"capture-missing"}`)
	if status != http.StatusNotFound || !containsJSON(body, `"code":"PROJECT_NOT_FOUND"`) || !containsJSON(body, `"requestId":`) {
		t.Fatalf("missing project = %d: %s", status, body)
	}
}

func TestCancelIntakeReturnsDurableReasonAndRequestIDOnStaleRevision(t *testing.T) {
	store := sqlitetest.MustOpen(t)
	if err := store.UpsertProject(t.Context(), domain.ProjectRecord{ID: "cancel-api", Path: "/tmp/cancel-api", RegisteredAt: time.Now().UTC()}); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	service := intakevc.New(store, intakevc.NewRuleBasedAnalyzer(), nil)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(httpd.NewRouterWithControl(config.Config{}, log, nil, httpd.APIDeps{Intakes: service}, httpd.ControlDeps{}))
	t.Cleanup(server.Close)
	body, status, _ := doRequest(t, server, http.MethodPost, "/api/v1/projects/cancel-api/intakes", `{"sourceSurface":"work","statement":"Release this intake","requestKey":"capture-cancel-api"}`)
	if status != http.StatusCreated {
		t.Fatalf("capture=%d: %s", status, body)
	}
	var captured struct {
		Intake struct {
			Session struct {
				ID string `json:"id"`
			} `json:"session"`
		} `json:"intake"`
	}
	if err := json.Unmarshal(body, &captured); err != nil {
		t.Fatalf("decode: %v", err)
	}
	path := "/api/v1/intakes/" + captured.Intake.Session.ID + "/cancellation"
	body, status, _ = doRequest(t, server, http.MethodPost, path, `{"expectedProposalRevision":1,"reason":"No longer needed"}`)
	if status != http.StatusConflict || !containsJSON(body, `"code":"INTAKE_REVISION_CONFLICT"`) || !containsJSON(body, `"requestId":`) {
		t.Fatalf("stale=%d: %s", status, body)
	}
	body, status, _ = doRequest(t, server, http.MethodPost, path, `{"expectedProposalRevision":0,"reason":"No longer needed"}`)
	if status != http.StatusOK || !containsJSON(body, `"status":"cancelled"`) || !containsJSON(body, `"cancellationReason":"No longer needed"`) {
		t.Fatalf("cancel=%d: %s", status, body)
	}
}

func containsJSON(body []byte, needle string) bool {
	return len(body) >= len(needle) && stringContains(string(body), needle)
}
func stringContains(value, needle string) bool {
	for index := 0; index+len(needle) <= len(value); index++ {
		if value[index:index+len(needle)] == needle {
			return true
		}
	}
	return false
}

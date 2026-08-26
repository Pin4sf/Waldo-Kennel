package controllers_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/config"
	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd"
	waldovc "github.com/aoagents/agent-orchestrator/backend/internal/service/waldoconversation"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/sqlitetest"
)

func TestProjectWaldoConversationRoutesPersistOrderedIdempotentTurnsAndContextAcrossRestart(t *testing.T) {
	store := sqlitetest.MustOpen(t)
	if err := store.UpsertProject(t.Context(), domain.ProjectRecord{
		ID: "waldo-api", DisplayName: "Waldo API", Path: "/tmp/waldo-api", RegisteredAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	service := waldovc.New(store, nil, nil)
	server := newProjectWaldoServer(t, service)

	body, status, _ := doRequest(t, server, http.MethodPost, "/api/v1/projects/waldo-api/waldo-conversation", `{}`)
	if status != http.StatusCreated {
		t.Fatalf("open conversation = %d: %s", status, body)
	}
	var opened waldoConversationEnvelope
	decodeWaldoEnvelope(t, body, &opened)
	if opened.Conversation.ProjectID != "waldo-api" || opened.Conversation.Revision != 0 {
		t.Fatalf("opened conversation = %+v", opened.Conversation)
	}

	body, status, _ = doRequest(t, server, http.MethodPost, "/api/v1/projects/waldo-api/waldo-conversation/episodes",
		`{"expectedRevision":0,"requestKey":"episode-api-1"}`)
	if status != http.StatusCreated {
		t.Fatalf("open episode = %d: %s", status, body)
	}
	var episodeOpened waldoConversationEnvelope
	decodeWaldoEnvelope(t, body, &episodeOpened)
	if episodeOpened.Conversation.Revision != 1 || len(episodeOpened.Episodes) != 1 || episodeOpened.Episodes[0].State != "active" {
		t.Fatalf("episode response = %+v", episodeOpened)
	}
	episodeID := episodeOpened.Episodes[0].ID

	turnRequest := `{"expectedRevision":1,"episodeId":"` + episodeID + `","role":"user","message":"What changed?","requestKey":"turn-api-1"}`
	body, status, _ = doRequest(t, server, http.MethodPost, "/api/v1/projects/waldo-api/waldo-conversation/turns", turnRequest)
	if status != http.StatusCreated {
		t.Fatalf("append turn = %d: %s", status, body)
	}
	var appended struct {
		Turn struct {
			ID       string `json:"id"`
			Sequence int64  `json:"sequence"`
			Message  string `json:"message"`
		} `json:"turn"`
		WaldoConversation waldoConversationEnvelope `json:"waldoConversation"`
	}
	if err := json.Unmarshal(body, &appended); err != nil {
		t.Fatalf("decode turn: %v: %s", err, body)
	}
	if appended.Turn.Sequence != 1 || appended.Turn.Message != "What changed?" || appended.WaldoConversation.Conversation.Revision != 2 {
		t.Fatalf("append response = %+v", appended)
	}

	replayBody, replayStatus, _ := doRequest(t, server, http.MethodPost, "/api/v1/projects/waldo-api/waldo-conversation/turns", turnRequest)
	if replayStatus != http.StatusOK {
		t.Fatalf("replay = %d: %s", replayStatus, replayBody)
	}
	var replayed struct {
		Turn struct {
			ID string `json:"id"`
		} `json:"turn"`
	}
	if err := json.Unmarshal(replayBody, &replayed); err != nil || replayed.Turn.ID != appended.Turn.ID {
		t.Fatalf("replay changed turn identity: %+v err=%v body=%s", replayed, err, replayBody)
	}

	conflictBody, conflictStatus, _ := doRequest(t, server, http.MethodPost, "/api/v1/projects/waldo-api/waldo-conversation/turns",
		`{"expectedRevision":2,"episodeId":"`+episodeID+`","role":"user","message":"Different","requestKey":"turn-api-1"}`)
	if conflictStatus != http.StatusConflict || !containsJSON(conflictBody, `"code":"WALDO_CONVERSATION_IDEMPOTENCY_CONFLICT"`) || !containsJSON(conflictBody, `"requestId":`) {
		t.Fatalf("idempotency conflict = %d: %s", conflictStatus, conflictBody)
	}

	body, status, _ = doRequest(t, server, http.MethodPost, "/api/v1/projects/waldo-api/waldo-conversation/context",
		`{"expectedRevision":2,"ref":{"kind":"project","objectId":"waldo-api","provenance":{"kind":"user","sourceId":"waldo-rail"}},"requestKey":"context-api-1"}`)
	if status != http.StatusCreated {
		t.Fatalf("attach context = %d: %s", status, body)
	}
	var attached waldoConversationEnvelope
	decodeWaldoEnvelope(t, body, &attached)
	if attached.Conversation.Revision != 3 || len(attached.ContextAttachments) != 1 || !attached.ContextAttachments[0].Active {
		t.Fatalf("attach response = %+v", attached)
	}
	attachmentID := attached.ContextAttachments[0].ID

	body, status, _ = doRequest(t, server, http.MethodPost,
		"/api/v1/projects/waldo-api/waldo-conversation/context/"+attachmentID+"/detach",
		`{"expectedRevision":3,"reason":"Return to Project context","requestKey":"context-api-2"}`)
	if status != http.StatusOK {
		t.Fatalf("detach context = %d: %s", status, body)
	}
	var detached waldoConversationEnvelope
	decodeWaldoEnvelope(t, body, &detached)
	if detached.Conversation.Revision != 4 || detached.ContextAttachments[0].Active || detached.ContextAttachments[0].DetachReason == "" {
		t.Fatalf("detach response = %+v", detached)
	}

	server.Close()
	restarted := newProjectWaldoServer(t, waldovc.New(store, nil, nil))
	body, status, _ = doRequest(t, restarted, http.MethodGet, "/api/v1/projects/waldo-api/waldo-conversation", "")
	if status != http.StatusOK {
		t.Fatalf("restart read = %d: %s", status, body)
	}
	var restored waldoConversationEnvelope
	decodeWaldoEnvelope(t, body, &restored)
	if restored.Conversation.Revision != 4 || len(restored.Turns) != 1 || restored.Turns[0].Sequence != 1 || restored.Turns[0].Message != "What changed?" || restored.ContextAttachments[0].Active {
		t.Fatalf("restart response = %+v", restored)
	}
}

func TestProjectWaldoConversationRoutesPreserveTypedErrorsAndRequestIDs(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	unwired := httptest.NewServer(httpd.NewRouterWithControl(config.Config{}, log, nil, httpd.APIDeps{}, httpd.ControlDeps{}))
	t.Cleanup(unwired.Close)
	body, status, _ := doRequest(t, unwired, http.MethodGet, "/api/v1/projects/project-1/waldo-conversation", "")
	if status != http.StatusNotImplemented || !containsJSON(body, `"code":"NOT_IMPLEMENTED"`) || !containsJSON(body, `"requestId":`) {
		t.Fatalf("unwired route = %d: %s", status, body)
	}

	store := sqlitetest.MustOpen(t)
	server := newProjectWaldoServer(t, waldovc.New(store, nil, nil))
	body, status, _ = doRequest(t, server, http.MethodGet, "/api/v1/projects/project-1/waldo-conversation", "")
	if status != http.StatusNotFound || !containsJSON(body, `"code":"WALDO_CONVERSATION_NOT_FOUND"`) || !containsJSON(body, `"requestId":`) {
		t.Fatalf("missing conversation = %d: %s", status, body)
	}
	body, status, _ = doRequest(t, server, http.MethodPost, "/api/v1/projects/project-1/waldo-conversation/turns", `{"unknown":true}`)
	if status != http.StatusBadRequest || !containsJSON(body, `"code":"INVALID_JSON"`) || !containsJSON(body, `"requestId":`) {
		t.Fatalf("invalid request = %d: %s", status, body)
	}
}

func TestProjectWaldoContinuationRoutePreservesUnwiredFactsErrorAndRequestID(t *testing.T) {
	store := sqlitetest.MustOpen(t)
	if err := store.UpsertProject(t.Context(), domain.ProjectRecord{
		ID: "waldo-continuation", DisplayName: "Waldo Continuation", Path: "/tmp/waldo-continuation", RegisteredAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	service := waldovc.New(store, nil, nil)
	server := newProjectWaldoServer(t, service)
	if _, status, _ := doRequest(t, server, http.MethodPost, "/api/v1/projects/waldo-continuation/waldo-conversation", `{}`); status != http.StatusCreated {
		t.Fatalf("open conversation = %d, want 201", status)
	}

	bindings := `{
		"projectId":"waldo-continuation",
		"outcomeId":"outcome-1",
		"contractRevisionId":"contract-1",
		"planRevisionId":"plan-1",
		"workUnitId":"work-unit-1",
		"attemptId":"attempt-1",
		"provider":"codex",
		"model":"gpt-5.6",
		"profile":"default",
		"role":"implementer",
		"authorityDigest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"budgetDigest":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"workspaceOwner":"worktree-1",
		"effectPolicyDigest":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	}`
	request := `{
		"fromAgentSessionRef":"session-ref-1",
		"reason":"context_reserve",
		"reasonDetail":"Provider context reserve reached",
		"triggerEvidence":{"kind":"provider_context_meter","reference":"meter-1"},
		"contextDigest":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"previousBindings":` + bindings + `,
		"replacementBindings":` + bindings + `,
		"effectsKnown":true,
		"lostMaterialContext":false,
		"sourceRevoked":false,
		"freshVerifier":false,
		"requestKey":"continuation-unwired-1"
	}`
	body, status, _ := doRequest(t, server, http.MethodPost,
		"/api/v1/projects/waldo-continuation/waldo-conversation/continuations", request)
	if status != http.StatusInternalServerError {
		t.Fatalf("unwired continuation = %d, want 500: %s", status, body)
	}
	var errorEnvelope struct {
		Code      string `json:"code"`
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(body, &errorEnvelope); err != nil {
		t.Fatalf("decode continuation error: %v: %s", err, body)
	}
	if errorEnvelope.Code != waldovc.CodeContinuationFactsUnwired || errorEnvelope.RequestID == "" {
		t.Fatalf("continuation error = %+v, want %s with requestId", errorEnvelope, waldovc.CodeContinuationFactsUnwired)
	}
}

type waldoConversationEnvelope struct {
	Conversation struct {
		ID        string `json:"id"`
		ProjectID string `json:"projectId"`
		Revision  int64  `json:"revision"`
	} `json:"conversation"`
	Episodes []struct {
		ID    string `json:"id"`
		State string `json:"state"`
	} `json:"episodes"`
	Turns []struct {
		Sequence int64  `json:"sequence"`
		Message  string `json:"message"`
	} `json:"turns"`
	ContextAttachments []struct {
		ID           string `json:"id"`
		Active       bool   `json:"active"`
		DetachReason string `json:"detachReason"`
	} `json:"contextAttachments"`
}

func newProjectWaldoServer(t *testing.T, service *waldovc.Service) *httptest.Server {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(httpd.NewRouterWithControl(config.Config{}, log, nil,
		httpd.APIDeps{WaldoConversations: service}, httpd.ControlDeps{}))
	t.Cleanup(server.Close)
	return server
}

func decodeWaldoEnvelope(t *testing.T, body []byte, target *waldoConversationEnvelope) {
	t.Helper()
	var envelope struct {
		WaldoConversation waldoConversationEnvelope `json:"waldoConversation"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode Waldo conversation: %v: %s", err, body)
	}
	*target = envelope.WaldoConversation
}

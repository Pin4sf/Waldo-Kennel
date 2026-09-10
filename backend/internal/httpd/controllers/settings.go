package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apispec"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/envelope"
	settingssvc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/settings"
)

// SettingsService is the controller-facing preferences contract.
type SettingsService interface {
	Get(ctx context.Context) (settingssvc.Snapshot, error)
	SetDefaultSessionMode(ctx context.Context, mode domain.SessionMode) (settingssvc.Snapshot, error)
	GetReasoning(ctx context.Context) (settingssvc.ReasoningStatus, error)
	SetReasoning(ctx context.Context, input settingssvc.ReasoningInput) (settingssvc.ReasoningStatus, error)
	ChatHarnesses(candidates []domain.AgentHarness) []domain.AgentHarness
}

// SettingsController owns the daemon-owned preference routes.
//
// These are daemon-owned rather than renderer-owned on purpose: desktop, mobile,
// and the CLI all resolve the same value, so a preference held in one client would
// disagree with the others.
type SettingsController struct {
	Svc SettingsService
}

// Register mounts the settings routes.
func (c *SettingsController) Register(r chi.Router) {
	r.Get("/settings", c.get)
	r.Patch("/settings/session-interface", c.setSessionInterface)
	r.Patch("/settings/reasoning", c.setReasoning)
	r.Post("/settings/reasoning/verification", c.verifyReasoning)
}

func (c *SettingsController) get(w http.ResponseWriter, r *http.Request) {
	if c.Svc == nil {
		apispec.NotImplemented(w, r, "GET", "/api/v1/settings")
		return
	}
	snapshot, err := c.Svc.Get(r.Context())
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, c.response(r.Context(), snapshot))
}

func (c *SettingsController) setReasoning(w http.ResponseWriter, r *http.Request) {
	if c.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPatch, "/api/v1/settings/reasoning")
		return
	}
	var req UpdateReasoningRequest
	if !decodeConversationBody(w, r, &req) {
		return
	}
	status, err := c.Svc.SetReasoning(r.Context(), settingssvc.ReasoningInput{
		Provider: req.Provider, Model: req.Model, Effort: req.Effort, APIKey: req.APIKey, ClearKey: req.ClearKey,
	})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, reasoningResponse(status))
}

func (c *SettingsController) setSessionInterface(w http.ResponseWriter, r *http.Request) {
	if c.Svc == nil {
		apispec.NotImplemented(w, r, "PATCH", "/api/v1/settings/session-interface")
		return
	}
	var req UpdateSessionInterfaceRequest
	if !decodeConversationBody(w, r, &req) {
		return
	}

	// Parsed strictly: an unrecognized value is rejected rather than collapsing to
	// a default the caller did not ask for.
	mode, err := domain.ParseSessionMode(req.DefaultSessionMode)
	if err != nil || mode == "" {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "validation",
			"SESSION_MODE_INVALID", `defaultSessionMode must be "chat" or "tui"`, nil)
		return
	}

	snapshot, err := c.Svc.SetDefaultSessionMode(r.Context(), mode)
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, c.response(r.Context(), snapshot))
}

func (c *SettingsController) response(ctx context.Context, snapshot settingssvc.Snapshot) SettingsResponse {
	// Reported so the client can warn that choosing chat narrows which agents are
	// available, instead of letting the user discover it at spawn time.
	chatHarnesses := c.Svc.ChatHarnesses(domain.AllHarnesses)
	names := make([]string, 0, len(chatHarnesses))
	for _, harness := range chatHarnesses {
		names = append(names, string(harness))
	}
	reasoning, err := c.Svc.GetReasoning(ctx)
	if err != nil {
		reasoning = settingssvc.ReasoningStatus{ErrorCode: "REASONING_STATUS_UNAVAILABLE", Error: "Reasoning readiness is unavailable"}
	}
	return SettingsResponse{
		DefaultSessionMode: string(snapshot.DefaultSessionMode),
		ChatHarnesses:      names,
		Reasoning:          reasoningResponse(reasoning),
	}
}

func reasoningResponse(status settingssvc.ReasoningStatus) ReasoningResponse {
	out := ReasoningResponse{
		Mode: status.Mode, Provider: status.Provider, Model: status.Model, Effort: status.Effort,
		Configured: status.Configured, Ready: status.Ready, KeyConfigured: status.KeyConfigured,
		Verified: status.Verified, ErrorCode: status.ErrorCode, Error: status.Error,
	}
	if status.VerifiedAt != nil {
		stamp := status.VerifiedAt.UTC().Format(time.RFC3339)
		out.VerifiedAt = &stamp
	}
	return out
}

// verifyReasoning runs one owner-triggered probe against the configured
// provider. It is a POST because it performs a real, possibly billed call: the
// daemon never probes on its own schedule.
func (c *SettingsController) verifyReasoning(w http.ResponseWriter, r *http.Request) {
	verifier, ok := c.Svc.(interface {
		VerifyReasoning(context.Context) (settingssvc.ReasoningStatus, error)
	})
	if c.Svc == nil || !ok {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/settings/reasoning/verification")
		return
	}
	status, err := verifier.VerifyReasoning(r.Context())
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, reasoningResponse(status))
}

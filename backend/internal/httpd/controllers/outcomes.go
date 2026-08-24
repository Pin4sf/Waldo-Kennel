package controllers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apispec"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/envelope"
	outcomevc "github.com/aoagents/agent-orchestrator/backend/internal/service/outcome"
)

// OutcomeService is the controller-facing Outcome contract (#21) plus the
// Decide & Authorize boundary (#26).
type OutcomeService interface {
	Create(ctx context.Context, in outcomevc.CreateInput) (outcomevc.OutcomeView, error)
	ReviseContract(ctx context.Context, id domain.OutcomeID, in outcomevc.ReviseContractInput) (outcomevc.OutcomeView, error)
	Get(ctx context.Context, id domain.OutcomeID) (outcomevc.OutcomeView, error)
	ProposePlan(ctx context.Context, id domain.OutcomeID, expectedContractRevision int64) (outcomevc.PlanView, error)
	ApprovePlan(ctx context.Context, id domain.OutcomeID, in outcomevc.ApprovePlanInput) (outcomevc.AuthorizedPlanView, error)
	GetLatestPlan(ctx context.Context, id domain.OutcomeID) (outcomevc.PlanView, error)
}

// AttemptManager is the Act & Observe boundary (#31). A nil field answers 501
// on the attempt routes, mirroring every other unwired capability.
type AttemptManager interface {
	StartAttempt(ctx context.Context, outcomeID domain.OutcomeID, in outcomevc.StartAttemptInput) (outcomevc.AttemptView, error)
	GetAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (outcomevc.AttemptView, error)
	ListAttempts(ctx context.Context, outcomeID domain.OutcomeID) ([]outcomevc.AttemptView, error)
	PauseAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (outcomevc.AttemptView, error)
	ResumeAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (outcomevc.AttemptView, error)
	CancelAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (outcomevc.AttemptView, error)
	RecordObservation(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID, in outcomevc.RecordObservationInput) (domain.AttemptObservation, error)
	RecoverAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID, in outcomevc.RecoveryInput) (outcomevc.RecoveryView, error)
}

// OutcomesController owns the canonical Outcome contract routes.
type OutcomesController struct {
	Svc      OutcomeService
	Attempts AttemptManager
}

// Register mounts the Outcome routes on the supplied router.
func (c *OutcomesController) Register(r chi.Router) {
	r.Post("/projects/{id}/outcomes", c.create)
	r.Get("/outcomes/{outcomeId}", c.get)
	r.Post("/outcomes/{outcomeId}/revisions", c.revise)
	r.Post("/outcomes/{outcomeId}/plans", c.proposePlan)
	r.Post("/outcomes/{outcomeId}/plans/{planId}/approval", c.approvePlan)
	r.Get("/outcomes/{outcomeId}/plan", c.latestPlan)
	r.Post("/outcomes/{outcomeId}/attempts", c.startAttempt)
	r.Get("/outcomes/{outcomeId}/attempts", c.listAttempts)
	r.Get("/outcomes/{outcomeId}/attempts/{attemptId}", c.getAttempt)
	r.Post("/outcomes/{outcomeId}/attempts/{attemptId}/observations", c.recordObservation)
	r.Post("/outcomes/{outcomeId}/attempts/{attemptId}/pause", c.pauseAttempt)
	r.Post("/outcomes/{outcomeId}/attempts/{attemptId}/resume", c.resumeAttempt)
	r.Post("/outcomes/{outcomeId}/attempts/{attemptId}/cancel", c.cancelAttempt)
	r.Post("/outcomes/{outcomeId}/attempts/{attemptId}/recovery", c.recoverAttempt)
}

func (c *OutcomesController) proposePlan(w http.ResponseWriter, r *http.Request) {
	if c.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/outcomes/{outcomeId}/plans")
		return
	}
	var req ProposePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	view, err := c.Svc.ProposePlan(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")), req.ExpectedContractRevision)
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusCreated, PlanEnvelope{Plan: planRevisionResponse(view.Plan)})
}

func (c *OutcomesController) approvePlan(w http.ResponseWriter, r *http.Request) {
	if c.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/outcomes/{outcomeId}/plans/{planId}/approval")
		return
	}
	var req ApprovePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	view, err := c.Svc.ApprovePlan(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")), outcomevc.ApprovePlanInput{
		PlanRevisionID:           domain.PlanRevisionID(chi.URLParam(r, "planId")),
		ExpectedContractRevision: req.ExpectedContractRevision,
	})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, PlanEnvelope{Plan: planRevisionResponse(view.Plan)})
}

func (c *OutcomesController) latestPlan(w http.ResponseWriter, r *http.Request) {
	if c.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodGet, "/api/v1/outcomes/{outcomeId}/plan")
		return
	}
	view, err := c.Svc.GetLatestPlan(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, PlanEnvelope{Plan: planRevisionResponse(view.Plan)})
}

func (c *OutcomesController) create(w http.ResponseWriter, r *http.Request) {
	if c.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/projects/{id}/outcomes")
		return
	}
	var req CreateOutcomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	view, err := c.Svc.Create(r.Context(), outcomevc.CreateInput{
		ProjectID:       domain.ProjectID(chi.URLParam(r, "id")),
		Title:           req.Title,
		Goal:            req.Goal,
		SuccessCriteria: req.SuccessCriteria,
		Review:          req.Review,
		Constraints:     req.Constraints,
		NonGoals:        req.NonGoals,
		Clarification:   req.Clarification,
		RequestKey:      req.RequestKey,
	})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusCreated, OutcomeEnvelope{Outcome: outcomeResponse(view)})
}

func (c *OutcomesController) get(w http.ResponseWriter, r *http.Request) {
	if c.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodGet, "/api/v1/outcomes/{outcomeId}")
		return
	}
	view, err := c.Svc.Get(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, OutcomeEnvelope{Outcome: outcomeResponse(view)})
}

func (c *OutcomesController) revise(w http.ResponseWriter, r *http.Request) {
	if c.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/outcomes/{outcomeId}/revisions")
		return
	}
	var req ReviseOutcomeContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	view, err := c.Svc.ReviseContract(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")), outcomevc.ReviseContractInput{
		ExpectedRevision: req.ExpectedRevision,
		Goal:             req.Goal,
		SuccessCriteria:  req.SuccessCriteria,
		Review:           req.Review,
		Constraints:      req.Constraints,
		NonGoals:         req.NonGoals,
		Clarification:    req.Clarification,
	})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, OutcomeEnvelope{Outcome: outcomeResponse(view)})
}

// startAttempt admits an approved plan onto a real provider session (#31).
func (c *OutcomesController) startAttempt(w http.ResponseWriter, r *http.Request) {
	if c.Attempts == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/outcomes/{outcomeId}/attempts")
		return
	}
	var req StartOutcomeAttemptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	view, err := c.Attempts.StartAttempt(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")), outcomevc.StartAttemptInput{
		PlanRevisionID: domain.PlanRevisionID(req.PlanRevisionID),
		Harness:        domain.AgentHarness(req.Harness),
		RequestKey:     req.RequestKey,
	})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusCreated, AttemptEnvelope{Attempt: attemptResponse(view)})
}

func (c *OutcomesController) listAttempts(w http.ResponseWriter, r *http.Request) {
	if c.Attempts == nil {
		apispec.NotImplemented(w, r, http.MethodGet, "/api/v1/outcomes/{outcomeId}/attempts")
		return
	}
	views, err := c.Attempts.ListAttempts(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	attempts := make([]AttemptResponse, 0, len(views))
	for _, view := range views {
		attempts = append(attempts, attemptResponse(view))
	}
	envelope.WriteJSON(w, http.StatusOK, AttemptListEnvelope{Attempts: attempts})
}

func (c *OutcomesController) getAttempt(w http.ResponseWriter, r *http.Request) {
	if c.Attempts == nil {
		apispec.NotImplemented(w, r, http.MethodGet, "/api/v1/outcomes/{outcomeId}/attempts/{attemptId}")
		return
	}
	view, err := c.Attempts.GetAttempt(r.Context(),
		domain.OutcomeID(chi.URLParam(r, "outcomeId")),
		domain.AttemptID(chi.URLParam(r, "attemptId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, AttemptEnvelope{Attempt: attemptResponse(view)})
}

func (c *OutcomesController) recordObservation(w http.ResponseWriter, r *http.Request) {
	if c.Attempts == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/outcomes/{outcomeId}/attempts/{attemptId}/observations")
		return
	}
	var req RecordObservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	obs, err := c.Attempts.RecordObservation(r.Context(),
		domain.OutcomeID(chi.URLParam(r, "outcomeId")),
		domain.AttemptID(chi.URLParam(r, "attemptId")),
		outcomevc.RecordObservationInput{Kind: req.Kind, Payload: req.Payload})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusCreated, ObservationEnvelope{Observation: attemptObservationResponse(obs)})
}

func (c *OutcomesController) pauseAttempt(w http.ResponseWriter, r *http.Request) {
	if c.Attempts == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/outcomes/{outcomeId}/attempts/{attemptId}/pause")
		return
	}
	view, err := c.Attempts.PauseAttempt(r.Context(),
		domain.OutcomeID(chi.URLParam(r, "outcomeId")),
		domain.AttemptID(chi.URLParam(r, "attemptId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, AttemptEnvelope{Attempt: attemptResponse(view)})
}

func (c *OutcomesController) resumeAttempt(w http.ResponseWriter, r *http.Request) {
	if c.Attempts == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/outcomes/{outcomeId}/attempts/{attemptId}/resume")
		return
	}
	view, err := c.Attempts.ResumeAttempt(r.Context(),
		domain.OutcomeID(chi.URLParam(r, "outcomeId")),
		domain.AttemptID(chi.URLParam(r, "attemptId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, AttemptEnvelope{Attempt: attemptResponse(view)})
}

func (c *OutcomesController) cancelAttempt(w http.ResponseWriter, r *http.Request) {
	if c.Attempts == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/outcomes/{outcomeId}/attempts/{attemptId}/cancel")
		return
	}
	view, err := c.Attempts.CancelAttempt(r.Context(),
		domain.OutcomeID(chi.URLParam(r, "outcomeId")),
		domain.AttemptID(chi.URLParam(r, "attemptId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, AttemptEnvelope{Attempt: attemptResponse(view)})
}

func (c *OutcomesController) recoverAttempt(w http.ResponseWriter, r *http.Request) {
	if c.Attempts == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/outcomes/{outcomeId}/attempts/{attemptId}/recovery")
		return
	}
	var req AttemptRecoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	view, err := c.Attempts.RecoverAttempt(r.Context(),
		domain.OutcomeID(chi.URLParam(r, "outcomeId")),
		domain.AttemptID(chi.URLParam(r, "attemptId")),
		outcomevc.RecoveryInput{Action: outcomevc.RecoveryAction(req.Action)})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	resp := AttemptRecoveryEnvelope{Attempt: attemptResponse(view.Attempt)}
	if view.Receipt != nil {
		receipt := recoveryReceiptResponse(*view.Receipt)
		resp.Receipt = &receipt
	}
	envelope.WriteJSON(w, http.StatusOK, resp)
}

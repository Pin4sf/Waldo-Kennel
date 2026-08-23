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

// OutcomesController owns the canonical Outcome contract routes.
type OutcomesController struct {
	Svc OutcomeService
}

// Register mounts the Outcome routes on the supplied router.
func (c *OutcomesController) Register(r chi.Router) {
	r.Post("/projects/{id}/outcomes", c.create)
	r.Get("/outcomes/{outcomeId}", c.get)
	r.Post("/outcomes/{outcomeId}/revisions", c.revise)
	r.Post("/outcomes/{outcomeId}/plans", c.proposePlan)
	r.Post("/outcomes/{outcomeId}/plans/{planId}/approval", c.approvePlan)
	r.Get("/outcomes/{outcomeId}/plan", c.latestPlan)
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

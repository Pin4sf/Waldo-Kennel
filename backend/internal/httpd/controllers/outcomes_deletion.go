package controllers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/envelope"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

type outcomeDeletionManager interface {
	PreviewOutcomeDeletion(context.Context, domain.OutcomeID) (ports.OutcomeDeletionPreview, error)
	ListTrashedOutcomes(context.Context, domain.ProjectID) ([]ports.OutcomeTrashEntry, error)
	ChangeOutcomeDeletion(context.Context, domain.OutcomeID, int64, string, string) error
}

func (c *OutcomesController) deletionManager(w http.ResponseWriter, r *http.Request) outcomeDeletionManager {
	manager, ok := c.Svc.(outcomeDeletionManager)
	if !ok {
		envelope.WriteAPIError(w, r, http.StatusNotImplemented, "not_implemented", "OUTCOME_DELETION_UNAVAILABLE", "Outcome deletion is unavailable", nil)
		return nil
	}
	return manager
}
func (c *OutcomesController) deletionPreview(w http.ResponseWriter, r *http.Request) {
	manager := c.deletionManager(w, r)
	if manager == nil {
		return
	}
	p, err := manager.PreviewOutcomeDeletion(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, OutcomeDeletionEnvelope{Deletion: p})
}
func (c *OutcomesController) trashedOutcomes(w http.ResponseWriter, r *http.Request) {
	manager := c.deletionManager(w, r)
	if manager == nil {
		return
	}
	items, err := manager.ListTrashedOutcomes(r.Context(), domain.ProjectID(chi.URLParam(r, "id")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, OutcomeTrashEnvelope{Outcomes: items})
}
func (c *OutcomesController) changeDeletion(w http.ResponseWriter, r *http.Request) {
	manager := c.deletionManager(w, r)
	if manager == nil {
		return
	}
	var req ChangeOutcomeDeletionRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid deletion request", nil)
		return
	}
	err := manager.ChangeOutcomeDeletion(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")), req.Revision, req.Action, req.Confirmation)
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, OutcomeDeletionResult{Action: req.Action})
}

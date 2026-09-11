package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apispec"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/envelope"
	outcomevc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

// DeliveryManager is the controller-facing durable delivery boundary.
type DeliveryManager interface {
	DeliveriesEnabled() bool
	ListDeliveries(context.Context, domain.OutcomeID) ([]domain.OutcomeDelivery, error)
	GetDelivery(context.Context, domain.OutcomeID, domain.DeliveryID) (domain.OutcomeDelivery, error)
	RequestDelivery(context.Context, domain.OutcomeID, outcomevc.RequestDeliveryInput) (domain.OutcomeDelivery, error)
}

// RunStateReader is the Mission supervision read boundary: where an Outcome
// stands and what the owner may do next. It is deliberately separate from the
// write interfaces so a daemon that can report state is never assumed able to
// change it.
type RunStateReader interface {
	GetRunState(context.Context, domain.OutcomeID) (outcomevc.RunStateView, error)
	ListProjectRunStates(context.Context, domain.ProjectID, bool) ([]outcomevc.RunStateView, error)
}

// RunCommander is the durable run-intent write boundary. RunIntentsEnabled
// answers separately from the method set because a daemon can implement the
// method and still have no storage behind it; that combination has to reach
// the client as an honest unavailable, not a 500.
type RunCommander interface {
	RunIntentsEnabled() bool
	CommandRun(context.Context, domain.OutcomeID, outcomevc.RunCommandInput) (outcomevc.RunStateView, error)
}

// registerRunRoutes mounts the Mission supervision, delivery and attributed
// usage routes.
//
// The write routes are registered before their services exist on purpose: an
// operation that answers 501 with a stable code is a truthful product state
// the renderer can show, whereas a missing route is an opaque 404 that invites
// a hand-written client-side substitute.
func (c *OutcomesController) registerRunRoutes(r chi.Router) {
	r.Get("/projects/{id}/outcome-run-states", c.listProjectRunStates)
	r.Get("/outcomes/{outcomeId}/run", c.getRunState)
	r.Post("/outcomes/{outcomeId}/run", c.commandRun)
	r.Get("/outcomes/{outcomeId}/deliveries", c.listDeliveries)
	r.Post("/outcomes/{outcomeId}/deliveries", c.requestDelivery)
	r.Get("/outcomes/{outcomeId}/deliveries/{deliveryId}", c.getDelivery)
	r.Get("/outcomes/{outcomeId}/usage", c.getOutcomeUsage)
	r.Get("/outcomes/{outcomeId}/documents", c.getDocumentContext)
	r.Post("/outcomes/{outcomeId}/documents", c.selectDocuments)
	r.Post("/outcomes/{outcomeId}/documents/approval", c.approveDocuments)
}

// DocumentContextManager is the supplied-document boundary. DocumentsEnabled
// answers separately from the method set so a daemon that cannot hold
// documents reports it honestly instead of failing mid-write.
type DocumentContextManager interface {
	DocumentsEnabled() bool
	SelectDocuments(context.Context, domain.OutcomeID, []string) (outcomevc.DocumentContextView, error)
	ApproveDocuments(context.Context, domain.OutcomeID, string) (outcomevc.DocumentContextView, error)
	GetDocumentContext(context.Context, domain.OutcomeID) (outcomevc.DocumentContextView, error)
}

func (c *OutcomesController) documents() (DocumentContextManager, bool) {
	manager, ok := c.Svc.(DocumentContextManager)
	return manager, ok && c.Svc != nil && manager.DocumentsEnabled()
}

func (c *OutcomesController) documentsUnavailable(w http.ResponseWriter, r *http.Request) {
	envelope.WriteAPIError(w, r, http.StatusNotImplemented, "not_implemented", "DOCUMENT_CONTEXT_UNAVAILABLE",
		"Supplied-document Outcomes are not available in this daemon", nil)
}

func (c *OutcomesController) getDocumentContext(w http.ResponseWriter, r *http.Request) {
	manager, ok := c.documents()
	if !ok {
		c.documentsUnavailable(w, r)
		return
	}
	view, err := manager.GetDocumentContext(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, OutcomeDocumentContextEnvelope{DocumentContext: documentContextResponse(view)})
}

func (c *OutcomesController) selectDocuments(w http.ResponseWriter, r *http.Request) {
	manager, ok := c.documents()
	if !ok {
		c.documentsUnavailable(w, r)
		return
	}
	var req SelectOutcomeDocumentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	view, err := manager.SelectDocuments(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")), req.Paths)
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusCreated, OutcomeDocumentContextEnvelope{DocumentContext: documentContextResponse(view)})
}

func (c *OutcomesController) approveDocuments(w http.ResponseWriter, r *http.Request) {
	manager, ok := c.documents()
	if !ok {
		c.documentsUnavailable(w, r)
		return
	}
	var req ApproveOutcomeDocumentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	view, err := manager.ApproveDocuments(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")), req.ExpectedDigest)
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, OutcomeDocumentContextEnvelope{DocumentContext: documentContextResponse(view)})
}

func documentContextResponse(view outcomevc.DocumentContextView) OutcomeDocumentContextResponse {
	sources := make([]DocumentSourceResponse, 0, len(view.Context.Sources))
	for _, source := range view.Context.Sources {
		sources = append(sources, DocumentSourceResponse{
			ID: source.ID, Position: source.Position, SourcePath: source.SourcePath,
			Name: source.Name, ContentDigest: source.ContentDigest, SizeBytes: source.SizeBytes,
		})
	}
	changed := view.ChangedSources
	if changed == nil {
		changed = []string{}
	}
	return OutcomeDocumentContextResponse{
		ID: string(view.Context.ID), OutcomeID: string(view.Context.OutcomeID),
		Revision: view.Context.Revision, Digest: view.Context.Digest,
		State: string(view.Context.State), SelectedAt: view.Context.SelectedAt,
		ApprovedAt: view.Context.ApprovedAt, Sources: sources, ChangedSources: changed,
	}
}

func (c *OutcomesController) runStates() (RunStateReader, bool) {
	reader, ok := c.Svc.(RunStateReader)
	return reader, ok && c.Svc != nil
}

func (c *OutcomesController) getRunState(w http.ResponseWriter, r *http.Request) {
	reader, ok := c.runStates()
	if !ok {
		apispec.NotImplemented(w, r, http.MethodGet, "/api/v1/outcomes/{outcomeId}/run")
		return
	}
	view, err := reader.GetRunState(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, OutcomeRunStateEnvelope{RunState: outcomeRunStateResponse(view)})
}

func (c *OutcomesController) listProjectRunStates(w http.ResponseWriter, r *http.Request) {
	reader, ok := c.runStates()
	if !ok {
		apispec.NotImplemented(w, r, http.MethodGet, "/api/v1/projects/{id}/outcome-run-states")
		return
	}
	// Contributing Outcomes belong inside their parent's Mission, so the Board
	// asks for top-level rows unless the caller explicitly widens the scope.
	topLevelOnly := r.URL.Query().Get("scope") != "all"
	views, err := reader.ListProjectRunStates(r.Context(), domain.ProjectID(chi.URLParam(r, "id")), topLevelOnly)
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	states := make([]OutcomeRunStateResponse, 0, len(views))
	for _, view := range views {
		states = append(states, outcomeRunStateResponse(view))
	}
	envelope.WriteJSON(w, http.StatusOK, OutcomeRunStatesEnvelope{RunStates: states, ObservedAt: time.Now().UTC()})
}

func (c *OutcomesController) commandRun(w http.ResponseWriter, r *http.Request) {
	commander, ok := c.Svc.(RunCommander)
	if !ok || c.Svc == nil || !commander.RunIntentsEnabled() {
		envelope.WriteAPIError(w, r, http.StatusNotImplemented, "not_implemented", "RUN_INTENT_UNAVAILABLE",
			"Durable run intent is not available in this daemon; start Attempts individually", nil)
		return
	}
	var req OutcomeRunCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	view, err := commander.CommandRun(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")), outcomevc.RunCommandInput{
		Command:                  domain.RunCommand(req.Action),
		PlanRevisionID:           domain.PlanRevisionID(req.PlanRevisionID),
		ExpectedContractRevision: req.ExpectedContractRevision,
		ExpectedGeneration:       req.ExpectedGeneration,
		RequestKey:               req.RequestKey,
	})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, OutcomeRunStateEnvelope{RunState: outcomeRunStateResponse(view)})
}

func (c *OutcomesController) listDeliveries(w http.ResponseWriter, r *http.Request) {
	manager, ok := c.deliveryManager()
	if !ok {
		c.deliveryUnavailable(w, r)
		return
	}
	deliveries, err := manager.ListDeliveries(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	items := make([]OutcomeDeliveryResponse, 0, len(deliveries))
	for _, delivery := range deliveries {
		items = append(items, outcomeDeliveryResponse(delivery))
	}
	envelope.WriteJSON(w, http.StatusOK, OutcomeDeliveriesEnvelope{Deliveries: items})
}

func (c *OutcomesController) requestDelivery(w http.ResponseWriter, r *http.Request) {
	manager, ok := c.deliveryManager()
	if !ok {
		c.deliveryUnavailable(w, r)
		return
	}
	var req RequestOutcomeDeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "DELIVERY_REQUEST_INVALID", "Invalid JSON body", nil)
		return
	}
	delivery, err := manager.RequestDelivery(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")), outcomevc.RequestDeliveryInput{
		AttemptID: domain.AttemptID(req.AttemptID), ArtifactVersion: req.ArtifactVersion,
		Destination: req.Destination, Disposition: domain.DeliveryDisposition(req.Disposition),
		AcceptanceDecisionID: domain.AcceptanceDecisionID(req.AcceptanceDecisionID), RequestKey: req.RequestKey,
	})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusCreated, OutcomeDeliveryEnvelope{Delivery: outcomeDeliveryResponse(delivery)})
}

func (c *OutcomesController) getDelivery(w http.ResponseWriter, r *http.Request) {
	manager, ok := c.deliveryManager()
	if !ok {
		c.deliveryUnavailable(w, r)
		return
	}
	delivery, err := manager.GetDelivery(r.Context(), domain.OutcomeID(chi.URLParam(r, "outcomeId")), domain.DeliveryID(chi.URLParam(r, "deliveryId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, OutcomeDeliveryEnvelope{Delivery: outcomeDeliveryResponse(delivery)})
}

func (c *OutcomesController) deliveryManager() (DeliveryManager, bool) {
	manager, ok := c.Svc.(DeliveryManager)
	return manager, ok && c.Svc != nil && manager.DeliveriesEnabled()
}

func (c *OutcomesController) deliveryUnavailable(w http.ResponseWriter, r *http.Request) {
	envelope.WriteAPIError(w, r, http.StatusNotImplemented, "not_implemented", "DELIVERY_UNAVAILABLE",
		"Durable delivery is not available in this daemon", nil)
}

func outcomeDeliveryResponse(delivery domain.OutcomeDelivery) OutcomeDeliveryResponse {
	return OutcomeDeliveryResponse{
		ID: string(delivery.ID), OutcomeID: string(delivery.OutcomeID), AttemptID: string(delivery.AttemptID),
		WorkUnitID: string(delivery.WorkUnitID), ArtifactVersion: delivery.ArtifactVersion,
		Disposition: string(delivery.Disposition), Destination: delivery.Destination, State: string(delivery.State),
		ManifestPath: delivery.ManifestPath, FileCount: delivery.FileCount, ByteCount: delivery.ByteCount,
		FailureCode: delivery.FailureCode, FailureDetail: delivery.FailureDetail,
		RequestedAt: delivery.RequestedAt, CompletedAt: delivery.CompletedAt,
	}
}

func (c *OutcomesController) getOutcomeUsage(w http.ResponseWriter, r *http.Request) {
	envelope.WriteAPIError(w, r, http.StatusNotImplemented, "not_implemented", "USAGE_ATTRIBUTION_UNAVAILABLE",
		"Usage attributed to this Outcome is not implemented in this daemon", nil)
}

func outcomeRunStateResponse(view outcomevc.RunStateView) OutcomeRunStateResponse {
	actions := make([]RunActionEligibilityResponse, 0, len(view.EligibleActions))
	for _, action := range view.EligibleActions {
		actions = append(actions, RunActionEligibilityResponse{
			Action: string(action.Action), Available: action.Available, Reason: action.Reason,
		})
	}
	out := OutcomeRunStateResponse{
		OutcomeID: string(view.OutcomeID), ProjectID: string(view.ProjectID), Title: view.Title,
		ParentOutcomeID: string(view.ParentOutcomeID), State: string(view.State),
		AttentionReason: view.AttentionReason, EligibleActions: actions,
		PlanStatus: string(view.PlanStatus), PlanBindsCurrentContract: view.PlanBindsCurrentContract,
		ActiveAttemptID: string(view.ActiveAttemptID), ActiveAttemptStatus: string(view.ActiveAttemptStatus),
		ProvenCriteria: view.ProvenCriteria, RequiredCriteria: view.RequiredCriteria,
		AcceptedAt: view.AcceptedAt,
		Freshness: RunFreshnessResponse{
			ObservedAt: view.Freshness.ObservedAt, ContractRevisionNumber: view.Freshness.ContractRevisionNumber,
			PlanRevisionID: string(view.Freshness.PlanRevisionID), ProofGeneration: view.Freshness.ProofGeneration,
		},
	}
	if view.Blocker != nil {
		out.Blocker = &RunBlockerResponse{Code: view.Blocker.Code, Message: view.Blocker.Message, Detail: view.Blocker.Detail}
	}
	if view.Intent != nil {
		out.Intent = &RunIntentResponse{
			Generation: view.Intent.Generation, Desired: view.Intent.Desired,
			PlanRevisionID:   string(view.Intent.PlanRevisionID),
			BindsCurrentPlan: view.Intent.BindsCurrentPlan,
			RequestedAt:      view.Intent.RequestedAt,
			AcknowledgedAt:   view.Intent.AcknowledgedAt, ActiveAttemptID: string(view.Intent.ActiveAttemptID),
			LastError: view.Intent.LastError,
		}
	}
	return out
}

package controllers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apispec"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/envelope"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	intakevc "github.com/aoagents/agent-orchestrator/backend/internal/service/intake"
)

// IntakeService defines the shared adaptive-intake application boundary.
type IntakeService interface {
	Capture(context.Context, intakevc.CaptureInput) (ports.IntakeSnapshot, error)
	Get(context.Context, domain.IntakeSessionID) (ports.IntakeSnapshot, error)
	Analyze(context.Context, domain.IntakeSessionID, intakevc.AnalyzeInput) (ports.IntakeSnapshot, error)
	AnswerClarification(context.Context, domain.IntakeSessionID, intakevc.AnswerClarificationInput) (ports.IntakeSnapshot, error)
	ReviseProposal(context.Context, domain.IntakeSessionID, intakevc.ReviseProposalInput) (ports.IntakeSnapshot, error)
	ConfirmOutcome(context.Context, domain.IntakeSessionID, intakevc.ConfirmOutcomeInput) (ports.IntakeSnapshot, error)
	Cancel(context.Context, domain.IntakeSessionID, intakevc.CancelInput) (ports.IntakeSnapshot, error)
	SubmitAgentProposal(context.Context, domain.IntakeAnalysisRequestID, string, ports.IntakeAnalysisResult, string) (ports.IntakeSnapshot, error)
	LatestAnalysisRequest(context.Context, domain.IntakeSessionID) (domain.IntakeAnalysisRequest, bool, error)
	CancelAnalysisRequest(context.Context, domain.IntakeSessionID) (ports.IntakeSnapshot, error)
}

// ResponsibilityLinkService defines explicit Home-to-Work lineage operations.
type ResponsibilityLinkService interface {
	CreateResponsibilityLink(context.Context, intakevc.CreateResponsibilityLinkInput) (domain.ResponsibilityLink, error)
	GetResponsibilityLink(context.Context, domain.ResponsibilityLinkID) (domain.ResponsibilityLink, error)
	EndResponsibilityLink(context.Context, domain.ResponsibilityLinkID, string) (domain.ResponsibilityLink, error)
}

// IntakesController exposes thin HTTP routes over intake services.
type IntakesController struct {
	Svc   IntakeService
	Links ResponsibilityLinkService
}

// Register mounts shared intake and ResponsibilityLink routes.
func (controller *IntakesController) Register(router chi.Router) {
	router.Post("/projects/{id}/intakes", controller.capture)
	router.Get("/intakes/{intakeId}", controller.get)
	router.Post("/intakes/{intakeId}/analysis", controller.analyze)
	router.Post("/intakes/{intakeId}/clarification", controller.answer)
	router.Post("/intakes/{intakeId}/proposals", controller.revise)
	router.Post("/intakes/{intakeId}/confirmation", controller.confirm)
	router.Post("/intakes/{intakeId}/cancellation", controller.cancel)
	router.Get("/intakes/{intakeId}/analysis-request", controller.latestAnalysisRequest)
	router.Post("/intakes/{intakeId}/analysis-request/cancellation", controller.cancelAnalysisRequest)
	// Addressed by REQUEST, not by intake: the answering agent knows only the
	// request it was handed, and scoping the callback that way is what stops a
	// confused agent answering for a different intake.
	router.Post("/intake-analysis-requests/{requestId}/proposal", controller.submitAgentProposal)
	router.Post("/responsibility-links", controller.createLink)
	router.Get("/responsibility-links/{responsibilityLinkId}", controller.getLink)
	router.Post("/responsibility-links/{responsibilityLinkId}/end", controller.endLink)
}

func (controller *IntakesController) capture(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/projects/{id}/intakes")
		return
	}
	var request CreateIntakeRequest
	if !decodeIntakeJSON(w, r, &request) {
		return
	}
	refs := make([]domain.IntakeConversationRef, 0, len(request.ConversationRefs))
	for _, ref := range request.ConversationRefs {
		refs = append(refs, domain.IntakeConversationRef{EpisodeID: ref.EpisodeID, TurnID: ref.TurnID, Position: ref.Position})
	}
	snapshot, err := controller.Svc.Capture(r.Context(), intakevc.CaptureInput{SourceSurface: domain.IntakeSourceSurface(request.SourceSurface), Purpose: domain.IntakePurposeOutcome, ProjectID: domain.ProjectID(chi.URLParam(r, "id")), SourceOpenLoopID: domain.OpenLoopID(request.SourceOpenLoopID), Statement: request.Statement, ConversationRefs: refs, RequestKey: request.RequestKey})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusCreated, IntakeEnvelope{Intake: intakeSnapshotResponse(snapshot)})
}

func (controller *IntakesController) get(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodGet, "/api/v1/intakes/{intakeId}")
		return
	}
	snapshot, err := controller.Svc.Get(r.Context(), domain.IntakeSessionID(chi.URLParam(r, "intakeId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, IntakeEnvelope{Intake: intakeSnapshotResponse(snapshot)})
}

func (controller *IntakesController) analyze(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/intakes/{intakeId}/analysis")
		return
	}
	var request AnalyzeIntakeRequest
	if !decodeIntakeJSON(w, r, &request) {
		return
	}
	snapshot, err := controller.Svc.Analyze(r.Context(), domain.IntakeSessionID(chi.URLParam(r, "intakeId")), intakevc.AnalyzeInput{ExpectedProposalRevision: request.ExpectedProposalRevision, Offline: request.Offline})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, IntakeEnvelope{Intake: intakeSnapshotResponse(snapshot)})
}

func (controller *IntakesController) answer(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/intakes/{intakeId}/clarification")
		return
	}
	var request AnswerIntakeClarificationRequest
	if !decodeIntakeJSON(w, r, &request) {
		return
	}
	snapshot, err := controller.Svc.AnswerClarification(r.Context(), domain.IntakeSessionID(chi.URLParam(r, "intakeId")), intakevc.AnswerClarificationInput{ExpectedProposalRevision: request.ExpectedProposalRevision, Answer: request.Answer})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, IntakeEnvelope{Intake: intakeSnapshotResponse(snapshot)})
}

func (controller *IntakesController) revise(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/intakes/{intakeId}/proposals")
		return
	}
	var request ReviseIntakeProposalRequest
	if !decodeIntakeJSON(w, r, &request) {
		return
	}
	snapshot, err := controller.Svc.ReviseProposal(r.Context(), domain.IntakeSessionID(chi.URLParam(r, "intakeId")), intakevc.ReviseProposalInput{ExpectedProposalRevision: request.ExpectedProposalRevision, Proposal: intakeProposalFromInput(request.Proposal)})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, IntakeEnvelope{Intake: intakeSnapshotResponse(snapshot)})
}

func (controller *IntakesController) confirm(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/intakes/{intakeId}/confirmation")
		return
	}
	var request ConfirmIntakeRequest
	if !decodeIntakeJSON(w, r, &request) {
		return
	}
	snapshot, err := controller.Svc.ConfirmOutcome(r.Context(), domain.IntakeSessionID(chi.URLParam(r, "intakeId")), intakevc.ConfirmOutcomeInput{ExpectedProposalRevision: request.ExpectedProposalRevision, RequestKey: request.RequestKey})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusCreated, IntakeEnvelope{Intake: intakeSnapshotResponse(snapshot)})
}

func (controller *IntakesController) cancel(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/intakes/{intakeId}/cancellation")
		return
	}
	var request CancelIntakeRequest
	if !decodeIntakeJSON(w, r, &request) {
		return
	}
	snapshot, err := controller.Svc.Cancel(r.Context(), domain.IntakeSessionID(chi.URLParam(r, "intakeId")), intakevc.CancelInput{ExpectedProposalRevision: request.ExpectedProposalRevision, Reason: request.Reason})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, IntakeEnvelope{Intake: intakeSnapshotResponse(snapshot)})
}

func (controller *IntakesController) createLink(w http.ResponseWriter, r *http.Request) {
	if controller.Links == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/responsibility-links")
		return
	}
	var request CreateResponsibilityLinkRequest
	if !decodeIntakeJSON(w, r, &request) {
		return
	}
	link, err := controller.Links.CreateResponsibilityLink(r.Context(), intakevc.CreateResponsibilityLinkInput{ProjectID: domain.ProjectID(request.ProjectID), SourceOpenLoopID: domain.OpenLoopID(request.SourceOpenLoopID), DestinationOutcomeID: domain.OutcomeID(request.DestinationOutcomeID), Reason: request.Reason, RequestKey: request.RequestKey})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusCreated, ResponsibilityLinkEnvelope{ResponsibilityLink: responsibilityLinkResponse(link)})
}

func (controller *IntakesController) getLink(w http.ResponseWriter, r *http.Request) {
	if controller.Links == nil {
		apispec.NotImplemented(w, r, http.MethodGet, "/api/v1/responsibility-links/{responsibilityLinkId}")
		return
	}
	link, err := controller.Links.GetResponsibilityLink(r.Context(), domain.ResponsibilityLinkID(chi.URLParam(r, "responsibilityLinkId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, ResponsibilityLinkEnvelope{ResponsibilityLink: responsibilityLinkResponse(link)})
}

func (controller *IntakesController) endLink(w http.ResponseWriter, r *http.Request) {
	if controller.Links == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/responsibility-links/{responsibilityLinkId}/end")
		return
	}
	var request EndResponsibilityLinkRequest
	if !decodeIntakeJSON(w, r, &request) {
		return
	}
	link, err := controller.Links.EndResponsibilityLink(r.Context(), domain.ResponsibilityLinkID(chi.URLParam(r, "responsibilityLinkId")), request.Reason)
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, ResponsibilityLinkEnvelope{ResponsibilityLink: responsibilityLinkResponse(link)})
}

func decodeIntakeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid JSON body", nil)
		return false
	}
	return true
}

func intakeProposalFromInput(input IntakeProposalInput) domain.OutcomeContractProposal {
	criteria := make([]domain.ProposedCriterion, 0, len(input.Criteria))
	for _, criterion := range input.Criteria {
		criteria = append(criteria, domain.ProposedCriterion{ID: domain.ProposedCriterionID(criterion.ID), Text: criterion.Text, EvidenceExpected: criterion.EvidenceExpected})
	}
	facets := make([]domain.ContractFacet, 0, len(input.Facets))
	for _, facet := range input.Facets {
		facets = append(facets, domain.ContractFacet{Kind: domain.ContractFacetKind(facet.Kind), Summary: facet.Summary, Requirements: facet.Requirements})
	}
	return domain.OutcomeContractProposal{Title: input.Title, DesiredState: input.DesiredState, Criteria: criteria, ReviewMethod: input.ReviewMethod, Constraints: input.Constraints, NonGoals: input.NonGoals, AuthorityCeiling: domain.ProposedAuthority{ReadWorkspace: input.AuthorityCeiling.ReadWorkspace, WriteWorkspace: input.AuthorityCeiling.WriteWorkspace, ExecuteLocal: input.AuthorityCeiling.ExecuteLocal, UseNetwork: input.AuthorityCeiling.UseNetwork, CommitLocal: input.AuthorityCeiling.CommitLocal, CreatePR: input.AuthorityCeiling.CreatePR, Deploy: input.AuthorityCeiling.Deploy, ExternalEffect: input.AuthorityCeiling.ExternalEffect}, StopConditions: input.StopConditions, ClarificationNotes: input.ClarificationNotes, TemporalCondition: input.TemporalCondition, Facets: facets}
}

func intakeSnapshotResponse(snapshot ports.IntakeSnapshot) IntakeSnapshotResponse {
	response := IntakeSnapshotResponse{Session: IntakeSessionResponse{ID: snapshot.Session.ID.String(), SourceSurface: string(snapshot.Session.SourceSurface), Purpose: string(snapshot.Session.Purpose), ProjectID: string(snapshot.Session.ProjectID), SourceOpenLoopID: snapshot.Session.SourceOpenLoopID.String(), Statement: snapshot.Session.Statement, Status: string(snapshot.Session.Status), CurrentProposalRevision: snapshot.Session.CurrentProposalRevision, ClarificationCount: snapshot.Session.ClarificationCount, ConfirmedOutcomeID: snapshot.Session.ConfirmedOutcomeID.String(), FailureCode: snapshot.Session.FailureCode, CancellationReason: snapshot.Session.CancellationReason, CreatedAt: snapshot.Session.CreatedAt, UpdatedAt: snapshot.Session.UpdatedAt}}
	for _, ref := range snapshot.ConversationRefs {
		response.ConversationRefs = append(response.ConversationRefs, IntakeConversationRefResponse{EpisodeID: ref.EpisodeID, TurnID: ref.TurnID, Position: ref.Position})
	}
	if snapshot.Proposal != nil {
		p := snapshot.Proposal
		proposal := IntakeProposalResponse{ID: string(p.ID), Revision: p.Revision, Title: p.Title, DesiredState: p.DesiredState, ReviewMethod: p.ReviewMethod, Constraints: p.Constraints, NonGoals: p.NonGoals, AuthorityCeiling: intakeAuthority(p.AuthorityCeiling), StopConditions: p.StopConditions, ClarificationNotes: p.ClarificationNotes, TemporalCondition: p.TemporalCondition, Facets: intakeFacets(p.Facets), CreatedAt: p.CreatedAt}
		for _, criterion := range p.Criteria {
			proposal.Criteria = append(proposal.Criteria, IntakeCriterionResponse{ID: string(criterion.ID), Text: criterion.Text, EvidenceExpected: criterion.EvidenceExpected})
		}
		response.Proposal = &proposal
	}
	if c := snapshot.Clarification; c != nil {
		response.Clarification = &IntakeClarificationResponse{ID: string(c.ID), Question: c.Question, Reason: c.Reason, Recommendation: c.Recommendation, Alternatives: c.Alternatives, DeferralConsequence: c.DeferralConsequence, Answer: c.Answer, AnsweredAt: c.AnsweredAt}
	}
	if o := snapshot.ConfirmedOutcome; o != nil {
		response.ConfirmedOutcome = &IntakeOutcomeResponse{ID: o.ID.String(), SpaceID: o.SpaceID.String(), Title: o.Title, CurrentRevisionNumber: o.CurrentRevisionNumber, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}
	}
	if c := snapshot.ConfirmedContract; c != nil {
		value := contractRevisionResponse(*c)
		response.ConfirmedContract = &value
	}
	return response
}
func responsibilityLinkResponse(link domain.ResponsibilityLink) ResponsibilityLinkResponse {
	return ResponsibilityLinkResponse{ID: link.ID.String(), SourceOpenLoopID: link.SourceOpenLoopID.String(), DestinationOutcomeID: link.DestinationOutcomeID.String(), Creator: string(link.Creator), Reason: link.Reason, CreatedAt: link.CreatedAt, EndedAt: link.EndedAt, EndedBy: string(link.EndedBy), EndedReason: link.EndedReason}
}

// IntakeCallbackTokenHeader carries the scoping token a spawned agent was
// handed. It is NOT authentication: the loopback listener is unauthenticated
// by deliberate decision, so the token scopes an answer to one request rather
// than proving who is answering.
const IntakeCallbackTokenHeader = "X-Kennel-Intake-Token" //nolint:gosec // header NAME, not a credential; the token itself is minted per request and never constant.

// maxAgentIntakeProposalBytes bounds what a spawned agent may post back. A
// runaway generation should be refused at the door rather than parsed.
const maxAgentIntakeProposalBytes = 1 << 20

// submitAgentProposal is the callback an agent-authored Contract proposal
// arrives on. It parses the SAME proposal shape a hand-authored revision uses,
// so an agent gets no special vocabulary and no special validation.
func (controller *IntakesController) submitAgentProposal(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/intake-analysis-requests/{requestId}/proposal")
		return
	}
	var request SubmitIntakeAnalysisRequest
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxAgentIntakeProposalBytes))
	if err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Could not read the proposal body", nil)
		return
	}
	if err := json.Unmarshal(raw, &request); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	result := ports.IntakeAnalysisResult{}
	if request.Proposal != nil {
		proposal := intakeProposalFromInput(*request.Proposal)
		result.Proposal = &proposal
	}
	if request.Clarification != nil {
		result.Clarification = &domain.ClarificationRequest{
			Question: request.Clarification.Question, Reason: request.Clarification.Reason,
			Recommendation: request.Clarification.Recommendation, Alternatives: request.Clarification.Alternatives,
			DeferralConsequence: request.Clarification.DeferralConsequence,
		}
	}
	snapshot, err := controller.Svc.SubmitAgentProposal(
		r.Context(),
		domain.IntakeAnalysisRequestID(chi.URLParam(r, "requestId")),
		r.Header.Get(IntakeCallbackTokenHeader),
		result,
		// The agent's own bytes are retained verbatim, so a refused draft is
		// exactly what it sent rather than a re-render of it.
		string(raw),
	)
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, IntakeEnvelope{Intake: intakeSnapshotResponse(snapshot)})
}

func (controller *IntakesController) latestAnalysisRequest(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodGet, "/api/v1/intakes/{intakeId}/analysis-request")
		return
	}
	request, found, err := controller.Svc.LatestAnalysisRequest(r.Context(), domain.IntakeSessionID(chi.URLParam(r, "intakeId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	if !found {
		envelope.WriteAPIError(w, r, http.StatusNotFound, "not_found", "INTAKE_ANALYSIS_REQUEST_NOT_FOUND", "No agent has been asked about this intake", nil)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, IntakeAnalysisRequestEnvelope{Request: intakeAnalysisRequestResponse(request, false)})
}

func (controller *IntakesController) cancelAnalysisRequest(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/intakes/{intakeId}/analysis-request/cancellation")
		return
	}
	snapshot, err := controller.Svc.CancelAnalysisRequest(r.Context(), domain.IntakeSessionID(chi.URLParam(r, "intakeId")))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, IntakeEnvelope{Intake: intakeSnapshotResponse(snapshot)})
}

func intakeAnalysisRequestResponse(request domain.IntakeAnalysisRequest, expired bool) IntakeAnalysisRequestResponse {
	return IntakeAnalysisRequestResponse{
		ID:                       request.ID.String(),
		IntakeID:                 request.IntakeID.String(),
		ExpectedProposalRevision: request.ExpectedProposalRevision,
		Status:                   string(request.Status),
		SessionID:                request.SessionID,
		Harness:                  string(request.Harness),
		ExpiresAt:                request.ExpiresAt,
		Expired:                  expired,
		RawProposal:              request.RawProposal,
		RefusalReason:            request.RefusalReason,
		CreatedAt:                request.CreatedAt,
		AnsweredAt:               request.AnsweredAt,
	}
}

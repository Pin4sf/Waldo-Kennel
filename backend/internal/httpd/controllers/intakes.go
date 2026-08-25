package controllers

import (
	"context"
	"encoding/json"
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
	snapshot, err := controller.Svc.Analyze(r.Context(), domain.IntakeSessionID(chi.URLParam(r, "intakeId")), intakevc.AnalyzeInput{ExpectedProposalRevision: request.ExpectedProposalRevision})
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
	response := IntakeSnapshotResponse{Session: IntakeSessionResponse{ID: snapshot.Session.ID.String(), SourceSurface: string(snapshot.Session.SourceSurface), Purpose: string(snapshot.Session.Purpose), ProjectID: string(snapshot.Session.ProjectID), SourceOpenLoopID: snapshot.Session.SourceOpenLoopID.String(), Statement: snapshot.Session.Statement, Status: string(snapshot.Session.Status), CurrentProposalRevision: snapshot.Session.CurrentProposalRevision, ClarificationCount: snapshot.Session.ClarificationCount, ConfirmedOutcomeID: snapshot.Session.ConfirmedOutcomeID.String(), FailureCode: snapshot.Session.FailureCode, CreatedAt: snapshot.Session.CreatedAt, UpdatedAt: snapshot.Session.UpdatedAt}}
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

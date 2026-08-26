package controllers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apispec"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/envelope"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	waldovc "github.com/aoagents/agent-orchestrator/backend/internal/service/waldoconversation"
)

type waldoConversationReader interface {
	Open(context.Context, domain.ProjectID) (ports.WaldoConversationSnapshot, error)
	Get(context.Context, domain.ProjectID) (ports.WaldoConversationSnapshot, error)
}

type waldoConversationWriter interface {
	OpenEpisode(context.Context, domain.ProjectID, waldovc.OpenEpisodeInput) (ports.WaldoConversationSnapshot, error)
	AppendTurn(context.Context, domain.ProjectID, waldovc.AppendTurnInput) (waldovc.AppendTurnResult, error)
	AttachContext(context.Context, domain.ProjectID, waldovc.AttachContextInput) (waldovc.ContextMutationResult, error)
	DetachContext(context.Context, domain.ProjectID, waldovc.DetachContextInput) (waldovc.ContextMutationResult, error)
	Continue(context.Context, domain.ProjectID, waldovc.ContinuationInput) (domain.ContinuationReceipt, error)
}

// WaldoConversationService is the Project-scoped durable conversation boundary.
type WaldoConversationService interface {
	waldoConversationReader
	waldoConversationWriter
}

// WaldoConversationsController keeps HTTP translation outside domain policy.
type WaldoConversationsController struct{ Svc WaldoConversationService }

// Register mounts the Project Waldo conversation routes.
func (controller *WaldoConversationsController) Register(router chi.Router) {
	router.Post("/projects/{id}/waldo-conversation", controller.open)
	router.Get("/projects/{id}/waldo-conversation", controller.get)
	router.Post("/projects/{id}/waldo-conversation/episodes", controller.openEpisode)
	router.Post("/projects/{id}/waldo-conversation/turns", controller.appendTurn)
	router.Post("/projects/{id}/waldo-conversation/context", controller.attachContext)
	router.Post("/projects/{id}/waldo-conversation/context/{attachmentId}/detach", controller.detachContext)
	router.Post("/projects/{id}/waldo-conversation/continuations", controller.continueConversation)
}

func (controller *WaldoConversationsController) open(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/projects/{id}/waldo-conversation")
		return
	}
	snapshot, err := controller.Svc.Open(r.Context(), projectID(r))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusCreated, waldoConversationEnvelope(snapshot))
}

func (controller *WaldoConversationsController) get(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodGet, "/api/v1/projects/{id}/waldo-conversation")
		return
	}
	snapshot, err := controller.Svc.Get(r.Context(), projectID(r))
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, waldoConversationEnvelope(snapshot))
}

func (controller *WaldoConversationsController) openEpisode(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/projects/{id}/waldo-conversation/episodes")
		return
	}
	var request OpenWaldoEpisodeRequest
	if !decodeWaldoJSON(w, r, &request) {
		return
	}
	var providerRef *domain.WaldoProviderEpisodeRef
	if request.ProviderRef != nil {
		providerRef = &domain.WaldoProviderEpisodeRef{
			Provider:               domain.AgentHarness(request.ProviderRef.Provider),
			ProviderConversationID: request.ProviderRef.ProviderConversationID,
			TranscriptRef:          request.ProviderRef.TranscriptRef,
		}
	}
	snapshot, err := controller.Svc.OpenEpisode(r.Context(), projectID(r), waldovc.OpenEpisodeInput{
		ExpectedRevision: request.ExpectedRevision, ProviderRef: providerRef, RequestKey: request.RequestKey,
	})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusCreated, waldoConversationEnvelope(snapshot))
}

func (controller *WaldoConversationsController) appendTurn(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/projects/{id}/waldo-conversation/turns")
		return
	}
	var request AppendWaldoTurnRequest
	if !decodeWaldoJSON(w, r, &request) {
		return
	}
	var providerRef *domain.WaldoProviderTurnRef
	if request.ProviderRef != nil {
		providerRef = &domain.WaldoProviderTurnRef{
			Provider: domain.AgentHarness(request.ProviderRef.Provider), ProviderConversationID: request.ProviderRef.ProviderConversationID,
			ProviderTurnID: request.ProviderRef.ProviderTurnID, TranscriptRef: request.ProviderRef.TranscriptRef,
		}
	}
	attachmentIDs := make([]domain.WaldoContextAttachmentID, 0, len(request.ContextAttachmentIDs))
	for _, id := range request.ContextAttachmentIDs {
		attachmentIDs = append(attachmentIDs, domain.WaldoContextAttachmentID(id))
	}
	result, err := controller.Svc.AppendTurn(r.Context(), projectID(r), waldovc.AppendTurnInput{
		ExpectedRevision: request.ExpectedRevision, EpisodeID: domain.WaldoConversationEpisodeID(request.EpisodeID),
		Role: domain.WaldoTurnRole(request.Role), Message: request.Message, ProviderRef: providerRef,
		ContextAttachmentIDs: attachmentIDs, RequestKey: request.RequestKey,
	})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	envelope.WriteJSON(w, status, WaldoTurnEnvelope{
		Turn: waldoTurnResponse(result.Turn), WaldoConversation: waldoConversationSnapshotResponse(result.Snapshot),
	})
}

func (controller *WaldoConversationsController) attachContext(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/projects/{id}/waldo-conversation/context")
		return
	}
	var request AttachWaldoContextRequest
	if !decodeWaldoJSON(w, r, &request) {
		return
	}
	result, err := controller.Svc.AttachContext(r.Context(), projectID(r), waldovc.AttachContextInput{
		ExpectedRevision: request.ExpectedRevision, Ref: waldoContextRef(request.Ref), RequestKey: request.RequestKey,
	})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusCreated, waldoConversationEnvelope(result.Snapshot))
}

func (controller *WaldoConversationsController) detachContext(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/projects/{id}/waldo-conversation/context/{attachmentId}/detach")
		return
	}
	var request DetachWaldoContextRequest
	if !decodeWaldoJSON(w, r, &request) {
		return
	}
	result, err := controller.Svc.DetachContext(r.Context(), projectID(r), waldovc.DetachContextInput{
		ExpectedRevision: request.ExpectedRevision, AttachmentID: domain.WaldoContextAttachmentID(chi.URLParam(r, "attachmentId")),
		Reason: request.Reason, RequestKey: request.RequestKey,
	})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, waldoConversationEnvelope(result.Snapshot))
}

func (controller *WaldoConversationsController) continueConversation(w http.ResponseWriter, r *http.Request) {
	if controller.Svc == nil {
		apispec.NotImplemented(w, r, http.MethodPost, "/api/v1/projects/{id}/waldo-conversation/continuations")
		return
	}
	var request ContinueWaldoConversationRequest
	if !decodeWaldoJSON(w, r, &request) {
		return
	}
	refs := make([]domain.WaldoContextRef, 0, len(request.ContextRefs))
	for _, ref := range request.ContextRefs {
		refs = append(refs, waldoContextRef(ref))
	}
	receipt, err := controller.Svc.Continue(r.Context(), projectID(r), waldovc.ContinuationInput{
		FromAgentSessionRef: domain.AttemptSessionRefID(request.FromAgentSessionRef), Reason: domain.ContinuationReason(request.Reason),
		ReasonDetail:    request.ReasonDetail,
		TriggerEvidence: domain.ContinuationTriggerEvidence{Kind: domain.ContinuationTriggerEvidenceKind(request.TriggerEvidence.Kind), Reference: request.TriggerEvidence.Reference},
		ContextDigest:   request.ContextDigest, ContextRefs: refs,
		PreviousBindings: waldoContinuationBindings(request.PreviousBindings), ReplacementBindings: waldoContinuationBindings(request.ReplacementBindings),
		EffectsKnown: request.EffectsKnown, LostMaterialContext: request.LostMaterialContext,
		SourceRevoked: request.SourceRevoked, FreshVerifier: request.FreshVerifier, RequestKey: request.RequestKey,
	})
	if err != nil {
		envelope.WriteError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusCreated, WaldoContinuationEnvelope{ContinuationReceipt: waldoContinuationReceiptResponse(receipt)})
}

func decodeWaldoJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := decodeJSONStrict(r, target); err != nil {
		envelope.WriteAPIError(w, r, http.StatusBadRequest, "bad_request", "INVALID_JSON", "Invalid JSON body", nil)
		return false
	}
	return true
}

func waldoConversationEnvelope(snapshot ports.WaldoConversationSnapshot) WaldoConversationEnvelope {
	return WaldoConversationEnvelope{WaldoConversation: waldoConversationSnapshotResponse(snapshot)}
}

func waldoConversationSnapshotResponse(snapshot ports.WaldoConversationSnapshot) WaldoConversationSnapshotResponse {
	response := WaldoConversationSnapshotResponse{
		Conversation: WaldoConversationResponse{
			ID: snapshot.Conversation.ID.String(), ProjectID: string(snapshot.Conversation.ProjectID), Revision: snapshot.Conversation.Revision,
			LatestTurnSequence: snapshot.Conversation.LatestTurnSequence, CreatedAt: snapshot.Conversation.CreatedAt, UpdatedAt: snapshot.Conversation.UpdatedAt,
		},
		Episodes:             make([]WaldoConversationEpisodeResponse, 0, len(snapshot.Episodes)),
		Turns:                make([]WaldoConversationTurnResponse, 0, len(snapshot.Turns)),
		ContextAttachments:   make([]WaldoContextAttachmentResponse, 0, len(snapshot.ContextAttachments)),
		ContinuationReceipts: make([]WaldoContinuationReceiptResponse, 0, len(snapshot.ContinuationReceipts)),
	}
	for _, episode := range snapshot.Episodes {
		item := WaldoConversationEpisodeResponse{
			ID: episode.ID.String(), ConversationID: episode.ConversationID.String(), ProjectID: string(episode.ProjectID), Ordinal: episode.Ordinal,
			State: string(episode.State), CreatedAt: episode.CreatedAt, SealedAt: episode.SealedAt, SealReason: episode.SealReason,
		}
		if episode.ProviderRef != nil {
			item.ProviderRef = &WaldoProviderEpisodeRefResponse{Provider: string(episode.ProviderRef.Provider), ProviderConversationID: episode.ProviderRef.ProviderConversationID, TranscriptRef: episode.ProviderRef.TranscriptRef}
		}
		response.Episodes = append(response.Episodes, item)
	}
	for _, turn := range snapshot.Turns {
		response.Turns = append(response.Turns, waldoTurnResponse(turn))
	}
	for _, attachment := range snapshot.ContextAttachments {
		response.ContextAttachments = append(response.ContextAttachments, WaldoContextAttachmentResponse{
			ID: attachment.ID.String(), ConversationID: attachment.ConversationID.String(), ProjectID: string(attachment.ProjectID),
			Ref: waldoContextRefResponse(attachment.Ref), AttachedRevision: attachment.AttachedRevision,
			DetachedRevision: attachment.DetachedRevision, Active: attachment.Active(), CreatedAt: attachment.CreatedAt,
			DetachedAt: attachment.DetachedAt, DetachReason: attachment.DetachReason,
		})
	}
	for _, receipt := range snapshot.ContinuationReceipts {
		response.ContinuationReceipts = append(response.ContinuationReceipts, waldoContinuationReceiptResponse(receipt))
	}
	return response
}

func waldoTurnResponse(turn domain.WaldoConversationTurn) WaldoConversationTurnResponse {
	response := WaldoConversationTurnResponse{
		ID: turn.ID.String(), ConversationID: turn.ConversationID.String(), EpisodeID: turn.EpisodeID.String(), ProjectID: string(turn.ProjectID),
		Sequence: turn.Sequence, Role: string(turn.Role), Message: turn.Message, CreatedAt: turn.CreatedAt,
		ContextRefs: make([]WaldoContextRefResponse, 0, len(turn.ContextRefs)),
	}
	if turn.ProviderRef != nil {
		response.ProviderRef = &WaldoProviderTurnRefResponse{Provider: string(turn.ProviderRef.Provider), ProviderConversationID: turn.ProviderRef.ProviderConversationID, ProviderTurnID: turn.ProviderRef.ProviderTurnID, TranscriptRef: turn.ProviderRef.TranscriptRef}
	}
	for _, ref := range turn.ContextRefs {
		response.ContextRefs = append(response.ContextRefs, waldoContextRefResponse(ref))
	}
	return response
}

func waldoContextRef(response WaldoContextRefResponse) domain.WaldoContextRef {
	return domain.WaldoContextRef{Kind: domain.WaldoContextRefKind(response.Kind), ObjectID: response.ObjectID, Revision: response.Revision,
		Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoContextProvenanceKind(response.Provenance.Kind), SourceID: response.Provenance.SourceID}}
}

func waldoContextRefResponse(ref domain.WaldoContextRef) WaldoContextRefResponse {
	return WaldoContextRefResponse{Kind: string(ref.Kind), ObjectID: ref.ObjectID, Revision: ref.Revision,
		Provenance: WaldoContextProvenanceResponse{Kind: string(ref.Provenance.Kind), SourceID: ref.Provenance.SourceID}}
}

func waldoContinuationBindings(response WaldoContinuationBindingsResponse) domain.ContinuationBindings {
	return domain.ContinuationBindings{
		ProjectID: domain.ProjectID(response.ProjectID), OutcomeID: domain.OutcomeID(response.OutcomeID),
		ContractRevisionID: domain.ContractRevisionID(response.ContractRevisionID), PlanRevisionID: domain.PlanRevisionID(response.PlanRevisionID),
		WorkUnitID: domain.WorkUnitID(response.WorkUnitID), AttemptID: domain.AttemptID(response.AttemptID), Provider: domain.AgentHarness(response.Provider),
		Model: response.Model, Profile: response.Profile, Role: response.Role, AuthorityDigest: response.AuthorityDigest,
		BudgetDigest: response.BudgetDigest, WorkspaceOwner: response.WorkspaceOwner, EffectPolicyDigest: response.EffectPolicyDigest,
	}
}

func waldoContinuationBindingsResponse(bindings domain.ContinuationBindings) WaldoContinuationBindingsResponse {
	return WaldoContinuationBindingsResponse{
		ProjectID: string(bindings.ProjectID), OutcomeID: bindings.OutcomeID.String(), ContractRevisionID: bindings.ContractRevisionID.String(),
		PlanRevisionID: bindings.PlanRevisionID.String(), WorkUnitID: bindings.WorkUnitID.String(), AttemptID: bindings.AttemptID.String(),
		Provider: string(bindings.Provider), Model: bindings.Model, Profile: bindings.Profile, Role: bindings.Role,
		AuthorityDigest: bindings.AuthorityDigest, BudgetDigest: bindings.BudgetDigest, WorkspaceOwner: bindings.WorkspaceOwner,
		EffectPolicyDigest: bindings.EffectPolicyDigest,
	}
}

func waldoContinuationReceiptResponse(receipt domain.ContinuationReceipt) WaldoContinuationReceiptResponse {
	response := WaldoContinuationReceiptResponse{
		ID: receipt.ID, OperationID: receipt.OperationID, ConversationID: receipt.ConversationID.String(), ProjectID: string(receipt.ProjectID),
		FromEpisodeID: receipt.FromEpisodeID.String(), ToEpisodeID: receipt.ToEpisodeID.String(), FromAgentSessionRef: receipt.FromAgentSessionRef.String(),
		ToAgentSessionRef: receipt.ToAgentSessionRef.String(), Action: string(receipt.Action), Reason: string(receipt.Reason), ReasonDetail: receipt.ReasonDetail,
		TriggerEvidence: WaldoContinuationEvidenceResponse{Kind: string(receipt.TriggerEvidence.Kind), Reference: receipt.TriggerEvidence.Reference},
		MaterialChange:  receipt.MaterialChange, ChangedFields: receipt.ChangedFields, ContextDigest: receipt.ContextDigest,
		ContextRefs:      make([]WaldoContextRefResponse, 0, len(receipt.ContextRefs)),
		PreviousBindings: waldoContinuationBindingsResponse(receipt.PreviousBindings), ReplacementBindings: waldoContinuationBindingsResponse(receipt.ReplacementBindings),
		EffectsKnown: receipt.EffectsKnown, OldSessionFenced: receipt.OldSessionFenced,
		ReplacementIdentityConfirmed: receipt.ReplacementIdentityConfirmed, FenceReceiptRef: receipt.FenceReceiptRef,
		ReconciliationRef: receipt.ReconciliationRef, NeedsUserReason: receipt.NeedsUserReason, CreatedAt: receipt.CreatedAt,
	}
	for _, ref := range receipt.ContextRefs {
		response.ContextRefs = append(response.ContextRefs, waldoContextRefResponse(ref))
	}
	return response
}

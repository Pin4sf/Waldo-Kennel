// Package waldoconversation owns durable Project-scoped Waldo conversation
// policy. Provider/runtime adapters remain leaves behind the continuation port.
package waldoconversation

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// Stable daemon error codes for Project-Waldo conversation policy.
const (
	CodeConversationNotFound     = "WALDO_CONVERSATION_NOT_FOUND"
	CodeConversationRevision     = "WALDO_CONVERSATION_REVISION_CONFLICT"
	CodeConversationIdempotency  = "WALDO_CONVERSATION_IDEMPOTENCY_CONFLICT"
	CodeContextRevision          = "WALDO_CONTEXT_REVISION_CONFLICT"
	CodeContextNotFound          = "WALDO_CONTEXT_NOT_FOUND"
	CodeContextNotActive         = "WALDO_CONTEXT_NOT_ACTIVE"
	CodeContinuationUnwired      = "WALDO_CONTINUATION_UNWIRED"
	CodeContinuationInputInvalid = "WALDO_CONTINUATION_INVALID"
)

// Service owns Project binding, context precedence, revision safety, and
// continuation policy. The store owns transactional persistence mechanics.
type Service struct {
	store    ports.WaldoConversationStore
	executor ports.WaldoContinuationExecutor
	clock    func() time.Time
}

// New constructs the Project conversation service.
func New(store ports.WaldoConversationStore, executor ports.WaldoContinuationExecutor, clock func() time.Time) *Service {
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, executor: executor, clock: clock}
}

// Open returns or creates the one durable conversation for a Project.
func (service *Service) Open(ctx context.Context, projectID domain.ProjectID) (ports.WaldoConversationSnapshot, error) {
	if strings.TrimSpace(string(projectID)) == "" {
		return ports.WaldoConversationSnapshot{}, apierr.Invalid("PROJECT_REQUIRED", "Choose a Project for this Waldo conversation", nil)
	}
	if _, found, err := service.store.GetProject(ctx, string(projectID)); err != nil {
		return ports.WaldoConversationSnapshot{}, err
	} else if !found {
		return ports.WaldoConversationSnapshot{}, apierr.NotFound("PROJECT_NOT_FOUND", "That Project does not exist")
	}
	if existing, found, err := service.store.GetWaldoConversationByProject(ctx, projectID); err != nil {
		return ports.WaldoConversationSnapshot{}, err
	} else if found {
		return existing, nil
	}
	now := service.clock()
	conversation := domain.WaldoConversation{
		ID:        "waldo-conversation-" + domain.WaldoConversationID(uuid.NewString()),
		ProjectID: projectID, CreatedAt: now, UpdatedAt: now,
	}
	return service.store.EnsureWaldoConversation(ctx, conversation)
}

// Get returns the complete restart-safe Project conversation.
func (service *Service) Get(ctx context.Context, projectID domain.ProjectID) (ports.WaldoConversationSnapshot, error) {
	snapshot, found, err := service.store.GetWaldoConversationByProject(ctx, projectID)
	if err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	if !found {
		return ports.WaldoConversationSnapshot{}, apierr.NotFound(CodeConversationNotFound, "This Project does not have a Waldo conversation yet")
	}
	return snapshot, nil
}

// OpenEpisodeInput opens one bounded episode. A provider ref is optional for a
// deterministic/manual episode and identifier-only when present.
type OpenEpisodeInput struct {
	ExpectedRevision int64
	ProviderRef      *domain.WaldoProviderEpisodeRef
	RequestKey       string
}

// OpenEpisode adds a bounded episode without replacing the Project relationship.
func (service *Service) OpenEpisode(ctx context.Context, projectID domain.ProjectID, input OpenEpisodeInput) (ports.WaldoConversationSnapshot, error) {
	snapshot, err := service.Get(ctx, projectID)
	if err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	input.RequestKey = strings.TrimSpace(input.RequestKey)
	if input.RequestKey == "" {
		return ports.WaldoConversationSnapshot{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this episode", nil)
	}
	episode := domain.WaldoConversationEpisode{
		ID:             "waldo-episode-" + domain.WaldoConversationEpisodeID(uuid.NewString()),
		ConversationID: snapshot.Conversation.ID, ProjectID: projectID,
		Ordinal: int64(len(snapshot.Episodes) + 1), State: domain.WaldoEpisodeActive,
		ProviderRef: input.ProviderRef, CreatedAt: service.clock(),
	}
	if err := episode.Validate(); err != nil {
		return ports.WaldoConversationSnapshot{}, apierr.Invalid("WALDO_EPISODE_INVALID", err.Error(), nil)
	}
	result, err := service.store.OpenWaldoEpisode(ctx, episode, ports.WaldoIdempotency{
		Key: input.RequestKey, Fingerprint: episodeFingerprint(projectID, input.ProviderRef),
	}, input.ExpectedRevision)
	return result, mapStoreError(err)
}

// AppendTurnInput contains one visible turn and explicit attachment selection.
type AppendTurnInput struct {
	ExpectedRevision     int64
	EpisodeID            domain.WaldoConversationEpisodeID
	Role                 domain.WaldoTurnRole
	Message              string
	ProviderRef          *domain.WaldoProviderTurnRef
	ContextAttachmentIDs []domain.WaldoContextAttachmentID
	RequestKey           string
}

// AppendTurnResult returns the stored turn and current snapshot.
type AppendTurnResult struct {
	Turn     domain.WaldoConversationTurn
	Snapshot ports.WaldoConversationSnapshot
}

// AppendTurn appends one ordered, idempotent Project-bound turn. Callers select
// durable attachments by id; they cannot smuggle unvalidated context refs.
func (service *Service) AppendTurn(ctx context.Context, projectID domain.ProjectID, input AppendTurnInput) (AppendTurnResult, error) {
	input.Message = strings.TrimSpace(input.Message)
	input.RequestKey = strings.TrimSpace(input.RequestKey)
	if input.RequestKey == "" {
		return AppendTurnResult{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this turn", nil)
	}
	if input.Message == "" {
		return AppendTurnResult{}, apierr.Invalid("WALDO_TURN_MESSAGE_REQUIRED", "Write a message before sending", nil)
	}
	fingerprint := turnFingerprint(projectID, input)
	if replay, storedFingerprint, found, err := service.store.FindWaldoTurnByRequestKey(ctx, input.RequestKey); err != nil {
		return AppendTurnResult{}, err
	} else if found {
		if storedFingerprint != fingerprint {
			return AppendTurnResult{}, apierr.Conflict(CodeConversationIdempotency, "That idempotency key belongs to different conversation input", map[string]any{"requestKey": input.RequestKey})
		}
		snapshot, err := service.Get(ctx, projectID)
		if err != nil {
			return AppendTurnResult{}, err
		}
		return AppendTurnResult{Turn: replay, Snapshot: snapshot}, nil
	}
	snapshot, err := service.Get(ctx, projectID)
	if err != nil {
		return AppendTurnResult{}, err
	}
	if !episodeActive(snapshot.Episodes, input.EpisodeID) {
		return AppendTurnResult{}, apierr.Conflict("WALDO_EPISODE_NOT_ACTIVE", "Choose an active conversation episode before sending", nil)
	}
	for _, attachmentID := range input.ContextAttachmentIDs {
		attachment, found := findContextAttachment(snapshot.ContextAttachments, attachmentID)
		if !found || !attachment.Active() {
			return AppendTurnResult{}, apierr.Conflict(CodeContextNotActive, "A selected context attachment is no longer active; reload before sending", map[string]any{"attachmentId": attachmentID})
		}
		current, found, err := service.store.ResolveWaldoContextRef(ctx, projectID, attachment.Ref)
		if err != nil {
			return AppendTurnResult{}, err
		}
		if !found {
			return AppendTurnResult{}, apierr.NotFound(CodeContextNotFound, "A selected context source is no longer available")
		}
		if current.Revision != attachment.Ref.Revision {
			return AppendTurnResult{}, contextRevisionConflict(attachment.Ref, current.Revision)
		}
	}
	turn := domain.WaldoConversationTurn{
		ID:             "waldo-turn-" + domain.WaldoConversationTurnID(uuid.NewString()),
		ConversationID: snapshot.Conversation.ID, EpisodeID: input.EpisodeID,
		ProjectID: projectID, Sequence: snapshot.Conversation.LatestTurnSequence + 1,
		Role: input.Role, Message: input.Message, ProviderRef: input.ProviderRef,
		CreatedAt: service.clock(),
	}
	updated, stored, err := service.store.AppendWaldoTurn(ctx, turn, input.ContextAttachmentIDs,
		ports.WaldoIdempotency{Key: input.RequestKey, Fingerprint: fingerprint}, input.ExpectedRevision)
	if err != nil {
		return AppendTurnResult{}, mapStoreError(err)
	}
	return AppendTurnResult{Turn: stored, Snapshot: updated}, nil
}

// AttachContextInput explicitly attaches current canonical truth.
type AttachContextInput struct {
	ExpectedRevision int64
	Ref              domain.WaldoContextRef
	RequestKey       string
}

// ContextMutationResult returns the changed snapshot.
type ContextMutationResult struct {
	Snapshot ports.WaldoConversationSnapshot
}

// AttachContext resolves canonical revision truth before persisting provenance.
func (service *Service) AttachContext(ctx context.Context, projectID domain.ProjectID, input AttachContextInput) (ContextMutationResult, error) {
	snapshot, err := service.Get(ctx, projectID)
	if err != nil {
		return ContextMutationResult{}, err
	}
	if err := input.Ref.Validate(); err != nil {
		return ContextMutationResult{}, apierr.Invalid("WALDO_CONTEXT_INVALID", err.Error(), nil)
	}
	input.RequestKey = strings.TrimSpace(input.RequestKey)
	if input.RequestKey == "" {
		return ContextMutationResult{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this context change", nil)
	}
	current, found, err := service.store.ResolveWaldoContextRef(ctx, projectID, input.Ref)
	if err != nil {
		return ContextMutationResult{}, err
	}
	if !found {
		return ContextMutationResult{}, apierr.NotFound(CodeContextNotFound, "That context object does not belong to this Project")
	}
	if input.Ref.Revision != current.Revision {
		return ContextMutationResult{}, contextRevisionConflict(input.Ref, current.Revision)
	}
	now := service.clock()
	attachment := domain.WaldoContextAttachment{
		ID:             "waldo-context-" + domain.WaldoContextAttachmentID(uuid.NewString()),
		ConversationID: snapshot.Conversation.ID, ProjectID: projectID, Ref: current,
		AttachedRevision: input.ExpectedRevision + 1, CreatedAt: now,
	}
	updated, err := service.store.AttachWaldoContext(ctx, attachment, ports.WaldoIdempotency{
		Key: input.RequestKey, Fingerprint: contextFingerprint(projectID, current, "attach", ""),
	}, input.ExpectedRevision)
	if err != nil {
		return ContextMutationResult{}, mapStoreError(err)
	}
	return ContextMutationResult{Snapshot: updated}, nil
}

// DetachContextInput consciously removes one attachment from future turns.
type DetachContextInput struct {
	ExpectedRevision int64
	AttachmentID     domain.WaldoContextAttachmentID
	Reason           string
	RequestKey       string
}

// DetachContext records an explicit, one-time detach; historical turns keep refs.
func (service *Service) DetachContext(ctx context.Context, projectID domain.ProjectID, input DetachContextInput) (ContextMutationResult, error) {
	snapshot, err := service.Get(ctx, projectID)
	if err != nil {
		return ContextMutationResult{}, err
	}
	input.Reason = strings.TrimSpace(input.Reason)
	input.RequestKey = strings.TrimSpace(input.RequestKey)
	if input.AttachmentID.IsZero() || input.Reason == "" || input.RequestKey == "" {
		return ContextMutationResult{}, apierr.Invalid("WALDO_CONTEXT_DETACH_INVALID", "Context, reason, and idempotency key are required", nil)
	}
	updated, err := service.store.DetachWaldoContext(ctx, snapshot.Conversation.ID, input.AttachmentID,
		input.Reason, ports.WaldoIdempotency{
			Key:         input.RequestKey,
			Fingerprint: fingerprint(string(projectID), input.AttachmentID.String(), input.Reason),
		}, input.ExpectedRevision, service.clock())
	if err != nil {
		return ContextMutationResult{}, mapStoreError(err)
	}
	return ContextMutationResult{Snapshot: updated}, nil
}

// ContextTier preserves the accepted context precedence below current intent
// and current canonical attachments.
type ContextTier int

// Context tiers order non-canonical sources beneath explicit current truth.
const (
	ContextTierProjectPolicy ContextTier = iota + 1
	ContextTierWorkspaceFact
	ContextTierPriorSummary
	ContextTierRetrieved
	ContextTierModelOutput
)

// ContextCandidate is an identifier-only optional source for bounded compilation.
type ContextCandidate struct {
	Tier ContextTier
	Ref  domain.WaldoContextRef
}

// CompileContextInput contains current intent plus lower-precedence candidates.
type CompileContextInput struct {
	CurrentIntent string
	Candidates    []ContextCandidate
	MaxReferences int
}

// BoundedContextPacket is the provider-neutral continuation/intake input.
type BoundedContextPacket struct {
	CurrentIntent        string
	ConversationRevision int64
	References           []domain.WaldoContextRef
	OmittedReferences    int
	Digest               string
}

// CompileContext keeps current user intent and current explicit canonical refs
// ahead of lower sources, deduplicates stale versions, and never copies transcripts.
func (service *Service) CompileContext(ctx context.Context, projectID domain.ProjectID, input CompileContextInput) (BoundedContextPacket, error) {
	snapshot, err := service.Get(ctx, projectID)
	if err != nil {
		return BoundedContextPacket{}, err
	}
	input.CurrentIntent = strings.TrimSpace(input.CurrentIntent)
	if input.CurrentIntent == "" {
		return BoundedContextPacket{}, apierr.Invalid("WALDO_CONTEXT_INTENT_REQUIRED", "Current user intent is required", nil)
	}
	if input.MaxReferences <= 0 {
		input.MaxReferences = 16
	}
	refs := make([]domain.WaldoContextRef, 0, len(snapshot.ContextAttachments)+len(input.Candidates))
	seen := map[string]struct{}{}
	for _, attachment := range snapshot.ContextAttachments {
		if !attachment.Active() {
			continue
		}
		current, found, err := service.store.ResolveWaldoContextRef(ctx, projectID, attachment.Ref)
		if err != nil {
			return BoundedContextPacket{}, err
		}
		if !found {
			return BoundedContextPacket{}, apierr.NotFound(CodeContextNotFound, "An attached context source is no longer available; detach or replace it before continuing")
		}
		if current.Revision != attachment.Ref.Revision {
			return BoundedContextPacket{}, contextRevisionConflict(attachment.Ref, current.Revision)
		}
		key := contextObjectKey(current)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		refs = append(refs, current)
	}
	sort.SliceStable(input.Candidates, func(i, j int) bool { return input.Candidates[i].Tier < input.Candidates[j].Tier })
	for _, candidate := range input.Candidates {
		if err := candidate.Ref.Validate(); err != nil {
			return BoundedContextPacket{}, apierr.Invalid("WALDO_CONTEXT_CANDIDATE_INVALID", err.Error(), nil)
		}
		key := contextObjectKey(candidate.Ref)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		refs = append(refs, candidate.Ref)
	}
	omitted := 0
	if len(refs) > input.MaxReferences {
		omitted = len(refs) - input.MaxReferences
		refs = refs[:input.MaxReferences]
	}
	packet := BoundedContextPacket{
		CurrentIntent: input.CurrentIntent, ConversationRevision: snapshot.Conversation.Revision,
		References: refs, OmittedReferences: omitted,
	}
	packet.Digest = contextPacketDigest(packet)
	return packet, nil
}

// ContinuationInput contains policy facts, never provider transcript content.
type ContinuationInput struct {
	FromAgentSessionRef domain.AttemptSessionRefID
	Reason              domain.ContinuationReason
	ReasonDetail        string
	ContextDigest       string
	ContextRefs         []domain.WaldoContextRef
	PreviousBindings    domain.ContinuationBindings
	ReplacementBindings domain.ContinuationBindings
	EffectsKnown        bool
	LostMaterialContext bool
	SourceRevoked       bool
	FreshVerifier       bool
	RequestKey          string
}

// Continue evaluates policy, contains the old executor, starts at most one
// replacement, and records every safe/unsafe outcome before returning.
func (service *Service) Continue(ctx context.Context, projectID domain.ProjectID, input ContinuationInput) (domain.ContinuationReceipt, error) {
	input.RequestKey = strings.TrimSpace(input.RequestKey)
	request := ports.WaldoIdempotency{Key: input.RequestKey, Fingerprint: continuationFingerprint(projectID, input)}
	if input.RequestKey == "" {
		return domain.ContinuationReceipt{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for continuation", nil)
	}
	if replay, storedFingerprint, found, err := service.store.FindContinuationReceiptByRequestKey(ctx, input.RequestKey); err != nil {
		return domain.ContinuationReceipt{}, err
	} else if found {
		if storedFingerprint != request.Fingerprint {
			return domain.ContinuationReceipt{}, apierr.Conflict(CodeConversationIdempotency, "That idempotency key belongs to different continuation input", map[string]any{"requestKey": input.RequestKey})
		}
		return replay, nil
	}
	snapshot, err := service.Get(ctx, projectID)
	if err != nil {
		return domain.ContinuationReceipt{}, err
	}
	if input.FromAgentSessionRef.IsZero() || strings.TrimSpace(input.ReasonDetail) == "" ||
		!input.Reason.Valid() || len(input.ContextDigest) != 64 {
		return domain.ContinuationReceipt{}, apierr.Invalid(CodeContinuationInputInvalid, "Continuation requires source identity, exact reason, and context digest", nil)
	}
	if err := input.PreviousBindings.Validate(); err != nil {
		return domain.ContinuationReceipt{}, apierr.Invalid(CodeContinuationInputInvalid, err.Error(), nil)
	}
	if err := input.ReplacementBindings.Validate(); err != nil {
		return domain.ContinuationReceipt{}, apierr.Invalid(CodeContinuationInputInvalid, err.Error(), nil)
	}
	for _, ref := range input.ContextRefs {
		if err := ref.Validate(); err != nil {
			return domain.ContinuationReceipt{}, apierr.Invalid(CodeContinuationInputInvalid, err.Error(), nil)
		}
	}
	changed := input.PreviousBindings.Changed(input.ReplacementBindings)
	if input.PreviousBindings.ProjectID != projectID || input.ReplacementBindings.ProjectID != projectID {
		changed = appendMissing(changed, "project")
	}
	if len(changed) > 0 || !input.EffectsKnown || input.LostMaterialContext || input.SourceRevoked || input.FreshVerifier {
		reason := needsUserReason(input, changed)
		return service.recordContinuation(ctx, snapshot, input, request, domain.ContinuationNeedsYou,
			true, changed, false, false, "", "", "", reason)
	}
	if service.executor == nil {
		return domain.ContinuationReceipt{}, apierr.Internal(CodeContinuationUnwired, "Provider continuation is not wired in this environment")
	}
	fence, fenceErr := service.executor.FenceForContinuation(ctx, input.FromAgentSessionRef)
	if fenceErr != nil || !fence.Fenced {
		detail := "The old session could not be safely fenced; inspect and reconcile before replacement."
		if fence.Detail != "" {
			detail = fence.Detail
		}
		return service.recordContinuation(ctx, snapshot, input, request, domain.ContinuationNeedsYou,
			false, nil, input.EffectsKnown, false, fence.FenceReceiptRef, fence.ReconciliationRef, "", detail)
	}
	started, startErr := service.executor.StartContinuation(ctx, ports.ContinuationStartRequest{
		FromAgentSessionRef: input.FromAgentSessionRef, Bindings: input.ReplacementBindings,
		ContextDigest: input.ContextDigest, ContextRefs: append([]domain.WaldoContextRef(nil), input.ContextRefs...),
	})
	if startErr != nil || !started.OutcomeKnown || !started.IdentityConfirmed || started.SessionRef.IsZero() {
		detail := "Replacement identity is ambiguous; reconcile before any retry."
		if started.Detail != "" {
			detail = started.Detail
		} else if startErr != nil {
			detail = startErr.Error()
		}
		return service.recordContinuation(ctx, snapshot, input, request, domain.ContinuationUnconfirmed,
			false, nil, input.EffectsKnown, true, fence.FenceReceiptRef,
			firstNonBlank(started.ReconciliationRef, fence.ReconciliationRef), "", detail)
	}
	return service.recordContinuation(ctx, snapshot, input, request, domain.ContinuationAutomatic,
		false, nil, true, true, fence.FenceReceiptRef,
		firstNonBlank(started.ReconciliationRef, fence.ReconciliationRef), started.SessionRef, "")
}

func (service *Service) recordContinuation(
	ctx context.Context,
	snapshot ports.WaldoConversationSnapshot,
	input ContinuationInput,
	request ports.WaldoIdempotency,
	action domain.ContinuationAction,
	material bool,
	changed []string,
	effectsKnown, fenced bool,
	fenceRef, reconciliationRef string,
	toRef domain.AttemptSessionRefID,
	needsUserReason string,
) (domain.ContinuationReceipt, error) {
	fromEpisode, found := currentActiveEpisode(snapshot.Episodes)
	if !found {
		return domain.ContinuationReceipt{}, apierr.Conflict("WALDO_CONTINUATION_EPISODE_MISSING", "No active bounded conversation episode can be continued", nil)
	}
	var replacementEpisode *domain.WaldoConversationEpisode
	toEpisodeID := domain.WaldoConversationEpisodeID("")
	now := service.clock()
	if action == domain.ContinuationAutomatic {
		episode := domain.WaldoConversationEpisode{
			ID:             "waldo-episode-" + domain.WaldoConversationEpisodeID(uuid.NewString()),
			ConversationID: snapshot.Conversation.ID, ProjectID: snapshot.Conversation.ProjectID,
			Ordinal: int64(len(snapshot.Episodes) + 1), State: domain.WaldoEpisodeActive,
			CreatedAt: now,
		}
		toEpisodeID = episode.ID
		replacementEpisode = &episode
	}
	receipt := domain.ContinuationReceipt{
		ID: "waldo-continuation-" + uuid.NewString(), ConversationID: snapshot.Conversation.ID,
		ProjectID: snapshot.Conversation.ProjectID, FromAgentSessionRef: input.FromAgentSessionRef,
		FromEpisodeID: fromEpisode.ID, ToEpisodeID: toEpisodeID,
		ToAgentSessionRef: toRef, Action: action, Reason: input.Reason,
		ReasonDetail: input.ReasonDetail, MaterialChange: material,
		ChangedFields: append([]string(nil), changed...), ContextDigest: input.ContextDigest,
		ContextRefs:      append([]domain.WaldoContextRef(nil), input.ContextRefs...),
		PreviousBindings: input.PreviousBindings, ReplacementBindings: input.ReplacementBindings,
		EffectsKnown: effectsKnown, OldSessionFenced: fenced,
		ReplacementIdentityConfirmed: action == domain.ContinuationAutomatic,
		FenceReceiptRef:              fenceRef, ReconciliationRef: reconciliationRef,
		NeedsUserReason: needsUserReason, CreatedAt: now,
	}
	if err := receipt.Validate(); err != nil {
		return domain.ContinuationReceipt{}, apierr.Invalid(CodeContinuationInputInvalid, err.Error(), nil)
	}
	stored, err := service.store.RecordContinuationReceipt(ctx, receipt, replacementEpisode, request)
	if err != nil {
		return domain.ContinuationReceipt{}, mapStoreError(err)
	}
	return stored, nil
}

func episodeActive(episodes []domain.WaldoConversationEpisode, id domain.WaldoConversationEpisodeID) bool {
	for _, episode := range episodes {
		if episode.ID == id && episode.State == domain.WaldoEpisodeActive {
			return true
		}
	}
	return false
}

func currentActiveEpisode(episodes []domain.WaldoConversationEpisode) (domain.WaldoConversationEpisode, bool) {
	for index := len(episodes) - 1; index >= 0; index-- {
		if episodes[index].State == domain.WaldoEpisodeActive {
			return episodes[index], true
		}
	}
	return domain.WaldoConversationEpisode{}, false
}

func contextObjectKey(ref domain.WaldoContextRef) string {
	return string(ref.Kind) + "\x00" + ref.ObjectID
}

func findContextAttachment(attachments []domain.WaldoContextAttachment, id domain.WaldoContextAttachmentID) (domain.WaldoContextAttachment, bool) {
	for _, attachment := range attachments {
		if attachment.ID == id {
			return attachment, true
		}
	}
	return domain.WaldoContextAttachment{}, false
}

func appendMissing(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func needsUserReason(input ContinuationInput, changed []string) string {
	switch {
	case input.FreshVerifier:
		return "Start a fresh verifier Attempt without inheriting implementer conclusions."
	case input.SourceRevoked:
		return "A context source was revoked; review what remains before continuing."
	case input.LostMaterialContext:
		return "Material context could not be preserved; review the bounded packet before continuing."
	case !input.EffectsKnown:
		return "A consequential effect has an unknown outcome; reconcile it before replacement."
	case len(changed) > 0:
		return "Material continuation bindings changed: " + strings.Join(changed, ", ") + "."
	default:
		return "Continuation requires owner review."
	}
}

func contextRevisionConflict(requested domain.WaldoContextRef, current string) error {
	return apierr.Conflict(CodeContextRevision,
		"That context object moved to a newer canonical revision; reload before attaching it",
		map[string]any{"kind": requested.Kind, "objectId": requested.ObjectID, "requestedRevision": requested.Revision, "currentRevision": current})
}

func mapStoreError(err error) error {
	if err == nil {
		return nil
	}
	var revision *ports.WaldoConversationRevisionConflictError
	if errors.As(err, &revision) {
		return apierr.Conflict(CodeConversationRevision, "The Project conversation changed; reload and retry", map[string]any{
			"conversationId": revision.ConversationID, "expectedRevision": revision.ExpectedRevision, "currentRevision": revision.CurrentRevision,
		})
	}
	var idempotency *ports.WaldoIdempotencyConflictError
	if errors.As(err, &idempotency) {
		return apierr.Conflict(CodeConversationIdempotency, "That idempotency key belongs to different conversation input", map[string]any{"requestKey": idempotency.Key})
	}
	var contextRevision *ports.WaldoContextRevisionConflictError
	if errors.As(err, &contextRevision) {
		return contextRevisionConflict(domain.WaldoContextRef{Kind: contextRevision.Kind, ObjectID: contextRevision.ObjectID, Revision: contextRevision.RequestedRevision}, contextRevision.CurrentRevision)
	}
	return err
}

func fingerprint(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = fmt.Fprintf(hash, "%d:%s;", len(part), part)
	}
	return fmt.Sprintf("v1:%x", hash.Sum(nil))
}

func episodeFingerprint(projectID domain.ProjectID, ref *domain.WaldoProviderEpisodeRef) string {
	if ref == nil {
		return fingerprint(string(projectID), "manual")
	}
	return fingerprint(string(projectID), string(ref.Provider), ref.ProviderConversationID, ref.TranscriptRef)
}

func turnFingerprint(projectID domain.ProjectID, input AppendTurnInput) string {
	parts := []string{string(projectID), input.EpisodeID.String(), string(input.Role), input.Message}
	if input.ProviderRef != nil {
		parts = append(parts, input.ProviderRef.CanonicalKey())
	}
	for _, id := range input.ContextAttachmentIDs {
		parts = append(parts, id.String())
	}
	return fingerprint(parts...)
}

func contextFingerprint(projectID domain.ProjectID, ref domain.WaldoContextRef, action, reason string) string {
	return fingerprint(string(projectID), action, string(ref.Kind), ref.ObjectID, ref.Revision,
		string(ref.Provenance.Kind), ref.Provenance.SourceID, reason)
}

func contextPacketDigest(packet BoundedContextPacket) string {
	parts := make([]string, 0, 2+5*len(packet.References)+1)
	parts = append(parts, packet.CurrentIntent, strconv.FormatInt(packet.ConversationRevision, 10))
	for _, ref := range packet.References {
		parts = append(parts, string(ref.Kind), ref.ObjectID, ref.Revision, string(ref.Provenance.Kind), ref.Provenance.SourceID)
	}
	parts = append(parts, strconv.Itoa(packet.OmittedReferences))
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(parts, "\x00"))))
}

func continuationFingerprint(projectID domain.ProjectID, input ContinuationInput) string {
	parts := []string{
		string(projectID), input.FromAgentSessionRef.String(), string(input.Reason), input.ReasonDetail,
		input.ContextDigest, strconv.FormatBool(input.EffectsKnown), strconv.FormatBool(input.LostMaterialContext),
		strconv.FormatBool(input.SourceRevoked), strconv.FormatBool(input.FreshVerifier),
	}
	appendBindings := func(bindings domain.ContinuationBindings) {
		parts = append(parts, string(bindings.ProjectID), bindings.OutcomeID.String(), bindings.ContractRevisionID.String(),
			bindings.PlanRevisionID.String(), bindings.WorkUnitID.String(), bindings.AttemptID.String(),
			string(bindings.Provider), bindings.Model, bindings.Profile, bindings.Role, bindings.AuthorityDigest,
			bindings.BudgetDigest, bindings.WorkspaceOwner, bindings.EffectPolicyDigest)
	}
	appendBindings(input.PreviousBindings)
	appendBindings(input.ReplacementBindings)
	for _, ref := range input.ContextRefs {
		parts = append(parts, string(ref.Kind), ref.ObjectID, ref.Revision, string(ref.Provenance.Kind), ref.Provenance.SourceID)
	}
	return fingerprint(parts...)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

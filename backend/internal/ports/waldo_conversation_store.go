package ports

import (
	"context"
	"fmt"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

// WaldoIdempotency binds a caller request key to normalized input.
type WaldoIdempotency struct {
	Key         string
	Fingerprint string
}

// WaldoConversationSnapshot is the complete restart-safe Project conversation.
type WaldoConversationSnapshot struct {
	Conversation         domain.WaldoConversation
	Episodes             []domain.WaldoConversationEpisode
	Turns                []domain.WaldoConversationTurn
	ContextAttachments   []domain.WaldoContextAttachment
	ContinuationReceipts []domain.ContinuationReceipt
}

// WaldoConversationRevisionConflictError reports a stale aggregate revision.
type WaldoConversationRevisionConflictError struct {
	ConversationID   domain.WaldoConversationID
	ExpectedRevision int64
	CurrentRevision  int64
}

func (err *WaldoConversationRevisionConflictError) Error() string {
	return fmt.Sprintf("Waldo conversation %s revision conflict: expected %d, current %d", err.ConversationID, err.ExpectedRevision, err.CurrentRevision)
}

// WaldoIdempotencyConflictError reports request-key reuse with different input.
type WaldoIdempotencyConflictError struct{ Key string }

func (err *WaldoIdempotencyConflictError) Error() string {
	return fmt.Sprintf("Waldo conversation idempotency key %q is already bound to different input", err.Key)
}

// WaldoContextRevisionConflictError reports an attachment request for stale canonical truth.
type WaldoContextRevisionConflictError struct {
	Kind              domain.WaldoContextRefKind
	ObjectID          string
	RequestedRevision string
	CurrentRevision   string
}

func (err *WaldoContextRevisionConflictError) Error() string {
	return fmt.Sprintf("Waldo context %s %s revision conflict: requested %s, current %s", err.Kind, err.ObjectID, err.RequestedRevision, err.CurrentRevision)
}

// WaldoConversationStore persists the Project conversation aggregate. All
// ordering/revision/idempotency mutations are atomic at this boundary.
type WaldoConversationStore interface {
	GetProject(context.Context, string) (domain.ProjectRecord, bool, error)
	EnsureWaldoConversation(context.Context, domain.WaldoConversation) (WaldoConversationSnapshot, error)
	GetWaldoConversationByProject(context.Context, domain.ProjectID) (WaldoConversationSnapshot, bool, error)
	OpenWaldoEpisode(context.Context, domain.WaldoConversationEpisode, WaldoIdempotency, int64) (WaldoConversationSnapshot, error)
	FindWaldoTurnByRequestKey(context.Context, string) (domain.WaldoConversationTurn, string, bool, error)
	AppendWaldoTurn(context.Context, domain.WaldoConversationTurn, []domain.WaldoContextAttachmentID, WaldoIdempotency, int64) (WaldoConversationSnapshot, domain.WaldoConversationTurn, error)
	ResolveWaldoContextRef(context.Context, domain.ProjectID, domain.WaldoContextRef) (domain.WaldoContextRef, bool, error)
	AttachWaldoContext(context.Context, domain.WaldoContextAttachment, WaldoIdempotency, int64) (WaldoConversationSnapshot, error)
	DetachWaldoContext(context.Context, domain.WaldoConversationID, domain.WaldoContextAttachmentID, string, WaldoIdempotency, int64, time.Time) (WaldoConversationSnapshot, error)
	ClaimWaldoContinuationOperation(context.Context, domain.WaldoContinuationOperation) (domain.WaldoContinuationOperation, bool, error)
	FindWaldoContinuationOperationByRequestKey(context.Context, string) (domain.WaldoContinuationOperation, bool, error)
	AdvanceWaldoContinuationOperation(context.Context, string, domain.WaldoContinuationOperationState, domain.WaldoContinuationOperationState, string, string, string, time.Time) (domain.WaldoContinuationOperation, error)
	ListPendingWaldoContinuationOperations(context.Context) ([]domain.WaldoContinuationOperation, error)
	FindContinuationReceiptByRequestKey(context.Context, string) (domain.ContinuationReceipt, string, bool, error)
	RecordContinuationReceipt(context.Context, domain.ContinuationReceipt, *domain.WaldoConversationEpisode, WaldoIdempotency) (domain.ContinuationReceipt, error)
}

// ContinuationFenceResult is the machine/owner proof that the predecessor is contained.
type ContinuationFenceResult struct {
	Fenced            bool
	FenceReceiptRef   string
	ReconciliationRef string
	Detail            string
}

// ContinuationStartRequest carries only frozen bindings and identifier-only context.
type ContinuationStartRequest struct {
	RequestKey          string
	FromAgentSessionRef domain.AttemptSessionRefID
	Bindings            domain.ContinuationBindings
	ContextDigest       string
	ContextRefs         []domain.WaldoContextRef
}

// ContinuationStartResult separates provider outcome knowledge from identity confirmation.
type ContinuationStartResult struct {
	OutcomeKnown      bool
	IdentityConfirmed bool
	SessionRef        domain.AttemptSessionRefID
	ReconciliationRef string
	Detail            string
}

// WaldoContinuationFactsRequest carries proposed input to the canonical facts
// resolver. Proposals are not authority; the resolver returns daemon truth.
type WaldoContinuationFactsRequest struct {
	ProjectID           domain.ProjectID
	FromAgentSessionRef domain.AttemptSessionRefID
	Reason              domain.ContinuationReason
	TriggerEvidence     domain.ContinuationTriggerEvidence
	PreviousBindings    domain.ContinuationBindings
	ReplacementBindings domain.ContinuationBindings
	EffectsKnown        bool
	LostMaterialContext bool
	SourceRevoked       bool
	FreshVerifier       bool
	ContextDigest       string
	ContextRefs         []domain.WaldoContextRef
}

// WaldoContinuationFacts is canonical policy truth resolved from daemon-owned
// session/admission/effect/config records.
type WaldoContinuationFacts struct {
	PreviousBindings    domain.ContinuationBindings
	ReplacementBindings domain.ContinuationBindings
	EffectsKnown        bool
	LostMaterialContext bool
	SourceRevoked       bool
	FreshVerifier       bool
	TriggerConfirmed    bool
	TriggerEvidence     domain.ContinuationTriggerEvidence
}

// WaldoContinuationFactsResolver prevents controller/caller input from
// authorizing provider effects and rechecks the bound replacement identity.
type WaldoContinuationFactsResolver interface {
	ResolveWaldoContinuationFacts(context.Context, WaldoContinuationFactsRequest) (WaldoContinuationFacts, error)
	ConfirmWaldoReplacementBindings(context.Context, domain.ProjectID, domain.AttemptSessionRefID, domain.ContinuationBindings) (domain.ContinuationBindings, bool, error)
}

// WaldoContinuationExecutor is the provider/runtime leaf beneath daemon policy.
type WaldoContinuationExecutor interface {
	FenceForContinuation(context.Context, domain.AttemptSessionRefID) (ContinuationFenceResult, error)
	StartContinuation(context.Context, ContinuationStartRequest) (ContinuationStartResult, error)
}

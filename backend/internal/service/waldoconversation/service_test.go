package waldoconversation

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

var serviceTestTime = time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)

type memoryWaldoStore struct {
	ports.WaldoConversationStore
	project      domain.ProjectRecord
	snapshot     ports.WaldoConversationSnapshot
	turnRequests map[string]struct {
		fingerprint string
		turn        domain.WaldoConversationTurn
	}
	receiptRequests map[string]struct {
		fingerprint string
		receipt     domain.ContinuationReceipt
	}
	continuationWrites int
}

func (store *memoryWaldoStore) GetProject(_ context.Context, id string) (domain.ProjectRecord, bool, error) {
	return store.project, store.project.ID == id, nil
}

func (store *memoryWaldoStore) EnsureWaldoConversation(_ context.Context, conversation domain.WaldoConversation) (ports.WaldoConversationSnapshot, error) {
	if store.snapshot.Conversation.ID.IsZero() {
		store.snapshot.Conversation = conversation
	}
	return store.snapshot, nil
}

func (store *memoryWaldoStore) GetWaldoConversationByProject(_ context.Context, projectID domain.ProjectID) (ports.WaldoConversationSnapshot, bool, error) {
	return store.snapshot, store.snapshot.Conversation.ProjectID == projectID, nil
}

func (store *memoryWaldoStore) OpenWaldoEpisode(_ context.Context, episode domain.WaldoConversationEpisode, request ports.WaldoIdempotency, expected int64) (ports.WaldoConversationSnapshot, error) {
	if store.snapshot.Conversation.Revision != expected {
		return ports.WaldoConversationSnapshot{}, &ports.WaldoConversationRevisionConflictError{ConversationID: episode.ConversationID, ExpectedRevision: expected, CurrentRevision: store.snapshot.Conversation.Revision}
	}
	store.snapshot.Conversation.Revision++
	store.snapshot.Conversation.UpdatedAt = episode.CreatedAt
	store.snapshot.Episodes = append(store.snapshot.Episodes, episode)
	return store.snapshot, nil
}

func (store *memoryWaldoStore) FindWaldoTurnByRequestKey(_ context.Context, key string) (domain.WaldoConversationTurn, string, bool, error) {
	replay, ok := store.turnRequests[key]
	return replay.turn, replay.fingerprint, ok, nil
}

func (store *memoryWaldoStore) AppendWaldoTurn(_ context.Context, turn domain.WaldoConversationTurn, attachmentIDs []domain.WaldoContextAttachmentID, request ports.WaldoIdempotency, expected int64) (ports.WaldoConversationSnapshot, domain.WaldoConversationTurn, error) {
	if store.turnRequests == nil {
		store.turnRequests = map[string]struct {
			fingerprint string
			turn        domain.WaldoConversationTurn
		}{}
	}
	if replay, ok := store.turnRequests[request.Key]; ok {
		if replay.fingerprint != request.Fingerprint {
			return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, &ports.WaldoIdempotencyConflictError{Key: request.Key}
		}
		return store.snapshot, replay.turn, nil
	}
	if store.snapshot.Conversation.Revision != expected {
		return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, &ports.WaldoConversationRevisionConflictError{ConversationID: turn.ConversationID, ExpectedRevision: expected, CurrentRevision: store.snapshot.Conversation.Revision}
	}
	turn.Sequence = store.snapshot.Conversation.LatestTurnSequence + 1
	for _, attachmentID := range attachmentIDs {
		for _, attachment := range store.snapshot.ContextAttachments {
			if attachment.ID == attachmentID && attachment.Active() {
				turn.ContextRefs = append(turn.ContextRefs, attachment.Ref)
			}
		}
	}
	store.snapshot.Conversation.Revision++
	store.snapshot.Conversation.LatestTurnSequence = turn.Sequence
	store.snapshot.Conversation.UpdatedAt = turn.CreatedAt
	store.snapshot.Turns = append(store.snapshot.Turns, turn)
	store.turnRequests[request.Key] = struct {
		fingerprint string
		turn        domain.WaldoConversationTurn
	}{request.Fingerprint, turn}
	return store.snapshot, turn, nil
}

func (store *memoryWaldoStore) ResolveWaldoContextRef(_ context.Context, projectID domain.ProjectID, ref domain.WaldoContextRef) (domain.WaldoContextRef, bool, error) {
	if projectID != store.snapshot.Conversation.ProjectID {
		return domain.WaldoContextRef{}, false, nil
	}
	current := ref
	if ref.Kind == domain.WaldoContextOutcome && ref.ObjectID == "outcome-1" {
		current.Revision = "4"
		return current, true, nil
	}
	if ref.Kind == domain.WaldoContextProject && ref.ObjectID == string(projectID) {
		return current, true, nil
	}
	if ref.Kind == domain.WaldoContextPlanRevision && ref.ObjectID == "plan-proposed" {
		current.Revision = "1:proposed"
		return current, true, nil
	}
	return domain.WaldoContextRef{}, false, nil
}

func (store *memoryWaldoStore) AttachWaldoContext(_ context.Context, attachment domain.WaldoContextAttachment, request ports.WaldoIdempotency, expected int64) (ports.WaldoConversationSnapshot, error) {
	if store.snapshot.Conversation.Revision != expected {
		return ports.WaldoConversationSnapshot{}, &ports.WaldoConversationRevisionConflictError{ConversationID: attachment.ConversationID, ExpectedRevision: expected, CurrentRevision: store.snapshot.Conversation.Revision}
	}
	store.snapshot.Conversation.Revision++
	attachment.AttachedRevision = store.snapshot.Conversation.Revision
	store.snapshot.ContextAttachments = append(store.snapshot.ContextAttachments, attachment)
	return store.snapshot, nil
}

func (store *memoryWaldoStore) DetachWaldoContext(_ context.Context, conversationID domain.WaldoConversationID, attachmentID domain.WaldoContextAttachmentID, reason string, request ports.WaldoIdempotency, expected int64, at time.Time) (ports.WaldoConversationSnapshot, error) {
	if store.snapshot.Conversation.Revision != expected {
		return ports.WaldoConversationSnapshot{}, &ports.WaldoConversationRevisionConflictError{ConversationID: conversationID, ExpectedRevision: expected, CurrentRevision: store.snapshot.Conversation.Revision}
	}
	store.snapshot.Conversation.Revision++
	for index := range store.snapshot.ContextAttachments {
		if store.snapshot.ContextAttachments[index].ID == attachmentID {
			store.snapshot.ContextAttachments[index].DetachedRevision = store.snapshot.Conversation.Revision
			store.snapshot.ContextAttachments[index].DetachedAt = &at
			store.snapshot.ContextAttachments[index].DetachReason = reason
		}
	}
	return store.snapshot, nil
}

func (store *memoryWaldoStore) FindContinuationReceiptByRequestKey(_ context.Context, key string) (domain.ContinuationReceipt, string, bool, error) {
	value, ok := store.receiptRequests[key]
	return value.receipt, value.fingerprint, ok, nil
}

func (store *memoryWaldoStore) RecordContinuationReceipt(_ context.Context, receipt domain.ContinuationReceipt, replacement *domain.WaldoConversationEpisode, request ports.WaldoIdempotency) (domain.ContinuationReceipt, error) {
	if store.receiptRequests == nil {
		store.receiptRequests = map[string]struct {
			fingerprint string
			receipt     domain.ContinuationReceipt
		}{}
	}
	if replay, ok := store.receiptRequests[request.Key]; ok {
		if replay.fingerprint != request.Fingerprint {
			return domain.ContinuationReceipt{}, &ports.WaldoIdempotencyConflictError{Key: request.Key}
		}
		return replay.receipt, nil
	}
	store.continuationWrites++
	if receipt.Action == domain.ContinuationAutomatic || receipt.Action == domain.ContinuationUnconfirmed {
		for index := range store.snapshot.Episodes {
			if store.snapshot.Episodes[index].ID == receipt.FromEpisodeID {
				store.snapshot.Episodes[index].State = domain.WaldoEpisodeSealed
				store.snapshot.Episodes[index].SealedAt = &receipt.CreatedAt
				store.snapshot.Episodes[index].SealReason = string(receipt.Reason)
			}
		}
	}
	if replacement != nil {
		store.snapshot.Episodes = append(store.snapshot.Episodes, *replacement)
	}
	store.receiptRequests[request.Key] = struct {
		fingerprint string
		receipt     domain.ContinuationReceipt
	}{request.Fingerprint, receipt}
	store.snapshot.ContinuationReceipts = append(store.snapshot.ContinuationReceipts, receipt)
	return receipt, nil
}

type fakeContinuationExecutor struct {
	fenceResult ports.ContinuationFenceResult
	fenceErr    error
	startResult ports.ContinuationStartResult
	startErr    error
	fenceCalls  int
	startCalls  int
	beforeFence func()
	beforeStart func(ports.ContinuationStartRequest)
}

func (executor *fakeContinuationExecutor) FenceForContinuation(context.Context, domain.AttemptSessionRefID) (ports.ContinuationFenceResult, error) {
	executor.fenceCalls++
	if executor.beforeFence != nil {
		executor.beforeFence()
	}
	return executor.fenceResult, executor.fenceErr
}

func (executor *fakeContinuationExecutor) StartContinuation(_ context.Context, request ports.ContinuationStartRequest) (ports.ContinuationStartResult, error) {
	executor.startCalls++
	if executor.beforeStart != nil {
		executor.beforeStart(request)
	}
	return executor.startResult, executor.startErr
}

type claimingMemoryStore struct {
	*memoryWaldoStore
	operations map[string]domain.WaldoContinuationOperation
}

type fakeContinuationFactsResolver struct {
	facts            ports.WaldoContinuationFacts
	replacement      domain.ContinuationBindings
	replacementFound bool
	replacementErr   error
}

type echoContinuationFactsResolver struct{}

func (echoContinuationFactsResolver) ResolveWaldoContinuationFacts(_ context.Context, request ports.WaldoContinuationFactsRequest) (ports.WaldoContinuationFacts, error) {
	return ports.WaldoContinuationFacts{
		PreviousBindings: request.PreviousBindings, ReplacementBindings: request.ReplacementBindings,
		EffectsKnown: request.EffectsKnown, LostMaterialContext: request.LostMaterialContext,
		SourceRevoked: request.SourceRevoked, FreshVerifier: request.FreshVerifier,
		TriggerConfirmed: true,
		TriggerEvidence:  request.TriggerEvidence,
	}, nil
}

func (echoContinuationFactsResolver) ConfirmWaldoReplacementBindings(_ context.Context, _ domain.ProjectID, _ domain.AttemptSessionRefID, expected domain.ContinuationBindings) (domain.ContinuationBindings, bool, error) {
	return expected, true, nil
}

func (resolver *fakeContinuationFactsResolver) ResolveWaldoContinuationFacts(context.Context, ports.WaldoContinuationFactsRequest) (ports.WaldoContinuationFacts, error) {
	return resolver.facts, nil
}

func (resolver *fakeContinuationFactsResolver) ConfirmWaldoReplacementBindings(context.Context, domain.ProjectID, domain.AttemptSessionRefID, domain.ContinuationBindings) (domain.ContinuationBindings, bool, error) {
	return resolver.replacement, resolver.replacementFound, resolver.replacementErr
}

func (store *claimingMemoryStore) ClaimWaldoContinuationOperation(_ context.Context, operation domain.WaldoContinuationOperation) (domain.WaldoContinuationOperation, bool, error) {
	if store.operations == nil {
		store.operations = map[string]domain.WaldoContinuationOperation{}
	}
	if replay, found := store.operations[operation.RequestKey]; found {
		if replay.RequestFingerprint != operation.RequestFingerprint {
			return domain.WaldoContinuationOperation{}, false, &ports.WaldoIdempotencyConflictError{Key: operation.RequestKey}
		}
		return replay, false, nil
	}
	store.operations[operation.RequestKey] = operation
	return operation, true, nil
}

func (store *claimingMemoryStore) FindWaldoContinuationOperationByRequestKey(_ context.Context, requestKey string) (domain.WaldoContinuationOperation, bool, error) {
	operation, found := store.operations[requestKey]
	return operation, found, nil
}

func (store *claimingMemoryStore) AdvanceWaldoContinuationOperation(_ context.Context, id string, expected, next domain.WaldoContinuationOperationState, fenceRef, reconciliationRef, needsUserReason string, at time.Time) (domain.WaldoContinuationOperation, error) {
	for key, operation := range store.operations {
		if operation.ID != id || operation.State != expected {
			continue
		}
		operation.State = next
		operation.FenceReceiptRef = fenceRef
		operation.ReconciliationRef = reconciliationRef
		operation.NeedsUserReason = needsUserReason
		operation.UpdatedAt = at
		store.operations[key] = operation
		return operation, nil
	}
	return domain.WaldoContinuationOperation{}, errors.New("continuation operation state conflict")
}

func (store *claimingMemoryStore) ListPendingWaldoContinuationOperations(context.Context) ([]domain.WaldoContinuationOperation, error) {
	var pending []domain.WaldoContinuationOperation
	for _, operation := range store.operations {
		if operation.State != domain.WaldoContinuationCompleted {
			pending = append(pending, operation)
		}
	}
	return pending, nil
}

func (store *claimingMemoryStore) RecordContinuationReceipt(ctx context.Context, receipt domain.ContinuationReceipt, replacement *domain.WaldoConversationEpisode, request ports.WaldoIdempotency) (domain.ContinuationReceipt, error) {
	for key, operation := range store.operations {
		if operation.ID == receipt.OperationID {
			operation.State = domain.WaldoContinuationCompleted
			operation.FenceReceiptRef = receipt.FenceReceiptRef
			operation.ReconciliationRef = receipt.ReconciliationRef
			operation.NeedsUserReason = receipt.NeedsUserReason
			operation.UpdatedAt = receipt.CreatedAt
			store.operations[key] = operation
		}
	}
	return store.memoryWaldoStore.RecordContinuationReceipt(ctx, receipt, replacement, request)
}

func TestAppendTurnIsOrderedIdempotentAndUsesOnlyExplicitActiveContext(t *testing.T) {
	store := conversationServiceFixture()
	service := New(store, nil, func() time.Time { return serviceTestTime })
	input := AppendTurnInput{
		ExpectedRevision: 2, EpisodeID: "episode-1", Role: domain.WaldoTurnRoleUser,
		Message:              "Explain the current Outcome without replaying provider transcripts.",
		ContextAttachmentIDs: []domain.WaldoContextAttachmentID{"attachment-active"},
		RequestKey:           "turn-request-1",
	}

	first, err := service.AppendTurn(context.Background(), "project-1", input)
	if err != nil {
		t.Fatalf("AppendTurn() error = %v", err)
	}
	if first.Replayed {
		t.Fatal("first append was marked as a replay")
	}
	detached := input
	detached.ExpectedRevision = first.Snapshot.Conversation.Revision
	detached.ContextAttachmentIDs = []domain.WaldoContextAttachmentID{"attachment-detached"}
	detached.RequestKey = "turn-request-detached"
	if _, err := service.AppendTurn(context.Background(), "project-1", detached); apiErrorCode(err) != CodeContextNotActive {
		t.Fatalf("detached context selection error = %v", err)
	}
	// Delivered-request replay must outrank later episode state and stale expected revision.
	store.snapshot.Episodes[0].State = domain.WaldoEpisodeSealed
	sealedAt := serviceTestTime.Add(time.Minute)
	store.snapshot.Episodes[0].SealedAt = &sealedAt
	store.snapshot.Episodes[0].SealReason = "context_reserve"
	replay, err := service.AppendTurn(context.Background(), "project-1", input)
	if err != nil {
		t.Fatalf("AppendTurn() replay error = %v", err)
	}
	if !replay.Replayed {
		t.Fatal("delivered append replay was not marked as a replay")
	}
	if replay.Turn.ID != first.Turn.ID || replay.Turn.Sequence != 1 || len(replay.Turn.ContextRefs) != 1 || replay.Turn.ContextRefs[0].ObjectID != "outcome-1" {
		t.Fatalf("replay = %+v, first = %+v", replay, first)
	}

	changed := input
	changed.Message = "Different content under the same request key"
	if _, err := service.AppendTurn(context.Background(), "project-1", changed); apiErrorCode(err) != CodeConversationIdempotency {
		t.Fatalf("changed idempotency replay error = %v", err)
	}
}

func TestAttachContextRejectsStaleCanonicalRevisionAndDetachIsExplicit(t *testing.T) {
	store := conversationServiceFixture()
	service := New(store, nil, func() time.Time { return serviceTestTime })
	_, err := service.AttachContext(context.Background(), "project-1", AttachContextInput{
		ExpectedRevision: 2,
		Ref:              domain.WaldoContextRef{Kind: domain.WaldoContextOutcome, ObjectID: "outcome-1", Revision: "3", Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceUser, SourceID: "turn-attach"}},
		RequestKey:       "attach-stale",
	})
	if apiErrorCode(err) != CodeContextRevision {
		t.Fatalf("stale canonical context error = %v", err)
	}

	attached, err := service.AttachContext(context.Background(), "project-1", AttachContextInput{
		ExpectedRevision: 2,
		Ref:              domain.WaldoContextRef{Kind: domain.WaldoContextOutcome, ObjectID: "outcome-1", Revision: "4", Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceUser, SourceID: "turn-attach"}},
		RequestKey:       "attach-current",
	})
	if err != nil {
		t.Fatalf("AttachContext() error = %v", err)
	}
	attachment := attached.Snapshot.ContextAttachments[len(attached.Snapshot.ContextAttachments)-1]
	detached, err := service.DetachContext(context.Background(), "project-1", DetachContextInput{
		ExpectedRevision: attached.Snapshot.Conversation.Revision, AttachmentID: attachment.ID,
		Reason: "The owner revoked this source.", RequestKey: "detach-source",
	})
	if err != nil {
		t.Fatalf("DetachContext() error = %v", err)
	}
	if detached.Snapshot.ContextAttachments[len(detached.Snapshot.ContextAttachments)-1].Active() {
		t.Fatal("detached context remained active")
	}
}

func TestAttachContextWithoutRequestedRevisionUsesCanonicalTruth(t *testing.T) {
	store := conversationServiceFixture()
	service := New(store, nil, func() time.Time { return serviceTestTime })

	attached, err := service.AttachContext(context.Background(), "project-1", AttachContextInput{
		ExpectedRevision: 2,
		Ref: domain.WaldoContextRef{
			Kind:       domain.WaldoContextOutcome,
			ObjectID:   "outcome-1",
			Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceUser, SourceID: "waldo-rail"},
		},
		RequestKey: "attach-current-without-revision",
	})
	if err != nil {
		t.Fatalf("AttachContext() error = %v", err)
	}
	attachment := attached.Snapshot.ContextAttachments[len(attached.Snapshot.ContextAttachments)-1]
	if attachment.Ref.Revision != "4" {
		t.Fatalf("attached canonical revision = %q, want 4", attachment.Ref.Revision)
	}
}

func TestCompileContextKeepsIntentAndCurrentCanonicalRevisionAheadOfLowerSources(t *testing.T) {
	store := conversationServiceFixture()
	service := New(store, nil, func() time.Time { return serviceTestTime })
	packet, err := service.CompileContext(context.Background(), "project-1", CompileContextInput{
		CurrentIntent: "Use the current approved Outcome revision.", MaxReferences: 3,
		Candidates: []ContextCandidate{
			{Tier: ContextTierPriorSummary, Ref: domain.WaldoContextRef{Kind: domain.WaldoContextOutcome, ObjectID: "outcome-1", Revision: "2", Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceProvider, SourceID: "summary-old"}}},
			{Tier: ContextTierModelOutput, Ref: domain.WaldoContextRef{Kind: domain.WaldoContextPlanRevision, ObjectID: "plan-proposed", Revision: "1:proposed", Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceProvider, SourceID: "model-turn"}}},
		},
	})
	if err != nil {
		t.Fatalf("CompileContext() error = %v", err)
	}
	if packet.CurrentIntent != "Use the current approved Outcome revision." {
		t.Fatalf("current intent = %q", packet.CurrentIntent)
	}
	if len(packet.References) == 0 || packet.References[0].ObjectID != "outcome-1" || packet.References[0].Revision != "4" {
		t.Fatalf("context precedence = %+v", packet.References)
	}
	for _, ref := range packet.References {
		if ref.ObjectID == "outcome-1" && ref.Revision == "2" {
			t.Fatal("stale provider summary survived current canonical context")
		}
	}
	if len(packet.Digest) != 64 {
		t.Fatalf("context digest = %q", packet.Digest)
	}
}

func TestCompileContextFailsClosedWhenAttachedCanonicalRevisionMoves(t *testing.T) {
	store := conversationServiceFixture()
	store.snapshot.ContextAttachments[0].Ref.Revision = "2"
	service := New(store, nil, func() time.Time { return serviceTestTime })
	_, err := service.CompileContext(context.Background(), "project-1", CompileContextInput{CurrentIntent: "Use current canonical truth."})
	if apiErrorCode(err) != CodeContextRevision {
		t.Fatalf("moved attached revision error = %v", err)
	}
}

func TestCompileContextFailsClosedWhenAttachedCanonicalSourceDisappears(t *testing.T) {
	store := conversationServiceFixture()
	store.snapshot.ContextAttachments[0].Ref.ObjectID = "deleted-outcome"
	service := New(store, nil, func() time.Time { return serviceTestTime })
	_, err := service.CompileContext(context.Background(), "project-1", CompileContextInput{CurrentIntent: "Continue safely."})
	if apiErrorCode(err) != CodeContextNotFound {
		t.Fatalf("missing attached source error = %v", err)
	}
}

func TestCompileContextRejectsStaleLowerTierCandidate(t *testing.T) {
	store := conversationServiceFixture()
	service := New(store, nil, func() time.Time { return serviceTestTime })
	_, err := service.CompileContext(context.Background(), "project-1", CompileContextInput{
		CurrentIntent: "Do not let stale retrieved context outrank canonical truth.",
		Candidates: []ContextCandidate{{
			Tier: ContextTierRetrieved,
			Ref: domain.WaldoContextRef{
				Kind: domain.WaldoContextPlanRevision, ObjectID: "plan-proposed", Revision: "1:approved",
				Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceRetrieval, SourceID: "retrieval-1"},
			},
		}},
	})
	if apiErrorCode(err) != CodeContextRevision {
		t.Fatalf("stale lower-tier candidate error = %v", err)
	}
}

func TestProviderBoundConversationTextIsByteBounded(t *testing.T) {
	store := conversationServiceFixture()
	service := New(store, nil, func() time.Time { return serviceTestTime })
	_, err := service.CompileContext(context.Background(), "project-1", CompileContextInput{
		CurrentIntent: strings.Repeat("x", MaxWaldoCurrentIntentBytes+1),
	})
	if apiErrorCode(err) != "WALDO_CONTEXT_TOO_LARGE" {
		t.Fatalf("oversized current intent error = %v", err)
	}
	_, err = service.AppendTurn(context.Background(), "project-1", AppendTurnInput{
		ExpectedRevision: 2, EpisodeID: "episode-1", Role: domain.WaldoTurnRoleUser,
		Message: strings.Repeat("x", MaxWaldoTurnMessageBytes+1), RequestKey: "oversized-turn",
	})
	if apiErrorCode(err) != "WALDO_TURN_TOO_LARGE" || len(store.snapshot.Turns) != 0 {
		t.Fatalf("oversized turn error=%v persisted=%d", err, len(store.snapshot.Turns))
	}
}

func TestContinuationPolicyAutomaticallyStartsOnlySafeSameAuthorityReplacement(t *testing.T) {
	for _, reason := range []domain.ContinuationReason{
		domain.ContinuationReasonContextReserve,
		domain.ContinuationReasonConservativeThreshold,
	} {
		t.Run(string(reason), func(t *testing.T) {
			store := conversationServiceFixture()
			executor := &fakeContinuationExecutor{
				fenceResult: ports.ContinuationFenceResult{Fenced: true, FenceReceiptRef: "fence-1", ReconciliationRef: "reconcile-old"},
				startResult: ports.ContinuationStartResult{OutcomeKnown: true, IdentityConfirmed: true, SessionRef: "session-ref-2", ReconciliationRef: "reconcile-new"},
			}
			service := newTestContinuationService(store, executor)
			input := safeContinuationInput()
			input.Reason = reason
			input.TriggerEvidence = triggerEvidenceFor(reason)
			input.RequestKey += "-" + string(reason)

			receipt, err := service.Continue(context.Background(), "project-1", input)
			if err != nil {
				t.Fatalf("Continue() error = %v", err)
			}
			if receipt.Action != domain.ContinuationAutomatic || receipt.ToAgentSessionRef != "session-ref-2" || executor.startCalls != 1 {
				t.Fatalf("receipt = %+v start calls = %d", receipt, executor.startCalls)
			}
			if receipt.FromEpisodeID != "episode-1" || receipt.ToEpisodeID.IsZero() || len(store.snapshot.Episodes) != 2 ||
				store.snapshot.Episodes[0].State != domain.WaldoEpisodeSealed || store.snapshot.Episodes[1].State != domain.WaldoEpisodeActive {
				t.Fatalf("automatic continuation episode lineage = receipt %+v episodes %+v", receipt, store.snapshot.Episodes)
			}

			replayed, err := service.Continue(context.Background(), "project-1", input)
			if err != nil || replayed.ID != receipt.ID || executor.startCalls != 1 {
				t.Fatalf("replay = %+v err=%v start calls=%d", replayed, err, executor.startCalls)
			}
		})
	}
}

func TestContinuationPolicyCreatesDurableNeedsYouWithoutStartingMaterialReplacement(t *testing.T) {
	base := safeContinuationInput()
	tests := map[string]func(*ContinuationInput){
		"Project change":        func(input *ContinuationInput) { input.ReplacementBindings.ProjectID = "project-2" },
		"Outcome change":        func(input *ContinuationInput) { input.ReplacementBindings.OutcomeID = "outcome-2" },
		"Contract change":       func(input *ContinuationInput) { input.ReplacementBindings.ContractRevisionID = "contract-2" },
		"Plan change":           func(input *ContinuationInput) { input.ReplacementBindings.PlanRevisionID = "plan-2" },
		"Work Unit change":      func(input *ContinuationInput) { input.ReplacementBindings.WorkUnitID = "work-unit-2" },
		"Attempt change":        func(input *ContinuationInput) { input.ReplacementBindings.AttemptID = "attempt-2" },
		"provider change":       func(input *ContinuationInput) { input.ReplacementBindings.Provider = "deepseek-harness" },
		"model change":          func(input *ContinuationInput) { input.ReplacementBindings.Model = "different-model" },
		"profile change":        func(input *ContinuationInput) { input.ReplacementBindings.Profile = "different-profile" },
		"role change":           func(input *ContinuationInput) { input.ReplacementBindings.Role = "planner" },
		"authority change":      func(input *ContinuationInput) { input.ReplacementBindings.AuthorityDigest = digest('e') },
		"budget change":         func(input *ContinuationInput) { input.ReplacementBindings.BudgetDigest = digest('e') },
		"workspace change":      func(input *ContinuationInput) { input.ReplacementBindings.WorkspaceOwner = "worktree-2" },
		"effect policy change":  func(input *ContinuationInput) { input.ReplacementBindings.EffectPolicyDigest = digest('e') },
		"unknown effect":        func(input *ContinuationInput) { input.EffectsKnown = false },
		"lost material context": func(input *ContinuationInput) { input.LostMaterialContext = true },
		"source revoked": func(input *ContinuationInput) {
			input.SourceRevoked = true
			input.Reason = domain.ContinuationReasonSourceRevoked
			input.TriggerEvidence = triggerEvidenceFor(input.Reason)
		},
		"fresh verifier": func(input *ContinuationInput) {
			input.FreshVerifier = true
			input.ReplacementBindings.Role = "verifier"
			input.Reason = domain.ContinuationReasonFreshVerifier
			input.TriggerEvidence = triggerEvidenceFor(input.Reason)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			store := conversationServiceFixture()
			executor := &fakeContinuationExecutor{}
			service := newTestContinuationService(store, executor)
			input := base
			input.RequestKey = "continuation-" + name
			mutate(&input)
			receipt, err := service.Continue(context.Background(), "project-1", input)
			if err != nil {
				t.Fatalf("Continue() error = %v", err)
			}
			if receipt.Action != domain.ContinuationNeedsYou || executor.fenceCalls != 0 || executor.startCalls != 0 || !receipt.ToAgentSessionRef.IsZero() {
				t.Fatalf("receipt = %+v fence=%d start=%d", receipt, executor.fenceCalls, executor.startCalls)
			}
		})
	}
}

func TestContinuationReasonIntrinsicallyStopsAutomaticReplacement(t *testing.T) {
	for _, reason := range []domain.ContinuationReason{
		domain.ContinuationReasonMaterialDigestChange,
		domain.ContinuationReasonIdentityLost,
		domain.ContinuationReasonSourceRevoked,
		domain.ContinuationReasonFreshVerifier,
		domain.ContinuationReasonUserRequested,
	} {
		t.Run(string(reason), func(t *testing.T) {
			store := conversationServiceFixture()
			executor := &fakeContinuationExecutor{}
			service := newTestContinuationService(store, executor)
			input := safeContinuationInput()
			input.Reason = reason
			input.TriggerEvidence = triggerEvidenceFor(reason)
			input.RequestKey = "reason-" + string(reason)

			receipt, err := service.Continue(context.Background(), "project-1", input)
			if err != nil {
				t.Fatalf("Continue() error = %v", err)
			}
			if receipt.Action != domain.ContinuationNeedsYou || executor.fenceCalls != 0 || executor.startCalls != 0 {
				t.Fatalf("reason %s produced receipt=%+v fence=%d start=%d", reason, receipt, executor.fenceCalls, executor.startCalls)
			}
		})
	}
}

func TestUnknownEffectsNeedOwnerWithoutInventingMaterialChange(t *testing.T) {
	store := conversationServiceFixture()
	service := newTestContinuationService(store, &fakeContinuationExecutor{})
	input := safeContinuationInput()
	input.EffectsKnown = false
	input.RequestKey = "unknown-effects-material-truth"

	receipt, err := service.Continue(context.Background(), "project-1", input)
	if err != nil {
		t.Fatalf("Continue() error = %v", err)
	}
	if receipt.Action != domain.ContinuationNeedsYou || receipt.MaterialChange || len(receipt.ChangedFields) != 0 {
		t.Fatalf("unknown effects receipt = %+v", receipt)
	}
}

func TestContinuationPersistsEveryEffectBoundaryBeforeExecutorCalls(t *testing.T) {
	store := conversationServiceFixture()
	var boundaryErrors []string
	executor := &fakeContinuationExecutor{
		fenceResult: ports.ContinuationFenceResult{Fenced: true, FenceReceiptRef: "fence-1", ReconciliationRef: "reconcile-old"},
		startResult: ports.ContinuationStartResult{OutcomeKnown: true, IdentityConfirmed: true, SessionRef: "session-ref-2", ReconciliationRef: "reconcile-new"},
		beforeFence: func() {
			operation := store.operations["continuation-safe"]
			if operation.State != domain.WaldoContinuationFencing {
				boundaryErrors = append(boundaryErrors, "fence called before durable fencing state")
			}
		},
		beforeStart: func(request ports.ContinuationStartRequest) {
			operation := store.operations["continuation-safe"]
			if operation.State != domain.WaldoContinuationStarting {
				boundaryErrors = append(boundaryErrors, "start called before durable starting state")
			}
			if request.RequestKey != "continuation-safe" {
				boundaryErrors = append(boundaryErrors, "executor did not receive the durable idempotency key")
			}
		},
	}
	service := newTestContinuationService(store, executor)
	if _, err := service.Continue(context.Background(), "project-1", safeContinuationInput()); err != nil {
		t.Fatalf("Continue() error = %v", err)
	}
	if len(boundaryErrors) != 0 {
		t.Fatalf("effect boundary errors = %v", boundaryErrors)
	}
}

func TestContinuationDoesNotRestartDurableStartingOperation(t *testing.T) {
	store := conversationServiceFixture()
	input := safeContinuationInput()
	store.operations[input.RequestKey] = continuationOperationFixture(input, domain.WaldoContinuationStarting)
	executor := &fakeContinuationExecutor{}
	service := newTestContinuationService(store, executor)
	_, err := service.Continue(context.Background(), "project-1", input)
	if apiErrorCode(err) != "WALDO_CONTINUATION_IN_PROGRESS" || executor.fenceCalls != 0 || executor.startCalls != 0 {
		t.Fatalf("pending continuation error=%v fence=%d start=%d", err, executor.fenceCalls, executor.startCalls)
	}
}

func TestContinuationRejectsStaleContextBeforeClaimOrEffects(t *testing.T) {
	store := conversationServiceFixture()
	executor := &fakeContinuationExecutor{}
	service := newTestContinuationService(store, executor)
	input := safeContinuationInput()
	input.ContextRefs = []domain.WaldoContextRef{{
		Kind: domain.WaldoContextPlanRevision, ObjectID: "plan-proposed", Revision: "1:approved",
		Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceCanonical, SourceID: "continuation-packet"},
	}}
	input.RequestKey = "continuation-stale-context"
	_, err := service.Continue(context.Background(), "project-1", input)
	if apiErrorCode(err) != CodeContextRevision || len(store.operations) != 0 || executor.fenceCalls != 0 || executor.startCalls != 0 {
		t.Fatalf("stale continuation context error=%v operations=%d fence=%d start=%d", err, len(store.operations), executor.fenceCalls, executor.startCalls)
	}
}

func TestContinuationUsesCanonicalFactsInsteadOfCallerAuthority(t *testing.T) {
	store := conversationServiceFixture()
	input := safeContinuationInput()
	canonical := input.PreviousBindings
	canonical.AuthorityDigest = digest('f')
	resolver := &fakeContinuationFactsResolver{facts: ports.WaldoContinuationFacts{
		PreviousBindings: canonical, ReplacementBindings: input.ReplacementBindings,
		EffectsKnown: true, TriggerConfirmed: true, TriggerEvidence: input.TriggerEvidence,
	}}
	executor := &fakeContinuationExecutor{}
	service := NewWithContinuationFacts(store, executor, resolver, func() time.Time { return serviceTestTime })
	input.RequestKey = "canonical-authority-change"
	receipt, err := service.Continue(context.Background(), "project-1", input)
	if err != nil {
		t.Fatalf("Continue() error = %v", err)
	}
	if receipt.Action != domain.ContinuationNeedsYou || !receipt.MaterialChange ||
		len(receipt.ChangedFields) != 1 || receipt.ChangedFields[0] != "authority" ||
		executor.fenceCalls != 0 || executor.startCalls != 0 {
		t.Fatalf("canonical authority receipt=%+v fence=%d start=%d", receipt, executor.fenceCalls, executor.startCalls)
	}
}

func TestContinuationUnknownCanonicalModelOrProfileStopsBeforeEffects(t *testing.T) {
	for _, field := range []string{"model", "profile"} {
		t.Run(field, func(t *testing.T) {
			store := conversationServiceFixture()
			input := safeContinuationInput()
			canonical := input.PreviousBindings
			if field == "model" {
				canonical.Model = ""
			} else {
				canonical.Profile = ""
			}
			resolver := &fakeContinuationFactsResolver{facts: ports.WaldoContinuationFacts{
				PreviousBindings: canonical, ReplacementBindings: canonical,
				EffectsKnown: true, TriggerConfirmed: true, TriggerEvidence: input.TriggerEvidence,
			}}
			executor := &fakeContinuationExecutor{}
			service := NewWithContinuationFacts(store, executor, resolver, func() time.Time { return serviceTestTime })
			input.RequestKey = "unknown-canonical-" + field
			receipt, err := service.Continue(context.Background(), "project-1", input)
			if err != nil {
				t.Fatalf("Continue() error = %v", err)
			}
			if receipt.Action != domain.ContinuationNeedsYou || receipt.MaterialChange || executor.fenceCalls != 0 || executor.startCalls != 0 {
				t.Fatalf("unknown %s receipt=%+v fence=%d start=%d", field, receipt, executor.fenceCalls, executor.startCalls)
			}
		})
	}
}

func TestContinuationUnwiredExecutorDoesNotLeavePendingClaim(t *testing.T) {
	store := conversationServiceFixture()
	service := NewWithContinuationFacts(store, nil, echoContinuationFactsResolver{}, func() time.Time { return serviceTestTime })
	_, err := service.Continue(context.Background(), "project-1", safeContinuationInput())
	if apiErrorCode(err) != CodeContinuationUnwired || len(store.operations) != 0 {
		t.Fatalf("unwired continuation error=%v pending=%d", err, len(store.operations))
	}
}

func TestContinuationRequiresCanonicalReplacementBindingConfirmation(t *testing.T) {
	store := conversationServiceFixture()
	input := safeContinuationInput()
	resolver := &fakeContinuationFactsResolver{facts: ports.WaldoContinuationFacts{
		PreviousBindings: input.PreviousBindings, ReplacementBindings: input.ReplacementBindings,
		EffectsKnown: true, TriggerConfirmed: true, TriggerEvidence: input.TriggerEvidence,
	}}
	executor := &fakeContinuationExecutor{
		fenceResult: ports.ContinuationFenceResult{Fenced: true, FenceReceiptRef: "fence-1", ReconciliationRef: "reconcile-old"},
		startResult: ports.ContinuationStartResult{OutcomeKnown: true, IdentityConfirmed: true, SessionRef: "session-ref-unverified", ReconciliationRef: "reconcile-new"},
	}
	service := NewWithContinuationFacts(store, executor, resolver, func() time.Time { return serviceTestTime })
	input.RequestKey = "canonical-replacement-unverified"
	receipt, err := service.Continue(context.Background(), "project-1", input)
	if err != nil {
		t.Fatalf("Continue() error = %v", err)
	}
	if receipt.Action != domain.ContinuationUnconfirmed || receipt.ReplacementIdentityConfirmed || !receipt.ToAgentSessionRef.IsZero() {
		t.Fatalf("unverified replacement receipt = %+v", receipt)
	}
}

func TestRecoverPendingContinuationNeverRestartsAmbiguousProviderStart(t *testing.T) {
	store := conversationServiceFixture()
	input := safeContinuationInput()
	operation := continuationOperationFixture(input, domain.WaldoContinuationStarting)
	store.operations[input.RequestKey] = operation
	executor := &fakeContinuationExecutor{}
	service := newTestContinuationService(store, executor)
	receipts, err := service.RecoverPendingContinuations(context.Background())
	if err != nil {
		t.Fatalf("RecoverPendingContinuations() error = %v", err)
	}
	if len(receipts) != 1 || receipts[0].Action != domain.ContinuationUnconfirmed ||
		executor.fenceCalls != 0 || executor.startCalls != 0 ||
		store.operations[input.RequestKey].State != domain.WaldoContinuationCompleted {
		t.Fatalf("recovery receipts=%+v operation=%+v fence=%d start=%d", receipts, store.operations[input.RequestKey], executor.fenceCalls, executor.startCalls)
	}
}

func TestContinuationPolicyFailsClosedForUnsafeFenceAndAmbiguousReplacement(t *testing.T) {
	t.Run("unsafe fence", func(t *testing.T) {
		store := conversationServiceFixture()
		executor := &fakeContinuationExecutor{fenceResult: ports.ContinuationFenceResult{Fenced: false}}
		service := newTestContinuationService(store, executor)
		receipt, err := service.Continue(context.Background(), "project-1", safeContinuationInput())
		if err != nil {
			t.Fatalf("Continue() error = %v", err)
		}
		if receipt.Action != domain.ContinuationNeedsYou || executor.startCalls != 0 {
			t.Fatalf("receipt = %+v start=%d", receipt, executor.startCalls)
		}
	})

	t.Run("ambiguous replacement", func(t *testing.T) {
		store := conversationServiceFixture()
		executor := &fakeContinuationExecutor{
			fenceResult: ports.ContinuationFenceResult{Fenced: true, FenceReceiptRef: "fence-1", ReconciliationRef: "reconcile-old"},
			startResult: ports.ContinuationStartResult{OutcomeKnown: false, IdentityConfirmed: false, Detail: "provider start timed out"},
		}
		service := newTestContinuationService(store, executor)
		receipt, err := service.Continue(context.Background(), "project-1", safeContinuationInput())
		if err != nil {
			t.Fatalf("Continue() error = %v", err)
		}
		if receipt.Action != domain.ContinuationUnconfirmed || executor.startCalls != 1 || !receipt.ToAgentSessionRef.IsZero() {
			t.Fatalf("receipt = %+v start=%d", receipt, executor.startCalls)
		}
	})
}

func conversationServiceFixture() *claimingMemoryStore {
	active := domain.WaldoContextAttachment{
		ID: "attachment-active", ConversationID: "conversation-1", ProjectID: "project-1",
		Ref:              domain.WaldoContextRef{Kind: domain.WaldoContextOutcome, ObjectID: "outcome-1", Revision: "4", Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceCanonical, SourceID: "contract-4"}},
		AttachedRevision: 1, CreatedAt: serviceTestTime,
	}
	detachedAt := serviceTestTime.Add(-time.Minute)
	detached := domain.WaldoContextAttachment{
		ID: "attachment-detached", ConversationID: "conversation-1", ProjectID: "project-1",
		Ref:              domain.WaldoContextRef{Kind: domain.WaldoContextPlanRevision, ObjectID: "plan-old", Revision: "1", Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceCanonical, SourceID: "plan-1"}},
		AttachedRevision: 1, DetachedRevision: 2, CreatedAt: serviceTestTime.Add(-time.Hour), DetachedAt: &detachedAt, DetachReason: "Superseded",
	}
	return &claimingMemoryStore{memoryWaldoStore: &memoryWaldoStore{
		project: domain.ProjectRecord{ID: "project-1"},
		snapshot: ports.WaldoConversationSnapshot{
			Conversation:       domain.WaldoConversation{ID: "conversation-1", ProjectID: "project-1", Revision: 2, CreatedAt: serviceTestTime, UpdatedAt: serviceTestTime},
			Episodes:           []domain.WaldoConversationEpisode{{ID: "episode-1", ConversationID: "conversation-1", ProjectID: "project-1", Ordinal: 1, State: domain.WaldoEpisodeActive, CreatedAt: serviceTestTime}},
			ContextAttachments: []domain.WaldoContextAttachment{active, detached},
		},
	}, operations: map[string]domain.WaldoContinuationOperation{}}
}

func safeContinuationInput() ContinuationInput {
	bindings := domain.ContinuationBindings{
		ProjectID: "project-1", OutcomeID: "outcome-1", ContractRevisionID: "contract-1",
		PlanRevisionID: "plan-1", WorkUnitID: "work-unit-1", AttemptID: "attempt-1",
		Provider: "codex", Model: "gpt-5.6", Profile: "default", Role: "implementer",
		AuthorityDigest: digest('a'), BudgetDigest: digest('b'), WorkspaceOwner: "worktree-1", EffectPolicyDigest: digest('c'),
	}
	return ContinuationInput{
		FromAgentSessionRef: "session-ref-1", Reason: domain.ContinuationReasonContextReserve,
		ReasonDetail: "Provider reported insufficient trustworthy reserve.", ContextDigest: digest('d'),
		TriggerEvidence:  triggerEvidenceFor(domain.ContinuationReasonContextReserve),
		PreviousBindings: bindings, ReplacementBindings: bindings, EffectsKnown: true,
		RequestKey: "continuation-safe",
	}
}

func newTestContinuationService(store ports.WaldoConversationStore, executor ports.WaldoContinuationExecutor) *Service {
	return NewWithContinuationFacts(store, executor, echoContinuationFactsResolver{}, func() time.Time { return serviceTestTime })
}

func continuationOperationFixture(input ContinuationInput, state domain.WaldoContinuationOperationState) domain.WaldoContinuationOperation {
	return domain.WaldoContinuationOperation{
		ID: "operation-" + input.RequestKey, ConversationID: "conversation-1", ProjectID: "project-1",
		FromEpisodeID: "episode-1", FromAgentSessionRef: input.FromAgentSessionRef,
		ExpectedConversationRevision: 2, State: state, Reason: input.Reason, ReasonDetail: input.ReasonDetail,
		TriggerEvidence: input.TriggerEvidence,
		MaterialChange:  continuationMaterialChange(input, input.PreviousBindings.Changed(input.ReplacementBindings)),
		ContextDigest:   input.ContextDigest, ContextRefs: input.ContextRefs,
		PreviousBindings: input.PreviousBindings, ReplacementBindings: input.ReplacementBindings,
		EffectsKnown: input.EffectsKnown, LostMaterialContext: input.LostMaterialContext,
		SourceRevoked: input.SourceRevoked, FreshVerifier: input.FreshVerifier, TriggerConfirmed: true,
		FenceReceiptRef: "fence-1", ReconciliationRef: "reconcile-1",
		RequestKey: input.RequestKey, RequestFingerprint: continuationFingerprint("project-1", input),
		CreatedAt: serviceTestTime, UpdatedAt: serviceTestTime,
	}
}

func digest(value byte) string {
	result := make([]byte, 64)
	for index := range result {
		result[index] = value
	}
	return string(result)
}

func triggerEvidenceFor(reason domain.ContinuationReason) domain.ContinuationTriggerEvidence {
	kinds := map[domain.ContinuationReason]domain.ContinuationTriggerEvidenceKind{
		domain.ContinuationReasonContextReserve:        domain.ContinuationEvidenceProviderContextMeter,
		domain.ContinuationReasonConservativeThreshold: domain.ContinuationEvidenceAdapterThreshold,
		domain.ContinuationReasonMaterialDigestChange:  domain.ContinuationEvidenceMaterialContextDigest,
		domain.ContinuationReasonIdentityLost:          domain.ContinuationEvidenceProviderIdentityLoss,
		domain.ContinuationReasonSourceRevoked:         domain.ContinuationEvidenceSourceRevocation,
		domain.ContinuationReasonFreshVerifier:         domain.ContinuationEvidenceVerifierBoundary,
		domain.ContinuationReasonUserRequested:         domain.ContinuationEvidenceOwnerRequest,
	}
	return domain.ContinuationTriggerEvidence{Kind: kinds[reason], Reference: "trigger-" + string(reason)}
}

func apiErrorCode(err error) string {
	var typed *apierr.Error
	if errors.As(err, &typed) {
		return typed.Code
	}
	return ""
}

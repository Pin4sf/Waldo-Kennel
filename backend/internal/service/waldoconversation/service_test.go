package waldoconversation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
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
}

func (executor *fakeContinuationExecutor) FenceForContinuation(context.Context, domain.AttemptSessionRefID) (ports.ContinuationFenceResult, error) {
	executor.fenceCalls++
	return executor.fenceResult, executor.fenceErr
}

func (executor *fakeContinuationExecutor) StartContinuation(context.Context, ports.ContinuationStartRequest) (ports.ContinuationStartResult, error) {
	executor.startCalls++
	return executor.startResult, executor.startErr
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

func TestCompileContextKeepsIntentAndCurrentCanonicalRevisionAheadOfLowerSources(t *testing.T) {
	store := conversationServiceFixture()
	service := New(store, nil, func() time.Time { return serviceTestTime })
	packet, err := service.CompileContext(context.Background(), "project-1", CompileContextInput{
		CurrentIntent: "Use the current approved Outcome revision.", MaxReferences: 3,
		Candidates: []ContextCandidate{
			{Tier: ContextTierPriorSummary, Ref: domain.WaldoContextRef{Kind: domain.WaldoContextOutcome, ObjectID: "outcome-1", Revision: "2", Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceProvider, SourceID: "summary-old"}}},
			{Tier: ContextTierModelOutput, Ref: domain.WaldoContextRef{Kind: domain.WaldoContextPlanRevision, ObjectID: "plan-proposed", Revision: "1", Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceProvider, SourceID: "model-turn"}}},
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
			service := New(store, executor, func() time.Time { return serviceTestTime })
			input := safeContinuationInput()
			input.Reason = reason
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
		},
		"fresh verifier": func(input *ContinuationInput) {
			input.FreshVerifier = true
			input.ReplacementBindings.Role = "verifier"
			input.Reason = domain.ContinuationReasonFreshVerifier
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			store := conversationServiceFixture()
			executor := &fakeContinuationExecutor{}
			service := New(store, executor, func() time.Time { return serviceTestTime })
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

func TestContinuationPolicyFailsClosedForUnsafeFenceAndAmbiguousReplacement(t *testing.T) {
	t.Run("unsafe fence", func(t *testing.T) {
		store := conversationServiceFixture()
		executor := &fakeContinuationExecutor{fenceResult: ports.ContinuationFenceResult{Fenced: false}}
		service := New(store, executor, func() time.Time { return serviceTestTime })
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
		service := New(store, executor, func() time.Time { return serviceTestTime })
		receipt, err := service.Continue(context.Background(), "project-1", safeContinuationInput())
		if err != nil {
			t.Fatalf("Continue() error = %v", err)
		}
		if receipt.Action != domain.ContinuationUnconfirmed || executor.startCalls != 1 || !receipt.ToAgentSessionRef.IsZero() {
			t.Fatalf("receipt = %+v start=%d", receipt, executor.startCalls)
		}
	})
}

func conversationServiceFixture() *memoryWaldoStore {
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
	return &memoryWaldoStore{
		project: domain.ProjectRecord{ID: "project-1"},
		snapshot: ports.WaldoConversationSnapshot{
			Conversation:       domain.WaldoConversation{ID: "conversation-1", ProjectID: "project-1", Revision: 2, CreatedAt: serviceTestTime, UpdatedAt: serviceTestTime},
			Episodes:           []domain.WaldoConversationEpisode{{ID: "episode-1", ConversationID: "conversation-1", ProjectID: "project-1", Ordinal: 1, State: domain.WaldoEpisodeActive, CreatedAt: serviceTestTime}},
			ContextAttachments: []domain.WaldoContextAttachment{active, detached},
		},
	}
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
		PreviousBindings: bindings, ReplacementBindings: bindings, EffectsKnown: true,
		RequestKey: "continuation-safe",
	}
}

func digest(value byte) string {
	result := make([]byte, 64)
	for index := range result {
		result[index] = value
	}
	return string(result)
}

func apiErrorCode(err error) string {
	var typed *apierr.Error
	if errors.As(err, &typed) {
		return typed.Code
	}
	return ""
}

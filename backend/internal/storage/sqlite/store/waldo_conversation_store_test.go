package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func TestWaldoConversationStorePersistsOrderedIdempotentTurnsAndExplicitContextAcrossRestart(t *testing.T) {
	dataDir := t.TempDir()
	s, err := sqlitetest.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	ctx := context.Background()
	seedProject(t, s, "waldo-restart")
	now := time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC)
	conversation := domain.WaldoConversation{
		ID: "waldo-conversation-restart", ProjectID: "waldo-restart",
		CreatedAt: now, UpdatedAt: now,
	}
	if _, err := s.EnsureWaldoConversation(ctx, conversation); err != nil {
		t.Fatalf("ensure conversation: %v", err)
	}
	episode := domain.WaldoConversationEpisode{
		ID: "waldo-episode-restart", ConversationID: conversation.ID, ProjectID: conversation.ProjectID,
		Ordinal: 1, State: domain.WaldoEpisodeActive, CreatedAt: now,
	}
	snapshot, err := s.OpenWaldoEpisode(ctx, episode, ports.WaldoIdempotency{Key: "episode-restart", Fingerprint: "episode-fp"}, 0)
	if err != nil || snapshot.Conversation.Revision != 1 {
		t.Fatalf("open episode snapshot=%+v err=%v", snapshot.Conversation, err)
	}
	attachment := domain.WaldoContextAttachment{
		ID: "waldo-context-project", ConversationID: conversation.ID, ProjectID: conversation.ProjectID,
		Ref: domain.WaldoContextRef{
			Kind: domain.WaldoContextProject, ObjectID: "waldo-restart",
			Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceUser, SourceID: "owner-selection"},
		},
		AttachedRevision: 2, CreatedAt: now,
	}
	snapshot, err = s.AttachWaldoContext(ctx, attachment, ports.WaldoIdempotency{Key: "attach-project", Fingerprint: "attach-fp"}, 1)
	if err != nil || snapshot.Conversation.Revision != 2 {
		t.Fatalf("attach snapshot=%+v err=%v", snapshot.Conversation, err)
	}
	turn := domain.WaldoConversationTurn{
		ID: "waldo-turn-1", ConversationID: conversation.ID, EpisodeID: episode.ID,
		ProjectID: conversation.ProjectID, Sequence: 1, Role: domain.WaldoTurnRoleUser,
		Message: "Current user intent outranks prior summaries.", CreatedAt: now,
	}
	request := ports.WaldoIdempotency{Key: "turn-request-1", Fingerprint: "turn-fp-1"}
	snapshot, stored, err := s.AppendWaldoTurn(ctx, turn, []domain.WaldoContextAttachmentID{attachment.ID}, request, 2)
	if err != nil || snapshot.Conversation.Revision != 3 || stored.Sequence != 1 || len(stored.ContextRefs) != 1 {
		t.Fatalf("append snapshot=%+v turn=%+v err=%v", snapshot.Conversation, stored, err)
	}
	replayed, replayTurn, err := s.AppendWaldoTurn(ctx, domain.WaldoConversationTurn{ID: "ignored-on-replay"}, nil, request, 2)
	if err != nil || replayTurn.ID != stored.ID || replayed.Conversation.Revision != 3 {
		t.Fatalf("replay snapshot=%+v turn=%+v err=%v", replayed.Conversation, replayTurn, err)
	}
	_, _, err = s.AppendWaldoTurn(ctx, turn, nil, ports.WaldoIdempotency{Key: request.Key, Fingerprint: "different-content"}, 3)
	var idempotencyConflict *ports.WaldoIdempotencyConflictError
	if !errors.As(err, &idempotencyConflict) {
		t.Fatalf("same key/different content error=%v", err)
	}
	stale := turn
	stale.ID, stale.Sequence, stale.Message = "waldo-turn-stale", 2, "Must not overwrite"
	_, _, err = s.AppendWaldoTurn(ctx, stale, nil, ports.WaldoIdempotency{Key: "turn-stale", Fingerprint: "turn-stale-fp"}, 2)
	var revisionConflict *ports.WaldoConversationRevisionConflictError
	if !errors.As(err, &revisionConflict) {
		t.Fatalf("stale append error=%v", err)
	}
	second := stale
	second.ID, second.Message, second.CreatedAt = "waldo-turn-2", "Read back in order.", now.Add(time.Second)
	snapshot, _, err = s.AppendWaldoTurn(ctx, second, nil, ports.WaldoIdempotency{Key: "turn-request-2", Fingerprint: "turn-fp-2"}, 3)
	if err != nil || snapshot.Conversation.LatestTurnSequence != 2 {
		t.Fatalf("second append snapshot=%+v err=%v", snapshot.Conversation, err)
	}
	snapshot, err = s.DetachWaldoContext(ctx, conversation.ID, attachment.ID, "source revoked", ports.WaldoIdempotency{Key: "detach-project", Fingerprint: "detach-fp"}, 4, now.Add(2*time.Second))
	if err != nil || snapshot.Conversation.Revision != 5 || snapshot.ContextAttachments[0].Active() {
		t.Fatalf("detach snapshot=%+v err=%v", snapshot, err)
	}
	events, err := s.EventsAfter(ctx, 0, 100)
	if err != nil {
		t.Fatalf("read CDC: %v", err)
	}
	seenAppend := false
	for _, event := range events {
		if event.Type == "waldo_conversation_turn_appended" {
			seenAppend = true
			if strings.Contains(string(event.Payload), turn.Message) {
				t.Fatalf("CDC copied visible/provider text: %s", event.Payload)
			}
		}
	}
	if !seenAppend {
		t.Fatal("trigger CDC did not record appended turn")
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reopened, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	restored, found, err := reopened.GetWaldoConversationByProject(ctx, conversation.ProjectID)
	if err != nil || !found {
		t.Fatalf("restart snapshot found=%v err=%v", found, err)
	}
	if restored.Conversation.Revision != 5 || len(restored.Episodes) != 1 || len(restored.Turns) != 2 || len(restored.ContextAttachments) != 1 {
		t.Fatalf("restart snapshot=%+v", restored)
	}
	if restored.Turns[0].Message != turn.Message || restored.Turns[0].ContextRefs[0].ObjectID != "waldo-restart" || restored.ContextAttachments[0].Active() {
		t.Fatalf("restart lost exact turn/context truth: %+v", restored)
	}
}

func TestWaldoConversationStoreResolvesCanonicalRevisionAndPersistsContinuationLineage(t *testing.T) {
	dataDir := t.TempDir()
	s, err := sqlitetest.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "waldo-continuation")
	now := time.Date(2026, 8, 26, 8, 30, 0, 0, time.UTC)
	attempt, err := s.CreateAttemptWithFence(ctx, outcomeID, plan, "waldo-attempt", domain.FenceSubjectForProject("waldo-continuation"), now)
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	fromRef, err := s.BindAttemptSession(ctx, attemptSession(attempt.ID, "provider-old"))
	if err != nil {
		t.Fatalf("bind predecessor: %v", err)
	}
	toRef, err := s.BindAttemptSession(ctx, attemptSession(attempt.ID, "provider-new"))
	if err != nil {
		t.Fatalf("bind replacement: %v", err)
	}
	resolved, found, err := s.ResolveWaldoContextRef(ctx, "waldo-continuation", domain.WaldoContextRef{
		Kind: domain.WaldoContextOutcome, ObjectID: outcomeID.String(), Revision: "stale",
		Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceCanonical, SourceID: "outcome-read"},
	})
	if err != nil || !found || !strings.HasPrefix(resolved.Revision, "1:") {
		t.Fatalf("resolve canonical outcome=%+v found=%v err=%v", resolved, found, err)
	}
	if _, found, err := s.ResolveWaldoContextRef(ctx, "another-project", resolved); err != nil || found {
		t.Fatalf("cross-project resolve found=%v err=%v", found, err)
	}
	conversation := domain.WaldoConversation{ID: "waldo-continuation-conversation", ProjectID: "waldo-continuation", CreatedAt: now, UpdatedAt: now}
	if _, err := s.EnsureWaldoConversation(ctx, conversation); err != nil {
		t.Fatalf("ensure conversation: %v", err)
	}
	oldEpisode := domain.WaldoConversationEpisode{ID: "waldo-continuation-old", ConversationID: conversation.ID, ProjectID: conversation.ProjectID, Ordinal: 1, State: domain.WaldoEpisodeActive, CreatedAt: now}
	if _, err := s.OpenWaldoEpisode(ctx, oldEpisode, ports.WaldoIdempotency{Key: "old-episode", Fingerprint: "old-episode-fp"}, 0); err != nil {
		t.Fatalf("open predecessor episode: %v", err)
	}
	staleRef := resolved
	staleRef.Revision = "0"
	_, err = s.AttachWaldoContext(ctx, domain.WaldoContextAttachment{
		ID: "waldo-stale-context", ConversationID: conversation.ID, ProjectID: conversation.ProjectID,
		Ref: staleRef, AttachedRevision: 2, CreatedAt: now,
	}, ports.WaldoIdempotency{Key: "stale-context", Fingerprint: "stale-context-fp"}, 1)
	var contextConflict *ports.WaldoContextRevisionConflictError
	if !errors.As(err, &contextConflict) || contextConflict.CurrentRevision != resolved.Revision {
		t.Fatalf("stale canonical attachment error=%v", err)
	}
	bindings := continuationBindings("waldo-continuation", outcomeID, plan, attempt.ID)
	replacement := domain.WaldoConversationEpisode{ID: "waldo-continuation-new", ConversationID: conversation.ID, ProjectID: conversation.ProjectID, Ordinal: 2, State: domain.WaldoEpisodeActive, CreatedAt: now.Add(time.Minute)}
	receipt := domain.ContinuationReceipt{
		ID: "waldo-continuation-receipt", OperationID: "waldo-continuation-operation", ConversationID: conversation.ID, ProjectID: conversation.ProjectID,
		FromEpisodeID: oldEpisode.ID, ToEpisodeID: replacement.ID,
		FromAgentSessionRef: fromRef.ID, ToAgentSessionRef: toRef.ID,
		Action: domain.ContinuationAutomatic, Reason: domain.ContinuationReasonContextReserve,
		ReasonDetail: "Provider reported a trustworthy reserve threshold.", ContextDigest: strings.Repeat("a", 64),
		TriggerEvidence: domain.ContinuationTriggerEvidence{Kind: domain.ContinuationEvidenceProviderContextMeter, Reference: "usage-fact-1"},
		ContextRefs:     []domain.WaldoContextRef{resolved}, PreviousBindings: bindings, ReplacementBindings: bindings,
		EffectsKnown: true, OldSessionFenced: true, ReplacementIdentityConfirmed: true,
		FenceReceiptRef: "fence-receipt-1", ReconciliationRef: "reconcile-1", CreatedAt: now.Add(time.Minute),
	}
	request := ports.WaldoIdempotency{Key: "continue-request", Fingerprint: "continue-fp"}
	claimContinuationOperation(t, s, receipt, request, 1, now)
	stored, err := s.RecordContinuationReceipt(ctx, receipt, &replacement, request)
	if err != nil || stored.ID != receipt.ID {
		t.Fatalf("record continuation=%+v err=%v", stored, err)
	}
	replay, err := s.RecordContinuationReceipt(ctx, domain.ContinuationReceipt{}, nil, request)
	if err != nil || replay.ID != receipt.ID {
		t.Fatalf("continuation replay=%+v err=%v", replay, err)
	}
	_, err = s.RecordContinuationReceipt(ctx, receipt, &replacement, ports.WaldoIdempotency{Key: request.Key, Fingerprint: "changed"})
	var idempotencyConflict *ports.WaldoIdempotencyConflictError
	if !errors.As(err, &idempotencyConflict) {
		t.Fatalf("continuation conflict error=%v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close before restart: %v", err)
	}
	reopened, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen after continuation: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	snapshot, found, err := reopened.GetWaldoConversationByProject(ctx, conversation.ProjectID)
	if err != nil || !found {
		t.Fatalf("continuation snapshot found=%v err=%v", found, err)
	}
	if len(snapshot.Episodes) != 2 || snapshot.Episodes[0].State != domain.WaldoEpisodeSealed || snapshot.Episodes[1].State != domain.WaldoEpisodeActive || len(snapshot.ContinuationReceipts) != 1 {
		t.Fatalf("continuation lineage=%+v", snapshot)
	}
	if snapshot.ContinuationReceipts[0].ToAgentSessionRef != toRef.ID || snapshot.ContinuationReceipts[0].ContextRefs[0].ObjectID != outcomeID.String() {
		t.Fatalf("continuation receipt lost bindings: %+v", snapshot.ContinuationReceipts[0])
	}
}

func TestWaldoConversationStorePersistsUnconfirmedWithoutPlausibleReplacementAcrossRestart(t *testing.T) {
	dataDir := t.TempDir()
	s, err := sqlitetest.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	ctx := context.Background()
	plan, outcomeID := seedApprovedPlan(t, s, "waldo-ambiguous")
	now := time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)
	attempt, err := s.CreateAttemptWithFence(ctx, outcomeID, plan, "ambiguous-attempt", domain.FenceSubjectForProject("waldo-ambiguous"), now)
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	fromRef, err := s.BindAttemptSession(ctx, attemptSession(attempt.ID, "provider-ambiguous-old"))
	if err != nil {
		t.Fatalf("bind predecessor: %v", err)
	}
	conversation := domain.WaldoConversation{ID: "waldo-ambiguous-conversation", ProjectID: "waldo-ambiguous", CreatedAt: now, UpdatedAt: now}
	if _, err := s.EnsureWaldoConversation(ctx, conversation); err != nil {
		t.Fatalf("ensure conversation: %v", err)
	}
	episode := domain.WaldoConversationEpisode{ID: "waldo-ambiguous-episode", ConversationID: conversation.ID, ProjectID: conversation.ProjectID, Ordinal: 1, State: domain.WaldoEpisodeActive, CreatedAt: now}
	if _, err := s.OpenWaldoEpisode(ctx, episode, ports.WaldoIdempotency{Key: "ambiguous-episode", Fingerprint: "ambiguous-episode-fp"}, 0); err != nil {
		t.Fatalf("open episode: %v", err)
	}
	bindings := continuationBindings("waldo-ambiguous", outcomeID, plan, attempt.ID)
	receipt := domain.ContinuationReceipt{
		ID: "waldo-ambiguous-receipt", OperationID: "waldo-ambiguous-operation", ConversationID: conversation.ID, ProjectID: conversation.ProjectID,
		FromEpisodeID: episode.ID, FromAgentSessionRef: fromRef.ID,
		Action: domain.ContinuationUnconfirmed, Reason: domain.ContinuationReasonConservativeThreshold,
		ReasonDetail:    "The replacement call timed out after the predecessor was fenced.",
		TriggerEvidence: domain.ContinuationTriggerEvidence{Kind: domain.ContinuationEvidenceAdapterThreshold, Reference: "adapter-threshold-1"},
		ContextDigest:   strings.Repeat("a", 64), PreviousBindings: bindings, ReplacementBindings: bindings,
		EffectsKnown: true, OldSessionFenced: true, FenceReceiptRef: "ambiguous-fence",
		ReconciliationRef: "ambiguous-reconcile", NeedsUserReason: "Replacement identity is ambiguous; reconcile before any retry.",
		CreatedAt: now.Add(time.Minute),
	}
	request := ports.WaldoIdempotency{Key: "ambiguous-continuation", Fingerprint: "ambiguous-continuation-fp"}
	claimContinuationOperation(t, s, receipt, request, 1, now)
	if _, err := s.AdvanceWaldoContinuationOperation(ctx, receipt.OperationID,
		domain.WaldoContinuationPrepared, domain.WaldoContinuationFencing, "", "", "", now); err != nil {
		t.Fatalf("advance to fencing: %v", err)
	}
	if _, err := s.AdvanceWaldoContinuationOperation(ctx, receipt.OperationID,
		domain.WaldoContinuationFencing, domain.WaldoContinuationFenced,
		receipt.FenceReceiptRef, receipt.ReconciliationRef, "", now); err != nil {
		t.Fatalf("advance to fenced: %v", err)
	}
	if _, err := s.AdvanceWaldoContinuationOperation(ctx, receipt.OperationID,
		domain.WaldoContinuationFenced, domain.WaldoContinuationStarting,
		receipt.FenceReceiptRef, receipt.ReconciliationRef, "", now); err != nil {
		t.Fatalf("advance to starting: %v", err)
	}
	if _, err := s.RecordContinuationReceipt(ctx, receipt, nil, request); err != nil {
		t.Fatalf("record unconfirmed: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close before restart: %v", err)
	}
	reopened, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	snapshot, found, err := reopened.GetWaldoConversationByProject(ctx, conversation.ProjectID)
	if err != nil || !found {
		t.Fatalf("restart snapshot found=%v err=%v", found, err)
	}
	if len(snapshot.Episodes) != 1 || snapshot.Episodes[0].State != domain.WaldoEpisodeSealed || len(snapshot.ContinuationReceipts) != 1 {
		t.Fatalf("ambiguous restart lineage=%+v", snapshot)
	}
	stored := snapshot.ContinuationReceipts[0]
	if stored.Action != domain.ContinuationUnconfirmed || !stored.ToEpisodeID.IsZero() || !stored.ToAgentSessionRef.IsZero() || stored.ReplacementIdentityConfirmed {
		t.Fatalf("ambiguous restart invented replacement identity: %+v", stored)
	}
}

func TestWaldoConversationStoreResolvesEveryTypedProjectContextReference(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	projectID := "waldo-context-kinds"
	plan, outcomeID := seedApprovedPlan(t, s, projectID)
	now := time.Date(2026, 8, 26, 9, 30, 0, 0, time.UTC)
	attempt, err := s.CreateAttemptWithFence(ctx, outcomeID, plan, "context-kinds-attempt", domain.FenceSubjectForProject(domain.ProjectID(projectID)), now)
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	sessionRef, err := s.BindAttemptSession(ctx, attemptSession(attempt.ID, "provider-context-kinds"))
	if err != nil {
		t.Fatalf("bind session: %v", err)
	}
	intake := domain.IntakeSession{
		ID: "intake-context-kinds", SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome,
		ProjectID: domain.ProjectID(projectID), Statement: "Resolve this intake by identifier only.",
		Status: domain.IntakeStatusCaptured, CreatedAt: now, UpdatedAt: now,
	}
	if _, err := s.CreateIntake(ctx, intake, nil, ports.IntakeIdempotency{Key: "context-kinds-intake", Fingerprint: "context-kinds-intake-fp"}); err != nil {
		t.Fatalf("create intake: %v", err)
	}
	tests := []struct {
		kind           domain.WaldoContextRefKind
		objectID       string
		revisionPrefix string
	}{
		{domain.WaldoContextProject, projectID, ""},
		{domain.WaldoContextOutcome, outcomeID.String(), "1:"},
		{domain.WaldoContextContractRevision, "cr-" + projectID, "1"},
		{domain.WaldoContextPlanRevision, plan.ID.String(), "1:approved:"},
		{domain.WaldoContextWorkUnit, plan.WorkUnits[0].ID.String(), "1:approved:"},
		{domain.WaldoContextAttempt, attempt.ID.String(), "1:queued:"},
		{domain.WaldoContextAgentSessionRef, sessionRef.ID.String(), "1:queued:"},
		{domain.WaldoContextIntakeSession, intake.ID.String(), "0:captured:"},
	}
	for _, test := range tests {
		t.Run(string(test.kind), func(t *testing.T) {
			requestedRevision := "requested"
			if test.kind == domain.WaldoContextProject {
				requestedRevision = ""
			}
			resolved, found, err := s.ResolveWaldoContextRef(ctx, domain.ProjectID(projectID), domain.WaldoContextRef{
				Kind: test.kind, ObjectID: test.objectID, Revision: requestedRevision,
				Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceCanonical, SourceID: "typed-context-test"},
			})
			if err != nil || !found || resolved.ObjectID != test.objectID || !strings.HasPrefix(resolved.Revision, test.revisionPrefix) {
				t.Fatalf("resolved=%+v found=%v err=%v", resolved, found, err)
			}
		})
	}
}

func TestWaldoAttemptContextRevisionChangesWithLifecycleTruth(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	projectID := domain.ProjectID("waldo-attempt-lifecycle")
	plan, outcomeID := seedApprovedPlan(t, s, string(projectID))
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	attempt, err := s.CreateAttemptWithFence(ctx, outcomeID, plan, "attempt-lifecycle", domain.FenceSubjectForProject(projectID), now)
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	ref := domain.WaldoContextRef{
		Kind: domain.WaldoContextAttempt, ObjectID: attempt.ID.String(), Revision: "requested",
		Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoProvenanceCanonical, SourceID: "attempt-lifecycle-test"},
	}
	before, found, err := s.ResolveWaldoContextRef(ctx, projectID, ref)
	if err != nil || !found {
		t.Fatalf("resolve queued attempt found=%v err=%v", found, err)
	}
	rows, err := s.TransitionAttemptStatus(ctx, outcomeID, attempt.ID, domain.AttemptQueued, domain.AttemptRunning, now.Add(time.Minute))
	if err != nil || rows != 1 {
		t.Fatalf("transition attempt rows=%d err=%v", rows, err)
	}
	after, found, err := s.ResolveWaldoContextRef(ctx, projectID, ref)
	if err != nil || !found {
		t.Fatalf("resolve running attempt found=%v err=%v", found, err)
	}
	if before.Revision == after.Revision || !strings.Contains(before.Revision, ":queued:") || !strings.Contains(after.Revision, ":running:") {
		t.Fatalf("attempt lifecycle revisions before=%q after=%q", before.Revision, after.Revision)
	}
}

func TestWaldoContinuationOperationClaimIsDurableAndIdempotentBeforeEffects(t *testing.T) {
	dataDir := t.TempDir()
	s, err := sqlitetest.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	ctx := context.Background()
	projectID := domain.ProjectID("waldo-continuation-claim")
	plan, outcomeID := seedApprovedPlan(t, s, string(projectID))
	now := time.Date(2026, 8, 26, 10, 30, 0, 0, time.UTC)
	attempt, err := s.CreateAttemptWithFence(ctx, outcomeID, plan, "claim-attempt", domain.FenceSubjectForProject(projectID), now)
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	source, err := s.BindAttemptSession(ctx, attemptSession(attempt.ID, "claim-source"))
	if err != nil {
		t.Fatalf("bind source: %v", err)
	}
	conversation := domain.WaldoConversation{ID: "waldo-claim-conversation", ProjectID: projectID, CreatedAt: now, UpdatedAt: now}
	if _, err := s.EnsureWaldoConversation(ctx, conversation); err != nil {
		t.Fatalf("ensure conversation: %v", err)
	}
	episode := domain.WaldoConversationEpisode{ID: "waldo-claim-episode", ConversationID: conversation.ID, ProjectID: projectID, Ordinal: 1, State: domain.WaldoEpisodeActive, CreatedAt: now}
	if _, err := s.OpenWaldoEpisode(ctx, episode, ports.WaldoIdempotency{Key: "claim-episode", Fingerprint: "claim-episode-fp"}, 0); err != nil {
		t.Fatalf("open episode: %v", err)
	}
	bindings := continuationBindings(string(projectID), outcomeID, plan, attempt.ID)
	operation := domain.WaldoContinuationOperation{
		ID: "waldo-operation-1", ConversationID: conversation.ID, ProjectID: projectID,
		FromEpisodeID: episode.ID, FromAgentSessionRef: source.ID,
		ExpectedConversationRevision: 1, State: domain.WaldoContinuationPrepared,
		Reason: domain.ContinuationReasonContextReserve, ReasonDetail: "Trustworthy provider reserve crossed.",
		TriggerEvidence: domain.ContinuationTriggerEvidence{Kind: domain.ContinuationEvidenceProviderContextMeter, Reference: "usage-fact-claim"},
		MaterialChange:  true, LostMaterialContext: true,
		ContextDigest: strings.Repeat("a", 64), PreviousBindings: bindings, ReplacementBindings: bindings,
		EffectsKnown: true, TriggerConfirmed: true,
		RequestKey: "continuation-claim", RequestFingerprint: "continuation-claim-fp",
		CreatedAt: now, UpdatedAt: now,
	}
	otherPlan, otherOutcomeID := seedApprovedPlan(t, s, "waldo-continuation-other")
	otherAttempt, err := s.CreateAttemptWithFence(ctx, otherOutcomeID, otherPlan, "other-attempt", domain.FenceSubjectForProject("waldo-continuation-other"), now)
	if err != nil {
		t.Fatalf("create other attempt: %v", err)
	}
	otherSource, err := s.BindAttemptSession(ctx, attemptSession(otherAttempt.ID, "other-source"))
	if err != nil {
		t.Fatalf("bind other source: %v", err)
	}
	crossProject := operation
	crossProject.ID = "waldo-operation-cross-project"
	crossProject.FromAgentSessionRef = otherSource.ID
	crossProject.RequestKey = "continuation-cross-project"
	crossProject.RequestFingerprint = "continuation-cross-project-fp"
	if _, _, err := s.ClaimWaldoContinuationOperation(ctx, crossProject); err == nil {
		t.Fatal("cross-Project source session was accepted for continuation")
	}
	first, claimed, err := s.ClaimWaldoContinuationOperation(ctx, operation)
	if err != nil || !claimed || first.ID != operation.ID || first.State != domain.WaldoContinuationPrepared {
		t.Fatalf("first claim=%+v claimed=%v err=%v", first, claimed, err)
	}
	replay, claimed, err := s.ClaimWaldoContinuationOperation(ctx, operation)
	if err != nil || claimed || replay.ID != first.ID {
		t.Fatalf("replay claim=%+v claimed=%v err=%v", replay, claimed, err)
	}
	changed := operation
	changed.RequestFingerprint = "different-input"
	if _, _, err := s.ClaimWaldoContinuationOperation(ctx, changed); err == nil {
		t.Fatal("same continuation key with changed fingerprint did not conflict")
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}
	reopened, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	restored, found, err := reopened.FindWaldoContinuationOperationByRequestKey(ctx, operation.RequestKey)
	if err != nil || !found || restored.ID != operation.ID || restored.State != domain.WaldoContinuationPrepared ||
		!restored.LostMaterialContext || restored.SourceRevoked || restored.FreshVerifier || !restored.TriggerConfirmed {
		t.Fatalf("restored operation=%+v found=%v err=%v", restored, found, err)
	}
	events, err := reopened.EventsAfter(ctx, 0, 100)
	if err != nil {
		t.Fatalf("read CDC: %v", err)
	}
	seenPrepared := false
	for _, event := range events {
		if event.Type == "waldo_conversation_continuation_prepared" {
			seenPrepared = true
		}
	}
	if !seenPrepared {
		t.Fatal("trigger CDC did not record durable continuation preparation")
	}
}

func attemptSession(attemptID domain.AttemptID, sessionID string) domain.AttemptSessionRef {
	return domain.AttemptSessionRef{
		AttemptID: attemptID, SessionID: sessionID, Harness: domain.HarnessCodex, Mode: domain.SessionModeTUI,
		RunBriefCoreDigest: strings.Repeat("b", 64), RunBriefCompiledDigest: strings.Repeat("c", 64),
		AdmissionSnapshot: `{"snapshotVersion":1}`,
	}
}

func continuationBindings(projectID string, outcomeID domain.OutcomeID, plan domain.PlanRevision, attemptID domain.AttemptID) domain.ContinuationBindings {
	return domain.ContinuationBindings{
		ProjectID: domain.ProjectID(projectID), OutcomeID: outcomeID,
		ContractRevisionID: domain.ContractRevisionID("cr-" + projectID), PlanRevisionID: plan.ID,
		WorkUnitID: plan.WorkUnits[0].ID, AttemptID: attemptID, Provider: domain.HarnessCodex,
		Model: "gpt-5", Profile: "balanced", Role: "implementer",
		AuthorityDigest: strings.Repeat("d", 64), BudgetDigest: strings.Repeat("e", 64),
		WorkspaceOwner: "waldo", EffectPolicyDigest: strings.Repeat("f", 64),
	}
}

func claimContinuationOperation(t *testing.T, s *sqlite.Store, receipt domain.ContinuationReceipt, request ports.WaldoIdempotency, expectedRevision int64, at time.Time) {
	t.Helper()
	operation := domain.WaldoContinuationOperation{
		ID: receipt.OperationID, ConversationID: receipt.ConversationID, ProjectID: receipt.ProjectID,
		FromEpisodeID: receipt.FromEpisodeID, FromAgentSessionRef: receipt.FromAgentSessionRef,
		ExpectedConversationRevision: expectedRevision, State: domain.WaldoContinuationPrepared,
		Reason: receipt.Reason, ReasonDetail: receipt.ReasonDetail, TriggerEvidence: receipt.TriggerEvidence, MaterialChange: receipt.MaterialChange,
		ChangedFields: receipt.ChangedFields, ContextDigest: receipt.ContextDigest, ContextRefs: receipt.ContextRefs,
		PreviousBindings: receipt.PreviousBindings, ReplacementBindings: receipt.ReplacementBindings,
		EffectsKnown: receipt.EffectsKnown, RequestKey: request.Key, RequestFingerprint: request.Fingerprint,
		CreatedAt: at, UpdatedAt: at,
	}
	if _, claimed, err := s.ClaimWaldoContinuationOperation(context.Background(), operation); err != nil || !claimed {
		t.Fatalf("claim continuation operation claimed=%v err=%v", claimed, err)
	}
}

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/gen"
)

// EnsureWaldoConversation returns the existing Project aggregate or creates it.
func (s *Store) EnsureWaldoConversation(ctx context.Context, conversation domain.WaldoConversation) (ports.WaldoConversationSnapshot, error) {
	if err := conversation.Validate(); err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	row, err := s.qw.GetWaldoConversationByProject(ctx, string(conversation.ProjectID))
	if err == nil {
		return s.waldoConversationSnapshot(ctx, s.qw, row)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ports.WaldoConversationSnapshot{}, fmt.Errorf("find Project Waldo conversation: %w", err)
	}
	if err := s.qw.CreateWaldoConversation(ctx, gen.CreateWaldoConversationParams{
		ID: conversation.ID.String(), ProjectID: string(conversation.ProjectID),
		Revision: conversation.Revision, LatestTurnSequence: conversation.LatestTurnSequence,
		CreatedAt: conversation.CreatedAt, UpdatedAt: conversation.UpdatedAt,
	}); err != nil {
		return ports.WaldoConversationSnapshot{}, fmt.Errorf("create Project Waldo conversation: %w", err)
	}
	return s.waldoConversationSnapshot(ctx, s.qw, waldoConversationRow(conversation))
}

// GetWaldoConversationByProject restores the complete aggregate from SQLite.
func (s *Store) GetWaldoConversationByProject(ctx context.Context, projectID domain.ProjectID) (ports.WaldoConversationSnapshot, bool, error) {
	row, err := s.qr.GetWaldoConversationByProject(ctx, string(projectID))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.WaldoConversationSnapshot{}, false, nil
	}
	if err != nil {
		return ports.WaldoConversationSnapshot{}, false, fmt.Errorf("get Project Waldo conversation: %w", err)
	}
	snapshot, err := s.waldoConversationSnapshot(ctx, s.qr, row)
	return snapshot, err == nil, err
}

// OpenWaldoEpisode appends one bounded episode under aggregate revision CAS.
func (s *Store) OpenWaldoEpisode(ctx context.Context, episode domain.WaldoConversationEpisode, request ports.WaldoIdempotency, expectedRevision int64) (ports.WaldoConversationSnapshot, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if replay, err := s.qw.FindWaldoEpisodeByRequestKey(ctx, request.Key); err == nil {
		if replay.RequestFingerprint != request.Fingerprint {
			return ports.WaldoConversationSnapshot{}, &ports.WaldoIdempotencyConflictError{Key: request.Key}
		}
		conversation, err := s.qw.GetWaldoConversationByID(ctx, replay.ConversationID)
		if err != nil {
			return ports.WaldoConversationSnapshot{}, err
		}
		return s.waldoConversationSnapshot(ctx, s.qw, conversation)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ports.WaldoConversationSnapshot{}, fmt.Errorf("find Waldo episode replay: %w", err)
	}
	if err := episode.Validate(); err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	conversation, err := s.qw.GetWaldoConversationByID(ctx, episode.ConversationID.String())
	if err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	if err := checkWaldoRevision(conversation, expectedRevision); err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	err = s.inTx(ctx, "open Waldo episode", func(q *gen.Queries) error {
		if err := q.CreateWaldoConversationEpisode(ctx, waldoEpisodeInsert(episode, request)); err != nil {
			return err
		}
		rows, err := q.AdvanceWaldoConversationRevision(ctx, gen.AdvanceWaldoConversationRevisionParams{
			UpdatedAt: episode.CreatedAt, ID: episode.ConversationID.String(), Revision: expectedRevision,
		})
		if err != nil {
			return err
		}
		if rows != 1 {
			return currentWaldoRevisionConflict(ctx, q, episode.ConversationID, expectedRevision)
		}
		return nil
	})
	if err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	conversation.Revision++
	conversation.UpdatedAt = episode.CreatedAt
	return s.waldoConversationSnapshot(ctx, s.qw, conversation)
}

// FindWaldoTurnByRequestKey supports a precondition-free replay check before
// service policy observes later episode or revision state.
func (s *Store) FindWaldoTurnByRequestKey(ctx context.Context, requestKey string) (domain.WaldoConversationTurn, string, bool, error) {
	row, err := s.qr.FindWaldoTurnByRequestKey(ctx, requestKey)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.WaldoConversationTurn{}, "", false, nil
	}
	if err != nil {
		return domain.WaldoConversationTurn{}, "", false, fmt.Errorf("find Waldo turn: %w", err)
	}
	turn := waldoTurnFromRow(row)
	refs, err := s.qr.ListWaldoTurnContextRefs(ctx, row.ConversationID)
	if err != nil {
		return domain.WaldoConversationTurn{}, "", false, fmt.Errorf("load replayed Waldo turn context: %w", err)
	}
	for _, ref := range refs {
		if ref.TurnID != row.ID {
			continue
		}
		turn.ContextRefs = append(turn.ContextRefs, domain.WaldoContextRef{
			Kind: domain.WaldoContextRefKind(ref.Kind), ObjectID: ref.ObjectID, Revision: ref.ObjectRevision,
			Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoContextProvenanceKind(ref.ProvenanceKind), SourceID: ref.ProvenanceRef},
		})
	}
	return turn, row.RequestFingerprint, true, nil
}

// AppendWaldoTurn atomically selects active attachments, allocates the next
// order, appends the turn, and advances the aggregate revision.
func (s *Store) AppendWaldoTurn(ctx context.Context, turn domain.WaldoConversationTurn, attachmentIDs []domain.WaldoContextAttachmentID, request ports.WaldoIdempotency, expectedRevision int64) (ports.WaldoConversationSnapshot, domain.WaldoConversationTurn, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if replay, err := s.qw.FindWaldoTurnByRequestKey(ctx, request.Key); err == nil {
		if replay.RequestFingerprint != request.Fingerprint {
			return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, &ports.WaldoIdempotencyConflictError{Key: request.Key}
		}
		conversation, err := s.qw.GetWaldoConversationByID(ctx, replay.ConversationID)
		if err != nil {
			return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, err
		}
		snapshot, err := s.waldoConversationSnapshot(ctx, s.qw, conversation)
		if err != nil {
			return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, err
		}
		for _, stored := range snapshot.Turns {
			if stored.ID.String() == replay.ID {
				return snapshot, stored, nil
			}
		}
		return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, fmt.Errorf("replayed Waldo turn %s missing from snapshot", replay.ID)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, fmt.Errorf("find Waldo turn replay: %w", err)
	}
	conversation, err := s.qw.GetWaldoConversationByID(ctx, turn.ConversationID.String())
	if err != nil {
		return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, err
	}
	if err := checkWaldoRevision(conversation, expectedRevision); err != nil {
		return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, err
	}
	attachments := make([]gen.WaldoContextAttachment, 0, len(attachmentIDs))
	turn.ContextRefs = make([]domain.WaldoContextRef, 0, len(attachmentIDs))
	for _, id := range attachmentIDs {
		row, err := s.qw.GetWaldoContextAttachment(ctx, gen.GetWaldoContextAttachmentParams{ID: id.String(), ConversationID: turn.ConversationID.String()})
		if err != nil {
			return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, fmt.Errorf("select Waldo turn context %s: %w", id, err)
		}
		if row.DetachedAt.Valid {
			return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, fmt.Errorf("waldo context attachment %s is detached", id)
		}
		ref := waldoContextRefFromRow(row)
		current, found, err := s.ResolveWaldoContextRef(ctx, turn.ProjectID, ref)
		if err != nil {
			return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, err
		}
		if !found {
			return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, fmt.Errorf("waldo context source %s %s is unavailable", ref.Kind, ref.ObjectID)
		}
		if current.Revision != ref.Revision {
			return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, &ports.WaldoContextRevisionConflictError{
				Kind: ref.Kind, ObjectID: ref.ObjectID, RequestedRevision: ref.Revision, CurrentRevision: current.Revision,
			}
		}
		attachments = append(attachments, row)
		turn.ContextRefs = append(turn.ContextRefs, current)
	}
	if err := turn.ValidateFor(waldoConversationFromRow(conversation)); err != nil {
		return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, err
	}
	err = s.inTx(ctx, "append Waldo turn", func(q *gen.Queries) error {
		if err := q.CreateWaldoConversationTurn(ctx, waldoTurnInsert(turn, request)); err != nil {
			return err
		}
		for index, attachment := range attachments {
			if err := q.CreateWaldoTurnContextRef(ctx, gen.CreateWaldoTurnContextRefParams{
				TurnID: turn.ID.String(), AttachmentID: attachment.ID, Position: int64(index + 1),
			}); err != nil {
				return err
			}
		}
		rows, err := q.AdvanceWaldoConversationTurn(ctx, gen.AdvanceWaldoConversationTurnParams{
			LatestTurnSequence: turn.Sequence, UpdatedAt: turn.CreatedAt, ID: turn.ConversationID.String(),
			Revision: expectedRevision, LatestTurnSequence_2: turn.Sequence - 1,
		})
		if err != nil {
			return err
		}
		if rows != 1 {
			return currentWaldoRevisionConflict(ctx, q, turn.ConversationID, expectedRevision)
		}
		return nil
	})
	if err != nil {
		return ports.WaldoConversationSnapshot{}, domain.WaldoConversationTurn{}, err
	}
	conversation.Revision++
	conversation.LatestTurnSequence = turn.Sequence
	conversation.UpdatedAt = turn.CreatedAt
	snapshot, err := s.waldoConversationSnapshot(ctx, s.qw, conversation)
	return snapshot, turn, err
}

// ResolveWaldoContextRef resolves current canonical revision and Project ownership.
func (s *Store) ResolveWaldoContextRef(ctx context.Context, projectID domain.ProjectID, ref domain.WaldoContextRef) (domain.WaldoContextRef, bool, error) {
	type resolved struct {
		id, revision string
		projectID    domain.ProjectID
	}
	var value resolved
	var err error
	switch ref.Kind {
	case domain.WaldoContextProject:
		var row gen.ResolveWaldoProjectContextRow
		row, err = s.qr.ResolveWaldoProjectContext(ctx, domain.ProjectID(ref.ObjectID))
		value = resolved{id: string(row.ObjectID), revision: row.ObjectRevision, projectID: row.ProjectID}
	case domain.WaldoContextOutcome:
		var row gen.ResolveWaldoOutcomeContextRow
		row, err = s.qr.ResolveWaldoOutcomeContext(ctx, domain.OutcomeID(ref.ObjectID))
		value = resolved{id: row.ObjectID.String(), revision: outcomeLifecycleRevision(row.RevisionNumber, row.LifecycleUpdatedAt), projectID: row.ProjectID}
	case domain.WaldoContextContractRevision:
		var row gen.ResolveWaldoContractRevisionContextRow
		row, err = s.qr.ResolveWaldoContractRevisionContext(ctx, domain.ContractRevisionID(ref.ObjectID))
		value = resolved{id: row.ObjectID.String(), revision: row.ObjectRevision, projectID: row.ProjectID}
	case domain.WaldoContextPlanRevision:
		var row gen.ResolveWaldoPlanRevisionContextRow
		row, err = s.qr.ResolveWaldoPlanRevisionContext(ctx, domain.PlanRevisionID(ref.ObjectID))
		value = resolved{id: row.ObjectID.String(), revision: planLifecycleRevision(row.RevisionNumber, row.LifecycleState, row.RunBriefCoreDigest, row.RunBriefCompiledDigest), projectID: row.ProjectID}
	case domain.WaldoContextWorkUnit:
		var row gen.ResolveWaldoWorkUnitContextRow
		row, err = s.qr.ResolveWaldoWorkUnitContext(ctx, domain.WorkUnitID(ref.ObjectID))
		value = resolved{id: row.ObjectID.String(), revision: planLifecycleRevision(row.RevisionNumber, row.LifecycleState, row.RunBriefCoreDigest, row.RunBriefCompiledDigest), projectID: row.ProjectID}
	case domain.WaldoContextAttempt:
		var row gen.ResolveWaldoAttemptContextRow
		row, err = s.qr.ResolveWaldoAttemptContext(ctx, domain.AttemptID(ref.ObjectID))
		value = resolved{id: row.ObjectID.String(), revision: mutableLifecycleRevision(row.RevisionNumber, string(row.LifecycleState), row.LifecycleUpdatedAt), projectID: row.ProjectID}
	case domain.WaldoContextAgentSessionRef:
		var row gen.ResolveWaldoAgentSessionContextRow
		row, err = s.qr.ResolveWaldoAgentSessionContext(ctx, ref.ObjectID)
		value = resolved{id: row.ObjectID, revision: mutableLifecycleRevision(row.RevisionNumber, string(row.LifecycleState), row.LifecycleUpdatedAt), projectID: row.ProjectID}
	case domain.WaldoContextIntakeSession:
		var row gen.ResolveWaldoIntakeSessionContextRow
		row, err = s.qr.ResolveWaldoIntakeSessionContext(ctx, ref.ObjectID)
		value = resolved{id: row.ObjectID, revision: mutableLifecycleRevision(row.RevisionNumber, row.LifecycleState, row.LifecycleUpdatedAt), projectID: domain.ProjectID(row.ProjectID.String)}
	default:
		return domain.WaldoContextRef{}, false, fmt.Errorf("unsupported Waldo context kind %q", ref.Kind)
	}
	if errors.Is(err, sql.ErrNoRows) || (err == nil && value.projectID != projectID) {
		return domain.WaldoContextRef{}, false, nil
	}
	if err != nil {
		return domain.WaldoContextRef{}, false, fmt.Errorf("resolve Waldo context %s %s: %w", ref.Kind, ref.ObjectID, err)
	}
	ref.ObjectID, ref.Revision = value.id, value.revision
	return ref, true, nil
}

func outcomeLifecycleRevision(number int64, updatedAt time.Time) string {
	return fmt.Sprintf("%d:%s", number, updatedAt.UTC().Format(time.RFC3339Nano))
}

func planLifecycleRevision(number int64, state, coreDigest, compiledDigest string) string {
	return fmt.Sprintf("%d:%s:%s:%s", number, state, coreDigest, compiledDigest)
}

func mutableLifecycleRevision(number int64, state string, updatedAt time.Time) string {
	return fmt.Sprintf("%d:%s:%s", number, state, updatedAt.UTC().Format(time.RFC3339Nano))
}

// AttachWaldoContext appends one explicit provenance-bearing attachment.
func (s *Store) AttachWaldoContext(ctx context.Context, attachment domain.WaldoContextAttachment, request ports.WaldoIdempotency, expectedRevision int64) (ports.WaldoConversationSnapshot, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if replay, err := s.qw.FindWaldoContextByAttachRequestKey(ctx, request.Key); err == nil {
		if replay.AttachRequestFingerprint != request.Fingerprint {
			return ports.WaldoConversationSnapshot{}, &ports.WaldoIdempotencyConflictError{Key: request.Key}
		}
		conversation, err := s.qw.GetWaldoConversationByID(ctx, replay.ConversationID)
		if err != nil {
			return ports.WaldoConversationSnapshot{}, err
		}
		return s.waldoConversationSnapshot(ctx, s.qw, conversation)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ports.WaldoConversationSnapshot{}, fmt.Errorf("find Waldo context replay: %w", err)
	}
	if err := attachment.Validate(); err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	conversation, err := s.qw.GetWaldoConversationByID(ctx, attachment.ConversationID.String())
	if err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	if err := checkWaldoRevision(conversation, expectedRevision); err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	if attachment.AttachedRevision != expectedRevision+1 {
		return ports.WaldoConversationSnapshot{}, fmt.Errorf("waldo context attached revision %d is not next after %d", attachment.AttachedRevision, expectedRevision)
	}
	current, found, err := s.ResolveWaldoContextRef(ctx, attachment.ProjectID, attachment.Ref)
	if err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	if !found {
		return ports.WaldoConversationSnapshot{}, fmt.Errorf("waldo context source %s %s is unavailable", attachment.Ref.Kind, attachment.Ref.ObjectID)
	}
	if current.Revision != attachment.Ref.Revision {
		return ports.WaldoConversationSnapshot{}, &ports.WaldoContextRevisionConflictError{
			Kind: attachment.Ref.Kind, ObjectID: attachment.Ref.ObjectID,
			RequestedRevision: attachment.Ref.Revision, CurrentRevision: current.Revision,
		}
	}
	err = s.inTx(ctx, "attach Waldo context", func(q *gen.Queries) error {
		if err := q.CreateWaldoContextAttachment(ctx, waldoAttachmentInsert(attachment, request)); err != nil {
			return err
		}
		rows, err := q.AdvanceWaldoConversationRevision(ctx, gen.AdvanceWaldoConversationRevisionParams{UpdatedAt: attachment.CreatedAt, ID: attachment.ConversationID.String(), Revision: expectedRevision})
		if err != nil {
			return err
		}
		if rows != 1 {
			return currentWaldoRevisionConflict(ctx, q, attachment.ConversationID, expectedRevision)
		}
		return nil
	})
	if err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	conversation.Revision++
	conversation.UpdatedAt = attachment.CreatedAt
	return s.waldoConversationSnapshot(ctx, s.qw, conversation)
}

// DetachWaldoContext records one concurrency-safe explicit detach.
func (s *Store) DetachWaldoContext(ctx context.Context, conversationID domain.WaldoConversationID, attachmentID domain.WaldoContextAttachmentID, reason string, request ports.WaldoIdempotency, expectedRevision int64, at time.Time) (ports.WaldoConversationSnapshot, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if replay, err := s.qw.FindWaldoContextByDetachRequestKey(ctx, sql.NullString{String: request.Key, Valid: true}); err == nil {
		if replay.DetachRequestFingerprint != request.Fingerprint {
			return ports.WaldoConversationSnapshot{}, &ports.WaldoIdempotencyConflictError{Key: request.Key}
		}
		conversation, err := s.qw.GetWaldoConversationByID(ctx, replay.ConversationID)
		if err != nil {
			return ports.WaldoConversationSnapshot{}, err
		}
		return s.waldoConversationSnapshot(ctx, s.qw, conversation)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ports.WaldoConversationSnapshot{}, fmt.Errorf("find Waldo detach replay: %w", err)
	}
	conversation, err := s.qw.GetWaldoConversationByID(ctx, conversationID.String())
	if err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	if err := checkWaldoRevision(conversation, expectedRevision); err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	if _, err := s.qw.GetWaldoContextAttachment(ctx, gen.GetWaldoContextAttachmentParams{ID: attachmentID.String(), ConversationID: conversationID.String()}); err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	err = s.inTx(ctx, "detach Waldo context", func(q *gen.Queries) error {
		rows, err := q.DetachWaldoContextAttachment(ctx, gen.DetachWaldoContextAttachmentParams{
			DetachedRevision: sql.NullInt64{Int64: expectedRevision + 1, Valid: true},
			DetachedAt:       sql.NullTime{Time: at, Valid: true}, DetachReason: reason,
			DetachRequestKey: sql.NullString{String: request.Key, Valid: true}, DetachRequestFingerprint: request.Fingerprint,
			ID: attachmentID.String(), ConversationID: conversationID.String(),
		})
		if err != nil {
			return err
		}
		if rows != 1 {
			return fmt.Errorf("waldo context attachment %s is already detached", attachmentID)
		}
		rows, err = q.AdvanceWaldoConversationRevision(ctx, gen.AdvanceWaldoConversationRevisionParams{UpdatedAt: at, ID: conversationID.String(), Revision: expectedRevision})
		if err != nil {
			return err
		}
		if rows != 1 {
			return currentWaldoRevisionConflict(ctx, q, conversationID, expectedRevision)
		}
		return nil
	})
	if err != nil {
		return ports.WaldoConversationSnapshot{}, err
	}
	conversation.Revision++
	conversation.UpdatedAt = at
	return s.waldoConversationSnapshot(ctx, s.qw, conversation)
}

// ClaimWaldoContinuationOperation atomically claims the active source episode
// before any fencing or provider-start effect. claimed=false is a same-input replay.
func (s *Store) ClaimWaldoContinuationOperation(ctx context.Context, operation domain.WaldoContinuationOperation) (domain.WaldoContinuationOperation, bool, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if replay, err := s.qw.FindWaldoContinuationOperationByRequestKey(ctx, operation.RequestKey); err == nil {
		if replay.RequestFingerprint != operation.RequestFingerprint {
			return domain.WaldoContinuationOperation{}, false, &ports.WaldoIdempotencyConflictError{Key: operation.RequestKey}
		}
		stored, err := waldoContinuationOperationFromRow(replay)
		return stored, false, err
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.WaldoContinuationOperation{}, false, fmt.Errorf("find Waldo continuation operation replay: %w", err)
	}
	if operation.State != domain.WaldoContinuationPrepared {
		return domain.WaldoContinuationOperation{}, false, fmt.Errorf("new continuation operation must be prepared")
	}
	if err := operation.Validate(); err != nil {
		return domain.WaldoContinuationOperation{}, false, err
	}
	if err := validateWaldoContinuationSessionLineage(ctx, s.qr, operation.FromAgentSessionRef, operation.PreviousBindings); err != nil {
		return domain.WaldoContinuationOperation{}, false, err
	}
	params, err := waldoContinuationOperationInsert(operation)
	if err != nil {
		return domain.WaldoContinuationOperation{}, false, err
	}
	if err := s.qw.CreateWaldoContinuationOperation(ctx, params); err != nil {
		return domain.WaldoContinuationOperation{}, false, fmt.Errorf("claim Waldo continuation operation: %w", err)
	}
	return operation, true, nil
}

// FindWaldoContinuationOperationByRequestKey restores durable pre-effect truth.
func (s *Store) FindWaldoContinuationOperationByRequestKey(ctx context.Context, requestKey string) (domain.WaldoContinuationOperation, bool, error) {
	row, err := s.qr.FindWaldoContinuationOperationByRequestKey(ctx, requestKey)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.WaldoContinuationOperation{}, false, nil
	}
	if err != nil {
		return domain.WaldoContinuationOperation{}, false, fmt.Errorf("find Waldo continuation operation: %w", err)
	}
	operation, err := waldoContinuationOperationFromRow(row)
	return operation, err == nil, err
}

// AdvanceWaldoContinuationOperation records a lifecycle boundary before or
// after its corresponding external effect.
func (s *Store) AdvanceWaldoContinuationOperation(ctx context.Context, id string, expected, next domain.WaldoContinuationOperationState, fenceRef, reconciliationRef, needsUserReason string, at time.Time) (domain.WaldoContinuationOperation, error) {
	if !expected.CanTransitionTo(next) {
		return domain.WaldoContinuationOperation{}, fmt.Errorf("illegal continuation operation transition %s -> %s", expected, next)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.qw.AdvanceWaldoContinuationOperation(ctx, gen.AdvanceWaldoContinuationOperationParams{
		State: string(next), FenceReceiptRef: fenceRef, ReconciliationRef: reconciliationRef,
		NeedsUserReason: needsUserReason, UpdatedAt: at, ID: id, State_2: string(expected),
	})
	if err != nil {
		return domain.WaldoContinuationOperation{}, fmt.Errorf("advance Waldo continuation operation: %w", err)
	}
	if rows != 1 {
		return domain.WaldoContinuationOperation{}, fmt.Errorf("continuation operation %s is not in state %s", id, expected)
	}
	row, err := s.qw.GetWaldoContinuationOperation(ctx, id)
	if err != nil {
		return domain.WaldoContinuationOperation{}, err
	}
	return waldoContinuationOperationFromRow(row)
}

// ListPendingWaldoContinuationOperations returns every restart-recovery claim.
func (s *Store) ListPendingWaldoContinuationOperations(ctx context.Context) ([]domain.WaldoContinuationOperation, error) {
	rows, err := s.qr.ListPendingWaldoContinuationOperations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pending Waldo continuation operations: %w", err)
	}
	operations := make([]domain.WaldoContinuationOperation, 0, len(rows))
	for _, row := range rows {
		operation, err := waldoContinuationOperationFromRow(row)
		if err != nil {
			return nil, err
		}
		operations = append(operations, operation)
	}
	return operations, nil
}

// FindContinuationReceiptByRequestKey performs the pre-effect idempotency check.
func (s *Store) FindContinuationReceiptByRequestKey(ctx context.Context, requestKey string) (domain.ContinuationReceipt, string, bool, error) {
	row, err := s.qr.FindWaldoContinuationReceiptByRequestKey(ctx, requestKey)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ContinuationReceipt{}, "", false, nil
	}
	if err != nil {
		return domain.ContinuationReceipt{}, "", false, fmt.Errorf("find Waldo continuation receipt: %w", err)
	}
	receipt, err := waldoContinuationFromRow(row)
	return receipt, row.RequestFingerprint, err == nil, err
}

// RecordContinuationReceipt atomically seals the safely fenced predecessor,
// optionally opens the confirmed replacement episode, and appends its receipt.
func (s *Store) RecordContinuationReceipt(ctx context.Context, receipt domain.ContinuationReceipt, replacement *domain.WaldoConversationEpisode, request ports.WaldoIdempotency) (domain.ContinuationReceipt, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if replay, err := s.qw.FindWaldoContinuationReceiptByRequestKey(ctx, request.Key); err == nil {
		if replay.RequestFingerprint != request.Fingerprint {
			return domain.ContinuationReceipt{}, &ports.WaldoIdempotencyConflictError{Key: request.Key}
		}
		return waldoContinuationFromRow(replay)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.ContinuationReceipt{}, fmt.Errorf("find Waldo continuation replay: %w", err)
	}
	if err := receipt.Validate(); err != nil {
		return domain.ContinuationReceipt{}, err
	}
	operationRow, err := s.qw.GetWaldoContinuationOperation(ctx, receipt.OperationID)
	if err != nil {
		return domain.ContinuationReceipt{}, fmt.Errorf("load Waldo continuation operation: %w", err)
	}
	operation, err := waldoContinuationOperationFromRow(operationRow)
	if err != nil {
		return domain.ContinuationReceipt{}, err
	}
	if !operation.State.CanTransitionTo(domain.WaldoContinuationCompleted) ||
		operation.RequestKey != request.Key || operation.RequestFingerprint != request.Fingerprint ||
		operation.ConversationID != receipt.ConversationID || operation.ProjectID != receipt.ProjectID ||
		operation.FromEpisodeID != receipt.FromEpisodeID || operation.FromAgentSessionRef != receipt.FromAgentSessionRef {
		return domain.ContinuationReceipt{}, fmt.Errorf("continuation receipt does not match its durable operation claim")
	}
	if receipt.Action == domain.ContinuationAutomatic {
		if replacement == nil {
			return domain.ContinuationReceipt{}, fmt.Errorf("automatic continuation requires a replacement episode")
		}
		if err := replacement.Validate(); err != nil {
			return domain.ContinuationReceipt{}, err
		}
		if replacement.ID != receipt.ToEpisodeID {
			return domain.ContinuationReceipt{}, fmt.Errorf("continuation replacement episode does not match receipt")
		}
		if err := validateWaldoContinuationSessionLineage(ctx, s.qr, receipt.ToAgentSessionRef, receipt.ReplacementBindings); err != nil {
			return domain.ContinuationReceipt{}, err
		}
	} else if replacement != nil {
		return domain.ContinuationReceipt{}, fmt.Errorf("non-automatic continuation cannot open a replacement episode")
	}
	conversation, err := s.qw.GetWaldoConversationByID(ctx, receipt.ConversationID.String())
	if err != nil {
		return domain.ContinuationReceipt{}, err
	}
	err = s.inTx(ctx, "record Waldo continuation", func(q *gen.Queries) error {
		if receipt.OldSessionFenced {
			rows, err := q.SealWaldoConversationEpisode(ctx, gen.SealWaldoConversationEpisodeParams{
				SealedAt: sql.NullTime{Time: receipt.CreatedAt, Valid: true}, SealReason: string(receipt.Reason),
				ID: receipt.FromEpisodeID.String(), ConversationID: receipt.ConversationID.String(),
			})
			if err != nil {
				return err
			}
			if rows != 1 {
				return fmt.Errorf("waldo predecessor episode %s is not active", receipt.FromEpisodeID)
			}
		}
		if replacement != nil {
			if err := q.CreateWaldoConversationEpisode(ctx, waldoEpisodeInsert(*replacement, ports.WaldoIdempotency{Key: request.Key + ":episode", Fingerprint: request.Fingerprint})); err != nil {
				return err
			}
		}
		params, err := waldoContinuationInsert(receipt, request)
		if err != nil {
			return err
		}
		if err := q.CreateWaldoContinuationReceipt(ctx, params); err != nil {
			return err
		}
		rows, err := q.AdvanceWaldoContinuationOperation(ctx, gen.AdvanceWaldoContinuationOperationParams{
			State: string(domain.WaldoContinuationCompleted), FenceReceiptRef: receipt.FenceReceiptRef,
			ReconciliationRef: receipt.ReconciliationRef, NeedsUserReason: receipt.NeedsUserReason,
			UpdatedAt: receipt.CreatedAt, ID: receipt.OperationID, State_2: string(operation.State),
		})
		if err != nil {
			return err
		}
		if rows != 1 {
			return fmt.Errorf("continuation operation %s completion conflict", receipt.OperationID)
		}
		rows, err = q.AdvanceWaldoConversationRevision(ctx, gen.AdvanceWaldoConversationRevisionParams{
			UpdatedAt: receipt.CreatedAt, ID: receipt.ConversationID.String(), Revision: conversation.Revision,
		})
		if err != nil {
			return err
		}
		if rows != 1 {
			return currentWaldoRevisionConflict(ctx, q, receipt.ConversationID, conversation.Revision)
		}
		return nil
	})
	if err != nil {
		return domain.ContinuationReceipt{}, err
	}
	return receipt, nil
}

func validateWaldoContinuationSessionLineage(ctx context.Context, q *gen.Queries, sessionRef domain.AttemptSessionRefID, bindings domain.ContinuationBindings) error {
	row, err := q.ResolveWaldoContinuationSessionLineage(ctx, sessionRef.String())
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("continuation session ref %s has no canonical lineage", sessionRef)
	}
	if err != nil {
		return fmt.Errorf("resolve continuation session lineage %s: %w", sessionRef, err)
	}
	if row.ProjectID != bindings.ProjectID || row.OutcomeID != bindings.OutcomeID ||
		row.ContractRevisionID != bindings.ContractRevisionID || row.PlanRevisionID != bindings.PlanRevisionID ||
		row.WorkUnitID != bindings.WorkUnitID || row.AttemptID != bindings.AttemptID ||
		row.Harness != bindings.Provider {
		return fmt.Errorf("continuation session ref %s does not match canonical Project/Attempt/provider bindings", sessionRef)
	}
	return nil
}

func (s *Store) waldoConversationSnapshot(ctx context.Context, q *gen.Queries, conversation gen.WaldoConversation) (ports.WaldoConversationSnapshot, error) {
	snapshot := ports.WaldoConversationSnapshot{Conversation: waldoConversationFromRow(conversation)}
	episodes, err := q.ListWaldoConversationEpisodes(ctx, conversation.ID)
	if err != nil {
		return ports.WaldoConversationSnapshot{}, fmt.Errorf("list Waldo episodes: %w", err)
	}
	for _, row := range episodes {
		snapshot.Episodes = append(snapshot.Episodes, waldoEpisodeFromRow(row))
	}
	turns, err := q.ListWaldoConversationTurns(ctx, conversation.ID)
	if err != nil {
		return ports.WaldoConversationSnapshot{}, fmt.Errorf("list Waldo turns: %w", err)
	}
	refs, err := q.ListWaldoTurnContextRefs(ctx, conversation.ID)
	if err != nil {
		return ports.WaldoConversationSnapshot{}, fmt.Errorf("list Waldo turn context: %w", err)
	}
	refsByTurn := make(map[string][]domain.WaldoContextRef, len(turns))
	for _, row := range refs {
		refsByTurn[row.TurnID] = append(refsByTurn[row.TurnID], domain.WaldoContextRef{
			Kind: domain.WaldoContextRefKind(row.Kind), ObjectID: row.ObjectID, Revision: row.ObjectRevision,
			Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoContextProvenanceKind(row.ProvenanceKind), SourceID: row.ProvenanceRef},
		})
	}
	for _, row := range turns {
		turn := waldoTurnFromRow(row)
		turn.ContextRefs = refsByTurn[row.ID]
		snapshot.Turns = append(snapshot.Turns, turn)
	}
	attachments, err := q.ListWaldoContextAttachments(ctx, conversation.ID)
	if err != nil {
		return ports.WaldoConversationSnapshot{}, fmt.Errorf("list Waldo context attachments: %w", err)
	}
	for _, row := range attachments {
		snapshot.ContextAttachments = append(snapshot.ContextAttachments, waldoAttachmentFromRow(row))
	}
	receipts, err := q.ListWaldoContinuationReceipts(ctx, conversation.ID)
	if err != nil {
		return ports.WaldoConversationSnapshot{}, fmt.Errorf("list Waldo continuation receipts: %w", err)
	}
	for _, row := range receipts {
		receipt, err := waldoContinuationFromRow(row)
		if err != nil {
			return ports.WaldoConversationSnapshot{}, err
		}
		snapshot.ContinuationReceipts = append(snapshot.ContinuationReceipts, receipt)
	}
	return snapshot, nil
}

func checkWaldoRevision(row gen.WaldoConversation, expected int64) error {
	if row.Revision == expected {
		return nil
	}
	return &ports.WaldoConversationRevisionConflictError{
		ConversationID: domain.WaldoConversationID(row.ID), ExpectedRevision: expected, CurrentRevision: row.Revision,
	}
}

func currentWaldoRevisionConflict(ctx context.Context, q *gen.Queries, id domain.WaldoConversationID, expected int64) error {
	row, err := q.GetWaldoConversationByID(ctx, id.String())
	if err != nil {
		return err
	}
	return &ports.WaldoConversationRevisionConflictError{ConversationID: id, ExpectedRevision: expected, CurrentRevision: row.Revision}
}

func waldoConversationRow(conversation domain.WaldoConversation) gen.WaldoConversation {
	return gen.WaldoConversation{ID: conversation.ID.String(), ProjectID: string(conversation.ProjectID), Revision: conversation.Revision, LatestTurnSequence: conversation.LatestTurnSequence, CreatedAt: conversation.CreatedAt, UpdatedAt: conversation.UpdatedAt}
}

func waldoConversationFromRow(row gen.WaldoConversation) domain.WaldoConversation {
	return domain.WaldoConversation{ID: domain.WaldoConversationID(row.ID), ProjectID: domain.ProjectID(row.ProjectID), Revision: row.Revision, LatestTurnSequence: row.LatestTurnSequence, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func waldoEpisodeInsert(episode domain.WaldoConversationEpisode, request ports.WaldoIdempotency) gen.CreateWaldoConversationEpisodeParams {
	params := gen.CreateWaldoConversationEpisodeParams{
		ID: episode.ID.String(), ConversationID: episode.ConversationID.String(), ProjectID: string(episode.ProjectID),
		Ordinal: episode.Ordinal, State: string(episode.State), RequestKey: request.Key,
		RequestFingerprint: request.Fingerprint, CreatedAt: episode.CreatedAt, SealReason: episode.SealReason,
	}
	if episode.ProviderRef != nil {
		params.Provider = string(episode.ProviderRef.Provider)
		params.ProviderConversationID = episode.ProviderRef.ProviderConversationID
		params.TranscriptRef = episode.ProviderRef.TranscriptRef
	}
	if episode.SealedAt != nil {
		params.SealedAt = sql.NullTime{Time: *episode.SealedAt, Valid: true}
	}
	return params
}

func waldoEpisodeFromRow(row gen.WaldoConversationEpisode) domain.WaldoConversationEpisode {
	episode := domain.WaldoConversationEpisode{
		ID: domain.WaldoConversationEpisodeID(row.ID), ConversationID: domain.WaldoConversationID(row.ConversationID),
		ProjectID: domain.ProjectID(row.ProjectID), Ordinal: row.Ordinal, State: domain.WaldoEpisodeState(row.State),
		CreatedAt: row.CreatedAt, SealReason: row.SealReason,
	}
	if row.Provider != "" {
		episode.ProviderRef = &domain.WaldoProviderEpisodeRef{Provider: domain.AgentHarness(row.Provider), ProviderConversationID: row.ProviderConversationID, TranscriptRef: row.TranscriptRef}
	}
	if row.SealedAt.Valid {
		sealed := row.SealedAt.Time
		episode.SealedAt = &sealed
	}
	return episode
}

func waldoTurnInsert(turn domain.WaldoConversationTurn, request ports.WaldoIdempotency) gen.CreateWaldoConversationTurnParams {
	params := gen.CreateWaldoConversationTurnParams{
		ID: turn.ID.String(), ConversationID: turn.ConversationID.String(), EpisodeID: turn.EpisodeID.String(),
		ProjectID: string(turn.ProjectID), Sequence: turn.Sequence, Role: string(turn.Role), Message: turn.Message,
		RequestKey: request.Key, RequestFingerprint: request.Fingerprint, CreatedAt: turn.CreatedAt,
	}
	if turn.ProviderRef != nil {
		params.Provider = string(turn.ProviderRef.Provider)
		params.ProviderConversationID = turn.ProviderRef.ProviderConversationID
		params.ProviderTurnID = turn.ProviderRef.ProviderTurnID
		params.TranscriptRef = turn.ProviderRef.TranscriptRef
	}
	return params
}

func waldoTurnFromRow(row gen.WaldoConversationTurn) domain.WaldoConversationTurn {
	turn := domain.WaldoConversationTurn{
		ID: domain.WaldoConversationTurnID(row.ID), ConversationID: domain.WaldoConversationID(row.ConversationID),
		EpisodeID: domain.WaldoConversationEpisodeID(row.EpisodeID), ProjectID: domain.ProjectID(row.ProjectID),
		Sequence: row.Sequence, Role: domain.WaldoTurnRole(row.Role), Message: row.Message, CreatedAt: row.CreatedAt,
	}
	if row.Provider != "" {
		turn.ProviderRef = &domain.WaldoProviderTurnRef{Provider: domain.AgentHarness(row.Provider), ProviderConversationID: row.ProviderConversationID, ProviderTurnID: row.ProviderTurnID, TranscriptRef: row.TranscriptRef}
	}
	return turn
}

func waldoAttachmentInsert(attachment domain.WaldoContextAttachment, request ports.WaldoIdempotency) gen.CreateWaldoContextAttachmentParams {
	params := gen.CreateWaldoContextAttachmentParams{
		ID: attachment.ID.String(), ConversationID: attachment.ConversationID.String(), ProjectID: string(attachment.ProjectID),
		Kind: string(attachment.Ref.Kind), ObjectID: attachment.Ref.ObjectID, ObjectRevision: attachment.Ref.Revision,
		ProvenanceKind: string(attachment.Ref.Provenance.Kind), ProvenanceRef: attachment.Ref.Provenance.SourceID,
		AttachedRevision: attachment.AttachedRevision, AttachRequestKey: request.Key,
		AttachRequestFingerprint: request.Fingerprint, CreatedAt: attachment.CreatedAt, DetachReason: attachment.DetachReason,
	}
	if attachment.DetachedAt != nil {
		params.DetachedRevision = sql.NullInt64{Int64: attachment.DetachedRevision, Valid: true}
		params.DetachedAt = sql.NullTime{Time: *attachment.DetachedAt, Valid: true}
	}
	return params
}

func waldoAttachmentFromRow(row gen.WaldoContextAttachment) domain.WaldoContextAttachment {
	attachment := domain.WaldoContextAttachment{
		ID: domain.WaldoContextAttachmentID(row.ID), ConversationID: domain.WaldoConversationID(row.ConversationID),
		ProjectID: domain.ProjectID(row.ProjectID), Ref: waldoContextRefFromRow(row), AttachedRevision: row.AttachedRevision,
		CreatedAt: row.CreatedAt, DetachReason: row.DetachReason,
	}
	if row.DetachedRevision.Valid {
		attachment.DetachedRevision = row.DetachedRevision.Int64
	}
	if row.DetachedAt.Valid {
		detached := row.DetachedAt.Time
		attachment.DetachedAt = &detached
	}
	return attachment
}

func waldoContextRefFromRow(row gen.WaldoContextAttachment) domain.WaldoContextRef {
	return domain.WaldoContextRef{
		Kind: domain.WaldoContextRefKind(row.Kind), ObjectID: row.ObjectID, Revision: row.ObjectRevision,
		Provenance: domain.WaldoContextProvenance{Kind: domain.WaldoContextProvenanceKind(row.ProvenanceKind), SourceID: row.ProvenanceRef},
	}
}

func waldoContinuationOperationInsert(operation domain.WaldoContinuationOperation) (gen.CreateWaldoContinuationOperationParams, error) {
	changed, err := json.Marshal(operation.ChangedFields)
	if err != nil {
		return gen.CreateWaldoContinuationOperationParams{}, err
	}
	refs, err := json.Marshal(operation.ContextRefs)
	if err != nil {
		return gen.CreateWaldoContinuationOperationParams{}, err
	}
	previous, err := json.Marshal(operation.PreviousBindings)
	if err != nil {
		return gen.CreateWaldoContinuationOperationParams{}, err
	}
	replacement, err := json.Marshal(operation.ReplacementBindings)
	if err != nil {
		return gen.CreateWaldoContinuationOperationParams{}, err
	}
	return gen.CreateWaldoContinuationOperationParams{
		ID: operation.ID, ConversationID: operation.ConversationID.String(), ProjectID: string(operation.ProjectID),
		FromEpisodeID: operation.FromEpisodeID.String(), FromAgentSessionRefID: operation.FromAgentSessionRef.String(),
		ExpectedConversationRevision: operation.ExpectedConversationRevision, State: string(operation.State),
		Reason: string(operation.Reason), ReasonDetail: operation.ReasonDetail,
		TriggerEvidenceKind: string(operation.TriggerEvidence.Kind), TriggerEvidenceRef: operation.TriggerEvidence.Reference,
		MaterialChange: boolInt(operation.MaterialChange), ChangedFields: string(changed),
		ContextDigest: operation.ContextDigest, ContextRefs: string(refs), PreviousBindings: string(previous),
		ReplacementBindings: string(replacement), EffectsKnown: boolInt(operation.EffectsKnown),
		LostMaterialContext: boolInt(operation.LostMaterialContext), SourceRevoked: boolInt(operation.SourceRevoked),
		FreshVerifier: boolInt(operation.FreshVerifier), TriggerConfirmed: boolInt(operation.TriggerConfirmed),
		FenceReceiptRef: operation.FenceReceiptRef, ReconciliationRef: operation.ReconciliationRef,
		NeedsUserReason: operation.NeedsUserReason, RequestKey: operation.RequestKey,
		RequestFingerprint: operation.RequestFingerprint, CreatedAt: operation.CreatedAt, UpdatedAt: operation.UpdatedAt,
	}, nil
}

func waldoContinuationOperationFromRow(row gen.WaldoContinuationOperation) (domain.WaldoContinuationOperation, error) {
	operation := domain.WaldoContinuationOperation{
		ID: row.ID, ConversationID: domain.WaldoConversationID(row.ConversationID), ProjectID: domain.ProjectID(row.ProjectID),
		FromEpisodeID: domain.WaldoConversationEpisodeID(row.FromEpisodeID), FromAgentSessionRef: domain.AttemptSessionRefID(row.FromAgentSessionRefID),
		ExpectedConversationRevision: row.ExpectedConversationRevision, State: domain.WaldoContinuationOperationState(row.State),
		Reason: domain.ContinuationReason(row.Reason), ReasonDetail: row.ReasonDetail,
		TriggerEvidence: domain.ContinuationTriggerEvidence{Kind: domain.ContinuationTriggerEvidenceKind(row.TriggerEvidenceKind), Reference: row.TriggerEvidenceRef},
		MaterialChange:  row.MaterialChange != 0, ContextDigest: row.ContextDigest, EffectsKnown: row.EffectsKnown != 0,
		LostMaterialContext: row.LostMaterialContext != 0, SourceRevoked: row.SourceRevoked != 0,
		FreshVerifier: row.FreshVerifier != 0, TriggerConfirmed: row.TriggerConfirmed != 0,
		FenceReceiptRef: row.FenceReceiptRef, ReconciliationRef: row.ReconciliationRef,
		NeedsUserReason: row.NeedsUserReason, RequestKey: row.RequestKey, RequestFingerprint: row.RequestFingerprint,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if err := json.Unmarshal([]byte(row.ChangedFields), &operation.ChangedFields); err != nil {
		return domain.WaldoContinuationOperation{}, fmt.Errorf("decode Waldo continuation operation changed fields: %w", err)
	}
	if err := json.Unmarshal([]byte(row.ContextRefs), &operation.ContextRefs); err != nil {
		return domain.WaldoContinuationOperation{}, fmt.Errorf("decode Waldo continuation operation context refs: %w", err)
	}
	if err := json.Unmarshal([]byte(row.PreviousBindings), &operation.PreviousBindings); err != nil {
		return domain.WaldoContinuationOperation{}, fmt.Errorf("decode Waldo continuation operation previous bindings: %w", err)
	}
	if err := json.Unmarshal([]byte(row.ReplacementBindings), &operation.ReplacementBindings); err != nil {
		return domain.WaldoContinuationOperation{}, fmt.Errorf("decode Waldo continuation operation replacement bindings: %w", err)
	}
	return operation, nil
}

func waldoContinuationInsert(receipt domain.ContinuationReceipt, request ports.WaldoIdempotency) (gen.CreateWaldoContinuationReceiptParams, error) {
	changed, err := json.Marshal(receipt.ChangedFields)
	if err != nil {
		return gen.CreateWaldoContinuationReceiptParams{}, err
	}
	refs, err := json.Marshal(receipt.ContextRefs)
	if err != nil {
		return gen.CreateWaldoContinuationReceiptParams{}, err
	}
	previous, err := json.Marshal(receipt.PreviousBindings)
	if err != nil {
		return gen.CreateWaldoContinuationReceiptParams{}, err
	}
	replacement, err := json.Marshal(receipt.ReplacementBindings)
	if err != nil {
		return gen.CreateWaldoContinuationReceiptParams{}, err
	}
	return gen.CreateWaldoContinuationReceiptParams{
		ID: receipt.ID, OperationID: receipt.OperationID, ConversationID: receipt.ConversationID.String(), ProjectID: string(receipt.ProjectID),
		FromEpisodeID: receipt.FromEpisodeID.String(), ToEpisodeID: nullString(receipt.ToEpisodeID.String()),
		FromAgentSessionRefID: receipt.FromAgentSessionRef.String(), ToAgentSessionRefID: nullString(receipt.ToAgentSessionRef.String()),
		Action: string(receipt.Action), Reason: string(receipt.Reason), ReasonDetail: receipt.ReasonDetail,
		TriggerEvidenceKind: string(receipt.TriggerEvidence.Kind), TriggerEvidenceRef: receipt.TriggerEvidence.Reference,
		MaterialChange: boolInt(receipt.MaterialChange), ChangedFields: string(changed), ContextDigest: receipt.ContextDigest,
		ContextRefs: string(refs), PreviousBindings: string(previous), ReplacementBindings: string(replacement),
		EffectsKnown: boolInt(receipt.EffectsKnown), OldSessionFenced: boolInt(receipt.OldSessionFenced),
		ReplacementIdentityConfirmed: boolInt(receipt.ReplacementIdentityConfirmed), FenceReceiptRef: receipt.FenceReceiptRef,
		ReconciliationRef: receipt.ReconciliationRef, NeedsUserReason: receipt.NeedsUserReason,
		RequestKey: request.Key, RequestFingerprint: request.Fingerprint, CreatedAt: receipt.CreatedAt,
	}, nil
}

func waldoContinuationFromRow(row gen.WaldoContinuationReceipt) (domain.ContinuationReceipt, error) {
	receipt := domain.ContinuationReceipt{
		ID: row.ID, OperationID: row.OperationID, ConversationID: domain.WaldoConversationID(row.ConversationID), ProjectID: domain.ProjectID(row.ProjectID),
		FromEpisodeID: domain.WaldoConversationEpisodeID(row.FromEpisodeID), ToEpisodeID: domain.WaldoConversationEpisodeID(row.ToEpisodeID.String),
		FromAgentSessionRef: domain.AttemptSessionRefID(row.FromAgentSessionRefID), ToAgentSessionRef: domain.AttemptSessionRefID(row.ToAgentSessionRefID.String),
		Action: domain.ContinuationAction(row.Action), Reason: domain.ContinuationReason(row.Reason), ReasonDetail: row.ReasonDetail,
		TriggerEvidence: domain.ContinuationTriggerEvidence{Kind: domain.ContinuationTriggerEvidenceKind(row.TriggerEvidenceKind), Reference: row.TriggerEvidenceRef},
		MaterialChange:  row.MaterialChange != 0, ContextDigest: row.ContextDigest, EffectsKnown: row.EffectsKnown != 0,
		OldSessionFenced: row.OldSessionFenced != 0, ReplacementIdentityConfirmed: row.ReplacementIdentityConfirmed != 0,
		FenceReceiptRef: row.FenceReceiptRef, ReconciliationRef: row.ReconciliationRef,
		NeedsUserReason: row.NeedsUserReason, CreatedAt: row.CreatedAt,
	}
	if err := json.Unmarshal([]byte(row.ChangedFields), &receipt.ChangedFields); err != nil {
		return domain.ContinuationReceipt{}, fmt.Errorf("decode Waldo continuation changed fields: %w", err)
	}
	if err := json.Unmarshal([]byte(row.ContextRefs), &receipt.ContextRefs); err != nil {
		return domain.ContinuationReceipt{}, fmt.Errorf("decode Waldo continuation context refs: %w", err)
	}
	if err := json.Unmarshal([]byte(row.PreviousBindings), &receipt.PreviousBindings); err != nil {
		return domain.ContinuationReceipt{}, fmt.Errorf("decode Waldo continuation previous bindings: %w", err)
	}
	if err := json.Unmarshal([]byte(row.ReplacementBindings), &receipt.ReplacementBindings); err != nil {
		return domain.ContinuationReceipt{}, fmt.Errorf("decode Waldo continuation replacement bindings: %w", err)
	}
	return receipt, nil
}

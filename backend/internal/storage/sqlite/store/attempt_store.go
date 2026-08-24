package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/gen"
	"github.com/google/uuid"
)

// GetOutcomeProjectID resolves the project backing an Outcome.
func (s *Store) GetOutcomeProjectID(ctx context.Context, outcomeID domain.OutcomeID) (domain.ProjectID, bool, error) {
	projectID, err := s.qr.GetOutcomeProjectID(ctx, outcomeID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("resolve project for outcome %s: %w", outcomeID, err)
	}
	return projectID, true, nil
}

// FindAttemptByIdempotencyKey resolves a previously delivered start request.
func (s *Store) FindAttemptByIdempotencyKey(ctx context.Context, key string) (domain.Attempt, bool, error) {
	row, err := s.qr.FindAttemptByIdempotencyKey(ctx, sql.NullString{String: key, Valid: key != ""})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Attempt{}, false, nil
	}
	if err != nil {
		return domain.Attempt{}, false, fmt.Errorf("find attempt by idempotency key: %w", err)
	}
	return attemptFromRow(row), true, nil
}

// CreateAttemptWithFence atomically persists the queued attempt and issues
// its custody fence. The partial unique index on open fences backstops the
// check; a conflict rolls EVERYTHING back so a failed admission leaves zero
// durable rows.
func (s *Store) CreateAttemptWithFence(ctx context.Context, outcomeID domain.OutcomeID, plan domain.PlanRevision, requestKey string, subject string, at time.Time) (domain.Attempt, error) {
	if len(plan.WorkUnits) != 1 {
		return domain.Attempt{}, fmt.Errorf("create attempt for %s: plan must carry exactly one work unit", plan.ID)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("begin create attempt for %s: %w", outcomeID, err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	maxNum, err := txq.MaxAttemptNumber(ctx, outcomeID)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("max attempt number for %s: %w", outcomeID, err)
	}
	number := int64(1)
	if v, ok := maxNum.(int64); ok {
		number = v + 1
	} else {
		return domain.Attempt{}, fmt.Errorf("max attempt number for %s: unexpected type %T", outcomeID, maxNum)
	}

	var key sql.NullString
	if requestKey != "" {
		key = sql.NullString{String: requestKey, Valid: true}
	}
	attempt := domain.Attempt{
		ID:                     domain.AttemptID("att-" + uuid.NewString()),
		OutcomeID:              outcomeID,
		PlanRevisionID:         plan.ID,
		WorkUnitID:             plan.WorkUnits[0].ID,
		Number:                 number,
		Status:                 domain.AttemptQueued,
		RequestKey:             requestKey,
		CreatedAt:              at,
		UpdatedAt:              at,
		ContractRevisionNumber: plan.ContractRevisionNumber,
	}
	if err := attempt.Validate(); err != nil {
		return domain.Attempt{}, err
	}
	if err := txq.CreateAttempt(ctx, gen.CreateAttemptParams{
		ID:                     attempt.ID,
		OutcomeID:              attempt.OutcomeID,
		PlanRevisionID:         attempt.PlanRevisionID,
		WorkUnitID:             attempt.WorkUnitID,
		Number:                 attempt.Number,
		Status:                 attempt.Status,
		ContractRevisionNumber: attempt.ContractRevisionNumber,
		RequestKey:             key,
	}); err != nil {
		return domain.Attempt{}, fmt.Errorf("create attempt for %s: %w", outcomeID, err)
	}

	fenceID := "fence-" + uuid.NewString()
	if err := txq.IssueAttemptFence(ctx, gen.IssueAttemptFenceParams{
		ID:        fenceID,
		Subject:   subject,
		AttemptID: attempt.ID,
	}); err != nil {
		if isSQLiteUnique(err) {
			holder := domain.AttemptID("")
			if open, findErr := txq.FindOpenFenceBySubject(ctx, subject); findErr == nil {
				holder = open.AttemptID
			}
			return domain.Attempt{}, &ports.AttemptFenceHeldError{Subject: subject, Holder: holder, OutcomeID: outcomeID}
		}
		return domain.Attempt{}, fmt.Errorf("issue fence for %s: %w", attempt.ID, err)
	}

	if err := tx.Commit(); err != nil {
		return domain.Attempt{}, fmt.Errorf("commit create attempt for %s: %w", outcomeID, err)
	}
	return attempt, nil
}

// GetAttempt reads one attempt of an Outcome; ok=false when absent.
func (s *Store) GetAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (domain.Attempt, bool, error) {
	row, err := s.qr.GetAttempt(ctx, gen.GetAttemptParams{ID: attemptID, OutcomeID: outcomeID})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Attempt{}, false, nil
	}
	if err != nil {
		return domain.Attempt{}, false, fmt.Errorf("get attempt %s: %w", attemptID, err)
	}
	return attemptFromRow(row), true, nil
}

// ListAttempts returns the Outcome's attempts in ascending order.
func (s *Store) ListAttempts(ctx context.Context, outcomeID domain.OutcomeID) ([]domain.Attempt, error) {
	rows, err := s.qr.ListAttemptsForOutcome(ctx, outcomeID)
	if err != nil {
		return nil, fmt.Errorf("list attempts for %s: %w", outcomeID, err)
	}
	out := make([]domain.Attempt, 0, len(rows))
	for _, row := range rows {
		out = append(out, attemptFromRow(row))
	}
	return out, nil
}

// TransitionAttemptStatus applies one guarded stored-status transition.
func (s *Store) TransitionAttemptStatus(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID, expected, next domain.AttemptStatus, at time.Time) (int64, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.qw.TransitionAttemptStatus(ctx, gen.TransitionAttemptStatusParams{
		Status:    next,
		UpdatedAt: at,
		ID:        attemptID,
		OutcomeID: outcomeID,
		Status_2:  expected,
	})
	if err != nil {
		return 0, fmt.Errorf("transition attempt %s %s->%s: %w", attemptID, expected, next, err)
	}
	return rows, nil
}

// ListAttemptsByStatus walks attempts currently in one stored status.
func (s *Store) ListAttemptsByStatus(ctx context.Context, status domain.AttemptStatus) ([]domain.Attempt, error) {
	rows, err := s.qr.ListAttemptsByStatus(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("list attempts by status %s: %w", status, err)
	}
	out := make([]domain.Attempt, 0, len(rows))
	for _, row := range rows {
		out = append(out, attemptFromRow(row))
	}
	return out, nil
}

// BindAttemptSession appends one immutable provider-session ref.
func (s *Store) BindAttemptSession(ctx context.Context, ref domain.AttemptSessionRef) (domain.AttemptSessionRef, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return domain.AttemptSessionRef{}, fmt.Errorf("begin bind session for %s: %w", ref.AttemptID, err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	seq, err := latestSessionRefSeq(ctx, txq, ref.AttemptID)
	if err != nil {
		return domain.AttemptSessionRef{}, err
	}
	ref.Seq = seq + 1
	if ref.ID.IsZero() {
		ref.ID = domain.AttemptSessionRefID("asr-" + uuid.NewString())
	}
	if ref.BoundAt.IsZero() {
		ref.BoundAt = time.Now().UTC()
	}
	if err := ref.Validate(); err != nil {
		return domain.AttemptSessionRef{}, err
	}
	if err := txq.CreateAttemptSessionRef(ctx, gen.CreateAttemptSessionRefParams{
		ID:                     string(ref.ID),
		AttemptID:              ref.AttemptID,
		Seq:                    ref.Seq,
		SessionID:              ref.SessionID,
		Harness:                ref.Harness,
		Mode:                   ref.Mode,
		RunBriefCoreDigest:     ref.RunBriefCoreDigest,
		RunBriefCompiledDigest: ref.RunBriefCompiledDigest,
		AdmissionSnapshot:      ref.AdmissionSnapshot,
	}); err != nil {
		return domain.AttemptSessionRef{}, fmt.Errorf("bind session for %s: %w", ref.AttemptID, err)
	}
	if err := tx.Commit(); err != nil {
		return domain.AttemptSessionRef{}, fmt.Errorf("commit bind session for %s: %w", ref.AttemptID, err)
	}
	return ref, nil
}

func latestSessionRefSeq(ctx context.Context, q *gen.Queries, attemptID domain.AttemptID) (int64, error) {
	latest, err := q.LatestAttemptSessionRef(ctx, attemptID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("latest session ref for %s: %w", attemptID, err)
	}
	return latest.Seq, nil
}

// LatestAttemptSessionRef resolves the most recent binding; ok=false when none.
func (s *Store) LatestAttemptSessionRef(ctx context.Context, attemptID domain.AttemptID) (domain.AttemptSessionRef, bool, error) {
	row, err := s.qr.LatestAttemptSessionRef(ctx, attemptID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AttemptSessionRef{}, false, nil
	}
	if err != nil {
		return domain.AttemptSessionRef{}, false, fmt.Errorf("latest session ref for %s: %w", attemptID, err)
	}
	return attemptSessionRefFromRow(row), true, nil
}

// ListAttemptSessionRefs returns every binding in ascending seq order.
func (s *Store) ListAttemptSessionRefs(ctx context.Context, attemptID domain.AttemptID) ([]domain.AttemptSessionRef, error) {
	rows, err := s.qr.ListAttemptSessionRefsForAttempt(ctx, attemptID)
	if err != nil {
		return nil, fmt.Errorf("list session refs for %s: %w", attemptID, err)
	}
	out := make([]domain.AttemptSessionRef, 0, len(rows))
	for _, row := range rows {
		out = append(out, attemptSessionRefFromRow(row))
	}
	return out, nil
}

// AppendAttemptObservation appends one ordered observation with the next seq.
// Observations are insertable for any attempt state (D5): inspection stays
// possible after replacement; nothing here touches current truth.
func (s *Store) AppendAttemptObservation(ctx context.Context, attemptID domain.AttemptID, kind string, payload string, at time.Time) (domain.AttemptObservation, error) {
	if payload == "" {
		payload = "{}"
	}
	obs := domain.AttemptObservation{
		ID:        "obs-" + uuid.NewString(),
		AttemptID: attemptID,
		Kind:      kind,
		Payload:   payload,
		CreatedAt: at,
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return domain.AttemptObservation{}, fmt.Errorf("begin observation for %s: %w", attemptID, err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	maxSeq, err := txq.MaxAttemptObservationSeq(ctx, attemptID)
	if err != nil {
		return domain.AttemptObservation{}, fmt.Errorf("max observation seq for %s: %w", attemptID, err)
	}
	switch v := maxSeq.(type) {
	case int64:
		obs.Seq = v + 1
	default:
		return domain.AttemptObservation{}, fmt.Errorf("max observation seq for %s: unexpected type %T", attemptID, maxSeq)
	}
	if err := obs.Validate(); err != nil {
		return domain.AttemptObservation{}, err
	}
	if err := txq.CreateAttemptObservation(ctx, gen.CreateAttemptObservationParams{
		ID:        obs.ID,
		AttemptID: obs.AttemptID,
		Seq:       obs.Seq,
		Kind:      obs.Kind,
		Payload:   obs.Payload,
	}); err != nil {
		return domain.AttemptObservation{}, fmt.Errorf("append observation for %s: %w", attemptID, err)
	}
	if err := tx.Commit(); err != nil {
		return domain.AttemptObservation{}, fmt.Errorf("commit observation for %s: %w", attemptID, err)
	}
	return obs, nil
}

// ListAttemptObservations returns the attempt's ordered observations.
func (s *Store) ListAttemptObservations(ctx context.Context, attemptID domain.AttemptID) ([]domain.AttemptObservation, error) {
	rows, err := s.qr.ListAttemptObservationsForAttempt(ctx, attemptID)
	if err != nil {
		return nil, fmt.Errorf("list observations for %s: %w", attemptID, err)
	}
	out := make([]domain.AttemptObservation, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.AttemptObservation{
			ID:        row.ID,
			AttemptID: row.AttemptID,
			Seq:       row.Seq,
			Kind:      row.Kind,
			Payload:   row.Payload,
			CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

// OpenFenceForSubject resolves the open fence over a worktree subject.
func (s *Store) OpenFenceForSubject(ctx context.Context, subject string) (domain.AttemptFence, bool, error) {
	row, err := s.qr.FindOpenFenceBySubject(ctx, subject)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AttemptFence{}, false, nil
	}
	if err != nil {
		return domain.AttemptFence{}, false, fmt.Errorf("open fence for %s: %w", subject, err)
	}
	return attemptFenceFromRow(row), true, nil
}

// ReleaseFenceForAttempt releases the attempt's open fence with a reason.
func (s *Store) ReleaseFenceForAttempt(ctx context.Context, attemptID domain.AttemptID, reason string, at time.Time) (int64, error) {
	if reason == "" {
		return 0, fmt.Errorf("release fence for %s: a released fence must record why", attemptID)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.qw.ReleaseAttemptFence(ctx, gen.ReleaseAttemptFenceParams{
		ReleasedAt:    sql.NullTime{Time: at, Valid: true},
		ReleaseReason: reason,
		AttemptID:     attemptID,
	})
	if err != nil {
		return 0, fmt.Errorf("release fence for %s: %w", attemptID, err)
	}
	return rows, nil
}

// CreateRecoveryReceipt appends one immutable recovery receipt.
func (s *Store) CreateRecoveryReceipt(ctx context.Context, receipt domain.AttemptRecoveryReceipt) error {
	if receipt.ID == "" {
		receipt.ID = "rcpt-" + uuid.NewString()
	}
	if err := receipt.Validate(); err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	detail := receipt.Detail
	if detail == "" {
		detail = "{}"
	} else if !json.Valid([]byte(detail)) {
		return fmt.Errorf("recovery receipt %s detail must be valid JSON", receipt.ID)
	}
	if err := s.qw.CreateRecoveryReceipt(ctx, gen.CreateRecoveryReceiptParams{
		ID:                   receipt.ID,
		AttemptID:            receipt.AttemptID,
		Resolution:           string(receipt.Resolution),
		ReplacementAttemptID: string(receipt.ReplacementAttemptID),
		Detail:               detail,
	}); err != nil {
		return fmt.Errorf("create recovery receipt for %s: %w", receipt.AttemptID, err)
	}
	return nil
}

// ListRecoveryReceipts returns the attempt's receipts in emission order.
func (s *Store) ListRecoveryReceipts(ctx context.Context, attemptID domain.AttemptID) ([]domain.AttemptRecoveryReceipt, error) {
	rows, err := s.qr.ListRecoveryReceiptsForAttempt(ctx, attemptID)
	if err != nil {
		return nil, fmt.Errorf("list receipts for %s: %w", attemptID, err)
	}
	out := make([]domain.AttemptRecoveryReceipt, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.AttemptRecoveryReceipt{
			ID:                   row.ID,
			AttemptID:            row.AttemptID,
			Resolution:           domain.RecoveryResolution(row.Resolution),
			ReplacementAttemptID: domain.AttemptID(row.ReplacementAttemptID),
			Detail:               row.Detail,
			CreatedAt:            row.CreatedAt,
		})
	}
	return out, nil
}

func attemptFromRow(row gen.Attempt) domain.Attempt {
	var requestKey string
	if row.RequestKey.Valid {
		requestKey = row.RequestKey.String
	}
	return domain.Attempt{
		ID:                     row.ID,
		OutcomeID:              row.OutcomeID,
		PlanRevisionID:         row.PlanRevisionID,
		WorkUnitID:             row.WorkUnitID,
		Number:                 row.Number,
		Status:                 row.Status,
		RequestKey:             requestKey,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
		ContractRevisionNumber: row.ContractRevisionNumber,
	}
}

func attemptSessionRefFromRow(row gen.AttemptSession) domain.AttemptSessionRef {
	return domain.AttemptSessionRef{
		ID:                     domain.AttemptSessionRefID(row.ID),
		AttemptID:              row.AttemptID,
		Seq:                    row.Seq,
		SessionID:              row.SessionID,
		Harness:                row.Harness,
		Mode:                   row.Mode,
		RunBriefCoreDigest:     row.RunBriefCoreDigest,
		RunBriefCompiledDigest: row.RunBriefCompiledDigest,
		AdmissionSnapshot:      row.AdmissionSnapshot,
		BoundAt:                row.BoundAt,
	}
}

func attemptFenceFromRow(row gen.AttemptFence) domain.AttemptFence {
	return domain.AttemptFence{
		ID:            row.ID,
		Subject:       row.Subject,
		AttemptID:     row.AttemptID,
		IssuedAt:      row.IssuedAt,
		ReleasedAt:    row.ReleasedAt.Time,
		ReleaseReason: row.ReleaseReason,
	}
}

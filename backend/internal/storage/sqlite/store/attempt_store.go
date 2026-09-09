package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/gen"
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

// CreateAttemptWithFence atomically persists one scheduler-selected WorkUnit
// Attempt and issues its custody fence. Storage receives exact canonical
// identity; it never inspects a Plan or chooses a WorkUnit itself.
func (s *Store) CreateAttemptWithFence(ctx context.Context, in ports.AttemptAdmission) (domain.Attempt, error) {
	if in.OutcomeID.IsZero() || in.PlanRevisionID.IsZero() || in.WorkUnitID.IsZero() {
		return domain.Attempt{}, fmt.Errorf("attempt admission requires outcome, plan revision, and work unit ids")
	}
	if in.ContractRevisionNumber < 1 {
		return domain.Attempt{}, fmt.Errorf("attempt admission requires a contract revision")
	}
	if strings.TrimSpace(in.RequestKey) == "" {
		return domain.Attempt{}, fmt.Errorf("attempt admission requires a request key")
	}
	if strings.TrimSpace(in.FenceSubject) == "" {
		return domain.Attempt{}, fmt.Errorf("attempt admission requires a fence subject")
	}
	if in.At.IsZero() {
		return domain.Attempt{}, fmt.Errorf("attempt admission requires a timestamp")
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("begin create attempt for %s: %w", in.OutcomeID, err)
	}
	defer func() { _ = tx.Rollback() }()
	txq := s.qw.WithTx(tx)

	maxNum, err := txq.MaxAttemptNumber(ctx, in.OutcomeID)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("max attempt number for %s: %w", in.OutcomeID, err)
	}
	priorCount, ok := maxNum.(int64)
	if !ok {
		return domain.Attempt{}, fmt.Errorf("max attempt number for %s: unexpected type %T", in.OutcomeID, maxNum)
	}

	key := sql.NullString{String: strings.TrimSpace(in.RequestKey), Valid: true}
	attempt := domain.Attempt{
		ID:                     domain.AttemptID("att-" + uuid.NewString()),
		OutcomeID:              in.OutcomeID,
		PlanRevisionID:         in.PlanRevisionID,
		WorkUnitID:             in.WorkUnitID,
		Number:                 priorCount + 1,
		Status:                 domain.AttemptQueued,
		RequestKey:             key.String,
		CreatedAt:              in.At,
		UpdatedAt:              in.At,
		ContractRevisionNumber: in.ContractRevisionNumber,
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
		if isSQLiteUnique(err) && strings.Contains(err.Error(), "request_key") {
			row, findErr := txq.FindAttemptByIdempotencyKey(ctx, key)
			if findErr == nil {
				winner := attemptFromRow(row)
				return winner, &ports.AttemptReplayError{Attempt: winner}
			}
		}
		return domain.Attempt{}, fmt.Errorf("create attempt for %s: %w", in.OutcomeID, err)
	}

	fenceID := "fence-" + uuid.NewString()
	if err := txq.IssueAttemptFence(ctx, gen.IssueAttemptFenceParams{
		ID:        fenceID,
		Subject:   in.FenceSubject,
		AttemptID: attempt.ID,
	}); err != nil {
		if isSQLiteUnique(err) {
			holder := domain.AttemptID("")
			if open, findErr := txq.FindOpenFenceBySubject(ctx, in.FenceSubject); findErr == nil {
				holder = open.AttemptID
			}
			return domain.Attempt{}, &ports.AttemptFenceHeldError{Subject: in.FenceSubject, Holder: holder, OutcomeID: in.OutcomeID}
		}
		return domain.Attempt{}, fmt.Errorf("issue fence for %s: %w", attempt.ID, err)
	}

	if err := tx.Commit(); err != nil {
		return domain.Attempt{}, fmt.Errorf("commit create attempt for %s: %w", in.OutcomeID, err)
	}
	return attempt, nil
}

// GetAttempt loads one attempt scoped to its Outcome.
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

// ListAttempts loads attempts belonging to an Outcome.
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

// TransitionAttemptStatus advances an attempt with optimistic concurrency.
func (s *Store) TransitionAttemptStatus(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID, expected, next domain.AttemptStatus, at time.Time) (int64, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.qw.TransitionAttemptStatus(ctx, gen.TransitionAttemptStatusParams{
		Status: next, UpdatedAt: at, ID: attemptID, OutcomeID: outcomeID, Status_2: expected,
	})
	if err != nil {
		return 0, fmt.Errorf("transition attempt %s %s->%s: %w", attemptID, expected, next, err)
	}
	return rows, nil
}

// ListAttemptsByStatus loads attempts in a durable status.
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

// BindAttemptSession records a provider session reference for an attempt.
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
		ID: string(ref.ID), AttemptID: ref.AttemptID, Seq: ref.Seq, SessionID: ref.SessionID,
		Harness: ref.Harness, Mode: ref.Mode, RunBriefCoreDigest: ref.RunBriefCoreDigest,
		RunBriefCompiledDigest: ref.RunBriefCompiledDigest, AdmissionSnapshot: ref.AdmissionSnapshot,
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

// LatestAttemptSessionRef loads the latest provider session reference.
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

// ListAttemptSessionRefs loads all provider session references for an attempt.
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

// AppendAttemptObservation records one bounded attempt observation.
func (s *Store) AppendAttemptObservation(ctx context.Context, attemptID domain.AttemptID, kind, payload string, at time.Time) (domain.AttemptObservation, error) {
	if payload == "" {
		payload = "{}"
	}
	obs := domain.AttemptObservation{ID: "obs-" + uuid.NewString(), AttemptID: attemptID, Kind: kind, Payload: payload, CreatedAt: at}

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
	if err := txq.CreateAttemptObservation(ctx, gen.CreateAttemptObservationParams{ID: obs.ID, AttemptID: obs.AttemptID, Seq: obs.Seq, Kind: obs.Kind, Payload: obs.Payload}); err != nil {
		return domain.AttemptObservation{}, fmt.Errorf("append observation for %s: %w", attemptID, err)
	}
	if err := tx.Commit(); err != nil {
		return domain.AttemptObservation{}, fmt.Errorf("commit observation for %s: %w", attemptID, err)
	}
	return obs, nil
}

// ListAttemptObservations loads observations for an attempt.
func (s *Store) ListAttemptObservations(ctx context.Context, attemptID domain.AttemptID) ([]domain.AttemptObservation, error) {
	rows, err := s.qr.ListAttemptObservationsForAttempt(ctx, attemptID)
	if err != nil {
		return nil, fmt.Errorf("list observations for %s: %w", attemptID, err)
	}
	out := make([]domain.AttemptObservation, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.AttemptObservation{ID: row.ID, AttemptID: row.AttemptID, Seq: row.Seq, Kind: row.Kind, Payload: row.Payload, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

// OpenFenceForSubject loads the active custody fence for a subject.
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

// ReleaseFenceForAttempt releases custody held by an attempt.
func (s *Store) ReleaseFenceForAttempt(ctx context.Context, attemptID domain.AttemptID, reason string, at time.Time) (int64, error) {
	if reason == "" {
		return 0, fmt.Errorf("release fence for %s: a released fence must record why", attemptID)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.qw.ReleaseAttemptFence(ctx, gen.ReleaseAttemptFenceParams{ReleasedAt: sql.NullTime{Time: at, Valid: true}, ReleaseReason: reason, AttemptID: attemptID})
	if err != nil {
		return 0, fmt.Errorf("release fence for %s: %w", attemptID, err)
	}
	return rows, nil
}

// RenewFenceForAttempt renews custody held by an attempt.
func (s *Store) RenewFenceForAttempt(ctx context.Context, attemptID domain.AttemptID, at time.Time) (int64, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.qw.RenewAttemptFence(ctx, gen.RenewAttemptFenceParams{LastRenewedAt: at, AttemptID: attemptID})
	if err != nil {
		return 0, fmt.Errorf("renew fence for %s: %w", attemptID, err)
	}
	return rows, nil
}

// CreateRecoveryReceipt persists one recovery decision receipt.
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
		ID: receipt.ID, AttemptID: receipt.AttemptID, Resolution: string(receipt.Resolution),
		ReplacementAttemptID: string(receipt.ReplacementAttemptID), Detail: detail,
	}); err != nil {
		return fmt.Errorf("create recovery receipt for %s: %w", receipt.AttemptID, err)
	}
	return nil
}

// ListRecoveryReceipts loads recovery receipts for an attempt.
func (s *Store) ListRecoveryReceipts(ctx context.Context, attemptID domain.AttemptID) ([]domain.AttemptRecoveryReceipt, error) {
	rows, err := s.qr.ListRecoveryReceiptsForAttempt(ctx, attemptID)
	if err != nil {
		return nil, fmt.Errorf("list receipts for %s: %w", attemptID, err)
	}
	out := make([]domain.AttemptRecoveryReceipt, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.AttemptRecoveryReceipt{
			ID: row.ID, AttemptID: row.AttemptID, Resolution: domain.RecoveryResolution(row.Resolution),
			ReplacementAttemptID: domain.AttemptID(row.ReplacementAttemptID), Detail: row.Detail, CreatedAt: row.CreatedAt,
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
		ID: row.ID, OutcomeID: row.OutcomeID, PlanRevisionID: row.PlanRevisionID, WorkUnitID: row.WorkUnitID,
		Number: row.Number, Status: row.Status, RequestKey: requestKey, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		ContractRevisionNumber: row.ContractRevisionNumber,
	}
}

func attemptSessionRefFromRow(row gen.AttemptSession) domain.AttemptSessionRef {
	return domain.AttemptSessionRef{
		ID: domain.AttemptSessionRefID(row.ID), AttemptID: row.AttemptID, Seq: row.Seq, SessionID: row.SessionID,
		Harness: row.Harness, Mode: row.Mode, RunBriefCoreDigest: row.RunBriefCoreDigest,
		RunBriefCompiledDigest: row.RunBriefCompiledDigest, AdmissionSnapshot: row.AdmissionSnapshot, BoundAt: row.BoundAt,
	}
}

func attemptFenceFromRow(row gen.AttemptFence) domain.AttemptFence {
	return domain.AttemptFence{
		ID: row.ID, Subject: row.Subject, AttemptID: row.AttemptID, IssuedAt: row.IssuedAt,
		LastRenewedAt: row.LastRenewedAt, ReleasedAt: row.ReleasedAt.Time, ReleaseReason: row.ReleaseReason,
	}
}

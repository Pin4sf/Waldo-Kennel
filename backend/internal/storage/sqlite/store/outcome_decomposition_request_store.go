package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/gen"
)

// CreateDecompositionRequest opens one durable ask for an agent-authored
// decomposition. Only the token's digest is persisted; the token itself is
// handed to the spawned session and never stored.
func (s *Store) CreateDecompositionRequest(ctx context.Context, request domain.DecompositionRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	if request.Status != domain.DecompositionRequested {
		return fmt.Errorf("a decomposition request opens as requested, not %q", request.Status)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.qw.CreateDecompositionRequest(ctx, gen.CreateDecompositionRequestParams{
		ID:                  request.ID,
		OutcomeID:           request.OutcomeID,
		ContractRevisionID:  request.ContractRevisionID,
		Status:              request.Status,
		CallbackTokenDigest: request.CallbackTokenDigest,
		SessionID:           request.SessionID,
		ExpiresAt:           request.ExpiresAt,
		CreatedAt:           request.CreatedAt,
	}); err != nil {
		return fmt.Errorf("open decomposition request %s: %w", request.ID, err)
	}
	return nil
}

// GetDecompositionRequest reads one ask; ok=false when absent.
func (s *Store) GetDecompositionRequest(ctx context.Context, id domain.DecompositionRequestID) (domain.DecompositionRequest, bool, error) {
	row, err := s.qr.GetDecompositionRequest(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.DecompositionRequest{}, false, nil
	}
	if err != nil {
		return domain.DecompositionRequest{}, false, fmt.Errorf("get decomposition request %s: %w", id, err)
	}
	return decompositionRequestFromRow(row), true, nil
}

// LatestDecompositionRequest returns an Outcome's newest ask of any status;
// ok=false when it has never been asked.
func (s *Store) LatestDecompositionRequest(ctx context.Context, outcomeID domain.OutcomeID) (domain.DecompositionRequest, bool, error) {
	row, err := s.qr.LatestDecompositionRequest(ctx, outcomeID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.DecompositionRequest{}, false, nil
	}
	if err != nil {
		return domain.DecompositionRequest{}, false, fmt.Errorf("latest decomposition request for %s: %w", outcomeID, err)
	}
	return decompositionRequestFromRow(row), true, nil
}

// AnswerDecompositionRequest closes an open ask, one way, with the agent's
// draft retained whatever the verdict.
//
// The guarded UPDATE is what makes the callback single-use: a second answer
// finds no open row and is refused, so a retrying agent cannot produce two
// decompositions.
func (s *Store) AnswerDecompositionRequest(ctx context.Context, answer ports.DecompositionRequestAnswer) error {
	if !answer.Status.Valid() || answer.Status.Open() {
		return fmt.Errorf("a decomposition request is answered with a closed status, not %q", answer.Status)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	var decompositionID sql.NullString
	if !answer.DecompositionID.IsZero() {
		decompositionID = sql.NullString{String: string(answer.DecompositionID), Valid: true}
	}
	rows, err := s.qw.AnswerDecompositionRequest(ctx, gen.AnswerDecompositionRequestParams{
		Status:          answer.Status,
		RawProposal:     answer.RawProposal,
		RefusalReason:   answer.RefusalReason,
		DecompositionID: decompositionID,
		AnsweredAt:      sql.NullTime{Time: answer.At, Valid: true},
		ID:              answer.RequestID,
	})
	if err != nil {
		return fmt.Errorf("answer decomposition request %s: %w", answer.RequestID, err)
	}
	if rows != 1 {
		return fmt.Errorf("%w: %s", ports.ErrDecompositionRequestClosed, answer.RequestID)
	}
	return nil
}

// ListOpenDecompositionRequests returns every unanswered ask, soonest expiry
// first, so a restarted daemon can close the ones that timed out while it was
// not running.
func (s *Store) ListOpenDecompositionRequests(ctx context.Context) ([]domain.DecompositionRequest, error) {
	rows, err := s.qr.ListOpenDecompositionRequests(ctx)
	if err != nil {
		return nil, fmt.Errorf("list open decomposition requests: %w", err)
	}
	out := make([]domain.DecompositionRequest, 0, len(rows))
	for _, row := range rows {
		out = append(out, decompositionRequestFromRow(row))
	}
	return out, nil
}

func decompositionRequestFromRow(row gen.DecompositionRequest) domain.DecompositionRequest {
	request := domain.DecompositionRequest{
		ID:                  row.ID,
		OutcomeID:           row.OutcomeID,
		ContractRevisionID:  row.ContractRevisionID,
		Status:              row.Status,
		CallbackTokenDigest: row.CallbackTokenDigest,
		SessionID:           row.SessionID,
		ExpiresAt:           row.ExpiresAt,
		RawProposal:         row.RawProposal,
		RefusalReason:       row.RefusalReason,
		CreatedAt:           row.CreatedAt,
	}
	if row.DecompositionID.Valid {
		request.DecompositionID = domain.DecompositionRevisionID(row.DecompositionID.String)
	}
	if row.AnsweredAt.Valid {
		at := row.AnsweredAt.Time
		request.AnsweredAt = &at
	}
	return request
}

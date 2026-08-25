package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/gen"
)

var _ ports.OutcomeProofStore = (*Store)(nil)

// CreateEvidenceItem appends one immutable Evidence record.
func (s *Store) CreateEvidenceItem(ctx context.Context, item domain.EvidenceItem) error {
	if err := item.Validate(); err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.qw.CreateEvidenceItem(ctx, gen.CreateEvidenceItemParams{
		ID: string(item.ID), OutcomeID: string(item.OutcomeID), ContractRevisionID: string(item.ContractRevisionID),
		CriterionID: string(item.CriterionID), SubjectType: string(item.SubjectType), SubjectID: item.SubjectID,
		SubjectRevision: item.SubjectRevision, Kind: string(item.Kind), SourceType: string(item.SourceType),
		SourceRef: item.SourceRef, ProducerType: string(item.ProducerType), ProducerRef: item.ProducerRef,
		Summary: item.Summary, ContentDigest: item.ContentDigest, RequestKey: item.RequestKey,
		RequestFingerprint: item.RequestFingerprint, CreatedAt: item.CreatedAt,
	}); err != nil {
		return fmt.Errorf("create evidence item %s: %w", item.ID, err)
	}
	return nil
}

// FindEvidenceItemByRequestKey resolves idempotent Evidence replays.
func (s *Store) FindEvidenceItemByRequestKey(ctx context.Context, key string) (domain.EvidenceItem, bool, error) {
	row, err := s.qr.FindEvidenceItemByRequestKey(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.EvidenceItem{}, false, nil
	}
	if err != nil {
		return domain.EvidenceItem{}, false, fmt.Errorf("find evidence by request key: %w", err)
	}
	return evidenceFromRow(row), true, nil
}

// GetEvidenceItem reads one Evidence record inside its Outcome lineage.
func (s *Store) GetEvidenceItem(ctx context.Context, outcomeID domain.OutcomeID, id domain.EvidenceItemID) (domain.EvidenceItem, bool, error) {
	row, err := s.qr.GetEvidenceItem(ctx, gen.GetEvidenceItemParams{ID: string(id), OutcomeID: string(outcomeID)})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.EvidenceItem{}, false, nil
	}
	if err != nil {
		return domain.EvidenceItem{}, false, fmt.Errorf("get evidence %s: %w", id, err)
	}
	return evidenceFromRow(row), true, nil
}

// ListEvidenceItems returns immutable Evidence in creation order.
func (s *Store) ListEvidenceItems(ctx context.Context, outcomeID domain.OutcomeID) ([]domain.EvidenceItem, error) {
	rows, err := s.qr.ListEvidenceItemsForOutcome(ctx, string(outcomeID))
	if err != nil {
		return nil, fmt.Errorf("list evidence for %s: %w", outcomeID, err)
	}
	out := make([]domain.EvidenceItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, evidenceFromRow(row))
	}
	return out, nil
}

// CreateVerificationRun appends one immutable VerificationRun.
func (s *Store) CreateVerificationRun(ctx context.Context, run domain.VerificationRun) error {
	if err := run.Validate(); err != nil {
		return err
	}
	ids, err := json.Marshal(run.EvidenceItemIDs)
	if err != nil {
		return fmt.Errorf("marshal verification evidence ids: %w", err)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.qw.CreateVerificationRun(ctx, gen.CreateVerificationRunParams{
		ID: string(run.ID), OutcomeID: string(run.OutcomeID), ContractRevisionID: string(run.ContractRevisionID),
		CriterionID: string(run.CriterionID), SubjectType: string(run.SubjectType), SubjectID: run.SubjectID,
		SubjectRevision: run.SubjectRevision, EvidenceItemIds: string(ids), Method: run.Method,
		IndependenceClass: string(run.IndependenceClass), Result: string(run.Result), ProducerRef: run.ProducerRef,
		VerifierRef: run.VerifierRef, ProducerProvider: run.ProducerProvider, VerifierProvider: run.VerifierProvider,
		Detail: run.Detail, RequestKey: run.RequestKey, RequestFingerprint: run.RequestFingerprint, CreatedAt: run.CreatedAt,
	}); err != nil {
		return fmt.Errorf("create verification run %s: %w", run.ID, err)
	}
	return nil
}

// FindVerificationRunByRequestKey resolves idempotent Verification replays.
func (s *Store) FindVerificationRunByRequestKey(ctx context.Context, key string) (domain.VerificationRun, bool, error) {
	row, err := s.qr.FindVerificationRunByRequestKey(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.VerificationRun{}, false, nil
	}
	if err != nil {
		return domain.VerificationRun{}, false, fmt.Errorf("find verification by request key: %w", err)
	}
	run, err := verificationFromRow(row)
	return run, true, err
}

// ListVerificationRuns returns immutable Verification runs in creation order.
func (s *Store) ListVerificationRuns(ctx context.Context, outcomeID domain.OutcomeID) ([]domain.VerificationRun, error) {
	rows, err := s.qr.ListVerificationRunsForOutcome(ctx, string(outcomeID))
	if err != nil {
		return nil, fmt.Errorf("list verification runs for %s: %w", outcomeID, err)
	}
	out := make([]domain.VerificationRun, 0, len(rows))
	for _, row := range rows {
		run, err := verificationFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, nil
}

// CreateAcceptanceDecision atomically appends a decision and optional correction.
func (s *Store) CreateAcceptanceDecision(ctx context.Context, decision domain.AcceptanceDecision, correction *domain.OutcomeCorrection) error {
	if err := decision.Validate(); err != nil {
		return err
	}
	if correction != nil {
		if err := correction.Validate(); err != nil {
			return err
		}
		if correction.DecisionID != decision.ID || correction.OutcomeID != decision.OutcomeID || correction.ContractRevisionID != decision.ContractRevisionID {
			return fmt.Errorf("correction lineage does not match acceptance decision")
		}
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.inTx(ctx, fmt.Sprintf("create acceptance decision %s", decision.ID), func(q *gen.Queries) error {
		if err := q.CreateAcceptanceDecision(ctx, gen.CreateAcceptanceDecisionParams{
			ID: string(decision.ID), OutcomeID: string(decision.OutcomeID), ContractRevisionID: string(decision.ContractRevisionID),
			Kind: string(decision.Kind), ActorType: string(decision.ActorType), Summary: decision.Summary,
			ResourceDisposition: string(decision.ResourceDisposition), RequestKey: decision.RequestKey,
			RequestFingerprint: decision.RequestFingerprint, CreatedAt: decision.CreatedAt,
		}); err != nil {
			return err
		}
		if correction != nil {
			return q.CreateOutcomeCorrection(ctx, gen.CreateOutcomeCorrectionParams{
				ID: string(correction.ID), DecisionID: string(correction.DecisionID), OutcomeID: string(correction.OutcomeID),
				ContractRevisionID: string(correction.ContractRevisionID), Feedback: correction.Feedback,
				TargetType: string(correction.TargetType), TargetID: correction.TargetID, CreatedAt: correction.CreatedAt,
			})
		}
		return nil
	})
}

// FindAcceptanceDecisionByRequestKey resolves idempotent decision replays.
func (s *Store) FindAcceptanceDecisionByRequestKey(ctx context.Context, key string) (domain.AcceptanceDecision, bool, error) {
	row, err := s.qr.FindAcceptanceDecisionByRequestKey(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AcceptanceDecision{}, false, nil
	}
	if err != nil {
		return domain.AcceptanceDecision{}, false, fmt.Errorf("find acceptance decision by request key: %w", err)
	}
	return acceptanceFromRow(row), true, nil
}

// ListAcceptanceDecisions returns explicit user decisions in creation order.
func (s *Store) ListAcceptanceDecisions(ctx context.Context, outcomeID domain.OutcomeID) ([]domain.AcceptanceDecision, error) {
	rows, err := s.qr.ListAcceptanceDecisionsForOutcome(ctx, string(outcomeID))
	if err != nil {
		return nil, fmt.Errorf("list acceptance decisions for %s: %w", outcomeID, err)
	}
	out := make([]domain.AcceptanceDecision, 0, len(rows))
	for _, row := range rows {
		out = append(out, acceptanceFromRow(row))
	}
	return out, nil
}

// ListOutcomeCorrections returns durable re-entry lineage in creation order.
func (s *Store) ListOutcomeCorrections(ctx context.Context, outcomeID domain.OutcomeID) ([]domain.OutcomeCorrection, error) {
	rows, err := s.qr.ListOutcomeCorrectionsForOutcome(ctx, string(outcomeID))
	if err != nil {
		return nil, fmt.Errorf("list outcome corrections for %s: %w", outcomeID, err)
	}
	out := make([]domain.OutcomeCorrection, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.OutcomeCorrection{
			ID: domain.OutcomeCorrectionID(row.ID), DecisionID: domain.AcceptanceDecisionID(row.DecisionID),
			OutcomeID: domain.OutcomeID(row.OutcomeID), ContractRevisionID: domain.ContractRevisionID(row.ContractRevisionID),
			Feedback: row.Feedback, TargetType: domain.ReentryTargetType(row.TargetType), TargetID: row.TargetID, CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func evidenceFromRow(row gen.EvidenceItem) domain.EvidenceItem {
	return domain.EvidenceItem{
		ID: domain.EvidenceItemID(row.ID), OutcomeID: domain.OutcomeID(row.OutcomeID),
		ContractRevisionID: domain.ContractRevisionID(row.ContractRevisionID), CriterionID: domain.CriterionID(row.CriterionID),
		SubjectType: domain.ProofSubjectType(row.SubjectType), SubjectID: row.SubjectID, SubjectRevision: row.SubjectRevision,
		Kind: domain.EvidenceKind(row.Kind), SourceType: domain.EvidenceSourceType(row.SourceType), SourceRef: row.SourceRef,
		ProducerType: domain.EvidenceProducerType(row.ProducerType), ProducerRef: row.ProducerRef, Summary: row.Summary,
		ContentDigest: row.ContentDigest, RequestKey: row.RequestKey, RequestFingerprint: row.RequestFingerprint, CreatedAt: row.CreatedAt,
	}
}

func verificationFromRow(row gen.VerificationRun) (domain.VerificationRun, error) {
	var ids []domain.EvidenceItemID
	if err := json.Unmarshal([]byte(row.EvidenceItemIds), &ids); err != nil {
		return domain.VerificationRun{}, fmt.Errorf("verification %s evidence ids: %w", row.ID, err)
	}
	return domain.VerificationRun{
		ID: domain.VerificationRunID(row.ID), OutcomeID: domain.OutcomeID(row.OutcomeID),
		ContractRevisionID: domain.ContractRevisionID(row.ContractRevisionID), CriterionID: domain.CriterionID(row.CriterionID),
		SubjectType: domain.ProofSubjectType(row.SubjectType), SubjectID: row.SubjectID, SubjectRevision: row.SubjectRevision,
		EvidenceItemIDs: ids, Method: row.Method, IndependenceClass: domain.VerificationIndependenceClass(row.IndependenceClass),
		Result: domain.VerificationResult(row.Result), ProducerRef: row.ProducerRef, VerifierRef: row.VerifierRef,
		ProducerProvider: row.ProducerProvider, VerifierProvider: row.VerifierProvider, Detail: row.Detail,
		RequestKey: row.RequestKey, RequestFingerprint: row.RequestFingerprint, CreatedAt: row.CreatedAt,
	}, nil
}

func acceptanceFromRow(row gen.AcceptanceDecision) domain.AcceptanceDecision {
	return domain.AcceptanceDecision{
		ID: domain.AcceptanceDecisionID(row.ID), OutcomeID: domain.OutcomeID(row.OutcomeID),
		ContractRevisionID: domain.ContractRevisionID(row.ContractRevisionID), Kind: domain.AcceptanceDecisionKind(row.Kind),
		ActorType: domain.AcceptanceActorType(row.ActorType), Summary: row.Summary,
		ResourceDisposition: domain.ResourceDisposition(row.ResourceDisposition), RequestKey: row.RequestKey,
		RequestFingerprint: row.RequestFingerprint, CreatedAt: row.CreatedAt,
	}
}

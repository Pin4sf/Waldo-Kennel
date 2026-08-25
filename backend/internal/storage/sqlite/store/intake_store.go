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
)

// CreateIntake atomically persists exact intent, provenance refs, and idempotency.
func (s *Store) CreateIntake(ctx context.Context, session domain.IntakeSession, refs []domain.IntakeConversationRef, request ports.IntakeIdempotency) (ports.IntakeSnapshot, error) {
	if err := session.Validate(); err != nil {
		return ports.IntakeSnapshot{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if replay, err := s.qw.FindIntakeByRequestKey(ctx, request.Key); err == nil {
		if replay.RequestFingerprint != request.Fingerprint {
			return ports.IntakeSnapshot{}, &ports.IntakeIdempotencyConflictError{Key: request.Key}
		}
		return s.intakeSnapshot(ctx, s.qw, replay)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ports.IntakeSnapshot{}, fmt.Errorf("find intake replay: %w", err)
	}
	err := s.inTx(ctx, "create intake", func(q *gen.Queries) error {
		if err := q.CreateIntakeSession(ctx, gen.CreateIntakeSessionParams{
			ID: session.ID.String(), SourceSurface: string(session.SourceSurface), Purpose: string(session.Purpose),
			ProjectID: nullString(string(session.ProjectID)), SourceOpenLoopID: nullString(session.SourceOpenLoopID.String()),
			Statement: session.Statement, Status: string(session.Status), CurrentProposalRevision: session.CurrentProposalRevision,
			ClarificationCount: session.ClarificationCount, ConfirmedOutcomeID: nullString(session.ConfirmedOutcomeID.String()),
			FailureCode: session.FailureCode, CancellationReason: session.CancellationReason,
			RequestKey: request.Key, RequestFingerprint: request.Fingerprint, CreatedAt: session.CreatedAt, UpdatedAt: session.UpdatedAt,
		}); err != nil {
			return err
		}
		for _, ref := range refs {
			if err := q.CreateIntakeConversationRef(ctx, gen.CreateIntakeConversationRefParams{IntakeID: session.ID.String(), EpisodeID: ref.EpisodeID, TurnID: ref.TurnID, Position: ref.Position}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	return s.intakeSnapshot(ctx, s.qw, gen.IntakeSession{
		ID: session.ID.String(), SourceSurface: string(session.SourceSurface), Purpose: string(session.Purpose), ProjectID: nullString(string(session.ProjectID)),
		SourceOpenLoopID: nullString(session.SourceOpenLoopID.String()), Statement: session.Statement, Status: string(session.Status),
		CurrentProposalRevision: session.CurrentProposalRevision, ClarificationCount: session.ClarificationCount,
		ConfirmedOutcomeID: nullString(session.ConfirmedOutcomeID.String()), FailureCode: session.FailureCode, CancellationReason: session.CancellationReason,
		RequestKey: request.Key, RequestFingerprint: request.Fingerprint, CreatedAt: session.CreatedAt, UpdatedAt: session.UpdatedAt,
	})
}

// GetIntake reconstructs one intake snapshot from durable rows.
func (s *Store) GetIntake(ctx context.Context, id domain.IntakeSessionID) (ports.IntakeSnapshot, bool, error) {
	row, err := s.qr.GetIntakeSession(ctx, id.String())
	if errors.Is(err, sql.ErrNoRows) {
		return ports.IntakeSnapshot{}, false, nil
	}
	if err != nil {
		return ports.IntakeSnapshot{}, false, fmt.Errorf("get intake %s: %w", id, err)
	}
	snapshot, err := s.intakeSnapshot(ctx, s.qr, row)
	return snapshot, err == nil, err
}

// BeginIntakeAnalysis enters analyzing under optimistic concurrency.
func (s *Store) BeginIntakeAnalysis(ctx context.Context, id domain.IntakeSessionID, expected int64, at time.Time) (ports.IntakeSnapshot, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	row, err := s.qw.GetIntakeSession(ctx, id.String())
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	rows, err := s.qw.UpdateIntakeAnalysisState(ctx, gen.UpdateIntakeAnalysisStateParams{Status: string(domain.IntakeStatusAnalyzing), UpdatedAt: at, ID: id.String(), CurrentProposalRevision: expected, Status_2: row.Status})
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	if rows != 1 {
		return ports.IntakeSnapshot{}, revisionConflict(row, id, expected)
	}
	row.Status, row.UpdatedAt = string(domain.IntakeStatusAnalyzing), at
	return s.intakeSnapshot(ctx, s.qw, row)
}

// CompleteIntakeWithProposal appends the analyzer's immutable proposal.
func (s *Store) CompleteIntakeWithProposal(ctx context.Context, id domain.IntakeSessionID, expected int64, proposal domain.OutcomeContractProposal, at time.Time) (ports.IntakeSnapshot, error) {
	return s.appendProposal(ctx, id, expected, proposal, at)
}

// AppendIntakeProposalRevision persists one user-authored immutable revision.
func (s *Store) AppendIntakeProposalRevision(ctx context.Context, id domain.IntakeSessionID, expected int64, proposal domain.OutcomeContractProposal, at time.Time) (ports.IntakeSnapshot, error) {
	return s.appendProposal(ctx, id, expected, proposal, at)
}

func (s *Store) appendProposal(ctx context.Context, id domain.IntakeSessionID, expected int64, proposal domain.OutcomeContractProposal, at time.Time) (ports.IntakeSnapshot, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	err := s.inTx(ctx, "append intake proposal", func(q *gen.Queries) error {
		row, err := q.GetIntakeSession(ctx, id.String())
		if err != nil {
			return err
		}
		if row.CurrentProposalRevision != expected {
			return revisionConflict(row, id, expected)
		}
		if err := insertIntakeProposal(ctx, q, proposal); err != nil {
			return err
		}
		rows, err := q.UpdateIntakeWithProposal(ctx, gen.UpdateIntakeWithProposalParams{CurrentProposalRevision: proposal.Revision, UpdatedAt: at, ID: id.String(), CurrentProposalRevision_2: expected})
		if err != nil {
			return err
		}
		if rows != 1 {
			return revisionConflict(row, id, expected)
		}
		return nil
	})
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	row, err := s.qw.GetIntakeSession(ctx, id.String())
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	return s.intakeSnapshot(ctx, s.qw, row)
}

// CompleteIntakeWithClarification persists the only allowed material question.
func (s *Store) CompleteIntakeWithClarification(ctx context.Context, id domain.IntakeSessionID, expected int64, clarification domain.ClarificationRequest, at time.Time) (ports.IntakeSnapshot, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	alternatives, _ := json.Marshal(clarification.Alternatives)
	err := s.inTx(ctx, "record intake clarification", func(q *gen.Queries) error {
		row, err := q.GetIntakeSession(ctx, id.String())
		if err != nil {
			return err
		}
		if row.CurrentProposalRevision != expected {
			return revisionConflict(row, id, expected)
		}
		if err := q.CreateIntakeClarification(ctx, gen.CreateIntakeClarificationParams{ID: string(clarification.ID), IntakeID: id.String(), Question: clarification.Question, Reason: clarification.Reason, Recommendation: clarification.Recommendation, Alternatives: string(alternatives), DeferralConsequence: clarification.DeferralConsequence, CreatedAt: clarification.CreatedAt}); err != nil {
			return err
		}
		rows, err := q.UpdateIntakeWithClarification(ctx, gen.UpdateIntakeWithClarificationParams{UpdatedAt: at, ID: id.String(), CurrentProposalRevision: expected})
		if err != nil {
			return err
		}
		if rows != 1 {
			return fmt.Errorf("intake %s cannot accept another clarification", id)
		}
		return nil
	})
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	row, err := s.qw.GetIntakeSession(ctx, id.String())
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	return s.intakeSnapshot(ctx, s.qw, row)
}

// AnswerIntakeClarification stores the answer and resumes analysis.
func (s *Store) AnswerIntakeClarification(ctx context.Context, id domain.IntakeSessionID, expected int64, answer string, at time.Time) (ports.IntakeSnapshot, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	err := s.inTx(ctx, "answer intake clarification", func(q *gen.Queries) error {
		row, err := q.GetIntakeSession(ctx, id.String())
		if err != nil {
			return err
		}
		if row.CurrentProposalRevision != expected {
			return revisionConflict(row, id, expected)
		}
		clarification, err := q.GetIntakeClarification(ctx, id.String())
		if err != nil {
			return err
		}
		if err := q.CreateIntakeClarificationAnswer(ctx, gen.CreateIntakeClarificationAnswerParams{ClarificationID: clarification.ID, Answer: answer, AnsweredAt: at}); err != nil {
			return err
		}
		rows, err := q.UpdateIntakeAnalysisState(ctx, gen.UpdateIntakeAnalysisStateParams{Status: string(domain.IntakeStatusAnalyzing), UpdatedAt: at, ID: id.String(), CurrentProposalRevision: expected, Status_2: string(domain.IntakeStatusNeedsUser)})
		if err != nil {
			return err
		}
		if rows != 1 {
			return fmt.Errorf("intake %s is not waiting for an answer", id)
		}
		return nil
	})
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	row, err := s.qw.GetIntakeSession(ctx, id.String())
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	return s.intakeSnapshot(ctx, s.qw, row)
}

// FailIntakeAnalysis preserves retryable failure truth on the intake.
func (s *Store) FailIntakeAnalysis(ctx context.Context, id domain.IntakeSessionID, expected int64, code string, at time.Time) (ports.IntakeSnapshot, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	row, err := s.qw.GetIntakeSession(ctx, id.String())
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	rows, err := s.qw.FailIntakeAnalysis(ctx, gen.FailIntakeAnalysisParams{FailureCode: code, UpdatedAt: at, ID: id.String(), CurrentProposalRevision: expected})
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	if rows != 1 {
		return ports.IntakeSnapshot{}, revisionConflict(row, id, expected)
	}
	row, err = s.qw.GetIntakeSession(ctx, id.String())
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	return s.intakeSnapshot(ctx, s.qw, row)
}

// ConfirmIntakeWithOutcome atomically creates exactly one Outcome and ContractRevision.
func (s *Store) ConfirmIntakeWithOutcome(ctx context.Context, id domain.IntakeSessionID, expected int64, outcome domain.Outcome, contract domain.ContractRevision, request ports.IntakeIdempotency, at time.Time) (ports.IntakeSnapshot, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if confirmation, err := s.qw.FindIntakeConfirmationByRequestKey(ctx, request.Key); err == nil {
		if confirmation.RequestFingerprint != request.Fingerprint {
			return ports.IntakeSnapshot{}, &ports.IntakeIdempotencyConflictError{Key: request.Key}
		}
		row, getErr := s.qw.GetIntakeSession(ctx, confirmation.IntakeID)
		if getErr != nil {
			return ports.IntakeSnapshot{}, getErr
		}
		return s.intakeSnapshot(ctx, s.qw, row)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ports.IntakeSnapshot{}, err
	}
	err := s.inTx(ctx, "confirm intake outcome", func(q *gen.Queries) error {
		row, err := q.GetIntakeSession(ctx, id.String())
		if err != nil {
			return err
		}
		if row.CurrentProposalRevision != expected {
			return revisionConflict(row, id, expected)
		}
		if row.Status != string(domain.IntakeStatusReady) {
			return fmt.Errorf("intake %s is not ready", id)
		}
		if err := outcome.Validate(); err != nil {
			return err
		}
		if err := q.CreateOutcome(ctx, gen.CreateOutcomeParams{ID: outcome.ID, SpaceID: outcome.SpaceID, Title: outcome.Title, CurrentRevisionNumber: 0}); err != nil {
			return err
		}
		contract.Number = 1
		if err := insertContractRevision(ctx, q, contract); err != nil {
			return err
		}
		if err := insertContractIntakeCore(ctx, q, contract); err != nil {
			return err
		}
		rows, err := q.AdvanceOutcomeCurrentRevision(ctx, gen.AdvanceOutcomeCurrentRevisionParams{CurrentRevisionNumber: 1, UpdatedAt: at, ID: outcome.ID, CurrentRevisionNumber_2: 0})
		if err != nil {
			return err
		}
		if rows != 1 {
			return fmt.Errorf("outcome %s revision pointer did not advance", outcome.ID)
		}
		if err := q.CreateIntakeConfirmation(ctx, gen.CreateIntakeConfirmationParams{IntakeID: id.String(), ProposalRevision: expected, OutcomeID: outcome.ID.String(), ContractRevisionID: contract.ID.String(), RequestKey: request.Key, RequestFingerprint: request.Fingerprint, ConfirmedAt: at}); err != nil {
			return err
		}
		rows, err = q.ConfirmIntake(ctx, gen.ConfirmIntakeParams{ConfirmedOutcomeID: nullString(outcome.ID.String()), UpdatedAt: at, ID: id.String(), CurrentProposalRevision: expected})
		if err != nil {
			return err
		}
		if rows != 1 {
			return revisionConflict(row, id, expected)
		}
		return nil
	})
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	row, err := s.qw.GetIntakeSession(ctx, id.String())
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	return s.intakeSnapshot(ctx, s.qw, row)
}

func (s *Store) intakeSnapshot(ctx context.Context, q *gen.Queries, row gen.IntakeSession) (ports.IntakeSnapshot, error) {
	snapshot := ports.IntakeSnapshot{Session: intakeSessionFromRow(row)}
	refs, err := q.ListIntakeConversationRefs(ctx, row.ID)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	for _, ref := range refs {
		snapshot.ConversationRefs = append(snapshot.ConversationRefs, domain.IntakeConversationRef{EpisodeID: ref.EpisodeID, TurnID: ref.TurnID, Position: ref.Position})
	}
	if proposalRow, err := q.GetLatestIntakeProposal(ctx, row.ID); err == nil {
		proposal, err := intakeProposalFromRow(proposalRow)
		if err != nil {
			return ports.IntakeSnapshot{}, err
		}
		snapshot.Proposal = &proposal
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ports.IntakeSnapshot{}, err
	}
	if clarificationRow, err := q.GetIntakeClarification(ctx, row.ID); err == nil {
		clarification, err := intakeClarificationFromRow(clarificationRow)
		if err != nil {
			return ports.IntakeSnapshot{}, err
		}
		snapshot.Clarification = &clarification
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ports.IntakeSnapshot{}, err
	}
	if row.ConfirmedOutcomeID.Valid {
		outcomeRow, err := q.GetOutcome(ctx, domain.OutcomeID(row.ConfirmedOutcomeID.String))
		if err != nil {
			return ports.IntakeSnapshot{}, err
		}
		outcome := outcomeFromRow(outcomeRow)
		snapshot.ConfirmedOutcome = &outcome
		confirmation, err := q.GetIntakeConfirmation(ctx, row.ID)
		if err != nil {
			return ports.IntakeSnapshot{}, err
		}
		contractRow, err := q.GetContractRevision(ctx, domain.ContractRevisionID(confirmation.ContractRevisionID))
		if err != nil {
			return ports.IntakeSnapshot{}, err
		}
		contract, err := contractRevisionFromRow(contractRow)
		if err != nil {
			return ports.IntakeSnapshot{}, err
		}
		criteria, err := q.ListContractCriteriaForRevision(ctx, string(contract.ID))
		if err != nil {
			return ports.IntakeSnapshot{}, err
		}
		for _, criterion := range criteria {
			contract.Criteria = append(contract.Criteria, domain.ContractCriterion{ID: domain.CriterionID(criterion.ID), ContractRevisionID: domain.ContractRevisionID(criterion.ContractRevisionID), Position: criterion.Position, Text: criterion.Text})
		}
		core, err := q.GetContractRevisionIntakeCore(ctx, contract.ID.String())
		if err != nil {
			return ports.IntakeSnapshot{}, err
		}
		if err := applyContractIntakeCore(&contract, core); err != nil {
			return ports.IntakeSnapshot{}, err
		}
		snapshot.ConfirmedContract = &contract
	}
	return snapshot, nil
}

func intakeSessionFromRow(row gen.IntakeSession) domain.IntakeSession {
	return domain.IntakeSession{ID: domain.IntakeSessionID(row.ID), SourceSurface: domain.IntakeSourceSurface(row.SourceSurface), Purpose: domain.IntakePurpose(row.Purpose), ProjectID: domain.ProjectID(row.ProjectID.String), SourceOpenLoopID: domain.OpenLoopID(row.SourceOpenLoopID.String), Statement: row.Statement, Status: domain.IntakeStatus(row.Status), CurrentProposalRevision: row.CurrentProposalRevision, ClarificationCount: row.ClarificationCount, ConfirmedOutcomeID: domain.OutcomeID(row.ConfirmedOutcomeID.String), FailureCode: row.FailureCode, CancellationReason: row.CancellationReason, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func insertIntakeProposal(ctx context.Context, q *gen.Queries, proposal domain.OutcomeContractProposal) error {
	criteria, _ := json.Marshal(proposal.Criteria)
	constraints, _ := json.Marshal(proposal.Constraints)
	nonGoals, _ := json.Marshal(proposal.NonGoals)
	authority, _ := json.Marshal(proposal.AuthorityCeiling)
	stops, _ := json.Marshal(proposal.StopConditions)
	notes, _ := json.Marshal(proposal.ClarificationNotes)
	facets, _ := json.Marshal(proposal.Facets)
	var temporal sql.NullString
	if proposal.TemporalCondition != nil {
		temporal = nullString(*proposal.TemporalCondition)
	}
	return q.CreateIntakeProposal(ctx, gen.CreateIntakeProposalParams{ID: string(proposal.ID), IntakeID: proposal.IntakeID.String(), Revision: proposal.Revision, Title: proposal.Title, DesiredState: proposal.DesiredState, Criteria: string(criteria), ReviewMethod: proposal.ReviewMethod, Constraints: string(constraints), NonGoals: string(nonGoals), AuthorityCeiling: string(authority), StopConditions: string(stops), ClarificationNotes: string(notes), TemporalCondition: temporal, Facets: string(facets), CreatedAt: proposal.CreatedAt})
}

func intakeProposalFromRow(row gen.IntakeProposalRevision) (domain.OutcomeContractProposal, error) {
	proposal := domain.OutcomeContractProposal{ID: domain.ProposalRevisionID(row.ID), IntakeID: domain.IntakeSessionID(row.IntakeID), Revision: row.Revision, Title: row.Title, DesiredState: row.DesiredState, ReviewMethod: row.ReviewMethod, CreatedAt: row.CreatedAt}
	fields := []struct {
		data   string
		target any
	}{
		{row.Criteria, &proposal.Criteria}, {row.Constraints, &proposal.Constraints},
		{row.NonGoals, &proposal.NonGoals}, {row.AuthorityCeiling, &proposal.AuthorityCeiling},
		{row.StopConditions, &proposal.StopConditions}, {row.ClarificationNotes, &proposal.ClarificationNotes},
		{row.Facets, &proposal.Facets},
	}
	for _, field := range fields {
		if err := json.Unmarshal([]byte(field.data), field.target); err != nil {
			return domain.OutcomeContractProposal{}, err
		}
	}
	if row.TemporalCondition.Valid {
		value := row.TemporalCondition.String
		proposal.TemporalCondition = &value
	}
	return proposal, nil
}

func intakeClarificationFromRow(row gen.GetIntakeClarificationRow) (domain.ClarificationRequest, error) {
	var alternatives []string
	if err := json.Unmarshal([]byte(row.Alternatives), &alternatives); err != nil {
		return domain.ClarificationRequest{}, err
	}
	clarification := domain.ClarificationRequest{ID: domain.ClarificationRequestID(row.ID), IntakeID: domain.IntakeSessionID(row.IntakeID), Question: row.Question, Reason: row.Reason, Recommendation: row.Recommendation, Alternatives: alternatives, DeferralConsequence: row.DeferralConsequence, Answer: row.Answer.String, CreatedAt: row.CreatedAt}
	if row.AnsweredAt.Valid {
		value := row.AnsweredAt.Time
		clarification.AnsweredAt = &value
	}
	return clarification, nil
}

func revisionConflict(row gen.IntakeSession, id domain.IntakeSessionID, expected int64) error {
	return &ports.IntakeRevisionConflictError{IntakeID: id, ExpectedRevision: expected, CurrentRevision: row.CurrentProposalRevision}
}
func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func insertContractIntakeCore(ctx context.Context, q *gen.Queries, contract domain.ContractRevision) error {
	evidence, err := json.Marshal(contract.EvidenceExpectations)
	if err != nil {
		return err
	}
	authority, err := json.Marshal(contract.AuthorityCeiling)
	if err != nil {
		return err
	}
	stops, err := json.Marshal(contract.StopConditions)
	if err != nil {
		return err
	}
	facets, err := json.Marshal(contract.Facets)
	if err != nil {
		return err
	}
	var temporal sql.NullString
	if contract.TemporalCondition != nil {
		temporal = nullString(*contract.TemporalCondition)
	}
	return q.CreateContractRevisionIntakeCore(ctx, gen.CreateContractRevisionIntakeCoreParams{ContractRevisionID: contract.ID.String(), EvidenceExpectations: string(evidence), AuthorityCeiling: string(authority), StopConditions: string(stops), TemporalCondition: temporal, Facets: string(facets), CreatedAt: contract.CreatedAt})
}

func applyContractIntakeCore(contract *domain.ContractRevision, core gen.ContractRevisionIntakeCore) error {
	fields := []struct {
		data   string
		target any
	}{{core.EvidenceExpectations, &contract.EvidenceExpectations}, {core.AuthorityCeiling, &contract.AuthorityCeiling}, {core.StopConditions, &contract.StopConditions}, {core.Facets, &contract.Facets}}
	for _, field := range fields {
		if err := json.Unmarshal([]byte(field.data), field.target); err != nil {
			return err
		}
	}
	if core.TemporalCondition.Valid {
		value := core.TemporalCondition.String
		contract.TemporalCondition = &value
	}
	return nil
}

// CreateResponsibilityLink atomically creates idempotent explicit lineage.
func (s *Store) CreateResponsibilityLink(ctx context.Context, link domain.ResponsibilityLink, request ports.ResponsibilityLinkIdempotency) (domain.ResponsibilityLink, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if replay, err := s.qw.FindResponsibilityLinkByRequestKey(ctx, request.Key); err == nil {
		if replay.RequestFingerprint != request.Fingerprint {
			return domain.ResponsibilityLink{}, &ports.IntakeIdempotencyConflictError{Key: request.Key}
		}
		return responsibilityLinkFromRow(replay), nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.ResponsibilityLink{}, err
	}
	if err := link.Validate(); err != nil {
		return domain.ResponsibilityLink{}, err
	}
	if existing, err := s.qw.FindActiveResponsibilityLinkPair(ctx, gen.FindActiveResponsibilityLinkPairParams{SourceOpenLoopID: link.SourceOpenLoopID.String(), DestinationOutcomeID: link.DestinationOutcomeID.String()}); err == nil {
		return domain.ResponsibilityLink{}, &ports.ResponsibilityLinkDuplicateError{SourceOpenLoopID: domain.OpenLoopID(existing.SourceOpenLoopID), DestinationOutcomeID: domain.OutcomeID(existing.DestinationOutcomeID)}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.ResponsibilityLink{}, err
	}
	projectID, found, err := s.GetOutcomeProjectID(ctx, link.DestinationOutcomeID)
	if err != nil {
		return domain.ResponsibilityLink{}, err
	}
	if !found {
		return domain.ResponsibilityLink{}, fmt.Errorf("destination outcome %s does not exist", link.DestinationOutcomeID)
	}
	if err := s.qw.CreateResponsibilityLink(ctx, gen.CreateResponsibilityLinkParams{ID: link.ID.String(), ProjectID: string(projectID), SourceOpenLoopID: link.SourceOpenLoopID.String(), DestinationOutcomeID: link.DestinationOutcomeID.String(), Creator: string(link.Creator), Reason: link.Reason, RequestKey: request.Key, RequestFingerprint: request.Fingerprint, CreatedAt: link.CreatedAt}); err != nil {
		return domain.ResponsibilityLink{}, err
	}
	return link, nil
}

// GetResponsibilityLink reads one lineage record.
func (s *Store) GetResponsibilityLink(ctx context.Context, id domain.ResponsibilityLinkID) (domain.ResponsibilityLink, bool, error) {
	row, err := s.qr.GetResponsibilityLink(ctx, id.String())
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ResponsibilityLink{}, false, nil
	}
	if err != nil {
		return domain.ResponsibilityLink{}, false, err
	}
	return responsibilityLinkFromRow(row), true, nil
}

// EndResponsibilityLink records the one allowed lineage end transition.
func (s *Store) EndResponsibilityLink(ctx context.Context, id domain.ResponsibilityLinkID, actor domain.ResponsibilityLinkCreator, reason string, at time.Time) (domain.ResponsibilityLink, bool, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.qw.EndResponsibilityLink(ctx, gen.EndResponsibilityLinkParams{EndedAt: sql.NullTime{Time: at, Valid: true}, EndedBy: string(actor), EndedReason: reason, ID: id.String()})
	if err != nil {
		return domain.ResponsibilityLink{}, false, err
	}
	if rows == 0 {
		if _, err := s.qw.GetResponsibilityLink(ctx, id.String()); errors.Is(err, sql.ErrNoRows) {
			return domain.ResponsibilityLink{}, false, nil
		}
		return domain.ResponsibilityLink{}, true, fmt.Errorf("responsibility link %s is already ended", id)
	}
	row, err := s.qw.GetResponsibilityLink(ctx, id.String())
	if err != nil {
		return domain.ResponsibilityLink{}, true, err
	}
	return responsibilityLinkFromRow(row), true, nil
}

func responsibilityLinkFromRow(row gen.ResponsibilityLink) domain.ResponsibilityLink {
	link := domain.ResponsibilityLink{ID: domain.ResponsibilityLinkID(row.ID), SourceOpenLoopID: domain.OpenLoopID(row.SourceOpenLoopID), DestinationOutcomeID: domain.OutcomeID(row.DestinationOutcomeID), Creator: domain.ResponsibilityLinkCreator(row.Creator), Reason: row.Reason, CreatedAt: row.CreatedAt, EndedBy: domain.ResponsibilityLinkCreator(row.EndedBy), EndedReason: row.EndedReason}
	if row.EndedAt.Valid {
		value := row.EndedAt.Time
		link.EndedAt = &value
	}
	return link
}

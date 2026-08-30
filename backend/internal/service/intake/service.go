// Package intake owns the one shared Home/Work adaptive understanding flow.
package intake

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// CaptureInput contains one statement and identifier-only provenance.
type CaptureInput struct {
	SourceSurface    domain.IntakeSourceSurface
	Purpose          domain.IntakePurpose
	ProjectID        domain.ProjectID
	SourceOpenLoopID domain.OpenLoopID
	Statement        string
	ConversationRefs []domain.IntakeConversationRef
	RequestKey       string
}

// AnalyzeInput guards analyzer work with an expected revision.
type AnalyzeInput struct {
	ExpectedProposalRevision int64
}

// AnswerClarificationInput answers the at-most-one material question.
type AnswerClarificationInput struct {
	ExpectedProposalRevision int64
	Answer                   string
}

// ReviseProposalInput appends a user-reviewed immutable proposal.
type ReviseProposalInput struct {
	ExpectedProposalRevision int64
	Proposal                 domain.OutcomeContractProposal
}

// ConfirmOutcomeInput explicitly confirms one proposal revision.
type ConfirmOutcomeInput struct {
	ExpectedProposalRevision int64
	RequestKey               string
}

// CancelInput consciously releases intake without creating responsibility.
type CancelInput struct {
	ExpectedProposalRevision int64
	Reason                   string
}

// Service owns the shared Home/Work adaptive intake state machine.
type Service struct {
	store    ports.IntakeStore
	analyzer ports.IntakeAnalyzer
	clock    func() time.Time
}

// New constructs the shared adaptive intake service.
func New(store ports.IntakeStore, analyzer ports.IntakeAnalyzer, clock func() time.Time) *Service {
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, analyzer: analyzer, clock: clock}
}

// Get returns one durable intake snapshot.
func (service *Service) Get(ctx context.Context, id domain.IntakeSessionID) (ports.IntakeSnapshot, error) {
	snapshot, found, err := service.store.GetIntake(ctx, id)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	if !found {
		return ports.IntakeSnapshot{}, apierr.NotFound("INTAKE_NOT_FOUND", "That intake does not exist")
	}
	return snapshot, nil
}

// RecoverInterruptedAnalyses turns crash-interrupted transient work into an
// explicit retryable durable failure during daemon startup.
func (service *Service) RecoverInterruptedAnalyses(ctx context.Context) (int64, error) {
	recovered, err := service.store.RecoverInterruptedIntakeAnalyses(ctx, service.clock())
	if err != nil {
		return 0, apierr.Internal("INTAKE_ANALYSIS_RECOVERY_FAILED", "Interrupted Outcome intake analysis could not be recovered")
	}
	return recovered, nil
}

// Capture persists exact intent before any analysis occurs.
func (service *Service) Capture(ctx context.Context, input CaptureInput) (ports.IntakeSnapshot, error) {
	input.Statement = strings.TrimSpace(input.Statement)
	input.RequestKey = strings.TrimSpace(input.RequestKey)
	if input.ProjectID == "" && input.Purpose == domain.IntakePurposeOutcome {
		return ports.IntakeSnapshot{}, apierr.Invalid("PROJECT_REQUIRED", "Choose the project this Outcome belongs to", nil)
	}
	if input.Statement == "" {
		return ports.IntakeSnapshot{}, apierr.Invalid("INTAKE_STATEMENT_REQUIRED", "Describe what you would like to make true", nil)
	}
	if input.RequestKey == "" {
		return ports.IntakeSnapshot{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this intake", nil)
	}
	if input.Purpose == domain.IntakePurposeOutcome {
		if _, found, err := service.store.GetProject(ctx, string(input.ProjectID)); err != nil {
			return ports.IntakeSnapshot{}, err
		} else if !found {
			return ports.IntakeSnapshot{}, apierr.NotFound("PROJECT_NOT_FOUND", "That Project does not exist")
		}
	}
	for _, ref := range input.ConversationRefs {
		if err := ref.Validate(); err != nil {
			return ports.IntakeSnapshot{}, apierr.Invalid("INTAKE_CONVERSATION_REF_INVALID", err.Error(), nil)
		}
	}
	now := service.clock()
	session := domain.IntakeSession{
		ID:               domain.IntakeSessionID("intake-" + uuid.NewString()),
		SourceSurface:    input.SourceSurface,
		Purpose:          input.Purpose,
		ProjectID:        input.ProjectID,
		SourceOpenLoopID: input.SourceOpenLoopID,
		Statement:        input.Statement,
		Status:           domain.IntakeStatusCaptured,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := session.Validate(); err != nil {
		return ports.IntakeSnapshot{}, apierr.Invalid("INTAKE_INVALID", err.Error(), nil)
	}
	request := ports.IntakeIdempotency{
		Key: input.RequestKey,
		Fingerprint: intakeFingerprint(
			string(input.SourceSurface), string(input.Purpose), string(input.ProjectID),
			input.SourceOpenLoopID.String(), input.Statement, conversationFingerprint(input.ConversationRefs),
		),
	}
	return service.store.CreateIntake(ctx, session, input.ConversationRefs, request)
}

// Analyze advances captured intent to one question or an editable proposal.
func (service *Service) Analyze(ctx context.Context, id domain.IntakeSessionID, input AnalyzeInput) (ports.IntakeSnapshot, error) {
	snapshot, found, err := service.store.GetIntake(ctx, id)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	if !found {
		return ports.IntakeSnapshot{}, apierr.NotFound("INTAKE_NOT_FOUND", "That intake does not exist")
	}
	if snapshot.Session.CurrentProposalRevision != input.ExpectedProposalRevision {
		return ports.IntakeSnapshot{}, intakeRevisionConflict(id, input.ExpectedProposalRevision, snapshot.Session.CurrentProposalRevision)
	}
	if !domain.CanTransitionIntake(snapshot.Session.Status, domain.IntakeStatusAnalyzing) {
		return ports.IntakeSnapshot{}, apierr.Conflict("INTAKE_STATE_CONFLICT", "This intake cannot be analyzed from its current state", nil)
	}
	if service.analyzer == nil {
		return ports.IntakeSnapshot{}, apierr.Internal("INTAKE_ANALYZER_UNAVAILABLE", "Outcome analysis is not configured")
	}
	now := service.clock()
	analyzing, err := service.store.BeginIntakeAnalysis(ctx, id, input.ExpectedProposalRevision, now)
	if err != nil {
		return ports.IntakeSnapshot{}, mapStoreError(err)
	}
	ticket, err := service.analyzer.Analyze(ctx, ports.IntakeAnalysisInput{
		Session: analyzing.Session, ConversationRefs: analyzing.ConversationRefs,
		PreviousProposal: analyzing.Proposal, Clarification: analyzing.Clarification,
	})
	if err != nil {
		_, _ = service.store.FailIntakeAnalysis(ctx, id, input.ExpectedProposalRevision, "INTAKE_ANALYSIS_FAILED", service.clock())
		return ports.IntakeSnapshot{}, err
	}
	result, err := service.settleTicket(ctx, id, input.ExpectedProposalRevision, ticket)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	return service.completeAnalysis(ctx, analyzing, input.ExpectedProposalRevision, result, "")
}

// AnswerClarification records the single answer and resumes analysis.
func (service *Service) AnswerClarification(ctx context.Context, id domain.IntakeSessionID, input AnswerClarificationInput) (ports.IntakeSnapshot, error) {
	input.Answer = strings.TrimSpace(input.Answer)
	if input.Answer == "" {
		return ports.IntakeSnapshot{}, apierr.Invalid("INTAKE_CLARIFICATION_ANSWER_REQUIRED", "Answer the material clarification before continuing", nil)
	}
	snapshot, found, err := service.store.GetIntake(ctx, id)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	if !found {
		return ports.IntakeSnapshot{}, apierr.NotFound("INTAKE_NOT_FOUND", "That intake does not exist")
	}
	if snapshot.Session.CurrentProposalRevision != input.ExpectedProposalRevision {
		return ports.IntakeSnapshot{}, intakeRevisionConflict(id, input.ExpectedProposalRevision, snapshot.Session.CurrentProposalRevision)
	}
	if snapshot.Session.Status != domain.IntakeStatusNeedsUser || snapshot.Clarification == nil {
		return ports.IntakeSnapshot{}, apierr.Conflict("INTAKE_STATE_CONFLICT", "This intake is not waiting for a clarification answer", nil)
	}
	if service.analyzer == nil {
		return ports.IntakeSnapshot{}, apierr.Internal("INTAKE_ANALYZER_UNAVAILABLE", "Outcome analysis is not configured")
	}
	now := service.clock()
	analyzing, err := service.store.AnswerIntakeClarification(ctx, id, input.ExpectedProposalRevision, input.Answer, now)
	if err != nil {
		return ports.IntakeSnapshot{}, mapStoreError(err)
	}
	ticket, err := service.analyzer.Analyze(ctx, ports.IntakeAnalysisInput{
		Session: analyzing.Session, ConversationRefs: analyzing.ConversationRefs,
		PreviousProposal: analyzing.Proposal, Clarification: analyzing.Clarification,
		ClarificationText: input.Answer,
	})
	if err != nil {
		_, _ = service.store.FailIntakeAnalysis(ctx, id, input.ExpectedProposalRevision, "INTAKE_ANALYSIS_FAILED", service.clock())
		return ports.IntakeSnapshot{}, err
	}
	result, err := service.settleTicket(ctx, id, input.ExpectedProposalRevision, ticket)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	return service.completeAnalysis(ctx, analyzing, input.ExpectedProposalRevision, result, input.Answer)
}

// settleTicket turns an analyzer's ticket into the result this phase can act
// on, and fails the intake loudly when it cannot.
//
// An analyzer that answers later returns no Inline. Nothing in the tree does
// that yet: the durable request, the callback route, and the awaiting_analyst
// state that such a ticket needs are the NEXT phase of this work. Until they
// exist, a deferred ticket is refused explicitly and the intake is moved to a
// retryable failure, rather than being left parked in `analyzing` where the
// startup sweep would silently reap it and the owner would see an intake that
// simply stopped.
//
// This is the exact seam the next phase replaces; it is deliberately loud so
// it cannot be mistaken for working.
func (service *Service) settleTicket(ctx context.Context, id domain.IntakeSessionID, expectedRevision int64, ticket ports.IntakeAnalysisTicket) (ports.IntakeAnalysisResult, error) {
	if ticket.Inline != nil {
		return *ticket.Inline, nil
	}
	_, _ = service.store.FailIntakeAnalysis(ctx, id, expectedRevision, "INTAKE_ANALYSIS_DEFERRED_UNSUPPORTED", service.clock())
	return ports.IntakeAnalysisResult{}, apierr.Internal("INTAKE_ANALYSIS_DEFERRED_UNSUPPORTED",
		"This analyzer answers later, and receiving a later answer is not built yet")
}

// ReviseProposal appends a user-authored immutable proposal revision. It never
// mutates a prior analyzer or user proposal in place.
func (service *Service) ReviseProposal(ctx context.Context, id domain.IntakeSessionID, input ReviseProposalInput) (ports.IntakeSnapshot, error) {
	snapshot, found, err := service.store.GetIntake(ctx, id)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	if !found {
		return ports.IntakeSnapshot{}, apierr.NotFound("INTAKE_NOT_FOUND", "That intake does not exist")
	}
	if snapshot.Session.CurrentProposalRevision != input.ExpectedProposalRevision {
		return ports.IntakeSnapshot{}, intakeRevisionConflict(id, input.ExpectedProposalRevision, snapshot.Session.CurrentProposalRevision)
	}
	if snapshot.Session.Status != domain.IntakeStatusReady {
		return ports.IntakeSnapshot{}, apierr.Conflict("INTAKE_STATE_CONFLICT", "This intake does not have a proposal that can be revised", nil)
	}
	proposal := input.Proposal
	proposal.ID = domain.ProposalRevisionID("proposal-" + uuid.NewString())
	proposal.IntakeID = id
	proposal.Revision = input.ExpectedProposalRevision + 1
	proposal.CreatedAt = service.clock()
	for index := range proposal.Criteria {
		if strings.TrimSpace(string(proposal.Criteria[index].ID)) == "" {
			proposal.Criteria[index].ID = domain.ProposedCriterionID("proposal-criterion-" + uuid.NewString())
		}
	}
	if err := proposal.Validate(); err != nil {
		return ports.IntakeSnapshot{}, apierr.Invalid("INTAKE_PROPOSAL_INVALID", err.Error(), nil)
	}
	revised, err := service.store.AppendIntakeProposalRevision(ctx, id, input.ExpectedProposalRevision, proposal, proposal.CreatedAt)
	if err != nil {
		return ports.IntakeSnapshot{}, mapStoreError(err)
	}
	return revised, nil
}

// Cancel consciously releases an unconfirmed intake without creating an Outcome.
func (service *Service) Cancel(ctx context.Context, id domain.IntakeSessionID, input CancelInput) (ports.IntakeSnapshot, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Reason == "" {
		return ports.IntakeSnapshot{}, apierr.Invalid("INTAKE_CANCELLATION_REASON_REQUIRED", "Say why this intake is being released", nil)
	}
	snapshot, found, err := service.store.GetIntake(ctx, id)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	if !found {
		return ports.IntakeSnapshot{}, apierr.NotFound("INTAKE_NOT_FOUND", "That intake does not exist")
	}
	if snapshot.Session.CurrentProposalRevision != input.ExpectedProposalRevision {
		return ports.IntakeSnapshot{}, intakeRevisionConflict(id, input.ExpectedProposalRevision, snapshot.Session.CurrentProposalRevision)
	}
	if !domain.CanTransitionIntake(snapshot.Session.Status, domain.IntakeStatusCancelled) {
		return ports.IntakeSnapshot{}, apierr.Conflict("INTAKE_STATE_CONFLICT", "This intake cannot be cancelled from its current state", nil)
	}
	cancelled, err := service.store.CancelIntake(ctx, id, input.ExpectedProposalRevision, input.Reason, service.clock())
	if err != nil {
		return ports.IntakeSnapshot{}, mapStoreError(err)
	}
	return cancelled, nil
}

// ConfirmOutcome compiles the latest reviewed proposal into exactly one
// canonical Outcome and ContractRevision. The store owns the atomic write and
// idempotency fence.
func (service *Service) ConfirmOutcome(ctx context.Context, id domain.IntakeSessionID, input ConfirmOutcomeInput) (ports.IntakeSnapshot, error) {
	input.RequestKey = strings.TrimSpace(input.RequestKey)
	if input.RequestKey == "" {
		return ports.IntakeSnapshot{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this confirmation", nil)
	}
	snapshot, found, err := service.store.GetIntake(ctx, id)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	if !found {
		return ports.IntakeSnapshot{}, apierr.NotFound("INTAKE_NOT_FOUND", "That intake does not exist")
	}
	if snapshot.Session.Status != domain.IntakeStatusConfirmed && snapshot.Session.CurrentProposalRevision != input.ExpectedProposalRevision {
		return ports.IntakeSnapshot{}, intakeRevisionConflict(id, input.ExpectedProposalRevision, snapshot.Session.CurrentProposalRevision)
	}
	if snapshot.Session.Status != domain.IntakeStatusReady && snapshot.Session.Status != domain.IntakeStatusConfirmed {
		return ports.IntakeSnapshot{}, apierr.Conflict("INTAKE_STATE_CONFLICT", "This intake is not ready for confirmation", nil)
	}
	if snapshot.Proposal == nil {
		return ports.IntakeSnapshot{}, apierr.Conflict("INTAKE_PROPOSAL_MISSING", "This intake has no Contract proposal to confirm", nil)
	}
	space, err := service.store.EnsureWorkResponsibilitySpace(ctx, snapshot.Session.ProjectID)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	now := service.clock()
	outcome := domain.Outcome{
		ID: domain.OutcomeID("out-" + uuid.NewString()), SpaceID: space.ID,
		Title: snapshot.Proposal.Title, CurrentRevisionNumber: 1, CreatedAt: now, UpdatedAt: now,
	}
	contract := contractFromProposal(outcome.ID, *snapshot.Proposal, now)
	request := ports.IntakeIdempotency{Key: input.RequestKey, Fingerprint: intakeFingerprint(id.String(), fmt.Sprintf("%d", input.ExpectedProposalRevision))}
	confirmed, err := service.store.ConfirmIntakeWithOutcome(ctx, id, input.ExpectedProposalRevision, outcome, contract, request, now)
	if err != nil {
		return ports.IntakeSnapshot{}, mapStoreError(err)
	}
	return confirmed, nil
}

func contractFromProposal(outcomeID domain.OutcomeID, proposal domain.OutcomeContractProposal, at time.Time) domain.ContractRevision {
	contract := domain.ContractRevision{
		ID: domain.ContractRevisionID("cr-" + uuid.NewString()), OutcomeID: outcomeID, Number: 1,
		Goal: proposal.DesiredState, Review: proposal.ReviewMethod,
		Constraints: append([]string(nil), proposal.Constraints...), NonGoals: append([]string(nil), proposal.NonGoals...),
		Clarification: strings.Join(proposal.ClarificationNotes, "\n"), AuthorityCeiling: proposal.AuthorityCeiling,
		StopConditions: append([]string(nil), proposal.StopConditions...), TemporalCondition: proposal.TemporalCondition,
		Facets: append([]domain.ContractFacet(nil), proposal.Facets...), CreatedAt: at,
	}
	for index, proposed := range proposal.Criteria {
		criterion := domain.ContractCriterion{
			ID: domain.CriterionID("crit-" + uuid.NewString()), ContractRevisionID: contract.ID,
			Position: int64(index + 1), Text: strings.TrimSpace(proposed.Text),
		}
		contract.Criteria = append(contract.Criteria, criterion)
		contract.SuccessCriteria = append(contract.SuccessCriteria, criterion.Text)
		contract.EvidenceExpectations = append(contract.EvidenceExpectations, domain.ContractEvidenceExpectation{
			CriterionID: criterion.ID, Descriptions: append([]string(nil), proposed.EvidenceExpected...),
		})
	}
	return contract
}

func (service *Service) completeAnalysis(ctx context.Context, analyzing ports.IntakeSnapshot, expectedRevision int64, result ports.IntakeAnalysisResult, clarificationAnswer string) (ports.IntakeSnapshot, error) {
	now := service.clock()
	if result.Clarification != nil {
		if result.Proposal != nil {
			_, _ = service.store.FailIntakeAnalysis(ctx, analyzing.Session.ID, expectedRevision, "INTAKE_ANALYSIS_INVALID", now)
			return ports.IntakeSnapshot{}, apierr.Invalid("INTAKE_ANALYSIS_INVALID", "Analyzer returned both a question and a Contract proposal", nil)
		}
		if !analyzing.Session.CanAskMaterialClarification() {
			_, _ = service.store.FailIntakeAnalysis(ctx, analyzing.Session.ID, expectedRevision, "INTAKE_MATERIAL_QUESTION_LIMIT", now)
			return ports.IntakeSnapshot{}, apierr.Invalid("INTAKE_MATERIAL_QUESTION_LIMIT", "Outcome intake permits at most one material clarification", nil)
		}
		clarification := *result.Clarification
		clarification.ID = domain.ClarificationRequestID("clarification-" + uuid.NewString())
		clarification.IntakeID = analyzing.Session.ID
		clarification.CreatedAt = now
		if err := clarification.Validate(); err != nil {
			_, _ = service.store.FailIntakeAnalysis(ctx, analyzing.Session.ID, expectedRevision, "INTAKE_ANALYSIS_INVALID", now)
			return ports.IntakeSnapshot{}, apierr.Invalid("INTAKE_ANALYSIS_INVALID", err.Error(), nil)
		}
		view, err := service.store.CompleteIntakeWithClarification(ctx, analyzing.Session.ID, expectedRevision, clarification, now)
		if err != nil {
			return ports.IntakeSnapshot{}, mapStoreError(err)
		}
		return view, nil
	}
	if result.Proposal == nil {
		_, _ = service.store.FailIntakeAnalysis(ctx, analyzing.Session.ID, expectedRevision, "INTAKE_ANALYSIS_INVALID", now)
		return ports.IntakeSnapshot{}, apierr.Invalid("INTAKE_ANALYSIS_INVALID", "Analyzer returned neither a question nor a Contract proposal", nil)
	}
	proposal := *result.Proposal
	proposal.ID = domain.ProposalRevisionID("proposal-" + uuid.NewString())
	proposal.IntakeID = analyzing.Session.ID
	proposal.Revision = expectedRevision + 1
	proposal.CreatedAt = now
	if clarificationAnswer != "" {
		proposal.ClarificationNotes = append(proposal.ClarificationNotes, clarificationAnswer)
	}
	for index := range proposal.Criteria {
		if strings.TrimSpace(string(proposal.Criteria[index].ID)) == "" {
			proposal.Criteria[index].ID = domain.ProposedCriterionID("proposal-criterion-" + uuid.NewString())
		}
	}
	if err := proposal.Validate(); err != nil {
		_, _ = service.store.FailIntakeAnalysis(ctx, analyzing.Session.ID, expectedRevision, "INTAKE_ANALYSIS_INVALID", now)
		return ports.IntakeSnapshot{}, apierr.Invalid("INTAKE_ANALYSIS_INVALID", err.Error(), nil)
	}
	ready, err := service.store.CompleteIntakeWithProposal(ctx, analyzing.Session.ID, expectedRevision, proposal, now)
	if err != nil {
		return ports.IntakeSnapshot{}, mapStoreError(err)
	}
	return ready, nil
}

func intakeRevisionConflict(id domain.IntakeSessionID, expected, current int64) error {
	return apierr.Conflict("INTAKE_REVISION_CONFLICT", fmt.Sprintf("Intake moved to proposal revision %d; reload and retry", current), map[string]any{
		"intakeId": string(id), "expectedRevision": expected, "currentRevision": current,
	})
}

func mapStoreError(err error) error {
	var revisionConflict *ports.IntakeRevisionConflictError
	if errors.As(err, &revisionConflict) {
		return intakeRevisionConflict(revisionConflict.IntakeID, revisionConflict.ExpectedRevision, revisionConflict.CurrentRevision)
	}
	var idempotencyConflict *ports.IntakeIdempotencyConflictError
	if errors.As(err, &idempotencyConflict) {
		return apierr.Conflict("INTAKE_IDEMPOTENCY_CONFLICT", "That idempotency key belongs to different intake input", map[string]any{"requestKey": idempotencyConflict.Key})
	}
	return err
}

func intakeFingerprint(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = fmt.Fprintf(hash, "%d:%s;", len(part), part)
	}
	return fmt.Sprintf("v1:%x", hash.Sum(nil))
}

func conversationFingerprint(refs []domain.IntakeConversationRef) string {
	parts := make([]string, 0, len(refs)*3)
	for _, ref := range refs {
		parts = append(parts, ref.EpisodeID, ref.TurnID, fmt.Sprintf("%d", ref.Position))
	}
	return strings.Join(parts, "\x00")
}

// Package intake owns the one shared Home/Work adaptive understanding flow.
package intake

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
	// Offline runs the deterministic floor instead of the configured analyzer.
	// It is how a person stops waiting for an agent and takes the proposal
	// that is always available, and it never asks an agent anything.
	Offline bool
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
	// offline is the deterministic floor, always present and never the
	// configured analyzer. Intake is the entry point to the whole product, so
	// unlike decomposition it must not fail closed when no agent can be asked:
	// there is always a proposal available, and the owner can always choose it
	// over waiting for one.
	offline ports.IntakeAnalyzer
	// reaper ends a proposing session once its ask is closed. Optional: a nil
	// reaper simply leaves sessions running, which is what the daemon did
	// before this existed.
	reaper ports.AnalystSessionReaper
	clock  func() time.Time
}

// WithAnalystSessionReaper wires session cleanup for answered asks.
func (service *Service) WithAnalystSessionReaper(reaper ports.AnalystSessionReaper) *Service {
	service.reaper = reaper
	return service
}

// reap ends the session that answered (or failed to answer) one ask.
//
// Called only AFTER the ask is durably closed, so a kill that fails costs a
// stray process rather than a record that disagrees with reality.
func (service *Service) reap(ctx context.Context, sessionID string) {
	if service.reaper == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	_ = service.reaper.Kill(ctx, sessionID)
}

// New constructs the shared adaptive intake service.
func New(store ports.IntakeStore, analyzer ports.IntakeAnalyzer, clock func() time.Time) *Service {
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, analyzer: analyzer, offline: NewRuleBasedAnalyzer(), clock: clock}
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
	deferral := service.newDeferral(id, input.ExpectedProposalRevision)
	ticket, err := service.chooseAnalyzer(input.Offline).Analyze(ctx, ports.IntakeAnalysisInput{
		Session: analyzing.Session, ConversationRefs: analyzing.ConversationRefs,
		PreviousProposal: analyzing.Proposal, Clarification: analyzing.Clarification,
		Defer: deferral.open,
	})
	if err != nil {
		// An analyzer that opened an ask and then failed has left one nothing
		// will ever answer; closing it here is what keeps a retry possible.
		deferral.abandon(ctx, err.Error())
		_, _ = service.store.FailIntakeAnalysis(ctx, id, input.ExpectedProposalRevision, "INTAKE_ANALYSIS_FAILED", service.clock())
		return ports.IntakeSnapshot{}, err
	}
	return service.settleTicket(ctx, analyzing, input.ExpectedProposalRevision, ticket, deferral, "")
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
	deferral := service.newDeferral(id, input.ExpectedProposalRevision)
	ticket, err := service.analyzer.Analyze(ctx, ports.IntakeAnalysisInput{
		Session: analyzing.Session, ConversationRefs: analyzing.ConversationRefs,
		PreviousProposal: analyzing.Proposal, Clarification: analyzing.Clarification,
		ClarificationText: input.Answer,
		Defer:             deferral.open,
	})
	if err != nil {
		deferral.abandon(ctx, err.Error())
		_, _ = service.store.FailIntakeAnalysis(ctx, id, input.ExpectedProposalRevision, "INTAKE_ANALYSIS_FAILED", service.clock())
		return ports.IntakeSnapshot{}, err
	}
	return service.settleTicket(ctx, analyzing, input.ExpectedProposalRevision, ticket, deferral, input.Answer)
}

// chooseAnalyzer returns the floor when the owner asked for it, or when no
// analyzer is configured at all. The floor is never absent, which is what lets
// intake refuse to fail closed.
func (service *Service) chooseAnalyzer(offline bool) ports.IntakeAnalyzer {
	if offline || service.analyzer == nil {
		return service.offline
	}
	return service.analyzer
}

// deferral mints the durable request an analyzer answers on, at most once.
//
// It exists so the analyzer can decide to defer without the service having to
// guess in advance: a request row is written only when an agent is actually
// about to be asked, so an offline analysis leaves no trace of an ask that
// never happened.
type deferral struct {
	service          *Service
	intakeID         domain.IntakeSessionID
	expectedRevision int64
	request          *domain.IntakeAnalysisRequest
}

func (service *Service) newDeferral(id domain.IntakeSessionID, expectedRevision int64) *deferral {
	return &deferral{service: service, intakeID: id, expectedRevision: expectedRevision}
}

// abandon closes an ask whose analyzer never got as far as answering.
//
// Without this a failed spawn leaves the request open for its full TTL, and
// the one-open-ask-at-a-time guard then refuses every retry for fifteen
// minutes — so the owner is locked out of asking again by a failure that
// happened instantly. Decomposition closes its request on spawn failure for
// the same reason.
func (d *deferral) abandon(ctx context.Context, reason string) {
	if d.request == nil {
		return
	}
	_ = d.service.store.AnswerIntakeAnalysisRequest(ctx, ports.IntakeAnalysisRequestAnswer{
		RequestID: d.request.ID, Status: domain.IntakeAnalysisRejected,
		RefusalReason: reason, At: d.service.clock(),
	})
}

func (d *deferral) open(ctx context.Context) (ports.IntakeCallback, error) {
	if d.request != nil {
		return ports.IntakeCallback{}, fmt.Errorf("this analysis already opened a callback")
	}
	now := d.service.clock()

	// One open ask at a time. Two agents answering the same intake would race
	// to produce competing proposals, and the second would be refused as a
	// closed request anyway — better to say so before spawning it.
	if existing, found, err := d.service.store.LatestIntakeAnalysisRequest(ctx, d.intakeID); err != nil {
		return ports.IntakeCallback{}, err
	} else if found && existing.Status.Open() && !existing.Expired(now) {
		return ports.IntakeCallback{}, apierr.Conflict("INTAKE_ANALYSIS_REQUEST_OPEN",
			"An agent is already proposing a Contract for this intake",
			map[string]any{"requestId": string(existing.ID), "expiresAt": existing.ExpiresAt})
	}

	token, err := newCallbackToken()
	if err != nil {
		return ports.IntakeCallback{}, apierr.Internal("INTAKE_CALLBACK_TOKEN_FAILED", "Could not mint a callback token")
	}
	request := domain.IntakeAnalysisRequest{
		ID:                       domain.IntakeAnalysisRequestID("ireq-" + uuid.NewString()),
		IntakeID:                 d.intakeID,
		ExpectedProposalRevision: d.expectedRevision,
		Status:                   domain.IntakeAnalysisRequested,
		CallbackTokenDigest:      domain.HashCallbackToken(token),
		ExpiresAt:                now.Add(domain.DefaultIntakeAnalysisRequestTTL),
		CreatedAt:                now,
	}
	// Persist BEFORE returning. An agent spawned with a token for a request
	// that was never recorded would have nowhere to answer, and its work would
	// be silently lost.
	if err := d.service.store.CreateIntakeAnalysisRequest(ctx, request); err != nil {
		return ports.IntakeCallback{}, err
	}
	d.request = &request
	return ports.IntakeCallback{RequestID: request.ID, Token: token, ExpiresAt: request.ExpiresAt}, nil
}

// settleTicket turns an analyzer's ticket into durable state.
//
// An inline answer completes the analysis exactly as it always did. A deferred
// one leaves the intake in `analyzing` with an open request beside it: that
// request is the durable fact that an agent is working, and the startup sweep
// consults it rather than reaping the intake (see RecoverInterruptedIntakeAnalyses).
func (service *Service) settleTicket(
	ctx context.Context,
	analyzing ports.IntakeSnapshot,
	expectedRevision int64,
	ticket ports.IntakeAnalysisTicket,
	d *deferral,
	clarificationAnswer string,
) (ports.IntakeSnapshot, error) {
	id := analyzing.Session.ID
	if ticket.Inline != nil {
		// An analyzer that opened a callback and then answered inline would
		// leave an open request nothing will ever answer. Close it rather than
		// letting it sit until expiry.
		if d.request != nil {
			_ = service.store.AnswerIntakeAnalysisRequest(ctx, ports.IntakeAnalysisRequestAnswer{
				RequestID: d.request.ID, Status: domain.IntakeAnalysisRequestCancelled,
				RefusalReason: "The analyzer answered inline instead", At: service.clock(),
			})
		}
		return service.completeAnalysis(ctx, analyzing, expectedRevision, *ticket.Inline, clarificationAnswer)
	}
	if d.request == nil {
		// Nothing was started and nothing was produced. Failing loudly beats
		// parking the intake in a state no answer will ever leave.
		_, _ = service.store.FailIntakeAnalysis(ctx, id, expectedRevision, "INTAKE_ANALYSIS_NOT_STARTED", service.clock())
		return ports.IntakeSnapshot{}, apierr.Internal("INTAKE_ANALYSIS_NOT_STARTED",
			"The analyzer neither answered nor started anything")
	}
	// Bind the answering session so a restart can tell what was working on
	// this, and so the waiting state can name the agent instead of showing an
	// anonymous spinner. A failed bind is not worth failing the ask over — the
	// agent is already running and can still answer — so it is not returned as
	// an error, but it does cost recovery its handle on that session.
	if ticket.SessionID != "" {
		_ = service.store.BindIntakeAnalysisRequestSession(ctx, d.request.ID, ticket.SessionID, ticket.Harness)
	}
	snapshot, found, err := service.store.GetIntake(ctx, id)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	if !found {
		return ports.IntakeSnapshot{}, apierr.NotFound("INTAKE_NOT_FOUND", "That intake does not exist")
	}
	return snapshot, nil
}

// SubmitAgentProposal is the callback an agent-authored proposal arrives on.
//
// Routing is checked BEFORE the proposal is parsed: an answer to a closed,
// expired, or wrongly-addressed request is a routing problem, not a validation
// one. Past that, the draft goes through exactly the same gates a
// hand-authored proposal does — there is no trusted-proposer path.
func (service *Service) SubmitAgentProposal(
	ctx context.Context,
	requestID domain.IntakeAnalysisRequestID,
	token string,
	result ports.IntakeAnalysisResult,
	raw string,
) (ports.IntakeSnapshot, error) {
	request, found, err := service.store.GetIntakeAnalysisRequest(ctx, requestID)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	if !found {
		return ports.IntakeSnapshot{}, apierr.NotFound("INTAKE_ANALYSIS_REQUEST_NOT_FOUND", "That intake analysis request does not exist")
	}
	now := service.clock()
	if err := request.AdmitProposalAnswer(token, now); err != nil {
		// Deliberately the same shape for every routing refusal: a caller
		// probing tokens learns only that its answer was not admitted.
		return ports.IntakeSnapshot{}, apierr.Conflict("INTAKE_ANALYSIS_REQUEST_NOT_ADMITTED", err.Error(),
			map[string]any{"requestId": string(requestID)})
	}

	analyzing, found, err := service.store.GetIntake(ctx, request.IntakeID)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	if !found {
		return ports.IntakeSnapshot{}, apierr.NotFound("INTAKE_NOT_FOUND", "That intake no longer exists")
	}
	// The request froze a proposal revision. An answer that arrives after the
	// owner already moved on is refused rather than rebound to whatever is
	// current now.
	if analyzing.Session.CurrentProposalRevision != request.ExpectedProposalRevision {
		return service.rejectAgentProposal(ctx, request, raw,
			"The intake moved on while the agent was working; this answers a superseded revision")
	}
	if analyzing.Session.Status != domain.IntakeStatusAnalyzing {
		return service.rejectAgentProposal(ctx, request, raw,
			"This intake is no longer waiting for an analysis")
	}

	completed, err := service.completeAnalysis(ctx, analyzing, request.ExpectedProposalRevision, result, "")
	if err != nil {
		// The daemon's own words are kept with the draft so the owner can see
		// exactly what was wrong with what the agent proposed.
		return service.rejectAgentProposal(ctx, request, raw, apierrMessage(err))
	}
	if err := service.store.AnswerIntakeAnalysisRequest(ctx, ports.IntakeAnalysisRequestAnswer{
		RequestID: request.ID, Status: domain.IntakeAnalysisFulfilled, RawProposal: raw, At: now,
	}); err != nil && !errors.Is(err, ports.ErrIntakeAnalysisRequestClosed) {
		return ports.IntakeSnapshot{}, err
	}
	service.reap(ctx, request.SessionID)
	return completed, nil
}

// rejectAgentProposal records a refusal WITH the draft and returns the intake
// to a retryable failure, so the owner can see what the agent proposed, why it
// was refused, and still reach a proposal through the offline floor.
func (service *Service) rejectAgentProposal(ctx context.Context, request domain.IntakeAnalysisRequest, raw, reason string) (ports.IntakeSnapshot, error) {
	now := service.clock()
	if err := service.store.AnswerIntakeAnalysisRequest(ctx, ports.IntakeAnalysisRequestAnswer{
		RequestID: request.ID, Status: domain.IntakeAnalysisRejected,
		RawProposal: raw, RefusalReason: reason, At: now,
	}); err != nil && !errors.Is(err, ports.ErrIntakeAnalysisRequestClosed) {
		return ports.IntakeSnapshot{}, err
	}
	_, _ = service.store.FailIntakeAnalysis(ctx, request.IntakeID, request.ExpectedProposalRevision, "INTAKE_ANALYSIS_REFUSED", now)
	service.reap(ctx, request.SessionID)
	return ports.IntakeSnapshot{}, apierr.Invalid("INTAKE_ANALYSIS_REFUSED", reason,
		map[string]any{"requestId": string(request.ID)})
}

// LatestAnalysisRequest reads an intake's newest ask, so a waiting state can
// name who is working and a refused draft stays inspectable.
func (service *Service) LatestAnalysisRequest(ctx context.Context, id domain.IntakeSessionID) (domain.IntakeAnalysisRequest, bool, error) {
	return service.store.LatestIntakeAnalysisRequest(ctx, id)
}

// CancelAnalysisRequest is the owner's way out of waiting. It closes the ask
// and returns the intake to a retryable failure, from which Analyze with
// Offline set reaches the deterministic proposal immediately.
func (service *Service) CancelAnalysisRequest(ctx context.Context, id domain.IntakeSessionID) (ports.IntakeSnapshot, error) {
	request, found, err := service.store.LatestIntakeAnalysisRequest(ctx, id)
	if err != nil {
		return ports.IntakeSnapshot{}, err
	}
	if !found || !request.Status.Open() {
		return ports.IntakeSnapshot{}, apierr.Conflict("INTAKE_ANALYSIS_REQUEST_NOT_OPEN", "No agent is working on this intake", nil)
	}
	now := service.clock()
	if err := service.store.AnswerIntakeAnalysisRequest(ctx, ports.IntakeAnalysisRequestAnswer{
		RequestID: request.ID, Status: domain.IntakeAnalysisRequestCancelled,
		RefusalReason: "The owner stopped waiting for an agent", At: now,
	}); err != nil && !errors.Is(err, ports.ErrIntakeAnalysisRequestClosed) {
		return ports.IntakeSnapshot{}, err
	}
	service.reap(ctx, request.SessionID)
	return service.store.FailIntakeAnalysis(ctx, id, request.ExpectedProposalRevision, "INTAKE_ANALYSIS_CANCELLED", now)
}

// ExpireStaleAnalysisRequests closes asks whose deadline passed, and returns
// each abandoned intake to a retryable failure.
//
// It runs at startup and on a timer, because expiry is a durable deadline
// rather than an in-memory timer: a request that timed out while the daemon
// was down still has to reach a verdict, and its intake must not be left
// waiting for an agent that stopped existing.
func (service *Service) ExpireStaleAnalysisRequests(ctx context.Context) (int, error) {
	open, err := service.store.ListOpenIntakeAnalysisRequests(ctx)
	if err != nil {
		return 0, err
	}
	now := service.clock()
	closed := 0
	for _, request := range open {
		if !request.Expired(now) {
			continue
		}
		if err := service.store.AnswerIntakeAnalysisRequest(ctx, ports.IntakeAnalysisRequestAnswer{
			RequestID: request.ID, Status: domain.IntakeAnalysisExpired,
			RefusalReason: "No proposal arrived before the request expired", At: now,
		}); err != nil && !errors.Is(err, ports.ErrIntakeAnalysisRequestClosed) {
			return closed, err
		}
		_, _ = service.store.FailIntakeAnalysis(ctx, request.IntakeID, request.ExpectedProposalRevision, "INTAKE_ANALYSIS_EXPIRED", now)
		service.reap(ctx, request.SessionID)
		closed++
	}
	return closed, nil
}

// newCallbackToken mints the scoping token handed to a spawned session.
//
// It is NOT authentication: the loopback listener is unauthenticated by
// deliberate decision, so any local process could already reach the callback.
// It scopes an answer to one request, single-use and expiring, which is what
// stops a confused or retrying agent answering for the wrong intake.
func newCallbackToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// apierrMessage extracts the daemon's own refusal text.
func apierrMessage(err error) string {
	var apiErr *apierr.Error
	if errors.As(err, &apiErr) {
		return apiErr.Message
	}
	return err.Error()
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

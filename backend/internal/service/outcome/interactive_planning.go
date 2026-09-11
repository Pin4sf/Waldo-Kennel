package outcome

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	intelligencesvc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/intelligence"
)

// StartPlanningInput selects one exact planner and approved context source.
type StartPlanningInput struct {
	ExpectedContractRevision int64
	CandidateID              string
	ContextMode              domain.PlanningContextMode
	RequestKey               string
}

// PlanningMessageInput appends one idempotent owner message.
type PlanningMessageInput struct {
	ExpectedSessionRevision int64
	Text                    string
	RequestKey              string
}

// PlanningFinalizeInput requests a Plan proposal from the selected planner.
type PlanningFinalizeInput struct {
	ExpectedSessionRevision int64
	RequestKey              string
}

// PlanningView is the daemon-owned projection rendered by Mission Control.
type PlanningView struct {
	Outcome      domain.Outcome
	Session      domain.PlanningSession
	Turns        []domain.PlanningTurn
	ProposedPlan *domain.PlanRevision
}

// PlanningCandidates lists exact bindings available for the confirmed Contract.
func (s *Service) PlanningCandidates(ctx context.Context, outcomeID domain.OutcomeID, expectedContractRevision int64) ([]ports.PlanningCandidate, error) {
	outcomeRecord, revision, err := s.planningLineage(ctx, outcomeID, expectedContractRevision)
	if err != nil {
		return nil, err
	}
	_ = outcomeRecord
	if s.planningDialogue == nil {
		return nil, apierr.Internal("PLANNING_UNWIRED", "Interactive planning is unavailable in this environment")
	}
	_ = revision
	candidates, err := s.planningDialogue.PlanningCandidates(ctx)
	if err != nil {
		return nil, intelligencesvc.APIError(err)
	}
	return candidates, nil
}

// StartPlanning opens one Contract-bound conversation with frozen context.
func (s *Service) StartPlanning(ctx context.Context, outcomeID domain.OutcomeID, in StartPlanningInput) (PlanningView, error) {
	if s.planningSessions == nil || s.planningDialogue == nil || s.intelligenceRuns == nil || s.routing == nil {
		return PlanningView{}, apierr.Internal("PLANNING_UNWIRED", "Interactive planning is unavailable in this environment")
	}
	if strings.TrimSpace(in.RequestKey) == "" {
		return PlanningView{}, apierr.Invalid("PLANNING_REQUEST_KEY_REQUIRED", "A request key is required", nil)
	}
	contextMode := in.ContextMode
	if contextMode == "" {
		contextMode = domain.PlanningContextRepositoryRead
	}
	if !contextMode.Valid() {
		return PlanningView{}, apierr.Invalid("PLANNING_CONTEXT_INVALID", "Choose repository context or an approved supplied-document packet", nil)
	}
	requestKey := strings.TrimSpace(in.RequestKey)
	fingerprint := planningStartFingerprint(outcomeID, in.ExpectedContractRevision, strings.TrimSpace(in.CandidateID), contextMode)
	if existing, found, err := s.planningSessions.GetPlanningSessionByRequestKey(ctx, requestKey); err != nil {
		return PlanningView{}, err
	} else if found {
		if existing.RequestFingerprint != fingerprint {
			return PlanningView{}, planningAPIError(&ports.PlanningRequestConflictError{RequestKey: requestKey})
		}
		outcomeRecord, found, err := s.store.GetOutcome(ctx, existing.OutcomeID)
		if err != nil {
			return PlanningView{}, err
		}
		if !found {
			return PlanningView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
		}
		existing, err = s.reconcilePlanningContract(ctx, outcomeRecord, existing)
		if err != nil {
			return PlanningView{}, err
		}
		return s.planningView(ctx, outcomeRecord, existing)
	}
	outcomeRecord, revision, err := s.planningLineage(ctx, outcomeID, in.ExpectedContractRevision)
	if err != nil {
		return PlanningView{}, err
	}
	if contextMode == domain.PlanningContextRepositoryRead && !planningRepositoryReadAllowed(revision) {
		return PlanningView{}, apierr.New(apierr.KindConflict, "PLANNING_REPOSITORY_READ_REQUIRED",
			"Confirm repository-reading authority before starting planning", nil)
	}
	if current, found, currentErr := s.planningSessions.GetCurrentPlanningSession(ctx, outcomeID); currentErr != nil {
		return PlanningView{}, currentErr
	} else if found && current.Status == domain.PlanningSessionActive {
		if current.ContractRevisionNumber != revision.Number {
			if _, closeErr := s.planningSessions.ClosePlanningSession(ctx, current.ID, current.Revision, domain.PlanningSessionSuperseded); closeErr != nil {
				return PlanningView{}, planningAPIError(closeErr)
			}
		} else if current.RequestKey != strings.TrimSpace(in.RequestKey) {
			return PlanningView{}, apierr.New(apierr.KindConflict, "PLANNING_ALREADY_ACTIVE", "Continue or cancel the current planning conversation first", map[string]any{"planningSessionId": current.ID.String()})
		}
	}
	candidates, err := s.planningDialogue.PlanningCandidates(ctx)
	if err != nil {
		return PlanningView{}, intelligencesvc.APIError(err)
	}
	var selected *ports.PlanningCandidate
	for index := range candidates {
		if candidates[index].ID == strings.TrimSpace(in.CandidateID) {
			selected = &candidates[index]
			break
		}
	}
	if selected == nil {
		return PlanningView{}, apierr.Invalid("PLANNING_CANDIDATE_UNKNOWN", "Choose an available planning agent", nil)
	}
	if !selected.Ready {
		return PlanningView{}, apierr.Unavailable(selected.UnavailableCode, selected.UnavailableDetail, map[string]any{"candidateId": selected.ID})
	}
	projectID, project, err := s.projectForOutcome(ctx, outcomeID)
	if err != nil {
		return PlanningView{}, err
	}
	var snapshot ports.RepositoryContextSnapshot
	if contextMode == domain.PlanningContextSuppliedPacket {
		snapshot.ProjectID = projectID
		if err := s.groundInSelectedDocuments(ctx, outcomeID, &snapshot); err != nil {
			return PlanningView{}, err
		}
		if snapshot.Root != "supplied-documents" {
			return PlanningView{}, apierr.New(apierr.KindConflict, "PLANNING_DOCUMENTS_NOT_APPROVED", "Approve the selected documents before planning from them", nil)
		}
		// The document adapter replaces repository facts, so bind this session
		// to the approved packet rather than retaining the old repository digest.
		snapshot.Digest = ""
		digestInput, digestErr := json.Marshal(snapshot)
		if digestErr != nil {
			return PlanningView{}, fmt.Errorf("encode supplied planning context: %w", digestErr)
		}
		snapshot.Digest = domain.DigestSHA256(digestInput)
	} else {
		var briefSource interface {
			GetCurrentProjectBriefRevision(context.Context, domain.ProjectID) (domain.ProjectBriefRevision, bool, error)
		}
		if candidate, ok := s.store.(interface {
			GetCurrentProjectBriefRevision(context.Context, domain.ProjectID) (domain.ProjectBriefRevision, bool, error)
		}); ok {
			briefSource = candidate
		}
		snapshot, err = intelligencesvc.BuildRepositoryContext(ctx, project, briefSource)
		if err != nil {
			return PlanningView{}, err
		}
		if snapshot.UnavailableReason != "" {
			return PlanningView{}, apierr.Unavailable("PLANNING_REPOSITORY_UNAVAILABLE", snapshot.UnavailableReason, nil)
		}
	}
	contextJSON, err := json.Marshal(snapshot)
	if err != nil {
		return PlanningView{}, fmt.Errorf("encode planning context: %w", err)
	}
	grantJSON, _ := json.Marshal(struct {
		RepositoryPacketRead bool `json:"repositoryPacketRead"`
		SuppliedPacketRead   bool `json:"suppliedPacketRead"`
		Commands             bool `json:"commands"`
		Writes               bool `json:"writes"`
		ExternalEffects      bool `json:"externalEffects"`
	}{RepositoryPacketRead: contextMode == domain.PlanningContextRepositoryRead, SuppliedPacketRead: contextMode == domain.PlanningContextSuppliedPacket})
	now := s.clock().UTC()
	session := domain.PlanningSession{
		ID: domain.PlanningSessionID("planning-" + uuid.NewString()), OutcomeID: outcomeID, ProjectID: projectID,
		ContractRevisionID: revision.ID, ContractRevisionNumber: revision.Number, Revision: 1,
		Status: domain.PlanningSessionActive, WaitingOn: domain.PlanningWaitingOwner, Binding: selected.Binding,
		ContextMode: contextMode, PlanningGrantDigest: domain.DigestSHA256(grantJSON), ContextDigest: snapshot.Digest,
		ContextSnapshotJSON: contextJSON, RequestKey: requestKey, RequestFingerprint: fingerprint,
		CreatedAt: now, UpdatedAt: now,
	}
	saved, _, err := s.planningSessions.CreatePlanningSession(ctx, session)
	if err != nil {
		return PlanningView{}, planningAPIError(err)
	}
	return s.planningView(ctx, outcomeRecord, saved)
}

// GetPlanning returns one planning conversation and its optional proposed Plan.
func (s *Service) GetPlanning(ctx context.Context, outcomeID domain.OutcomeID, sessionID domain.PlanningSessionID) (PlanningView, error) {
	if s.planningSessions == nil {
		return PlanningView{}, apierr.Internal("PLANNING_UNWIRED", "Interactive planning is unavailable in this environment")
	}
	outcomeRecord, found, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return PlanningView{}, err
	}
	if !found {
		return PlanningView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	session, found, err := s.planningSessions.GetPlanningSession(ctx, outcomeID, sessionID)
	if err != nil {
		return PlanningView{}, err
	}
	if !found {
		return PlanningView{}, apierr.NotFound("PLANNING_SESSION_NOT_FOUND", "That planning conversation does not exist")
	}
	session, err = s.reconcilePlanningContract(ctx, outcomeRecord, session)
	if err != nil {
		return PlanningView{}, err
	}
	return s.planningView(ctx, outcomeRecord, session)
}

// GetCurrentPlanning returns the newest planning conversation for an Outcome.
func (s *Service) GetCurrentPlanning(ctx context.Context, outcomeID domain.OutcomeID) (PlanningView, error) {
	if s.planningSessions == nil {
		return PlanningView{}, apierr.Internal("PLANNING_UNWIRED", "Interactive planning is unavailable in this environment")
	}
	outcomeRecord, found, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return PlanningView{}, err
	}
	if !found {
		return PlanningView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	session, found, err := s.planningSessions.GetCurrentPlanningSession(ctx, outcomeID)
	if err != nil {
		return PlanningView{}, err
	}
	if !found {
		return PlanningView{}, apierr.NotFound("PLANNING_SESSION_NOT_FOUND", "This Outcome has no planning conversation yet")
	}
	session, err = s.reconcilePlanningContract(ctx, outcomeRecord, session)
	if err != nil {
		return PlanningView{}, err
	}
	return s.planningView(ctx, outcomeRecord, session)
}

// ContinuePlanning exchanges one owner message for one structured planner turn.
func (s *Service) ContinuePlanning(ctx context.Context, outcomeID domain.OutcomeID, sessionID domain.PlanningSessionID, in PlanningMessageInput) (PlanningView, error) {
	if strings.TrimSpace(in.Text) == "" {
		return PlanningView{}, apierr.Invalid("PLANNING_MESSAGE_REQUIRED", "Write a short message for the planning agent", nil)
	}
	return s.runPlanningTurn(ctx, outcomeID, sessionID, in.ExpectedSessionRevision, strings.TrimSpace(in.Text), in.RequestKey, false)
}

// FinalizePlanning requests a proposal without bypassing validation or approval.
func (s *Service) FinalizePlanning(ctx context.Context, outcomeID domain.OutcomeID, sessionID domain.PlanningSessionID, in PlanningFinalizeInput) (PlanningView, error) {
	return s.runPlanningTurn(ctx, outcomeID, sessionID, in.ExpectedSessionRevision, "Propose the plan now.", in.RequestKey, true)
}

// CancelPlanning closes an active conversation without creating execution state.
func (s *Service) CancelPlanning(ctx context.Context, outcomeID domain.OutcomeID, sessionID domain.PlanningSessionID, expectedRevision int64) (PlanningView, error) {
	view, err := s.GetPlanning(ctx, outcomeID, sessionID)
	if err != nil {
		return PlanningView{}, err
	}
	closed, err := s.planningSessions.ClosePlanningSession(ctx, sessionID, expectedRevision, domain.PlanningSessionCancelled)
	if err != nil {
		return PlanningView{}, planningAPIError(err)
	}
	return s.planningView(ctx, view.Outcome, closed)
}

// RecoverInterruptedPlanning returns crash-interrupted direct-API waits to the
// owner. It never replays a possibly billed provider request automatically.
func (s *Service) RecoverInterruptedPlanning(ctx context.Context) (int64, error) {
	if s.planningSessions == nil {
		return 0, apierr.Internal("PLANNING_UNWIRED", "Interactive planning is unavailable in this environment")
	}
	recovered, err := s.planningSessions.RecoverInterruptedPlanningSessions(ctx, s.clock())
	if err != nil {
		return 0, apierr.Internal("PLANNING_RECOVERY_FAILED", "Interrupted planning conversations could not be recovered")
	}
	return recovered, nil
}

func (s *Service) runPlanningTurn(ctx context.Context, outcomeID domain.OutcomeID, sessionID domain.PlanningSessionID, expectedRevision int64, text, requestKey string, finalize bool) (PlanningView, error) {
	if s.planningSessions == nil || s.planningDialogue == nil || s.intelligenceRuns == nil || s.routing == nil {
		return PlanningView{}, apierr.Internal("PLANNING_UNWIRED", "Interactive planning is unavailable in this environment")
	}
	if strings.TrimSpace(requestKey) == "" {
		return PlanningView{}, apierr.Invalid("PLANNING_REQUEST_KEY_REQUIRED", "A request key is required", nil)
	}
	view, err := s.GetPlanning(ctx, outcomeID, sessionID)
	if err != nil {
		return PlanningView{}, err
	}
	if view.Outcome.CurrentRevisionNumber != view.Session.ContractRevisionNumber || view.Session.Status == domain.PlanningSessionSuperseded {
		return PlanningView{}, apierr.New(apierr.KindConflict, "PLANNING_CONTRACT_STALE", "The Contract changed. Start a new planning conversation.", map[string]any{"currentRevision": view.Outcome.CurrentRevisionNumber})
	}
	payload, _ := json.Marshal(struct {
		SessionID string `json:"sessionId"`
		Revision  int64  `json:"revision"`
		Text      string `json:"text"`
		Finalize  bool   `json:"finalize"`
	}{sessionID.String(), expectedRevision, text, finalize})
	now := s.clock().UTC()
	ownerTurn := domain.PlanningTurn{
		ID: domain.PlanningTurnID("planning-turn-" + uuid.NewString()), PlanningSessionID: sessionID,
		Role: domain.PlanningTurnOwner, Kind: domain.PlanningTurnMessage, Text: text,
		RequestKey: strings.TrimSpace(requestKey), RequestFingerprint: domain.DigestSHA256(payload), CreatedAt: now,
	}
	if finalize {
		ownerTurn.Kind = domain.PlanningTurnFinalizeRequest
	}
	session, storedOwner, replay, err := s.planningSessions.AppendPlanningOwnerTurn(ctx, sessionID, expectedRevision, ownerTurn)
	if err != nil {
		if view.Session.Status != domain.PlanningSessionActive {
			return PlanningView{}, apierr.New(apierr.KindConflict, "PLANNING_SESSION_CLOSED", "This planning conversation is closed", nil)
		}
		return PlanningView{}, planningAPIError(err)
	}
	if replay {
		return s.resumePlanningReply(ctx, view.Outcome, session, storedOwner)
	}
	var snapshot ports.RepositoryContextSnapshot
	if err := json.Unmarshal(session.ContextSnapshotJSON, &snapshot); err != nil {
		return PlanningView{}, apierr.Internal("PLANNING_CONTEXT_CORRUPT", "The frozen planning context could not be read")
	}
	revision, err := s.currentRevision(ctx, view.Outcome)
	if err != nil {
		return PlanningView{}, err
	}
	aliases, err := criterionAliases(revision)
	if err != nil {
		return PlanningView{}, err
	}
	turns, err := s.planningSessions.ListPlanningTurns(ctx, sessionID)
	if err != nil {
		return PlanningView{}, err
	}
	request := ports.PlanningDiscussionRequest{Binding: session.Binding, Outcome: view.Outcome, Contract: revision, CriterionAliases: aliases, RepositoryContext: snapshot, Turns: turns, Finalize: finalize}
	encodedRequest, _ := json.Marshal(request)
	run := domain.IntelligenceRun{
		ID: domain.IntelligenceRunID("intel-" + uuid.NewString()), Kind: domain.IntelligenceRunPlanDraft,
		ProjectID: session.ProjectID, OutcomeID: outcomeID, ContractRevisionID: revision.ID, SourceRevision: revision.Number,
		RequestedProvider: session.Binding.Provider, RequestedModel: session.Binding.Model, InputDigest: domain.DigestSHA256(encodedRequest),
		Status: domain.IntelligenceRunRequested, CreatedAt: now,
	}
	if err := s.intelligenceRuns.CreateIntelligenceRun(ctx, run); err != nil {
		return PlanningView{}, err
	}
	if err := s.intelligenceRuns.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunRunning, "", "", "", nil); err != nil {
		return PlanningView{}, err
	}
	started := time.Now()
	response, err := s.planningDialogue.DiscussPlan(ctx, request)
	if err != nil {
		completed := s.clock().UTC()
		duration := time.Since(started).Milliseconds()
		cleanup, cancel := intelligencesvc.TerminalizationContext(ctx)
		defer cancel()
		_ = s.intelligenceRuns.RecordIntelligenceRunMetrics(cleanup, run.ID, nil, nil, &duration)
		code, detail := intelligencesvc.TerminalReason(err)
		_ = s.intelligenceRuns.UpdateIntelligenceRunStatus(cleanup, run.ID, domain.IntelligenceRunFailed, "", code, detail, &completed)
		_, _ = s.planningSessions.SetPlanningSessionFailure(cleanup, sessionID, session.Revision, code, detail)
		return PlanningView{}, intelligencesvc.APIError(err)
	}
	provenanceFailureCode, provenanceFailureDetail := "", ""
	if response.Provenance.EffectiveProvider.IsZero() || response.Provenance.EffectiveProvider != session.Binding.Provider {
		provenanceFailureCode = "PLANNING_PROVIDER_MISMATCH"
		provenanceFailureDetail = "The planning response did not come from the selected provider"
	} else if session.Binding.ModelSelection == domain.PlanningModelExplicit && strings.TrimSpace(response.Provenance.EffectiveModel) != session.Binding.Model {
		provenanceFailureCode = "PLANNING_MODEL_MISMATCH"
		provenanceFailureDetail = "The planning response did not use the selected explicit model"
	}
	if provenanceFailureCode != "" {
		completed := s.clock().UTC()
		_ = s.intelligenceRuns.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFailed, "", provenanceFailureCode, provenanceFailureDetail, &completed)
		_, _ = s.planningSessions.SetPlanningSessionFailure(ctx, sessionID, session.Revision, provenanceFailureCode, provenanceFailureDetail)
		return PlanningView{}, apierr.New(apierr.KindConflict, provenanceFailureCode, "The planning provider or model changed. Start a new planning conversation.", nil)
	}
	encodedResult, err := json.Marshal(response.Result)
	if err != nil {
		return PlanningView{}, fmt.Errorf("encode planning reply: %w", err)
	}
	duration := time.Since(started).Milliseconds()
	if err := s.intelligenceRuns.RecordIntelligenceRunMetrics(ctx, run.ID, response.Provenance.InputTokens, response.Provenance.OutputTokens, &duration); err != nil {
		return PlanningView{}, err
	}
	if err := s.intelligenceRuns.RecordIntelligenceRunEffectiveProvenance(ctx, run.ID, response.Provenance.EffectiveProvider, response.Provenance.EffectiveModel, response.Provenance.NativeSessionRef); err != nil {
		return PlanningView{}, err
	}
	completed := s.clock().UTC()
	if err := s.intelligenceRuns.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFulfilled, domain.DigestSHA256(encodedResult), "", "", &completed); err != nil {
		return PlanningView{}, err
	}
	plannerKind, err := planningTurnKind(response.Result)
	if err != nil {
		return PlanningView{}, err
	}
	plannerTurn := domain.PlanningTurn{
		ID: domain.PlanningTurnID("planning-turn-" + uuid.NewString()), PlanningSessionID: sessionID,
		ReplyToTurnID: storedOwner.ID, Role: domain.PlanningTurnPlanner, Kind: plannerKind,
		Text: strings.TrimSpace(response.Result.Message), StructuredPayload: encodedResult, IntelligenceRunID: run.ID, CreatedAt: completed,
	}
	session, err = s.planningSessions.AppendPlanningProviderTurn(ctx, sessionID, session.Revision, plannerTurn,
		response.Provenance.EffectiveProvider, response.Provenance.EffectiveModel, response.Provenance.NativeSessionRef)
	if err != nil {
		return PlanningView{}, planningAPIError(err)
	}
	currentOutcome, found, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return PlanningView{}, err
	}
	if !found {
		return PlanningView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	session, err = s.reconcilePlanningContract(ctx, currentOutcome, session)
	if err != nil {
		return PlanningView{}, err
	}
	if session.Status == domain.PlanningSessionSuperseded {
		return PlanningView{}, apierr.New(apierr.KindConflict, "PLANNING_CONTRACT_STALE", "The Contract changed. Start a new planning conversation.", map[string]any{"currentRevision": currentOutcome.CurrentRevisionNumber})
	}
	if response.Result.Kind == ports.PlanningResultPlanProposal {
		return s.finishPlanningProposal(ctx, currentOutcome, revision, session, run.ID, response.Result)
	}
	return s.planningView(ctx, currentOutcome, session)
}

func (s *Service) resumePlanningReply(ctx context.Context, outcomeRecord domain.Outcome, session domain.PlanningSession, owner domain.PlanningTurn) (PlanningView, error) {
	turns, err := s.planningSessions.ListPlanningTurns(ctx, session.ID)
	if err != nil {
		return PlanningView{}, err
	}
	for _, turn := range turns {
		if turn.ReplyToTurnID != owner.ID {
			continue
		}
		if turn.Kind != domain.PlanningTurnPlanProposal || session.Status != domain.PlanningSessionActive {
			return s.planningView(ctx, outcomeRecord, session)
		}
		var result ports.PlanningResult
		if err := json.Unmarshal(turn.StructuredPayload, &result); err != nil {
			return PlanningView{}, apierr.Internal("PLANNING_REPLY_CORRUPT", "The saved planning reply could not be read")
		}
		revision, err := s.currentRevision(ctx, outcomeRecord)
		if err != nil {
			return PlanningView{}, err
		}
		return s.finishPlanningProposal(ctx, outcomeRecord, revision, session, turn.IntelligenceRunID, result)
	}
	return s.planningView(ctx, outcomeRecord, session)
}

func (s *Service) finishPlanningProposal(ctx context.Context, outcomeRecord domain.Outcome, revision domain.ContractRevision, session domain.PlanningSession, runID domain.IntelligenceRunID, result ports.PlanningResult) (PlanningView, error) {
	if result.PlanProposal == nil {
		return PlanningView{}, apierr.Internal("PLANNING_PROPOSAL_MISSING", "The planning agent returned no Plan proposal")
	}
	if existing, found, err := s.planningSessions.GetPlanRevisionByPlanningSession(ctx, outcomeRecord.ID, session.ID); err != nil {
		return PlanningView{}, err
	} else if found {
		if session.Status == domain.PlanningSessionActive {
			session, err = s.planningSessions.LinkPlanningSessionPlan(ctx, session.ID, session.Revision, existing.ID, existing.SourceIntelligenceRunID)
			if err != nil {
				return PlanningView{}, planningAPIError(err)
			}
		}
		return s.planningView(ctx, outcomeRecord, session)
	}
	projectID, project, err := s.projectForOutcome(ctx, outcomeRecord.ID)
	if err != nil {
		return PlanningView{}, err
	}
	preference, hasPreference, err := domain.ResolveEffectiveExecutionPreference(revision.ExecutionPreference, project.Config)
	if err != nil {
		return PlanningView{}, apierr.Invalid("PLAN_PREFERENCE_INVALID", err.Error(), nil)
	}
	aliases, err := criterionAliases(revision)
	if err != nil {
		return PlanningView{}, err
	}
	units, decisions, err := s.compileAndRoutePlan(ctx, projectID, revision, *result.PlanProposal, aliases, routingPreferenceFromExecution(preference, hasPreference))
	if err != nil {
		return PlanningView{}, err
	}
	grants := grantsForUnits(units)
	if err := s.authorizeCapabilities(revision, grants, units); err != nil {
		return PlanningView{}, err
	}
	digest, err := domain.ComputePlanRunBriefCoreDigest(revision, units, grants)
	if err != nil {
		return PlanningView{}, err
	}
	plan := domain.PlanRevision{
		ID: domain.PlanRevisionID("plan-" + uuid.NewString()), OutcomeID: outcomeRecord.ID, ContractRevisionNumber: revision.Number,
		Status: domain.PlanStatusProposed, Summary: result.PlanProposal.Summary, Assumptions: append([]string(nil), result.PlanProposal.Assumptions...),
		Blockers: append([]string(nil), result.PlanProposal.Blockers...), WorkUnits: units, Grants: grants, RoutingDecisions: decisions,
		RunBriefCoreDigest: digest, PlanningSessionID: session.ID, SourceIntelligenceRunID: runID,
	}
	validation := plan
	validation.Number = 1
	if err := validation.ValidateAgainstContract(revision); err != nil {
		return PlanningView{}, apierr.Invalid("PLAN_DRAFT_CRITERIA_INVALID", err.Error(), nil)
	}
	saved, err := s.store.AppendPlanRevision(ctx, outcomeRecord.ID, plan)
	if err != nil {
		if existing, found, findErr := s.planningSessions.GetPlanRevisionByPlanningSession(ctx, outcomeRecord.ID, session.ID); findErr == nil && found {
			saved = existing
		} else {
			currentOutcome, outcomeFound, outcomeErr := s.store.GetOutcome(ctx, outcomeRecord.ID)
			if outcomeErr != nil {
				return PlanningView{}, outcomeErr
			}
			if outcomeFound {
				session, outcomeErr = s.reconcilePlanningContract(ctx, currentOutcome, session)
				if outcomeErr != nil {
					return PlanningView{}, outcomeErr
				}
				if session.Status == domain.PlanningSessionSuperseded {
					return PlanningView{}, apierr.New(apierr.KindConflict, "PLANNING_CONTRACT_STALE", "The Contract changed. Start a new planning conversation.", map[string]any{"currentRevision": currentOutcome.CurrentRevisionNumber})
				}
			}
			return PlanningView{}, err
		}
	}
	session, found, err := s.planningSessions.GetPlanningSession(ctx, outcomeRecord.ID, session.ID)
	if err != nil {
		return PlanningView{}, err
	}
	if !found {
		return PlanningView{}, apierr.NotFound("PLANNING_SESSION_NOT_FOUND", "That planning conversation does not exist")
	}
	// Compatibility for a Plan written by an older feature build that crashed
	// before the session link. New canonical SQLite writes link atomically.
	if session.Status == domain.PlanningSessionActive {
		session, err = s.planningSessions.LinkPlanningSessionPlan(ctx, session.ID, session.Revision, saved.ID, saved.SourceIntelligenceRunID)
		if err != nil {
			currentOutcome, outcomeFound, outcomeErr := s.store.GetOutcome(ctx, outcomeRecord.ID)
			if outcomeErr != nil {
				return PlanningView{}, outcomeErr
			}
			if outcomeFound {
				session, outcomeErr = s.reconcilePlanningContract(ctx, currentOutcome, session)
				if outcomeErr != nil {
					return PlanningView{}, outcomeErr
				}
				if session.Status == domain.PlanningSessionSuperseded {
					return PlanningView{}, apierr.New(apierr.KindConflict, "PLANNING_CONTRACT_STALE", "The Contract changed. Start a new planning conversation.", map[string]any{"currentRevision": currentOutcome.CurrentRevisionNumber})
				}
			}
			return PlanningView{}, planningAPIError(err)
		}
	}
	return s.planningView(ctx, outcomeRecord, session)
}

func (s *Service) reconcilePlanningContract(ctx context.Context, outcomeRecord domain.Outcome, session domain.PlanningSession) (domain.PlanningSession, error) {
	for attempt := 0; attempt < 2; attempt++ {
		if session.Status != domain.PlanningSessionActive || session.ContractRevisionNumber == outcomeRecord.CurrentRevisionNumber {
			return session, nil
		}
		closed, err := s.planningSessions.ClosePlanningSession(ctx, session.ID, session.Revision, domain.PlanningSessionSuperseded)
		if err == nil {
			return closed, nil
		}
		var conflict *ports.PlanningSessionRevisionConflictError
		if !errors.As(err, &conflict) {
			return domain.PlanningSession{}, planningAPIError(err)
		}
		var found bool
		session, found, err = s.planningSessions.GetPlanningSession(ctx, outcomeRecord.ID, session.ID)
		if err != nil {
			return domain.PlanningSession{}, err
		}
		if !found {
			return domain.PlanningSession{}, apierr.NotFound("PLANNING_SESSION_NOT_FOUND", "That planning conversation does not exist")
		}
	}
	return domain.PlanningSession{}, apierr.New(apierr.KindConflict, "PLANNING_REVISION_CONFLICT", "The planning conversation changed. Reload and try again.", nil)
}

func (s *Service) planningView(ctx context.Context, outcomeRecord domain.Outcome, session domain.PlanningSession) (PlanningView, error) {
	turns, err := s.planningSessions.ListPlanningTurns(ctx, session.ID)
	if err != nil {
		return PlanningView{}, err
	}
	view := PlanningView{Outcome: outcomeRecord, Session: session, Turns: turns}
	if plan, found, err := s.planningSessions.GetPlanRevisionByPlanningSession(ctx, outcomeRecord.ID, session.ID); err != nil {
		return PlanningView{}, err
	} else if found {
		view.ProposedPlan = &plan
	}
	return view, nil
}

func (s *Service) planningLineage(ctx context.Context, outcomeID domain.OutcomeID, expected int64) (domain.Outcome, domain.ContractRevision, error) {
	outcomeRecord, found, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return domain.Outcome{}, domain.ContractRevision{}, err
	}
	if !found {
		return domain.Outcome{}, domain.ContractRevision{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	if expected < 1 {
		return domain.Outcome{}, domain.ContractRevision{}, apierr.Invalid("EXPECTED_REVISION_REQUIRED", "State which Contract revision planning uses", nil)
	}
	if outcomeRecord.CurrentRevisionNumber != expected {
		return domain.Outcome{}, domain.ContractRevision{}, apierr.New(apierr.KindConflict, "PLANNING_CONTRACT_STALE", "Reload the Outcome before planning", map[string]any{"currentRevision": outcomeRecord.CurrentRevisionNumber})
	}
	revision, err := s.currentRevision(ctx, outcomeRecord)
	return outcomeRecord, revision, err
}

func planningRepositoryReadAllowed(revision domain.ContractRevision) bool {
	return revision.AuthorityCeiling.ReadWorkspace || revision.AuthorityCeiling.WriteWorkspace || revision.AuthorityCeiling.ExecuteLocal
}

func planningStartFingerprint(outcomeID domain.OutcomeID, revision int64, candidateID string, contextMode domain.PlanningContextMode) domain.SHA256Digest {
	payload, _ := json.Marshal(struct {
		OutcomeID string                     `json:"outcomeId"`
		Revision  int64                      `json:"contractRevision"`
		Candidate string                     `json:"candidateId"`
		Context   domain.PlanningContextMode `json:"contextMode"`
	}{string(outcomeID), revision, candidateID, contextMode})
	return domain.DigestSHA256(payload)
}

func planningTurnKind(result ports.PlanningResult) (domain.PlanningTurnKind, error) {
	switch result.Kind {
	case ports.PlanningResultClarification:
		return domain.PlanningTurnClarification, nil
	case ports.PlanningResultContractChange:
		return domain.PlanningTurnContractChangeProposal, nil
	case ports.PlanningResultPlanProposal:
		return domain.PlanningTurnPlanProposal, nil
	default:
		return "", apierr.Internal("PLANNING_REPLY_INVALID", "The planning agent returned an unsupported reply")
	}
}

func planningAPIError(err error) error {
	var revision *ports.PlanningSessionRevisionConflictError
	if errors.As(err, &revision) {
		return apierr.New(apierr.KindConflict, "PLANNING_REVISION_CONFLICT", "The planning conversation changed. Reload and try again.", map[string]any{"currentRevision": revision.Current})
	}
	var request *ports.PlanningRequestConflictError
	if errors.As(err, &request) {
		return apierr.New(apierr.KindConflict, "PLANNING_REQUEST_CONFLICT", "That request key was already used for a different planning action", nil)
	}
	var finalize *ports.PlanningFinalizeConflictError
	if errors.As(err, &finalize) {
		return apierr.New(apierr.KindConflict, "PLANNING_FINALIZE_CONFLICT", "The Contract or planning conversation changed before the Plan could be saved", nil)
	}
	return err
}

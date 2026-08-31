package intake

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

type memoryIntakeStore struct {
	requests []domain.IntakeAnalysisRequest
	ports.IntakeStore
	snapshot        ports.IntakeSnapshot
	proposalHistory []domain.OutcomeContractProposal
	confirmations   map[string]ports.IntakeSnapshot
	confirmWrites   int
	recoverWrites   int64
}

func (store *memoryIntakeStore) RecoverInterruptedIntakeAnalyses(_ context.Context, at time.Time) (int64, error) {
	if store.snapshot.Session.Status != domain.IntakeStatusAnalyzing {
		return 0, nil
	}
	store.snapshot.Session.Status = domain.IntakeStatusAnalysisFailed
	store.snapshot.Session.FailureCode = "INTAKE_ANALYSIS_INTERRUPTED"
	store.snapshot.Session.UpdatedAt = at
	store.recoverWrites++
	return 1, nil
}

func (store *memoryIntakeStore) CancelIntake(_ context.Context, _ domain.IntakeSessionID, expected int64, reason string, at time.Time) (ports.IntakeSnapshot, error) {
	if store.snapshot.Session.CurrentProposalRevision != expected {
		return ports.IntakeSnapshot{}, &ports.IntakeRevisionConflictError{ExpectedRevision: expected, CurrentRevision: store.snapshot.Session.CurrentProposalRevision}
	}
	store.snapshot.Session.Status = domain.IntakeStatusCancelled
	store.snapshot.Session.CancellationReason = reason
	store.snapshot.Session.UpdatedAt = at
	return store.snapshot, nil
}

func (store *memoryIntakeStore) GetProject(_ context.Context, id string) (domain.ProjectRecord, bool, error) {
	if id == "project-1" {
		return domain.ProjectRecord{ID: id}, true, nil
	}
	return domain.ProjectRecord{}, false, nil
}

func (store *memoryIntakeStore) CreateIntake(_ context.Context, session domain.IntakeSession, refs []domain.IntakeConversationRef, _ ports.IntakeIdempotency) (ports.IntakeSnapshot, error) {
	store.snapshot = ports.IntakeSnapshot{Session: session, ConversationRefs: append([]domain.IntakeConversationRef(nil), refs...)}
	return store.snapshot, nil
}

func (store *memoryIntakeStore) GetIntake(_ context.Context, _ domain.IntakeSessionID) (ports.IntakeSnapshot, bool, error) {
	return store.snapshot, !store.snapshot.Session.ID.IsZero(), nil
}

func (store *memoryIntakeStore) BeginIntakeAnalysis(_ context.Context, _ domain.IntakeSessionID, expectedRevision int64, at time.Time) (ports.IntakeSnapshot, error) {
	if store.snapshot.Session.CurrentProposalRevision != expectedRevision {
		return ports.IntakeSnapshot{}, &ports.IntakeRevisionConflictError{ExpectedRevision: expectedRevision, CurrentRevision: store.snapshot.Session.CurrentProposalRevision}
	}
	store.snapshot.Session.Status = domain.IntakeStatusAnalyzing
	store.snapshot.Session.UpdatedAt = at
	return store.snapshot, nil
}

func (store *memoryIntakeStore) CompleteIntakeWithProposal(_ context.Context, _ domain.IntakeSessionID, expectedRevision int64, proposal domain.OutcomeContractProposal, at time.Time) (ports.IntakeSnapshot, error) {
	if store.snapshot.Session.CurrentProposalRevision != expectedRevision {
		return ports.IntakeSnapshot{}, &ports.IntakeRevisionConflictError{ExpectedRevision: expectedRevision, CurrentRevision: store.snapshot.Session.CurrentProposalRevision}
	}
	store.snapshot.Session.Status = domain.IntakeStatusReady
	store.snapshot.Session.CurrentProposalRevision = proposal.Revision
	store.snapshot.Session.UpdatedAt = at
	store.snapshot.Proposal = &proposal
	store.proposalHistory = append(store.proposalHistory, proposal)
	return store.snapshot, nil
}

func (store *memoryIntakeStore) AppendIntakeProposalRevision(_ context.Context, _ domain.IntakeSessionID, expectedRevision int64, proposal domain.OutcomeContractProposal, at time.Time) (ports.IntakeSnapshot, error) {
	if store.snapshot.Session.CurrentProposalRevision != expectedRevision {
		return ports.IntakeSnapshot{}, &ports.IntakeRevisionConflictError{ExpectedRevision: expectedRevision, CurrentRevision: store.snapshot.Session.CurrentProposalRevision}
	}
	store.snapshot.Session.Status = domain.IntakeStatusReady
	store.snapshot.Session.CurrentProposalRevision = proposal.Revision
	store.snapshot.Session.UpdatedAt = at
	store.snapshot.Proposal = &proposal
	store.proposalHistory = append(store.proposalHistory, proposal)
	return store.snapshot, nil
}

func (store *memoryIntakeStore) ConfirmIntakeWithOutcome(_ context.Context, _ domain.IntakeSessionID, expectedRevision int64, outcome domain.Outcome, contract domain.ContractRevision, request ports.IntakeIdempotency, at time.Time) (ports.IntakeSnapshot, error) {
	if store.confirmations == nil {
		store.confirmations = map[string]ports.IntakeSnapshot{}
	}
	if replay, ok := store.confirmations[request.Key]; ok {
		return replay, nil
	}
	if store.snapshot.Session.Status == domain.IntakeStatusConfirmed {
		return store.snapshot, nil
	}
	if store.snapshot.Session.CurrentProposalRevision != expectedRevision {
		return ports.IntakeSnapshot{}, &ports.IntakeRevisionConflictError{ExpectedRevision: expectedRevision, CurrentRevision: store.snapshot.Session.CurrentProposalRevision}
	}
	store.confirmWrites++
	store.snapshot.Session.Status = domain.IntakeStatusConfirmed
	store.snapshot.Session.ConfirmedOutcomeID = outcome.ID
	store.snapshot.Session.UpdatedAt = at
	store.snapshot.ConfirmedOutcome = &outcome
	store.snapshot.ConfirmedContract = &contract
	store.confirmations[request.Key] = store.snapshot
	return store.snapshot, nil
}

func (store *memoryIntakeStore) EnsureWorkResponsibilitySpace(_ context.Context, projectID domain.ProjectID) (domain.ResponsibilitySpace, error) {
	return domain.ResponsibilitySpace{ID: domain.ResponsibilitySpaceID("space-" + projectID), ProjectID: projectID, Kind: domain.ResponsibilitySpaceWorkProject}, nil
}

func (store *memoryIntakeStore) CompleteIntakeWithClarification(_ context.Context, _ domain.IntakeSessionID, expectedRevision int64, clarification domain.ClarificationRequest, at time.Time) (ports.IntakeSnapshot, error) {
	if store.snapshot.Session.CurrentProposalRevision != expectedRevision {
		return ports.IntakeSnapshot{}, &ports.IntakeRevisionConflictError{ExpectedRevision: expectedRevision, CurrentRevision: store.snapshot.Session.CurrentProposalRevision}
	}
	store.snapshot.Session.Status = domain.IntakeStatusNeedsUser
	store.snapshot.Session.ClarificationCount++
	store.snapshot.Session.UpdatedAt = at
	store.snapshot.Clarification = &clarification
	return store.snapshot, nil
}

func (store *memoryIntakeStore) AnswerIntakeClarification(_ context.Context, _ domain.IntakeSessionID, expectedRevision int64, answer string, at time.Time) (ports.IntakeSnapshot, error) {
	if store.snapshot.Session.CurrentProposalRevision != expectedRevision {
		return ports.IntakeSnapshot{}, &ports.IntakeRevisionConflictError{ExpectedRevision: expectedRevision, CurrentRevision: store.snapshot.Session.CurrentProposalRevision}
	}
	store.snapshot.Session.Status = domain.IntakeStatusAnalyzing
	store.snapshot.Session.UpdatedAt = at
	store.snapshot.Clarification.Answer = answer
	store.snapshot.Clarification.AnsweredAt = &at
	return store.snapshot, nil
}

func (store *memoryIntakeStore) FailIntakeAnalysis(_ context.Context, _ domain.IntakeSessionID, expectedRevision int64, code string, at time.Time) (ports.IntakeSnapshot, error) {
	if store.snapshot.Session.CurrentProposalRevision != expectedRevision {
		return ports.IntakeSnapshot{}, &ports.IntakeRevisionConflictError{ExpectedRevision: expectedRevision, CurrentRevision: store.snapshot.Session.CurrentProposalRevision}
	}
	store.snapshot.Session.Status = domain.IntakeStatusAnalysisFailed
	store.snapshot.Session.FailureCode = code
	store.snapshot.Session.UpdatedAt = at
	return store.snapshot, nil
}

type scriptedAnalyzer struct {
	result   ports.IntakeAnalysisResult
	results  []ports.IntakeAnalysisResult
	err      error
	seen     ports.IntakeAnalysisInput
	calls    int
	deferred bool
}

// The fake is scripted in IntakeAnalysisResult terms and wraps each one as an
// inline ticket, so these tests keep asserting the same behaviour they did
// before the port grew a deferred shape. deferred covers the other half.
func (analyzer *scriptedAnalyzer) Analyze(_ context.Context, input ports.IntakeAnalysisInput) (ports.IntakeAnalysisTicket, error) {
	analyzer.seen = input
	if analyzer.deferred {
		return ports.IntakeAnalysisTicket{SessionID: "sess-analyst", Detail: "an agent is proposing"}, analyzer.err
	}
	if analyzer.calls < len(analyzer.results) {
		result := analyzer.results[analyzer.calls]
		analyzer.calls++
		return ports.IntakeAnalysisTicket{Inline: &result}, analyzer.err
	}
	analyzer.calls++
	result := analyzer.result
	return ports.IntakeAnalysisTicket{Inline: &result}, analyzer.err
}

func TestCaptureAndAnalyzeSimpleOutcomeAdvancesDirectlyToReady(t *testing.T) {
	now := time.Date(2026, 8, 26, 2, 0, 0, 0, time.UTC)
	store := &memoryIntakeStore{}
	analyzer := &scriptedAnalyzer{result: ports.IntakeAnalysisResult{
		Proposal: &domain.OutcomeContractProposal{
			Title:        "Keyboard settings navigation",
			DesiredState: "Every settings control is reachable and operable by keyboard.",
			Criteria: []domain.ProposedCriterion{{
				ID:               "proposal-criterion-1",
				Text:             "Tab and Shift+Tab reach every interactive settings control in logical order.",
				EvidenceExpected: []string{"A deterministic keyboard-navigation component test passes."},
			}},
			ReviewMethod:     "Run deterministic checks and complete an owner walkthrough.",
			AuthorityCeiling: domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true, ExecuteLocal: true},
			StopConditions:   []string{"Stop before any remote effect."},
			Facets: []domain.ContractFacet{{
				Kind: domain.ContractFacetSoftware, Summary: "Desktop renderer accessibility",
			}},
		},
	}}
	service := New(store, analyzer, func() time.Time { return now })

	captured, err := service.Capture(context.Background(), CaptureInput{
		SourceSurface: domain.IntakeSourceWork,
		Purpose:       domain.IntakePurposeOutcome,
		ProjectID:     "project-1",
		Statement:     "Add keyboard navigation to the settings screen",
		ConversationRefs: []domain.IntakeConversationRef{{
			EpisodeID: "episode-1", TurnID: "turn-7", Position: 1,
		}},
		RequestKey: "capture-1",
	})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if captured.Session.Status != domain.IntakeStatusCaptured {
		t.Fatalf("Capture() status = %q, want captured", captured.Session.Status)
	}

	ready, err := service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if ready.Session.Status != domain.IntakeStatusReady || ready.Proposal == nil || ready.Proposal.Revision != 1 {
		t.Fatalf("Analyze() view = %+v, want ready proposal revision 1", ready)
	}
	if analyzer.seen.Session.Statement != captured.Session.Statement {
		t.Fatalf("analyzer statement = %q, want exact captured statement %q", analyzer.seen.Session.Statement, captured.Session.Statement)
	}
	if len(analyzer.seen.ConversationRefs) != 1 || analyzer.seen.ConversationRefs[0].TurnID != "turn-7" {
		t.Fatalf("analyzer provenance = %+v, want referenced turn id", analyzer.seen.ConversationRefs)
	}
}

func TestMaterialClarificationIsAskedOnceThenAnswerProducesProposal(t *testing.T) {
	now := time.Date(2026, 8, 26, 2, 30, 0, 0, time.UTC)
	store := &memoryIntakeStore{}
	analyzer := &scriptedAnalyzer{results: []ports.IntakeAnalysisResult{
		{Clarification: &domain.ClarificationRequest{
			Question:            "What does today mean for this Outcome?",
			Reason:              "The date boundary changes which records count toward success.",
			Recommendation:      "Use the Mac's local calendar day.",
			Alternatives:        []string{"Mac local calendar day", "Rolling 24 hours"},
			DeferralConsequence: "The proposal will state the local-calendar-day assumption.",
		}},
		{Proposal: &domain.OutcomeContractProposal{
			Title:        "Daily focus total",
			DesiredState: "The app shows focus time for the Mac's local calendar day.",
			Criteria: []domain.ProposedCriterion{{
				Text:             "Focus blocks starting in the Mac's local calendar day are summed.",
				EvidenceExpected: []string{"A deterministic local-midnight boundary test passes."},
			}},
			ReviewMethod:     "Run deterministic date-boundary checks and an owner walkthrough.",
			AuthorityCeiling: domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true, ExecuteLocal: true},
			StopConditions:   []string{"Stop before any remote effect."},
			Facets:           []domain.ContractFacet{{Kind: domain.ContractFacetSoftware, Summary: "Local date behavior"}},
		}},
	}}
	service := New(store, analyzer, func() time.Time { return now })
	captured, err := service.Capture(context.Background(), CaptureInput{
		SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome,
		ProjectID: "project-1", Statement: "Show my total focus time today", RequestKey: "capture-today",
	})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	needsUser, err := service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if needsUser.Session.Status != domain.IntakeStatusNeedsUser || needsUser.Clarification == nil || needsUser.Session.ClarificationCount != 1 {
		t.Fatalf("Analyze() view = %+v, want one material clarification", needsUser)
	}

	ready, err := service.AnswerClarification(context.Background(), captured.Session.ID, AnswerClarificationInput{
		ExpectedProposalRevision: 0,
		Answer:                   "Use the Mac's local calendar day.",
	})
	if err != nil {
		t.Fatalf("AnswerClarification() error = %v", err)
	}
	if ready.Session.Status != domain.IntakeStatusReady || ready.Proposal == nil || ready.Session.ClarificationCount != 1 {
		t.Fatalf("AnswerClarification() view = %+v, want ready proposal with one-question history", ready)
	}
	if analyzer.seen.ClarificationText != "Use the Mac's local calendar day." {
		t.Fatalf("analyzer clarification answer = %q", analyzer.seen.ClarificationText)
	}
}

func TestReviseProposalIsAppendOnlyAndRejectsStaleExpectedRevision(t *testing.T) {
	now := time.Date(2026, 8, 26, 3, 0, 0, 0, time.UTC)
	store := &memoryIntakeStore{}
	analyzer := &scriptedAnalyzer{result: ports.IntakeAnalysisResult{Proposal: validProposalDraft("Original desired state")}}
	service := New(store, analyzer, func() time.Time { return now })
	captured, err := service.Capture(context.Background(), CaptureInput{
		SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome,
		ProjectID: "project-1", Statement: "Improve keyboard navigation", RequestKey: "capture-revise",
	})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	ready, err := service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	edited := *validProposalDraft("User-corrected desired state")
	edited.Criteria[0].ID = ready.Proposal.Criteria[0].ID
	revised, err := service.ReviseProposal(context.Background(), captured.Session.ID, ReviseProposalInput{
		ExpectedProposalRevision: 1,
		Proposal:                 edited,
	})
	if err != nil {
		t.Fatalf("ReviseProposal() error = %v", err)
	}
	if revised.Proposal == nil || revised.Proposal.Revision != 2 || revised.Proposal.DesiredState != "User-corrected desired state" {
		t.Fatalf("ReviseProposal() = %+v, want revision 2 with user edit", revised)
	}
	if len(store.proposalHistory) != 2 || store.proposalHistory[0].DesiredState != "Original desired state" {
		t.Fatalf("proposal history was overwritten: %+v", store.proposalHistory)
	}

	_, err = service.ReviseProposal(context.Background(), captured.Session.ID, ReviseProposalInput{
		ExpectedProposalRevision: 1,
		Proposal:                 edited,
	})
	assertAPIErrorCode(t, err, "INTAKE_REVISION_CONFLICT")
}

func TestConfirmOutcomeIsIdempotentAndPreservesStableCriterionIdentity(t *testing.T) {
	now := time.Date(2026, 8, 26, 3, 30, 0, 0, time.UTC)
	store := &memoryIntakeStore{}
	analyzer := &scriptedAnalyzer{result: ports.IntakeAnalysisResult{Proposal: validProposalDraft("Keyboard navigation is complete")}}
	service := New(store, analyzer, func() time.Time { return now })
	captured, err := service.Capture(context.Background(), CaptureInput{
		SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome,
		ProjectID: "project-1", Statement: "Improve keyboard navigation", RequestKey: "capture-confirm",
	})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	ready, err := service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	input := ConfirmOutcomeInput{ExpectedProposalRevision: ready.Proposal.Revision, RequestKey: "confirm-1"}
	first, err := service.ConfirmOutcome(context.Background(), captured.Session.ID, input)
	if err != nil {
		t.Fatalf("ConfirmOutcome() error = %v", err)
	}
	second, err := service.ConfirmOutcome(context.Background(), captured.Session.ID, input)
	if err != nil {
		t.Fatalf("ConfirmOutcome() replay error = %v", err)
	}
	if store.confirmWrites != 1 || first.ConfirmedOutcome == nil || second.ConfirmedOutcome == nil || first.ConfirmedOutcome.ID != second.ConfirmedOutcome.ID {
		t.Fatalf("confirmation writes/outcomes = %d/%+v/%+v, want one stable Outcome", store.confirmWrites, first.ConfirmedOutcome, second.ConfirmedOutcome)
	}
	contract := first.ConfirmedContract
	if contract == nil || len(contract.Criteria) != 1 || contract.Criteria[0].ID.IsZero() || contract.Criteria[0].ContractRevisionID != contract.ID {
		t.Fatalf("confirmed Contract criterion identity is unstable: %+v", contract)
	}
	if len(contract.StopConditions) != 1 || len(contract.Facets) != 1 || len(contract.EvidenceExpectations) != 1 {
		t.Fatalf("confirmed Contract dropped typed stable core: %+v", contract)
	}
}

func TestAnalyzerFailureIsDurableAndRetryable(t *testing.T) {
	now := time.Date(2026, 8, 26, 3, 45, 0, 0, time.UTC)
	store := &memoryIntakeStore{}
	analyzer := &scriptedAnalyzer{err: context.DeadlineExceeded}
	service := New(store, analyzer, func() time.Time { return now })
	captured, err := service.Capture(context.Background(), CaptureInput{SourceSurface: domain.IntakeSourceHome, Purpose: domain.IntakePurposeOutcome, ProjectID: "project-1", Statement: "Make the flow durable", RequestKey: "capture-failure"})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	_, err = service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0})
	if err == nil {
		t.Fatal("Analyze() error = nil, want analyzer failure")
	}
	if store.snapshot.Session.Status != domain.IntakeStatusAnalysisFailed || store.snapshot.Session.FailureCode != "INTAKE_ANALYSIS_FAILED" {
		t.Fatalf("durable failure = %+v", store.snapshot.Session)
	}

	analyzer.err = nil
	analyzer.result = ports.IntakeAnalysisResult{Proposal: validProposalDraft("Retry produced a proposal")}
	retried, err := service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0})
	if err != nil || retried.Session.Status != domain.IntakeStatusReady {
		t.Fatalf("retry = %+v err=%v", retried, err)
	}
}

func TestInvalidAnalyzerOutputLeavesDurableRetryableFailure(t *testing.T) {
	tests := []struct {
		name   string
		result ports.IntakeAnalysisResult
	}{
		{name: "invalid clarification", result: ports.IntakeAnalysisResult{Clarification: &domain.ClarificationRequest{}}},
		{name: "invalid proposal", result: ports.IntakeAnalysisResult{Proposal: &domain.OutcomeContractProposal{}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &memoryIntakeStore{}
			service := New(store, &scriptedAnalyzer{result: test.result}, func() time.Time {
				return time.Date(2026, 8, 26, 3, 50, 0, 0, time.UTC)
			})
			captured, err := service.Capture(context.Background(), CaptureInput{
				SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome,
				ProjectID: "project-1", Statement: "Keep invalid analysis retryable", RequestKey: "capture-" + test.name,
			})
			if err != nil {
				t.Fatalf("Capture() error = %v", err)
			}
			_, err = service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0})
			assertAPIErrorCode(t, err, "INTAKE_ANALYSIS_INVALID")
			if store.snapshot.Session.Status != domain.IntakeStatusAnalysisFailed || store.snapshot.Session.FailureCode != "INTAKE_ANALYSIS_INVALID" {
				t.Fatalf("durable invalid-analysis failure = %+v", store.snapshot.Session)
			}
		})
	}
}

func TestConfirmedIntakeIsIdempotentAcrossRetryRequestKeys(t *testing.T) {
	now := time.Date(2026, 8, 26, 3, 55, 0, 0, time.UTC)
	store := &memoryIntakeStore{}
	service := New(store, &scriptedAnalyzer{result: ports.IntakeAnalysisResult{Proposal: validProposalDraft("Ready")}}, func() time.Time { return now })
	captured, err := service.Capture(context.Background(), CaptureInput{SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome, ProjectID: "project-1", Statement: "Confirm once", RequestKey: "capture-once"})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	ready, err := service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	first, err := service.ConfirmOutcome(context.Background(), captured.Session.ID, ConfirmOutcomeInput{ExpectedProposalRevision: ready.Proposal.Revision, RequestKey: "confirm-first"})
	if err != nil {
		t.Fatalf("ConfirmOutcome() error = %v", err)
	}
	second, err := service.ConfirmOutcome(context.Background(), captured.Session.ID, ConfirmOutcomeInput{ExpectedProposalRevision: ready.Proposal.Revision, RequestKey: "confirm-retry-new-key"})
	if err != nil {
		t.Fatalf("ConfirmOutcome() retry error = %v", err)
	}
	if store.confirmWrites != 1 || first.ConfirmedOutcome.ID != second.ConfirmedOutcome.ID {
		t.Fatalf("confirmation writes/outcomes = %d/%s/%s, want one stable Outcome", store.confirmWrites, first.ConfirmedOutcome.ID, second.ConfirmedOutcome.ID)
	}
}

func TestCancelRecordsConsciousReleaseAndRejectsStaleRevision(t *testing.T) {
	store := &memoryIntakeStore{}
	service := New(store, nil, func() time.Time { return time.Date(2026, 8, 26, 4, 0, 0, 0, time.UTC) })
	captured, err := service.Capture(context.Background(), CaptureInput{SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome, ProjectID: "project-1", Statement: "Maybe later", RequestKey: "capture-cancel"})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	_, err = service.Cancel(context.Background(), captured.Session.ID, CancelInput{ExpectedProposalRevision: 1, Reason: "No longer needed"})
	assertAPIErrorCode(t, err, "INTAKE_REVISION_CONFLICT")
	cancelled, err := service.Cancel(context.Background(), captured.Session.ID, CancelInput{ExpectedProposalRevision: 0, Reason: "  No longer needed  "})
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if cancelled.Session.Status != domain.IntakeStatusCancelled || cancelled.Session.CancellationReason != "No longer needed" || cancelled.ConfirmedOutcome != nil {
		t.Fatalf("Cancel() = %+v", cancelled)
	}
}

func TestRecoverInterruptedAnalysesMakesTransientStateRetryable(t *testing.T) {
	store := &memoryIntakeStore{snapshot: ports.IntakeSnapshot{Session: domain.IntakeSession{ID: "intake-interrupted", Status: domain.IntakeStatusAnalyzing}}}
	service := New(store, nil, func() time.Time { return time.Date(2026, 8, 26, 4, 5, 0, 0, time.UTC) })
	recovered, err := service.RecoverInterruptedAnalyses(context.Background())
	if err != nil || recovered != 1 || store.snapshot.Session.Status != domain.IntakeStatusAnalysisFailed || store.snapshot.Session.FailureCode != "INTAKE_ANALYSIS_INTERRUPTED" {
		t.Fatalf("RecoverInterruptedAnalyses() recovered=%d snapshot=%+v err=%v", recovered, store.snapshot, err)
	}
}

func validProposalDraft(desiredState string) *domain.OutcomeContractProposal {
	return &domain.OutcomeContractProposal{
		Title: "Keyboard settings navigation", DesiredState: desiredState,
		Criteria: []domain.ProposedCriterion{{
			Text: "Every settings control is reachable by keyboard", EvidenceExpected: []string{"A deterministic component test passes"},
		}},
		ReviewMethod: "Run deterministic checks and an owner walkthrough",
		AuthorityCeiling: domain.ProposedAuthority{
			ReadWorkspace: true, WriteWorkspace: true, ExecuteLocal: true,
		},
		StopConditions: []string{"Stop before any remote effect"},
		Facets:         []domain.ContractFacet{{Kind: domain.ContractFacetSoftware, Summary: "Desktop accessibility"}},
	}
}

func assertAPIErrorCode(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want code %s", want)
	}
	var apiError *apierr.Error
	if !errors.As(err, &apiError) || apiError.Code != want {
		t.Fatalf("error = %#v, want api error code %s", err, want)
	}
}

// --- Agent-authored analysis: the durable request and its callback ---

func (store *memoryIntakeStore) CreateIntakeAnalysisRequest(_ context.Context, request domain.IntakeAnalysisRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	store.requests = append(store.requests, request)
	return nil
}

func (store *memoryIntakeStore) GetIntakeAnalysisRequest(_ context.Context, id domain.IntakeAnalysisRequestID) (domain.IntakeAnalysisRequest, bool, error) {
	for _, request := range store.requests {
		if request.ID == id {
			return request, true, nil
		}
	}
	return domain.IntakeAnalysisRequest{}, false, nil
}

func (store *memoryIntakeStore) LatestIntakeAnalysisRequest(_ context.Context, _ domain.IntakeSessionID) (domain.IntakeAnalysisRequest, bool, error) {
	if len(store.requests) == 0 {
		return domain.IntakeAnalysisRequest{}, false, nil
	}
	return store.requests[len(store.requests)-1], true, nil
}

func (store *memoryIntakeStore) ListOpenIntakeAnalysisRequests(_ context.Context) ([]domain.IntakeAnalysisRequest, error) {
	open := make([]domain.IntakeAnalysisRequest, 0, len(store.requests))
	for _, request := range store.requests {
		if request.Status.Open() {
			open = append(open, request)
		}
	}
	return open, nil
}

func (store *memoryIntakeStore) BindIntakeAnalysisRequestSession(_ context.Context, id domain.IntakeAnalysisRequestID, sessionID string, harness domain.AgentHarness) error {
	for i := range store.requests {
		if store.requests[i].ID == id && store.requests[i].Status.Open() {
			store.requests[i].SessionID = sessionID
			store.requests[i].Harness = harness
			return nil
		}
	}
	return ports.ErrIntakeAnalysisRequestClosed
}

// The single-use guard: an answer only lands on a still-open ask.
func (store *memoryIntakeStore) AnswerIntakeAnalysisRequest(_ context.Context, answer ports.IntakeAnalysisRequestAnswer) error {
	for i := range store.requests {
		if store.requests[i].ID != answer.RequestID {
			continue
		}
		if !store.requests[i].Status.Open() {
			return ports.ErrIntakeAnalysisRequestClosed
		}
		store.requests[i].Status = answer.Status
		store.requests[i].RawProposal = answer.RawProposal
		store.requests[i].RefusalReason = answer.RefusalReason
		at := answer.At
		store.requests[i].AnsweredAt = &at
		return nil
	}
	return ports.ErrIntakeAnalysisRequestClosed
}

// deferringAnalyzer opens a callback and answers nothing, as an agent-backed
// analyzer does. It records the callback so a test can answer on it.
type deferringAnalyzer struct {
	callback      ports.IntakeCallback
	openErr       error
	failAfterOpen error
}

func (a *deferringAnalyzer) Analyze(ctx context.Context, input ports.IntakeAnalysisInput) (ports.IntakeAnalysisTicket, error) {
	callback, err := input.Defer(ctx)
	if err != nil {
		a.openErr = err
		return ports.IntakeAnalysisTicket{}, err
	}
	a.callback = callback
	if a.failAfterOpen != nil {
		return ports.IntakeAnalysisTicket{}, a.failAfterOpen
	}
	return ports.IntakeAnalysisTicket{SessionID: "sess-analyst", Harness: "codex", Detail: "codex is proposing a Contract"}, nil
}

func deferredService(t *testing.T, now time.Time) (*Service, *memoryIntakeStore, *deferringAnalyzer, domain.IntakeSessionID) {
	t.Helper()
	store := &memoryIntakeStore{}
	analyzer := &deferringAnalyzer{}
	service := New(store, analyzer, func() time.Time { return now })
	captured, err := service.Capture(context.Background(), CaptureInput{SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome, ProjectID: "project-1", Statement: "Let an agent draft this", RequestKey: "capture-deferred"})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if _, err := service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0}); err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	return service, store, analyzer, captured.Session.ID
}

func TestDeferredAnalysisOpensOneDurableRequestAndKeepsTheIntakeWaiting(t *testing.T) {
	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	_, store, analyzer, _ := deferredService(t, now)

	if len(store.requests) != 1 {
		t.Fatalf("want exactly one durable request, got %d", len(store.requests))
	}
	request := store.requests[0]
	if !request.Status.Open() || request.SessionID != "sess-analyst" || request.Harness != "codex" {
		t.Fatalf("request did not record the answering session: %+v", request)
	}
	// The token is never stored; only its digest is, and the digest must match
	// the token the analyzer was handed.
	if request.CallbackTokenDigest != domain.HashCallbackToken(analyzer.callback.Token) {
		t.Fatal("stored digest does not address the token handed to the analyzer")
	}
	if strings.Contains(request.CallbackTokenDigest, analyzer.callback.Token) {
		t.Fatal("the raw callback token was stored")
	}
	// The intake stays analyzing: the open request beside it is what says an
	// agent is working, and it must not be a failure.
	if store.snapshot.Session.Status != domain.IntakeStatusAnalyzing {
		t.Fatalf("deferred intake status = %q, want analyzing", store.snapshot.Session.Status)
	}
}

func TestAgentProposalOnTheCallbackBecomesTheEditableProposal(t *testing.T) {
	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	service, store, analyzer, _ := deferredService(t, now)

	ready, err := service.SubmitAgentProposal(context.Background(), analyzer.callback.RequestID, analyzer.callback.Token,
		ports.IntakeAnalysisResult{Proposal: validProposalDraft("The agent read the repo and proposed this")}, `{"raw":true}`)
	if err != nil {
		t.Fatalf("SubmitAgentProposal() error = %v", err)
	}
	if ready.Session.Status != domain.IntakeStatusReady || ready.Proposal == nil {
		t.Fatalf("fulfilled analysis = %+v", ready.Session)
	}
	if store.requests[0].Status != domain.IntakeAnalysisFulfilled || store.requests[0].RawProposal != `{"raw":true}` {
		t.Fatalf("request not closed as fulfilled with its draft: %+v", store.requests[0])
	}

	// Single-use: the same agent retrying must not append a second proposal.
	if _, err := service.SubmitAgentProposal(context.Background(), analyzer.callback.RequestID, analyzer.callback.Token,
		ports.IntakeAnalysisResult{Proposal: validProposalDraft("A second answer")}, "{}"); err == nil {
		t.Fatal("a second answer on the same request was admitted")
	}
}

func TestCallbackRefusesWrongTokenAndExpiredRequestBeforeParsing(t *testing.T) {
	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	service, store, analyzer, _ := deferredService(t, now)

	// A confused agent answering with the wrong token is a routing problem,
	// and is refused without the proposal being looked at.
	if _, err := service.SubmitAgentProposal(context.Background(), analyzer.callback.RequestID, "not-the-token",
		ports.IntakeAnalysisResult{Proposal: validProposalDraft("Wrongly addressed")}, "{}"); err == nil {
		t.Fatal("an answer carrying the wrong token was admitted")
	}
	if store.requests[0].Status != domain.IntakeAnalysisRequested {
		t.Fatalf("a misrouted answer closed the request: %+v", store.requests[0])
	}

	// Expiry is derived from the clock, so a daemon that was not running still
	// sees an ask that timed out.
	late := New(store, analyzer, func() time.Time { return now.Add(domain.DefaultIntakeAnalysisRequestTTL + time.Minute) })
	if _, err := late.SubmitAgentProposal(context.Background(), analyzer.callback.RequestID, analyzer.callback.Token,
		ports.IntakeAnalysisResult{Proposal: validProposalDraft("Too late")}, "{}"); err == nil {
		t.Fatal("an answer after expiry was admitted")
	}
}

func TestRefusedAgentProposalIsRetainedAndLeavesTheIntakeRetryable(t *testing.T) {
	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	service, store, analyzer, id := deferredService(t, now)

	// An invalid proposal passes exactly the gates a hand-authored one does.
	if _, err := service.SubmitAgentProposal(context.Background(), analyzer.callback.RequestID, analyzer.callback.Token,
		ports.IntakeAnalysisResult{Proposal: &domain.OutcomeContractProposal{}}, `{"bad":true}`); err == nil {
		t.Fatal("an invalid agent proposal was accepted")
	}
	request := store.requests[0]
	if request.Status != domain.IntakeAnalysisRejected || request.RawProposal != `{"bad":true}` || request.RefusalReason == "" {
		t.Fatalf("refused draft was not retained with the daemon's words: %+v", request)
	}
	if store.snapshot.Session.Status != domain.IntakeStatusAnalysisFailed {
		t.Fatalf("refusal left status %q, want a retryable failure", store.snapshot.Session.Status)
	}

	// The floor is always reachable: the owner still gets a proposal.
	ready, err := service.Analyze(context.Background(), id, AnalyzeInput{ExpectedProposalRevision: 0, Offline: true})
	if err != nil || ready.Session.Status != domain.IntakeStatusReady {
		t.Fatalf("offline fallback after refusal = %+v err=%v", ready.Session, err)
	}
	if len(store.requests) != 1 {
		t.Fatalf("the offline floor opened a request it never needed: %d", len(store.requests))
	}
}

func TestOwnerCanStopWaitingAndTakeTheOfflineProposal(t *testing.T) {
	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	service, store, _, id := deferredService(t, now)

	if _, err := service.CancelAnalysisRequest(context.Background(), id); err != nil {
		t.Fatalf("CancelAnalysisRequest() error = %v", err)
	}
	if store.requests[0].Status != domain.IntakeAnalysisRequestCancelled {
		t.Fatalf("cancel did not close the ask: %+v", store.requests[0])
	}
	ready, err := service.Analyze(context.Background(), id, AnalyzeInput{ExpectedProposalRevision: 0, Offline: true})
	if err != nil || ready.Session.Status != domain.IntakeStatusReady {
		t.Fatalf("offline proposal after cancel = %+v err=%v", ready.Session, err)
	}
}

func TestExpiryClosesAbandonedRequestsAndFreesTheIntake(t *testing.T) {
	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	_, store, _, _ := deferredService(t, now)

	later := New(store, &deferringAnalyzer{}, func() time.Time { return now.Add(domain.DefaultIntakeAnalysisRequestTTL + time.Minute) })
	closed, err := later.ExpireStaleAnalysisRequests(context.Background())
	if err != nil || closed != 1 {
		t.Fatalf("ExpireStaleAnalysisRequests() = %d err=%v, want 1", closed, err)
	}
	if store.requests[0].Status != domain.IntakeAnalysisExpired {
		t.Fatalf("expiry verdict = %+v", store.requests[0])
	}
	// An intake left waiting for an agent that never answered has to become a
	// retryable failure, not sit in analyzing forever.
	if store.snapshot.Session.Status != domain.IntakeStatusAnalysisFailed {
		t.Fatalf("expired intake status = %q", store.snapshot.Session.Status)
	}
}

func TestOnlyOneAgentMayWorkOnAnIntakeAtATime(t *testing.T) {
	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	service, store, _, id := deferredService(t, now)

	// The intake is analyzing with an open ask; a second ask must be refused
	// before another agent is spawned to race the first.
	store.snapshot.Session.Status = domain.IntakeStatusAnalysisFailed
	if _, err := service.Analyze(context.Background(), id, AnalyzeInput{ExpectedProposalRevision: 0}); err == nil {
		t.Fatal("a second concurrent analysis request was admitted")
	}
	if len(store.requests) != 1 {
		t.Fatalf("a competing request was written anyway: %d", len(store.requests))
	}
}

// A spawn that fails leaves an ask nothing will ever answer. Holding it open
// would make the one-open-ask-at-a-time guard refuse every retry for the full
// TTL, so an instant failure would lock the owner out for fifteen minutes.
func TestAFailedAnalyzerClosesTheAskItOpened(t *testing.T) {
	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	store := &memoryIntakeStore{}
	analyzer := &deferringAnalyzer{failAfterOpen: errors.New("duplicate session: mesa-5")}
	service := New(store, analyzer, func() time.Time { return now })
	captured, err := service.Capture(context.Background(), CaptureInput{SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome, ProjectID: "project-1", Statement: "Spawn will fail", RequestKey: "capture-spawnfail"})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	if _, err := service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0}); err == nil {
		t.Fatal("Analyze() error = nil, want the spawn failure")
	}
	if len(store.requests) != 1 {
		t.Fatalf("want the opened ask retained, got %d", len(store.requests))
	}
	if store.requests[0].Status.Open() {
		t.Fatalf("the ask stayed open after the analyzer failed: %+v", store.requests[0])
	}
	if !strings.Contains(store.requests[0].RefusalReason, "duplicate session") {
		t.Fatalf("the ask was closed without the adapter's own words: %q", store.requests[0].RefusalReason)
	}

	// Retrying must now be possible rather than refused as a conflict.
	analyzer.failAfterOpen = nil
	if _, err := service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0}); err != nil {
		t.Fatalf("retry after a failed spawn = %v, want a fresh ask", err)
	}
	if len(store.requests) != 2 {
		t.Fatalf("retry did not open a new ask: %d", len(store.requests))
	}
}

type recordingReaper struct{ killed []string }

func (r *recordingReaper) Kill(_ context.Context, sessionID string) error {
	r.killed = append(r.killed, sessionID)
	return nil
}

// A proposing session has exactly one job and is finished the moment its ask
// closes — however it closes. Leaving it running holds a worktree and a
// runtime name the next spawn for the same project collides with.
func TestEveryClosedAskReapsItsProposingSession(t *testing.T) {
	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)

	for _, test := range []struct {
		name  string
		close func(*testing.T, *Service, *deferringAnalyzer, domain.IntakeSessionID)
	}{
		{name: "fulfilled", close: func(t *testing.T, s *Service, a *deferringAnalyzer, _ domain.IntakeSessionID) {
			if _, err := s.SubmitAgentProposal(context.Background(), a.callback.RequestID, a.callback.Token,
				ports.IntakeAnalysisResult{Proposal: validProposalDraft("Agent proposal")}, "{}"); err != nil {
				t.Fatalf("submit: %v", err)
			}
		}},
		{name: "refused", close: func(t *testing.T, s *Service, a *deferringAnalyzer, _ domain.IntakeSessionID) {
			if _, err := s.SubmitAgentProposal(context.Background(), a.callback.RequestID, a.callback.Token,
				ports.IntakeAnalysisResult{Proposal: &domain.OutcomeContractProposal{}}, "{}"); err == nil {
				t.Fatal("an invalid proposal was accepted")
			}
		}},
		{name: "cancelled", close: func(t *testing.T, s *Service, _ *deferringAnalyzer, id domain.IntakeSessionID) {
			if _, err := s.CancelAnalysisRequest(context.Background(), id); err != nil {
				t.Fatalf("cancel: %v", err)
			}
		}},
		{name: "expired", close: func(t *testing.T, s *Service, _ *deferringAnalyzer, _ domain.IntakeSessionID) {
			late := New(s.store, nil, func() time.Time { return now.Add(domain.DefaultIntakeAnalysisRequestTTL + time.Minute) })
			late.reaper = s.reaper
			if closed, err := late.ExpireStaleAnalysisRequests(context.Background()); err != nil || closed != 1 {
				t.Fatalf("expire = %d err=%v", closed, err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &memoryIntakeStore{}
			analyzer := &deferringAnalyzer{}
			reaper := &recordingReaper{}
			service := New(store, analyzer, func() time.Time { return now }).WithAnalystSessionReaper(reaper)
			captured, err := service.Capture(context.Background(), CaptureInput{SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome, ProjectID: "project-1", Statement: "Let an agent draft this", RequestKey: "capture-" + test.name})
			if err != nil {
				t.Fatalf("Capture() error = %v", err)
			}
			if _, err := service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0}); err != nil {
				t.Fatalf("Analyze() error = %v", err)
			}

			test.close(t, service, analyzer, captured.Session.ID)

			if len(reaper.killed) != 1 || reaper.killed[0] != "sess-analyst" {
				t.Fatalf("closing as %s reaped %v, want the one proposing session", test.name, reaper.killed)
			}
		})
	}
}

// Reaping runs only after the ask is durably closed, so a kill that fails
// costs a stray process rather than a record that disagrees with reality.
func TestAFailedReapNeverUndoesTheDurableClose(t *testing.T) {
	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	store := &memoryIntakeStore{}
	analyzer := &deferringAnalyzer{}
	service := New(store, analyzer, func() time.Time { return now }).WithAnalystSessionReaper(failingReaper{})
	captured, _ := service.Capture(context.Background(), CaptureInput{SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome, ProjectID: "project-1", Statement: "Let an agent draft this", RequestKey: "capture-reapfail"})
	if _, err := service.Analyze(context.Background(), captured.Session.ID, AnalyzeInput{ExpectedProposalRevision: 0}); err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	ready, err := service.SubmitAgentProposal(context.Background(), analyzer.callback.RequestID, analyzer.callback.Token,
		ports.IntakeAnalysisResult{Proposal: validProposalDraft("Agent proposal")}, "{}")
	if err != nil {
		t.Fatalf("a failing reap broke the answer: %v", err)
	}
	if ready.Session.Status != domain.IntakeStatusReady || store.requests[0].Status != domain.IntakeAnalysisFulfilled {
		t.Fatalf("close did not stand: %+v / %+v", ready.Session, store.requests[0])
	}
}

type failingReaper struct{}

func (failingReaper) Kill(context.Context, string) error { return errors.New("tmux is gone") }

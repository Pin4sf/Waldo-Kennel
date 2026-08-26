package intake

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

type memoryIntakeStore struct {
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
	result  ports.IntakeAnalysisResult
	results []ports.IntakeAnalysisResult
	err     error
	seen    ports.IntakeAnalysisInput
	calls   int
}

func (analyzer *scriptedAnalyzer) Analyze(_ context.Context, input ports.IntakeAnalysisInput) (ports.IntakeAnalysisResult, error) {
	analyzer.seen = input
	if analyzer.calls < len(analyzer.results) {
		result := analyzer.results[analyzer.calls]
		analyzer.calls++
		return result, analyzer.err
	}
	analyzer.calls++
	return analyzer.result, analyzer.err
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

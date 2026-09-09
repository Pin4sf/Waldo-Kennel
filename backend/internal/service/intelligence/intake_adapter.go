package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// IntakeAnalyzer adapts the provider-neutral intelligence port to the existing
// shared Intake state machine while the public Intake API remains stable. It
// creates IntelligenceRun provenance and always returns structured inline
// proposal material; it never creates an execution Session/Attempt.
type IntakeAnalyzer struct {
	provider ports.IntelligenceProvider
	runs     ports.IntelligenceRunStore
	clock    func() time.Time
	projects ProjectSource
}

// NewIntakeAnalyzer constructs the Intake intelligence adapter.
func NewIntakeAnalyzer(provider ports.IntelligenceProvider, runs ports.IntelligenceRunStore, clock func() time.Time) *IntakeAnalyzer {
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &IntakeAnalyzer{provider: provider, runs: runs, clock: clock}
}

// WithRepositoryContextSource enables bounded grounding from the registered
// project. It is optional so historical/unit callers can still exercise the
// provider seam without a filesystem.
func (a *IntakeAnalyzer) WithRepositoryContextSource(source ProjectSource) *IntakeAnalyzer {
	a.projects = source
	return a
}

// Analyze records bounded contract-analysis provenance and returns its proposal.
func (a *IntakeAnalyzer) Analyze(ctx context.Context, input ports.IntakeAnalysisInput) (ports.IntakeAnalysisTicket, error) {
	if a == nil || a.provider == nil || a.runs == nil {
		return ports.IntakeAnalysisTicket{}, fmt.Errorf("intelligence provider is not configured")
	}
	request := ports.ContractIntelligenceRequest{
		Session:           input.Session,
		ConversationRefs:  input.ConversationRefs,
		PreviousProposal:  input.PreviousProposal,
		Clarification:     input.Clarification,
		ClarificationText: input.ClarificationText,
	}
	if a.projects != nil {
		project, found, err := a.projects.GetProject(ctx, string(input.Session.ProjectID))
		if err != nil {
			return ports.IntakeAnalysisTicket{}, fmt.Errorf("load project for contract grounding: %w", err)
		}
		if !found {
			return ports.IntakeAnalysisTicket{}, fmt.Errorf("project %s is not registered", input.Session.ProjectID)
		}
		var brief briefSource
		if candidate, ok := a.projects.(briefSource); ok {
			brief = candidate
		}
		request.RepositoryContext, err = BuildRepositoryContext(ctx, project, brief)
		if err != nil {
			return ports.IntakeAnalysisTicket{}, err
		}
	}
	inputDigest, err := digestContractRequest(request)
	if err != nil {
		return ports.IntakeAnalysisTicket{}, err
	}
	now := a.clock().UTC()
	started := time.Now()
	run := domain.IntelligenceRun{
		ID:                domain.IntelligenceRunID("intel-" + uuid.NewString()),
		Kind:              domain.IntelligenceRunContractAnalysis,
		ProjectID:         input.Session.ProjectID,
		IntakeID:          input.Session.ID,
		SourceRevision:    input.Session.CurrentProposalRevision,
		RequestedProvider: a.provider.ID(),
		InputDigest:       inputDigest,
		Status:            domain.IntelligenceRunRequested,
		CreatedAt:         now,
	}
	if err := a.runs.CreateIntelligenceRun(ctx, run); err != nil {
		return ports.IntakeAnalysisTicket{}, err
	}
	if err := a.runs.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunRunning, "", "", "", nil); err != nil {
		return ports.IntakeAnalysisTicket{}, err
	}

	response, err := a.provider.AnalyzeContract(ctx, request)
	if err != nil {
		completed := a.clock().UTC()
		duration := time.Since(started).Milliseconds()
		if metricErr := a.runs.RecordIntelligenceRunMetrics(ctx, run.ID, nil, nil, &duration); metricErr != nil {
			_ = a.runs.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFailed, "", "INTELLIGENCE_STATUS_PERSIST_FAILED", "Reasoning status could not be persisted safely", &completed)
			return ports.IntakeAnalysisTicket{}, fmt.Errorf("contract intelligence failed and recovery state could not be recorded: %w", metricErr)
		}
		// Persist a stable code and non-sensitive summary; provider-native errors
		// may contain prompts/tokens and do not belong in canonical provenance.
		_ = a.runs.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFailed, "", "INTELLIGENCE_PROVIDER_FAILED", "Contract intelligence provider failed", &completed)
		return ports.IntakeAnalysisTicket{}, err
	}
	duration := time.Since(started).Milliseconds()
	if err := a.runs.RecordIntelligenceRunMetrics(ctx, run.ID, responseMetrics(response), responseOutputMetrics(response), &duration); err != nil {
		completed := a.clock().UTC()
		_ = a.runs.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFailed, "", "INTELLIGENCE_STATUS_PERSIST_FAILED", "Reasoning status could not be persisted safely", &completed)
		return ports.IntakeAnalysisTicket{}, fmt.Errorf("record contract intelligence metrics: %w", err)
	}
	if err := a.runs.RecordIntelligenceRunEffectiveProvenance(ctx, run.ID,
		response.Provenance.EffectiveProvider, response.Provenance.EffectiveModel, response.Provenance.NativeSessionRef); err != nil {
		return ports.IntakeAnalysisTicket{}, err
	}
	outputDigest, err := digestContractResult(response.Result)
	if err != nil {
		return ports.IntakeAnalysisTicket{}, err
	}
	completed := a.clock().UTC()
	if err := a.runs.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFulfilled, outputDigest, "", "", &completed); err != nil {
		return ports.IntakeAnalysisTicket{}, err
	}
	result := response.Result
	return ports.IntakeAnalysisTicket{Inline: &result, Detail: "Contract proposal ready"}, nil
}

func responseMetrics(response ports.ContractIntelligenceResponse) *int64 {
	return response.Provenance.InputTokens
}
func responseOutputMetrics(response ports.ContractIntelligenceResponse) *int64 {
	return response.Provenance.OutputTokens
}

func digestContractRequest(request ports.ContractIntelligenceRequest) (domain.SHA256Digest, error) {
	payload := struct {
		IntakeID            string                          `json:"intakeId"`
		ProjectID           string                          `json:"projectId"`
		Statement           string                          `json:"statement"`
		ProposalRevision    int64                           `json:"proposalRevision"`
		ConversationRefs    []domain.IntakeConversationRef  `json:"conversationRefs"`
		PreviousProposal    *domain.OutcomeContractProposal `json:"previousProposal,omitempty"`
		Clarification       *domain.ClarificationRequest    `json:"clarification,omitempty"`
		ClarificationAnswer string                          `json:"clarificationAnswer,omitempty"`
		RepositoryContext   ports.RepositoryContextSnapshot `json:"repositoryContext"`
	}{
		IntakeID: request.Session.ID.String(), ProjectID: string(request.Session.ProjectID),
		Statement: request.Session.Statement, ProposalRevision: request.Session.CurrentProposalRevision,
		ConversationRefs: request.ConversationRefs, PreviousProposal: request.PreviousProposal,
		Clarification: request.Clarification, ClarificationAnswer: request.ClarificationText,
		RepositoryContext: request.RepositoryContext,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode contract intelligence input: %w", err)
	}
	return domain.DigestSHA256(encoded), nil
}

func digestContractResult(result ports.IntakeAnalysisResult) (domain.SHA256Digest, error) {
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode contract intelligence result: %w", err)
	}
	return domain.DigestSHA256(encoded), nil
}

var _ ports.IntakeAnalyzer = (*IntakeAnalyzer)(nil)

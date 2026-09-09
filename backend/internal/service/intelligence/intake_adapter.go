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
}

// NewIntakeAnalyzer constructs the Intake intelligence adapter.
func NewIntakeAnalyzer(provider ports.IntelligenceProvider, runs ports.IntelligenceRunStore, clock func() time.Time) *IntakeAnalyzer {
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &IntakeAnalyzer{provider: provider, runs: runs, clock: clock}
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
	inputDigest, err := digestContractRequest(request)
	if err != nil {
		return ports.IntakeAnalysisTicket{}, err
	}
	now := a.clock().UTC()
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
		// Persist a stable code and non-sensitive summary; provider-native errors
		// may contain prompts/tokens and do not belong in canonical provenance.
		_ = a.runs.UpdateIntelligenceRunStatus(ctx, run.ID, domain.IntelligenceRunFailed, "", "INTELLIGENCE_PROVIDER_FAILED", "Contract intelligence provider failed", &completed)
		return ports.IntakeAnalysisTicket{}, err
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
	}{
		IntakeID: request.Session.ID.String(), ProjectID: string(request.Session.ProjectID),
		Statement: request.Session.Statement, ProposalRevision: request.Session.CurrentProposalRevision,
		ConversationRefs: request.ConversationRefs, PreviousProposal: request.PreviousProposal,
		Clarification: request.Clarification, ClarificationAnswer: request.ClarificationText,
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

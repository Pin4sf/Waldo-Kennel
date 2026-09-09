package ports

import (
	"context"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// RepositoryContextSnapshot is a bounded, read-only grounding packet. It is
// evidence for reasoning, never an execution grant or a claim that checks ran.
type RepositoryContextSnapshot struct {
	ProjectID         domain.ProjectID
	Root              string
	Revision          string
	Dirty             bool
	UnavailableReason string
	Instructions      []RepositoryContextFile
	Files             []RepositoryContextFile
	ProjectBrief      *domain.ProjectBriefRevision
	CheckCommands     []string
	Digest            domain.SHA256Digest
}

type RepositoryContextFile struct {
	Path    string
	Content string
}

// IntelligenceProvenance is provider-reported reasoning provenance. Unknown
// fields stay empty; adapters must never fabricate effective model identity.
type IntelligenceProvenance struct {
	EffectiveProvider domain.IntelligenceProviderID
	EffectiveModel    string
	NativeSessionRef  string
	InputTokens       *int64
	OutputTokens      *int64
}

// ContractIntelligenceRequest is bounded pre-Outcome reasoning input. It carries
// only canonical/intake data; provider-specific permission flags belong inside
// the adapter.
type ContractIntelligenceRequest struct {
	Session           domain.IntakeSession
	ConversationRefs  []domain.IntakeConversationRef
	PreviousProposal  *domain.OutcomeContractProposal
	Clarification     *domain.ClarificationRequest
	ClarificationText string
	RepositoryContext RepositoryContextSnapshot
}

// ContractIntelligenceResponse is structured proposal material. It is not a
// ContractRevision and cannot create responsibility without owner confirmation.
type ContractIntelligenceResponse struct {
	Result     IntakeAnalysisResult
	Provenance IntelligenceProvenance
}

// PlanIntelligenceRequest freezes the exact Outcome/Contract input the provider
// is asked to reason about. CriterionAliases maps model-friendly C1/C2 labels
// to the immutable Contract criteria the control plane owns.
type PlanIntelligenceRequest struct {
	Outcome           domain.Outcome
	Contract          domain.ContractRevision
	CriterionAliases  map[string]domain.CriterionID
	RepositoryContext RepositoryContextSnapshot
	ReplanFeedback    string
}

// PlanIntelligenceResponse is non-authoritative planning material. The control
// plane still validates graph shape, derives authority, routes, and compiles the
// canonical PlanRevision.
type PlanIntelligenceResponse struct {
	Proposal   domain.PlanDraftProposal
	Provenance IntelligenceProvenance
}

// IntelligenceProvider proposes understanding/plans without execution
// authority. Provider implementations may use a local runtime, direct API, or
// deterministic logic; the Outcome control plane never depends on that choice.
type IntelligenceProvider interface {
	ID() domain.IntelligenceProviderID
	AnalyzeContract(context.Context, ContractIntelligenceRequest) (ContractIntelligenceResponse, error)
	DraftPlan(context.Context, PlanIntelligenceRequest) (PlanIntelligenceResponse, error)
}

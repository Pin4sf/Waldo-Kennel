package ports

import (
	"context"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

// OutcomeProofStore is the append-only durability boundary for Work E. It is
// deliberately separate from OutcomeStore so earlier contract/plan/attempt
// fakes do not gain proof authority by interface widening.
type OutcomeProofStore interface {
	CreateEvidenceItem(context.Context, domain.EvidenceItem) error
	FindEvidenceItemByRequestKey(context.Context, string) (domain.EvidenceItem, bool, error)
	GetEvidenceItem(context.Context, domain.OutcomeID, domain.EvidenceItemID) (domain.EvidenceItem, bool, error)
	ListEvidenceItems(context.Context, domain.OutcomeID) ([]domain.EvidenceItem, error)

	CreateVerificationRun(context.Context, domain.VerificationRun) error
	FindVerificationRunByRequestKey(context.Context, string) (domain.VerificationRun, bool, error)
	ListVerificationRuns(context.Context, domain.OutcomeID) ([]domain.VerificationRun, error)

	// CreateAcceptanceDecision persists the decision and optional correction
	// atomically: rework/reopen can never exist without its re-entry lineage.
	CreateAcceptanceDecision(context.Context, domain.AcceptanceDecision, *domain.OutcomeCorrection) error
	FindAcceptanceDecisionByRequestKey(context.Context, string) (domain.AcceptanceDecision, bool, error)
	ListAcceptanceDecisions(context.Context, domain.OutcomeID) ([]domain.AcceptanceDecision, error)
	ListOutcomeCorrections(context.Context, domain.OutcomeID) ([]domain.OutcomeCorrection, error)
}

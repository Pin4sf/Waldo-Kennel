package ports

import (
	"context"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// IntelligenceRunStore is the narrow durable provenance seam used by the
// control plane. It owns reasoning-run state only; no method can create an
// Attempt, WorkUnit, AgentSessionRef, capability grant, or AcceptanceDecision.
type IntelligenceRunStore interface {
	CreateIntelligenceRun(context.Context, domain.IntelligenceRun) error
	GetIntelligenceRun(context.Context, domain.IntelligenceRunID) (domain.IntelligenceRun, bool, error)
	ListNonTerminalIntelligenceRuns(context.Context) ([]domain.IntelligenceRun, error)
	UpdateIntelligenceRunStatus(
		context.Context,
		domain.IntelligenceRunID,
		domain.IntelligenceRunStatus,
		string,
		string,
		string,
		*time.Time,
	) error
}

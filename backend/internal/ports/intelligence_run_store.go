package ports

import (
	"context"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// IntelligenceRunStore owns durable pre-authorization reasoning provenance.
type IntelligenceRunStore interface {
	CreateIntelligenceRun(context.Context, domain.IntelligenceRun) error
	GetIntelligenceRun(context.Context, domain.IntelligenceRunID) (domain.IntelligenceRun, bool, error)
	ListNonTerminalIntelligenceRuns(context.Context) ([]domain.IntelligenceRun, error)
	RecordIntelligenceRunEffectiveProvenance(
		context.Context,
		domain.IntelligenceRunID,
		domain.IntelligenceProviderID,
		string,
		string,
	) error
	UpdateIntelligenceRunStatus(
		context.Context,
		domain.IntelligenceRunID,
		domain.IntelligenceRunStatus,
		domain.SHA256Digest,
		string,
		string,
		*time.Time,
	) error
}

// IntakeAnalysisIntelligenceLinkStore associates the existing single-use
// callback envelope with one canonical IntelligenceRun during migration.
type IntakeAnalysisIntelligenceLinkStore interface {
	BindIntakeAnalysisRequestIntelligenceRun(context.Context, domain.IntakeAnalysisRequestID, domain.IntelligenceRunID) error
	GetIntakeAnalysisRequestIntelligenceRun(context.Context, domain.IntakeAnalysisRequestID) (domain.IntelligenceRunID, bool, error)
}

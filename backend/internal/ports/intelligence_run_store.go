package ports

import (
	"context"
	"errors"
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

var (
	// ErrIntakeAnalysisIntelligenceRunBound indicates an existing provenance link.
	ErrIntakeAnalysisIntelligenceRunBound = errors.New("intake analysis request already has intelligence provenance")
	// ErrIntakeAnalysisIntelligenceRunUsed indicates a run already linked elsewhere.
	ErrIntakeAnalysisIntelligenceRunUsed = errors.New("intelligence run is already bound to an intake analysis request")
	// ErrIntakeAnalysisIntelligenceLineage indicates mismatched provenance.
	ErrIntakeAnalysisIntelligenceLineage = errors.New("intelligence run does not match intake analysis request lineage")
)

// IntakeAnalysisIntelligenceLinkStore associates the existing single-use
// callback envelope with one canonical IntelligenceRun during migration.
type IntakeAnalysisIntelligenceLinkStore interface {
	BindIntakeAnalysisRequestIntelligenceRun(context.Context, domain.IntakeAnalysisRequestID, domain.IntelligenceRunID) error
	GetIntakeAnalysisRequestIntelligenceRun(context.Context, domain.IntakeAnalysisRequestID) (domain.IntelligenceRunID, bool, error)
}

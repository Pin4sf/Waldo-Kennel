package domain

import (
	"fmt"
	"strings"
	"time"
)

// IntelligenceRunID identifies one bounded pre-authorization reasoning run.
type IntelligenceRunID string

func (id IntelligenceRunID) IsZero() bool  { return strings.TrimSpace(string(id)) == "" }
func (id IntelligenceRunID) String() string { return string(id) }

// IntelligenceProviderID is opaque intelligence provenance. It is deliberately
// independent from AgentHarness because direct APIs and hosted inference are not
// execution harnesses.
type IntelligenceProviderID string

func (id IntelligenceProviderID) IsZero() bool  { return strings.TrimSpace(string(id)) == "" }
func (id IntelligenceProviderID) String() string { return string(id) }

// IntelligenceRunKind identifies the bounded proposal-producing purpose.
type IntelligenceRunKind string

const (
	IntelligenceRunContractAnalysis IntelligenceRunKind = "contract_analysis"
	IntelligenceRunPlanDraft        IntelligenceRunKind = "plan_draft"
)

func (k IntelligenceRunKind) Valid() bool {
	switch k {
	case IntelligenceRunContractAnalysis, IntelligenceRunPlanDraft:
		return true
	default:
		return false
	}
}

// IntelligenceRunStatus is durable reconciliation state for one intelligence run.
type IntelligenceRunStatus string

const (
	IntelligenceRunRequested IntelligenceRunStatus = "requested"
	IntelligenceRunRunning   IntelligenceRunStatus = "running"
	IntelligenceRunFulfilled IntelligenceRunStatus = "fulfilled"
	IntelligenceRunFailed    IntelligenceRunStatus = "failed"
	IntelligenceRunCancelled IntelligenceRunStatus = "cancelled"
	IntelligenceRunExpired   IntelligenceRunStatus = "expired"
)

func (s IntelligenceRunStatus) Valid() bool {
	switch s {
	case IntelligenceRunRequested, IntelligenceRunRunning, IntelligenceRunFulfilled,
		IntelligenceRunFailed, IntelligenceRunCancelled, IntelligenceRunExpired:
		return true
	default:
		return false
	}
}

func (s IntelligenceRunStatus) Terminal() bool {
	switch s {
	case IntelligenceRunFulfilled, IntelligenceRunFailed, IntelligenceRunCancelled, IntelligenceRunExpired:
		return true
	default:
		return false
	}
}

// IntelligenceRun is durable, bounded, non-authoritative reasoning provenance.
// Requested/effective provider and model identities are recorded only when
// known; empty means unknown, never an implicit fallback.
type IntelligenceRun struct {
	ID                 IntelligenceRunID
	Kind               IntelligenceRunKind
	ProjectID          ProjectID
	IntakeID           IntakeSessionID
	OutcomeID          OutcomeID
	ContractRevisionID ContractRevisionID
	SourceRevision     int64

	RequestedProvider IntelligenceProviderID
	RequestedModel    string
	EffectiveProvider IntelligenceProviderID
	EffectiveModel    string
	NativeSessionRef  string

	InputDigest   SHA256Digest
	OutputDigest  SHA256Digest
	Status        IntelligenceRunStatus
	FailureCode   string
	FailureDetail string
	CreatedAt     time.Time
	CompletedAt   *time.Time
}

// Validate checks source lineage, provenance, digest representation, and
// terminal timestamp semantics. Execution authority cannot be expressed here.
func (r IntelligenceRun) Validate() error {
	if r.ID.IsZero() {
		return fmt.Errorf("intelligence run id is required")
	}
	if !r.Kind.Valid() {
		return fmt.Errorf("unsupported intelligence run kind %q", r.Kind)
	}
	if strings.TrimSpace(string(r.ProjectID)) == "" {
		return fmt.Errorf("intelligence run project id is required")
	}
	if !r.Status.Valid() {
		return fmt.Errorf("unsupported intelligence run status %q", r.Status)
	}
	if r.SourceRevision < 0 {
		return fmt.Errorf("intelligence run source revision must not be negative")
	}
	if !r.InputDigest.Valid() {
		return fmt.Errorf("intelligence run input digest must be a SHA-256 hexadecimal digest")
	}
	if !r.OutputDigest.IsZero() && !r.OutputDigest.Valid() {
		return fmt.Errorf("intelligence run output digest must be a SHA-256 hexadecimal digest")
	}
	if r.CreatedAt.IsZero() {
		return fmt.Errorf("intelligence run created time is required")
	}

	switch r.Kind {
	case IntelligenceRunContractAnalysis:
		if r.IntakeID.IsZero() {
			return fmt.Errorf("contract-analysis intelligence run requires intake id")
		}
		if !r.OutcomeID.IsZero() || !r.ContractRevisionID.IsZero() {
			return fmt.Errorf("contract-analysis intelligence run must not carry outcome or contract lineage")
		}
	case IntelligenceRunPlanDraft:
		if !r.IntakeID.IsZero() {
			return fmt.Errorf("plan-draft intelligence run must not carry intake lineage")
		}
		if r.OutcomeID.IsZero() {
			return fmt.Errorf("plan-draft intelligence run requires outcome id")
		}
		if r.ContractRevisionID.IsZero() {
			return fmt.Errorf("plan-draft intelligence run requires exact contract revision id")
		}
		if r.SourceRevision < 1 {
			return fmt.Errorf("plan-draft intelligence run source revision must be at least 1")
		}
	}

	if r.RequestedProvider.IsZero() && strings.TrimSpace(r.RequestedModel) != "" {
		return fmt.Errorf("requested intelligence model requires requested provider")
	}
	if r.EffectiveProvider.IsZero() {
		if strings.TrimSpace(r.EffectiveModel) != "" {
			return fmt.Errorf("effective intelligence model requires effective provider")
		}
		if strings.TrimSpace(r.NativeSessionRef) != "" {
			return fmt.Errorf("native intelligence session reference requires effective provider")
		}
	}

	if r.Status == IntelligenceRunFulfilled && r.OutputDigest.IsZero() {
		return fmt.Errorf("fulfilled intelligence run requires output digest")
	}
	if r.Status.Terminal() {
		if r.CompletedAt == nil || r.CompletedAt.IsZero() {
			return fmt.Errorf("terminal intelligence run requires completion time")
		}
	} else if r.CompletedAt != nil {
		return fmt.Errorf("non-terminal intelligence run must not have completion time")
	}
	return nil
}

// TransitionTo applies the durable state machine. Terminal completion time is
// supplied by the control plane; the domain never reads the wall clock itself.
func (r *IntelligenceRun) TransitionTo(next IntelligenceRunStatus, completedAt *time.Time) error {
	if r == nil {
		return fmt.Errorf("intelligence run is nil")
	}
	if !next.Valid() {
		return fmt.Errorf("unsupported intelligence run status %q", next)
	}
	if next.Terminal() {
		if completedAt == nil || completedAt.IsZero() {
			return fmt.Errorf("terminal intelligence run transition requires completion time")
		}
	} else if completedAt != nil {
		return fmt.Errorf("non-terminal intelligence run transition must not set completion time")
	}

	if r.Status == next {
		if next.Terminal() && (r.CompletedAt == nil || !r.CompletedAt.Equal(completedAt.UTC())) {
			return fmt.Errorf("terminal intelligence run completion time is immutable")
		}
		return nil
	}
	if r.Status.Terminal() {
		return fmt.Errorf("terminal intelligence run status %q cannot transition to %q", r.Status, next)
	}

	allowed := false
	switch r.Status {
	case IntelligenceRunRequested:
		allowed = next == IntelligenceRunRunning || next.Terminal()
	case IntelligenceRunRunning:
		allowed = next.Terminal()
	}
	if !allowed {
		return fmt.Errorf("intelligence run status %q cannot transition to %q", r.Status, next)
	}

	r.Status = next
	if next.Terminal() {
		t := completedAt.UTC()
		r.CompletedAt = &t
	}
	return nil
}

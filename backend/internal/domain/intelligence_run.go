package domain

import (
	"fmt"
	"strings"
	"time"
)

// IntelligenceRunID identifies one bounded pre-authorization reasoning run.
// It is deliberately distinct from AttemptID and provider execution sessions.
type IntelligenceRunID string

func (id IntelligenceRunID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }
func (id IntelligenceRunID) String() string { return string(id) }

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

// IntelligenceRunStatus is durable reconciliation state for one intelligence
// run. Terminal states never transition back into execution.
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

// IntelligenceRunModelSelection records model provenance without turning it
// into WorkUnit execution authority.
type IntelligenceRunModelSelection string

const (
	IntelligenceRunModelProviderDefault IntelligenceRunModelSelection = "provider_default"
	IntelligenceRunModelExplicit        IntelligenceRunModelSelection = "explicit"
	// Unknown is valid provenance when a provider runtime owns its effective
	// model and cannot report a stable identity. It is never copied into an
	// execution binding.
	IntelligenceRunModelUnknown IntelligenceRunModelSelection = "unknown"
)

// IntelligenceRun is a durable, bounded, non-authoritative reasoning fact.
// It may produce Contract or Plan proposals, but it intentionally carries no
// Attempt, WorkUnit, AgentSessionRef, capability grant, acceptance, or
// execution-binding fields.
type IntelligenceRun struct {
	ID                   IntelligenceRunID
	Kind                 IntelligenceRunKind
	ProjectID            ProjectID
	IntakeID             IntakeSessionID
	OutcomeID            OutcomeID
	ContractRevisionID   ContractRevisionID
	SourceRevision       int64
	Provider             AgentHarness
	ModelSelection       IntelligenceRunModelSelection
	Model                string
	InputDigest          string
	OutputDigest         string
	NativeSessionRef     string
	Status               IntelligenceRunStatus
	FailureCode          string
	FailureDetail        string
	CreatedAt            time.Time
	CompletedAt          *time.Time
}

// Validate checks identity, source lineage, and provenance only. Execution
// authority is intentionally impossible to express on this object.
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
	if strings.TrimSpace(r.InputDigest) == "" {
		return fmt.Errorf("intelligence run input digest is required")
	}
	if r.CreatedAt.IsZero() {
		return fmt.Errorf("intelligence run created time is required")
	}

	switch r.Kind {
	case IntelligenceRunContractAnalysis:
		if strings.TrimSpace(string(r.IntakeID)) == "" {
			return fmt.Errorf("contract-analysis intelligence run requires intake id")
		}
	case IntelligenceRunPlanDraft:
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

	provider := strings.TrimSpace(string(r.Provider))
	model := strings.TrimSpace(r.Model)
	if provider == "" {
		if r.ModelSelection != "" || model != "" {
			return fmt.Errorf("intelligence model provenance requires provider")
		}
		if strings.TrimSpace(r.NativeSessionRef) != "" {
			return fmt.Errorf("native intelligence session reference requires provider")
		}
	} else {
		if !r.Provider.IsRecognizedPersisted() {
			return fmt.Errorf("unsupported intelligence provider %q", r.Provider)
		}
		switch r.ModelSelection {
		case IntelligenceRunModelExplicit:
			if model == "" {
				return fmt.Errorf("explicit intelligence model is required")
			}
		case IntelligenceRunModelProviderDefault, IntelligenceRunModelUnknown:
			if model != "" {
				return fmt.Errorf("%s intelligence model selection must not name a model", r.ModelSelection)
			}
		default:
			return fmt.Errorf("intelligence provider requires model-selection provenance")
		}
	}
	if r.Status == IntelligenceRunFulfilled && strings.TrimSpace(r.OutputDigest) == "" {
		return fmt.Errorf("fulfilled intelligence run requires output digest")
	}
	return nil
}

// TransitionTo applies the small durable run-state machine. Replaying the same
// status is idempotent; terminal runs cannot be revived.
func (r *IntelligenceRun) TransitionTo(next IntelligenceRunStatus) error {
	if r == nil {
		return fmt.Errorf("intelligence run is nil")
	}
	if !next.Valid() {
		return fmt.Errorf("unsupported intelligence run status %q", next)
	}
	if r.Status == next {
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
		now := time.Now().UTC()
		r.CompletedAt = &now
	}
	return nil
}

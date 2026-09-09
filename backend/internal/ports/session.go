package ports

import (
	"errors"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// ErrSessionNotFound indicates that the requested session does not exist.
var ErrSessionNotFound = errors.New("session not found")

// SpawnConfig starts one subordinate provider session. Ordinary session callers
// may rely on Project role defaults. ExactExecutionBinding is reserved for an
// approved Outcome WorkUnit: when present, provider/model selection is already
// authority and the session manager must not inherit or substitute mutable
// Project provider/model preferences.
type SpawnConfig struct {
	ProjectID       domain.ProjectID
	IssueID         domain.IssueID
	TrackerProvider domain.TrackerProvider
	IssueContext    string
	Kind            domain.SessionKind
	Harness         domain.AgentHarness
	Branch          string
	Prompt          string
	AgentConfig     AgentConfig

	// ExactExecutionBinding is non-nil only for governed Attempt execution.
	// provider_default deliberately clears any Project model preference;
	// explicit requires exactly Binding.Model. It never changes permissions or
	// other provider-neutral session configuration.
	ExactExecutionBinding *domain.ExecutionBinding

	RequestedMode domain.SessionMode
	DisplayName   string
	Attachments   []SpawnAttachment
}

// SpawnAttachment carries bounded context attached to a spawned session.
type SpawnAttachment struct {
	Ext  string
	Data []byte
}

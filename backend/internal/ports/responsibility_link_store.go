package ports

import (
	"context"
	"fmt"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

// ResponsibilityLinkIdempotency binds a request key to lineage input.
type ResponsibilityLinkIdempotency struct {
	Key         string
	Fingerprint string
}

// ResponsibilityLinkDuplicateError reports an already active endpoint pair.
type ResponsibilityLinkDuplicateError struct {
	SourceOpenLoopID     domain.OpenLoopID
	DestinationOutcomeID domain.OutcomeID
}

func (err *ResponsibilityLinkDuplicateError) Error() string {
	return fmt.Sprintf("active responsibility link already exists from %s to %s", err.SourceOpenLoopID, err.DestinationOutcomeID)
}

// ResponsibilityLinkStore persists explicit lineage independently of lifecycle.
type ResponsibilityLinkStore interface {
	GetProject(context.Context, string) (domain.ProjectRecord, bool, error)
	GetOutcomeProjectID(context.Context, domain.OutcomeID) (domain.ProjectID, bool, error)
	CreateResponsibilityLink(context.Context, domain.ResponsibilityLink, ResponsibilityLinkIdempotency) (domain.ResponsibilityLink, error)
	GetResponsibilityLink(context.Context, domain.ResponsibilityLinkID) (domain.ResponsibilityLink, bool, error)
	EndResponsibilityLink(context.Context, domain.ResponsibilityLinkID, domain.ResponsibilityLinkCreator, string, time.Time) (domain.ResponsibilityLink, bool, error)
}

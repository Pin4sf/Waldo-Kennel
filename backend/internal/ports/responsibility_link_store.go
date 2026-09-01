package ports

import (
	"context"
	"fmt"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
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

// ResponsibilityLinkEndConflictError reports an already-ended lineage link.
type ResponsibilityLinkEndConflictError struct{ ID domain.ResponsibilityLinkID }

func (err *ResponsibilityLinkEndConflictError) Error() string {
	return fmt.Sprintf("responsibility link %s is already ended", err.ID)
}

// ResponsibilityLinkStore persists explicit lineage independently of lifecycle.
type ResponsibilityLinkStore interface {
	GetProject(context.Context, string) (domain.ProjectRecord, bool, error)
	GetOutcomeProjectID(context.Context, domain.OutcomeID) (domain.ProjectID, bool, error)
	CreateResponsibilityLink(context.Context, domain.ResponsibilityLink, ResponsibilityLinkIdempotency) (domain.ResponsibilityLink, error)
	GetResponsibilityLink(context.Context, domain.ResponsibilityLinkID) (domain.ResponsibilityLink, bool, error)
	EndResponsibilityLink(context.Context, domain.ResponsibilityLinkID, domain.ResponsibilityLinkCreator, string, time.Time) (domain.ResponsibilityLink, bool, error)
}

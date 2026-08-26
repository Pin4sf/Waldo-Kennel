package intake

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// CreateResponsibilityLinkInput records owner-confirmed lineage.
type CreateResponsibilityLinkInput struct {
	ProjectID            domain.ProjectID
	SourceOpenLoopID     domain.OpenLoopID
	DestinationOutcomeID domain.OutcomeID
	Reason               string
	RequestKey           string
}

// ResponsibilityLinkService owns lineage without owning either lifecycle.
type ResponsibilityLinkService struct {
	store ports.ResponsibilityLinkStore
	clock func() time.Time
}

// NewResponsibilityLinks constructs the lineage service.
func NewResponsibilityLinks(store ports.ResponsibilityLinkStore, clock func() time.Time) *ResponsibilityLinkService {
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &ResponsibilityLinkService{store: store, clock: clock}
}

// CreateResponsibilityLink creates idempotent explicit lineage.
func (service *ResponsibilityLinkService) CreateResponsibilityLink(ctx context.Context, input CreateResponsibilityLinkInput) (domain.ResponsibilityLink, error) {
	input.ProjectID = domain.ProjectID(strings.TrimSpace(string(input.ProjectID)))
	input.Reason = strings.TrimSpace(input.Reason)
	input.RequestKey = strings.TrimSpace(input.RequestKey)
	if input.ProjectID == "" {
		return domain.ResponsibilityLink{}, apierr.Invalid("PROJECT_REQUIRED", "Choose the Work project that owns the destination Outcome", nil)
	}
	if input.SourceOpenLoopID.IsZero() || input.DestinationOutcomeID.IsZero() || input.Reason == "" {
		return domain.ResponsibilityLink{}, apierr.Invalid("RESPONSIBILITY_LINK_INVALID", "Source Open Loop, destination Outcome, and owner reason are required", nil)
	}
	if input.RequestKey == "" {
		return domain.ResponsibilityLink{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this Responsibility Link", nil)
	}
	if _, found, err := service.store.GetProject(ctx, string(input.ProjectID)); err != nil {
		return domain.ResponsibilityLink{}, err
	} else if !found {
		return domain.ResponsibilityLink{}, apierr.NotFound("PROJECT_NOT_FOUND", "That Project does not exist")
	}
	destinationProjectID, found, err := service.store.GetOutcomeProjectID(ctx, input.DestinationOutcomeID)
	if err != nil {
		return domain.ResponsibilityLink{}, err
	}
	if !found {
		return domain.ResponsibilityLink{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That destination Outcome does not exist")
	}
	if destinationProjectID != input.ProjectID {
		return domain.ResponsibilityLink{}, apierr.Conflict("RESPONSIBILITY_LINK_PROJECT_CONFLICT", "The destination Outcome belongs to another Project", map[string]any{
			"projectId": input.ProjectID, "destinationProjectId": destinationProjectID,
		})
	}
	link := domain.ResponsibilityLink{
		ID:               domain.ResponsibilityLinkID("rlink-" + uuid.NewString()),
		SourceOpenLoopID: input.SourceOpenLoopID, DestinationOutcomeID: input.DestinationOutcomeID,
		Creator: domain.ResponsibilityLinkCreatorOwner, Reason: input.Reason, CreatedAt: service.clock(),
	}
	if err := link.Validate(); err != nil {
		return domain.ResponsibilityLink{}, apierr.Invalid("RESPONSIBILITY_LINK_INVALID", err.Error(), nil)
	}
	request := ports.ResponsibilityLinkIdempotency{
		Key:         input.RequestKey,
		Fingerprint: intakeFingerprint(string(input.ProjectID), input.SourceOpenLoopID.String(), input.DestinationOutcomeID.String(), input.Reason),
	}
	created, err := service.store.CreateResponsibilityLink(ctx, link, request)
	if err != nil {
		var duplicate *ports.ResponsibilityLinkDuplicateError
		if errors.As(err, &duplicate) {
			return domain.ResponsibilityLink{}, apierr.Conflict("RESPONSIBILITY_LINK_EXISTS", "An active Responsibility Link already preserves this lineage", map[string]any{
				"sourceOpenLoopId": duplicate.SourceOpenLoopID, "destinationOutcomeId": duplicate.DestinationOutcomeID,
			})
		}
		var idempotency *ports.IntakeIdempotencyConflictError
		if errors.As(err, &idempotency) {
			return domain.ResponsibilityLink{}, apierr.Conflict("RESPONSIBILITY_LINK_IDEMPOTENCY_CONFLICT", "That idempotency key belongs to different lineage input", map[string]any{"requestKey": idempotency.Key})
		}
		return domain.ResponsibilityLink{}, err
	}
	return created, nil
}

// EndResponsibilityLink ends lineage once without changing either endpoint.
func (service *ResponsibilityLinkService) EndResponsibilityLink(ctx context.Context, id domain.ResponsibilityLinkID, reason string) (domain.ResponsibilityLink, error) {
	reason = strings.TrimSpace(reason)
	if id.IsZero() || reason == "" {
		return domain.ResponsibilityLink{}, apierr.Invalid("RESPONSIBILITY_LINK_END_INVALID", "Responsibility Link and owner reason are required", nil)
	}
	ended, found, err := service.store.EndResponsibilityLink(ctx, id, domain.ResponsibilityLinkCreatorOwner, reason, service.clock())
	if err != nil {
		var conflict *ports.ResponsibilityLinkEndConflictError
		if errors.As(err, &conflict) {
			return domain.ResponsibilityLink{}, apierr.Conflict("RESPONSIBILITY_LINK_END_CONFLICT", err.Error(), nil)
		}
		return domain.ResponsibilityLink{}, apierr.Internal("RESPONSIBILITY_LINK_END_FAILED", "The Responsibility Link could not be ended")
	}
	if !found {
		return domain.ResponsibilityLink{}, apierr.NotFound("RESPONSIBILITY_LINK_NOT_FOUND", "That Responsibility Link does not exist")
	}
	return ended, nil
}

// GetResponsibilityLink returns one durable lineage record.
func (service *ResponsibilityLinkService) GetResponsibilityLink(ctx context.Context, id domain.ResponsibilityLinkID) (domain.ResponsibilityLink, error) {
	link, found, err := service.store.GetResponsibilityLink(ctx, id)
	if err != nil {
		return domain.ResponsibilityLink{}, err
	}
	if !found {
		return domain.ResponsibilityLink{}, apierr.NotFound("RESPONSIBILITY_LINK_NOT_FOUND", "That Responsibility Link does not exist")
	}
	return link, nil
}

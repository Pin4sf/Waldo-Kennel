package intake

import (
	"context"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

type memoryResponsibilityLinkStore struct {
	projects        map[string]domain.ProjectRecord
	outcomeProjects map[domain.OutcomeID]domain.ProjectID
	links           map[domain.ResponsibilityLinkID]domain.ResponsibilityLink
	requests        map[string]domain.ResponsibilityLinkID
	writes          int
	endErr          error
}

func (store *memoryResponsibilityLinkStore) GetProject(_ context.Context, id string) (domain.ProjectRecord, bool, error) {
	project, ok := store.projects[id]
	return project, ok, nil
}

func (store *memoryResponsibilityLinkStore) GetOutcomeProjectID(_ context.Context, id domain.OutcomeID) (domain.ProjectID, bool, error) {
	projectID, ok := store.outcomeProjects[id]
	return projectID, ok, nil
}

func (store *memoryResponsibilityLinkStore) CreateResponsibilityLink(_ context.Context, link domain.ResponsibilityLink, request ports.ResponsibilityLinkIdempotency) (domain.ResponsibilityLink, error) {
	if store.requests == nil {
		store.requests = map[string]domain.ResponsibilityLinkID{}
	}
	if store.links == nil {
		store.links = map[domain.ResponsibilityLinkID]domain.ResponsibilityLink{}
	}
	if replayID, ok := store.requests[request.Key]; ok {
		return store.links[replayID], nil
	}
	for _, existing := range store.links {
		if existing.EndedAt == nil && existing.SourceOpenLoopID == link.SourceOpenLoopID && existing.DestinationOutcomeID == link.DestinationOutcomeID {
			return domain.ResponsibilityLink{}, &ports.ResponsibilityLinkDuplicateError{SourceOpenLoopID: link.SourceOpenLoopID, DestinationOutcomeID: link.DestinationOutcomeID}
		}
	}
	store.writes++
	store.links[link.ID] = link
	store.requests[request.Key] = link.ID
	return link, nil
}

func (store *memoryResponsibilityLinkStore) GetResponsibilityLink(_ context.Context, id domain.ResponsibilityLinkID) (domain.ResponsibilityLink, bool, error) {
	link, ok := store.links[id]
	return link, ok, nil
}

func (store *memoryResponsibilityLinkStore) EndResponsibilityLink(_ context.Context, id domain.ResponsibilityLinkID, actor domain.ResponsibilityLinkCreator, reason string, at time.Time) (domain.ResponsibilityLink, bool, error) {
	if store.endErr != nil {
		return domain.ResponsibilityLink{}, true, store.endErr
	}
	link, ok := store.links[id]
	if !ok {
		return domain.ResponsibilityLink{}, false, nil
	}
	ended, err := link.End(actor, reason, at)
	if err != nil {
		return domain.ResponsibilityLink{}, true, err
	}
	store.links[id] = ended
	return ended, true, nil
}

func TestEndResponsibilityLinkDoesNotMisreportStoreFailureAsConflict(t *testing.T) {
	store := &memoryResponsibilityLinkStore{endErr: context.DeadlineExceeded}
	service := NewResponsibilityLinks(store, nil)
	_, err := service.EndResponsibilityLink(context.Background(), "rlink-1", "Owner requested release")
	assertAPIErrorCode(t, err, "RESPONSIBILITY_LINK_END_FAILED")
}

func TestCreateResponsibilityLinkRequiresProjectAndMatchingDestination(t *testing.T) {
	store := &memoryResponsibilityLinkStore{
		projects:        map[string]domain.ProjectRecord{"project-1": {ID: "project-1"}},
		outcomeProjects: map[domain.OutcomeID]domain.ProjectID{"out-1": "project-1"},
	}
	service := NewResponsibilityLinks(store, func() time.Time { return time.Date(2026, 8, 26, 4, 0, 0, 0, time.UTC) })

	_, err := service.CreateResponsibilityLink(context.Background(), CreateResponsibilityLinkInput{
		ProjectID: "missing", SourceOpenLoopID: "loop-1", DestinationOutcomeID: "out-1", Reason: "This Home responsibility needs bounded Work execution.", RequestKey: "link-missing",
	})
	assertAPIErrorCode(t, err, "PROJECT_NOT_FOUND")

	_, err = service.CreateResponsibilityLink(context.Background(), CreateResponsibilityLinkInput{
		ProjectID: "project-1", SourceOpenLoopID: "loop-1", DestinationOutcomeID: "out-404", Reason: "This Home responsibility needs bounded Work execution.", RequestKey: "link-outcome-missing",
	})
	assertAPIErrorCode(t, err, "OUTCOME_NOT_FOUND")
}

func TestResponsibilityLinkCreateIsIdempotentDuplicateSafeAndEndsWithoutLifecycleCoupling(t *testing.T) {
	now := time.Date(2026, 8, 26, 4, 30, 0, 0, time.UTC)
	store := &memoryResponsibilityLinkStore{
		projects:        map[string]domain.ProjectRecord{"project-1": {ID: "project-1"}},
		outcomeProjects: map[domain.OutcomeID]domain.ProjectID{"out-1": "project-1"},
	}
	service := NewResponsibilityLinks(store, func() time.Time { return now })
	input := CreateResponsibilityLinkInput{
		ProjectID: "project-1", SourceOpenLoopID: "loop-1", DestinationOutcomeID: "out-1", Reason: "Execute the bounded Work outcome while Home retains the open responsibility.", RequestKey: "link-1",
	}
	first, err := service.CreateResponsibilityLink(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateResponsibilityLink() error = %v", err)
	}
	replay, err := service.CreateResponsibilityLink(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateResponsibilityLink() replay error = %v", err)
	}
	if store.writes != 1 || replay.ID != first.ID {
		t.Fatalf("writes/replay = %d/%s, want one/%s", store.writes, replay.ID, first.ID)
	}

	input.RequestKey = "link-duplicate"
	_, err = service.CreateResponsibilityLink(context.Background(), input)
	assertAPIErrorCode(t, err, "RESPONSIBILITY_LINK_EXISTS")

	ended, err := service.EndResponsibilityLink(context.Background(), first.ID, "Owner no longer needs this lineage")
	if err != nil {
		t.Fatalf("EndResponsibilityLink() error = %v", err)
	}
	if ended.EndedAt == nil || ended.SourceOpenLoopID != "loop-1" || ended.DestinationOutcomeID != "out-1" {
		t.Fatalf("ended link changed lineage: %+v", ended)
	}
}

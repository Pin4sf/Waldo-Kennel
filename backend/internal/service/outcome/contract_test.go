package outcome_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	"github.com/aoagents/agent-orchestrator/backend/internal/service/outcome"
)

// fakeStore is an in-memory ports.OutcomeStore faithful to the durability
// contract under test: atomic append-with-CAS, replay resolution, and
// append-only history. SQLite-level behavior is proven in the store suite.
type fakeStore struct {
	mu       sync.Mutex
	spaces   map[domain.ProjectID]domain.ResponsibilitySpace
	outcomes map[domain.OutcomeID]domain.Outcome
	revs     map[domain.OutcomeID][]domain.ContractRevision
	keys     map[string]domain.OutcomeID
	plans    map[domain.OutcomeID]domain.PlanRevision
	links    map[domain.OutcomeID][]domain.ContributionLink

	decompositions     map[domain.DecompositionRevisionID]domain.DecompositionRevision
	decompositionOrder []domain.DecompositionRevisionID

	order  []domain.OutcomeID
	writes int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		spaces:   map[domain.ProjectID]domain.ResponsibilitySpace{},
		outcomes: map[domain.OutcomeID]domain.Outcome{},
		revs:     map[domain.OutcomeID][]domain.ContractRevision{},
		keys:     map[string]domain.OutcomeID{},
		plans:    map[domain.OutcomeID]domain.PlanRevision{},
	}
}

func (f *fakeStore) EnsureWorkResponsibilitySpace(_ context.Context, projectID domain.ProjectID) (domain.ResponsibilitySpace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if space, ok := f.spaces[projectID]; ok {
		return space, nil
	}
	space := domain.ResponsibilitySpace{ID: domain.ResponsibilitySpaceID("rsp-" + projectID), Kind: domain.ResponsibilitySpaceWorkProject, ProjectID: projectID}
	f.spaces[projectID] = space
	return space, nil
}

func (f *fakeStore) FindOutcomeByIdempotencyKey(_ context.Context, key string) (domain.Outcome, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.keys[key]
	if !ok {
		return domain.Outcome{}, false, nil
	}
	return f.outcomes[id], true, nil
}

var errUniqueKey = errors.New("UNIQUE constraint failed: outcomes.idempotency_key")

func (f *fakeStore) CreateOutcomeWithContract(_ context.Context, o domain.Outcome, first domain.ContractRevision, requestKey string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if requestKey != "" {
		if _, taken := f.keys[requestKey]; taken {
			return errUniqueKey
		}
	}
	first.Number = 1
	o.CurrentRevisionNumber = 1
	f.outcomes[o.ID] = o
	f.revs[o.ID] = []domain.ContractRevision{first}
	f.order = append(f.order, o.ID)
	if requestKey != "" {
		f.keys[requestKey] = o.ID
	}
	f.writes++
	return nil
}

func (f *fakeStore) GetOutcome(_ context.Context, id domain.OutcomeID) (domain.Outcome, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.outcomes[id]
	return o, ok, nil
}

func (f *fakeStore) ListOutcomesByProject(_ context.Context, projectID domain.ProjectID) ([]domain.Outcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	space, ok := f.spaces[projectID]
	if !ok {
		return []domain.Outcome{}, nil
	}
	out := make([]domain.Outcome, 0)
	for _, id := range f.order {
		if candidate := f.outcomes[id]; candidate.SpaceID == space.ID {
			out = append(out, candidate)
		}
	}
	return out, nil
}

func (f *fakeStore) AppendContractRevision(_ context.Context, id domain.OutcomeID, expectedCurrent int64, rev domain.ContractRevision) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.outcomes[id]
	if !ok {
		return 0, &ports.OutcomeConflictError{OutcomeID: id, ExpectedRevisionNum: expectedCurrent, CurrentRevisionNum: -1}
	}
	number := int64(len(f.revs[id])) + 1
	if expectedCurrent != o.CurrentRevisionNumber {
		return 0, &ports.OutcomeConflictError{OutcomeID: id, ExpectedRevisionNum: expectedCurrent, CurrentRevisionNum: o.CurrentRevisionNumber}
	}
	rev.Number = number
	f.revs[id] = append(f.revs[id], rev)
	o.CurrentRevisionNumber = number
	f.outcomes[id] = o
	f.writes++
	return number, nil
}

func (f *fakeStore) ListContractRevisions(_ context.Context, id domain.OutcomeID) ([]domain.ContractRevision, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.ContractRevision, len(f.revs[id]))
	copy(out, f.revs[id])
	return out, nil
}

func validCreateInput() outcome.CreateInput {
	return outcome.CreateInput{
		ProjectID:       "mer",
		Title:           "Local Focus Ledger",
		Goal:            "A user can record and review today's protected focus time locally.",
		SuccessCriteria: []string{"Entering positive whole minutes creates one focus block."},
		Review:          "Deterministic checks plus owner walkthrough.",
		RequestKey:      "req-create-1",
	}
}

func TestListByProjectReturnsCanonicalOutcomeViews(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	first := validCreateInput()
	if _, err := svc.Create(ctx, first); err != nil {
		t.Fatalf("create first: %v", err)
	}
	second := first
	second.Title = "Second Outcome"
	second.RequestKey = "req-create-2"
	if _, err := svc.Create(ctx, second); err != nil {
		t.Fatalf("create second: %v", err)
	}
	store.plans[store.order[1]] = domain.PlanRevision{
		ID: "plan-second", OutcomeID: store.order[1], Number: 1,
		ContractRevisionNumber: 1, Status: domain.PlanStatusApproved,
	}
	other := first
	other.ProjectID = "other"
	other.RequestKey = "req-create-other"
	if _, err := svc.Create(ctx, other); err != nil {
		t.Fatalf("create other: %v", err)
	}

	views, err := svc.ListByProject(ctx, "mer")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(views) != 2 || views[0].Outcome.Title != first.Title || views[1].Outcome.Title != second.Title {
		t.Fatalf("views = %+v", views)
	}
	for _, view := range views {
		if view.Current.Number != 1 || len(view.History) != 1 {
			t.Fatalf("listed view lost contract: %+v", view)
		}
	}
	if views[0].LatestPlan != nil || views[1].LatestPlan == nil || views[1].LatestPlan.Status != domain.PlanStatusApproved {
		t.Fatalf("latest durable plan facts = %+v, %+v", views[0].LatestPlan, views[1].LatestPlan)
	}
}

func TestListByProjectRejectsMissingProject(t *testing.T) {
	svc, _ := newService()
	_, err := svc.ListByProject(context.Background(), "  ")
	var apiErr *apierr.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "PROJECT_REQUIRED" {
		t.Fatalf("err = %v, want PROJECT_REQUIRED", err)
	}
}

func newService() (outcome.Manager, *fakeStore) {
	store := newFakeStore()
	return outcome.New(store, nil), store
}

func TestCreateBuildsFirstContractAndReplayReturnsSameOutcome(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()

	view, err := svc.Create(ctx, validCreateInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Outcome.ID == "" || view.Outcome.SpaceID == "" {
		t.Fatalf("view missing canonical identity: %+v", view.Outcome)
	}
	if view.Outcome.CurrentRevisionNumber != 1 {
		t.Fatalf("pointer = %d, want 1", view.Outcome.CurrentRevisionNumber)
	}
	if view.Current.Number != 1 || view.Current.Goal != validCreateInput().Goal {
		t.Fatalf("current revision = %+v", view.Current)
	}
	if len(view.History) != 1 {
		t.Fatalf("history = %d revisions, want 1", len(view.History))
	}

	replayed, err := svc.Create(ctx, validCreateInput())
	if err != nil {
		t.Fatalf("replayed create: %v", err)
	}

	if replayed.Outcome.ID != view.Outcome.ID {
		t.Fatalf("replay produced %s, want original %s", replayed.Outcome.ID, view.Outcome.ID)
	}
	if len(replayed.History) != 1 || store.writes != 1 {
		t.Fatalf("replay must not write: writes=%d history=%d", store.writes, len(replayed.History))
	}
}

func TestCreateRejectsIncompleteContracts(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		wantCode string
	}{
		{name: "project", field: "project", wantCode: "PROJECT_REQUIRED"},
		{name: "request key", field: "requestKey", wantCode: "REQUEST_KEY_REQUIRED"},
		{name: "title", field: "title", wantCode: "OUTCOME_TITLE_REQUIRED"},
		{name: "goal", field: "goal", wantCode: "OUTCOME_GOAL_REQUIRED"},
		{name: "criteria", field: "criteria", wantCode: "OUTCOME_CRITERIA_REQUIRED"},
		{name: "review", field: "review", wantCode: "OUTCOME_REVIEW_REQUIRED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store := newService()
			in := validCreateInput()
			switch tt.field {
			case "project":
				in.ProjectID = ""
			case "requestKey":
				in.RequestKey = ""
			case "title":
				in.Title = "  "
			case "goal":
				in.Goal = ""
			case "criteria":
				in.SuccessCriteria = nil
			case "review":
				in.Review = ""
			}
			_, err := svc.Create(context.Background(), in)
			var apiErr *apierr.Error
			if !errors.As(err, &apiErr) || apiErr.Code != tt.wantCode {
				t.Fatalf("err = %v, want apierr code %s", err, tt.wantCode)
			}
			if store.writes != 0 {
				t.Fatalf("invalid input persisted %d writes", store.writes)
			}
		})
	}
}

func TestReviseAppendsImmutableHistoryAndMovesPointer(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()
	view, err := svc.Create(ctx, validCreateInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	revised, err := svc.ReviseContract(ctx, view.Outcome.ID, outcome.ReviseContractInput{
		ExpectedRevision: 1,
		Goal:             "Revised goal.",
		SuccessCriteria:  []string{"Criterion A.", "Criterion B."},
		Review:           "Deterministic checks.",
		NonGoals:         []string{"No timers."},
		Clarification:    "today means the Mac's local calendar day",
	})
	if err != nil {
		t.Fatalf("revise: %v", err)
	}
	if revised.Outcome.CurrentRevisionNumber != 2 {
		t.Fatalf("pointer = %d, want 2", revised.Outcome.CurrentRevisionNumber)
	}
	if revised.Current.Number != 2 || revised.Current.Clarification == "" || len(revised.Current.NonGoals) != 1 {
		t.Fatalf("revision 2 = %+v", revised.Current)
	}
	if len(revised.History) != 2 || revised.History[0].Number != 1 || revised.History[0].Goal != view.Current.Goal {
		t.Fatal("revision 1 must remain untouched immutable history")
	}
}

func TestReviseStaleExpectationConflictsWithoutPersisting(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	view, err := svc.Create(ctx, validCreateInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err = svc.ReviseContract(ctx, view.Outcome.ID, outcome.ReviseContractInput{
		ExpectedRevision: 7,
		Goal:             "Should never land.",
		SuccessCriteria:  []string{"c"},
		Review:           "r",
	})
	var apiErr *apierr.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "OUTCOME_CONTRACT_CONFLICT" {
		t.Fatalf("err = %v, want OUTCOME_CONTRACT_CONFLICT", err)
	}
	history, _ := store.ListContractRevisions(ctx, view.Outcome.ID)
	if len(history) != 1 {
		t.Fatalf("rejected revise persisted history len=%d", len(history))
	}
}

func TestGetUnknownOutcomeIsNotFound(t *testing.T) {
	svc, _ := newService()
	_, err := svc.Get(context.Background(), domain.OutcomeID("out-missing"))
	var apiErr *apierr.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "OUTCOME_NOT_FOUND" {
		t.Fatalf("err = %v, want OUTCOME_NOT_FOUND", err)
	}
}

func TestTrimmedInputsPersistTrimmed(t *testing.T) {
	svc, _ := newService()
	in := validCreateInput()
	in.Title = "  Local Focus Ledger  "
	view, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if strings.TrimSpace(view.Outcome.Title) != view.Outcome.Title {
		t.Fatalf("title stored untrimmed: %q", view.Outcome.Title)
	}
}

// Plan methods are exercised by the plan service tests; the contract fake
// only needs to satisfy the widened OutcomeStore interface.
func (f *fakeStore) AppendPlanRevision(_ context.Context, _ domain.OutcomeID, plan domain.PlanRevision) (domain.PlanRevision, error) {
	return plan, nil
}

func (f *fakeStore) LatestProposedPlanRevision(_ context.Context, _ domain.OutcomeID, _ int64) (domain.PlanRevision, bool, error) {
	return domain.PlanRevision{}, false, nil
}

func (f *fakeStore) GetPlanRevision(_ context.Context, _ domain.OutcomeID, id domain.PlanRevisionID) (domain.PlanRevision, bool, error) {
	return domain.PlanRevision{}, false, nil
}

func (f *fakeStore) GetLatestPlanRevision(_ context.Context, outcomeID domain.OutcomeID) (domain.PlanRevision, bool, error) {
	plan, ok := f.plans[outcomeID]
	return plan, ok, nil
}

func (f *fakeStore) ApprovePlanRevision(_ context.Context, _ domain.OutcomeID, id domain.PlanRevisionID) (domain.PlanRevision, bool, error) {
	return domain.PlanRevision{ID: id, Status: domain.PlanStatusApproved}, true, nil
}

// --- Composed Outcomes (ADR 0007) ---
//
// The fake enforces the same invariants storage does, so a service test that
// passes here would also pass against SQLite. A fake that silently accepted a
// third level or an unlinked child would prove nothing.

func (f *fakeStore) CreateContributionWithContract(
	_ context.Context,
	child domain.Outcome,
	first domain.ContractRevision,
	links []domain.ContributionLink,
	requestKey string,
) error {
	if err := domain.ValidateContributionLinkSet(child.ID, links); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if requestKey != "" {
		if _, taken := f.keys[requestKey]; taken {
			return errUniqueKey
		}
	}
	parent, ok := f.outcomes[child.ParentID]
	if !ok {
		return fmt.Errorf("parent outcome %s not found", child.ParentID)
	}
	if parent.IsContributing() {
		return fmt.Errorf("composition depth limit: a contributing outcome cannot be a parent")
	}
	first.Number = 1
	child.CurrentRevisionNumber = 1
	f.outcomes[child.ID] = child
	f.revs[child.ID] = []domain.ContractRevision{first}
	f.order = append(f.order, child.ID)
	if f.links == nil {
		f.links = map[domain.OutcomeID][]domain.ContributionLink{}
	}
	f.links[child.ID] = append(f.links[child.ID], links...)
	if requestKey != "" {
		f.keys[requestKey] = child.ID
	}
	f.writes++
	return nil
}

func (f *fakeStore) ListContributingOutcomes(_ context.Context, parent domain.OutcomeID) ([]domain.Outcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.Outcome, 0)
	for _, id := range f.order {
		if o, ok := f.outcomes[id]; ok && o.ParentID == parent && !parent.IsZero() {
			out = append(out, o)
		}
	}
	return out, nil
}

func (f *fakeStore) ListContributionLinksForParent(_ context.Context, parent domain.OutcomeID) ([]domain.ContributionLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.ContributionLink, 0)
	for _, id := range f.order {
		for _, link := range f.links[id] {
			if link.ParentOutcomeID == parent {
				out = append(out, link)
			}
		}
	}
	return out, nil
}

func (f *fakeStore) ListContributionLinksForChild(_ context.Context, child domain.OutcomeID) ([]domain.ContributionLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.ContributionLink, 0, len(f.links[child]))
	return append(out, f.links[child]...), nil
}

// --- Decomposition authority (ADR 0007 phase 2) ---
//
// The fake keeps proposals and authorization separate exactly as storage does,
// so a test cannot pass by treating a proposal as if it had already created
// the contributing Outcomes.

func (f *fakeStore) AppendDecompositionRevision(_ context.Context, revision domain.DecompositionRevision) (domain.DecompositionRevision, error) {
	if err := revision.Validate(); err != nil {
		return domain.DecompositionRevision{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.decompositions == nil {
		f.decompositions = map[domain.DecompositionRevisionID]domain.DecompositionRevision{}
	}
	highest := int64(0)
	for _, existing := range f.decompositions {
		if existing.OutcomeID == revision.OutcomeID && existing.Number > highest {
			highest = existing.Number
		}
	}
	revision.Number = highest + 1
	f.decompositions[revision.ID] = revision
	f.decompositionOrder = append(f.decompositionOrder, revision.ID)
	return revision, nil
}

func (f *fakeStore) AuthorizeDecompositionRevision(
	_ context.Context,
	outcomeID domain.OutcomeID,
	decompositionID domain.DecompositionRevisionID,
	contributions []ports.AuthorizedContribution,
	at time.Time,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	revision, ok := f.decompositions[decompositionID]
	if !ok || revision.OutcomeID != outcomeID || revision.Status != domain.DecompositionProposed {
		return fmt.Errorf("%w: %s", ports.ErrDecompositionNotProposed, decompositionID)
	}
	if f.links == nil {
		f.links = map[domain.OutcomeID][]domain.ContributionLink{}
	}
	byRef := make(map[string]domain.OutcomeID, len(contributions))
	for _, contribution := range contributions {
		child := contribution.Outcome
		child.CurrentRevisionNumber = 1
		first := contribution.First
		first.Number = 1
		f.outcomes[child.ID] = child
		f.revs[child.ID] = []domain.ContractRevision{first}
		f.order = append(f.order, child.ID)
		f.links[child.ID] = append(f.links[child.ID], contribution.Links...)
		byRef[contribution.Ref] = child.ID
		f.writes++
	}
	for i := range revision.Contributors {
		if id, ok := byRef[revision.Contributors[i].Ref]; ok {
			revision.Contributors[i].ChildOutcomeID = id
		}
	}
	revision.Status = domain.DecompositionAuthorized
	revision.AuthorizedAt = &at
	f.decompositions[decompositionID] = revision
	return nil
}

func (f *fakeStore) GetDecompositionRevision(_ context.Context, outcomeID domain.OutcomeID, id domain.DecompositionRevisionID) (domain.DecompositionRevision, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	revision, ok := f.decompositions[id]
	if !ok || revision.OutcomeID != outcomeID {
		return domain.DecompositionRevision{}, false, nil
	}
	return revision, true, nil
}

func (f *fakeStore) LatestDecompositionRevision(_ context.Context, outcomeID domain.OutcomeID) (domain.DecompositionRevision, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var latest domain.DecompositionRevision
	found := false
	for _, id := range f.decompositionOrder {
		revision := f.decompositions[id]
		if revision.OutcomeID == outcomeID && (!found || revision.Number > latest.Number) {
			latest, found = revision, true
		}
	}
	return latest, found, nil
}

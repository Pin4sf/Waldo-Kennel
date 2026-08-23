package outcome_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

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
	writes   int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		spaces:   map[domain.ProjectID]domain.ResponsibilitySpace{},
		outcomes: map[domain.OutcomeID]domain.Outcome{},
		revs:     map[domain.OutcomeID][]domain.ContractRevision{},
		keys:     map[string]domain.OutcomeID{},
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

func (f *fakeStore) GetLatestPlanRevision(_ context.Context, _ domain.OutcomeID) (domain.PlanRevision, bool, error) {
	return domain.PlanRevision{}, false, nil
}

func (f *fakeStore) ApprovePlanRevision(_ context.Context, _ domain.OutcomeID, id domain.PlanRevisionID) (domain.PlanRevision, bool, error) {
	return domain.PlanRevision{ID: id, Status: domain.PlanStatusApproved}, true, nil
}

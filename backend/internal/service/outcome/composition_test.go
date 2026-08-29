package outcome_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/service/outcome"
)

// seedDecomposableParent creates a parent whose contract carries authority the
// contribution tests can narrow from, and returns its stable criterion ids.
func seedDecomposableParent(t *testing.T, svc outcome.Manager, store *fakeStore) (domain.OutcomeID, []domain.CriterionID) {
	t.Helper()
	in := validCreateInput()
	in.SuccessCriteria = []string{"Selectable for every mission role.", "A switched-away session resolves truthfully."}
	view, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	// Give the parent a real ceiling; Create does not set one.
	store.mu.Lock()
	revs := store.revs[view.Outcome.ID]
	revs[0].AuthorityCeiling = domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true, ExecuteLocal: true}
	store.revs[view.Outcome.ID] = revs
	store.mu.Unlock()

	ids := make([]domain.CriterionID, 0, len(revs[0].Criteria))
	for _, criterion := range revs[0].Criteria {
		ids = append(ids, criterion.ID)
	}
	if len(ids) != 2 {
		t.Fatalf("parent needs two stable criteria, got %d", len(ids))
	}
	return view.Outcome.ID, ids
}

func contributionInput(key string, claimed ...domain.CriterionID) outcome.CreateContributionInput {
	return outcome.CreateContributionInput{
		RequestKey:      key,
		Title:           "Admission gates admit OpenCode",
		Goal:            "Every admission predicate admits opencode.",
		SuccessCriteria: []string{"All three predicates return true."},
		Review:          "Deterministic tests.",
		ClaimedCriteria: claimed,
		Authority:       domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true},
	}
}

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var apiErr *apierr.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("error %v is not an API error", err)
	}
	return apiErr.Code
}

func TestCreateContributionBindsCriteriaAndDerivesShape(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	child, err := svc.CreateContribution(ctx, parentID, contributionInput("req-c1", criteria[0]))
	if err != nil {
		t.Fatalf("create contribution: %v", err)
	}
	if child.Outcome.ParentID != parentID {
		t.Fatalf("child parent = %q, want %q", child.Outcome.ParentID, parentID)
	}
	if child.Outcome.CurrentRevisionNumber != 1 {
		t.Fatalf("child must carry its own contract revision 1, got %d", child.Outcome.CurrentRevisionNumber)
	}

	view, err := svc.Composition(ctx, parentID)
	if err != nil {
		t.Fatalf("composition: %v", err)
	}
	if view.Shape != domain.OutcomeShapeDecomposed {
		t.Fatalf("shape = %q, want decomposed", view.Shape)
	}
	if len(view.Contributors) != 1 || view.Contributors[0].Outcome.ID != child.Outcome.ID {
		t.Fatalf("contributors = %+v", view.Contributors)
	}
	if view.Contributors[0].Stale {
		t.Fatal("a contribution bound to the current revision must not be stale")
	}
	// The second criterion is nobody's job yet, and the view must say so
	// rather than implying the decomposition is complete.
	unclaimed := view.Unclaimed()
	if len(unclaimed) != 1 || unclaimed[0].CriterionID != criteria[1] {
		t.Fatalf("unclaimed = %+v, want the second criterion", unclaimed)
	}
}

// The cap is what makes cycles unreachable, so the service must refuse a third
// level before storage ever sees it.
func TestCreateContributionRefusesThirdLevel(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	child, err := svc.CreateContribution(ctx, parentID, contributionInput("req-c1", criteria[0]))
	if err != nil {
		t.Fatalf("create contribution: %v", err)
	}
	childCriteria := child.Current.Criteria
	if len(childCriteria) == 0 {
		t.Fatal("child contract must carry a stable criterion")
	}

	_, err = svc.CreateContribution(ctx, child.Outcome.ID, contributionInput("req-c2", childCriteria[0].ID))
	if err == nil {
		t.Fatal("a contributing outcome must not be decomposable")
	}
	if got := codeOf(t, err); got != "COMPOSITION_DEPTH_LIMIT" {
		t.Fatalf("code = %q, want COMPOSITION_DEPTH_LIMIT", got)
	}
}

func TestCreateContributionRefusesWidenedAuthority(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	in := contributionInput("req-c1", criteria[0])
	in.Authority = domain.ProposedAuthority{ReadWorkspace: true, Deploy: true, ExternalEffect: true}
	_, err := svc.CreateContribution(ctx, parentID, in)
	if err == nil {
		t.Fatal("a child claiming authority its parent lacks must be refused")
	}
	if got := codeOf(t, err); got != "AUTHORITY_WIDENED" {
		t.Fatalf("code = %q, want AUTHORITY_WIDENED", got)
	}
	// The refusal has to name what was over-claimed, or it is unexplainable.
	for _, want := range []string{"deploy", "externalEffect"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal %q must name %q", err.Error(), want)
		}
	}
}

func TestCreateContributionRefusesUnknownAndUnboundClaims(t *testing.T) {
	tests := []struct {
		name     string
		claimed  []domain.CriterionID
		wantCode string
	}{
		{name: "no claim at all", claimed: nil, wantCode: "CONTRIBUTION_UNBOUND"},
		{name: "criterion not in contract", claimed: []domain.CriterionID{"crit-ghost"}, wantCode: "CRITERION_NOT_IN_CURRENT_CONTRACT"},
		{name: "blank criterion", claimed: []domain.CriterionID{"  "}, wantCode: "CRITERION_REQUIRED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store := newService()
			ctx := context.Background()
			parentID, _ := seedDecomposableParent(t, svc, store)

			_, err := svc.CreateContribution(ctx, parentID, contributionInput("req-c1", tt.claimed...))
			if err == nil {
				t.Fatal("claim must be refused")
			}
			if got := codeOf(t, err); got != tt.wantCode {
				t.Fatalf("code = %q, want %q", got, tt.wantCode)
			}
			if store.writes != 1 {
				t.Fatalf("store writes = %d, want only the parent's; a refusal must persist nothing", store.writes)
			}
		})
	}
}

func TestCreateContributionRejectsDuplicateClaim(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	_, err := svc.CreateContribution(ctx, parentID, contributionInput("req-c1", criteria[0], criteria[0]))
	if err == nil {
		t.Fatal("claiming one criterion twice must be refused")
	}
	if got := codeOf(t, err); got != "CRITERION_CLAIMED_TWICE" {
		t.Fatalf("code = %q, want CRITERION_CLAIMED_TWICE", got)
	}
}

// A delivered create never writes twice, and composition inherits that.
func TestCreateContributionReplayIsIdempotent(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	first, err := svc.CreateContribution(ctx, parentID, contributionInput("req-same", criteria[0]))
	if err != nil {
		t.Fatalf("first contribution: %v", err)
	}
	writesAfterFirst := store.writes

	second, err := svc.CreateContribution(ctx, parentID, contributionInput("req-same", criteria[0]))
	if err != nil {
		t.Fatalf("replayed contribution: %v", err)
	}
	if second.Outcome.ID != first.Outcome.ID {
		t.Fatalf("replay produced %q, want the original %q", second.Outcome.ID, first.Outcome.ID)
	}
	if store.writes != writesAfterFirst {
		t.Fatalf("replay wrote again: %d then %d", writesAfterFirst, store.writes)
	}
}

func TestCreateContributionRefusesUnknownParent(t *testing.T) {
	svc, _ := newService()
	_, err := svc.CreateContribution(context.Background(), "out-ghost", contributionInput("req-c1", "crit-1"))
	if err == nil {
		t.Fatal("contributing to a missing outcome must be refused")
	}
	if got := codeOf(t, err); got != "OUTCOME_NOT_FOUND" {
		t.Fatalf("code = %q, want OUTCOME_NOT_FOUND", got)
	}
}

// Every Outcome that predates composition is the direct case. Its composition
// read is a fact ("direct, no contributors"), never a 404.
func TestCompositionOfDirectOutcomeIsDirect(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()
	created, err := svc.Create(ctx, validCreateInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	view, err := svc.Composition(ctx, created.Outcome.ID)
	if err != nil {
		t.Fatalf("composition of a direct outcome must succeed: %v", err)
	}
	if view.Shape != domain.OutcomeShapeDirect {
		t.Fatalf("shape = %q, want direct", view.Shape)
	}
	if len(view.Contributors) != 0 || len(view.Coverage) != 0 || view.Parent != nil {
		t.Fatalf("a direct outcome composes nothing: %+v", view)
	}
}

// A contributing Outcome reads its parent back, which is what lets Mission
// Control navigate up without a second lookup.
func TestCompositionOfContributorNamesItsParent(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)
	child, err := svc.CreateContribution(ctx, parentID, contributionInput("req-c1", criteria[0]))
	if err != nil {
		t.Fatalf("create contribution: %v", err)
	}

	view, err := svc.Composition(ctx, child.Outcome.ID)
	if err != nil {
		t.Fatalf("composition: %v", err)
	}
	if view.Parent == nil || view.Parent.ID != parentID {
		t.Fatalf("contributor must name its parent, got %+v", view.Parent)
	}
	if view.Shape != domain.OutcomeShapeDirect {
		t.Fatalf("a contributor with no children of its own is direct, got %q", view.Shape)
	}
}

// The parent revising its contract must drop coverage rather than carry the
// old claims forward as if the new criteria were already someone's job.
func TestCompositionAfterParentRevisionReportsStaleAndUncovered(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)
	if _, err := svc.CreateContribution(ctx, parentID, contributionInput("req-c1", criteria[0])); err != nil {
		t.Fatalf("create contribution: %v", err)
	}

	if _, err := svc.ReviseContract(ctx, parentID, outcome.ReviseContractInput{
		ExpectedRevision: 1,
		Goal:             "OpenCode is selectable, resumable, and leaves Codex untouched.",
		SuccessCriteria:  []string{"Selectable for every mission role."},
		Review:           "Separate-session review.",
	}); err != nil {
		t.Fatalf("revise parent: %v", err)
	}

	view, err := svc.Composition(ctx, parentID)
	if err != nil {
		t.Fatalf("composition: %v", err)
	}
	if len(view.Contributors) != 1 || !view.Contributors[0].Stale {
		t.Fatalf("the contributor must report stale after the parent revises: %+v", view.Contributors)
	}
	if len(view.Unclaimed()) != 1 {
		t.Fatalf("the new revision's criterion must read unclaimed, got %+v", view.Unclaimed())
	}
}

package outcome_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

func contributionOffer(ref string, claimed ...domain.CriterionID) outcome.ProposedContributionInput {
	return outcome.ProposedContributionInput{
		Ref: ref, Title: "Contribution " + ref, Goal: "Make " + ref + " true.",
		SuccessCriteria: []string{"It is demonstrably true."},
		Review:          "Deterministic checks.",
		Authority:       domain.ProposedAuthority{ReadWorkspace: true},
		ClaimedCriteria: claimed,
	}
}

// Omitting contributors asks the daemon for its mechanical starting point:
// identical contracts must yield identical decompositions, with no model in
// the loop and nothing pretending to be a recommendation.
func TestProposeDecompositionDerivesADeterministicDefault(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	first, err := svc.ProposeDecomposition(ctx, parentID, outcome.ProposeDecompositionInput{ExpectedContractRevision: 1})
	if err != nil {
		t.Fatalf("propose decomposition: %v", err)
	}
	if first.Decomposition.Status != domain.DecompositionProposed {
		t.Fatalf("status = %q, want proposed", first.Decomposition.Status)
	}
	if len(first.Decomposition.Contributors) != len(criteria) {
		t.Fatalf("derived %d contributors, want one per criterion (%d)", len(first.Decomposition.Contributors), len(criteria))
	}
	if strings.TrimSpace(first.Decomposition.Rationale) == "" {
		t.Fatal("the daemon's own proposal must still explain itself")
	}

	// A proposal creates no responsibility. Nothing exists until authorization.
	composition, err := svc.Composition(ctx, parentID)
	if err != nil {
		t.Fatalf("composition: %v", err)
	}
	if composition.Shape != domain.OutcomeShapeDirect || len(composition.Contributors) != 0 {
		t.Fatalf("a proposal must create nothing, got %+v", composition)
	}

	second, err := svc.ProposeDecomposition(ctx, parentID, outcome.ProposeDecompositionInput{ExpectedContractRevision: 1})
	if err != nil {
		t.Fatalf("re-propose: %v", err)
	}
	if len(second.Decomposition.Contributors) != len(first.Decomposition.Contributors) {
		t.Fatal("the deterministic default must not vary between calls")
	}
	for i := range first.Decomposition.Contributors {
		a, b := first.Decomposition.Contributors[i], second.Decomposition.Contributors[i]
		if a.Ref != b.Ref || a.Title != b.Title || len(a.ClaimedCriteria) != len(b.ClaimedCriteria) {
			t.Fatalf("contributor %d differs between identical proposals: %+v vs %+v", i, a, b)
		}
	}
	if second.Decomposition.Number != first.Decomposition.Number+1 {
		t.Fatalf("proposals are append-only history: numbers %d then %d", first.Decomposition.Number, second.Decomposition.Number)
	}
}

// The gate that makes a decomposition trustworthy. An unclassified criterion
// is how a project reports itself done while missing something material.
func TestProposeDecompositionRefusesUnclassifiedCriteria(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	_, err := svc.ProposeDecomposition(ctx, parentID, outcome.ProposeDecompositionInput{
		ExpectedContractRevision: 1,
		Rationale:                "Only the first slice.",
		Contributors:             []outcome.ProposedContributionInput{contributionOffer("c1", criteria[0])},
	})
	if err == nil {
		t.Fatal("leaving a criterion neither claimed nor retained must be refused")
	}
	if got := codeOf(t, err); got != "DECOMPOSITION_INCOMPLETE" {
		t.Fatalf("code = %q, want DECOMPOSITION_INCOMPLETE", got)
	}
}

// Retention is the owner's explicit alternative to delegating.
func TestProposeDecompositionAcceptsRetainedCriteria(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	view, err := svc.ProposeDecomposition(ctx, parentID, outcome.ProposeDecompositionInput{
		ExpectedContractRevision: 1,
		Rationale:                "One slice delegated; I will prove the regression check myself.",
		Contributors:             []outcome.ProposedContributionInput{contributionOffer("c1", criteria[0])},
		RetainedCriteria:         []domain.CriterionID{criteria[1]},
	})
	if err != nil {
		t.Fatalf("retention must satisfy coverage: %v", err)
	}
	if len(view.Decomposition.RetainedCriteria) != 1 || view.Decomposition.RetainedCriteria[0] != criteria[1] {
		t.Fatalf("retained = %v, want the second criterion", view.Decomposition.RetainedCriteria)
	}
}

func TestProposeDecompositionRefusesUnknownCriteriaAndWidenedAuthority(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*outcome.ProposeDecompositionInput, []domain.CriterionID)
		wantCode string
	}{
		{
			name: "criterion outside the current contract",
			mutate: func(in *outcome.ProposeDecompositionInput, ids []domain.CriterionID) {
				in.Contributors = []outcome.ProposedContributionInput{contributionOffer("c1", ids[0], ids[1], "crit-ghost")}
			},
			wantCode: "CRITERION_NOT_IN_CURRENT_CONTRACT",
		},
		{
			name: "contributor widens the parent ceiling",
			mutate: func(in *outcome.ProposeDecompositionInput, ids []domain.CriterionID) {
				greedy := contributionOffer("c1", ids[0], ids[1])
				greedy.Authority = domain.ProposedAuthority{ReadWorkspace: true, Deploy: true}
				in.Contributors = []outcome.ProposedContributionInput{greedy}
			},
			wantCode: "AUTHORITY_WIDENED",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store := newService()
			ctx := context.Background()
			parentID, criteria := seedDecomposableParent(t, svc, store)

			in := outcome.ProposeDecompositionInput{ExpectedContractRevision: 1, Rationale: "One slice."}
			tt.mutate(&in, criteria)
			_, err := svc.ProposeDecomposition(ctx, parentID, in)
			if err == nil {
				t.Fatal("proposal must be refused")
			}
			if got := codeOf(t, err); got != tt.wantCode {
				t.Fatalf("code = %q, want %q", got, tt.wantCode)
			}
		})
	}
}

func TestProposeDecompositionRefusesDependencyCycle(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	_, err := svc.ProposeDecomposition(ctx, parentID, outcome.ProposeDecompositionInput{
		ExpectedContractRevision: 1,
		Rationale:                "Two slices that wait on each other.",
		Contributors: []outcome.ProposedContributionInput{
			contributionOffer("c1", criteria[0]),
			contributionOffer("c2", criteria[1]),
		},
		Dependencies: []outcome.ContributionDependencyInput{
			{FromRef: "c1", ToRef: "c2"},
			{FromRef: "c2", ToRef: "c1"},
		},
	})
	if err == nil {
		t.Fatal("a dependency cycle must be refused")
	}
	if got := codeOf(t, err); got != "DECOMPOSITION_INVALID" && got != "DEPENDENCY_CYCLE" {
		t.Fatalf("code = %q, want a cycle refusal", got)
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("refusal %q must say it is a cycle", err.Error())
	}
}

func TestProposeDecompositionRefusesStaleContractAndContributingParent(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	if _, err := svc.ProposeDecomposition(ctx, parentID, outcome.ProposeDecompositionInput{
		ExpectedContractRevision: 7,
	}); err == nil {
		t.Fatal("naming a revision that is not current must be refused")
	} else if got := codeOf(t, err); got != "DECOMPOSITION_STALE" {
		t.Fatalf("code = %q, want DECOMPOSITION_STALE", got)
	}

	// A contributing Outcome cannot itself be decomposed.
	child, err := svc.CreateContribution(ctx, parentID, contributionInput("req-c1", criteria[0]))
	if err != nil {
		t.Fatalf("create contribution: %v", err)
	}
	if _, err := svc.ProposeDecomposition(ctx, child.Outcome.ID, outcome.ProposeDecompositionInput{
		ExpectedContractRevision: 1,
	}); err == nil {
		t.Fatal("decomposing a contributing outcome must be refused")
	} else if got := codeOf(t, err); got != "COMPOSITION_DEPTH_LIMIT" {
		t.Fatalf("code = %q, want COMPOSITION_DEPTH_LIMIT", got)
	}
}

func TestAuthorizeDecompositionCreatesTheContributingOutcomes(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	proposed, err := svc.ProposeDecomposition(ctx, parentID, outcome.ProposeDecompositionInput{
		ExpectedContractRevision: 1,
		Rationale:                "Two independent slices.",
		Contributors: []outcome.ProposedContributionInput{
			contributionOffer("c1", criteria[0]),
			contributionOffer("c2", criteria[1]),
		},
	})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}

	authorized, err := svc.AuthorizeDecomposition(ctx, parentID, proposed.Decomposition.ID)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if authorized.Decomposition.Status != domain.DecompositionAuthorized {
		t.Fatalf("status = %q, want authorized", authorized.Decomposition.Status)
	}
	if authorized.Decomposition.AuthorizedAt == nil {
		t.Fatal("authorization must record when the owner decided")
	}
	for _, contributor := range authorized.Decomposition.Contributors {
		if contributor.ChildOutcomeID.IsZero() {
			t.Fatalf("contributor %q must resolve to a real Outcome", contributor.Ref)
		}
	}

	composition, err := svc.Composition(ctx, parentID)
	if err != nil {
		t.Fatalf("composition: %v", err)
	}
	if composition.Shape != domain.OutcomeShapeDecomposed || len(composition.Contributors) != 2 {
		t.Fatalf("composition = %+v, want two contributors", composition)
	}
	if unclaimed := composition.Unclaimed(); len(unclaimed) != 0 {
		t.Fatalf("an authorized full decomposition leaves nothing unclaimed, got %+v", unclaimed)
	}
	// Each contributor owns its own contract, not a share of the parent's.
	for _, contributor := range composition.Contributors {
		view, err := svc.Get(ctx, contributor.Outcome.ID)
		if err != nil {
			t.Fatalf("read contributor: %v", err)
		}
		if view.Current.Number != 1 || strings.TrimSpace(view.Current.Goal) == "" {
			t.Fatalf("contributor %s must own contract revision 1, got %+v", contributor.Outcome.ID, view.Current)
		}
	}
}

// Re-authorizing is a replay, not an error: the contributing Outcomes already
// exist and a second attempt must not create them twice.
func TestAuthorizeDecompositionIsIdempotent(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	proposed, err := svc.ProposeDecomposition(ctx, parentID, outcome.ProposeDecompositionInput{
		ExpectedContractRevision: 1, Rationale: "One slice, one retention.",
		Contributors:     []outcome.ProposedContributionInput{contributionOffer("c1", criteria[0])},
		RetainedCriteria: []domain.CriterionID{criteria[1]},
	})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if _, err := svc.AuthorizeDecomposition(ctx, parentID, proposed.Decomposition.ID); err != nil {
		t.Fatalf("authorize: %v", err)
	}
	writesAfterFirst := store.writes

	if _, err := svc.AuthorizeDecomposition(ctx, parentID, proposed.Decomposition.ID); err != nil {
		t.Fatalf("re-authorize must replay, not fail: %v", err)
	}
	if store.writes != writesAfterFirst {
		t.Fatalf("replay created outcomes again: %d then %d", writesAfterFirst, store.writes)
	}
	composition, err := svc.Composition(ctx, parentID)
	if err != nil {
		t.Fatalf("composition: %v", err)
	}
	if len(composition.Contributors) != 1 {
		t.Fatalf("replay must not duplicate contributors, got %d", len(composition.Contributors))
	}
}

// A proposal answers a contract. If the contract moves, the proposal answers a
// question the owner is no longer asking.
func TestAuthorizeDecompositionRefusesAStaleProposal(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	proposed, err := svc.ProposeDecomposition(ctx, parentID, outcome.ProposeDecompositionInput{
		ExpectedContractRevision: 1, Rationale: "Two slices.",
		Contributors: []outcome.ProposedContributionInput{
			contributionOffer("c1", criteria[0]),
			contributionOffer("c2", criteria[1]),
		},
	})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if _, err := svc.ReviseContract(ctx, parentID, outcome.ReviseContractInput{
		ExpectedRevision: 1,
		Goal:             "A different goal entirely.",
		SuccessCriteria:  []string{"Something else is true."},
		Review:           "Deterministic checks.",
	}); err != nil {
		t.Fatalf("revise parent: %v", err)
	}

	_, err = svc.AuthorizeDecomposition(ctx, parentID, proposed.Decomposition.ID)
	if err == nil {
		t.Fatal("authorizing against a superseded contract must be refused")
	}
	if got := codeOf(t, err); got != "DECOMPOSITION_STALE" {
		t.Fatalf("code = %q, want DECOMPOSITION_STALE", got)
	}
	composition, err := svc.Composition(ctx, parentID)
	if err != nil {
		t.Fatalf("composition: %v", err)
	}
	if len(composition.Contributors) != 0 {
		t.Fatalf("a refused authorization must create nothing, got %+v", composition.Contributors)
	}

	latest, err := svc.LatestDecomposition(ctx, parentID)
	if err != nil {
		t.Fatalf("latest decomposition: %v", err)
	}
	if !latest.Stale {
		t.Fatal("the proposal must report stale after the parent revises")
	}
}

func TestAuthorizeDecompositionRefusesUnknownIdentifiers(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, _ := seedDecomposableParent(t, svc, store)

	if _, err := svc.AuthorizeDecomposition(ctx, parentID, "dec-ghost"); err == nil {
		t.Fatal("authorizing a decomposition that does not exist must be refused")
	} else if got := codeOf(t, err); got != "DECOMPOSITION_NOT_FOUND" {
		t.Fatalf("code = %q, want DECOMPOSITION_NOT_FOUND", got)
	}
	if _, err := svc.AuthorizeDecomposition(ctx, "out-ghost", "dec-1"); err == nil {
		t.Fatal("authorizing under an outcome that does not exist must be refused")
	} else if got := codeOf(t, err); got != "OUTCOME_NOT_FOUND" {
		t.Fatalf("code = %q, want OUTCOME_NOT_FOUND", got)
	}
}

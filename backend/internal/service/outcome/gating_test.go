package outcome_test

import (
	"context"
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/service/outcome"
)

// seedAuthorizedDecomposition builds a parent with two ordered contributors:
// c1 must finish before c2.
func seedAuthorizedDecomposition(t *testing.T, svc outcome.Manager, store *fakeStore) (domain.OutcomeID, domain.DecompositionRevision) {
	t.Helper()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	proposed, err := svc.ProposeDecomposition(ctx, parentID, outcome.ProposeDecompositionInput{
		ExpectedContractRevision: 1,
		Rationale:                "The second slice builds on the first.",
		Contributors: []outcome.ProposedContributionInput{
			contributionOffer("c1", criteria[0]),
			contributionOffer("c2", criteria[1]),
		},
		Dependencies: []outcome.ContributionDependencyInput{{FromRef: "c1", ToRef: "c2"}},
	})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	authorized, err := svc.AuthorizeDecomposition(ctx, parentID, proposed.Decomposition.ID)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	return parentID, authorized.Decomposition
}

func contributorOutcomeID(t *testing.T, decomposition domain.DecompositionRevision, ref string) domain.OutcomeID {
	t.Helper()
	for _, contributor := range decomposition.Contributors {
		if contributor.Ref == ref {
			return contributor.ChildOutcomeID
		}
	}
	t.Fatalf("decomposition has no contribution %q", ref)
	return ""
}

// The gate reaches the owner through the composition read model, so a blocked
// contributor is visible before anyone tries to start it.
func TestCompositionReportsBlockedAndWaitingContributors(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, decomposition := seedAuthorizedDecomposition(t, svc, store)

	view, err := svc.Composition(ctx, parentID)
	if err != nil {
		t.Fatalf("composition: %v", err)
	}
	if len(view.Contributors) != 2 {
		t.Fatalf("want two contributors, got %d", len(view.Contributors))
	}

	upstream := contributorOutcomeID(t, decomposition, "c1")
	downstream := contributorOutcomeID(t, decomposition, "c2")
	for _, contributor := range view.Contributors {
		switch contributor.Outcome.ID {
		case upstream:
			if !contributor.Gate.Clear() {
				t.Fatalf("the upstream must not be blocked: %+v", contributor.Gate.Blocked)
			}
		case downstream:
			if contributor.Gate.Clear() {
				t.Fatal("the downstream must be blocked while the upstream is unaccepted")
			}
			if contributor.Attention.Kind != domain.AttentionWaiting {
				t.Fatalf("a blocked contributor is Waiting, got %q", contributor.Attention.Kind)
			}
			// The owner must be able to act without opening a transcript.
			if !strings.Contains(contributor.Attention.Reason, "waiting on") ||
				strings.TrimSpace(contributor.Attention.NextAction) == "" {
				t.Fatalf("attention must explain itself and offer a next action: %+v", contributor.Attention)
			}
		}
	}

	// Every roll-up item names its contributor, so the parent never becomes an
	// undifferentiated activity feed.
	if view.Attention.Contributors != 2 || len(view.Attention.Items) != 2 {
		t.Fatalf("attention summary = %+v", view.Attention)
	}
	for _, item := range view.Attention.Items {
		if item.OutcomeID.IsZero() || strings.TrimSpace(item.Reason) == "" {
			t.Fatalf("roll-up item must name a contributor and explain itself: %+v", item)
		}
	}
}

func TestWaiveContributionDependencyClearsTheGateAndStaysVisible(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, decomposition := seedAuthorizedDecomposition(t, svc, store)
	downstream := contributorOutcomeID(t, decomposition, "c2")

	if _, err := svc.WaiveContributionDependency(ctx, parentID, outcome.WaiveDependencyInput{
		FromRef: "c1", ToRef: "c2", Reason: "The interface c2 needs is already frozen.",
	}); err != nil {
		t.Fatalf("waive: %v", err)
	}

	view, err := svc.Composition(ctx, parentID)
	if err != nil {
		t.Fatalf("composition: %v", err)
	}
	for _, contributor := range view.Contributors {
		if contributor.Outcome.ID != downstream {
			continue
		}
		if !contributor.Gate.Clear() {
			t.Fatalf("a waived dependency must not block: %+v", contributor.Gate.Blocked)
		}
		// Overridden, not forgotten.
		if len(contributor.Gate.Waived) != 1 || contributor.Gate.Waived[0].Ref != "c1" {
			t.Fatalf("the waived dependency must stay visible: %+v", contributor.Gate.Waived)
		}
	}
}

func TestWaiveContributionDependencyRefusesUndeclaredAndUnexplained(t *testing.T) {
	tests := []struct {
		name     string
		in       outcome.WaiveDependencyInput
		wantCode string
	}{
		{
			name:     "no reason",
			in:       outcome.WaiveDependencyInput{FromRef: "c1", ToRef: "c2", Reason: "   "},
			wantCode: "WAIVER_REASON_REQUIRED",
		},
		{
			name:     "ordering nobody declared",
			in:       outcome.WaiveDependencyInput{FromRef: "c2", ToRef: "c1", Reason: "Reversed."},
			wantCode: "DEPENDENCY_NOT_DECLARED",
		},
		{
			name:     "self dependency",
			in:       outcome.WaiveDependencyInput{FromRef: "c1", ToRef: "c1", Reason: "Nonsense."},
			wantCode: "WAIVER_REFS_REQUIRED",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store := newService()
			ctx := context.Background()
			parentID, _ := seedAuthorizedDecomposition(t, svc, store)

			_, err := svc.WaiveContributionDependency(ctx, parentID, tt.in)
			if err == nil {
				t.Fatal("waiver must be refused")
			}
			if got := codeOf(t, err); got != tt.wantCode {
				t.Fatalf("code = %q, want %q", got, tt.wantCode)
			}
		})
	}
}

// Waiving before the owner has authorized anything would record consent to a
// plan that does not exist.
func TestWaiveContributionDependencyRequiresAnAuthorizedDecomposition(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, criteria := seedDecomposableParent(t, svc, store)

	if _, err := svc.ProposeDecomposition(ctx, parentID, outcome.ProposeDecompositionInput{
		ExpectedContractRevision: 1, Rationale: "Two slices.",
		Contributors: []outcome.ProposedContributionInput{
			contributionOffer("c1", criteria[0]),
			contributionOffer("c2", criteria[1]),
		},
		Dependencies: []outcome.ContributionDependencyInput{{FromRef: "c1", ToRef: "c2"}},
	}); err != nil {
		t.Fatalf("propose: %v", err)
	}

	_, err := svc.WaiveContributionDependency(ctx, parentID, outcome.WaiveDependencyInput{
		FromRef: "c1", ToRef: "c2", Reason: "Too early.",
	})
	if err == nil {
		t.Fatal("waiving against an unauthorized decomposition must be refused")
	}
	if got := codeOf(t, err); got != "DECOMPOSITION_NOT_AUTHORIZED" {
		t.Fatalf("code = %q, want DECOMPOSITION_NOT_AUTHORIZED", got)
	}
}

// A contributor with no declared upstreams is never gated, and neither is a
// Project-level Outcome.
func TestUngatedOutcomesStayUngated(t *testing.T) {
	svc, store := newService()
	ctx := context.Background()
	parentID, decomposition := seedAuthorizedDecomposition(t, svc, store)

	view, err := svc.Composition(ctx, parentID)
	if err != nil {
		t.Fatalf("composition: %v", err)
	}
	// The parent itself composes; it is not a contributor and has no gate.
	if view.Shape != domain.OutcomeShapeDecomposed {
		t.Fatalf("shape = %q, want decomposed", view.Shape)
	}
	upstream := contributorOutcomeID(t, decomposition, "c1")
	upstreamView, err := svc.Composition(ctx, upstream)
	if err != nil {
		t.Fatalf("composition of contributor: %v", err)
	}
	if upstreamView.Shape != domain.OutcomeShapeDirect || upstreamView.Parent == nil {
		t.Fatalf("a contributor is direct and names its parent: %+v", upstreamView)
	}
}

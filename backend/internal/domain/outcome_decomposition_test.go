package domain_test

import (
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

func proposedContribution(ref string, position int64, claimed ...domain.CriterionID) domain.ProposedContribution {
	return domain.ProposedContribution{
		Ref: ref, Position: position,
		Title: "Contribution " + ref, Goal: "Make " + ref + " true.",
		SuccessCriteria: []string{"It is demonstrably true."},
		Review:          "Deterministic checks.",
		Authority:       domain.ProposedAuthority{ReadWorkspace: true},
		ClaimedCriteria: claimed,
	}
}

func decomposition(contributors []domain.ProposedContribution, retained []domain.CriterionID, deps ...domain.ContributionDependency) domain.DecompositionRevision {
	return domain.DecompositionRevision{
		ID: "dec-1", OutcomeID: "out-parent", Number: 1, ContractRevisionID: "cr-1",
		Status: domain.DecompositionProposed, Rationale: "Three independent slices.",
		Contributors: contributors, RetainedCriteria: retained, Dependencies: deps,
	}
}

func TestDecompositionRevisionValidatesTheHappyPath(t *testing.T) {
	d := decomposition(
		[]domain.ProposedContribution{
			proposedContribution("c1", 1, "crit-a"),
			proposedContribution("c2", 2, "crit-b"),
		},
		[]domain.CriterionID{"crit-c"},
		domain.ContributionDependency{ID: "dep-1", FromRef: "c1", ToRef: "c2"},
	)
	if err := d.Validate(); err != nil {
		t.Fatalf("a well-formed decomposition must validate: %v", err)
	}
}

// A decomposition the owner cannot evaluate is not reviewable, so the
// rationale is required rather than decorative.
func TestDecompositionRequiresRationaleAndContributors(t *testing.T) {
	d := decomposition([]domain.ProposedContribution{proposedContribution("c1", 1, "crit-a")}, nil)
	d.Rationale = "   "
	if err := d.Validate(); err == nil || !strings.Contains(err.Error(), "rationale") {
		t.Fatalf("a decomposition without a rationale must be refused, got %v", err)
	}

	empty := decomposition(nil, nil)
	if err := empty.Validate(); err == nil || !strings.Contains(err.Error(), "at least one contributing") {
		t.Fatalf("a decomposition with no contributors must be refused, got %v", err)
	}
}

// Retaining a criterion a contributor also claims says the owner proves it and
// delegates it at once.
func TestDecompositionRejectsRetainedAndClaimedCriterion(t *testing.T) {
	d := decomposition(
		[]domain.ProposedContribution{proposedContribution("c1", 1, "crit-a")},
		[]domain.CriterionID{"crit-a"},
	)
	err := d.Validate()
	if err == nil || !strings.Contains(err.Error(), "both retained") {
		t.Fatalf("a criterion cannot be retained and claimed, got %v", err)
	}
}

func TestDecompositionRejectsDuplicateRefsAndUnknownDependencies(t *testing.T) {
	dupes := decomposition([]domain.ProposedContribution{
		proposedContribution("c1", 1, "crit-a"),
		proposedContribution("c1", 2, "crit-b"),
	}, nil)
	if err := dupes.Validate(); err == nil || !strings.Contains(err.Error(), "twice") {
		t.Fatalf("duplicate refs must be refused, got %v", err)
	}

	dangling := decomposition([]domain.ProposedContribution{proposedContribution("c1", 1, "crit-a")}, nil,
		domain.ContributionDependency{ID: "dep-1", FromRef: "c1", ToRef: "ghost"})
	if err := dangling.Validate(); err == nil || !strings.Contains(err.Error(), "unknown contribution") {
		t.Fatalf("a dependency on a contribution that is not proposed must be refused, got %v", err)
	}
}

// The depth cap makes parent/child cycles unreachable, but sibling
// dependencies are a real directed graph. A cycle leaves every contributor in
// it permanently ungatable.
func TestDependencyCycleIsRefusedAndNamed(t *testing.T) {
	contributors := []domain.ProposedContribution{
		proposedContribution("c1", 1, "crit-a"),
		proposedContribution("c2", 2, "crit-b"),
		proposedContribution("c3", 3, "crit-c"),
	}
	deps := []domain.ContributionDependency{
		{ID: "dep-1", FromRef: "c1", ToRef: "c2"},
		{ID: "dep-2", FromRef: "c2", ToRef: "c3"},
		{ID: "dep-3", FromRef: "c3", ToRef: "c1"},
	}
	err := domain.ValidateDependencyOrdering(contributors, deps)
	if err == nil {
		t.Fatal("a dependency cycle must be refused")
	}
	// The owner has to know which ordering to cut, so the message names the
	// cycle rather than merely reporting one exists.
	for _, want := range []string{"c1", "c2", "c3", "->"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("cycle report %q must name %q", err.Error(), want)
		}
	}
}

func TestSelfDependencyIsRefused(t *testing.T) {
	d := domain.ContributionDependency{ID: "dep-1", FromRef: "c1", ToRef: "c1"}
	if err := d.Validate(); err == nil || !strings.Contains(err.Error(), "cannot depend on itself") {
		t.Fatalf("a self-dependency must be refused, got %v", err)
	}
}

// A diamond is a legitimate topology and must not be mistaken for a cycle.
func TestDiamondDependenciesAreAdmitted(t *testing.T) {
	contributors := []domain.ProposedContribution{
		proposedContribution("c1", 1, "crit-a"),
		proposedContribution("c2", 2, "crit-b"),
		proposedContribution("c3", 3, "crit-c"),
		proposedContribution("c4", 4, "crit-d"),
	}
	deps := []domain.ContributionDependency{
		{ID: "d1", FromRef: "c1", ToRef: "c2"},
		{ID: "d2", FromRef: "c1", ToRef: "c3"},
		{ID: "d3", FromRef: "c2", ToRef: "c4"},
		{ID: "d4", FromRef: "c3", ToRef: "c4"},
	}
	if err := domain.ValidateDependencyOrdering(contributors, deps); err != nil {
		t.Fatalf("a diamond is not a cycle: %v", err)
	}
}

func revisionWithCriteria(ids ...string) domain.ContractRevision {
	criteria := make([]domain.ContractCriterion, 0, len(ids))
	for i, id := range ids {
		criteria = append(criteria, domain.ContractCriterion{
			ID: domain.CriterionID(id), ContractRevisionID: "cr-1",
			Position: int64(i + 1), Text: "Criterion " + id,
		})
	}
	return domain.ContractRevision{ID: "cr-1", OutcomeID: "out-parent", Number: 1, Criteria: criteria}
}

// Coverage is the gate that makes a decomposition trustworthy: every criterion
// is delegated or retained, and there is no third state.
func TestUncoveredCriteriaFindsTheUnclassifiedOnes(t *testing.T) {
	current := revisionWithCriteria("crit-a", "crit-b", "crit-c", "crit-d")
	contributors := []domain.ProposedContribution{
		proposedContribution("c1", 1, "crit-a"),
		proposedContribution("c2", 2, "crit-b"),
	}
	uncovered := domain.UncoveredCriteria(current, contributors, []domain.CriterionID{"crit-c"})
	if len(uncovered) != 1 || uncovered[0].ID != "crit-d" {
		t.Fatalf("uncovered = %+v, want only crit-d", uncovered)
	}

	full := domain.UncoveredCriteria(current, contributors, []domain.CriterionID{"crit-c", "crit-d"})
	if len(full) != 0 {
		t.Fatalf("a fully classified decomposition covers everything, got %+v", full)
	}
}

func TestUnknownClaimedCriteriaRejectsGhostIdentities(t *testing.T) {
	current := revisionWithCriteria("crit-a")
	contributors := []domain.ProposedContribution{proposedContribution("c1", 1, "crit-a", "crit-ghost")}
	unknown := domain.UnknownClaimedCriteria(current, contributors, []domain.CriterionID{"crit-retired"})
	if len(unknown) != 2 {
		t.Fatalf("unknown = %v, want both the ghost claim and the retired retention", unknown)
	}
	if domain.UnknownClaimedCriteria(current, []domain.ProposedContribution{proposedContribution("c1", 1, "crit-a")}, nil) == nil {
		t.Fatal("a clean decomposition must return an empty slice, not nil")
	}
}

func TestOverClaimedAuthorityNamesEveryOffendingContribution(t *testing.T) {
	parent := domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true}
	fine := proposedContribution("c1", 1, "crit-a")
	greedy := proposedContribution("c2", 2, "crit-b")
	greedy.Authority = domain.ProposedAuthority{ReadWorkspace: true, Deploy: true}

	offenders := domain.OverClaimedAuthority(parent, []domain.ProposedContribution{fine, greedy})
	if len(offenders) != 1 {
		t.Fatalf("offenders = %v, want only c2", offenders)
	}
	if got := offenders["c2"]; len(got) != 1 || got[0] != "deploy" {
		t.Fatalf("c2 over-claims = %v, want [deploy]", got)
	}
}

// Two contributors may both prove one criterion: that is redundancy, not a
// contradiction, and the owner may well want it for a critical result.
func TestTwoContributorsMayShareOneCriterion(t *testing.T) {
	d := decomposition([]domain.ProposedContribution{
		proposedContribution("c1", 1, "crit-a"),
		proposedContribution("c2", 2, "crit-a"),
	}, nil)
	if err := d.Validate(); err != nil {
		t.Fatalf("shared criterion coverage must be admitted: %v", err)
	}
}

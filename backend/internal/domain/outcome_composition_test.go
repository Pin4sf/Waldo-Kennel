package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

func criterion(revision domain.ContractRevisionID, id string, position int64, text string) domain.ContractCriterion {
	return domain.ContractCriterion{
		ID:                 domain.CriterionID(id),
		ContractRevisionID: revision,
		Position:           position,
		Text:               text,
	}
}

func parentRevision(id domain.ContractRevisionID, criteria ...domain.ContractCriterion) domain.ContractRevision {
	return domain.ContractRevision{ID: id, OutcomeID: "out-parent", Number: 1, Criteria: criteria}
}

func link(child domain.OutcomeID, revision domain.ContractRevisionID, criterionID string) domain.ContributionLink {
	return domain.ContributionLink{
		ID:                       domain.ContributionLinkID("cl-" + string(child) + "-" + criterionID),
		ParentOutcomeID:          "out-parent",
		ChildOutcomeID:           child,
		ParentContractRevisionID: revision,
		ParentCriterionID:        domain.CriterionID(criterionID),
		CreatedAt:                time.Unix(0, 0).UTC(),
	}
}

// The cap is the mechanism that makes cycles impossible: a graph that cannot
// reach depth 3 cannot contain one. If this constant ever rises, cycle
// detection stops being free and must be implemented explicitly.
func TestCompositionDepthLimitIsTwo(t *testing.T) {
	if domain.CompositionDepthLimit != 2 {
		t.Fatalf("CompositionDepthLimit = %d; raising it makes cycles reachable and needs explicit detection",
			domain.CompositionDepthLimit)
	}
}

func TestOutcomeShapeIsDerivedFromChildren(t *testing.T) {
	if got := domain.ShapeForChildCount(0); got != domain.OutcomeShapeDirect {
		t.Fatalf("no children = %q, want direct", got)
	}
	if got := domain.ShapeForChildCount(3); got != domain.OutcomeShapeDecomposed {
		t.Fatalf("three children = %q, want decomposed", got)
	}
}

func TestOutcomeRejectsSelfContribution(t *testing.T) {
	o := domain.Outcome{ID: "out-1", SpaceID: "rs-1", ParentID: "out-1", Title: "loop"}
	if err := o.Validate(); err == nil {
		t.Fatal("an outcome contributing to itself must not validate")
	}
}

func TestOutcomeWithoutParentIsNotContributing(t *testing.T) {
	o := domain.Outcome{ID: "out-1", SpaceID: "rs-1", Title: "root"}
	if err := o.Validate(); err != nil {
		t.Fatalf("a root outcome must still validate: %v", err)
	}
	if o.IsContributing() {
		t.Fatal("an outcome with no parent must not report as contributing")
	}
}

func TestValidateContributionLinkSetRequiresAClaim(t *testing.T) {
	err := domain.ValidateContributionLinkSet("out-child", nil)
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("an unlinked contribution must be refused, got %v", err)
	}
}

// A child bound across two parent revisions would be current and superseded at
// once, leaving staleness undecidable.
func TestValidateContributionLinkSetRejectsSplitRevisions(t *testing.T) {
	err := domain.ValidateContributionLinkSet("out-child", []domain.ContributionLink{
		link("out-child", "cr-1", "crit-a"),
		link("out-child", "cr-2", "crit-b"),
	})
	if err == nil || !strings.Contains(err.Error(), "two parent contract revisions") {
		t.Fatalf("split-revision binding must be refused, got %v", err)
	}
}

func TestValidateContributionLinkSetRejectsDuplicateCriterion(t *testing.T) {
	err := domain.ValidateContributionLinkSet("out-child", []domain.ContributionLink{
		link("out-child", "cr-1", "crit-a"),
		link("out-child", "cr-1", "crit-a"),
	})
	if err == nil || !strings.Contains(err.Error(), "claimed twice") {
		t.Fatalf("duplicate criterion claim must be refused, got %v", err)
	}
}

func TestValidateContributionLinkSetRejectsForeignChild(t *testing.T) {
	err := domain.ValidateContributionLinkSet("out-child", []domain.ContributionLink{
		link("out-other", "cr-1", "crit-a"),
	})
	if err == nil || !strings.Contains(err.Error(), "belongs to outcome") {
		t.Fatalf("a link for another outcome must be refused, got %v", err)
	}
}

// Effective authority is the intersection of every layer: composition adds a
// layer that may narrow and may never widen.
func TestAuthorityWideningNamesEveryOffender(t *testing.T) {
	parent := domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true}
	child := domain.ProposedAuthority{
		ReadWorkspace: true, WriteWorkspace: true,
		ExecuteLocal: true, CreatePR: true, Deploy: true,
	}
	widened := domain.AuthorityWidenings(parent, child)
	if len(widened) != 3 {
		t.Fatalf("widenings = %v, want executeLocal, createPR, deploy", widened)
	}
	joined := strings.Join(widened, ",")
	for _, want := range []string{"executeLocal", "createPR", "deploy"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("widenings %v must name %q so a refusal is explainable", widened, want)
		}
	}
	if err := domain.AuthorityContains(parent, child); err == nil {
		t.Fatal("a widened child ceiling must fail closed")
	}
}

func TestAuthorityContainsAdmitsNarrowerChild(t *testing.T) {
	parent := domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true, ExecuteLocal: true}
	child := domain.ProposedAuthority{ReadWorkspace: true}
	if err := domain.AuthorityContains(parent, child); err != nil {
		t.Fatalf("a child claiming less than its parent must be admitted: %v", err)
	}
	if got := domain.AuthorityWidenings(parent, parent); len(got) != 0 {
		t.Fatalf("an identical ceiling widens nothing, got %v", got)
	}
}

func TestCriterionCoverageReportsClaimsInPositionOrder(t *testing.T) {
	current := parentRevision("cr-1",
		criterion("cr-1", "crit-b", 2, "Second"),
		criterion("cr-1", "crit-a", 1, "First"),
		criterion("cr-1", "crit-c", 3, "Third"),
	)
	links := []domain.ContributionLink{
		link("out-y", "cr-1", "crit-a"),
		link("out-x", "cr-1", "crit-a"),
		link("out-x", "cr-1", "crit-b"),
	}
	coverage := domain.CriterionCoverage(current, links)
	if len(coverage) != 3 {
		t.Fatalf("coverage has %d rows, want one per criterion", len(coverage))
	}
	if coverage[0].CriterionID != "crit-a" || coverage[2].CriterionID != "crit-c" {
		t.Fatalf("coverage must follow criterion position, got %+v", coverage)
	}
	// Claimants are ordered so the projection is stable across reads.
	if len(coverage[0].ClaimedBy) != 2 || coverage[0].ClaimedBy[0] != "out-x" {
		t.Fatalf("crit-a claimants = %v, want a stable [out-x out-y]", coverage[0].ClaimedBy)
	}
	if coverage[2].Claimed() {
		t.Fatal("crit-c is claimed by nobody and must report unclaimed")
	}

	unclaimed := domain.UnclaimedCriteria(current, links)
	if len(unclaimed) != 1 || unclaimed[0].CriterionID != "crit-c" {
		t.Fatalf("unclaimed = %+v, want only crit-c", unclaimed)
	}
}

// A contract revision must drop coverage back to unclaimed. Carrying claims
// forward would let a parent report itself covered against criteria that no
// contributor ever agreed to.
func TestCriterionCoverageIgnoresSupersededLinks(t *testing.T) {
	current := parentRevision("cr-2", criterion("cr-2", "crit-a", 1, "First"))
	links := []domain.ContributionLink{link("out-x", "cr-1", "crit-a")}

	coverage := domain.CriterionCoverage(current, links)
	if len(coverage) != 1 || coverage[0].Claimed() {
		t.Fatalf("a link bound to a superseded revision must not cover the new one: %+v", coverage)
	}
	if !domain.ContributionStale(current, links) {
		t.Fatal("a child bound to the prior revision must report stale")
	}
	if domain.ContributionStale(current, []domain.ContributionLink{link("out-x", "cr-2", "crit-a")}) {
		t.Fatal("a child bound to the current revision must not report stale")
	}
}

func TestContributionStaleOnEmptyLinksIsFalse(t *testing.T) {
	// No links means nothing is bound to a superseded revision. Whether an
	// unlinked child may exist at all is enforced at creation, not here.
	if domain.ContributionStale(parentRevision("cr-1"), nil) {
		t.Fatal("no bindings cannot be stale bindings")
	}
}

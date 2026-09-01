package store_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

// seedParent creates a Project-level Outcome whose contract carries two stable
// criteria, which is the minimum a decomposition can bind against.
func seedParent(t *testing.T, s *sqlite.Store, project domain.ProjectID, suffix string) (domain.Outcome, domain.ContractRevision) {
	t.Helper()
	ctx := context.Background()
	seedProject(t, s, string(project))
	space, err := s.EnsureWorkResponsibilitySpace(ctx, project)
	if err != nil {
		t.Fatalf("ensure space: %v", err)
	}
	at := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	parent := domain.Outcome{
		ID: domain.OutcomeID("out-parent-" + suffix), SpaceID: space.ID,
		Title: "OpenCode is a first-class harness", CreatedAt: at, UpdatedAt: at,
	}
	revisionID := domain.ContractRevisionID("cr-parent-" + suffix)
	first := domain.ContractRevision{
		ID: revisionID, OutcomeID: parent.ID,
		Goal:            "OpenCode is selectable, resumable, and usable without a provider login.",
		SuccessCriteria: []string{"Selectable for every mission role.", "A switched-away session resolves truthfully."},
		Review:          "Separate-session review.",
		Criteria: []domain.ContractCriterion{
			{ID: "crit-" + domain.CriterionID(suffix) + "-1", ContractRevisionID: revisionID, Position: 1, Text: "Selectable for every mission role."},
			{ID: "crit-" + domain.CriterionID(suffix) + "-2", ContractRevisionID: revisionID, Position: 2, Text: "A switched-away session resolves truthfully."},
		},
		CreatedAt: at,
	}
	if err := s.CreateOutcomeWithContract(ctx, parent, first, "req-parent-"+suffix); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	return parent, first
}

func contributionOf(parent domain.Outcome, revision domain.ContractRevision, suffix string, criteria ...domain.CriterionID) (domain.Outcome, domain.ContractRevision, []domain.ContributionLink) {
	at := time.Date(2026, 8, 29, 11, 0, 0, 0, time.UTC)
	child := domain.Outcome{
		ID: domain.OutcomeID("out-child-" + suffix), SpaceID: parent.SpaceID, ParentID: parent.ID,
		Title: "Admission gates admit OpenCode", CreatedAt: at, UpdatedAt: at,
	}
	childRevID := domain.ContractRevisionID("cr-child-" + suffix)
	first := domain.ContractRevision{
		ID: childRevID, OutcomeID: child.ID,
		Goal:            "Every admission predicate admits opencode.",
		SuccessCriteria: []string{"All three predicates return true."},
		Review:          "Deterministic tests.",
		Criteria:        []domain.ContractCriterion{{ID: domain.CriterionID("crit-child-" + suffix), ContractRevisionID: childRevID, Position: 1, Text: "All three predicates return true."}},
		CreatedAt:       at,
	}
	links := make([]domain.ContributionLink, 0, len(criteria))
	for i, criterionID := range criteria {
		links = append(links, domain.ContributionLink{
			ID:                       domain.ContributionLinkID("cl-" + suffix + "-" + string(rune('a'+i))),
			ParentOutcomeID:          parent.ID,
			ChildOutcomeID:           child.ID,
			ParentContractRevisionID: revision.ID,
			ParentCriterionID:        criterionID,
			CreatedAt:                at,
		})
	}
	return child, first, links
}

func TestComposition_CreateContributionIsAtomicAndReadable(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "1")

	child, first, links := contributionOf(parent, revision, "1", revision.Criteria[0].ID)
	if err := s.CreateContributionWithContract(ctx, child, first, links, "req-child-1"); err != nil {
		t.Fatalf("create contribution: %v", err)
	}

	got, ok, err := s.GetOutcome(ctx, child.ID)
	if err != nil || !ok {
		t.Fatalf("get child ok=%v err=%v", ok, err)
	}
	if got.ParentID != parent.ID {
		t.Fatalf("child parent = %q, want %q", got.ParentID, parent.ID)
	}
	if got.CurrentRevisionNumber != 1 {
		t.Fatalf("child revision pointer = %d, want 1", got.CurrentRevisionNumber)
	}

	children, err := s.ListContributingOutcomes(ctx, parent.ID)
	if err != nil {
		t.Fatalf("list contributors: %v", err)
	}
	if len(children) != 1 || children[0].ID != child.ID {
		t.Fatalf("contributors = %+v, want just the child", children)
	}

	held, err := s.ListContributionLinksForChild(ctx, child.ID)
	if err != nil {
		t.Fatalf("list child links: %v", err)
	}
	if len(held) != 1 || held[0].ParentCriterionID != revision.Criteria[0].ID {
		t.Fatalf("child links = %+v, want the claimed criterion", held)
	}
	if held[0].ParentContractRevisionID != revision.ID {
		t.Fatalf("link revision = %q, want the parent's current %q", held[0].ParentContractRevisionID, revision.ID)
	}

	// The parent stays a root, and reads as decomposed only through its
	// children — nothing about its own row changed.
	parentRow, _, err := s.GetOutcome(ctx, parent.ID)
	if err != nil {
		t.Fatalf("get parent: %v", err)
	}
	if parentRow.IsContributing() {
		t.Fatal("the parent must remain a root outcome")
	}
}

// Every Outcome that existed before composition is the direct case, and must
// keep reading exactly as it did.
func TestComposition_DirectOutcomeIsUnchanged(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedProject(t, s, "mer")
	space, err := s.EnsureWorkResponsibilitySpace(ctx, "mer")
	if err != nil {
		t.Fatalf("ensure space: %v", err)
	}
	outcome, first := focusLedgerContract(space.ID, "solo")
	if err := s.CreateOutcomeWithContract(ctx, outcome, first, "req-solo"); err != nil {
		t.Fatalf("create outcome: %v", err)
	}

	got, ok, err := s.GetOutcome(ctx, outcome.ID)
	if err != nil || !ok {
		t.Fatalf("get outcome ok=%v err=%v", ok, err)
	}
	if got.IsContributing() {
		t.Fatalf("a direct outcome must have no parent, got %q", got.ParentID)
	}
	children, err := s.ListContributingOutcomes(ctx, outcome.ID)
	if err != nil {
		t.Fatalf("list contributors: %v", err)
	}
	if len(children) != 0 {
		t.Fatalf("a direct outcome has no contributors, got %+v", children)
	}
	if shape := domain.ShapeForChildCount(len(children)); shape != domain.OutcomeShapeDirect {
		t.Fatalf("shape = %q, want direct", shape)
	}
}

// The depth cap is enforced in SQLite as well as the domain: it is what makes
// cycles unreachable, so it must hold even against a caller that bypasses the
// service.
func TestComposition_DepthCapIsEnforcedByStorage(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "2")

	child, first, links := contributionOf(parent, revision, "2", revision.Criteria[0].ID)
	if err := s.CreateContributionWithContract(ctx, child, first, links, "req-child-2"); err != nil {
		t.Fatalf("create contribution: %v", err)
	}

	childAsParent, _, _ := s.GetOutcome(ctx, child.ID)
	grandchild, grandFirst, grandLinks := contributionOf(childAsParent, first, "3", first.Criteria[0].ID)
	err := s.CreateContributionWithContract(ctx, grandchild, grandFirst, grandLinks, "req-grandchild")
	if err == nil {
		t.Fatal("a third composition level must be refused by storage")
	}
	if !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("refusal must name the depth limit, got %v", err)
	}
	if _, ok, _ := s.GetOutcome(ctx, grandchild.ID); ok {
		t.Fatal("the refused grandchild must leave no row behind")
	}
}

// The immutability triggers exist to defend against writers that are not the
// store, so the test has to be one: it opens the database file directly.
func TestComposition_LinksAreAppendOnly(t *testing.T) {
	dir := t.TempDir()
	s := sqlitetest.MustOpenAt(t, dir)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "4")
	child, first, links := contributionOf(parent, revision, "4", revision.Criteria[0].ID)
	if err := s.CreateContributionWithContract(ctx, child, first, links, "req-child-4"); err != nil {
		t.Fatalf("create contribution: %v", err)
	}

	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "kennel.db"))
	if err != nil {
		t.Fatalf("open database directly: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.ExecContext(ctx, `UPDATE contribution_links SET parent_criterion_id = 'x'`); err == nil {
		t.Fatal("contribution links must reject UPDATE")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("refusal must name the append-only rule, got %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM contribution_links`); err == nil {
		t.Fatal("contribution links must reject DELETE")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("refusal must name the append-only rule, got %v", err)
	}

	// A child that already has bindings cannot be re-bound to a second parent
	// revision, which is what keeps staleness decidable.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO contribution_links (id, parent_outcome_id, child_outcome_id, parent_contract_revision_id, parent_criterion_id)
		 VALUES ('cl-split', ?, ?, 'cr-other', ?)`,
		string(parent.ID), string(child.ID), string(revision.Criteria[1].ID)); err == nil {
		t.Fatal("binding a child to a second parent revision must be refused")
	}
}

// A binding that names a parent the child does not have would let an unrelated
// Outcome claim criteria it has no relationship to.
func TestComposition_StorageRejectsMismatchedBinding(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "5")
	other, otherRevision := seedParent(t, s, "other", "6")

	child, first, links := contributionOf(parent, revision, "5", revision.Criteria[0].ID)
	// Repoint the binding at an unrelated parent and its revision.
	links[0].ParentOutcomeID = other.ID
	links[0].ParentContractRevisionID = otherRevision.ID
	links[0].ParentCriterionID = otherRevision.Criteria[0].ID

	if err := s.CreateContributionWithContract(ctx, child, first, links, "req-child-5"); err == nil {
		t.Fatal("a link naming a parent the child does not have must be refused")
	}
	if _, ok, _ := s.GetOutcome(ctx, child.ID); ok {
		t.Fatal("the refused contribution must leave no outcome row behind")
	}
}

func TestComposition_StorageRejectsUnknownCriterion(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "7")
	child, first, links := contributionOf(parent, revision, "7", "crit-does-not-exist")

	if err := s.CreateContributionWithContract(ctx, child, first, links, "req-child-7"); err == nil {
		t.Fatal("binding to a criterion that does not exist must be refused")
	}
	if _, ok, _ := s.GetOutcome(ctx, child.ID); ok {
		t.Fatal("the refused contribution must leave no outcome row behind")
	}
}

func TestComposition_BindingEmitsChangeEvent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "8")
	child, first, links := contributionOf(parent, revision, "8", revision.Criteria[0].ID, revision.Criteria[1].ID)
	if err := s.CreateContributionWithContract(ctx, child, first, links, "req-child-8"); err != nil {
		t.Fatalf("create contribution: %v", err)
	}

	events, err := s.EventsAfter(ctx, 0, 200)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	bound := 0
	for _, event := range events {
		if string(event.Type) != "outcome_contribution_bound" {
			continue
		}
		bound++
		if event.ProjectID != "mer" {
			t.Fatalf("binding event project = %s, want mer", event.ProjectID)
		}
		payload := string(event.Payload)
		// The child's own outcome_created event says nothing about what it
		// contributes to; this event has to carry the whole relationship.
		for _, want := range []string{string(parent.ID), string(child.ID), string(revision.ID)} {
			if !strings.Contains(payload, want) {
				t.Fatalf("binding payload %s must name %q", payload, want)
			}
		}
	}
	if bound != 2 {
		t.Fatalf("emitted %d binding events, want one per claimed criterion", bound)
	}
}

// A contract revision must not carry old claims forward. After the parent
// revises, its contributors read stale and its criteria read unclaimed.
func TestComposition_ParentRevisionStalesContributors(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "9")
	child, first, links := contributionOf(parent, revision, "9", revision.Criteria[0].ID)
	if err := s.CreateContributionWithContract(ctx, child, first, links, "req-child-9"); err != nil {
		t.Fatalf("create contribution: %v", err)
	}

	next := domain.ContractRevision{
		ID: "cr-parent-9-v2", OutcomeID: parent.ID,
		Goal:            "OpenCode is selectable, resumable, usable, and does not disturb Codex.",
		SuccessCriteria: []string{"Selectable for every mission role."},
		Review:          "Separate-session review.",
		Criteria:        []domain.ContractCriterion{{ID: "crit-9-v2-1", ContractRevisionID: "cr-parent-9-v2", Position: 1, Text: "Selectable for every mission role."}},
		CreatedAt:       time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC),
	}
	if _, err := s.AppendContractRevision(ctx, parent.ID, 1, next); err != nil {
		t.Fatalf("revise parent contract: %v", err)
	}

	held, err := s.ListContributionLinksForParent(ctx, parent.ID)
	if err != nil {
		t.Fatalf("list parent links: %v", err)
	}
	// Superseded links stay readable so the stale contribution can be
	// explained rather than silently vanishing.
	if len(held) != 1 {
		t.Fatalf("superseded links must remain readable, got %+v", held)
	}
	if !domain.ContributionStale(next, held) {
		t.Fatal("a contributor bound to the prior revision must report stale")
	}
	if unclaimed := domain.UnclaimedCriteria(next, held); len(unclaimed) != 1 {
		t.Fatalf("the new revision's criterion must read unclaimed, got %+v", unclaimed)
	}
}

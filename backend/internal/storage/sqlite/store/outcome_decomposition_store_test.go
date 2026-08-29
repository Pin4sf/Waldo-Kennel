package store_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/sqlitetest"
)

var decomposedAt = time.Date(2026, 8, 29, 13, 0, 0, 0, time.UTC)

func proposalFor(parent domain.Outcome, revision domain.ContractRevision, id string) domain.DecompositionRevision {
	return domain.DecompositionRevision{
		ID: domain.DecompositionRevisionID(id), OutcomeID: parent.ID, Number: 1,
		ContractRevisionID: revision.ID, Status: domain.DecompositionProposed,
		Rationale: "Two independent slices.",
		Contributors: []domain.ProposedContribution{
			{
				Ref: "c1", Position: 1, Title: "Admission gates admit OpenCode",
				Goal:            "Every admission predicate admits opencode.",
				SuccessCriteria: []string{"All three predicates return true."},
				Review:          "Deterministic tests.",
				Authority:       domain.ProposedAuthority{ReadWorkspace: true},
				ClaimedCriteria: []domain.CriterionID{revision.Criteria[0].ID},
			},
			{
				Ref: "c2", Position: 2, Title: "Continuation reports availability",
				Goal:            "A switched-away session resolves truthfully.",
				SuccessCriteria: []string{"The probe never reports a live session as deleted."},
				Review:          "Deterministic tests.",
				Authority:       domain.ProposedAuthority{ReadWorkspace: true},
				ClaimedCriteria: []domain.CriterionID{revision.Criteria[1].ID},
			},
		},
		Dependencies: []domain.ContributionDependency{{ID: "cdep-" + id, FromRef: "c1", ToRef: "c2"}},
		CreatedAt:    decomposedAt,
	}
}

func authorizedFrom(parent domain.Outcome, revision domain.ContractRevision, proposal domain.DecompositionRevision) []ports.AuthorizedContribution {
	out := make([]ports.AuthorizedContribution, 0, len(proposal.Contributors))
	for _, contributor := range proposal.Contributors {
		childID := domain.OutcomeID("out-auth-" + contributor.Ref + "-" + string(proposal.ID))
		revID := domain.ContractRevisionID("cr-auth-" + contributor.Ref + "-" + string(proposal.ID))
		links := make([]domain.ContributionLink, 0, len(contributor.ClaimedCriteria))
		for j, criterion := range contributor.ClaimedCriteria {
			links = append(links, domain.ContributionLink{
				ID:                       domain.ContributionLinkID("cl-auth-" + contributor.Ref + string(rune('a'+j)) + "-" + string(proposal.ID)),
				ParentOutcomeID:          parent.ID,
				ChildOutcomeID:           childID,
				ParentContractRevisionID: revision.ID,
				ParentCriterionID:        criterion,
				CreatedAt:                decomposedAt,
			})
		}
		out = append(out, ports.AuthorizedContribution{
			Ref: contributor.Ref,
			Outcome: domain.Outcome{
				ID: childID, SpaceID: parent.SpaceID, ParentID: parent.ID,
				Title: contributor.Title, CreatedAt: decomposedAt, UpdatedAt: decomposedAt,
			},
			First: domain.ContractRevision{
				ID: revID, OutcomeID: childID, Goal: contributor.Goal,
				SuccessCriteria: contributor.SuccessCriteria, Review: contributor.Review,
				Criteria: []domain.ContractCriterion{{
					ID:                 domain.CriterionID("crit-auth-" + contributor.Ref + "-" + string(proposal.ID)),
					ContractRevisionID: revID, Position: 1, Text: contributor.SuccessCriteria[0],
				}},
				CreatedAt: decomposedAt,
			},
			Links: links,
		})
	}
	return out
}

func TestDecomposition_ProposalCreatesNoResponsibility(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "d1")

	stored, err := s.AppendDecompositionRevision(ctx, proposalFor(parent, revision, "dec-d1"))
	if err != nil {
		t.Fatalf("append decomposition: %v", err)
	}
	if stored.Number != 1 || stored.Status != domain.DecompositionProposed {
		t.Fatalf("stored = %+v, want proposed revision 1", stored)
	}

	// The whole point of a proposal: nothing exists yet.
	children, err := s.ListContributingOutcomes(ctx, parent.ID)
	if err != nil {
		t.Fatalf("list contributors: %v", err)
	}
	if len(children) != 0 {
		t.Fatalf("a proposal must create no contributing outcome, got %+v", children)
	}
	links, err := s.ListContributionLinksForParent(ctx, parent.ID)
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("a proposal must create no binding, got %+v", links)
	}

	read, ok, err := s.GetDecompositionRevision(ctx, parent.ID, stored.ID)
	if err != nil || !ok {
		t.Fatalf("get decomposition ok=%v err=%v", ok, err)
	}
	if len(read.Contributors) != 2 || len(read.Dependencies) != 1 {
		t.Fatalf("round-trip lost structure: %+v", read)
	}
	if read.Contributors[0].Ref != "c1" || !read.Contributors[0].ChildOutcomeID.IsZero() {
		t.Fatalf("an unauthorized contributor must not name an outcome: %+v", read.Contributors[0])
	}
	if read.Dependencies[0].FromRef != "c1" || read.Dependencies[0].ToRef != "c2" {
		t.Fatalf("dependency round-trip = %+v", read.Dependencies[0])
	}
}

func TestDecomposition_AuthorizationCreatesEverythingAtomically(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "d2")
	proposal, err := s.AppendDecompositionRevision(ctx, proposalFor(parent, revision, "dec-d2"))
	if err != nil {
		t.Fatalf("append decomposition: %v", err)
	}

	if err := s.AuthorizeDecompositionRevision(ctx, parent.ID, proposal.ID, authorizedFrom(parent, revision, proposal), decomposedAt); err != nil {
		t.Fatalf("authorize: %v", err)
	}

	children, err := s.ListContributingOutcomes(ctx, parent.ID)
	if err != nil {
		t.Fatalf("list contributors: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("authorization must create both contributors, got %d", len(children))
	}
	for _, child := range children {
		if child.ParentID != parent.ID || child.CurrentRevisionNumber != 1 {
			t.Fatalf("contributor %+v must own contract revision 1 under the parent", child)
		}
	}
	links, err := s.ListContributionLinksForParent(ctx, parent.ID)
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("authorization must bind both criteria, got %d", len(links))
	}

	read, _, err := s.GetDecompositionRevision(ctx, parent.ID, proposal.ID)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if read.Status != domain.DecompositionAuthorized || read.AuthorizedAt == nil {
		t.Fatalf("status = %q authorizedAt = %v", read.Status, read.AuthorizedAt)
	}
	for _, contributor := range read.Contributors {
		if contributor.ChildOutcomeID.IsZero() {
			t.Fatalf("contributor %q must resolve to its authorized outcome", contributor.Ref)
		}
	}
}

// A second authorization must not create a second set of contributing
// Outcomes. Storage refuses; the service turns that into a replay.
func TestDecomposition_AuthorizationIsClaimedOnce(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "d3")
	proposal, err := s.AppendDecompositionRevision(ctx, proposalFor(parent, revision, "dec-d3"))
	if err != nil {
		t.Fatalf("append decomposition: %v", err)
	}
	if err := s.AuthorizeDecompositionRevision(ctx, parent.ID, proposal.ID, authorizedFrom(parent, revision, proposal), decomposedAt); err != nil {
		t.Fatalf("first authorize: %v", err)
	}

	second := authorizedFrom(parent, revision, proposal)
	for i := range second {
		second[i].Outcome.ID += "-again"
		second[i].First.ID += "-again"
		for j := range second[i].Links {
			second[i].Links[j].ID += "-again"
			second[i].Links[j].ChildOutcomeID = second[i].Outcome.ID
		}
	}
	err = s.AuthorizeDecompositionRevision(ctx, parent.ID, proposal.ID, second, decomposedAt)
	if !errors.Is(err, ports.ErrDecompositionNotProposed) {
		t.Fatalf("re-authorization = %v, want ErrDecompositionNotProposed", err)
	}
	children, _ := s.ListContributingOutcomes(ctx, parent.ID)
	if len(children) != 2 {
		t.Fatalf("a refused re-authorization must create nothing, got %d contributors", len(children))
	}
}

// A partial failure would leave some contributing Outcomes existing and others
// not — a decomposition nobody authorized.
func TestDecomposition_FailedAuthorizationRollsEverythingBack(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "d4")
	proposal, err := s.AppendDecompositionRevision(ctx, proposalFor(parent, revision, "dec-d4"))
	if err != nil {
		t.Fatalf("append decomposition: %v", err)
	}

	contributions := authorizedFrom(parent, revision, proposal)
	// Break the SECOND contribution only: the first must not survive.
	contributions[1].Links[0].ParentCriterionID = "crit-does-not-exist"

	if err := s.AuthorizeDecompositionRevision(ctx, parent.ID, proposal.ID, contributions, decomposedAt); err == nil {
		t.Fatal("a contribution binding an unknown criterion must fail authorization")
	}
	children, err := s.ListContributingOutcomes(ctx, parent.ID)
	if err != nil {
		t.Fatalf("list contributors: %v", err)
	}
	if len(children) != 0 {
		t.Fatalf("a failed authorization must leave no contributor behind, got %+v", children)
	}
	read, _, err := s.GetDecompositionRevision(ctx, parent.ID, proposal.ID)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if read.Status != domain.DecompositionProposed {
		t.Fatalf("status = %q, want the proposal still open after rollback", read.Status)
	}
}

// The freeze triggers exist to defend against writers that are not the store,
// so the test has to be one.
func TestDecomposition_RecordsAreFrozenAgainstDirectWrites(t *testing.T) {
	dir := t.TempDir()
	s := sqlitetest.MustOpenAt(t, dir)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "d5")
	proposal, err := s.AppendDecompositionRevision(ctx, proposalFor(parent, revision, "dec-d5"))
	if err != nil {
		t.Fatalf("append decomposition: %v", err)
	}

	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "kennel.db"))
	if err != nil {
		t.Fatalf("open database directly: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Editing what the owner is being asked to agree to.
	if _, err := db.ExecContext(ctx, `UPDATE decomposition_revisions SET rationale = 'rewritten' WHERE id = ?`, string(proposal.ID)); err == nil {
		t.Fatal("a decomposition's rationale must be frozen")
	} else if !strings.Contains(err.Error(), "frozen") {
		t.Fatalf("refusal must name the freeze, got %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE decomposition_contributions SET goal = 'rewritten' WHERE decomposition_id = ?`, string(proposal.ID)); err == nil {
		t.Fatal("a proposed contribution's goal must be frozen")
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM decomposition_revisions WHERE id = ?`, string(proposal.ID)); err == nil {
		t.Fatal("decomposition revisions must be append-only")
	}
	if _, err := db.ExecContext(ctx, `UPDATE contribution_dependencies SET to_ref = 'c1' WHERE decomposition_id = ?`, string(proposal.ID)); err == nil {
		t.Fatal("contribution dependencies must be append-only")
	}
	// Reverting an authorized decomposition to proposed would let it be
	// authorized twice.
	if _, err := db.ExecContext(ctx, `UPDATE decomposition_revisions SET status = 'proposed' WHERE id = ?`, string(proposal.ID)); err == nil {
		t.Fatal("only the one-way move to authorized is permitted")
	}
}

func TestDecomposition_ProposalsAreAppendOnlyHistory(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "d6")

	first, err := s.AppendDecompositionRevision(ctx, proposalFor(parent, revision, "dec-d6a"))
	if err != nil {
		t.Fatalf("first proposal: %v", err)
	}
	second, err := s.AppendDecompositionRevision(ctx, proposalFor(parent, revision, "dec-d6b"))
	if err != nil {
		t.Fatalf("second proposal: %v", err)
	}
	if second.Number != first.Number+1 {
		t.Fatalf("numbers %d then %d, want append-only", first.Number, second.Number)
	}
	latest, ok, err := s.LatestDecompositionRevision(ctx, parent.ID)
	if err != nil || !ok {
		t.Fatalf("latest ok=%v err=%v", ok, err)
	}
	if latest.ID != second.ID {
		t.Fatalf("latest = %s, want the newer proposal %s", latest.ID, second.ID)
	}
	// Superseded proposals stay readable: a correction's history is part of
	// how the owner explains what they agreed to.
	if _, ok, _ := s.GetDecompositionRevision(ctx, parent.ID, first.ID); !ok {
		t.Fatal("an earlier proposal must remain readable")
	}
}

func TestDecomposition_EmitsProposalAndAuthorizationEvents(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "d7")
	proposal, err := s.AppendDecompositionRevision(ctx, proposalFor(parent, revision, "dec-d7"))
	if err != nil {
		t.Fatalf("append decomposition: %v", err)
	}
	if err := s.AuthorizeDecompositionRevision(ctx, parent.ID, proposal.ID, authorizedFrom(parent, revision, proposal), decomposedAt); err != nil {
		t.Fatalf("authorize: %v", err)
	}

	events, err := s.EventsAfter(ctx, 0, 300)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	var proposed, authorized int
	for _, event := range events {
		switch string(event.Type) {
		case "outcome_decomposition_proposed":
			proposed++
		case "outcome_decomposition_authorized":
			authorized++
		}
	}
	// The two are separate events because they mean different things: one
	// records what was offered, the other that responsibilities now exist.
	if proposed != 1 || authorized != 1 {
		t.Fatalf("proposed=%d authorized=%d, want exactly one of each", proposed, authorized)
	}
}

func TestDecomposition_RejectsPersistingAnAlreadyAuthorizedRevision(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "d8")
	proposal := proposalFor(parent, revision, "dec-d8")
	proposal.Status = domain.DecompositionAuthorized

	if _, err := s.AppendDecompositionRevision(ctx, proposal); err == nil {
		t.Fatal("a decomposition is persisted as proposed; authorization is a separate decision")
	}
}

// --- Dependency waivers (phase 3) ---

func TestDecomposition_WaiverRequiresADeclaredOrdering(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "w1")
	proposal, err := s.AppendDecompositionRevision(ctx, proposalFor(parent, revision, "dec-w1"))
	if err != nil {
		t.Fatalf("append decomposition: %v", err)
	}

	// The proposal declares c1 -> c2 only.
	if err := s.AppendContributionDependencyWaiver(ctx, domain.ContributionDependencyWaiver{
		ID: "cw-good", DecompositionID: proposal.ID, FromRef: "c1", ToRef: "c2",
		Reason: "The interface is frozen.", WaivedBy: domain.AcceptanceActorUser, CreatedAt: decomposedAt,
	}); err != nil {
		t.Fatalf("waiving a declared ordering must succeed: %v", err)
	}

	// Waiving an ordering nobody declared would record consent to nothing.
	err = s.AppendContributionDependencyWaiver(ctx, domain.ContributionDependencyWaiver{
		ID: "cw-bad", DecompositionID: proposal.ID, FromRef: "c2", ToRef: "c1",
		Reason: "Reversed.", WaivedBy: domain.AcceptanceActorUser, CreatedAt: decomposedAt,
	})
	if err == nil {
		t.Fatal("waiving an undeclared ordering must be refused by storage")
	}
	if !strings.Contains(err.Error(), "no such declared dependency") {
		t.Fatalf("refusal must name the reason, got %v", err)
	}

	waivers, err := s.ListContributionDependencyWaivers(ctx, proposal.ID)
	if err != nil {
		t.Fatalf("list waivers: %v", err)
	}
	if len(waivers) != 1 || waivers[0].FromRef != "c1" || waivers[0].Reason == "" {
		t.Fatalf("waivers = %+v, want the one declared override with its reason", waivers)
	}
	if waivers[0].WaivedBy != domain.AcceptanceActorUser {
		t.Fatalf("waivedBy = %q, want the owner", waivers[0].WaivedBy)
	}
}

// A waiver is a decision, so it is append-only and publishes like one.
func TestDecomposition_WaiversAreAppendOnlyAndPublished(t *testing.T) {
	dir := t.TempDir()
	s := sqlitetest.MustOpenAt(t, dir)
	ctx := context.Background()
	parent, revision := seedParent(t, s, "mer", "w2")
	proposal, err := s.AppendDecompositionRevision(ctx, proposalFor(parent, revision, "dec-w2"))
	if err != nil {
		t.Fatalf("append decomposition: %v", err)
	}
	if err := s.AppendContributionDependencyWaiver(ctx, domain.ContributionDependencyWaiver{
		ID: "cw-w2", DecompositionID: proposal.ID, FromRef: "c1", ToRef: "c2",
		Reason: "The interface is frozen.", WaivedBy: domain.AcceptanceActorUser, CreatedAt: decomposedAt,
	}); err != nil {
		t.Fatalf("waive: %v", err)
	}

	events, err := s.EventsAfter(ctx, 0, 300)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	waived := 0
	for _, event := range events {
		if string(event.Type) == "outcome_contribution_dependency_waived" {
			waived++
			if !strings.Contains(string(event.Payload), "c1") || !strings.Contains(string(event.Payload), string(proposal.ID)) {
				t.Fatalf("waiver payload must name the ordering and decomposition: %s", event.Payload)
			}
		}
	}
	if waived != 1 {
		t.Fatalf("emitted %d waiver events, want exactly one", waived)
	}

	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "kennel.db"))
	if err != nil {
		t.Fatalf("open database directly: %v", err)
	}
	defer func() { _ = db.Close() }()
	// Withdrawing a waiver is a new decomposition, never a delete.
	if _, err := db.ExecContext(ctx, `DELETE FROM contribution_dependency_waivers`); err == nil {
		t.Fatal("dependency waivers must reject DELETE")
	}
	if _, err := db.ExecContext(ctx, `UPDATE contribution_dependency_waivers SET reason = 'rewritten'`); err == nil {
		t.Fatal("dependency waivers must reject UPDATE")
	}
	// Only the owner may waive, at the storage layer too.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO contribution_dependency_waivers (id, decomposition_id, from_ref, to_ref, reason, waived_by)
		 VALUES ('cw-agent', ?, 'c1', 'c2', 'agent decided', 'agent')`, string(proposal.ID)); err == nil {
		t.Fatal("only the user may waive a dependency")
	}
}

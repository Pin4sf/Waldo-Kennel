package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func authorizedDecomposition(deps ...domain.ContributionDependency) domain.DecompositionRevision {
	return domain.DecompositionRevision{
		ID: "dec-1", OutcomeID: "out-parent", Number: 1, ContractRevisionID: "cr-1",
		Status: domain.DecompositionAuthorized, Rationale: "Three slices.",
		Contributors: []domain.ProposedContribution{
			{Ref: "c1", Position: 1, Title: "Slice one", ChildOutcomeID: "out-c1"},
			{Ref: "c2", Position: 2, Title: "Slice two", ChildOutcomeID: "out-c2"},
			{Ref: "c3", Position: 3, Title: "Slice three", ChildOutcomeID: "out-c3"},
		},
		Dependencies: deps,
	}
}

func TestStartGateDoesNotApplyWithoutAnAuthorizedDecomposition(t *testing.T) {
	proposal := authorizedDecomposition(domain.ContributionDependency{ID: "d1", FromRef: "c1", ToRef: "c2"})
	proposal.Status = domain.DecompositionProposed

	// An offer must never be able to block real work.
	gate := domain.DeriveContributionStartGate("out-c2", proposal, nil, nil)
	if gate.Applies || !gate.Clear() {
		t.Fatalf("a proposed decomposition must not gate anything, got %+v", gate)
	}

	// Neither may a decomposition that does not contain this Outcome.
	stranger := domain.DeriveContributionStartGate("out-elsewhere", authorizedDecomposition(), nil, nil)
	if stranger.Applies {
		t.Fatalf("an unrelated outcome must not be gated, got %+v", stranger)
	}
}

func TestStartGateBlocksOnUnacceptedUpstream(t *testing.T) {
	decomposition := authorizedDecomposition(
		domain.ContributionDependency{ID: "d1", FromRef: "c1", ToRef: "c3"},
		domain.ContributionDependency{ID: "d2", FromRef: "c2", ToRef: "c3"},
	)
	accepted := map[domain.OutcomeID]bool{"out-c1": true}

	gate := domain.DeriveContributionStartGate("out-c3", decomposition, nil, accepted)
	if !gate.Applies {
		t.Fatal("a contributor with declared upstreams must be gated")
	}
	if gate.Clear() {
		t.Fatal("an unaccepted upstream must block")
	}
	if len(gate.Blocked) != 1 || gate.Blocked[0].Ref != "c2" {
		t.Fatalf("blocked = %+v, want only the unaccepted c2", gate.Blocked)
	}
	// The refusal has to be explainable without opening a transcript.
	if gate.Blocked[0].Title != "Slice two" || gate.Blocked[0].Reason == "" {
		t.Fatalf("a block must name the contribution and say why: %+v", gate.Blocked[0])
	}

	// With both upstreams accepted the gate clears.
	accepted["out-c2"] = true
	if unblocked := domain.DeriveContributionStartGate("out-c3", decomposition, nil, accepted); !unblocked.Clear() {
		t.Fatalf("all upstreams accepted must clear the gate, got %+v", unblocked.Blocked)
	}
}

// The gate is directional: c1 gates c3, never the reverse.
func TestStartGateIsDirectional(t *testing.T) {
	decomposition := authorizedDecomposition(domain.ContributionDependency{ID: "d1", FromRef: "c1", ToRef: "c3"})
	upstream := domain.DeriveContributionStartGate("out-c1", decomposition, nil, nil)
	if !upstream.Clear() {
		t.Fatalf("the upstream contribution must not be blocked by its own dependents: %+v", upstream.Blocked)
	}
}

// A waived dependency is overridden, not forgotten: it stays visible next to
// whatever the early start produced.
func TestStartGateHonoursWaiversWithoutHidingThem(t *testing.T) {
	decomposition := authorizedDecomposition(domain.ContributionDependency{ID: "d1", FromRef: "c1", ToRef: "c2"})
	waivers := []domain.ContributionDependencyWaiver{{
		ID: "cw-1", DecompositionID: "dec-1", FromRef: "c1", ToRef: "c2",
		Reason: "The interface is already frozen.", WaivedBy: domain.AcceptanceActorUser,
	}}

	gate := domain.DeriveContributionStartGate("out-c2", decomposition, waivers, nil)
	if !gate.Clear() {
		t.Fatalf("a waived dependency must not block, got %+v", gate.Blocked)
	}
	if len(gate.Waived) != 1 || gate.Waived[0].Ref != "c1" {
		t.Fatalf("a waived dependency must stay visible, got %+v", gate.Waived)
	}

	// A waiver for a different edge must not clear this one.
	elsewhere := []domain.ContributionDependencyWaiver{{
		ID: "cw-2", DecompositionID: "dec-1", FromRef: "c3", ToRef: "c2",
		Reason: "Unrelated.", WaivedBy: domain.AcceptanceActorUser,
	}}
	if unrelated := domain.DeriveContributionStartGate("out-c2", decomposition, elsewhere, nil); unrelated.Clear() {
		t.Fatal("a waiver on a different edge must not clear this dependency")
	}
}

// Only the owner may override an ordering they authorized, and a waiver
// nobody can explain later is indistinguishable from a mistake.
func TestWaiverRequiresOwnerAndReason(t *testing.T) {
	base := domain.ContributionDependencyWaiver{
		ID: "cw-1", DecompositionID: "dec-1", FromRef: "c1", ToRef: "c2",
		Reason: "Interface is frozen.", WaivedBy: domain.AcceptanceActorUser,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("a well-formed waiver must validate: %v", err)
	}

	noReason := base
	noReason.Reason = "  "
	if err := noReason.Validate(); err == nil || !strings.Contains(err.Error(), "reason") {
		t.Fatalf("a waiver without a reason must be refused, got %v", err)
	}

	notOwner := base
	notOwner.WaivedBy = "agent"
	if err := notOwner.Validate(); err == nil || !strings.Contains(err.Error(), "only the user") {
		t.Fatalf("only the owner may waive, got %v", err)
	}
}

// An Outcome accepted and later reopened is not accepted. Treating it as
// accepted would let downstream work start on a result the owner withdrew.
func TestLatestDecisionAcceptsFollowsTheNewestDecision(t *testing.T) {
	early := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	accept := domain.AcceptanceDecision{Kind: domain.AcceptanceAccept, CreatedAt: early}
	reopen := domain.AcceptanceDecision{Kind: domain.AcceptanceReopen, CreatedAt: early.Add(time.Hour)}

	if !domain.LatestDecisionAccepts([]domain.AcceptanceDecision{accept}) {
		t.Fatal("a lone acceptance is an acceptance")
	}
	if domain.LatestDecisionAccepts([]domain.AcceptanceDecision{accept, reopen}) {
		t.Fatal("a reopen after an acceptance must not read as accepted")
	}
	// Order in the slice must not matter; the timestamp decides.
	if domain.LatestDecisionAccepts([]domain.AcceptanceDecision{reopen, accept}) {
		t.Fatal("the newest decision wins regardless of slice order")
	}
	if domain.LatestDecisionAccepts(nil) {
		t.Fatal("no decision is not an acceptance")
	}
}

func facts(id domain.OutcomeID, mutate func(*domain.ContributorFacts)) domain.ContributorFacts {
	f := domain.ContributorFacts{OutcomeID: id, Title: string(id)}
	if mutate != nil {
		mutate(&f)
	}
	return f
}

// A live session blocked on its user outranks everything else.
func TestAttentionPutsNeedsYouFirst(t *testing.T) {
	blocked := facts("out-a", func(f *domain.ContributorFacts) {
		f.Attempts = []domain.AttemptPresentation{{
			Phase: domain.AttemptPhaseNeedsInput, Attention: domain.AttemptAttentionWaitingInput,
			NextAction: "Answer the agent",
		}}
		// Even with everything else looking fine.
		f.ReadyForAcceptance = true
	})
	item := domain.DeriveContributorAttention(blocked)
	if item.Kind != domain.AttentionNeedsYou {
		t.Fatalf("kind = %q, want needs_you", item.Kind)
	}
	if item.NextAction != "Answer the agent" {
		t.Fatalf("the attempt's own next action must survive: %q", item.NextAction)
	}
}

func TestAttentionClassifiesEachSituation(t *testing.T) {
	tests := []struct {
		name string
		in   domain.ContributorFacts
		want domain.AttentionKind
	}{
		{
			name: "unconfirmed liveness needs the owner",
			in: facts("out-a", func(f *domain.ContributorFacts) {
				f.Attempts = []domain.AttemptPresentation{{Phase: domain.AttemptPhaseUnconfirmed}}
			}),
			want: domain.AttentionActionRequired,
		},
		{
			name: "stale binding needs the owner before anything else",
			in:   facts("out-b", func(f *domain.ContributorFacts) { f.Stale = true }),
			want: domain.AttentionActionRequired,
		},
		{
			name: "proved work awaits the owner's decision",
			in:   facts("out-c", func(f *domain.ContributorFacts) { f.ReadyForAcceptance = true }),
			want: domain.AttentionReadyForAcceptance,
		},
		{
			name: "an unmet dependency is waiting, not action required",
			in: facts("out-d", func(f *domain.ContributorFacts) {
				f.Gate = domain.ContributionStartGate{Applies: true, Blocked: []domain.UpstreamBlock{{Ref: "c1", Title: "Slice one"}}}
			}),
			want: domain.AttentionWaiting,
		},
		{
			name: "running work needs nothing",
			in: facts("out-e", func(f *domain.ContributorFacts) {
				f.Attempts = []domain.AttemptPresentation{{Phase: domain.AttemptPhaseExecuting}}
			}),
			want: domain.AttentionRunning,
		},
		{
			name: "accepted is closed",
			in:   facts("out-f", func(f *domain.ContributorFacts) { f.Accepted = true }),
			want: domain.AttentionAccepted,
		},
		{
			name: "nothing started yet is waiting with a next action",
			in:   facts("out-g", nil),
			want: domain.AttentionWaiting,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := domain.DeriveContributorAttention(tt.in)
			if item.Kind != tt.want {
				t.Fatalf("kind = %q, want %q", item.Kind, tt.want)
			}
			if item.OutcomeID != tt.in.OutcomeID {
				t.Fatal("every roll-up item must name the contributor it came from")
			}
			if strings.TrimSpace(item.Reason) == "" {
				t.Fatal("every roll-up item must explain itself")
			}
		})
	}
}

func TestParentAttentionRollsUpMostDemandingFirst(t *testing.T) {
	summary := domain.SummariseParentAttention([]domain.ContributorFacts{
		facts("out-accepted", func(f *domain.ContributorFacts) { f.Accepted = true }),
		facts("out-waiting", func(f *domain.ContributorFacts) {
			f.Gate = domain.ContributionStartGate{Applies: true, Blocked: []domain.UpstreamBlock{{Ref: "c1"}}}
		}),
		facts("out-blocked", func(f *domain.ContributorFacts) {
			f.Attempts = []domain.AttemptPresentation{{Phase: domain.AttemptPhaseNeedsInput, Attention: "blocked"}}
		}),
		facts("out-ready", func(f *domain.ContributorFacts) { f.ReadyForAcceptance = true }),
	})

	if summary.Headline != domain.AttentionNeedsYou {
		t.Fatalf("headline = %q, want the most demanding kind", summary.Headline)
	}
	if summary.Contributors != 4 || summary.AcceptedOf != 1 {
		t.Fatalf("summary counts = %+v", summary)
	}
	wantOrder := []domain.AttentionKind{
		domain.AttentionNeedsYou, domain.AttentionReadyForAcceptance,
		domain.AttentionWaiting, domain.AttentionAccepted,
	}
	for i, want := range wantOrder {
		if summary.Items[i].Kind != want {
			t.Fatalf("item %d = %q, want %q (order: %+v)", i, summary.Items[i].Kind, want, summary.Items)
		}
	}
	if summary.Counts[domain.AttentionNeedsYou] != 1 {
		t.Fatalf("counts = %+v", summary.Counts)
	}
}

// An empty decomposition has nothing to report rather than a false headline.
func TestParentAttentionOfNoContributorsIsEmpty(t *testing.T) {
	summary := domain.SummariseParentAttention(nil)
	if summary.Headline != "" || len(summary.Items) != 0 || summary.Contributors != 0 {
		t.Fatalf("summary = %+v, want empty", summary)
	}
}

package outcome_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

// fakeProposer records what a proposer was handed and hands back the token so
// a test can answer exactly as the spawned agent would.
type fakeProposer struct {
	mu    sync.Mutex
	input ports.DecompositionProposalInput
	err   error
}

func (f *fakeProposer) Propose(_ context.Context, in ports.DecompositionProposalInput) (ports.DecompositionProposalTicket, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.input = in
	if f.err != nil {
		return ports.DecompositionProposalTicket{}, f.err
	}
	return ports.DecompositionProposalTicket{SessionID: "sess-proposer", Detail: "spawned"}, nil
}

func (f *fakeProposer) token() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.input.CallbackToken
}

var requestClockStart = time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)

func newAskService(t *testing.T) (*outcome.Service, *attemptFakeStore, *fakeProposer, domain.OutcomeID, []domain.CriterionID) {
	t.Helper()
	store := newAttemptFakeStore()
	proposer := &fakeProposer{}
	svc := outcome.New(store, nil).WithDecompositionProposer(proposer)
	parentID, criteria := seedDecomposableParent(t, svc, store.fakeStore)
	return svc, store, proposer, parentID, criteria
}

func agentProposal(criteria []domain.CriterionID) outcome.ProposeDecompositionInput {
	return outcome.ProposeDecompositionInput{
		Rationale: "Two independent slices.",
		Contributors: []outcome.ProposedContributionInput{
			contributionOffer("c1", criteria[0]),
			contributionOffer("c2", criteria[1]),
		},
	}
}

func TestAskForDecompositionOpensADurableRequest(t *testing.T) {
	svc, _, proposer, parentID, _ := newAskService(t)
	ctx := context.Background()

	view, err := svc.AskForDecomposition(ctx, parentID, 1)
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if view.Request.Status != domain.DecompositionRequested {
		t.Fatalf("status = %q, want requested", view.Request.Status)
	}
	if view.Request.ExpiresAt.IsZero() {
		t.Fatal("the deadline must be durable, not an in-memory timer")
	}
	// The token reaches the agent; only its digest is ever recorded.
	if proposer.token() == "" {
		t.Fatal("the proposer must be handed a callback token")
	}
	if strings.Contains(view.Request.CallbackTokenDigest, proposer.token()) {
		t.Fatal("the raw token must never appear in the stored request")
	}
	// Criterion identities must reach the agent, or it cannot bind anything.
	if len(proposer.input.Contract.Criteria) == 0 {
		t.Fatal("the proposer must receive the parent's stable criterion identities")
	}
	if proposer.input.MaxContributions != domain.MaxProposedContributions {
		t.Fatalf("the proposer must be told the cap, got %d", proposer.input.MaxContributions)
	}
}

// A second agent answering the same Outcome would race to produce competing
// proposals, so the ask refuses before spawning one.
func TestAskForDecompositionRefusesASecondOpenRequest(t *testing.T) {
	svc, _, _, parentID, _ := newAskService(t)
	ctx := context.Background()
	if _, err := svc.AskForDecomposition(ctx, parentID, 1); err != nil {
		t.Fatalf("first ask: %v", err)
	}
	_, err := svc.AskForDecomposition(ctx, parentID, 1)
	if err == nil {
		t.Fatal("a second open ask must be refused")
	}
	if got := codeOf(t, err); got != "DECOMPOSITION_REQUEST_OPEN" {
		t.Fatalf("code = %q, want DECOMPOSITION_REQUEST_OPEN", got)
	}
}

// A failed spawn means nothing will ever answer, so the request is closed
// rather than left open with no agent behind it.
func TestAskForDecompositionClosesTheRequestWhenTheSpawnFails(t *testing.T) {
	store := newAttemptFakeStore()
	proposer := &fakeProposer{err: errors.New("no coordinator is ready")}
	svc := outcome.New(store, nil).WithDecompositionProposer(proposer)
	parentID, _ := seedDecomposableParent(t, svc, store.fakeStore)
	ctx := context.Background()

	if _, err := svc.AskForDecomposition(ctx, parentID, 1); err == nil {
		t.Fatal("a failed spawn must surface")
	}
	view, err := svc.LatestDecompositionRequest(ctx, parentID)
	if err != nil {
		t.Fatalf("read request: %v", err)
	}
	if view.Request.Status != domain.DecompositionRejected {
		t.Fatalf("status = %q, want rejected so nothing waits on an agent that never started", view.Request.Status)
	}
	if !strings.Contains(view.Request.RefusalReason, "no coordinator is ready") {
		t.Fatalf("the refusal must carry the spawn failure: %q", view.Request.RefusalReason)
	}
}

func TestSubmitAgentProposalProducesACorrectableProposal(t *testing.T) {
	svc, _, proposer, parentID, criteria := newAskService(t)
	ctx := context.Background()
	ask, err := svc.AskForDecomposition(ctx, parentID, 1)
	if err != nil {
		t.Fatalf("ask: %v", err)
	}

	in := agentProposal(criteria)
	view, err := svc.SubmitAgentProposal(ctx, ask.Request.ID, proposer.token(), in, outcome.MarshalRawProposal(in))
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if view.Request.Status != domain.DecompositionFulfilled || view.Request.DecompositionID.IsZero() {
		t.Fatalf("a valid proposal must fulfil the request: %+v", view.Request)
	}
	// It lands as an ordinary proposed decomposition — correctable, and not
	// authorized by anything the agent did.
	latest, err := svc.LatestDecomposition(ctx, parentID)
	if err != nil {
		t.Fatalf("latest decomposition: %v", err)
	}
	if latest.Decomposition.Status != domain.DecompositionProposed {
		t.Fatalf("status = %q; an agent proposal must not authorize itself", latest.Decomposition.Status)
	}
}

// Routing is checked before the proposal is parsed, and every refusal reads
// the same so a caller probing tokens learns nothing from the difference.
func TestSubmitAgentProposalRefusesBadRouting(t *testing.T) {
	tests := []struct {
		name  string
		token func(open string) string
		twice bool
	}{
		{name: "wrong token", token: func(string) string { return "not-the-token" }},
		{name: "no token", token: func(string) string { return "" }},
		{name: "second answer", token: func(open string) string { return open }, twice: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, proposer, parentID, criteria := newAskService(t)
			ctx := context.Background()
			ask, err := svc.AskForDecomposition(ctx, parentID, 1)
			if err != nil {
				t.Fatalf("ask: %v", err)
			}
			in := agentProposal(criteria)
			if tt.twice {
				if _, err := svc.SubmitAgentProposal(ctx, ask.Request.ID, proposer.token(), in, ""); err != nil {
					t.Fatalf("first submit: %v", err)
				}
			}
			_, err = svc.SubmitAgentProposal(ctx, ask.Request.ID, tt.token(proposer.token()), in, "")
			if err == nil {
				t.Fatal("the answer must not be admitted")
			}
			if got := codeOf(t, err); got != "DECOMPOSITION_REQUEST_NOT_ADMITTED" {
				t.Fatalf("code = %q, want DECOMPOSITION_REQUEST_NOT_ADMITTED", got)
			}
		})
	}
}

// A model proposal gets no special treatment: it fails the same gate a typed
// one would, and the draft is kept so the owner corrects it in the editor.
func TestSubmitAgentProposalKeepsARefusedDraft(t *testing.T) {
	svc, _, proposer, parentID, criteria := newAskService(t)
	ctx := context.Background()
	ask, err := svc.AskForDecomposition(ctx, parentID, 1)
	if err != nil {
		t.Fatalf("ask: %v", err)
	}

	// Only one criterion claimed: coverage refuses this.
	in := outcome.ProposeDecompositionInput{
		Rationale:    "Only the first slice.",
		Contributors: []outcome.ProposedContributionInput{contributionOffer("c1", criteria[0])},
	}
	view, err := svc.SubmitAgentProposal(ctx, ask.Request.ID, proposer.token(), in, outcome.MarshalRawProposal(in))
	if err != nil {
		t.Fatalf("a refused proposal is recorded, not an error: %v", err)
	}
	if view.Request.Status != domain.DecompositionRejected {
		t.Fatalf("status = %q, want rejected", view.Request.Status)
	}
	if !strings.Contains(view.Request.RefusalReason, "claimed by a contributing Outcome") {
		t.Fatalf("the daemon's own words must be kept: %q", view.Request.RefusalReason)
	}
	// The draft survives so one field can be fixed instead of regenerating.
	if !strings.Contains(view.Request.RawProposal, "c1") {
		t.Fatalf("the refused draft must be retained: %q", view.Request.RawProposal)
	}
	if _, err := svc.LatestDecomposition(ctx, parentID); err == nil {
		t.Fatal("a refused proposal must not create a decomposition")
	}
}

func TestSubmitAgentProposalRefusesARunawayGeneration(t *testing.T) {
	svc, _, proposer, parentID, criteria := newAskService(t)
	ctx := context.Background()
	ask, err := svc.AskForDecomposition(ctx, parentID, 1)
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	in := outcome.ProposeDecompositionInput{Rationale: "Too many."}
	for i := 0; i <= domain.MaxProposedContributions; i++ {
		in.Contributors = append(in.Contributors, contributionOffer("c"+string(rune('a'+i)), criteria[0]))
	}
	view, err := svc.SubmitAgentProposal(ctx, ask.Request.ID, proposer.token(), in, "")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if view.Request.Status != domain.DecompositionRejected ||
		!strings.Contains(view.Request.RefusalReason, "at most") {
		t.Fatalf("a runaway proposal must be refused by the cap: %+v", view.Request)
	}
}

// Expiry is a durable deadline, so a request that timed out while the daemon
// was down still reaches a verdict on the next start.
func TestExpireStaleDecompositionRequestsClosesTimedOutAsks(t *testing.T) {
	store := newAttemptFakeStore()
	proposer := &fakeProposer{}
	now := requestClockStart
	svc := outcome.New(store, func() time.Time { return now }).WithDecompositionProposer(proposer)
	parentID, _ := seedDecomposableParent(t, svc, store.fakeStore)
	ctx := context.Background()

	if _, err := svc.AskForDecomposition(ctx, parentID, 1); err != nil {
		t.Fatalf("ask: %v", err)
	}
	closed, err := svc.ExpireStaleDecompositionRequests(ctx)
	if err != nil || closed != 0 {
		t.Fatalf("a fresh request must not expire: closed=%d err=%v", closed, err)
	}

	now = now.Add(domain.DefaultDecompositionRequestTTL + time.Minute)
	closed, err = svc.ExpireStaleDecompositionRequests(ctx)
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if closed != 1 {
		t.Fatalf("closed %d, want the one timed-out request", closed)
	}
	view, err := svc.LatestDecompositionRequest(ctx, parentID)
	if err != nil {
		t.Fatalf("read request: %v", err)
	}
	if view.Request.Status != domain.DecompositionExpired {
		t.Fatalf("status = %q, want expired", view.Request.Status)
	}
	// And once expired the ask is free to be made again.
	if _, err := svc.AskForDecomposition(ctx, parentID, 1); err != nil {
		t.Fatalf("an expired ask must not block a new one: %v", err)
	}
}

// The row is written before the spawn, so the answering session can only be
// recorded afterwards. Without this a restart has no handle on what was
// working, which is exactly what the first real run on Mesa exposed.
func TestAskForDecompositionBindsTheAnsweringSession(t *testing.T) {
	svc, store, _, parentID, _ := newAskService(t)
	ctx := context.Background()

	view, err := svc.AskForDecomposition(ctx, parentID, 1)
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if view.SessionBindingFailed {
		t.Fatal("the binding must succeed on an open, unbound request")
	}
	if view.Request.SessionID != "sess-proposer" {
		t.Fatalf("returned session = %q, want the spawned one", view.Request.SessionID)
	}
	// And it must be DURABLE, not just present on the returned value — that
	// was the whole defect.
	stored, found, err := store.GetDecompositionRequest(ctx, view.Request.ID)
	if err != nil || !found {
		t.Fatalf("re-read request: found=%v err=%v", found, err)
	}
	if stored.SessionID != "sess-proposer" {
		t.Fatalf("stored session = %q, want it persisted so recovery can find it", stored.SessionID)
	}
}

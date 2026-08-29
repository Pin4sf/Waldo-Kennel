package outcome_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/service/outcome"
)

// decomposedProofHarness builds a parent with two contributors, each with its
// own contract, and returns everything needed to prove them.
func decomposedProofHarness(t *testing.T) (*outcome.Service, *proofFakeStore, domain.OutcomeID, []domain.OutcomeID) {
	t.Helper()
	store := &proofFakeStore{fakeStore: newFakeStore()}
	now := time.Date(2026, 8, 29, 20, 0, 0, 0, time.UTC)
	svc := outcome.New(store, func() time.Time {
		now = now.Add(time.Minute)
		return now
	}).WithProofStore(store)

	ctx := context.Background()
	in := validCreateInput()
	in.SuccessCriteria = []string{"The first slice is true.", "The second slice is true."}
	parent, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	store.mu.Lock()
	revs := store.revs[parent.Outcome.ID]
	revs[0].AuthorityCeiling = domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true}
	criteria := revs[0].Criteria
	store.revs[parent.Outcome.ID] = revs
	store.mu.Unlock()

	proposed, err := svc.ProposeDecomposition(ctx, parent.Outcome.ID, outcome.ProposeDecompositionInput{
		ExpectedContractRevision: 1,
		Rationale:                "Two independent slices.",
		Contributors: []outcome.ProposedContributionInput{
			contributionOffer("c1", criteria[0].ID),
			contributionOffer("c2", criteria[1].ID),
		},
	})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	authorized, err := svc.AuthorizeDecomposition(ctx, parent.Outcome.ID, proposed.Decomposition.ID)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	children := make([]domain.OutcomeID, 0, 2)
	for _, contributor := range authorized.Decomposition.Contributors {
		children = append(children, contributor.ChildOutcomeID)
	}
	return svc, store, parent.Outcome.ID, children
}

// proveContributor records supporting evidence and one passing verification of
// the given independence class for every criterion of a contributor.
func proveContributor(t *testing.T, svc *outcome.Service, child domain.OutcomeID, class domain.VerificationIndependenceClass, keyPrefix string) {
	t.Helper()
	ctx := context.Background()
	view, err := svc.Get(ctx, child)
	if err != nil {
		t.Fatalf("read contributor: %v", err)
	}
	for i, criterion := range view.Current.Criteria {
		suffix := keyPrefix + "-" + string(rune('a'+i))
		proof, err := svc.RecordEvidence(ctx, child, evidenceInput(view, criterion, "ev-"+suffix))
		if err != nil {
			t.Fatalf("record evidence: %v", err)
		}
		item := proof.Criteria[i].Evidence[len(proof.Criteria[i].Evidence)-1]
		verification := verificationInput(view, criterion, item.ID, "ver-"+suffix)
		verification.IndependenceClass = class
		switch class {
		case domain.VerificationProducerSelfCheck:
			verification.ProducerRef, verification.VerifierRef = "worker-1", "worker-1"
		case domain.VerificationSeparateSession:
			verification.ProducerRef, verification.VerifierRef = "worker-1", "reviewer-1"
		}
		if _, err := svc.RecordVerification(ctx, child, verification); err != nil {
			t.Fatalf("record verification: %v", err)
		}
	}
}

func batchInput(key string) outcome.AcceptBatchInput {
	return outcome.AcceptBatchInput{
		ExpectedContractRevision: 1,
		Summary:                  "Reviewed both slices together; the evidence holds.",
		ResourceDisposition:      domain.ResourceDispositionRetain,
		RequestKey:               key,
	}
}

// One sitting, N separate immutable decisions. This is the whole answer to
// composition's cost: batch the keystrokes, never the authority.
func TestAcceptContributorBatchWritesOneDecisionPerOutcome(t *testing.T) {
	svc, _, parentID, children := decomposedProofHarness(t)
	ctx := context.Background()
	proveContributor(t, svc, children[0], domain.VerificationSeparateSession, "c1")
	proveContributor(t, svc, children[1], domain.VerificationSeparateSession, "c2")

	view, err := svc.AcceptContributorBatch(ctx, parentID, batchInput("sitting-1"))
	if err != nil {
		t.Fatalf("accept batch: %v", err)
	}
	if len(view.Accepted) != 2 {
		t.Fatalf("accepted %d, want one decision per contributor", len(view.Accepted))
	}
	seen := map[domain.OutcomeID]bool{}
	for _, decision := range view.Accepted {
		if decision.Kind != domain.AcceptanceAccept || decision.ActorType != domain.AcceptanceActorUser {
			t.Fatalf("every decision is a user acceptance: %+v", decision)
		}
		if seen[decision.OutcomeID] {
			t.Fatalf("outcome %s decided twice in one sitting", decision.OutcomeID)
		}
		seen[decision.OutcomeID] = true
		// Separate records, not one decision fanned out.
		if decision.ID == "" || decision.ContractRevisionID == "" {
			t.Fatalf("each decision carries its own identity and contract: %+v", decision)
		}
	}
	for _, child := range children {
		proof, err := svc.GetProof(ctx, child)
		if err != nil {
			t.Fatalf("read contributor proof: %v", err)
		}
		if proof.Status != outcome.ProofStatusAccepted {
			t.Fatalf("contributor %s status = %q, want accepted", child, proof.Status)
		}
	}
}

// All contributors accepted makes the parent READY, never accepted.
func TestAcceptedContributorsMakeTheParentReadyNotAccepted(t *testing.T) {
	svc, _, parentID, children := decomposedProofHarness(t)
	ctx := context.Background()

	before, err := svc.GetProof(ctx, parentID)
	if err != nil {
		t.Fatalf("parent proof: %v", err)
	}
	if before.Status == outcome.ProofStatusReadyForAcceptance {
		t.Fatal("a parent with unproved contributors must not be ready")
	}
	for _, criterion := range before.Criteria {
		if !criterion.Delegated {
			t.Fatalf("every claimed criterion must read as delegated: %+v", criterion)
		}
	}

	proveContributor(t, svc, children[0], domain.VerificationSeparateSession, "c1")
	proveContributor(t, svc, children[1], domain.VerificationSeparateSession, "c2")
	if _, err := svc.AcceptContributorBatch(ctx, parentID, batchInput("sitting-1")); err != nil {
		t.Fatalf("accept batch: %v", err)
	}

	after, err := svc.GetProof(ctx, parentID)
	if err != nil {
		t.Fatalf("parent proof: %v", err)
	}
	if after.Status != outcome.ProofStatusReadyForAcceptance {
		t.Fatalf("parent status = %q, want ready_for_acceptance", after.Status)
	}
}

// A weakly-verified contributor is withheld, and its exclusion keeps the
// parent out of reach. This is what makes the batch not a rubber stamp.
func TestAcceptContributorBatchExcludesWeaklyVerifiedWork(t *testing.T) {
	svc, _, parentID, children := decomposedProofHarness(t)
	ctx := context.Background()
	proveContributor(t, svc, children[0], domain.VerificationSeparateSession, "c1")
	proveContributor(t, svc, children[1], domain.VerificationProducerSelfCheck, "c2")

	view, err := svc.AcceptContributorBatch(ctx, parentID, batchInput("sitting-1"))
	if err != nil {
		t.Fatalf("accept batch: %v", err)
	}
	if len(view.Accepted) != 1 || view.Accepted[0].OutcomeID != children[0] {
		t.Fatalf("only the independently verified contributor may be accepted: %+v", view.Accepted)
	}
	if len(view.Excluded) != 1 || view.Excluded[0].OutcomeID != children[1] {
		t.Fatalf("the weakly verified contributor must be excluded and escalated: %+v", view.Excluded)
	}
	if !strings.Contains(view.Excluded[0].Reason, "self-check") || view.Excluded[0].Remedy == "" {
		t.Fatalf("exclusion must name the weakness and the remedy: %+v", view.Excluded[0])
	}
	if view.Parent.Status == outcome.ProofStatusReadyForAcceptance {
		t.Fatal("an excluded contributor must keep the parent out of reach")
	}
}

func TestAcceptContributorBatchRefusesToAcceptAnUnprovedParent(t *testing.T) {
	svc, _, parentID, children := decomposedProofHarness(t)
	ctx := context.Background()
	proveContributor(t, svc, children[0], domain.VerificationSeparateSession, "c1")

	in := batchInput("sitting-1")
	in.AcceptParent = true
	_, err := svc.AcceptContributorBatch(ctx, parentID, in)
	if err == nil {
		t.Fatal("accepting the parent while a criterion is unproved must be refused")
	}
	if got := codeOf(t, err); got != "OUTCOME_NOT_READY_FOR_ACCEPTANCE" {
		t.Fatalf("code = %q, want OUTCOME_NOT_READY_FOR_ACCEPTANCE", got)
	}
	// The refusal must have written nothing at all, including the eligible
	// contributor's decision: one sitting is all-or-nothing.
	proof, err := svc.GetProof(ctx, children[0])
	if err != nil {
		t.Fatalf("read contributor proof: %v", err)
	}
	if proof.Status == outcome.ProofStatusAccepted {
		t.Fatal("a refused sitting must not have accepted anything")
	}
}

// The parent and its contributors are decided together in one sitting, each
// keeping its own immutable record.
func TestAcceptContributorBatchAcceptsTheParentWhenEverythingIsProved(t *testing.T) {
	svc, _, parentID, children := decomposedProofHarness(t)
	ctx := context.Background()
	proveContributor(t, svc, children[0], domain.VerificationSeparateSession, "c1")
	proveContributor(t, svc, children[1], domain.VerificationSeparateSession, "c2")

	in := batchInput("sitting-1")
	in.AcceptParent = true
	view, err := svc.AcceptContributorBatch(ctx, parentID, in)
	if err != nil {
		t.Fatalf("accept batch with parent: %v", err)
	}
	if view.ParentAccepted == nil || view.ParentAccepted.OutcomeID != parentID {
		t.Fatalf("the parent's own decision must be recorded separately: %+v", view.ParentAccepted)
	}
	if len(view.Accepted) != 2 {
		t.Fatalf("accepted %d contributors, want 2 alongside the parent", len(view.Accepted))
	}
	if view.Parent.Status != outcome.ProofStatusAccepted {
		t.Fatalf("parent status = %q, want accepted", view.Parent.Status)
	}
}

// A delivered sitting never decides twice.
func TestAcceptContributorBatchReplayIsIdempotent(t *testing.T) {
	svc, store, parentID, children := decomposedProofHarness(t)
	ctx := context.Background()
	proveContributor(t, svc, children[0], domain.VerificationSeparateSession, "c1")
	proveContributor(t, svc, children[1], domain.VerificationSeparateSession, "c2")

	if _, err := svc.AcceptContributorBatch(ctx, parentID, batchInput("sitting-1")); err != nil {
		t.Fatalf("first sitting: %v", err)
	}
	store.proofMu.Lock()
	afterFirst := len(store.decisions)
	store.proofMu.Unlock()

	if _, err := svc.AcceptContributorBatch(ctx, parentID, batchInput("sitting-1")); err != nil {
		t.Fatalf("replayed sitting must not fail: %v", err)
	}
	store.proofMu.Lock()
	afterReplay := len(store.decisions)
	store.proofMu.Unlock()
	if afterReplay != afterFirst {
		t.Fatalf("replay wrote again: %d then %d decisions", afterFirst, afterReplay)
	}
}

func TestBatchEligibilityReportsWithoutDeciding(t *testing.T) {
	svc, store, parentID, children := decomposedProofHarness(t)
	ctx := context.Background()
	proveContributor(t, svc, children[0], domain.VerificationSeparateSession, "c1")

	verdicts, err := svc.BatchEligibility(ctx, parentID)
	if err != nil {
		t.Fatalf("eligibility: %v", err)
	}
	if len(verdicts) != 2 {
		t.Fatalf("want a verdict per contributor, got %d", len(verdicts))
	}
	eligible := 0
	for _, verdict := range verdicts {
		if verdict.Eligible {
			eligible++
		}
		if strings.TrimSpace(verdict.Reason) == "" {
			t.Fatalf("every verdict must explain itself: %+v", verdict)
		}
	}
	if eligible != 1 {
		t.Fatalf("eligible = %d, want only the proved contributor", eligible)
	}
	store.proofMu.Lock()
	defer store.proofMu.Unlock()
	if len(store.decisions) != 0 {
		t.Fatal("reporting eligibility must decide nothing")
	}
}

// A direct Outcome has no contributors to batch.
func TestAcceptContributorBatchRefusesADirectOutcome(t *testing.T) {
	svc, _, view := newProofService(t)
	_, err := svc.AcceptContributorBatch(context.Background(), view.Outcome.ID, batchInput("sitting-1"))
	if err == nil {
		t.Fatal("a direct Outcome has no contributors to accept together")
	}
	if got := codeOf(t, err); got != "OUTCOME_NOT_DECOMPOSED" {
		t.Fatalf("code = %q, want OUTCOME_NOT_DECOMPOSED", got)
	}
}

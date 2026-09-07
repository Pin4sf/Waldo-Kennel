package domain_test

import (
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// The classes are not comparable on one axis, so the only real threshold is
// independence from the producer. Anything else is an exact requirement.
func TestIndependenceSatisfiesRefusesToInventATotalOrder(t *testing.T) {
	independent := []domain.VerificationIndependenceClass{
		domain.VerificationSeparateSession, domain.VerificationCrossProvider,
		domain.VerificationDeterministic, domain.VerificationOwnerWalkthrough,
	}
	for _, actual := range independent {
		if !domain.IndependenceSatisfies(actual, domain.VerificationSeparateSession) {
			t.Fatalf("%q is independent of the producer and must satisfy the independence bar", actual)
		}
	}
	if domain.IndependenceSatisfies(domain.VerificationProducerSelfCheck, domain.VerificationSeparateSession) {
		t.Fatal("a producer self-check must never satisfy the independence bar")
	}
	// Nothing satisfies a bar when nothing verified it.
	if domain.IndependenceSatisfies("", domain.VerificationSeparateSession) {
		t.Fatal("an unverified criterion satisfies no bar")
	}
	// A named class other than the independence threshold is exact.
	if domain.IndependenceSatisfies(domain.VerificationSeparateSession, domain.VerificationCrossProvider) {
		t.Fatal("cross-provider is an exact requirement, not a rank a separate session clears")
	}
	if !domain.IndependenceSatisfies(domain.VerificationCrossProvider, domain.VerificationCrossProvider) {
		t.Fatal("an exact requirement is satisfied by its own class")
	}
	// The floor accepts anything that verified at all.
	if !domain.IndependenceSatisfies(domain.VerificationProducerSelfCheck, domain.VerificationProducerSelfCheck) {
		t.Fatal("a self-check bar accepts a self-check")
	}
}

func TestWeakestIndependencePrefersTheSelfCheck(t *testing.T) {
	got := domain.WeakestIndependence([]domain.VerificationIndependenceClass{
		domain.VerificationDeterministic, domain.VerificationProducerSelfCheck, domain.VerificationCrossProvider,
	})
	if got != domain.VerificationProducerSelfCheck {
		t.Fatalf("weakest = %q, want the self-check: one weak criterion weakens the whole contribution", got)
	}
	if domain.WeakestIndependence(nil) != "" {
		t.Fatal("no verification is not a class")
	}
}

func proofFacts(mutate func(*domain.ContributorProofFacts)) domain.ContributorProofFacts {
	f := domain.ContributorProofFacts{
		OutcomeID: "out-c1", Title: "Slice one", Ready: true,
		BackingIndependence: domain.VerificationSeparateSession,
	}
	if mutate != nil {
		mutate(&f)
	}
	return f
}

func TestBatchEntryAdmitsProvedIndependentlyVerifiedWork(t *testing.T) {
	verdict := domain.EligibleForAcceptanceBatch(proofFacts(nil), domain.MinimumBatchIndependence)
	if !verdict.Eligible {
		t.Fatalf("proved and independently verified work must enter the batch: %+v", verdict)
	}
	if verdict.OutcomeID != "out-c1" || verdict.Reason == "" {
		t.Fatalf("a verdict must name its Outcome and explain itself: %+v", verdict)
	}
}

// The exclusion cases are what make a batch not a rubber stamp.
func TestBatchEntryExcludesWeakOrContradictedWork(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*domain.ContributorProofFacts)
		wantReason string
	}{
		{
			name:       "producer self-check only",
			mutate:     func(f *domain.ContributorProofFacts) { f.BackingIndependence = domain.VerificationProducerSelfCheck },
			wantReason: "self-check",
		},
		{
			name:       "nothing verified it",
			mutate:     func(f *domain.ContributorProofFacts) { f.BackingIndependence = "" },
			wantReason: "nothing",
		},
		{
			name:       "contradicted evidence",
			mutate:     func(f *domain.ContributorProofFacts) { f.Contradicted = true },
			wantReason: "contradicts",
		},
		{
			name:       "not yet proved",
			mutate:     func(f *domain.ContributorProofFacts) { f.Ready = false },
			wantReason: "not fully proved",
		},
		{
			name:       "bound to a superseded parent revision",
			mutate:     func(f *domain.ContributorProofFacts) { f.Stale = true },
			wantReason: "superseded",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verdict := domain.EligibleForAcceptanceBatch(proofFacts(tt.mutate), domain.MinimumBatchIndependence)
			if verdict.Eligible {
				t.Fatal("this contributor must be withheld from the batch")
			}
			if !strings.Contains(verdict.Reason, tt.wantReason) {
				t.Fatalf("reason %q must mention %q", verdict.Reason, tt.wantReason)
			}
			// Exclusion is escalation, not rejection: it names the way out.
			if strings.TrimSpace(verdict.Remedy) == "" {
				t.Fatalf("an excluded contributor must be told the smallest remedy: %+v", verdict)
			}
		})
	}
}

// A contradiction outranks a missing proof: telling the owner to "finish the
// evidence" when the evidence says the opposite would be misleading.
func TestBatchEntryReportsContradictionBeforeIncompleteness(t *testing.T) {
	verdict := domain.EligibleForAcceptanceBatch(proofFacts(func(f *domain.ContributorProofFacts) {
		f.Contradicted = true
		f.Ready = false
	}), domain.MinimumBatchIndependence)
	if !strings.Contains(verdict.Reason, "contradicts") {
		t.Fatalf("reason = %q, want the contradiction named first", verdict.Reason)
	}
}

func TestBatchEntrySkipsAlreadyAcceptedWork(t *testing.T) {
	verdict := domain.EligibleForAcceptanceBatch(proofFacts(func(f *domain.ContributorProofFacts) {
		f.Accepted = true
	}), domain.MinimumBatchIndependence)
	if verdict.Eligible {
		t.Fatal("an accepted contributor has nothing left to decide")
	}
	if verdict.Reason != "already accepted" {
		t.Fatalf("reason = %q, want a plain statement rather than a complaint", verdict.Reason)
	}
}

// A criterion two contributors share is only true when BOTH are accepted.
func TestDelegatedCriteriaRequireEveryClaimant(t *testing.T) {
	current := revisionWithCriteria("crit-a", "crit-b")
	links := []domain.ContributionLink{
		link("out-x", "cr-1", "crit-a"),
		link("out-y", "cr-1", "crit-a"),
		link("out-x", "cr-1", "crit-b"),
	}
	titles := map[domain.OutcomeID]string{"out-x": "Slice X", "out-y": "Slice Y"}

	partial := domain.DelegatedCriteria(current, links, map[domain.OutcomeID]bool{"out-x": true}, titles)
	if partial["crit-a"].Proved {
		t.Fatal("a shared criterion is not proved while one claimant is unaccepted")
	}
	if !strings.Contains(partial["crit-a"].Gap, "Slice Y") {
		t.Fatalf("the gap must name who is outstanding: %q", partial["crit-a"].Gap)
	}
	if !partial["crit-b"].Proved {
		t.Fatal("a criterion whose only claimant is accepted is proved")
	}

	full := domain.DelegatedCriteria(current, links, map[domain.OutcomeID]bool{"out-x": true, "out-y": true}, titles)
	if !full["crit-a"].Proved || full["crit-a"].Gap != "" {
		t.Fatalf("both claimants accepted proves the criterion: %+v", full["crit-a"])
	}
}

// Retained criteria stay on the parent's own evidence path, so they must not
// appear as delegated.
func TestDelegatedCriteriaOmitRetainedAndSupersededOnes(t *testing.T) {
	current := revisionWithCriteria("crit-a", "crit-retained")
	links := []domain.ContributionLink{
		link("out-x", "cr-1", "crit-a"),
		// Bound to a superseded parent revision: must not carry forward.
		link("out-y", "cr-0", "crit-retained"),
	}
	delegated := domain.DelegatedCriteria(current, links, nil, nil)
	if _, ok := delegated["crit-retained"]; ok {
		t.Fatal("a link bound to a superseded revision must not delegate the new one")
	}
	if _, ok := delegated["crit-a"]; !ok {
		t.Fatal("a current link must delegate its criterion")
	}
}

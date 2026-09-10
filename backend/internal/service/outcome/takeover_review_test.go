package outcome_test

import (
	"context"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

func TestTakeoverReview_RunKeyCannotChangeCommand(t *testing.T) {
	h := newRunHarness(t)
	h.mustCommand(t, domain.RunCommandStart, "same-key")
	if _, err := h.command(t, domain.RunCommandCancel, "same-key"); err == nil {
		t.Fatal("cancel with the Start key silently succeeded without cancelling")
	}
}

func TestTakeoverReview_RunKeyBindsCompleteCommandIdentity(t *testing.T) {
	cases := []struct {
		name  string
		input outcome.RunCommandInput
	}{
		{name: "plan", input: outcome.RunCommandInput{Command: domain.RunCommandStart, PlanRevisionID: "different-plan", ExpectedContractRevision: 1}},
		{name: "contract", input: outcome.RunCommandInput{Command: domain.RunCommandStart, ExpectedContractRevision: 2}},
		{name: "generation", input: outcome.RunCommandInput{Command: domain.RunCommandStart, ExpectedContractRevision: 1, ExpectedGeneration: 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newRunHarness(t)
			h.mustCommand(t, domain.RunCommandStart, "same-key")
			tc.input.RequestKey = "same-key"
			if tc.input.PlanRevisionID == "" {
				tc.input.PlanRevisionID = h.planID
			}
			if _, err := h.svc.CommandRun(context.Background(), h.outcomeID, tc.input); err == nil {
				t.Fatalf("reused key with changed %s semantics silently succeeded", tc.name)
			}
		})
	}
}

func TestTakeoverReview_StartRequiresReviewedContract(t *testing.T) {
	h := newRunHarness(t)
	_, err := h.svc.CommandRun(context.Background(), h.outcomeID, outcome.RunCommandInput{
		Command: domain.RunCommandStart, PlanRevisionID: h.planID,
		ExpectedContractRevision: 999, RequestKey: "wrong-contract",
	})
	if err == nil {
		t.Fatal("Start ignored the caller's expected Contract revision")
	}
}

func TestTakeoverReview_DocumentApprovalRequiresDigest(t *testing.T) {
	h := newDocumentHarness(t)
	ctx := context.Background()
	if _, err := h.svc.SelectDocuments(ctx, h.outcomeID, []string{h.writeDoc(t, "brief.md", "approved material")}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.ApproveDocuments(ctx, h.outcomeID, ""); err == nil {
		t.Fatal("approved documents without naming the reviewed digest")
	}
}

func TestTakeoverReview_ActiveCheckReservationIsNotInterrupted(t *testing.T) {
	h := newCheckedHarness(t, nil)
	ctx := context.Background()
	h.runner.observe = func(check domain.ApprovedCheck) ports.AttemptCheckObservation {
		// Re-enter at the real service boundary while the first invocation is active.
		if err := h.svc.ReconcileAttemptOutcomes(ctx); err != nil {
			t.Fatalf("second reconcile: %v", err)
		}
		run, found, err := h.runs.GetAttemptCheckRun(ctx, h.attempt.ID, h.check.ID, h.receipt.ArtifactVersion)
		if err != nil || !found {
			t.Fatalf("reservation: %v %v", found, err)
		}
		if run.State != ports.CheckRunReserved {
			t.Fatalf("active check was marked %s while its owner is still executing", run.State)
		}
		return failingObservation(check)
	}
	if err := h.svc.ReconcileAttemptOutcomes(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestTakeoverReview_CheckOwnershipIsPublishedBeforeReservationReturns(t *testing.T) {
	h := newCheckedHarness(t, nil)
	ctx := context.Background()
	wrapper := &reentrantAfterCommitCheckRunStore{checkRunFakeStore: h.runs}
	h.svc.WithCheckRunner(h.runner, wrapper)
	wrapper.reenter = func() error {
		return h.svc.ReconcileAttemptOutcomes(ctx)
	}

	if err := h.svc.ReconcileAttemptOutcomes(ctx); err != nil {
		t.Fatalf("outer reconcile: %v", err)
	}
	if wrapper.reentryErr != nil {
		t.Fatalf("re-entrant reconcile: %v", wrapper.reentryErr)
	}
	run, found, err := h.runs.GetAttemptCheckRun(ctx, h.attempt.ID, h.check.ID, h.receipt.ArtifactVersion)
	if err != nil || !found {
		t.Fatalf("reservation: found=%v err=%v", found, err)
	}
	if run.State != ports.CheckRunObserved {
		t.Fatalf("reservation state = %s, want observed; a live row was treated as abandoned", run.State)
	}
	if got := h.runner.totalChecksInvoked(); got != 1 {
		t.Fatalf("check invocations = %d, want one", got)
	}
}

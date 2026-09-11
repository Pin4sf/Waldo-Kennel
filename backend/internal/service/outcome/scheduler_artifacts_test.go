package outcome

import (
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// provenPredecessorSchedule builds the state that makes this distinction
// visible: wu-a's criterion is proved, so wu-b is no longer waiting on proof.
func provenPredecessorSchedule(t *testing.T, artifactBlocks map[domain.WorkUnitID]string) ScheduleView {
	t.Helper()
	plan := schedulerPlanFixture()
	producer := domain.Attempt{
		ID: "att-a", OutcomeID: plan.OutcomeID, PlanRevisionID: plan.ID, WorkUnitID: "wu-a",
		ContractRevisionNumber: plan.ContractRevisionNumber, Number: 1, Status: domain.AttemptSucceeded,
	}
	proof := schedulerProofFixture(plan)
	addProvenAttemptCriterion(&proof, plan, producer, retainedArtifactV1, "crit-a", time.Unix(100, 0).UTC())

	view, err := deriveSchedule(plan, []domain.Attempt{producer}, proof, artifactBlocks)
	if err != nil {
		t.Fatalf("deriveSchedule error = %v", err)
	}
	return view
}

func scheduleEntry(t *testing.T, view ScheduleView, id domain.WorkUnitID) WorkUnitScheduleView {
	t.Helper()
	for _, entry := range view.WorkUnits {
		if entry.WorkUnit.ID == id {
			return entry
		}
	}
	t.Fatalf("no schedule entry for %s", id)
	return WorkUnitScheduleView{}
}

// TestDeriveSchedule_AProvedDependencyWithUnusableOutputIsNotRunnable is the
// alignment rule between the graph and admission. A dependency can satisfy its
// Contract criterion while the bytes it produced cannot be handed down, and a
// graph that showed the successor as runnable would be offering a Start the
// daemon is certain to refuse.
func TestDeriveSchedule_AProvedDependencyWithUnusableOutputIsNotRunnable(t *testing.T) {
	runnable := provenPredecessorSchedule(t, nil)
	if got := scheduleEntry(t, runnable, "wu-b"); got.State != WorkUnitScheduleRunnable {
		t.Fatalf("baseline wu-b state = %q, want runnable (blocked reason %q)", got.State, got.BlockedReason)
	}
	if runnable.NextRunnableID != "wu-b" {
		t.Fatalf("baseline next runnable = %q, want wu-b", runnable.NextRunnableID)
	}

	blocked := provenPredecessorSchedule(t, map[domain.WorkUnitID]string{"wu-b": CodeUpstreamArtifactUnreviewed})
	entry := scheduleEntry(t, blocked, "wu-b")
	if entry.State != WorkUnitScheduleBlocked || entry.BlockedReason != BlockedUpstreamArtifactUnavailable {
		t.Fatalf("wu-b = %q/%q, want blocked/%s", entry.State, entry.BlockedReason, BlockedUpstreamArtifactUnavailable)
	}
	// The specific refusal has to survive: "unavailable" alone does not tell
	// the owner whether the result is missing, incomplete or merely unfrozen.
	if entry.BlockedDetail != CodeUpstreamArtifactUnreviewed {
		t.Fatalf("blocked detail = %q, want %s", entry.BlockedDetail, CodeUpstreamArtifactUnreviewed)
	}
	// It is blocked on its artifact, not on proof: reporting the dependency as
	// unproved would send the owner to repair evidence that is already fine.
	if len(entry.BlockingDependencies) != 0 {
		t.Fatalf("blocking dependencies = %v, want none", entry.BlockingDependencies)
	}
	if blocked.NextRunnableID == "wu-b" {
		t.Fatal("a unit whose inputs cannot be provisioned was offered as next runnable")
	}
	// wu-a keeps its own truthful state.
	if got := scheduleEntry(t, blocked, "wu-a"); got.State != WorkUnitScheduleProven {
		t.Fatalf("wu-a state = %q, want proven", got.State)
	}
}

// TestDeriveSchedule_UnprovedDependencyStillReportsAwaitingProof keeps the two
// blocked reasons distinct: the owner's next move is different for each.
func TestDeriveSchedule_UnprovedDependencyStillReportsAwaitingProof(t *testing.T) {
	plan := schedulerPlanFixture()
	view, err := deriveSchedule(plan, nil, schedulerProofFixture(plan), map[domain.WorkUnitID]string{"wu-b": CodeUpstreamArtifactMissing})
	if err != nil {
		t.Fatalf("deriveSchedule error = %v", err)
	}
	entry := scheduleEntry(t, view, "wu-b")
	if entry.BlockedReason != BlockedAwaitingDependencyProof {
		t.Fatalf("wu-b blocked reason = %q, want %s", entry.BlockedReason, BlockedAwaitingDependencyProof)
	}
	if len(entry.BlockingDependencies) != 1 || entry.BlockingDependencies[0] != "wu-a" {
		t.Fatalf("blocking dependencies = %v, want [wu-a]", entry.BlockingDependencies)
	}
}

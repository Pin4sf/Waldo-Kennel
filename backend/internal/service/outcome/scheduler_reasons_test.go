package outcome

import (
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func scheduleFor(t *testing.T, attempts []domain.Attempt) ScheduleView {
	t.Helper()
	plan := schedulerPlanFixture()
	view, err := deriveSchedule(plan, attempts, schedulerProofFixture(plan), nil)
	if err != nil {
		t.Fatalf("deriveSchedule error = %v", err)
	}
	return view
}

func unitState(t *testing.T, view ScheduleView, id domain.WorkUnitID) WorkUnitScheduleView {
	t.Helper()
	for _, entry := range view.WorkUnits {
		if entry.WorkUnit.ID == id {
			return entry
		}
	}
	t.Fatalf("unit %s missing from the schedule; every approved unit must appear", id)
	return WorkUnitScheduleView{}
}

// "Blocked" used to mean two unrelated things: this unit's dependencies are
// unproven, and some other unit is holding the serial custody fence. The
// owner's next move differs completely, so the graph cannot draw them alike.
func TestScheduleDistinguishesDependencyProofFromCustodyHeld(t *testing.T) {
	// Nothing running: B waits only on A's proof.
	idle := scheduleFor(t, nil)
	b := unitState(t, idle, "wu-b")
	if b.State != WorkUnitScheduleBlocked || b.BlockedReason != BlockedAwaitingDependencyProof {
		t.Fatalf("B = %s/%s, want blocked awaiting dependency proof", b.State, b.BlockedReason)
	}
	if len(b.BlockingDependencies) != 1 || b.BlockingDependencies[0] != "wu-a" {
		t.Fatalf("B blocking dependencies = %v, want wu-a named", b.BlockingDependencies)
	}
	if idle.NextRunnableID != "wu-a" {
		t.Fatalf("next runnable = %q, want wu-a", idle.NextRunnableID)
	}
	if idle.NoRunnableReason != "" {
		t.Fatalf("noRunnableReason = %q, want empty while something is runnable", idle.NoRunnableReason)
	}

	// A running: A executes and holds custody; B is still dependency-blocked,
	// which is the stronger of the two reasons and the one that must show.
	running := scheduleFor(t, []domain.Attempt{{
		ID: "att-a", OutcomeID: "out-scheduler", PlanRevisionID: "plan-scheduler",
		WorkUnitID: "wu-a", ContractRevisionNumber: 1, Status: domain.AttemptRunning,
	}})
	if got := unitState(t, running, "wu-a"); got.State != WorkUnitScheduleExecuting {
		t.Fatalf("A = %s, want executing", got.State)
	}
	if got := unitState(t, running, "wu-b"); got.BlockedReason != BlockedAwaitingDependencyProof {
		t.Fatalf("B reason = %s, want dependency proof to outrank the fence", got.BlockedReason)
	}
	if running.CustodyHeldBy != "wu-a" {
		t.Fatalf("custodyHeldBy = %q, want wu-a", running.CustodyHeldBy)
	}
	if running.NoRunnableReason != NoRunnableExecuting {
		t.Fatalf("noRunnableReason = %q, want %q", running.NoRunnableReason, NoRunnableExecuting)
	}
}

// A paused attempt still holds custody, but it is not progressing and needs the
// owner. Reporting it as "executing" would make the Mission look busy while it
// is actually waiting on a decision.
func TestSchedulePausedAttemptIsNotReportedAsExecuting(t *testing.T) {
	view := scheduleFor(t, []domain.Attempt{{
		ID: "att-a", OutcomeID: "out-scheduler", PlanRevisionID: "plan-scheduler",
		WorkUnitID: "wu-a", ContractRevisionNumber: 1, Status: domain.AttemptPaused,
	}})
	if got := unitState(t, view, "wu-a"); got.State != WorkUnitSchedulePaused {
		t.Fatalf("A = %s, want paused", got.State)
	}
	if view.NoRunnableReason != NoRunnablePaused {
		t.Fatalf("noRunnableReason = %q, want %q", view.NoRunnableReason, NoRunnablePaused)
	}
}

// AttemptLost is reached only through explicit owner recovery, which
// deliberately releases custody so a replacement may start. So a lost attempt
// must leave its unit retryable and the fence open — reporting it as "unknown"
// would misdescribe a reconciliation the owner has already made and discourage
// a legitimate retry.
//
// Unresolved liveness is a real blocking state, but it is enforced upstream by
// keeping an unaccountable attempt Running; see the note in scheduler.go.
func TestScheduleTreatsOwnerReconciledLossAsRetryable(t *testing.T) {
	view := scheduleFor(t, []domain.Attempt{{
		ID: "att-a", OutcomeID: "out-scheduler", PlanRevisionID: "plan-scheduler",
		WorkUnitID: "wu-a", ContractRevisionNumber: 1, Status: domain.AttemptLost,
	}})
	a := unitState(t, view, "wu-a")
	if a.State != WorkUnitScheduleRetryable {
		t.Fatalf("A = %s, want retryable after owner-reconciled loss", a.State)
	}
	if !view.CustodyHeldBy.IsZero() {
		t.Fatalf("custodyHeldBy = %q, want the fence released", view.CustodyHeldBy)
	}
	if view.NextRunnableID != "wu-a" {
		t.Fatalf("next runnable = %q, want wu-a admissible for a governed retry", view.NextRunnableID)
	}
}

// An empty runnable set must always carry a reason. A Mission that can show a
// spinner with no explanation is the specific failure this prevents.
func TestScheduleAlwaysExplainsAnEmptyRunnableSet(t *testing.T) {
	plan := schedulerPlanFixture()
	proof := schedulerProofFixture(plan)
	attempt := domain.Attempt{
		ID: "att-a", OutcomeID: plan.OutcomeID, PlanRevisionID: plan.ID,
		WorkUnitID: "wu-a", ContractRevisionNumber: 1, Status: domain.AttemptRunning,
	}
	addProvenAttemptCriterion(&proof, plan, attempt, retainedArtifactV1, "crit-a", time.Unix(100, 0).UTC())

	// A is proven; B is now admissible, so there is still something runnable.
	view, err := deriveSchedule(plan, []domain.Attempt{attempt}, proof, nil)
	if err != nil {
		t.Fatal(err)
	}
	if view.NextRunnableID.IsZero() && view.NoRunnableReason == "" {
		t.Fatal("an empty runnable set must never be unexplained")
	}

	// Every unit proven: the reason is completion, not waiting.
	addProvenAttemptCriterion(&proof, plan, domain.Attempt{
		ID: "att-b", OutcomeID: plan.OutcomeID, PlanRevisionID: plan.ID,
		WorkUnitID: "wu-b", ContractRevisionNumber: 1, Status: domain.AttemptRunning,
	}, retainedArtifactV1, "crit-b", time.Unix(200, 0).UTC())
	done, err := deriveSchedule(plan, []domain.Attempt{
		{ID: "att-a", OutcomeID: plan.OutcomeID, PlanRevisionID: plan.ID, WorkUnitID: "wu-a", ContractRevisionNumber: 1, Status: domain.AttemptReconciled},
		{ID: "att-b", OutcomeID: plan.OutcomeID, PlanRevisionID: plan.ID, WorkUnitID: "wu-b", ContractRevisionNumber: 1, Status: domain.AttemptReconciled},
	}, proof, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !done.NextRunnableID.IsZero() {
		t.Fatalf("next runnable = %q, want nothing left", done.NextRunnableID)
	}
	if done.NoRunnableReason != NoRunnableAllProven {
		t.Fatalf("noRunnableReason = %q, want %q", done.NoRunnableReason, NoRunnableAllProven)
	}
}

// Every approved unit appears in the projection regardless of the order the
// plan happens to serialize them in. The graph cannot show all nodes if the
// schedule silently omits one.
func TestScheduleContainsEveryApprovedUnitInDependencyOrder(t *testing.T) {
	plan := schedulerPlanFixture()
	// Serialize B before A: dependency correctness must not depend on order.
	plan.WorkUnits = []domain.WorkUnit{plan.WorkUnits[1], plan.WorkUnits[0]}
	view, err := deriveSchedule(plan, nil, schedulerProofFixture(plan), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.WorkUnits) != 2 {
		t.Fatalf("units = %d, want both", len(view.WorkUnits))
	}
	if view.WorkUnits[0].WorkUnit.ID != "wu-a" || view.WorkUnits[1].WorkUnit.ID != "wu-b" {
		t.Fatalf("order = %s,%s; want topological A,B despite B being listed first",
			view.WorkUnits[0].WorkUnit.ID, view.WorkUnits[1].WorkUnit.ID)
	}
	if view.NextRunnableID != "wu-a" {
		t.Fatalf("next runnable = %q, want wu-a", view.NextRunnableID)
	}
}

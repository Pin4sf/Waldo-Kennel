package outcome

import (
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func schedulerPlanFixture() domain.PlanRevision {
	return domain.PlanRevision{
		ID: "plan-scheduler", OutcomeID: "out-scheduler", Number: 1, ContractRevisionNumber: 1,
		Status: domain.PlanStatusApproved, Summary: "A then B", RunBriefCoreDigest: string(make([]byte, 64)),
		WorkUnits: []domain.WorkUnit{
			{ID: "wu-a", Kind: domain.WorkUnitDirect, Title: "A", ContractRevisionNumber: 1, OutputSummary: "A output", EvidenceChecks: []string{"A evidence"}, VerificationRequirement: "verify A", StopConditions: []string{"stop"}, CriterionIDs: []domain.CriterionID{"crit-a"}, RequiredCapabilities: []string{domain.CapabilityWorktreeRead}},
			{ID: "wu-b", Kind: domain.WorkUnitDirect, Title: "B", ContractRevisionNumber: 1, OutputSummary: "B output", EvidenceChecks: []string{"B evidence"}, VerificationRequirement: "verify B", StopConditions: []string{"stop"}, DependsOn: []domain.WorkUnitID{"wu-a"}, CriterionIDs: []domain.CriterionID{"crit-b"}, RequiredCapabilities: []string{domain.CapabilityWorktreeRead}},
		},
	}
}

func TestNextRunnableWorkUnitProofGatesDependencies(t *testing.T) {
	plan := schedulerPlanFixture()

	unit, ok, err := nextRunnableWorkUnit(plan, nil, nil)
	if err != nil || !ok || unit.ID != "wu-a" {
		t.Fatalf("initial runnable = %s ok=%v err=%v, want wu-a", unit.ID, ok, err)
	}

	attempts := []domain.Attempt{{ID: "att-a", OutcomeID: plan.OutcomeID, PlanRevisionID: plan.ID, WorkUnitID: "wu-a", Status: domain.AttemptRunning}}
	if unit, ok, err := nextRunnableWorkUnit(plan, attempts, nil); err != nil || ok {
		t.Fatalf("active A must block another unit: unit=%s ok=%v err=%v", unit.ID, ok, err)
	}

	// Provider/session completion is not success. Even after custody is
	// reconciled, B is still blocked until verification proves A.
	attempts[0].Status = domain.AttemptReconciled
	unit, ok, err = nextRunnableWorkUnit(plan, attempts, nil)
	if err != nil || !ok || unit.ID != "wu-a" {
		t.Fatalf("unverified reconciled A should remain retryable: unit=%s ok=%v err=%v", unit.ID, ok, err)
	}

	verifications := []domain.VerificationRun{{
		OutcomeID: plan.OutcomeID, ContractRevisionID: "cr-1", CriterionID: "crit-a",
		SubjectType: domain.ProofSubjectAttempt, SubjectID: "att-a", Result: domain.VerificationPassed,
	}}
	unit, ok, err = nextRunnableWorkUnit(plan, attempts, verifications)
	if err != nil || !ok || unit.ID != "wu-b" {
		t.Fatalf("verified A should release B: unit=%s ok=%v err=%v", unit.ID, ok, err)
	}
}

func TestNextRunnableWorkUnitRequiresEveryCriterionOfDependency(t *testing.T) {
	plan := schedulerPlanFixture()
	plan.WorkUnits[0].CriterionIDs = []domain.CriterionID{"crit-a-1", "crit-a-2"}
	attempts := []domain.Attempt{{ID: "att-a", OutcomeID: plan.OutcomeID, PlanRevisionID: plan.ID, WorkUnitID: "wu-a", Status: domain.AttemptReconciled}}
	verifications := []domain.VerificationRun{{
		OutcomeID: plan.OutcomeID, CriterionID: "crit-a-1", SubjectType: domain.ProofSubjectWorkUnit,
		SubjectID: "wu-a", Result: domain.VerificationPassed,
	}}
	unit, ok, err := nextRunnableWorkUnit(plan, attempts, verifications)
	if err != nil || !ok || unit.ID != "wu-a" {
		t.Fatalf("partial proof must not release B: unit=%s ok=%v err=%v", unit.ID, ok, err)
	}
	verifications = append(verifications, domain.VerificationRun{
		OutcomeID: plan.OutcomeID, CriterionID: "crit-a-2", SubjectType: domain.ProofSubjectWorkUnit,
		SubjectID: "wu-a", Result: domain.VerificationPassed,
	})
	unit, ok, err = nextRunnableWorkUnit(plan, attempts, verifications)
	if err != nil || !ok || unit.ID != "wu-b" {
		t.Fatalf("complete proof should release B: unit=%s ok=%v err=%v", unit.ID, ok, err)
	}
}

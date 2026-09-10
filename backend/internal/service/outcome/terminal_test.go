package outcome

import (
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func attemptOn(id domain.AttemptID, unit domain.WorkUnitID, plan domain.PlanRevision, status domain.AttemptStatus) domain.Attempt {
	return domain.Attempt{
		ID: id, OutcomeID: plan.OutcomeID, PlanRevisionID: plan.ID, WorkUnitID: unit,
		ContractRevisionNumber: plan.ContractRevisionNumber, Status: status,
	}
}

// Success is derived from execution having ended plus proof bound to that exact
// attempt. This is the whole point of Phase C: a provider finishing is not a
// result.
func TestAttemptProvenRequiresProofBoundToThisAttempt(t *testing.T) {
	plan := schedulerPlanFixture()
	unit := plan.WorkUnits[0]
	proof := schedulerProofFixture(plan)
	attempt := attemptOn("att-a", unit.ID, plan, domain.AttemptReconciled)

	// Ended, but nothing proved: staying reconciled is the truthful state.
	if attemptProven(unit, attempt, retainedArtifactV1, proof) {
		t.Fatal("an ended attempt with no proof must not be classified as succeeded")
	}

	addProvenAttemptCriterion(&proof, plan, attempt, retainedArtifactV1, "crit-a", time.Unix(100, 0).UTC())
	if !attemptProven(unit, attempt, retainedArtifactV1, proof) {
		t.Fatal("proof bound to this attempt must classify it as succeeded")
	}
}

// The falsifier that made this function necessary. workUnitProven matches proof
// across every attempt of a unit, so using it here would classify an attempt
// that produced nothing as succeeded on a later attempt's evidence.
func TestAttemptProvenDoesNotBorrowAnotherAttemptsProof(t *testing.T) {
	plan := schedulerPlanFixture()
	unit := plan.WorkUnits[0]
	proof := schedulerProofFixture(plan)
	produced := attemptOn("att-second", unit.ID, plan, domain.AttemptReconciled)
	producedNothing := attemptOn("att-first", unit.ID, plan, domain.AttemptReconciled)

	addProvenAttemptCriterion(&proof, plan, produced, retainedArtifactV1, "crit-a", time.Unix(100, 0).UTC())

	if !attemptProven(unit, produced, retainedArtifactV1, proof) {
		t.Fatal("the attempt that produced the proof must be classified as succeeded")
	}
	if attemptProven(unit, producedNothing, retainedArtifactV1, proof) {
		t.Fatal("an attempt must never be classified as succeeded on another attempt's proof")
	}

	// And the scheduler's own question still answers yes: the unit IS proved.
	// The two questions are different, which is exactly why they are separate
	// functions.
	if !workUnitProven(plan, unit, []domain.Attempt{produced, producedNothing}, proof) {
		t.Fatal("the unit itself should read as proved for scheduling")
	}
}

// WorkUnit-scoped proof names no attempt, so it cannot say which one succeeded.
// It legitimately unblocks the scheduler; it must not classify an attempt.
func TestAttemptProvenIgnoresWorkUnitScopedProof(t *testing.T) {
	plan := schedulerPlanFixture()
	unit := plan.WorkUnits[0]
	proof := schedulerProofFixture(plan)
	attempt := attemptOn("att-a", unit.ID, plan, domain.AttemptReconciled)

	for i := range proof.Criteria {
		if proof.Criteria[i].Criterion.ID != "crit-a" {
			continue
		}
		evidenceID := domain.EvidenceItemID("ev-unit")
		proof.Criteria[i].Evidence = append(proof.Criteria[i].Evidence, domain.EvidenceItem{
			ID: evidenceID, OutcomeID: plan.OutcomeID, ContractRevisionID: proof.Contract.ID, CriterionID: "crit-a",
			SubjectType: domain.ProofSubjectWorkUnit, SubjectID: string(unit.ID), SubjectRevision: string(plan.ID),
			Kind: domain.EvidenceSupporting, SourceType: domain.EvidenceSourceArtifact, SourceRef: "artifact",
			ProducerType: domain.EvidenceProducerProvider, ProducerRef: "unit", Summary: "support",
			ContentDigest: strings.Repeat("b", 64), RequestKey: "ev-unit-key",
			RequestFingerprint: strings.Repeat("c", 64), CreatedAt: time.Unix(100, 0).UTC(),
		})
		proof.Criteria[i].Verifications = append(proof.Criteria[i].Verifications, domain.VerificationRun{
			ID: "ver-unit", OutcomeID: plan.OutcomeID, ContractRevisionID: proof.Contract.ID, CriterionID: "crit-a",
			SubjectType: domain.ProofSubjectWorkUnit, SubjectID: string(unit.ID), SubjectRevision: string(plan.ID),
			EvidenceItemIDs: []domain.EvidenceItemID{evidenceID},
			Method:          "deterministic", IndependenceClass: domain.VerificationDeterministic,
			Result: domain.VerificationPassed, VerifierRef: "test", RequestKey: "ver-unit-key",
			RequestFingerprint: strings.Repeat("d", 64), CreatedAt: time.Unix(101, 0).UTC(),
		})
	}

	if attemptProven(unit, attempt, retainedArtifactV1, proof) {
		t.Fatal("WorkUnit-scoped proof names no attempt and must not classify one")
	}
	if !workUnitProven(plan, unit, []domain.Attempt{attempt}, proof) {
		t.Fatal("WorkUnit-scoped proof should still unblock the scheduler")
	}
}

// A unit that carries no criteria cannot be proved, so it can never be
// classified as succeeded. Treating "nothing to check" as success is how a
// plan with missing coverage would silently pass.
func TestAttemptProvenRefusesAUnitWithNoCriteria(t *testing.T) {
	plan := schedulerPlanFixture()
	unit := plan.WorkUnits[0]
	unit.CriterionIDs = nil
	attempt := attemptOn("att-a", unit.ID, plan, domain.AttemptReconciled)
	if attemptProven(unit, attempt, retainedArtifactV1, schedulerProofFixture(plan)) {
		t.Fatal("a unit with no criteria must never classify as succeeded")
	}
}

// Partial proof is not proof: every criterion the unit carries has to be ready.
func TestAttemptProvenRequiresEveryCriterion(t *testing.T) {
	plan := schedulerPlanFixture()
	// Give unit A both criteria so one proved criterion is genuinely partial.
	unit := plan.WorkUnits[0]
	unit.CriterionIDs = []domain.CriterionID{"crit-a", "crit-b"}
	proof := schedulerProofFixture(plan)
	attempt := attemptOn("att-a", unit.ID, plan, domain.AttemptReconciled)

	addProvenAttemptCriterion(&proof, plan, attempt, retainedArtifactV1, "crit-a", time.Unix(100, 0).UTC())
	if attemptProven(unit, attempt, retainedArtifactV1, proof) {
		t.Fatal("one proved criterion out of two must not classify as succeeded")
	}
	addProvenAttemptCriterion(&proof, plan, attempt, retainedArtifactV1, "crit-b", time.Unix(101, 0).UTC())
	if !attemptProven(unit, attempt, retainedArtifactV1, proof) {
		t.Fatal("every criterion proved must classify as succeeded")
	}
}

// A delegated criterion is a contributing Outcome's responsibility, so this
// attempt cannot claim it.
func TestAttemptProvenRefusesDelegatedCriteria(t *testing.T) {
	plan := schedulerPlanFixture()
	unit := plan.WorkUnits[0]
	proof := schedulerProofFixture(plan)
	attempt := attemptOn("att-a", unit.ID, plan, domain.AttemptReconciled)
	addProvenAttemptCriterion(&proof, plan, attempt, retainedArtifactV1, "crit-a", time.Unix(100, 0).UTC())
	for i := range proof.Criteria {
		if proof.Criteria[i].Criterion.ID == "crit-a" {
			proof.Criteria[i].Delegated = true
		}
	}
	if attemptProven(unit, attempt, retainedArtifactV1, proof) {
		t.Fatal("a delegated criterion must not classify this attempt as succeeded")
	}
}

// The transition into succeeded exists only from reconciled. Success is never
// assigned from a live process, and a classified success is never walked back.
func TestOnlyAnEndedAttemptCanBecomeSucceeded(t *testing.T) {
	for _, from := range []domain.AttemptStatus{
		domain.AttemptQueued, domain.AttemptRunning, domain.AttemptPaused,
		domain.AttemptFailed, domain.AttemptCancelled, domain.AttemptLost,
	} {
		if domain.AttemptTransitionLegal(from, domain.AttemptSucceeded) {
			t.Fatalf("%s -> succeeded must be rejected", from)
		}
	}
	if !domain.AttemptTransitionLegal(domain.AttemptReconciled, domain.AttemptSucceeded) {
		t.Fatal("reconciled -> succeeded must be the one legal route")
	}
	if domain.AttemptTransitionLegal(domain.AttemptSucceeded, domain.AttemptReconciled) {
		t.Fatal("a classified success must not be walked back")
	}
}

// Proof about an Attempt is proof about the bytes it produced. If the output
// changes after it was checked, the same proof no longer describes the result,
// and the honest state is unproved -- not succeeded on a check of something
// that is gone.
func TestAttemptProvenRefusesProofRecordedAgainstAnEarlierArtifact(t *testing.T) {
	plan := schedulerPlanFixture()
	unit := plan.WorkUnits[0]
	proof := schedulerProofFixture(plan)
	attempt := attemptOn("att-a", unit.ID, plan, domain.AttemptReconciled)

	addProvenAttemptCriterion(&proof, plan, attempt, retainedArtifactV1, "crit-a", time.Unix(100, 0).UTC())
	if !attemptProven(unit, attempt, retainedArtifactV1, proof) {
		t.Fatal("proof recorded against the retained artifact must classify it")
	}

	// The workspace produced new bytes after the check ran.
	if attemptProven(unit, attempt, "artifact-v2", proof) {
		t.Fatal("proof that examined v1 must not classify v2 as succeeded")
	}

	// And once the new version is itself checked, it classifies again.
	addProvenAttemptCriterion(&proof, plan, attempt, "artifact-v2", "crit-a", time.Unix(200, 0).UTC())
	if !attemptProven(unit, attempt, "artifact-v2", proof) {
		t.Fatal("proof recorded against the new artifact must classify it")
	}
}

// An attempt with nothing retained has no artifact for proof to be about.
func TestAttemptProvenRequiresARetainedArtifactVersion(t *testing.T) {
	plan := schedulerPlanFixture()
	unit := plan.WorkUnits[0]
	proof := schedulerProofFixture(plan)
	attempt := attemptOn("att-a", unit.ID, plan, domain.AttemptReconciled)
	addProvenAttemptCriterion(&proof, plan, attempt, retainedArtifactV1, "crit-a", time.Unix(100, 0).UTC())

	if attemptProven(unit, attempt, "", proof) {
		t.Fatal("an attempt with no retained artifact must never classify as succeeded")
	}
	if attemptProven(unit, attempt, "   ", proof) {
		t.Fatal("a blank artifact version must never classify as succeeded")
	}
}

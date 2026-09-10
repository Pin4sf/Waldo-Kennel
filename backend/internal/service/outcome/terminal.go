package outcome

import (
	"context"
	"errors"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// Truthful terminal state for an Attempt.
//
// The sequence this implements is specified in
// docs/verification/2026-09-09-execution-to-admission-sequence.md. In short:
// execution ending is a runtime fact that produces `reconciled`, and success is
// derived from that fact plus the WorkUnit's proof. Nothing here reads provider
// prose, a transcript marker, or a process exit code as a result.

// ReconcileAttemptOutcomes promotes ended attempts whose proof is satisfied.
//
// This is the daemon-owned half of "provider completion is not success". Arrow
// one — running to reconciled — already happens in EvaluateAttemptLiveness when
// runtime facts show the session gone. This closes arrow four: a reconciled
// attempt whose WorkUnit criteria are proved against that exact attempt becomes
// `succeeded`.
//
// It is a restart-safe reconciliation of durable facts. The final transition
// is committed by the storage boundary as one conditional operation; it is
// not a pure function because it records an observation, freezes evidence and
// releases custody.
func (s *Service) ReconcileAttemptOutcomes(ctx context.Context) error {
	ended, err := s.store.ListAttemptsByStatus(ctx, domain.AttemptReconciled)
	if err != nil {
		return fmt.Errorf("list reconciled attempts: %w", err)
	}

	var failures []error
	// Proof and plan reads are per Outcome, so group to avoid re-reading them
	// once per attempt.
	for outcomeID, attempts := range groupAttemptsByOutcome(ended) {
		if err := s.reconcileOutcomeAttempts(ctx, outcomeID, attempts); err != nil {
			failures = append(failures, fmt.Errorf("outcome %s: %w", outcomeID, err))
		}
	}

	switch len(failures) {
	case 0:
		return nil
	case 1:
		return failures[0]
	default:
		return fmt.Errorf("attempt outcome reconciliation: %d failures: %w", len(failures), errors.Join(failures...))
	}
}

func (s *Service) reconcileOutcomeAttempts(ctx context.Context, outcomeID domain.OutcomeID, ended []domain.Attempt) error {
	proof, err := s.GetProof(ctx, outcomeID)
	if err != nil {
		return err
	}
	plans := map[domain.PlanRevisionID]domain.PlanRevision{}
	for _, attempt := range ended {
		plan, ok := plans[attempt.PlanRevisionID]
		if !ok {
			loaded, found, err := s.store.GetPlanRevision(ctx, outcomeID, attempt.PlanRevisionID)
			if err != nil {
				return err
			}
			if !found {
				// A missing plan is not a reason to guess a result.
				continue
			}
			plan, plans[attempt.PlanRevisionID] = loaded, loaded
		}
		unit, ok := planWorkUnit(plan, attempt.WorkUnitID)
		if !ok {
			continue
		}
		if !attemptProven(unit, attempt, proof) {
			// Absent, failing or contradictory proof leaves the attempt
			// reconciled. That is a truthful resting state, not a failure, and
			// the owner can see exactly which criteria are unmet.
			continue
		}
		if s.receipts == nil {
			return fmt.Errorf("attempt %s cannot be classified: receipt storage is unavailable", attempt.ID)
		}
		receipt, found, err := s.receipts.GetAttemptReceipt(ctx, attempt.ID)
		if err != nil {
			return fmt.Errorf("read receipt for %s: %w", attempt.ID, err)
		}
		if !found {
			return fmt.Errorf("attempt %s cannot be classified: %w", attempt.ID, ports.ErrAttemptReceiptMissing)
		}
		if !receipt.RetentionState.Complete() {
			return fmt.Errorf("attempt %s cannot be classified: %w", attempt.ID, ports.ErrAttemptReceiptNotReady)
		}
		if err := s.promoteAttemptToSucceeded(ctx, attempt); err != nil {
			return err
		}
	}
	return nil
}

// promoteAttemptToSucceeded records the transition and freezes the receipt the
// success was judged against.
//
// Freezing here is the point at which "what the owner reviewed" becomes stable:
// after this, a later retention pass over the same workspace cannot replace the
// manifest that success was assigned on.
func (s *Service) promoteAttemptToSucceeded(ctx context.Context, attempt domain.Attempt) error {
	if s.receipts == nil {
		return fmt.Errorf("receipt storage is unavailable")
	}
	receipt, found, err := s.receipts.GetAttemptReceipt(ctx, attempt.ID)
	if err != nil {
		return fmt.Errorf("read receipt for %s: %w", attempt.ID, err)
	}
	if !found {
		return fmt.Errorf("%w for %s", ports.ErrAttemptReceiptMissing, attempt.ID)
	}
	if !receipt.RetentionState.Complete() {
		return fmt.Errorf("%w for %s", ports.ErrAttemptReceiptNotReady, attempt.ID)
	}
	finalizer, ok := s.store.(ports.AttemptSuccessFinalizer)
	if !ok {
		return fmt.Errorf("attempt success finalizer is unavailable")
	}
	payload := mustJSON(map[string]any{
		"outcome": "work unit proved against this attempt; result classified as succeeded",
	})
	if err := finalizer.ClassifyAttemptSucceeded(ctx, ports.ClassifyAttemptInput{
		OutcomeID: attempt.OutcomeID, AttemptID: attempt.ID, ExpectedStatus: domain.AttemptReconciled,
		ArtifactVersion: receipt.ArtifactVersion, ObservationKind: domain.ObservationAttemptClassified,
		ObservationPayload: payload, At: s.clock(),
	}); err != nil {
		if errors.Is(err, ports.ErrAttemptClassificationStale) {
			// Another reconciler won the conditional transition. A subsequent
			// read will observe its durable result.
			return nil
		}
		return fmt.Errorf("classify attempt %s: %w", attempt.ID, err)
	}
	return nil
}

// attemptProven reports whether every criterion the WorkUnit carries is proved
// against THIS attempt.
//
// Scoping to the single attempt is deliberate. workUnitProven answers a
// different question — is this unit proved at all — and matches proof across
// every attempt of the unit, including WorkUnit-scoped proof that names no
// attempt. That is right for scheduling, but using it here would classify the
// wrong attempt: if a unit had one attempt that produced nothing and a later
// one that did the work, unit-level proof would mark both as succeeded.
//
// It deliberately takes no plan. Classification depends only on the unit's
// criteria and on proof naming this attempt, so passing a plan would imply the
// answer could vary with plan identity when it cannot.
func attemptProven(unit domain.WorkUnit, attempt domain.Attempt, proof ProofView) bool {
	if len(unit.CriterionIDs) == 0 {
		return false
	}
	for _, criterionID := range unit.CriterionIDs {
		criterion, ok := criterionProofForID(proof, criterionID)
		if !ok || criterion.Delegated {
			return false
		}
		scoped := CriterionProofView{Criterion: criterion.Criterion}
		for _, item := range criterion.Evidence {
			if proofFactNamesAttempt(attempt, item.SubjectType, item.SubjectID, item.SubjectRevision) {
				scoped.Evidence = append(scoped.Evidence, item)
			}
		}
		for _, run := range criterion.Verifications {
			if proofFactNamesAttempt(attempt, run.SubjectType, run.SubjectID, run.SubjectRevision) {
				scoped.Verifications = append(scoped.Verifications, run)
			}
		}
		// The same readiness evaluator Prove & Close uses, so a criterion
		// cannot be "ready enough" for success but not for the owner.
		if ready, _ := criterionReady(scoped, proof.ProofHorizon); !ready {
			return false
		}
	}
	return true
}

// proofFactNamesAttempt matches only proof bound to this exact attempt.
// WorkUnit-scoped proof names no attempt and therefore cannot say which one
// succeeded.
func proofFactNamesAttempt(attempt domain.Attempt, subjectType domain.ProofSubjectType, subjectID, subjectRevision string) bool {
	if subjectType != domain.ProofSubjectAttempt {
		return false
	}
	if subjectID != subjectRevision {
		return false
	}
	return subjectID == string(attempt.ID)
}

func planWorkUnit(plan domain.PlanRevision, id domain.WorkUnitID) (domain.WorkUnit, bool) {
	for _, unit := range plan.WorkUnits {
		if unit.ID == id {
			return unit, true
		}
	}
	return domain.WorkUnit{}, false
}

func groupAttemptsByOutcome(attempts []domain.Attempt) map[domain.OutcomeID][]domain.Attempt {
	grouped := make(map[domain.OutcomeID][]domain.Attempt)
	for _, attempt := range attempts {
		grouped[attempt.OutcomeID] = append(grouped[attempt.OutcomeID], attempt)
	}
	return grouped
}

package outcome

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	if s.receipts == nil {
		return fmt.Errorf("outcome %s cannot be classified: receipt storage is unavailable", outcomeID)
	}
	finalizer, ok := s.store.(ports.AttemptSuccessFinalizer)
	if !ok {
		return fmt.Errorf("attempt success finalizer is unavailable")
	}
	generation, proof, err := s.refreshProofSnapshot(ctx, outcomeID, finalizer)
	if err != nil {
		return err
	}
	plans := map[domain.PlanRevisionID]domain.PlanRevision{}
	var failures []error
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
		// Arrow two runs before arrow three: the artifact has to exist before
		// proof can be about it. Retaining only after judging proof would mean
		// the bytes classified as successful were produced after the check.
		if s.retainer != nil {
			if err := s.retainer.RetainAttempt(ctx, attempt); err != nil {
				// An unretainable workspace leaves this attempt reconciled. It
				// is not a reason to abandon its siblings.
				failures = append(failures, fmt.Errorf("retain result for %s: %w", attempt.ID, err))
				continue
			}
		}
		receipt, found, err := s.receipts.GetAttemptReceipt(ctx, attempt.ID)
		if err != nil {
			return fmt.Errorf("read receipt for %s: %w", attempt.ID, err)
		}
		if !found {
			failures = append(failures, fmt.Errorf("attempt %s cannot be classified: %w", attempt.ID, ports.ErrAttemptReceiptMissing))
			continue
		}
		if !receipt.RetentionState.Complete() {
			failures = append(failures, fmt.Errorf("attempt %s cannot be classified: %w", attempt.ID, ports.ErrAttemptReceiptNotReady))
			continue
		}
		// Approved checks run before proof is judged, and their results are
		// what proof is judged on. Process completion is not criterion proof;
		// something has to have actually checked the retained bytes.
		wrote, checkErr := s.runApprovedChecks(ctx, attempt, plan, unit, receipt, proof.Contract)
		if checkErr != nil {
			failures = append(failures, checkErr)
			continue
		}
		if wrote {
			// New proof rows moved the append-only generation, so the reads
			// this classification commits against have to be taken again.
			if generation, proof, err = s.refreshProofSnapshot(ctx, outcomeID, finalizer); err != nil {
				return err
			}
		}
		if !attemptProven(unit, attempt, receipt.ArtifactVersion, proof) {
			// Absent, failing or contradictory proof leaves the attempt
			// reconciled. So does proof that named an earlier artifact version:
			// the output changed after it was checked, so it is unproved again.
			continue
		}
		if err := s.promoteAttemptToSucceeded(ctx, attempt, receipt, generation); err != nil {
			return err
		}
	}
	if len(failures) == 1 {
		return failures[0]
	}
	if len(failures) > 1 {
		return errors.Join(failures...)
	}
	return nil
}

// refreshProofSnapshot reads the proof generation and then the proof.
//
// The order is the guarantee, not a detail. Classification commits against
// both, and the commit refuses if the generation moved. Reading the
// generation FIRST means any record that lands afterwards — including a
// contradicting one this snapshot cannot see — makes the observed generation
// stale, so the commit refuses. Reading the proof first inverts that: a
// record committing between the two reads is counted in the generation while
// missing from the snapshot, and the commit accepts evidence that no longer
// holds.
//
// The generation returned must therefore never be newer than the proof it
// accompanies.
func (s *Service) refreshProofSnapshot(
	ctx context.Context,
	outcomeID domain.OutcomeID,
	finalizer ports.AttemptSuccessFinalizer,
) (int64, ProofView, error) {
	generation, err := finalizer.OutcomeProofGeneration(ctx, outcomeID)
	if err != nil {
		return 0, ProofView{}, err
	}
	proof, err := s.GetProof(ctx, outcomeID)
	if err != nil {
		return 0, ProofView{}, err
	}
	return generation, proof, nil
}

// promoteAttemptToSucceeded records the transition and freezes the receipt the
// success was judged against.
//
// Freezing here is the point at which "what the owner reviewed" becomes stable:
// after this, a later retention pass over the same workspace cannot replace the
// manifest that success was assigned on.
func (s *Service) promoteAttemptToSucceeded(ctx context.Context, attempt domain.Attempt, receipt domain.AttemptReceipt, proofGeneration int64) error {
	// The receipt is the one proof was judged against, passed in rather than
	// read again. A third read could return a different artifact version than
	// the one the judgement used.
	finalizer, ok := s.store.(ports.AttemptSuccessFinalizer)
	if !ok {
		return fmt.Errorf("attempt success finalizer is unavailable")
	}
	payload := mustJSON(map[string]any{
		"outcome": "work unit proved against this attempt; result classified as succeeded",
	})
	if err := finalizer.ClassifyAttemptSucceeded(ctx, ports.ClassifyAttemptInput{
		OutcomeID: attempt.OutcomeID, AttemptID: attempt.ID, ExpectedStatus: domain.AttemptReconciled,
		ArtifactVersion: receipt.ArtifactVersion, ContractRevisionNumber: attempt.ContractRevisionNumber,
		ProofGeneration: &proofGeneration, ObservationKind: domain.ObservationAttemptClassified,
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
// against THIS attempt AND against the exact bytes it retained.
//
// The artifact version is the second half of the binding. An attempt id alone
// says a check ran on this attempt; it does not say the check ran on what the
// attempt now holds. Proof recorded against an earlier retained version leaves
// the attempt unproved once the output changes, which is the honest answer:
// nobody has checked the current result.
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
func attemptProven(unit domain.WorkUnit, attempt domain.Attempt, artifactVersion string, proof ProofView) bool {
	if strings.TrimSpace(artifactVersion) == "" {
		return false
	}
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
			if proofFactNamesAttempt(attempt, artifactVersion, item.SubjectType, item.SubjectID, item.SubjectRevision) {
				scoped.Evidence = append(scoped.Evidence, item)
			}
		}
		for _, run := range criterion.Verifications {
			if proofFactNamesAttempt(attempt, artifactVersion, run.SubjectType, run.SubjectID, run.SubjectRevision) {
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

// proofFactNamesAttempt matches only proof bound to this exact attempt and the
// exact artifact version it produced.
//
// For an attempt subject, SubjectID is the attempt and SubjectRevision is the
// retained artifact version the fact was recorded against -- the same shape
// every other subject type uses, where the revision identifies which version of
// the subject was examined. WorkUnit-scoped proof names no attempt and
// therefore cannot say which one succeeded.
func proofFactNamesAttempt(attempt domain.Attempt, artifactVersion string, subjectType domain.ProofSubjectType, subjectID, subjectRevision string) bool {
	if subjectType != domain.ProofSubjectAttempt {
		return false
	}
	if subjectID != string(attempt.ID) {
		return false
	}
	return subjectRevision == artifactVersion
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

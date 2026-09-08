package outcome

import (
	"context"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
)

// WorkUnitScheduleState is derived control-plane state. It is never persisted;
// canonical Plan, Attempt and proof facts remain the source of truth.
type WorkUnitScheduleState string

const (
	WorkUnitScheduleBlocked   WorkUnitScheduleState = "blocked"
	WorkUnitScheduleRunnable  WorkUnitScheduleState = "runnable"
	WorkUnitScheduleExecuting WorkUnitScheduleState = "executing"
	WorkUnitScheduleProven    WorkUnitScheduleState = "proven"
	WorkUnitScheduleRetryable WorkUnitScheduleState = "retryable"
)

// WorkUnitScheduleView is the Mission-Control-ready scheduler projection for
// one canonical WorkUnit.
type WorkUnitScheduleView struct {
	WorkUnit             domain.WorkUnit
	State                WorkUnitScheduleState
	Attempts             []domain.Attempt
	BlockingDependencies []domain.WorkUnitID
	CriterionReady       map[domain.CriterionID]bool
}

// ScheduleView is derived from one approved current Plan plus canonical proof.
type ScheduleView struct {
	Plan           domain.PlanRevision
	WorkUnits      []WorkUnitScheduleView
	NextRunnableID domain.WorkUnitID
	ActiveAttempt  *domain.Attempt
}

func attemptActiveForScheduling(status domain.AttemptStatus) bool {
	switch status {
	case domain.AttemptQueued, domain.AttemptRunning, domain.AttemptPaused:
		return true
	default:
		return false
	}
}

// nextRunnableWorkUnit is the pure serial scheduler decision. It uses the same
// criterionReady evaluator as Prove & Close after narrowing facts to the exact
// dependency WorkUnit/Attempt lineage.
func nextRunnableWorkUnit(plan domain.PlanRevision, attempts []domain.Attempt, proof ProofView) (domain.WorkUnit, bool, error) {
	if proof.OutcomeID != plan.OutcomeID {
		return domain.WorkUnit{}, false, fmt.Errorf("scheduler proof outcome %s does not match plan outcome %s", proof.OutcomeID, plan.OutcomeID)
	}
	if proof.Contract.Number != plan.ContractRevisionNumber {
		return domain.WorkUnit{}, false, fmt.Errorf("scheduler proof contract revision %d does not match plan revision binding %d", proof.Contract.Number, plan.ContractRevisionNumber)
	}
	ordered, err := plan.TopologicalWorkUnits()
	if err != nil {
		return domain.WorkUnit{}, false, err
	}
	currentAttempts := attemptsForPlan(plan, attempts)
	for _, attempt := range currentAttempts {
		if attemptActiveForScheduling(attempt.Status) {
			return domain.WorkUnit{}, false, nil
		}
	}

	proven := make(map[domain.WorkUnitID]bool, len(ordered))
	for _, unit := range ordered {
		proven[unit.ID] = workUnitProven(plan, unit, currentAttempts, proof)
	}
	for _, unit := range ordered {
		if proven[unit.ID] {
			continue
		}
		ready := true
		for _, dependency := range unit.DependsOn {
			if !proven[dependency] {
				ready = false
				break
			}
		}
		if ready {
			return unit, true, nil
		}
	}
	return domain.WorkUnit{}, false, nil
}

func attemptsForPlan(plan domain.PlanRevision, attempts []domain.Attempt) []domain.Attempt {
	out := make([]domain.Attempt, 0, len(attempts))
	for _, attempt := range attempts {
		if attempt.OutcomeID != plan.OutcomeID || attempt.PlanRevisionID != plan.ID || attempt.ContractRevisionNumber != plan.ContractRevisionNumber {
			continue
		}
		out = append(out, attempt)
	}
	return out
}

func attemptsForWorkUnit(unitID domain.WorkUnitID, attempts []domain.Attempt) []domain.Attempt {
	out := make([]domain.Attempt, 0)
	for _, attempt := range attempts {
		if attempt.WorkUnitID == unitID {
			out = append(out, attempt)
		}
	}
	return out
}

func workUnitProven(plan domain.PlanRevision, unit domain.WorkUnit, attempts []domain.Attempt, proof ProofView) bool {
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
			if proofFactBelongsToWorkUnit(plan, unit, attempts, item.SubjectType, item.SubjectID, item.SubjectRevision) {
				scoped.Evidence = append(scoped.Evidence, item)
			}
		}
		for _, run := range criterion.Verifications {
			if proofFactBelongsToWorkUnit(plan, unit, attempts, run.SubjectType, run.SubjectID, run.SubjectRevision) {
				scoped.Verifications = append(scoped.Verifications, run)
			}
		}
		ready, _ := criterionReady(scoped, proof.ProofHorizon)
		if !ready {
			return false
		}
	}
	return true
}

func criterionProofForID(proof ProofView, criterionID domain.CriterionID) (CriterionProofView, bool) {
	for _, criterion := range proof.Criteria {
		if criterion.Criterion.ID == criterionID {
			return criterion, true
		}
	}
	return CriterionProofView{}, false
}

func proofFactBelongsToWorkUnit(plan domain.PlanRevision, unit domain.WorkUnit, attempts []domain.Attempt, subjectType domain.ProofSubjectType, subjectID, subjectRevision string) bool {
	switch subjectType {
	case domain.ProofSubjectWorkUnit:
		return subjectID == string(unit.ID) && subjectRevision == string(plan.ID)
	case domain.ProofSubjectAttempt:
		if subjectID != subjectRevision {
			return false
		}
		for _, attempt := range attempts {
			if string(attempt.ID) == subjectID && attempt.WorkUnitID == unit.ID && attempt.PlanRevisionID == plan.ID && attempt.OutcomeID == plan.OutcomeID && attempt.ContractRevisionNumber == plan.ContractRevisionNumber {
				return true
			}
		}
	}
	return false
}

// GetSchedule returns derived scheduler truth for Mission Control. The Plan
// must be approved and bind the current Contract; no frontend lifecycle state
// is invented or persisted.
func (s *Service) GetSchedule(ctx context.Context, outcomeID domain.OutcomeID, planID domain.PlanRevisionID) (ScheduleView, error) {
	outcomeRecord, ok, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return ScheduleView{}, err
	}
	if !ok {
		return ScheduleView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	plan, found, err := s.store.GetPlanRevision(ctx, outcomeID, planID)
	if err != nil {
		return ScheduleView{}, err
	}
	if !found {
		return ScheduleView{}, apierr.NotFound("PLAN_NOT_FOUND", "That plan does not exist")
	}
	if plan.Status != domain.PlanStatusApproved {
		return ScheduleView{}, apierr.Conflict(CodePlanNotApproved, "Authorize this plan before scheduling work", map[string]any{"planId": plan.ID})
	}
	if !plan.BindsCurrentContract(outcomeRecord.CurrentRevisionNumber) {
		return ScheduleView{}, apierr.Conflict(CodePlanBriefInvalidated, "This plan no longer binds the current Contract", map[string]any{"planId": plan.ID})
	}
	attempts, err := s.store.ListAttempts(ctx, outcomeID)
	if err != nil {
		return ScheduleView{}, err
	}
	proof, err := s.GetProof(ctx, outcomeID)
	if err != nil {
		return ScheduleView{}, err
	}
	return deriveSchedule(plan, attempts, proof)
}

func deriveSchedule(plan domain.PlanRevision, attempts []domain.Attempt, proof ProofView) (ScheduleView, error) {
	ordered, err := plan.TopologicalWorkUnits()
	if err != nil {
		return ScheduleView{}, err
	}
	currentAttempts := attemptsForPlan(plan, attempts)
	view := ScheduleView{Plan: plan}
	var active *domain.Attempt
	for i := range currentAttempts {
		if attemptActiveForScheduling(currentAttempts[i].Status) {
			copy := currentAttempts[i]
			active = &copy
			break
		}
	}
	view.ActiveAttempt = active

	proven := make(map[domain.WorkUnitID]bool, len(ordered))
	for _, unit := range ordered {
		proven[unit.ID] = workUnitProven(plan, unit, currentAttempts, proof)
	}
	for _, unit := range ordered {
		entry := WorkUnitScheduleView{WorkUnit: unit, Attempts: attemptsForWorkUnit(unit.ID, currentAttempts), CriterionReady: map[domain.CriterionID]bool{}}
		for _, criterionID := range unit.CriterionIDs {
			criterion, ok := criterionProofForID(proof, criterionID)
			if !ok {
				entry.CriterionReady[criterionID] = false
				continue
			}
			scoped := CriterionProofView{Criterion: criterion.Criterion}
			for _, item := range criterion.Evidence {
				if proofFactBelongsToWorkUnit(plan, unit, currentAttempts, item.SubjectType, item.SubjectID, item.SubjectRevision) {
					scoped.Evidence = append(scoped.Evidence, item)
				}
			}
			for _, run := range criterion.Verifications {
				if proofFactBelongsToWorkUnit(plan, unit, currentAttempts, run.SubjectType, run.SubjectID, run.SubjectRevision) {
					scoped.Verifications = append(scoped.Verifications, run)
				}
			}
			ready, _ := criterionReady(scoped, proof.ProofHorizon)
			entry.CriterionReady[criterionID] = ready
		}
		switch {
		case proven[unit.ID]:
			entry.State = WorkUnitScheduleProven
		case active != nil && active.WorkUnitID == unit.ID:
			entry.State = WorkUnitScheduleExecuting
		default:
			for _, dependency := range unit.DependsOn {
				if !proven[dependency] {
					entry.BlockingDependencies = append(entry.BlockingDependencies, dependency)
				}
			}
			if len(entry.BlockingDependencies) > 0 || active != nil {
				entry.State = WorkUnitScheduleBlocked
			} else if len(entry.Attempts) > 0 {
				entry.State = WorkUnitScheduleRetryable
				if view.NextRunnableID.IsZero() {
					view.NextRunnableID = unit.ID
				}
			} else {
				entry.State = WorkUnitScheduleRunnable
				if view.NextRunnableID.IsZero() {
					view.NextRunnableID = unit.ID
				}
			}
		}
		view.WorkUnits = append(view.WorkUnits, entry)
	}
	return view, nil
}

// selectWorkUnitForAttempt enforces the serial scheduler decision for an
// explicit WorkUnit request. It never chooses a different unit as a fallback.
func (s *Service) selectWorkUnitForAttempt(ctx context.Context, outcomeID domain.OutcomeID, plan domain.PlanRevision, requested domain.WorkUnitID) (domain.WorkUnit, error) {
	if requested.IsZero() {
		return domain.WorkUnit{}, apierr.Invalid("WORK_UNIT_REQUIRED", "Choose the approved WorkUnit this Attempt executes", nil)
	}
	attempts, err := s.store.ListAttempts(ctx, outcomeID)
	if err != nil {
		return domain.WorkUnit{}, err
	}
	proof, err := s.GetProof(ctx, outcomeID)
	if err != nil {
		return domain.WorkUnit{}, err
	}
	next, ok, err := nextRunnableWorkUnit(plan, attempts, proof)
	if err != nil {
		return domain.WorkUnit{}, err
	}
	if !ok {
		return domain.WorkUnit{}, apierr.Conflict(CodeNoRunnableWorkUnit, "No WorkUnit is runnable until the active work or required proof is resolved", map[string]any{"planId": plan.ID})
	}
	if next.ID != requested {
		return domain.WorkUnit{}, apierr.Conflict("WORK_UNIT_NOT_RUNNABLE", "That WorkUnit is not the next dependency-ready unit", map[string]any{"requestedWorkUnitId": requested, "nextRunnableWorkUnitId": next.ID})
	}
	return next, nil
}

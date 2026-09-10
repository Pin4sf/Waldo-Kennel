package outcome

import (
	"context"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// MissionState is the owner-facing milestone of one Outcome.
//
// It is derived on every read from Contract, Plan, Attempt and proof facts and
// is never stored. Persisting it would create a second lifecycle that could
// disagree with the facts it summarises — which is exactly the failure the
// Board is meant to prevent.
type MissionState string

// The six milestones the Board and Mission header group by.
const (
	// MissionDefine means there is no proposed Plan to look at yet.
	MissionDefine MissionState = "define"
	// MissionReadyToAuthorize means a Plan is proposed and awaits the owner.
	MissionReadyToAuthorize MissionState = "ready_to_authorize"
	// MissionInProgress means governed execution is under way.
	MissionInProgress MissionState = "in_progress"
	// MissionNeedsYou means nothing can proceed without an owner decision.
	// It always carries a concrete reason; a bare "needs you" is a bug.
	MissionNeedsYou MissionState = "needs_you"
	// MissionReadyForReview means proof supports acceptance and the owner has
	// not decided. Ready for review is never acceptance.
	MissionReadyForReview MissionState = "ready_for_review"
	// MissionAccepted means the owner accepted this Outcome.
	MissionAccepted MissionState = "accepted"
)

// RunAction is the fixed vocabulary of owner moves a Mission can offer.
// Rendering is the renderer's business; eligibility is the daemon's.
type RunAction string

// The complete action vocabulary. Authorization and execution are deliberately
// two actions, because approving a Plan and starting work are two decisions.
const (
	RunActionClarify        RunAction = "clarify"
	RunActionProposePlan    RunAction = "propose_plan"
	RunActionReviewPlan     RunAction = "review_plan"
	RunActionApprovePlan    RunAction = "approve_plan"
	RunActionStart          RunAction = "start"
	RunActionPause          RunAction = "pause"
	RunActionResume         RunAction = "resume"
	RunActionCancel         RunAction = "cancel"
	RunActionReviewResult   RunAction = "review_result"
	RunActionRequestChanges RunAction = "request_changes"
	RunActionAccept         RunAction = "accept"
	RunActionExport         RunAction = "export"
)

// Stable reason codes for an unavailable action or a blocked Mission. These
// are policy identity, not display copy: the renderer owns the wording.
const (
	ReasonNoPlan               = "no_plan_proposed"
	ReasonPlanAlreadyProposed  = "plan_already_proposed"
	ReasonPlanNotProposed      = "plan_not_proposed"
	ReasonPlanNotApproved      = "plan_not_approved"
	ReasonPlanAlreadyApproved  = "plan_already_approved"
	ReasonPlanStale            = "plan_no_longer_binds_current_contract"
	ReasonAttemptActive        = "attempt_active"
	ReasonAttemptPaused        = "attempt_paused"
	ReasonNoActiveRun          = "no_active_run"
	ReasonNothingRunnable      = "nothing_runnable"
	ReasonStartRequired        = "start_required"
	ReasonProofIncomplete      = "proof_incomplete"
	ReasonReworkRequired       = "rework_required"
	ReasonAlreadyAccepted      = "already_accepted"
	ReasonContributionBlocked  = "contribution_blocked"
	ReasonAttemptNeedsRecovery = "attempt_needs_recovery"
	ReasonRunIntentUnavailable = "run_intent_unavailable"
	ReasonDeliveryUnavailable  = "delivery_unavailable"
	ReasonNotAccepted          = "not_accepted"
)

// RunActionEligibility is one action and whether the daemon will honour it.
// Reason is set exactly when Available is false.
type RunActionEligibility struct {
	Action    RunAction
	Available bool
	Reason    string
}

// RunIntentView is the persisted authorization to keep running a Plan. It is
// nil until durable run intent exists; a nil intent is a truthful answer, not
// an error, and means only per-Attempt Start is available.
type RunIntentView struct {
	Generation      int64
	Desired         string
	PlanRevisionID  domain.PlanRevisionID
	RequestedAt     time.Time
	AcknowledgedAt  *time.Time
	ActiveAttemptID domain.AttemptID
	LastError       string
}

// RunBlocker is the single concrete reason an Outcome needs its owner.
type RunBlocker struct {
	Code    string
	Message string
	Detail  map[string]any
}

// RunFreshness lets a client discard a response older than state it already
// holds. ProofGeneration is the append-only proof record count the projection
// was computed from.
type RunFreshness struct {
	ObservedAt             time.Time
	ContractRevisionNumber int64
	PlanRevisionID         domain.PlanRevisionID
	ProofGeneration        int64
}

// RunStateView is the Mission header projection: where this Outcome stands,
// what the owner may do next, and what is in the way.
type RunStateView struct {
	OutcomeID       domain.OutcomeID
	ProjectID       domain.ProjectID
	Title           string
	ParentOutcomeID domain.OutcomeID
	State           MissionState
	// AttentionReason is non-empty exactly when State is MissionNeedsYou.
	AttentionReason string
	Intent          *RunIntentView
	EligibleActions []RunActionEligibility
	Blocker         *RunBlocker
	Freshness       RunFreshness
	PlanStatus      domain.PlanStatus
	// PlanBindsCurrentContract is false for the stale-Plan case, where the
	// Contract moved past the Plan that was approved against it.
	PlanBindsCurrentContract bool
	ActiveAttemptID          domain.AttemptID
	ActiveAttemptStatus      domain.AttemptStatus
	ProvenCriteria           int
	RequiredCriteria         int
	AcceptedAt               *time.Time
}

// GetRunState derives one Outcome's Mission state and eligible actions.
//
// Everything here is composed from the canonical services that already own
// each fact — Contract, Plan, schedule, proof, contribution gating — so there
// is exactly one authority per fact and this projection can never disagree
// with the screen the owner drills into.
func (s *Service) GetRunState(ctx context.Context, outcomeID domain.OutcomeID) (RunStateView, error) {
	record, ok, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return RunStateView{}, err
	}
	if !ok {
		return RunStateView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	projectID, _, err := s.store.GetOutcomeProjectID(ctx, outcomeID)
	if err != nil {
		return RunStateView{}, err
	}
	return s.runStateFor(ctx, record, projectID)
}

// ListProjectRunStates derives the Board projection for one Project in a
// single call, so a Board does not need one round trip per card.
//
// topLevelOnly excludes contributing Outcomes, which belong inside their
// parent's Mission rather than beside it on the Board.
func (s *Service) ListProjectRunStates(ctx context.Context, projectID domain.ProjectID, topLevelOnly bool) ([]RunStateView, error) {
	records, err := s.store.ListOutcomesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	states := make([]RunStateView, 0, len(records))
	for _, record := range records {
		if topLevelOnly && record.IsContributing() {
			continue
		}
		state, err := s.runStateFor(ctx, record, projectID)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, nil
}

func (s *Service) runStateFor(ctx context.Context, record domain.Outcome, projectID domain.ProjectID) (RunStateView, error) {
	view := RunStateView{
		OutcomeID: record.ID, ProjectID: projectID, Title: record.Title, ParentOutcomeID: record.ParentID,
		Freshness: RunFreshness{ObservedAt: s.clock(), ContractRevisionNumber: record.CurrentRevisionNumber},
	}

	plan, planFound, err := s.store.GetLatestPlanRevision(ctx, record.ID)
	if err != nil {
		return RunStateView{}, err
	}
	if planFound {
		view.PlanStatus = plan.Status
		view.PlanBindsCurrentContract = plan.BindsCurrentContract(record.CurrentRevisionNumber)
		view.Freshness.PlanRevisionID = plan.ID
	}

	// Proof is not optional here. A Mission state derived without it would
	// have to guess whether work is proved, and guessing is exactly what this
	// projection exists to stop — so an unreadable proof store is an error,
	// not a quietly degraded card.
	proof, err := s.GetProof(ctx, record.ID)
	if err != nil {
		return RunStateView{}, err
	}
	view.RequiredCriteria, view.ProvenCriteria = criterionCounts(proof)
	if generation, ok := s.proofGeneration(ctx, record.ID); ok {
		view.Freshness.ProofGeneration = generation
	}
	if accepted := latestAcceptance(proof); accepted != nil {
		view.AcceptedAt = accepted
	}

	attempts, err := s.store.ListAttempts(ctx, record.ID)
	if err != nil {
		return RunStateView{}, err
	}
	active := activeAttempt(attempts)
	if active != nil {
		view.ActiveAttemptID, view.ActiveAttemptStatus = active.ID, active.Status
	}

	gate, err := s.startGateFor(ctx, record)
	if err != nil {
		return RunStateView{}, err
	}
	intent, hasIntent, err := s.currentRunIntent(ctx, record.ID)
	if err != nil {
		return RunStateView{}, err
	}
	if hasIntent {
		view.Intent = &RunIntentView{
			Generation: intent.Generation, Desired: string(intent.Desired),
			PlanRevisionID: intent.PlanRevisionID, RequestedAt: intent.RequestedAt,
			AcknowledgedAt: intent.AcknowledgedAt, ActiveAttemptID: view.ActiveAttemptID,
		}
	}

	var schedule *ScheduleView
	if planFound && plan.Status == domain.PlanStatusApproved && view.PlanBindsCurrentContract {
		derived, scheduleErr := s.GetSchedule(ctx, record.ID, plan.ID)
		if scheduleErr == nil {
			schedule = &derived
		}
	}

	view.State, view.AttentionReason, view.Blocker = deriveMissionState(missionInputs{
		planFound: planFound, plan: plan, planBinds: view.PlanBindsCurrentContract,
		proof: proof, gate: gate, active: active, schedule: schedule,
	})
	view.EligibleActions = deriveEligibleActions(view, missionInputs{
		planFound: planFound, plan: plan, planBinds: view.PlanBindsCurrentContract,
		proof: proof, gate: gate, active: active, schedule: schedule,
	})
	return view, nil
}

// missionInputs is the exact fact set the two derivations read, so the state
// and the offered actions can never be computed from different snapshots.
type missionInputs struct {
	planFound bool
	plan      domain.PlanRevision
	planBinds bool
	proof     ProofView
	gate      domain.ContributionStartGate
	active    *domain.Attempt
	schedule  *ScheduleView
}

func deriveMissionState(in missionInputs) (MissionState, string, *RunBlocker) {
	if in.proof.Status == ProofStatusAccepted {
		return MissionAccepted, "", nil
	}
	if !in.gate.Clear() {
		return MissionNeedsYou, ReasonContributionBlocked, &RunBlocker{
			Code:    ReasonContributionBlocked,
			Message: "This contribution waits on an upstream contribution to be accepted or waived",
		}
	}
	if !in.planFound {
		return MissionDefine, "", nil
	}
	if in.plan.Status != domain.PlanStatusApproved {
		return MissionReadyToAuthorize, "", nil
	}
	if !in.planBinds {
		return MissionNeedsYou, ReasonPlanStale, &RunBlocker{
			Code:    ReasonPlanStale,
			Message: "The Contract moved past the approved Plan — propose and approve a fresh Plan",
		}
	}
	if in.active != nil {
		switch in.active.Status {
		case domain.AttemptQueued, domain.AttemptRunning:
			return MissionInProgress, "", nil
		case domain.AttemptPaused:
			return MissionNeedsYou, ReasonAttemptPaused, &RunBlocker{
				Code:    ReasonAttemptPaused,
				Message: "A paused Attempt still holds workspace custody — resume or cancel it",
				Detail:  map[string]any{"attemptId": string(in.active.ID)},
			}
		}
	}
	switch in.proof.Status {
	case ProofStatusReadyForAcceptance:
		return MissionReadyForReview, "", nil
	case ProofStatusReworkRequired:
		return MissionNeedsYou, ReasonReworkRequired, &RunBlocker{
			Code:    ReasonReworkRequired,
			Message: "Recorded proof does not support acceptance — request changes or repair the evidence",
		}
	}
	if in.schedule != nil && !in.schedule.NextRunnableID.IsZero() {
		// Something can start and nothing is running. Without durable run
		// intent the owner is the only thing that can start it, so this is
		// honestly an owner decision rather than progress.
		return MissionNeedsYou, ReasonStartRequired, &RunBlocker{
			Code:    ReasonStartRequired,
			Message: "The next WorkUnit is ready to start",
			Detail:  map[string]any{"workUnitId": string(in.schedule.NextRunnableID)},
		}
	}
	if in.schedule != nil && in.schedule.NoRunnableReason != "" {
		return MissionNeedsYou, string(in.schedule.NoRunnableReason), &RunBlocker{
			Code:    string(in.schedule.NoRunnableReason),
			Message: "Nothing can be admitted right now",
		}
	}
	return MissionNeedsYou, ReasonProofIncomplete, &RunBlocker{
		Code:    ReasonProofIncomplete,
		Message: "Proof for this Outcome is incomplete",
	}
}

func deriveEligibleActions(view RunStateView, in missionInputs) []RunActionEligibility {
	planApproved := in.planFound && in.plan.Status == domain.PlanStatusApproved && in.planBinds
	executing := in.active != nil && (in.active.Status == domain.AttemptQueued || in.active.Status == domain.AttemptRunning)
	paused := in.active != nil && in.active.Status == domain.AttemptPaused
	accepted := in.proof.Status == ProofStatusAccepted
	runnable := in.schedule != nil && !in.schedule.NextRunnableID.IsZero()

	allow := func(action RunAction, ok bool, reason string) RunActionEligibility {
		if ok {
			return RunActionEligibility{Action: action, Available: true}
		}
		return RunActionEligibility{Action: action, Available: false, Reason: reason}
	}

	// Clarification stays open for as long as the Outcome is open: an owner
	// question is never the thing that has to wait.
	clarify := allow(RunActionClarify, !accepted, ReasonAlreadyAccepted)

	proposeReason := ReasonPlanAlreadyProposed
	if accepted {
		proposeReason = ReasonAlreadyAccepted
	}
	propose := allow(RunActionProposePlan, !accepted && (!in.planFound || !in.planBinds), proposeReason)

	review := allow(RunActionReviewPlan, in.planFound, ReasonNoPlan)

	approveReason := ReasonPlanNotProposed
	if in.planFound && in.plan.Status == domain.PlanStatusApproved {
		approveReason = ReasonPlanAlreadyApproved
	}
	approve := allow(RunActionApprovePlan, in.planFound && in.plan.Status == domain.PlanStatusProposed && in.gate.Clear(), approveReason)

	startReason := ReasonPlanNotApproved
	switch {
	case executing:
		startReason = ReasonAttemptActive
	case paused:
		startReason = ReasonAttemptPaused
	case !in.gate.Clear():
		startReason = ReasonContributionBlocked
	case planApproved && !runnable:
		startReason = ReasonNothingRunnable
	case in.planFound && !in.planBinds:
		startReason = ReasonPlanStale
	}
	start := allow(RunActionStart, planApproved && runnable && !executing && !paused && in.gate.Clear(), startReason)

	// Pause and Resume need durable run intent to mean anything: without it
	// there is no authorization to suspend or reinstate. Say so rather than
	// offering a control that would silently do nothing.
	pause := allow(RunActionPause, false, ReasonRunIntentUnavailable)
	resume := allow(RunActionResume, false, ReasonRunIntentUnavailable)
	if view.Intent != nil {
		pause = allow(RunActionPause, view.Intent.Desired == string(domain.RunIntentRunning), ReasonNoActiveRun)
		resume = allow(RunActionResume, view.Intent.Desired == string(domain.RunIntentPaused), ReasonNoActiveRun)
	}

	cancel := allow(RunActionCancel, executing || paused, ReasonNoActiveRun)

	hasResult := view.ProvenCriteria > 0 || in.active != nil
	reviewResult := allow(RunActionReviewResult, hasResult, ReasonProofIncomplete)

	requestChanges := allow(RunActionRequestChanges, hasResult && !accepted, ReasonProofIncomplete)
	if accepted {
		requestChanges = allow(RunActionRequestChanges, false, ReasonAlreadyAccepted)
	}

	acceptReason := ReasonProofIncomplete
	if accepted {
		acceptReason = ReasonAlreadyAccepted
	}
	accept := allow(RunActionAccept, in.proof.Status == ProofStatusReadyForAcceptance, acceptReason)

	// Delivery is a separate slice; until it exists an accepted Outcome still
	// cannot be exported, and pretending otherwise would be the one thing an
	// owner would most reasonably act on.
	export := allow(RunActionExport, false, ReasonDeliveryUnavailable)

	return []RunActionEligibility{
		clarify, propose, review, approve, start, pause, resume, cancel,
		reviewResult, requestChanges, accept, export,
	}
}

func criterionCounts(proof ProofView) (required, proven int) {
	for _, criterion := range proof.Criteria {
		if criterion.Delegated {
			continue
		}
		required++
		if criterion.Ready {
			proven++
		}
	}
	return required, proven
}

func latestAcceptance(proof ProofView) *time.Time {
	if !domain.LatestDecisionAccepts(proof.Decisions) {
		return nil
	}
	var latest *domain.AcceptanceDecision
	for i := range proof.Decisions {
		decision := proof.Decisions[i]
		if latest == nil || decision.CreatedAt.After(latest.CreatedAt) {
			latest = &decision
		}
	}
	if latest == nil {
		return nil
	}
	at := latest.CreatedAt
	return &at
}

func activeAttempt(attempts []domain.Attempt) *domain.Attempt {
	var latest *domain.Attempt
	for i := range attempts {
		attempt := attempts[i]
		if !attemptActiveForScheduling(attempt.Status) {
			continue
		}
		if latest == nil || attempt.CreatedAt.After(latest.CreatedAt) {
			latest = &attempt
		}
	}
	return latest
}

// proofGeneration reads the append-only proof record count when the store
// supports it. A store that cannot report one yields no generation rather than
// a fabricated zero-that-looks-current.
func (s *Service) proofGeneration(ctx context.Context, outcomeID domain.OutcomeID) (int64, bool) {
	finalizer, ok := s.store.(ports.AttemptSuccessFinalizer)
	if !ok {
		return 0, false
	}
	generation, err := finalizer.OutcomeProofGeneration(ctx, outcomeID)
	if err != nil {
		return 0, false
	}
	return generation, true
}

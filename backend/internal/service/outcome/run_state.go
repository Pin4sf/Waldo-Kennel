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
	// ReasonRunAlreadyAuthorized means durable run intent already authorizes
	// continuation, so Start has nothing left to authorize. CommandRun refuses
	// a second Start for exactly this reason.
	ReasonRunAlreadyAuthorized = "run_already_authorized"
	// ReasonRunPaused means the owner paused this run. The next move is Resume
	// — reporting "start required" here would name a different command.
	ReasonRunPaused = "run_paused"
	// ReasonRunIntentPlanSuperseded means the authorized run names a Plan
	// revision the Outcome has moved past, so continuation can admit nothing.
	// Cancel and re-authorize is the only way forward.
	ReasonRunIntentPlanSuperseded = "run_intent_plan_superseded"
	// ReasonPlanRevisionRequired means the owner's recorded correction names
	// the Plan itself. Re-running the same approved Plan would reproduce the
	// result they rejected.
	ReasonPlanRevisionRequired = "plan_revision_required"
	// ReasonContractRevisionRequired means the correction names the Contract.
	// Nothing below it can be revised while the agreement is what is wrong.
	ReasonContractRevisionRequired = "contract_revision_required"
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
	Generation     int64
	Desired        string
	PlanRevisionID domain.PlanRevisionID
	// BindsCurrentPlan is false when the authorization names a Plan revision
	// the Outcome has since moved past. Continuation schedules against the
	// Plan the intent names, so a superseded one admits nothing.
	BindsCurrentPlan bool
	RequestedAt      time.Time
	AcknowledgedAt   *time.Time
	ActiveAttemptID  domain.AttemptID
	LastError        string
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
	var currentIntent *domain.OutcomeRunIntent
	if hasIntent {
		held := intent
		currentIntent = &held
		view.Intent = &RunIntentView{
			Generation: intent.Generation, Desired: string(intent.Desired),
			PlanRevisionID:   intent.PlanRevisionID,
			BindsCurrentPlan: planFound && intent.PlanRevisionID == plan.ID,
			RequestedAt:      intent.RequestedAt,
			AcknowledgedAt:   intent.AcknowledgedAt, ActiveAttemptID: view.ActiveAttemptID,
		}
	}

	var schedule *ScheduleView
	if planFound && plan.Status == domain.PlanStatusApproved && view.PlanBindsCurrentContract {
		derived, scheduleErr := s.GetSchedule(ctx, record.ID, plan.ID)
		if scheduleErr == nil {
			schedule = &derived
		}
	}

	inputs := missionInputs{
		planFound: planFound, plan: plan, planBinds: view.PlanBindsCurrentContract,
		proof: proof, gate: gate, active: active, schedule: schedule,
		intent: currentIntent, runIntentsEnabled: s.RunIntentsEnabled(),
	}
	view.State, view.AttentionReason, view.Blocker = deriveMissionState(inputs)
	view.EligibleActions = deriveEligibleActions(view, inputs)
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
	// intent is the durable authorization to keep running. Nil means none has
	// been recorded, which reads as idle rather than as an error.
	intent *domain.OutcomeRunIntent
	// runIntentsEnabled distinguishes a daemon that cannot hold run intent
	// from one that simply has none yet. The two refuse Pause for different
	// reasons and only the first is a wiring fault.
	runIntentsEnabled bool
}

// desiredRun is the authorization the daemon will act on. No recorded intent
// is idle: an Outcome nobody has started yet.
func (in missionInputs) desiredRun() domain.RunIntentDesired {
	if in.intent == nil {
		return domain.RunIntentIdle
	}
	return in.intent.Desired
}

// commandApplies asks the same transition policy CommandRun asks. Routing both
// through domain.NextRunIntent is what stops the Mission offering a move the
// command endpoint rejects — the KUX-002 contradiction.
func (in missionInputs) commandApplies(command domain.RunCommand) bool {
	_, ok := domain.NextRunIntent(in.desiredRun(), command)
	return ok
}

func (in missionInputs) correctionRequires() domain.ReentryTargetType {
	return correctionRequiringRevision(in.proof, in.planFound, in.plan.ID)
}

// correctionRequiringRevision names the lineage seam the owner's standing
// correction says must change before execution may be authorized again, when
// that seam is above the WorkUnit. A correction targeting an Attempt or
// WorkUnit is satisfied by running the approved Plan again, so it names nothing.
//
// This is the single source for both the Mission's offered moves and the
// admission that honours them. Deriving it twice is how an authority boundary
// ends up living in a client: the projection refuses Start while the daemon
// accepts it.
//
// Corrections against a superseded Contract revision are already filtered out
// of the proof, and a Plan correction stops naming the current Plan as soon as
// a fresh one is proposed — so both resolve themselves rather than needing a
// separate "correction addressed" record.
func correctionRequiringRevision(proof ProofView, planFound bool, planID domain.PlanRevisionID) domain.ReentryTargetType {
	correction := proof.ActiveCorrection
	if correction == nil {
		return ""
	}
	switch correction.TargetType {
	case domain.ReentryTargetContract:
		return domain.ReentryTargetContract
	case domain.ReentryTargetPlan:
		if planFound && correction.TargetID == string(planID) {
			return domain.ReentryTargetPlan
		}
	}
	return ""
}

// correctionRefusalReason maps a blocking correction target to the stable
// eligibility reason clients already branch on, so the refused command and the
// refused action name the same policy.
func correctionRefusalReason(target domain.ReentryTargetType) string {
	if target == domain.ReentryTargetContract {
		return ReasonContractRevisionRequired
	}
	return ReasonPlanRevisionRequired
}

// refuseExecutionAgainstCorrection is the admission half of the correction
// policy. Every path that authorizes or admits execution runs it, because a
// boundary only one route honours is a boundary with a way around it.
func (s *Service) refuseExecutionAgainstCorrection(ctx context.Context, outcomeID domain.OutcomeID) error {
	if s.proof == nil {
		return nil
	}
	proof, err := s.GetProof(ctx, outcomeID)
	if err != nil {
		return err
	}
	if proof.ActiveCorrection == nil {
		// The overwhelmingly common case, and the only Plan read is avoided.
		return nil
	}
	plan, planFound, err := s.store.GetLatestPlanRevision(ctx, outcomeID)
	if err != nil {
		return err
	}
	target := correctionRequiringRevision(proof, planFound, plan.ID)
	if target == "" {
		return nil
	}
	message := "The owner's correction names this Outcome's Plan — propose and approve a revised Plan before authorizing work"
	if target == domain.ReentryTargetContract {
		message = "The owner's correction names this Outcome's Contract — revise and confirm the Contract before authorizing work"
	}
	return apierr.Conflict(CodeCorrectionRevisionRequired, message, map[string]any{
		"outcomeId":  string(outcomeID),
		"targetType": string(target),
		"targetId":   proof.ActiveCorrection.TargetID,
		"decisionId": string(proof.ActiveCorrection.DecisionID),
		"reason":     correctionRefusalReason(target),
	})
}

// continuationAdmits reports whether the recorded authorization can still put
// work in flight without the owner. ContinueAuthorizedRuns schedules against
// the Plan the intent names, so an intent naming a superseded Plan admits
// nothing however runnable the current Plan looks.
func (in missionInputs) continuationAdmits() bool {
	if in.desiredRun() != domain.RunIntentRunning || !in.planFound {
		return false
	}
	return in.intent.PlanRevisionID == in.plan.ID
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
			Detail:  correctionDetail(in.proof.ActiveCorrection),
		}
	}
	// Nothing is in flight. What happens next is decided by the durable
	// authorization, not by the schedule alone: an authorized run continues
	// without the owner, and only an unauthorized one is their move.
	switch in.desiredRun() {
	case domain.RunIntentRunning:
		if !in.continuationAdmits() {
			return MissionNeedsYou, ReasonRunIntentPlanSuperseded, &RunBlocker{
				Code:    ReasonRunIntentPlanSuperseded,
				Message: "The authorized run names a Plan this Outcome has moved past — cancel it and authorize the current Plan",
				Detail:  map[string]any{"authorizedPlanId": string(in.intent.PlanRevisionID)},
			}
		}
		if in.schedule != nil && !in.schedule.NextRunnableID.IsZero() {
			// The owner already authorized continuation and the daemon admits
			// the next unit itself. Asking for another Start here would offer a
			// command CommandRun refuses.
			return MissionInProgress, "", nil
		}
	case domain.RunIntentPaused:
		return MissionNeedsYou, ReasonRunPaused, &RunBlocker{
			Code:    ReasonRunPaused,
			Message: "This run is paused — resume it to admit further work, or cancel it",
			Detail:  map[string]any{"generation": in.intent.Generation},
		}
	}
	if in.schedule != nil && !in.schedule.NextRunnableID.IsZero() {
		// Something can start and no authorization is in force, so the owner is
		// the only thing that can start it. That is honestly an owner decision
		// rather than progress.
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
	// A correction naming the current Plan is a request for a different Plan,
	// so proposing one is the move — even though a binding approved Plan
	// already exists, which is otherwise the reason to refuse.
	planCorrected := in.correctionRequires() == domain.ReentryTargetPlan
	propose := allow(RunActionProposePlan, !accepted && (!in.planFound || !in.planBinds || planCorrected), proposeReason)

	review := allow(RunActionReviewPlan, in.planFound, ReasonNoPlan)

	approveReason := ReasonPlanNotProposed
	if in.planFound && in.plan.Status == domain.PlanStatusApproved {
		approveReason = ReasonPlanAlreadyApproved
	}
	approve := allow(RunActionApprovePlan, in.planFound && in.plan.Status == domain.PlanStatusProposed && in.gate.Clear(), approveReason)

	// Start, Pause, Resume and Cancel are run-intent commands, so their
	// eligibility is asked of the same transition policy CommandRun applies.
	// An Attempt of unknown status also refuses Start, but only a queued or
	// running Attempt can be unaccountable, and that is already the
	// attempt_active case below.
	startApplies := in.commandApplies(domain.RunCommandStart)
	// An owner correction naming the Plan or the Contract is not satisfied by
	// running the same approved Plan again: doing so would reproduce the result
	// they rejected. Only corrections at or below the WorkUnit are.
	correctionTarget := in.correctionRequires()
	startReason := ReasonPlanNotApproved
	switch {
	case executing:
		startReason = ReasonAttemptActive
	case paused:
		startReason = ReasonAttemptPaused
	case !in.gate.Clear():
		startReason = ReasonContributionBlocked
	case !startApplies:
		startReason = ReasonRunAlreadyAuthorized
	case correctionTarget == domain.ReentryTargetContract:
		startReason = ReasonContractRevisionRequired
	case correctionTarget == domain.ReentryTargetPlan:
		startReason = ReasonPlanRevisionRequired
	case planApproved && !runnable:
		startReason = ReasonNothingRunnable
	case in.planFound && !in.planBinds:
		startReason = ReasonPlanStale
	}
	start := allow(RunActionStart,
		planApproved && runnable && !executing && !paused && in.gate.Clear() &&
			startApplies && correctionTarget == "", startReason)

	// Pause and Resume need durable run intent to mean anything: without the
	// storage there is no authorization to suspend or reinstate. Say so rather
	// than offering a control that would silently do nothing.
	pause := allow(RunActionPause, false, ReasonRunIntentUnavailable)
	resume := allow(RunActionResume, false, ReasonRunIntentUnavailable)
	cancel := allow(RunActionCancel, executing || paused, ReasonNoActiveRun)
	if in.runIntentsEnabled {
		pause = allow(RunActionPause, in.commandApplies(domain.RunCommandPause), ReasonNoActiveRun)
		// Resuming re-authorizes execution, so CommandRun re-validates the Plan
		// exactly as Start does. Offering Resume against a stale Plan would
		// name a command that is about to be refused.
		resumeReason := ReasonNoActiveRun
		if in.commandApplies(domain.RunCommandResume) && !planApproved {
			resumeReason = ReasonPlanStale
			if !in.planFound || in.plan.Status != domain.PlanStatusApproved {
				resumeReason = ReasonPlanNotApproved
			}
		}
		resume = allow(RunActionResume, in.commandApplies(domain.RunCommandResume) && planApproved, resumeReason)
		// Cancel stays available for a live Attempt even with no intent, because
		// CancelAttempt is its own path; an authorized run with nothing yet in
		// flight must also be stoppable, or the owner cannot undo a Start.
		cancel = allow(RunActionCancel, executing || paused || in.commandApplies(domain.RunCommandCancel), ReasonNoActiveRun)
	}

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

// correctionDetail carries what the owner asked for into the blocker, so a
// Mission saying "rework required" can also say what has to change and in
// their own words. Nil correction yields nil detail rather than empty keys.
func correctionDetail(correction *domain.OutcomeCorrection) map[string]any {
	if correction == nil {
		return nil
	}
	detail := map[string]any{
		"decisionId": string(correction.DecisionID),
		"targetType": string(correction.TargetType),
		"feedback":   correction.Feedback,
	}
	if correction.TargetID != "" {
		detail["targetId"] = correction.TargetID
	}
	return detail
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

package outcome

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// Stable refusals for owner run commands.
const (
	// CodeRunIntentUnavailable means this daemon has no durable run intent.
	CodeRunIntentUnavailable = "RUN_INTENT_UNAVAILABLE"
	// CodeRunActionInvalid means the command is not one of the four.
	CodeRunActionInvalid = "RUN_ACTION_INVALID"
	// CodeRunIntentStale means a newer generation exists, so this command was
	// composed against a state the owner has already moved past.
	CodeRunIntentStale = "RUN_INTENT_STALE"
	// CodeRunActionUnavailable means the command does not apply to the
	// current state — resuming a cancelled run, pausing an idle one.
	CodeRunActionUnavailable = "RUN_ACTION_UNAVAILABLE"
	// CodeRunCustodyUnknown means execution of unknown status survives, so
	// authorizing more work could duplicate it.
	CodeRunCustodyUnknown = "RUN_CUSTODY_UNKNOWN"
	// CodeCorrectionRevisionRequired means the owner's standing correction
	// names the Plan or the Contract, so running the same approved Plan again
	// would reproduce the result they rejected. The detail carries the target
	// and the matching eligibility reason.
	CodeCorrectionRevisionRequired = "CORRECTION_REVISION_REQUIRED"
)

// RunCommandInput is one owner command against an Outcome's run intent.
type RunCommandInput struct {
	Command                  domain.RunCommand
	PlanRevisionID           domain.PlanRevisionID
	ExpectedContractRevision int64
	// ExpectedGeneration is optimistic concurrency against the intent the
	// caller last read. Zero means no expectation.
	ExpectedGeneration int64
	RequestKey         string
}

// CommandRun records the owner's durable authorization change.
//
// Recording intent is deliberately separate from launching: Start authorizes
// serial continuation across the approved Plan, and the daemon admits each
// eligible WorkUnit itself. That separation is what makes the authorization
// survive a restart — a run that only existed as a running process would be
// lost, and one that only existed as an open screen was never authorized.
func (s *Service) CommandRun(ctx context.Context, outcomeID domain.OutcomeID, in RunCommandInput) (RunStateView, error) {
	if !s.RunIntentsEnabled() {
		// Reaching here means the route was mounted against a daemon without
		// run-intent storage, which is a wiring fault rather than an owner
		// error: the controller answers an unwired capability with 501.
		return RunStateView{}, apierr.Internal(CodeRunIntentUnavailable, "Durable run intent is not wired in this daemon")
	}
	if !in.Command.Valid() {
		return RunStateView{}, apierr.Invalid(CodeRunActionInvalid,
			"Name one of start, pause, resume or cancel", map[string]any{"action": string(in.Command)})
	}
	if strings.TrimSpace(in.RequestKey) == "" {
		return RunStateView{}, apierr.Invalid(CodeRunActionInvalid,
			"A request key is required so a repeated command cannot authorize work twice", nil)
	}

	record, ok, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return RunStateView{}, err
	}
	if !ok {
		return RunStateView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}

	// Replay is resolved before anything else. The same command arriving
	// twice — a double click, a reconnect retry — must return what it already
	// authorized, not be judged against the state its own first copy created.
	// Validating first would refuse a second Start as "already running".
	if replay, found, err := s.runIntents.FindRunIntentByRequestKey(ctx, strings.TrimSpace(in.RequestKey)); err != nil {
		return RunStateView{}, err
	} else if found {
		if replay.OutcomeID != outcomeID || replay.RequestFingerprint == "" || replay.RequestFingerprint != runCommandFingerprint(outcomeID, in) {
			return RunStateView{}, apierr.Conflict(CodeRunActionInvalid,
				"That request key already authorized different run-command semantics",
				map[string]any{"requestKey": strings.TrimSpace(in.RequestKey), "outcomeId": string(replay.OutcomeID)})
		}
		return s.GetRunState(ctx, outcomeID)
	}

	current, hasCurrent, err := s.runIntents.CurrentRunIntent(ctx, outcomeID)
	if err != nil {
		return RunStateView{}, err
	}
	if in.ExpectedGeneration > 0 && (!hasCurrent || current.Generation != in.ExpectedGeneration) {
		return RunStateView{}, apierr.Conflict(CodeRunIntentStale,
			"This Outcome's run intent moved on; reload and decide again",
			map[string]any{"expectedGeneration": in.ExpectedGeneration, "currentGeneration": current.Generation})
	}
	// An omitted generation still gets a compare-and-swap expectation at the
	// durable boundary. For the first command that expectation is "no row";
	// for later commands it is the generation just read. This closes the race
	// where two callers both observe the same state and both append a command.
	expectedGeneration := in.ExpectedGeneration
	if expectedGeneration == 0 && hasCurrent {
		expectedGeneration = current.Generation
	}
	currentDesired := domain.RunIntentIdle
	if hasCurrent {
		currentDesired = current.Desired
	}
	desired, ok := domain.NextRunIntent(currentDesired, in.Command)
	if !ok {
		return RunStateView{}, apierr.Conflict(CodeRunActionUnavailable,
			fmt.Sprintf("%s does not apply while this Outcome's run intent is %s", in.Command, currentDesired),
			map[string]any{"action": string(in.Command), "currentDesired": string(currentDesired)})
	}

	planID, contractRevision := in.PlanRevisionID, in.ExpectedContractRevision
	if desired == domain.RunIntentRunning {
		if in.ExpectedContractRevision < 1 || in.ExpectedContractRevision != record.CurrentRevisionNumber {
			return RunStateView{}, apierr.Conflict(CodeRunIntentStale,
				"Start or resume must name the current reviewed Contract revision",
				map[string]any{"expectedContractRevision": in.ExpectedContractRevision, "currentRevision": record.CurrentRevisionNumber})
		}
		if in.PlanRevisionID.IsZero() {
			return RunStateView{}, apierr.Conflict(CodeRunIntentStale,
				"Start or resume must name the reviewed Plan revision", nil)
		}
		// Authorizing work re-validates the Plan every time. A pause that
		// happened before the Contract moved on must not be resumable into
		// execution against an approved Plan that no longer binds it.
		plan, err := s.approvedPlanForRun(ctx, record, in.PlanRevisionID)
		if err != nil {
			return RunStateView{}, err
		}
		planID, contractRevision = plan.ID, plan.ContractRevisionNumber
		// A standing correction naming the Plan or Contract refuses both Start
		// and Resume, because both authorize execution of the very Plan the
		// owner rejected. The Mission refuses the same move for the same
		// reason; this is what makes that refusal a boundary rather than a
		// rendering choice.
		if err := s.refuseExecutionAgainstCorrection(ctx, outcomeID); err != nil {
			return RunStateView{}, err
		}
		if err := s.refuseUnknownSurvivingWork(ctx, outcomeID); err != nil {
			return RunStateView{}, err
		}
	} else if hasCurrent {
		// Stopping applies to whatever is currently authorized, so it carries
		// that Plan forward rather than requiring the caller to name one.
		planID, contractRevision = current.PlanRevisionID, current.ContractRevisionNumber
	}

	appended, err := s.runIntents.AppendRunIntent(ctx, domain.OutcomeRunIntent{
		ID: domain.RunIntentID("ri-" + uuid.NewString()), OutcomeID: outcomeID,
		Desired: desired, PlanRevisionID: planID, ContractRevisionNumber: contractRevision,
		Command: in.Command, ExpectedGeneration: expectedGeneration,
		ExpectedGenerationSet: true,
		RequestFingerprint:    runCommandFingerprint(outcomeID, in),
		RequestKey:            strings.TrimSpace(in.RequestKey), RequestedAt: s.clock(),
	})
	if err != nil {
		var generationConflict *ports.RunIntentGenerationConflictError
		if errors.As(err, &generationConflict) {
			return RunStateView{}, apierr.Conflict(CodeRunIntentStale,
				"This run intent changed while the command was being recorded; reload and decide again", nil)
		}
		var replayConflict *ports.RunIntentReplayConflictError
		if errors.As(err, &replayConflict) {
			return RunStateView{}, apierr.Conflict(CodeRunActionInvalid,
				"That request key already authorized different run-command semantics", nil)
		}
		return RunStateView{}, err
	}
	if err := s.applyStopIntent(ctx, outcomeID, appended); err != nil {
		return RunStateView{}, err
	}
	return s.GetRunState(ctx, outcomeID)
}

// HaltRunForCorrection ends the authorization an owner correction invalidates.
//
// Requesting rework or reopening is the owner saying the recorded result is not
// the one they want. Leaving the previous authorization in force has two
// consequences, both wrong: the daemon would keep admitting WorkUnits against a
// result the owner has just rejected, and — because Start does not apply to an
// already-running intent — the owner could not authorize the revised work at
// all. So the run is cancelled, which is the existing "stop, and require a
// fresh Start" transition rather than a new state. Kennel never re-authorizes
// on the owner's behalf; it only stops claiming they already did.
//
// The decision is the replay identity, so retrying a correction whose halt
// failed completes it instead of appending a second cancellation — and a replay
// arriving after the owner has authorized new work does nothing at all.
func (s *Service) HaltRunForCorrection(ctx context.Context, outcomeID domain.OutcomeID, decisionID domain.AcceptanceDecisionID) error {
	if s.runIntents == nil {
		return nil
	}
	current, found, err := s.runIntents.CurrentRunIntent(ctx, outcomeID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	requestKey := correctionHaltRequestKey(decisionID)

	// Resolve this correction's own cancellation before deciding anything. It
	// may already exist for two very different reasons, and they must not be
	// confused: its stop effect may have failed and be owed a retry, or the
	// owner may have authorized fresh work since, which supersedes it.
	if recorded, ok, err := s.runIntents.FindRunIntentByRequestKey(ctx, requestKey); err != nil {
		return err
	} else if ok {
		if recorded.OutcomeID != outcomeID || recorded.Generation != current.Generation {
			// Superseded. A decision the owner has moved past may not stop the
			// work they authorized after it.
			return nil
		}
		// Still the authorization in force, so its stop effect is still owed.
		// applyStopIntent is idempotent: an already-cancelled Attempt and an
		// already-acknowledged generation both no-op.
		return s.applyStopIntent(ctx, outcomeID, recorded)
	}

	desired, ok := domain.NextRunIntent(current.Desired, domain.RunCommandCancel)
	if !ok {
		// Already idle or cancelled by something else: there is no
		// authorization of ours to end, and appending one would invent an owner
		// command they never gave.
		return nil
	}
	appended, err := s.runIntents.AppendRunIntent(ctx, domain.OutcomeRunIntent{
		ID: domain.RunIntentID("ri-" + uuid.NewString()), OutcomeID: outcomeID,
		Desired: desired, PlanRevisionID: current.PlanRevisionID,
		ContractRevisionNumber: current.ContractRevisionNumber,
		Command:                domain.RunCommandCancel,
		ExpectedGeneration:     current.Generation, ExpectedGenerationSet: true,
		RequestFingerprint: correctionHaltFingerprint(outcomeID, decisionID),
		RequestKey:         requestKey, RequestedAt: s.clock(),
	})
	if err != nil {
		// A concurrent owner command already moved the intent. It cannot have
		// moved it into a state that admits work — only Start does that, and a
		// Start racing a rejection is the owner's own ordering to resolve — so
		// re-reading and retrying here would fight them. The correction stands
		// either way.
		var generationConflict *ports.RunIntentGenerationConflictError
		if errors.As(err, &generationConflict) {
			return nil
		}
		return err
	}
	// The durable store resolves a repeated request key before it checks
	// generations, so the row that came back may be an older cancellation
	// rather than the one this call appended. Applying that to whatever is
	// running now is how a stale decision cancels current work, so the effect
	// is fenced to the authorization actually in force.
	inForce, found, err := s.runIntents.CurrentRunIntent(ctx, outcomeID)
	if err != nil {
		return err
	}
	if !found || inForce.Generation != appended.Generation {
		return nil
	}
	return s.applyStopIntent(ctx, outcomeID, appended)
}

// correctionHaltRequestKey is the replay identity of "this decision ended this
// Outcome's run".
func correctionHaltRequestKey(decisionID domain.AcceptanceDecisionID) string {
	return "correction:" + string(decisionID)
}

// correctionHaltFingerprint is the replay identity of "this decision ended this
// Outcome's run". It is not a runCommandFingerprint because no owner run
// command was issued: the acceptance decision is the authority.
func correctionHaltFingerprint(outcomeID domain.OutcomeID, decisionID domain.AcceptanceDecisionID) string {
	sum := sha256.Sum256([]byte("correction-halt:" + string(outcomeID) + ":" + string(decisionID)))
	return hex.EncodeToString(sum[:])
}

// runCommandFingerprint is the replay identity of the complete owner request,
// not merely its idempotency key or resulting desired state. JSON gives the
// storage boundary one stable, opaque value to compare without reinterpreting
// a historical command.
func runCommandFingerprint(outcomeID domain.OutcomeID, in RunCommandInput) string {
	request := struct {
		OutcomeID          domain.OutcomeID
		Command            domain.RunCommand
		PlanRevisionID     domain.PlanRevisionID
		ExpectedContract   int64
		ExpectedGeneration int64
		RequestKey         string
	}{
		OutcomeID: outcomeID, Command: in.Command, PlanRevisionID: in.PlanRevisionID,
		ExpectedContract: in.ExpectedContractRevision, ExpectedGeneration: in.ExpectedGeneration,
		RequestKey: strings.TrimSpace(in.RequestKey),
	}
	encoded, _ := json.Marshal(request)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// RunIntentsEnabled reports whether this daemon can hold durable run intent.
// A daemon without it still schedules and reports truthfully; it just has no
// authorization to continue past one Attempt.
func (s *Service) RunIntentsEnabled() bool { return s != nil && s.runIntents != nil }

// approvedPlanForRun resolves the Plan a Start may authorize.
func (s *Service) approvedPlanForRun(ctx context.Context, record domain.Outcome, requested domain.PlanRevisionID) (domain.PlanRevision, error) {
	plan, found, err := s.store.GetLatestPlanRevision(ctx, record.ID)
	if err != nil {
		return domain.PlanRevision{}, err
	}
	if !found {
		return domain.PlanRevision{}, apierr.Conflict(CodePlanNotApproved, "Propose and approve a Plan before starting work", nil)
	}
	if !requested.IsZero() && plan.ID != requested {
		return domain.PlanRevision{}, apierr.Conflict(CodeRunIntentStale,
			"That Plan is no longer this Outcome's latest; reload and authorize the current one",
			map[string]any{"requestedPlanId": string(requested), "currentPlanId": string(plan.ID)})
	}
	if plan.Status != domain.PlanStatusApproved {
		return domain.PlanRevision{}, apierr.Conflict(CodePlanNotApproved,
			"Authorize this Plan before starting work", map[string]any{"planId": plan.ID, "status": plan.Status})
	}
	if !plan.BindsCurrentContract(record.CurrentRevisionNumber) {
		return domain.PlanRevision{}, apierr.Conflict(CodePlanBriefInvalidated,
			"The Contract moved past the approved Plan — propose and approve a fresh Plan",
			map[string]any{"planId": plan.ID, "planRevisionBinding": plan.ContractRevisionNumber, "currentRevision": record.CurrentRevisionNumber})
	}
	return plan, nil
}

// refuseUnknownSurvivingWork blocks authorizing more work while an Attempt of
// unknown status still holds custody.
//
// A failed probe is not death. Authorizing continuation over an Attempt that
// may still be running is exactly how the same WorkUnit gets executed twice.
func (s *Service) refuseUnknownSurvivingWork(ctx context.Context, outcomeID domain.OutcomeID) error {
	attempts, err := s.store.ListAttempts(ctx, outcomeID)
	if err != nil {
		return err
	}
	for _, attempt := range attempts {
		if attempt.Status != domain.AttemptRunning && attempt.Status != domain.AttemptQueued {
			continue
		}
		facts, err := s.heartbeatFacts(ctx, attempt.ID)
		if err != nil {
			return err
		}
		if facts.terminated() {
			continue
		}
		if facts.alive() {
			// Live work is not a conflict for authorization: continuation
			// serialises behind it. Only unaccountable work blocks.
			continue
		}
		return apierr.Conflict(CodeRunCustodyUnknown,
			"An Attempt's runtime status is unknown, so more work cannot be authorized until it is reconciled",
			map[string]any{"attemptId": string(attempt.ID)})
	}
	return nil
}

// applyStopIntent carries out the part of a stop that is not just a record.
//
// Pause deliberately does nothing to work already running: it prevents
// subsequent admission, and is acknowledged once nothing can follow. Cancel
// additionally asks the active Attempt to stop, and is acknowledged only when
// that stop is proven — an unproven stop leaves the request visible rather
// than reporting a cancellation that may not have happened.
func (s *Service) applyStopIntent(ctx context.Context, outcomeID domain.OutcomeID, intent domain.OutcomeRunIntent) error {
	switch intent.Desired {
	case domain.RunIntentPaused, domain.RunIntentCancelled:
	default:
		return nil
	}
	attempts, err := s.store.ListAttempts(ctx, outcomeID)
	if err != nil {
		return err
	}
	active := activeAttempt(attempts)
	if active == nil {
		// Nothing is running, so no further admission can follow this
		// generation and the request has already taken effect.
		return s.runIntents.AcknowledgeRunIntent(ctx, outcomeID, intent.Generation, s.clock())
	}
	if intent.Desired == domain.RunIntentPaused {
		// The active Attempt runs to its own end. Acknowledgement waits for
		// that, and reconcileRunIntents sets it.
		return nil
	}
	if _, err := s.CancelAttempt(ctx, outcomeID, active.ID); err != nil {
		// A refused cancel is recorded on the Attempt and left visible. The
		// intent stays unacknowledged, which is the truthful state: the owner
		// asked to cancel and it has not happened yet.
		var api *apierr.Error
		if asAPIErr(err, &api) {
			return nil
		}
		return err
	}
	return s.runIntents.AcknowledgeRunIntent(ctx, outcomeID, intent.Generation, s.clock())
}

// ReconcileRunIntents carries out stop requests whose effect has not happened
// yet and acknowledges the ones whose work has since ended. It is restart-safe:
// everything is derived from durable facts.
//
// Carrying the effect out — rather than only acknowledging — is what closes the
// window between a cancellation's durable append and the stop it implies. A
// process that died in that window left the Attempt running under an
// authorization that already said it was cancelled, and nothing afterwards
// looked again. applyStopIntent is the same helper the original command used,
// so pause still leaves running work alone and cancel still acknowledges only a
// proven stop.
func (s *Service) ReconcileRunIntents(ctx context.Context) error {
	if s.runIntents == nil {
		return nil
	}
	var failures []error
	for _, desired := range []domain.RunIntentDesired{domain.RunIntentPaused, domain.RunIntentCancelled} {
		intents, err := s.runIntents.ListOutcomesWithRunIntent(ctx, desired)
		if err != nil {
			return err
		}
		for _, intent := range intents {
			if intent.Acknowledged() {
				continue
			}
			// Only the current generation is listed, so this cannot apply a
			// superseded stop to work authorized after it.
			if err := s.applyStopIntent(ctx, intent.OutcomeID, intent); err != nil {
				failures = append(failures, err)
			}
		}
	}
	if len(failures) == 1 {
		return failures[0]
	}
	if len(failures) > 1 {
		return fmt.Errorf("run intent reconciliation: %w", joinErrors(failures))
	}
	return nil
}

// ContinueAuthorizedRuns admits the next eligible WorkUnit for every Outcome
// whose current intent authorizes running.
//
// Serial by construction: it admits at most one Attempt per Outcome per tick
// and the schedule reports nothing runnable while an Attempt is active. The
// request key is derived from the Outcome, the intent generation and the
// WorkUnit, so repeated ticks, a restart mid-admission, and two reconcilers
// racing all converge on exactly one Attempt for that unit under that
// authorization.
func (s *Service) ContinueAuthorizedRuns(ctx context.Context) error {
	if s.runIntents == nil {
		return nil
	}
	intents, err := s.runIntents.ListOutcomesWithRunIntent(ctx, domain.RunIntentRunning)
	if err != nil {
		return err
	}
	var failures []error
	for _, intent := range intents {
		if err := s.continueOneRun(ctx, intent); err != nil {
			failures = append(failures, fmt.Errorf("outcome %s: %w", intent.OutcomeID, err))
		}
	}
	if len(failures) == 1 {
		return failures[0]
	}
	if len(failures) > 1 {
		return fmt.Errorf("authorized run continuation: %w", joinErrors(failures))
	}
	return nil
}

func (s *Service) continueOneRun(ctx context.Context, intent domain.OutcomeRunIntent) error {
	schedule, err := s.GetSchedule(ctx, intent.OutcomeID, intent.PlanRevisionID)
	if err != nil {
		// A Plan that is no longer approvable, or an Outcome that moved on,
		// is a durable owner-visible fact the Mission already reports; it is
		// not a reason to fail every other authorized run. A genuine storage
		// failure still surfaces.
		var api *apierr.Error
		if asAPIErr(err, &api) {
			return nil
		}
		return err
	}
	if schedule.NextRunnableID.IsZero() {
		return nil
	}
	_, err = s.StartAttempt(ctx, intent.OutcomeID, StartAttemptInput{
		PlanRevisionID: intent.PlanRevisionID,
		WorkUnitID:     schedule.NextRunnableID,
		RequestKey:     runContinuationKey(intent, schedule.NextRunnableID),
	})
	if err == nil {
		return nil
	}
	// Every refusal here is already a durable, owner-visible fact: a blocked
	// dependency, a held fence, an unready provider. Continuation reports
	// them rather than retrying, because an automatic retry after an unproved
	// failure is exactly the loop this design refuses to build.
	var api *apierr.Error
	if asAPIErr(err, &api) {
		return nil
	}
	return err
}

// runContinuationKey is the replay identity of "this generation admitting
// this WorkUnit". Including the generation means a fresh Start after a pause
// legitimately re-admits a unit the previous authorization never reached,
// while the same authorization can never admit it twice.
func runContinuationKey(intent domain.OutcomeRunIntent, unitID domain.WorkUnitID) string {
	return fmt.Sprintf("run:%s:%d:%s", intent.OutcomeID, intent.Generation, unitID)
}

// refuseAdmissionAgainstRunIntent stops a new Attempt while the owner has
// paused or cancelled this Outcome's run.
//
// It applies to every admission path, including a direct per-Attempt Start:
// a pause the owner can click past by using a different button is not a
// pause. An Outcome that has never been started has no intent and is not
// gated — Start is how one begins.
func (s *Service) refuseAdmissionAgainstRunIntent(ctx context.Context, outcomeID domain.OutcomeID) (int64, error) {
	intent, found, err := s.currentRunIntent(ctx, outcomeID)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, nil
	}
	switch intent.Desired {
	case domain.RunIntentPaused, domain.RunIntentCancelled:
		return 0, apierr.Conflict(CodeRunActionUnavailable,
			fmt.Sprintf("This Outcome's run is %s; resume or start it before admitting more work", intent.Desired),
			map[string]any{"outcomeId": string(outcomeID), "desired": string(intent.Desired), "generation": intent.Generation})
	}
	return intent.Generation, nil
}

// currentRunIntent reads the authorization admission must respect.
func (s *Service) currentRunIntent(ctx context.Context, outcomeID domain.OutcomeID) (domain.OutcomeRunIntent, bool, error) {
	if s.runIntents == nil {
		return domain.OutcomeRunIntent{}, false, nil
	}
	return s.runIntents.CurrentRunIntent(ctx, outcomeID)
}

// asAPIErr reports whether err carries a typed control-plane refusal.
func asAPIErr(err error, target **apierr.Error) bool {
	return errors.As(err, target)
}

func joinErrors(errs []error) error { return errors.Join(errs...) }

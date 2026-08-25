// Act & Observe (#31): governed execution of an approved plan onto a real
// provider session. Admission is fail-closed and ordered BEFORE any durable
// row: approved plan -> current-contract binding -> grants survive the
// authority intersection -> RunBrief core digest recomputes equal ->
// profile-readiness probe. Only then do the attempt row, fence, session ref,
// and running transition land. A spawner crash after the row exists records a
// truthful `failed` attempt plus a recovery receipt; replacement is always a
// NEW attempt row.

package outcome

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// heartbeatSource resolves bound-session heartbeat facts for derived
// presentation. It is satisfied by the storage layer's ordinary session read;
// services never touch SQLite directly.
type heartbeatSource interface {
	GetSession(ctx context.Context, id domain.SessionID) (domain.SessionRecord, bool, error)
}

// AttemptManager is the controller-facing Act & Observe boundary.
type AttemptManager interface {
	StartAttempt(ctx context.Context, outcomeID domain.OutcomeID, in StartAttemptInput) (AttemptView, error)
	GetAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (AttemptView, error)
	ListAttempts(ctx context.Context, outcomeID domain.OutcomeID) ([]AttemptView, error)
	PauseAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (AttemptView, error)
	ResumeAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (AttemptView, error)
	CancelAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (AttemptView, error)
	RecordObservation(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID, in RecordObservationInput) (domain.AttemptObservation, error)
	RecoverAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID, in RecoveryInput) (RecoveryView, error)
}

// StartAttemptInput names the authorized plan to execute and carries the
// client's idempotency key: replaying a delivered RequestKey resolves to the
// original attempt without admitting anything twice.
type StartAttemptInput struct {
	PlanRevisionID domain.PlanRevisionID
	// Harness optionally names the worker provider; empty defaults to the v0
	// locked fixture (Codex-first). There is no fallback chain.
	Harness    domain.AgentHarness
	RequestKey string
}

// RecordObservationInput appends one ordered observation. Observations are
// inspectable on ANY attempt state but never mutate current truth.
type RecordObservationInput struct {
	Kind    string
	Payload string
}

// RecoveryAction enumerates the owner-directed recovery verbs.
type RecoveryAction string

// Recovery verbs accepted by the recovery route; each maps to one handler.
const (
	RecoveryActionContain   RecoveryAction = "contain"
	RecoveryActionReconcile RecoveryAction = "reconcile"
	RecoveryActionResume    RecoveryAction = "resume"
	RecoveryActionReplace   RecoveryAction = "replace"
	RecoveryActionAttention RecoveryAction = "attention"
)

// Valid reports whether a is a supported recovery action.
func (a RecoveryAction) Valid() bool {
	switch a {
	case RecoveryActionContain, RecoveryActionReconcile, RecoveryActionResume,
		RecoveryActionReplace, RecoveryActionAttention:
		return true
	}
	return false
}

// RecoveryInput directs containment/reconciliation for one attempt.
type RecoveryInput struct {
	Action RecoveryAction
	// ConfirmProviderStopped is the OWNER's explicit assertion that the bound
	// provider session is no longer running. Owner authority (invariant 5)
	// permits custody release without machine proof, but the assertion is
	// recorded as its own containment observation so it stays auditable.
	ConfirmProviderStopped bool
}

// AttemptView is the Act & Observe read model: durable lineage plus DERIVED
// presentation computed from stored status and heartbeat facts. Nothing here
// is persisted back.
type AttemptView struct {
	Outcome      domain.Outcome
	Attempt      domain.Attempt
	Sessions     []domain.AttemptSessionRef
	Observations []domain.AttemptObservation
	Receipts     []domain.AttemptRecoveryReceipt
	// Fence is the attempt's custody fence when it currently holds the subject.
	Fence        *domain.AttemptFence
	Presentation domain.AttemptPresentation
}

// RecoveryView pairs the post-recovery read model with the receipt that
// recorded the verdict.
type RecoveryView struct {
	Attempt AttemptView
	Receipt *domain.AttemptRecoveryReceipt
}

const (
	// CodeAgentProfileNotReady mirrors the shared admission vocabulary: the
	// probed harness configuration cannot launch yet.
	CodeAgentProfileNotReady = "AGENT_PROFILE_NOT_READY"
	// CodeAgentBinaryNotFound mirrors ports.ErrAgentBinaryNotFound surfacing.
	CodeAgentBinaryNotFound = "AGENT_BINARY_NOT_FOUND"
	// CodePlanNotApproved refuses executing anything but an owner-approved plan.
	CodePlanNotApproved = "PLAN_NOT_APPROVED"
	// CodePlanBriefInvalidated reports the frozen brief no longer matches the
	// current contract (or its own digest): a fresh proposal + approval is
	// required before any admission.
	CodePlanBriefInvalidated = "PLAN_BRIEF_INVALIDATED"
	// CodeAttemptCapabilityUnauthorized reports narrowed authority at start time.
	CodeAttemptCapabilityUnauthorized = "ATTEMPT_CAPABILITY_UNAUTHORIZED"
	// CodeAttemptFenceHeld reports another attempt holds worktree custody.
	CodeAttemptFenceHeld = "ATTEMPT_FENCE_HELD"
	// CodeAttemptNotFound reports an unknown attempt under this Outcome.
	CodeAttemptNotFound = "ATTEMPT_NOT_FOUND"
	// CodeAttemptLivenessUnproven refuses resume/replace decisions without
	// provable liveness evidence.
	CodeAttemptLivenessUnproven = "ATTEMPT_LIVENESS_UNPROVEN"
	// CodeAttemptActivationUnresolved reports a LIVE provider whose running
	// transition could not be recorded durably.
	CodeAttemptActivationUnresolved = "ATTEMPT_ACTIVATION_UNRESOLVED"
	// CodeAttemptCustodyUnproven refuses custody release while the bound
	// provider's stop is unproven — the anti-duplicate-writer gate.
	CodeAttemptCustodyUnproven = "ATTEMPT_CUSTODY_UNPROVEN"
	// CodeAttemptProviderStopFailed reports the terminate call failed; status
	// is unchanged and custody stays held.
	CodeAttemptProviderStopFailed = "ATTEMPT_PROVIDER_STOP_FAILED"
)

// NewWithExecution builds the service with the Act & Observe seams wired.
func NewWithExecution(store ports.OutcomeStore, clock func() time.Time, spawner ports.AttemptSessionSpawner, heartbeats heartbeatSource) *Service {
	svc := New(store, clock)
	svc.spawner = spawner
	svc.heartbeats = heartbeats
	svc.staleHeartbeat = domain.DefaultStaleHeartbeatWindow
	return svc
}

var _ AttemptManager = (*Service)(nil)

// StartAttempt admits one authorized plan onto a real provider session using
// the ratified fail-closed ordering. Every refusal below happens BEFORE any
// durable row exists.
func (s *Service) StartAttempt(ctx context.Context, outcomeID domain.OutcomeID, in StartAttemptInput) (AttemptView, error) {
	if s.spawner == nil || s.heartbeats == nil {
		return AttemptView{}, apierr.Internal("ATTEMPT_EXECUTION_UNWIRED", "Attempt execution is not wired in this environment")
	}
	if strings.TrimSpace(in.RequestKey) == "" {
		return AttemptView{}, apierr.Invalid("REQUEST_KEY_REQUIRED", "Provide an idempotency key for this start request", nil)
	}

	// Replay first: a delivered start never admits twice.
	if existing, ok, err := s.store.FindAttemptByIdempotencyKey(ctx, in.RequestKey); err != nil {
		return AttemptView{}, err
	} else if ok {
		return s.GetAttempt(ctx, existing.OutcomeID, existing.ID)
	}

	outcome, ok, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return AttemptView{}, err
	}
	if !ok {
		return AttemptView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	plan, found, err := s.store.GetPlanRevision(ctx, outcomeID, in.PlanRevisionID)
	if err != nil {
		return AttemptView{}, err
	}
	if !found {
		return AttemptView{}, apierr.NotFound("PLAN_NOT_FOUND", "That plan does not exist")
	}
	// Gate 1: only an owner-approved plan may execute.
	if plan.Status != domain.PlanStatusApproved {
		return AttemptView{}, apierr.Conflict(CodePlanNotApproved,
			"Authorize this plan before starting an attempt",
			map[string]any{"planId": string(plan.ID), "status": string(plan.Status)})
	}
	// Gate 2: the plan must still bind the Outcome's CURRENT contract revision.
	if !plan.BindsCurrentContract(outcome.CurrentRevisionNumber) {
		return AttemptView{}, apierr.Conflict(CodePlanBriefInvalidated,
			fmt.Sprintf("Plan binds contract revision %s; the Outcome is at %s — propose and approve a fresh plan",
				formatI64(plan.ContractRevisionNumber), formatI64(outcome.CurrentRevisionNumber)),
			map[string]any{
				"outcomeId":           string(outcomeID),
				"planId":              string(plan.ID),
				"planRevisionBinding": plan.ContractRevisionNumber,
				"currentRevision":     outcome.CurrentRevisionNumber,
			})
	}
	// Gate 3: every grant must still survive the authority intersection.
	if err := s.authorizeAttemptCapabilities(plan.Grants); err != nil {
		return AttemptView{}, err
	}
	// Gate 4: recompute the frozen RunBrief core digest from the CURRENT
	// contract content; any material drift invalidates the brief.
	revision, err := s.currentRevision(ctx, outcome)
	if err != nil {
		return AttemptView{}, err
	}
	recomputed, err := domain.ComputeRunBriefCoreDigest(revision, plan.WorkUnits[0], plan.Grants)
	if err != nil {
		return AttemptView{}, err
	}
	if recomputed != plan.RunBriefCoreDigest {
		return AttemptView{}, apierr.Conflict(CodePlanBriefInvalidated,
			"The frozen RunBrief no longer matches the contract — propose and approve a fresh plan",
			map[string]any{"outcomeId": string(outcomeID), "planId": string(plan.ID)})
	}
	projectID, ok, err := s.store.GetOutcomeProjectID(ctx, outcomeID)
	if err != nil {
		return AttemptView{}, err
	}
	if !ok {
		return AttemptView{}, apierr.NotFound("PROJECT_NOT_FOUND", "Register that project before starting attempts")
	}

	// Gate 5: profile readiness through the same checker and config merge
	// session spawn uses, resolved for THIS project's worker defaults.
	harness := in.Harness
	if harness == "" {
		harness = domain.HarnessCodex
	}
	if err := s.probeReadiness(ctx, projectID, harness); err != nil {
		return AttemptView{}, err
	}

	now := s.clock()
	attempt, err := s.store.CreateAttemptWithFence(ctx, outcomeID, plan, strings.TrimSpace(in.RequestKey), domain.FenceSubjectForProject(projectID), now)
	if err != nil {
		// A lost same-request-key race serves the WINNER's attempt and never
		// spawns a second provider session.
		var replay *ports.AttemptReplayError
		if errors.As(err, &replay) {
			return s.GetAttempt(ctx, replay.Attempt.OutcomeID, replay.Attempt.ID)
		}
		var held *ports.AttemptFenceHeldError
		if errors.As(err, &held) {
			return AttemptView{}, apierr.Conflict(CodeAttemptFenceHeld,
				"Another attempt holds custody of this project's worktree — reconcile it first",
				map[string]any{
					"subject":      held.Subject,
					"holder":       string(held.Holder),
					"attemptedFor": string(held.OutcomeID),
				})
		}
		return AttemptView{}, err
	}

	// Durable rows now exist: any failure from here on must leave TRUTHFUL
	// state — failed attempt, observation, receipt — never a silent retry or
	// an in-place replacement.
	prompt := renderRunBriefPrompt(revision, plan)
	session, err := s.spawner.Spawn(ctx, ports.AttemptSpawnRequest{
		ProjectID:   projectID,
		Harness:     harness,
		Prompt:      prompt,
		DisplayName: fmt.Sprintf("%s · attempt %d", outcome.Title, attempt.Number),
	})
	if err != nil {
		// An UNKNOWN spawn outcome (deadline/cancellation after the request
		// was sent) is NOT a clean failure: the provider session may exist.
		// The attempt stays queued and derives as unconfirmed until reconcile
		// decides; the held fence already blocks duplicate admission.
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return AttemptView{}, s.admitUnresolved(ctx, attempt.ID, domain.ObservationAdmissionAmbiguous, err)
		}
		return AttemptView{}, s.admitFailed(ctx, outcomeID, attempt.ID, err)
	}

	mode := session.Mode
	compiled := computeCompiledBriefDigest(harness, mode, recomputed)
	snapshot, err := json.Marshal(map[string]any{
		"snapshotVersion":        domain.AdmissionSnapshotVersion,
		"harness":                string(harness),
		"mode":                   string(mode),
		"runBriefCoreDigest":     recomputed,
		"runBriefCompiledDigest": compiled,
		"sessionId":              session.ID,
		"requestedAt":            now,
	})
	if err != nil {
		return AttemptView{}, s.admitFailed(ctx, outcomeID, attempt.ID, err)
	}
	if _, err := s.store.BindAttemptSession(ctx, domain.AttemptSessionRef{
		AttemptID:              attempt.ID,
		SessionID:              string(session.ID),
		Harness:                harness,
		Mode:                   mode,
		RunBriefCoreDigest:     recomputed,
		RunBriefCompiledDigest: compiled,
		AdmissionSnapshot:      string(snapshot),
		BoundAt:                s.clock(),
	}); err != nil {
		return AttemptView{}, s.admitFailed(ctx, outcomeID, attempt.ID, err)
	}
	rows, err := s.store.TransitionAttemptStatus(ctx, outcomeID, attempt.ID, domain.AttemptQueued, domain.AttemptRunning, s.clock())
	if err != nil || rows != 1 {
		// The provider session IS LIVE here; only the durable promotion
		// failed. Never a raw 500: record actionable ambiguity, keep the
		// fence, and let reconcile decide.
		return AttemptView{}, s.admitUnresolved(ctx, attempt.ID, domain.ObservationActivationAmbiguous,
			fmt.Errorf("activation not recorded (rows=%d): %w", rows, err))
	}
	return s.GetAttempt(ctx, outcomeID, attempt.ID)
}

// admitUnresolved records an UNKNOWN admission/activation outcome: status
// stays queued (never failed, never silently running), an ambiguous
// observation and a needs_attention receipt land, and the fence stays HELD so
// no duplicate writer can start before reconcile decides. Internal recording
// failures are joined into the returned error instead of being dropped.
func (s *Service) admitUnresolved(ctx context.Context, attemptID domain.AttemptID, kind string, cause error) error {
	detail := ""
	if cause != nil {
		detail = cause.Error()
	}
	payload, _ := json.Marshal(map[string]any{"error": detail, "unresolved": true})
	var errs []error
	if _, err := s.store.AppendAttemptObservation(ctx, attemptID, kind, string(payload), s.clock()); err != nil {
		errs = append(errs, fmt.Errorf("record ambiguous start (%s) for %s: %w", kind, attemptID, err))
	}
	if err := s.store.CreateRecoveryReceipt(ctx, domain.AttemptRecoveryReceipt{
		ID:         "rcpt-" + uuid.NewString(),
		AttemptID:  attemptID,
		Resolution: domain.RecoveryNeedsAttention,
		Detail:     string(payload),
		CreatedAt:  s.clock(),
	}); err != nil {
		errs = append(errs, fmt.Errorf("record ambiguous activation receipt for %s: %w", attemptID, err))
	}
	code, headline := CodeAttemptActivationUnresolved, "The activation outcome is unknown"
	if kind == domain.ObservationAdmissionAmbiguous {
		code, headline = "ATTEMPT_START_UNRESOLVED", "The start outcome is unknown"
	}
	unresolved := apierr.Conflict(code,
		headline+" — the attempt stays unconfirmed until you reconcile it",
		map[string]any{"attemptId": string(attemptID)})
	if len(errs) > 0 {
		return errors.Join(append(errs, unresolved)...)
	}
	return unresolved
}

// admitFailed records the truthful aftermath of a spawn/bind crash after the
// durable row existed: status failed, an admission_failed observation, and a
// needs_attention receipt. The fence stays HELD — replacement inherits
// custody only through reconcile.
func (s *Service) admitFailed(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID, cause error) error {
	detail := ""
	if cause != nil {
		detail = cause.Error()
	}
	var errs []error
	if rows, err := s.store.TransitionAttemptStatus(ctx, outcomeID, attemptID, domain.AttemptQueued, domain.AttemptFailed, s.clock()); err != nil {
		errs = append(errs, fmt.Errorf("record failed status for %s: %w", attemptID, err))
	} else if rows != 1 {
		errs = append(errs, fmt.Errorf("record failed status for %s: status moved concurrently", attemptID))
	}
	payload, _ := json.Marshal(map[string]any{"error": detail})
	if _, err := s.store.AppendAttemptObservation(ctx, attemptID, domain.ObservationAdmissionFailed, string(payload), s.clock()); err != nil {
		errs = append(errs, fmt.Errorf("record admission-failure observation for %s: %w", attemptID, err))
	}
	if err := s.store.CreateRecoveryReceipt(ctx, domain.AttemptRecoveryReceipt{
		ID:         "rcpt-" + uuid.NewString(),
		AttemptID:  attemptID,
		Resolution: domain.RecoveryNeedsAttention,
		Detail:     string(payload),
		CreatedAt:  s.clock(),
	}); err != nil {
		errs = append(errs, fmt.Errorf("record admission-failure receipt for %s: %w", attemptID, err))
	}
	admitted := cause
	if admitted == nil {
		admitted = fmt.Errorf("unknown admission failure")
	}
	if apiErr := (*apierr.Error)(nil); !errors.As(admitted, &apiErr) {
		admitted = apierr.Conflict("ATTEMPT_ADMIT_FAILED",
			"The provider session could not be started; the attempt is recorded as failed",
			map[string]any{"attemptId": string(attemptID), "error": detail})
	}
	if len(errs) > 0 {
		return errors.Join(append(errs, admitted)...)
	}
	return admitted
}

// PauseAttempt suspends a running attempt.
func (s *Service) PauseAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (AttemptView, error) {
	attempt, _, err := s.requireAttempt(ctx, outcomeID, attemptID)
	if err != nil {
		return AttemptView{}, err
	}
	rows, err := s.store.TransitionAttemptStatus(ctx, outcomeID, attemptID, domain.AttemptRunning, domain.AttemptPaused, s.clock())
	if err != nil {
		return AttemptView{}, err
	}
	if rows == 0 {
		return AttemptView{}, apierr.Conflict("ATTEMPT_NOT_RUNNING",
			"Only a running attempt can be paused", map[string]any{"status": string(attempt.Status)})
	}
	payload, _ := json.Marshal(map[string]any{"previousStatus": string(domain.AttemptRunning)})
	if _, err := s.store.AppendAttemptObservation(ctx, attemptID, domain.ObservationOwnerPause, string(payload), s.clock()); err != nil {
		return AttemptView{}, err
	}
	return s.GetAttempt(ctx, outcomeID, attemptID)
}

// ResumeAttempt returns a paused attempt to running AFTER enforcing spawn's
// readiness contract: the same profile probe admission runs, and a harness
// that can no longer launch refuses the resume closed.
func (s *Service) ResumeAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (AttemptView, error) {
	attempt, _, err := s.requireAttempt(ctx, outcomeID, attemptID)
	if err != nil {
		return AttemptView{}, err
	}
	if attempt.Status != domain.AttemptPaused {
		return AttemptView{}, apierr.Conflict("ATTEMPT_NOT_PAUSED",
			"Only a paused attempt can be resumed", map[string]any{"status": string(attempt.Status)})
	}
	harness := domain.HarnessCodex
	if ref, ok, err := s.store.LatestAttemptSessionRef(ctx, attemptID); err != nil {
		return AttemptView{}, err
	} else if ok && ref.Harness != "" {
		harness = ref.Harness
	}
	projectID, _, err := s.store.GetOutcomeProjectID(ctx, outcomeID)
	if err != nil {
		return AttemptView{}, err
	}
	if err := s.probeReadiness(ctx, projectID, harness); err != nil {
		return AttemptView{}, err
	}
	rows, err := s.store.TransitionAttemptStatus(ctx, outcomeID, attemptID, domain.AttemptPaused, domain.AttemptRunning, s.clock())
	if err != nil {
		return AttemptView{}, err
	}
	if rows == 0 {
		return AttemptView{}, apierr.Conflict("ATTEMPT_STATUS_MOVED",
			"The attempt changed state concurrently; reload and retry", nil)
	}
	payload, _ := json.Marshal(map[string]any{"previousStatus": string(domain.AttemptPaused)})
	if _, err := s.store.AppendAttemptObservation(ctx, attemptID, domain.ObservationOwnerResume, string(payload), s.clock()); err != nil {
		return AttemptView{}, err
	}
	return s.GetAttempt(ctx, outcomeID, attemptID)
}

// probeReadiness runs the admission profile probe against the spawner.
func (s *Service) probeReadiness(ctx context.Context, projectID domain.ProjectID, harness domain.AgentHarness) error {
	readiness, err := s.spawner.ProfileReadiness(ctx, projectID, harness)
	if err != nil {
		if errors.Is(err, ports.ErrAgentBinaryNotFound) {
			return apierr.Conflict(CodeAgentBinaryNotFound,
				"The agent binary is not installed on this machine", map[string]any{"harness": string(harness)})
		}
		return err
	}
	if !readiness.Ready {
		return apierr.Conflict(CodeAgentProfileNotReady,
			"The selected agent profile is not ready to launch", map[string]any{
				"harness": string(harness),
				"detail":  readiness.Detail,
			})
	}
	return nil
}

// CancelAttempt ends an active attempt by owner decision AND stops the bound
// provider through the execution seam: canonical cancellation requires
// provider authority, not a database status flip. If termination fails the
// status is left untouched, the fence stays held, and a needs_attention
// receipt records exactly what could not be stopped.
func (s *Service) CancelAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (AttemptView, error) {
	attempt, _, err := s.requireAttempt(ctx, outcomeID, attemptID)
	if err != nil {
		return AttemptView{}, err
	}
	switch attempt.Status {
	case domain.AttemptQueued, domain.AttemptRunning, domain.AttemptPaused:
	default:
		return AttemptView{}, apierr.Conflict("ATTEMPT_ALREADY_ENDED",
			fmt.Sprintf("Attempt already ended as %s", attempt.Status),
			map[string]any{"status": string(attempt.Status)})
	}
	ref, bound, err := s.store.LatestAttemptSessionRef(ctx, attemptID)
	if err != nil {
		return AttemptView{}, err
	}
	projectID, _, err := s.store.GetOutcomeProjectID(ctx, outcomeID)
	if err != nil {
		return AttemptView{}, err
	}
	providerStopped := false
	if bound {
		if termErr := s.spawner.Terminate(ctx, projectID, ref.SessionID); termErr != nil {
			payload := mustJSON(map[string]any{"error": termErr.Error(), "sessionId": ref.SessionID})
			var errs []error
			if _, obsErr := s.store.AppendAttemptObservation(ctx, attemptID, domain.ObservationProviderStopFailed, payload, s.clock()); obsErr != nil {
				errs = append(errs, fmt.Errorf("record stop-failure observation for %s: %w", attemptID, obsErr))
			}
			if rcptErr := s.store.CreateRecoveryReceipt(ctx, domain.AttemptRecoveryReceipt{
				ID:         "rcpt-" + uuid.NewString(),
				AttemptID:  attemptID,
				Resolution: domain.RecoveryNeedsAttention,
				Detail:     payload,
				CreatedAt:  s.clock(),
			}); rcptErr != nil {
				errs = append(errs, fmt.Errorf("record stop-failure receipt for %s: %w", attemptID, rcptErr))
			}
			refused := apierr.Conflict(CodeAttemptProviderStopFailed,
				"The provider session could not be stopped — cancel was NOT recorded; stop it and retry",
				map[string]any{"attemptId": string(attemptID), "sessionId": ref.SessionID})
			if len(errs) > 0 {
				return AttemptView{}, errors.Join(append(errs, refused)...)
			}
			return AttemptView{}, refused
		}
		providerStopped = true
	}
	rows, err := s.store.TransitionAttemptStatus(ctx, outcomeID, attemptID, attempt.Status, domain.AttemptCancelled, s.clock())
	if err != nil {
		return AttemptView{}, err
	}
	if rows == 0 {
		return AttemptView{}, apierr.Conflict("ATTEMPT_STATUS_MOVED", "The attempt changed state concurrently; reload and retry", nil)
	}
	payload, _ := json.Marshal(map[string]any{"previousStatus": string(attempt.Status), "providerStopped": providerStopped})
	if _, err := s.store.AppendAttemptObservation(ctx, attemptID, domain.ObservationOwnerCancel, string(payload), s.clock()); err != nil {
		return AttemptView{}, err
	}
	return s.GetAttempt(ctx, outcomeID, attemptID)
}

// GetAttempt reads one attempt's full read model with derived presentation.
func (s *Service) GetAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (AttemptView, error) {
	attempt, outcome, err := s.requireAttempt(ctx, outcomeID, attemptID)
	if err != nil {
		return AttemptView{}, err
	}
	return s.readModel(ctx, outcome, attempt)
}

// ListAttempts reads every attempt of the Outcome in lineage order.
func (s *Service) ListAttempts(ctx context.Context, outcomeID domain.OutcomeID) ([]AttemptView, error) {
	outcome, ok, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	attempts, err := s.store.ListAttempts(ctx, outcomeID)
	if err != nil {
		return nil, err
	}
	views := make([]AttemptView, 0, len(attempts))
	for _, attempt := range attempts {
		view, err := s.readModel(ctx, outcome, attempt)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

// RecordObservation appends one ordered observation. Insertable ALWAYS (D5):
// stale attempts stay inspectable, and no observation ever mutates current
// truth by itself.
func (s *Service) RecordObservation(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID, in RecordObservationInput) (domain.AttemptObservation, error) {
	if _, _, err := s.requireAttempt(ctx, outcomeID, attemptID); err != nil {
		return domain.AttemptObservation{}, err
	}
	kind := strings.TrimSpace(in.Kind)
	if kind == "" {
		return domain.AttemptObservation{}, apierr.Invalid("OBSERVATION_KIND_REQUIRED", "Name what was observed", nil)
	}
	payload := in.Payload
	if payload == "" {
		payload = "{}"
	} else if !json.Valid([]byte(payload)) {
		return domain.AttemptObservation{}, apierr.Invalid("OBSERVATION_PAYLOAD_INVALID", "Payload must be valid JSON", nil)
	}
	return s.store.AppendAttemptObservation(ctx, attemptID, kind, payload, s.clock())
}

// requireAttempt loads the attempt under its Outcome or answers typed 404s.
func (s *Service) requireAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID) (domain.Attempt, domain.Outcome, error) {
	outcome, ok, err := s.store.GetOutcome(ctx, outcomeID)
	if err != nil {
		return domain.Attempt{}, domain.Outcome{}, err
	}
	if !ok {
		return domain.Attempt{}, domain.Outcome{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome does not exist")
	}
	attempt, found, err := s.store.GetAttempt(ctx, outcomeID, attemptID)
	if err != nil {
		return domain.Attempt{}, domain.Outcome{}, err
	}
	if !found {
		return domain.Attempt{}, domain.Outcome{}, apierr.NotFound(CodeAttemptNotFound, "That attempt does not exist for this Outcome")
	}
	return attempt, outcome, nil
}

// readModel assembles the view: lineage, observations, receipts, open fence,
// and presentation derived from stored status plus heartbeat facts.
func (s *Service) readModel(ctx context.Context, outcome domain.Outcome, attempt domain.Attempt) (AttemptView, error) {
	sessions, err := s.store.ListAttemptSessionRefs(ctx, attempt.ID)
	if err != nil {
		return AttemptView{}, err
	}
	observations, err := s.store.ListAttemptObservations(ctx, attempt.ID)
	if err != nil {
		return AttemptView{}, err
	}
	receipts, err := s.store.ListRecoveryReceipts(ctx, attempt.ID)
	if err != nil {
		return AttemptView{}, err
	}
	facts := domain.SessionHeartbeatFacts{}
	if latest, ok, err := s.store.LatestAttemptSessionRef(ctx, attempt.ID); err != nil {
		return AttemptView{}, err
	} else if ok && s.heartbeats != nil {
		if rec, present, err := s.heartbeats.GetSession(ctx, domain.SessionID(latest.SessionID)); err != nil {
			return AttemptView{}, err
		} else if present {
			facts = domain.SessionHeartbeatFacts{
				Present:        true,
				ActivityState:  rec.Activity.State,
				FirstSignalAt:  rec.FirstSignalAt,
				LastActivityAt: rec.Activity.LastActivityAt,
				IsTerminated:   rec.IsTerminated,
			}
		}
	}
	unresolvedAdmission := false
	for _, obs := range observations {
		if obs.Kind == domain.ObservationAdmissionAmbiguous || obs.Kind == domain.ObservationActivationAmbiguous {
			unresolvedAdmission = true // kinds are only ever appended
		}
	}
	subjectProject, ok, err := s.store.GetOutcomeProjectID(ctx, outcome.ID)
	if err != nil {
		return AttemptView{}, err
	}
	var fence *domain.AttemptFence
	if ok {
		if open, held, err := s.store.OpenFenceForSubject(ctx, domain.FenceSubjectForProject(subjectProject)); err != nil {
			return AttemptView{}, err
		} else if held && open.AttemptID == attempt.ID {
			fence = &open
		}
	}
	return AttemptView{
		Outcome:      outcome,
		Attempt:      attempt,
		Sessions:     sessions,
		Observations: observations,
		Receipts:     receipts,
		Fence:        fence,
		Presentation: domain.DeriveAttemptPresentation(attempt.Status, facts, unresolvedAdmission,
			domain.LivenessPolicy{Now: s.clock(), StaleHeartbeatAfter: s.staleHeartbeat}),
	}, nil
}

// authorizeAttemptCapabilities applies the same fail-closed gates as approval
// but speaks the attempt vocabulary.
func (s *Service) authorizeAttemptCapabilities(grants []domain.CapabilityGrant) error {
	authoritative := s.authoritativeCapabilities()
	if err := domain.GrantsFailClosed(grants, authoritative); err != nil {
		names := make([]string, 0, len(grants))
		for _, grant := range grants {
			names = append(names, grant.Name)
		}
		return apierr.New(apierr.KindConflict, CodeAttemptCapabilityUnauthorized,
			err.Error(),
			map[string]any{"granted": names, "authoritative": authoritative})
	}
	if missing := domain.MissingRequiredCapabilities(grants); len(missing) > 0 {
		return apierr.Invalid(CodeAttemptCapabilityUnauthorized,
			"This environment cannot offer every capability the plan requires: "+strings.Join(missing, ", "),
			map[string]any{"missing": missing})
	}
	return nil
}

// computeCompiledBriefDigest freezes the adapter-compiled brief identity:
// core digest plus the concrete harness/mode it was compiled for. Versioned
// so later slices can evolve compilation without reinterpreting old refs.
func computeCompiledBriefDigest(harness domain.AgentHarness, mode domain.SessionMode, core string) string {
	sum := sha256.Sum256([]byte("v0|" + string(harness) + "|" + string(mode) + "|" + core))
	return hex.EncodeToString(sum[:])
}

// renderRunBriefPrompt derives the deterministic provider-neutral task text
// from the frozen contract and plan. It is a BRIEF, not a transcript: no
// provider prose is parsed or persisted anywhere in #31.
func renderRunBriefPrompt(revision domain.ContractRevision, plan domain.PlanRevision) string {
	unit := plan.WorkUnits[0]
	var b strings.Builder
	b.WriteString("Execute the following approved Work Unit inside your isolated worktree.\n\n")
	b.WriteString("Goal: " + revision.Goal + "\n")
	b.WriteString("Work unit: " + unit.Title + "\n")
	b.WriteString("Expected output: " + unit.OutputSummary + "\n")
	b.WriteString("Evidence checks:\n")
	for _, check := range unit.EvidenceChecks {
		b.WriteString("- " + check + "\n")
	}
	b.WriteString("Verification: " + unit.VerificationRequirement + "\n")
	if len(unit.StopConditions) > 0 {
		b.WriteString("Stop conditions:\n")
		for _, stop := range unit.StopConditions {
			b.WriteString("- " + stop + "\n")
		}
	}
	if len(revision.Constraints) > 0 {
		b.WriteString("Constraints:\n")
		for _, constraint := range revision.Constraints {
			b.WriteString("- " + constraint + "\n")
		}
	}
	b.WriteString("\nReport completion honestly; provider completion is not final acceptance.\n")
	return b.String()
}

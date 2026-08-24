// Recovery and liveness for Act & Observe (#31).
//
// Containment marks suspicion without inventing facts; reconcile decides from
// durable evidence only: a bound session that is present, never terminated,
// and has signalled proves the attempt alive (resumed); anything else cannot
// prove aliveness, so custody is released through reconcile and replacement
// happens on a NEW attempt row. Stale observations remain inspectable but can
// never mutate current truth.
package outcome

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/google/uuid"
)

// RecoverAttempt applies one owner-directed recovery verb:
//
//   - contain: mark suspicion; custody stays held; nothing is decided.
//   - reconcile: auto-verdict from heartbeat evidence — proven-alive records
//     `resumed`; otherwise the attempt becomes lost, its fence releases with
//     a reason, and a replacement_attempt receipt lands.
//   - resume: like reconcile's proven-alive path, plus paused -> running.
//   - replace: force the lost+release+receipt verdict so a new attempt may
//     acquire the fence.
//   - attention: record needs_attention without deciding custody.
func (s *Service) RecoverAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID, in RecoveryInput) (RecoveryView, error) {
	if !in.Action.Valid() {
		return RecoveryView{}, apierr.Invalid("RECOVERY_ACTION_INVALID",
			"Recovery action must be contain, reconcile, resume, replace, or attention", nil)
	}
	attempt, _, err := s.requireAttempt(ctx, outcomeID, attemptID)
	if err != nil {
		return RecoveryView{}, err
	}
	switch in.Action {
	case RecoveryActionContain:
		return s.containAttempt(ctx, attempt)
	case RecoveryActionReconcile:
		return s.reconcileAttempt(ctx, attempt)
	case RecoveryActionResume:
		return s.recoveryResume(ctx, outcomeID, attemptID, attempt)
	case RecoveryActionReplace:
		return s.recoveryReplace(ctx, attempt)
	case RecoveryActionAttention:
		return s.recoveryAttention(ctx, attempt)
	}
	return RecoveryView{}, apierr.Invalid("RECOVERY_ACTION_INVALID", "Unknown recovery action", nil)
}

// containAttempt records suspicion. It mutates NO stored status: containment
// is an observation, and the open fence already blocks duplicate admission.
func (s *Service) containAttempt(ctx context.Context, attempt domain.Attempt) (RecoveryView, error) {
	switch attempt.Status {
	case domain.AttemptQueued, domain.AttemptRunning, domain.AttemptPaused:
	default:
		return RecoveryView{}, apierr.Conflict("ATTEMPT_ALREADY_ENDED",
			fmt.Sprintf("Attempt already ended as %s; nothing to contain", attempt.Status),
			map[string]any{"status": string(attempt.Status)})
	}
	payload := mustJSON(map[string]any{"reason": "liveness suspect"})
	if _, err := s.store.AppendAttemptObservation(ctx, attempt.ID, domain.ObservationAttemptContained, payload, s.clock()); err != nil {
		return RecoveryView{}, err
	}
	view, err := s.GetAttempt(ctx, attempt.OutcomeID, attempt.ID)
	if err != nil {
		return RecoveryView{}, err
	}
	return RecoveryView{Attempt: view}, nil
}

// reconcileAttempt auto-verdicts from durable evidence.
func (s *Service) reconcileAttempt(ctx context.Context, attempt domain.Attempt) (RecoveryView, error) {
	facts, err := s.heartbeatFacts(ctx, attempt.ID)
	if err != nil {
		return RecoveryView{}, err
	}
	if attempt.Status == domain.AttemptRunning && facts.alive() {
		receipt, err := s.recordReceipt(ctx, attempt.ID, domain.RecoveryResumed, map[string]any{
			"evidence": "bound session present, signalled, not terminated",
		})
		if err != nil {
			return RecoveryView{}, err
		}
		view, err := s.GetAttempt(ctx, attempt.OutcomeID, attempt.ID)
		if err != nil {
			return RecoveryView{}, err
		}
		return RecoveryView{Attempt: view, Receipt: receipt}, nil
	}
	return s.forceLost(ctx, attempt, domain.RecoveryReplacement, "reconcile could not prove liveness")
}

// recoveryResume proves liveness first, then returns the attempt to running.
func (s *Service) recoveryResume(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID, attempt domain.Attempt) (RecoveryView, error) {
	if attempt.Status != domain.AttemptPaused && attempt.Status != domain.AttemptRunning {
		return RecoveryView{}, apierr.Conflict(CodeAttemptLivenessUnproven,
			fmt.Sprintf("A %s attempt cannot be resumed in place", attempt.Status),
			map[string]any{"status": string(attempt.Status)})
	}
	facts, err := s.heartbeatFacts(ctx, attempt.ID)
	if err != nil {
		return RecoveryView{}, err
	}
	harness := domain.HarnessCodex
	if ref, ok, err := s.store.LatestAttemptSessionRef(ctx, attemptID); err != nil {
		return RecoveryView{}, err
	} else if ok && ref.Harness != "" {
		harness = ref.Harness
	}
	projectID, _, err := s.store.GetOutcomeProjectID(ctx, attempt.OutcomeID)
	if err != nil {
		return RecoveryView{}, err
	}
	if !facts.alive() {
		return RecoveryView{}, apierr.Conflict(CodeAttemptLivenessUnproven,
			"Liveness is unproven — replace the attempt instead of resuming it",
			map[string]any{"attemptId": string(attemptID)})
	}
	if err := s.probeReadiness(ctx, projectID, harness); err != nil {
		return RecoveryView{}, err
	}
	if attempt.Status == domain.AttemptPaused {
		rows, err := s.store.TransitionAttemptStatus(ctx, outcomeID, attemptID, domain.AttemptPaused, domain.AttemptRunning, s.clock())
		if err != nil {
			return RecoveryView{}, err
		}
		if rows == 0 {
			return RecoveryView{}, apierr.Conflict("ATTEMPT_STATUS_MOVED", "The attempt changed state concurrently; reload and retry", nil)
		}
	}
	receipt, err := s.recordReceipt(ctx, attempt.ID, domain.RecoveryResumed, map[string]any{
		"evidence": "bound session proven alive; readiness re-probed",
	})
	if err != nil {
		return RecoveryView{}, err
	}
	view, err := s.GetAttempt(ctx, attempt.OutcomeID, attempt.ID)
	if err != nil {
		return RecoveryView{}, err
	}
	return RecoveryView{Attempt: view, Receipt: receipt}, nil
}

// recoveryReplace forces the lost verdict and hands custody back so the next
// StartAttempt may issue a fresh fence. Replacement is always a NEW row.
func (s *Service) recoveryReplace(ctx context.Context, attempt domain.Attempt) (RecoveryView, error) {
	switch attempt.Status {
	case domain.AttemptQueued, domain.AttemptRunning, domain.AttemptPaused:
		return s.forceLost(ctx, attempt, domain.RecoveryReplacement, "owner directed replacement")
	case domain.AttemptLost:
		// Custody may already be released by reconcile; make sure it is, then
		// stamp the replacement receipt idempotently.
		if err := s.releaseCustody(ctx, attempt.ID, "replacement_attempt"); err != nil {
			return RecoveryView{}, err
		}
		receipt, err := s.recordReceipt(ctx, attempt.ID, domain.RecoveryReplacement, map[string]any{
			"evidence": "attempt already lost; custody handed to replacement",
		})
		if err != nil {
			return RecoveryView{}, err
		}
		view, err := s.GetAttempt(ctx, attempt.OutcomeID, attempt.ID)
		if err != nil {
			return RecoveryView{}, err
		}
		return RecoveryView{Attempt: view, Receipt: receipt}, nil
	default:
		return RecoveryView{}, apierr.Conflict("ATTEMPT_ALREADY_ENDED",
			fmt.Sprintf("Attempt already ended as %s", attempt.Status),
			map[string]any{"status": string(attempt.Status)})
	}
}

// recoveryAttention escalates to the owner without mutating status or custody.
func (s *Service) recoveryAttention(ctx context.Context, attempt domain.Attempt) (RecoveryView, error) {
	payload := mustJSON(map[string]any{"reason": "owner escalation"})
	if _, err := s.store.AppendAttemptObservation(ctx, attempt.ID, domain.ObservationRecoveryAttention, payload, s.clock()); err != nil {
		return RecoveryView{}, err
	}
	receipt, err := s.recordReceipt(ctx, attempt.ID, domain.RecoveryNeedsAttention, map[string]any{
		"evidence": "owner escalation recorded",
	})
	if err != nil {
		return RecoveryView{}, err
	}
	view, err := s.GetAttempt(ctx, attempt.OutcomeID, attempt.ID)
	if err != nil {
		return RecoveryView{}, err
	}
	return RecoveryView{Attempt: view, Receipt: receipt}, nil
}

// EvaluateAttemptLiveness is the daemon reconcile-loop hook. For every
// RUNNING attempt whose bound provider session is durably terminated, it
// records the exit as an ordered observation and moves the attempt to
// `reconciled`: ended and accounted for, result unclassified. Missing or
// stale heartbeats mutate NOTHING here — they derive as unconfirmed at read
// time and wait for explicit contain/reconcile.
func (s *Service) EvaluateAttemptLiveness(ctx context.Context) error {
	running, err := s.store.ListAttemptsByStatus(ctx, domain.AttemptRunning)
	if err != nil {
		return fmt.Errorf("list running attempts: %w", err)
	}
	var failures []error
	for _, attempt := range running {
		facts, err := s.heartbeatFacts(ctx, attempt.ID)
		if err != nil {
			failures = append(failures, fmt.Errorf("attempt %s: %w", attempt.ID, err))
			continue
		}
		if !facts.terminated() {
			continue
		}
		rows, err := s.store.TransitionAttemptStatus(ctx, attempt.OutcomeID, attempt.ID,
			domain.AttemptRunning, domain.AttemptReconciled, s.clock())
		if err != nil {
			failures = append(failures, fmt.Errorf("attempt %s: %w", attempt.ID, err))
			continue
		}
		if rows == 0 {
			continue // moved concurrently; next tick sees the truth
		}
		payload := mustJSON(map[string]any{
			"sessionId": facts.sessionID,
			"outcome":   "provider session ended; result unclassified",
		})
		if _, err := s.store.AppendAttemptObservation(ctx, attempt.ID, domain.ObservationProviderExit, payload, s.clock()); err != nil {
			failures = append(failures, fmt.Errorf("observation for %s: %w", attempt.ID, err))
		}
	}
	switch len(failures) {
	case 0:
		return nil
	case 1:
		return failures[0]
	default:
		return fmt.Errorf("attempt liveness evaluation: %d failures: %v", len(failures), failures[0])
	}
}

// attemptFacts pairs derived heartbeat facts with the binding they came from.
type attemptFacts struct {
	sessionID string
	facts     domain.SessionHeartbeatFacts
}

// alive reports provable liveness: a present session row that has signalled
// and is NOT terminated. Missing heartbeat = unconfirmed, never dead.
func (f attemptFacts) alive() bool {
	return f.facts.Present && !f.facts.IsTerminated && !f.facts.FirstSignalAt.IsZero()
}

func (f attemptFacts) terminated() bool {
	return f.facts.Present && f.facts.IsTerminated
}

// heartbeatFacts resolves the bound session's heartbeat facts for an attempt.
func (s *Service) heartbeatFacts(ctx context.Context, attemptID domain.AttemptID) (attemptFacts, error) {
	ref, ok, err := s.store.LatestAttemptSessionRef(ctx, attemptID)
	if err != nil {
		return attemptFacts{}, err
	}
	if !ok || s.heartbeats == nil {
		return attemptFacts{}, nil
	}
	rec, present, err := s.heartbeats.GetSession(ctx, domain.SessionID(ref.SessionID))
	if err != nil {
		return attemptFacts{}, err
	}
	if !present {
		return attemptFacts{sessionID: ref.SessionID}, nil
	}
	return attemptFacts{
		sessionID: ref.SessionID,
		facts: domain.SessionHeartbeatFacts{
			Present:       true,
			ActivityState: rec.Activity.State,
			FirstSignalAt: rec.FirstSignalAt,
			IsTerminated:  rec.IsTerminated,
		},
	}, nil
}

// forceLost walks the lost verdict: guarded transition to lost (legal from
// queued/running/paused), custody release with a reason, and the receipt.
func (s *Service) forceLost(ctx context.Context, attempt domain.Attempt, resolution domain.RecoveryResolution, evidence string) (RecoveryView, error) {
	rows, err := s.store.TransitionAttemptStatus(ctx, attempt.OutcomeID, attempt.ID, attempt.Status, domain.AttemptLost, s.clock())
	if err != nil {
		return RecoveryView{}, err
	}
	if rows == 0 && attempt.Status != domain.AttemptLost {
		return RecoveryView{}, apierr.Conflict("ATTEMPT_STATUS_MOVED", "The attempt changed state concurrently; reload and retry", nil)
	}
	if err := s.releaseCustody(ctx, attempt.ID, "reconciled_lost"); err != nil {
		return RecoveryView{}, err
	}
	receipt, err := s.recordReceipt(ctx, attempt.ID, resolution, map[string]any{"evidence": evidence})
	if err != nil {
		return RecoveryView{}, err
	}
	view, err := s.GetAttempt(ctx, attempt.OutcomeID, attempt.ID)
	if err != nil {
		return RecoveryView{}, err
	}
	return RecoveryView{Attempt: view, Receipt: receipt}, nil
}

// releaseCustody releases the attempt's open fence; releasing without holding
// one is a tolerated no-op so recovery verbs stay idempotent.
func (s *Service) releaseCustody(ctx context.Context, attemptID domain.AttemptID, reason string) error {
	_, err := s.store.ReleaseFenceForAttempt(ctx, attemptID, reason, s.clock())
	return err
}

func (s *Service) recordReceipt(ctx context.Context, attemptID domain.AttemptID, resolution domain.RecoveryResolution, detail map[string]any) (*domain.AttemptRecoveryReceipt, error) {
	receipt := domain.AttemptRecoveryReceipt{
		ID:         "rcpt-" + uuid.NewString(),
		AttemptID:  attemptID,
		Resolution: resolution,
		Detail:     mustJSON(detail),
		CreatedAt:  s.clock(),
	}
	if err := s.store.CreateRecoveryReceipt(ctx, receipt); err != nil {
		return nil, err
	}
	return &receipt, nil
}

func mustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

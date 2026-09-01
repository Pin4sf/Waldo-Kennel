// Recovery and liveness for Act & Observe (#31).
//
// Containment marks suspicion without inventing facts; reconcile decides from
// durable evidence only. CUSTODY LAW: a fence may be released ONLY once the
// bound provider's stop is proven — a durably terminated session, or an
// explicit owner assertion that is recorded as its own containment
// observation. Stored status alone is never proof. Unproven liveness
// escalates as needs_attention; it NEVER releases custody, so a replacement
// can never write beside a possibly-live original provider. Stale
// observations remain inspectable but can never mutate current truth.
package outcome

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
)

// RecoverAttempt applies one owner-directed recovery verb:
//
//   - contain: mark suspicion; custody stays held; nothing is decided.
//   - reconcile: auto-verdict from heartbeat evidence — proven-alive records
//     `resumed`; otherwise the attempt becomes lost, its fence releases with
//     a reason, and a replacement_attempt receipt lands.
//   - replace: force the lost+release+receipt verdict so a new attempt may
//     acquire the fence.
//   - attention: record needs_attention without deciding custody.
//
// There is deliberately no resume verb: nothing in #31 can command a provider
// to resume, so proving an already-running provider alive is reconcile's job.
func (s *Service) RecoverAttempt(ctx context.Context, outcomeID domain.OutcomeID, attemptID domain.AttemptID, in RecoveryInput) (RecoveryView, error) {
	if !in.Action.Valid() {
		return RecoveryView{}, apierr.Invalid("RECOVERY_ACTION_INVALID",
			"Recovery action must be contain, reconcile, replace, or attention", nil)
	}
	attempt, _, err := s.requireAttempt(ctx, outcomeID, attemptID)
	if err != nil {
		return RecoveryView{}, err
	}
	switch in.Action {
	case RecoveryActionContain:
		return s.containAttempt(ctx, attempt)
	case RecoveryActionReconcile:
		return s.reconcileAttempt(ctx, in, attempt)
	case RecoveryActionReplace:
		return s.recoveryReplace(ctx, in, attempt)
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

// custodyProof states WHY releasing a fence is safe. Custody may move only
// once the bound provider's stop is proven — the anti-duplicate-writer
// contract — or when the owner explicitly asserts containment (invariant 5).
type custodyProof struct {
	reason  string
	proven  bool
	byOwner bool
}

// proveProviderStopped resolves the release gate from durable facts:
//
//   - terminated bound session -> machine proof
//   - explicit owner assertion -> accepted and recorded as containment
//
// Stored status alone — including `failed` — is NEVER proof: a spawn
// failure's true point-of-failure is not knowable after the fact, so only
// termination facts or the owner's recorded assertion unlock custody.
func (s *Service) proveProviderStopped(facts attemptFacts, ownerConfirmed bool) custodyProof {
	switch {
	case facts.facts.Present && facts.facts.IsTerminated:
		return custodyProof{reason: "bound session durably terminated", proven: true}
	case ownerConfirmed:
		return custodyProof{reason: "owner asserted the provider is stopped", proven: true, byOwner: true}
	default:
		return custodyProof{}
	}
}

// refuseUnprovenCustody records the escalation receipt and refuses release.
func (s *Service) refuseUnprovenCustody(ctx context.Context, attempt domain.Attempt) error {
	receipt, rErr := s.recordReceipt(ctx, attempt.ID, domain.RecoveryNeedsAttention, map[string]any{
		"evidence": "cannot prove the bound provider stopped — terminate it or confirm containment",
	})
	if rErr != nil {
		return fmt.Errorf("record unproven-custody receipt for %s: %w", attempt.ID, rErr)
	}
	return apierr.Conflict(CodeAttemptCustodyUnproven,
		"Custody stays held: the bound provider's stop is not proven. Terminate it, then reconcile — or replace with explicit confirmation",
		map[string]any{"attemptId": string(attempt.ID), "receiptId": receipt.ID})
}

// maybeRecordOwnerContainment makes an owner stop-assertion inspectable.
func (s *Service) maybeRecordOwnerContainment(ctx context.Context, attempt domain.Attempt, proof custodyProof) error {
	if !proof.byOwner {
		return nil
	}
	payload := mustJSON(map[string]any{"assertion": "owner asserts the bound provider is stopped"})
	_, err := s.store.AppendAttemptObservation(ctx, attempt.ID, domain.ObservationOwnerContained, payload, s.clock())
	return err
}

// reconcileAttempt auto-verdicts from durable evidence. Every custody release
// requires proved provider stop; anything else escalates without deciding.
func (s *Service) reconcileAttempt(ctx context.Context, in RecoveryInput, attempt domain.Attempt) (RecoveryView, error) {
	if attempt.Status == domain.AttemptSucceeded {
		return RecoveryView{}, apierr.Conflict("ATTEMPT_ALREADY_ENDED",
			fmt.Sprintf("Attempt already ended as %s", attempt.Status),
			map[string]any{"status": string(attempt.Status)})
	}
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
	proof := s.proveProviderStopped(facts, in.ConfirmProviderStopped)
	if !proof.proven {
		return RecoveryView{}, s.refuseUnprovenCustody(ctx, attempt)
	}
	if err := s.maybeRecordOwnerContainment(ctx, attempt, proof); err != nil {
		return RecoveryView{}, err
	}
	switch attempt.Status {
	case domain.AttemptFailed, domain.AttemptCancelled, domain.AttemptReconciled:
		// Terminal by truthful record: never rewrite to lost. Account for the
		// predecessor and hand custody to the replacement lineage.
		return s.accountTerminalCustody(ctx, attempt, proof.reason)
	default:
		return s.forceLost(ctx, attempt, domain.RecoveryReplacement,
			"reconcile could not prove liveness; provider stop "+proof.reason)
	}
}

// accountTerminalCustody releases the fence a TERMINAL predecessor still
// holds (failed-before-spawn, owner-cancelled, ended-unclassified) without
// mutating its immutable record, then stamps the replacement receipt. This is
// the reconcile -> release(old) -> issue(new) path D4 guarantees; without it a
// cancelled or failed attempt would hold worktree custody forever.
func (s *Service) accountTerminalCustody(ctx context.Context, attempt domain.Attempt, reason string) (RecoveryView, error) {
	if err := s.releaseCustody(ctx, attempt.ID, "reconciled_terminal_"+string(attempt.Status)); err != nil {
		return RecoveryView{}, err
	}
	receipt, err := s.recordReceipt(ctx, attempt.ID, domain.RecoveryReplacement, map[string]any{
		"evidence":       "terminal predecessor accounted for (" + reason + "); custody handed to replacement",
		"previousStatus": string(attempt.Status),
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
func (s *Service) recoveryReplace(ctx context.Context, in RecoveryInput, attempt domain.Attempt) (RecoveryView, error) {
	facts, fErr := s.heartbeatFacts(ctx, attempt.ID)
	if fErr != nil {
		return RecoveryView{}, fErr
	}
	proof := s.proveProviderStopped(facts, in.ConfirmProviderStopped)
	if !proof.proven {
		return RecoveryView{}, s.refuseUnprovenCustody(ctx, attempt)
	}
	if err := s.maybeRecordOwnerContainment(ctx, attempt, proof); err != nil {
		return RecoveryView{}, err
	}
	switch attempt.Status {
	case domain.AttemptQueued, domain.AttemptRunning, domain.AttemptPaused:
		return s.forceLost(ctx, attempt, domain.RecoveryReplacement,
			"owner directed replacement; provider stop "+proof.reason)
	case domain.AttemptLost, domain.AttemptFailed, domain.AttemptCancelled, domain.AttemptReconciled:
		// Custody may already be released by reconcile; make sure it is, then
		// stamp the replacement receipt idempotently. Terminal records are
		// never rewritten — only their custody moves.
		if err := s.releaseCustody(ctx, attempt.ID, "replacement_attempt"); err != nil {
			return RecoveryView{}, err
		}
		receipt, err := s.recordReceipt(ctx, attempt.ID, domain.RecoveryReplacement, map[string]any{
			"evidence":       "predecessor accounted for (" + proof.reason + "); custody handed to replacement",
			"previousStatus": string(attempt.Status),
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
		// Health-gated lease: only PROVABLY alive custodians renew their
		// fence stamp. Unhealthy/unproven attempts keep their OLD stamp — a
		// stale renewal is exactly how custody that may outlive its provider
		// becomes visible.
		if facts.alive() {
			if _, rErr := s.store.RenewFenceForAttempt(ctx, attempt.ID, s.clock()); rErr != nil {
				failures = append(failures, fmt.Errorf("renew fence for %s: %w", attempt.ID, rErr))
			}
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
		return fmt.Errorf("attempt liveness evaluation: %d failures: %w", len(failures), errors.Join(failures...))
	}
}

// attemptFacts pairs derived heartbeat facts with the binding they came from.
type attemptFacts struct {
	sessionID string
	facts     domain.SessionHeartbeatFacts
}

// alive reports PROVABLE current liveness: present, signalled, not
// terminated, and durably active inside the staleness window (sticky states
// exempt). A session that vanished long ago is NOT alive forever.
func (f attemptFacts) alive() bool {
	if !f.facts.Present || f.facts.IsTerminated || f.facts.FirstSignalAt.IsZero() {
		return false
	}
	return f.facts.RecentlyActive(domain.LivenessPolicy{
		Now:                 time.Now().UTC(),
		StaleHeartbeatAfter: domain.DefaultStaleHeartbeatWindow,
	})
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
			Present:        true,
			ActivityState:  rec.Activity.State,
			FirstSignalAt:  rec.FirstSignalAt,
			LastActivityAt: rec.Activity.LastActivityAt,
			IsTerminated:   rec.IsTerminated,
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

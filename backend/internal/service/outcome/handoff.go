package outcome

import (
	"context"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
)

// Stable refusals for a successor that cannot be given its predecessors' work.
const (
	// CodeUpstreamArtifactMissing means a dependency produced nothing that was
	// retained, so there is nothing to hand down.
	CodeUpstreamArtifactMissing = "UPSTREAM_ARTIFACT_MISSING"
	// CodeUpstreamArtifactIncomplete means a dependency's snapshot hit a bound
	// or held something it could not represent.
	CodeUpstreamArtifactIncomplete = "UPSTREAM_ARTIFACT_INCOMPLETE"
	// CodeUpstreamArtifactUnreviewed means a dependency's result was never
	// frozen, so what it holds can still change under the successor.
	CodeUpstreamArtifactUnreviewed = "UPSTREAM_ARTIFACT_UNREVIEWED"
	// CodeUpstreamLineageMismatch means a retained receipt does not belong to
	// the attempt or plan the successor is being admitted under.
	CodeUpstreamLineageMismatch = "UPSTREAM_LINEAGE_MISMATCH"
)

// resolveUpstreamReceipts returns, in dependency order, the exact retained
// results a successor must be given before it may start.
//
// This is the admission half of artifact continuity. Starting a successor from
// a fresh worktree off the original branch is not a handoff: the predecessor's
// work would simply be absent, and the successor would redo or contradict it
// without anyone being told. So a dependency that has no retained, complete,
// frozen result blocks admission with a named reason rather than being started
// on an empty workspace.
//
// A unit with no dependencies resolves to nothing, which is not an error.
func (s *Service) resolveUpstreamReceipts(
	ctx context.Context,
	plan domain.PlanRevision,
	unit domain.WorkUnit,
) ([]domain.AttemptReceipt, error) {
	if len(unit.DependsOn) == 0 {
		return nil, nil
	}
	if s.receipts == nil {
		// Refusing is the only truthful option: the successor's inputs cannot be
		// established at all, so admitting it would start work on an unknown base.
		return nil, apierr.Conflict(CodeUpstreamArtifactMissing,
			"Artifact retention is unavailable, so this WorkUnit's inputs cannot be established",
			map[string]any{"workUnitId": unit.ID})
	}
	attempts, err := s.store.ListAttempts(ctx, plan.OutcomeID)
	if err != nil {
		return nil, err
	}
	return upstreamReceiptsFor(unit, attemptsForPlan(plan, attempts), func(id domain.AttemptID) (domain.AttemptReceipt, bool, error) {
		return s.receipts.GetAttemptReceipt(ctx, id)
	})
}

// upstreamReceiptsFor is the decision itself, separated from the two store
// reads so every refusal is exercised directly rather than through a fake of
// the whole Outcome store.
func upstreamReceiptsFor(
	unit domain.WorkUnit,
	scoped []domain.Attempt,
	lookup func(domain.AttemptID) (domain.AttemptReceipt, bool, error),
) ([]domain.AttemptReceipt, error) {
	receipts := make([]domain.AttemptReceipt, 0, len(unit.DependsOn))
	for _, dependencyID := range unit.DependsOn {
		producer, ok := producingAttempt(dependencyID, scoped)
		if !ok {
			return nil, apierr.Conflict(CodeUpstreamArtifactMissing,
				"A WorkUnit this one depends on has not produced a proved result yet",
				map[string]any{"workUnitId": unit.ID, "dependencyId": dependencyID})
		}
		receipt, found, err := lookup(producer.ID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, apierr.Conflict(CodeUpstreamArtifactMissing,
				"The Attempt that satisfied a dependency retained no result",
				map[string]any{"workUnitId": unit.ID, "dependencyId": dependencyID, "attemptId": producer.ID})
		}
		if err := upstreamReceiptUsable(unit, dependencyID, producer, receipt); err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}
	return receipts, nil
}

// producingAttempt finds the succeeded attempt of a dependency WorkUnit.
//
// Success is the only status that may hand work down: it is the one status
// derived from retained bytes plus proof about those bytes. A reconciled
// attempt has ended without being classified, and its output may still change.
func producingAttempt(unitID domain.WorkUnitID, scoped []domain.Attempt) (domain.Attempt, bool) {
	var latest domain.Attempt
	var found bool
	for _, attempt := range attemptsForWorkUnit(unitID, scoped) {
		if attempt.Status != domain.AttemptSucceeded {
			continue
		}
		if !found || attempt.Number > latest.Number {
			latest, found = attempt, true
		}
	}
	return latest, found
}

// upstreamReceiptUsable refuses a receipt that cannot honestly be handed down.
func upstreamReceiptUsable(unit domain.WorkUnit, dependencyID domain.WorkUnitID, producer domain.Attempt, receipt domain.AttemptReceipt) error {
	detail := map[string]any{
		"workUnitId": unit.ID, "dependencyId": dependencyID,
		"attemptId": producer.ID, "artifactVersion": receipt.ArtifactVersion,
	}
	if err := receipt.Validate(); err != nil {
		detail["detail"] = err.Error()
		return apierr.Conflict(CodeUpstreamLineageMismatch, "A predecessor's retained result is not a valid receipt", detail)
	}
	// The receipt must describe the attempt that actually produced it. A
	// mismatch means the successor would inherit somebody else's work.
	if receipt.AttemptID != producer.ID || receipt.WorkUnitID != dependencyID ||
		receipt.PlanRevisionID != producer.PlanRevisionID ||
		receipt.ContractRevisionNumber != producer.ContractRevisionNumber {
		return apierr.Conflict(CodeUpstreamLineageMismatch,
			"A predecessor's retained result does not belong to the Attempt that satisfied the dependency", detail)
	}
	if !receipt.RetentionState.Complete() {
		detail["retentionState"] = string(receipt.RetentionState)
		detail["retentionDetail"] = receipt.RetentionDetail
		return apierr.Conflict(CodeUpstreamArtifactIncomplete,
			fmt.Sprintf("A predecessor's result was retained as %s, so it cannot be handed to this WorkUnit", receipt.RetentionState), detail)
	}
	if !receipt.Frozen() {
		// An unfrozen receipt can still be replaced by a later retention pass,
		// so the successor could be built on bytes that change underneath it.
		return apierr.Conflict(CodeUpstreamArtifactUnreviewed,
			"A predecessor's result is not frozen yet, so it could still change under this WorkUnit", detail)
	}
	return nil
}

// requireUpstreamArtifacts admits a successor only when every dependency's
// exact result is available to hand down.
//
// It resolves the receipts and keeps only the verdict. Materializing those
// bytes into the successor's workspace is the other half of continuity and is
// not wired yet, so this deliberately does not pretend to deliver them -- it
// refuses the cases where delivery would be impossible or would silently hand
// over the wrong work.
func (s *Service) requireUpstreamArtifacts(ctx context.Context, plan domain.PlanRevision, unit domain.WorkUnit) error {
	_, err := s.resolveUpstreamReceipts(ctx, plan, unit)
	return err
}

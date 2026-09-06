-- Make analysis expiry one durable terminal fact.
--
-- Before this migration, the service closed intake_analysis_requests first and
-- then separately called FailIntakeAnalysis. A daemon stop or storage error
-- between those writes could leave the request durably `expired` while the
-- intake itself remained `analyzing`. On restart the renderer would therefore
-- reconstruct a waiting intake even though the provider ask had already ended.
--
-- SQLite triggers are already the persistence-level invariant mechanism for
-- this relation (see 0110's freeze trigger) and for CDC. Keeping this transition
-- here makes request expiry and the matching intake failure part of the same
-- UPDATE statement/transaction. The revision + status fence prevents a stale
-- request from regressing an intake whose owner has already moved on.

-- +goose Up
-- +goose StatementBegin
CREATE TRIGGER intake_analysis_request_expiry_fails_matching_intake
AFTER UPDATE OF status ON intake_analysis_requests
WHEN OLD.status = 'requested'
     AND NEW.status = 'expired'
BEGIN
    UPDATE intake_sessions
    SET status = 'analysis_failed',
        failure_code = 'INTAKE_ANALYSIS_EXPIRED',
        updated_at = COALESCE(NEW.answered_at, datetime('now'))
    WHERE id = NEW.intake_id
      AND current_proposal_revision = NEW.expected_proposal_revision
      AND status = 'analyzing';
END;
-- +goose StatementEnd

-- Heal the exact split state created by pre-0112 builds. Only the latest ask
-- for the intake's CURRENT proposal revision may drive the repair; if the owner
-- has already started a newer retry, that newer request wins and remains live.
-- +goose StatementBegin
UPDATE intake_sessions
SET status = 'analysis_failed',
    failure_code = 'INTAKE_ANALYSIS_EXPIRED',
    updated_at = COALESCE(
        (
            SELECT latest.answered_at
            FROM intake_analysis_requests AS latest
            WHERE latest.intake_id = intake_sessions.id
              AND latest.expected_proposal_revision = intake_sessions.current_proposal_revision
            ORDER BY latest.created_at DESC, latest.id DESC
            LIMIT 1
        ),
        intake_sessions.updated_at
    )
WHERE intake_sessions.status = 'analyzing'
  AND 'expired' = (
      SELECT latest.status
      FROM intake_analysis_requests AS latest
      WHERE latest.intake_id = intake_sessions.id
        AND latest.expected_proposal_revision = intake_sessions.current_proposal_revision
      ORDER BY latest.created_at DESC, latest.id DESC
      LIMIT 1
  );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS intake_analysis_request_expiry_fails_matching_intake;
-- +goose StatementEnd

-- Down intentionally does not resurrect rows repaired by the one-time backfill:
-- once the durable provider ask is expired, returning its intake to `analyzing`
-- would recreate the inconsistent state this migration removes.

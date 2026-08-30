-- Model-authored Contract proposals, phase 2: durable requests for an
-- agent-authored intake proposal.
--
-- The daemon has no synchronous model call, so a proposal arrives later over
-- the API from a session the daemon spawned. This relation is what makes that
-- callback addressable, bounded, single-use, and expiring. It mirrors
-- decomposition_requests (0109) rather than inventing a second vocabulary for
-- the same mechanism.
--
-- NOTE ON `intake_sessions.status`: this deliberately adds NO new status.
-- An intake with an agent working on it stays `analyzing`, and "an agent is
-- working" is read from THIS relation, which is the durable fact. A second
-- representation of the same fact in a CHECK-constrained column could
-- disagree with it, and altering that CHECK would mean rebuilding a table four
-- relations and the CDC writers point at. What made a new status look
-- necessary was the startup sweep; that is fixed where it belongs, in the
-- sweep's own WHERE clause.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE intake_analysis_requests (
    id                         TEXT PRIMARY KEY,
    intake_id                  TEXT NOT NULL REFERENCES intake_sessions (id),
    expected_proposal_revision INTEGER NOT NULL CHECK (expected_proposal_revision >= 0),
    status                     TEXT NOT NULL CHECK (status IN ('requested','fulfilled','rejected','expired','cancelled')),
    callback_token_digest      TEXT NOT NULL CHECK (length(callback_token_digest) = 64),
    session_id                 TEXT NOT NULL DEFAULT '',
    harness                    TEXT NOT NULL DEFAULT '',
    expires_at                 TIMESTAMP NOT NULL,
    raw_proposal               TEXT NOT NULL DEFAULT '',
    refusal_reason             TEXT NOT NULL DEFAULT '',
    created_at                 TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    answered_at                TIMESTAMP
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_intake_analysis_requests_intake
    ON intake_analysis_requests (intake_id, created_at);
-- +goose StatementEnd

-- A request is frozen except for two changes: binding the session that was
-- spawned to answer it, and its one-way move out of 'requested'.
--
-- The session binding has to be a separate update because the row is written
-- BEFORE the spawn — an agent holding a token for an unrecorded request would
-- have nowhere to answer — so the session id does not exist yet at insert.
-- Nothing may reopen an answered ask, which is what keeps the callback
-- single-use.
-- +goose StatementBegin
CREATE TRIGGER intake_analysis_requests_freeze_update
BEFORE UPDATE ON intake_analysis_requests
WHEN OLD.status <> 'requested'
     OR NEW.id <> OLD.id
     OR NEW.intake_id <> OLD.intake_id
     OR NEW.expected_proposal_revision <> OLD.expected_proposal_revision
     OR NEW.callback_token_digest <> OLD.callback_token_digest
     OR NEW.expires_at <> OLD.expires_at
     -- While still open, the ONLY permitted change is the session binding.
     OR (NEW.status = 'requested' AND NEW.session_id IS OLD.session_id)
BEGIN SELECT RAISE(ABORT, 'an intake analysis request is frozen except for its session binding and its one-way answer'); END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS intake_analysis_requests_freeze_update;
DROP INDEX IF EXISTS idx_intake_analysis_requests_intake;
DROP TABLE IF EXISTS intake_analysis_requests;
-- +goose StatementEnd

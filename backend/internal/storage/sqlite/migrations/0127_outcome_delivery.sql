-- +goose Up

-- Durable owner-triggered transfer of one exact retained Attempt result.
-- Filesystem bytes live in the artifact store; this row is the restart-safe
-- request/result ledger and never claims success while state is pending.
CREATE TABLE outcome_deliveries (
    id                     TEXT PRIMARY KEY,
    outcome_id             TEXT NOT NULL REFERENCES outcomes (id),
    attempt_id             TEXT NOT NULL REFERENCES attempts (id),
    work_unit_id           TEXT NOT NULL REFERENCES work_units (id),
    artifact_version       TEXT NOT NULL,
    disposition             TEXT NOT NULL CHECK (disposition IN ('accepted', 'draft')),
    destination             TEXT NOT NULL,
    acceptance_decision_id TEXT NOT NULL DEFAULT '',
    request_key            TEXT NOT NULL,
    request_fingerprint    TEXT NOT NULL,
    state                  TEXT NOT NULL CHECK (state IN ('pending', 'succeeded', 'failed', 'cancelled')),
    manifest_path          TEXT NOT NULL DEFAULT '',
    file_count             INTEGER NOT NULL DEFAULT 0 CHECK (file_count >= 0),
    byte_count             INTEGER NOT NULL DEFAULT 0 CHECK (byte_count >= 0),
    failure_code           TEXT NOT NULL DEFAULT '',
    failure_detail         TEXT NOT NULL DEFAULT '',
    requested_at           TIMESTAMP NOT NULL,
    completed_at           TIMESTAMP
);

CREATE UNIQUE INDEX idx_outcome_deliveries_request_key
    ON outcome_deliveries (request_key);
CREATE INDEX idx_outcome_deliveries_outcome
    ON outcome_deliveries (outcome_id, requested_at, id);

-- Identity and request provenance are immutable. A pending row may move once
-- to a terminal state, but a terminal result can never be rewritten or deleted.
-- +goose StatementBegin
CREATE TRIGGER outcome_deliveries_immutable_update
BEFORE UPDATE ON outcome_deliveries
WHEN OLD.id <> NEW.id
  OR OLD.outcome_id <> NEW.outcome_id
  OR OLD.attempt_id <> NEW.attempt_id
  OR OLD.work_unit_id <> NEW.work_unit_id
  OR OLD.artifact_version <> NEW.artifact_version
  OR OLD.disposition <> NEW.disposition
  OR OLD.destination <> NEW.destination
  OR OLD.acceptance_decision_id <> NEW.acceptance_decision_id
  OR OLD.request_key <> NEW.request_key
  OR OLD.request_fingerprint <> NEW.request_fingerprint
  OR OLD.requested_at <> NEW.requested_at
  OR OLD.state <> 'pending'
  OR (OLD.state = 'pending' AND NEW.state = 'pending')
  OR (OLD.state = 'pending' AND NEW.completed_at IS NULL)
BEGIN
    SELECT RAISE(ABORT, 'delivery identity and terminal results are immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER outcome_deliveries_immutable_delete
BEFORE DELETE ON outcome_deliveries
BEGIN
    SELECT RAISE(ABORT, 'delivery records are immutable');
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS outcome_deliveries_immutable_delete;
DROP TRIGGER IF EXISTS outcome_deliveries_immutable_update;
DROP INDEX IF EXISTS idx_outcome_deliveries_outcome;
DROP INDEX IF EXISTS idx_outcome_deliveries_request_key;
DROP TABLE IF EXISTS outcome_deliveries;

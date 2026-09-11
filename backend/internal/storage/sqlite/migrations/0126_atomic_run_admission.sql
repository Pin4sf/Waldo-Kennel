-- +goose Up

-- The replay key identifies one complete owner command, not merely its
-- resulting desired state. Existing rows retain an empty fingerprint and are
-- treated conservatively by the service when replayed.
ALTER TABLE outcome_run_intents ADD COLUMN request_fingerprint TEXT NOT NULL DEFAULT '';

-- An Attempt records which durable run authorization admitted it. Zero keeps
-- compatibility with individually started Attempts before run intent existed.
ALTER TABLE attempts ADD COLUMN run_intent_generation INTEGER NOT NULL DEFAULT 0
    CHECK (run_intent_generation >= 0);

-- A reserved check is owned by a daemon invocation epoch. A row alone is not
-- proof that its invoker abandoned the callback.
ALTER TABLE attempt_check_runs ADD COLUMN reservation_epoch TEXT NOT NULL DEFAULT '';

-- Recreate immutable guards so the new identity fields cannot be rewritten.
DROP TRIGGER IF EXISTS outcome_run_intents_immutable_update;
-- +goose StatementBegin
CREATE TRIGGER outcome_run_intents_immutable_update
BEFORE UPDATE ON outcome_run_intents
WHEN OLD.id <> NEW.id
     OR OLD.outcome_id <> NEW.outcome_id
     OR OLD.generation <> NEW.generation
     OR OLD.desired <> NEW.desired
     OR OLD.plan_revision_id <> NEW.plan_revision_id
     OR OLD.contract_revision_number <> NEW.contract_revision_number
     OR OLD.request_key <> NEW.request_key
     OR OLD.request_fingerprint <> NEW.request_fingerprint
     OR OLD.requested_at <> NEW.requested_at
     OR (OLD.acknowledged_at IS NOT NULL AND OLD.acknowledged_at IS NOT NEW.acknowledged_at)
BEGIN
    SELECT RAISE(ABORT, 'run intent generations are immutable');
END;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS attempts_immutable_update;
-- +goose StatementBegin
CREATE TRIGGER attempts_immutable_update
BEFORE UPDATE ON attempts
WHEN OLD.id <> NEW.id
     OR OLD.outcome_id <> NEW.outcome_id
     OR OLD.plan_revision_id <> NEW.plan_revision_id
     OR OLD.work_unit_id <> NEW.work_unit_id
     OR OLD.number <> NEW.number
     OR OLD.contract_revision_number <> NEW.contract_revision_number
     OR OLD.run_intent_generation <> NEW.run_intent_generation
     OR OLD.request_key IS NOT NEW.request_key
     OR OLD.created_at <> NEW.created_at
BEGIN
    SELECT RAISE(ABORT, 'attempts are immutable');
END;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS attempt_check_runs_identity_immutable;
-- +goose StatementBegin
CREATE TRIGGER attempt_check_runs_identity_immutable
BEFORE UPDATE ON attempt_check_runs
WHEN OLD.id <> NEW.id
     OR OLD.attempt_id <> NEW.attempt_id
     OR OLD.check_id <> NEW.check_id
     OR OLD.artifact_version <> NEW.artifact_version
     OR OLD.reservation_epoch <> NEW.reservation_epoch
     OR OLD.reserved_at <> NEW.reserved_at
BEGIN
    SELECT RAISE(ABORT, 'recorded check observations are immutable');
END;
-- +goose StatementEnd

-- +goose Down
-- This migration is additive and intentionally has no destructive downgrade.

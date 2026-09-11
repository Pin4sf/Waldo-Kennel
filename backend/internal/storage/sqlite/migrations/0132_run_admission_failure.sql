-- +goose Up

-- A running authorization may be unable to admit its next WorkUnit before an
-- Attempt exists (for example, an unavailable owner-selected profile). Keep
-- that blocker on the exact generation that encountered it. A deliberate
-- pause/resume or new Start appends a clean generation instead of rewriting
-- this history.
ALTER TABLE outcome_run_intents ADD COLUMN admission_failure_code TEXT NOT NULL DEFAULT '';
ALTER TABLE outcome_run_intents ADD COLUMN admission_failure_message TEXT NOT NULL DEFAULT '';
ALTER TABLE outcome_run_intents ADD COLUMN admission_failure_detail TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(admission_failure_detail));
ALTER TABLE outcome_run_intents ADD COLUMN admission_failure_work_unit_id TEXT NOT NULL DEFAULT '';
ALTER TABLE outcome_run_intents ADD COLUMN admission_failed_at TIMESTAMP;

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
     OR ((OLD.admission_failure_code <> NEW.admission_failure_code
          OR OLD.admission_failure_message <> NEW.admission_failure_message
          OR OLD.admission_failure_detail <> NEW.admission_failure_detail
          OR OLD.admission_failure_work_unit_id <> NEW.admission_failure_work_unit_id
          OR OLD.admission_failed_at IS NOT NEW.admission_failed_at)
         AND NOT (OLD.admission_failure_code = ''
                  AND OLD.admission_failure_message = ''
                  AND OLD.admission_failure_detail = '{}'
                  AND OLD.admission_failure_work_unit_id = ''
                  AND OLD.admission_failed_at IS NULL
                  AND NEW.admission_failure_code <> ''
                  AND NEW.admission_failure_message <> ''
                  AND NEW.admission_failure_work_unit_id <> ''
                  AND NEW.admission_failed_at IS NOT NULL))
BEGIN
    SELECT RAISE(ABORT, 'run intent generations are immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER outcome_run_intents_cdc_failure
AFTER UPDATE ON outcome_run_intents
WHEN OLD.admission_failure_code = '' AND NEW.admission_failure_code <> ''
BEGIN
    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)
    VALUES (
        (SELECT rs.project_id FROM outcomes o JOIN responsibility_spaces rs ON rs.id = o.space_id WHERE o.id = NEW.outcome_id),
        NULL,
        'outcome_run_intent_changed',
        json_object('outcomeId', NEW.outcome_id, 'generation', NEW.generation, 'desired', NEW.desired, 'admissionFailed', 1),
        NEW.admission_failed_at
    );
END;
-- +goose StatementEnd

-- +goose Down
-- Additive production history is intentionally retained on downgrade.

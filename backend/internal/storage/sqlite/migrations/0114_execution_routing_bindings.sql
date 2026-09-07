-- WT3: preference-aware routing provenance and exact provider/model authority.
-- 0112 and 0113 are shipped history and intentionally remain untouched.

-- +goose Up
-- +goose StatementBegin
ALTER TABLE contract_revisions
    ADD COLUMN execution_preference_json TEXT
        CHECK (execution_preference_json IS NULL OR json_valid(execution_preference_json));

ALTER TABLE plan_revisions
    ADD COLUMN routing_decision_json TEXT
        CHECK (routing_decision_json IS NULL OR json_valid(routing_decision_json));

ALTER TABLE work_unit_provider_bindings
    ADD COLUMN model_selection TEXT
        CHECK (model_selection IS NULL OR model_selection IN ('explicit', 'provider_default'));

ALTER TABLE work_unit_provider_bindings
    ADD COLUMN model TEXT;

-- Existing 0112 rows legitimately have NULL model_selection. New WT3 writes
-- must carry explicit model semantics. This trigger validates every non-legacy
-- insert without rewriting historical truth.
DROP TRIGGER IF EXISTS work_unit_execution_binding_validate_insert;
CREATE TRIGGER work_unit_execution_binding_validate_insert
BEFORE INSERT ON work_unit_provider_bindings
WHEN NEW.model_selection IS NOT NULL
BEGIN
    SELECT CASE
        WHEN NEW.model_selection = 'explicit'
             AND (NEW.model IS NULL OR length(trim(NEW.model)) = 0)
        THEN RAISE(ABORT, 'explicit work unit model binding requires model')
        WHEN NEW.model_selection = 'provider_default'
             AND NEW.model IS NOT NULL
             AND length(trim(NEW.model)) > 0
        THEN RAISE(ABORT, 'provider-default work unit model binding must not name model')
    END;
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS work_unit_execution_binding_validate_insert;
ALTER TABLE work_unit_provider_bindings DROP COLUMN model;
ALTER TABLE work_unit_provider_bindings DROP COLUMN model_selection;
ALTER TABLE plan_revisions DROP COLUMN routing_decision_json;
ALTER TABLE contract_revisions DROP COLUMN execution_preference_json;
-- +goose StatementEnd

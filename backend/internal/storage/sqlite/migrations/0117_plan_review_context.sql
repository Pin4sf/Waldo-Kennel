-- +goose Up
ALTER TABLE plan_revisions ADD COLUMN assumptions_json TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(assumptions_json));
ALTER TABLE plan_revisions ADD COLUMN blockers_json TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(blockers_json));

-- +goose StatementBegin
DROP TRIGGER IF EXISTS plan_revisions_immutable_update;
CREATE TRIGGER plan_revisions_immutable_update
BEFORE UPDATE ON plan_revisions
WHEN OLD.id <> NEW.id
     OR OLD.outcome_id <> NEW.outcome_id
     OR OLD.number <> NEW.number
     OR OLD.contract_revision_number <> NEW.contract_revision_number
     OR OLD.summary <> NEW.summary
     OR OLD.assumptions_json <> NEW.assumptions_json
     OR OLD.blockers_json <> NEW.blockers_json
     OR OLD.run_brief_core_digest <> NEW.run_brief_core_digest
     OR OLD.run_brief_compiled_digest IS NOT NEW.run_brief_compiled_digest
     OR OLD.created_at <> NEW.created_at
BEGIN
    SELECT RAISE(ABORT, 'plan revisions are immutable');
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS plan_revisions_immutable_update;
ALTER TABLE plan_revisions DROP COLUMN blockers_json;
ALTER TABLE plan_revisions DROP COLUMN assumptions_json;
DROP TRIGGER IF EXISTS plan_revisions_immutable_update;
CREATE TRIGGER plan_revisions_immutable_update
BEFORE UPDATE ON plan_revisions
WHEN OLD.id <> NEW.id
     OR OLD.outcome_id <> NEW.outcome_id
     OR OLD.number <> NEW.number
     OR OLD.contract_revision_number <> NEW.contract_revision_number
     OR OLD.summary <> NEW.summary
     OR OLD.run_brief_core_digest <> NEW.run_brief_core_digest
     OR OLD.run_brief_compiled_digest IS NOT NEW.run_brief_compiled_digest
     OR OLD.created_at <> NEW.created_at
BEGIN
    SELECT RAISE(ABORT, 'plan revisions are immutable');
END;
-- +goose StatementEnd

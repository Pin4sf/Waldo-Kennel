-- WT3 + production-shaped direct Plan graph metadata. 0114 has not reached beta,
-- so this feature-branch version is intentionally stabilized before shipping.
-- 0112/0113 are shipped history and remain untouched.

-- +goose Up
-- +goose StatementBegin
ALTER TABLE contract_revisions
    ADD COLUMN execution_preference_json TEXT
        CHECK (execution_preference_json IS NULL OR json_valid(execution_preference_json));

ALTER TABLE plan_revisions
    ADD COLUMN routing_decisions_json TEXT
        CHECK (routing_decisions_json IS NULL OR json_valid(routing_decisions_json));

ALTER TABLE work_unit_provider_bindings
    ADD COLUMN model_selection TEXT
        CHECK (model_selection IS NULL OR model_selection IN ('explicit', 'provider_default'));

ALTER TABLE work_unit_provider_bindings
    ADD COLUMN model TEXT;

-- Existing 0112 provider-only rows legitimately have NULL model_selection. New
-- WT3 writes must carry exact explicit/provider-default semantics.
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

-- Canonical dependency truth is relational and independent from WorkUnit list
-- serialization. MVP scheduling may use concurrency=1 over this same graph.
CREATE TABLE work_unit_dependencies (
    work_unit_id            TEXT NOT NULL REFERENCES work_units (id),
    depends_on_work_unit_id TEXT NOT NULL REFERENCES work_units (id),
    PRIMARY KEY (work_unit_id, depends_on_work_unit_id),
    CHECK (work_unit_id <> depends_on_work_unit_id)
);

CREATE TRIGGER work_unit_dependencies_same_plan
BEFORE INSERT ON work_unit_dependencies
WHEN NOT EXISTS (
    SELECT 1
    FROM work_units child
    JOIN work_units parent
      ON parent.id = NEW.depends_on_work_unit_id
    WHERE child.id = NEW.work_unit_id
      AND child.plan_revision_id = parent.plan_revision_id
)
BEGIN
    SELECT RAISE(ABORT, 'work unit dependency must stay inside one plan revision');
END;

CREATE TRIGGER work_unit_dependencies_immutable_update
BEFORE UPDATE ON work_unit_dependencies
BEGIN SELECT RAISE(ABORT, 'work unit dependencies are immutable'); END;
CREATE TRIGGER work_unit_dependencies_immutable_delete
BEFORE DELETE ON work_unit_dependencies
BEGIN SELECT RAISE(ABORT, 'work unit dependencies are immutable'); END;

-- Criterion coverage is pinned to the exact ContractRevision identity rather
-- than relying on display text or criterion position.
CREATE TABLE work_unit_criterion_bindings (
    work_unit_id         TEXT NOT NULL REFERENCES work_units (id),
    contract_revision_id TEXT NOT NULL,
    criterion_id         TEXT NOT NULL,
    PRIMARY KEY (work_unit_id, contract_revision_id, criterion_id),
    FOREIGN KEY (contract_revision_id, criterion_id)
        REFERENCES contract_criteria (contract_revision_id, id)
);

CREATE TRIGGER work_unit_criterion_binding_matches_plan
BEFORE INSERT ON work_unit_criterion_bindings
WHEN NOT EXISTS (
    SELECT 1
    FROM work_units wu
    JOIN plan_revisions pr ON pr.id = wu.plan_revision_id
    JOIN contract_revisions cr
      ON cr.id = NEW.contract_revision_id
     AND cr.outcome_id = pr.outcome_id
     AND cr.number = pr.contract_revision_number
    WHERE wu.id = NEW.work_unit_id
)
BEGIN
    SELECT RAISE(ABORT, 'work unit criterion must belong to the plan contract revision');
END;

CREATE TRIGGER work_unit_criterion_bindings_immutable_update
BEFORE UPDATE ON work_unit_criterion_bindings
BEGIN SELECT RAISE(ABORT, 'work unit criterion bindings are immutable'); END;
CREATE TRIGGER work_unit_criterion_bindings_immutable_delete
BEFORE DELETE ON work_unit_criterion_bindings
BEGIN SELECT RAISE(ABORT, 'work unit criterion bindings are immutable'); END;

-- Required capabilities are the deterministic minimum derived for each unit.
-- Plan-level capability grants authorize the union; they do not imply every
-- WorkUnit needs every granted capability.
CREATE TABLE work_unit_required_capabilities (
    work_unit_id TEXT NOT NULL REFERENCES work_units (id),
    capability   TEXT NOT NULL CHECK (length(trim(capability)) > 0),
    PRIMARY KEY (work_unit_id, capability)
);
CREATE TRIGGER work_unit_required_capabilities_immutable_update
BEFORE UPDATE ON work_unit_required_capabilities
BEGIN SELECT RAISE(ABORT, 'work unit required capabilities are immutable'); END;
CREATE TRIGGER work_unit_required_capabilities_immutable_delete
BEFORE DELETE ON work_unit_required_capabilities
BEGIN SELECT RAISE(ABORT, 'work unit required capabilities are immutable'); END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS work_unit_required_capabilities_immutable_delete;
DROP TRIGGER IF EXISTS work_unit_required_capabilities_immutable_update;
DROP TABLE IF EXISTS work_unit_required_capabilities;
DROP TRIGGER IF EXISTS work_unit_criterion_bindings_immutable_delete;
DROP TRIGGER IF EXISTS work_unit_criterion_bindings_immutable_update;
DROP TRIGGER IF EXISTS work_unit_criterion_binding_matches_plan;
DROP TABLE IF EXISTS work_unit_criterion_bindings;
DROP TRIGGER IF EXISTS work_unit_dependencies_immutable_delete;
DROP TRIGGER IF EXISTS work_unit_dependencies_immutable_update;
DROP TRIGGER IF EXISTS work_unit_dependencies_same_plan;
DROP TABLE IF EXISTS work_unit_dependencies;
DROP TRIGGER IF EXISTS work_unit_execution_binding_validate_insert;
ALTER TABLE work_unit_provider_bindings DROP COLUMN model;
ALTER TABLE work_unit_provider_bindings DROP COLUMN model_selection;
ALTER TABLE plan_revisions DROP COLUMN routing_decisions_json;
ALTER TABLE contract_revisions DROP COLUMN execution_preference_json;
-- +goose StatementEnd

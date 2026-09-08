-- Execution routing and the direct Plan WorkUnit graph.
--
-- This is not a migration. Migration 0114 records the version; the schema is
-- installed by reconcileExecutionRoutingSchema, for the same reason the
-- composition schema is: a burned 0099/0100 ledger entry marks those versions
-- applied while leaving contract_revisions, plan_revisions and work_units
-- physically absent, and ALTER TABLE cannot be made conditional inside
-- migration SQL. Every statement here is idempotent, so a repaired profile
-- heals on the next start.

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
CREATE TABLE IF NOT EXISTS work_unit_dependencies (
    work_unit_id            TEXT NOT NULL REFERENCES work_units (id),
    depends_on_work_unit_id TEXT NOT NULL REFERENCES work_units (id),
    PRIMARY KEY (work_unit_id, depends_on_work_unit_id),
    CHECK (work_unit_id <> depends_on_work_unit_id)
);

DROP TRIGGER IF EXISTS work_unit_dependencies_same_plan;
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

DROP TRIGGER IF EXISTS work_unit_dependencies_immutable_update;
CREATE TRIGGER work_unit_dependencies_immutable_update
BEFORE UPDATE ON work_unit_dependencies
BEGIN SELECT RAISE(ABORT, 'work unit dependencies are immutable'); END;

DROP TRIGGER IF EXISTS work_unit_dependencies_immutable_delete;
CREATE TRIGGER work_unit_dependencies_immutable_delete
BEFORE DELETE ON work_unit_dependencies
BEGIN SELECT RAISE(ABORT, 'work unit dependencies are immutable'); END;

-- Criterion coverage is pinned to the exact ContractRevision identity rather
-- than relying on display text or criterion position.
CREATE TABLE IF NOT EXISTS work_unit_criterion_bindings (
    work_unit_id         TEXT NOT NULL REFERENCES work_units (id),
    contract_revision_id TEXT NOT NULL,
    criterion_id         TEXT NOT NULL,
    PRIMARY KEY (work_unit_id, contract_revision_id, criterion_id),
    FOREIGN KEY (contract_revision_id, criterion_id)
        REFERENCES contract_criteria (contract_revision_id, id)
);

DROP TRIGGER IF EXISTS work_unit_criterion_binding_matches_plan;
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

DROP TRIGGER IF EXISTS work_unit_criterion_bindings_immutable_update;
CREATE TRIGGER work_unit_criterion_bindings_immutable_update
BEFORE UPDATE ON work_unit_criterion_bindings
BEGIN SELECT RAISE(ABORT, 'work unit criterion bindings are immutable'); END;

DROP TRIGGER IF EXISTS work_unit_criterion_bindings_immutable_delete;
CREATE TRIGGER work_unit_criterion_bindings_immutable_delete
BEFORE DELETE ON work_unit_criterion_bindings
BEGIN SELECT RAISE(ABORT, 'work unit criterion bindings are immutable'); END;

-- Required capabilities are the deterministic minimum derived for each unit.
-- Plan-level capability grants authorize the union; they do not imply every
-- WorkUnit needs every granted capability.
CREATE TABLE IF NOT EXISTS work_unit_required_capabilities (
    work_unit_id TEXT NOT NULL REFERENCES work_units (id),
    capability   TEXT NOT NULL CHECK (length(trim(capability)) > 0),
    PRIMARY KEY (work_unit_id, capability)
);

DROP TRIGGER IF EXISTS work_unit_required_capabilities_immutable_update;
CREATE TRIGGER work_unit_required_capabilities_immutable_update
BEFORE UPDATE ON work_unit_required_capabilities
BEGIN SELECT RAISE(ABORT, 'work unit required capabilities are immutable'); END;

DROP TRIGGER IF EXISTS work_unit_required_capabilities_immutable_delete;
CREATE TRIGGER work_unit_required_capabilities_immutable_delete
BEFORE DELETE ON work_unit_required_capabilities
BEGIN SELECT RAISE(ABORT, 'work unit required capabilities are immutable'); END;

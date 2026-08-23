-- Canonical Decide & Authorize contracts (#26): PlanRevision -> one direct
-- WorkUnit -> scoped CapabilityGrants, frozen by the RunBrief core digest.
-- Plans bind their contract revision; a later revision supersedes them and
-- forces a fresh proposal. Approval is the owner's authority gate.
--
-- Same checked-table rebuild discipline as 0099, with one deliberate
-- difference: EVERY change_log writer stays detached after the rebuild and
-- is restored exclusively by cdc_restore.go. A skipped/burned 0099 profile
-- (upstream ledger collisions) has no outcomes table here, so recreating
-- even 0099's writers inside this file would abort the whole migration.
-- Same checked-table rebuild discipline as 0099: every change_log writer —
-- the 23 inherited ones plus 0099's Outcome writers — is detached before the
-- rebuild, and the Outcome writers are recreated verbatim afterwards (their
-- subject tables are FK-guaranteed to exist by the time this migration runs).
-- The 23 inherited writers are restored by cdc_restore.go after goose
-- completes, exactly as before.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE plan_revisions (
    id                        TEXT PRIMARY KEY,
    outcome_id                TEXT NOT NULL REFERENCES outcomes (id),
    number                    INTEGER NOT NULL CHECK (number >= 1),
    contract_revision_number  INTEGER NOT NULL CHECK (contract_revision_number >= 1),
    status                    TEXT NOT NULL CHECK (status IN ('proposed', 'approved')),
    summary                   TEXT NOT NULL DEFAULT '',
    run_brief_core_digest     TEXT NOT NULL CHECK (length(run_brief_core_digest) = 64),
    run_brief_compiled_digest TEXT NOT NULL DEFAULT '',
    created_at                TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX idx_plan_revisions_outcome_number
    ON plan_revisions (outcome_id, number);

CREATE TABLE work_units (
    id                       TEXT PRIMARY KEY,
    plan_revision_id         TEXT NOT NULL REFERENCES plan_revisions (id),
    kind                     TEXT NOT NULL CHECK (kind IN ('direct')),
    title                    TEXT NOT NULL,
    contract_revision_number INTEGER NOT NULL CHECK (contract_revision_number >= 1),
    output_summary           TEXT NOT NULL,
    evidence_checks          TEXT NOT NULL CHECK (json_valid(evidence_checks)),
    verification_requirement TEXT NOT NULL,
    stop_conditions          TEXT NOT NULL CHECK (json_valid(stop_conditions))
);

CREATE TABLE capability_grants (
    id               TEXT PRIMARY KEY,
    plan_revision_id TEXT NOT NULL REFERENCES plan_revisions (id),
    name             TEXT NOT NULL,
    scope            TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_capability_grants_plan_name
    ON capability_grants (plan_revision_id, name);

-- Plans are append-only history. The only permitted mutation is the approval
-- status transition proposed -> approved; identity, binding, content, and the
-- frozen digests may never change.
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

DROP TRIGGER IF EXISTS plan_revisions_immutable_delete;
CREATE TRIGGER plan_revisions_immutable_delete
BEFORE DELETE ON plan_revisions
BEGIN
    SELECT RAISE(ABORT, 'plan revisions are immutable');
END;

DROP TRIGGER IF EXISTS work_units_immutable_update;
CREATE TRIGGER work_units_immutable_update
BEFORE UPDATE ON work_units
BEGIN
    SELECT RAISE(ABORT, 'work units are immutable');
END;

DROP TRIGGER IF EXISTS work_units_immutable_delete;
CREATE TRIGGER work_units_immutable_delete
BEFORE DELETE ON work_units
BEGIN
    SELECT RAISE(ABORT, 'work units are immutable');
END;

DROP TRIGGER IF EXISTS capability_grants_immutable_update;
CREATE TRIGGER capability_grants_immutable_update
BEFORE UPDATE ON capability_grants
BEGIN
    SELECT RAISE(ABORT, 'capability grants are immutable');
END;

DROP TRIGGER IF EXISTS capability_grants_immutable_delete;
CREATE TRIGGER capability_grants_immutable_delete
BEFORE DELETE ON capability_grants
BEGIN
    SELECT RAISE(ABORT, 'capability grants are immutable');
END;

-- Detach inherited writers before touching the table.
DROP TRIGGER IF EXISTS agent_switches_cdc_insert;
DROP TRIGGER IF EXISTS agent_switches_cdc_update;
DROP TRIGGER IF EXISTS conversation_activities_cdc_insert;
DROP TRIGGER IF EXISTS conversation_activities_cdc_update;
DROP TRIGGER IF EXISTS conversation_messages_cdc_insert;
DROP TRIGGER IF EXISTS conversation_messages_cdc_update;
DROP TRIGGER IF EXISTS conversation_turns_cdc_update;
DROP TRIGGER IF EXISTS pr_cdc_insert;
DROP TRIGGER IF EXISTS pr_cdc_update;
DROP TRIGGER IF EXISTS pr_checks_cdc_insert;
DROP TRIGGER IF EXISTS pr_checks_cdc_update;
DROP TRIGGER IF EXISTS pr_review_threads_cdc_insert;
DROP TRIGGER IF EXISTS pr_review_threads_cdc_update;
DROP TRIGGER IF EXISTS pr_session_cdc_update;
DROP TRIGGER IF EXISTS session_cleanup_facts_cdc_insert;
DROP TRIGGER IF EXISTS session_cleanup_facts_cdc_update;
DROP TRIGGER IF EXISTS session_interface_transitions_cdc_insert;
DROP TRIGGER IF EXISTS session_interface_transitions_cdc_update;
DROP TRIGGER IF EXISTS sessions_cdc_insert;
DROP TRIGGER IF EXISTS sessions_cdc_update;
DROP TRIGGER IF EXISTS usage_bindings_cdc_insert;
DROP TRIGGER IF EXISTS usage_bindings_cdc_update;
DROP TRIGGER IF EXISTS usage_sources_cdc_update;

-- Detach the Outcome writers this migration's rebuild would strand; they are
-- recreated below because their subject tables necessarily exist here.
DROP TRIGGER IF EXISTS responsibility_outcomes_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_outcomes_cdc_update;
DROP TRIGGER IF EXISTS responsibility_contract_revisions_cdc_insert;

CREATE TABLE change_log_new (
    seq        INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL REFERENCES projects (id),
    session_id TEXT REFERENCES sessions (id),
    event_type TEXT NOT NULL
        CHECK (event_type IN (
            'session_created',
            'session_updated',
            'pr_created',
            'pr_updated',
            'pr_check_recorded',
            'pr_session_changed',
            'pr_review_thread_added',
            'pr_review_thread_resolved',
            'outcome_created',
            'outcome_updated',
            'outcome_contract_revised',
            'outcome_plan_proposed',
            'outcome_plan_approved'
        )),
    payload    TEXT NOT NULL CHECK (json_valid(payload)),
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO change_log_new (seq, project_id, session_id, event_type, payload, created_at)
SELECT seq, project_id, session_id, event_type, payload, created_at
FROM change_log;

DROP INDEX IF EXISTS idx_change_log_project;
DROP TABLE change_log;
ALTER TABLE change_log_new RENAME TO change_log;
CREATE INDEX idx_change_log_project ON change_log (project_id, seq);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS outcome_plans_cdc_update;
DROP TRIGGER IF EXISTS outcome_plans_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_contract_revisions_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_outcomes_cdc_update;
DROP TRIGGER IF EXISTS responsibility_outcomes_cdc_insert;
DROP TRIGGER IF EXISTS plan_revisions_immutable_delete;
DROP TRIGGER IF EXISTS plan_revisions_immutable_update;
DROP TRIGGER IF EXISTS work_units_immutable_delete;
DROP TRIGGER IF EXISTS work_units_immutable_update;
DROP TRIGGER IF EXISTS capability_grants_immutable_delete;
DROP TRIGGER IF EXISTS capability_grants_immutable_update;

-- Detach inherited writers before restoring the prior log shape.
DROP TRIGGER IF EXISTS agent_switches_cdc_insert;
DROP TRIGGER IF EXISTS agent_switches_cdc_update;
DROP TRIGGER IF EXISTS conversation_activities_cdc_insert;
DROP TRIGGER IF EXISTS conversation_activities_cdc_update;
DROP TRIGGER IF EXISTS conversation_messages_cdc_insert;
DROP TRIGGER IF EXISTS conversation_messages_cdc_update;
DROP TRIGGER IF EXISTS conversation_turns_cdc_update;
DROP TRIGGER IF EXISTS pr_cdc_insert;
DROP TRIGGER IF EXISTS pr_cdc_update;
DROP TRIGGER IF EXISTS pr_checks_cdc_insert;
DROP TRIGGER IF EXISTS pr_checks_cdc_update;
DROP TRIGGER IF EXISTS pr_review_threads_cdc_insert;
DROP TRIGGER IF EXISTS pr_review_threads_cdc_update;
DROP TRIGGER IF EXISTS pr_session_cdc_update;
DROP TRIGGER IF EXISTS session_cleanup_facts_cdc_insert;
DROP TRIGGER IF EXISTS session_cleanup_facts_cdc_update;
DROP TRIGGER IF EXISTS session_interface_transitions_cdc_insert;
DROP TRIGGER IF EXISTS session_interface_transitions_cdc_update;
DROP TRIGGER IF EXISTS sessions_cdc_insert;
DROP TRIGGER IF EXISTS sessions_cdc_update;
DROP TRIGGER IF EXISTS usage_bindings_cdc_insert;
DROP TRIGGER IF EXISTS usage_bindings_cdc_update;
DROP TRIGGER IF EXISTS usage_sources_cdc_update;

CREATE TABLE change_log_down (
    seq        INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL REFERENCES projects (id),
    session_id TEXT REFERENCES sessions (id),
    event_type TEXT NOT NULL
        CHECK (event_type IN (
            'session_created',
            'session_updated',
            'pr_created',
            'pr_updated',
            'pr_check_recorded',
            'pr_session_changed',
            'pr_review_thread_added',
            'pr_review_thread_resolved'
        )),
    payload    TEXT NOT NULL CHECK (json_valid(payload)),
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO change_log_down (seq, project_id, session_id, event_type, payload, created_at)
SELECT seq, project_id, session_id, event_type, payload, created_at
FROM change_log
WHERE event_type NOT IN ('outcome_plan_proposed', 'outcome_plan_approved');

DROP INDEX IF EXISTS idx_change_log_project;
DROP TABLE change_log;
ALTER TABLE change_log_down RENAME TO change_log;
CREATE INDEX idx_change_log_project ON change_log (project_id, seq);

DROP TABLE IF EXISTS capability_grants;
DROP TABLE IF EXISTS work_units;
DROP TABLE IF EXISTS plan_revisions;
-- +goose StatementEnd

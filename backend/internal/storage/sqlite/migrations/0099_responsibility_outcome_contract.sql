-- Canonical Work responsibility contracts (#21): ResponsibilitySpace ->
-- Outcome -> immutable ContractRevision. Outcomes carry no lifecycle status;
-- acceptance is a separate owner decision (#35).
--
-- Extends change_log with Outcome event types. Every inherited CDC trigger
-- that writes change_log is dropped before the checked-table rebuild: dropping
-- the table alone would strand those triggers against a missing relation.
-- Restoration is intentionally NOT done here: degraded field profiles may not
-- yet have every writer's subject table, so migrate() restores the canonical
-- set from cdc_restore.go after goose completes (sqlite_master-guarded).

-- +goose Up
-- +goose StatementBegin
CREATE TABLE responsibility_spaces (
    id         TEXT PRIMARY KEY,
    kind       TEXT NOT NULL CHECK (kind IN ('WorkProject')),
    project_id TEXT NOT NULL REFERENCES projects (id),
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_responsibility_spaces_project
    ON responsibility_spaces (project_id, kind);

CREATE UNIQUE INDEX idx_responsibility_spaces_work_project
    ON responsibility_spaces (project_id)
    WHERE kind = 'WorkProject';

CREATE TABLE outcomes (
    id                      TEXT PRIMARY KEY,
    space_id                TEXT NOT NULL REFERENCES responsibility_spaces (id),
    title                   TEXT NOT NULL,
    current_revision_number INTEGER NOT NULL DEFAULT 0 CHECK (current_revision_number >= 0),
    created_at              TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at              TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_outcomes_space ON outcomes (space_id, created_at);

CREATE TABLE contract_revisions (
    id               TEXT PRIMARY KEY,
    outcome_id       TEXT NOT NULL REFERENCES outcomes (id),
    number           INTEGER NOT NULL CHECK (number >= 1),
    goal             TEXT NOT NULL,
    success_criteria TEXT NOT NULL CHECK (json_valid(success_criteria)),
    review           TEXT NOT NULL,
    constraints      TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(constraints)),
    non_goals        TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(non_goals)),
    clarification    TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX idx_contract_revisions_outcome_number
    ON contract_revisions (outcome_id, number);
-- +goose StatementEnd

-- +goose StatementBegin
-- Detach every inherited change_log writer before touching the table.
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

-- Contract revisions are append-only history; storage offers no mutation path.
DROP TRIGGER IF EXISTS contract_revisions_immutable_update;
CREATE TRIGGER contract_revisions_immutable_update
BEFORE UPDATE ON contract_revisions
BEGIN
    SELECT RAISE(ABORT, 'contract revisions are immutable');
END;

DROP TRIGGER IF EXISTS contract_revisions_immutable_delete;
CREATE TRIGGER contract_revisions_immutable_delete
BEFORE DELETE ON contract_revisions
BEGIN
    SELECT RAISE(ABORT, 'contract revisions are immutable');
END;

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
            'outcome_contract_revised'
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


-- Outcome CDC: canonical facts through the shared trigger pipeline.
DROP TRIGGER IF EXISTS responsibility_outcomes_cdc_insert;
CREATE TRIGGER responsibility_outcomes_cdc_insert
AFTER INSERT ON outcomes
BEGIN
    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)
    VALUES (
        (SELECT project_id FROM responsibility_spaces WHERE id = NEW.space_id),
        NULL,
        'outcome_created',
        json_object(
            'id', NEW.id,
            'spaceId', NEW.space_id,
            'title', NEW.title,
            'currentRevisionNumber', NEW.current_revision_number
        ),
        NEW.created_at);
END;

DROP TRIGGER IF EXISTS responsibility_outcomes_cdc_update;
CREATE TRIGGER responsibility_outcomes_cdc_update
AFTER UPDATE ON outcomes
WHEN OLD.title <> NEW.title
     OR OLD.current_revision_number <> NEW.current_revision_number
BEGIN
    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)
    VALUES (
        (SELECT project_id FROM responsibility_spaces WHERE id = NEW.space_id),
        NULL,
        'outcome_updated',
        json_object(
            'id', NEW.id,
            'spaceId', NEW.space_id,
            'title', NEW.title,
            'previousRevisionNumber', OLD.current_revision_number,
            'currentRevisionNumber', NEW.current_revision_number
        ),
        NEW.updated_at);
END;

DROP TRIGGER IF EXISTS responsibility_contract_revisions_cdc_insert;
CREATE TRIGGER responsibility_contract_revisions_cdc_insert
AFTER INSERT ON contract_revisions
BEGIN
    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)
    VALUES (
        (SELECT s.project_id
           FROM outcomes o
           JOIN responsibility_spaces s ON s.id = o.space_id
          WHERE o.id = NEW.outcome_id),
        NULL,
        'outcome_contract_revised',
        json_object(
            'revisionId', NEW.id,
            'outcomeId', NEW.outcome_id,
            'number', NEW.number,
            'goal', NEW.goal
        ),
        NEW.created_at);
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS responsibility_contract_revisions_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_outcomes_cdc_update;
DROP TRIGGER IF EXISTS responsibility_outcomes_cdc_insert;
DROP TRIGGER IF EXISTS contract_revisions_immutable_delete;
DROP TRIGGER IF EXISTS contract_revisions_immutable_update;

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
WHERE event_type NOT IN ('outcome_created', 'outcome_updated', 'outcome_contract_revised');

DROP INDEX IF EXISTS idx_change_log_project;
DROP TABLE change_log;
ALTER TABLE change_log_down RENAME TO change_log;
CREATE INDEX idx_change_log_project ON change_log (project_id, seq);


DROP TABLE IF EXISTS contract_revisions;
DROP TABLE IF EXISTS outcomes;
DROP TABLE IF EXISTS responsibility_spaces;
-- +goose StatementEnd

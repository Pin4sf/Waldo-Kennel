-- Canonical Act & Observe execution contracts (#31): Attempt ->
-- AttemptSessionRef -> ordered AttemptObservations, custody Fences, and
-- RecoveryReceipts. Attempts are append-only lineage: a replacement is always
-- a NEW attempt row, never an update of a prior one. Exactly one OPEN fence
-- may exist per worktree subject (partial unique index on released_at IS
-- NULL); replacement inherits custody only through reconcile releasing the
-- old fence before the new attempt issues its own. Stored statuses are
-- trigger-guarded to domain.LegalAttemptTransitions; `unconfirmed` is
-- deliberately absent from the CHECK because it is derived at read time from
-- bound-session heartbeat facts, never stored.
--
-- attempt_sessions.session_id is TEXT with NO foreign key into sessions(id):
-- a spawn rollback deletes seed session rows, and ref history must outlive
-- session-row GC so restart replays exact lineage.
--
-- Same checked-table rebuild discipline as 0100: every change_log writer is
-- detached before the rebuild and restored exclusively by cdc_restore.go
-- after goose completes. A skipped/burned earlier ledger entry can leave this
-- migration running without the outcomes tables its writers join through, so
-- recreating any writer inside this file would abort the whole migration.
-- The five new event types enter the vocabulary through the rebuilt CHECK.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE attempts (
    id                       TEXT PRIMARY KEY,
    outcome_id               TEXT NOT NULL REFERENCES outcomes (id),
    plan_revision_id         TEXT NOT NULL REFERENCES plan_revisions (id),
    work_unit_id             TEXT NOT NULL REFERENCES work_units (id),
    number                   INTEGER NOT NULL CHECK (number >= 1),
    status                   TEXT NOT NULL CHECK (status IN (
                                 'queued', 'running', 'paused', 'succeeded',
                                 'failed', 'cancelled', 'lost', 'reconciled')),
    contract_revision_number INTEGER NOT NULL CHECK (contract_revision_number >= 1),
    request_key              TEXT,
    created_at               TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at              TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX idx_attempts_outcome_number
    ON attempts (outcome_id, number);

CREATE UNIQUE INDEX idx_attempts_request_key
    ON attempts (request_key) WHERE request_key IS NOT NULL;

CREATE TABLE attempt_sessions (
    id                        TEXT PRIMARY KEY,
    attempt_id                TEXT NOT NULL REFERENCES attempts (id),
    seq                       INTEGER NOT NULL CHECK (seq >= 1),
    -- Deliberately NO FK into sessions(id): refs must outlive session-row GC.
    session_id                TEXT NOT NULL,
    harness                   TEXT NOT NULL DEFAULT '',
    mode                      TEXT NOT NULL DEFAULT '',
    run_brief_core_digest     TEXT NOT NULL CHECK (length(run_brief_core_digest) = 64),
    run_brief_compiled_digest TEXT NOT NULL DEFAULT '',
    admission_snapshot        TEXT NOT NULL CHECK (json_valid(admission_snapshot)),
    bound_at                  TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX idx_attempt_sessions_attempt_seq
    ON attempt_sessions (attempt_id, seq);

CREATE TABLE attempt_observations (
    id         TEXT PRIMARY KEY,
    attempt_id TEXT NOT NULL REFERENCES attempts (id),
    seq        INTEGER NOT NULL CHECK (seq >= 1),
    kind       TEXT NOT NULL CHECK (length(trim(kind)) > 0),
    payload    TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(payload)),
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX idx_attempt_observations_attempt_seq
    ON attempt_observations (attempt_id, seq);

CREATE TABLE attempt_fences (
    id             TEXT PRIMARY KEY,
    subject        TEXT NOT NULL,
    attempt_id     TEXT NOT NULL REFERENCES attempts (id),
    issued_at      TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    released_at    TIMESTAMP,
    release_reason TEXT NOT NULL DEFAULT ''
);

-- D4: exactly one open fence per worktree subject. Released fences stay as
-- history; only reconcile releases them.
CREATE UNIQUE INDEX idx_attempt_fences_open_subject
    ON attempt_fences (subject) WHERE released_at IS NULL;

CREATE TABLE attempt_recovery_receipts (
    id                     TEXT PRIMARY KEY,
    attempt_id             TEXT NOT NULL REFERENCES attempts (id),
    resolution             TEXT NOT NULL CHECK (resolution IN ('resumed', 'replacement_attempt', 'needs_attention')),
    -- Not FK'd on purpose: it may name the replacement attempt before that row exists.
    replacement_attempt_id TEXT NOT NULL DEFAULT '',
    detail                 TEXT NOT NULL DEFAULT '' CHECK (detail = '' OR json_valid(detail)),
    created_at             TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

-- Attempts are append-only lineage. Identity, binding, and content never
-- change; the ONLY permitted mutation is the trigger-guarded status
-- transition below.
DROP TRIGGER IF EXISTS attempts_immutable_update;
CREATE TRIGGER attempts_immutable_update
BEFORE UPDATE ON attempts
WHEN OLD.id <> NEW.id
     OR OLD.outcome_id <> NEW.outcome_id
     OR OLD.plan_revision_id <> NEW.plan_revision_id
     OR OLD.work_unit_id <> NEW.work_unit_id
     OR OLD.number <> NEW.number
     OR OLD.contract_revision_number <> NEW.contract_revision_number
     OR OLD.request_key IS NOT NEW.request_key
     OR OLD.created_at <> NEW.created_at
BEGIN
    SELECT RAISE(ABORT, 'attempts are immutable');
END;

DROP TRIGGER IF EXISTS attempts_status_transition;
CREATE TRIGGER attempts_status_transition
BEFORE UPDATE ON attempts
WHEN OLD.status <> NEW.status
     AND NOT (
         (OLD.status = 'queued' AND NEW.status IN ('running', 'failed', 'cancelled', 'lost'))
      OR (OLD.status = 'running' AND NEW.status IN ('paused', 'failed', 'cancelled', 'lost', 'reconciled'))
      OR (OLD.status = 'paused' AND NEW.status IN ('running', 'cancelled', 'lost'))
     )
BEGIN
    SELECT RAISE(ABORT, 'illegal attempt status transition');
END;

DROP TRIGGER IF EXISTS attempts_immutable_delete;
CREATE TRIGGER attempts_immutable_delete
BEFORE DELETE ON attempts
BEGIN
    SELECT RAISE(ABORT, 'attempts are append-only');
END;

DROP TRIGGER IF EXISTS attempt_sessions_immutable_update;
CREATE TRIGGER attempt_sessions_immutable_update
BEFORE UPDATE ON attempt_sessions
BEGIN
    SELECT RAISE(ABORT, 'attempt session refs are immutable');
END;

DROP TRIGGER IF EXISTS attempt_sessions_immutable_delete;
CREATE TRIGGER attempt_sessions_immutable_delete
BEFORE DELETE ON attempt_sessions
BEGIN
    SELECT RAISE(ABORT, 'attempt session refs are immutable');
END;

DROP TRIGGER IF EXISTS attempt_observations_immutable_update;
CREATE TRIGGER attempt_observations_immutable_update
BEFORE UPDATE ON attempt_observations
BEGIN
    SELECT RAISE(ABORT, 'attempt observations are append-only');
END;

DROP TRIGGER IF EXISTS attempt_observations_immutable_delete;
CREATE TRIGGER attempt_observations_immutable_delete
BEFORE DELETE ON attempt_observations
BEGIN
    SELECT RAISE(ABORT, 'attempt observations are append-only');
END;

DROP TRIGGER IF EXISTS attempt_fences_immutable_identity;
CREATE TRIGGER attempt_fences_immutable_identity
BEFORE UPDATE ON attempt_fences
WHEN OLD.id <> NEW.id
     OR OLD.subject <> NEW.subject
     OR OLD.attempt_id <> NEW.attempt_id
     OR OLD.issued_at <> NEW.issued_at
BEGIN
    SELECT RAISE(ABORT, 'attempt fence identity is immutable');
END;

-- A fence may be released exactly once, and only with a reason. Once
-- released, the row is frozen: even rewriting the reason is refused.
DROP TRIGGER IF EXISTS attempt_fences_release_once;
CREATE TRIGGER attempt_fences_release_once
BEFORE UPDATE ON attempt_fences
WHEN OLD.released_at IS NOT NULL
     AND NOT (OLD.released_at IS NEW.released_at
              AND OLD.release_reason IS NEW.release_reason)
BEGIN
    SELECT RAISE(ABORT, 'attempt fence release is final');
END;

DROP TRIGGER IF EXISTS attempt_fences_release_reason;
CREATE TRIGGER attempt_fences_release_reason
BEFORE UPDATE ON attempt_fences
WHEN OLD.released_at IS NULL
     AND NEW.released_at IS NOT NULL
     AND trim(NEW.release_reason) = ''
BEGIN
    SELECT RAISE(ABORT, 'a released attempt fence must record why');
END;

DROP TRIGGER IF EXISTS attempt_fences_immutable_delete;
CREATE TRIGGER attempt_fences_immutable_delete
BEFORE DELETE ON attempt_fences
BEGIN
    SELECT RAISE(ABORT, 'attempt fences are append-only');
END;

DROP TRIGGER IF EXISTS attempt_recovery_receipts_immutable_update;
CREATE TRIGGER attempt_recovery_receipts_immutable_update
BEFORE UPDATE ON attempt_recovery_receipts
BEGIN
    SELECT RAISE(ABORT, 'recovery receipts are append-only');
END;

DROP TRIGGER IF EXISTS attempt_recovery_receipts_immutable_delete;
CREATE TRIGGER attempt_recovery_receipts_immutable_delete
BEFORE DELETE ON attempt_recovery_receipts
BEGIN
    SELECT RAISE(ABORT, 'recovery receipts are append-only');
END;

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
DROP TRIGGER IF EXISTS responsibility_outcomes_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_outcomes_cdc_update;
DROP TRIGGER IF EXISTS responsibility_contract_revisions_cdc_insert;
DROP TRIGGER IF EXISTS outcome_plans_cdc_insert;
DROP TRIGGER IF EXISTS outcome_plans_cdc_update;

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
            'outcome_plan_approved',
            'outcome_attempt_started',
            'outcome_attempt_updated',
            'outcome_attempt_session_bound',
            'outcome_attempt_observed',
            'outcome_attempt_recovered'
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
DROP TRIGGER IF EXISTS responsibility_outcomes_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_outcomes_cdc_update;
DROP TRIGGER IF EXISTS responsibility_contract_revisions_cdc_insert;
DROP TRIGGER IF EXISTS outcome_plans_cdc_insert;
DROP TRIGGER IF EXISTS outcome_plans_cdc_update;

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

INSERT INTO change_log_down (seq, project_id, session_id, event_type, payload, created_at)
SELECT seq, project_id, session_id, event_type, payload, created_at
FROM change_log
WHERE event_type NOT IN (
    'outcome_attempt_started',
    'outcome_attempt_updated',
    'outcome_attempt_session_bound',
    'outcome_attempt_observed',
    'outcome_attempt_recovered'
);

DROP INDEX IF EXISTS idx_change_log_project;
DROP TABLE change_log;
ALTER TABLE change_log_down RENAME TO change_log;
CREATE INDEX idx_change_log_project ON change_log (project_id, seq);

DROP TABLE IF EXISTS attempt_recovery_receipts;
DROP TABLE IF EXISTS attempt_fences;
DROP TABLE IF EXISTS attempt_observations;
DROP TABLE IF EXISTS attempt_sessions;
DROP TABLE IF EXISTS attempts;
-- +goose StatementEnd

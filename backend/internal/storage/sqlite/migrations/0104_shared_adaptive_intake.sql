-- Shared Home/Work adaptive intake and explicit Home-to-Work responsibility
-- lineage (#32). Proposals and conversation references remain non-canonical;
-- only atomic confirmation creates an Outcome and ContractRevision.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE intake_sessions (
    id                        TEXT PRIMARY KEY,
    source_surface            TEXT NOT NULL CHECK (source_surface IN ('home', 'work')),
    purpose                   TEXT NOT NULL CHECK (purpose IN ('note', 'open_loop', 'outcome', 'responsibility_link', 'dismiss')),
    project_id                TEXT REFERENCES projects (id),
    source_open_loop_id       TEXT,
    statement                 TEXT NOT NULL CHECK (length(trim(statement)) > 0),
    status                    TEXT NOT NULL CHECK (status IN ('captured', 'analyzing', 'needs_user', 'ready', 'confirmed', 'analysis_failed', 'cancelled')),
    current_proposal_revision INTEGER NOT NULL DEFAULT 0 CHECK (current_proposal_revision >= 0),
    clarification_count       INTEGER NOT NULL DEFAULT 0 CHECK (clarification_count BETWEEN 0 AND 1),
    confirmed_outcome_id      TEXT REFERENCES outcomes (id),
    failure_code              TEXT NOT NULL DEFAULT '',
    cancellation_reason       TEXT NOT NULL DEFAULT '',
    request_key               TEXT NOT NULL UNIQUE,
    request_fingerprint       TEXT NOT NULL,
    created_at                TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at                TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    CHECK (purpose <> 'outcome' OR project_id IS NOT NULL),
    CHECK ((status = 'confirmed') = (confirmed_outcome_id IS NOT NULL))
);

CREATE INDEX idx_intake_sessions_project ON intake_sessions (project_id, created_at, id);

CREATE TABLE intake_conversation_refs (
    intake_id  TEXT NOT NULL REFERENCES intake_sessions (id),
    episode_id TEXT NOT NULL CHECK (length(trim(episode_id)) > 0),
    turn_id    TEXT NOT NULL CHECK (length(trim(turn_id)) > 0),
    position   INTEGER NOT NULL CHECK (position >= 1),
    PRIMARY KEY (intake_id, position),
    UNIQUE (intake_id, episode_id, turn_id)
);

CREATE TABLE intake_clarifications (
    id                   TEXT PRIMARY KEY,
    intake_id            TEXT NOT NULL UNIQUE REFERENCES intake_sessions (id),
    question             TEXT NOT NULL CHECK (length(trim(question)) > 0),
    reason               TEXT NOT NULL CHECK (length(trim(reason)) > 0),
    recommendation       TEXT NOT NULL DEFAULT '',
    alternatives         TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(alternatives)),
    deferral_consequence TEXT NOT NULL DEFAULT '',
    created_at           TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE intake_clarification_answers (
    clarification_id TEXT PRIMARY KEY REFERENCES intake_clarifications (id),
    answer           TEXT NOT NULL CHECK (length(trim(answer)) > 0),
    answered_at      TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE intake_proposal_revisions (
    id                  TEXT PRIMARY KEY,
    intake_id           TEXT NOT NULL REFERENCES intake_sessions (id),
    revision            INTEGER NOT NULL CHECK (revision >= 1),
    title               TEXT NOT NULL CHECK (length(trim(title)) > 0),
    desired_state       TEXT NOT NULL CHECK (length(trim(desired_state)) > 0),
    criteria            TEXT NOT NULL CHECK (json_valid(criteria)),
    review_method       TEXT NOT NULL CHECK (length(trim(review_method)) > 0),
    constraints         TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(constraints)),
    non_goals           TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(non_goals)),
    authority_ceiling   TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(authority_ceiling)),
    stop_conditions     TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(stop_conditions)),
    clarification_notes TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(clarification_notes)),
    temporal_condition  TEXT,
    facets              TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(facets)),
    created_at          TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    UNIQUE (intake_id, revision)
);

CREATE TABLE intake_confirmations (
    intake_id           TEXT PRIMARY KEY REFERENCES intake_sessions (id),
    proposal_revision   INTEGER NOT NULL CHECK (proposal_revision >= 1),
    outcome_id          TEXT NOT NULL UNIQUE REFERENCES outcomes (id),
    contract_revision_id TEXT NOT NULL UNIQUE REFERENCES contract_revisions (id),
    request_key         TEXT NOT NULL UNIQUE,
    request_fingerprint TEXT NOT NULL,
    confirmed_at        TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE contract_revision_intake_core (
    contract_revision_id TEXT PRIMARY KEY REFERENCES contract_revisions (id),
    evidence_expectations TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(evidence_expectations)),
    authority_ceiling     TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(authority_ceiling)),
    stop_conditions       TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(stop_conditions)),
    temporal_condition    TEXT,
    facets                TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(facets)),
    created_at            TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE responsibility_links (
    id                     TEXT PRIMARY KEY,
    project_id             TEXT NOT NULL REFERENCES projects (id),
    source_open_loop_id    TEXT NOT NULL CHECK (length(trim(source_open_loop_id)) > 0),
    destination_outcome_id TEXT NOT NULL REFERENCES outcomes (id),
    creator                TEXT NOT NULL CHECK (creator = 'owner'),
    reason                 TEXT NOT NULL CHECK (length(trim(reason)) > 0),
    request_key            TEXT NOT NULL UNIQUE,
    request_fingerprint    TEXT NOT NULL,
    created_at             TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    ended_at               TIMESTAMP,
    ended_by               TEXT NOT NULL DEFAULT '' CHECK (ended_by IN ('', 'owner')),
    ended_reason           TEXT NOT NULL DEFAULT '',
    CHECK ((ended_at IS NULL AND ended_by = '' AND ended_reason = '') OR
           (ended_at IS NOT NULL AND ended_by = 'owner' AND length(trim(ended_reason)) > 0))
);

CREATE UNIQUE INDEX idx_responsibility_links_active_pair
    ON responsibility_links (source_open_loop_id, destination_outcome_id)
    WHERE ended_at IS NULL;
CREATE INDEX idx_responsibility_links_project ON responsibility_links (project_id, created_at, id);

CREATE TRIGGER intake_sessions_intent_immutable BEFORE UPDATE ON intake_sessions
WHEN OLD.source_surface <> NEW.source_surface OR OLD.purpose <> NEW.purpose
  OR OLD.project_id IS NOT NEW.project_id OR OLD.source_open_loop_id IS NOT NEW.source_open_loop_id
  OR OLD.statement <> NEW.statement OR OLD.request_key <> NEW.request_key
  OR OLD.request_fingerprint <> NEW.request_fingerprint OR OLD.created_at <> NEW.created_at
BEGIN SELECT RAISE(ABORT, 'intake intent is immutable'); END;

CREATE TRIGGER intake_conversation_refs_immutable_update BEFORE UPDATE ON intake_conversation_refs
BEGIN SELECT RAISE(ABORT, 'intake conversation refs are append-only'); END;
CREATE TRIGGER intake_conversation_refs_immutable_delete BEFORE DELETE ON intake_conversation_refs
BEGIN SELECT RAISE(ABORT, 'intake conversation refs are append-only'); END;
CREATE TRIGGER intake_clarifications_immutable_update BEFORE UPDATE ON intake_clarifications
BEGIN SELECT RAISE(ABORT, 'intake clarifications are append-only'); END;
CREATE TRIGGER intake_clarifications_immutable_delete BEFORE DELETE ON intake_clarifications
BEGIN SELECT RAISE(ABORT, 'intake clarifications are append-only'); END;
CREATE TRIGGER intake_clarification_answers_immutable_update BEFORE UPDATE ON intake_clarification_answers
BEGIN SELECT RAISE(ABORT, 'intake clarification answers are append-only'); END;
CREATE TRIGGER intake_clarification_answers_immutable_delete BEFORE DELETE ON intake_clarification_answers
BEGIN SELECT RAISE(ABORT, 'intake clarification answers are append-only'); END;
CREATE TRIGGER intake_proposals_immutable_update BEFORE UPDATE ON intake_proposal_revisions
BEGIN SELECT RAISE(ABORT, 'intake proposals are append-only'); END;
CREATE TRIGGER intake_proposals_immutable_delete BEFORE DELETE ON intake_proposal_revisions
BEGIN SELECT RAISE(ABORT, 'intake proposals are append-only'); END;
CREATE TRIGGER intake_confirmations_immutable_update BEFORE UPDATE ON intake_confirmations
BEGIN SELECT RAISE(ABORT, 'intake confirmations are append-only'); END;
CREATE TRIGGER intake_confirmations_immutable_delete BEFORE DELETE ON intake_confirmations
BEGIN SELECT RAISE(ABORT, 'intake confirmations are append-only'); END;
CREATE TRIGGER contract_revision_intake_core_immutable_update BEFORE UPDATE ON contract_revision_intake_core
BEGIN SELECT RAISE(ABORT, 'contract intake core is append-only'); END;
CREATE TRIGGER contract_revision_intake_core_immutable_delete BEFORE DELETE ON contract_revision_intake_core
BEGIN SELECT RAISE(ABORT, 'contract intake core is append-only'); END;

CREATE TRIGGER responsibility_links_one_time_end BEFORE UPDATE ON responsibility_links
WHEN OLD.id <> NEW.id OR OLD.project_id <> NEW.project_id
  OR OLD.source_open_loop_id <> NEW.source_open_loop_id
  OR OLD.destination_outcome_id <> NEW.destination_outcome_id
  OR OLD.creator <> NEW.creator OR OLD.reason <> NEW.reason
  OR OLD.request_key <> NEW.request_key OR OLD.request_fingerprint <> NEW.request_fingerprint
  OR OLD.created_at <> NEW.created_at OR OLD.ended_at IS NOT NULL
BEGIN SELECT RAISE(ABORT, 'responsibility link lineage is immutable'); END;
CREATE TRIGGER responsibility_links_immutable_delete BEFORE DELETE ON responsibility_links
BEGIN SELECT RAISE(ABORT, 'responsibility links are append-only'); END;

-- Detach every writer before rebuilding the checked CDC vocabulary. Startup's
-- restoreChangeLogWriters recreates the complete trigger set after migrations.
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
DROP TRIGGER IF EXISTS attempts_cdc_insert;
DROP TRIGGER IF EXISTS attempts_cdc_update;
DROP TRIGGER IF EXISTS attempt_sessions_cdc_insert;
DROP TRIGGER IF EXISTS attempt_observations_cdc_insert;
DROP TRIGGER IF EXISTS attempt_recovery_receipts_cdc_insert;
DROP TRIGGER IF EXISTS evidence_items_cdc_insert;
DROP TRIGGER IF EXISTS verification_runs_cdc_insert;
DROP TRIGGER IF EXISTS acceptance_decisions_cdc_insert;
DROP TRIGGER IF EXISTS outcome_corrections_cdc_insert;

CREATE TABLE change_log_new (
    seq INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL REFERENCES projects (id),
    session_id TEXT REFERENCES sessions (id),
    event_type TEXT NOT NULL CHECK (event_type IN (
        'session_created', 'session_updated', 'pr_created', 'pr_updated',
        'pr_check_recorded', 'pr_session_changed', 'pr_review_thread_added',
        'pr_review_thread_resolved', 'outcome_created', 'outcome_updated',
        'outcome_contract_revised', 'outcome_plan_proposed', 'outcome_plan_approved',
        'outcome_attempt_started', 'outcome_attempt_updated',
        'outcome_attempt_session_bound', 'outcome_attempt_observed',
        'outcome_attempt_recovered', 'outcome_evidence_recorded',
        'outcome_verification_recorded', 'outcome_acceptance_decided',
        'outcome_correction_recorded', 'intake_captured', 'intake_updated',
        'intake_proposal_revised', 'intake_confirmed',
        'responsibility_link_created', 'responsibility_link_ended')),
    payload TEXT NOT NULL CHECK (json_valid(payload)),
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO change_log_new (seq, project_id, session_id, event_type, payload, created_at)
SELECT seq, project_id, session_id, event_type, payload, created_at FROM change_log;
DROP INDEX IF EXISTS idx_change_log_project;
DROP TABLE change_log;
ALTER TABLE change_log_new RENAME TO change_log;
CREATE INDEX idx_change_log_project ON change_log (project_id, seq);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE responsibility_links;
DROP TABLE contract_revision_intake_core;
DROP TABLE intake_confirmations;
DROP TABLE intake_proposal_revisions;
DROP TABLE intake_clarification_answers;
DROP TABLE intake_clarifications;
DROP TABLE intake_conversation_refs;
DROP TABLE intake_sessions;
-- +goose StatementEnd

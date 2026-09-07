-- Project Brief: durable, immutable Project context revisions.
--
-- The Brief grounds Outcome creation and planning. It deliberately carries no
-- Outcome lifecycle, authority, proof, acceptance, or execution state.

-- +goose Up
-- +goose StatementBegin
-- Detach every canonical change_log writer before rebuilding its checked
-- vocabulary. project_brief_revisions_cdc_insert is included even though it is
-- first created later in this migration so future repaired profiles are safe.
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
DROP TRIGGER IF EXISTS contribution_links_cdc_insert;
DROP TRIGGER IF EXISTS decomposition_revisions_cdc_insert;
DROP TRIGGER IF EXISTS decomposition_revisions_cdc_update;
DROP TRIGGER IF EXISTS contribution_waivers_cdc_insert;
DROP TRIGGER IF EXISTS decomposition_requests_cdc_insert;
DROP TRIGGER IF EXISTS decomposition_requests_cdc_update;
DROP TRIGGER IF EXISTS intake_sessions_cdc_insert;
DROP TRIGGER IF EXISTS intake_sessions_cdc_update;
DROP TRIGGER IF EXISTS intake_proposals_cdc_insert;
DROP TRIGGER IF EXISTS intake_confirmations_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_links_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_links_cdc_update;
DROP TRIGGER IF EXISTS waldo_conversations_cdc_insert;
DROP TRIGGER IF EXISTS waldo_conversation_episodes_cdc_insert;
DROP TRIGGER IF EXISTS waldo_conversation_episodes_cdc_update;
DROP TRIGGER IF EXISTS waldo_conversation_turns_cdc_insert;
DROP TRIGGER IF EXISTS waldo_context_attachments_cdc_insert;
DROP TRIGGER IF EXISTS waldo_context_attachments_cdc_update;
DROP TRIGGER IF EXISTS waldo_continuation_operations_cdc_insert;
DROP TRIGGER IF EXISTS waldo_continuation_operations_cdc_update;
DROP TRIGGER IF EXISTS waldo_continuation_receipts_cdc_insert;
DROP TRIGGER IF EXISTS project_brief_revisions_cdc_insert;

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
        'responsibility_link_created', 'responsibility_link_ended',
        'waldo_conversation_created', 'waldo_conversation_episode_opened',
        'waldo_conversation_episode_sealed', 'waldo_conversation_turn_appended',
        'waldo_conversation_context_attached', 'waldo_conversation_context_detached',
        'waldo_conversation_continuation_prepared', 'waldo_conversation_continuation_progressed',
        'waldo_conversation_continuation_recorded',
        'outcome_contribution_bound',
        'outcome_decomposition_proposed', 'outcome_decomposition_authorized',
        'outcome_contribution_dependency_waived',
        'outcome_decomposition_requested', 'outcome_decomposition_request_answered',
        'project_brief_revised')),
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

-- +goose StatementBegin
CREATE TABLE project_brief_revisions (
    id                   TEXT PRIMARY KEY,
    project_id           TEXT NOT NULL REFERENCES projects (id),
    revision_number      INTEGER NOT NULL CHECK (revision_number >= 1),
    purpose              TEXT NOT NULL CHECK (length(trim(purpose)) > 0),
    product_context      TEXT NOT NULL DEFAULT '',
    technical_context    TEXT NOT NULL DEFAULT '',
    architecture_summary TEXT NOT NULL DEFAULT '',
    conventions_json     TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(conventions_json) AND json_type(conventions_json) = 'array'),
    constraints_json     TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(constraints_json) AND json_type(constraints_json) = 'array'),
    setup_expectations   TEXT NOT NULL DEFAULT '',
    run_expectations     TEXT NOT NULL DEFAULT '',
    test_expectations    TEXT NOT NULL DEFAULT '',
    provenance_json      TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(provenance_json) AND json_type(provenance_json) = 'array'),
    created_at           TIMESTAMP NOT NULL,
    UNIQUE (project_id, revision_number)
);
CREATE INDEX idx_project_brief_revisions_project
    ON project_brief_revisions (project_id, revision_number DESC);

-- The head is an explicit read projection, not authority. Updating it cannot
-- change any prior revision; it only selects which immutable revision is current.
CREATE TABLE project_brief_heads (
    project_id              TEXT PRIMARY KEY REFERENCES projects (id),
    current_revision_number INTEGER NOT NULL CHECK (current_revision_number >= 1),
    updated_at              TIMESTAMP NOT NULL,
    FOREIGN KEY (project_id, current_revision_number)
        REFERENCES project_brief_revisions (project_id, revision_number)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER project_brief_revisions_freeze_update
BEFORE UPDATE ON project_brief_revisions
BEGIN
    SELECT RAISE(ABORT, 'project brief revisions are immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER project_brief_revisions_freeze_delete
BEFORE DELETE ON project_brief_revisions
BEGIN
    SELECT RAISE(ABORT, 'project brief revisions are immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER project_brief_revisions_cdc_insert
AFTER INSERT ON project_brief_revisions
BEGIN
    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)
    VALUES (
        NEW.project_id,
        NULL,
        'project_brief_revised',
        json_object('revisionId', NEW.id, 'revision', NEW.revision_number),
        NEW.created_at
    );
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS project_brief_revisions_cdc_insert;
DROP TRIGGER IF EXISTS project_brief_revisions_freeze_update;
DROP TRIGGER IF EXISTS project_brief_revisions_freeze_delete;
DROP TABLE IF EXISTS project_brief_heads;
DROP INDEX IF EXISTS idx_project_brief_revisions_project;
DROP TABLE IF EXISTS project_brief_revisions;
-- +goose StatementEnd

-- +goose StatementBegin
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
DROP TRIGGER IF EXISTS contribution_links_cdc_insert;
DROP TRIGGER IF EXISTS decomposition_revisions_cdc_insert;
DROP TRIGGER IF EXISTS decomposition_revisions_cdc_update;
DROP TRIGGER IF EXISTS contribution_waivers_cdc_insert;
DROP TRIGGER IF EXISTS decomposition_requests_cdc_insert;
DROP TRIGGER IF EXISTS decomposition_requests_cdc_update;
DROP TRIGGER IF EXISTS intake_sessions_cdc_insert;
DROP TRIGGER IF EXISTS intake_sessions_cdc_update;
DROP TRIGGER IF EXISTS intake_proposals_cdc_insert;
DROP TRIGGER IF EXISTS intake_confirmations_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_links_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_links_cdc_update;
DROP TRIGGER IF EXISTS waldo_conversations_cdc_insert;
DROP TRIGGER IF EXISTS waldo_conversation_episodes_cdc_insert;
DROP TRIGGER IF EXISTS waldo_conversation_episodes_cdc_update;
DROP TRIGGER IF EXISTS waldo_conversation_turns_cdc_insert;
DROP TRIGGER IF EXISTS waldo_context_attachments_cdc_insert;
DROP TRIGGER IF EXISTS waldo_context_attachments_cdc_update;
DROP TRIGGER IF EXISTS waldo_continuation_operations_cdc_insert;
DROP TRIGGER IF EXISTS waldo_continuation_operations_cdc_update;
DROP TRIGGER IF EXISTS waldo_continuation_receipts_cdc_insert;
DROP TRIGGER IF EXISTS project_brief_revisions_cdc_insert;

CREATE TABLE change_log_down (
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
        'responsibility_link_created', 'responsibility_link_ended',
        'waldo_conversation_created', 'waldo_conversation_episode_opened',
        'waldo_conversation_episode_sealed', 'waldo_conversation_turn_appended',
        'waldo_conversation_context_attached', 'waldo_conversation_context_detached',
        'waldo_conversation_continuation_prepared', 'waldo_conversation_continuation_progressed',
        'waldo_conversation_continuation_recorded',
        'outcome_contribution_bound',
        'outcome_decomposition_proposed', 'outcome_decomposition_authorized',
        'outcome_contribution_dependency_waived',
        'outcome_decomposition_requested', 'outcome_decomposition_request_answered')),
    payload TEXT NOT NULL CHECK (json_valid(payload)),
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO change_log_down (seq, project_id, session_id, event_type, payload, created_at)
SELECT seq, project_id, session_id, event_type, payload, created_at
FROM change_log WHERE event_type <> 'project_brief_revised';
DROP INDEX IF EXISTS idx_change_log_project;
DROP TABLE change_log;
ALTER TABLE change_log_down RENAME TO change_log;
CREATE INDEX idx_change_log_project ON change_log (project_id, seq);
-- +goose StatementEnd

-- +goose Up

-- Durable run intent: what the owner has authorized the daemon to keep doing
-- with an approved Plan.
--
-- Generations are append-only rows rather than a mutable status column. A
-- pause is a new generation, not an edit of the running one, which is what
-- lets a restarted daemon read the latest row and know exactly what it may
-- do, lets a client's stale command be refused by generation, and leaves the
-- owner a decision history that was never rewritten.
--
-- Only acknowledged_at is ever written after insert, and only once: it is the
-- difference between a pause the owner requested and a pause that has taken
-- effect.
CREATE TABLE outcome_run_intents (
    id                       TEXT PRIMARY KEY,
    outcome_id               TEXT NOT NULL REFERENCES outcomes (id),
    generation               INTEGER NOT NULL CHECK (generation >= 1),
    desired                  TEXT NOT NULL CHECK (desired IN ('idle', 'running', 'paused', 'cancelled')),
    plan_revision_id         TEXT NOT NULL DEFAULT '',
    contract_revision_number INTEGER NOT NULL DEFAULT 0,

    -- The owner command's replay identity. A repeated key returns the same
    -- generation instead of authorizing a second one, so a double click or a
    -- reconnect retry cannot start work twice.
    request_key     TEXT NOT NULL,
    requested_at    TIMESTAMP NOT NULL,
    acknowledged_at TIMESTAMP,

    UNIQUE (outcome_id, generation)
);

CREATE UNIQUE INDEX idx_outcome_run_intents_request_key
    ON outcome_run_intents (request_key);

CREATE INDEX idx_outcome_run_intents_current
    ON outcome_run_intents (outcome_id, generation DESC);

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
     OR OLD.requested_at <> NEW.requested_at
     -- Acknowledgement is write-once. Clearing it would let a pause that has
     -- already taken effect look like one still waiting.
     OR (OLD.acknowledged_at IS NOT NULL AND OLD.acknowledged_at IS NOT NEW.acknowledged_at)
BEGIN
    SELECT RAISE(ABORT, 'run intent generations are immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER outcome_run_intents_immutable_delete
BEFORE DELETE ON outcome_run_intents
BEGIN
    SELECT RAISE(ABORT, 'run intent generations are immutable');
END;
-- +goose StatementEnd

-- change_log's event vocabulary is a CHECK constraint, so admitting a new
-- canonical event means rebuilding the relation. Three are added at once
-- rather than rebuilding again per slice: run intent (emitted below),
-- retention and delivery (their writers arrive with their own tables).
--
-- Every CDC writer must be detached first and is restored by
-- restoreChangeLogWriters after goose completes; see cdc_restore.go.
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
DROP TRIGGER IF EXISTS outcome_run_intents_cdc_insert;
DROP TRIGGER IF EXISTS outcome_run_intents_cdc_acknowledged;
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
        'project_brief_revised',
        'outcome_run_intent_changed', 'outcome_attempt_retained',
        'outcome_delivery_changed')),
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
DROP TRIGGER IF EXISTS outcome_run_intents_cdc_acknowledged;
DROP TRIGGER IF EXISTS outcome_run_intents_cdc_insert;
DROP TRIGGER IF EXISTS outcome_run_intents_immutable_delete;
DROP TRIGGER IF EXISTS outcome_run_intents_immutable_update;
DROP INDEX IF EXISTS idx_outcome_run_intents_current;
DROP INDEX IF EXISTS idx_outcome_run_intents_request_key;
DROP TABLE IF EXISTS outcome_run_intents;

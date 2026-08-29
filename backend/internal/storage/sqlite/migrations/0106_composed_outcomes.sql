-- Composed Outcomes, phase 1 (ADR 0007): an Outcome may contribute to a
-- parent Outcome, and every contribution is bound to an exact parent
-- criterion at an exact parent ContractRevision.
--
-- Shape is NOT stored. Whether an Outcome is direct or decomposed is derived
-- from whether contributing Outcomes exist, so a row can never disagree with
-- its own children. Nothing here changes an Outcome that has no parent: every
-- Outcome created before this migration is the direct case.
--
-- This file only extends the checked change_log vocabulary. The composition
-- schema itself — the outcomes column, contribution_links, and the depth-cap
-- and binding guards — is installed by reconcileComposedOutcomesSchema after
-- goose completes, sqlite_master-guarded. A burned 0099 ledger entry leaves
-- `outcomes` physically absent even though its version is marked, and
-- ALTER TABLE cannot be made conditional in migration SQL.

-- +goose Up
-- +goose StatementBegin
-- Detach every writer before rebuilding the checked CDC vocabulary. Startup's
-- restoreChangeLogWriters recreates only writers whose tables exist.
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
DROP TRIGGER IF EXISTS intake_sessions_cdc_insert;
DROP TRIGGER IF EXISTS intake_sessions_cdc_update;
DROP TRIGGER IF EXISTS intake_proposals_cdc_insert;
DROP TRIGGER IF EXISTS intake_confirmations_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_links_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_links_cdc_update;

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
        'outcome_contribution_bound')),
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
-- Composition objects are dropped where present. The nullable
-- parent_outcome_id column is deliberately left in place: SQLite cannot drop
-- it conditionally, it is NULL for every Outcome once the links are gone, and
-- re-running the reconcile seam is idempotent.
DROP TRIGGER IF EXISTS contribution_links_cdc_insert;
DROP TRIGGER IF EXISTS contribution_links_immutable_delete;
DROP TRIGGER IF EXISTS contribution_links_immutable_update;
DROP TRIGGER IF EXISTS contribution_links_single_revision_guard;
DROP TRIGGER IF EXISTS contribution_links_binding_guard;
DROP TRIGGER IF EXISTS outcomes_composition_parent_guard;
DROP TRIGGER IF EXISTS outcomes_composition_depth_update;
DROP TRIGGER IF EXISTS outcomes_composition_depth_insert;

DROP INDEX IF EXISTS idx_contribution_links_child;
DROP INDEX IF EXISTS idx_contribution_links_parent;
DROP INDEX IF EXISTS idx_contribution_links_child_criterion;
DROP TABLE IF EXISTS contribution_links;
DROP INDEX IF EXISTS idx_outcomes_parent;
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
DROP TRIGGER IF EXISTS intake_sessions_cdc_insert;
DROP TRIGGER IF EXISTS intake_sessions_cdc_update;
DROP TRIGGER IF EXISTS intake_proposals_cdc_insert;
DROP TRIGGER IF EXISTS intake_confirmations_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_links_cdc_insert;
DROP TRIGGER IF EXISTS responsibility_links_cdc_update;

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
        'waldo_conversation_continuation_recorded')),
    payload TEXT NOT NULL CHECK (json_valid(payload)),
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO change_log_down (seq, project_id, session_id, event_type, payload, created_at)
SELECT seq, project_id, session_id, event_type, payload, created_at
FROM change_log WHERE event_type <> 'outcome_contribution_bound';
DROP INDEX IF EXISTS idx_change_log_project;
DROP TABLE change_log;
ALTER TABLE change_log_down RENAME TO change_log;
CREATE INDEX idx_change_log_project ON change_log (project_id, seq);
-- +goose StatementEnd

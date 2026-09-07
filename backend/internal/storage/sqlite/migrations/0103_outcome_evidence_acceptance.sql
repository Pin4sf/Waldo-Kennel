-- Work E (#35): stable ContractCriterion identity plus append-only Evidence,
-- Verification, explicit user Acceptance/Rework/Reopen, and correction
-- lineage. Display status remains derived at service read time.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE contract_criteria (
    id                   TEXT NOT NULL,
    contract_revision_id TEXT NOT NULL REFERENCES contract_revisions (id),
    position             INTEGER NOT NULL CHECK (position >= 1),
    text                 TEXT NOT NULL CHECK (length(trim(text)) > 0),
    PRIMARY KEY (contract_revision_id, id),
    UNIQUE (contract_revision_id, position)
);

-- Existing revisions are backfilled by reconcileOutcomeProofSchema after
-- Goose completes. Keeping that data-copy in the startup seam lets this
-- migration survive a field profile where upstream burned 0099 and therefore
-- contract_revisions is physically absent even though its version is marked.

CREATE TABLE evidence_items (
    id                   TEXT PRIMARY KEY,
    outcome_id           TEXT NOT NULL,
    contract_revision_id TEXT NOT NULL,
    criterion_id         TEXT NOT NULL,
    subject_type         TEXT NOT NULL CHECK (subject_type IN ('outcome', 'contract', 'plan', 'work_unit', 'attempt')),
    subject_id           TEXT NOT NULL CHECK (length(trim(subject_id)) > 0),
    subject_revision     TEXT NOT NULL CHECK (length(trim(subject_revision)) > 0),
    kind                 TEXT NOT NULL CHECK (kind IN ('supporting', 'contradicting')),
    source_type          TEXT NOT NULL CHECK (source_type IN ('artifact', 'deterministic_check', 'provider_output', 'owner_walkthrough')),
    source_ref           TEXT NOT NULL CHECK (length(trim(source_ref)) > 0),
    producer_type        TEXT NOT NULL CHECK (producer_type IN ('user', 'provider', 'tool')),
    producer_ref         TEXT NOT NULL CHECK (length(trim(producer_ref)) > 0),
    summary              TEXT NOT NULL CHECK (length(trim(summary)) > 0),
    content_digest       TEXT NOT NULL CHECK (length(content_digest) = 64),
    request_key          TEXT NOT NULL,
    request_fingerprint  TEXT NOT NULL CHECK (length(request_fingerprint) = 64),
    created_at           TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (outcome_id) REFERENCES outcomes (id),
    FOREIGN KEY (contract_revision_id) REFERENCES contract_revisions (id),
    FOREIGN KEY (contract_revision_id, criterion_id)
        REFERENCES contract_criteria (contract_revision_id, id)
);

CREATE UNIQUE INDEX idx_evidence_items_request_key ON evidence_items (request_key);
CREATE INDEX idx_evidence_items_outcome_revision
    ON evidence_items (outcome_id, contract_revision_id, criterion_id, created_at, id);

CREATE TABLE verification_runs (
    id                   TEXT PRIMARY KEY,
    outcome_id           TEXT NOT NULL,
    contract_revision_id TEXT NOT NULL,
    criterion_id         TEXT NOT NULL,
    subject_type         TEXT NOT NULL CHECK (subject_type IN ('outcome', 'contract', 'plan', 'work_unit', 'attempt')),
    subject_id           TEXT NOT NULL CHECK (length(trim(subject_id)) > 0),
    subject_revision     TEXT NOT NULL CHECK (length(trim(subject_revision)) > 0),
    evidence_item_ids    TEXT NOT NULL CHECK (json_valid(evidence_item_ids)),
    method               TEXT NOT NULL CHECK (length(trim(method)) > 0),
    independence_class   TEXT NOT NULL CHECK (independence_class IN (
                             'deterministic', 'producer_self_check', 'separate_session',
                             'cross_provider', 'owner_walkthrough')),
    result               TEXT NOT NULL CHECK (result IN ('passed', 'failed', 'inconclusive', 'exception')),
    producer_ref         TEXT NOT NULL DEFAULT '',
    verifier_ref         TEXT NOT NULL CHECK (length(trim(verifier_ref)) > 0),
    producer_provider    TEXT NOT NULL DEFAULT '',
    verifier_provider    TEXT NOT NULL DEFAULT '',
    detail               TEXT NOT NULL DEFAULT '',
    request_key          TEXT NOT NULL,
    request_fingerprint  TEXT NOT NULL CHECK (length(request_fingerprint) = 64),
    created_at           TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (outcome_id) REFERENCES outcomes (id),
    FOREIGN KEY (contract_revision_id) REFERENCES contract_revisions (id),
    FOREIGN KEY (contract_revision_id, criterion_id)
        REFERENCES contract_criteria (contract_revision_id, id)
);

CREATE UNIQUE INDEX idx_verification_runs_request_key ON verification_runs (request_key);
CREATE INDEX idx_verification_runs_outcome_revision
    ON verification_runs (outcome_id, contract_revision_id, criterion_id, created_at, id);

CREATE TABLE acceptance_decisions (
    id                   TEXT PRIMARY KEY,
    outcome_id           TEXT NOT NULL,
    contract_revision_id TEXT NOT NULL,
    kind                 TEXT NOT NULL CHECK (kind IN ('accept', 'request_rework', 'reopen')),
    actor_type           TEXT NOT NULL CHECK (actor_type = 'user'),
    summary              TEXT NOT NULL CHECK (length(trim(summary)) > 0),
    resource_disposition TEXT NOT NULL CHECK (resource_disposition IN ('retain', 'cleanup_later', 'not_applicable')),
    request_key          TEXT NOT NULL,
    request_fingerprint  TEXT NOT NULL CHECK (length(request_fingerprint) = 64),
    created_at           TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (outcome_id) REFERENCES outcomes (id),
    FOREIGN KEY (contract_revision_id) REFERENCES contract_revisions (id)
);

CREATE UNIQUE INDEX idx_acceptance_decisions_request_key ON acceptance_decisions (request_key);
CREATE INDEX idx_acceptance_decisions_outcome_revision
    ON acceptance_decisions (outcome_id, contract_revision_id, created_at, id);

CREATE TABLE outcome_corrections (
    id                   TEXT PRIMARY KEY,
    decision_id          TEXT NOT NULL UNIQUE REFERENCES acceptance_decisions (id),
    outcome_id           TEXT NOT NULL,
    contract_revision_id TEXT NOT NULL,
    feedback             TEXT NOT NULL CHECK (length(trim(feedback)) > 0),
    target_type          TEXT NOT NULL CHECK (target_type IN ('attempt', 'work_unit', 'plan', 'contract')),
    target_id            TEXT NOT NULL CHECK (length(trim(target_id)) > 0),
    created_at           TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (outcome_id) REFERENCES outcomes (id),
    FOREIGN KEY (contract_revision_id) REFERENCES contract_revisions (id)
);

DROP TRIGGER IF EXISTS contract_criteria_immutable_update;
CREATE TRIGGER contract_criteria_immutable_update BEFORE UPDATE ON contract_criteria
BEGIN SELECT RAISE(ABORT, 'contract criteria are append-only'); END;
DROP TRIGGER IF EXISTS contract_criteria_immutable_delete;
CREATE TRIGGER contract_criteria_immutable_delete BEFORE DELETE ON contract_criteria
BEGIN SELECT RAISE(ABORT, 'contract criteria are append-only'); END;

DROP TRIGGER IF EXISTS evidence_items_immutable_update;
CREATE TRIGGER evidence_items_immutable_update BEFORE UPDATE ON evidence_items
BEGIN SELECT RAISE(ABORT, 'evidence items are append-only'); END;
DROP TRIGGER IF EXISTS evidence_items_immutable_delete;
CREATE TRIGGER evidence_items_immutable_delete BEFORE DELETE ON evidence_items
BEGIN SELECT RAISE(ABORT, 'evidence items are append-only'); END;

DROP TRIGGER IF EXISTS verification_runs_immutable_update;
CREATE TRIGGER verification_runs_immutable_update BEFORE UPDATE ON verification_runs
BEGIN SELECT RAISE(ABORT, 'verification runs are append-only'); END;
DROP TRIGGER IF EXISTS verification_runs_immutable_delete;
CREATE TRIGGER verification_runs_immutable_delete BEFORE DELETE ON verification_runs
BEGIN SELECT RAISE(ABORT, 'verification runs are append-only'); END;

DROP TRIGGER IF EXISTS acceptance_decisions_immutable_update;
CREATE TRIGGER acceptance_decisions_immutable_update BEFORE UPDATE ON acceptance_decisions
BEGIN SELECT RAISE(ABORT, 'acceptance decisions are append-only'); END;
DROP TRIGGER IF EXISTS acceptance_decisions_immutable_delete;
CREATE TRIGGER acceptance_decisions_immutable_delete BEFORE DELETE ON acceptance_decisions
BEGIN SELECT RAISE(ABORT, 'acceptance decisions are append-only'); END;

DROP TRIGGER IF EXISTS outcome_corrections_immutable_update;
CREATE TRIGGER outcome_corrections_immutable_update BEFORE UPDATE ON outcome_corrections
BEGIN SELECT RAISE(ABORT, 'outcome corrections are append-only'); END;
DROP TRIGGER IF EXISTS outcome_corrections_immutable_delete;
CREATE TRIGGER outcome_corrections_immutable_delete BEFORE DELETE ON outcome_corrections
BEGIN SELECT RAISE(ABORT, 'outcome corrections are append-only'); END;

-- Detach every change_log writer before rebuilding its checked vocabulary.
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

CREATE TABLE change_log_new (
    seq        INTEGER PRIMARY KEY AUTOINCREMENT,
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
        'outcome_correction_recorded')),
    payload    TEXT NOT NULL CHECK (json_valid(payload)),
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
DROP TRIGGER IF EXISTS evidence_items_cdc_insert;
DROP TRIGGER IF EXISTS verification_runs_cdc_insert;
DROP TRIGGER IF EXISTS acceptance_decisions_cdc_insert;
DROP TRIGGER IF EXISTS outcome_corrections_cdc_insert;

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

CREATE TABLE change_log_down (
    seq        INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL REFERENCES projects (id),
    session_id TEXT REFERENCES sessions (id),
    event_type TEXT NOT NULL CHECK (event_type IN (
        'session_created', 'session_updated', 'pr_created', 'pr_updated',
        'pr_check_recorded', 'pr_session_changed', 'pr_review_thread_added',
        'pr_review_thread_resolved', 'outcome_created', 'outcome_updated',
        'outcome_contract_revised', 'outcome_plan_proposed', 'outcome_plan_approved',
        'outcome_attempt_started', 'outcome_attempt_updated',
        'outcome_attempt_session_bound', 'outcome_attempt_observed',
        'outcome_attempt_recovered')),
    payload    TEXT NOT NULL CHECK (json_valid(payload)),
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO change_log_down (seq, project_id, session_id, event_type, payload, created_at)
SELECT seq, project_id, session_id, event_type, payload, created_at
FROM change_log WHERE event_type NOT IN (
    'outcome_evidence_recorded', 'outcome_verification_recorded',
    'outcome_acceptance_decided', 'outcome_correction_recorded');
DROP INDEX IF EXISTS idx_change_log_project;
DROP TABLE change_log;
ALTER TABLE change_log_down RENAME TO change_log;
CREATE INDEX idx_change_log_project ON change_log (project_id, seq);

DROP TRIGGER IF EXISTS outcome_corrections_immutable_delete;
DROP TRIGGER IF EXISTS outcome_corrections_immutable_update;
DROP TRIGGER IF EXISTS acceptance_decisions_immutable_delete;
DROP TRIGGER IF EXISTS acceptance_decisions_immutable_update;
DROP TRIGGER IF EXISTS verification_runs_immutable_delete;
DROP TRIGGER IF EXISTS verification_runs_immutable_update;
DROP TRIGGER IF EXISTS evidence_items_immutable_delete;
DROP TRIGGER IF EXISTS evidence_items_immutable_update;
DROP TRIGGER IF EXISTS contract_criteria_immutable_delete;
DROP TRIGGER IF EXISTS contract_criteria_immutable_update;
DROP TABLE IF EXISTS outcome_corrections;
DROP TABLE IF EXISTS acceptance_decisions;
DROP TABLE IF EXISTS verification_runs;
DROP TABLE IF EXISTS evidence_items;
DROP TABLE IF EXISTS contract_criteria;
-- +goose StatementEnd

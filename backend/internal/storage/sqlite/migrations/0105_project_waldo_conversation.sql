-- Durable Project-scoped Waldo conversation and bounded provider continuation
-- lineage (#77). Provider-native transcripts remain outside this aggregate:
-- only opaque provider episode/turn/transcript identifiers are retained.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE waldo_conversations (
    id                   TEXT PRIMARY KEY,
    project_id           TEXT NOT NULL UNIQUE REFERENCES projects (id),
    revision             INTEGER NOT NULL DEFAULT 0 CHECK (revision >= 0),
    latest_turn_sequence INTEGER NOT NULL DEFAULT 0 CHECK (latest_turn_sequence >= 0),
    created_at           TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at           TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE waldo_conversation_episodes (
    id                       TEXT PRIMARY KEY,
    conversation_id          TEXT NOT NULL REFERENCES waldo_conversations (id),
    project_id               TEXT NOT NULL REFERENCES projects (id),
    ordinal                  INTEGER NOT NULL CHECK (ordinal >= 1),
    state                    TEXT NOT NULL CHECK (state IN ('active', 'sealed')),
    provider                 TEXT NOT NULL DEFAULT '',
    provider_conversation_id TEXT NOT NULL DEFAULT '',
    transcript_ref           TEXT NOT NULL DEFAULT '',
    request_key              TEXT NOT NULL UNIQUE,
    request_fingerprint      TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    created_at               TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    sealed_at                TIMESTAMP,
    seal_reason              TEXT NOT NULL DEFAULT '',
    UNIQUE (conversation_id, ordinal),
    CHECK ((provider = '' AND provider_conversation_id = '' AND transcript_ref = '') OR
           (provider <> '' AND (provider_conversation_id <> '' OR transcript_ref <> ''))),
    CHECK ((state = 'active' AND sealed_at IS NULL AND seal_reason = '') OR
           (state = 'sealed' AND sealed_at IS NOT NULL AND length(trim(seal_reason)) > 0))
);

CREATE UNIQUE INDEX idx_waldo_conversation_episodes_active
    ON waldo_conversation_episodes (conversation_id) WHERE state = 'active';

CREATE TABLE waldo_conversation_turns (
    id                       TEXT PRIMARY KEY,
    conversation_id          TEXT NOT NULL REFERENCES waldo_conversations (id),
    episode_id               TEXT NOT NULL REFERENCES waldo_conversation_episodes (id),
    project_id               TEXT NOT NULL REFERENCES projects (id),
    sequence                 INTEGER NOT NULL CHECK (sequence >= 1),
    role                     TEXT NOT NULL CHECK (role IN ('user', 'waldo')),
    message                  TEXT NOT NULL CHECK (length(trim(message)) > 0),
    provider                 TEXT NOT NULL DEFAULT '',
    provider_conversation_id TEXT NOT NULL DEFAULT '',
    provider_turn_id         TEXT NOT NULL DEFAULT '',
    transcript_ref           TEXT NOT NULL DEFAULT '',
    request_key              TEXT NOT NULL UNIQUE,
    request_fingerprint      TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    created_at               TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    UNIQUE (conversation_id, sequence),
    CHECK ((provider = '' AND provider_conversation_id = '' AND provider_turn_id = '' AND transcript_ref = '') OR
           (provider <> '' AND provider_turn_id <> '' AND
            (provider_conversation_id <> '' OR transcript_ref <> '')))
);

CREATE INDEX idx_waldo_conversation_turns_episode
    ON waldo_conversation_turns (episode_id, sequence);

CREATE TABLE waldo_context_attachments (
    id                         TEXT PRIMARY KEY,
    conversation_id            TEXT NOT NULL REFERENCES waldo_conversations (id),
    project_id                 TEXT NOT NULL REFERENCES projects (id),
    kind                       TEXT NOT NULL CHECK (kind IN (
                                   'project', 'outcome', 'contract_revision', 'plan_revision',
                                   'work_unit', 'attempt', 'agent_session_ref', 'intake_session')),
    object_id                  TEXT NOT NULL CHECK (length(trim(object_id)) > 0),
    object_revision            TEXT NOT NULL,
    provenance_kind            TEXT NOT NULL CHECK (provenance_kind IN (
                                   'user', 'canonical', 'intake', 'provider', 'retrieval', 'correction')),
    provenance_ref             TEXT NOT NULL CHECK (length(trim(provenance_ref)) > 0),
    attached_revision          INTEGER NOT NULL CHECK (attached_revision >= 1),
    attach_request_key         TEXT NOT NULL UNIQUE,
    attach_request_fingerprint TEXT NOT NULL CHECK (length(trim(attach_request_fingerprint)) > 0),
    created_at                 TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    detached_revision          INTEGER,
    detached_at                TIMESTAMP,
    detach_reason              TEXT NOT NULL DEFAULT '',
    detach_request_key         TEXT UNIQUE,
    detach_request_fingerprint TEXT NOT NULL DEFAULT '',
    CHECK ((kind = 'project' AND object_revision = '') OR
           (kind <> 'project' AND length(trim(object_revision)) > 0)),
    CHECK ((detached_at IS NULL AND detached_revision IS NULL AND detach_reason = '' AND
            detach_request_key IS NULL AND detach_request_fingerprint = '') OR
           (detached_at IS NOT NULL AND detached_revision > attached_revision AND
            length(trim(detach_reason)) > 0 AND length(trim(detach_request_key)) > 0 AND
            length(trim(detach_request_fingerprint)) > 0))
);

CREATE UNIQUE INDEX idx_waldo_context_attachments_active
    ON waldo_context_attachments (conversation_id, kind, object_id)
    WHERE detached_at IS NULL;

CREATE TABLE waldo_turn_context_refs (
    turn_id       TEXT NOT NULL REFERENCES waldo_conversation_turns (id),
    attachment_id TEXT NOT NULL REFERENCES waldo_context_attachments (id),
    position      INTEGER NOT NULL CHECK (position >= 1),
    PRIMARY KEY (turn_id, position),
    UNIQUE (turn_id, attachment_id)
);

CREATE TABLE waldo_continuation_operations (
    id                             TEXT PRIMARY KEY,
    conversation_id                TEXT NOT NULL REFERENCES waldo_conversations (id),
    project_id                     TEXT NOT NULL REFERENCES projects (id),
    from_episode_id                TEXT NOT NULL REFERENCES waldo_conversation_episodes (id),
    from_agent_session_ref_id      TEXT NOT NULL REFERENCES attempt_sessions (id),
    expected_conversation_revision INTEGER NOT NULL CHECK (expected_conversation_revision >= 1),
    state                          TEXT NOT NULL CHECK (state IN (
                                       'prepared', 'fencing', 'fenced', 'starting', 'completed')),
    reason                         TEXT NOT NULL CHECK (reason IN (
                                       'context_reserve', 'conservative_threshold',
                                       'material_digest_change', 'identity_lost',
                                       'source_revoked', 'fresh_verifier', 'user_requested')),
    reason_detail                  TEXT NOT NULL CHECK (length(trim(reason_detail)) > 0),
    trigger_evidence_kind          TEXT NOT NULL CHECK (trigger_evidence_kind IN (
                                       'provider_context_meter', 'adapter_conservative_threshold',
                                       'material_context_digest', 'provider_identity_loss',
                                       'source_revocation', 'verifier_boundary', 'owner_request')),
    trigger_evidence_ref           TEXT NOT NULL CHECK (length(trim(trigger_evidence_ref)) > 0),
    material_change                INTEGER NOT NULL CHECK (material_change IN (0, 1)),
    changed_fields                 TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(changed_fields)),
    context_digest                 TEXT NOT NULL CHECK (length(context_digest) = 64),
    context_refs                   TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(context_refs)),
    previous_bindings              TEXT NOT NULL CHECK (json_valid(previous_bindings)),
    replacement_bindings           TEXT NOT NULL CHECK (json_valid(replacement_bindings)),
    effects_known                  INTEGER NOT NULL CHECK (effects_known IN (0, 1)),
    lost_material_context          INTEGER NOT NULL CHECK (lost_material_context IN (0, 1)),
    source_revoked                 INTEGER NOT NULL CHECK (source_revoked IN (0, 1)),
    fresh_verifier                 INTEGER NOT NULL CHECK (fresh_verifier IN (0, 1)),
    trigger_confirmed              INTEGER NOT NULL CHECK (trigger_confirmed IN (0, 1)),
    fence_receipt_ref              TEXT NOT NULL DEFAULT '',
    reconciliation_ref             TEXT NOT NULL DEFAULT '',
    needs_user_reason              TEXT NOT NULL DEFAULT '',
    request_key                    TEXT NOT NULL UNIQUE,
    request_fingerprint            TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    created_at                     TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at                     TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    CHECK ((reason = 'context_reserve' AND trigger_evidence_kind = 'provider_context_meter') OR
           (reason = 'conservative_threshold' AND trigger_evidence_kind = 'adapter_conservative_threshold') OR
           (reason = 'material_digest_change' AND trigger_evidence_kind = 'material_context_digest') OR
           (reason = 'identity_lost' AND trigger_evidence_kind = 'provider_identity_loss') OR
           (reason = 'source_revoked' AND trigger_evidence_kind = 'source_revocation') OR
           (reason = 'fresh_verifier' AND trigger_evidence_kind = 'verifier_boundary') OR
           (reason = 'user_requested' AND trigger_evidence_kind = 'owner_request')),
    CHECK (material_change = CASE WHEN
           json_array_length(changed_fields) > 0 OR lost_material_context = 1 OR
           source_revoked = 1 OR reason = 'material_digest_change'
           THEN 1 ELSE 0 END),
    CHECK (state <> 'prepared' OR (fence_receipt_ref = '' AND reconciliation_ref = '')),
    CHECK (state NOT IN ('fenced', 'starting') OR length(trim(fence_receipt_ref)) > 0)
);

CREATE UNIQUE INDEX idx_waldo_continuation_operations_active
    ON waldo_continuation_operations (conversation_id) WHERE state <> 'completed';

CREATE TABLE waldo_continuation_receipts (
    id                             TEXT PRIMARY KEY,
    operation_id                   TEXT NOT NULL UNIQUE REFERENCES waldo_continuation_operations (id),
    conversation_id                TEXT NOT NULL REFERENCES waldo_conversations (id),
    project_id                     TEXT NOT NULL REFERENCES projects (id),
    from_episode_id                TEXT NOT NULL REFERENCES waldo_conversation_episodes (id),
    to_episode_id                  TEXT REFERENCES waldo_conversation_episodes (id),
    from_agent_session_ref_id      TEXT NOT NULL REFERENCES attempt_sessions (id),
    to_agent_session_ref_id        TEXT REFERENCES attempt_sessions (id),
    action                         TEXT NOT NULL CHECK (action IN ('automatic', 'needs_you', 'unconfirmed')),
    reason                         TEXT NOT NULL CHECK (reason IN (
                                       'context_reserve', 'conservative_threshold',
                                       'material_digest_change', 'identity_lost',
                                       'source_revoked', 'fresh_verifier', 'user_requested')),
    reason_detail                  TEXT NOT NULL CHECK (length(trim(reason_detail)) > 0),
    trigger_evidence_kind          TEXT NOT NULL CHECK (trigger_evidence_kind IN (
                                       'provider_context_meter', 'adapter_conservative_threshold',
                                       'material_context_digest', 'provider_identity_loss',
                                       'source_revocation', 'verifier_boundary', 'owner_request')),
    trigger_evidence_ref           TEXT NOT NULL CHECK (length(trim(trigger_evidence_ref)) > 0),
    material_change                INTEGER NOT NULL CHECK (material_change IN (0, 1)),
    changed_fields                 TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(changed_fields)),
    context_digest                 TEXT NOT NULL CHECK (length(context_digest) = 64),
    context_refs                   TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(context_refs)),
    previous_bindings              TEXT NOT NULL CHECK (json_valid(previous_bindings)),
    replacement_bindings           TEXT NOT NULL CHECK (json_valid(replacement_bindings)),
    effects_known                  INTEGER NOT NULL CHECK (effects_known IN (0, 1)),
    old_session_fenced             INTEGER NOT NULL CHECK (old_session_fenced IN (0, 1)),
    replacement_identity_confirmed INTEGER NOT NULL CHECK (replacement_identity_confirmed IN (0, 1)),
    fence_receipt_ref              TEXT NOT NULL DEFAULT '',
    reconciliation_ref             TEXT NOT NULL DEFAULT '',
    needs_user_reason              TEXT NOT NULL DEFAULT '',
    request_key                    TEXT NOT NULL UNIQUE,
    request_fingerprint            TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    created_at                     TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    CHECK ((reason = 'context_reserve' AND trigger_evidence_kind = 'provider_context_meter') OR
           (reason = 'conservative_threshold' AND trigger_evidence_kind = 'adapter_conservative_threshold') OR
           (reason = 'material_digest_change' AND trigger_evidence_kind = 'material_context_digest') OR
           (reason = 'identity_lost' AND trigger_evidence_kind = 'provider_identity_loss') OR
           (reason = 'source_revoked' AND trigger_evidence_kind = 'source_revocation') OR
           (reason = 'fresh_verifier' AND trigger_evidence_kind = 'verifier_boundary') OR
           (reason = 'user_requested' AND trigger_evidence_kind = 'owner_request')),
    CHECK (
        (action = 'automatic' AND material_change = 0 AND effects_known = 1 AND
         old_session_fenced = 1 AND replacement_identity_confirmed = 1 AND
         to_episode_id IS NOT NULL AND to_agent_session_ref_id IS NOT NULL AND
         length(trim(fence_receipt_ref)) > 0 AND needs_user_reason = '')
        OR
        (action = 'needs_you' AND to_episode_id IS NULL AND to_agent_session_ref_id IS NULL AND
         replacement_identity_confirmed = 0 AND length(trim(needs_user_reason)) > 0)
        OR
        (action = 'unconfirmed' AND to_episode_id IS NULL AND to_agent_session_ref_id IS NULL AND
         old_session_fenced = 1 AND replacement_identity_confirmed = 0 AND
         length(trim(needs_user_reason)) > 0)
    )
);

CREATE INDEX idx_waldo_continuation_receipts_conversation
    ON waldo_continuation_receipts (conversation_id, created_at, id);

CREATE TRIGGER waldo_conversations_revision_guard BEFORE UPDATE ON waldo_conversations
WHEN OLD.id <> NEW.id OR OLD.project_id <> NEW.project_id OR OLD.created_at <> NEW.created_at
  OR NEW.revision <> OLD.revision + 1
  OR NEW.latest_turn_sequence < OLD.latest_turn_sequence
  OR NEW.latest_turn_sequence > OLD.latest_turn_sequence + 1
  OR EXISTS (SELECT 1 FROM waldo_continuation_operations operation
             WHERE operation.conversation_id = OLD.id AND operation.state <> 'completed')
BEGIN SELECT RAISE(ABORT, 'Waldo conversation revision/order conflict'); END;
CREATE TRIGGER waldo_conversations_immutable_delete BEFORE DELETE ON waldo_conversations
BEGIN SELECT RAISE(ABORT, 'Waldo conversations are durable'); END;

CREATE TRIGGER waldo_episodes_project_binding BEFORE INSERT ON waldo_conversation_episodes
WHEN NOT EXISTS (
    SELECT 1 FROM waldo_conversations c
    WHERE c.id = NEW.conversation_id AND c.project_id = NEW.project_id)
  OR NEW.ordinal <> (SELECT COALESCE(MAX(ordinal), 0) + 1
                     FROM waldo_conversation_episodes
                     WHERE conversation_id = NEW.conversation_id)
BEGIN SELECT RAISE(ABORT, 'Waldo episode Project binding conflict'); END;
CREATE TRIGGER waldo_episodes_one_time_seal BEFORE UPDATE ON waldo_conversation_episodes
WHEN OLD.id <> NEW.id OR OLD.conversation_id <> NEW.conversation_id OR OLD.project_id <> NEW.project_id
  OR OLD.ordinal <> NEW.ordinal OR OLD.provider <> NEW.provider
  OR OLD.provider_conversation_id <> NEW.provider_conversation_id OR OLD.transcript_ref <> NEW.transcript_ref
  OR OLD.request_key <> NEW.request_key OR OLD.request_fingerprint <> NEW.request_fingerprint
  OR OLD.created_at <> NEW.created_at OR OLD.state <> 'active' OR NEW.state <> 'sealed'
BEGIN SELECT RAISE(ABORT, 'Waldo episode lineage is immutable'); END;
CREATE TRIGGER waldo_episodes_immutable_delete BEFORE DELETE ON waldo_conversation_episodes
BEGIN SELECT RAISE(ABORT, 'Waldo episodes are append-only'); END;

CREATE TRIGGER waldo_turns_binding BEFORE INSERT ON waldo_conversation_turns
WHEN NOT EXISTS (
    SELECT 1 FROM waldo_conversation_episodes e
    JOIN waldo_conversations c ON c.id = e.conversation_id
    WHERE e.id = NEW.episode_id AND e.conversation_id = NEW.conversation_id
      AND e.project_id = NEW.project_id AND c.project_id = NEW.project_id AND e.state = 'active'
      AND NEW.sequence = c.latest_turn_sequence + 1)
BEGIN SELECT RAISE(ABORT, 'Waldo turn episode/Project binding conflict'); END;
CREATE TRIGGER waldo_turns_immutable_update BEFORE UPDATE ON waldo_conversation_turns
BEGIN SELECT RAISE(ABORT, 'Waldo turns are append-only'); END;
CREATE TRIGGER waldo_turns_immutable_delete BEFORE DELETE ON waldo_conversation_turns
BEGIN SELECT RAISE(ABORT, 'Waldo turns are append-only'); END;

CREATE TRIGGER waldo_context_binding BEFORE INSERT ON waldo_context_attachments
WHEN NOT EXISTS (
    SELECT 1 FROM waldo_conversations c
    WHERE c.id = NEW.conversation_id AND c.project_id = NEW.project_id
      AND NEW.attached_revision = c.revision + 1)
  OR (NEW.kind = 'project' AND NEW.object_id <> NEW.project_id)
BEGIN SELECT RAISE(ABORT, 'Waldo context Project binding conflict'); END;
CREATE TRIGGER waldo_context_one_time_detach BEFORE UPDATE ON waldo_context_attachments
WHEN OLD.id <> NEW.id OR OLD.conversation_id <> NEW.conversation_id OR OLD.project_id <> NEW.project_id
  OR OLD.kind <> NEW.kind OR OLD.object_id <> NEW.object_id OR OLD.object_revision <> NEW.object_revision
  OR OLD.provenance_kind <> NEW.provenance_kind OR OLD.provenance_ref <> NEW.provenance_ref
  OR OLD.attached_revision <> NEW.attached_revision
  OR OLD.attach_request_key <> NEW.attach_request_key
  OR OLD.attach_request_fingerprint <> NEW.attach_request_fingerprint
  OR OLD.created_at <> NEW.created_at OR OLD.detached_at IS NOT NULL
  OR NEW.detached_revision <> (SELECT revision + 1 FROM waldo_conversations
                               WHERE id = NEW.conversation_id)
BEGIN SELECT RAISE(ABORT, 'Waldo context detach is one-time'); END;
CREATE TRIGGER waldo_context_immutable_delete BEFORE DELETE ON waldo_context_attachments
BEGIN SELECT RAISE(ABORT, 'Waldo context attachments are append-only'); END;

CREATE TRIGGER waldo_turn_context_binding BEFORE INSERT ON waldo_turn_context_refs
WHEN NOT EXISTS (
    SELECT 1 FROM waldo_conversation_turns t
    JOIN waldo_context_attachments a ON a.id = NEW.attachment_id
    WHERE t.id = NEW.turn_id AND t.conversation_id = a.conversation_id
      AND t.project_id = a.project_id AND a.detached_at IS NULL)
BEGIN SELECT RAISE(ABORT, 'Waldo turn context binding conflict'); END;
CREATE TRIGGER waldo_turn_context_immutable_update BEFORE UPDATE ON waldo_turn_context_refs
BEGIN SELECT RAISE(ABORT, 'Waldo turn context refs are append-only'); END;
CREATE TRIGGER waldo_turn_context_immutable_delete BEFORE DELETE ON waldo_turn_context_refs
BEGIN SELECT RAISE(ABORT, 'Waldo turn context refs are append-only'); END;

CREATE TRIGGER waldo_continuation_operation_binding BEFORE INSERT ON waldo_continuation_operations
WHEN NOT EXISTS (
    SELECT 1 FROM waldo_conversation_episodes source
    JOIN waldo_conversations c ON c.id = source.conversation_id
    JOIN attempt_sessions session_ref ON session_ref.id = NEW.from_agent_session_ref_id
    WHERE source.id = NEW.from_episode_id AND source.conversation_id = NEW.conversation_id
      AND source.project_id = NEW.project_id AND source.state = 'active'
      AND c.project_id = NEW.project_id AND c.revision = NEW.expected_conversation_revision
      AND length(trim(session_ref.attempt_id)) > 0)
BEGIN SELECT RAISE(ABORT, 'Waldo continuation operation binding conflict'); END;
CREATE TRIGGER waldo_continuation_operation_transition BEFORE UPDATE ON waldo_continuation_operations
WHEN OLD.id <> NEW.id OR OLD.conversation_id <> NEW.conversation_id OR OLD.project_id <> NEW.project_id
  OR OLD.from_episode_id <> NEW.from_episode_id
  OR OLD.from_agent_session_ref_id <> NEW.from_agent_session_ref_id
  OR OLD.expected_conversation_revision <> NEW.expected_conversation_revision
  OR OLD.reason <> NEW.reason OR OLD.reason_detail <> NEW.reason_detail
  OR OLD.trigger_evidence_kind <> NEW.trigger_evidence_kind
  OR OLD.trigger_evidence_ref <> NEW.trigger_evidence_ref
  OR OLD.material_change <> NEW.material_change OR OLD.changed_fields <> NEW.changed_fields
  OR OLD.context_digest <> NEW.context_digest OR OLD.context_refs <> NEW.context_refs
  OR OLD.previous_bindings <> NEW.previous_bindings
  OR OLD.replacement_bindings <> NEW.replacement_bindings OR OLD.effects_known <> NEW.effects_known
  OR OLD.request_key <> NEW.request_key OR OLD.request_fingerprint <> NEW.request_fingerprint
  OR OLD.created_at <> NEW.created_at OR NEW.updated_at < OLD.updated_at
  OR NOT ((OLD.state = 'prepared' AND NEW.state IN ('fencing', 'completed'))
       OR (OLD.state = 'fencing' AND NEW.state IN ('fenced', 'completed'))
       OR (OLD.state = 'fenced' AND NEW.state IN ('starting', 'completed'))
       OR (OLD.state = 'starting' AND NEW.state = 'completed'))
BEGIN SELECT RAISE(ABORT, 'Waldo continuation operation transition conflict'); END;
CREATE TRIGGER waldo_continuation_operation_immutable_delete BEFORE DELETE ON waldo_continuation_operations
BEGIN SELECT RAISE(ABORT, 'Waldo continuation operations are durable'); END;

CREATE TRIGGER waldo_continuation_binding BEFORE INSERT ON waldo_continuation_receipts
WHEN NOT EXISTS (
    SELECT 1 FROM waldo_conversation_episodes source
    JOIN waldo_conversations c ON c.id = source.conversation_id
    WHERE source.id = NEW.from_episode_id AND source.conversation_id = NEW.conversation_id
      AND source.project_id = NEW.project_id AND c.project_id = NEW.project_id)
  OR NOT EXISTS (
    SELECT 1 FROM waldo_continuation_operations operation
    WHERE operation.id = NEW.operation_id AND operation.conversation_id = NEW.conversation_id
      AND operation.project_id = NEW.project_id AND operation.from_episode_id = NEW.from_episode_id
      AND operation.from_agent_session_ref_id = NEW.from_agent_session_ref_id)
  OR (NEW.to_episode_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM waldo_conversation_episodes replacement
    WHERE replacement.id = NEW.to_episode_id AND replacement.conversation_id = NEW.conversation_id
      AND replacement.project_id = NEW.project_id))
BEGIN SELECT RAISE(ABORT, 'Waldo continuation binding conflict'); END;
CREATE TRIGGER waldo_continuation_immutable_update BEFORE UPDATE ON waldo_continuation_receipts
BEGIN SELECT RAISE(ABORT, 'Waldo continuation receipts are append-only'); END;
CREATE TRIGGER waldo_continuation_immutable_delete BEFORE DELETE ON waldo_continuation_receipts
BEGIN SELECT RAISE(ABORT, 'Waldo continuation receipts are append-only'); END;

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
        'waldo_conversation_continuation_recorded')),
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
DROP TRIGGER IF EXISTS waldo_conversations_cdc_insert;
DROP TRIGGER IF EXISTS waldo_conversation_episodes_cdc_insert;
DROP TRIGGER IF EXISTS waldo_conversation_episodes_cdc_update;
DROP TRIGGER IF EXISTS waldo_conversation_turns_cdc_insert;
DROP TRIGGER IF EXISTS waldo_context_attachments_cdc_insert;
DROP TRIGGER IF EXISTS waldo_context_attachments_cdc_update;
DROP TRIGGER IF EXISTS waldo_continuation_operations_cdc_insert;
DROP TRIGGER IF EXISTS waldo_continuation_operations_cdc_update;
DROP TRIGGER IF EXISTS waldo_continuation_receipts_cdc_insert;

DROP TABLE waldo_continuation_receipts;
DROP TABLE waldo_continuation_operations;
DROP TABLE waldo_turn_context_refs;
DROP TABLE waldo_context_attachments;
DROP TABLE waldo_conversation_turns;
DROP TABLE waldo_conversation_episodes;
DROP TABLE waldo_conversations;

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
        'responsibility_link_created', 'responsibility_link_ended')),
    payload TEXT NOT NULL CHECK (json_valid(payload)),
    created_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO change_log_down (seq, project_id, session_id, event_type, payload, created_at)
SELECT seq, project_id, session_id, event_type, payload, created_at
FROM change_log WHERE event_type NOT LIKE 'waldo_conversation_%';
DROP INDEX IF EXISTS idx_change_log_project;
DROP TABLE change_log;
ALTER TABLE change_log_down RENAME TO change_log;
CREATE INDEX idx_change_log_project ON change_log (project_id, seq);
-- +goose StatementEnd

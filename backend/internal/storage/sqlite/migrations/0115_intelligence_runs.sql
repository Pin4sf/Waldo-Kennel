-- Outcome-control-plane MVP: durable, non-authoritative intelligence provenance.
-- 0115 has not reached beta yet, so this feature-branch revision intentionally
-- fixes its integrity model before the migration becomes shipped history.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE intelligence_runs (
    id                   TEXT PRIMARY KEY,
    kind                 TEXT NOT NULL CHECK (kind IN ('contract_analysis','plan_draft')),
    project_id           TEXT NOT NULL REFERENCES projects (id),
    intake_id            TEXT REFERENCES intake_sessions (id),
    outcome_id           TEXT REFERENCES outcomes (id),
    contract_revision_id TEXT REFERENCES contract_revisions (id),
    source_revision      INTEGER NOT NULL CHECK (source_revision >= 0),

    requested_provider   TEXT NOT NULL DEFAULT '',
    requested_model      TEXT NOT NULL DEFAULT '',
    effective_provider   TEXT NOT NULL DEFAULT '',
    effective_model      TEXT NOT NULL DEFAULT '',
    native_session_ref   TEXT NOT NULL DEFAULT '',

    input_digest         TEXT NOT NULL
        CHECK (length(input_digest) = 64 AND input_digest NOT GLOB '*[^0-9a-f]*'),
    output_digest        TEXT NOT NULL DEFAULT ''
        CHECK (output_digest = '' OR (length(output_digest) = 64 AND output_digest NOT GLOB '*[^0-9a-f]*')),
    status               TEXT NOT NULL CHECK (status IN ('requested','running','fulfilled','failed','cancelled','expired')),
    failure_code         TEXT NOT NULL DEFAULT '',
    failure_detail       TEXT NOT NULL DEFAULT '',
    created_at           TIMESTAMP NOT NULL,
    completed_at         TIMESTAMP,

    CHECK (requested_provider <> '' OR requested_model = ''),
    CHECK (effective_provider <> '' OR (effective_model = '' AND native_session_ref = '')),
    CHECK (status <> 'fulfilled' OR output_digest <> ''),
    CHECK ((status IN ('fulfilled','failed','cancelled','expired')) = (completed_at IS NOT NULL)),
    CHECK (
        (kind = 'contract_analysis'
         AND intake_id IS NOT NULL
         AND outcome_id IS NULL
         AND contract_revision_id IS NULL)
        OR
        (kind = 'plan_draft'
         AND intake_id IS NULL
         AND outcome_id IS NOT NULL
         AND contract_revision_id IS NOT NULL
         AND source_revision >= 1)
    )
);

CREATE INDEX idx_intelligence_runs_open
    ON intelligence_runs (status, created_at, id)
    WHERE status IN ('requested','running');
CREATE INDEX idx_intelligence_runs_intake
    ON intelligence_runs (intake_id, created_at, id)
    WHERE intake_id IS NOT NULL;
CREATE INDEX idx_intelligence_runs_outcome
    ON intelligence_runs (outcome_id, created_at, id)
    WHERE outcome_id IS NOT NULL;

-- Core lineage/requested provenance is immutable. Effective provider/model and
-- provider-native reference may only move from unknown to known while the run
-- remains non-terminal; once known they cannot be replaced.
CREATE TRIGGER intelligence_runs_provenance_update_guard
BEFORE UPDATE ON intelligence_runs
WHEN OLD.id <> NEW.id
     OR OLD.kind <> NEW.kind
     OR OLD.project_id <> NEW.project_id
     OR OLD.intake_id IS NOT NEW.intake_id
     OR OLD.outcome_id IS NOT NEW.outcome_id
     OR OLD.contract_revision_id IS NOT NEW.contract_revision_id
     OR OLD.source_revision <> NEW.source_revision
     OR OLD.requested_provider <> NEW.requested_provider
     OR OLD.requested_model <> NEW.requested_model
     OR OLD.input_digest <> NEW.input_digest
     OR OLD.created_at <> NEW.created_at
     OR (OLD.effective_provider <> '' AND NEW.effective_provider <> OLD.effective_provider)
     OR (OLD.effective_model <> '' AND NEW.effective_model <> OLD.effective_model)
     OR (OLD.native_session_ref <> '' AND NEW.native_session_ref <> OLD.native_session_ref)
BEGIN
    SELECT RAISE(ABORT, 'intelligence run lineage and known provenance are immutable');
END;

-- Terminal result/provenance is frozen. Identical terminal replays are handled
-- as no-ops by the store and therefore never issue an UPDATE.
CREATE TRIGGER intelligence_runs_terminal_immutable
BEFORE UPDATE ON intelligence_runs
WHEN OLD.status IN ('fulfilled','failed','cancelled','expired')
BEGIN
    SELECT RAISE(ABORT, 'terminal intelligence runs are immutable');
END;
-- +goose StatementEnd

-- The existing request remains the durable single-use callback envelope. New
-- rows may reference exactly one canonical IntelligenceRun; historical
-- session-backed rows remain readable with a NULL run link.
-- +goose StatementBegin
ALTER TABLE intake_analysis_requests
    ADD COLUMN intelligence_run_id TEXT REFERENCES intelligence_runs (id);

CREATE UNIQUE INDEX idx_intake_analysis_requests_intelligence_run
    ON intake_analysis_requests (intelligence_run_id)
    WHERE intelligence_run_id IS NOT NULL;

DROP TRIGGER IF EXISTS intake_analysis_requests_freeze_update;
CREATE TRIGGER intake_analysis_requests_freeze_update
BEFORE UPDATE ON intake_analysis_requests
WHEN OLD.status <> 'requested'
     OR NEW.id <> OLD.id
     OR NEW.intake_id <> OLD.intake_id
     OR NEW.expected_proposal_revision <> OLD.expected_proposal_revision
     OR NEW.callback_token_digest <> OLD.callback_token_digest
     OR NEW.expires_at <> OLD.expires_at
     OR (NEW.status = 'requested' AND NOT (
            -- one-time legacy session binding during the migration window
            (OLD.session_id = '' AND OLD.harness = '' AND OLD.intelligence_run_id IS NULL
             AND NEW.session_id <> '' AND NEW.harness <> '' AND NEW.intelligence_run_id IS NULL)
            OR
            -- one-time canonical intelligence binding
            (OLD.session_id = '' AND OLD.harness = '' AND OLD.intelligence_run_id IS NULL
             AND NEW.session_id = '' AND NEW.harness = '' AND NEW.intelligence_run_id IS NOT NULL)
        ))
     OR (NEW.status <> 'requested' AND (
            NEW.session_id <> OLD.session_id
            OR NEW.harness <> OLD.harness
            OR NEW.intelligence_run_id IS NOT OLD.intelligence_run_id
        ))
BEGIN
    SELECT RAISE(ABORT, 'an intake analysis request is frozen except for one-time provenance binding and its one-way answer');
END;

CREATE TRIGGER intake_analysis_intelligence_lineage_guard
BEFORE UPDATE OF intelligence_run_id ON intake_analysis_requests
WHEN NEW.intelligence_run_id IS NOT NULL
BEGIN
    SELECT CASE
        WHEN NEW.session_id <> '' OR NEW.harness <> ''
        THEN RAISE(ABORT, 'intake analysis request cannot mix session and intelligence-run provenance')
    END;
    SELECT CASE
        WHEN NOT EXISTS (
            SELECT 1
            FROM intelligence_runs r
            JOIN intake_sessions i ON i.id = NEW.intake_id
            WHERE r.id = NEW.intelligence_run_id
              AND r.kind = 'contract_analysis'
              AND r.intake_id = NEW.intake_id
              AND r.outcome_id IS NULL
              AND r.contract_revision_id IS NULL
              AND r.project_id = i.project_id
        )
        THEN RAISE(ABORT, 'intelligence run does not match intake analysis request lineage')
    END;
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS intake_analysis_intelligence_lineage_guard;
DROP TRIGGER IF EXISTS intake_analysis_requests_freeze_update;
DROP INDEX IF EXISTS idx_intake_analysis_requests_intelligence_run;
ALTER TABLE intake_analysis_requests DROP COLUMN intelligence_run_id;
DROP TRIGGER IF EXISTS intelligence_runs_terminal_immutable;
DROP TRIGGER IF EXISTS intelligence_runs_provenance_update_guard;
DROP INDEX IF EXISTS idx_intelligence_runs_outcome;
DROP INDEX IF EXISTS idx_intelligence_runs_intake;
DROP INDEX IF EXISTS idx_intelligence_runs_open;
DROP TABLE IF EXISTS intelligence_runs;

CREATE TRIGGER intake_analysis_requests_freeze_update
BEFORE UPDATE ON intake_analysis_requests
WHEN OLD.status <> 'requested'
     OR NEW.id <> OLD.id
     OR NEW.intake_id <> OLD.intake_id
     OR NEW.expected_proposal_revision <> OLD.expected_proposal_revision
     OR NEW.callback_token_digest <> OLD.callback_token_digest
     OR NEW.expires_at <> OLD.expires_at
     OR (NEW.status = 'requested' AND NEW.session_id IS OLD.session_id)
BEGIN SELECT RAISE(ABORT, 'an intake analysis request is frozen except for its session binding and its one-way answer'); END;
-- +goose StatementEnd

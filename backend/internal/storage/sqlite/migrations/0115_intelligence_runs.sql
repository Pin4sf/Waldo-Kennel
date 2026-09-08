-- Outcome-control-plane MVP: durable, non-authoritative intelligence provenance.
--
-- Intelligence runs are proposal-producing analysis/planning facts. They are
-- deliberately separate from Attempts and execution AgentSessionRefs.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE intelligence_runs (
    id                   TEXT PRIMARY KEY,
    kind                 TEXT NOT NULL CHECK (kind IN ('contract_analysis','plan_draft')),
    project_id           TEXT NOT NULL,
    intake_id            TEXT NOT NULL DEFAULT '',
    outcome_id           TEXT NOT NULL DEFAULT '',
    contract_revision_id TEXT NOT NULL DEFAULT '',
    source_revision      INTEGER NOT NULL CHECK (source_revision >= 0),
    provider             TEXT NOT NULL DEFAULT '',
    model_selection      TEXT NOT NULL DEFAULT '' CHECK (model_selection IN ('','provider_default','explicit','unknown')),
    model                TEXT NOT NULL DEFAULT '',
    input_digest         TEXT NOT NULL,
    output_digest        TEXT NOT NULL DEFAULT '',
    native_session_ref   TEXT NOT NULL DEFAULT '',
    status               TEXT NOT NULL CHECK (status IN ('requested','running','fulfilled','failed','cancelled','expired')),
    failure_code         TEXT NOT NULL DEFAULT '',
    failure_detail       TEXT NOT NULL DEFAULT '',
    created_at           TIMESTAMP NOT NULL,
    completed_at         TIMESTAMP,
    CHECK (provider <> '' OR (model_selection = '' AND model = '' AND native_session_ref = '')),
    CHECK (model_selection <> 'explicit' OR length(trim(model)) > 0),
    CHECK (model_selection <> 'provider_default' OR model = ''),
    CHECK (model_selection <> 'unknown' OR model = ''),
    CHECK (status <> 'fulfilled' OR length(trim(output_digest)) > 0),
    CHECK (kind <> 'contract_analysis' OR intake_id <> ''),
    CHECK (kind <> 'plan_draft' OR (outcome_id <> '' AND contract_revision_id <> '' AND source_revision >= 1))
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_intelligence_runs_open
    ON intelligence_runs (status, created_at, id)
    WHERE status IN ('requested','running');
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_intelligence_runs_intake
    ON intelligence_runs (intake_id, created_at, id)
    WHERE intake_id <> '';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_intelligence_runs_outcome
    ON intelligence_runs (outcome_id, created_at, id)
    WHERE outcome_id <> '';
-- +goose StatementEnd

-- The existing request remains the single-use callback envelope. New requests
-- can point at their canonical IntelligenceRun while historical session_id /
-- harness columns stay readable during migration.
-- +goose StatementBegin
ALTER TABLE intake_analysis_requests
    ADD COLUMN intelligence_run_id TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- Recreate the frozen-row guard so an open request may bind either legacy
-- session provenance or the new IntelligenceRun exactly once. Answered rows
-- remain fully frozen.
-- +goose StatementBegin
DROP TRIGGER IF EXISTS intake_analysis_requests_freeze_update;
CREATE TRIGGER intake_analysis_requests_freeze_update
BEFORE UPDATE ON intake_analysis_requests
WHEN OLD.status <> 'requested'
     OR NEW.id <> OLD.id
     OR NEW.intake_id <> OLD.intake_id
     OR NEW.expected_proposal_revision <> OLD.expected_proposal_revision
     OR NEW.callback_token_digest <> OLD.callback_token_digest
     OR NEW.expires_at <> OLD.expires_at
     OR (OLD.session_id <> '' AND NEW.session_id <> OLD.session_id)
     OR (OLD.intelligence_run_id <> '' AND NEW.intelligence_run_id <> OLD.intelligence_run_id)
     OR (NEW.status = 'requested'
         AND NEW.session_id IS OLD.session_id
         AND NEW.intelligence_run_id IS OLD.intelligence_run_id)
BEGIN
    SELECT RAISE(ABORT, 'an intake analysis request is frozen except for one-time provenance binding and its one-way answer');
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS intake_analysis_requests_freeze_update;
ALTER TABLE intake_analysis_requests DROP COLUMN intelligence_run_id;
DROP INDEX IF EXISTS idx_intelligence_runs_outcome;
DROP INDEX IF EXISTS idx_intelligence_runs_intake;
DROP INDEX IF EXISTS idx_intelligence_runs_open;
DROP TABLE IF EXISTS intelligence_runs;

-- Restore the exact 0110 guard when rolling back.
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

-- L2 reasoning configuration is non-secret app state. The credential remains
-- in the daemon-owned local secret store; these columns only remember the
-- owner's selected provider/model/effort.

-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_settings ADD COLUMN reasoning_provider TEXT NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN reasoning_model TEXT NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN reasoning_effort TEXT NOT NULL DEFAULT '';

ALTER TABLE intelligence_runs ADD COLUMN input_tokens INTEGER;
ALTER TABLE intelligence_runs ADD COLUMN output_tokens INTEGER;
ALTER TABLE intelligence_runs ADD COLUMN duration_ms INTEGER;

CREATE TRIGGER intelligence_runs_metrics_guard
BEFORE UPDATE ON intelligence_runs
WHEN (OLD.input_tokens IS NOT NULL AND NEW.input_tokens IS NOT OLD.input_tokens)
  OR (OLD.output_tokens IS NOT NULL AND NEW.output_tokens IS NOT OLD.output_tokens)
  OR (OLD.duration_ms IS NOT NULL AND NEW.duration_ms IS NOT OLD.duration_ms)
BEGIN
    SELECT RAISE(ABORT, 'intelligence run metrics are immutable once known');
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS intelligence_runs_metrics_guard;
ALTER TABLE intelligence_runs DROP COLUMN duration_ms;
ALTER TABLE intelligence_runs DROP COLUMN output_tokens;
ALTER TABLE intelligence_runs DROP COLUMN input_tokens;
ALTER TABLE app_settings DROP COLUMN reasoning_effort;
ALTER TABLE app_settings DROP COLUMN reasoning_model;
ALTER TABLE app_settings DROP COLUMN reasoning_provider;
-- +goose StatementEnd

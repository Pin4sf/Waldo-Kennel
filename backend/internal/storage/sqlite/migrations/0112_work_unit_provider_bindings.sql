-- Provider identity is part of the immutable WorkUnit authorization boundary.
-- Keep it in a 1:1 append-only side relation so plans written before this
-- migration remain truthfully readable as provider-unbound history; no legacy
-- row is silently rewritten to Codex or to the Project's current worker.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE work_unit_provider_bindings (
    work_unit_id TEXT PRIMARY KEY REFERENCES work_units (id),
    provider     TEXT NOT NULL CHECK (length(trim(provider)) > 0),
    created_at   TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

DROP TRIGGER IF EXISTS work_unit_provider_bindings_immutable_update;
CREATE TRIGGER work_unit_provider_bindings_immutable_update
BEFORE UPDATE ON work_unit_provider_bindings
BEGIN
    SELECT RAISE(ABORT, 'work unit provider bindings are immutable');
END;

DROP TRIGGER IF EXISTS work_unit_provider_bindings_immutable_delete;
CREATE TRIGGER work_unit_provider_bindings_immutable_delete
BEFORE DELETE ON work_unit_provider_bindings
BEGIN
    SELECT RAISE(ABORT, 'work unit provider bindings are immutable');
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS work_unit_provider_bindings_immutable_delete;
DROP TRIGGER IF EXISTS work_unit_provider_bindings_immutable_update;
DROP TABLE IF EXISTS work_unit_provider_bindings;
-- +goose StatementEnd

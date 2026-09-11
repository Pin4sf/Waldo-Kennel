-- +goose Up
CREATE TABLE outcome_trash (
    outcome_id TEXT PRIMARY KEY REFERENCES outcomes(id),
    trash_root_id TEXT NOT NULL,
    erasing INTEGER NOT NULL DEFAULT 0 CHECK(erasing IN (0,1)),
    deleted_at TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down
DROP TRIGGER IF EXISTS outcome_delete_cdc;
DROP TRIGGER IF EXISTS outcome_trash_document_guard;
DROP TRIGGER IF EXISTS outcome_trash_delivery_guard;
DROP TRIGGER IF EXISTS outcome_trash_session_resume_guard;
DROP TRIGGER IF EXISTS outcome_trash_child_guard;
DROP TRIGGER IF EXISTS outcome_trash_run_guard;
DROP TRIGGER IF EXISTS outcome_trash_contract_guard;
DROP TRIGGER IF EXISTS outcome_trash_plan_guard;
DROP TRIGGER IF EXISTS outcome_trash_attempt_guard;
DROP TABLE outcome_trash;

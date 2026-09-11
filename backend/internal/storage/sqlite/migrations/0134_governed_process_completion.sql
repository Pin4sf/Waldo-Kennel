-- Persist governed execution recovery and exact supervised-process completion
-- facts on the owning session generation.

-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN governed_execution_policy_digest TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN supervisor_capability_verifier TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN supervised_process_exit_code INTEGER;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN supervised_process_exit_reason TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN supervised_process_exit_reason;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN supervised_process_exit_code;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN supervisor_capability_verifier;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN governed_execution_policy_digest;
-- +goose StatementEnd

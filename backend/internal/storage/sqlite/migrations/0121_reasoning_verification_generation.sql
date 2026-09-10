-- Bind reasoning verification to the exact persisted selection/credential
-- generation and to a non-secret fingerprint of all effective inputs.

-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_settings ADD COLUMN reasoning_generation INTEGER NOT NULL DEFAULT 0;
ALTER TABLE app_settings ADD COLUMN reasoning_verified_generation INTEGER NOT NULL DEFAULT 0;
ALTER TABLE app_settings ADD COLUMN reasoning_verification_fingerprint TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_settings DROP COLUMN reasoning_verification_fingerprint;
ALTER TABLE app_settings DROP COLUMN reasoning_verified_generation;
ALTER TABLE app_settings DROP COLUMN reasoning_generation;
-- +goose StatementEnd

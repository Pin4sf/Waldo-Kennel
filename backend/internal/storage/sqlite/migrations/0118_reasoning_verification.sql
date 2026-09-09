-- Reasoning readiness had one flag, `ready`, set purely because a credential
-- string was non-empty. A present key is not a working key: it can be revoked,
-- mistyped, or belong to a provider whose model the owner cannot access, and
-- reporting that as "ready" sends the owner into a Plan proposal that then
-- fails at the provider.
--
-- These columns record the outcome of an actual owner-triggered probe, so
-- "configured" and "verified" stay separate facts.
--
-- The verified provider and model are stored alongside the timestamp on
-- purpose: verification belongs to the exact pair that was probed. Switching
-- provider or model must not inherit an older provider's proof, which is the
-- same class of mistake as reusing another provider's credential.

-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_settings ADD COLUMN reasoning_verified_at TEXT;
ALTER TABLE app_settings ADD COLUMN reasoning_verified_provider TEXT NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN reasoning_verified_model TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_settings DROP COLUMN reasoning_verified_model;
ALTER TABLE app_settings DROP COLUMN reasoning_verified_provider;
ALTER TABLE app_settings DROP COLUMN reasoning_verified_at;
-- +goose StatementEnd

-- K0 device bridge (ADR 0018, contract v0.1): durable state for the paired
-- device principal, the outbound bridge outbox, and the replay-safe inbound
-- command journal.
--
-- These tables are bridge infrastructure, not work-state: they deliberately
-- have NO change_log triggers, so bridge state never broadcasts over SSE to
-- local clients and never becomes event authority. The CDC rail's own cursor
-- stays in the poller; the outbox seq below is the bridge's independent
-- durable cursor.
--
-- No secret material is stored: device_bridge_devices holds the public key
-- and an opaque key_custody_ref into the daemon's secret custody, never the
-- private key.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE device_bridge_devices (
    device_id          TEXT NOT NULL PRIMARY KEY,
    owner_id           TEXT NOT NULL,
    label              TEXT NOT NULL,
    public_key         TEXT NOT NULL,
    key_custody_ref    TEXT NOT NULL,
    contract_version   TEXT NOT NULL,
    capability_classes TEXT NOT NULL,
    state              TEXT NOT NULL,
    paired_at          TIMESTAMP,
    revoked_at         TIMESTAMP,
    last_seen_at       TIMESTAMP,
    created_at         TIMESTAMP NOT NULL,
    updated_at         TIMESTAMP NOT NULL
);
-- +goose StatementEnd

-- At most one paired device per owner at a time. Pairing a second device
-- while one is live conflicts at the database, not in application logic.
-- +goose StatementBegin
CREATE UNIQUE INDEX device_bridge_one_paired_per_owner
    ON device_bridge_devices(owner_id) WHERE state = 'paired';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE device_bridge_outbox (
    seq          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    message_id   TEXT NOT NULL,
    command_id   TEXT NOT NULL,
    revision     INTEGER NOT NULL,
    class        TEXT NOT NULL,
    payload      TEXT NOT NULL,
    state        TEXT NOT NULL,
    created_at   TIMESTAMP NOT NULL,
    sent_at      TIMESTAMP,
    receipted_at TIMESTAMP
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX device_bridge_outbox_message_id ON device_bridge_outbox(message_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX device_bridge_outbox_command_revision ON device_bridge_outbox(command_id, revision);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE device_bridge_inbox_journal (
    command_id          TEXT NOT NULL,
    revision            INTEGER NOT NULL,
    idempotency_key     TEXT NOT NULL,
    payload_fingerprint TEXT NOT NULL,
    class               TEXT NOT NULL,
    payload             TEXT NOT NULL,
    state               TEXT NOT NULL,
    reject_reason       TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMP NOT NULL,
    updated_at          TIMESTAMP NOT NULL,
    PRIMARY KEY (command_id, revision)
);
-- +goose StatementEnd

-- One idempotency key journals exactly one command. A redelivery with a
-- conflicting fingerprint is rejected by the store layer after read-back.
-- +goose StatementBegin
CREATE UNIQUE INDEX device_bridge_inbox_idempotency ON device_bridge_inbox_journal(idempotency_key);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX device_bridge_inbox_idempotency;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE device_bridge_inbox_journal;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX device_bridge_outbox_command_revision;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX device_bridge_outbox_message_id;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE device_bridge_outbox;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX device_bridge_one_paired_per_owner;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE device_bridge_devices;
-- +goose StatementEnd

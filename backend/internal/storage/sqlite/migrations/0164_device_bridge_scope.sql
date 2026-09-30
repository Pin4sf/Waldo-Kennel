-- Gate B: preserve legacy rows only when their device is explicitly bound.
-- No owner/device guessing, no 0162 rewrite, no parallel active tables.
-- +goose Up
-- +goose StatementBegin
CREATE TEMP TABLE bridge_scope_sequence AS SELECT seq FROM sqlite_sequence WHERE name='device_bridge_outbox';
CREATE TEMP TABLE bridge_scope_guard(ok INTEGER NOT NULL CHECK(ok=1));
INSERT INTO bridge_scope_guard SELECT CASE WHEN NOT EXISTS (
 SELECT 1 FROM device_bridge_inbox_journal j WHERE NOT EXISTS (
 SELECT 1 FROM device_bridge_devices d WHERE d.device_id=json_extract(CASE WHEN json_valid(j.payload) THEN j.payload ELSE '{}' END,'$.device_id') AND (json_extract(j.payload,'$.owner_id') IS NULL OR d.owner_id=json_extract(j.payload,'$.owner_id')))
) AND NOT EXISTS (
 SELECT 1 FROM device_bridge_outbox o WHERE NOT EXISTS (
 SELECT 1 FROM device_bridge_devices d WHERE d.device_id=json_extract(CASE WHEN json_valid(o.payload) THEN o.payload ELSE '{}' END,'$.device_id') AND (json_extract(o.payload,'$.owner_id') IS NULL OR d.owner_id=json_extract(o.payload,'$.owner_id')))
) THEN 1 ELSE 0 END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TEMP TABLE device_bridge_inbox_journal_scope_old AS SELECT * FROM device_bridge_inbox_journal;
DROP TABLE device_bridge_inbox_journal;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TABLE device_bridge_inbox_journal(device_id TEXT NOT NULL REFERENCES device_bridge_devices(device_id), command_id TEXT NOT NULL,revision INTEGER NOT NULL,idempotency_key TEXT NOT NULL,payload_fingerprint TEXT NOT NULL,class TEXT NOT NULL,payload TEXT NOT NULL,state TEXT NOT NULL,reject_reason TEXT NOT NULL DEFAULT '',created_at TIMESTAMP NOT NULL,updated_at TIMESTAMP NOT NULL, frame TEXT, expires_at INTEGER, result_frame TEXT, PRIMARY KEY(device_id,command_id,revision), UNIQUE(device_id,idempotency_key));
INSERT INTO device_bridge_inbox_journal(device_id,command_id,revision,idempotency_key,payload_fingerprint,class,payload,state,reject_reason,created_at,updated_at) SELECT json_extract(payload,'$.device_id'),command_id,revision,idempotency_key,payload_fingerprint,class,payload,state,reject_reason,created_at,updated_at FROM device_bridge_inbox_journal_scope_old;
INSERT INTO bridge_scope_guard SELECT CASE WHEN (SELECT count(*) FROM device_bridge_inbox_journal)=(SELECT count(*) FROM device_bridge_inbox_journal_scope_old) THEN 1 ELSE 0 END;
DROP TABLE device_bridge_inbox_journal_scope_old;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TEMP TABLE device_bridge_outbox_scope_old AS SELECT * FROM device_bridge_outbox;
DROP TABLE device_bridge_outbox;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TABLE device_bridge_outbox(device_id TEXT NOT NULL REFERENCES device_bridge_devices(device_id), seq INTEGER PRIMARY KEY AUTOINCREMENT,message_id TEXT NOT NULL,command_id TEXT NOT NULL,revision INTEGER NOT NULL,class TEXT NOT NULL,payload TEXT NOT NULL,state TEXT NOT NULL,created_at TIMESTAMP NOT NULL,sent_at TIMESTAMP,receipted_at TIMESTAMP, frame TEXT, idempotency_key TEXT, fingerprint TEXT, UNIQUE(device_id,message_id), UNIQUE(device_id,command_id,revision));
INSERT INTO device_bridge_outbox(device_id,seq,message_id,command_id,revision,class,payload,state,created_at,sent_at,receipted_at) SELECT json_extract(payload,'$.device_id'),seq,message_id,command_id,revision,class,payload,state,created_at,sent_at,receipted_at FROM device_bridge_outbox_scope_old;
INSERT INTO bridge_scope_guard SELECT CASE WHEN (SELECT count(*) FROM device_bridge_outbox)=(SELECT count(*) FROM device_bridge_outbox_scope_old) THEN 1 ELSE 0 END;
DROP TABLE device_bridge_outbox_scope_old;
-- +goose StatementEnd
-- +goose StatementBegin
INSERT INTO sqlite_sequence(name,seq) SELECT 'device_bridge_outbox',seq FROM bridge_scope_sequence WHERE NOT EXISTS(SELECT 1 FROM sqlite_sequence WHERE name='device_bridge_outbox');
UPDATE sqlite_sequence SET seq=MAX(seq,COALESCE((SELECT seq FROM bridge_scope_sequence),0)) WHERE name='device_bridge_outbox';
DROP TABLE bridge_scope_sequence;
CREATE TABLE device_bridge_fingerprints(device_id TEXT NOT NULL REFERENCES device_bridge_devices(device_id),message_id TEXT NOT NULL,fingerprint TEXT NOT NULL,PRIMARY KEY(device_id,message_id));
CREATE TABLE device_bridge_receipts(device_id TEXT NOT NULL REFERENCES device_bridge_devices(device_id),result_message_id TEXT NOT NULL,command_id TEXT NOT NULL,revision INTEGER NOT NULL,idempotency_key TEXT NOT NULL,result_fingerprint TEXT NOT NULL,expires_at INTEGER NOT NULL,received_at INTEGER NOT NULL,PRIMARY KEY(device_id,result_message_id));
DROP TABLE bridge_scope_guard;

-- +goose StatementEnd
-- +goose Down
-- Empty databases can roll back. Used databases retain their replay safety.
-- +goose StatementBegin
CREATE TEMP TABLE bridge_down_guard(ok INTEGER NOT NULL CHECK(ok=1));
INSERT INTO bridge_down_guard SELECT CASE WHEN
 (SELECT count(*) FROM device_bridge_inbox_journal)=0 AND
 (SELECT count(*) FROM device_bridge_outbox)=0 AND
 (SELECT count(*) FROM device_bridge_fingerprints)=0 AND
 (SELECT count(*) FROM device_bridge_receipts)=0 THEN 1 ELSE 0 END;
DROP TABLE device_bridge_receipts;
DROP TABLE device_bridge_fingerprints;
DROP TABLE device_bridge_outbox;
DROP TABLE device_bridge_inbox_journal;
DROP TABLE bridge_down_guard;
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

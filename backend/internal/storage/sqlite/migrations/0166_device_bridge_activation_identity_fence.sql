-- Connect to Waldo B1 fix: the revocation fence binds the exact revoked device
-- identity, not the owner. 0165 kept one row per owner and froze it once
-- revoked, so an owner could never pair a new device (contract v0.2.3: re-pairing
-- mints a new device identity; it never resurrects a revoked row).
--
-- Each activation becomes its own row. Revoked rows are retained, fully
-- immutable audit. At most one live (unrevoked) row per owner, and a device id
-- appears in at most one activation row ever, so a revoked identity can never be
-- paired, claimed or started again. 0165 is untouched; its rows copy verbatim.

-- +goose Up
-- +goose StatementBegin
CREATE TEMP TABLE device_bridge_activations_0166_old AS SELECT * FROM device_bridge_activations;
CREATE TEMP TABLE activation_identity_guard(ok INTEGER NOT NULL CHECK(ok=1));
DROP TABLE device_bridge_activations;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE device_bridge_activations (
    activation_id              INTEGER NOT NULL PRIMARY KEY,
    owner_id                   TEXT NOT NULL CHECK (owner_id <> ''),
    attempt                    TEXT NOT NULL DEFAULT '',
    phase                      TEXT NOT NULL CHECK (phase IN ('pending','blocked','paired','revoked')),
    checkpoint_device_id       TEXT NOT NULL DEFAULT '',
    checkpoint_key_custody_ref TEXT NOT NULL DEFAULT '',
    checkpoint_public_key      TEXT NOT NULL DEFAULT '',
    device_id                  TEXT REFERENCES device_bridge_devices(device_id),
    revoked                    INTEGER NOT NULL DEFAULT 0 CHECK (revoked IN (0,1)),
    claim_generation           INTEGER NOT NULL DEFAULT 0 CHECK (claim_generation >= 0),
    created_at                 TIMESTAMP NOT NULL,
    updated_at                 TIMESTAMP NOT NULL,
    CHECK (phase NOT IN ('pending','blocked') OR attempt <> ''),
    CHECK (phase NOT IN ('paired','revoked') OR (attempt = '' AND device_id IS NOT NULL)),
    CHECK ((phase = 'revoked') = (revoked = 1))
);
INSERT INTO device_bridge_activations(owner_id,attempt,phase,checkpoint_device_id,checkpoint_key_custody_ref,checkpoint_public_key,device_id,revoked,claim_generation,created_at,updated_at)
SELECT owner_id,attempt,phase,checkpoint_device_id,checkpoint_key_custody_ref,checkpoint_public_key,device_id,revoked,claim_generation,created_at,updated_at FROM device_bridge_activations_0166_old;
INSERT INTO activation_identity_guard SELECT CASE WHEN (SELECT count(*) FROM device_bridge_activations)=(SELECT count(*) FROM device_bridge_activations_0166_old) THEN 1 ELSE 0 END;
DROP TABLE device_bridge_activations_0166_old;
DROP TABLE activation_identity_guard;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX device_bridge_activations_one_live_per_owner
    ON device_bridge_activations(owner_id) WHERE revoked = 0;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX device_bridge_activations_device_once
    ON device_bridge_activations(device_id) WHERE device_id IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER device_bridge_activations_no_delete
BEFORE DELETE ON device_bridge_activations
BEGIN
    SELECT RAISE(ABORT, 'device bridge activations are retained for audit');
END;
-- +goose StatementEnd

-- A revoked row is never resurrected or edited; the fence generation never
-- moves back; the row key, owner and creation time are immutable.
-- +goose StatementBegin
CREATE TRIGGER device_bridge_activations_fence_monotonic
BEFORE UPDATE ON device_bridge_activations
WHEN OLD.revoked = 1
  OR NEW.claim_generation < OLD.claim_generation
  OR NEW.activation_id <> OLD.activation_id
  OR NEW.owner_id <> OLD.owner_id
  OR NEW.created_at <> OLD.created_at
BEGIN
    SELECT RAISE(ABORT, 'device bridge activation fence is monotonic');
END;
-- +goose StatementEnd

-- +goose Down
-- Empty databases can roll back. Used databases keep their reservations and fence.
-- +goose StatementBegin
CREATE TEMP TABLE activation_identity_down_guard(ok INTEGER NOT NULL CHECK(ok=1));
INSERT INTO activation_identity_down_guard SELECT CASE WHEN (SELECT count(*) FROM device_bridge_activations)=0 THEN 1 ELSE 0 END;
DROP TABLE activation_identity_down_guard;
DROP TABLE device_bridge_activations;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE device_bridge_activations (
    owner_id                   TEXT NOT NULL PRIMARY KEY CHECK (owner_id <> ''),
    attempt                    TEXT NOT NULL DEFAULT '',
    phase                      TEXT NOT NULL CHECK (phase IN ('pending','blocked','paired','revoked')),
    checkpoint_device_id       TEXT NOT NULL DEFAULT '',
    checkpoint_key_custody_ref TEXT NOT NULL DEFAULT '',
    checkpoint_public_key      TEXT NOT NULL DEFAULT '',
    device_id                  TEXT REFERENCES device_bridge_devices(device_id),
    revoked                    INTEGER NOT NULL DEFAULT 0 CHECK (revoked IN (0,1)),
    claim_generation           INTEGER NOT NULL DEFAULT 0 CHECK (claim_generation >= 0),
    created_at                 TIMESTAMP NOT NULL,
    updated_at                 TIMESTAMP NOT NULL,
    CHECK (phase NOT IN ('pending','blocked') OR attempt <> ''),
    CHECK (phase NOT IN ('paired','revoked') OR (attempt = '' AND device_id IS NOT NULL)),
    CHECK ((phase = 'revoked') = (revoked = 1))
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER device_bridge_activations_no_delete
BEFORE DELETE ON device_bridge_activations
BEGIN
    SELECT RAISE(ABORT, 'device bridge activations are retained for audit');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER device_bridge_activations_fence_monotonic
BEFORE UPDATE ON device_bridge_activations
WHEN (OLD.revoked = 1 AND (NEW.revoked <> 1 OR NEW.phase <> 'revoked' OR NEW.device_id IS NOT OLD.device_id))
  OR NEW.claim_generation < OLD.claim_generation
  OR NEW.owner_id <> OLD.owner_id
  OR NEW.created_at <> OLD.created_at
BEGIN
    SELECT RAISE(ABORT, 'device bridge activation fence is monotonic');
END;
-- +goose StatementEnd

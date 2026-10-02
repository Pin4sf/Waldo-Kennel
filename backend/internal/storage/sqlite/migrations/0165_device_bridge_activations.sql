-- Connect to Waldo B1: durable pairing reservation + revocation fence.
-- One row per owner. Holds NO pairing code and NO key bytes: only the opaque
-- reservation token, a phase, safe checkpoint fields (device id, opaque custody
-- ref, public key) and the revocation fence (revoked flag + claim generation).
-- Additive: no existing table is rewritten, 0162/0164 are untouched.

-- +goose Up
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

-- A revoked identity is never resurrected; the fence generation never moves back;
-- the owner key and creation time are immutable.
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

-- +goose Down
-- Empty databases can roll back. Used databases keep their reservations and fence.
-- +goose StatementBegin
CREATE TEMP TABLE activation_down_guard(ok INTEGER NOT NULL CHECK(ok=1));
INSERT INTO activation_down_guard SELECT CASE WHEN (SELECT count(*) FROM device_bridge_activations)=0 THEN 1 ELSE 0 END;
DROP TABLE activation_down_guard;
DROP TRIGGER device_bridge_activations_fence_monotonic;
DROP TRIGGER device_bridge_activations_no_delete;
DROP TABLE device_bridge_activations;
-- +goose StatementEnd

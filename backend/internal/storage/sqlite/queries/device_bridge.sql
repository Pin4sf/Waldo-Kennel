-- name: InsertDeviceBridgeDevice :exec
INSERT INTO device_bridge_devices (device_id,owner_id,label,public_key,key_custody_ref,contract_version,capability_classes,state,paired_at,revoked_at,last_seen_at,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?);

-- name: GetDeviceBridgeDevice :one
SELECT * FROM device_bridge_devices WHERE device_id=?;

-- name: GetPairedDeviceBridgeDevice :one
SELECT * FROM device_bridge_devices WHERE owner_id=? AND state='paired';

-- name: TransitionDeviceBridgeDeviceToPaired :execrows
UPDATE device_bridge_devices SET state='paired', paired_at=sqlc.arg(now), updated_at=sqlc.arg(now)
WHERE device_id=sqlc.arg(device_id) AND state='pairing';

-- name: TransitionDeviceBridgeDeviceToRevoked :execrows
UPDATE device_bridge_devices SET state='revoked', revoked_at=sqlc.arg(now), updated_at=sqlc.arg(now)
WHERE device_id=sqlc.arg(device_id) AND state IN ('pairing','paired');

-- name: TouchDeviceBridgeDeviceLastSeen :execrows
UPDATE device_bridge_devices SET last_seen_at=sqlc.arg(now), updated_at=sqlc.arg(now)
WHERE device_id=sqlc.arg(device_id) AND state='paired' AND (last_seen_at IS NULL OR last_seen_at < sqlc.arg(now));

-- name: EnqueueDeviceBridgeOutbox :execrows
INSERT INTO device_bridge_outbox (message_id,command_id,revision,class,payload,state,created_at)
VALUES (?,?,?,? ,?,'pending',?);

-- name: GetDeviceBridgeOutboxByMessageID :one
SELECT * FROM device_bridge_outbox WHERE message_id=?;

-- name: ListPendingDeviceBridgeOutbox :many
SELECT * FROM device_bridge_outbox WHERE state IN ('pending','sent') ORDER BY seq LIMIT ?;

-- name: MarkDeviceBridgeOutboxSent :execrows
UPDATE device_bridge_outbox SET state='sent', sent_at=sqlc.arg(now)
WHERE seq=sqlc.arg(seq) AND state='pending';

-- name: MarkDeviceBridgeOutboxReceipted :execrows
UPDATE device_bridge_outbox SET state='receipted', receipted_at=sqlc.arg(now)
WHERE seq=sqlc.arg(seq) AND state IN ('pending','sent');

-- name: CountPendingDeviceBridgeOutbox :one
SELECT count(*) FROM device_bridge_outbox WHERE state IN ('pending','sent');

-- name: InsertDeviceBridgeInbound :execrows
INSERT INTO device_bridge_inbox_journal (command_id,revision,idempotency_key,payload_fingerprint,class,payload,state,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?) ON CONFLICT DO NOTHING;

-- name: GetDeviceBridgeInboundByIdempotencyKey :one
SELECT * FROM device_bridge_inbox_journal WHERE idempotency_key=?;

-- name: TransitionDeviceBridgeInboundReceived :execrows
UPDATE device_bridge_inbox_journal SET state=sqlc.arg(to_state), reject_reason=sqlc.arg(reject_reason), updated_at=sqlc.arg(now)
WHERE command_id=sqlc.arg(command_id) AND revision=sqlc.arg(revision) AND state='received';

-- name: TransitionDeviceBridgeInboundAcked :execrows
UPDATE device_bridge_inbox_journal SET state='resulted', updated_at=sqlc.arg(now)
WHERE command_id=sqlc.arg(command_id) AND revision=sqlc.arg(revision) AND state='acked';

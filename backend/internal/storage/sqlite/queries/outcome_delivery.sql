-- Durable owner-triggered artifact delivery.

-- name: CreateOutcomeDelivery :exec
INSERT INTO outcome_deliveries (
    id, outcome_id, attempt_id, work_unit_id, artifact_version,
    disposition, destination, acceptance_decision_id, request_key,
    request_fingerprint, state, manifest_path, file_count, byte_count,
    failure_code, failure_detail, requested_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: FindOutcomeDeliveryByRequestKey :one
SELECT * FROM outcome_deliveries WHERE request_key = ?;

-- name: GetOutcomeDelivery :one
SELECT * FROM outcome_deliveries WHERE outcome_id = ? AND id = ?;

-- name: ListOutcomeDeliveries :many
SELECT * FROM outcome_deliveries WHERE outcome_id = ? ORDER BY requested_at, id;

-- name: CompleteOutcomeDelivery :execrows
UPDATE outcome_deliveries
SET state = ?, manifest_path = ?, file_count = ?, byte_count = ?,
    failure_code = ?, failure_detail = ?, completed_at = ?
WHERE id = ? AND state = 'pending';

-- name: FailPendingOutcomeDeliveries :execrows
UPDATE outcome_deliveries
SET state = 'failed', failure_code = ?, failure_detail = ?, completed_at = ?
WHERE state = 'pending';

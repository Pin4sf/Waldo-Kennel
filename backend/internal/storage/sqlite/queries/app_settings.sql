-- Daemon-owned user preferences. One row, seeded by migration 0042, so a read
-- never has to handle absence.

-- name: GetAppSettings :one
SELECT * FROM app_settings WHERE id = 1;

-- name: SetDefaultSessionMode :exec
UPDATE app_settings SET default_session_mode = ?, updated_at = ? WHERE id = 1;

-- name: SetReasoningSettings :exec
UPDATE app_settings
SET reasoning_provider = ?, reasoning_model = ?, reasoning_effort = ?, reasoning_generation = reasoning_generation + 1, updated_at = ?
WHERE id = 1;

-- name: SetReasoningVerification :exec
UPDATE app_settings
SET reasoning_verified_at = ?, reasoning_verified_provider = ?, reasoning_verified_model = ?, reasoning_verified_generation = 0, reasoning_verification_fingerprint = '', updated_at = ?
WHERE id = 1;

-- name: SetReasoningVerificationForGeneration :execrows
UPDATE app_settings
SET reasoning_verified_at = ?, reasoning_verified_provider = ?, reasoning_verified_model = ?,
    reasoning_verified_generation = CASE WHEN ? IS NULL THEN 0 ELSE ? END,
    reasoning_verification_fingerprint = CASE WHEN ? IS NULL THEN '' ELSE ? END,
    updated_at = ?
WHERE id = 1 AND reasoning_generation = ?;

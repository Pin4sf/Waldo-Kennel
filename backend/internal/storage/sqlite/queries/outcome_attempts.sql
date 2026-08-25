-- Canonical Act & Observe execution contracts (#31): attempts, session refs,
-- ordered observations, custody fences, and recovery receipts.

-- name: CreateAttempt :exec
INSERT INTO attempts (id, outcome_id, plan_revision_id, work_unit_id, number, status, contract_revision_number, request_key)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAttempt :one
SELECT id, outcome_id, plan_revision_id, work_unit_id, number, status, contract_revision_number, request_key, created_at, updated_at
FROM attempts WHERE id = ? AND outcome_id = ?;

-- name: FindAttemptByIdempotencyKey :one
SELECT id, outcome_id, plan_revision_id, work_unit_id, number, status, contract_revision_number, request_key, created_at, updated_at
FROM attempts WHERE request_key = ?;

-- name: ListAttemptsForOutcome :many
SELECT id, outcome_id, plan_revision_id, work_unit_id, number, status, contract_revision_number, request_key, created_at, updated_at
FROM attempts WHERE outcome_id = ? ORDER BY number;

-- name: MaxAttemptNumber :one
SELECT COALESCE(MAX(number), 0) FROM attempts WHERE outcome_id = ?;

-- name: TransitionAttemptStatus :execrows
UPDATE attempts SET status = ?, updated_at = ?
WHERE id = ? AND outcome_id = ? AND status = ?;

-- name: ListAttemptsByStatus :many
SELECT id, outcome_id, plan_revision_id, work_unit_id, number, status, contract_revision_number, request_key, created_at, updated_at
FROM attempts WHERE status = ? ORDER BY outcome_id, number;

-- name: CreateAttemptSessionRef :exec
INSERT INTO attempt_sessions (id, attempt_id, seq, session_id, harness, mode, run_brief_core_digest, run_brief_compiled_digest, admission_snapshot)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListAttemptSessionRefsForAttempt :many
SELECT id, attempt_id, seq, session_id, harness, mode, run_brief_core_digest, run_brief_compiled_digest, admission_snapshot, bound_at
FROM attempt_sessions WHERE attempt_id = ? ORDER BY seq;

-- name: LatestAttemptSessionRef :one
SELECT id, attempt_id, seq, session_id, harness, mode, run_brief_core_digest, run_brief_compiled_digest, admission_snapshot, bound_at
FROM attempt_sessions WHERE attempt_id = ? ORDER BY seq DESC LIMIT 1;

-- name: CreateAttemptObservation :exec
INSERT INTO attempt_observations (id, attempt_id, seq, kind, payload)
VALUES (?, ?, ?, ?, ?);

-- name: MaxAttemptObservationSeq :one
SELECT COALESCE(MAX(seq), 0) FROM attempt_observations WHERE attempt_id = ?;

-- name: ListAttemptObservationsForAttempt :many
SELECT id, attempt_id, seq, kind, payload, created_at
FROM attempt_observations WHERE attempt_id = ? ORDER BY seq;

-- name: IssueAttemptFence :exec
INSERT INTO attempt_fences (id, subject, attempt_id)
VALUES (?, ?, ?);

-- name: FindOpenFenceBySubject :one
SELECT id, subject, attempt_id, issued_at, last_renewed_at, released_at, release_reason
FROM attempt_fences WHERE subject = ? AND released_at IS NULL;

-- name: RenewAttemptFence :execrows
UPDATE attempt_fences SET last_renewed_at = ?
WHERE attempt_id = ? AND released_at IS NULL;

-- name: ReleaseAttemptFence :execrows
UPDATE attempt_fences SET released_at = ?, release_reason = ?
WHERE attempt_id = ? AND released_at IS NULL;

-- name: CreateRecoveryReceipt :exec
INSERT INTO attempt_recovery_receipts (id, attempt_id, resolution, replacement_attempt_id, detail)
VALUES (?, ?, ?, ?, ?);

-- name: ListRecoveryReceiptsForAttempt :many
SELECT id, attempt_id, resolution, replacement_attempt_id, detail, created_at
FROM attempt_recovery_receipts WHERE attempt_id = ? ORDER BY created_at, id;

-- The project backing an Outcome, resolved through its responsibility space.
-- Admission needs it to spawn the worker session on the right project.
-- name: GetOutcomeProjectID :one
SELECT s.project_id
FROM outcomes o
JOIN responsibility_spaces s ON s.id = o.space_id
WHERE o.id = ?;

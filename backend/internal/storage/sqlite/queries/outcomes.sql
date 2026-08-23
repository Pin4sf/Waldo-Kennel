-- Canonical Work responsibility contracts (#21): ResponsibilitySpace ->
-- Outcome -> immutable ContractRevision.

-- name: GetResponsibilitySpace :one
SELECT id, kind, project_id, created_at
FROM responsibility_spaces WHERE id = ?;

-- name: FindWorkResponsibilitySpaceByProject :one
SELECT id, kind, project_id, created_at
FROM responsibility_spaces WHERE project_id = ? AND kind = 'WorkProject';

-- name: CreateResponsibilitySpace :exec
INSERT INTO responsibility_spaces (id, kind, project_id)
VALUES (?, 'WorkProject', ?);

-- name: CreateOutcome :exec
INSERT INTO outcomes (id, space_id, title, current_revision_number, idempotency_key)
VALUES (?, ?, ?, ?, ?);

-- name: FindOutcomeByIdempotencyKey :one
SELECT id, space_id, title, current_revision_number, idempotency_key, created_at, updated_at
FROM outcomes WHERE idempotency_key = ?;

-- name: GetOutcome :one
SELECT id, space_id, title, current_revision_number, idempotency_key, created_at, updated_at
FROM outcomes WHERE id = ?;

-- name: AdvanceOutcomeCurrentRevision :execrows
UPDATE outcomes
SET current_revision_number = ?, updated_at = ?
WHERE id = ? AND current_revision_number = ?;

-- name: CreateContractRevision :exec
INSERT INTO contract_revisions (id, outcome_id, number, goal, success_criteria, review, constraints, non_goals, clarification)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetContractRevisionByNumber :one
SELECT id, outcome_id, number, goal, success_criteria, review, constraints, non_goals, clarification, created_at
FROM contract_revisions WHERE outcome_id = ? AND number = ?;

-- name: ListContractRevisions :many
SELECT id, outcome_id, number, goal, success_criteria, review, constraints, non_goals, clarification, created_at
FROM contract_revisions WHERE outcome_id = ? ORDER BY number;

-- name: MaxContractRevisionNumber :one
SELECT COALESCE(MAX(number), 0) FROM contract_revisions WHERE outcome_id = ?;

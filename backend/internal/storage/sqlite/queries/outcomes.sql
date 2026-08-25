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

-- name: ListOutcomesByProject :many
SELECT o.id, o.space_id, o.title, o.current_revision_number, o.idempotency_key, o.created_at, o.updated_at
FROM outcomes o
JOIN responsibility_spaces rs ON rs.id = o.space_id
WHERE rs.project_id = ? AND rs.kind = 'WorkProject'
ORDER BY o.created_at, o.id;

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

-- Canonical Decide & Authorize (#26): Outcome -> immutable PlanRevision with
-- exactly one direct WorkUnit and scoped CapabilityGrants.

-- name: CreatePlanRevision :exec
INSERT INTO plan_revisions (id, outcome_id, number, contract_revision_number, status, summary, run_brief_core_digest, run_brief_compiled_digest)
VALUES (?, ?, ?, ?, ?, ?, ?, '');

-- name: ApprovePlanRevision :execrows
UPDATE plan_revisions SET status = 'approved'
WHERE id = ? AND outcome_id = ? AND status = 'proposed';

-- name: MaxPlanRevisionNumber :one
SELECT COALESCE(MAX(number), 0) FROM plan_revisions WHERE outcome_id = ?;

-- name: LatestProposedPlanRevision :one
SELECT id, outcome_id, number, contract_revision_number, status, summary, run_brief_core_digest, run_brief_compiled_digest, created_at
FROM plan_revisions WHERE outcome_id = ? AND contract_revision_number = ? AND status = 'proposed'
ORDER BY number DESC LIMIT 1;

-- name: GetPlanRevision :one
SELECT id, outcome_id, number, contract_revision_number, status, summary, run_brief_core_digest, run_brief_compiled_digest, created_at
FROM plan_revisions WHERE id = ? AND outcome_id = ?;

-- name: GetLatestPlanRevision :one
SELECT id, outcome_id, number, contract_revision_number, status, summary, run_brief_core_digest, run_brief_compiled_digest, created_at
FROM plan_revisions WHERE outcome_id = ? ORDER BY number DESC LIMIT 1;

-- name: CreateWorkUnit :exec
INSERT INTO work_units (id, plan_revision_id, kind, title, contract_revision_number, output_summary, evidence_checks, verification_requirement, stop_conditions)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListWorkUnitsForPlan :many
SELECT id, plan_revision_id, kind, title, contract_revision_number, output_summary, evidence_checks, verification_requirement, stop_conditions
FROM work_units WHERE plan_revision_id = ?;

-- name: CreateCapabilityGrant :exec
INSERT INTO capability_grants (id, plan_revision_id, name, scope)
VALUES (?, ?, ?, ?);

-- name: ListCapabilityGrantsForPlan :many
SELECT id, plan_revision_id, name, scope
FROM capability_grants WHERE plan_revision_id = ?;

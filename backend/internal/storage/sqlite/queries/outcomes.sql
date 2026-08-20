-- name: CreateOutcome :exec
INSERT INTO outcomes (
  id, project_id, title, definition, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: CreateOutcomePlanRevision :exec
INSERT INTO outcome_plan_revisions (
  outcome_id, revision, plan_json, created_at
) VALUES (?, ?, ?, ?);

-- name: CreateOutcomeTask :exec
INSERT INTO outcome_tasks (
  id, outcome_id, title, brief, requested_harness, assigned_harness, status,
  hold_reason, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CreateOutcomeTaskDependency :exec
INSERT INTO outcome_task_dependencies (task_id, depends_on_task_id)
VALUES (?, ?);

-- name: GetOutcome :one
SELECT id, project_id, title, definition, status, approved_revision, created_at, updated_at
FROM outcomes
WHERE id = ? AND deleted_at IS NULL;

-- name: ListOutcomeTasks :many
SELECT id, outcome_id, title, brief, requested_harness, assigned_harness,
       worker_session_id, status, hold_reason, created_at, updated_at
FROM outcome_tasks
WHERE outcome_id = ?
ORDER BY rowid;

-- name: ListOutcomeTaskDependencies :many
SELECT task_id, depends_on_task_id
FROM outcome_task_dependencies
WHERE task_id IN (SELECT id FROM outcome_tasks WHERE outcome_id = ?)
ORDER BY task_id, depends_on_task_id;

-- name: GetOutcomePlanRevision :one
SELECT outcome_id, revision, plan_json, approved_at, approved_by, created_at
FROM outcome_plan_revisions
WHERE outcome_id = ? AND revision = ?;

-- name: ApproveOutcomePlanRevision :execrows
UPDATE outcome_plan_revisions
SET approved_at = ?, approved_by = ?
WHERE outcome_id = ? AND revision = ? AND approved_at IS NULL;

-- name: SetOutcomeApprovedRevision :execrows
UPDATE outcomes
SET approved_revision = ?, status = 'in_progress', updated_at = ?
WHERE id = ? AND deleted_at IS NULL AND approved_revision IS NULL;

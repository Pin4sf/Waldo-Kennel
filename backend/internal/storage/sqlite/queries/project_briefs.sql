-- name: MaxProjectBriefRevisionNumber :one
SELECT COALESCE(MAX(revision_number), 0) AS max_revision_number
FROM project_brief_revisions
WHERE project_id = ?;

-- name: CreateProjectBriefRevision :exec
INSERT INTO project_brief_revisions (
    id,
    project_id,
    revision_number,
    purpose,
    product_context,
    technical_context,
    architecture_summary,
    conventions_json,
    constraints_json,
    setup_expectations,
    run_expectations,
    test_expectations,
    provenance_json,
    created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: AdvanceProjectBriefHead :execrows
INSERT INTO project_brief_heads (
    project_id,
    current_revision_number,
    updated_at
) VALUES (
    sqlc.arg(project_id),
    sqlc.arg(current_revision_number),
    sqlc.arg(updated_at)
)
ON CONFLICT(project_id) DO UPDATE SET
    current_revision_number = excluded.current_revision_number,
    updated_at = excluded.updated_at
WHERE project_brief_heads.current_revision_number = sqlc.arg(expected_current_revision_number);

-- name: GetProjectBriefHead :one
SELECT current_revision_number
FROM project_brief_heads
WHERE project_id = ?;

-- name: GetCurrentProjectBriefRevision :one
SELECT
    r.id,
    r.project_id,
    r.revision_number,
    r.purpose,
    r.product_context,
    r.technical_context,
    r.architecture_summary,
    r.conventions_json,
    r.constraints_json,
    r.setup_expectations,
    r.run_expectations,
    r.test_expectations,
    r.provenance_json,
    r.created_at
FROM project_brief_heads h
JOIN project_brief_revisions r
  ON r.project_id = h.project_id
 AND r.revision_number = h.current_revision_number
WHERE h.project_id = ?;

-- name: ListProjectBriefRevisions :many
SELECT
    id,
    project_id,
    revision_number,
    purpose,
    product_context,
    technical_context,
    architecture_summary,
    conventions_json,
    constraints_json,
    setup_expectations,
    run_expectations,
    test_expectations,
    provenance_json,
    created_at
FROM project_brief_revisions
WHERE project_id = ?
ORDER BY revision_number ASC;

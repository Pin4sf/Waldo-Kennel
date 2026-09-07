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
--
-- The head may only move to the revision that immediately follows the one it
-- currently points at, so the optimistic check needs no separate parameter:
-- "expected" is by construction the new revision minus one. Expressing it as
-- excluded.current_revision_number - 1 also keeps the guard inside the single
-- statement, which is what makes a racing writer lose at the database instead
-- of in a read-then-compare window the caller would have to hold open.
--
-- Do NOT reintroduce sqlc.arg() inside this ON CONFLICT ... WHERE clause: sqlc
-- v1.31.1 does not substitute it there and emits the literal text into the
-- generated SQL, which SQLite then rejects at runtime.
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
WHERE project_brief_heads.current_revision_number = excluded.current_revision_number - 1;

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

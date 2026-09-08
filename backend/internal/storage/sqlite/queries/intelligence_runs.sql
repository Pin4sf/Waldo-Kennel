-- name: CreateIntelligenceRun :exec
INSERT INTO intelligence_runs (
    id, kind, project_id, intake_id, outcome_id, contract_revision_id,
    source_revision, provider, model_selection, model, input_digest, output_digest,
    native_session_ref, status, failure_code, failure_detail, created_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetIntelligenceRun :one
SELECT * FROM intelligence_runs WHERE id = ?;

-- name: ListNonTerminalIntelligenceRuns :many
SELECT * FROM intelligence_runs
WHERE status IN ('requested','running')
ORDER BY created_at, id;

-- name: UpdateIntelligenceRunStatus :execrows
UPDATE intelligence_runs
SET status = ?,
    output_digest = ?,
    failure_code = ?,
    failure_detail = ?,
    completed_at = ?
WHERE id = ? AND status = ?;

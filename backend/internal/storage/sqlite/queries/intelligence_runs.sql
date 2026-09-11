-- name: CreateIntelligenceRun :exec
INSERT INTO intelligence_runs (
    id, kind, project_id, intake_id, outcome_id, contract_revision_id,
    source_revision, requested_provider, requested_model, effective_provider,
    effective_model, native_session_ref, input_digest, output_digest, status,
    failure_code, failure_detail, created_at, completed_at, input_tokens,
    output_tokens, duration_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetIntelligenceRun :one
SELECT
    id, kind, project_id, intake_id, outcome_id, contract_revision_id,
    source_revision, requested_provider, requested_model, effective_provider,
    effective_model, native_session_ref, input_digest, output_digest, status,
    failure_code, failure_detail, created_at, completed_at, input_tokens,
    output_tokens, duration_ms
FROM intelligence_runs
WHERE id = ?;

-- name: ListNonTerminalIntelligenceRuns :many
SELECT
    id, kind, project_id, intake_id, outcome_id, contract_revision_id,
    source_revision, requested_provider, requested_model, effective_provider,
    effective_model, native_session_ref, input_digest, output_digest, status,
    failure_code, failure_detail, created_at, completed_at, input_tokens,
    output_tokens, duration_ms
FROM intelligence_runs
WHERE status IN ('requested','running')
ORDER BY created_at, id;

-- name: RecordIntelligenceRunEffectiveProvenance :execrows
UPDATE intelligence_runs
SET effective_provider = ?, effective_model = ?, native_session_ref = ?
WHERE id = ? AND status = ?
  AND effective_provider = ? AND effective_model = ? AND native_session_ref = ?;

-- name: UpdateIntelligenceRunStatus :execrows
UPDATE intelligence_runs
SET status = ?,
    output_digest = ?,
    failure_code = ?,
    failure_detail = ?,
    completed_at = ?
WHERE id = ? AND status = ?;

-- name: RecordIntelligenceRunMetrics :execrows
UPDATE intelligence_runs
SET input_tokens = ?, output_tokens = ?, duration_ms = ?
WHERE id = ?;

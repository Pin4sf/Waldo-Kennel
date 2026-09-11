-- name: BindIntakeAnalysisRequestIntelligenceRun :execrows
UPDATE intake_analysis_requests
SET intelligence_run_id = ?
WHERE id = ? AND status = 'requested' AND intelligence_run_id = '';

-- name: GetIntakeAnalysisRequestIntelligenceRun :one
SELECT intelligence_run_id
FROM intake_analysis_requests
WHERE id = ?;

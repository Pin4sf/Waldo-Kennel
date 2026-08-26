-- Shared adaptive intake and ResponsibilityLink lineage (#32).

-- name: CreateIntakeSession :exec
INSERT INTO intake_sessions
    (id, source_surface, purpose, project_id, source_open_loop_id, statement, status,
     current_proposal_revision, clarification_count, confirmed_outcome_id,
     failure_code, cancellation_reason, request_key, request_fingerprint, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: FindIntakeByRequestKey :one
SELECT * FROM intake_sessions WHERE request_key = ?;

-- name: GetIntakeSession :one
SELECT * FROM intake_sessions WHERE id = ?;

-- name: CreateIntakeConversationRef :exec
INSERT INTO intake_conversation_refs (intake_id, episode_id, turn_id, position)
VALUES (?, ?, ?, ?);

-- name: ListIntakeConversationRefs :many
SELECT intake_id, episode_id, turn_id, position
FROM intake_conversation_refs WHERE intake_id = ? ORDER BY position;

-- name: GetLatestIntakeProposal :one
SELECT * FROM intake_proposal_revisions WHERE intake_id = ? ORDER BY revision DESC LIMIT 1;

-- name: CreateIntakeProposal :exec
INSERT INTO intake_proposal_revisions
    (id, intake_id, revision, title, desired_state, criteria, review_method, constraints,
     non_goals, authority_ceiling, stop_conditions, clarification_notes, temporal_condition,
     facets, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CreateIntakeClarification :exec
INSERT INTO intake_clarifications
    (id, intake_id, question, reason, recommendation, alternatives, deferral_consequence, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetIntakeClarification :one
SELECT c.id, c.intake_id, c.question, c.reason, c.recommendation, c.alternatives,
       c.deferral_consequence, c.created_at, a.answer, a.answered_at
FROM intake_clarifications c
LEFT JOIN intake_clarification_answers a ON a.clarification_id = c.id
WHERE c.intake_id = ?;

-- name: CreateIntakeClarificationAnswer :exec
INSERT INTO intake_clarification_answers (clarification_id, answer, answered_at)
VALUES (?, ?, ?);

-- name: UpdateIntakeAnalysisState :execrows
UPDATE intake_sessions SET status = ?, updated_at = ?
WHERE id = ? AND current_proposal_revision = ? AND status = ?;

-- name: FailIntakeAnalysis :execrows
UPDATE intake_sessions SET status = 'analysis_failed', failure_code = ?, updated_at = ?
WHERE id = ? AND current_proposal_revision = ? AND status = 'analyzing';

-- name: RecoverInterruptedIntakeAnalyses :execrows
UPDATE intake_sessions
SET status = 'analysis_failed', failure_code = 'INTAKE_ANALYSIS_INTERRUPTED', updated_at = ?
WHERE status = 'analyzing';

-- name: CancelIntake :execrows
UPDATE intake_sessions
SET status = 'cancelled', cancellation_reason = ?, failure_code = '', updated_at = ?
WHERE id = ? AND current_proposal_revision = ?
  AND status IN ('captured', 'needs_user', 'ready', 'analysis_failed');

-- name: UpdateIntakeWithProposal :execrows
UPDATE intake_sessions SET status = 'ready', current_proposal_revision = ?, failure_code = '', updated_at = ?
WHERE id = ? AND current_proposal_revision = ? AND status IN ('analyzing', 'ready');

-- name: UpdateIntakeWithClarification :execrows
UPDATE intake_sessions SET status = 'needs_user', clarification_count = clarification_count + 1, updated_at = ?
WHERE id = ? AND current_proposal_revision = ? AND status = 'analyzing' AND clarification_count = 0;

-- name: ConfirmIntake :execrows
UPDATE intake_sessions SET status = 'confirmed', confirmed_outcome_id = ?, updated_at = ?
WHERE id = ? AND current_proposal_revision = ? AND status = 'ready';

-- name: CreateIntakeConfirmation :exec
INSERT INTO intake_confirmations
    (intake_id, proposal_revision, outcome_id, contract_revision_id, request_key, request_fingerprint, confirmed_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: FindIntakeConfirmationByRequestKey :one
SELECT * FROM intake_confirmations WHERE request_key = ?;

-- name: GetIntakeConfirmation :one
SELECT * FROM intake_confirmations WHERE intake_id = ?;

-- name: CreateContractRevisionIntakeCore :exec
INSERT INTO contract_revision_intake_core
    (contract_revision_id, evidence_expectations, authority_ceiling, stop_conditions,
     temporal_condition, facets, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetContractRevisionIntakeCore :one
SELECT * FROM contract_revision_intake_core WHERE contract_revision_id = ?;

-- name: CreateResponsibilityLink :exec
INSERT INTO responsibility_links
    (id, project_id, source_open_loop_id, destination_outcome_id, creator, reason,
     request_key, request_fingerprint, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: FindResponsibilityLinkByRequestKey :one
SELECT * FROM responsibility_links WHERE request_key = ?;

-- name: GetResponsibilityLink :one
SELECT * FROM responsibility_links WHERE id = ?;

-- name: FindActiveResponsibilityLinkPair :one
SELECT * FROM responsibility_links
WHERE source_open_loop_id = ? AND destination_outcome_id = ? AND ended_at IS NULL;

-- name: EndResponsibilityLink :execrows
UPDATE responsibility_links
SET ended_at = ?, ended_by = ?, ended_reason = ?
WHERE id = ? AND ended_at IS NULL;

-- name: CreateWaldoConversation :exec
INSERT INTO waldo_conversations (
    id, project_id, revision, latest_turn_sequence, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetWaldoConversationByProject :one
SELECT * FROM waldo_conversations WHERE project_id = ?;

-- name: GetWaldoConversationByID :one
SELECT * FROM waldo_conversations WHERE id = ?;

-- name: AdvanceWaldoConversationRevision :execrows
UPDATE waldo_conversations
SET revision = revision + 1, updated_at = ?
WHERE id = ? AND revision = ?;

-- name: AdvanceWaldoConversationTurn :execrows
UPDATE waldo_conversations
SET revision = revision + 1, latest_turn_sequence = ?, updated_at = ?
WHERE id = ? AND revision = ? AND latest_turn_sequence = ?;

-- name: CreateWaldoConversationEpisode :exec
INSERT INTO waldo_conversation_episodes (
    id, conversation_id, project_id, ordinal, state,
    provider, provider_conversation_id, transcript_ref,
    request_key, request_fingerprint, created_at, sealed_at, seal_reason
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: FindWaldoEpisodeByRequestKey :one
SELECT * FROM waldo_conversation_episodes WHERE request_key = ?;

-- name: ListWaldoConversationEpisodes :many
SELECT * FROM waldo_conversation_episodes
WHERE conversation_id = ? ORDER BY ordinal, id;

-- name: SealWaldoConversationEpisode :execrows
UPDATE waldo_conversation_episodes
SET state = 'sealed', sealed_at = ?, seal_reason = ?
WHERE id = ? AND conversation_id = ? AND state = 'active';

-- name: CreateWaldoConversationTurn :exec
INSERT INTO waldo_conversation_turns (
    id, conversation_id, episode_id, project_id, sequence, role, message,
    provider, provider_conversation_id, provider_turn_id, transcript_ref,
    request_key, request_fingerprint, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: FindWaldoTurnByRequestKey :one
SELECT * FROM waldo_conversation_turns WHERE request_key = ?;

-- name: ListWaldoConversationTurns :many
SELECT * FROM waldo_conversation_turns
WHERE conversation_id = ? ORDER BY sequence, id;

-- name: CreateWaldoContextAttachment :exec
INSERT INTO waldo_context_attachments (
    id, conversation_id, project_id, kind, object_id, object_revision,
    provenance_kind, provenance_ref, attached_revision,
    attach_request_key, attach_request_fingerprint, created_at,
    detached_revision, detached_at, detach_reason,
    detach_request_key, detach_request_fingerprint
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: FindWaldoContextByAttachRequestKey :one
SELECT * FROM waldo_context_attachments WHERE attach_request_key = ?;

-- name: FindWaldoContextByDetachRequestKey :one
SELECT * FROM waldo_context_attachments WHERE detach_request_key = ?;

-- name: GetWaldoContextAttachment :one
SELECT * FROM waldo_context_attachments WHERE id = ? AND conversation_id = ?;

-- name: ListWaldoContextAttachments :many
SELECT * FROM waldo_context_attachments
WHERE conversation_id = ? ORDER BY attached_revision, id;

-- name: DetachWaldoContextAttachment :execrows
UPDATE waldo_context_attachments
SET detached_revision = ?, detached_at = ?, detach_reason = ?,
    detach_request_key = ?, detach_request_fingerprint = ?
WHERE id = ? AND conversation_id = ? AND detached_at IS NULL;

-- name: CreateWaldoTurnContextRef :exec
INSERT INTO waldo_turn_context_refs (turn_id, attachment_id, position)
VALUES (?, ?, ?);

-- name: ListWaldoTurnContextRefs :many
SELECT r.turn_id, r.attachment_id, r.position,
       a.kind, a.object_id, a.object_revision, a.provenance_kind, a.provenance_ref
FROM waldo_turn_context_refs r
JOIN waldo_context_attachments a ON a.id = r.attachment_id
JOIN waldo_conversation_turns t ON t.id = r.turn_id
WHERE t.conversation_id = ?
ORDER BY t.sequence, r.position;

-- name: CreateWaldoContinuationReceipt :exec
INSERT INTO waldo_continuation_receipts (
    id, conversation_id, project_id, from_episode_id, to_episode_id,
    from_agent_session_ref_id, to_agent_session_ref_id, action, reason, reason_detail,
    material_change, changed_fields, context_digest, context_refs,
    previous_bindings, replacement_bindings, effects_known, old_session_fenced,
    replacement_identity_confirmed, fence_receipt_ref, reconciliation_ref,
    needs_user_reason, request_key, request_fingerprint, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: FindWaldoContinuationReceiptByRequestKey :one
SELECT * FROM waldo_continuation_receipts WHERE request_key = ?;

-- name: ListWaldoContinuationReceipts :many
SELECT * FROM waldo_continuation_receipts
WHERE conversation_id = ? ORDER BY created_at, id;

-- name: ResolveWaldoProjectContext :one
SELECT id AS object_id, '' AS object_revision, id AS project_id
FROM projects WHERE id = ?;

-- name: ResolveWaldoOutcomeContext :one
SELECT o.id AS object_id, CAST(o.current_revision_number AS TEXT) AS object_revision,
       s.project_id AS project_id
FROM outcomes o
JOIN responsibility_spaces s ON s.id = o.space_id
WHERE o.id = ?;

-- name: ResolveWaldoContractRevisionContext :one
SELECT cr.id AS object_id, CAST(cr.number AS TEXT) AS object_revision,
       s.project_id AS project_id
FROM contract_revisions cr
JOIN outcomes o ON o.id = cr.outcome_id
JOIN responsibility_spaces s ON s.id = o.space_id
WHERE cr.id = ?;

-- name: ResolveWaldoPlanRevisionContext :one
SELECT pr.id AS object_id, CAST(pr.number AS TEXT) AS object_revision,
       s.project_id AS project_id
FROM plan_revisions pr
JOIN outcomes o ON o.id = pr.outcome_id
JOIN responsibility_spaces s ON s.id = o.space_id
WHERE pr.id = ?;

-- name: ResolveWaldoWorkUnitContext :one
SELECT wu.id AS object_id, CAST(pr.number AS TEXT) AS object_revision,
       s.project_id AS project_id
FROM work_units wu
JOIN plan_revisions pr ON pr.id = wu.plan_revision_id
JOIN outcomes o ON o.id = pr.outcome_id
JOIN responsibility_spaces s ON s.id = o.space_id
WHERE wu.id = ?;

-- name: ResolveWaldoAttemptContext :one
SELECT a.id AS object_id, CAST(a.number AS TEXT) AS object_revision,
       s.project_id AS project_id
FROM attempts a
JOIN outcomes o ON o.id = a.outcome_id
JOIN responsibility_spaces s ON s.id = o.space_id
WHERE a.id = ?;

-- name: ResolveWaldoAgentSessionContext :one
SELECT session_ref.id AS object_id, CAST(session_ref.seq AS TEXT) AS object_revision,
       s.project_id AS project_id
FROM attempt_sessions session_ref
JOIN attempts a ON a.id = session_ref.attempt_id
JOIN outcomes o ON o.id = a.outcome_id
JOIN responsibility_spaces s ON s.id = o.space_id
WHERE session_ref.id = ?;

-- name: ResolveWaldoIntakeSessionContext :one
SELECT i.id AS object_id, CAST(i.current_proposal_revision AS TEXT) AS object_revision,
       i.project_id AS project_id
FROM intake_sessions i
WHERE i.id = ? AND i.project_id IS NOT NULL;

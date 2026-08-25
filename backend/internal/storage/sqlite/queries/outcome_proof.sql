-- Canonical Work E proof and explicit owner-decision lineage (#35).

-- name: CreateEvidenceItem :exec
INSERT INTO evidence_items (
    id, outcome_id, contract_revision_id, criterion_id, subject_type, subject_id,
    subject_revision, kind, source_type, source_ref, producer_type, producer_ref,
    summary, content_digest, request_key, request_fingerprint, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: FindEvidenceItemByRequestKey :one
SELECT id, outcome_id, contract_revision_id, criterion_id, subject_type, subject_id,
       subject_revision, kind, source_type, source_ref, producer_type, producer_ref,
       summary, content_digest, request_key, request_fingerprint, created_at
FROM evidence_items WHERE request_key = ?;

-- name: GetEvidenceItem :one
SELECT id, outcome_id, contract_revision_id, criterion_id, subject_type, subject_id,
       subject_revision, kind, source_type, source_ref, producer_type, producer_ref,
       summary, content_digest, request_key, request_fingerprint, created_at
FROM evidence_items WHERE id = ? AND outcome_id = ?;

-- name: ListEvidenceItemsForOutcome :many
SELECT id, outcome_id, contract_revision_id, criterion_id, subject_type, subject_id,
       subject_revision, kind, source_type, source_ref, producer_type, producer_ref,
       summary, content_digest, request_key, request_fingerprint, created_at
FROM evidence_items WHERE outcome_id = ? ORDER BY created_at, id;

-- name: CreateVerificationRun :exec
INSERT INTO verification_runs (
    id, outcome_id, contract_revision_id, criterion_id, subject_type, subject_id,
    subject_revision, evidence_item_ids, method, independence_class, result,
    producer_ref, verifier_ref, producer_provider, verifier_provider, detail,
    request_key, request_fingerprint, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: FindVerificationRunByRequestKey :one
SELECT id, outcome_id, contract_revision_id, criterion_id, subject_type, subject_id,
       subject_revision, evidence_item_ids, method, independence_class, result,
       producer_ref, verifier_ref, producer_provider, verifier_provider, detail,
       request_key, request_fingerprint, created_at
FROM verification_runs WHERE request_key = ?;

-- name: ListVerificationRunsForOutcome :many
SELECT id, outcome_id, contract_revision_id, criterion_id, subject_type, subject_id,
       subject_revision, evidence_item_ids, method, independence_class, result,
       producer_ref, verifier_ref, producer_provider, verifier_provider, detail,
       request_key, request_fingerprint, created_at
FROM verification_runs WHERE outcome_id = ? ORDER BY created_at, id;

-- name: CreateAcceptanceDecision :exec
INSERT INTO acceptance_decisions (
    id, outcome_id, contract_revision_id, kind, actor_type, summary,
    resource_disposition, request_key, request_fingerprint, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: FindAcceptanceDecisionByRequestKey :one
SELECT id, outcome_id, contract_revision_id, kind, actor_type, summary,
       resource_disposition, request_key, request_fingerprint, created_at
FROM acceptance_decisions WHERE request_key = ?;

-- name: ListAcceptanceDecisionsForOutcome :many
SELECT id, outcome_id, contract_revision_id, kind, actor_type, summary,
       resource_disposition, request_key, request_fingerprint, created_at
FROM acceptance_decisions WHERE outcome_id = ? ORDER BY created_at, id;

-- name: CreateOutcomeCorrection :exec
INSERT INTO outcome_corrections (
    id, decision_id, outcome_id, contract_revision_id, feedback, target_type,
    target_id, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListOutcomeCorrectionsForOutcome :many
SELECT id, decision_id, outcome_id, contract_revision_id, feedback, target_type,
       target_id, created_at
FROM outcome_corrections WHERE outcome_id = ? ORDER BY created_at, id;

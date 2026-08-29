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
INSERT INTO outcomes (id, space_id, title, current_revision_number, idempotency_key, parent_outcome_id)
VALUES (?, ?, ?, ?, ?, ?);

-- name: FindOutcomeByIdempotencyKey :one
SELECT id, space_id, title, current_revision_number, idempotency_key, created_at, updated_at, parent_outcome_id
FROM outcomes WHERE idempotency_key = ?;

-- name: GetOutcome :one
SELECT id, space_id, title, current_revision_number, idempotency_key, created_at, updated_at, parent_outcome_id
FROM outcomes WHERE id = ?;

-- name: ListOutcomesByProject :many
SELECT o.id, o.space_id, o.title, o.current_revision_number, o.idempotency_key, o.created_at, o.updated_at, o.parent_outcome_id
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

-- Stable criterion identities added by Work E (#35). The JSON text column on
-- contract_revisions remains a compatibility projection only.
-- name: CreateContractCriterion :exec
INSERT INTO contract_criteria (id, contract_revision_id, position, text)
VALUES (?, ?, ?, ?);

-- name: ListContractCriteriaForRevision :many
SELECT id, contract_revision_id, position, text
FROM contract_criteria WHERE contract_revision_id = ? ORDER BY position;

-- name: GetContractRevisionByNumber :one
SELECT id, outcome_id, number, goal, success_criteria, review, constraints, non_goals, clarification, created_at
FROM contract_revisions WHERE outcome_id = ? AND number = ?;

-- name: GetContractRevision :one
SELECT id, outcome_id, number, goal, success_criteria, review, constraints, non_goals, clarification, created_at
FROM contract_revisions WHERE id = ?;

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

-- Composed Outcomes (ADR 0007). Contribution is criterion-bound and
-- append-only; there is deliberately no update or delete query.

-- name: ListContributingOutcomes :many
SELECT id, space_id, title, current_revision_number, idempotency_key, created_at, updated_at, parent_outcome_id
FROM outcomes WHERE parent_outcome_id = ?
ORDER BY created_at, id;

-- name: CountContributingOutcomes :one
SELECT COUNT(*) FROM outcomes WHERE parent_outcome_id = ?;

-- name: CreateContributionLink :exec
INSERT INTO contribution_links (id, parent_outcome_id, child_outcome_id, parent_contract_revision_id, parent_criterion_id)
VALUES (?, ?, ?, ?, ?);

-- name: ListContributionLinksForParent :many
SELECT id, parent_outcome_id, child_outcome_id, parent_contract_revision_id, parent_criterion_id, created_at
FROM contribution_links WHERE parent_outcome_id = ?
ORDER BY created_at, id;

-- name: ListContributionLinksForChild :many
SELECT id, parent_outcome_id, child_outcome_id, parent_contract_revision_id, parent_criterion_id, created_at
FROM contribution_links WHERE child_outcome_id = ?
ORDER BY created_at, id;

-- Decomposition authority (ADR 0007 phase 2). Proposals are append-only; the
-- only permitted mutation is the one-way move to authorized.

-- name: CreateDecompositionRevision :exec
INSERT INTO decomposition_revisions (id, outcome_id, number, contract_revision_id, status, rationale, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: MaxDecompositionRevisionNumber :one
SELECT CAST(COALESCE(MAX(number), 0) AS INTEGER) FROM decomposition_revisions WHERE outcome_id = ?;

-- name: GetDecompositionRevision :one
SELECT id, outcome_id, number, contract_revision_id, status, rationale, created_at, authorized_at
FROM decomposition_revisions WHERE id = ? AND outcome_id = ?;

-- name: LatestDecompositionRevision :one
SELECT id, outcome_id, number, contract_revision_id, status, rationale, created_at, authorized_at
FROM decomposition_revisions WHERE outcome_id = ?
ORDER BY number DESC LIMIT 1;

-- name: AuthorizeDecompositionRevision :execrows
UPDATE decomposition_revisions
SET status = 'authorized', authorized_at = ?
WHERE id = ? AND outcome_id = ? AND status = 'proposed';

-- name: CreateDecompositionContribution :exec
INSERT INTO decomposition_contributions
    (id, decomposition_id, ref, position, title, goal, success_criteria, review, constraints, non_goals, authority, claimed_criteria)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListDecompositionContributions :many
SELECT id, decomposition_id, ref, position, title, goal, success_criteria, review, constraints, non_goals, authority, claimed_criteria, child_outcome_id
FROM decomposition_contributions WHERE decomposition_id = ?
ORDER BY position, ref;

-- name: BindDecompositionContributionOutcome :execrows
UPDATE decomposition_contributions
SET child_outcome_id = ?
WHERE id = ? AND child_outcome_id IS NULL;

-- name: CreateDecompositionRetainedCriterion :exec
INSERT INTO decomposition_retained_criteria (id, decomposition_id, parent_criterion_id)
VALUES (?, ?, ?);

-- name: ListDecompositionRetainedCriteria :many
SELECT id, decomposition_id, parent_criterion_id
FROM decomposition_retained_criteria WHERE decomposition_id = ?
ORDER BY parent_criterion_id;

-- name: CreateContributionDependency :exec
INSERT INTO contribution_dependencies (id, decomposition_id, from_ref, to_ref)
VALUES (?, ?, ?, ?);

-- name: ListContributionDependencies :many
SELECT id, decomposition_id, from_ref, to_ref
FROM contribution_dependencies WHERE decomposition_id = ?
ORDER BY from_ref, to_ref;

-- name: CreateContributionDependencyWaiver :exec
INSERT INTO contribution_dependency_waivers (id, decomposition_id, from_ref, to_ref, reason, waived_by, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: ListContributionDependencyWaivers :many
SELECT id, decomposition_id, from_ref, to_ref, reason, waived_by, created_at
FROM contribution_dependency_waivers WHERE decomposition_id = ?
ORDER BY created_at, id;

-- name: CreateDecompositionRequest :exec
INSERT INTO decomposition_requests (id, outcome_id, contract_revision_id, status, callback_token_digest, session_id, expires_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetDecompositionRequest :one
SELECT id, outcome_id, contract_revision_id, status, callback_token_digest, session_id, expires_at, raw_proposal, refusal_reason, decomposition_id, created_at, answered_at
FROM decomposition_requests WHERE id = ?;

-- name: LatestDecompositionRequest :one
SELECT id, outcome_id, contract_revision_id, status, callback_token_digest, session_id, expires_at, raw_proposal, refusal_reason, decomposition_id, created_at, answered_at
FROM decomposition_requests WHERE outcome_id = ?
ORDER BY created_at DESC, id DESC LIMIT 1;

-- name: AnswerDecompositionRequest :execrows
UPDATE decomposition_requests
SET status = ?, raw_proposal = ?, refusal_reason = ?, decomposition_id = ?, answered_at = ?
WHERE id = ? AND status = 'requested';

-- name: ListOpenDecompositionRequests :many
SELECT id, outcome_id, contract_revision_id, status, callback_token_digest, session_id, expires_at, raw_proposal, refusal_reason, decomposition_id, created_at, answered_at
FROM decomposition_requests WHERE status = 'requested'
ORDER BY expires_at;

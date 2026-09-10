-- Phase C: truthful terminal state and artifact continuity.
--
-- Two problems are closed here.
--
-- 1. `succeeded` was unreachable. Migration 0102's attempts_status_transition
--    trigger permits no transition into it, so no amount of service code could
--    have produced it. `reconciled` already means "execution ended, result
--    unclassified", which is exactly the honest waypoint before proof, so the
--    only missing edge is reconciled -> succeeded. 0102 is merged history and
--    is not rewritten; this recreates the trigger with that edge added.
--
--    Deliberately still absent: running -> succeeded. Success must never be
--    assigned straight from a live process. It is derived only after execution
--    has ended AND the WorkUnit's proof is satisfied.
--
-- 2. There was no durable record of what an attempt produced. Provider claims
--    are not artifacts, and a downstream WorkUnit cannot receive "whatever is
--    in some worktree".
--
-- The receipt carries everything an export manifest needs — producing lineage,
-- relative paths with content digests, context identity, base/result revision,
-- retention state and an immutable artifact version. That is on purpose: the
-- export mechanism is later work, but provenance it would need cannot be
-- backfilled into receipts that are already frozen.

-- +goose Up
-- +goose StatementBegin

-- One retained snapshot per attempt.
CREATE TABLE attempt_receipts (
    attempt_id               TEXT PRIMARY KEY REFERENCES attempts(id) ON DELETE RESTRICT,
    outcome_id               TEXT NOT NULL,
    plan_revision_id         TEXT NOT NULL,
    work_unit_id             TEXT NOT NULL,
    contract_revision_number INTEGER NOT NULL,

    -- Immutable identity of this retained set: a digest over the file manifest.
    -- An accepted-result export binds to this exact value, so a later attempt
    -- cannot silently change what was reviewed.
    artifact_version TEXT NOT NULL,

    -- Custody shape. A plain folder is NOT a worktree and must never be
    -- described as one, so the kind is recorded rather than assumed. Phase A2's
    -- supplied-document Outcomes stage into 'staged_folder'.
    workspace_kind TEXT NOT NULL CHECK (workspace_kind IN ('git_worktree', 'staged_folder')),
    workspace_path TEXT NOT NULL DEFAULT '',

    -- Context identity, for a manifest that has to say what this came from.
    repository_path     TEXT NOT NULL DEFAULT '',
    repository_identity TEXT NOT NULL DEFAULT '',

    -- Revisions are meaningful only for a git_worktree; empty otherwise rather
    -- than a fabricated value.
    base_revision   TEXT NOT NULL DEFAULT '',
    result_revision TEXT NOT NULL DEFAULT '',
    workspace_dirty INTEGER NOT NULL DEFAULT 0 CHECK (workspace_dirty IN (0, 1)),

    -- Retention truthfulness. 'incomplete' and 'unsupported' are real answers:
    -- a receipt that cannot represent what it found must say so instead of
    -- presenting a partial snapshot as the artifact.
    retention_state  TEXT NOT NULL CHECK (retention_state IN ('retained', 'incomplete', 'unsupported', 'failed')),
    retention_detail TEXT NOT NULL DEFAULT '',

    -- Why execution ended, as observed. Never parsed from provider prose.
    termination_reason TEXT NOT NULL DEFAULT '',

    observed_at TIMESTAMP NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    updated_at  TIMESTAMP NOT NULL DEFAULT (datetime('now')),

    -- Set once the receipt has been used as review evidence. A frozen receipt
    -- is never overwritten, so later work cannot silently replace what the
    -- owner reviewed.
    frozen_at TIMESTAMP
);

CREATE INDEX attempt_receipts_outcome_idx ON attempt_receipts(outcome_id, work_unit_id);

-- One row per changed path in the retained snapshot.
CREATE TABLE attempt_artifact_files (
    id            TEXT PRIMARY KEY,
    attempt_id    TEXT NOT NULL REFERENCES attempt_receipts(attempt_id) ON DELETE CASCADE,
    relative_path TEXT NOT NULL,

    -- Deletions are output too: a downstream unit that re-creates a file the
    -- upstream removed has not received the upstream's work.
    change_kind TEXT NOT NULL CHECK (change_kind IN ('added', 'modified', 'deleted', 'untracked')),

    -- Empty for a deletion, and for content the snapshot declined to read.
    content_digest TEXT NOT NULL DEFAULT '',
    size_bytes     INTEGER,
    file_mode      INTEGER,
    is_binary      INTEGER NOT NULL DEFAULT 0 CHECK (is_binary IN (0, 1)),

    -- Set when this specific path could not be represented, so an unsupported
    -- case is visible per file rather than collapsing the whole receipt.
    unsupported_reason TEXT NOT NULL DEFAULT '',

    UNIQUE (attempt_id, relative_path)
);

-- A frozen receipt is immutable apart from staying frozen. This is the schema
-- half of "later work cannot silently overwrite a reviewed artifact"; the
-- service refuses first, and this makes the invariant hold even if it does not.
CREATE TRIGGER attempt_receipts_frozen_immutable
BEFORE UPDATE ON attempt_receipts
WHEN OLD.frozen_at IS NOT NULL
     AND (OLD.artifact_version <> NEW.artifact_version
       OR OLD.retention_state <> NEW.retention_state
       OR OLD.base_revision <> NEW.base_revision
       OR OLD.result_revision <> NEW.result_revision)
BEGIN
    SELECT RAISE(ABORT, 'a frozen attempt receipt cannot be replaced');
END;

CREATE TRIGGER attempt_artifact_files_frozen_immutable
BEFORE DELETE ON attempt_artifact_files
WHEN (SELECT frozen_at FROM attempt_receipts WHERE attempt_id = OLD.attempt_id) IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'a frozen attempt receipt cannot be replaced');
END;

-- Recreate 0102's transition guard with reconciled -> succeeded permitted.
DROP TRIGGER IF EXISTS attempts_status_transition;
CREATE TRIGGER attempts_status_transition
BEFORE UPDATE ON attempts
WHEN OLD.status <> NEW.status
     AND NOT (
         (OLD.status = 'queued' AND NEW.status IN ('running', 'failed', 'cancelled', 'lost'))
      OR (OLD.status = 'running' AND NEW.status IN ('paused', 'failed', 'cancelled', 'lost', 'reconciled'))
      OR (OLD.status = 'paused' AND NEW.status IN ('running', 'cancelled', 'lost'))
      OR (OLD.status = 'reconciled' AND NEW.status = 'succeeded')
     )
BEGIN
    SELECT RAISE(ABORT, 'illegal attempt status transition');
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS attempts_status_transition;
CREATE TRIGGER attempts_status_transition
BEFORE UPDATE ON attempts
WHEN OLD.status <> NEW.status
     AND NOT (
         (OLD.status = 'queued' AND NEW.status IN ('running', 'failed', 'cancelled', 'lost'))
      OR (OLD.status = 'running' AND NEW.status IN ('paused', 'failed', 'cancelled', 'lost', 'reconciled'))
      OR (OLD.status = 'paused' AND NEW.status IN ('running', 'cancelled', 'lost'))
     )
BEGIN
    SELECT RAISE(ABORT, 'illegal attempt status transition');
END;

DROP TRIGGER IF EXISTS attempt_artifact_files_frozen_immutable;
DROP TRIGGER IF EXISTS attempt_receipts_frozen_immutable;
DROP TABLE IF EXISTS attempt_artifact_files;
DROP TABLE IF EXISTS attempt_receipts;
-- +goose StatementEnd

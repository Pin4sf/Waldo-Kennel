-- Harden Phase C receipt immutability after 0119.
--
-- A frozen receipt is review evidence. Every provenance/content field and its
-- child manifest must remain stable, while a repeated freeze remains a no-op.

-- +goose Up
-- +goose StatementBegin

DROP TRIGGER IF EXISTS attempt_receipts_frozen_immutable;
CREATE TRIGGER attempt_receipts_frozen_immutable
BEFORE UPDATE ON attempt_receipts
WHEN OLD.frozen_at IS NOT NULL AND (
       OLD.attempt_id IS NOT NEW.attempt_id
    OR OLD.outcome_id IS NOT NEW.outcome_id
    OR OLD.plan_revision_id IS NOT NEW.plan_revision_id
    OR OLD.work_unit_id IS NOT NEW.work_unit_id
    OR OLD.contract_revision_number IS NOT NEW.contract_revision_number
    OR OLD.artifact_version IS NOT NEW.artifact_version
    OR OLD.workspace_kind IS NOT NEW.workspace_kind
    OR OLD.workspace_path IS NOT NEW.workspace_path
    OR OLD.repository_path IS NOT NEW.repository_path
    OR OLD.repository_identity IS NOT NEW.repository_identity
    OR OLD.base_revision IS NOT NEW.base_revision
    OR OLD.result_revision IS NOT NEW.result_revision
    OR OLD.workspace_dirty IS NOT NEW.workspace_dirty
    OR OLD.retention_state IS NOT NEW.retention_state
    OR OLD.retention_detail IS NOT NEW.retention_detail
    OR OLD.termination_reason IS NOT NEW.termination_reason
    OR OLD.observed_at IS NOT NEW.observed_at
    OR OLD.created_at IS NOT NEW.created_at
    OR OLD.updated_at IS NOT NEW.updated_at
    OR OLD.frozen_at IS NOT NEW.frozen_at
)
BEGIN
    SELECT RAISE(ABORT, 'a frozen attempt receipt cannot be replaced');
END;

DROP TRIGGER IF EXISTS attempt_receipts_frozen_delete;
CREATE TRIGGER attempt_receipts_frozen_delete
BEFORE DELETE ON attempt_receipts
WHEN OLD.frozen_at IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'a frozen attempt receipt cannot be deleted');
END;

DROP TRIGGER IF EXISTS attempt_artifact_files_frozen_immutable;
CREATE TRIGGER attempt_artifact_files_frozen_immutable
BEFORE UPDATE ON attempt_artifact_files
WHEN (SELECT frozen_at FROM attempt_receipts WHERE attempt_id = OLD.attempt_id) IS NOT NULL
     AND (
          OLD.id IS NOT NEW.id
       OR OLD.attempt_id IS NOT NEW.attempt_id
       OR OLD.relative_path IS NOT NEW.relative_path
       OR OLD.change_kind IS NOT NEW.change_kind
       OR OLD.content_digest IS NOT NEW.content_digest
       OR OLD.size_bytes IS NOT NEW.size_bytes
       OR OLD.file_mode IS NOT NEW.file_mode
       OR OLD.is_binary IS NOT NEW.is_binary
       OR OLD.unsupported_reason IS NOT NEW.unsupported_reason
     )
BEGIN
    SELECT RAISE(ABORT, 'a frozen attempt receipt cannot be replaced');
END;

DROP TRIGGER IF EXISTS attempt_artifact_files_frozen_insert;
CREATE TRIGGER attempt_artifact_files_frozen_insert
BEFORE INSERT ON attempt_artifact_files
WHEN (SELECT frozen_at FROM attempt_receipts WHERE attempt_id = NEW.attempt_id) IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'a frozen attempt receipt cannot be replaced');
END;

DROP TRIGGER IF EXISTS attempt_artifact_files_frozen_delete;
CREATE TRIGGER attempt_artifact_files_frozen_delete
BEFORE DELETE ON attempt_artifact_files
WHEN (SELECT frozen_at FROM attempt_receipts WHERE attempt_id = OLD.attempt_id) IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'a frozen attempt receipt cannot be replaced');
END;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS attempt_artifact_files_frozen_delete;
DROP TRIGGER IF EXISTS attempt_artifact_files_frozen_insert;
DROP TRIGGER IF EXISTS attempt_artifact_files_frozen_immutable;
DROP TRIGGER IF EXISTS attempt_receipts_frozen_delete;
DROP TRIGGER IF EXISTS attempt_receipts_frozen_immutable;

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

CREATE TRIGGER attempt_artifact_files_frozen_delete
BEFORE DELETE ON attempt_artifact_files
WHEN (SELECT frozen_at FROM attempt_receipts WHERE attempt_id = OLD.attempt_id) IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'a frozen attempt receipt cannot be replaced');
END;
-- +goose StatementEnd

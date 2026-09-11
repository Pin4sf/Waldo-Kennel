-- +goose Up

-- Which local documents an Outcome is about.
--
-- Revisions are append-only: re-selecting after an edit produces a new one
-- rather than changing what was already approved, so a Plan grounded in one
-- set of bytes can never silently come to mean another. Only approved_at is
-- ever written after insert, and only once.
CREATE TABLE outcome_document_contexts (
    id         TEXT PRIMARY KEY,
    outcome_id TEXT NOT NULL REFERENCES outcomes (id),
    revision   INTEGER NOT NULL CHECK (revision >= 1),

    -- Identity of the selected bytes as a set. This is what binds a proposal
    -- to the material it was grounded in.
    digest TEXT NOT NULL,
    state  TEXT NOT NULL CHECK (state IN ('selected', 'approved')),

    selected_at TIMESTAMP NOT NULL,
    approved_at TIMESTAMP,

    UNIQUE (outcome_id, revision)
);

CREATE INDEX idx_outcome_document_contexts_current
    ON outcome_document_contexts (outcome_id, revision DESC);

-- One row per selected document, in the order the owner chose.
--
-- source_path is provenance only. Execution reads the snapshot beside the
-- retained artifacts, never the original, because a supplied document is
-- input and must not be writable output.
CREATE TABLE outcome_document_sources (
    id             TEXT PRIMARY KEY,
    context_id     TEXT NOT NULL REFERENCES outcome_document_contexts (id),
    position       INTEGER NOT NULL CHECK (position >= 0),
    source_path    TEXT NOT NULL,
    name           TEXT NOT NULL,
    content_digest TEXT NOT NULL CHECK (length(content_digest) = 64),
    size_bytes     INTEGER NOT NULL CHECK (size_bytes >= 0),

    UNIQUE (context_id, position),
    UNIQUE (context_id, name)
);

-- +goose StatementBegin
CREATE TRIGGER outcome_document_contexts_immutable_update
BEFORE UPDATE ON outcome_document_contexts
WHEN OLD.id <> NEW.id
     OR OLD.outcome_id <> NEW.outcome_id
     OR OLD.revision <> NEW.revision
     OR OLD.digest <> NEW.digest
     OR OLD.selected_at <> NEW.selected_at
     -- Approval is write-once and one-way. An approved scope that could be
     -- withdrawn or moved would not be a scope the owner approved.
     OR (OLD.state = 'approved' AND NEW.state <> 'approved')
     OR (OLD.approved_at IS NOT NULL AND OLD.approved_at IS NOT NEW.approved_at)
BEGIN
    SELECT RAISE(ABORT, 'document context revisions are immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER outcome_document_contexts_immutable_delete
BEFORE DELETE ON outcome_document_contexts
BEGIN
    SELECT RAISE(ABORT, 'document context revisions are immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER outcome_document_sources_immutable_update
BEFORE UPDATE ON outcome_document_sources
BEGIN
    SELECT RAISE(ABORT, 'document context revisions are immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER outcome_document_sources_immutable_delete
BEFORE DELETE ON outcome_document_sources
BEGIN
    SELECT RAISE(ABORT, 'document context revisions are immutable');
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS outcome_document_sources_immutable_delete;
DROP TRIGGER IF EXISTS outcome_document_sources_immutable_update;
DROP TRIGGER IF EXISTS outcome_document_contexts_immutable_delete;
DROP TRIGGER IF EXISTS outcome_document_contexts_immutable_update;
DROP INDEX IF EXISTS idx_outcome_document_contexts_current;
DROP TABLE IF EXISTS outcome_document_sources;
DROP TABLE IF EXISTS outcome_document_contexts;

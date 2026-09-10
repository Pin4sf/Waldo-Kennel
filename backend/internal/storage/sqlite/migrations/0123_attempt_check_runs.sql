-- +goose Up

-- One durable record per (Attempt, approved check, artifact version).
--
-- A deterministic check is a real process with real effects, so "did this
-- already run?" has to be a durable fact rather than something inferred from
-- whatever proof happens to exist. Reconciliation re-enumerates every ended
-- Attempt on every tick; without this a failing check would relaunch its
-- command every tick forever, and a crash between writing evidence and
-- writing its verification would relaunch it too.
--
-- The row is reserved BEFORE the command is invoked. That is the only
-- ordering under which an interrupted run is visible at all: a row left
-- reserved after a restart means the command may have run and had effects,
-- which is unknown, not "not yet run".
CREATE TABLE attempt_check_runs (
    id               TEXT PRIMARY KEY,
    attempt_id       TEXT NOT NULL REFERENCES attempts (id),
    check_id         TEXT NOT NULL,

    -- The exact retained result the run is about. A new artifact version is
    -- a different question and legitimately gets its own run.
    artifact_version TEXT NOT NULL,

    -- reserved: invocation was about to happen, outcome not yet recorded.
    -- observed: the immutable observation below is complete.
    -- unknown:  a reservation that did not complete; never auto-retried.
    state TEXT NOT NULL CHECK (state IN ('reserved', 'observed', 'unknown')),

    ran                 INTEGER NOT NULL DEFAULT 0 CHECK (ran IN (0, 1)),
    passed              INTEGER NOT NULL DEFAULT 0 CHECK (passed IN (0, 1)),
    exit_code           INTEGER NOT NULL DEFAULT 0,
    enforced_by         TEXT    NOT NULL DEFAULT '',
    timed_out           INTEGER NOT NULL DEFAULT 0 CHECK (timed_out IN (0, 1)),
    cancelled           INTEGER NOT NULL DEFAULT 0 CHECK (cancelled IN (0, 1)),
    termination_unknown INTEGER NOT NULL DEFAULT 0 CHECK (termination_unknown IN (0, 1)),
    output_truncated    INTEGER NOT NULL DEFAULT 0 CHECK (output_truncated IN (0, 1)),
    output              TEXT    NOT NULL DEFAULT '',
    unavailable         TEXT    NOT NULL DEFAULT '',

    -- Whether the run changed the very result it was checking, and what the
    -- workspace measured afterwards.
    artifact_changed          INTEGER NOT NULL DEFAULT 0 CHECK (artifact_changed IN (0, 1)),
    observed_artifact_version TEXT    NOT NULL DEFAULT '',

    reserved_at TIMESTAMP NOT NULL,
    observed_at TIMESTAMP,

    UNIQUE (attempt_id, check_id, artifact_version)
);

CREATE INDEX idx_attempt_check_runs_attempt
    ON attempt_check_runs (attempt_id, artifact_version);

-- An observation is immutable once recorded. Evidence and verification are
-- rebuilt from it after a restart, so a changed observation would silently
-- change what the proof said.
-- +goose StatementBegin
CREATE TRIGGER attempt_check_runs_observation_immutable
BEFORE UPDATE ON attempt_check_runs
WHEN OLD.state = 'observed'
BEGIN
    SELECT RAISE(ABORT, 'recorded check observations are immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER attempt_check_runs_identity_immutable
BEFORE UPDATE ON attempt_check_runs
WHEN OLD.id <> NEW.id
     OR OLD.attempt_id <> NEW.attempt_id
     OR OLD.check_id <> NEW.check_id
     OR OLD.artifact_version <> NEW.artifact_version
     OR OLD.reserved_at <> NEW.reserved_at
BEGIN
    SELECT RAISE(ABORT, 'recorded check observations are immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER attempt_check_runs_no_delete
BEFORE DELETE ON attempt_check_runs
BEGIN
    SELECT RAISE(ABORT, 'recorded check observations are immutable');
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS attempt_check_runs_no_delete;
DROP TRIGGER IF EXISTS attempt_check_runs_identity_immutable;
DROP TRIGGER IF EXISTS attempt_check_runs_observation_immutable;
DROP INDEX IF EXISTS idx_attempt_check_runs_attempt;
DROP TABLE IF EXISTS attempt_check_runs;

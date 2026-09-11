-- +goose Up

-- Deterministic checks the owner authorized when approving a Plan.
--
-- These are what Kennel runs itself and records as independent observation.
-- work_units.evidence_checks stays prose the provider reads; a sentence is not
-- a command, and a pass has to come from something that actually ran.
--
-- argv is a JSON array of discrete arguments, never a command line: a shell
-- string would make approved authority unreadable and would let one reviewed
-- entry smuggle an arbitrary program past the review.
CREATE TABLE work_unit_checks (
    id              TEXT PRIMARY KEY,
    work_unit_id    TEXT NOT NULL REFERENCES work_units (id),
    criterion_id    TEXT NOT NULL,
    position        INTEGER NOT NULL CHECK (position >= 0),
    argv            TEXT NOT NULL CHECK (json_valid(argv) AND json_array_length(argv) >= 1),
    timeout_seconds INTEGER NOT NULL CHECK (timeout_seconds > 0 AND timeout_seconds <= 900)
);

CREATE UNIQUE INDEX idx_work_unit_checks_position
    ON work_unit_checks (work_unit_id, position);

CREATE INDEX idx_work_unit_checks_criterion
    ON work_unit_checks (work_unit_id, criterion_id);

-- Approved authority is frozen. A check that could be edited or removed after
-- approval is not the check the owner approved, and an Attempt already running
-- under it would be judged against something else.
-- +goose StatementBegin
CREATE TRIGGER work_unit_checks_immutable_update
BEFORE UPDATE ON work_unit_checks
BEGIN
    SELECT RAISE(ABORT, 'approved checks are immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER work_unit_checks_immutable_delete
BEFORE DELETE ON work_unit_checks
BEGIN
    SELECT RAISE(ABORT, 'approved checks are immutable');
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS work_unit_checks_immutable_delete;
DROP TRIGGER IF EXISTS work_unit_checks_immutable_update;
DROP INDEX IF EXISTS idx_work_unit_checks_criterion;
DROP INDEX IF EXISTS idx_work_unit_checks_position;
DROP TABLE IF EXISTS work_unit_checks;

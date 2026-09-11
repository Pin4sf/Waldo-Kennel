-- +goose Up

-- What the same approved check did against a known-wrong baseline.
--
-- A zero exit proves the criterion only if a non-zero exit was possible. A
-- command that passes with none of the work present cannot be evidence that
-- the work happened, and that is the exact failure recorded against the
-- committed live planning evidence: a check that printed the expected string
-- and exited zero whatever the Attempt did.
--
-- The baseline is the same command run under the same frozen policy against a
-- pristine empty workspace. It establishes workspace dependence, which is a
-- NECESSARY condition for the check to be evidence — not a sufficient one.
-- Demonstrating that a check tests the right thing needs owner-supplied
-- fixtures (internal/planquality); this establishes only that its result could
-- have differed.
--
-- baseline_ran = 0 means no baseline was established, which is not the same as
-- a baseline that passed, and neither may support a criterion.
ALTER TABLE attempt_check_runs
    ADD COLUMN baseline_ran INTEGER NOT NULL DEFAULT 0 CHECK (baseline_ran IN (0, 1));
ALTER TABLE attempt_check_runs
    ADD COLUMN baseline_passed INTEGER NOT NULL DEFAULT 0 CHECK (baseline_passed IN (0, 1));
ALTER TABLE attempt_check_runs
    ADD COLUMN baseline_detail TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE attempt_check_runs DROP COLUMN baseline_detail;
ALTER TABLE attempt_check_runs DROP COLUMN baseline_passed;
ALTER TABLE attempt_check_runs DROP COLUMN baseline_ran;

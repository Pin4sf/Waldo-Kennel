-- WT3 + production-shaped direct Plan graph metadata.
--
-- This migration deliberately installs nothing. Its schema needs ALTER TABLE
-- against contract_revisions, plan_revisions and work_unit_provider_bindings,
-- and a burned 0099/0100 ledger entry can leave those tables physically absent
-- while their versions read as applied. ALTER TABLE cannot be made conditional
-- inside migration SQL, so the whole thing would abort and take the rest of the
-- ledger with it.
--
-- reconcileExecutionRoutingSchema in db.go installs it instead, exactly as the
-- composition and Outcome-proof schemas are installed: it checks that the
-- subject tables exist, adds the columns idempotently, then applies
-- schema/execution_routing.sql. A degraded profile defers without inventing
-- routing state and heals on the next start once the tables land.
--
-- 0112/0113 are shipped history and remain untouched.

-- +goose Up
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd

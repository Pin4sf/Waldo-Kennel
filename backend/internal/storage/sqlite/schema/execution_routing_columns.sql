-- Read by sqlc ONLY; never executed from this file.
--
-- reconcileExecutionRoutingSchema performs these ALTERs conditionally in Go,
-- because SQLite cannot guard ALTER TABLE and the columns must not be added on
-- a degraded profile where contract_revisions or plan_revisions are absent.
-- sqlc still has to know the columns exist to type the outcome and plan
-- queries, so they are declared here.
ALTER TABLE contract_revisions
    ADD COLUMN execution_preference_json TEXT
        CHECK (execution_preference_json IS NULL OR json_valid(execution_preference_json));

ALTER TABLE plan_revisions
    ADD COLUMN routing_decisions_json TEXT
        CHECK (routing_decisions_json IS NULL OR json_valid(routing_decisions_json));

ALTER TABLE work_unit_provider_bindings
    ADD COLUMN model_selection TEXT
        CHECK (model_selection IS NULL OR model_selection IN ('explicit', 'provider_default'));

ALTER TABLE work_unit_provider_bindings
    ADD COLUMN model TEXT;

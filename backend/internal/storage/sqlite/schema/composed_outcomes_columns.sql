-- Read by sqlc ONLY; never executed from this file.
--
-- reconcileComposedOutcomesSchema performs this ALTER conditionally in Go,
-- because SQLite cannot guard ALTER TABLE and the column must not be added on
-- a degraded profile where `outcomes` is absent. sqlc still has to know the
-- column exists to type the outcome queries, so it is declared here.
-- TestComposedOutcomesSchemaMatchesTheSeam asserts the two agree.
ALTER TABLE outcomes ADD COLUMN parent_outcome_id TEXT REFERENCES outcomes (id);

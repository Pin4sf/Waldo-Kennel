-- +goose Up
-- Transaction-local authority for explicit owner-requested erasure. The store
-- inserts exact row identities and clears them before committing. Normal
-- writes retain all append-only protections.
CREATE TABLE outcome_purge_scope (
    table_name TEXT NOT NULL,
    row_id INTEGER NOT NULL,
    PRIMARY KEY (table_name, row_id)
);
-- +goose Down
DROP TABLE outcome_purge_scope;

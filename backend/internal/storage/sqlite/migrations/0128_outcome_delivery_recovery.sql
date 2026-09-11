-- +goose Up

-- How a delivery's terminal result was established.
--
-- 'observed' means this daemon watched the transfer commit and then wrote the
-- row. 'recovered' means the row was left pending by a crash between the
-- filesystem commit and the ledger write, and the result was established
-- afterwards by reading the destination's manifest and re-digesting every
-- delivered artifact against the retained result.
--
-- The distinction is durable because it is not the same evidence. A recovered
-- success proves the bytes arrived; nobody watched them arrive. Historical rows
-- predate recovery and are truthfully 'observed'.
ALTER TABLE outcome_deliveries
    ADD COLUMN completion_source TEXT NOT NULL DEFAULT 'observed'
        CHECK (completion_source IN ('observed', 'recovered'));

-- Reconciliation reads one row per pending request rather than closing them
-- all with a single blanket UPDATE, because each destination has to be
-- inspected on its own before anything is concluded about it.
CREATE INDEX idx_outcome_deliveries_pending
    ON outcome_deliveries (state, requested_at, id)
    WHERE state = 'pending';

-- +goose Down
DROP INDEX IF EXISTS idx_outcome_deliveries_pending;
ALTER TABLE outcome_deliveries DROP COLUMN completion_source;

-- Composed Outcomes schema (ADR 0007), phases 1 and 2.
--
-- This file is BOTH the DDL reconcileComposedOutcomesSchema executes at
-- startup (it is go:embed-ed into db.go) AND a schema input for sqlc. Keeping
-- one copy is deliberate: these relations cannot live in migration SQL,
-- because a burned 0099 ledger entry leaves `outcomes` and `contract_criteria`
-- physically absent, and the seam must be able to defer. A second hand-written
-- copy for codegen would drift from the one that actually runs.
--
-- Every statement must stay idempotent: the seam re-runs on every start, so a
-- repaired profile heals rather than failing on a duplicate object.

CREATE INDEX IF NOT EXISTS idx_outcomes_parent
    ON outcomes (parent_outcome_id, created_at)
    WHERE parent_outcome_id IS NOT NULL;

-- One immutable binding from a contributing Outcome to one parent criterion.
-- The composite reference to contract_criteria pins the binding to the exact
-- parent revision the criterion belongs to, so a later parent revision cannot
-- silently inherit claims made against the superseded one.
CREATE TABLE IF NOT EXISTS contribution_links (
    id                          TEXT PRIMARY KEY,
    parent_outcome_id           TEXT NOT NULL REFERENCES outcomes (id),
    child_outcome_id            TEXT NOT NULL REFERENCES outcomes (id),
    parent_contract_revision_id TEXT NOT NULL REFERENCES contract_revisions (id),
    parent_criterion_id         TEXT NOT NULL,
    created_at                  TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    CHECK (parent_outcome_id <> child_outcome_id),
    FOREIGN KEY (parent_contract_revision_id, parent_criterion_id)
        REFERENCES contract_criteria (contract_revision_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contribution_links_child_criterion
    ON contribution_links (child_outcome_id, parent_criterion_id);
CREATE INDEX IF NOT EXISTS idx_contribution_links_parent
    ON contribution_links (parent_outcome_id, created_at);
CREATE INDEX IF NOT EXISTS idx_contribution_links_child
    ON contribution_links (child_outcome_id);

-- Depth cap (ADR 0007): a contributing Outcome may never itself be a parent.
-- Enforced here as well as in the domain because the cap is what makes cycles
-- impossible by construction — a graph that cannot reach depth 3 cannot loop.
DROP TRIGGER IF EXISTS outcomes_composition_depth_insert;
CREATE TRIGGER outcomes_composition_depth_insert
BEFORE INSERT ON outcomes
WHEN NEW.parent_outcome_id IS NOT NULL
     AND (SELECT parent.parent_outcome_id FROM outcomes parent WHERE parent.id = NEW.parent_outcome_id) IS NOT NULL
BEGIN SELECT RAISE(ABORT, 'composition depth limit: a contributing outcome cannot be a parent'); END;

DROP TRIGGER IF EXISTS outcomes_composition_depth_update;
CREATE TRIGGER outcomes_composition_depth_update
BEFORE UPDATE OF parent_outcome_id ON outcomes
WHEN NEW.parent_outcome_id IS NOT NULL
     AND (SELECT parent.parent_outcome_id FROM outcomes parent WHERE parent.id = NEW.parent_outcome_id) IS NOT NULL
BEGIN SELECT RAISE(ABORT, 'composition depth limit: a contributing outcome cannot be a parent'); END;

-- The same cap read from the other direction: an Outcome that already has
-- contributors cannot itself become a contributor.
DROP TRIGGER IF EXISTS outcomes_composition_parent_guard;
CREATE TRIGGER outcomes_composition_parent_guard
BEFORE UPDATE OF parent_outcome_id ON outcomes
WHEN NEW.parent_outcome_id IS NOT NULL
     AND EXISTS (SELECT 1 FROM outcomes child WHERE child.parent_outcome_id = NEW.id)
BEGIN SELECT RAISE(ABORT, 'composition depth limit: an outcome with contributors cannot become one'); END;

-- A link must describe a real contribution: the child must actually name this
-- parent, and the revision must actually belong to it.
DROP TRIGGER IF EXISTS contribution_links_binding_guard;
CREATE TRIGGER contribution_links_binding_guard
BEFORE INSERT ON contribution_links
WHEN (SELECT child.parent_outcome_id FROM outcomes child WHERE child.id = NEW.child_outcome_id)
         IS NOT NEW.parent_outcome_id
     OR NOT EXISTS (SELECT 1 FROM contract_revisions revision
                     WHERE revision.id = NEW.parent_contract_revision_id
                       AND revision.outcome_id = NEW.parent_outcome_id)
BEGIN SELECT RAISE(ABORT, 'contribution link does not describe this parent and child'); END;

-- Every link for one child names the same parent revision. A child bound
-- across two revisions would be current and superseded at once, and nothing
-- downstream could decide which.
DROP TRIGGER IF EXISTS contribution_links_single_revision_guard;
CREATE TRIGGER contribution_links_single_revision_guard
BEFORE INSERT ON contribution_links
WHEN EXISTS (SELECT 1 FROM contribution_links existing
              WHERE existing.child_outcome_id = NEW.child_outcome_id
                AND existing.parent_contract_revision_id <> NEW.parent_contract_revision_id)
BEGIN SELECT RAISE(ABORT, 'a contributing outcome binds to exactly one parent contract revision'); END;

DROP TRIGGER IF EXISTS contribution_links_immutable_update;
CREATE TRIGGER contribution_links_immutable_update
BEFORE UPDATE ON contribution_links
BEGIN SELECT RAISE(ABORT, 'contribution links are append-only'); END;

DROP TRIGGER IF EXISTS contribution_links_immutable_delete;
CREATE TRIGGER contribution_links_immutable_delete
BEFORE DELETE ON contribution_links
BEGIN SELECT RAISE(ABORT, 'contribution links are append-only'); END;

-- Phase 2 (0107): the DecompositionRevision — a decomposed Outcome's plan.
-- Proposed rows describe contributing Outcomes that do not exist yet;
-- authorization is what creates them, so a refused proposal creates nothing.
CREATE TABLE IF NOT EXISTS decomposition_revisions (
    id                   TEXT PRIMARY KEY,
    outcome_id           TEXT NOT NULL REFERENCES outcomes (id),
    number               INTEGER NOT NULL CHECK (number >= 1),
    contract_revision_id TEXT NOT NULL REFERENCES contract_revisions (id),
    status               TEXT NOT NULL CHECK (status IN ('proposed', 'authorized')),
    rationale            TEXT NOT NULL CHECK (length(trim(rationale)) > 0),
    created_at           TIMESTAMP NOT NULL DEFAULT (datetime('now')),
    authorized_at        TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_decomposition_revisions_outcome_number
    ON decomposition_revisions (outcome_id, number);

-- One contributing Outcome as proposed. child_outcome_id stays NULL until
-- authorization resolves the proposal into a real Outcome.
CREATE TABLE IF NOT EXISTS decomposition_contributions (
    id               TEXT PRIMARY KEY,
    decomposition_id TEXT NOT NULL REFERENCES decomposition_revisions (id),
    ref              TEXT NOT NULL,
    position         INTEGER NOT NULL CHECK (position >= 1),
    title            TEXT NOT NULL CHECK (length(trim(title)) > 0),
    goal             TEXT NOT NULL CHECK (length(trim(goal)) > 0),
    success_criteria TEXT NOT NULL CHECK (json_valid(success_criteria)),
    review           TEXT NOT NULL CHECK (length(trim(review)) > 0),
    constraints      TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(constraints)),
    non_goals        TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(non_goals)),
    authority        TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(authority)),
    claimed_criteria TEXT NOT NULL CHECK (json_valid(claimed_criteria)),
    child_outcome_id TEXT REFERENCES outcomes (id),
    UNIQUE (decomposition_id, ref)
);

CREATE INDEX IF NOT EXISTS idx_decomposition_contributions_decomposition
    ON decomposition_contributions (decomposition_id, position);

-- Criteria the owner keeps rather than delegating. Retention decides who
-- proves a criterion, never whether it is proved.
CREATE TABLE IF NOT EXISTS decomposition_retained_criteria (
    id                  TEXT PRIMARY KEY,
    decomposition_id    TEXT NOT NULL REFERENCES decomposition_revisions (id),
    parent_criterion_id TEXT NOT NULL,
    UNIQUE (decomposition_id, parent_criterion_id)
);

CREATE TABLE IF NOT EXISTS contribution_dependencies (
    id               TEXT PRIMARY KEY,
    decomposition_id TEXT NOT NULL REFERENCES decomposition_revisions (id),
    from_ref         TEXT NOT NULL,
    to_ref           TEXT NOT NULL,
    CHECK (from_ref <> to_ref),
    UNIQUE (decomposition_id, from_ref, to_ref)
);

-- A decomposition revision is frozen except for the one-way move to
-- authorized. Nothing may edit what the owner agreed to after they agreed.
DROP TRIGGER IF EXISTS decomposition_revisions_freeze_update;
CREATE TRIGGER decomposition_revisions_freeze_update
BEFORE UPDATE ON decomposition_revisions
WHEN OLD.status <> 'proposed'
     OR NEW.status <> 'authorized'
     OR NEW.id <> OLD.id
     OR NEW.outcome_id <> OLD.outcome_id
     OR NEW.number <> OLD.number
     OR NEW.contract_revision_id <> OLD.contract_revision_id
     OR NEW.rationale <> OLD.rationale
BEGIN SELECT RAISE(ABORT, 'a decomposition revision is frozen except for authorization'); END;

DROP TRIGGER IF EXISTS decomposition_revisions_immutable_delete;
CREATE TRIGGER decomposition_revisions_immutable_delete
BEFORE DELETE ON decomposition_revisions
BEGIN SELECT RAISE(ABORT, 'decomposition revisions are append-only'); END;

-- A proposed contribution is frozen except for gaining the Outcome that
-- authorization created for it, once.
DROP TRIGGER IF EXISTS decomposition_contributions_freeze_update;
CREATE TRIGGER decomposition_contributions_freeze_update
BEFORE UPDATE ON decomposition_contributions
WHEN OLD.child_outcome_id IS NOT NULL
     OR NEW.child_outcome_id IS NULL
     OR NEW.decomposition_id <> OLD.decomposition_id
     OR NEW.ref <> OLD.ref
     OR NEW.title <> OLD.title
     OR NEW.goal <> OLD.goal
     OR NEW.success_criteria <> OLD.success_criteria
     OR NEW.review <> OLD.review
     OR NEW.authority <> OLD.authority
     OR NEW.claimed_criteria <> OLD.claimed_criteria
BEGIN SELECT RAISE(ABORT, 'a proposed contribution is frozen except for binding its authorized outcome'); END;

DROP TRIGGER IF EXISTS decomposition_contributions_immutable_delete;
CREATE TRIGGER decomposition_contributions_immutable_delete
BEFORE DELETE ON decomposition_contributions
BEGIN SELECT RAISE(ABORT, 'decomposition contributions are append-only'); END;

DROP TRIGGER IF EXISTS decomposition_retained_immutable_update;
CREATE TRIGGER decomposition_retained_immutable_update
BEFORE UPDATE ON decomposition_retained_criteria
BEGIN SELECT RAISE(ABORT, 'retained criteria are append-only'); END;

DROP TRIGGER IF EXISTS decomposition_retained_immutable_delete;
CREATE TRIGGER decomposition_retained_immutable_delete
BEFORE DELETE ON decomposition_retained_criteria
BEGIN SELECT RAISE(ABORT, 'retained criteria are append-only'); END;

DROP TRIGGER IF EXISTS contribution_dependencies_immutable_update;
CREATE TRIGGER contribution_dependencies_immutable_update
BEFORE UPDATE ON contribution_dependencies
BEGIN SELECT RAISE(ABORT, 'contribution dependencies are append-only'); END;

DROP TRIGGER IF EXISTS contribution_dependencies_immutable_delete;
CREATE TRIGGER contribution_dependencies_immutable_delete
BEFORE DELETE ON contribution_dependencies
BEGIN SELECT RAISE(ABORT, 'contribution dependencies are append-only'); END;

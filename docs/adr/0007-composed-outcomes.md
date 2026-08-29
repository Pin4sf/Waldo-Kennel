# ADR 0007: Composed Outcomes

- Status: Accepted
- Date: 2026-08-29
- Scope: Work-lane responsibility ontology, decomposition authority, proof roll-up, and acceptance interaction
- Amends: the canonical ontology, Work Outcome lineage, Outcome state model, and Work control-plane composition of the [v0/v1 product architecture](../product/kennel-v1-product-architecture.md)
- Program plan: [Composed Outcomes](../superpowers/plans/2026-08-29-composed-outcomes-program.md)

## Context

The accepted architecture decomposes a delegated responsibility exactly once, and one level down: an `Outcome` owns a `PlanRevision`, and that plan owns `WorkUnit`s. Everything below the plan — Attempts, sessions, Evidence, Verification — hangs off that single Outcome, and a single `AcceptanceDecision` concludes it.

That shape is implemented and durable on `beta` (migrations `0099`–`0105`), but it is narrower still than the architecture allows: `PlanRevision.Validate` requires exactly one `WorkUnit` of kind `direct`, and `ProposePlan` derives that unit deterministically from the contract rather than proposing it. Kennel today supports exactly one shape — one Outcome, one contract, one work unit, one Attempt, one acceptance.

Real delegated work does not arrive in that shape. "Ship authentication" is not one bounded unit of work with one evidence obligation; it is several results that each have to become true, each with its own success criteria, its own proof, and its own risk of being wrong. Expressing that as WorkUnits inside one plan forces every part to share one contract, one authority ceiling, one staleness fate, and one accept-or-reopen decision. A single weak part reopens the whole delegated goal.

Two mechanisms could carry the missing structure: widen `PlanRevision` into a Work Unit graph with dependencies, roles, and per-unit providers; or let an Outcome be composed of contributing Outcomes. Building both would create two decomposition systems with two authority models, two staleness rules, and two places a reader must look to learn how a result is being pursued.

## Decision

### Composition

An `Outcome` may be decomposed into contributing `Outcome`s. A contributing Outcome is a full responsibility: it owns its own `ContractRevision`s, its own plan, its own Attempts, its own Evidence and Verification, and its own `AcceptanceDecision`.

The parent holds a `ContractRevision` and a `DecompositionRevision`. It holds no plan and no Attempt of its own — **its decomposition is its plan**.

An Outcome is therefore exactly one of two shapes, and never both:

- **Direct** — no children; owns `PlanRevision`s and `Attempt`s. Every Outcome that exists today is direct, and its behavior is unchanged.
- **Decomposed** — has children; owns a `DecompositionRevision`; may never start an Attempt.

### The contributing layer is the graph

`PlanRevision` stays at exactly one direct `WorkUnit`. Dependencies, ordering, and role assignment live **between** contributing Outcomes, not between Work Units inside a plan.

This is the mechanism choice, made once: composition carries the structure that a Work Unit graph would otherwise carry. Widening `PlanRevision` remains available later if a single contributing Outcome is ever *proven* to need internal branching, but it may not be introduced as a second, parallel way to express the same thing.

### Depth

Composition is capped at two levels: a Project-level Outcome and its contributing Outcomes. A contributing Outcome may not itself be decomposed. Storage carries headroom; the domain rejects deeper nesting.

The cap is a governance decision, not a schema limitation. Each additional level multiplies authority intersection, staleness cascade, coverage validation, and attention roll-up, and no evidence yet shows a third level is needed. Raising it requires evidence, not convenience.

### Contribution must be criterion-bound

Every contributing Outcome declares, in an immutable `ContributionLink`, which parent `CriterionID`s it contributes to. Without that binding, "contributing" is decorative and the parent's proof roll-up is fiction.

Every parent criterion is either claimed by at least one contributing Outcome, or explicitly marked **parent-retained** in the `DecompositionRevision`. Authorization fails closed on any criterion that is neither. Retention decides *who proves a criterion*, never *whether it is proved*: a retained criterion carries its own Evidence and Verification obligation, and the parent cannot reach Ready for Acceptance while one is unproved.

An unclassified criterion is exactly how a project would report itself done while missing something material. There is no third state.

### Authority never widens downward

A contributing Outcome's `AuthorityCeiling` must be a subset of its parent's. A child may narrow; it may never widen. This is the existing intersection rule — effective authority is the intersection of every layer, and a lower layer may only narrow — applied to a layer that did not previously exist.

### Staleness cascades

A new parent `ContractRevision` supersedes the `DecompositionRevision` bound to the prior one, and every contributing Outcome bound through it becomes **stale**. A stale child's plan cannot be approved and its Attempts cannot start until it is re-bound.

The cascade blocks *new* authorization only. A running Attempt keeps its tactical freedom and reconciles at its own fence, matching the existing lease design: a superseded parent contract is not proof that in-flight work is dead.

### Acceptance: batched interaction, unbatched authority

Contributing Outcomes reach Ready for Acceptance independently and wait. The parent's Prove & Close surface presents every ready contributor together, each with its criteria, Evidence, Verification result, and declared independence class visible and separately inspectable. One human sitting produces **N separate immutable `AcceptanceDecision` records, one per Outcome accepted** — never one decision fanned out, and never a parent decision that implies its children.

The daemon's only power over the batch is **exclusion**. A contributing Outcome whose Evidence is contradicted or missing, or whose Verification carries a weaker independence class than its own contract required, is withheld from the batch and escalated individually with the reason named. The daemon may withhold; it may not approve.

All contributors accepted makes a parent **Ready for Acceptance**, never Accepted. Reopening a parent does not reopen its children; it produces a successor or a new contributing Outcome.

No automated actor creates an `AcceptanceDecision`. This decision batches keystrokes, not authority.

## Consequences

### Benefits

- Failure is contained. A contributing Outcome can be reopened, replanned, or abandoned without disturbing the project goal or its siblings.
- Proof composes. The parent's readiness is the union of criterion-bound child proof, so a project goal never has to be re-derived from transcripts.
- The ontology matches how work is actually delegated, rather than forcing several results into one bounded unit.
- The change is additive. Every existing Outcome is the `direct` case, and no shipped behavior changes.
- One decomposition mechanism, so authority, staleness, and topology have a single reading.

### Costs

- **Acceptance multiplies**, and that directly attacks the product's primary success measure — median active supervision minutes per accepted Outcome, which must stay at least 30% below the direct-Codex baseline. Batched interaction is the answer, and it is a claim to be *measured*, not assumed. If supervision cost rises, this decision is reported as failing its gate rather than relabeled.
- Batching only pays when contributors converge in time. A contributor that finishes days early either waits or is accepted alone, and the sitting never forms.
- Coverage validation, authority intersection, and the staleness cascade are new fail-closed paths that can block work; each needs the offender named, or it becomes an unexplainable refusal.
- Attention roll-up must attribute every item to the contributing Outcome it came from, or the parent becomes a second undifferentiated activity feed.
- Migrations, DTO registry, route registration, and generated API contracts are shared surfaces; composition PRs need a named integration owner.

### Rejected alternatives

1. **Widen `PlanRevision` into a Work Unit graph.** Rejected as the primary mechanism: WorkUnits share one contract, one authority ceiling, and one acceptance, so a graph of them cannot give each part independent governance. Retained as a possible *later* refinement inside a single contributing Outcome, never as a parallel decomposition system.
2. **Build both a Work Unit graph and Outcome composition.** Rejected. Two decomposition mechanisms means two authority models, two staleness rules, and two places to look.
3. **Unbounded nesting depth.** Rejected without evidence of need. The cost is paid in every governance path, not just in storage.
4. **Delegated or policy-driven child acceptance.** Rejected. Pre-authorizing an acceptance rule and letting it fire later would let an automated actor conclude a responsibility, which no measure of convenience justifies.
5. **Per-child acceptance with no batching.** Rejected as the default because it converts one delegated result into N interruptions, attacking the supervision-cost measure the product is judged by. It remains the fallback if Phase 6 shows batching does not form in practice.
6. **Implicit parent acceptance when all children are accepted.** Rejected. A set of proved parts is not proof that the whole goal became true, and only the user accepts.

## Implementation boundary

The phased delivery, invariants, storage shape, and evaluation gate live in the [Composed Outcomes program plan](../superpowers/plans/2026-08-29-composed-outcomes-program.md).

This ADR authorizes the ontology and its implementation planning. It does not by itself authorize a code PR, merge, push, release, deployment, or a migration against a user's existing data.

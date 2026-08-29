# Composed Outcomes — program plan

- **Status:** Direction confirmed by the owner 2026-08-29. Not authorization to merge, publish, or migrate a user's data.
- **Date:** 2026-08-29
- **Base:** `origin/beta` at `38b112a`; work branch `claude/composed-outcomes`
- **Target experience:** one durable Project-level Outcome decomposed into several independently governed contributing Outcomes
- **Inputs:** [`kennel-work-experience-visual-guide.html`](../../product/kennel-work-experience-visual-guide.html), [`kennel-v1-product-architecture.md`](../../product/kennel-v1-product-architecture.md), [`work-control-plane-canonical-flow-design.md`](../specs/2026-08-25-work-control-plane-canonical-flow-design.md), [`work-experience-screen-interaction-spec.md`](../specs/2026-08-25-work-experience-screen-interaction-spec.md), [`kennel-build-program.md`](../../product/kennel-build-program.md)

---

## 1. What I understand the product to be

Kennel is a local-first Mac app whose thesis is a single sentence from the visual guide:

> Waldo Kennel lets a user supervise the truth of an Outcome while Kennel coordinates the temporary agent sessions required to make it real.

Everything else is machinery in service of that. The load-bearing consequences:

- **The Outcome is the durable responsibility; a provider session is disposable machinery underneath it.** There is no "Outcome session." Codex / OpenCode / DeepSeek are replaceable executors bound to an Attempt.
- **Truth flows one direction.** A model may *propose*; deterministic daemon policy *validates*; only the user *authorizes* and *accepts*. Provider completion, commits, PRs, and green checks are observations or Evidence candidates — never Acceptance.
- **Everything fails closed.** Missing probes, stale contracts, unproven custody, and inconclusive readiness block rather than assume. The tri-state (`authorized` / `unauthorized` / `unknown`) discipline in agent admission is the same discipline the Outcome lineage uses everywhere.
- **The measure of success is supervision cost, not throughput.** The dogfood gate demands median active supervision minutes per accepted Outcome ≥30% below direct-Codex use, full-transcript reconstruction in ≤20% of Outcomes, and ≥80% attention precision. Any design that adds user decisions has to pay for them.

### What is actually built today (verified against `beta`, not the docs)

`docs/STATUS.md` understates this — it predates several merged PRs. Reading the code:

| Layer | State on `beta` |
| --- | --- |
| `ResponsibilitySpace` → `Outcome` → `ContractRevision` | **Real and durable.** Migration `0099`; append-only revisions enforced by SQLite triggers; stable `CriterionID`s added in `0103`. |
| `PlanRevision` → `WorkUnit` → `CapabilityGrant` | **Real but deliberately narrow.** Migration `0100`. `PlanRevision.Validate` hard-requires **exactly one** WorkUnit of kind `direct`. |
| `Attempt` → `AgentSessionRef` | **Real execution.** Migration `0102`. Fail-closed admission ordered before any durable write, real provider spawn, fences, cancel with proven custody, recovery receipts. |
| `EvidenceItem` / `VerificationRun` / `AcceptanceDecision` | **Real.** Migration `0103`. Criterion-bound, independence-classed, user-only acceptance. |
| Shared adaptive intake | **Real.** Migration `0104`. |
| Project Waldo conversation | **Real.** Migration `0105`, with truthful continuation receipts. |
| Agent admission | Codex (all roles), DeepSeek Harness (worker), OpenCode (all roles, uncommitted on `beta`). |
| Mission roles | `ResolveMissionRoles` resolves analyzer / coordinator / worker / verifier per Project against live inventory. |
| Work UI | **A four-stage linear wizard.** `/work?project=&stage=&outcome=` renders Enter → Understand → Decide & Authorize → Act & Observe → Prove & Close for **one** Outcome. |

So the vertical slice genuinely works end to end for exactly one shape: **one Outcome, one contract, one hardcoded work unit, one attempt, one acceptance.**

### The three gaps between that and the intended experience

1. **No composition.** `Outcome` has `SpaceID` but no parent. There is no child Outcome, no contribution binding, no roll-up. Grepping the entire `docs/` tree for "child outcome", "sub-outcome", "parent outcome", "contributing outcome" returns nothing. **The shape in the attached diagram exists in neither the code nor the accepted architecture.** This plan's core is introducing it.
2. **No Mission.** `ProposePlan` calls `v0WorkUnit()`, which deterministically derives one unit from the contract. Nothing is proposed by a model, nothing branches, nothing has dependencies. The visual guide's small/medium/complex adaptive topologies are not expressible.
3. **No navigation hierarchy.** The canonical flow is Project Board/List → Outcome Mission Control → Session Inspector. What exists is a wizard keyed on a search param, plus the inherited AO session board at `/projects/$projectId`.

---

## 2. What the diagram actually changes

```text
Project
└── Project-level Outcome
    ├── ContractRevision
    ├── Child Outcome A → Contract + Mission
    ├── Child Outcome B → Contract + Mission
    └── Child Outcome C → Contract + Mission
```

The critical detail is what the parent **does not** have: **the parent has a Contract but no Mission of its own.** Its plan *is* the set of contributing Outcomes.

This moves decomposition up one level. Today the accepted architecture decomposes inside a PlanRevision, into WorkUnits. The diagram decomposes into **child Outcomes, each carrying its own full governance stack** — its own contract revisions, its own mission, its own attempts, its own evidence, its own verification, its own acceptance.

### Why this is the better model

- **Failure is contained.** A contributing Outcome can be reopened, replanned, or abandoned without touching the project goal or the other contributors.
- **Proof composes.** Each child proves its own criteria; the parent's proof is the union, so the parent never has to re-derive truth from transcripts.
- **It matches how the work is actually delegated.** "Ship auth" is not one bounded unit of work; it is four results that each have to become true.

### The one thing it costs, and how this plan pays for it

**Acceptance multiplies.** If four children each demand an explicit user acceptance plus a parent acceptance, one Outcome went from one decision to five. That attacks the product's own falsification gate head-on.

The resolution is to **batch the interaction without delegating the authority**:

- Children reach **Ready for Acceptance** independently and wait there.
- The parent's Prove & Close surface presents every ready child together, each with its criteria, evidence, verification, and independence class visible and inspectable.
- One human act produces **N immutable per-child `AcceptanceDecision` records plus the parent's** — real, separate, attributable decisions, made in one sitting.
- Any child whose evidence is contradicted, missing, or verified by a weaker independence class than its contract demanded is **excluded from the batch** and escalated individually.

No automated actor ever creates an `AcceptanceDecision`. The user still decides every one. We batch the *keystrokes*, never the *authority*. This must be measured in Phase 6 — if supervision minutes go up, the design is wrong and we say so.

---

## 3. Proposed ontology extension

### 3.1 Outcome shape — a mutually exclusive invariant

An Outcome is exactly one of:

- **Direct** — no children; owns a `PlanRevision` and `Attempt`s. *This is every Outcome that exists today.*
- **Decomposed** — has children; owns a `DecompositionRevision` and **may never start an Attempt**.

Enforcing mutual exclusion is what makes this a purely additive change: existing behavior is the `direct` case, untouched.

### 3.2 New durable objects

| Object | Definition and invariant |
| --- | --- |
| `Outcome.ParentID` | Nullable. Non-null makes this a contributing Outcome. Acyclic by construction under the depth cap. |
| `DecompositionRevision` | The parent's immutable analogue of `PlanRevision`: the authorized set of contributing Outcomes, their contribution bindings, and their dependencies, frozen and bound to one parent `ContractRevision`. Append-only. |
| `ContributionLink` | Immutable binding from one child Outcome to one or more parent `CriterionID`s. This is what makes "contributing" mean something instead of being decorative — without it, roll-up is fiction. |
| `ContributionDependency` | Declared ordering between two sibling children within one `DecompositionRevision`. Cycles rejected at authorization. |

### 3.3 Invariants the daemon must enforce, all fail-closed

1. **Depth cap of 1.** A child may not itself be decomposed. Schema allows deeper; the domain rejects it. Deeper nesting multiplies authority intersection and roll-up ambiguity with no proven need — revisit only with evidence.
2. **Criterion coverage.** Every parent criterion is either claimed by at least one child or explicitly marked `parent-retained` in the `DecompositionRevision`. Authorization fails on anything else — an unclassified criterion is exactly how a project reports "done" while missing something material. A retained criterion still needs its own evidence and verification before the parent is Ready; retaining is a choice about *who proves it*, never an exemption from proof.
3. **Authority containment.** Child `AuthorityCeiling` ⊆ parent `AuthorityCeiling`. `ProposedAuthority` is a flat bool struct, so this is a boolean-AND intersection with a named offender on violation. A child may narrow; it may never widen.
4. **Staleness cascade.** A new parent `ContractRevision` supersedes the `DecompositionRevision` bound to the prior one. Children bound to the superseded revision are **stale**: their plans cannot be approved and their attempts cannot start until re-bound. This mirrors the existing plan-staleness rule exactly.
5. **No parent attempts.** A decomposed Outcome cannot spawn a session. Its execution is entirely its children's.
6. **Dependency gating.** A child's first Attempt cannot start while a declared upstream sibling is unaccepted, unless the user explicitly records a waiver — which is itself a durable, attributable decision.
7. **Parent acceptance is not implied.** All children accepted makes the parent **Ready for Acceptance**, never Accepted. Reopening a parent does not auto-reopen children; it creates a successor or a new contributing Outcome.

### 3.4 What the Mission becomes

**Recommendation: do not widen `PlanRevision` to N WorkUnits.** Let the child-Outcome layer be the graph.

The visual guide draws the Mission as a graph of Work Units. With composition, that graph moves up: the **parent's decomposition is the topology**, and each child keeps the shape that already works — one contract, one plan, one direct WorkUnit, one Attempt. Dependencies live between children, not between work units.

This is dramatically less to build, and it delivers the diagram exactly as drawn. Widening `PlanRevision` remains available later if a single contributing Outcome is ever proven to need internal branching — but building both graphs now would be two overlapping decomposition mechanisms with two sets of authority and staleness rules.

---

## 4. Implementation phases

Each phase is one issue and one PR, branched from latest `origin/beta`, PR'd back to `beta`. Per `AGENTS.md`, **every PR owns its full vertical** — domain, storage, CDC, service, API, generated contracts, UI, recovery, and tests. No PR may leave a schema or a screen for a later PR to make true. Next free migration number is **0106**; each phase claims its number in its issue before editing schema.

### Phase 0 — Record the decision (docs only)

The product architecture doc is the contract, and it currently says decomposition happens at the WorkUnit level. Shipping nested Outcomes without amending it puts code and contract in conflict.

- New **ADR 0007 — Composed Outcomes**: the shape, the mutual-exclusion invariant, the depth cap, the batched-acceptance resolution and its risk.
- Amend `kennel-v1-product-architecture.md`: canonical ontology, the Work Outcome lineage, the Outcome state model, and the Work control-plane composition.
- Amend `kennel-build-program.md` shared-ownership table: `DecompositionRevision` and `ContributionLink` are shared surfaces needing a named integration owner.
- Correct `docs/STATUS.md`, which understates what has merged.

**Gate:** the written architecture and the intended code agree before any code is written.

### Phase 1 — Composition domain, storage, and read API — **delivered 2026-08-29**

**Scope adjusted during implementation.** `DecompositionRevision` and `ContributionDependency` moved to Phase 2, where their writer lives. Shipping their tables here would have been exactly the horizontal schema the repository forbids — a relation no code can write, waiting for a later PR to make it true. Phase 1 delivers composition end to end without them; Phase 2 adds the frozen, authorized decomposition on top.

Delivered:

- `Outcome.ParentID`, derived `OutcomeShape`, `ContributionLink`, authority containment, criterion coverage, and staleness in `internal/domain`.
- Migration `0106` extends the `change_log` vocabulary with `outcome_contribution_bound`. The composition schema itself installs through `reconcileComposedOutcomesSchema`, a startup seam matching `reconcileOutcomeProofSchema`: a burned `0099` ledger entry leaves `outcomes` physically absent while its version is marked, and `ALTER TABLE` cannot be made conditional in goose SQL.
- `POST /api/v1/outcomes/{outcomeId}/contributions` and `GET /api/v1/outcomes/{outcomeId}/composition`; `parentId` added to every Outcome projection. `openapi.yaml` and `schema.ts` regenerated.
- **Tests** at all three layers: depth cap (domain, service, and raw SQL, in both directions), authority containment naming every over-claimed authority, unknown/blank/duplicate criterion claims, mismatched bindings, link immutability against a writer that bypasses the store, split-revision bindings, CDC emission, replay idempotency, parent-revision staleness, degraded-profile deferral and seam idempotency, and that an Outcome with no parent behaves exactly as it does today.

**Gate met:** a decomposed Outcome round-trips through restart; direct Outcomes are unchanged; the full backend suite and `-race` pass except one pre-existing DeepSeek failure; lint adds no new issues.

### Phase 2 — Decomposition proposal and authorization — **delivered 2026-08-29**

Owns `DecompositionRevision` and `ContributionDependency`, inherited from Phase 1 along with parent-retained criteria — all three exist to be *authorized*, so they land with the flow that authorizes them.

Delivered:

- `DecompositionRevision` (proposed → authorized), `ProposedContribution`, `ContributionDependency`, cycle detection, and the coverage/containment predicates in `internal/domain`.
- Migration `0107` extends the `change_log` vocabulary; the relations install through the shared `reconcileComposedOutcomesSchema` seam, with freeze triggers permitting only the one-way move to authorized.
- `POST /outcomes/{id}/decompositions`, `POST /outcomes/{id}/decompositions/{decompositionId}/authorization`, `GET /outcomes/{id}/decomposition`.
- Omitting contributors yields the daemon's deterministic default: one contributing Outcome per criterion, mechanical rather than clever, labelled as a starting point rather than a recommendation.
- **Tests** at domain, service, and SQLite layers, including transactional rollback of a partially-failing authorization and freeze enforcement against a writer that bypasses the store.

**Scope changes made during implementation, both recorded here rather than in a commit message alone:**

1. **The ad-hoc `POST /outcomes/{id}/contributions` route from Phase 1 is retired.** A contributing Outcome is now born only by authorizing a decomposition. Leaving both paths open would have been a governance hole: the ad-hoc route bypasses coverage, containment, and ordering — exactly the deterministic validation ADR 0007 requires before the owner sees a proposal. Adding a contributor later is a new decomposition revision, which is the revision discipline the rest of the codebase already uses. `CreateContribution` survives as the service-level transactional primitive.
2. **Adapter-readiness and worktree-overlap validation deferred to Phase 3.** The plan listed them here, but roles are not assigned per contributor until execution, `StartAttempt` already probes readiness fail-closed, and worktree isolation is structurally guaranteed one-per-Attempt. Validating them at propose time would duplicate a gate that already exists downstream, against data that does not exist yet.

**Gate met:** a rejected proposal fails closed with the offender named and creates nothing — verified for uncovered criteria, unknown criteria, widened authority, dependency cycles, stale contracts, and a mid-authorization storage failure.

### Phase 3 — Execution across contributing Outcomes — **partially delivered 2026-08-29**

Delivered:

- Dependency gating on `StartAttempt`, running before any plan question and pre-durable like every other start gate: a blocked start writes nothing and spawns nothing. Only an *authorized* decomposition gates; the gate is directional; acceptance follows the newest decision, so an accepted-then-reopened upstream stops clearing it.
- Durable, attributable, reasoned waivers, owner-only, append-only, refused for an ordering nobody declared. A waived dependency stays visible rather than disappearing.
- Parent attention roll-up over each contributor's derived attempt presentation, gate, staleness, and proof status, ordered most-demanding first, every item naming its contributor. Derived at read time, never stored.
- Migration `0108` and the waiver relation in the shared schema file.

**Not delivered, and the reason is a correction to this plan.** Phase 3's original text claimed "each child's Attempt keeps its own isolated worktree — one fenced writer each, which the existing Attempt fence already guarantees per attempt." **That is wrong.** The attempt fence subject is `project:<projectID>`, not per-worktree — a deliberate v0 simplification whose own comment reads "one governed isolated worktree per project, so two Outcomes can never hold simultaneous writers against the same tree."

The consequence: **contributing Outcomes cannot currently run in parallel.** The second contributor's `StartAttempt` fails closed with `AttemptFenceHeldError`. Sessions do get their own worktrees, so the physical isolation the architecture requires already exists; the fence is simply stricter than that isolation needs.

This is a gap between what the visual guide and product architecture promise (parallel contributors on isolated worktrees) and what the code does. It is **not** a correctness problem — composition still delivers independent governance, contained failure, and composable proof, and serialized execution is safe. It is a throughput and experience limitation.

Widening the fence subject is a material change to the mechanism that prevents two writers on one tree, and ADR 0007's consequences never considered it. It therefore needs its own decision rather than being made inside an implementation phase. Two candidate subjects: per contributing Outcome (`outcome:<id>`), or per resolved worktree once the spawn seam can name one before the fence is taken.

**Concurrency budget deferred with it.** While the fence serializes contributors to one at a time, a budget bounding how many may run concurrently has nothing to bound. It also cannot be enforced honestly by a check-then-act count outside the attempt-creation transaction — that reads as enforcement while racing — so it belongs with the fence decision and the wider budget envelope (time, retries, cost), not as a one-off counter.

**Adapter readiness** inherited from Phase 2 is already covered: `StartAttempt`'s existing readiness probe fails closed per attempt through the same checker session spawn uses.

**Gate:** the recovery half of the Phase 3 gate is unchanged and still holds — killing a contributor's provider session contains, reconciles, and produces a safe next action, now surfaced at the parent through the attention roll-up. The parallel-execution half cannot be met until the fence decision is made.

### Phase 4 — Roll-up proof and parent acceptance

- Parent proof view: each parent criterion, the children claiming it, their evidence, verification result and independence class, and any contradiction.
- Parent Ready-for-Acceptance derived from coverage — never stored, always derived, matching the existing "no lifecycle status field" rule.
- The batched acceptance surface from §2, producing N+1 separate immutable `AcceptanceDecision`s from one sitting, with weak or contradicted children excluded and escalated.
- Parent-retained criteria surface as owner actions with their own evidence and verification obligation; the parent cannot reach Ready while one is unproved.
- Reopen and successor semantics.

**Gate:** a child with contradicted evidence or a weaker-than-contracted independence class cannot enter the batch, and the parent cannot reach Ready.

### Phase 5 — Board → Mission Control → Session Inspector

Replace the stage wizard with the canonical three-level navigation:

- **Project Board/List** — cards are Project-level Outcomes with current attention, contributor count, and proof state.
- **Outcome Mission Control** — the contributing-Outcome graph plus an accessible list projection, per the visual guide's "graph plus accessible list" rule.
- **Session Inspector** — one provider session, reached on demand, never the homepage.
- Project Waldo's context chip gains the Outcome scope it already anticipates.
- All of it through `tokens.css` and shadcn primitives per `DESIGN.md`; demoed with `kennel preview`, not described.

**Gate:** restart returns the user to the exact canonical state, selection, and next safe action.

### Phase 6 — Evaluation against the falsification gate

Run the dogfood protocol on composed Outcomes and compare against both direct-Codex use *and* flat single-Outcome Kennel use. The measure that decides whether §2's cost was paid: **median active supervision minutes per accepted Outcome.**

If composition raises supervision cost, we report that plainly and revisit batched acceptance — not relabel the result.

---

## 5. Risks, honestly stated

| Risk | Why it matters | Mitigation |
| --- | --- | --- |
| **Acceptance multiplication** | Directly attacks the product's primary success measure. | Batched interaction with unbatched authority (§2); measured in Phase 6, not assumed. |
| **Coverage theatre** | A decomposition that claims every parent criterion while children prove them weakly makes "done" less trustworthy than the flat model, not more. | Independence class enforced per criterion at roll-up; weak proof cannot enter the acceptance batch. |
| **Two decomposition mechanisms** | Child Outcomes and multi-WorkUnit Missions would be two systems with two authority and staleness models. | Do not widen `PlanRevision`; the child layer *is* the graph (§3.4). |
| **Staleness storms** | One parent contract revision invalidating four in-flight children could stall everything. | Cascade blocks *new* plan approval and attempt start only; running attempts keep tactical freedom and reconcile at their fence, matching the existing lease design. |
| **Shared-surface collision** | Phases 1–2 touch DTO registry, route registration, migration numbers, and generated API files — the build program's named-integration-owner surfaces. | One PR owns each shared file at a time; claim the migration number in the issue before editing schema. |
| **Serialized contributors** | The attempt fence is `project:<id>`, so contributors cannot run in parallel — a gap against the promised experience, found in Phase 3. | Named as its own decision (see Phase 3) rather than resolved by widening a safety mechanism inside an implementation phase. |
| **Docs drifting from code** | `STATUS.md` already understates `beta`. | Phase 0 is a real gate, not a formality. |

---

## 6. Decisions resolved

Confirmed by the owner on 2026-08-29:

1. **Depth cap: 2 levels.** Project-level Outcome plus contributing Outcomes. Schema carries headroom; the domain rejects deeper.
2. **Batched acceptance: adopted.** One review sitting produces N separate immutable `AcceptanceDecision`s. The daemon may only *exclude* a child from the batch, never approve one. Its cost is measured in Phase 6, not assumed away.
3. **Mission scope: the child layer is the graph.** `PlanRevision` stays at exactly one direct WorkUnit. Dependencies live between contributing Outcomes.
4. **Parent-retained criteria: allowed.** A parent may keep criteria no child claims, proved directly by the owner. Decomposition need not be total — but retention must be *explicit* in the `DecompositionRevision`, so an unclaimed criterion is a deliberate choice and never a silent gap. Parent readiness requires every retained criterion to carry its own evidence and verification.
5. **Phase 5 confirmed, sequenced after Phase 4.** The navigation rework lands on top of a working composition model rather than in parallel behind the wizard. Assumption on record: if you meant "run it in parallel", say so and I will re-sequence.

### Bundling

The uncommitted OpenCode admission work and the `AgentOptionalAuth` readiness fix ride on this branch rather than taking their own. They are a prerequisite in practice: OpenCode is the second all-roles harness, and a decomposed Outcome whose children resolve to different harnesses is the first thing that exercises multi-harness role resolution.

## 7. Out of scope

Home / Personal Agent, capture, durable Memory, Learning L1–L4, Gmail Communication Loops, hosted attachment, mobile. Composed Outcomes touch only the Work lane. A `ResponsibilityLink` from a Home Open Loop may point at a Project-level Outcome exactly as it points at a flat one today.

No merge, push, release, or destructive change is authorized by this document.

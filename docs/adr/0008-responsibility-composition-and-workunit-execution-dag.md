# ADR 0008: Responsibility Composition and WorkUnit Execution DAG

- **Status:** Accepted
- **Date:** 2026-09-04
- **Scope:** Outcome decomposition semantics, direct-Outcome Plan shape, WorkUnit dependencies, staleness, and Mission Graph truth
- **Supersedes:** the mechanism-choice clauses of ADR 0007 that treated Outcome composition and a WorkUnit graph as competing decomposition systems
- **Preserves from ADR 0007:** criterion-bound contribution, authority narrowing, stale-parent handling, independent child proof, user-only acceptance, and the v1 composition-depth cap

## Context

ADR 0007 correctly introduced independently governed contributing Outcomes, criterion-bound contribution, fail-closed coverage, authority narrowing, proof roll-up, and user-only acceptance. It also made a narrower mechanism decision: `PlanRevision` would remain exactly one direct WorkUnit and all dependencies/ordering would live between contributing Outcomes because a WorkUnit graph and Outcome composition were considered competing decomposition systems.

That mechanism conflates two different problems.

A responsibility can split into independently meaningful results that deserve separate Contracts, proof, acceptance, and authority. Separately, one bounded responsibility can require several execution steps with dependencies, parallelism, workspaces, retries, and provider roles while still sharing one Contract and one acceptance decision.

Forcing the second case into contributing Outcomes makes implementation steps look like user responsibilities, multiplies acceptance unnecessarily, and makes the Board/Mission hierarchy harder to reason about. Forcing the first case into WorkUnits hides independent governance and proof.

## Decision

### 1. Separate the two decomposition layers

The canonical rule is:

> **Create another Outcome when responsibility splits. Create another WorkUnit when execution splits.**

- **Outcome composition** is responsibility decomposition.
- **WorkUnit DAG** is execution decomposition inside one direct Outcome.

These are complementary layers, not alternative encodings of the same graph.

### 2. Direct Outcome shape

A direct Outcome owns:

```text
Outcome
└── ContractRevision
    └── PlanRevision
        └── bounded WorkUnit DAG
```

A PlanRevision may contain multiple WorkUnits and explicit dependency edges. The plan remains immutable/versioned once authorized.

A WorkUnit is the smallest independently schedulable Kennel execution node. It may have its own:

- objective/output summary;
- dependency IDs;
- provider/role capability requirements;
- workspace/write/effect needs;
- evidence/verification responsibility;
- stop/recovery policy;
- retry Attempts.

### 3. Decomposed Outcome shape

A decomposed parent owns:

```text
Outcome
└── ContractRevision
    └── DecompositionRevision
        ├── Contributing Outcome A
        ├── Contributing Outcome B
        └── Contributing Outcome C
```

Each contributing Outcome is a full responsibility and, when direct, owns its own Contract → Plan → WorkUnit DAG → Attempts → proof → Acceptance lineage.

A decomposed parent does **not** also own direct WorkUnits in v1. This avoids a hybrid in which some responsibility is governed by child Contracts while other execution is hidden in the parent Plan.

If parent-level integration is itself an independently meaningful result, model it as a contributing Outcome. If it is merely proof/integration necessary to establish the parent Contract, model it as parent verification/acceptance work rather than a hidden direct execution branch.

### 4. Composition depth remains bounded

The v1 composition depth cap from ADR 0007 remains: a top-level/decomposed Outcome may have one contributing layer, and a contributing Outcome may not recursively decompose again.

This is a governance/evaluation limit, not a schema claim. Raise it only after dogfood demonstrates a need.

### 5. Contribution remains criterion-bound

Every `ContributionLink` names the exact parent criterion IDs it contributes to. Every parent criterion is either:

- claimed by one or more contributors; or
- explicitly parent-retained.

Authorization fails closed if any criterion is unclassified. Parent-retained means the parent owns the proof obligation; it does not waive proof.

### 6. Authority never widens downward

A contributing Outcome's effective authority is constrained by its parent and may only narrow. WorkUnits and Attempts also operate under the intersection of Contract authority, Plan grants, provider capability, workspace custody, effect policy, and runtime budget.

No model-authored plan/decomposition can bypass these intersections.

### 7. WorkUnit DAG validation is deterministic

Before authorization, daemon validation must reject:

- duplicate WorkUnit IDs;
- unknown dependencies;
- self-dependencies;
- cycles;
- unsupported/unknown capabilities;
- invalid evidence/verification obligations;
- authority or workspace requirements outside the Contract/Project ceiling.

Cycle/dependency validity is a deterministic graph invariant, never an LLM judgment.

The authorized Plan freezes the graph topology and relevant execution requirements. Material topology/dependency/authority/provider-role/workspace/effect changes require a new PlanRevision.

### 8. Staleness has separate responsibility and execution semantics

A new `ContractRevision` may stale:

- the current PlanRevision for a direct Outcome;
- the current DecompositionRevision for a decomposed Outcome;
- Evidence/Verification bound to superseded criterion identity.

A new parent ContractRevision stales the DecompositionRevision bound to the prior parent revision and blocks new child authorization until explicit re-binding/reconciliation.

A Plan revision does not create a new Outcome unless the responsibility itself changed. Failed execution creates a new Attempt (or Plan revision if strategy materially changes), not a fake Outcome.

An in-flight Attempt is never assumed dead merely because its Contract/Plan became stale. It is reconciled/fenced under ADR 0009 and its retained artifacts/evidence remain provenance-bearing historical facts.

### 9. Provider-native subagents are not automatically WorkUnits

A provider may spawn internal/native subagents. Kennel does not mirror those into canonical WorkUnits by default.

Promote a piece of provider-native work into a WorkUnit only when it needs independent Kennel-level scheduling, authority, workspace/effect boundaries, retry semantics, artifacts, dependencies, or proof.

This prevents a Mission with five WorkUnits and twenty provider child sessions/retries from becoming a twenty-node user graph.

### 10. Mission Graph must represent daemon/scheduler truth

The user-facing Graph is a projection of authorized composition + WorkUnit DAG + execution facts.

For a direct Outcome, Graph opens on the WorkUnit DAG.

For a decomposed Outcome, Graph shows the contribution layer and allows drill-down into each child's DAG.

Attempts/Sessions are subordinate detail by default.

Do not ship graph edges, parallel animation, or “running concurrently” semantics before the daemon actually enforces/schedules those relationships.

## Consequences

### Benefits

- User responsibility and implementation structure no longer compete for one graph abstraction.
- A bounded Outcome can use parallel/dependent execution without multiplying Contracts/acceptance.
- Independently meaningful results retain separate proof/authority/acceptance.
- Provider sessions and native subagents stay at the execution layer rather than inflating product ontology.
- Board → Mission Control → WorkUnit → Session becomes mechanically explainable.

### Costs

- `PlanRevision` domain/storage/API must migrate from exactly one direct WorkUnit to a bounded DAG.
- Staleness/revision logic must reason about WorkUnit-specific execution/evidence.
- Scheduler/workspace semantics become required; a graph-only frontend would be dishonest.
- Tests must cover cycles, dependency admission, partial failure, replan, contract revision, and composed-child interactions.

## Rejected alternatives

1. **Keep ADR 0007's “contributing layer is the only graph.”** Rejected because execution steps would continue to masquerade as independent responsibilities.
2. **Use only a WorkUnit graph and remove composed Outcomes.** Rejected because independently meaningful results need their own Contracts, authority, proof, and acceptance.
3. **Allow a decomposed parent to also own arbitrary direct WorkUnits in v1.** Rejected because the hybrid has ambiguous governance and proof ownership.
4. **Mirror every provider child/subagent into WorkUnits.** Rejected because provider-internal decomposition is an execution optimization unless Kennel-level boundaries are needed.
5. **Let a coordinator model decide whether dependency cycles/authority conflicts are acceptable.** Rejected because these are deterministic safety invariants.

## Implementation boundary

This ADR changes architecture authority; it does not itself change the current runtime. Implementation follows `docs/product/kennel-build-program.md`:

1. widen domain/storage/API;
2. implement WorkspaceLease/scheduler semantics under ADR 0009;
3. only then render the final truthful Mission Graph.

Already-merged migrations remain immutable. New persistence shape lands through additive migrations and generated contract updates.

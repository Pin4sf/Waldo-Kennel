# ADR 0007: Composed Outcomes

- **Status:** Accepted; **superseded in part by ADR 0008**
- **Date:** 2026-08-29
- **Consolidated note:** 2026-09-04
- **Scope still authoritative:** responsibility composition, criterion-bound contribution, authority narrowing, stale-parent handling, independent child proof, user-only acceptance, v1 composition-depth cap
- **Superseded mechanism:** the former claim that Outcome composition replaces/competes with an internal WorkUnit graph
- **Current mechanism authority:** [`0008-responsibility-composition-and-workunit-execution-dag.md`](0008-responsibility-composition-and-workunit-execution-dag.md)

The original accepted ADR text remains available in Git history. This consolidated record preserves the decisions that still govern v1 while making the supersession explicit for future coding agents.

## Context

Kennel needs to represent a delegated responsibility that itself separates into several independently meaningful results. Those parts may require different success criteria, evidence, risk, authority, recovery, and human acceptance. Encoding all of them as anonymous execution steps under one responsibility would make proof and governance ambiguous.

ADR 0007 therefore introduced **contributing Outcomes**: a parent Outcome can be decomposed into child Outcomes that are full responsibilities in their own right.

The original ADR also chose composition *instead of* a WorkUnit graph. That part is no longer current. ADR 0008 clarifies that the two structures operate at different semantic layers:

- contributing Outcomes = **responsibility decomposition**;
- WorkUnit DAG = **execution decomposition** inside one direct Outcome.

## Decision still in force

### 1. A decomposed Outcome owns a DecompositionRevision

A v1 Outcome is either:

- **Direct:** owns a PlanRevision and executes through a WorkUnit DAG; or
- **Decomposed:** owns a DecompositionRevision and contributing Outcomes, and does not also own direct WorkUnits in v1.

Each contributing Outcome is a complete responsibility with its own Contract, direct Plan/DAG when applicable, Attempts, Evidence, Verification, and AcceptanceDecision.

### 2. Composition depth is capped in v1

A top-level/decomposed Outcome may have one contributing layer. A contributing Outcome may not recursively decompose again in v1.

The cap is a governance/product limit, not a storage claim. Raise it only after dogfood demonstrates a real need.

### 3. Contribution is criterion-bound

Every contributing Outcome declares which exact parent `CriterionID`s it contributes to through immutable contribution lineage.

Every parent criterion must be either:

- claimed by at least one contributor; or
- explicitly parent-retained.

Authorization fails closed when a criterion is unclassified.

Parent-retained means the parent keeps the proof obligation. It never means the criterion can be ignored.

### 4. Authority never widens downward

A contributing Outcome's authority ceiling must be a subset/intersection of its parent's authority. A child may narrow authority; it may never widen it.

The same principle continues downward through Plan grants, provider capability, workspace/effect custody, and runtime admission.

### 5. Parent Contract changes stale composition safely

A new parent ContractRevision supersedes the DecompositionRevision bound to the prior revision and makes the old contribution binding stale for new authorization/proof decisions.

A stale parent does **not** prove an already-running child Attempt is dead. In-flight execution is reconciled/fenced according to ADR 0009 and remains historical/provenance-bearing.

### 6. Child proof remains independently inspectable

A contributing Outcome earns its own criterion-bound Evidence and Verification. Parent proof/readiness can roll up contribution only through explicit criterion bindings and current valid proof.

A weak/contradicted child cannot be hidden by sibling success.

### 7. Acceptance authority is never delegated to automation

The product may batch the user's review interaction for several ready contributors, but each responsibility retains its own immutable AcceptanceDecision.

No model, provider session, verifier, CI result, or daemon policy may create AcceptanceDecision on the user's behalf.

All contributors becoming accepted can make the parent **Ready for Review**; it does not implicitly accept the parent. The parent still requires its own explicit user decision.

### 8. Agent-authored decomposition proposals are untrusted proposals

A provider/model may propose a decomposition through the daemon's structured callback/API path. It passes the same deterministic validation as a human-authored proposal:

- criterion coverage;
- authority containment;
- criterion identity;
- dependency/cycle rules where applicable;
- stale-revision binding;
- idempotency/request scope.

There is no trusted-proposer bypass.

Any callback token used for request scoping on the unauthenticated loopback listener is a **scoping/idempotency mechanism, not authentication**. Do not describe it as a security boundary against hostile local processes.

## Superseded decision

The original ADR stated that the contributing-Outcome layer should be the only graph and that `PlanRevision` should remain exactly one direct WorkUnit.

That decision is superseded by ADR 0008.

Current rule:

> **Create another Outcome when responsibility splits. Create another WorkUnit when execution splits.**

A direct Outcome may therefore own a bounded WorkUnit DAG without weakening any of the contribution, authority, proof, staleness, or acceptance rules above.

## Consequences

### Benefits retained

- independently meaningful results receive independent governance and proof;
- child failure/replanning does not rewrite sibling responsibility identity;
- parent proof can be explained criterion by criterion;
- authority narrowing is explicit;
- human closure remains conscious and auditable.

### Costs retained

- contribution coverage and staleness are fail-closed paths that require clear error attribution;
- several child responsibilities can multiply review burden, so composition should be used only when the responsibility truly splits;
- composition depth and supervision cost remain dogfood/evaluation concerns.

## Rejected alternatives still rejected

- unbounded recursive Outcome nesting without evidence;
- implicit parent Acceptance when children are accepted;
- model/policy-driven Acceptance;
- contribution without criterion binding;
- child authority wider than parent authority.

## Current implementation references

For current architecture and sequencing use:

- [`../product/kennel-v1-product-architecture.md`](../product/kennel-v1-product-architecture.md)
- [`0008-responsibility-composition-and-workunit-execution-dag.md`](0008-responsibility-composition-and-workunit-execution-dag.md)
- [`0009-workunit-scheduling-workspace-leases-and-effect-fencing.md`](0009-workunit-scheduling-workspace-leases-and-effect-fencing.md)
- [`../product/kennel-build-program.md`](../product/kennel-build-program.md)
- [`../superpowers/plans/2026-09-04-kennel-builds-kennel.md`](../superpowers/plans/2026-09-04-kennel-builds-kennel.md)

The historical 2026-08-29 program path is retained only as a superseded pointer; the verbatim original ADR/program are available in Git history when provenance is needed.

# Waldo Kennel documentation

This directory contains both current architecture authority and historical/future research. They are **not** equal sources of truth.

## Kernel authority order

For Work/kernel implementation, use this order:

1. [`../AGENTS.md`](../AGENTS.md) — repository rules and non-negotiable boundaries.
2. [`product/kennel-v1-product-architecture.md`](product/kennel-v1-product-architecture.md) — canonical v1 kernel/product ontology.
3. [`adr/0008-responsibility-composition-and-workunit-execution-dag.md`](adr/0008-responsibility-composition-and-workunit-execution-dag.md) — Outcome composition vs WorkUnit DAG.
4. [`adr/0009-workunit-scheduling-workspace-leases-and-effect-fencing.md`](adr/0009-workunit-scheduling-workspace-leases-and-effect-fencing.md) — scheduling, workspaces, concurrency, effects, and recovery.
5. [`STATUS.md`](STATUS.md) — what exists on `beta` today.
6. [`product/kennel-build-program.md`](product/kennel-build-program.md) — ordered implementation slices and gates.
7. [`superpowers/plans/2026-09-04-kennel-builds-kennel.md`](superpowers/plans/2026-09-04-kennel-builds-kennel.md) — next-session implementation plan.
8. [`product/kennel-dogfood-acceptance-matrix.md`](product/kennel-dogfood-acceptance-matrix.md) — falsifiable self-hosting acceptance tests.

Companion Work specifications:

- [`superpowers/specs/2026-08-25-work-control-plane-canonical-flow-design.md`](superpowers/specs/2026-08-25-work-control-plane-canonical-flow-design.md)
- [`superpowers/specs/2026-08-25-work-experience-screen-interaction-spec.md`](superpowers/specs/2026-08-25-work-experience-screen-interaction-spec.md)

Technical references:

- [`architecture.md`](architecture.md) — current Go daemon/package/lifecycle chassis.
- [`research/2026-09-04-kernel-runtime-reference-index.md`](research/2026-09-04-kernel-runtime-reference-index.md) — provider protocols, runtime references, and implementation patterns.
- [`cli/README.md`](cli/README.md) — thin CLI contract.
- [`plans/island-app-unification.md`](plans/island-app-unification.md) — Island/Electron process integration when working on Island lifecycle.

## ADR precedence

ADRs are historical decisions and may be superseded by later ADRs. Do not infer authority from the highest amount of detail in an older ADR.

For the current Work ontology:

- ADR 0007 remains useful for composed-Outcome governance, but its claim that Outcome composition and a WorkUnit graph are competing mechanisms is superseded by ADR 0008.
- ADR 0008 defines responsibility decomposition vs execution decomposition.
- ADR 0009 defines WorkUnit scheduling and workspace/effect custody.
- Earlier local-first, LAN-listener, Home, learning, and durable-Waldo ADRs remain authoritative only inside their stated scope unless a newer ADR says otherwise.

## Future product lanes

Home, personal capture, durable Memory, communication, learning/skill evolution, mobile/health, hosted attachment, and cross-device work are future or parallel lanes. Their research/specifications remain in the repository for scoped tasks, but **kernel coding agents must not load them by default** or use them to override the Work authority chain.

Notable future-lane documents include:

- `superpowers/specs/2026-08-21-home-personal-agent-memory-design.md`
- `superpowers/specs/2026-08-21-waldo-learning-skill-evolution-design.md`
- `adr/0005-governed-project-learning-and-skill-evolution.md`
- `adr/0006-one-durable-waldo-multiple-governed-presences.md`

## Historical documents

Dated plans/specs not linked from the kernel authority order are historical implementation records unless their header explicitly says otherwise. Git history preserves removed handoffs, review packets, prototypes, and superseded delivery plans.

Rules for agents:

- do not recursively ingest all of `docs/` into context;
- do not resurrect a rule solely because an older handoff says it was once approved;
- prefer a current ADR over an older plan;
- prefer `STATUS.md` for implemented reality;
- prefer the canonical architecture for accepted target semantics;
- when current code and accepted target differ, implement only through the ordered build program and preserve compatibility/recovery explicitly.

## One-sentence mental model

> Provider Sessions are execution machinery; Outcomes are what the user owns; Kennel keeps that machinery coherently moving toward the user's definition of done.

# Kennel documentation

Kennel currently consists of an AO-derived Go daemon and Electron supervisor plus a prototype Outcome UI. The documents below separate that shipped foundation from the accepted, still-unimplemented Waldo Kennel desktop launch architecture. Home, Open Loops, Communication Loops, Daily Snapshot, Mission, personal continuity, Waldo authority/verification semantics, and named reference adaptations are not shipped.

## Foundation first

| Document | Purpose |
| --- | --- |
| [Foundation acceptance — 2026-08-18](foundation-acceptance-2026-08-18.md) | F0-F6 scope, evidence, exclusions, dev-dependency debt, and the next architecture boundary. |
| [Identity and state](identity-and-state.md) | Kennel-owned bundle, executable, protocol, updater, state, environment, port, branch, and renderer identities. |
| [Upstream provenance](upstream-provenance.md) | Pinned AO source, license, non-destructive sync seam, and the separately approved ancestry-repair plan. |
| [Status](STATUS.md) | Current chassis capabilities, donor surfaces, and work that is not shipped. |
| [Development](development.md) | Reproducible bootstrap and verification commands. |

## Accepted product architecture

| Document | Purpose |
| --- | --- |
| [Waldo Kennel v0 dogfood and provider-neutral v1 architecture](product/kennel-v1-product-architecture.md) | Canonical local Home + Work thesis, responsibility ontology, governance, provider-neutral adapter/conformance seam, v0 Codex-only testing constraint, RunBrief compilation, dogfood gate, and implementation entry. It is a design baseline, not a shipped-feature claim. |
| [v0 dogfood and provider-neutral v1 team review packet](product/kennel-v1-team-review-packet.md) | Shareable founder/product/design/engineering/privacy review of the local dogfood structure, provider-neutral v1 direction, ontology, lineage, governance, screens, UX flows, failures, scope, and PR entry boundary. |
| [Clickable v0 dogfood review prototype](product/kennel-v1-review-prototype.html) | Low-fidelity walkthrough of Home, Daily Snapshot, communication/Open Loops, Work Outcomes, provider admission, attention, evidence, acceptance, re-entry, optional Desktop Context, inspector, and control surfaces. It is not product code or a shipped UI. |
| [v0 dogfood Excalidraw session seed](product/kennel-v1-excalidraw-session-seed.md) | Twenty-seven review frames, provider-neutral admission prompts, canonical lineage/state strips, failure injections, dogfood measures, and facilitation agenda. |
| [ADR 0003: Local-first Waldo Core](adr/0003-local-first-waldo-core.md) | Accepted v1 deployment and custody decision: Waldo Core inside the Kennel daemon, local canonical storage, user-authenticated providers, and later explicit hosted attachment. |
| [PR convergence and architecture gate plan](superpowers/plans/2026-08-20-pr-convergence-and-architecture-gate.md) | Safe sequence for accepting F0-F6, replacing PR #11 with bounded post-foundation cleanup, and holding the first feature implementation gate. |

## Chassis references

These documents describe the inherited and currently working orchestration implementation. AO vocabulary can remain where it identifies a source compatibility seam; it is not Kennel's installed identity.

| Document | Purpose |
| --- | --- |
| [Architecture](architecture.md) | Backend model, persistence/CDC, lifecycle, API, terminal, browser, and load-bearing boundaries. |
| [Backend code structure](backend-code-structure.md) | Go package ownership and dependency rules. |
| [CLI](cli/README.md) | The `kennel` thin client over daemon HTTP. |
| [Stack](stack.md) | Accepted and pending runtime/library choices inherited by the chassis. |
| [Telemetry](telemetry.md) | Current optional telemetry behavior and privacy safeguards. |
| [Cloud development](cloud-development.md) | Retained optional cloud donor/compatibility work; not a deployed Kennel cloud claim. |
| [Cloud refactor](cloud-refactor.md) | Existing shared contracts and donor boundaries. |

## Architectural rule

Persist durable facts and derive display status. Session activity, termination, PR/check/review facts, and change-log events are durable; a display label is not. Product architecture added after the foundation must preserve truthful, inspectable evidence and explicit human authority.

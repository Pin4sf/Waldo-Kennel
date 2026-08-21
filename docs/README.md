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
| [Waldo Kennel v0 dogfood and provider-neutral v1 architecture](product/kennel-v1-product-architecture.md) | Canonical three-destination, five-stage local Home + Work thesis, responsibility ontology, governance, provider-neutral adapter/conformance seam, v0 Codex-only testing constraint, RunBrief compilation, dogfood gate, and implementation entry. It is a design baseline, not a shipped-feature claim. |
| [v0 dogfood and provider-neutral v1 team review packet](product/kennel-v1-team-review-packet.md) | Shareable founder/product/design/engineering/privacy review of the converged lifecycle, ontology, lineage, governance, adaptive surfaces, failures, scope, and PR entry boundary. |
| [Clickable five-stage review prototype](product/kennel-v1-review-prototype.html) | Presenter-ready low-fidelity walkthrough with the five-stage spine, complete F01-F27 plus F02A screen atlas, per-screen problem/purpose/states, Work-first entry, dense Work/Kanban, Home -> Work lineage, inspector, and control overlays. It is not product code or a shipped UI. |
| [Five-stage Excalidraw session seed](product/kennel-v1-excalidraw-session-seed.md) | Five adaptive lifecycle frames organizing the complete detailed screen subframes, destination/context rail, canonical lineage strips, failure injections, Focus Ledger overlay, privacy defaults, and facilitation agenda. |
| [First complete Outcome slice](product/kennel-v0-first-outcome-slice.md) | Locked Local Focus Ledger contract and evaluation protocol across Enter, Understand, Decide & Authorize, Act & Observe, and Prove & Close. |
| [ADR 0003: Local-first Waldo Core](adr/0003-local-first-waldo-core.md) | Accepted v1 deployment and custody decision: Waldo Core inside the Kennel daemon, local canonical storage, user-authenticated providers, and later explicit hosted attachment. |
| [ADR 0004: Parallel Home/Personal Agent and required governed capture](adr/0004-parallel-home-personal-agent-and-required-capture.md) | Accepted sequencing amendment: Work and Home may proceed in parallel; desktop screen/audio are required capabilities behind explicit CaptureGrants; durable Memory remains separately gated. |
| [Home, Personal Agent, capture, and memory design](superpowers/specs/2026-08-21-home-personal-agent-memory-design.md) | Approved written specification for Home/OpenLoop, shared intake, required screen/audio capture, candidate-first memory, retrieval, deletion, evaluation, and non-overlapping Work/Home ownership. |
| [ADR 0005: Governed project learning and skill evolution](adr/0005-governed-project-learning-and-skill-evolution.md) | Accepted Paxel-observer + AutoResearch-experimenter + Waldo-governor decision; no automatic promotion, opaque scoring, or self-authorizing skill changes. |
| [Waldo Learning and Skill Evolution design](superpowers/specs/2026-08-21-waldo-learning-skill-evolution-design.md) | Written specification for consented LearningEpisodes, candidates, experiments, held-out evaluation, skill registry, provisional activation, rollback, deletion, and lane ownership. |
| [ADR 0006: One durable Waldo with multiple governed presences](adr/0006-one-durable-waldo-multiple-governed-presences.md) | Accepted final ecosystem target: one durable Waldo agent, Kennel desktop Work presence, and a Health-aware mobile presence over the same identity and continuity. |
| [AO legacy retirement audit](product/ao-legacy-retirement-audit.md) | Retain/rename/retire/migrate inventory for provenance, compatibility seams, active AO product strings/assets, donor landing/mobile trees, frozen packages, and stale docs. |
| [Waldo Kennel build program](product/kennel-build-program.md) | Dependency graph, lane ownership, canonical specs/plans, milestone map, and implementation gates across Foundation, Work, Home, Learning, and the later ecosystem. |
| [PR convergence and architecture gate plan](superpowers/plans/2026-08-20-pr-convergence-and-architecture-gate.md) | Completed prerequisite record for F0-F6 and the bounded PR #11 donor extraction, plus the still-current first-feature gate. |
| [First Outcome execution handoff](superpowers/plans/2026-08-20-first-outcome-execution-handoff.md) | Exact new-session PR/issue sequence, file ownership, tests, dependencies, acceptance, falsifiers, and handoff constraints for the five-stage Focus Ledger milestone. |
| [Home and Personal Agent foundations plan](superpowers/plans/2026-08-21-home-personal-agent-foundations.md) | Issue-sized shared-intake, Home/OpenLoop, capture, candidate-memory, retrieval, and evaluation sequence. |
| [AO legacy retirement plan](superpowers/plans/2026-08-21-ao-legacy-retirement.md) | Active rebrand plus donor retirement sequence with consumer checks, rollback, and a compatibility allowlist. |
| [Learning Experience Ledger plan](superpowers/plans/2026-08-21-learning-experience-ledger.md) | L1 consent, attribution, rebuildable episodes, candidate routing, correction, deletion, and shadow evaluation. |
| [Learning Experiment/Evaluation plan](superpowers/plans/2026-08-21-learning-experiment-evaluation.md) | L2 immutable campaigns, isolated variants, hidden held-out/no-regression evaluation, budgets, and review. |
| [Learning Skill Registry plan](superpowers/plans/2026-08-21-learning-skill-registry.md) | L3 daemon registry, Project-scoped provisional promotion, RunBrief binding, receipts, monitoring, rollback, revoke, and deletion. |

## Personal Agent, memory, and learning research

| Document | Purpose |
| --- | --- |
| [Omi implementation reference](research/2026-08-21-omi-implementation-reference.md) | Pinned source review of Omi capture, admission, authority, task, memory, and deletion mechanisms with Waldo Adopt/Adapt/Reject decisions. |
| [Project Minimi reference](research/2026-08-21-project-minimi-reference.md) | First-party product/artifact review of local durability, cloud-assisted processing, ambient continuity, Open Loop claims, and Waldo boundaries. |
| [Dayflow versus Omi capture](research/2026-08-21-dayflow-vs-omi-capture-reference.md) | Source-backed comparison of desktop visual capture and conversational audio capture for Waldo's required governed source plane. |
| [Personal-agent memory benchmark](research/2026-08-21-personal-agent-memory-research-benchmark.md) | MIRIX, Hindsight, MemOS, and A-MEM comparison covering typed memory, temporal claims, provenance, consolidation, deletion, and evaluation. |
| [Agent-memory infrastructure benchmark](research/2026-08-21-agent-memory-infrastructure-benchmark.md) | Mem0, Graphiti/Zep, and Supermemory comparison covering canonical storage, retrieval projections, correction, deletion, and failure behavior. |
| [Orchestration/runtime memory benchmark](research/2026-08-21-orchestration-runtime-memory-benchmark.md) | Letta, LangGraph/LangMem, and AutoGen comparison separating checkpoints, context, durable knowledge, responsibility, and proof. |
| [Paxel and AutoResearch learning reference](research/2026-08-21-paxel-autoresearch-learning-reference.md) | Clean-room project/session analysis plus pinned bounded-experiment reference, synthesized into Waldo Adopt/Adapt/Reject and a governed skill-evolution boundary. |

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

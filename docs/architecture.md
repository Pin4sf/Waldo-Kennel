# Kennel technical chassis architecture

- **Status:** Current technical reference for the implemented daemon/runtime chassis
- **Consolidated:** 2026-09-04
- **Product/kernel authority:** [`product/kennel-v1-product-architecture.md`](product/kennel-v1-product-architecture.md)
- **Implemented reality:** [`STATUS.md`](STATUS.md)

This document describes the technical chassis that already exists in Kennel: daemon boundaries, ports/adapters, persistence/CDC, session/runtime lifecycle, terminal/browser supervision, and recovery foundations. It is **not** the authority for Outcome/Contract/Plan/WorkUnit semantics, scheduler policy, or product UX. Read the canonical product architecture plus ADR 0008/0009 for those decisions.

Historical AO-derived names may still exist in source comments or compatibility seams. Kennel is a standalone product/repository; do not restore AO product identity or its broad provider/task ontology from old history.

## 1. System boundary

Kennel is local-first. The Go daemon is the canonical runtime/control process and SQLite is the canonical Work-state writer.

```text
Electron / CLI / Island / provider shims
                │
                ▼
        loopback daemon API
                │
      services + domain policy
                │
   ports ───── adapters/runtime
                │
      SQLite + change_log
```

The renderer is a client/projection. Closing it does not define whether an Attempt/session exists.

## 2. Repository layout

```text
backend/
  cmd/kennel/                  CLI/daemon entrypoint
  internal/domain/             canonical domain vocabulary/invariants
  internal/ports/              adapter/runtime/storage contracts
  internal/service/            controller-facing application services
  internal/httpd/              REST/SSE/terminal endpoints + generated spec source
  internal/storage/sqlite/     migrations, queries, stores, trigger CDC
  internal/adapters/           provider/runtime/workspace/SCM implementations
  internal/lifecycle/          lifecycle/reconciliation logic
  internal/daemon/             production wiring

frontend/                      Electron + React supervisor
packages/kennel-island/        ambient projection inside the desktop architecture
docs/                          architecture/ADRs/specs/programs
```

Inspect the actual current tree before relying on a path not listed here; package names evolve as implementation slices land.

## 3. Core technical principles

### 3.1 Ports/adapters

Core services/domain logic should depend on narrow port interfaces rather than concrete provider/runtime implementations. Provider capabilities belong at adapter/manifest boundaries and are admitted through deterministic policy.

### 3.2 Durable facts, derived projections

Persist canonical facts and derive UI/attention status from them. Do not persist a second display lifecycle simply because a frontend needs `working`, `waiting`, or `needs_you`.

For the v1 kernel this principle extends above sessions: Outcome/Contract/Plan/WorkUnit/Attempt/Evidence/Verification/Acceptance facts drive Board/Mission/Island projections.

### 3.3 Trigger-backed CDC

SQLite mutations flow through database triggers into `change_log`; the daemon tails/broadcasts changes to clients. Do not add a second manual event authority from store methods without an explicit architecture decision.

```text
SQLite mutation
   ↓ trigger
change_log
   ↓ poll/broadcast
SSE / clients
```

### 3.4 Thin clients

CLI, frontend, Island, and provider integrations call daemon APIs. They do not open canonical SQLite directly or recreate policy locally.

## 4. Daemon/network boundary

The primary daemon listener remains loopback-only (`127.0.0.1`) under the repository's existing security decision. The separately governed opt-in LAN listener remains subject to ADR 0001 and its authentication/control-route restrictions.

Do not broaden the network surface as part of unrelated kernel work.

## 5. Persistence and generated contracts

SQLite schema changes are additive. Never edit already-merged migrations.

When changing storage:

1. add the next migration;
2. update source queries/schema;
3. run `npm run sqlc`;
4. keep trigger/change-log behavior intact.

When changing daemon API DTO/routes:

1. edit controller/spec source;
2. run `npm run api`;
3. commit generated OpenAPI + frontend schema together;
4. run route/spec parity tests.

Generated files are outputs, not editing surfaces.

## 6. Existing session/runtime chassis

The repository already supports Project/session lifecycle, provider adapters, terminal/native chat modes, Git worktrees, browser preview, PR/check/review observation, process/reaper foundations, and recovery facts.

A provider session is an execution mechanism. The canonical v1 architecture places it beneath:

```text
Outcome → ContractRevision → PlanRevision → WorkUnit → Attempt → AgentSessionRef
```

Do not extend legacy session/task machinery upward into responsibility truth.

## 7. Provider boundary

PR #92 reduced the active first-class provider surface to:

- Codex
- Claude Code
- OpenCode
- Cursor
- Pi

Readiness and role admission are capability-derived and machine-aware. Historical provider identities may remain readable for compatibility/recovery but are not active new-work providers.

Native structured protocols/ACP/RPC/CLI are adapter concerns. None becomes the Kennel domain model.

See [`research/2026-09-04-kernel-runtime-reference-index.md`](research/2026-09-04-kernel-runtime-reference-index.md).

## 8. Workspace/runtime evolution boundary

Today the existing session/worktree machinery is the implementation substrate. The accepted target under ADR 0009 evolves it into explicit WorkUnit-level `WorkspaceLease` and scheduler custody.

Until that migration lands:

- do not claim real WorkUnit concurrency that the current Project-wide Attempt fence still prevents;
- preserve existing recovery safety while replacing the fence in verified slices;
- never infer provider death/completion from a failed or missing probe;
- never force-delete dirty/unknown worktrees.

## 9. Session interface continuity

Where a provider has proven compatible native identities/protocol behavior, Kennel may support same-provider controller/interface transition or resume without changing the higher-level responsibility identity.

Across providers, the canonical operation is a new Attempt plus an attributed handoff/continuation packet unless an exact provider-native transition is proven. Do not describe cross-provider handoff as lossless hidden-state migration.

## 10. Observation and reconciliation

External/process/provider observations are inputs to canonical state, not truth by themselves.

The lifecycle/reconciliation layer must distinguish:

- observed alive/active;
- confirmed terminal;
- interrupted/failed;
- unknown/unconfirmed.

On restart, load non-terminal canonical facts, probe the strongest available provider/workspace/runtime identity, reconcile, fence duplicates/effects, and only then create recovery work when policy permits.

## 11. Browser/terminal/PR surfaces

Terminal, native chat, browser/preview, diff, branch/commit, PR/check/review observation, and provider-native child details remain valuable deep execution surfaces. In the product hierarchy they belong primarily in WorkUnit/Attempt/Session Inspector context, not as the default Work responsibility view.

## 12. Current architectural transition

The chassis is intentionally being evolved rather than rewritten:

```text
existing durable daemon + SQLite + worktrees + recovery
                    ↓
ProjectBriefRevision + WorkUnit DAG
                    ↓
WorkspaceLease + dependency scheduler
                    ↓
structured provider conformance + receipts
                    ↓
Board / Mission Graph / Island projections
                    ↓
Kennel-builds-Kennel dogfood
```

For exact sequencing and shipped-vs-target truth, use:

- [`product/kennel-build-program.md`](product/kennel-build-program.md)
- [`superpowers/plans/2026-09-04-kennel-builds-kennel.md`](superpowers/plans/2026-09-04-kennel-builds-kennel.md)
- [`STATUS.md`](STATUS.md)

## 13. Non-negotiable technical invariants

- daemon/SQLite remain canonical; frontend/plugins do not become competing writers;
- deterministic dependency/authority/idempotency/effect/recovery checks remain deterministic;
- session/provider completion does not accept an Outcome;
- failed/unknown probes do not prove death;
- retries/recovery create traceable Attempt lineage rather than rewriting history;
- dirty/unknown workspaces are preserved for inspection;
- migrations are additive;
- generated contracts are regenerated from source;
- UI concurrency must not outrun scheduler truth.

This file should remain a concise chassis reference. Deep historical AO implementation details are available in Git history if needed; they should not be loaded by default into future kernel coding sessions.

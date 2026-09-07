# Kennel status

**Runtime baseline:** `beta` after merged PR #92 (`5e85cd5`, 2026-09-04)  
**Architecture authority:** [`product/kennel-v1-product-architecture.md`](product/kennel-v1-product-architecture.md)  
**Current build focus:** first self-hosting Waldo Kennel kernel

This file separates **implemented runtime truth** from **accepted target architecture**. A design document, Figma frame, or plan is not evidence that a feature is shipped.

## Implemented on current `beta`

### Chassis

- standalone Kennel Go daemon bound to loopback with the existing governed opt-in LAN path;
- SQLite persistence with additive migrations;
- trigger-backed `change_log` CDC and SSE projection/update flow;
- generated OpenAPI + frontend TypeScript contracts;
- thin Cobra `kennel` CLI over daemon HTTP;
- Electron + React desktop supervisor;
- project/session lifecycle;
- Git worktree management and cleanup/recovery machinery;
- native chat, terminal, diff/browser/preview surfaces;
- PR/check/review observation;
- restart/reaper/reconciliation foundations.

### Provider core

PR #92 is merged. The active first-class provider surface for new work is exactly:

- Codex
- Claude Code
- OpenCode
- Cursor
- Pi

The refactor removed the broad donor provider registry from the active product path, made readiness machine-aware, removed hidden Codex fallback from product selection, and made project/reviewer/switch UX derive from provider inventory rather than a single hardcoded option.

Provider **identity** support does not imply every provider has passed every structured-control role. Current structured coordinator/reviewer/switch capability is narrower than the five-provider worker surface; new roles must be admitted through conformance, not optimistic brand lists.

### Canonical Outcome foundation

`beta` already carries the durable responsibility lineage through:

```text
Outcome
→ ContractRevision
→ PlanRevision
→ WorkUnit
→ Attempt
→ AgentSessionRef
→ EvidenceItem
→ VerificationRun
→ AcceptanceDecision
```

Implemented properties include:

- immutable Contract revisions and stable criterion identity;
- owner-gated Plan authorization and capability grants;
- real provider Attempts and provider session references;
- recovery/reconciliation receipts/facts for the current execution path;
- criterion-bound Evidence;
- explicit verification identity/independence classification;
- user-only Acceptance decisions;
- adaptive intake/callback path without canonical transcript-marker parsing;
- durable bounded Project Waldo conversation;
- composed Outcomes with criterion-bound contribution, authority narrowing, dependency gating/waivers, stale-parent handling, proof roll-up, and batched interaction with separate Acceptance decisions;
- an Outcome Mission Control destination and session drill-down.

## Accepted target but not implemented yet

These are the immediate kernel gaps. Do not describe them as shipped.

### 1. `ProjectBriefRevision`

There is no persisted/versioned Project Brief/Charter object yet. The target separates persistent Project context from finite Outcome Contracts.

### 2. Real WorkUnit DAG inside a direct Outcome

Current `PlanRevision` still validates exactly one `direct` WorkUnit. The target widens a direct Outcome's Plan into a bounded DAG with explicit dependencies.

ADR 0008 supersedes the old rule that dependencies belong only between contributing Outcomes.

### 3. WorkUnit scheduler and `WorkspaceLease`

Current Attempt execution still uses a Project-wide fence. That prevents truthful parallel contributing/WorkUnit writes.

The target introduces:

- dependency-aware WorkUnit admission;
- explicit WorkspaceLease/worktree ownership;
- concurrency budgets;
- staged workspace provisioning;
- narrower write/integration/effect fences;
- restart reconciliation with `unknown` / `unconfirmed` preserved as real states.

ADR 0009 is authoritative.

### 4. Truthful Mission Graph

Mission Control exists, but a final execution Graph is not shipped. Until the scheduler can really run independent branches concurrently, the UI must not imply concurrency it cannot provide.

### 5. Canonical receipts and Project continuity

The durable Project conversation is useful, but the target continuity path still needs:

```text
structured provider events + workspace facts
→ SessionReceipt
→ WorkUnitReceipt
→ Outcome current brief / ledger
→ governed Project Context candidate
```

The user should be able to supervise routine Outcomes without reconstructing provider transcripts.

### 6. Capability-derived provider role admission

The five-provider product surface is merged, but deeper structured control remains uneven. Cursor ACP and Pi RPC/SDK integrations require explicit driver/conformance work before coordinator/switch/reviewer roles are enabled. Claude/OpenCode/Codex paths also require ongoing version-pinned conformance.

### 7. External provider ingress

`Governed | Observed | Untracked` external activity and an explicit execution/binding envelope are accepted architecture but not yet first-class daemon protocol. No fuzzy auto-attachment is permitted.

### 8. Donor Outcome overlay cleanup

The legacy `OutcomeTask` / `completed` presentation overlay is superseded by canonical Outcome lineage and remains cleanup work.

### 9. Final Board/List + Project Brief UI

Current Work surfaces partially represent the desired hierarchy. The final active Board/List should project top-level Outcomes, while Project Brief and Mission Control become explicit durable surfaces.

### 10. Island consequence projection

Island exists as part of the desktop app architecture, but the final Outcome-first consequence/attention projection depends on the same scheduler/receipt truth above. It must not maintain a separate execution database.

## Explicitly deferred from first self-hosting kernel

The following do not block Kennel-builds-Kennel dogfood:

- learned automatic provider routing;
- automatic skill promotion;
- complete personal/cross-project Memory;
- hosted/cloud workspace runtime;
- multiplayer/team governance;
- deep recursive Outcome composition;
- arbitrary lossless capture of already-running external sessions without a provider hook/protocol;
- provider-role parity across all five providers before conformance;
- autonomous final Outcome acceptance;
- health/mobile/relationship surfaces.

The governed learning architecture in ADR 0005 may operate in shadow/candidate mode later, but it does not block the execution kernel.

## Current implementation order

1. documentation/ADR consolidation — this PR;
2. domain + persistence: Project Brief + WorkUnit DAG;
3. WorkspaceLease + scheduler + narrower fences;
4. provider structured-driver conformance;
5. SessionReceipt / WorkUnitReceipt / Outcome brief;
6. canonical Board/List + Project Brief UI + donor overlay removal;
7. Mission Control Contract | Graph backed by scheduler truth;
8. external ingress;
9. Island consequence projection;
10. Kennel-builds-Kennel dogfood and evaluation.

See [`product/kennel-build-program.md`](product/kennel-build-program.md).

## Dogfood gates

The first kernel is judged by falsifiable behavior, not feature count:

- silent provider fallback: **0**;
- duplicate Attempts caused by restart/retry: **0**;
- cross-worktree corruption: **0**;
- unknown runtime silently treated as complete: **0**;
- routine Attempts reaching truthful terminal/recoverable state: target **≥90%**;
- routine dogfood Outcomes reaching Ready for Review without reading provider transcripts: target **≥80%**;
- active human supervision time versus direct single-agent baseline: target **30–50% lower**.

See [`product/kennel-dogfood-acceptance-matrix.md`](product/kennel-dogfood-acceptance-matrix.md).

## Verification commands

From a normalized checkout:

```bash
npm run bootstrap
npm run lint
npm run frontend:typecheck
cd backend && go build ./... && go test ./... && go test -race ./... && go vet ./...
cd ../frontend && npm run typecheck && npm run build
```

When API or storage contracts change, also regenerate and verify `npm run api` / `npm run sqlc`. For user-visible flows, run the real-daemon desktop/browser preview path rather than treating fixtures as runtime proof.

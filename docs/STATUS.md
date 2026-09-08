# Kennel status

**Runtime baseline:** `beta` lineage; current implementation/planning branch `feat/wt3-routing-outcome-first`  
**Architecture authority:** [`product/kennel-v1-product-architecture.md`](product/kennel-v1-product-architecture.md), amended for the MVP by [ADR 0010](adr/0010-outcome-first-control-plane-and-session-subordination.md), [ADR 0011](adr/0011-go-control-plane-and-non-authoritative-intelligence.md), and [`product/2026-09-08-outcome-control-plane-mvp-reset.md`](product/2026-09-08-outcome-control-plane-mvp-reset.md)  
**Current build focus:** working vertical Outcome control-plane MVP

This file separates **implemented runtime truth** from **accepted target architecture**. A design document, Figma frame, partial branch implementation, or plan is not evidence that a feature is shipped.

## Implemented foundation inherited from beta

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

The active first-class execution provider surface for new work is:

- Codex
- Claude Code
- OpenCode
- Cursor
- Pi

Readiness is machine-aware and product selection no longer requires a hidden Codex fallback. Provider **identity** support does not imply every provider has passed every structured-control role; role admission remains capability/conformance driven.

### Canonical Outcome foundation

The repository already carries the durable responsibility lineage:

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

Implemented foundations include:

- immutable Contract revisions and stable criterion identity;
- owner-gated Plan authorization and capability grants;
- real provider Attempts and provider session references;
- recovery/reconciliation facts for the current execution path;
- criterion-bound Evidence;
- explicit verification identity/independence classification;
- user-only Acceptance decisions;
- adaptive intake/callback path;
- durable bounded Project Waldo conversation;
- composed Outcomes with contribution/dependency/proof semantics;
- Outcome-oriented Work surfaces including Understand, Decide & Authorize, Act & Observe/Mission Control, and Prove & Close;
- session/terminal drill-down infrastructure.

### Existing Contract intake behavior worth preserving

The current intake stack already provides:

- exact intent capture before analysis;
- immutable proposal revisions;
- optimistic revision guards;
- material clarification support;
- deterministic offline proposal floor;
- callback/refusal/expiry handling for model-backed proposals;
- owner confirmation before canonical Outcome creation.

This is useful product/control-plane machinery. It is being refactored, not discarded.

## Architectural problem confirmed on 2026-09-08

Despite the correct durable ontology above, the end-to-end experience still leaks inherited Agent Orchestrator/session-first behavior:

- ordinary provider sessions can still act like the primary project/work destination;
- session-first routes and UI remain broadly exposed;
- Project worker/orchestrator configuration still influences launch behavior too directly in parts of the service layer;
- Contract analysis is currently backed by spawning an ordinary `KindWorker` session/worktree;
- Plan proposal is still largely a deterministic direct-WorkUnit construction path rather than a distinct intelligence/review phase;
- legacy task/session entry points can bypass the intended Outcome → Contract → Plan → Mission Control flow.

The architecture correction is accepted in ADR 0010/0011. **It is not yet implemented end-to-end.**

## Current branch work: partial WT3 implementation

`feat/wt3-routing-outcome-first` contains in-progress preference-aware provider/model routing work that must be preserved and integrated into the new Plan authority flow.

Current partial work includes:

- `ExecutionPreference` domain semantics;
- ContractRevision execution preference field;
- immutable `ExecutionBinding` semantics including provider-default / explicit / historical-unbound representation;
- provider-neutral routing domain types/decision skeleton;
- WorkUnit provider/model binding fields;
- routing-decision provenance on PlanRevision;
- RunBrief digest inclusion of provider/model semantics;
- migration `0114_execution_routing_bindings.sql`;
- Contract execution-preference persistence path;
- exact WorkUnit binding/routing provenance persistence path;
- initial execution-preference RED tests.

This work is **not complete or verified**. Known review/integration items include:

- finish service-layer integration around Outcome/Plan authority;
- verify/fix any compile issues from assumed harness helpers;
- ensure explicit model support is candidate/provider-local;
- fix any storage error swallowing in routing-decision reads;
- pass exact model semantics through Attempt spawn;
- prohibit historical provider-only bindings from new execution;
- remove approval-time re-reading of mutable Project worker choice;
- test no-candidate, no-implicit-Codex, model isolation, and role-separation semantics.

Do not describe WT3 as shipped until those paths are implemented and verified.

## Accepted MVP reset — not implemented yet

The next implementation session follows:

- [`product/2026-09-08-outcome-control-plane-mvp-reset.md`](product/2026-09-08-outcome-control-plane-mvp-reset.md)
- [`superpowers/plans/2026-09-08-outcome-control-plane-mvp-reset.md`](superpowers/plans/2026-09-08-outcome-control-plane-mvp-reset.md)

The required user journey is:

```text
Register Project
→ set coordinator/worker preferences
→ capture Outcome
→ Contract intelligence / material clarification
→ owner reviews + confirms Contract
→ Plan intelligence
→ routing + provider/model recommendation
→ owner approves Plan
→ Mission Control
→ exact-bound Attempt/session execution
→ Evidence + Verification
→ owner AcceptanceDecision
```

### Explicit authority rules

- creating/capturing an Outcome does not launch an execution Attempt/session;
- Contract confirmation does not launch execution;
- pre-execution model work is a non-authoritative `IntelligenceRun`, not an Attempt;
- Plan approval is the execution authority boundary;
- Kennel's Go daemon is the orchestrator; models propose or execute bounded authorized work;
- Project coordinator/worker choices are preferences/baselines, not direct execution authority;
- approved WorkUnit provider/model binding is immutable;
- Attempt consumes the approved binding and never silently re-routes from mutable Project preference;
- selecting an Outcome opens its Outcome workspace, not a provider session route;
- terminal/native chat is an explicit Attempt/session drill-down;
- provider completion does not equal Outcome acceptance;
- only the owner/user creates final AcceptanceDecision.

## MVP cut line

The first working product loop **must** include:

1. real Project registration;
2. Outcome capture;
3. intelligent or explicit offline Contract proposal;
4. material clarification path;
5. editable Contract review and owner confirmation;
6. intelligent or explicit offline Plan proposal;
7. preference-aware explainable provider/model routing;
8. explicit Plan approval;
9. truthful Mission Control;
10. serial execution of approved WorkUnits if necessary;
11. exact provider/model Attempt spawn;
12. session/terminal drill-down;
13. Evidence + Verification;
14. owner acceptance/continue decision;
15. restart/reload without duplicate execution;
16. historical session readability.

The MVP does **not** need to block on full parallel WorkUnit DAG scheduling.

## Accepted target after the vertical MVP

The following remain accepted architecture rather than abandoned work:

### Full direct-Outcome WorkUnit DAG

ADR 0008 remains authoritative for bounded dependency graphs inside a direct Outcome.

### WorkspaceLease + dependency scheduler

ADR 0009 remains authoritative for explicit workspace ownership, concurrency budgets, narrower effect fences, and truthful restart reconciliation.

### Truthful parallel Mission Graph

Only expose real parallelism after the scheduler can enforce it.

### Structured receipts and Project continuity

Target continuity remains:

```text
structured provider/workspace facts
→ SessionReceipt
→ WorkUnitReceipt
→ Outcome current brief / ledger
→ governed Project Context candidate
```

### Provider structured-driver conformance

Role admission remains narrower than provider identity support and must be proven through conformance.

### External provider ingress, Island consequence projection, richer Project Brief

These remain later kernel/product slices and do not block the first Outcome-control-plane MVP.

## Direct intelligence API key policy

No external API key is required by architecture alone.

During implementation:

1. first evaluate whether an existing configured provider runtime has a **proven enforced read-only** mode suitable for Contract/Plan intelligence;
2. if not, implement the provider-neutral intelligence port;
3. then choose one direct model API adapter and consult current official API documentation;
4. request/configure the user's key only at that point;
5. keep deterministic/manual fallback available;
6. never persist the key as plaintext canonical Work data or place it in prompts/logs.

A direct API key, when used, powers the intelligence plane. It does not become execution authority or a hidden provider fallback.

## Current implementation order

1. baseline verification + audit partial WT3 branch;
2. durable `IntelligenceRun` domain/persistence;
3. provider-neutral intelligence adapter boundary;
4. Contract-intelligence adapter and intake decoupling from ordinary execution session semantics;
5. Plan intelligence/review;
6. finish WT3 routing and exact binding;
7. Plan approval → exact Attempt spawn;
8. truthful serialized scheduler/Mission Control;
9. Outcome-first sidebar/routes + gate legacy task/session bypasses;
10. Evidence/Verification/Acceptance integration;
11. remove remaining AO authority semantics;
12. full generation/tests/build/restart dogfood;
13. only then resume full DAG/WorkspaceLease parallel scheduler work.

See the detailed implementation plan for file-level tasks and commands.

## MVP acceptance gates

The vertical MVP is judged by falsifiable behavior:

- execution Attempts before Plan approval: **0**;
- automatic Outcome → session-route navigation: **0**;
- hidden provider fallback: **0**;
- implicit Codex selection with no preference: **0**;
- approved WorkUnits silently re-routed after Project preference change: **0**;
- duplicate Attempts caused by restart/retry: **0**;
- provider completion auto-accepting Outcome: **0**;
- unknown runtime silently treated as complete: **0**;
- historical provider-only binding executing as new authorized work: **0**.

The principal real-daemon dogfood scenario and full checklist are in `docs/superpowers/plans/2026-09-08-outcome-control-plane-mvp-reset.md`.

## Verification commands

From a normalized checkout:

```bash
npm run bootstrap
npm run lint
npm run frontend:typecheck
npm run test:foundation
cd backend && go build ./... && go test ./... && go test -race ./... && go vet ./...
cd ../frontend && npm run typecheck && npm run build
```

When API or storage contracts change, regenerate and verify `npm run api` / `npm run sqlc`.

For user-visible flows, run the real-daemon desktop/browser path against an isolated profile/repository rather than treating fixtures or static rendering as runtime proof.

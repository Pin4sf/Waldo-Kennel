# Project Understanding & Advisor intake (Work lane)

- **Date:** 2026-08-23
- **Status:** Proposed design — coordination draft for the Work lane. Planning only; no implementation or merge authorization.
- **Parent plans:** [Work lane parallel execution plan](../plans/2026-08-23-work-lane-parallel-execution.md) (#53), [first Outcome execution handoff](../plans/2026-08-20-first-outcome-execution-handoff.md), [first-slice spec](../../../docs/product/kennel-v0-first-outcome-slice.md), ADR 0004 (parallel lanes)
- **Depends on:** #26 Orchestration Advisor substrate (locked in #53); #21 Understand surface (PR #55)

## Summary

When a Project is registered or selected, Waldo starts a **Project Understanding pass**: a background, cancellable daemon job that builds a workspace + prior-session understanding, then proposes **candidate Outcomes** the owner can adopt into the Understand stage with one click. The pass feeds both first-run onboarding ("adopt your ongoing mess") and the #26 Orchestration Advisor's retrieval layer.

## Decision boundary

### Locked once accepted

1. **Two-layer model.** Layer 1 is deterministic fact harvesting with zero model calls. Layer 2 is advisory digest generation performed **only** by the daemon-owned project orchestrator Chat session — the exact proposer mechanism locked for #26. No new invocation infrastructure.
2. **Candidates are projections, never responsibility.** A candidate Outcome is advisory content until the owner adopts it; adoption prefills the Understand form and the owner saves a real `ContractRevision` through the existing daemon API. Handoff Task 9 rules apply verbatim: minimized origin provenance is recorded at adoption; an inferred task never becomes canonical responsibility by itself.
3. **No new SQLite migrations for understanding state.** The orientation cache is derived data: rebuildable under `~/.kennel` (`<data>/understanding/<projectId>/…`), recomputed to identical results. The migration ledger stays reserved for lifecycle contracts (0100 = #26, 0101 = #31, 0102 = #35).
4. **Transcript reads live only inside Layer 2 advisory calls.** Deterministic Layer 1 facts come from git, the filesystem scan, and Kennel's own durable records (sessions, PRs, checks, change_log activity). Nothing parsed from provider prose is ever persisted as canonical state; digests are cached as advisory content and carry their provenance inputs.
5. **Selection never blocks.** Registering or selecting a project never waits on indexing; results arrive asynchronously and surfaces render truthfully while the pass is pending/running/failed.

### Observed

- Beta already stores durable per-session facts (harness, mode, status lineage, PR/check outcomes) and trigger-emitted CDC — Layer 1 reads these without new collection.
- PR #54 restyled the retired marker intake, confirming visual investment keeps landing on Understand-adjacent surfaces; candidate cards must use the current Figma system tokens.

### Inference

- v0 ships deterministic-only heuristics (Layer 1 + fixed templates) so value lands before any model dependency; Layer 2 turns on behind the same flag machinery as the #26 advisor.

## Layer 1 — deterministic orientation facts

Harvested per project, refreshed on demand and on registration:

| Source | Facts |
| --- | --- |
| Git | branches ahead of default, dirty worktrees, WIP commits, stale/open PRs, recent merge cadence |
| Filesystem | languages, top-level layout, detected build/test commands (package.json / go.mod / Makefile / CI workflows) |
| Kennel records | prior sessions: harness/mode, end states (merged / abandoned / terminated mid-flight), durations, touched paths, check outcomes |
| change_log | activity heatmap over time |

All of it is recomputable; nothing here is lifecycle state, so no CDC obligations attach to the cache.

## Layer 2 — advisory digests

One bounded orchestrator Chat call composes:

- a **workspace digest** (what this repo is, how it builds/tests, conventions worth respecting);
- **3–5 candidate Outcomes**, each citing its evidence chips (e.g. "3 fix-attempts abandoned in payments/", "CI flaky on main since Tuesday", "branch feat/retry-logic untouched 9 days").

Contract rules carried from #53: every skill/tool a candidate's eventual plan cites must resolve against installed registries at validation time — proposals that cannot resolve fail loudly, never silently dropped. Clarification discipline follows the shipped Understand pattern: ask only materially contract-changing questions, recommended answer pre-checked.

## Candidate → Outcome lifecycle

```text
orientation cache ─► candidates (projection, provenance chips)
     │ owner clicks Adopt
     ▼
Understand form prefilled (title/goal/criteria drafts + provenance note)
     │ owner edits & saves
     ▼
POST /api/v1/projects/{id}/outcomes ─► ContractRevision 1 (canonical)
```

Adoption records minimized provenance (sources + timestamps, no transcript bodies). Dismissing a candidate suppresses that candidate; suppression lives in the rebuildable cache.

## Scheduling & failure behavior

- Triggers: project registration, project selection (throttled), manual refresh, contract adoption (re-rank).
- Execution: single-flight per project, cancellable, idle-preferring; bounded runtime; failures leave the previous cache intact and surfaces state "understanding stale as of <time>" instead of pretending freshness.
- Offline daemon: renderer shows its standard offline states; the pass simply does not run locally-side effects.

## Surfaces

1. **Enter:** after registering an existing repo, offer to adopt in-flight work (dirty branches, open PRs, half-finished sessions) as candidate Outcomes.
2. **Understand:** candidate cards above the form with evidence chips and Adopt/dismiss; adopted prefill marks provenance inline.
3. **Contract revisions (later, #26+):** when r(n+1) supersedes r(n), show blast radius — plans/grants/evidence bound to r(n) that are invalidated.
4. **Re-entry pulse (Island/rail adjacency):** per-Outcome one-line state so re-entry stays under the 60-second bar.

## Sequencing

| Step | When | Content |
| --- | --- | --- |
| v0 | immediately after #26 service merges | Layer 1 harvest + template-driven candidates, deterministic only; Enter adoption list; Understand cards |
| v1 | with advisor wiring | Layer 2 digest via orchestrator session; evidence-chip ranking; clarification mining |
| later | #31/#35 adjacency | revision blast radius; re-entry pulse |

Out of scope for this design: Home/OpenLoop capture, Gmail/hosted IDs/durable Memory (separately owned lane), hosted processing, network effects beyond the local ceiling.

## Falsifiers

Stop and revise if any of these occurs:

- a candidate Outcome reaches canonical state without an explicit owner save;
- understanding state leaks into SQLite migrations or canonical CDC;
- selection/registration latency regresses because indexing runs inline;
- transcript-derived content survives anywhere except inside the advisory digest cache;
- cached digests present themselves as fresh after their sources changed without staleness labeling.

## Boundaries carried forward

Push/PR requires per-issue user authorization. Local-first effect ceiling. Provider completion never equals acceptance. Renderer stays thin. The SQLite daemon is the sole canonical writer. No transcript parsing in any canonical path.

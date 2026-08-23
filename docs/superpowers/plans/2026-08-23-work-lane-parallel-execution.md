# Work lane parallel execution plan

- **Date:** 2026-08-23
- **Status:** Coordination authority for the Work lane build-out; planning only, no implementation or merge authorization
- **Parent plans:** [first Outcome execution handoff](2026-08-20-first-outcome-execution-handoff.md), [Kennel build program](../../product/kennel-build-program.md), ADR 0004 (parallel lanes)
- **Supersedes:** sequential-only reading of handoff Tasks 4–8; stage contracts and acceptance criteria unchanged

## Decisions locked going in

1. **v0 completion bar** (unchanged): one real Outcome traverses Enter → Understand → Decide & Authorize → Act & Observe → Prove & Close on a real repository with a real harness. Not multi-worker, not hosted.
2. **Advisor proposer mechanism (#26):** the daemon-owned project orchestrator Chat session proposes; deterministic validator clamps. Zero new invocation infrastructure.
3. **Advisor input assembly decomposes into skills + JIT retrieval** (below).

## Orchestration Advisor v2 (built inside #26)

```text
Frozen ContractRevision
   |  JIT RETRIEVAL LAYER (deterministic, local-only)
   |- Repo orientation scan        (languages, build/test commands, layout)
   |- Installed skills registry    (existing service/chat/skills.go surface)
   |- Harness-advertised tools / MCP servers (provider admission data)
   |- Task archetype templates     (bugfix|feature|refactor|spike -> strategy
      principles: smallest-sufficient unit, verification-first, stop conditions)
   |  ORCHESTRATOR MODEL (project orchestrator Chat session)
   One direct Work Unit proposal citing the skills/tools it intends to use
   |  DETERMINISTIC VALIDATOR
   Capability envelope - budget - evidence requirements - every cited skill
   resolves to an installed versioned skill - placement/worktree rules
   |  PlanRevision + CapabilityGrant + RunBrief core/compiled digests
```

Contract rule introduced here (feeds RunBrief digests and later Learning L3): a proposal referencing a skill/tool that does not resolve fails validation — never silently dropped, never ambient.

## Tracks (concurrent agent runs)

| Track | Runs | Scope | Est. |
| --- | --- | --- | --- |
| **A — spine** | 1 session | #21 Understand UI finish + marker retirement → #26 service + migration 0100 + validator → advisor wiring → #31 executor core → #35 services | 5.5–7d |
| **B — harness/e2e** | 2 sessions | DeepSeek adapter (`adapters/agent/deepseekharness`, template `grok/`) + `dsh/waldo-profile` → lifecycle e2e scaffold + failure-injection fixtures → #38 instrumentation | 1.5–2d |
| **C — support** | 3 sessions | #17 close-out; hygiene closes (#48 fixed by merged #49; #50 superseded/conflicting — Home lane call); JIT-retrieval spike: map chat-skills registry + MCP advertisements onto RunBrief references | 1–1.5d |
| **A′ — surfaces** | on call | DecideAuthorize / ActObserve / ProveClose surfaces, one stage behind Track A DTOs | absorbed |

## Shared-file coordination

Owned exclusively by Track A on merge days: `httpd/controllers/dto.go`, `api.go`, `apispec/specgen/build.go`, `frontend/src/api/schema.ts`, migrations ledger, `routeTree.gen.ts`. Tracks B/C never edit these; B/C PRs rebase onto current beta immediately after each A merge lands. Generated files are regenerated from source; never select a generated side wholesale.

Migration ledger: 0099 consumed by merged #51 → 0100 = #26, 0101 = #31, 0102 = #35 (Work reservation). If beta advances first, renumber the unmerged issue atomically before code edits.

Merge stagger: A merges land on odd half-days; B/C PRs target even half-days.

## Dogfood meta-experiment (start measuring at the #26 service merge)

Register Waldo-Kennel itself as a project. From the moment ApprovePlan ships, file each remaining Work-lane issue as a real Outcome and let the five-stage lifecycle carry it (Codex-first admission; DeepSeek joins when Track B lands and authority gates exist).

Instruments (from the #38 protocol, collected continuously): supervision minutes, transcript opens, material interventions, time-to-state after interruption, false-ready/reopen, unauthorized/duplicate effects.

By #38, the evaluation summarizes accumulated evidence rather than starting a new study — this is the empirical answer to whether Kennel is needed.

## Timeline

| Scenario | Wall clock |
| --- | --- |
| Sequential single-agent baseline | 8–10 days |
| This plan (3 concurrent + 1 on-call) | 3.5–4.5 days optimistic; 5–6 with review overhead |

Risks and fallbacks: advisor proposal quality below bar → tighten archetype templates before touching the invocation path; executor fencing friction → scope stays at one direct WorkUnit (already locked); shared-file contention → A′ absorbs surfaces only after DTOs land, never concurrently.

## Boundaries carried forward

Push/PR requires per-issue user authorization. Local-first effect ceiling. Provider completion never equals acceptance. Renderer stays thin. The SQLite daemon is the sole canonical writer. No transcript parsing in any canonical path.

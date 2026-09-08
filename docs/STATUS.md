# Kennel status

**Source baseline:** merged PR #99, beta `67d6946fdd5e5bba1aca7f7002ba75185e7da998` (2026-09-08). Refresh the SHA before implementation.
**Current objective:** finish the usable Outcome Continuity loop, not rebuild the foundation.
**Execution authority:** [post-PR99 implementation plan](superpowers/plans/2026-09-08-outcome-control-plane-mvp-reset.md).
**Fresh checks:** [baseline verification record](verification/2026-09-08-post-pr99-launch-baseline.md).

## Implemented foundation

Source and automated checks support these implementation claims; they are not end-to-end launch acceptance:

- Go loopback daemon, SQLite canonical writer, additive migrations, trigger CDC/SSE, generated HTTP/TypeScript contracts, thin CLI, Electron/React supervisor.
- Project/session lifecycle, Git worktree/runtime/recovery infrastructure and session/terminal/diff inspection.
- Durable Outcome → ContractRevision → PlanRevision → WorkUnit → Attempt → AgentSessionRef and criterion-bound Evidence/Verification/user Acceptance.
- Immutable Contract/Plan semantics, capability validation, Attempt admission fences and recovery facts.
- Five active execution-provider identities: Codex, Claude Code, OpenCode, Cursor and Pi. Identity is not role/capability conformance.
- PR99: preference-aware routing, persisted approved provider/model binding, historical-unbound rejection, provider-local model semantics, graph validation and serial scheduler decisions. **Actual model launch integration is defective:** Manager.Spawn does not consume the exact-binding resolver; see the reproduced failure below.
- PR99: IntelligenceRun storage (migration 0115), provider-neutral intelligence/LLM ports, direct Anthropic/OpenAI reasoning adapters, model-backed Contract and Plan proposals.
- Existing Understand/Decide/Act/Prove Work surfaces, Project Brief/conversation foundation and composed-Outcome storage remain available to evolve.

## Current policy: ADR0012

Waldo reasoning requires the owner's configured reasoning credential. Configuration is currently environment-based through `KENNEL_WALDO_PROVIDER`, `KENNEL_WALDO_API_KEY`, `KENNEL_WALDO_MODEL`, `KENNEL_WALDO_EFFORT`, with documented vendor-key resolution in `daemon/waldo_reasoning.go`.

There is **no deterministic/offline proposal floor** and no hidden alternate-model fallback. Missing configuration must be recoverable setup failure. The old session-spawn intake/decomposition proposers were removed; model-backed decomposition proposal remains unavailable. Do not reconstruct those retired paths from old plans. Reasoning secrets must not enter canonical Work rows or logs.

ADRs 0010/0011/0012, product architecture and ADRs 0008/0009 govern the target. ADR0012 supersedes older fallback/key-optional wording. Migrations through 0115 are merged and immutable; use the next unused migration number for fixes.

## Confirmed remaining source gaps

| Area | Current gap | Plan slice |
|---|---|---|
| Exact model launch | Manager.Spawn ignores exact model semantics; explicit/provider-default both use mutable Project model in a recording-adapter regression | L1a |
| Runtime authority | Attempt spawn does not carry structured WorkUnit grants/effect policy; non-model Project settings still merge. Live enforcement not proved | L1b |
| Reasoning readiness/recovery | Environment-only setup; nonterminal IntelligenceRun listing has no runtime reconciliation caller; adapter/service behavioral coverage incomplete | L2 |
| Grounding/replan | No repository snapshot in current intelligence request; previous proposal context reduced to title; clarification copied into temporal field; explicit replan and material draft assumptions/blockers need completion | L3 |
| Plan/Mission UI | First-WorkUnit assumptions, redundant mutable harness input, no production schedule HTTP projection | L4 |
| Proof/continuation | UI Outcome-level proof does not satisfy scheduler WorkUnit proof scope; automated check/artifact collection and downstream workspace handoff need integration | L5 |
| Re-entry/navigation | Bounded prior-result context and Outcome-first normal entry paths require real journey verification and cleanup | L6 |
| Release | Packaged installation, live provider enforcement, restart, full proof/acceptance loop and measured performance remain unaccepted | L7 |

The daemon correctly rejects a supplied provider different from the approved binding. The frontend still sends Project preference, so this is a client integration defect, not evidence of silent daemon rerouting.

The serial scheduler currently uses a Project custody fence. Full WorkUnit WorkspaceLease parallel scheduling remains later work. Do not remove that fence merely to make a graph look concurrent.

## Current verification truth

The baseline record distinguishes pass/fail/not-run. Fresh frontend tests have **23 failures in 4 files**, with 2752 passed and 6 skipped. Typecheck passes. Lint reports **190 issues**; details and triage requirements are in the baseline record/plan. The macOS arm64 package build and package identity check pass, but were not launched. Do not treat the whole foundation gate as green. Failures cluster in TaskComposer, NewTaskDialog, Sidebar and SwitchAgentDialog; root causes need classification, not automatic test deletion or restoration of hidden defaults.

No live-model, real-provider permission, packaged Electron journey, or owner-accepted Outcome is claimed by this documentation update. Green service tests do not establish those facts.

## Next work

Start with L0 baseline failure triage and **L1a exact model launch**, then L1b runtime authority from the implementation plan. L2 can be assigned separately only with explicit ownership. Complete L3–L6 in dependency order, then L7 on an integrated SHA. Every slice updates this file with exact observed evidence and remaining limitations.

The release gate remains: real repo → grounded Contract → full Plan approval → exact bounded execution → retained artifacts/checks → understandable proof → owner acceptance/rework, including interruption and restart without duplicate execution.

# Kennel status

**Source baseline:** merged PR #99, beta `0f5def7ce3823487eeab89401f9cd5fd10d26cc2` (2026-09-09 verification).
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
| Exact model launch | Manager.Spawn now resolves the approved exact binding before readiness and TUI/chat launch; live provider conformance and restart/recovery semantics remain unproved | L1b / L7 |
| Runtime authority | Attempt spawn does not carry structured WorkUnit grants/effect policy; non-model Project settings still merge. Live enforcement not proved | L1b |
| Reasoning readiness/recovery | Environment-only setup; nonterminal IntelligenceRun listing has no runtime reconciliation caller; adapter/service behavioral coverage incomplete | L2 |
| Grounding/replan | No repository snapshot in current intelligence request; previous proposal context reduced to title; clarification copied into temporal field; explicit replan and material draft assumptions/blockers need completion | L3 |
| Plan/Mission UI | First-WorkUnit assumptions, redundant mutable harness input, no production schedule HTTP projection | L4 |
| Proof/continuation | UI Outcome-level proof does not satisfy scheduler WorkUnit proof scope; automated check/artifact collection and downstream workspace handoff need integration | L5 |
| Re-entry/navigation | Bounded prior-result context and Outcome-first normal entry paths require real journey verification and cleanup | L6 |
| Release | Packaged installation, live provider enforcement, restart, full proof/acceptance loop and measured performance remain unaccepted | L7 |

The daemon correctly rejects a supplied provider different from the approved binding. The frontend still sends Project preference, so this is a client integration defect, not evidence of silent daemon rerouting.

The serial scheduler currently uses a Project custody fence. Full WorkUnit WorkspaceLease parallel scheduling remains later work. Do not remove that fence merely to make a graph look concurrent.

## L0 baseline cleanup evidence

The four-file frontend RED reproduction failed 23 of 114 tests because fixtures still assumed hidden Codex/default-model selection, while current provider-neutral entry paths require explicit admitted selection. The corrected tests preserve explicit selection, keyboard submission, errors, legacy readability and role admission. Narrow GREEN: 114/114 passed. Full frontend GREEN: 230 files, 2775 passed, 6 skipped. Logs: `/tmp/kennel-l0-frontend-red.log`, `/tmp/kennel-l0-frontend-green.log`, `/tmp/kennel-l0-frontend-full.log`.

The Go lint baseline fell from 190 findings to zero unexplained findings. L1a wires the retained `prepareSpawnExecution` helper at the Manager.Spawn boundary and removes `normalizedExactModel` only after proving it had no callers; no lint rule was disabled or suppressed. The final full-lint output is `/tmp/kennel-l1a-lint-final.log`. The first `npm run lint` attempt exposed an L0-introduced stale generated-contract failure after the `AgentRoles` → `Roles` rename; `npm run api` regenerated both contracts and HTTP/spec parity now passes. The initial failing wrapper output is `/tmp/kennel-l0-lint-command-final.log`; regeneration output is `/tmp/kennel-l0-api-regenerate.log` and parity output is `/tmp/kennel-l0-api-parity.log`.

Full Go package tests pass after generated-contract repair; frontend typecheck passes (`/tmp/kennel-l1a-frontend-typecheck.log`). L0 changes no SQL schema or migration and does not change provider runtime behavior; generated OpenAPI/TypeScript contracts were updated to reflect the internal `Roles` name. L1a changes only request-local launch configuration: explicit approved models reach the adapter unchanged, provider-default semantics clear mutable Project model values, and persisted Project configuration remains unchanged. Manager-boundary TUI and Chat tests pass, including the provider-default case. Targeted tests, race tests and vet output are recorded in `/tmp/kennel-l1a-go-targeted.log`, `/tmp/kennel-l1a-go-race.log` and `/tmp/kennel-l1a-go-vet.log`; RED/GREEN launch regression output is in `/tmp/kennel-l1a-manager-red.log`, `/tmp/kennel-l1a-manager-green.log` and `/tmp/kennel-l1a-manager-green-expanded.log`.

Fresh verification on this branch also passed `go test -race ./...` (exit 0), `npm run sqlc`, `npm run api`, and generated-contract parity (exit 0). `npm run test:foundation` reached the shared cloud-client check after provenance and backend gates passed, then stopped at the pre-existing generated cloud schema drift (`Kennel` versus `AO`, exit 1); no unrelated cloud contract was changed.

## Current verification truth

The baseline record distinguishes pass/fail/not-run. The historical fresh run had **23 failures in 4 files**; L0 now has no frontend test failures. The full frontend suite is green at 230 files, 2775 passed and 6 skipped; frontend typecheck, HTTP/spec parity and full lint pass. Full backend race coverage also passes. The earlier full `npm run lint` wrapper failure was caused by L0’s stale generated contracts and is repaired. The foundation wrapper remains blocked by the unrelated cloud-client generated-schema drift described above. L1a’s exact-binding regression is green at the Manager launch boundary for both TUI and Chat, with targeted race and vet checks passing. The macOS arm64 package build and package identity check pass, but were not launched. Do not treat the whole foundation or provider conformance gate as green.

No live-model, real-provider permission, packaged Electron journey, or owner-accepted Outcome is claimed by this documentation update. Green service tests do not establish those facts.

## Next work

Next is **L1b governed capabilities/replay** from the implementation plan. L1a still needs live provider conformance and restart/recovery verification before those claims can be accepted. L2 can be assigned separately only with explicit ownership. Complete L3–L6 in dependency order, then L7 on an integrated SHA. Every slice updates this file with exact observed evidence and remaining limitations.

The release gate remains: real repo → grounded Contract → full Plan approval → exact bounded execution → retained artifacts/checks → understandable proof → owner acceptance/rework, including interruption and restart without duplicate execution.

# Kennel status

Reviewed against `beta` revision `b9f69d949` on 2026-09-12. The latest runtime
checkpoint is PR #110, integrated at `9e45f99de034572a7ad6d77148705e16c1bbe95a`;
its tested head is `5a58434ad275a7f5078c6d50e7f714651e8d0907`.
This page separates implementation, recorded verification and remaining launch
work. It does not claim a release or owner Acceptance.

## Current implementation

- Go loopback daemon and SQLite canonical state with additive migrations,
  trigger-backed changes and generated HTTP/TypeScript contracts; thin CLI and
  Electron/React supervisor.
- Outcome, immutable Contract/Plan, bounded WorkUnit dependency graph, Attempt,
  Session, Evidence, Verification and explicit user Acceptance foundations.
- Owner-configured OpenAI/Anthropic reasoning and native Codex packet-mode
  reasoning. Contract-bound planning uses bounded disclosed context and creates
  no execution authority or Attempt. Missing configuration fails explicitly;
  there is no offline proposal floor or hidden provider fallback.
- Approved provider/model bindings, capability admission, serial scheduling and
  restart/replay fences. The normal frontend Start request uses the approved
  Plan identity and request key, not a mutable Project provider preference.
- Work is the default destination. Plan/graph views and attached Attempt
  supervision expose daemon facts; provider Sessions remain technical detail.
  Island startup is opt-in with `KENNEL_ENABLE_ISLAND=1`.
- Codex, Claude Code, OpenCode, Cursor and Pi are active execution-provider
  identities; this does not establish every role's live conformance.

The scheduler has concurrency **1** behind a Project custody fence. WorkUnit
WorkspaceLeases and safe parallel execution remain later work under ADR 0009.
Retained-artifact, governed-check, supplied-document, proof and handoff primitives
exist, but complete integration and live closure are still open below.

## Recorded launch verification

The checkpoint includes direct OpenAI/Anthropic reasoning configuration,
Contract-bound interactive planning, and native Codex read-only packet-mode
reasoning. It does not silently fall back between providers. Repository-context
settings have strict PATCH/persistence semantics and an advanced Settings UI;
settings-read failures fail closed. Generated RunBriefs carry exact Contract
criteria, approved check argv, review command and bounded context requirements.

The packaged macOS application and package identity passed. In an isolated
profile, a real Codex Attempt initialized through Codex App Server, found the
packaged sidecar/hook path, emitted Kennel activity hooks, exited zero and
reconciled. Plan and graph are separate views; Execution shows one Work graph
followed by Attempt lineage, correct provider branding, current-session Engage,
exact attention reasons and replacement controls. This closes the historical
tested-path blocker where a symlinked Codex CLI could not find
`codex-code-mode-host`. It does not establish native tool/plugin/approval parity
or provider conformance across every installation.

The real Attempt returned `needs_you` and produced no report because the
approved WorkUnit exposed its validator command but did not derive a separate
repository-inspection capability/tool affordance. Provider exit and Attempt
reconciliation therefore prove launch mechanics only; no artifact,
WorkUnit-scoped Verification or `AcceptanceDecision` was created.

The [PR #110 handoff](handoffs/2026-09-12-pr110-launch-fixes/HANDOFF.md)
and [execution ledger](handoffs/2026-09-12-pr110-launch-fixes/EXECUTION-LEDGER.md)
retain exact command scopes and live-run provenance. These are recorded results
at the tested revision, not checks rerun by this documentation cleanup.

Earlier scoped evidence remains in the
[Wednesday follow-up](verification/2026-09-09-wednesday-followup.md),
[completion ledger](superpowers/plans/2026-09-10-luna-kennel-work-completion.md),
and [launch usability record](verification/2026-09-11-launch-usability-iteration.md).
Their historical missing-host, branch-integration and unlaunched-package notes
must not override the later PR #110 canary. Full provider permission, recovery,
packaged journey and owner-acceptance gates remain distinct.

## Remaining launch and roadmap gaps

| Priority | Area | Current gap |
| --- | --- | --- |
| 1 | WorkUnit capability/tool delivery | Derive and deliver bounded repository inspection/tool affordances in addition to the validator command; the verified Attempt returned `needs_you` without authoring its report. |
| 2 | Autonomous proof and closure | Prove a fresh Outcome through artifact production, governed checks, WorkUnit-scoped Evidence/Verification and separate owner Acceptance or rework. |
| 3 | Evidence and Result experience | Automate artifact/check collection, complete WorkUnit-scoped proof and result review, and materialize retained outputs for downstream WorkUnits. |
| 4 | Mission Control | Complete the direct WorkUnit DAG projection and integrated Board/List navigation while retaining the session Kanban beneath the graph. |
| 5 | Parallel scheduling | Replace the intentional concurrency-`1` Project fence only after durable WorkspaceLease, dependency, integration, recovery and cleanup gates prove safe. |
| 6 | Session continuity | Decide and implement historical-session inspection or engagement beyond the currently engageable active session. |
| 7 | Release/update | Publish and test actual install/update artifacts. Package identity is verified, but the updater reports no published GitHub versions. |
| 8 | Remaining manual acceptance | Complete applicable mobile rendering and manually verify reduced-motion behavior. |

General non-repository Outcomes, supplied-document wiring and durable delivery
also need their own integration/verification evidence; repository planning alone
does not establish those paths. Model-backed decomposition proposals remain
outside the current reasoning surface.

The launch gate is: real repository → grounded Contract → approved Plan →
bounded execution → retained artifact and governed checks → understandable
proof → owner acceptance or rework, including restart without duplicate work.
A zero-exit provider session is only one part of this journey.

## Public release readiness

There are no published release artifacts at this checkpoint. Installation/update
publication, CI enforcement and private security reporting still need maintainer
work; follow the [launch checklist](../ROADMAP.md#public-release-readiness) and
[open issues](https://github.com/Pin4sf/Waldo-Kennel/issues).
The [roadmap](../ROADMAP.md) defines later milestones. Contributors start from
current `beta`; maintainers promote tested work to `main` separately.

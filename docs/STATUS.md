# Kennel status

Reviewed against `origin/beta` revision `f2132a83c6a09145cd38234d55097128815de432`
on 2026-09-12. This local integration candidate combines the completed Issue
#115 work with the shell/onboarding/Mission UI lane; it is not pushed, merged,
deployed or released.
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
- Codex governed Attempts receive a private, frozen-policy repository surface:
  bounded listing/text reads, scoped text writes when granted, and execution of
  exact approved check IDs. The allocated workspace root is frozen into the
  admission snapshot; provider-native execution remains read-only, while a
  required positively allowlisted Kennel MCP carries the narrow write/check
  authority. Generic shell, unified-exec, web and plugin surfaces are disabled;
  unsupported policy shapes and unverified App Server injection fail closed.
- Work is the default destination. Plan/graph views and attached Attempt
  supervision expose daemon facts; provider Sessions remain technical detail.
  Kennel Island starts with the desktop when the display supports it; its
  persisted visibility preference remains owner-controlled in Settings.
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

The Issue #115 packaged canary closed that repository-affordance blocker for
Codex. A real approved Outcome repaired `report.md` only in its leased worktree,
then ran the exact frozen SHA-256 check through Kennel. The original checkout
remained unchanged. A prelaunch workspace failure held custody until explicit
replacement, and restart retained the two historical Attempts without replaying
the succeeded provider session. The then-current pre-repair packaged daemon was
re-probed under native read-only Codex: it repaired the disposable report, passed the
same exact check, and refused mixed-case `.GIT` custody and traversal writes.
Exact IDs, hashes and refusal evidence are in
the [Issue #115 verification record](verification/2026-09-12-issue-115-governed-repository-tools.md).
This proves the bounded execution slice, not retained-artifact automation,
WorkUnit-scoped Verification or an owner `AcceptanceDecision`.

Post-review correctness repairs now preserve executable modes across governed
text replacement and durably fence write/check effects when approved-check
termination is unknown, including private-server restart and Outcome recovery
before ordinary liveness observation. One final-code pre-remote Attempt remains
unconfirmed because the environment refused the explicit owner assertion needed
to replace it; that case was not bypassed.

An independent corrected-code canary then started with a resolvable local
remote before execution. The real packaged Electron UI imported the disposable
repository, selected Codex, showed the Outcome in progress, and later rendered
it `Ready for review`. One Codex Attempt modified only its retained leased
worktree, passed the exact approved check under the macOS seatbelt runner, and
created canonical deterministic Evidence and Verification. Restart with the
same isolated profile/data preserved exactly one succeeded Attempt and one
terminated session without automatic duplication. Post-restart session
inspection subsequently exposed a restore/duplicate-session error and app-exit
disposal warning; those observations were left open at that checkpoint and are
not folded into Issue #115 completion. No owner Acceptance was created.

A [post-canary lifecycle follow-up](verification/2026-09-12-post-canary-lifecycle-and-issue-35-delta.md)
now blocks manual, resume, and startup restoration of a terminal governed
Attempt and guards destroyed-window composition disposal. Full Go tests,
focused Electron lifecycle tests, typecheck, isolated-cache lint, and a fresh
package build pass. That package has not been launched, so the UI-driven quit
and restart behavior remains runtime-unverified. The follow-up also records the
exact Issue #35 delta, including the still-open duplicate check invocation
between the provider tool and terminal reconciliation.

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
| 1 | Autonomous proof and closure | Carry the produced artifact and governed-check result into WorkUnit-scoped Evidence/Verification, then leave the separate owner Acceptance or rework decision explicit. |
| 2 | Evidence and Result experience | Automate artifact/check collection, complete WorkUnit-scoped proof and result review, and materialize retained outputs for downstream WorkUnits. |
| 3 | Mission Control | Complete the direct WorkUnit DAG projection and integrated Board/List navigation while retaining the session Kanban beneath the graph. |
| 4 | Parallel scheduling | Replace the intentional concurrency-`1` Project fence only after durable WorkspaceLease, dependency, integration, recovery and cleanup gates prove safe. |
| 5 | Session continuity | Decide and implement historical-session inspection or engagement beyond the currently engageable active session. |
| 6 | Release/update | Publish and test actual install/update artifacts. Package identity is verified, but the updater reports no published GitHub versions. |
| 7 | Remaining manual acceptance | Complete applicable mobile rendering and manually verify reduced-motion behavior. |

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

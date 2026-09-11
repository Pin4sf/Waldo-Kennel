# Launch integration checkpoint

This report covers the isolated `codex/launch-integration-20260911` branch, based on freshly fetched `origin/beta` commit `88f63f040e843cf9dbf3d87cd01e5943f4c70b81`. It is not release or Outcome acceptance.

## Integrated scope

- Known prelaunch admission failures terminate the failed Attempt and release its fence; ambiguous provider-start failures retain recovery boundaries.
- Admission blockers survive restart and the failure-record/run-intent crash window.
- Supervisor sockets use bounded paths and packaged daemon attachment checks process build identity.
- WorkUnit/Attempt engagement selects the attributed session and renders its actual chat or terminal mode.
- Contract-bound planning conversations use explicitly selected direct-API reasoning, bounded repository or approved-document context, durable turns, and proposed Plans. Plan approval remains separate from execution.
- Planning review repairs cover request replay, crash interruption, governed deletion, Contract revision races, explicit model provenance, and supplied-document isolation.

## Verification provenance

- Execution-only checkpoint `447b7e9e4140b18457e0b2823299668d3cecbbd4`: full Go build/vet/test/race, frontend typecheck, 2,760 frontend tests (6 skipped), Electron package, generated contract parity, and lint passed across the recorded base run and four-file lint followup.
- Combined backend checkpoint `7a9082b8ecabb1f0b0a435edb7f8c7d6e385b99b`: `npm run sqlc` and `npm run api` passed with no generated diff.
- `npm run bootstrap` passed in the isolated integration worktree.
- `npx @redwoodjs/agent-ci run --all` exited zero but reported no relevant workflows. This is not evidence that aggregate CI checks ran.
- Backend checkpoint `525ebe4b94c658fbcf1490396a4b214694b8db36`: full Go build/vet/test/race and lint passed. Cancellation interrupts cooperative requests and prevents late results publishing; focused cancellation race tests also passed.
- Planning UI integrated at `b637a4915ed3dcb83a192ff48a8b083bfc5278e1` after 41 focused tests and typecheck passed. Review corrections cover Contract-scoped caches, cancellation, retry revisions, stale responses, candidate errors, and CDC refresh. Legacy proposal controls were removed from this Mission flow.
- Live-browser verification found demo terminal output rendered against real daemon facts. `a44af3525442d9b1616206abb3afedcd463f616e` makes that preview explicitly require desktop terminal access. Terminal/i18n tests (51) and typecheck passed; the corrected message was observed through CUA.
- The [real Codex canary](2026-09-11-codex-serial-canary.md) proved binding, serial dependency checks, two successful retained results, and no duplicate Attempt after restart. It required a corrected local executable/tmux environment, explicit recovery, and manual termination of completed TUI sessions. It is not a zero-intervention orchestration pass.
- Final source checkpoint `a44af3525442d9b1616206abb3afedcd463f616e`: full frontend typecheck passed; Vitest passed 230 files / 2,768 tests, with 6 skipped; fresh Electron package and `package:identity` passed. Final docs-only commits do not change that tested source.
- Packaged desktop launched with independent `KENNEL_ELECTRON_DATA_DIR` against the disposable profile and started its own daemon (runfile owner `app`, PID 94888 at observation). The initial window appeared blank, then rendered onboarding after refresh. Native automation subsequently returned inconsistent state and `noWindowsAvailable`; actual desktop terminal input/output/resize was not verified. Browser terminal fixtures are explicitly excluded from that acceptance claim.

## Remaining acceptance boundaries

Native Codex conversational planning remains unavailable; the direct-API path is explicit and does not grant native filesystem/tool access. Five provider identities do not imply five-provider conformance. Serial scheduling is the agreed launch scope.

The new UI currently offers repository-context planning only; the approved-document backend path is not offered until document availability can be projected reliably. Live planning dialogue was not exercised in the disposable profile because no reasoning provider was configured; its unavailable state was verified visually.

Real session binding, two criterion checks, and restart idempotency have the bounded canary evidence above. Live desktop terminal interaction, unassisted provider completion, a live planning conversation, broader proof discrimination, and delivery acceptance remain open. Provider completion and green checks do not accept an Outcome. The historical queued/unconfirmed user Attempt remains untouched; reconciliation requires truthful stop evidence or the owner's assertion.

## Follow-up: onboarding and accessible reasoning settings

- `4be9f994550c746ef24d9bdc20450b08ccc17131` removes the Session alerts onboarding test. The previous screen declared an alert sent immediately after a notification request, without delivery confirmation. Actual notification functionality is unchanged. Eight focused tests and typecheck passed.
- `72443a2bc5e767cd4ff5b1e05592f483463882d6` exposes Settings in the Work sidebar and adds a dedicated AI providers & API keys section. The Work layout previously hid the entire sidebar footer. Ninety-nine focused tests and typecheck passed; the sidebar entry and dedicated settings page were visually verified in the live-daemon browser at desktop width.
- A fresh Settings-enabled desktop package and package identity checks passed at `72443a2bc5e767cd4ff5b1e05592f483463882d6`, in the separate locked `/private/tmp/kennel-onboarding-package-20260911` worktree. The original running package was not overwritten. Logs: `/private/tmp/kennel-settings-package-build.log` and `/private/tmp/kennel-settings-package-identity.log`.
- Packaged updater configuration names `Pin4sf/Waldo-Kennel` and `kennel-updater`; it is not configured against the old Agent Orchestrator repository. Release download and installation were not exercised.
- Agent Orchestrator was imported through the daemon project API from `/Users/shivanshfulper/Developer/Pin4sf/agent-orchestrator`, observed clean at `e01c02eeef1be7b5a7d48f916919a16ae3770740`. The user selected a read-only architecture and gap assessment. No assessment Outcome or provider execution has yet been started; native reasoning reports `REASONING_NOT_READY`, and direct-API setup awaits the user's private key entry. The repository remains unmodified.

## Saved state

Active worktrees remain locked. Local recovery bundles and dirty-file snapshots are under `/Users/shivanshfulper/Developer/Pin4sf/kennel-launch-safety/20260911T155523Z`; ownership is recorded in the adjacent `ACTIVE-WORK.md`. The primary checkout and unrelated worktrees are preserved. No push, merge, deployment, or owner AcceptanceDecision is performed by this checkpoint.

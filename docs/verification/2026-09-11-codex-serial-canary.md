# Real-daemon Codex two-node serial canary

Date: 2026-09-11

## Scope and identities

- Integration source commit used to build the canary binary: `7a9082b8ecabb1f0b0a435edb7f8c7d6e385b99b`
- Binary: `/private/tmp/kennel-live-canary.Ot5oEU/kennel`
- Binary SHA-256: `6b837ed76f426f5fa1a5b88e7b8c5f9685b33287713dd9daf845c5d513d8a084`
- Binary module: `github.com/Pin4sf/Waldo-Kennel/backend/cmd/kennel`, built with Go `1.26.4`, `darwin/arm64`, `CGO_ENABLED=1`, `-trimpath`
- Codex provider: bundled authenticated Codex CLI `0.153.4`; provider reported effective model `gpt-6-astra medium`
- Disposable project: `/private/tmp/kennel-live-canary.Ot5oEU/project`
- Disposable daemon data/profile: `/private/tmp/kennel-live-canary.Ot5oEU/data`
- Run file: `/private/tmp/kennel-live-canary.Ot5oEU/running.json`
- Daemon API: `http://127.0.0.1:43861`
- Outcome: `out-3ebba19b-1880-451d-83fc-4f1ee39e71b6`
- Plan: `plan-serial-canary`
- Plan core digest: `f3c4042a30889dae843e71437a21e3b788251688a436b7d075fed17942284ba8`

No integration source file was edited for this canary. The Plan was seeded through the production domain and SQLite store APIs using a temporary Go overlay while the daemon was stopped. No user/live Outcome was touched. No acceptance decision was created.

## Result

The corrected two-node run passed the real daemon/provider boundary:

1. Before launch, `wu-serial-a` was `runnable`; `wu-serial-b` was `blocked` with `awaiting_dependency_proof`; zero Attempts existed.
2. Approval and run intent created a real Attempt and immutable Codex AgentSessionRef.
3. After the environment correction described below, `wu-serial-a` ran in Codex session `serial-canary-2`, wrote `first.txt`, and its approved `grep -Fx first complete first.txt` check passed under `kennel-governed-check/macos-seatbelt-workspace-write`.
4. Attempt `att-8c089bf4-ab2d-4771-aa6b-2e8a5904f15f` was classified `succeeded` at `2026-09-11T17:44:34Z`.
5. Only after A was proved did Kennel automatically create node B Attempt `att-a8bd9b2e-09f1-4861-8183-4b64ceca34f2` at `2026-09-11T17:45:19Z`, bound to Codex session `serial-canary-3`.
6. Node B read retained `first.txt`, wrote `second.txt`, and its approved `grep -Fx 'second saw: first complete' second.txt` check passed under the same governed checker.
7. Attempt B was classified `succeeded` at `2026-09-11T17:46:04Z`.
8. The final daemon restart retained exactly three historical Attempts: one explicitly cancelled pre-execution configuration failure plus the two succeeded WorkUnits. It created no fourth/duplicate Attempt.
9. Final run state is `ready_for_review`; proof is `ready_for_acceptance`; `2/2` criteria are proven; acceptance decisions count is `0`.

Retained node B worktree content:

- `first.txt`: 14 bytes, SHA-256 `3fc6141b63558d5215824903891264e3191c0586716201643c64205de8034c0b`, exact bytes `first complete`
- `second.txt`: 26 bytes, SHA-256 `2f07d920457b4e25633d6df87f49921f311901d6d32eed909ee0456016a7fa52`, exact bytes `second saw: first complete`

## First-attempt configuration failure and recovery

The first real Codex Attempt (`att-cf036fce-b06b-4d69-be4f-3f33c41bdd0a`, session `serial-canary-1`) reached the provider but made no worktree edit. Codex reported:

> Code Mode is unavailable because failed to spawn code-mode host /Users/shivanshfulper/.local/bin/codex-code-mode-host: host executable was not found.

Its hooks also logged that the custom-runfile daemon was not discoverable through the pre-existing shared tmux server. The Attempt was cancelled through the run API, producing an `owner_cancelled` observation with `providerStopped=true` and `workspaceFreed=true`. The durably terminated predecessor was then reconciled through the recovery API, which issued replacement receipt `rcpt-c8ac2392-9ebb-4e8e-aa8f-7f9a8cdaf38e` and safely released custody.

The retry used the authenticated bundled `/Applications/ChatGPT.app/Contents/Resources/codex`, which has its adjacent code-mode host, plus a private disposable tmux server so `KENNEL_RUN_FILE` reached hooks without changing user state.

## Manual interventions versus automation

- Automatic: initial run intent admitted the first Attempt; after the corrected A result was classified and the prior run error was cleared, the resumed run automatically admitted B only after A proof.
- Manual: cancelled the blocked first Attempt; reconciled its durably terminated custody; explicitly started replacement A; terminated each interactive TUI session after Codex reported `KENNEL_WORK_STATUS: ready_to_merge`, enabling daemon liveness reconciliation and approved-check classification; paused/resumed the run once to clear the prior sticky `ATTEMPT_FENCE_HELD` run-intent error before B continuation.
- Therefore this proves real provider startup, bindings, dependency gating, retained-artifact checks, classification, and restart idempotency. It does not prove a zero-intervention end-to-end run from the initially misconfigured local environment.

## Final live state and safe handoff

- Final daemon PID: see `post-restart-health.json` (currently `86078` at report creation).
- No provider session remains live; both successful Codex TUIs were terminated after completion.
- No Attempt is active; both execution Attempts are `succeeded`.
- It is safe for the frontend verifier to stop only PID from `/private/tmp/kennel-live-canary.Ot5oEU/running.json`, then launch the packaged app against the same disposable data and runfile paths with an independent Electron `userData` directory.
- Do not delete the data directory or record acceptance; the current `ready_for_review` / `ready_for_acceptance` state is the desired UI fixture.

Suggested Work route for the live renderer:

`/work?project=serial-canary&stage=prove_close&outcome=out-3ebba19b-1880-451d-83fc-4f1ee39e71b6`

## Evidence files

- `post-restart-health.json`
- `post-restart-attempts.json`
- `post-restart-run.json`
- `post-restart-proof.json`
- `post-restart-schedule.json`
- `post-restart-sessions.json`
- `daemon-1.log`, `daemon-2.log`, `daemon-3.log`, `daemon-4.log`
- `/private/tmp/kennel-live-canary.Ot5oEU/data/hooks.log`

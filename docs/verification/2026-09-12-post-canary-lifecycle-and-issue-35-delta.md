# Post-canary lifecycle findings and Issue #35 delta

Date: 2026-09-12

Scope: source, actual-boundary regression, and unlaunched package-build work.
No Electron process was launched or stopped for this follow-up, no live profile
was changed, and no owner containment assertion or AcceptanceDecision was made.

## Governed session restore

The stale accessibility action recorded in the Issue #115 canary was an
accidental trigger, but it exposed a real authority defect. `RestoreWithMode`
accepted any terminated session with a restorable workspace. It then reloaded
the frozen Attempt policy and relaunched the provider without checking whether
the owning Attempt was already terminal. `ResumeAgentWithMode` had the same
gap for an exited provider in a still-live session.

The Manager boundary now resolves the exact Attempt from the durable session
reference and refuses restore or resume when its status is terminal. The check
runs before workspace restoration or runtime creation, including the separate
startup `RestoreAll` path. Startup clears the consumed shutdown marker for the
closed Attempt without deleting retained artifacts or workspace data. A normal
ungoverned session has no Attempt execution binding and retains the existing
restore and resume behavior. Missing or unreadable governed lineage continues
to fail closed.

Red evidence before the fix:

```text
TestRestoreGovernedSessionRejectsTerminalAttemptBeforeWorkspaceOrRuntimeMutation
  RestoreWithMode error = <nil>, want ErrGovernedAttemptClosed
TestResumeGovernedSessionRejectsTerminalAttemptBeforeRuntimeMutation
  ResumeAgentWithMode error = <nil>, want ErrGovernedAttemptClosed
TestRestoreAllSkipsTerminalGovernedAttemptBeforeWorkspaceOrRuntimeMutation
  startup restored terminal Attempt execution: workspace restores=1 runtime creates=1
```

The focused Manager tests, the complete `session_manager` and session-service
packages, and the API error mapping pass after the fix. The API reports
`GOVERNED_ATTEMPT_CLOSED`; rework or retry must create a new Attempt rather
than reopen the old provider execution.

## Electron shutdown disposal

The packaged warning stack's second frame maps to the `mainWindow` `closed`
callback that calls `composition.dispose()`. The corresponding bundled
`dispose` called `mainWindow.contentView.removeListener(...)` outside either a
destroyed-window guard or a `try` block. Electron had already destroyed the
`BaseWindow` content hierarchy, so that first call threw `TypeError: Object has
been destroyed`; the promise returned by `.finally(...)` then had no rejection
owner and produced the warning.

Window-composition disposal is now idempotent and returns immediately once the
window is destroyed. Live-window removal and close operations remain
best-effort. A regression that makes `contentView.removeListener` throw the
real Electron error failed before the fix and passes after it. The relevant
window-composition, browser-view-host, and agent-browser-runtime suites report
76 passed and 1 skipped.

The previously recorded canary package embeds code revision
`3192b27ea4fa659edde5320d5077755d1ab5dd6d`; it does not contain this follow-up.
A fresh package build from immutable code revision
`f7d57d80c3229053d1f3ce96ab398b7d172fca2d` passed. Its daemon SHA-256 is
`1bc6f48d08ff7eccbffa388b09d055a4cc792c68be0ec635b7d732d39a9136fa`
and its `app.asar` SHA-256 is
`5ef99c5e0368fa32655316ba5670319257fab753ac33264f25644bbbb2a7460b`.
The package was not launched; a UI-driven quit is still separate runtime
verification owned by the desktop test lane.

## Exact Issue #35 delta from the successful canary

Issue #35 remained open when refreshed from GitHub on 2026-09-12. The canary
does establish more than the older roadmap wording, but it does not satisfy the
whole Result/rework outcome.

### What was initiated manually or through the loopback API

- The disposable Project was imported and Codex was selected through the UI.
- Contract/Plan setup, approval, and Attempt start were owner/operator actions;
  Contract and Plan creation used the loopback API for reproducibility.
- The provider chose to invoke the granted `run_approved_check` tool during its
  authorized turn. This was not an owner Evidence or Verification API call.
- No owner AcceptanceDecision was created.

### What the daemon did automatically

- The liveness reconciler moved the ended provider execution to `reconciled`.
- The artifact retainer read the exact bound session workspace, retained its
  changed-file manifest and bytes, and persisted artifact version
  `937a09a99404232b6e7e530aae24a8a8bd6784a2b56ffd11711a859a343902ea`.
- Outcome reconciliation reserved and ran the approved `cmp source.txt
  report.md` check against that retained Attempt workspace, including the
  known-wrong baseline.
- It persisted one canonical observed check run, then deterministically created
  one criterion-scoped supporting EvidenceItem and one passed deterministic
  VerificationRun. Their request keys bind Attempt, artifact version, and check
  identity, so restart reconciliation replays rather than duplicates them.
- The success finalizer atomically changed the Attempt from `reconciled` to
  `succeeded`, froze the receipt, recorded classification, and released
  custody. The UI derived `Ready for review` from those daemon facts.

There is an important duplication gap: the provider-facing
`run_approved_check` server records only the uncertainty fence and returns the
command result to the provider. It does not write `attempt_check_runs`.
Terminal reconciliation therefore reserves and executes the same approved
check again to create canonical proof. The database's single observed row does
not mean the command ran only once. Issue #35 should converge these into one
durable invocation path, or stop asking the provider to invoke a check that the
daemon will independently own after termination.

### Remaining acceptance delta

| Issue #35 criterion | Current evidence | Remaining gap |
| --- | --- | --- |
| Automatic artifact/check collection with provenance | One real Codex/one-WorkUnit canary automatically retained exact-workspace bytes and created a canonical check observation. | Check execution is duplicated across the provider tool and terminal reconciler. The check row records command/result/timing/sandbox/artifact identity, but not a complete explicit environment and provider-generation provenance packet. Other workspace/provider shapes remain unverified live. |
| Criterion/WorkUnit-scoped support, contradiction and inconclusive proof | Backend derives all three verdict classes, binds proof to criterion + Attempt + artifact version, and uses deterministic replay keys. The canary proved the supporting/passed path. | Failed, inconclusive, contradictory, correction, and replay-conflict paths have test evidence but not the complete live Result journey. WorkUnit lineage is resolved through the Attempt rather than presented clearly to the owner. |
| Understandable Outcome Result | Mission state and the Prove & Close surface expose proof rows and owner decisions. | There is no cohesive Result summary of changed artifacts, checks, uncertainty, omissions, and next safe action. The current surface is proof-entry oriented and exposes raw re-entry IDs for non-Contract targets. |
| Retained upstream outputs for dependent WorkUnits | Backend admission resolves frozen predecessor receipts, verifies blobs/modes/digests, and materializes them before successor launch. | No real two-WorkUnit provider canary has proved A's retained output reaches B exactly once across restart. |
| Reject/rework/reopen and restart safety | Durable correction/acceptance horizons, run halting, replacement Attempts, and restart-idempotent proof exist in backend tests. This follow-up prevents a terminal governed session from silently rerunning the old Attempt. | The normal owner-facing Result-to-rework flow, bounded continuation packet, and live restart/rework journey remain unverified and the non-Contract target UX still requires raw identity. |
| Daemon-derived Ready for Review; owner-only Acceptance | The successful canary reached Ready for review from retained proof, survived restart without a duplicate Attempt, and created zero Acceptance decisions. | Broader negative/live cases and the full owner review/rework/accept journey are not yet accepted. Provider/check completion remains evidence, never Acceptance. |

Therefore the successful canary is real evidence for automatic retention,
canonical deterministic proof, success classification, and restart
idempotency on one bounded path. It is not Issue #35 completion: the Result UX,
single-owner check invocation, complete provenance, live multi-unit handoff,
and ordinary rework/reopen journey remain open.

# Lane W implementation notes: #31 Act & Observe — for the Codex lane

- Status: pushed on `work/31-act-observe`, open as **PR #68** (round-3 remediation applied; merge blocked pending re-review)
- Date: 2026-08-25
- Base: `origin/beta` @ `b33e41a96`; commit count: see `git rev-list --count b33e41a96..HEAD` (authoritative — earlier prose counts were wrong once)
- Reader: Codex lane owners of Slice B / #32 / #40 and reviewers of the #31 PR

## 1. Custody law (v2 — the part to internalize)

A fence over a worktree subject may be released ONLY when the bound
provider's stop is PROVEN:

1. the bound session is durably terminated (`sessions.is_terminated`);
2. or the owner explicitly asserts containment in the recovery request
   (`confirmProviderStopped: true`) — recorded as an `owner_contained`
   observation so the assertion stays auditable.

Stored status alone — including `failed` — is NEVER proof: after-the-fact we
cannot know where a spawn actually failed, so no #31 path writes `failed` at
all any more (the status stays reserved in the schema). Unproven reconcile/
replace escalates as needs_attention and returns typed
`ATTEMPT_CUSTODY_UNPROVEN` with the fence still held. This closes the
duplicate-writer falsifier structurally, not by UI discipline.

## 2. Admission honesty (v2)

Every failure AFTER admission begins routes to ambiguity:

- spawn errors of ANY kind → queued stays, `admission_ambiguous` observation,
  typed `ATTEMPT_START_UNRESOLVED`;
- snapshot/bind/activation failures after a LIVE spawn →
  `activation_ambiguous`, typed `ATTEMPT_ACTIVATION_UNRESOLVED`. The bind
  failure is injectable in tests (`failBindOnce` on the fake store) and the
  full ambiguity → owner containment → release → replacement sequence is
  pinned end to end.

Both derive as unconfirmed; both keep custody; duplicate start is impossible
(fence). Internal recording failures join the returned error instead of being
dropped. Cancel of an unbound ambiguous start REFUSES (`ATTEMPT_START_UNRESOLVED`)
— cancelling "as if nothing ran" would be a false stop record.

## 3. Provider authority via the execution seam

`ports.AttemptSessionSpawner.Terminate(ctx, projectID, sessionID)
(ports.TerminationResult, error)` adapted over `service/session.Kill`. The
two facts a stop produces are deliberately SEPARATE:

- `ProviderStopped` — derived from the DURABLE session record after teardown.
  Kill's boolean says nothing about liveness and is never used as proof.
- `WorkspaceFreed` — mirrors whether the workspace was reclaimed. A preserved
  dirty worktree yields `{true, false}`: the provider IS stopped, the
  evidence is kept. Cancellation proceeds and records `"workspaceFreed"` in
  the `owner_cancelled` observation payload.

An absent session row or a record without the termination fact is UNKNOWN →
the adapter returns `ports.ErrProviderStopUnproven`; cancel refuses
(`ATTEMPT_PROVIDER_STOP_FAILED`), leaving status+custody untouched with a
receipt.

**Pause AND public resume are deliberately absent** until a real
provider-control contract exists (ADR 0007 territory): a DB label flip over a
running CLI agent is exactly the dishonesty this stage forbids. There is no
resume RECOVERY verb either (`RECOVERY_ACTION_INVALID`); proving an
already-running provider alive is reconcile's job (`resumed` receipt). The
`paused` status and its transitions remain reserved in schema/domain.

## 4. Renewable lease + recency liveness

- `attempt_fences.last_renewed_at`: renewed ONLY for provably-alive running
  custodians each liveness pass (health-gated — renewal happens after the
  alive check, so stale/vanished custody keeps its OLD stamp); frozen
  post-release by trigger. Pinned by test: alive renews, backdated does not,
  sticky waiting_input does, vanished does not.
- Aliveness = present + signalled + not terminated + durable activity inside
  `DefaultStaleHeartbeatWindow` (sticky waiting_input/blocked exempt). Long-
  vanished sessions are NOT alive forever; staleness derives UNCONFIRMED,
  never dead.
- Terminated sessions promote running attempts to `reconciled` +
  `provider_exit` observation (ended ≠ classified).

## 5. Attention contract (truthful separation)

Three distinct read-time truths, never merged:

- **Waiting on you** (`needs_input`, attention `waiting_input`) — provider is
  PROVABLY alive and asked a question ("The agent asked you something").
- **Approval required** (`needs_input`, attention `blocked`) — a specific
  permission dialog awaits a decision.
- **Liveness unproven** (`unconfirmed`) — Kennel cannot prove aliveness;
  contain/reconcile/owner-assertion apply.

Titles are per-kind i18n keys (`waitingInputTitle` / `blockedTitle` /
`unconfirmedTitle`); no card borrows another truth's headline. Phases cross
the API enum-typed (`AttemptPresentationResponse.phase`), mirrored
renderer-side via `ATTEMPT_PHASES` constants.

## 6. Owner-containment release path (UI)

The unconfirmed recovery card offers Contain, Reconcile, and an explicit
**assert-stopped** action behind a TWO-STEP confirmation: first click reveals
a destructive-toned panel stating that Kennel cannot prove the provider is
running, that the owner must have verified externally, and that asserting it
releases custody on their word; the assert button then submits
`reconcile + confirmProviderStopped:true`. This unblocks the unbound
post-spawn failure case that can never obtain machine proof. Testids:
`outcome-run-owner-stop{,-confirm,-assert,-back}`.

## 7. API surface changes since round 1

- REMOVED: POST .../pause and .../resume (routes, spec ops, hooks).
- REMOVED (round 3): `resume` from the recovery action enum (service Valid,
  DTO enum tag, OpenAPI, generated client, hook union) — wire rejects it with
  `RECOVERY_ACTION_INVALID`.
- ADDED: `confirmProviderStopped` on recovery body; `attention` +
  `lastRenewedAt` on read models; phase enum; two-fact terminate contract.
- OpenAPI/schema.ts regenerated together; parity + drift green.

## 8. Test map additions (rounds 2–3)

bind-failure→activation-ambiguity incl. full owner-contained release +
replacement sequence; terminate-or-refuse cancel incl. absent-row unproven
and dirty-preserved-workspace cancellation; failed-status-not-proof gating;
health-gated lease renewal (alive/stale/sticky/vanished matrix at service
level, plus the store lease test); durable-record-derived stop proof at the
daemon adapter (terminated+dirty vs clean vs absent vs kill-error);
stale-vs-sticky derivation; attention titles per kind; owner-stop two-step
release flow; cancelled-card custody path; pause/resume unrouted assertions;
recovery-resume-verb rejection over HTTP.

## 9. Composition rules unchanged for intake/#32/#40

Same as before: compose CreateOutcomeWithContract/ReviseContract; approval
and admission stay separate durable facts; checkpoint E consumes observations
+ presentation; zero provider-name policy; 0102 lease releases at merge → #35
claims 0103. #67 rebase is next integration action once this head passes
review (shared execution/API contract changed again).

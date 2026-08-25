# Lane W implementation notes: #31 Act & Observe — for the Codex lane

- Status: pushed on `work/31-act-observe`, open as **PR #68** (round-2 remediation applied; merge blocked pending re-review)
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
- snapshot/bind failures after a LIVE spawn → `activation_ambiguous`,
  typed `ATTEMPT_ACTIVATION_UNRESOLVED`.

Both derive as unconfirmed; both keep custody; duplicate start is impossible
(fence). Internal recording failures join the returned error instead of being
dropped. Cancel of an unbound ambiguous start REFUSES (`ATTEMPT_START_UNRESOLVED`)
— cancelling "as if nothing ran" would be a false stop record.

## 3. Provider authority via the execution seam

`ports.AttemptSessionSpawner.Terminate(ctx, projectID, sessionID) (freed bool, err error)`:
adapted over `service/session.Kill`. `freed=false` (session row absent) is an
UNKNOWN outcome → adapter returns `ports.ErrProviderStopUnproven`; cancel
refuses (`ATTEMPT_PROVIDER_STOP_FAILED`), leaving status+custody untouched
with a receipt. Successful termination writes the durable stop-fact that later
unlocks release WITHOUT owner confirmation.

**Pause/resume are deliberately absent** until a real provider-control
contract exists (ADR 0007 territory): a DB "paused" label over a running CLI
agent is exactly the dishonesty this stage forbids. The `paused` status and
its transitions remain reserved in schema/domain.

## 4. Renewable lease + recency liveness

- `attempt_fences.last_renewed_at`: renewed ONLY for provably-alive running
  custodians each liveness pass (health-gated); frozen post-release by trigger.
  Stale stamps make possibly-dead custody visible.
- Aliveness = present + signalled + not terminated + durable activity inside
  `DefaultStaleHeartbeatWindow` (sticky waiting_input/blocked exempt). Long-
  vanished sessions are NOT alive forever; staleness derives UNCONFIRMED,
  never dead.
- Terminated sessions promote running attempts to `reconciled` +
  `provider_exit` observation (ended ≠ classified).

## 5. Attention contract

`presentation.attention` ∈ {waiting_input, blocked} accompanies phase
needs_input with decision-specific copy ("respond to its question" vs
"approve/deny the dialog"). Phases cross the API enum-typed
(`AttemptPresentationResponse.phase`), mirrored renderer-side via
`ATTEMPT_PHASES` constants.

## 6. API surface changes since round 1

- REMOVED: POST .../pause and .../resume (routes, spec ops, hooks).
- ADDED: `confirmProviderStopped` on recovery body; `attention` +
  `lastRenewedAt` on read models; phase enum.
- OpenAPI/schema.ts regenerated together; parity + drift green.

## 7. Test map additions (round 2)

bind-failure→activation-ambiguity; terminate-or-refuse cancel incl.
missing-session unproven; failed-status-not-proof gating; health-gated lease
renewal; stale-vs-sticky derivation; attention copy per kind; cancelled-card
custody path; pause/resume unrouted assertions.

## 8. Composition rules unchanged for intake/#32/#40

Same as before: compose CreateOutcomeWithContract/ReviseContract; approval
and admission stay separate durable facts; checkpoint E consumes observations
+ presentation; zero provider-name policy; 0102 lease releases at merge → #35
claims 0103. #67 rebase is next integration action once this head passes
review (shared execution/API contract changed again).

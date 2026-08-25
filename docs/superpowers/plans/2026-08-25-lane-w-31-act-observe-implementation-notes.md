# Lane W implementation notes: #31 Act & Observe — for the Codex lane

- Status: local untracked notes for cross-lane review; not yet in a docs PR
- Date: 2026-08-25
- Branch: `work/31-act-observe`, 10 commits ahead of `origin/beta` @ `b33e41a96`
- PR status: NOT pushed; awaiting owner push/PR authorization
- Reader: Codex lane owners of Slice B / #32 / #40 and reviewers of the #31 PR

## 1. What landed (one paragraph)

#31 turns an approved PlanRevision into a governed execution on a real
provider session: a fail-closed admission gauntlet writes one Attempt row +
worktree custody fence, spawns the worker through the EXISTING session-spawn
path behind a narrow port, binds an AgentSessionRef with frozen digests and a
versioned admission snapshot, records ordered append-only observations via DB
triggers into CDC, derives truthful presentation (unconfirmed / ended-
unclassified) at read time from heartbeat facts, and provides custody-safe
recovery verbs (contain / reconcile / resume / replace / attention) whose
receipts are the append-only record of every verdict.

## 2. Commits

| Commit | Layer |
| --- | --- |
| 06a22ce66 | feat(domain): attempt statuses, legal transitions, derived presentation |
| ed7e0a649 | feat(storage): migration 0102 + CDC writers + sqlc + store |
| 8402099cb | feat(service): admission ordering, recovery, liveness |
| 904c1af7b | feat(httpd): routes + specgen + daemon wiring + openapi/schema.ts regen |
| 5460bc7ee | feat(web): useOutcome attempt hooks + OutcomeRunSurface + act_observe stage |
| 91dd91c87 | fix(domain): queued->lost custody edge + unresolved-admission derivation |
| c7eccc998 | fix(service): terminal-custody release, replay-safe starts, ambiguous admission |
| 2ff00da8e | style(httpd): APIDeps gofmt |

## 3. Durable contracts now in SQLite (migration 0102)

- `attempts` — append-only lineage. Columns: id, outcome_id, plan_revision_id,
  work_unit_id, number (per-outcome), status CHECK(8 values),
  contract_revision_number (binding snapshot), request_key (partial UNIQUE =
  idempotency). ONLY legal mutation: status transition, enforced by trigger.
- `attempt_sessions` — AgentSessionRef bindings. session_id is TEXT **with NO
  FK into sessions(id)** by locked ruling D6 (refs must outlive spawn-rollbacks
  and session GC). Also: harness, mode, run_brief_core_digest,
  run_brief_compiled_digest, admission_snapshot JSON (snapshotVersion=1).
  UNIQUE(attempt_id, seq).
- `attempt_observations` — append-only, UNIQUE(attempt_id, seq), kind+JSON
  payload. Insertable for ANY attempt state (stale attempts stay inspectable)
  but nothing ever reads them as truth except the ambiguous-admission flag.
- `attempt_fences` — custody lock. Partial UNIQUE(subject) WHERE released_at
  IS NULL = exactly one open fence per worktree subject. Subject v0 naming:
  `"project:" + projectID` (domain.FenceSubjectForProject) — one governed
  writer per project tree. Release requires non-empty reason; release is final
  (row freezes).
- `attempt_recovery_receipts` — resolution CHECK IN ('resumed',
  'replacement_attempt', 'needs_attention'), replacement_attempt_id TEXT not
  FK'd on purpose (may name a future row).
- change_log CHECK extended with outcome_attempt_started/_updated/
  _session_bound/_observed/_recovered. All writers live ONLY in
  cdc_restore.go changeLogWriters (deps outcomes+responsibility_spaces);
  migration detaches everything and rebuilds per 0100 discipline.

## 4. Stored statuses and legal edges (trigger-guarded)

```
queued    -> running | failed | cancelled | lost
running   -> paused | failed | cancelled | lost | reconciled
paused    -> running | cancelled | lost
failed/cancelled/succeeded/lost/reconciled : terminal (no out-edges)
```

`unconfirmed` is NEVER stored — it is derived. `queued->lost` exists so an
admission whose outcome is UNKNOWN can resolve custody without being dressed
up as a clean failure. No path anywhere writes `succeeded`; that arrives with
#35's Verification binding.

## 5. The admission gauntlet (StartAttempt order matters)

ALL five gates run BEFORE any durable row exists — a refusal leaves zero rows:

1. plan approved (`PLAN_NOT_APPROVED`)
2. plan still binds Outcome's CURRENT contract revision (`PLAN_BRIEF_INVALIDATED` if superseded)
3. grants survive the authority intersection; required trio present (`ATTEMPT_CAPABILITY_UNAUTHORIZED`)
4. RunBrief core digest recomputed from current contract == plan's frozen digest (`PLAN_BRIEF_INVALIDATED`) — any material drift forces fresh proposal+approval
5. profile readiness probed through `session_manager.ProfileReadinessForSpawn`
   — same checker AND same config merge (project worker role defaults) that
   real Spawn enforces, so "ready" means launchable (`AGENT_PROFILE_NOT_READY`,
   `AGENT_BINARY_NOT_FOUND`)

Then, atomically: attempt row (queued) + fence issued in ONE transaction
(fence conflict => *ports.AttemptFenceHeldError with NOTHING persisted).
Then spawn through the narrow port; then bind ref; then queued->running.

## 6. Failure taxonomy after the row exists

- Definite spawn refusal -> status `failed` + `admission_failed` observation +
  needs_attention receipt. Fence stays HELD.
- UNKNOWN outcome (context.DeadlineExceeded / context.Canceled after send) ->
  status STAYS QUEUED + `admission_ambiguous` observation + needs_attention
  receipt; returns typed `ATTEMPT_START_UNRESOLVED`. Derivation presents
  unconfirmed. Fence HELD. Duplicate start structurally impossible (fence).
  Reconcile later resolves to lost + release + replacement receipt
  (`queued->lost` legal edge added for exactly this).
- Ref binding failure -> failed path as well (truthful: cannot observe it).
- Heuristic note: ambiguity classification keys off context errors today.
  When Slice C adds real provider routing, spawner adapters MUST classify
  post-send unknowns as deadline-ish errors or we extend the port with an
  explicit outcome enum.

## 7. Custody model (the part to internalize)

The fence is the anti-duplicate-effect device: two governed writers can never
hold one project worktree. Release happens ONLY through recovery verbs:
- reconcile/replace on LIVE states (queued/running/paused) prove-or-fail
  liveness -> lost -> release -> receipt.
- reconcile/replace on TERMINAL predecessors (failed/cancelled/reconciled)
  NEVER rewrite history; they release custody with reason
  `reconciled_terminal_<status>` / `replacement_attempt` and stamp a receipt
  naming previousStatus. This was a reviewed-in blocker fix: without it,
  cancelled/failed/reconciled attempts deadlocked custody forever.
- succeeded refuses both (not reachable in #31 anyway).

## 8. Read-time derivation (what the renderer is allowed to know)

`domain.DeriveAttemptPresentation(status, heartbeatFacts, unresolvedAdmission)`:
- running + no session row OR never signalled -> phase `unconfirmed`
  (missing heartbeat = unconfirmed, NEVER dead)
- running + terminated -> `ended_unclassified` ("provider completion ≠ done")
- running + signalled + alive -> `executing`
- queued + unresolvedAdmission observation -> `unconfirmed`
- paused/failed/cancelled/lost/reconciled/queued speak for themselves
Heartbeat facts triple read straight from sessions columns:
present, activity_state, first_signal_at, is_terminated. No transcript, no
derived-status reads. Known limitation (deliberate, joins #38): staleness is
binary — a long-silent-but-not-terminated session still derives executing.

## 9. Recovery verbs = POST .../attempts/{id}/recovery {action}

- contain: suspicion marker observation on active states; decides nothing.
- reconcile: auto-verdict. alive evidence -> receipt(resumed); terminal
  predecessor -> accountTerminalCustody; else forceLost (lost + release +
  receipt(replacement_attempt)).
- resume: paused->running ONLY after heartbeat-alive proof AND readiness
  re-probe (`ATTEMPT_LIVENESS_UNPROVEN` otherwise).
- replace: force custody handover; idempotent for lost/failed/cancelled/
  reconciled predecessors.
- attention: needs_attention receipt + escalation observation; mutates nothing.
Receipts are the audit trail; every verdict is inspectable after restart.

## 10. Liveness hook (daemon wiring)

`runAttemptLivenessLoop`: immediate evaluation on boot (folds exits missed
while down), then every 15s: for each RUNNING attempt, read latest bound ref's
session facts; terminated -> guarded transition to `reconciled` +
`provider_exit` observation. Silent heartbeats mutate NOTHING (they derive as
unconfirmed and wait for explicit contain/reconcile).

## 11. HTTP surface (all under /api/v1, envelope + request IDs preserved)

POST /outcomes/{outcomeId}/attempts {planRevisionId, harness?, requestKey}
GET  /outcomes/{outcomeId}/attempts            GET .../attempts/{attemptId}
POST .../attempts/{attemptId}/observations {kind, payload?}
POST .../attempts/{attemptId}/pause|resume|cancel
POST .../attempts/{attemptId}/recovery {action: contain|reconcile|resume|replace|attention}
Unwired => 501 like every optional surface. openapi.yaml + schema.ts committed
together; route/spec parity + gate drift checks green.

## 12. What Intake/#32/#40 must compose (NOT re-implement)

- Confirmation of an intake proposal keeps using the existing idempotent
  CreateOutcomeWithContract (+ ReviseContract for edits). #31 adds NO second
  creation path and expects none.
- "Authorize and start" one-gesture composition (#40) should call ApprovePlan
  then StartAttempt and surface BOTH receipts; the durable facts are already
  separate. Never merge them into one write.
- Checkpoint E (divergence) should consume the observations stream + attempt
  presentation; #31 preserves those seams only (drift detection joins #38).
- Provider naming: renderer shows zero provider-name policy today; keep it
  that way. Harness is recorded as fact on refs/snapshots only.
- Lease chain stands: release 0102 at #31 merge -> #35 claims 0103.

## 13. Test map (v0 failure matrix coverage)

- Fail-closed quartet leaves ZERO rows (service test, 4 subtests)
- Replay idempotency incl. same-key race resolved to winner (store + service)
- Ambiguous start stays queued/unconfirmed + reconcile resolves (service)
- Terminal-custody release for cancelled/failed/reconciled predecessors
  without history rewrite (service, 3 subtests)
- contain->reconcile->replacement flow w/ stale-observation inertness (service)
- proven-alive => resumed; liveness termination => reconciled (service)
- pause/resume guards incl. readiness re-probe (service)
- migration roundtrip, trigger aborts, partial-unique fence, CDC events,
  degraded burned-ledger profiles (storage tests)
- route/spec parity + 501-unwired + functional HTTP happy path (controllers)
- renderer: waiting vs needs-you vs ended-unclassified distinction, provider
  name absence, recovery verbs hit custody route (vitest, 7 cases)

## 14. Known limits (tracked forward, not hidden)

1. Heartbeat staleness binary (joins #38).
2. Ambiguity heuristic keyed on context errors (Slice C contract).
3. One fence per PROJECT subject: two Outcomes on one project contend by
   design; #40 workspace must present contention honestly.
4. Cancel ends governance but does not kill the provider pane; teardown rides
   existing session surfaces. Revisit if dogfooding demands cancel-to-kill
   (small follow-up behind the spawner port).

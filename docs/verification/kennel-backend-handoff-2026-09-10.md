# Kennel backend completion — handoff

Written 2026-09-10 by the Claude backend task, for whoever continues this
assignment. Nothing here is owner acceptance, live-provider conformance, or a
release decision.

## Where the work is

| Fact | Value |
|---|---|
| Branch | `codex/kennel-execution-completion` |
| Worktree | `<repo>/.claude/worktrees/kennel-execution-completion` |
| Base | `670a42238795b3da0c9fd61a5434132e9b622dd0` (= `origin/beta` at dispatch; unchanged since) |
| Head | `517e182e4` |
| Pushed? | **No.** Nothing pushed, nothing merged. Working tree clean. |

The worktree was created with `git worktree add`; it is not the primary
checkout and no other worktree was touched. One stash entry belonging to a
different session (`work/31-act-observe`) is still on the shared stash stack —
leave it alone.

## Ordered commits

| SHA | Slice |
|---|---|
| `7929898` | delta map + Mission Control interface note |
| `34e2705` | **interface commit** — Mission run state; run/delivery/usage declared with honest 501s |
| `f350bef` | C13 artifact continuity |
| `16b492b` | governed checks in the production lifecycle |
| `49c1cc5` | review R1 — proof generation read before proof |
| `97601dd` | review R3 — argv preserved verbatim |
| `1afc59b` | review R2 — durable check-run identity + reservation |
| `091f138` | C13 launch-boundary tests; published `approvedChecks` contract delta |
| `a2822d9` | durable run intent + serial continuation |
| `517e182` | supplied-document Outcomes from an approved snapshot |

**Dependency SHAs for the Mission Control task:** `34e2705e4` (the interface
commit) and `091f138fc` (interface note §2.6). Later commits implement things
that note already describes.

## Migrations added

`0122` approved checks · `0123` attempt check runs · `0124` outcome run
intents · `0125` outcome document contexts.

`0124` rebuilds `change_log` to admit three event types
(`outcome_run_intent_changed`, `outcome_attempt_retained`,
`outcome_delivery_changed`). Only the first has a writer today; the other two
were added in the same rebuild so the delivery and retention slices do not
each need another one. **Every CDC writer must be detached in a rebuild** —
there is a guard test that enumerates them from `cdc_restore.go`, and it will
tell you exactly which ones you missed.

None of these has reached `beta`, so renumbering is still permitted if a
conflict appears. `0124` was already renumbered once for this reason.

## What is done

Each of these is `automated` — real filesystem, real SQLite, real macOS
seatbelt for the check tests. Evidence tables per slice are in
`docs/verification/kennel-backend-completion.md`.

- **C13 artifact continuity.** `UPSTREAM_MATERIALIZATION_UNAVAILABLE` is gone
  because provisioning works, not because the gate was relaxed. Admission
  pins the producing Attempt *and* artifact version; the session manager
  materializes onto the approved base before any launch; failure is a known
  pre-launch refusal (`UPSTREAM_MATERIALIZATION_FAILED`), not an unknown start.
- **Governed checks.** Approved commands are Plan authority frozen in the
  RunBrief digest and in schema; they run under the Attempt's own policy in
  the workspace that produced the result; the workspace is re-measured
  afterwards so a check that rewrote what it checked cannot pass; results
  become criterion-bound evidence and verification.
- **Durable run intent.** Append-only generations; Start authorizes without
  launching; Pause blocks admission on *every* path including per-Attempt
  Start; Cancel acknowledges only on a proven stop; serial continuation admits
  each eligible WorkUnit exactly once.
- **Supplied documents.** Select → snapshot → approve → stage, with
  edited-source refusal and grounding bound to the snapshot digest.

## What is NOT done

In assignment order:

1. **Rework** (part of slice 3). Owner feedback that records a decision and
   moves the relevant revision/proof horizon *without rewriting accepted
   evidence* is not implemented. The pieces it needs exist
   (`DecideAcceptance` with re-entry targets, `OutcomeCorrection`,
   `ProofHorizon`), so this is wiring plus tests, not new authority.
2. **Slice 5 — durable delivery.** `artifactstore/export.go` is complete and
   unit-tested but still has **no production caller**. The API is declared and
   answers `501 DELIVERY_UNAVAILABLE`
   (`controllers.OutcomesController.requestDelivery` et al). Needs: a
   deliveries table, a canonical service, and wiring; the DTO shapes are
   already published in the interface note §2.4, so follow them.
3. **Slice 6 — agent bridge.** Not started. Inventory first:
   `adapters/agent/hookutil`, `claudecode/hooks.go`, `cursor/hooks.go`, and
   the existing token-scoped callback at
   `POST /decomposition-requests/{requestId}/proposal` — that callback is the
   closest existing pattern for Attempt-scoped identity.
4. **Slice 7 — usage attribution.** Not started. `GET .../usage` answers
   `501 USAGE_ATTRIBUTION_UNAVAILABLE`. Existing collection is session-scoped
   (`controllers/usage.go`, `service/usage`). Shape published in interface
   note §2.5: reuse `UsageTotalsResponse`, keep nullables meaning unknown.

## Things that will bite you

- **No live provider has ever proposed a check command.** The reasoning
  adapter accepts `checkCommands`, the compiler validates and freezes them,
  execution runs them — but a real Plan's `approvedChecks` is empty today, so
  those criteria stay unproved. That is fail-closed, not a silent pass. Do
  not describe checks as proven end to end until a real proposal produces one.
- **The daemon-level C13 and check tests use fakes for the provider.** No
  provider process has been launched by any test in this branch.
  `TestSpawn_MaterializesInputsBeforeTheProviderIsLaunched` proves ordering
  and refusal through the session manager's fakes — it is not conformance.
- **No booted-daemon or packaged journey has been run at all.**
- `npm run bootstrap` fails at `frontend/src/landing` (no lockfile). This is
  pre-existing at the base commit, not caused by this work. Everything else
  in that script runs.
- **Two cross-lane one-liners.** Widening the schedule `blockedReason` enum
  broke `frontend:typecheck`, because the renderer maps it exhaustively to
  i18n keys. I added one key to each of the eight
  `frontend/src/renderer/i18n/*.json` and `approvedChecks: []` to
  `frontend/src/renderer/lib/preview-outcome-store.ts`. No renderer component
  was changed; the Mission task owns the wording and may replace both.
- The migration ledger in
  `internal/storage/sqlite/migrate_burned_versions_test.go` must be updated in
  the same change as any new migration.

## How to verify what is here

From the worktree root, with a disposable `KENNEL_DATA_DIR`:

```sh
cd backend && go build ./... && go vet ./... && go test ./...
cd .. && npm run lint && npm run sqlc && npm run api && npm run frontend:typecheck
```

All of the above pass at `517e182e4`, and `sqlc`/`api` produce no drift.
`go test -race` was run on the touched packages (`artifactstore`, `daemon`,
`service/outcome`, `session_manager`, `storage/sqlite/...`, `httpd/...`).

The tests worth reading first, because they encode the invariants rather than
the implementation:

- `internal/daemon` `TestProvisionAttemptInputs_HandsTheExactRetainedResultToASuccessorAfterRestart`
- `internal/service/outcome` `TestReconcileAttemptOutcomes_RefusesProofThatChangedUnderTheSnapshot`
- `internal/service/outcome` `TestApprovedChecks_AFailingCheckIsNotRelaunchedOnEveryTick`
- `internal/service/outcome` `TestContinueAuthorizedRuns_AdmitsEachEligibleWorkUnitOnce`
- `internal/service/outcome` `TestDocumentOutcome_SelectApproveThenStage`

## Conventions this branch follows

- One canonical writer per aggregate; new capability goes behind a port with
  an `xEnabled()` predicate so an unwired daemon answers 501 rather than 500.
- Every durable authority table is append-only with immutability triggers, and
  the store — not the caller — assigns revisions and generations.
- A refusal carries a stable code; `inconclusive` is never collapsed into
  `failed`; unknown is never rendered as zero.
- Comments explain why an ordering or a refusal exists, not what the line does.

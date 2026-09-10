# Kennel backend completion — delta, evidence and boundaries

Opened 2026-09-10 for the backend half of the parallel Kennel completion build.
The renderer/Mission-Control half is a separate task and owns
`docs/verification/kennel-mission-completion.md`.

**Nothing in this file is owner acceptance, live-provider conformance, or a
release decision.** Proof levels follow the ledger convention already in
`kennel-work-completion-ledger.md`: `source` < `automated` < `daemon-fixture` <
`real-provider` < `packaged` < `owner`. A missing runtime or credential is
**blocked**, never pass.

## Branch and base

| Fact | Value |
|---|---|
| Branch | `codex/kennel-execution-completion` |
| Worktree | `<repo>/.claude/worktrees/kennel-execution-completion` |
| Base commit | `670a42238795b3da0c9fd61a5434132e9b622dd0` |
| `origin/beta` at fetch time (2026-09-10) | `670a42238795b3da0c9fd61a5434132e9b622dd0` — identical to the dispatched base, so no rebase was needed |
| Parallel renderer task | `codex/mission-control-frontend`, also based at `670a4223` |

## Baseline on the base commit, before any implementation

Run from `backend/` with a disposable `KENNEL_DATA_DIR`.

| Gate | Result |
|---|---|
| `go build ./...` | pass (exit 0) |
| `go vet ./...` | pass (exit 0) |
| `go test ./...` | pass (exit 0) |

Repo-wide gates (`npm run lint`, `frontend:typecheck`, `sqlc`, `api`) are run
per touched area and recorded at the commits that touch them.

## Source-to-requirement delta

Read as: *requirement → what already exists at `670a4223` → what is missing*.
Requirement numbers are the assignment's implementation sequence.

### 1. C13 artifact continuity

| Requirement | Present | Missing |
|---|---|---|
| Validate frozen predecessor receipts/blobs, digests, lineage | `service/outcome/handoff.go` — `resolveUpstreamReceipts`, `upstreamReceiptUsable`; refusal codes `UPSTREAM_ARTIFACT_MISSING` / `_INCOMPLETE` / `_UNREVIEWED` / `UPSTREAM_LINEAGE_MISMATCH` | nothing — this half is complete and tested |
| Deterministic multi-predecessor composition | `artifactstore/handoff.go` — `Compose` (repeat-version, incompatible-base and conflicting-path refusals) | it is called from no production path |
| Materialize into the successor workspace | `artifactstore/handoff.go` — `Apply` (digest + mode verification) | `Apply` requires an **empty** destination and skips deletions, so it cannot materialize *onto an approved base*; unchanged base files, deletions and executable modes are therefore not yet handled at a real launch |
| Provision before provider launch | — | `service/outcome/attempt.go:246` calls `requireUpstreamArtifacts`, which validates and then unconditionally refuses with `UPSTREAM_MATERIALIZATION_UNAVAILABLE` (`handoff.go:156`). No seam exists between `createSessionWorkspace`/`provisionWorkspace` and the runtime launch in `session_manager/manager.go` for admitted inputs |
| Record exact input artifact versions in admission/replay | admission snapshot exists (`attempt.go`, `domain.AdmissionSnapshotVersion`) | no `inputArtifactVersions`; a restart would re-resolve "latest" upstream rather than replaying the authorized inputs |
| Failed provisioning must not launch, and must stay inspectable | `admitUnresolved` records ambiguous starts | provisioning failure is a *known* pre-launch failure and must not be reported as an unknown start |

### 2. Governed checks and evidence

| Requirement | Present | Missing |
|---|---|---|
| Enforced check runner | `internal/governedcheck` — discrete argv, no daemon env, `Enforcement` required, bounded output/time, cancellation, process-tree control, `TerminationUnknown` | **no production caller** — `grep governedcheck` outside its own package returns nothing |
| Canonical evidence/verification | `service/outcome/proof.go` — `RecordEvidence`, `RecordVerification`, criterion binding, proof horizon | results are client-supplied claims; there is no independently observed run bound to producer Attempt + artifact version |
| Immutable receipts, atomic classification | `ports.AttemptSuccessFinalizer.ClassifyAttemptSucceeded`, migrations 0119/0120 | — (R1/R3 landed before this base) |

### 3. Durable run intent and rework

| Requirement | Present | Missing |
|---|---|---|
| Derived schedule projection | `service/outcome/scheduler.go` — states, `BlockedReason`, `NoRunnableReason`, `CustodyHeldBy` | — |
| Per-attempt cancel | `CancelAttempt` with proven-stop refusal | — |
| Persisted run intent/generation | — | no `run_intent` table, no Outcome-level Start/Pause/Resume/Cancel, no replay protection or recovery reconciliation for intent. `grep RunIntent` returns nothing |
| Serial continuation admitting each eligible unit once | `nextRunnableWorkUnit` is a pure decision | nothing drives it after an Attempt succeeds |

### 4. Supplied-document path

| Requirement | Present | Missing |
|---|---|---|
| Bounded document reader | `service/intelligence/supplied_context.go` | not wired to selected-context identity, grounding digest, approval, or staged workspace custody |
| Staged-folder custody in retention | `domain.WorkspaceStagedFolder`, artifactstore staged capture | no document Outcome integration path |

### 5. Durable delivery

| Requirement | Present | Missing |
|---|---|---|
| Export helper and durable delivery | `artifactstore/export.go`; SQLite `outcome_deliveries`; Outcome delivery service/routes | pending request/result ledger, exact receipt and current owner-acceptance binding, idempotency, atomic staging, destination safety, and restart interruption are implemented; real owner export/packaged acceptance remains open |

### 6. Agent bridge baseline

| Requirement | Present | Missing |
|---|---|---|
| Provider hooks | `adapters/agent/hookutil`, `claudecode/hooks.go`, `cursor/hooks.go` | observations only |
| Bounded agent submission | `POST /decomposition-requests/{requestId}/proposal` with a callback token | no Attempt-scoped bridge, no tool surface, no skill packaging |

### 7. Usage attribution

| Requirement | Present | Missing |
|---|---|---|
| Usage collection | `httpd/controllers/usage.go` — `/usage/sessions` | session-scoped only; no Outcome/WorkUnit/Attempt attribution, no planning-usage attribution, no double-count guard |

## Corrections to inherited status text

- `docs/superpowers/plans/2026-09-10-luna-kennel-work-completion.md` slice R
  (R1–R5) and slice C-12 describe work that **is present** at this base
  (`ports.AttemptSuccessFinalizer`, `accountSucceededCustody`, migration 0120,
  `domain.ArtifactManifestDigest` including file mode and component-wise path
  validation, `internal/artifactstore`). They are not re-opened here.
- C-13 is genuinely open, exactly as `docs/STATUS.md` and the fresh-beta
  boundary document state.

## Implementation record

| Slice | Commit | Gates run | Claim level |
|---|---|---|---|
| Delta + interface note | `7929898` | n/a (docs) | `source` |
| Mission run state + declared run/delivery/usage API | `34e2705` | `go build`, `go vet`, `go test ./...`, `npm run lint`, `npm run frontend:typecheck` | `automated` |
| C13 artifact continuity | `f350bef` | `go build`, `go vet`, `go test ./...`, `go test -race` on touched packages, `npm run lint`, `npm run frontend:typecheck` | `automated`; real-provider and packaged acceptance open |
| Governed checks in the production lifecycle | `16b492b` | `go build`, `go vet`, `go test ./...`, `go test -race` on touched packages, `npm run lint`, `npm run sqlc`, `npm run api`, `npm run frontend:typecheck` | `automated`; real-provider and packaged acceptance open |
| Review corrections R1–R3 + launch-boundary test | `49c1cc5`, `97601dd`, `1afc59b`, _(this commit)_ | `go build`, `go vet`, `go test ./...`, `npm run lint`, `npm run sqlc`, `npm run api`, `npm run frontend:typecheck` | `automated` |

## C13 artifact continuity — what is now true, and what is not

### Behavior

`UPSTREAM_MATERIALIZATION_UNAVAILABLE` is gone because provisioning actually
happens at the launch seam, not because the gate was relaxed. The path is:

1. `service/outcome/attempt.go` resolves `admittedInputsFor` **before** taking
   the worktree fence. It returns `ports.AttemptInputRef` values pinning the
   producing Attempt, its WorkUnit and its **exact artifact version**.
2. Those references travel through `ports.AttemptSpawnRequest.Inputs` →
   `ports.SpawnConfig.AttemptInputs` into the session manager.
3. `Manager.provisionAttemptInputs` runs after the workspace holds the approved
   base and before any launch path. No provisioner wired means the spawn is
   refused, never silently skipped.
4. `daemon.attemptInputProvisioner` re-reads each receipt by Attempt id and
   refuses unless the stored artifact version, WorkUnit and frozen state all
   match what was admitted. `artifactstore.Compose` then reads and digest-checks
   every blob, and `artifactstore.Materialize` writes the changes onto the base
   checkout — additions, modifications, executable modes, binary bytes and
   deletions — then re-reads what it wrote before returning.
5. The exact versions are recorded on the admission snapshot
   (`inputArtifactVersions`) and folded into the compiled brief digest, so a
   replay fingerprint distinguishes the same WorkUnit run against different
   predecessor output.

Failure is reported as a **known pre-launch failure**, not an unknown start:
`UPSTREAM_MATERIALIZATION_FAILED` plus an `input_provisioning_failed`
observation, and the Attempt is ended so its custody is released for a
deliberate retry. The session manager destroys the workspace only when it is
clean, so a partially provisioned tree stays inspectable.

The schedule projection gained `upstream_artifact_unavailable`
(`WorkUnitScheduleView.BlockedReason`) with a `blockedDetail` naming the
specific refusal, so a unit whose dependency is *proved* but whose output
cannot be handed down no longer displays as runnable.

### Two defects found and fixed while implementing this

- `artifactstore.Compose` deduplicated predecessors by **artifact version**.
  Because that version is a digest over the manifest, two different WorkUnits
  producing byte-identical output shared one, and a legitimate shared-ancestry
  join was refused as a repeated predecessor. It now deduplicates by producing
  Attempt.
- `artifactstore.Apply` required an **empty** destination and skipped
  deletions, so it could not materialize onto an approved base at all. It is
  replaced by `Materialize`, which writes onto the base checkout, applies
  deletions, verifies base compatibility, and refuses paths that leave custody
  through a name or through an existing symlink.

### Evidence

| Behavior | Test | Level |
|---|---|---|
| A writes distinctive bytes → durable state closed and reopened → A's workspace removed → B holds exactly those bytes, executable mode, binary content, deletion and unchanged base files | `internal/daemon` `TestProvisionAttemptInputs_HandsTheExactRetainedResultToASuccessorAfterRestart` (real SQLite + real blob store + real Git clones) | `automated` |
| Corrupt, replaced, unfrozen, wrong-WorkUnit and missing upstream results all refuse, writing nothing | `internal/daemon` `TestProvisionAttemptInputs_RefusesAnythingButTheAdmittedArtifact` | `automated` |
| Unwired retention refuses instead of launching without inputs | `internal/daemon` `..._RefusesWhenRetentionIsUnwired`, `internal/session_manager` `TestProvisionAttemptInputs_RefusesWhenHandoffIsUnwired` | `automated` |
| Base files, modes, binaries and deletions survive materialization onto a base | `internal/artifactstore` `TestMaterialize_GivesTheSuccessorTheExactPredecessorTreeOnTopOfItsBase` | `automated` |
| Incompatible base refuses and writes nothing | `TestMaterialize_RefusesAWorkspaceOnADifferentBase` | `automated` |
| Traversal, absolute paths and symlinked directories refuse | `TestMaterialize_RefusesPathsThatLeaveTheSuccessorWorkspace` | `automated` |
| Conflicting bytes/modes and repeated predecessors refuse; byte-identical shared ancestry joins | `TestCompose_RefusesPredecessorsThatCannotBeJoined` | `automated` |
| Provisioning failure ends the Attempt, records a non-ambiguous observation, binds no session | `internal/service/outcome` `TestStartAttempt_InputProvisioningFailureEndsTheAttemptWithoutLaunching` | `automated` |
| A replayed Start returns the same Attempt and launches once | `TestStartAttempt_ReplayedRequestKeyDoesNotLaunchTwice` | `automated` |
| A proved dependency with unusable output is not offered as runnable | `TestDeriveSchedule_AProvedDependencyWithUnusableOutputIsNotRunnable` | `automated` |

### What this is **not**

- Not live-provider evidence. No provider process was launched in any of these
  tests; the spawner is a double at the service level, and the daemon-level
  test exercises provisioning directly.
- Not packaged or real-daemon acceptance. Nothing here was observed in the
  built desktop package or through a booted daemon end to end.
- Not owner acceptance.

### Cross-lane touch

Widening the `blockedReason` enum broke `frontend:typecheck`, because the
renderer maps that enum exhaustively to i18n keys. One line was added to each
of the eight `frontend/src/renderer/i18n/*.json` files
(`outcome.missionGraph.blocked.upstream_artifact_unavailable`) to keep the
repository typechecking. No renderer component was changed; the Mission Control
task owns the wording and may replace it.


## Governed checks and evidence — what is now true, and what is not

### Behavior

`internal/governedcheck` had no production caller. It now has one, and the
Plan carries the commands it runs.

1. **Approved checks are Plan authority.** `domain.ApprovedCheck` is an
   argument vector bound to a criterion, with a bounded timeout. It lives on
   the WorkUnit alongside the prose `EvidenceChecks` the provider reads, and it
   is folded into the RunBrief core digest — so if the command could change
   after approval, the frozen digest would no longer match and the Attempt
   would be refused. Migration **0122** adds `work_unit_checks` with
   update/delete triggers, a `json_array_length(argv) >= 1` constraint and a
   timeout bound, so approved authority is frozen in the schema too.
2. **Model output is a proposal.** `compileApprovedChecks` resolves the
   criterion alias to internal identity, refuses a criterion the WorkUnit does
   not own, mints the identifier itself, bounds an absent or absurd timeout,
   and hands the command to `domain.ValidateApprovedChecks`, which refuses a
   shell (`sh`, `bash`, `env`, …) and an executable given as a path.
3. **Checks run under the Attempt's own frozen policy**, in the workspace that
   Attempt produced, via `daemon.attemptCheckRunner` →
   `governedcheck.Run`. Capabilities are not widened for checking: a check that
   could do more than the work it checks is a hole in the same fence.
4. **The result is re-measured afterwards.** `artifactstore.ObserveVersion`
   recomputes the workspace manifest; if it differs from the retained artifact
   version — or cannot be measured at all — every pass from that run becomes
   `inconclusive` rather than being attached to bytes that no longer exist.
5. **Proof is written through the canonical services.** Each observation
   produces an `EvidenceItem` (`deterministic_check` / producer `tool`) and a
   `VerificationRun` (`deterministic`), both bound to criterion + producing
   Attempt + exact artifact version, with a deterministic request key over
   those three so a repeated reconciliation tick replays instead of
   double-recording.
6. **Classification consumes that proof.** `ReconcileAttemptOutcomes` runs the
   checks before judging `attemptProven`, then re-reads the proof projection
   **and** the append-only proof generation, because new rows moved it. Process
   completion still proves nothing on its own.

A check that never launched — no enforcement mechanism, an invalid command —
is recorded `inconclusive`, never `failed`. A host without a sandbox has a
setup problem; a red check has a work problem, and the owner fixes them
differently.

### Evidence

| Behavior | Test | Level |
|---|---|---|
| Approved commands run enforced in the producing workspace; pass and failure are distinguished and the enforcing mechanism is named | `internal/daemon` `TestRunAttemptChecks_ObservesPassAndFailureInTheProducingWorkspace` | `automated` (real seatbelt on macOS; skips where no mechanism exists) |
| A check that rewrites the result it is checking is detected | `TestRunAttemptChecks_DetectsACheckThatRewritesTheResultItChecks` | `automated` |
| An unenforceable policy reports "did not run", not "failed" | `TestRunAttemptChecks_ReportsAnUnenforceablePolicyAsNotRun` | `automated` |
| did-not-run / failed / changed-under-check / unconfirmed-termination map to distinct proof verdicts | `internal/service/outcome` `TestCheckVerdict_DistinguishesDidNotRunFromFailed` | `automated` |
| Evidence never implies a confinement that did not exist | `TestCheckVerifierRef_NeverImpliesConfinementThatDidNotExist` | `automated` |
| Repeated ticks replay; a new artifact version gets its own observation | `TestCheckRequestKey_IsPerAttemptPerArtifactPerCheck` | `automated` |
| Shells, path executables, unknown and unowned criteria are refused at compile time; ids and timeouts are re-decided by the daemon | `TestCompileApprovedChecks_*` | `automated` |
| Changing a command or timeout changes the frozen Plan digest | `TestApprovedChecksAreFrozenByThePlanDigest` | `automated` |
| Checks survive readback in approved order with their criterion binding | `internal/storage/sqlite/store` `TestApprovedChecksRoundTripInApprovedOrder` | `automated` |
| Rewriting, re-binding, re-ordering, widening or deleting an approved check is refused by the schema | `internal/storage/sqlite` `TestMigration0122ApprovedChecksAreFrozenAuthority`, `..._RefusesUnboundedAndEmptyChecks` | `automated` |

Existing enforcement canaries in `internal/governedcheck/runner_test.go`
(out-of-workspace write denied, network denied, process-tree termination) are
unchanged and still cover the boundary itself.

### What this is **not**

- **Not proof that any real Plan carries checks yet.** The reasoning adapter
  now accepts a `checkCommands` field, but no live provider run has produced
  one. Until a real proposal does, a Plan's `approvedChecks` is empty and the
  criteria it would prove stay unproved — which is the fail-closed direction,
  not a silent pass.
- Not live-provider conformance and not packaged acceptance.
- The two enforced-execution tests **skip** on a host with no enforcement
  mechanism. On this machine (macOS, `/usr/bin/sandbox-exec` present) they ran.


## Review corrections — checkpoint `16b492bea`

A reviewer traced three source defects in the checks slice. Each was
reproduced with a failing regression before being fixed, and none was
addressed by weakening a guard.

### R1 — the generation fence was defeated by the read order (P1)

After checks wrote new proof, the refresh read `GetProof` first and
`OutcomeProofGeneration` second. A contradicting record committing between
them is counted in the generation while missing from the snapshot, so
classification's commit-time revalidation sees the generation it expects and
accepts proof that no longer holds.

Fixed by reading the generation first, in one function whose contract *is*
the ordering. **Red-green shown:** with the previous order the regression
reports the Attempt classified as `succeeded` on the stale snapshot; with the
fix it stays `reconciled`. The interleaved record commits on a store hook, not
a sleep.

- `TestReconcileAttemptOutcomes_RefusesProofThatChangedUnderTheSnapshot`
- `TestReconcileAttemptOutcomes_ClassifiesAProvedAttempt` (the same fixture
  without an interleaved write, so the refusal is attributable to the race)

### R2 — checks re-executed on every reconciliation tick (P1)

`RunAttemptChecks` was invoked before consulting any stored result, and
reconciliation re-enumerates every ended Attempt each tick. A failing check
relaunched its command indefinitely; a crash between the evidence write and
its verification relaunched it; two reconcilers could launch it at once. A
rerun also produced different output under the same request key, so the proof
write returned a replay conflict that never resolved.

Fixed with durable check-run identity (migration **0123**, `attempt_check_runs`):
one row per (Attempt, check, artifact version), **reserved before invocation**,
completed once with an immutable observation. Both proof writes derive from
that stored observation, so a restart between them rebuilds byte-identical
content. A reservation that survives a restart is `unknown` — never retried,
never reported as failed — and its partially written fields are discarded,
because an incomplete observation is none.

- `TestApprovedChecks_AFailingCheckIsNotRelaunchedOnEveryTick` (three ticks, one invocation)
- `TestApprovedChecks_ARestartAfterEvidenceCompletesProofWithoutRerunning`
- `TestApprovedChecks_ConcurrentReconcilersLaunchTheCommandOnce`
- `TestApprovedChecks_AnInterruptedRunIsUnknownAndNeverRetried`
- `TestApprovedChecks_APassingCheckProvesTheCriterionOnce` (green half)
- store level: `TestReserveAttemptCheckRun_AdmitsExactlyOneInvoker`,
  `TestRecordAttemptCheckObservation_IsWriteOnce`,
  `TestMarkAttemptCheckRunUnknown_OnlyClosesAnIncompleteReservation`

### R3 — normalization rewrote argv (P2)

Proposed arguments went through `trimAll`, which trims each one and drops
empties — changing what the command does and shifting positional arguments.
Now the vector is copied verbatim; the executable is validated separately (a
bare name, no path, no shell, no surrounding whitespace) and an empty or
whitespace-only argument is refused **by position** rather than removed.

- `TestPlanDraftChecks_PreservesTheArgumentVectorExactly`
- `TestPlanDraftChecks_DoesNotCopyTheProposalsBackingArray`
- `TestApprovedCheckValidate_AcceptsMeaningfulWhitespaceInsideArguments`
- `TestApprovedCheckValidate_RefusesArgumentsItWillNotSilentlyRewrite`

### C13 launch boundary (reviewer follow-up)

The reviewer noted the daemon-level C13 test calls provisioning directly. A
real launch-boundary test now drives `Manager.Spawn`:

- `TestSpawn_MaterializesInputsBeforeTheProviderIsLaunched` — measures that
  the runtime had created nothing when provisioning was asked for
- `TestSpawn_RefusesToLaunchWhenInputsCannotBeMaterialized` — corrupt input
  and unwired handoff both leave the runtime with **zero** sessions created
- `TestSpawn_WithoutAdmittedInputsDoesNotConsultTheProvisioner`

These use the session-manager fakes, so they prove the launch **ordering and
refusal**, not provider conformance.

### Contract delta published

`docs/verification/kennel-backend-interface.md` §2.6 now states the
`approvedChecks` shape on `PlanWorkUnit`, that authorization must display it,
and that `inconclusive` must not be rendered as `failed`.


## Durable run intent and serial continuation

### Behavior

Authorization is now a durable, append-only record rather than a running
process or an open screen. Migration **0124** adds `outcome_run_intents`: one
row per generation, immutable except for a write-once `acknowledged_at`, plus
the `outcome_run_intent_changed` CDC event (the change_log vocabulary was
rebuilt once to admit it, and the retention/delivery events their slices will
need).

- **Start** records `running` and launches nothing. The daemon's reconcile
  tick admits the next eligible WorkUnit itself, keyed
  `run:<outcome>:<generation>:<unit>`, so repeated ticks, a restart
  mid-admission and two reconcilers all converge on one Attempt.
- **Pause** prevents subsequent admission and leaves running work alone. It is
  enforced in `StartAttempt`, so the low-level per-Attempt route cannot be
  used to click past it. Acknowledged immediately when nothing is active,
  otherwise by the reconciler once the active Attempt ends.
- **Cancel** additionally terminates the active Attempt and is acknowledged
  only when the stop is proven; an unproven stop leaves the request visible.
- **Resume** is refused from `cancelled` — a fresh Start is the honest way to
  ask for more work after ending a run.
- Authorizing work re-validates the approved Plan every time, and is refused
  while an Attempt's runtime status is unknown (`RUN_CUSTODY_UNKNOWN`): a
  failed probe is not death, and continuing over it duplicates work.
- Replay is resolved before the transition is decided, so a repeated command
  returns its own earlier generation instead of being judged against the state
  it created.

### Evidence

| Behavior | Test | Level |
|---|---|---|
| Start authorizes without launching | `TestCommandRun_StartAuthorizesWithoutLaunching` | `automated` |
| A repeated command authorizes once | `TestCommandRun_ARepeatedCommandAuthorizesOnce` | `automated` |
| Pause-from-idle, resume-from-running and resume-from-cancelled are refused; Start after cancel is allowed | `TestCommandRun_RefusesCommandsThatDoNotApply` | `automated` |
| A stale generation is refused | `TestCommandRun_RefusesAStaleGeneration` | `automated` |
| A pause stops admission on the per-Attempt route too, and resume restores it | `TestRunIntent_PauseStopsSubsequentAdmission` | `automated` |
| A pause with nothing running is acknowledged immediately | `TestRunIntent_PauseWithNoActiveWorkIsAcknowledgedImmediately` | `automated` |
| Three continuation ticks admit exactly one Attempt | `TestContinueAuthorizedRuns_AdmitsEachEligibleWorkUnitOnce` | `automated` |
| A paused or cancelled run admits nothing | `TestContinueAuthorizedRuns_DoesNothingWhilePausedOrCancelled` | `automated` |
| The store numbers generations, ignores a guessed one, and replays a repeated key without appending | `TestAppendRunIntent_NumbersGenerationsAndReplaysARepeatedCommand` | `automated` |
| Only the current generation authorizes work | `TestCurrentRunIntent_IsTheLatestGenerationOnly` | `automated` |
| Acknowledgement is write-once | `TestAcknowledgeRunIntent_IsWriteOnce`, `TestMigration0124RunIntentGenerationsAreAppendOnly` | `automated` |
| Authorizing and acknowledging both emit canonical CDC | `TestMigration0124EmitsRunIntentChangeEvents` | `automated` |
| A daemon without run-intent storage reports it unavailable | `TestCommandRun_UnwiredRunIntentsReportUnavailable` | `automated` |

### What this is **not**

- **Rework is not implemented in this slice.** Owner feedback changing the
  revision/proof horizon without rewriting accepted evidence remains open.
- No restart of a live daemon was exercised; restart safety is argued from
  durable reads and proven at the store level, not from a booted process.
- Not live-provider or packaged acceptance.

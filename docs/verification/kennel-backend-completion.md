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
| Export helper | `artifactstore/export.go` — accepted/draft disposition, artifact-version binding, manifest, destination confinement | **no production caller**; no service, no API, no durable delivery record, no retry/partial-failure state |

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
| C13 artifact continuity | _(this commit)_ | `go build`, `go vet`, `go test ./...`, `go test -race` on touched packages, `npm run lint`, `npm run frontend:typecheck` | `automated`; real-provider and packaged acceptance open |

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

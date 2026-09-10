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
| _(appended per commit below)_ | | | |

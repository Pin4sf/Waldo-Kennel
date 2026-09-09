# The execution → proof → admission sequence

Date: 2026-09-09. Status: **implementation contract for Phases C and D.**
Written before that code, at the owner's instruction, so the ugly paths are
designed rather than discovered.

```text
execution ended → retained artifact → verification → eligible success
              → custody handling → downstream admission
```

Every arrow below has a named condition and a named refusal. Owner Acceptance is
outside this sequence entirely: the furthest it reaches on its own is Ready for
Review.

## What already exists, and what the gap actually is

`domain.AttemptStatus` already carries the states this needs, and one of them
already means what "execution ended" has to mean:

| Status | Meaning today |
|---|---|
| `running` | admitted and executing |
| `paused` | owner paused; still holds custody |
| `reconciled` | **ended, result unclassified** — the UI already says "Completion is not final acceptance — classification happens through Verification" |
| `failed` | truthful spawn/runner failure |
| `cancelled` | owner cancelled |
| `lost` | owner-driven recovery could not account for it; custody released |
| `succeeded` | **unreachable** |

`LegalAttemptTransitions` has **no entry producing `succeeded`**, and the SQL
triggers reject anything outside that map — so the gap is not only missing
service code. `running → reconciled` is already legal, and `reconciled` is
already the honest "execution ended, proof pending" state.

So Phase C does not invent a lifecycle. It:

1. makes execution-end reliably reach `reconciled` from runtime facts;
2. retains artifacts at that moment;
3. adds `reconciled → succeeded`, gated on proof.

## The arrows

### 1. execution ended

**Condition.** Runtime facts say the session is gone: `ports.RuntimeFacts`
reports `ProbeDead` for the runtime, or a provider terminal observation is
recorded. `ProbeFailed` is explicitly **not** a death conclusion and never
advances this arrow.

**Refusal.** Ambiguity keeps the attempt `running`. A quiet provider is not a
dead one, and `refuseUnprovenCustody` already holds the fence in that case.
Never derive execution-end from a transcript marker, prose, or a bare process
exit code alone.

**Result.** `running → reconciled`. This says the process is over and says
nothing about whether the work is right.

### 2. retained artifact

**Condition.** The owned workspace is snapshotted and its content digested, so
what was produced is a Kennel fact rather than a provider claim.

**Refusal.** If the workspace cannot be read, or holds content the snapshot
cannot represent, the attempt stays `reconciled` with retention recorded as
failed. It does **not** advance, and cleanup does not run — failed cleanup must
leave inspectable debris rather than delete unknown work.

### 3. verification

**Condition.** Evidence and Verification bound to this exact Attempt and
WorkUnit, covering the unit's criteria, evaluated by the existing
`criterionReady`.

**Refusal.** Outcome-level proof, stale proof past the horizon, or proof whose
lineage belongs to another Attempt or unit does not count — and is not accepted
merely to unblock scheduling. An agent's own submission is a **claim** and is
labelled as one; only an independently observed check is labelled independent.

### 4. eligible success

**Condition.** Runtime-terminal (`reconciled`) **and** the unit's proof
satisfied.

**No circular gate.** This is the part worth being explicit about. Success is
derived from a runtime fact plus a proof fact, and **proof never consults the
Attempt's own success**: `workUnitProven` matches evidence and verification
lineage against Attempt identity, not against Attempt status. So there is no
path where success is required to establish the proof that establishes success.

**Refusal.** Proof absent, failing, or contradictory leaves the attempt
`reconciled`. That is a truthful resting state, not a failure, and the owner can
see exactly which criteria are unmet.

**Result.** `reconciled → succeeded` — the transition that needs a migration.

### 5. custody handling

**Condition.** The fence releases when the attempt reaches a terminal status
that is *accounted for*: `succeeded`, `failed`, `cancelled`, or an
owner-reconciled `lost`.

**Refusal.** Custody is never released because the provider went quiet. It is
also never released while retention has failed, because the next attempt would
start from a workspace nobody has accounted for.

### 6. downstream admission

**Condition.** The downstream unit's dependencies are all proven, the fence is
free, and the exact retained upstream artifact and base revision are handed over
with provenance.

**Refusal.** A missing or digest-mismatched upstream artifact blocks admission.
Starting a fresh worktree from the original branch is **not** a handoff and does
not satisfy this. A cancelled or unaccounted predecessor never auto-spawns a
successor.

## Crash points

The sequence is resumable at every arrow, because a daemon restart between any
two steps must resolve to a truthful state rather than a guess.

| Crash between | State found on restart | Resolution |
|---|---|---|
| execution ended, before `reconciled` is written | `running`, session gone | Liveness evaluation observes the dead runtime and completes arrow 1. Duplicate observations are idempotent. |
| `reconciled`, before retention | `reconciled`, no retained artifact | Retention is retried. It reads the workspace, which is still there because cleanup never ran. |
| retention, partially written | `reconciled`, retention marked incomplete | Incomplete retention is not a retained artifact. Re-run overwrites the incomplete record; a *verified* artifact is frozen and is never overwritten. |
| retention done, before verification | `reconciled` with artifacts | Nothing to resume: verification is owner- or check-driven, not automatic. |
| verification recorded, before `succeeded` | `reconciled`, proof satisfied | Reconciliation re-derives eligibility and completes arrow 4. Deriving it again is safe because it is a pure function of durable facts. |
| `succeeded`, before custody release | `succeeded`, fence held | Release is idempotent; `releaseCustody` already tolerates releasing without holding. |
| custody released, before downstream admission | fence free, upstream proven | The scheduler admits the next unit exactly once, under the existing replay key. |

## Other paths that must not advance the sequence

- **Failed check.** Verification recorded as failing. The attempt stays
  `reconciled`; the unit is retryable; no retry loop runs on its own.
- **Duplicate observation.** Observations are append-only and ordered by `Seq`;
  re-ingesting one changes no state. Out-of-order arrival cannot regress a
  status, because transitions are validated against the current status.
- **Duplicate submission.** Same request key and same canonical semantics
  returns the same Attempt; different semantics conflicts. Already enforced on
  both the service fast path and the SQLite race path.
- **Owner cancellation mid-sequence.** Cancel stops *continuation*, not just a
  process. A cancelled attempt never advances to success even if proof arrives
  afterwards, and never auto-spawns a successor.
- **Contradictory runtime state.** Stays blocked until reconciled. Unknown is a
  valid answer and the only safe one.

## Owner Acceptance

No arrow in this sequence creates an `AcceptanceDecision`, and no agent may.
`succeeded` plus satisfied proof produces **Ready for Review**. Acceptance is a
separate, explicit owner decision against the reviewed revision, and provider
completion, a green check, an export or a process exit never substitutes for it.

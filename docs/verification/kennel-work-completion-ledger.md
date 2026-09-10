# Kennel Work completion — implementation and evidence ledger

Opened 2026-09-09 for the assignment in
[`docs/superpowers/plans/2026-09-09-claude-code-kennel-work-completion.md`](../superpowers/plans/2026-09-09-claude-code-kennel-work-completion.md).
The 2026-09-10 Luna continuation is recorded in
[`docs/superpowers/plans/2026-09-10-luna-kennel-work-completion.md`](../superpowers/plans/2026-09-10-luna-kennel-work-completion.md).

This ledger is durable working state so context compaction cannot reset the
task. It records what already exists, what each phase changes, and at what
evidence level each claim stands. **Nothing here is owner acceptance or a
release decision.**

## Proof levels

These are separate outcomes and are never collapsed into one green checkbox:

| Level | Meaning |
|---|---|
| `source` | Source inspected. Proves code exists, not that it behaves. |
| `automated` | A behavioral test reproduces it. Narrow or suite. |
| `daemon-fixture` | Real daemon + real SQLite/HTTP, with a **test-only** fixture reasoning/execution provider. Not provider conformance. |
| `real-provider` | A live configured provider actually served the call. |
| `packaged` | Observed in the built desktop package. |
| `owner` | The owner accepted it. Only the owner can reach this level. |

A missing runtime, credential or dependency is **blocked**, never pass. Test
failures are classified by reproduction against the base commit, never by
whether a file was edited. Current date and slice labels are not evidence.

## Branch and base

| Fact | Value |
|---|---|
| Branch | `codex/kennel-work-completion` |
| Worktree | `~/.codex/worktrees/kennel-work-completion` |
| Base commit | `c83684c11627c5851c5a5f2a23e96264b9b9498b` (Wednesday implementation HEAD) |
| Remote `origin/beta` at fetch time | `9396c3844ee00cf2c350df0426f4224d33ef87de` — re-fetched and unchanged |
| Base vs remote beta | base is exactly 5 commits ahead; **no divergence**, so no rebase or cherry-pick was needed |
| Inherited dependency commits | L2 `0b867790c`, L3 `684b9c0a9`, L4 `170230bf0`, fixes `d617b4ee3`, evidence `c83684c11` |

The local `beta` ref (`9c15272d4`) is an *ancestor* of remote beta and is stale;
it is not the fork point. The originating `codex/wednesday-milestone-2026-09-09`
checkout keeps its uncommitted working copy untouched.

## Baseline on the base commit, before any implementation

Measured on this branch with only the inherited-planning docs commit applied,
so every result below is inherited from `c83684c11`.

| Gate | Result | Evidence |
|---|---|---|
| `npm run bootstrap` | pass (exit 0) | `00-bootstrap.log` |
| `go build ./...` | pass (exit 0) | `01-go-build.log` |
| `go vet ./...` | pass (exit 0) | `02-go-vet.log` |
| `go test ./...` | **fail (exit 1)** — 1 package | `03-go-test.log` |
| `npm run lint` (runs `go test ./...` then golangci-lint) | pass (exit 0), `0 issues` | `04-lint.log` |
| `npm run frontend:typecheck` | pass (exit 0) | `05-fe-typecheck.log` |

**Resolution of B1:** fixed at its cause in `db3ad8c87`, not by loosening the
assertion. See the A1 rows below.

### Inherited baseline defect B1 — flaky reasoning-timeout test

`TestCompleteHonorsCancellationAndTimeout` in
`backend/internal/adapters/llm/openai/client_test.go` is **flaky, not broken**:
it failed under `go test ./...` and passed under `npm run lint` in the same
working tree. Direct repetition on the base commit gives 5 failures in 8 runs.

```
client_test.go:97: error = waldo reasoning call failed: Post "http://127.0.0.1:.../v1/responses":
  net/http: request canceled (Client.Timeout exceeded while awaiting headers), want deadline
```

Cause: the test asserts on **error-string text** (`strings.Contains(err.Error(),
"deadline exceeded")`) for a failure produced by a race between two timeout
mechanisms — the injected `http.Client.Timeout` and the provider SDK's own
request context deadline. Whichever fires first decides the wording, so the
assertion is timing-dependent. The Anthropic adapter carries the identical test
and happens to win the race consistently.

This classifies as an inherited defect reproduced on the base commit, and it is
**not** merely a fixture problem: it exposes that reasoning failures carry no
stable classification at all (see requirement A1-1). The Wednesday branch's
recorded `go test ./...` exit 0 was a passing sample of a flaky test, so that
package's prior evidence is unreliable rather than false.

## Requirement delta map

Requirement IDs are local to this ledger and referenced by commits and tests.

### Phase A — verify and finish L2/L3 foundations

| ID | Requirement | Existing source | State | Remaining change | Behavioral test | Proof level |
|---|---|---|---|---|---|---|
| A1-1 | Reasoning failures carry a stable, typed classification | none — `service/intelligence/llm.go:241,353` and both adapters returned bare `fmt.Errorf` prose | **done** `db3ad8c87` | `ports.ReasoningFailure` + `ClassifyReasoningTransport`; both adapters classify at the edge; `service/intelligence/failure.go` maps to stable codes once | `adapters/llm/*/client_test.go` assert `Kind` per path; `failure_test.go` covers the mapping | automated |
| A1-2 | Missing reasoning setup returns a typed actionable error, not `500 INTERNAL_ERROR` | `MISSING_CREDENTIAL` existed only as a *readiness status field*; propose endpoints raised untyped prose | **done** `db3ad8c87` | service-layer only — `envelope.WriteError` already renders `apierr` from anywhere in the chain, so no controller changed. `apierr.KindUnavailable`/503 added for upstream faults | `outcomes_reasoning_failure_test.go` asserts code + status + `retryable` through the real router | automated |
| A1-3 | Configured / key-present is separate from verified readiness | `ready` was set purely because the credential string was non-empty | **done** `06e3234fb` | `Verified`/`VerifiedAt` + owner-triggered `POST /settings/reasoning/verification`; migration **0118** binds verification to the exact provider/model pair | `reasoning_verification_test.go`; real probe exercised against a local stand-in in `waldo_reasoning_probe_test.go` | automated (live provider **blocked**) |
| A1-4 | Bounded retries and truthful terminal state on timeout/cancel/refusal/auth/rate-limit/invalid-output | adapter tests asserted only `calls == 1`; every failure persisted the same flat `INTELLIGENCE_PROVIDER_FAILED` | **done** `db3ad8c87` | classified reason persisted per failure; `TerminalizationContext` survives caller cancellation so a cancelled request no longer leaves the run `running` | `failure_test.go` cancelled-caller cases; adapter per-kind tests | automated |
| A1-5 | Restart reconciliation leaves no run visibly active and never auto-repeats a billable call | `ReconcileInterruptedRuns`; SQLite terminal-immutability and monotonic-provenance guards | **verified, inherited** | none — independently reproduced, not taken on report | `TestReconcileInterruptedRunsExpiresWithoutRetryingProvider`, `TestIntelligenceRunStoreRoundTripAndTerminalImmutability`, `…EffectiveProvenanceIsMonotonic` all pass and assert what their names claim | automated |
| A1-6 | No cross-provider credential transfer | provider-bound file secret store | **verified, inherited** | none — independently reproduced | `TestSetReasoningDoesNotReuseCredentialWhenProviderChanges` is a real canary test (Anthropic key never populates OpenAI); `TestFileStoreBindsCredentialsToProvider` | automated |
| A1-7 | The machine code the UI switches on does not come from error prose | `reasoningError` derived its code with `strings.Contains` over its own messages | **done** `06e3234fb` | classification by `Kind`, named sentinels for local setup errors | `TestReasoningErrorCodesComeFromSentinelsNotMessageText` — red-green: a classified rejection previously answered `REASONING_NOT_READY` | automated |
| A2-1 | Grounding is bounded, secret-safe, symlink-confined, cancellable, fail-closed on Git error | `service/intelligence/repository_context.go` | **verified independently** | none | real secret canaries + symlink escape + real `git init` in `TestBuildRepositoryContextBoundsFilesAndExcludesIgnoredSymlinkedSecrets`; `TestIgnoredByGitDistinguishesNotIgnoredFromGitFailure` plus the fail-closed call site at `repository_context.go:130` | automated |
| A2-2 | Non-repository (supplied document) context with an explicit custody model | **repository-only today** | **missing — deferred by owner decision** | specified in [the A2 deferred-scope plan](../superpowers/plans/2026-09-09-a2-supplied-document-context.md); resumes after the B–D checkpoint | a document Outcome grounds from supplied files and is refused when unsupported | **deferred** |
| A2-3 | Bounded context enters both Contract and Plan input digests | Contract digest includes the snapshot (per `d617b4ee3`) | **partial — deferred with A2-2** | confirm the Plan digest too | changed context changes both digests | **deferred** |
| A2-4 | Unresolved assumptions/blockers are surfaced before approval | migration 0117 stores them; Plan API returns them | **partial — deferred with A2-2** | surface in the approval surface rather than discarding | approval shows unresolved blockers | **deferred** |

**A2 deferral.** The owner chose on 2026-09-09 to land Phases B, C and D before
A2's supplied-document capability, to get the working software loop in hand
first. A2 remains in assignment scope. Its safety half (A2-1) was *not*
deferred and is verified above. The three conditions that keep the deferral from
creating a retrofit — no document control in B, a staged-workspace custody model
in C, and D's document row recorded blocked — are recorded in the A2 plan and
are binding on those phases.

### Phase B — Board/List and Mission Control

| ID | Requirement | Existing source | State | Behavioral test | Proof level |
|---|---|---|---|---|---|
| B2-1 | A direct Outcome has a real WorkUnit DAG, separate from the decomposition graph | `DecompositionGraph.tsx` rendered contributing Outcomes only | **done** `129cfe7d3` | `MissionWorkUnitGraph.test.tsx` — 13 cases incl. the assignment falsifier (plan serializes B before A, B depends on A, A still ordered first) | automated |
| B2-2 | Layering is shared, no graph dependency added | `layerContributions` was local to the decomposition graph | **done** `129cfe7d3` — extracted to `lib/dependency-layers.ts`; no reactflow/dagre/elkjs added | `dependency-layers.test.ts` — 9 cases incl. cycle and unknown-upstream bounds | automated |
| B2-3 | Proposed topology before authorization; daemon overlay after | plan cards only | **done** `129cfe7d3` | graph carries `data-state="proposed"` with no schedule | automated |
| B2-4 | The schedule distinguishes waiting-for-proof, custody held and paused, and always explains an empty runnable set | `blocked` conflated dependency proof with the custody fence; no reason for an empty runnable set | **done** `2464865aa` | `scheduler_reasons_test.go` — 5 cases | automated |
| B2-5 | No raw IDs or inline English as primary content | `Next: ${workUnitId}`, criterion/dependency ID lists, 5 hardcoded English strings | **done** `159ccb873` | test asserts no `wu-` id reaches the graph face; 31 keys added across all 8 locales | automated |
| B1 | Board and List as equivalent Outcome projections with filters and columns | `OutcomesOverviewSurface.tsx` is a flat project-grouped list | **open** | — | — |
| B3 | Shared Mission header, one primary next action, decision history | partial | **open** | — | — |

### Phase C — truthful terminal state and artifact continuity

| ID | Requirement | Existing source | State | Behavioral test | Proof level |
|---|---|---|---|---|---|
| C-0 | The whole sequence is specified before the code | none | **done** `5670e1cc7` | [the sequence contract](2026-09-09-execution-to-admission-sequence.md) — every arrow has a named refusal, 7 crash points resolved | source |
| C-1 | `succeeded` is reachable | **unreachable**: migration 0102's trigger permits no transition into it, and `LegalAttemptTransitions` had no entry | **done** `67f6daa74` — migration **0119** recreates the trigger with `reconciled → succeeded` | `TestOnlyAnEndedAttemptCanBecomeSucceeded` | automated |
| C-2 | Success is never assigned from a live process | — | **done** `67f6daa74` | `running → succeeded` still rejected, as are queued/paused | automated |
| C-3 | A durable record of what an attempt produced | none — no `WorkUnitReceipt`/`SessionReceipt` type existed | **done** `67f6daa74` | `attempt_receipt_store_test.go` — 6 cases | automated |
| C-4 | The receipt satisfies a delivery manifest without retrofit | — | **done** `67f6daa74` — lineage, paths + digests, context identity, base/result revision, retention state, immutable artifact version | round-trip test asserts lineage and revisions survive | automated |
| C-5 | Custody shape is recorded, not assumed; a staged folder is not a worktree | — | **done** `67f6daa74` — keeps A2-2 additive | `TestStagedFolderReceiptCannotClaimRevisions` | automated |
| C-6 | A frozen receipt cannot be overwritten | — | **done** `67f6daa74`, hardened in R3 — refused in the write path and by SQL triggers for every parent/file mutation | `TestFrozenAttemptReceiptRefusesReplacement` plus R3 SQL/store regressions | automated |
| C-7 | Partial retention is reported as partial | — | **done** `67f6daa74` — only `retained` satisfies a handoff | `TestIncompleteRetentionIsRecordedAsIncomplete` | automated |
| C-8 | Artifact paths cannot escape custody | — | **done** `67f6daa74` | `TestArtifactPathCannotEscapeTheWorkspace` | automated |
| C-9 | Execution end reaches `reconciled` from runtime facts | **already existed** in `EvaluateAttemptLiveness` | **verified, inherited** | health-gated; `ProbeFailed` is not a death conclusion | automated |
| C-10 | An ended attempt is classified from proof bound to that exact attempt | none | **done** (this commit) | `terminal_test.go` — 7 cases | automated |
| C-11 | Classification never borrows another attempt's proof | — | **done** (this commit) | `TestAttemptProvenDoesNotBorrowAnotherAttemptsProof` — red-green shown inline: `workUnitProven` answers yes for the pair, `attemptProven` only for the producer | automated |
| C-12 | Artifact retention actually snapshots the workspace | `backend/internal/artifactstore` plus daemon reconciliation wiring | **implemented, full acceptance open** `888f8f568` | `internal/artifactstore/store_test.go` — staged bytes/mode, committed+dirty Git output, secret/symlink refusal, composition/apply; daemon/store/outcome focused regressions | automated; daemon-restart/packaged acceptance open |
| C-13 | A downstream WorkUnit receives the exact retained upstream artifact | `backend/internal/artifactstore/handoff.go` composition primitives | **partial/open** `888f8f568` | composition conflict/base checks and byte/mode verification pass; canonical successor admission/provisioning and input-version snapshot remain | automated primitive only |

### Correction slice R — review closure

| ID | Requirement | State | Evidence | Proof level |
|---|---|---|---|---|
| R1 | Classification, receipt freeze, observation and custody release are one conditional persistence operation | **done** `646de9f73` | `/tmp/kennel-work-r-final-backend.log`; SQLite finalizer regression | automated |
| R2 | Succeeded Attempts with complete frozen custody are restart-repairable without authorizing work | **done** `646de9f73` | `accountSucceededCustody`; focused outcome tests | automated |
| R3 | Receipt/file immutability, lineage checks and coherent reads | **done** `646de9f73` | migration 0120; focused store tests | automated/source |
| R4 | Artifact identity includes semantic mode and component-safe path validation | **done** `646de9f73` | focused domain/store tests | automated |
| R5 | Verification is bound to a generation and non-secret effective-input fingerprint; owner-triggered Verify is visible | **done** `646de9f73` | `/tmp/kennel-work-r5-frontend-typecheck-final.log`; settings backend regression | automated/source |

### Deferred-phase gaps established during planning

- **B2** — there is no graph library in `frontend/package.json` and
  `DecompositionGraph.tsx` renders the contributing-Outcome graph, so the direct
  WorkUnit DAG is **missing**. Its `layerContributions` longest-path layering is
  reusable and will be extracted rather than adding a dependency.
- **B1** — `OutcomesOverviewSurface.tsx` is a flat project-grouped list:
  no Board, filters, columns, milestone, blocker or proof summary.
- **D** — there is no check runner. `internal/process/command.go` is a
  two-function `exec.Cmd` wrapper and enforces nothing; the real enforcement
  boundary is the immutable `AttemptExecutionPolicy` mapped to provider
  mechanisms inside the adapters.
- **E** — there is no export/deliver/patch/bundle helper anywhere in the
  backend.

## Live verification boundaries

Recorded **blocked**, at the boundary that actually blocks, per owner decision:

| Path | Boundary |
|---|---|
| Live reasoning conformance | no configured live reasoning credential in the isolated profile |
| Model-generated terminal execution | `~/.local/bin/codex-code-mode-host` is absent |

Fixture reasoning and execution providers are **test-only**: registered from
test/dev wiring, never in the production registry, unreachable from normal
provider selection, and no production fallback resolves to them. Their evidence
is labelled `daemon-fixture` and never described as live conformance.

## Phase log

| Phase | Commits | Gates run | Outcome |
|---|---|---|---|
| Step 0 isolate | `d675f8b53` planning baseline, `f180541e2` this ledger | bootstrap, build, vet, `go test ./...`, lint, frontend typecheck | baseline established; inherited defect B1 recorded |
| A1 classification | `db3ad8c87` | `go test ./...` exit 0; `-race -count=3` on touched packages exit 0; vet, build, lint `0 issues`; `npm run api` no diff | A1-1, A1-2, A1-4 closed. Inherited flake B1 fixed at its cause: openai package 8/8 stable, was 3/8 failing. Two inherited test-quality defects found and fixed — the refusal/malformed/incomplete rows in both adapters never reached the code they named |
| B2 graph + schedule reasons | `2464865aa`, `129cfe7d3`, `159ccb873` | `go test ./...` exit 0; frontend 232 files / 2798 passed / 6 skipped (was 230/2776); typecheck clean; lint `0 issues`; api regenerated | B2 closed. No graph dependency added. An `unresolved` schedule state was written and then removed: deriving it from `AttemptLost` would have misreported an attempt the owner had already reconciled |
| C foundation + classification | `5670e1cc7`, `67f6daa74`, this commit | `go test ./...` exit 0; `-race -count=2` on touched packages; vet, build, lint `0 issues`; migration **0119** ledgered; sqlc regenerated | C-1 … C-11 closed. C-12 retention adapter and C-13 handoff remain open |
| A1 readiness | `06e3234fb` | `go test ./...` exit 0; vet, build, lint `0 issues`; frontend typecheck exit 0; sqlc + api regenerated with sources, no further drift | A1-3, A1-7 closed. Migration **0118** added (not an amendment to unmerged 0116: the owner may already have applied it locally). A1-5/A1-6 independently reproduced from inherited tests |
| R corrections | `646de9f73` | focused Go R regressions and frontend typecheck pass; migrations 0120/0121 and sqlc generated locally | R1–R5 implemented; live provider verification remains blocked |
| C retention | `888f8f568` | `go test ./internal/artifactstore ./internal/storage/sqlite/store ./internal/service/outcome ./internal/daemon` pass | C-12 adapter and C-13 composition primitives implemented; canonical successor admission remains open; [checkpoint](2026-09-10-c-retention-checkpoint.md) |

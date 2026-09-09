# Kennel Work completion — implementation and evidence ledger

Opened 2026-09-09 for the assignment in
[`docs/superpowers/plans/2026-09-09-claude-code-kennel-work-completion.md`](../superpowers/plans/2026-09-09-claude-code-kennel-work-completion.md).

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
| A1-1 | Reasoning failures carry a stable, typed classification | none — `service/intelligence/llm.go:241,353` and both adapters return bare `fmt.Errorf` prose | **missing** | classify into stable codes via the canonical `httpd/apierr` error type | each failure path asserts its code, not a substring | pending |
| A1-2 | Missing reasoning setup returns a typed actionable error, not `500 INTERNAL_ERROR` | `MISSING_CREDENTIAL` exists only as a *readiness status field* in `service/settings/service.go:249`; propose endpoints raise untyped prose | **missing** | map through controllers with one `errors.As`; renderer offers a settings action and preserves the draft | HTTP test asserts typed code + status on a no-credential propose | pending |
| A1-3 | Configured / key-present is separate from verified readiness | `service/settings` distinguishes configured / key-present / ready | **partial — to verify** | confirm `ready` is not merely a nonempty check; add an explicit owner-triggered probe if it is | readiness with a present-but-unverified key is not `ready` | pending |
| A1-4 | Bounded retries and truthful terminal state on timeout/cancel/refusal/auth/rate-limit/invalid-output | adapter tests exist and assert `calls == 1` with retries disabled | **partial** | fix B1 by asserting classification; persist terminalization under a cancelled caller context via a bounded cleanup context | each path terminalizes with its stable reason | pending |
| A1-5 | Restart reconciliation leaves no run visibly active and never auto-repeats a billable call | `service/intelligence/recovery.go` `ReconcileInterruptedRuns` | **partial — to verify** | verify late responses cannot overwrite a newer revision or a terminal run | late response against terminal run is refused | pending |
| A1-6 | No cross-provider credential transfer | provider-bound file secret store, `secretstore/file.go` | **partial — to verify** | reverify behaviorally with canaries | provider switch without its key reports missing credential; other key never used | pending |
| A2-1 | Grounding is bounded, secret-safe, fail-closed | `service/intelligence/repository_context.go` | **partial — to verify** | reverify behaviorally: nested pruning, symlink/root confinement, cancellation during traversal *and* reads, Git failure fails closed | negative tests per assignment §4 | pending |
| A2-2 | Non-repository (supplied document) context is supported with an explicit custody model | **repository-only today** | **missing** | explicit supported-local-file context path reusing existing attachment staging + Project Brief; defined formats; refuse unsupported before launch; never silently `git init` | a document Outcome grounds from supplied files and is refused when unsupported | pending |
| A2-3 | Bounded context enters both Contract and Plan input digests | Contract digest includes the snapshot (per `d617b4ee3`) | **partial** | confirm the Plan digest too | changed context changes both digests | pending |
| A2-4 | Unresolved assumptions/blockers are surfaced before approval | migration 0117 stores them; Plan API returns them | **partial** | surface in the approval surface rather than discarding | approval shows unresolved blockers | pending |

### Phases B, C, D

Delta maps are added at the start of each phase, after its own source
inspection. Recorded gaps established during planning:

- **B2** — there is no graph library in `frontend/package.json` and
  `DecompositionGraph.tsx` renders the contributing-Outcome graph, so the direct
  WorkUnit DAG is **missing**. Its `layerContributions` longest-path layering is
  reusable and will be extracted rather than adding a dependency.
- **B1** — `OutcomesOverviewSurface.tsx` is a flat project-grouped list:
  no Board, filters, columns, milestone, blocker or proof summary.
- **C** — **nothing in production writes `domain.AttemptSucceeded`.**
  `service/outcome/recover.go` writes only `Lost`/`Reconciled`;
  `attempt.go:413` writes only `Cancelled`. There is no `WorkUnitReceipt` or
  `SessionReceipt` type. This is the single blocking gap for artifact handoff,
  checks, continuation and delivery.
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
| Step 0 isolate | `d675f8b53` planning baseline, this ledger | bootstrap, build, vet, `go test ./...`, lint, frontend typecheck | baseline established; inherited defect B1 recorded |

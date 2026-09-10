# Independent review and Claude continuation brief

Reviewed HEAD: `ae7aa569007bcd717bbf7bd67ddbe1cedc01f671`.
Fixed comparison: `55d916cec973473cf965ece7bc69ab230432955d...ae7aa569007bcd717bbf7bd67ddbe1cedc01f671`.
Worktree: `/Users/shivanshfulper/.codex/worktrees/kennel-work-completion`.
Date: 2026-09-10. Result: **requires fixes; implementation remains partial**.

This review covers Luna's delta from the previous reviewed checkpoint, not a fresh audit of all changes since beta. Repository status was clean at review start. No live provider, package launch, push, merge or live-profile mutation occurred. This review document is the only retained addition.

## Evidence gathered by the primary reviewer

Existing focused suites passed:

```sh
cd backend
go test ./internal/artifactstore ./internal/governedcheck ./internal/service/outcome ./internal/service/settings ./internal/storage/sqlite/store ./internal/service/intelligence
```

Some existing test results were cached. Full frontend, full race, package build and executable hash from Luna's report were not independently repeated in this review. Source inspection confirms useful new transactional finalization, stronger receipt guards, mode-aware identity, retained content and readiness UI work; that does not close every R requirement.

Four temporary regression checks were executed with `-count=1` and failed:

1. Call `governedcheck.Run` with the existing `checkPolicy()`, a temporary workspace, and `touch` targeting a different temporary directory. The outside file was created and Run returned nil error.
2. Retain a staged output named `KENNEL-EXPORT.json`, export it as draft to an empty temporary directory, and compare output bytes. Export returned success after replacing the original artifact with metadata.
3. Retain a five-byte regular file using `Config{MaxBytes:4}`. Expected a persistable incomplete receipt; got `retained receipt invalid: artifact file "large.txt" is missing content identity`.
4. Parse NUL-delimited porcelain `??  report.txt \x00`. Expected filename ` report.txt `; actual map key was `report.txt`.

Temporary test sources were removed. These need permanent regression tests in the fix commits. An additional static symlink-parent canary did not reproduce an outside read; no reproduced symlink exploit is claimed.

## Standards axis

Manifest-backed checking is unavailable: no effective standards manifest was established. Rules below are descriptive identifiers for current AGENTS.md boundaries, not invented versioned manifest rules. Reviewer: standards_review, with primary Codex validation where specified. All findings are open; no waivers identified. Source-only crash/cancellation findings need the specified behavioral tests.

| ID / rule | Severity / confidence | Evidence at reviewed HEAD | Required correction |
|---|---|---|---|
| ST1 / truthful bounded receipts | P1 / high, reproduced | `backend/internal/artifactstore/store.go:278` appends a byte-limited file without digest or unsupported reason; domain validation rejects the receipt. | Preserve a valid incomplete record with the specific declined-content reason; test Retain through receipt persistence. |
| ST2 / effect fencing and recovery | P1 / high, source | `backend/internal/governedcheck/runner.go:79`, `process_unix.go:10`: Setpgid creates a group, but default CommandContext cancellation kills only its leader. No bounded pipe wait is configured. | Implement supported process-tree termination and bounded wait; prove descendants stop and Run returns on timeout/cancel. Unknown termination must retain custody. |
| ST3 / exact artifact provenance | P2 / high, reproduced | `backend/internal/artifactstore/export.go:103` overwrites payload named KENNEL-EXPORT.json and returns success. | Separate payload/metadata namespaces or reject collisions before writes; verify exported payload against retained manifest. |
| ST4 / exact artifact provenance | P2 / high, reproduced | `backend/internal/artifactstore/store.go:427` trims legal filename whitespace from porcelain paths. | Remove exactly the format separator, preserve path bytes; exercise real Git retention with leading/trailing spaces and newline names. |
| ST5 / restart-safe durable receipts | P1 / high for source defect | `backend/internal/artifactstore/store.go:530` syncTree syncs only the top directory, not file contents/nested directories, before publication and DB finalization. | Sync supported content/directory publication in the proper order before durable success; distinguish process restart tests from filesystem crash guarantees. |

Additional source concerns to investigate before wiring production: lexical confinement does not protect against concurrent intermediate symlink replacement; Git command output/path collection is unbounded before final manifest truncation; bytesTotal repeatedly walks accumulated content. Do not report an exploit or performance measurement without a matching reproducer. Implement bounds at the operation that allocates/reads, not only at the final result.

## Spec axis

Source: `docs/superpowers/plans/2026-09-10-luna-kennel-work-completion.md`. Reviewer: spec_review, with primary Codex validation where specified.

| ID | Requirement and finding | Correction |
|---|---|---|
| SP1 — P1 | D1 requires checks through the actual enforcement boundary. `governedcheck/runner.go:59–81` rejects some shell names then calls unrestricted exec.CommandContext. There is no real executable allowlist or filesystem/network sandbox. Outside write reproduced. | Do not wire this as governed execution. Implement actual supported enforcement or fail closed as unavailable. Shell denial/cwd/environment filtering does not enforce capabilities. |
| SP2 — implementation missing | C-13, A2 and E require production flows. `artifactstore/handoff.go:37`, `artifactstore/export.go:38`, `service/intelligence/supplied_context.go:23` are disconnected helpers. Full D intent/evidence/rework integration is also acknowledged open. | Wire canonical admission/input versions, supplied-context selection/proposal/staging, checks/run intent and delivery service/API/UI. These are coding tasks, not only pending acceptance tests. |
| SP3 — P1/P2 | E requires exact AcceptanceDecision/artifact binding. Export checks only owner/kind/OutcomeID (`export.go:49–52`), allowing an older decision for that Outcome to label another revision accepted. Metadata collision is also ST3. | Resolve canonical current acceptance and bind exact reviewed revision/artifact; test stale decision and same-Outcome wrong result. Implement actual portable delivery with recorded base/deletions, not just copied changed files. |
| SP4 — implementation partial | B1 requires fact-derived status columns and attention/history filtering. `OutcomesOverviewSurface.tsx:119` switches layout to a two-column grid; this is not a status-column board. Contributors still render by default. | Finish real Board/List semantics, top-level default, filtering and selection restoration. Complete B3 contextual actions/freshness with real daemon UI tests. |
| SP5 — P2 | F requires disabling Island background startup. Flag handling hides renderer controls, but `frontend/src/main.ts:2158` still initializes Island. | Apply launch configuration to Electron main-process startup as well as renderer navigation, then build and launch the focused package. |
| SP6 — P1 | A2 requires bounded safe reading. `supplied_context.go:60–68` checks size before unbounded os.ReadFile; concurrent growth/replacement can allocate/read beyond the declared limit before rejection. | Use bounded descriptor reads and identity/confinement checks; test growth/replacement and cancellation before wiring owner document ingestion. |

### Primary reviewer addition: proof is still not bound to retained bytes

`service/outcome/terminal.go` evaluates attemptProven first, then invokes retention, then reads the current receipt again before finalization. The finalizer checks that receipt's version/status/lineage, but receives no checked proof version/horizon and does not revalidate proof/current Contract in its transaction. Proof naming the Attempt does not establish that subsequently retained bytes were checked. A newer contradiction/revision can also arrive between proof read and final commit.

This leaves R1/D2 incomplete despite the improved atomic status/freeze/observation/fence transaction. Capture and freeze exact input/result identity at the correct point; bind Verification to that identity; carry and conditionally validate the relevant proof/revision generation into finalization. Test proof for V1 followed by changed output V2, a contradictory Verification racing with finalization, and Contract revision change. Do not simply insert a second nontransactional proof read and call the race fixed.

## Claude continuation assignment

Continue from this branch after verifying current HEAD/status and reading AGENTS.md. Preserve Luna's useful implementation. Read this review and the original 2026-09-10 Luna handoff. No push/merge/live-profile mutation or live model spend is authorized by this continuation brief.

1. Reproduce and fix ST1–ST5, SP1/SP3/SP5/SP6 and the proof-to-artifact binding gap. Add permanent tests at actual service/store/filesystem/process boundaries, not only helpers. Preserve/add migrations as required; do not rewrite already-applied history.
2. Finish C-13 and D as one end-to-end milestone: authorized A produces retained verified bytes; daemon restart; B is admitted exactly once with those bytes and base/modes/deletions; failed checks/pause/cancel/unknown custody stop continuation. `Apply` currently accepts an empty destination, so explicitly design composition with the approved repository base and shared ancestry rather than losing unchanged files.
3. Finish B1/B3 and expose actual daemon facts and next actions. Run the real-daemon fixture journey. A CSS grid and disconnected helper are not a completed product surface.
4. Wire A2 document Outcomes fully and E owner-triggered durable delivery. Test a supported document and repository task through approval, execution, checks, review, owner acceptance and usable export. Do not redefine the product as software-only to close the remaining rows.
5. Finish F, build with focused mode enabled, launch using isolated state, and complete G's matrix. Record live provider/OS/owner evidence separately. Missing infrastructure blocks the affected evidence, not independent implementation.

Commit fixes in reviewable slices and update STATUS/ledger to distinguish implemented adapter, production-wired behavior, automated proof, daemon fixture, real provider and packaged/owner acceptance. Do not call missing production code “acceptance-open” without also marking implementation partial/missing. Keep progressing without routine continue prompts. Return exact SHAs, tests, migrations, launch commands, package identity and remaining blockers for another review.

Standards: 5 findings; worst severity P1 (bounded retention/process custody/durability). Spec: 6 findings; worst severity P1 (unenforced checks/exact result binding/bounded input), plus the separately recorded primary proof-binding finding.

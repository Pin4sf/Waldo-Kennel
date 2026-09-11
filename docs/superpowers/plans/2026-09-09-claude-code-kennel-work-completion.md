# Claude Code execution prompt: complete the Kennel Work launch loop

Prepared 2026-09-09. This is an implementation assignment when the owner supplies it to Claude Code. It authorizes the remaining scope below, subject to repository invariants and the review boundary. It is not evidence that implementation or release acceptance has happened.

## Mission and working relationship

You are implementing the remaining Kennel Work product. Work through the bounded phases below, inspect and reuse existing code, and keep progressing through all authorized phases without requesting routine confirmations. The planning/review agent will review your final commits. The owner will then run and evaluate the product locally. Your task includes implementation, appropriate automated checks, and safe isolated local verification; the owner's later testing is not a substitute for your checks.

Do not push, merge, publish, deploy, create external PRs, spend on live model calls, change real credentials, delete user data, or alter the owner's live `~/.kennel` profile. You may create an isolated branch/worktree, edit code/docs, run local tests and builds, use synthetic credentials/local HTTP fixtures, and make reviewable local commits. Network needed for ordinary dependency tooling is allowed; do not add dependencies without first checking existing facilities. Ask only for genuinely missing product decisions or authorization for actions outside this ceiling. Continue independent work when a live test is blocked.

Implement the supported local Work loop, not every future Waldo feature. Do not remove safeguards or replace real behavior with mocks to make it appear complete. Explain a blocking architectural conflict before proceeding across it.

## 1. Establish the correct starting point

Known baseline, to reverify:

- Repository: `/Users/shivanshfulper/Developer/Pin4sf/Waldo-Kennel`.
- Existing implementation worktree: `/Users/shivanshfulper/.codex/worktrees/wednesday-milestone`.
- Branch: `codex/wednesday-milestone-2026-09-09`.
- Integrated beta last checked at `9396c3844ee00cf2c350df0426f4224d33ef87de`.
- Wednesday implementation HEAD last checked at `c83684c11627c5851c5a5f2a23e96264b9b9498b`, five commits ahead of that beta.
- Local commits include L2 `0b867790c`, L3 `684b9c0a9`, L4 `170230bf0`, fixes `d617b4ee3`, evidence `c83684c11`.
- The worktree additionally contains intended, uncommitted planning updates: `docs/STATUS.md`, `docs/product/2026-09-08-outcome-control-plane-mvp-reset.md`, `docs/superpowers/plans/2026-09-08-outcome-control-plane-mvp-reset.md`, and the new `docs/product/2026-09-09-kennel-work-launch-experience.md`, plus this prompt.

Inspect status, refs, remotes and recent commits before any mutation. Fetch beta and compare ancestry; do not assume it is unchanged. Do not start from old beta and accidentally omit the Wednesday code. If beta advanced, inspect compatibility and preserve both changes. Do not reset/clean/stash someone else's changes.

Prefer a new isolated worktree on `codex/kennel-work-completion` based on the verified Wednesday implementation HEAD, or an unused similarly named branch. Copy only the explicitly listed planning documents into the new worktree after inspecting them; the originating checkout must stay intact. Commit the copied planning baseline separately so the reviewer can distinguish inherited plans from implementation. If an existing completion branch is present, inspect it before creating duplicates. Record exact base/dependency SHAs. Do not cherry-pick a mixed later commit blindly onto an L2-only base.

Assignment history does not erase working code. L2 and repository-focused L3 are substantially present; L4 has Plan cards, schedule API and CDC work. Review and finish the gaps, not rewrite everything. The current Mission graph is primarily decomposition-oriented; it does not prove a direct WorkUnit DAG experience exists.

## 2. Read authority and produce a delta map

Read `AGENTS.md` and follow its current authority order, including ADRs 0010/0011/0012, `docs/product/kennel-v1-product-architecture.md`, ADRs 0008/0009, the product reset and UX/debt audit. Then read:

- `docs/product/2026-09-09-kennel-work-launch-experience.md` — desired owner flow and delivery contract.
- `docs/superpowers/plans/2026-09-08-outcome-control-plane-mvp-reset.md` — existing per-slice requirements and tests.
- `docs/STATUS.md` and `docs/verification/2026-09-09-wednesday-followup.md` — evidence and unresolved claims.
- The two 2026-08-25 Work specifications where the current authority links to them.

This assignment authorizes the phases below in order, superseding earlier handoff wording requiring another assignment between L2/L3/L4. It does not supersede domain, custody, permission, verification or owner-acceptance boundaries.

Create `docs/verification/kennel-work-completion-ledger.md` with a concise delta map:

Requirement ID | existing source | present/partial/missing | remaining change | behavioral test | proof level.

Use separate proof levels: source inspected, automated, real daemon with fixture provider, real provider, packaged desktop, owner accepted. Test failures must be classified by reproduction, not by whether a file was edited. Current date/slice labels are not evidence. Update the ledger after each phase.

## 3. Product promise and non-negotiable behavior

Kennel helps an owner turn a desired Outcome into a verified, usable result while retaining control. Outcomes are general responsibilities: a report based on supplied documents, an analysis, or a software change are valid examples. Do not hardcode software criteria, Git commands, or code-review language for all Outcomes.

The loop is:

Project/context → Outcome/Contract → proposed Plan → owner authorization → governed WorkUnits → retained artifacts and evidence → review/rework → owner Acceptance → usable delivery.

Primary navigation:

Work Board/List of Outcomes → one Mission Control → WorkUnit details → optional Attempt/Session inspector.

Board/List are two views of the same responsibilities. Mission Control contains Contract, Plan/Graph and Evidence/Result with contextual questions and decisions. Existing five-stage surfaces may be reused inside it; avoid competing navigation and duplicate Work applications. A transcript is never required for ordinary supervision.

Keep these boundaries:

- Go daemon + SQLite is canonical authority. React/CLI/providers are clients.
- Models propose; deterministic policy validates, authorizes, schedules and binds evidence.
- Provider completion, process exit, commits, green checks and exports never create owner Acceptance.
- Plans are immutable revisions. Stale proposals/proof cannot bind new authority.
- A direct Outcome has a WorkUnit DAG; a decomposed parent has contributing Outcomes. Do not mix graph node semantics.
- Approved provider/model and capability authority survive launch, retries and recovery. No fallback or mutable Project override.
- Retain current serial custody fencing. A branching graph does not imply parallel execution.
- Unknown prior process/effect blocks duplicate execution. Cleanup never force-deletes dirty work.
- Trigger CDC remains canonical event authority. Do not add a parallel event/status database.
- API is code-first; regenerate OpenAPI and frontend contracts together. Never edit sqlc output manually or rewrite merged migrations.

Launch focus excludes Home, Island/notch startup and standalone Waldo side chat from the normal experience. Preserve their code/data and historical readability. Keep Outcome-local clarification/feedback. Parallel scheduling, automatic Git merge/PR creation, deployments/sending, general memory, mobile/cloud expansion, and automated composed-Outcome planning are not part of this assignment. Preserve existing composition behavior; offer clear scope reduction when the proposer is unavailable. Advertise only provider/capability/OS combinations actually tested. Do not make five-provider identity into five-provider conformance.

## 4. Phase A — verify and finish existing L2/L3 foundations

First run relevant baseline tests. Inspect the follow-up fixes before trusting the report.

### A1. Reasoning setup and lifecycle

Entry points: `service/settings/service.go`, `secretstore/file.go`, settings controllers/UI, `daemon/waldo_reasoning.go`, `service/intelligence/`, `adapters/llm/`, IntelligenceRun storage/recovery.

Implement any remaining gaps:

- Provider-bound credentials; switching provider without its key reports `MISSING_CREDENTIAL`. Environment precedence cannot reuse another provider's credential/model inadvertently.
- Secret never enters API responses, Work records, logs, generated evidence or exception text. Avoid retaining the submitted key unnecessarily in UI state. Missing key recovery returns to the original draft.
- Separate configured/key-present from verified provider readiness. A field being nonempty is not a successful live readiness probe.
- Proposal endpoints expose typed, actionable missing-credential/provider errors. The recorded generic `500 INTERNAL_ERROR` for missing setup is a remaining UX problem; do not present it as a successful setup journey.
- Timeout/cancel/refusal/auth/rate-limit/invalid-output paths have bounded retries and truthful terminal state. Persist terminalization even when the caller context is cancelled, using a bounded safe cleanup context where needed.
- Restart reconciliation does not leave runs visibly active or auto-repeat a potentially billed call. Late responses cannot overwrite newer revisions or terminal runs.
- Retain effective provider/model, actual input/output provenance, nullable usage and duration without inventing measurements.

### A2. Safe and general grounding

Entry points: `service/intelligence/repository_context.go`, `intake_adapter.go`, `llm.go`, `service/outcome/plan_intelligence.go`, Plan service, context DTOs.

- Verify secret exclusions, nested dependency pruning, regular-file checks, symlink/root confinement, cancellation and byte/file/traversal limits. Git/check-ignore failure must fail closed; non-Git context must use an explicit supported path rather than misrepresent a failed Git lookup.
- Context explicitly selected by the owner may include a registered repository/folder, Project Brief and supported local files. Reuse existing attachment/input mechanisms. Define supported local text/document formats and truthful errors; don't invent a universal ingestion platform.
- Inspecting input cannot execute scripts, mutate the source or spawn providers. Use safe file bounds, handle special files without blocking, and never replay an immortal transcript.
- Include actual bounded context in Contract and Plan input digests. Distinguish inspected facts, missing context, citations and assumptions. Paths/content from repositories are untrusted data.
- Preserve full clarification/replan feedback. Non-temporal answers cannot become deadlines. Reload reuses the current proposal; explicit replan has expected revision + stable retry identity and creates a new immutable proposal once.
- Show unresolved assumptions/blockers before approval; do not discard them to make a Plan valid.
- Non-repository document Outcomes must have an explicit custody model. Support safe single-writer workspace/artifact staging using existing facilities; do not silently initialize Git or pretend a folder has worktree isolation. If input/provider capability is unsupported, refuse before launch.
- A request needing external research/network cannot be completed with fabricated citations under a no-network policy. Explain the missing capability or propose an owner-reviewed narrower scope.

Required negative tests: provider-switch credential canaries; ignored/unignored sensitive files; cancellation/Git errors; symlink/special-file handling; context changes alter digest; stale/duplicate replan; no pre-approval Attempt; unavailable provider never triggers another.

Exit Phase A with reusable foundation commits and an updated evidence ledger. Continue to Phase B; do not wait for a live credential to implement independent local behavior.

## 5. Phase B — complete L4: Board/List and Mission Control

Entry points: `OutcomesOverviewSurface`, `WorkShell`, `_shell.work`, `OutcomeMissionControl`, `OutcomeDecideAuthorizeSurface`, `OutcomeRunSurface`, `useOutcome`, `event-transport`, daemon scheduler and outcome controllers.

### B1. Board/List and navigation

- Show top-level Outcomes by default, filter by Project/attention/history, and keep Board/List equivalent. Session boards may be reused only for subordinate history.
- Cards show human-readable title, Project, current milestone, blocker/next decision and proof summary when known. Map lifecycle labels from daemon facts; do not persist duplicate display state.
- Card selection opens one Mission. Preserve selection/filter/scroll when returning from WorkUnit/Session inspection. Never start execution on navigation or drag.
- Empty state creates an Outcome; missing provider does not prevent Project registration. No-provider, offline, loading, stale and failed reads must be distinct.

### B2. Direct WorkUnit graph

Implement the actual direct-Outcome DAG, not only repeated cards or the contributing-Outcome graph. Reuse graph components/libraries where possible. Use stable WorkUnit IDs, dependency edges and existing maximum graph bounds. Layout is a projection; no scheduling logic in React.

- Proposed graph before authorization; daemon-derived execution overlay afterwards.
- Node: title, truthful state, blocker, provider/model semantics and evidence summary. Detail panel: objective/output, criterion text, dependencies by title, capability scope, routing reason, Attempt history and artifacts.
- Explain serial execution. Independent branches can be drawn, but cannot appear simultaneously executing without daemon facts.
- Distinguish waiting for proof, custody held, unavailable capability, paused, failure and unknown. Zero runnable candidates needs a reason, not a spinner.
- Fit/zoom, keyboard node navigation, visible focus, reduced motion, and equivalent accessible list. Use existing visual vocabulary and localization; do not show opaque IDs or inline English as normal primary content.
- One Mission concept supports existing direct/decomposed shapes without pretending decomposition edges are execution edges. Preserve existing user acceptance rules for contributors.

### B3. Actions and freshness

- Normal Start sends the approved Plan + replay key; daemon chooses next eligible WorkUnit. Keep deliberate named-unit API assertions validated.
- Approve and Start remain distinct effects. Surface exact authority and unresolved blockers before approval.
- CDC and local mutations invalidate schedule/Outcome/Attempt/proof as appropriate. Reconnect refetches facts; staleTime alone does not create polling or freshness.
- Double-click/retry/reconnect cannot duplicate Attempts. Changing Project defaults after approval cannot change execution.
- Place contextual clarification/replan within the Mission. Handle typed missing-credential responses with a settings action and preserved draft.

Exit tests: Plan JSON ordered B,A with B dependent on A; all nodes visible; A selected by daemon; B disabled until proof; independent C shown truthfully serial; CDC updates open graph after completion/recovery/proof; cancel/start failures reconcile; Board/List and back navigation preserve context. Use a real daemon with isolated fixture provider state for UI checks when live models are unavailable. Do not describe fixture-provider execution as live conformance.

## 6. Phase C — L5a: truthful terminal state and artifact continuity

Entry points: outcome attempt/recover/scheduler, runtime/session observations, proof ports/domain/store, workspace adapters, current receipt machinery.

Before editing schema, document existing receipt/artifact fields and only add missing durable facts. Retain Outcome/Contract/Plan/WorkUnit/Attempt identity, provider session, workspace, base/result revisions where meaningful, changed/dirty output artifacts, content digests, timestamps, command results and uncertainty.

- Add the missing production reconciliation path for Attempt terminal outcomes. Derive success under existing proof semantics; never assign success from transcript markers or a bare process exit. Define how execution-ended and proof-pending differ so the model doesn't need to keep a process alive for review.
- Duplicate/out-of-order observations are idempotent. Contradictory or unconfirmed runtime state remains blocked until reconciled. Do not release custody merely because the provider is quiet.
- Capture outputs without relying on a provider to describe them accurately. Preserve untracked/binary/deleted files or explicitly refuse unsupported cases. Snapshot only the owned workspace and exclude credentials/temporary machinery.
- A downstream WorkUnit receives the exact retained upstream artifact/base transfer with provenance. Starting another worktree from the original branch does not count as handoff.
- Freeze what was verified; later work cannot silently overwrite a reviewed artifact. Failed cleanup leaves inspectable retained state.

Exit: A writes a distinctive artifact, daemon restarts, B receives that exact content; missing/mismatched artifact blocks B; cancelled/uncertain A does not auto-spawn B; unrelated user files remain untouched.

## 7. Phase D — L5b: checks, evidence, continuation and review

- Reuse canonical Evidence/Verification services. Add a bounded agent-facing submission surface only if needed; use authenticated/scoped attempt identity and structural validation. Reject forged foreign lineage, oversized payloads, path escapes and duplicates.
- Agent-submitted evidence is a claim. Independently observed checks are labelled independently. Model labels cannot turn self-report into verification.
- Execute approved check specifications through governed runtime authority, not an unrestricted shell HTTP endpoint. Retain executable/args/cwd, start/end, exit/timeout/cancellation and bounded stdout/stderr artifacts. Treat generated commands as untrusted proposals requiring appropriate authority.
- Bind proof to exact WorkUnit/Attempt and criteria. Do not accept Outcome-level or stale proof merely to unblock scheduling. Present failing, contradictory and missing evidence.
- Implement daemon-owned serial reconciliation/continuation using existing admission/fence/replay boundaries. Define durable run-generation intent so Pause or Cancel survives restart; enable automatic continuation only within explicitly authorized run intent. No retry loops after an unproved failure.
- Owner can inspect result/checks, request changes, and accept the reviewed revision. Retain previous evidence on rework, carry bounded feedback into the next Attempt/Plan, and invalidate stale acceptance eligibility.
- Never allow agents to create AcceptanceDecision. Independent validation success may produce Ready for Review, not Accepted.

Exit: failing check blocks; real correction passes; duplicate submission/start continues once; owner rework retains evidence; provider completion and passing tests alone leave acceptance unset. A supported document task can use appropriate artifact/criterion checks without meaningless software test commands.

## 8. Phase E — L5c: deliver the result

A completed Outcome must expose a usable deliverable, not leave the owner searching hidden worktrees.

- Owner-triggered Export result. Support the declared local artifact formats for document/research work and a portable patch/bundle for software. Inspect available Git/export helpers before adding another mechanism.
- Bind export to a retained immutable artifact version and, for an accepted-result export, its AcceptanceDecision. Draft export must be clearly labelled and does not imply acceptance.
- Record manifest: producing lineage, relative paths/digests, context/repository identity, base/result revision where applicable, verification references, export format/destination and observed status.
- Software export must preserve supported additions/deletions/binary content/file modes. Document limitations and stop instead of silently dropping output. Do not include secrets or user files outside custody.
- Stage export atomically where possible. Protect destination paths and existing files; an overwrite requires explicit owner action. Failure/cancellation/partial output cannot be marked delivered.
- Test patch application or bundle extraction in a disposable destination and compare actual content/digests. Supply clear opening/application instructions and base prerequisites. Handle a conflicting destination explicitly.
- Export must not merge, reset or mutate the owner's source branch. Delivery is not evidence of application, merge, publication or deployment. Retain source artifacts until safe cleanup policy applies.

Exit: accepted document opens from export; accepted code patch applies at recorded base and reproduces checked changes; repeated export is truthful/idempotent for its request; missing artifact, conflict, unsafe path and partial failure remain visible.

## 9. Phase F — L6: focus and finish the owner experience

- Default launch goes to Work. Reversibly disable Home navigation, Island/notch automatic startup and standalone Waldo side chat for this launch mode. Do not delete modules, databases or historical records. A hidden surface must not leave background agents running.
- Keep Projects, reasoning/provider settings, Outcome contextual reasoning, evidence/result UI and Session inspector.
- Existing unavailable deep links show a concise explanation and return path. Don't add a second onboarding/navigation system.
- Finish keyboard focus, back/forward, empty/error/offline states, meaningful button labels and recovery after app reopen. Use existing design primitives and translations.
- Add `docs/user/kennel-work-guide.md`: what an Outcome is; setup; supplied context; Contract/Plan approval; run/pause/questions; evidence and uncertainty; rework/acceptance; export/apply; recovery and capability limits. Include one software and one supported document example. Explain reasoning credentials versus execution-provider authentication without exposing implementation jargon in ordinary screens.
- Add concise in-product guidance where it resolves a decision, not developer commentary in the UI.

Exit: owner can find next action at every stage, return after restart without reconstructing transcripts, and obtain a usable result.

## 10. Phase G — L7 implementation readiness and reviewer handoff

Run appropriate baseline/narrow checks first and full required gates at integration:

```sh
npm run bootstrap
npm run lint
npm run frontend:typecheck
npm run sqlc
npm run api
npx @redwoodjs/agent-ci run --all
```

Backend from `backend/`: `go build ./...`, `go test ./...`, `go test -race ./...`, `go vet ./...`.
Frontend from `frontend/`: `npm run typecheck`, `npm test`, `npm run build`.

Inspect actual command output. A no-op agent-ci wrapper is not coverage. After generation require no unexpected diff; commit intended generated contracts with sources. Record full-suite failures without blanket skips or declaring them inherited without base reproduction. Avoid fragile implementation-mirroring tests; prove the behavior and negative boundary.

Use documented real-daemon preview and an isolated temporary profile/repository. Exercise settings, Board/List, graph, approval, actions, proof and delivery using controlled fixtures where necessary. The owner will do the final local acceptance, but you must provide exact setup/start commands, package path and the test matrix below. Build the intended macOS package; do not sign/publish or claim Windows/Linux rehearsal from available makers.

Live provider calls require separate permission/credentials. Missing execution code-mode host blocks that execution path, not all daemon/UI/direct-HTTP-adapter work. Diagnose the boundary read-only; do not modify global Codex configuration, install opaque executables or bypass policy to make it run. Record installed binary/mode/model only when actually checked. Do not claim a provider refused effects when the command never ran.

Measure existing representative latency/context size/query volume before targeted optimization. Bound work, remove duplicate calls and fix observed hotspots; no speculative rewrite or cache that reuses stale authority.

## 11. Required final acceptance matrix

For each row record pass/fail/blocked, exact SHA, environment, proof level and evidence path:

1. Fresh profile with zero execution providers registers Project and preserves draft.
2. Reasoning provider switch never transfers another provider's key; missing setup has an actionable UI path.
3. Supported supplied-document and repository context are safe, bounded and honestly represented.
4. Clarification/replan handles non-temporal answers, duplicates and stale revisions.
5. Board/List opens the same Outcome Mission; graph has all WorkUnits and accessible list.
6. No execution occurs before authorization; mutable preferences cannot widen/reroute approved work.
7. Serial A→B and independent C reflect actual schedule and CDC updates; double-click creates one Attempt.
8. Actual denied write/network probes for each advertised execution path, or explicitly blocked live conformance.
9. Restart/recovery preserves frozen model/policy; missing/corrupt evidence refuses safely.
10. A's exact artifact reaches B after restart; failure/cancel prevents unsafe continuation.
11. Check failure/contradiction/wrong lineage blocks proof; provider claim cannot forge independent validation.
12. Owner rework preserves history; only owner Acceptance closes responsibility.
13. Exported document/patch reproduces the reviewed result; partial/conflicting/unsafe export is not delivered.
14. Work-only shell hides extras without losing data; next action remains discoverable after reopen.
15. Packaged local journey and owner guide work without inherited development configuration, or precise remaining blockers are recorded.

Do not compress this into a single green checkbox. Implementation ready for reviewer, live conformance, and owner acceptance are separate outcomes.

## 12. Engineering quality and final response contract

Keep commits phase-sized: foundations, Mission experience, terminal/artifact model, checks/continuation, delivery, focused shell/docs, integration corrections. Do not make one unreviewable mega-commit. Keep a durable ledger so context compaction does not reset the task. No automatic merge or next task creation.

Use existing ports/components; add abstractions only for repeated real behavior. Avoid hidden defaults, broad catch-and-continue, per-file subprocess loops with unbounded traversal, magic model/provider names, raw HTML status derivation, duplicated SQL schema authority and gigantic service/controller functions. Comments explain invariants and non-obvious reasons, not narrate each line. Don't delete useful comments or suppress lint to hit a zero count. Keep intentional compatibility paths explicitly bounded and tested.

Before ending, inspect git status and diff, remove only your temporary test files, and ensure intended work is committed. Deliver:

- Exact base/head/branch/worktree and ordered commits/dependencies.
- Requirement delta map and phase-by-phase implementation summary.
- Test matrix with proof levels, raw log locations and redacted durable evidence.
- Migrations/API changes and upgrade/recovery considerations.
- Known failures, skipped/unavailable checks and product/provider limitations.
- Exact commands for the reviewer/owner to build and run with an isolated profile; expected screens and sample inputs; safe cleanup steps limited to your disposable state.
- A short reviewer guide to the riskiest boundaries: secrets, grounding, revision/replay, runtime policy/recovery, evidence, export and UI authority.
- An honest recommendation: ready for code review, requires fixes, or blocked on specified evidence. Do not claim launch/owner acceptance or push anything.

Keep working through the authorized implementation phases until the remaining code is complete or a concrete external blocker prevents that phase. When a live check is unavailable, finish independent implementation and explicitly leave the check open. The deliverable is a reviewable working Work loop, not a collection of stubs or a rewritten plan.

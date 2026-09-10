# Luna implementation handoff: finish the Kennel Work launch loop

Prepared 2026-09-10 against inspected HEAD `55d916cec` on `codex/kennel-work-completion`. This is an implementation assignment when supplied by the owner. It is not implementation evidence or release acceptance.

## 1. Assignment and working agreement

Complete the remaining Kennel Work build through Phase G, in the slices below. Reuse and repair existing work. Do not restart the implementation from the old L1–L7 labels. The owner will ask the planning agent to review your commits, then test locally. Your own automated and isolated application checks are required before that review.

You may edit source/docs, run local tests/builds, create disposable test repositories/profiles, and make local conventional commits. Keep progressing across routine slice boundaries without asking “say continue.” Report progress and update the ledger instead. A blocked live-provider test does not block independent code work. Do not describe blocked tests as passed or weaken enforcement to unblock them.

Do not push, merge, publish, deploy, create external PRs, alter real credentials/global provider configuration, make paid/live model calls, or mutate the owner's live `~/.kennel` profile. Those actions need separate authorization. Product features for owner-authorized export belong in scope; invoking external effects during development does not. Preserve unrelated worktrees and user files. Do not install or fabricate an opaque missing host executable. No new agent/task spawning is required.

Use bounded commits with tests and evidence. At the C/D checkpoint write a reviewer handoff, then continue independent B/A2/E/F/G work under this assignment unless the owner explicitly asks you to pause. Incorporate review corrections when received. Never claim a checkpoint was reviewed just because a report was written.

## 2. Starting point: verify before editing

- Worktree: `/Users/shivanshfulper/.codex/worktrees/kennel-work-completion`.
- Branch: `codex/kennel-work-completion`.
- Last inspected HEAD: `55d916cec`; clean before this handoff was added.
- Existing branch base: `c83684c11`, the Wednesday implementation; earlier integrated beta was `9396c3844`.
- The new handoff itself may be uncommitted. Preserve it and commit it separately from product changes.
- Preserve `/Users/shivanshfulper/.codex/worktrees/wednesday-milestone` and the primary checkout.

Read status/log, resolve full SHAs, fetch `origin/beta`, and compare ancestry. Remote state is not established by the historical values above. Continue this branch if compatible. If beta advanced, inspect and integrate dependencies in an isolated checkout without dropping the completion commits or touching someone else's dirty tree. Never reset/clean/stash unrelated work. Do not cherry-pick already-present commits or start another duplicate completion branch unnecessarily.

Read `AGENTS.md` and its authority chain, especially product architecture and ADRs 0008/0009. Then read:

- `docs/product/2026-09-09-kennel-work-launch-experience.md`
- `docs/superpowers/plans/2026-09-09-claude-code-kennel-work-completion.md`
- `docs/superpowers/plans/2026-09-09-a2-supplied-document-context.md`
- `docs/verification/2026-09-09-execution-to-admission-sequence.md`
- `docs/verification/kennel-work-completion-ledger.md`
- `docs/STATUS.md` and applicable current ADRs/specs referenced there.

This handoff updates the earlier assignment's starting point, ordering and known corrections. It does not replace canonical authority or silently remove requirements from that assignment. Resolve any inconsistent execution/custody text against source and invariants before coding.

## 3. Product and actual delta

The product is general, supported local Outcomes: software changes, analysis and results based on supplied local documents. It is not unrestricted autonomous external research. Unsupported formats/effects need an explicit refusal, never invented content. Launch promises only combinations verified in the final matrix.

Owner flow:

Project/context → Outcome → clarification/Contract → proposed Plan and WorkUnit graph → approve → explicitly start → observe/pause/questions → retained result and checks → request changes or accept → export usable result.

Navigation: Work Board/List → one Outcome's Mission Control → WorkUnit detail → optional Session inspector. Board/List represent Outcomes. The direct graph represents WorkUnits; the composition graph represents contributing Outcomes. They must stay distinct. Ordinary supervision must not require reading provider transcripts.

Present at the inspected baseline:

- L0/L1 launch binding, capability policy and recovery code; live enforcement remains unproved.
- L2 reasoning foundations and repository-focused L3 grounding.
- L4 Plan/schedule foundations and B2 direct WorkUnit graph.
- A1 typed reasoning errors and verification stamps, with unresolved review issues below.
- C receipt schema/store and attempt-specific proof predicate; success is now reachable in Go and SQL.

Still required: correction slice R, actual C retention/handoff, D enforcement/checks/continuation/rework, B1 Board/List, B3 Mission header/actions, A2 supplied-document implementation, E delivery, F focused shell/guide, G integration/package evidence. A2 safety inspection is not A2 document implementation. C receipt fields are not retained bytes. B2 tests are not full L4 acceptance.

Preserve serial custody for launch. Parallel scheduling, automated composed-Outcome proposals, implicit Git integration/PR creation, deployment/sending, general memory, mobile/cloud and landing changes remain outside this build. Preserve readable historical data and existing composition behavior; explain unavailable proposal capability. Hide Home, Island/notch startup and standalone side chat in launch mode, retaining Outcome-local questions.

## 4. Cross-cutting implementation rules

1. Daemon/SQLite owns authorization, revision validation, replay, scheduling, custody, evidence and acceptance rules. Models propose; frontend projects facts.
2. Approval and Start are distinct. Execution binding/capability ceilings cannot widen through mutable settings, retries or recovery. No hidden fallback.
3. Unknown runtime/effect status blocks duplicate work. A failed probe is not death. A provider message, exit code or green check never creates AcceptanceDecision.
4. Every async transition needs stale-state protection and restart behavior. A multi-write procedure is not a pure function merely because its inputs are durable.
5. One proof/artifact identity must refer to one immutable result. Evidence for one Attempt cannot classify another. Verification must remain tied to the bytes it checked.
6. Source queries/schema first, then sqlc/API generation. Add migrations; do not rewrite 0118/0119 merely because they are unmerged: they may have been applied locally. Allocate the next migration number after inspecting the current tree.
7. Trigger CDC remains canonical. New durable facts needed by open UI require corresponding event/invalidation handling; no manual parallel status authority.
8. Controllers stay thin. Prefer existing ports/adapters/components over frameworks, universal registries, broad refactors or hidden defaults. Comments explain invariants; remove inaccurate claims rather than adding prose to excuse them.
9. Use fixture providers only in test/development wiring. They must never appear as production fallback or claim live-provider conformance.
10. Keep raw logs outside source, and a concise redacted verification record inside source. Separate source inspection, automated tests, daemon fixtures, live provider, packaged desktop, owner acceptance.

## 5. Slice R — close review findings before extending C

Reproduce each issue first. Source pointers refer to the inspected baseline; follow renames as needed.

### R1. Transactional classification and receipt binding

`backend/internal/service/outcome/terminal.go` currently changes status before freezing the receipt and writing the observation. It only scans `reconciled` Attempts. A crash after the first write is never repaired. `FreezeAttemptReceipt` also succeeds when no receipt exists.

Implement one coherent persistence operation, or an explicit recoverable durable protocol, that validates expected status and exact retained version, records classification and observation, and freezes the artifact consistently. Prefer a transaction for database-only changes. Bind the proof to the reviewed artifact version, not whichever mutable receipt happens to exist at freeze time. Recheck relevant revision/proof preconditions at the commit boundary. Preserve bounded retry and replay semantics.

Missing/incomplete/failed retention cannot silently pass the defined sequence. If a supported no-output task needs an empty artifact, represent and verify an explicit empty retained manifest; absence is not an empty result. Freeze at the boundary required to prevent evidence referring to changing bytes. If refinement is needed after freezing, create new attributed result/Attempt history rather than rewriting what was checked.

Tests: real SQLite service-level classification; inject failures between writes; restart and rerun; concurrent classification executes once; missing/wrong-version/incomplete receipt refuses; stale proof cannot freeze new content; observation is not lost or duplicated. Existing pure `attemptProven` tests alone do not close this row.

### R2. Successful Attempt custody

Classification does not release its fence, while existing `recover.go` rejects recovery of succeeded Attempts. Implement explicit, idempotent handling for succeeded Attempts with outstanding custody. Release only after required runtime/retention facts are accounted for. Keep durable downstream run intent separate from releasing a lock. A release must not automatically authorize new work.

Tests: succeeded Attempt holding a fence is repaired after restart; retained failure stays blocked; surviving/ambiguous process is not freed; cross-Outcome attempts in the same Project still respect the fence; no duplicate successor. Reconcile this with C-13 admission before calling C complete.

### R3. Immutable receipts and coherent reads

Migration 0119's guards allow updating/inserting frozen manifest files, clearing `frozen_at`, and then replacing the receipt. Protect all immutable provenance/content fields and file insert/update/delete operations, including attempts to unfreeze or delete the parent. Preserve legitimate idempotent repeat operations without opening mutation paths.

Validate receipt lineage against its actual Attempt/Plan/WorkUnit/Contract, and file lineage against the receipt. Validate declared artifact version against the canonical manifest. `retained` cannot carry unknown required content as though complete. Read receipt and manifest from one coherent database snapshot: separate untransactional reads can mix versions during retention replacement.

Tests: direct SQL rejection of forbidden mutations, normal store refusal, concurrent read/replacement consistency, wrong lineage/version rejection, valid empty manifest, immutable repeated freeze. Update ledger C-6 only after these tests pass.

### R4. Artifact identity and path semantics

`domain.ArtifactManifestDigest` omits file mode. Canonical identity must include supported semantic metadata, especially executable mode, and use unambiguous encoding. Define regular files/deletions/binaries and supported or rejected links explicitly. Validate paths by path components and destination platform rules, not a blanket substring check for `..`. Do not permit absolute/drive/UNC paths or traversal; ordinary names containing two dots are not necessarily traversal.

Tests: executable-bit change changes identity; order does not; changed bytes/deletions do; traversal/absolute/platform escapes fail; supported normal filenames work. Resolve or refuse links at filesystem boundaries, not just string validation.

### R5. Unresolved A1 readiness

Recheck `service/settings/service.go`, `ReasoningSettingsSection.tsx`, `useSettings.ts`. At the inspected baseline an old probe can mark a replacement key for the same provider/model verified. Old failures can clear newer verification. UI still uses `ready` and exposes no verification action.

Bind probe results to the exact configuration/credential generation and effective endpoint/model inputs. Store no raw secrets in fingerprints, logs or API responses. Apply success/failure only if that generation remains current. Invalidating inputs must invalidate verified status, including supported environment overrides. Decide verification freshness explicitly; configured/key-present/verified must be understandable and distinct in the UI. Add a deliberate Verify action and preserve the original draft when setup interrupts work. Do not add automatic billed probing.

Tests: probe A in flight → replace key B → A success/failure cannot stamp/clear B; provider/model change; relevant environment change; UI shows configured but unverified and handles successful/failed verification truthfully. This is unfinished review work, not an optional polish task.

R exit: focused regressions green, receipt/custody sequence doc corrected, inaccurate “pure function”/automatic retry claims removed, ledger reopened/closed with evidence, local commit(s).

## 6. C-12 — retain real output, not only a manifest

Start at `ports/outbound.go` WorkspaceObserver, `adapters/workspace/gitworktree/workspace.go` ObserveWorkspace, scratch/router adapters, existing receipt store and session workspace ownership. ObserveWorkspace reports facts; verify its bounds and do not assume it preserves content.

- Add the smallest port/adapter needed to capture an owned, quiescent workspace into retained immutable content plus manifest. Digests without durable retrievable bytes do not survive cleanup or supply a successor.
- Derive custody from daemon-owned Attempt/workspace identity, never an arbitrary client path. Account for tracked committed changes since the frozen base as well as dirty/untracked/deleted output. HEAD-only diff loses work an agent committed.
- Record honest base/result revisions and source identity; do not fabricate commits or treat a dirty HEAD as the complete result. Staged folders have no Git revisions.
- Use streaming hashes, explicit traversal/file/total-byte/time bounds, cancellation and safe regular-file handling. Keep secret exclusions and input/output roles explicit. Unexpected changes you cannot represent must produce an incomplete/unsupported result with a reason, not silently disappear from a supposedly complete snapshot.
- Do not follow symlinks outside custody, block on FIFOs/devices, or trust a pre-read path check against concurrent replacement. Detect/refuse inconsistent reads. Require runtime quiescence before capture and verify bytes actually stored, not just bytes observed earlier.
- Store under the documented isolated application-state root. Stage files and atomically publish only verified content; filesystem and database crashes need retryable records and conservative debris handling. No force-cleaning unknown files.
- Invoke retention from a restart-safe reconciliation path for ended Attempts. A boot scan must find missed/incomplete work; hooking only the live transition misses crashes.

Tests use temporary real Git repositories and staged folders: committed plus uncommitted changes, executable/binary/untracked/deleted outputs, secret canary, symlink/special-file refusal, file mutation during read, bound exhaustion, cancellation, partial publication/restart, repeat snapshot and frozen-result refusal. Confirm retained bytes remain available when the original workspace is unavailable. Avoid per-file unbounded subprocesses.

## 7. C-13 — exact downstream artifact handoff

- Before launch, resolve dependency results from canonical lineage and verify each retained manifest and blob. Missing/corrupt/incomplete artifacts block admission with a human-readable reason.
- Provision the successor using the approved base and dependency outputs. Verify the destination content/mode/deletions before spawning. Do not start from the original Project branch and describe a prompt referencing A as handoff.
- Define multiple-predecessor composition: preserve lineage, deterministically combine compatible results, recognize shared ancestry, and stop on conflicting writes/deletions or base mismatch. Do not silently use last-writer-wins or require parallel scheduling to test a branching DAG.
- Record input artifact versions on the successor admission snapshot/receipt so restart uses the same inputs. Keep replay fingerprints in sync with new semantics. Never let newer upstream output silently replace already-authorized input.
- Coordinate reservation, filesystem provisioning, admission and launch with existing fences. No launched duplicate after a crash; partial workspace remains attributable and recoverable. Changes to frozen authority require explicit replan/authorization.

Required integration test: A produces a distinctive file and executable mode → retained proof → terminate/restart daemon → B receives identical bytes/mode → B uses that result. Also deletion/binary transfer, original workspace unavailable, corrupt blob, conflicting join, duplicate Start, cancelled intent and surviving prior runtime. Assert actual files in B, not only manifest equality or prompt text.

Write a C checkpoint report with exact SHAs and failures. C is complete only when R/C-12/C-13 hold together.

## 8. D — enforced checks, evidence and durable continuation

### D1. Identify the enforcing runtime first

Map production call paths for governed provider execution and deterministic checks. `internal/process/command.go` wrapping `exec.Command` is not sandbox enforcement. The missing `~/.local/bin/codex-code-mode-host` may block a specific path; inspect before assuming it blocks all adapters. Do not bypass it or relabel a fake runner as conformance.

Checks run through the actual supported enforcement boundary with authority no wider than the approved Attempt. Check commands from model/repository are untrusted proposals. There must be no arbitrary unrestricted-shell API. Pin cwd, executable/args, environment policy, timeout, output bounds, cancellation and network/write policy. Control the spawned process tree and retain termination uncertainty honestly.

Behavioral canaries in isolated disposable state: authorized in-workspace write succeeds; outside write fails; prohibited network access fails; cancellation stops governed work or leaves unknown custody. Distinguish denied operation from “command never launched.” Tests that inspect flags are useful unit coverage, not effect proof. Live provider calls remain separately authorized; blocked conformance leaves launch acceptance open.

### D2. Evidence and verification

Reuse canonical Evidence/Verification services. Implement an agent-facing submission route only where missing, scoped/authenticated to an Attempt. Reject foreign lineage, stale revisions, oversized inputs, path escape and conflicting replay. Agent submissions remain claims; only independently observed execution gets independent-check provenance.

Retain exact check definition, input artifact version, criterion binding, runner identity, timing, exit/timeout/cancel and bounded redacted logs. Bind pass to the content actually checked. If checks mutate result files, explicitly account for those changes and invalidate mismatched proof; do not attach a pass from pre-change bytes to post-change output.

Test failing/missing/contradicting checks, wrong producer, spoofed independence, stale artifact version, duplicate request, timeout and cancellation. Neither a model's prose nor a client-supplied `passed` may bypass this boundary.

### D3. Run intent, scheduling and rework

Persist an explicit authorized run generation with desired running/paused/cancelled semantics. Add only necessary schema, generated contracts and CDC. Define what Start authorizes across the approved Plan, what Pause stops, how Resume creates/continues intent, and how Cancel prevents downstream work. Do not infer intent from an open screen or a finished Attempt.

Continue serially through the existing admission path. Use stable per-generation/per-unit replay identities and conditional writes. Restart, repeated clicks and concurrent reconciler ticks must converge on one Attempt. Pause/cancel racing with provisioning must be rechecked before launch. Unknown prior custody blocks. No automatic retries after an unproved failure or unauthorized provider switching.

Rework preserves evidence/history and bounded owner feedback. Invalidate eligibility for obsolete proof/acceptance when revisions change. Only the owner creates AcceptanceDecision; passing checks produces review readiness. Retain exact reviewed revision/artifact references.

Tests: restart running/paused/cancelled; stop at each admission stage; duplicate request/tick; failed A blocks B; user rework → corrected check → Ready for Review → explicit owner acceptance; no acceptance without owner action. Include document-appropriate checks; do not require a fake Go/npm command for a report.

D checkpoint: produce a focused review guide, update the 15-row matrix, distinguish implementation done from blocked behavioral conformance. Continue independent remaining slices while external evidence is blocked.

## 9. B1/B3 — complete the owner supervision UI

Use `OutcomesOverviewSurface`, `OutcomeMissionControl`, `MissionWorkUnitGraph`, existing Work shells/hooks and daemon schedule projections. Preserve B2 unless a regression requires a fix.

B1: Board and List are views over the same top-level Outcomes. Derive columns/states from canonical facts; no separate frontend workflow state machine, session cards as Outcomes, or drag-to-execute. Support Project/attention/history filtering, loading/offline/error/empty states, selection/back navigation and useful cards. Preserve drafts with no ready provider. Localize using the existing locale set and primitives.

B3: Mission header shows title, Project, current Contract/Plan context, blocker and primary next action. Contract, Plan/Graph, result/evidence and decisions remain discoverable. Choose the action from daemon facts: clarify, review proposal, approve, Start, pause/resume, inspect failed proof, request changes, accept or export. Approval and Start stay separate. Show current/stale/reconnecting data truthfully and prevent stale mutations with daemon validation.

Invalidate on relevant CDC, mutations and reconnect for Outcome/schedule/Attempt/proof/receipt/delivery. Preserve filters, scroll and selected unit when entering/exiting Session inspector. Graph remains serial and has keyboard navigation/list equivalent, focus and reduced-motion behavior. No raw IDs or hardcoded English as ordinary primary content.

Tests: same Outcome from Board/List; no mutation on navigation; multi-unit B-before-A serialization; serial independent branches; open Mission updates after proof/retention/recovery; typed setup error preserves draft; repeated Start once; back/reopen; keyboard and error states. Run the real-daemon preview and visually inspect these flows with isolated fixture state. Renderer mocks alone do not establish L4 completion.

## 10. A2 — supplied-document Outcomes remain in scope

Implement the existing A2 spec after checking its assumptions against C's actual staged custody. Support a deliberately bounded set of local text/document formats with explicit selection, readable failure reasons, file/content limits and provenance. Do not add PDF/OCR/network ingestion silently if not supported.

Owner selects context → daemon safely reads bounded input → context digest binds proposal → owner reviews scope → immutable approved context is staged for a single writer → provider receives only authorized input → retained result is checked and exported. Source documents are not writable workspace output. Secret exclusion and symlink/special-file safety apply. No silent Git initialization or false worktree claim.

Context changes invalidate stale proposal approval or require an explicit refresh/reproposal; reopening a screen must not silently reread changed files into approved execution. Define a narrow capability mapping for the supported staged path at the actual adapter boundary. Do not weaken the existing exact `worktree/*` policy validation to accept arbitrary folders. If a new custody scope is necessary, extend it end to end with fail-closed tests and explicit mapping, or leave that provider/path unsupported.

Tests: supplied input → grounded Contract/Plan with citations/assumptions → staged run → retained report → criterion checks → owner review. Include source unchanged, changed input, secret canary, escaping file, unsupported format/provider, restart custody and document export later in E. Until this works, general-document acceptance remains open; do not quietly redefine launch as software-only.

## 11. E — usable delivery, with validation

Receipt schema is a prerequisite, not a finished delivery mechanism. Implement owner-triggered Export result using existing helpers where sound. Support declared document artifacts and a portable software patch/bundle. Bind accepted export to exact artifact and AcceptanceDecision; any allowed draft export must be visibly labelled draft.

Preserve additions/deletions/binaries/modes and recorded base requirements. Include producing lineage, content digests, verification references, format and observed export state in a manifest. Stage output safely, validate destination confinement, avoid overwriting existing files without explicit owner action, and record incomplete/cancelled/failed export honestly. Never merge/reset/mutate the source branch. An export is not an applied change or deployment.

Test actual application/extraction into a disposable destination at the recorded base; compare files, hashes, modes and deletions against retained result. Test missing/corrupt artifact, incompatible base, existing destination, traversal, interruption and repeated request. Show a discoverable Open/Reveal action and clear use instructions. Keep source artifacts until safe retention policy permits cleanup; export alone is not permission to delete them.

## 12. F — focused shell and owner guide

Add a reversible launch configuration using existing config mechanisms. Work is the default. Hide Home navigation, disable Island/notch automatic startup and standalone Waldo side chat, including unintended background startup. Preserve code/data and supported historical/deep-link handling. Keep Projects/settings, Outcome-local questions, results and optional Session inspection. Do not preserve invisible background activity just because routes stay mounted.

Write `docs/user/kennel-work-guide.md` with software and supported document walkthroughs: setup, reasoning credentials versus provider authentication, input context, Contract/Plan authorization, serial graph, pause/questions, failed checks/rework, acceptance, export/application and restart recovery. Use product language and precise limitations. Verify guidance against actual screens and commands.

## 13. G — integration, packaging and handoff

At slice start run a relevant baseline. Run narrow behavioral regressions first, then touched-area gates. At final integration run repository-required commands, inspecting actual scripts and output:

```sh
npm run bootstrap
npm run lint
npm run frontend:typecheck
npm run sqlc
npm run api
npx @redwoodjs/agent-ci run --all
```

From backend: `go build ./...`, `go test ./...`, `go test -race ./...`, `go vet ./...`.
From frontend: `npm run typecheck`, the actual full test command, `npm run build`.

Generation must match committed sources with no drift. Check migration upgrades from the recorded beta schema and profiles already containing 0116–0119, not only an empty database. Verify restart after upgrades. Do not rewrite migration history or mask failures with broad skips. Do not call a failure inherited without comparison against the baseline.

Build the intended local macOS package using repository instructions. Do not sign/publish or claim Windows/Linux success from their makers. Exercise a fresh isolated profile and a recovered profile through the real daemon and desktop. Record exact package path, SHA and environment. Identify any development executable/config the package still requires. Missing signing/live credentials are separate blockers, not reasons to skip runnable local checks.

Use the earlier handoff's 15-row acceptance matrix, extending it with R1–R5 and multi-predecessor/mode tests. Each row needs pass/fail/blocked, SHA, proof level, environment, observed result and evidence path. Measure representative proposal/context size and schedule/UI latency/query volume before targeted optimization; report measured values and limits, not invented performance claims.

Final deliverable:

- Exact base/head/branch/worktree and ordered commits; clean status or explicit remaining changes.
- Updated STATUS and completion ledger with real delta, migrations/API changes, proof levels and blockers.
- Review guide centered on secrets, artifact identity, atomic transitions, custody, enforcement, replay, export and UI truth.
- Exact verified commands to run/build with isolated state, package path, sample repository and document inputs, and the expected owner journey.
- Matrix and concise redacted evidence; raw log locations; remaining live/provider/package/owner checks.
- Honest readiness recommendation. Owner testing and acceptance remain separate. No push or release claim.

## 14. Progress and stopping rules

Suggested commit groups: R atomicity/custody; R receipt identity; R readiness; C retention; C handoff; D enforcement/evidence; D continuation/rework; B Board/List; B Mission; A2 documents; E delivery; F shell/guide; G integration fixes/evidence. Split further if a group becomes difficult to review.

Keep the ledger usable after context compaction: exact HEAD, last green gate, next unfinished requirement and blockers. Do not reimplement completed slices after compaction. Do not use estimated “sessions” as acceptance criteria or rush into a broad rewrite to meet them.

If external evidence is unavailable, finish independent authorized implementation and list the exact open row. If a design would require weakening an invariant, stop only that dependent path, explain the concrete conflict and continue independent work. Never replace a real behavior with a stub/fixture and mark it complete. The goal is the usable end-to-end Work loop, with unsupported paths explicitly excluded and the general supported document path retained in scope.

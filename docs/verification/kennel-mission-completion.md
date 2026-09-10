# Mission frontend lane — 2026-09-10

## Boundary

Base: fetched `origin/beta` and HEAD both `670a42238795b3da0c9fd61a5434132e9b622dd0`.
Branch: `codex/mission-control-frontend`.
Worktree: `/Users/shivanshfulper/.codex/worktrees/9494/Waldo-Kennel`.
Renderer, renderer tests, this note and user guide only. Backend, generated contracts,
Island, shared STATUS and live user profile are not changed. No push, merge or deployment.

## Source delta and implemented behavior

| Before | After | Why |
|---|---|---|
| Selecting an Outcome replaced the portfolio with a stage page | Persistent portfolio with a selected direct Mission; Contract, Plan, Execution, Evidence/result and history views | Review one responsibility with its identity and revision always visible |
| Graph selection had no detail; Plan output/checks came from WorkUnit 0 | Graph/Table share selected WorkUnit, output, dependencies, binding, checks, capabilities, stops and schedule Attempt summaries | Inspect the unit actually selected |
| Contributor rows mixed into the default overview | Top-level view with explicit contributor inclusion | Preserve responsibility hierarchy |
| Stale Plan offered approval until server refusal | Known Contract mismatch immediately withholds approval | Do not invite stale authority |
| Reconnect/CDC invalidated schedule but left Mission caches stale | Portfolio, Contract, Plan, Attempts, proof and schedule refetch together | Stream connectivity alone does not establish freshness |
| Focused sidebar still offered Home | Focused sidebar omits Home switch | Keep normal entry centered on Work |
| No contextual replan control | Existing generated replan endpoint, preserved feedback and reconciliation after error | Deliberate proposal without blind retries |

Mission width is keyboard adjustable; full-width expansion preserves mounted portfolio
filters and views. Below 1050px of Work content, portfolio collapses while sidebar
Outcome navigation stays available. Selecting a different Outcome keys its Mission
identity so previous forms and details cannot leak. Graph/Table stay mounted across
view toggles, preserving zoom; refreshed API array ordering is stabilized by ID within
dependency layers. This is display geometry, never scheduling authority.

Existing acceptance/evidence UI is reused, owner-explicit and separate from unavailable
export. Session terminal remains an optional WorkShell drill-down. Existing session
usage is shown only for distinct canonically bound session IDs, with incomplete and
unknown values retained; it is not an Outcome cost or planning-usage total.

New catalog keys exist in all locales with explicit English fallback text; translation
quality in non-English locales has not been reviewed.

## API inventory and backend binding gaps

Usable: Project/Outcome list and reads, immutable Contract history, current Plan,
Plan proposal/approval, schedule, Attempt list/admission/cancel/recovery, proof,
owner acceptance/rework decisions, reasoning settings, and session usage.

1. **Unconfigured reasoning error mapping:** isolated profile
   `/tmp/kennel-mission-profile`, `KENNEL_PORT=43731`, matching isolated run file;
   registered disposable Git repo `/tmp/kennel-mission-repo`.
   `POST /api/v1/outcomes/out-12a000d6-e5d7-4912-82db-cd94adb1886b/plans`, body
   `{"expectedContractRevision":1}`, returned HTTP 500 `INTERNAL_ERROR`.
   Renderer displayed `Internal server error (INTERNAL_ERROR)`. At
   `2026-09-10T15:01:13.215+05:30`, daemon request log attributed the failure to
   `reasoning provider is not configured; choose anthropic or openai`.
   Evidence: `/tmp/kennel-mission-daemon.log`. No credential or live provider call.
   Mission now uses existing settings readiness to explain setup and disable initial
   proposal before this request. Backend still needs a typed actionable response for
   races and other API clients. No error text is parsed into domain state.
2. **Replan replay:** generated `ReplanPlanRequest` has only
   `expectedContractRevision` and `feedback`, no replay identity. Automatic retry is
   disabled; any error retains feedback and withholds resubmission, offering refetch
   and Plan review. Backend needs request key plus durable reconciliation identity
   before safe same-request replay can be offered.
3. **Portfolio status:** bounded proof/schedule fan-out now supplies lifecycle columns,
   using the same cache as Mission (24 Outcomes per Project, Show more adds 24).
   A bulk attention projection remains a scaling improvement. Failure/rework carries
   the daemon reason; an approved Plan alone displays Authorized, never In progress
   or Accepted. Accepted history and contributors are explicitly available in Filters.
4. **Delivery/results:** generated API has no retained-artifact content/list/export
   or durable delivery operation. Needed: exact artifact version, producing lineage,
   retention/integrity state, current proof eligibility, accepted result identity,
   export destination/conflict/result and replay key. No pretend download/export.
5. **Documents:** no selected-document Outcome context binding in current generated
   contract. Need bounded selected-context identity/digest and admission semantics.
6. **Run intent:** existing Attempt cancel/recovery is not durable Outcome
   pause/continue intent. No speculative Pause/Resume controls added.
7. **Usage:** session totals exist; attributed planning metrics/cost and exact
   Outcome/WorkUnit usage accounting need backend contracts. No synthetic percentage.
8. **Real graph journey:** no model-backed Plan can be produced in the unconfigured
   isolated profile. A→B plus independent C is tested with fixtures, not presented
   as real-provider or daemon-backed execution proof.

## Verification

| Gate | Evidence |
|---|---|
| Baseline | 38/38 targeted tests; frontend typecheck passed |
| Full frontend | 217 files, 2697 passed, 6 skipped; `/tmp/kennel-mission-frontend-full.log` |
| Additional targeted pass | 19 files, 208 passed; `/tmp/kennel-mission-final-targeted.log` |
| Typecheck | `/tmp/kennel-mission-types.log` |
| Lint | Backend full tests plus golangci: zero issues; `/tmp/kennel-mission-lint.log` |
| Build | macOS arm64 package build passed; `/tmp/kennel-mission-build.log`; package not launched |
| Generators | sqlc/API run as parity checks, no generated diff; `/tmp/kennel-mission-sqlc.log`, `/tmp/kennel-mission-api.log` |
| Local CI wrapper | No relevant workflows for branch; not a CI pass; `/tmp/kennel-mission-agent-ci.log` |
| Bootstrap | Frontend installed, later `frontend/src/landing` npm ci failed for missing lockfile; no dependency changes |

Rendered with CUA against real daemon on 43731 and live renderer on 43732:
- PASS: two real Outcomes created through daemon API in disposable registered repo;
  Board → selected Mission Contract, long title, exact Project/Contract header.
- PASS: 1600×1000 split view, keyboard width slider, expand/restore, 1280×720
  portfolio collapse. Screenshots/DOM evidence are in this task's tool history.
- PASS: known missing reasoning setup shown in Mission, Draft Plan disabled,
  Home absent in focused mode, terminal disabled when no Attempt exists.
- PASS: external `POST /outcomes/{id}/revisions` created Contract 2; open Mission
  updated to `Updated through daemon CDC: preserve every source attribution.`
  without page reload, preserving selection and full-width view.
- BLOCKED: real Plan Graph/Table/Attempt/proof/acceptance/export loop at missing
  reasoning configuration and missing delivery APIs. No paid provider calls.

Automated tests cover graph/table selection, reordered refresh, removed selection,
unknown proof/dependency reason, stale approval withholding, lost replan response,
CDC/reconnect invalidation, focused navigation and existing owner-only acceptance.
Full launch acceptance, packaged interaction and provider enforcement are open.

## Review correction: safety controls

Parent review of `d12bb9cb5` correctly identified that disabling the entire Execution
fieldset on stale authority or disconnected SSE also disabled cancellation and
containment. The correction passes an admission-only block to OutcomeRunSurface:
Start/replacement admission is withheld, while cancel/contain/reconcile continue to
call their existing HTTP endpoints under daemon validation. Lost SSE is a freshness
warning, not evidence that HTTP control is unreachable. Mutation pending still prevents
duplicate safety requests; an actual HTTP refusal remains visible.

`OutcomeMissionSafety.test.tsx` mounts the real Execution surface inside Mission and
covers stale/current Contract versus connected/disconnected SSE, empty admission,
active cancellation and unconfirmed containment. Six cases pass. RED against the two
original d12bb9cb5 components and GREEN after restoring the fix are recorded in
`/tmp/kennel-mission-safety-red.log` and `/tmp/kennel-mission-safety-green.log`.
The bootstrap failure is a root-script cleanup gap: bootstrap still attempts the
removed frontend/src/landing package. This lane does not change the root script.


## Owner correction: Project Board continuity (2026-09-10)

The actual missing-Outcome defect was at the route boundary: `/projects/$projectId`
mounted `SessionsBoard`; `/work?view=outcomes` mounted the Outcome portfolio. In focused
mode Project links now redirect to Work with the exact Project ID and portfolio scope.
This also covers existing bookmarks, command palette and project-selection links.
Opening/closing a Mission preserves that scope; global Outcomes explicitly clears it.
Direct Outcome cards and sidebar Mission shortcuts now share the same destination.
Contributing-Outcome decomposition remains a separate existing responsibility projection.

Figma context inspected: file Dl0WP9uIvx6QbSzZi7cZQY, frame 3253:35386 and actual card
3253:35428. The card context specifies 18px inset/radius, quiet dark surface, secondary
context, next-action text and a stronger title. Adapted to existing tokens and typography;
no fabricated provider icon, artifact link or progress ring when those facts are absent.
The existing WorkShell List/Board control is the only view switch. Lifecycle columns
remain visible when empty; Filters is secondary. Resize uses a focusable pointer divider
with arrows/Home/End/reset; close includes X and returns focus to the opened card.
No new dependencies, backend or generated-contract changes.

The repository benchmark authority at `docs/product/kennel-v1-product-architecture.md`
section 12 retains AO/Vibe Kanban supervision lessons. No historical numeric usability
threshold was located in the inspected repository sources; none is claimed here.

### Fresh verification

- Full frontend: 221 files, 2712 passing, 6 skipped. After the final shortcut destination
  correction: 88/88 targeted tests, including actual router redirect, Sidebar, portfolio,
  mounted filter/focus retention, keyboard divider bounds, and all six admission-safety cases.
- Typecheck, package and root lint logs: `/tmp/kennel-mission-correction-types.log`,
  `/tmp/kennel-mission-correction-build.log`, `/tmp/kennel-mission-correction-lint.log`.
- Real isolated daemon: `127.0.0.1:43731`; live renderer `43732`; profile and repository
  remain `/tmp/kennel-mission-profile` and `/tmp/kennel-mission-repo`.
- PASS: direct Project URL resolves to its Outcome Board. Daemon returned exactly
  `out-12a000d6-e5d7-4912-82db-cd94adb1886b` and
  `out-ae5482d9-dc25-420a-8b0b-ccd507cc7d9b`; rendered card IDs match, and sidebar lists
  those same two titles. Navigation does not POST or create replacement work records.
- PASS: Board/List, card open/close, sidebar re-entry to the same Outcome ID, retained
  search `source`, close restores card focus, Project scope on refresh, Project switch
  to empty Scratch and back, browser back restores Scratch scope.
- PASS: 1600x1000 split, keyboard divider, expand/restore, 1280x720 Mission collapse,
  persistent close, long title visible in Mission. Plan shows unconfigured reasoning
  and disabled draft action, without inventing a Plan.
- BLOCKED: real Graph/Table/Attempt/result journey requires a configured reasoning
  provider and real Plan. No paid credential calls made. Existing component tests cover
  Graph/Table selection; that is not live runtime acceptance.
- UNVERIFIED: actual pointer drag, OS reduced-motion toggle, and packaged Electron
  interaction acceptance. Changed hover/divider transitions disable under reduced motion;
  no new ornamental or fake progress animation was added.
- Test-environment observation: several same-origin preview tabs delayed HTTP requests
  despite connected event streams; closing redundant tester tabs allowed loading to finish.
  This is consistent with browser connection contention, not a verified daemon defect.

Screenshots (local inspectable artifacts):
`docs/verification/kennel-mission-evidence/before-project-session-board.jpg`,
`docs/verification/kennel-mission-evidence/after-project-outcome-board.jpg`,
`docs/verification/kennel-mission-evidence/after-list.jpg`,
`docs/verification/kennel-mission-evidence/after-mission-wide.jpg`,
`docs/verification/kennel-mission-evidence/after-mission-narrow.jpg`.
API comparison: `/tmp/kennel-mission-evidence/api-outcomes.json`.
Earlier dropdown portfolio screenshot also remains in this task's browser tool history.

### Agent-led integration gaps

Existing AdaptiveIntakeSurface already uses material clarification/analysis APIs before
Outcome creation; Mission replan feedback uses the existing structured replan endpoint.
The generated conversation contract is Project-scoped. Do not present it as an
Outcome-scoped conversation or derive authority from its prose. Backend needs durable
Outcome message/question/answer identity, explicit authorized context references,
proposal-to-Contract/Plan revision linkage, pending/error/cancellation/replay semantics,
and attributable progress explanations. Automatic advancement after owner start needs
a durable authorized run-intent boundary; frontend per-WorkUnit starts cannot substitute
for it. These remain essential baseline integration gaps, not completed by this layout fix.


## Intake continuity follow-up (2026-09-10)

Three existing-API fixes after dd4f5870f:

- A confirmed-intake reload offers Open Outcome using `confirmedOutcome.id`; navigation
  performs no confirmation or creation POST.
- Successful confirm and reopen use the same Project-scoped Mission search parameters
  as the portfolio correction. Closing Mission retains that Project destination.
- A ready proposal starts with a read-only summary of all existing proposal fields and
  all eight permission flags. Optional Edit draft reuses IntakeContractReview and
  IntakeAuthorityEditor; returning to review retains edits. Provenance, validation,
  explicit confirmation, expectedProposalRevision and confirmation replay key remain.

Fresh checks: full frontend 221 files, 2715 passing, 6 skipped. Targeted intake plus
workspace/router checks: 21 passing. Typecheck, package and root lint succeeded.
Logs: `/tmp/kennel-intake-all-tests.log`, `/tmp/kennel-intake-final-targeted.log`,
`/tmp/kennel-intake-types.log`, `/tmp/kennel-intake-build.log`, `/tmp/kennel-intake-lint.log`.
New tests prove no write on confirmed reload/open, exact Project and Outcome navigation,
read-first constraints/evidence/permissions, revision fence and preserved edits on error.
These ready/confirmed states are API fixtures, not live model-backed proposal evidence.
Existing workspace tests prove close preserves portfolio input/focus; the complete
confirmed-intake -> Mission -> close live journey is still blocked before proposal.

Actual browser: submitted a natural-language statement against the disposable Project;
created intake `intake-3b3b0e84-458b-4f20-a921-889cccbef5eb`. Analysis returned500 at
15:47:42.931+05:30 because reasoning is not configured. Reload retained the same intake
ID and explicit failure; no Outcome was created and no paid provider call occurred.
Screenshot: `docs/verification/kennel-mission-evidence/intake-live-reasoning-blocked.jpg`.
The existing recovery UI still advertises Use the offline proposal instead, conflicting
with ADR0012; it was not invoked. Reported to the review task as an additional concrete
integration defect, unchanged by this bounded intake continuity follow-up.

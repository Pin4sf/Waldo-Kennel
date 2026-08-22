# Home and Personal Agent Foundations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the smallest useful Personal Home through confirmed Open Loops, Home-to-Work linking, governed desktop capture, and candidate-memory retrieval without enabling automatic durable Memory.

**Architecture:** The Kennel daemon and SQLite remain the only canonical writer. Home is a peer destination over shared `ResponsibilitySpace` and intake contracts; Work owns Outcome execution/proof, while Home owns Open Loops, capture sources, Context Episodes, and MemoryCandidates. Each task is an issue-sized vertical slice across domain, storage, trigger CDC, service, API, generated client, UI, recovery, and tests.

**Tech Stack:** Go 1.25, SQLite/goose/sqlc, Chi/OpenAPI, Electron/React 19/TanStack Router, Vitest/Testing Library/Playwright

**Spec:** `docs/superpowers/specs/2026-08-21-home-personal-agent-memory-design.md`

## Global Constraints

- Start each issue from current `origin/beta` in its own worktree and open the issue PR back to `beta`; promotion to `main` is a separate tested maintainer action.
- Coordinate migration numbers before editing. Numbers written below are the current allocation ledger, not permanent pins; if `beta` advances, the integration owner assigns the next unused number and renames the affected unmerged migration before its first implementation commit.
- Work owns Outcome through Acceptance. Home never writes Work lifecycle facts or infers Acceptance.
- Implement shared intake once in `backend/internal/service/intake`; Home and Work cannot create private Q&A systems.
- SQLite triggers write `change_log`; services do not emit parallel CDC.
- The renderer uses daemon APIs/generated types and never owns canonical Home state.
- Required capture capability never means default-on recording. Every modality needs a visible, revocable `CaptureGrant`.
- Sources and episodes produce candidates only. Durable `MemoryRevision` use is excluded.
- No hosted sync, mobile/wearable implementation, raw health data, autonomous remote effects, push, merge, release, or deployment.

---

## Cross-session execution framework

GitHub issues are the canonical status and ownership ledger. This plan is the canonical technical sequence and verification ledger. The approved specification remains the product/architecture authority. A session may refine implementation detail, but it must not silently change product truth, consent, responsibility, memory admission, or closure boundaries.

| Issue | Vertical slice | Owner | Blocked by | May proceed beside |
| --- | --- | --- | --- | --- |
| #18 | Home shell, five stable routes, truthful fixtures | `@Pin4sf` | None | Work lane |
| #23 | PersonalHome, Quick Capture, confirmed Open Loops | `@Pin4sf` | #21 | #36 after shared-file coordination |
| #29 | Today, contextual Catch Up, detail, Ready to Close | `@Pin4sf` | #23 | #32, #44, #45 |
| #44 | Daily Close receipt and exact re-entry | `@Pin4sf` | #23 | #29, #32, #45; consumes #36 later without waiting |
| #45 | Continuity-first History | `@Pin4sf` | #23 | #29, #32, #44; consumes Daily Close receipts when available |
| #32 | Shared intake and Home-to-Work lineage | `@Pin4sf` | #21, #23 | #29, #44, #45 |
| #36 | Governed screen/audio source plane | `@Pin4sf` | #21 | #23, #29, #32, #44, #45 |
| #39 | MemoryCandidate review and retrieval | `@Pin4sf` | #36 | Completed Home projections |
| #40 | Work-side shared integration | `@Developerr86`, `@Pin4sf` | #38, #32 | Home-only verification |
| #41 | Home privacy, recovery, and usefulness gate | `@Pin4sf` | #18, #23, #29, #32, #36, #39, #40, #44, #45 | None |

### Start an issue

1. Fetch current `origin/beta` without changing another issue's worktree.
2. Create or reopen the issue-recorded worktree and branch from current `origin/beta`; for example, #44 uses `codex/issue-44-daily-close` and #45 uses `codex/issue-45-home-history`.
3. Confirm a clean tracked worktree, preserving unrelated untracked research and mockups.
4. Comment on the issue with the branch, worktree, current beta commit as dated evidence, plan task number, shared files claimed, and first failing test command.
5. Allocate migrations and generated-contract ownership before editing shared files.

### Execute an issue

Use red-green-refactor in vertical increments: domain invariant, storage/CDC, service, HTTP/OpenAPI, thin renderer, recovery, and evaluation. Every increment starts with the named failing test, ends with focused green verification, and receives a conventional commit. A UI-only fixture issue follows the same cycle at route/component/integration-test boundaries.

Do not keep meaningful progress only in chat. After each independently reviewable increment, update the checkbox in this plan and add a compact issue comment containing the commit and verification command. Raw command output stays in the worktree or CI; the issue records the command, result, and relevant failure evidence.

### End a session

Add this exact handoff block to the issue before leaving work incomplete:

```markdown
### Session handoff — YYYY-MM-DD

- Branch/worktree:
- Beta base used as dated evidence:
- Completed commits:
- Verification passed:
- Current failing test or blocker:
- Shared files currently claimed:
- Next exact step:
- Product/architecture decision reopened: none | description and evidence
```

The next session reads the issue body, latest handoff, approved specification, this plan task, `git status`, and the branch diff before changing code. It refreshes from current `beta` before final verification, never treats an old base commit as a permanent pin, and never resets another session's work.

### Review and completion gates

1. **Contract gate:** domain invariants and authorization failures have focused tests before persistence or UI work.
2. **Boundary gate:** SQLite is the sole writer, trigger CDC is verified, API/spec/generated types land together, and the renderer stores no canonical truth.
3. **Experience gate:** populated, empty, partial/capture-off, stale, offline, keyboard/focus, reduced-motion, and exact-return states pass component tests and desktop preview.
4. **Issue gate:** focused backend/frontend tests, typecheck, API/sqlc drift checks where touched, and `git diff --check` pass.
5. **Integration gate:** refresh from current `beta`, resolve shared-file conflicts by contract ownership, rerun affected suites, request review, then open the PR to `beta`.
6. **Program gate:** #41 records the cross-issue privacy, recovery, deletion, usefulness, and Home-to-Work decision. No individual PR claims program completion.

---

### Task 1 (#18): Home destination shell, stable routes, and truthful fixtures

**Files:**
- Create: `frontend/src/renderer/routes/_shell.home.tsx`
- Create: `frontend/src/renderer/routes/_shell.home_.open-loops.tsx`
- Create: `frontend/src/renderer/routes/_shell.home_.memory.tsx`
- Create: `frontend/src/renderer/routes/_shell.home_.daily-close.tsx`
- Create: `frontend/src/renderer/routes/_shell.home_.history.tsx`
- Create: `frontend/src/renderer/components/home/HomeShell.tsx`
- Create: `frontend/src/renderer/components/home/HomeNavigation.tsx`
- Create: `frontend/src/renderer/components/home/HomeScreenFixture.tsx`
- Create: `frontend/src/renderer/components/home/ProvenanceInspector.tsx`
- Create: `frontend/src/renderer/components/home/HomeShell.test.tsx`
- Create: `frontend/src/renderer/lib/home-fixture.ts`
- Modify: `frontend/src/renderer/routes/_shell.tsx`
- Regenerate: `frontend/src/renderer/routeTree.gen.ts`
- Test: `frontend/src/renderer/__tests__/integration/home-entry.test.tsx`

**Interfaces:**
- Consumes: daemon readiness and Work-first routing
- Produces: `HomeShell`, five stable Home routes, contextual Catch Up fixture state, shared provenance-inspector fixture, read-only empty/partial/capture-off/offline modes; no persistence

- [ ] **Step 1: Write failing route and truthfulness tests**

Assert Home is directly selectable, Work remains recommended, `/home` opens Today, the other four stable routes select the matching navigation item, Catch Up is contextual rather than permanent navigation, and fixture cards say `Architecture preview` rather than pretending to be user data. Cover empty, partial/capture-off, stale, offline, inspector exact-return, keyboard focus, and narrow-width behavior.

- [ ] **Step 2: Verify failure**

```bash
npm --prefix frontend test -- --run home-entry HomeShell
```

Expected: FAIL because `/home` and `HomeShell` do not exist.

- [ ] **Step 3: Implement the minimal shell**

```ts
export type HomeDestination = "today" | "open_loops" | "memory" | "daily_close" | "history";
export type HomeMode = HomeDestination | "catch_up" | "ready_to_close";
export type HomeFixtureState = {
    kind: "preview_fixture";
    sourceLabel: "Architecture preview";
    mode: HomeMode;
    availability: "ready" | "partial" | "capture_off" | "offline";
};
```

Keep the desktop v0 navigation in the adaptive sidebar. Quick Capture is a persistent fixture action with an expanded Today fixture; it must not imply that intake persistence exists. Do not persist fixture state in localStorage or SQLite.

- [ ] **Step 4: Verify and commit**

```bash
npm --prefix frontend test -- --run home-entry HomeShell
npm --prefix frontend run typecheck
git add frontend/src/renderer
git commit -m "feat: add truthful Personal Home shell"
```

### Task 2 (#23): PersonalHome, Quick Capture, and confirmed Open Loops

**Files:**
- Create: `backend/internal/domain/home.go`, `open_loop.go`, `open_loop_test.go`
- Create: `backend/internal/ports/home_store.go`
- Create: `backend/internal/service/home/service.go`, `service_test.go`
- Create: `backend/internal/storage/sqlite/migrations/0103_personal_home_open_loops.sql`
- Create: `backend/internal/storage/sqlite/queries/home.sql`
- Create: `backend/internal/storage/sqlite/store/home_store.go`, `home_store_test.go`
- Create: `backend/internal/httpd/controllers/home.go`, `home_test.go`
- Modify: `backend/internal/httpd/controllers/dto.go`, `backend/internal/httpd/api.go`, `backend/internal/httpd/apispec/specgen/build.go`
- Regenerate: `backend/internal/storage/sqlite/gen/*`, `backend/internal/httpd/apispec/openapi.yaml`, `frontend/src/api/schema.ts`
- Create: `frontend/src/renderer/hooks/usePersonalHome.ts`
- Create: `frontend/src/renderer/components/home/QuickCapture.tsx`, `QuickCapture.test.tsx`

**Interfaces:**
- Consumes: `domain.ResponsibilitySpaceID` from the Work/shared contract PR
- Produces: `PersonalHome`, `OpenLoop`, `LoopDisposition`, `CreateOpenLoop`, `RecordDisposition`

- [ ] **Step 1: Write failing lifecycle tests**

```go
func TestOpenLoopRequiresExplicitConfirmation(t *testing.T) {
    candidate := domain.NewOpenLoopCandidate("Renew passport")
    if candidate.IsCanonical() { t.Fatal("candidate became responsibility") }
}
```

Also cover immutable dispositions, provider completion denial, idempotency replay, revision conflict, trigger CDC, restart, release, reopen, and transfer.

- [ ] **Step 2: Verify failure**

```bash
cd backend
go test ./internal/domain ./internal/service/home ./internal/storage/sqlite/store ./internal/httpd/controllers -run 'Home|OpenLoop|Disposition' -count=1
```

- [ ] **Step 3: Implement the minimum service contract**

```go
type Manager interface {
    GetHome(context.Context) (HomeView, error)
    Capture(context.Context, CaptureInput) (CaptureResult, error)
    CreateOpenLoop(context.Context, CreateOpenLoopInput) (OpenLoopView, error)
    RecordDisposition(context.Context, domain.OpenLoopID, DispositionInput) (OpenLoopView, error)
}
```

Quick Capture may save a note/candidate; only an explicit confirmation or unambiguous direct command creates `OpenLoop`.

- [ ] **Step 4: Add API and UI**

Expose `/api/v1/home`, `/api/v1/home/captures`, `/api/v1/open-loops`, `/api/v1/open-loops/{id}`, and `/api/v1/open-loops/{id}/dispositions`. Render source, owner, trigger, recheck, uncertainty, and confirm/dismiss choices.

- [ ] **Step 5: Regenerate, verify, and commit**

```bash
npm run sqlc && npm run api
cd backend && go test ./internal/domain ./internal/service/home ./internal/storage/sqlite/store ./internal/httpd/... -count=1
cd ../frontend && npm test -- --run QuickCapture && npm run typecheck
git add backend frontend
git commit -m "feat: add confirmed Home Open Loops"
```

### Task 3 (#29): Today, Catch Up, detail, and Ready to Close

**Files:**
- Create: `backend/internal/service/home/projection.go`, `projection_test.go`
- Create: `frontend/src/renderer/components/home/TodayBrief.tsx`
- Create: `frontend/src/renderer/components/home/CatchUp.tsx`
- Create: `frontend/src/renderer/components/home/OpenLoopDetail.tsx`
- Create: `frontend/src/renderer/components/home/ReadyToClose.tsx`
- Create: `frontend/src/renderer/components/home/HomeFlows.test.tsx`
- Modify: `frontend/src/renderer/routes/_shell.home.tsx`
- Test: `frontend/e2e/home-reentry.spec.ts`

**Interfaces:**
- Consumes: Task 2 APIs and durable disposition facts
- Produces: derived `TodayItem` and `CatchUpItem`; never a second status store

- [ ] **Step 1: Write failing projection tests**

Assert deterministic ordering from urgency/recheck/source freshness rather than activity volume; stale facts remain visible; Ready to Close is a suggestion; restart rebuilds the same projection.

- [ ] **Step 2: Implement projection and flows**

```go
type TodayItem struct {
    Kind, SubjectID, Reason, SourceFreshness, SuggestedAction string
    RequiresConfirmation bool
}
```

Catch Up shows one material decision at a time. Close/release/reopen/transfer always append `LoopDisposition`.

- [ ] **Step 3: Verify and commit**

```bash
cd backend && go test ./internal/service/home -run 'Today|CatchUp|ReadyToClose' -count=1
cd ../frontend && npm test -- --run HomeFlows && npm run typecheck
npx playwright test e2e/home-reentry.spec.ts
git add backend/internal/service/home frontend/src/renderer frontend/e2e
git commit -m "feat: add Home brief and conscious closure"
```

### Task 4 (#44): Daily Close receipt and exact re-entry

**Files:**
- Create: `backend/internal/domain/daily_close.go`, `daily_close_test.go`
- Create: `backend/internal/ports/daily_close_store.go`
- Create: `backend/internal/service/home/daily_close.go`, `daily_close_test.go`
- Create: `backend/internal/storage/sqlite/migrations/0107_daily_close_receipts.sql`
- Create: `backend/internal/storage/sqlite/queries/daily_close.sql`
- Create: `backend/internal/storage/sqlite/store/daily_close_store.go`, `daily_close_store_test.go`
- Create: `backend/internal/httpd/controllers/daily_close.go`, `daily_close_test.go`
- Modify: `backend/internal/httpd/controllers/dto.go`, `backend/internal/httpd/api.go`, `backend/internal/httpd/apispec/specgen/build.go`
- Regenerate: `backend/internal/storage/sqlite/gen/*`, `backend/internal/httpd/apispec/openapi.yaml`, `frontend/src/api/schema.ts`
- Create: `frontend/src/renderer/components/home/DailyCloseScreen.tsx`, `DailyCloseScreen.test.tsx`
- Create: `frontend/src/renderer/components/home/DailyCloseTimeline.tsx`
- Create: `frontend/src/renderer/components/home/ReEntrySelection.tsx`, `ReEntrySelection.test.tsx`
- Modify: `frontend/src/renderer/routes/_shell.home_.daily-close.tsx`
- Test: `frontend/e2e/home-daily-close.spec.ts`

**Interfaces:**
- Consumes: Task 2 confirmed Open Loops, trusted Home/Work facts, explicit notes, optional Context Episodes, shared provenance inspection
- Produces: `DailyCloseProjection`, immutable `DailyCloseReceipt`, `FinishDailyClose`, and exact `ReEntryReference`; never a responsibility disposition

- [ ] **Step 1: Write failing domain and service tests**

Cover local-date boundaries, partial source coverage, acknowledged gaps, immutable receipts, idempotency-key replay, a second distinct review for the same date, current-state resolution of re-entry references, and denial of implicit Open Loop/Outcome mutation.

```go
func TestFinishDailyCloseDoesNotDisposeResponsibilities(t *testing.T) {
    receipt, err := manager.Finish(ctx, FinishDailyCloseInput{
        LocalDate: "2026-08-22",
        ReviewedThrough: time.Date(2026, 8, 22, 22, 0, 0, 0, time.Local),
        ReEntry: []domain.ReEntryReference{{Kind: "open_loop", ID: "loop-1"}},
        IdempotencyKey: "close-2026-08-22-device-1",
    })
    require.NoError(t, err)
    require.Equal(t, "loop-1", receipt.ReEntry[0].ID)
    require.Empty(t, dispositionStore.Appended())
}
```

- [ ] **Step 2: Run the focused tests to verify failure**

```bash
cd backend
go test ./internal/domain ./internal/service/home ./internal/storage/sqlite/store -run 'DailyClose|ReEntry' -count=1
```

Expected: FAIL because the Daily Close contracts and store do not exist.

- [ ] **Step 3: Implement the minimum canonical and read contracts**

```go
type DailyCloseReceiptID string

type ReEntryReference struct {
    Kind string
    ID   string
}

type DailyCloseReceipt struct {
    ID                  DailyCloseReceiptID
    LocalDate           string
    ReviewedThrough     time.Time
    AcknowledgedGapRefs []string
    Note                string
    ReEntry             []ReEntryReference
    IdempotencyKey      string
    CreatedAt           time.Time
}

type FinishDailyCloseInput struct {
    LocalDate       string
    ReviewedThrough time.Time
    AcknowledgedGapRefs []string
    Note            string
    ReEntry         []domain.ReEntryReference
    IdempotencyKey  string
}

type DailyCloseItem struct {
    Kind            string
    CanonicalID     string
    Summary         string
    OccurredAt      time.Time
    ProvenanceState string
}

type DailyCloseProjection struct {
    LocalDate       string
    State           string
    ReviewedThrough time.Time
    Items           []DailyCloseItem
    GapRefs         []string
}

type ReEntryView struct {
    ReceiptID domain.DailyCloseReceiptID
    Resolved  []domain.ReEntryReference
    Missing   []domain.ReEntryReference
}

type DailyCloseManager interface {
    Projection(context.Context, string) (DailyCloseProjection, error)
    Finish(context.Context, FinishDailyCloseInput) (domain.DailyCloseReceipt, error)
    ReEntry(context.Context, domain.DailyCloseReceiptID) (ReEntryView, error)
}
```

`DailyCloseProjection` is rebuildable and never stored as display status. `DailyCloseReceipt` stores the reviewed-through watermark, acknowledged gap references, optional note, and responsibility references. Native correction and disposition services remain the only way to change responsibility truth.

- [ ] **Step 4: Add storage, CDC, HTTP, generated types, and UI**

Expose `GET /api/v1/home/daily-close?date=YYYY-MM-DD`, `POST /api/v1/home/daily-close/receipts`, and `GET /api/v1/home/daily-close/receipts/{id}/re-entry`. The UI renders collecting, partial, ready, empty, and offline states; makes gaps inspectable; preserves an unsent note; and asks for explicit native commands when the user changes an Open Loop.

- [ ] **Step 5: Verify recovery, experience, and API drift**

```bash
npm run sqlc && npm run api
cd backend && go test ./internal/domain ./internal/service/home ./internal/storage/sqlite/store ./internal/httpd/... -run 'DailyClose|ReEntry' -count=1
cd ../frontend && npm test -- --run DailyCloseScreen ReEntrySelection && npm run typecheck
npx playwright test e2e/home-daily-close.spec.ts
```

Expected: PASS; restarting or repeating the same idempotency key returns one receipt and no implicit dispositions.

- [ ] **Step 6: Commit the complete slice**

```bash
git add backend frontend
git commit -m "feat: add Daily Close and exact re-entry"
```

### Task 5 (#45): Continuity-first Home History

**Files:**
- Create: `backend/internal/service/home/continuity.go`, `continuity_test.go`
- Create: `backend/internal/httpd/controllers/home_history.go`, `home_history_test.go`
- Modify: `backend/internal/httpd/controllers/dto.go`, `backend/internal/httpd/api.go`, `backend/internal/httpd/apispec/specgen/build.go`
- Regenerate: `backend/internal/httpd/apispec/openapi.yaml`, `frontend/src/api/schema.ts`
- Create: `frontend/src/renderer/components/home/HistoryScreen.tsx`, `HistoryScreen.test.tsx`
- Create: `frontend/src/renderer/components/home/ContinuityTimeline.tsx`, `ContinuityTimeline.test.tsx`
- Create: `frontend/src/renderer/components/home/HistoryFilters.tsx`
- Modify: `frontend/src/renderer/routes/_shell.home_.history.tsx`
- Test: `frontend/e2e/home-history.spec.ts`

**Interfaces:**
- Consumes: Task 2 canonical Home facts, native correction/disposition records, permitted Work facts, optional Daily Close receipts, and shared provenance inspection
- Produces: deterministic `ContinuityEventPage` with stable cursor ordering; no new canonical event store

- [ ] **Step 1: Write failing read-model and pagination tests**

Cover continuity-first ordering, identical timestamps ordered by stable ID, bounded cursor pagination without duplicate/skip, missing/deleted provenance, activity/audit filters, deterministic rebuild, and absence of raw `change_log` payloads from product DTOs.

```go
func TestContinuityHistoryDoesNotExposeRawChangeLog(t *testing.T) {
    page, err := service.History(ctx, HistoryQuery{View: "continuity", Limit: 25})
    require.NoError(t, err)
    require.NotEmpty(t, page.Events)
    require.NotEmpty(t, page.Events[0].CanonicalRef.ID)
    encoded, err := json.Marshal(page.Events[0])
    require.NoError(t, err)
    require.NotContains(t, string(encoded), "rawChange")
}
```

- [ ] **Step 2: Run the focused tests to verify failure**

```bash
cd backend
go test ./internal/service/home ./internal/httpd/controllers -run 'Continuity|HomeHistory' -count=1
```

Expected: FAIL because the History query and DTO do not exist.

- [ ] **Step 3: Implement the rebuildable History contract**

```go
type ContinuityEvent struct {
    ID               string
    Family           string
    OccurredAt       time.Time
    CanonicalRef     ContinuityReference
    Summary          string
    ProvenanceState  string
    SourceRef        *ContinuityReference
}

type ContinuityReference struct {
    Kind string
    ID   string
}

type HistoryQuery struct {
    View   string
    Cursor string
    Limit  int
}

type ContinuityEventPage struct {
    Events     []ContinuityEvent
    NextCursor string
}

type ContinuitySource interface {
    Events(context.Context, HistoryQuery) ([]ContinuityEvent, error)
}
```

Derive events from native records at read time or through a rebuildable projection. Default families are explicit capture, correction, candidate decision, Open Loop disposition, ResponsibilityLink change, permitted Outcome fact, Daily Close receipt, and re-entry. Activity and audit are opt-in views. Register native fact adapters behind `ContinuitySource`; the Daily Close adapter is registered only when Task 4 is present, so Tasks 4 and 5 can land in either order without redefining each other's contracts.

- [ ] **Step 4: Add bounded API and continuity-first UI**

Expose `GET /api/v1/home/history?view=continuity|activity|audit&cursor=<opaque>&limit=25`. The UI defaults to continuity, discloses missing or deleted sources, opens the shared inspector, returns focus/scroll to the originating event, and provides populated, empty, partial, and offline states.

- [ ] **Step 5: Verify rebuild, pagination, accessibility, and generated types**

```bash
npm run api
cd backend && go test ./internal/service/home ./internal/httpd/... -run 'Continuity|HomeHistory' -count=1
cd ../frontend && npm test -- --run HistoryScreen ContinuityTimeline && npm run typecheck
npx playwright test e2e/home-history.spec.ts
```

Expected: PASS; a rebuild returns the same ordered event IDs, and page boundaries neither duplicate nor skip events.

- [ ] **Step 6: Commit the complete slice**

```bash
git add backend frontend
git commit -m "feat: add continuity-first Home History"
```

### Task 6 (#32): Shared adaptive intake and Home-to-Work lineage

**Files:**
- Create: `backend/internal/domain/intake.go`, `responsibility_link.go`, `intake_test.go`
- Create: `backend/internal/ports/intake_store.go`
- Create: `backend/internal/service/intake/service.go`, `service_test.go`
- Create: `backend/internal/storage/sqlite/migrations/0104_shared_intake_responsibility_links.sql`
- Create: `backend/internal/storage/sqlite/queries/intake.sql`
- Create: `backend/internal/storage/sqlite/store/intake_store.go`
- Create: `backend/internal/httpd/controllers/intake.go`, `intake_test.go`
- Modify: `backend/internal/httpd/controllers/dto.go`, `backend/internal/httpd/api.go`, `backend/internal/httpd/apispec/specgen/build.go`
- Regenerate: `backend/internal/storage/sqlite/gen/*`, `backend/internal/httpd/apispec/openapi.yaml`, `frontend/src/api/schema.ts`
- Create: `frontend/src/renderer/components/intake/AdaptiveIntake.tsx`
- Create: `frontend/src/renderer/components/home/ConnectHomeToWork.tsx`, `ConnectHomeToWork.test.tsx`

**Interfaces:**
- Consumes: current Work `OutcomeID`, Task 2 `OpenLoopID`, Project API
- Produces: `IntakeSession`, `ClarificationRequest`, `ResponsibilityProposal`, immutable `ResponsibilityLink`

- [ ] **Step 1: Write failing separation tests**

Cover one material question, non-canonical proposals, no lifecycle coupling through links, missing-Project denial, idempotent duplicate link, and ended-reason lineage.

- [ ] **Step 2: Implement the shared proposal vocabulary**

```go
type ProposalKind string
const (
    ProposalNote ProposalKind = "note"
    ProposalOpenLoop ProposalKind = "open_loop"
    ProposalOutcome ProposalKind = "outcome"
    ProposalLink ProposalKind = "responsibility_link"
    ProposalDismiss ProposalKind = "dismiss"
)
```

Both Home and Work call this service; neither duplicates its transition rules.

- [ ] **Step 3: Regenerate, verify, and commit**

```bash
npm run sqlc && npm run api
cd backend && go test ./internal/domain ./internal/service/intake ./internal/httpd/... -count=1
cd ../frontend && npm test -- --run ConnectHomeToWork AdaptiveIntake && npm run typecheck
git add backend frontend
git commit -m "feat: add shared intake and Home-to-Work lineage"
```

### Task 7 (#36): Governed desktop screen and audio source plane

**Files:**
- Create: `backend/internal/domain/capture.go`, `source.go`
- Create: `backend/internal/ports/capture_store.go`, `capture_adapter.go`
- Create: `backend/internal/service/capture/service.go`, `worker.go`, tests
- Create: `backend/internal/storage/sqlite/migrations/0105_capture_sources_context_episodes.sql`
- Create: `backend/internal/storage/sqlite/queries/capture.sql`
- Create: `backend/internal/storage/sqlite/store/capture_store.go`
- Create: `backend/internal/httpd/controllers/capture.go`, `capture_test.go`
- Create: `frontend/src/main/capture-broker.ts`, `capture-broker.test.ts`
- Create: `frontend/src/renderer/components/settings/CaptureGrants.tsx`, `CaptureGrants.test.tsx`
- Create: `frontend/src/renderer/components/home/ContextEpisodeReview.tsx`
- Modify: `backend/internal/httpd/controllers/dto.go`, `backend/internal/httpd/api.go`, `backend/internal/httpd/apispec/specgen/build.go`
- Regenerate: `backend/internal/storage/sqlite/gen/*`, `backend/internal/httpd/apispec/openapi.yaml`, `frontend/src/api/schema.ts`

**Interfaces:**
- Consumes: OS permission state via Electron main and `ResponsibilitySpaceID`
- Produces: `CaptureGrant`, `SourceArtifact`, `Observation`, `ContextEpisode`; never Memory truth

- [ ] **Step 1: Write failing grant/deletion tests**

Cover disabled-by-default modalities, complete purpose/scope/route/retention, exclusions before persistence, visible gaps, idempotent ingestion, crash recovery, deletion generation, and derived scrubbing.

- [ ] **Step 2: Implement grant-first ports and deterministic fakes**

```go
type Adapter interface {
    Capabilities(context.Context) ([]domain.CaptureModality, error)
    Start(context.Context, domain.CaptureGrant) (<-chan Segment, error)
    Stop(context.Context, domain.CaptureGrantID) error
}
```

CI uses fake screen/system-audio/microphone adapters. A real macOS adapter is injected only through this port after privacy tests pass.

- [ ] **Step 3: Add user controls and episode correction**

Render disabled/denied/enabled/paused/stale/failed, exclusions, disclosure, retention/storage cap, export/revoke/delete. Episode review supports correct/split/merge/dismiss/expire and source gaps.

- [ ] **Step 4: Regenerate, verify, and commit**

```bash
npm run sqlc && npm run api
cd backend && go test ./internal/domain ./internal/service/capture ./internal/storage/sqlite/store ./internal/httpd/... -count=1
cd ../frontend && npm test -- --run CaptureGrants capture-broker ContextEpisode && npm run typecheck
git add backend frontend
git commit -m "feat: add governed desktop capture source plane"
```

### Task 8 (#39): MemoryCandidate review and purpose-bound retrieval

**Files:**
- Create: `backend/internal/domain/memory_candidate.go`, `memory_candidate_test.go`
- Create: `backend/internal/ports/memory_candidate_store.go`
- Create: `backend/internal/service/memorycandidate/service.go`, `retrieval.go`, tests
- Create: `backend/internal/storage/sqlite/migrations/0106_memory_candidates_retrieval_receipts.sql`
- Create: `backend/internal/storage/sqlite/queries/memory_candidates.sql`
- Create: `backend/internal/storage/sqlite/store/memory_candidate_store.go`
- Create: `backend/internal/httpd/controllers/memory_candidates.go`, tests
- Create: `frontend/src/renderer/components/home/MemoryCandidateReview.tsx`, tests
- Create: `frontend/src/renderer/components/settings/MemoryControls.tsx`, tests
- Modify: `backend/internal/httpd/controllers/dto.go`, `backend/internal/httpd/api.go`, `backend/internal/httpd/apispec/specgen/build.go`
- Regenerate: `backend/internal/storage/sqlite/gen/*`, `backend/internal/httpd/apispec/openapi.yaml`, `frontend/src/api/schema.ts`

**Interfaces:**
- Consumes: source/episode generations and responsibility scope policy
- Produces: `MemoryCandidate`, candidate review decisions, `RetrievalReceipt`; no active `MemoryRevision`

- [ ] **Step 1: Write failing boundary tests**

Cover provenance spans, uncertainty, valid-time proposal, sensitivity, counter-evidence, expiry, revocation, deletion generation, cross-space denial, stale-index hydration, non-resurrection, and zero authority/responsibility/proof creation.

- [ ] **Step 2: Implement candidate and retrieval contracts**

```go
type RetrievalRequest struct {
    Purpose, Caller string
    EligibleSpaces []domain.ResponsibilitySpaceID
    MaxItems int
    PolicyGeneration int64
}
```

Use SQLite FTS as a rebuildable lexical projection. Semantic retrieval is an injected interface with a deterministic fake; do not add a second canonical store.

- [ ] **Step 3: Regenerate, verify, and commit**

```bash
npm run sqlc && npm run api
cd backend && go test ./internal/domain ./internal/service/memorycandidate ./internal/storage/sqlite/store ./internal/httpd/... -count=1
cd ../frontend && npm test -- --run MemoryCandidate MemoryControls && npm run typecheck
git add backend frontend
git commit -m "feat: add governed memory candidates and retrieval"
```

### Task 9 (#41): Privacy, recovery, and usefulness gate

**Files:**
- Create: `backend/e2e/home_personal_agent_test.go`
- Create: `backend/e2e/home_deletion_non_resurrection_test.go`
- Create: `test/fixtures/home-personal-agent/*.json`
- Create: `frontend/e2e/home-personal-agent.spec.ts`
- Create: `docs/evaluation/home-personal-agent-gate.md`
- Modify: `scripts/test-foundation.sh`

**Interfaces:**
- Consumes: Tasks 1-8, #40 cross-lane integration, and current Work Outcome fixtures
- Produces: reproducible release-gate report; no automatic promotion

- [ ] **Step 1: Add adversarial fixtures and failing assertions**

Include correction, ambiguous speaker, third-party claim, deleted source, changed preference, capture gap, prompt injection, provider failure, cross-space request, Home-to-Work disclosure, false Ready to Close, restart, reindex, and replay.

- [ ] **Step 2: Run the focused gate**

```bash
cd backend && go test ./e2e -run 'Home|DeletionNonResurrection' -count=1
cd ../frontend && npx playwright test e2e/home-personal-agent.spec.ts
```

- [ ] **Step 3: Run the repository gate and record results**

```bash
npm run test:foundation
npm run audit:production
npm run lint
npm run frontend:typecheck
cd backend && go test -race ./...
```

Record candidate precision, review burden, provenance completeness, false-loop/false-close rate, storage/energy/model cost, and p50/p95 latency. A failed blocker keeps durable Memory disabled.

- [ ] **Step 4: Commit**

```bash
git add backend/e2e frontend/e2e test/fixtures docs/evaluation scripts/test-foundation.sh
git commit -m "test: gate Home and Personal Agent foundations"
```

## Execution Handoff

Task 1 may run beside Work immediately. Task 2 waits for shared `ResponsibilitySpaceID`. Tasks 3, 4, and 5 wait for Task 2 but have no semantic dependency on one another. Task 6 waits for current Work `OutcomeID` and Task 2. Task 7 may use deterministic fake adapters after its shared contract prerequisite; Task 4 must disclose capture gaps and can consume Task 7 episodes later without waiting. Task 8 waits for Task 7 and cannot activate durable Memory. Task 9 waits for Tasks 1-8 plus #40.

Execute every task as its linked issue and one PR using the ownership matrix in the specification. Keep issue comments and this plan synchronized through the start, incremental-commit, end-of-session, review, and integration ledger rules above.

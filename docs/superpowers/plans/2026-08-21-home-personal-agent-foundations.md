# Home and Personal Agent Foundations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the smallest useful Personal Home through confirmed Open Loops, Home-to-Work linking, governed desktop capture, and candidate-memory retrieval without enabling automatic durable Memory.

**Architecture:** The Kennel daemon and SQLite remain the only canonical writer. Home is a peer destination over shared `ResponsibilitySpace` and intake contracts; Work owns Outcome execution/proof, while Home owns Open Loops, capture sources, Context Episodes, and MemoryCandidates. Each task is an issue-sized vertical slice across domain, storage, trigger CDC, service, API, generated client, UI, recovery, and tests.

**Tech Stack:** Go 1.25, SQLite/goose/sqlc, Chi/OpenAPI, Electron/React 19/TanStack Router, Vitest/Testing Library/Playwright

**Spec:** `docs/superpowers/specs/2026-08-21-home-personal-agent-memory-design.md`

## Global Constraints

- Start each issue from current `origin/beta` in its own worktree and open the issue PR back to `beta`; promotion to `main` is a separate tested maintainer action.
- Coordinate migration numbers before editing. Names below reserve Work `0099-0102` and Home `0103-0106`; if `beta` advances, the integration owner renumbers the affected unmerged issue before its first commit.
- Work owns Outcome through Acceptance. Home never writes Work lifecycle facts or infers Acceptance.
- Implement shared intake once in `backend/internal/service/intake`; Home and Work cannot create private Q&A systems.
- SQLite triggers write `change_log`; services do not emit parallel CDC.
- The renderer uses daemon APIs/generated types and never owns canonical Home state.
- Required capture capability never means default-on recording. Every modality needs a visible, revocable `CaptureGrant`.
- Sources and episodes produce candidates only. Durable `MemoryRevision` use is excluded.
- No hosted sync, mobile/wearable implementation, raw health data, autonomous remote effects, push, merge, release, or deployment.

---

### Task 1: Home destination shell and truthful fixtures

**Files:**
- Create: `frontend/src/renderer/routes/_shell.home.tsx`
- Create: `frontend/src/renderer/components/home/HomeShell.tsx`
- Create: `frontend/src/renderer/components/home/HomeShell.test.tsx`
- Create: `frontend/src/renderer/lib/home-fixture.ts`
- Modify: `frontend/src/renderer/routes/_shell.tsx`
- Regenerate: `frontend/src/renderer/routeTree.gen.ts`
- Test: `frontend/src/renderer/__tests__/integration/home-entry.test.tsx`

**Interfaces:**
- Consumes: daemon readiness and Work-first routing
- Produces: `HomeShell`, `/home`, read-only empty/fixture modes; no persistence

- [ ] **Step 1: Write failing route and truthfulness tests**

Assert Home is directly selectable, Work remains recommended, Home has useful empty/capture-disabled/offline states, and fixture cards say `Architecture preview` rather than pretending to be user data.

- [ ] **Step 2: Verify failure**

```bash
npm --prefix frontend test -- --run home-entry HomeShell
```

Expected: FAIL because `/home` and `HomeShell` do not exist.

- [ ] **Step 3: Implement the minimal shell**

```ts
export type HomeMode = "today" | "catch_up" | "open_loop" | "ready_to_close";
export type HomeFixtureState = { kind: "preview_fixture"; sourceLabel: "Architecture preview"; mode: HomeMode };
```

Do not persist fixture state in localStorage or SQLite.

- [ ] **Step 4: Verify and commit**

```bash
npm --prefix frontend test -- --run home-entry HomeShell
npm --prefix frontend run typecheck
git add frontend/src/renderer
git commit -m "feat: add truthful Personal Home shell"
```

### Task 2: PersonalHome, Quick Capture, and confirmed Open Loops

**Files:**
- Create: `backend/internal/domain/home.go`, `open_loop.go`, `open_loop_test.go`
- Create: `backend/internal/ports/home_store.go`
- Create: `backend/internal/service/home/service.go`, `service_test.go`
- Create: `backend/internal/storage/sqlite/migrations/0103_personal_home_open_loops.sql`
- Create: `backend/internal/storage/sqlite/queries/home.sql`
- Create: `backend/internal/storage/sqlite/store/home_store.go`, `home_store_test.go`
- Create: `backend/internal/httpd/controllers/home.go`, `home_test.go`
- Modify: `backend/internal/httpd/controllers/dto.go`, `backend/internal/httpd/api.go`, `backend/internal/httpd/apispec/specgen/build.go`
- Regenerate: sqlc, OpenAPI, frontend schema
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

### Task 3: Today, Catch Up, detail, and Ready to Close

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

### Task 4: Shared adaptive intake and Home-to-Work lineage

**Files:**
- Create: `backend/internal/domain/intake.go`, `responsibility_link.go`, `intake_test.go`
- Create: `backend/internal/ports/intake_store.go`
- Create: `backend/internal/service/intake/service.go`, `service_test.go`
- Create: `backend/internal/storage/sqlite/migrations/0104_shared_intake_responsibility_links.sql`
- Create: `backend/internal/storage/sqlite/queries/intake.sql`
- Create: `backend/internal/storage/sqlite/store/intake_store.go`
- Create: `backend/internal/httpd/controllers/intake.go`, `intake_test.go`
- Modify/regenerate: DTO, API registry, sqlc, OpenAPI, frontend schema
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

### Task 5: Governed desktop screen and audio source plane

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
- Modify/regenerate: DTO, API registry, sqlc, OpenAPI, frontend schema

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

### Task 6: MemoryCandidate review and purpose-bound retrieval

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
- Modify/regenerate: DTO, routes, sqlc, OpenAPI, frontend schema

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

### Task 7: Privacy, recovery, and usefulness gate

**Files:**
- Create: `backend/e2e/home_personal_agent_test.go`
- Create: `backend/e2e/home_deletion_non_resurrection_test.go`
- Create: `test/fixtures/home-personal-agent/*.json`
- Create: `frontend/e2e/home-personal-agent.spec.ts`
- Create: `docs/evaluation/home-personal-agent-gate.md`
- Modify: `scripts/test-foundation.sh`

**Interfaces:**
- Consumes: Tasks 1-6 and current Work Outcome fixtures
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

Task 1 may run beside Work immediately. Task 2 waits for shared `ResponsibilitySpaceID`; Task 4 waits for current Work `OutcomeID`; Tasks 5-6 may use fake adapters but cannot activate capture or durable Memory before their gates. Execute each task as a separate issue and PR using the ownership matrix in the specification.

# Work Control Plane Delivery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the approved Board -> Outcome Mission Control -> Session Inspector flow with durable Project conversation, adaptive Outcome intake, capability-based role resolution, bounded provider continuation, and one evaluated adaptive Mission.

**Architecture:** Keep the daemon and SQLite authoritative. Reuse the existing session chat/runtime as provider machinery while adding Project/Outcome conversation and intake aggregates that reference rather than copy provider transcripts. Deliver independent vertical slices in dependency order; each slice crosses persistence, CDC, service, API, UI, restart, and user-visible error behavior required for its truth boundary.

**Tech Stack:** Go daemon and services, SQLite/sqlc with trigger-backed `change_log`, code-first OpenAPI and generated TypeScript, Electron, React 19, TanStack Router, Vitest/Testing Library, Playwright.

**Spec:** `docs/superpowers/specs/2026-08-25-work-control-plane-canonical-flow-design.md`

## Global Constraints

- The Outcome lineage is `Outcome -> ContractRevision -> PlanRevision -> WorkUnit -> Attempt -> AgentSessionRef -> EvidenceItem -> VerificationRun -> AcceptanceDecision`.
- Provider/session completion, commits, checks, PRs, screenshots, and assistant confidence never accept or close an Outcome.
- The renderer never calls providers, compiles authoritative context, writes SQLite, stores derived session status, or emits manual CDC.
- A material provider/model/role/topology/authority/workspace/budget change creates a new revision or Attempt and receives the required user decision; there is no silent provider fallback.
- Project conversation continuity does not depend on one provider-native session or imply lossless context.
- SQLite is the canonical writer; Markdown, indexes, and provider transcripts are subordinate projections or sources.
- Each task starts from freshly fetched `origin/beta`, claims the next unused migration/shared API lease atomically, regenerates source-owned artifacts, and opens one issue-specific PR back to `beta`.
- Home integration, ambient capture, durable admitted Memory, hosted attachment, and automatic learned-policy activation remain outside these Work slices.

---

### Task 1: Project default agent and capability-based role resolution

**Files:**
- Modify: `backend/internal/domain/projectconfig.go`
- Modify: `backend/internal/domain/projectconfig_test.go`
- Modify: `backend/internal/service/project/service.go`
- Modify: `backend/internal/service/project/service_test.go`
- Modify: `backend/internal/service/agent/service.go`
- Modify: `backend/internal/httpd/controllers/dto.go`
- Modify: `backend/internal/httpd/controllers/projects.go`
- Modify: `backend/internal/httpd/apispec/specgen/build.go`
- Modify: `frontend/src/renderer/components/CreateProjectAgentSheet.tsx`
- Modify: `frontend/src/renderer/components/ProjectSettingsForm.tsx`
- Modify: associated component and controller tests
- Regenerate: `backend/internal/httpd/apispec/openapi.yaml`
- Regenerate: `frontend/src/api/schema.ts`

**Interfaces:**
- Consumes: daemon agent inventory entries with `roles.worker`, `roles.coordinator`, `roles.switchTarget`, `requiresProfile`, readiness, installed, and authorization facts.
- Produces: persisted Project preference for the default coding agent plus a daemon-resolved role proposal that distinguishes preference from current eligibility.

```go
type ProjectAgentPreferences struct {
	DefaultWorker string
	Analyzer      string
	Coordinator   string
	Verifier      string
}

type ResolvedMissionRoles struct {
	Analyzer    ResolvedAgentRole
	Coordinator ResolvedAgentRole
	Worker      ResolvedAgentRole
	Verifier    ResolvedAgentRole
}
```

- [ ] **Step 1: Write failing domain/service tests** proving Project creation persists the selected default worker, a required DeepSeek profile fails closed, unsupported role preferences are rejected, and changing defaults does not rewrite historical sessions or approved Plans.
- [ ] **Step 2: Run the narrow tests** with `cd backend && go test ./internal/domain ./internal/service/project ./internal/service/agent` and confirm the new cases fail for missing preference/role resolution behavior.
- [ ] **Step 3: Implement daemon-owned preference validation and role resolution** using the served capability inventory rather than provider-name checks in React.
- [ ] **Step 4: Add controller/API tests** for missing profile, unavailable default, capability mismatch, safe error envelope, request ID, and successful read-back.
- [ ] **Step 5: Regenerate API artifacts** with `npm run api` and run `cd backend && go test ./internal/httpd/...`.
- [ ] **Step 6: Simplify onboarding UI** so the default path asks for Default coding agent plus required profile/model, while advanced role preferences remain explicit and actual Mission assignments are labeled as proposals.
- [ ] **Step 7: Run frontend tests and typecheck** with the exact component tests and `npm run frontend:typecheck`.
- [ ] **Step 8: Commit** with `git commit -m "feat: resolve project agent roles by capability"`.

### Task 2: Durable Project Waldo conversation and bounded continuation

**Files:**
- Create: `backend/internal/domain/waldo_conversation.go`
- Create: `backend/internal/ports/waldo_conversation_store.go`
- Create: `backend/internal/service/waldoconversation/service.go`
- Create: `backend/internal/service/waldoconversation/service_test.go`
- Create: next unused additive SQLite migration for Project conversation episode/turn references and continuation receipts
- Create: `backend/internal/storage/sqlite/queries/waldo_conversations.sql`
- Create: `backend/internal/storage/sqlite/store/waldo_conversation_store.go`
- Create: `backend/internal/storage/sqlite/store/waldo_conversation_store_test.go`
- Create: `backend/internal/httpd/controllers/waldo_conversations.go`
- Modify: `backend/internal/httpd/controllers/dto.go`
- Modify: `backend/internal/httpd/apispec/specgen/build.go`
- Modify: daemon/controller registration
- Modify: `frontend/src/renderer/components/waldo/WaldoRail.tsx`
- Modify: `frontend/src/renderer/components/waldo/WaldoRailContext.tsx`
- Add: focused backend/frontend/e2e tests
- Regenerate: sqlc and API artifacts

**Interfaces:**
- Consumes: existing provider-native session chat, conversation compaction, session recovery/readmission, Project/Outcome IDs, adapter usage/context signals, and generated daemon client.
- Produces: provider-neutral Project conversation episodes/turns, visible context attachments, bounded continuation receipts, and one Project conversation projection independent of provider session identity.

```go
type ConversationEpisode struct {
	ID                    string
	ResponsibilitySpaceID string
	ProjectID             string
	State                 string
	CreatedAt             time.Time
}

type ConversationContextRef struct {
	Kind     string
	ObjectID string
	Revision string
}

type ContinuationReceipt struct {
	FromAgentSessionRef string
	ToAgentSessionRef   string
	Reason              string
	MaterialChange      bool
	ContextDigest       string
}
```

- [ ] **Step 1: Write failing domain/store tests** for ordered idempotent turns, Project binding, explicit context refs, restart read-back, no transcript copying into Intake, and trigger-backed CDC.
- [ ] **Step 2: Run narrow tests** and verify the missing domain/store contracts fail.
- [ ] **Step 3: Implement the minimum conversation aggregate, store, and service** while reusing existing session chat only behind provider/runtime ports.
- [ ] **Step 4: Write failing continuation-policy tests** for trustworthy context reserve, conservative adapter threshold, material digest change, lost/non-resumable identity, fresh-verifier boundary, source revocation, safe automatic same-authority rollover, material-change Needs You, and ambiguous replacement without duplicate start.
- [ ] **Step 5: Implement continuation decisions and receipts** so automatic rollover requires unchanged canonical bindings, safe fencing, no unknown effect, and confirmed replacement identity.
- [ ] **Step 6: Add typed HTTP endpoints and error coverage**, regenerate sqlc/OpenAPI/TypeScript, and run drift tests.
- [ ] **Step 7: Wire the Waldo rail** to show `Waldo · <project>`, an editable context chip, non-disruptive continuation receipt, unavailable/offline truth, and explicit Home redirect for personal scope.
- [ ] **Step 8: Verify Electron re-entry** across daemon restart, automatic rollover, material rollover refusal, context detach, keyboard/focus return, and narrow layout.
- [ ] **Step 9: Commit** with `git commit -m "feat: add durable project Waldo conversation"`.

### Task 3: Shared adaptive IntakeSession and stable-core Contract proposal

**Files:**
- Create: `backend/internal/domain/intake.go`
- Create: `backend/internal/domain/intake_test.go`
- Create: `backend/internal/ports/intake_store.go`
- Create: `backend/internal/ports/intake_analyzer.go`
- Create: `backend/internal/service/intake/service.go`
- Create: `backend/internal/service/intake/service_test.go`
- Create: next unused additive SQLite migration for IntakeSession, clarification, proposal revisions, and conversation references
- Create: `backend/internal/storage/sqlite/queries/intakes.sql`
- Create: `backend/internal/storage/sqlite/store/intake_store.go`
- Create: `backend/internal/storage/sqlite/store/intake_store_test.go`
- Create: `backend/internal/httpd/controllers/intakes.go`
- Modify: `backend/internal/httpd/controllers/dto.go`
- Modify: `backend/internal/httpd/apispec/specgen/build.go`
- Modify: daemon/controller registration
- Modify: `frontend/src/renderer/components/Sidebar.tsx`
- Modify: `frontend/src/renderer/components/outcome/WorkEnterSurface.tsx`
- Modify: `frontend/src/renderer/components/outcome/OutcomeUnderstandSurface.tsx`
- Create: adaptive intake modal and proposal components under `frontend/src/renderer/components/outcome/`
- Add: focused backend/frontend/e2e tests

**Interfaces:**
- Consumes: Project conversation span references, Project context compiler, analyzer port, existing Outcome/Contract service, Project agent preferences, and idempotency/revision infrastructure.
- Produces: immutable proposal revisions with stable Contract core and typed facets; confirmation creates exactly one Outcome/ContractRevision.

```go
type OutcomeContractProposal struct {
	Title              string
	DesiredState       string
	SuccessCriteria    []ProposedCriterion
	EvidenceExpected   []ProposedEvidence
	ReviewMethod       string
	Constraints        []string
	NonGoals           []string
	AuthorityCeiling   ProposedAuthority
	StopConditions     []string
	Clarifications     []ClarificationAnswer
	TemporalCondition  *string
	Facets             []ContractFacet
}
```

- [ ] **Step 1: Write failing state-machine tests** for captured/analyzing/needs_user/ready/confirmed/analysis_failed/cancelled, one material question, immutable proposal revisions, stale expected revision, idempotent confirmation, and conversation references without copied turns.
- [ ] **Step 2: Implement shared Intake contracts behind #32's sole ownership** and preserve trigger-backed CDC plus daemon error envelopes.
- [ ] **Step 3: Add analyzer/context tests** proving user edits outrank Project policy, workspace facts outrank attributed summaries, unrelated/personal requests do not force an Outcome, malformed output preserves the last valid proposal, and no raw all-history transcript packet is compiled.
- [ ] **Step 4: Implement Project New Outcome modal and adaptive proposal UI** with direct edits, re-analysis confirmation, manual fallback, Project scope, and truthful daemon-offline behavior.
- [ ] **Step 5: Run backend, API drift, frontend, accessibility, and Playwright tests** including mouse/keyboard entry, restart, idempotent read-back, and compact/wide layout.
- [ ] **Step 6: Commit** with `git commit -m "feat: add adaptive shared outcome intake"`.

### Task 4: Board, Outcome Mission Control, and Session Inspector composition

**Files:**
- Modify: `frontend/src/renderer/components/SessionsBoard.tsx`
- Modify: `frontend/src/renderer/components/SessionsBoardAdapters.tsx`
- Modify: `frontend/src/renderer/components/Sidebar.tsx`
- Modify: `frontend/src/renderer/components/outcome/OutcomeLifecycleShell.tsx`
- Modify: `frontend/src/renderer/components/outcome/OutcomeDecideAuthorizeSurface.tsx`
- Modify: `frontend/src/renderer/components/outcome/OutcomeRunSurface.tsx`
- Create: `frontend/src/renderer/components/outcome/OutcomeMissionGraph.tsx`
- Create: `frontend/src/renderer/components/outcome/OutcomeMissionControl.tsx`
- Modify/reuse: `packages/product-ui/src/SessionInspectorView.tsx`
- Modify: Work routes and generated route tree through source regeneration
- Add: focused component/integration/e2e tests

**Interfaces:**
- Consumes: current Outcome/Contract/Plan, attention projection, WorkUnit/Attempt/AgentSessionRef facts from #31, Evidence/Verification/Acceptance from #35, Project conversation context, and generated client.
- Produces: Board/List Outcome projection, adaptive Mission Control, WorkUnit/Attempt selection, and individual Session Inspector without storing display status.

- [ ] **Step 1: Write failing projection tests** proving one card per Outcome, active session aggregation without session-as-Outcome wording, Board/List equivalence, exact next action, and operational Sessions cross-Outcome projection.
- [ ] **Step 2: Implement Board/List adapters** using daemon facts and derived presentation only.
- [ ] **Step 3: Write failing Mission Control tests** for adaptive intake/plan/run/proof primary surfaces, graph nodes by WorkUnit/Attempt/session, revision history, Needs You routing, and selected Outcome context chip.
- [ ] **Step 4: Implement Mission Control and graph** with accessible keyboard navigation, list fallback, reduced motion, wide/narrow layouts, and truthful unavailable states.
- [ ] **Step 5: Write failing Session Inspector tests** for direct tactical instruction, material-change escalation, pause/resume/cancel/takeover, terminal/browser availability, worktree/artifact links, and exact back navigation.
- [ ] **Step 6: Integrate the existing inspector/runtime surfaces** without duplicating provider chat or canonical state.
- [ ] **Step 7: Run visual/e2e acceptance** for Project Board -> Outcome -> WorkUnit -> Attempt -> session -> return, plus restart and focus restoration.
- [ ] **Step 8: Commit** with `git commit -m "feat: add outcome Mission Control workspace"`.

### Task 5: One adaptive multi-WorkUnit Mission vertical and evaluation

> **Superseded 2026-08-29 by [ADR 0007](../../adr/0007-composed-outcomes.md).** `PlanRevision` stays at exactly one direct Work Unit; the topology moves up into contributing Outcomes. Its replacement is the [Composed Outcomes program](2026-08-29-composed-outcomes-program.md). The task is kept for provenance; do not plan from it.

**Files:**
- Modify: Outcome Plan/WorkUnit domain and service under current owners
- Modify: Attempt execution/admission service under #31 ownership
- Modify: role resolver from Task 1
- Modify: Mission Control graph from Task 4
- Add: deterministic multi-WorkUnit fake adapters and full boundary tests
- Update: `docs/product/kennel-v0-first-outcome-slice.md` only if the evaluated phase decision explicitly amends its locked proof

**Interfaces:**
- Consumes: stable Contract, role-resolved provider inventory, collection-shaped PlanRevision, isolated-worktree execution, durable handoffs, proof/acceptance, and promoted Project skills/policies only after their gates pass.
- Produces: one approved dependency graph that can execute on a single harness through separate sessions or on several admitted harnesses without changing Outcome truth.

- [ ] **Step 1: Write failing planner-policy tests** for direct, sequential, parallel isolated, implementer/verifier, and single-harness fallback proposals; reject cycles, overlapping writers, missing role capability, hidden cost, silent fallback, and unverifiable completion.
- [ ] **Step 2: Implement the smallest deterministic validator** around model-proposed WorkUnits, dependencies, roles, Evidence, Verification, budgets, and recovery.
- [ ] **Step 3: Write failing execution tests** for separate RunBriefs, dependency handoff provenance, isolated worktrees, coordinator replacement, worker failure, integration Attempt, fresh verifier, and no provider/session acceptance.
- [ ] **Step 4: Execute one end-to-end Mission** first with one harness/multiple sessions and then, when available and authorized, with two admitted providers.
- [ ] **Step 5: Evaluate supervision cost, transcript reconstruction, attention precision, Evidence coverage, false readiness, recovery, authority safety, and re-entry against #38's gate.**
- [ ] **Step 6: Record a continue/revise/stop decision**; only a continue decision may unlock broader learned orchestration policy under #33.
- [ ] **Step 7: Commit** with `git commit -m "feat: execute adaptive multi-work-unit missions"`.

## Final verification

- [ ] Run `npm run sqlc` when any query/schema source changed and confirm no hand-edited generated SQL files.
- [ ] Run `npm run api` when any API source changed and commit OpenAPI plus generated TypeScript together.
- [ ] Run `npm run lint`, `npm run frontend:typecheck`, `cd frontend && npm run build`, `cd backend && go test ./...`, and the focused race tests for new concurrency/recovery paths.
- [ ] Run `npm run test:foundation` and the relevant Electron Playwright suite.
- [ ] Launch the real Electron build and capture Project setup, Board/List, New Outcome, adaptive Contract, Mission authorization, multi-session Mission Control, Session Inspector, continuation receipt, proof, and Reopen/Acceptance.
- [ ] Verify no local run state, credentials, generated preview truth, temporary worktrees, or build output is committed.

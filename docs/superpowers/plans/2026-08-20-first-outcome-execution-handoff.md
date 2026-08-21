# Kennel First Outcome Execution Handoff

> **2026-08-21 sequencing amendment:** [ADR 0004](../../adr/0004-parallel-home-personal-agent-and-required-capture.md) permits a separately owned Home/Personal Agent lane to execute in parallel. This handoff continues to govern only the Work Local Focus Ledger milestone. Its Home exclusions are file/API ownership boundaries for this lane, not a global prohibition on parallel Home work. The Home lane uses the [Home/Personal Agent design](../specs/2026-08-21-home-personal-agent-memory-design.md).

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Starting from the accepted prerequisite baseline on `main`, deliver one complete Local Focus Ledger Outcome through Enter → Understand → Decide & Authorize → Act & Observe → Prove & Close.

**Architecture:** Home, Work, and Settings remain destinations; five adaptive lifecycle surfaces are the common product spine. The first milestone is Work-first and local-only. Each stage lands as one issue-sized end-to-end PR across domain/storage/CDC/API/UI/recovery/evaluation, and the milestone is not complete until all five stages and the evaluation protocol pass.

**Tech Stack:** Go daemon, SQLite/goose/sqlc, trigger-based `change_log` CDC, Chi/OpenAPI, Electron/React 19/TanStack Router, generated TypeScript client, Codex provider adapter, Playwright/Vitest/Go tests

**Spec:** `docs/product/kennel-v0-first-outcome-slice.md`

## Decision labels for this handoff

- **Locked:** the five-stage product contract, Work-first/non-blocking-Home entry, first Focus Ledger fixture, provider-neutral core with v0 Codex admission, separate Acceptance authority, local effect ceiling, recovery invariants, and evaluation/falsification rules.
- **Observed:** the dated PR/SHA facts below and the current donor/code paths inspected at commit `5a38593f`; refresh them before execution because repository and PR state can change.
- **Inference:** five stage-aligned end-to-end PRs are the smallest review boundaries that preserve user-visible truth while avoiding horizontal schema/UI ownership.
- **Proposed implementation detail:** filenames, Go interfaces, route/component names, migration numbers, and exact PR cuts below. The implementing agent must reconcile them with current `origin/main` and may narrow them without changing the locked contract.
- **Unknown until execution evidence:** implementation effort, exact issue-sized Work file ownership after reconciling current `main`, and whether the first-slice trials justify integrating or revising shared Work/Home contracts. Home may begin separately under ADR 0004.

## Global constraints

- This plan does not authorize implementation, branch mutation outside a new worktree, push, merge, deploy, publish, or release.
- Start every implementation branch from current `origin/main`, never from this docs branch or historical PR branches.
- PR #11 is closed unmerged and remains a donor record only. Never cherry-pick or merge it wholesale.
- One issue, one branch, one stage-aligned review boundary per PR.
- The renderer uses the generated daemon API and never writes SQLite or provider state directly.
- Durable mutations emit through SQLite triggers into `change_log`; service methods do not emit parallel CDC.
- Migrations are additive and committed with queries, regenerated sqlc, OpenAPI, and frontend types when applicable.
- All application state remains under `~/.kennel`; the daemon remains loopback-only on `127.0.0.1`.
- v0 new Work uses Codex only through provider-neutral contracts; historical provider identities remain readable.
- Provider/session completion, process exit, checks, and Verification cannot create Acceptance.
- The first milestone excludes Home/OpenLoop persistence, Gmail, Desktop Context, hosted IDs, durable Memory, provider expansion, and remote effects.

## Current observed prerequisite facts

Read-only refresh on 2026-08-21:

| Item | Observed state |
| --- | --- |
| PR #1 | Merged as `aa8241c2157d974e46a10e35892c49816ea4e1d1`; establishes F0-F6. |
| PR #12 | Merged as `348851867e50b6341c251f51083c72dedaeff85b`; removes the user-facing legacy import flow. |
| PR #13 | Merged as `ad79f3c5fd9f578bc1717a437f295551aee29fd1`; establishes provider-neutral Codex admission and historical compatibility. |
| PR #14 | Merged as `aef1ede793c8fa9736cb7e05ab836041e7021e86`; narrows public CLI discovery while preserving direct invocation. |
| PR #11 | Closed unmerged on 2026-08-21 as a superseded donor; its Outcome store/schema and locale deletion remain rejected. |
| `origin/main` at this handoff refresh | `aef1ede793c8fa9736cb7e05ab836041e7021e86`. |

Refresh `origin/main` at execution time and use its current SHA as the base. The historical merge SHAs above are provenance, not branch targets.

---

### Task 1: Resolve the foundation gate and PR #11 disposition — completed

This task is retained as historical verification evidence. Do not repeat its old branch/SHA procedure.

**Files:**
- Verify: `docs/foundation-acceptance-2026-08-18.md`
- Verify: `scripts/test-foundation.sh`
- Inspect donor: PR #11 changed paths
- Update after maintainer decision: PR descriptions or local execution ledger only

**Interfaces:**
- Consumes: PR #1 and PR #11 live metadata
- Produces: one named accepted foundation base plus an explicit per-file PR #11 disposition

- [ ] **Step 1: Refresh read-only PR facts**

```bash
gh pr view 1 --repo Pin4sf/Waldo-Kennel --json state,isDraft,headRefName,headRefOid,baseRefName,mergeable,mergeStateStatus,statusCheckRollup
gh pr view 11 --repo Pin4sf/Waldo-Kennel --json state,isDraft,headRefName,headRefOid,baseRefName,mergeable,mergeStateStatus,statusCheckRollup
git ls-remote origin refs/heads/main refs/heads/codex/foundation-gate-f0-f6 refs/heads/codex/kennel-clean-build
```

Expected: exact live facts are recorded; no check is invented when `statusCheckRollup` is empty.

- [ ] **Step 2: Verify PR #1 in a clean isolated worktree**

```bash
foundation_parent=$(mktemp -d /tmp/kennel-foundation.XXXXXX)
foundation_worktree="$foundation_parent/worktree"
git worktree add --detach "$foundation_worktree" ef0baffd7850702480cba84ae97e2790367eab12
cd "$foundation_worktree"
npm run bootstrap
npm run test:foundation
npm run audit:production
npm run lint
cd backend && go test -race ./...
```

Expected: every command exits 0. A full-suite failure keeps PR #1 blocked even when an isolated retry passes.

- [ ] **Step 3: Produce the PR #11 donor ledger**

```bash
git diff --name-status b1bcd37fd0911a86da5bb21c07ac3a0b9adecb3b...dbebb0956d2e039fefc77bb963c6ef9bc0c7e28b
git diff --stat b1bcd37fd0911a86da5bb21c07ac3a0b9adecb3b...dbebb0956d2e039fefc77bb963c6ef9bc0c7e28b
```

Classify every path as `reapply`, `rewrite`, `defer`, or `drop`. Drop PR #11's Outcome migration/store, locale deletion, and any provider/CLI narrowing not covered by a separately accepted contract.

- [ ] **Step 4: Stop for maintainer authority**

Do not merge or close either PR. Record the verified foundation SHA and donor ledger, then request the explicit merge/close/branch authority needed for the next task.

### Task 2: Land bounded cleanup prerequisites — completed by PRs #12-#14

This task is retained as the accepted donor disposition. The next executable task is Task 3.

**Files:**
- Delete/rewrite: the legacy user-import paths enumerated in [the convergence plan](2026-08-20-pr-convergence-and-architecture-gate.md#task-3-remove-only-the-user-facing-legacy-import-flow)
- Modify: `backend/internal/domain/harness.go`
- Modify: `backend/internal/adapters/agent/registry/registry.go`
- Modify: `backend/internal/adapters/reviewer/registry.go`
- Modify: `backend/internal/storage/sqlite/store/agent_switching_store.go`
- Modify: `frontend/src/renderer/lib/agent-select-options.ts`
- Modify: `frontend/src/renderer/lib/reviewer-harnesses.ts`

**Interfaces:**
- Consumes: accepted foundation base and PR #11 donor ledger
- Produces: two independent PRs: legacy-import removal, then provider-neutral admission/historical recovery

- [ ] **Step 1: Execute legacy removal as its own TDD PR**

Follow Task 3 of the convergence plan exactly. Regenerate API artifacts and prove active product/help surfaces expose neither `ao import` nor `kennel import`. Preserve dev-import and source-compatibility seams.

- [ ] **Step 2: Execute provider admission as its own TDD PR**

Add failing tests for `IsRecognizedPersisted` versus `IsSelectable`, required/optional capability profiles, per-Attempt readmission, and historical recovery. Implement the provider-neutral boundary without deleting persisted identities.

- [ ] **Step 3: Run each PR's complete gate**

```bash
npm run test:foundation
npm run audit:production
npm run lint
cd backend && go test -race ./...
```

Expected: each PR is independently green and contains no Outcome schema or UI lifecycle implementation.

### Task 3: PR A — Enter surface and Work-first shell

**Files:**
- Create: `frontend/src/renderer/components/outcome/OutcomeLifecycleShell.tsx`
- Create: `frontend/src/renderer/components/outcome/OutcomeLifecycleShell.test.tsx`
- Create: `frontend/src/renderer/routes/_shell.work.tsx`
- Modify: `frontend/src/renderer/routes/_shell.tsx`
- Modify: `frontend/src/renderer/routeTree.gen.ts` through the existing router generator
- Modify: `frontend/src/renderer/components/CreateProjectFlow.tsx`
- Test: `frontend/src/renderer/__tests__/integration/work-first-entry.test.tsx`

**Interfaces:**
- Consumes: existing Project list/add/readiness APIs and provider admission projection
- Produces: `OutcomeLifecycleShell({stage, projectId, outcomeId?, children})` and a Work-first Enter surface; no Outcome mutation yet

- [ ] **Step 1: Write failing Work-first entry tests**

Assert that first run offers `Start with Work` as the v0 dogfood recommendation and `Set up Home` as an equal non-blocking alternative; selecting Work opens Project selection/readiness; no Home record is created; invalid folder, daemon offline, and Codex action-required states are distinct.

- [ ] **Step 2: Verify failure**

```bash
npm --prefix frontend test -- --run work-first-entry OutcomeLifecycleShell
```

Expected: FAIL because the five-stage shell and Work route do not exist.

- [ ] **Step 3: Implement the minimal shell**

Use exactly five stage keys:

```ts
export type OutcomeStage = "enter" | "understand" | "decide_authorize" | "act_observe" | "prove_close";
```

Home, Work, and Settings remain destinations outside this stage enum. Settings opens control; it is not a sixth lifecycle stage.

- [ ] **Step 4: Verify and commit**

```bash
npm --prefix frontend test -- --run work-first-entry OutcomeLifecycleShell
npm --prefix frontend run typecheck
git add frontend
git commit -m "feat: add Work-first Outcome lifecycle shell"
```

### Task 4: PR B — Understand surface and immutable contract

**Files:**
- Replace: `backend/internal/domain/outcome.go`
- Delete: `backend/internal/service/outcome/planning.go`
- Delete: `backend/internal/service/outcome/planning_test.go`
- Create: `backend/internal/domain/responsibility.go`
- Create: `backend/internal/domain/outcome_contract_test.go`
- Create: `backend/internal/service/outcome/service.go`
- Create: `backend/internal/service/outcome/contract.go`
- Create: `backend/internal/service/outcome/contract_test.go`
- Create: `backend/internal/ports/outcome_store.go`
- Create: `backend/internal/storage/sqlite/migrations/0099_responsibility_outcome_contract.sql`
- Create: `backend/internal/storage/sqlite/queries/outcomes.sql`
- Create: `backend/internal/storage/sqlite/store/outcome_store.go`
- Create: `backend/internal/storage/sqlite/store/outcome_store_test.go`
- Create: `backend/internal/httpd/controllers/outcomes.go`
- Create: `backend/internal/httpd/controllers/outcomes_test.go`
- Modify: `backend/internal/httpd/controllers/dto.go`
- Modify: `backend/internal/httpd/api.go`
- Modify: `backend/internal/httpd/apispec/specgen/build.go`
- Regenerate: `backend/internal/httpd/apispec/openapi.yaml`
- Regenerate: `frontend/src/api/schema.ts`
- Create: `frontend/src/renderer/hooks/useOutcome.ts`
- Create: `frontend/src/renderer/components/outcome/OutcomeUnderstandSurface.tsx`
- Create: `frontend/src/renderer/components/outcome/OutcomeUnderstandSurface.test.tsx`
- Modify: `frontend/src/renderer/routes/_shell.work.tsx`

**Interfaces:**
- Consumes: `ProjectID`, Work-first shell
- Produces: `CreateOutcome`, `ReviseOutcomeContract`, `GetOutcome`; immutable `ContractRevision`; current stage derived from durable facts

- [ ] **Step 1: Write failing domain/store/controller tests**

Cover required Goal/Success/Review, local-midnight clarification, immutable revision numbering, idempotency key replay, stale expected-revision conflict, restart persistence, and trigger-emitted CDC. Assert `completed` is not a valid Outcome state.

- [ ] **Step 2: Verify failure**

```bash
cd backend
go test ./internal/domain ./internal/service/outcome ./internal/storage/sqlite/store ./internal/httpd/controllers -run 'Outcome|ContractRevision' -count=1
```

- [ ] **Step 3: Implement the minimum model**

The public service contract is:

```go
type Manager interface {
    Create(ctx context.Context, in CreateInput) (OutcomeView, error)
    ReviseContract(ctx context.Context, id domain.OutcomeID, in ReviseContractInput) (OutcomeView, error)
    Get(ctx context.Context, id domain.OutcomeID) (OutcomeView, error)
}
```

Persist facts and revision IDs; derive `enter`/`understand` presentation. Do not persist screen labels.

- [ ] **Step 4: Implement the Understand UI and retire marker authority**

Use generated API types for Goal, Success, Review, constraints, non-goals, and one material clarification. Remove `OutcomeIntakePanel`, `OutcomeOrchestrationGraph`, and marker parsing only when every call site in `SessionsBoard.tsx`/`TaskComposer.tsx` is replaced; do not leave two Outcome authorities.

- [ ] **Step 5: Regenerate, verify, and commit**

```bash
npm run sqlc
npm run api
cd backend && go test ./internal/domain ./internal/service/outcome ./internal/storage/sqlite/store ./internal/httpd/... -count=1
cd ../frontend && npm test -- --run OutcomeUnderstand && npm run typecheck
git add backend frontend
git commit -m "feat: add immutable Outcome contract"
```

### Task 5: PR C — Decide & Authorize surface

**Files:**
- Create: `backend/internal/domain/outcome_plan.go`
- Create: `backend/internal/domain/outcome_plan_test.go`
- Create: `backend/internal/service/outcome/plan.go`
- Create: `backend/internal/service/outcome/plan_test.go`
- Modify: `backend/internal/ports/outcome_store.go`
- Create: `backend/internal/storage/sqlite/migrations/0100_outcome_plan_authority.sql`
- Modify: `backend/internal/storage/sqlite/queries/outcomes.sql`
- Modify: `backend/internal/storage/sqlite/store/outcome_store.go`
- Create: `backend/internal/httpd/controllers/outcome_plans_test.go`
- Modify: `backend/internal/httpd/controllers/outcomes.go`
- Modify/regenerate: API spec and frontend types
- Create: `frontend/src/renderer/components/outcome/OutcomeDecideAuthorizeSurface.tsx`
- Create: `frontend/src/renderer/components/outcome/OutcomeDecideAuthorizeSurface.test.tsx`

**Interfaces:**
- Consumes: immutable ContractRevision and provider capability profile
- Produces: immutable PlanRevision, one WorkUnit, scoped CapabilityGrant, provider-neutral RunBrief core/compiled digests

- [ ] **Step 1: Write failing policy tests**

Prove one direct Work Unit is chosen for Focus Ledger, optional capabilities only remove dependent routes, required capability absence fails closed, authority is the intersection of all layers, a lower layer cannot widen, and any material contract/authority/worktree/provider change requires a new RunBrief/Attempt.

- [ ] **Step 2: Implement deterministic validation around model proposals**

Expose service methods:

```go
ProposePlan(ctx context.Context, outcomeID domain.OutcomeID, expectedContractRevision int64) (PlanView, error)
ApprovePlan(ctx context.Context, outcomeID domain.OutcomeID, in ApprovePlanInput) (AuthorizedPlanView, error)
```

The model may propose; deterministic code validates dependency shape, capability/effect envelope, Project/worktree placement, Evidence/Verification requirements, budget, stop, and recovery.

- [ ] **Step 3: Verify and commit**

```bash
npm run sqlc
npm run api
cd backend && go test ./internal/domain ./internal/service/outcome ./internal/storage/sqlite/store ./internal/httpd/... -count=1
cd ../frontend && npm test -- --run OutcomeDecideAuthorize && npm run typecheck
git add backend frontend
git commit -m "feat: add Outcome plan and authority gate"
```

### Task 6: PR D — Act & Observe surface

**Files:**
- Create: `backend/internal/domain/outcome_attempt.go`
- Create: `backend/internal/domain/outcome_attempt_test.go`
- Create: `backend/internal/ports/outcome_executor.go`
- Create: `backend/internal/service/outcome/attempt.go`
- Create: `backend/internal/service/outcome/attempt_test.go`
- Create: `backend/internal/storage/sqlite/migrations/0101_outcome_attempt_recovery.sql`
- Modify: `backend/internal/storage/sqlite/queries/outcomes.sql`
- Modify: `backend/internal/storage/sqlite/store/outcome_store.go`
- Modify: `backend/internal/httpd/controllers/outcomes.go`
- Create: `backend/internal/httpd/controllers/outcome_attempts_test.go`
- Modify: `backend/internal/daemon/lifecycle_wiring.go`
- Modify/regenerate: API spec and frontend types
- Create: `frontend/src/renderer/components/outcome/OutcomeActObserveSurface.tsx`
- Create: `frontend/src/renderer/components/outcome/OutcomeActObserveSurface.test.tsx`

**Interfaces:**
- Consumes: approved WorkUnit/CapabilityGrant/RunBrief, provider admission, existing session/worktree lifecycle
- Produces: Attempt/AgentSessionRef, ordered observations, attention reasons, reconciliation and replacement-attempt receipt

- [ ] **Step 1: Write failing execution/recovery tests**

Cover per-start admission, one fenced writer, silent lease renewal, `unconfirmed` heartbeat state, stale-event rejection, retained dirty work, no silent provider fallback, and contain/reconcile/narrow replacement Attempt. Assert ordinary reasoning/exploration is not blocked by a missed heartbeat while canonical mutations/effects are.

- [ ] **Step 2: Implement the executor adapter**

Keep Outcome service dependent on `ports.OutcomeExecutor`; adapt the existing Codex session/worktree lifecycle behind that port. Never call adapters or SQLite from controllers or renderer.

- [ ] **Step 3: Implement adaptive attention modes**

Within Act & Observe, derive `Needs You`, `Action Required`, and `Waiting` from durable decision/dependency/admission/recovery facts. Do not persist those labels.

- [ ] **Step 4: Verify and commit**

```bash
npm run sqlc
npm run api
cd backend && go test ./internal/domain ./internal/service/outcome ./internal/lifecycle ./internal/integration ./internal/httpd/... -count=1
cd ../frontend && npm test -- --run OutcomeActObserve && npm run typecheck
git add backend frontend
git commit -m "feat: execute and recover Outcome attempts"
```

### Task 7: PR E — Prove & Close surface

**Files:**
- Create: `backend/internal/domain/outcome_proof.go`
- Create: `backend/internal/domain/outcome_proof_test.go`
- Create: `backend/internal/service/outcome/proof.go`
- Create: `backend/internal/service/outcome/proof_test.go`
- Create: `backend/internal/storage/sqlite/migrations/0102_outcome_evidence_acceptance.sql`
- Modify: `backend/internal/storage/sqlite/queries/outcomes.sql`
- Modify: `backend/internal/storage/sqlite/store/outcome_store.go`
- Modify: `backend/internal/httpd/controllers/outcomes.go`
- Create: `backend/internal/httpd/controllers/outcome_proof_test.go`
- Modify/regenerate: API spec and frontend types
- Create: `frontend/src/renderer/components/outcome/OutcomeProveCloseSurface.tsx`
- Create: `frontend/src/renderer/components/outcome/OutcomeProveCloseSurface.test.tsx`
- Create: `test/e2e-pod/outcome-lifecycle.spec.ts`

**Interfaces:**
- Consumes: current ContractRevision, subject digest, Attempt outputs, deterministic verifier and owner input
- Produces: EvidenceItem, VerificationRun, immutable AcceptanceDecision, resource disposition, SuccessorLink/Re-entry

- [ ] **Step 1: Write failing proof/authority tests**

Prove Evidence is criterion/subject/revision bound; stale Evidence cannot make readiness; producer self-check is labeled non-independent; deterministic verification runs outside the producer session; only owner input creates Acceptance; reopen preserves prior decision; dirty worktree is never force-deleted.

- [ ] **Step 2: Implement proof and close services**

Expose:

```go
RecordEvidence(ctx context.Context, in RecordEvidenceInput) (EvidenceView, error)
Verify(ctx context.Context, in VerifyInput) (VerificationView, error)
DecideAcceptance(ctx context.Context, in AcceptanceInput) (OutcomeView, error)
Close(ctx context.Context, in CloseInput) (CloseReceipt, error)
```

- [ ] **Step 3: Add the five-stage E2E**

The Playwright test enters Work, registers the fixture Project, defines Focus Ledger, approves the direct plan/local authority, observes the Attempt, reviews three criteria, records owner Acceptance, retains a dirty worktree, and re-enters from the immutable close receipt.

- [ ] **Step 4: Verify and commit**

```bash
npm run sqlc
npm run api
cd backend && go test ./internal/domain ./internal/service/outcome ./internal/storage/sqlite/store ./internal/httpd/... -count=1
cd ../frontend && npm test -- --run OutcomeProveClose && npm run typecheck
cd .. && npx playwright test -c test/e2e-pod/playwright.electron.config.ts test/e2e-pod/outcome-lifecycle.spec.ts
git add backend frontend test
git commit -m "feat: prove and close accepted Outcomes"
```

### Task 8: Execute the first-slice evaluation protocol

**Files:**
- Create: `docs/evaluation/kennel-v0-first-outcome-results.md`
- Create: `test/e2e-pod/fixtures/focus-ledger/`
- Modify: `docs/STATUS.md` only with observed results

**Interfaces:**
- Consumes: all five accepted stage PRs
- Produces: contract-conformance evidence, failure-injection receipts, five paired dogfood results, and an explicit continue/revise/stop decision

- [ ] **Step 1: Run the normal and injected lifecycle matrix**

Use the exact eight injections in the first-slice spec. Record command/build SHA, expected state, observed state, causal receipt, and whether transcript reconstruction was needed.

- [ ] **Step 2: Run five matched direct-Codex/Kennel pairs**

Record supervision minutes, transcript opens, interventions, re-entry time, false ready/reopen, and effects. Do not claim the 20-Outcome launch threshold from five trials.

- [ ] **Step 3: Run the complete repository gate**

```bash
npm run test:foundation
npm run audit:production
npm run lint
npm run frontend:typecheck
cd backend && go test -race ./...
```

- [ ] **Step 4: Record the milestone decision**

Choose exactly one: `continue shared integration`, `revise the Outcome core`, or `stop/falsified`. Home may be active in parallel; this decision governs whether the Work contract is stable enough to integrate. A failure remains a recorded product result and cannot be relabeled green.

### Task 9: Prepare Work-side integration with the parallel Home lane

**Files:**
- Read: `docs/product/kennel-v1-product-architecture.md`
- Read: `docs/product/kennel-v1-team-review-packet.md`
- Read: `docs/superpowers/specs/2026-08-21-home-personal-agent-memory-design.md`
- Create after both lane contracts are reviewable: a focused Home-to-Work integration plan

**Interfaces:**
- Consumes: stable Work Outcome/ContractRevision references plus the separately owned Home/OpenLoop contracts
- Produces: coordinated `ResponsibilityLink`, shared-intake, and provenance behavior without moving either lifecycle into this Work lane

- [ ] **Step 1: Preserve separate lineages**

An Open Loop remains in exactly one ResponsibilitySpace with its own owner/recheck/closure. A Work Outcome retains its own contract/Evidence/Verification/Acceptance. Creating or ending a link changes neither lifecycle.

- [ ] **Step 2: Preserve candidate boundaries**

Suggested Next Actions remain projections. Direct candidate-to-Outcome creation records minimized origin provenance; it does not persist an inferred task as canonical responsibility.

- [ ] **Step 3: Keep later systems out**

Do not include Gmail, governed capture, durable Memory, Health, Relationship, phone/wearable, hosted attachment, or broad providers in this Work milestone. The Home/Personal Agent lane owns its separately approved capture and candidate-memory scope.

## New-session start checklist

1. Read `AGENTS.md`, the canonical architecture, first-slice spec, this handoff, and `docs/STATUS.md` completely.
2. Fetch and record the current `origin/main` SHA; confirm PRs #1 and #12-#14 remain ancestors and PR #11 remains closed unmerged.
3. Obtain explicit user authority for exactly one issue/PR task.
4. Create a fresh `codex/` worktree from current `origin/main`.
5. State claimed files, dependencies, acceptance, falsifier, verification commands, rollback, and worktree disposition before editing.
6. Run the task's failing test first, implement only that boundary, run focused then full gates, and stop for review.

No step in this document authorizes a merge, push, deploy, publish, release, force-push, history rewrite, destructive cleanup, or product implementation in the current docs session.

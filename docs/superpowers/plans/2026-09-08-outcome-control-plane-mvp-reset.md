# Outcome Control Plane MVP Reset — Implementation Plan

> **Execution instruction:** In the implementation session, use the repository's Superpowers workflow and execute this plan task-by-task. Do not re-open the product architecture unless evidence forces a change. Preserve falsifiable verification evidence for every completed slice.

**Goal:** Deliver a working Waldo Kennel MVP in which a user can register a Project, state an Outcome, review/confirm an intelligently derived Contract, review/approve an intelligently derived and preference-aware Plan, supervise exact-bound execution in Mission Control, inspect sessions only as Attempt drill-down, review evidence/verification, and explicitly accept or continue the Outcome.

**Architecture:** Keep the Go daemon/SQLite as deterministic control plane. Add a bounded non-authoritative `IntelligenceRun` seam for Contract/Plan proposals. Preserve the existing Outcome lineage and partial WT3 execution-preference/routing work. Remove inherited Agent Orchestrator authority/navigation bypasses. The MVP scheduler may execute WorkUnits serially; ADR 0008/0009 parallel DAG/WorkspaceLease work follows after the vertical product loop is proven.

**Primary architecture references:**

- `docs/adr/0010-outcome-first-control-plane-and-session-subordination.md`
- `docs/adr/0011-go-control-plane-and-non-authoritative-intelligence.md`
- `docs/product/2026-09-08-outcome-control-plane-mvp-reset.md`
- `docs/product/kennel-v1-product-architecture.md`
- `docs/adr/0008-responsibility-composition-and-workunit-execution-dag.md`
- `docs/adr/0009-workunit-scheduling-workspace-leases-and-effect-fencing.md`
- `docs/superpowers/plans/2026-09-07-wt3-preference-aware-routing.md`

**Tech stack:** Go daemon, SQLite/sqlc, generated OpenAPI, Electron + React + TypeScript + TanStack Router/Query, existing provider runtime/session/worktree adapters.

---

## 0. New-session handoff and non-negotiables

Before editing code in the next session:

1. Checkout/fetch `Pin4sf/Waldo-Kennel`.
2. Work from `feat/wt3-routing-outcome-first` unless the user explicitly asks to split the work.
3. Confirm the branch contains the partial WT3 files listed below.
4. Confirm draft PR #99 remains unmerged.
5. Read ADR 0010, ADR 0011, the MVP reset doc, this plan, and the WT3 plan.
6. Do not restore any hidden Codex default.
7. Never modify merged migrations 0112 or 0113.
8. Treat existing migration 0114 as feature-branch work: audit it before relying on it. If a correction is required before merge, make it deliberately and regenerate/tests; otherwise add 0115+ for new persistence.
9. Do not claim tests passed unless they actually ran.
10. Do not make a Node/Python daemon rewrite part of the MVP.

### Partial WT3 work that must be preserved and completed

Expected on the branch:

- `backend/internal/domain/execution_preference.go`
- `backend/internal/domain/execution_preference_test.go`
- `backend/internal/domain/execution_binding.go`
- `backend/internal/domain/routing.go`
- `backend/internal/domain/outcome.go` execution-preference field
- `backend/internal/domain/outcome_plan.go` provider/model binding + routing decision
- `backend/internal/storage/sqlite/migrations/0114_execution_routing_bindings.sql`
- `backend/internal/storage/sqlite/store/outcome_execution_preference_store.go`
- `backend/internal/storage/sqlite/store/outcome_provider_store.go`

Known review items before integration:

- verify the `AgentHarness` persisted-recognition method used by `execution_binding.go` actually exists;
- fix `GetPlanRoutingDecision` so non-`sql.ErrNoRows` query errors are not swallowed by checking `raw.Valid` first;
- make explicit model bindings validate against that candidate/provider's model support rather than a global cross-provider model requirement;
- keep routing provider-neutral and deterministic;
- preserve `historical_unbound` as readable but non-executable;
- ensure model selection changes alter the RunBrief/authorization digest.

### Baseline commands

Run and record the baseline before changing behavior:

```bash
npm run bootstrap
npm run frontend:typecheck
cd backend && go test ./internal/domain ./internal/service/intake ./internal/service/outcome ./internal/storage/sqlite/store
```

If baseline failures exist, record them separately from regressions. Use systematic debugging before changing unrelated code.

---

## Task 1 — Lock the domain boundary: IntelligenceRun is not Attempt

**Purpose:** Give pre-authorization model work a durable identity without making it execution authority.

**Files:**

- Create `backend/internal/domain/intelligence_run.go`
- Create `backend/internal/domain/intelligence_run_test.go`
- Inspect/update `backend/internal/domain/intake_analysis_request.go`
- Inspect `backend/internal/domain/outcome.go`
- Inspect `backend/internal/domain/outcome_plan.go`

### Domain shape

Prefer one generic durable object rather than separate AnalysisRun/PlanningRun tables:

```go
type IntelligenceRunKind string
const (
    IntelligenceRunContractAnalysis IntelligenceRunKind = "contract_analysis"
    IntelligenceRunPlanDraft        IntelligenceRunKind = "plan_draft"
)

type IntelligenceRunStatus string
// requested, running, fulfilled, failed, cancelled, expired

type IntelligenceRun struct {
    ID                IntelligenceRunID
    Kind              IntelligenceRunKind
    ProjectID         ProjectID
    IntakeID          IntakeSessionID // optional by kind
    OutcomeID         OutcomeID       // optional by kind
    SourceRevision    int64
    Provider          AgentHarness    // optional when offline/manual
    ModelSelection    ...             // explicit/provider_default/unknown where appropriate
    Model             string
    InputDigest       string
    OutputDigest      string
    NativeSessionRef  string          // provenance only, never AgentSessionRef
    Status            IntelligenceRunStatus
    FailureCode       string
    FailureDetail     string
    CreatedAt         time.Time
    CompletedAt       *time.Time
}
```

Use existing repository value-object style rather than blindly copying the sketch. The invariant is more important than the exact fields.

### Required tests

Write RED tests proving:

1. contract-analysis run may reference Intake before Outcome exists;
2. plan-draft run must reference Outcome + exact ContractRevision;
3. offline/manual run can have no provider/model;
4. explicit model requires provider;
5. terminal statuses cannot transition back to running;
6. `IntelligenceRun` cannot be converted to or mistaken for `Attempt`/`AgentSessionRef` through any helper;
7. provider-native session reference is provenance only.

### Run

```bash
cd backend
go test ./internal/domain -run 'Test.*IntelligenceRun'
```

Commit after domain tests pass.

---

## Task 2 — Persist IntelligenceRun and preserve intake callback durability

**Purpose:** Replace anonymous/ordinary session-backed reasoning with canonical reasoning provenance while keeping the strong callback/revision/refusal machinery already present.

**Files:**

- Add `backend/internal/storage/sqlite/migrations/0115_intelligence_runs.sql`
- Add/update `backend/internal/storage/sqlite/queries/intelligence_runs.sql`
- Update `backend/sqlc.yaml` only when type overrides are required
- Create `backend/internal/ports/intelligence_run_store.go`
- Create `backend/internal/storage/sqlite/store/intelligence_run_store.go`
- Add `backend/internal/storage/sqlite/store/intelligence_run_store_test.go`
- Update `backend/internal/ports/intake_store.go` only where existing analysis-request linkage needs an IntelligenceRun ID
- Update `backend/internal/storage/sqlite/queries/intakes.sql` / `store/intake_store.go` only as required

### Persistence requirements

- additive migration;
- immutable/revision-safe provenance;
- no API secret stored;
- no transcript body required in the canonical run row;
- existing `intake_analysis_requests` callback/refusal/expiry semantics stay usable during migration;
- link the old request to `IntelligenceRun` rather than pretending the bounded provider process is an execution session;
- historical request rows without a run link remain readable.

### Required tests

1. create/read/update status of a run;
2. no plaintext secret/token persisted in run fields;
3. source revision persists exactly;
4. fulfilled output digest persists;
5. restart query returns non-terminal runs for reconciliation;
6. old intake-analysis rows remain readable;
7. migration up succeeds on a seeded pre-0115 database.

### Run

```bash
npm run sqlc
cd backend
go test ./internal/storage/sqlite/store -run 'Test.*Intelligence|Test.*IntakeAnalysis'
```

Commit migration + generated sqlc output together.

---

## Task 3 — Define a provider-neutral IntelligenceProvider port

**Purpose:** Make Contract/Plan intelligence replaceable without giving provider SDKs control-plane authority.

**Files:**

- Create `backend/internal/ports/intelligence_provider.go`
- Add tests/fakes in the relevant service test packages
- Refactor/bridge `backend/internal/ports/intake_analyzer.go`
- Inspect `backend/internal/daemon/intake_analyzer.go`
- Inspect `backend/internal/service/intake/analyzer.go`

### Contract

The port should accept bounded canonical snapshots and return structured proposals plus provenance. It must not expose daemon mutation or Attempt-launch APIs.

Suggested semantic operations:

```go
AnalyzeContract(ctx, ContractAnalysisInput) (IntelligenceTicket, error)
DraftPlan(ctx, PlanDraftInput) (IntelligenceTicket, error)
```

A ticket may complete inline or defer asynchronously, but asynchronous completion must point back to one durable `IntelligenceRun`.

### Keep

- deterministic offline Contract analyzer;
- immutable proposal validation;
- one-open-analysis protection;
- expiry/cancellation/refusal;
- structured callback admission.

### Remove from the semantic contract

- `KindWorker` as the meaning of Contract analysis;
- ordinary execution `SessionID` as the canonical analysis identity;
- any implication that opening a worktree/provider process creates responsibility.

### Tests

- fake intelligence provider cannot start Attempt through the interface;
- inline/offline proposal remains valid;
- deferred result is tied to exactly one IntelligenceRun;
- invalid model proposal is rejected by service/domain validation, not trusted because a model produced it.

### Run

```bash
cd backend
go test ./internal/service/intake ./internal/daemon -run 'Test.*Intake|Test.*Intelligence'
```

---

## Task 4 — Choose and implement the MVP intelligence adapter

**Purpose:** Make Contract and Plan drafting actually intelligent while preserving a real effect boundary.

This task has a deliberate decision gate. Do not ask the user for a key before completing the first two checks.

### Step 4A — Evaluate existing provider runtimes

Inspect the actual current provider/session adapters for a proven read-only mode suitable for pre-authorization reasoning.

Acceptance for a provider-backed intelligence adapter:

- can read required repository/context;
- cannot write workspace or execute arbitrary mutation under the configured mode, by enforcement rather than prompt text alone;
- can return a structured result reliably;
- can be cancelled/reaped;
- does not create an execution Attempt/AgentSessionRef;
- model/provider provenance is knowable.

If a provider satisfies those requirements, implement it behind `IntelligenceProvider`.

### Step 4B — Otherwise use a direct model API adapter

If the coding-agent runtime cannot enforce the read-only boundary, implement one direct model API adapter behind the same port.

Before implementing vendor-specific code, check current official API documentation in that session.

The adapter must:

- use structured/schema-constrained output where available;
- receive bounded repository/context excerpts assembled by Kennel;
- never receive API keys in prompts;
- set timeouts and cancellation;
- record provider/model provenance;
- return structured Contract/Plan proposal only;
- have no direct control-plane mutation access.

**This is the point at which the user may provide an API key.** Ask only for the chosen provider's key or instruct them to configure it via the secure development mechanism. Do not request that they paste a key into repository code or a persisted domain field.

### Step 4C — Preserve the floor

If intelligence is unavailable, Contract/Plan drafting must expose an explicit deterministic/manual fallback. Never silently switch to Codex or any other provider.

### Tests

Use a fake HTTP server/provider adapter; tests must not consume real paid inference.

- valid structured Contract result;
- material clarification result;
- invalid schema rejected;
- timeout/cancellation;
- missing key returns explicit unavailable state;
- no secret appears in logs/error/provenance;
- offline fallback remains reachable by explicit policy.

---

## Task 5 — Make intake truly Contract-first and execution-free

**Purpose:** Ensure the existing excellent intake UX can no longer leak into ordinary execution sessions.

**Primary files:**

- `backend/internal/service/intake/service.go`
- `backend/internal/daemon/intake_analyzer.go`
- `backend/internal/domain/intake_analysis_request.go`
- `backend/internal/httpd/controllers/intakes.go`
- `backend/internal/httpd/controllers/dto.go`
- `backend/internal/httpd/apispec/specgen/build.go`
- `frontend/src/renderer/components/outcome/AdaptiveIntakeSurface.tsx`
- `frontend/src/renderer/components/outcome/IntakeAnalysisWaiting.tsx`
- `frontend/src/renderer/components/outcome/IntakeContractReview.tsx`
- `frontend/src/renderer/hooks/useIntakeAnalysisRequest.ts`

### Required behavioral changes

- capture persists intent only;
- automatic analysis may create IntelligenceRun, not execution Attempt/session;
- waiting UI shows intelligence provenance, not “orchestrator session” ownership;
- clarification remains bounded/material;
- Contract review remains editable;
- confirmation is explicit owner confirmation of the proposal;
- confirmation creates/advances Outcome + immutable ContractRevision;
- no execution Attempt or execution AgentSessionRef exists after confirmation.

### Regression tests

Add service/controller tests that count Attempt/session writes/spawn calls and assert zero through:

```text
Capture → Analyze → Clarify(optional) → Confirm
```

Also test crash/retry/expiry behavior and offline floor.

### API generation

If DTO/routes change:

```bash
npm run api
npm run frontend:typecheck
```

Do not hand-edit generated OpenAPI/schema files.

---

## Task 6 — Add Plan intelligence and make Contract confirmation lead to planning

**Purpose:** Replace the deterministic “smallest direct WorkUnit” as the only user experience with a real pre-execution Plan proposal.

**Primary files:**

- `backend/internal/service/outcome/plan.go`
- `backend/internal/service/outcome/service.go`
- Create `backend/internal/service/outcome/plan_intelligence.go` if it keeps the service smaller
- `backend/internal/domain/outcome_plan.go`
- `backend/internal/httpd/controllers/outcomes.go`
- `backend/internal/httpd/controllers/dto.go`
- `backend/internal/httpd/apispec/specgen/build.go`
- `frontend/src/renderer/components/outcome/OutcomeDecideAuthorizeSurface.tsx`
- `frontend/src/renderer/hooks/useOutcome.ts`

### Plan proposal semantics

A Plan proposal is non-authoritative. It is derived from:

- exact current ContractRevision;
- Project Brief/config facts available today;
- capability/effect ceilings;
- execution preferences;
- repository/context evidence when available.

For the MVP, accept one or several **serializable** WorkUnits. Do not expose parallel edges the scheduler cannot execute truthfully.

Each proposed WorkUnit needs enough structure for:

- description/intent;
- expected output/evidence;
- dependency ordering;
- capability requirements;
- verification intent;
- routing requirements.

### UI

`OutcomeDecideAuthorizeSurface` should show a real Plan review state:

- “Waldo is planning” while IntelligenceRun is active;
- WorkUnit list/order;
- assumptions/blockers;
- recommended worker/provider/model + explanation;
- capabilities/effects;
- **Approve plan** as a distinct button;
- replan/edit path without executing.

### Tests

- Contract confirmation alone causes no execution;
- plan intelligence output tied to exact Contract revision;
- stale Contract revision refuses plan result;
- invalid Plan rejected;
- plan can be reproposed before approval;
- no provider session/Attempt is spawned as a side effect of `ProposePlan` except a non-authoritative IntelligenceRun adapter process if that adapter is explicitly used.

---

## Task 7 — Finish WT3 routing inside Plan formation

**Purpose:** Complete the already-started provider/model work at the correct authority boundary.

**Primary files:**

- `backend/internal/domain/execution_preference.go`
- `backend/internal/domain/execution_binding.go`
- `backend/internal/domain/routing.go`
- `backend/internal/domain/outcome_plan.go`
- `backend/internal/service/outcome/provider_binding.go`
- `backend/internal/service/outcome/plan.go`
- `backend/internal/storage/sqlite/store/outcome_execution_preference_store.go`
- `backend/internal/storage/sqlite/store/outcome_provider_store.go`
- `backend/internal/storage/sqlite/migrations/0114_execution_routing_bindings.sql`
- provider capability/readiness/model-catalog adapters under `backend/internal/adapters/agent/...`

### Required semantics

1. Outcome execution preference overrides Project baseline.
2. Project provider+model means explicit binding preference.
3. Project provider/no model means provider-default semantics.
4. no preference means no implicit provider and specifically no implicit Codex.
5. model without provider invalid.
6. unknown readiness/capability/model support cannot satisfy hard requirements.
7. explicit model support is evaluated **within its provider candidate only**; never require the same model string across providers.
8. worker and coordinator/intelligence role admission are independent.
9. router has no provider-brand branches and no Codex tie rule.
10. no admissible candidate returns `NO_VALID_CANDIDATE` / Action Required and launches nothing.
11. routing decision/provenance is persisted with the proposed Plan.
12. Plan approval freezes exact WorkUnit execution binding semantics.

### Domain tests

Add fictional-provider routing tests (use IDs that prove the algorithm is brand-neutral):

- preferred candidate wins equivalent admissible candidates;
- materially stronger nonpreferred candidate may win per explicit scoring policy;
- unavailable preferred candidate loses;
- unknown hard capability rejected;
- unsupported explicit model rejected only for that provider;
- no valid candidate returns first-class no-candidate decision;
- coordinator and worker role tests are independent;
- stable deterministic tie ordering is not provider-brand special casing.

### Approval regression

Remove the old rule that re-reads mutable Project worker preference during Plan approval and rejects the already-proposed provider when Project config changed.

Preference influences recommendation. **Approved Plan is authority.**

---

## Task 8 — Freeze exact binding at approval and pass it through Attempt spawn

**Purpose:** Ensure execution cannot silently reroute after approval.

**Primary files:**

- `backend/internal/service/outcome/plan.go`
- `backend/internal/service/outcome/attempt.go`
- `backend/internal/ports/attempt_execution.go`
- production `AttemptSessionSpawner` implementation(s)
- `backend/internal/domain/outcome_plan.go`
- `backend/internal/storage/sqlite/store/outcome_provider_store.go`
- relevant attempt/provider tests

### Changes

- approval validates proposed routing decision + current Contract + capability grants;
- approved WorkUnit has exact immutable `ExecutionBinding`;
- `AttemptSpawnRequest` carries provider plus model-selection semantics/model (or a narrow equivalent `AgentConfig` that cannot re-resolve Project preference);
- `StartAttempt` reads the approved WorkUnit binding only;
- it may check readiness for **that exact provider/model**, but may not pick another candidate;
- mutable Project config is not re-read to choose execution;
- `historical_unbound` returns explicit Action Required/non-executable state;
- RunBrief digest validation includes exact binding;
- retry creates new Attempt lineage using the same authorized binding unless a new PlanRevision is approved.

### Tests

- explicit model reaches spawner exactly;
- provider-default semantics reaches spawner without inventing a model name;
- Project preference changed after approval has no effect;
- unavailable approved provider pauses/action-required rather than reroutes;
- historical provider-only row cannot execute;
- tampered model changes digest and is refused;
- idempotent/restarted start does not duplicate Attempt.

---

## Task 9 — Add a truthful serialized scheduler for the MVP

**Purpose:** Make Mission Control advance approved WorkUnits without blocking on the full WorkspaceLease parallel scheduler.

**Files:**

- Prefer a new focused package such as `backend/internal/service/outcome/scheduler.go`
- existing `backend/internal/lifecycle/` reconciliation hooks
- `backend/internal/service/outcome/attempt.go`
- relevant ports/store queries
- scheduler tests

### MVP policy

- only approved current PlanRevision is schedulable;
- choose the first dependency-satisfied non-terminal WorkUnit in stable plan order;
- at most one mutable execution WorkUnit for the Outcome/Project when existing safety fences require it;
- explicit `NeedsAction` when binding/readiness/authority fails;
- never imply parallelism in UI;
- successful provider process termination does not itself accept WorkUnit/Outcome; canonical completion/evidence rules still apply;
- restart reconciliation checks canonical Attempt/session state before spawning anything new.

Do **not** implement fake DAG concurrency. ADR 0008/0009 parallel scheduling follows after MVP.

### Tests

- no Plan approval → no scheduling;
- approved serial WorkUnits run in order;
- failed/blocked unit prevents dependent unit from starting;
- restart with live Attempt does not spawn duplicate;
- unknown runtime does not count as dead/completed;
- cancellation/retry produces traceable lineage.

---

## Task 10 — Make Outcome/Mission Control the only normal new-work destination

**Purpose:** Eliminate the user-visible AO/session-first bypass.

**Primary frontend files:**

- `frontend/src/renderer/routes/_shell.work.tsx`
- `frontend/src/renderer/routes/_shell.tsx`
- `frontend/src/renderer/components/Sidebar.tsx`
- `frontend/src/renderer/components/outcome/WorkShell.tsx`
- `frontend/src/renderer/components/outcome/OutcomesOverviewSurface.tsx`
- `frontend/src/renderer/components/outcome/OutcomeLifecycleShell.tsx`
- `frontend/src/renderer/components/outcome/OutcomeRunSurface.tsx`
- `frontend/src/renderer/components/outcome/OutcomeMissionControl.tsx`
- `frontend/src/renderer/components/outcome/OutcomeAttemptTerminalPanel.tsx`
- `frontend/src/renderer/lib/outcome-tree.ts`

### Legacy paths to audit/gate

- `frontend/src/renderer/components/TaskComposer.tsx`
- `NewTaskDialog.tsx`
- `GlobalNewTaskDialog.tsx`
- `SessionsBoard.tsx`
- `ShellTopbar.tsx`
- `OrchestratorReplacementDialog.tsx`
- `SessionInspector.tsx`
- `frontend/src/renderer/lib/navigate-to-session.ts`
- `frontend/src/renderer/lib/restart-orchestrator.ts`
- `frontend/src/renderer/lib/command-palette.ts`
- `/projects/$projectId/sessions/$sessionId`
- `/sessions/$sessionId`

The session routes may remain for deep links/history. Remove them from normal Outcome ownership/navigation.

### Required UX

- sidebar/list displays Outcomes and their derived state/attention;
- click Outcome → `/work` with Outcome + derived stage;
- after Contract confirmation → Decide & Authorize;
- after Plan approval → Act & Observe/Mission Control;
- **Open terminal** drills into the current Attempt/session without changing Outcome identity;
- browser back from terminal returns to the Outcome context;
- generic session board is not the default Work board;
- hide/disable unfinished Home primary nav for MVP;
- global empty-Outcomes CTA chooses/registers Project then enters Outcome capture, not new session.

### Frontend tests

Update/add:

- `Sidebar.test.tsx`
- `OutcomeLifecycleShell.test.tsx`
- `AdaptiveIntakeSurface.test.tsx`
- `OutcomeDecideAuthorizeSurface.test.tsx`
- `OutcomeMissionControl.test.tsx`
- `OutcomeRunSurface.test.tsx`
- `OutcomesOverviewSurface.test.tsx`
- route/navigation tests under `frontend/src/renderer/routes` or nearest existing harness.

Assertions must include **no automatic navigation to a session route** from Outcome create/confirm/approve/open.

---

## Task 11 — Evidence, verification, and owner acceptance end-to-end

**Purpose:** Complete the Outcome loop rather than stopping when the coding agent exits.

**Primary files:**

- existing Outcome evidence/verification/acceptance domain/service/store files
- `frontend/src/renderer/components/outcome/OutcomeProveCloseSurface.tsx`
- `frontend/src/renderer/components/outcome/OutcomeRunSurface.tsx`
- existing verification/acceptance tests

### MVP requirements

- WorkUnit/Attempt output creates attributable evidence candidates/facts;
- verification maps back to stable Contract criterion IDs;
- failed/unconfirmed verification is visible and can return the Outcome to execution/action-required;
- provider/session “done” does not create AcceptanceDecision;
- only owner/user action accepts/rejects/continues;
- accepted Outcome stays inspectable with full lineage.

For the first MVP, verification may combine deterministic repository checks and owner review. Do not invent automated proof where no reliable verifier exists.

---

## Task 12 — Remove remaining Agent Orchestrator authority semantics

**Purpose:** Finish the architectural cut after the replacement path is working.

Search the repository for:

```text
orchestrator
mission role
new task
default worker
Spawn(... KindWorker ...)
projects/$projectId/sessions
navigate-to-session
TaskComposer
SessionsBoard
```

Classify every match:

1. **keep as execution/chassis compatibility**;
2. **rename/reframe** (e.g. coordinator preference rather than orchestrator authority);
3. **remove/gate from new work**;
4. **historical only**.

Do not mechanically rename provider-native concepts that are technically accurate. The target is authority semantics, not grep cleanliness.

### Backend rules to enforce

- no Outcome create/confirm endpoint calls an execution spawn;
- no Plan proposal endpoint calls an execution spawn;
- no Project config setter replaces/restarts an “orchestrator session” as a side effect unless invoked through explicit legacy/session-management UI;
- no new-work service chooses a provider from Project config after Plan approval;
- generic session create API remains an explicit low-level/debug capability only if still required by product/chassis.

### Frontend rules

- user-facing primary settings say Coordinator/Planning preference and Worker/Execution preference once copy is stable;
- no “orchestrator is the project” visual affordance;
- session lifecycle does not drive Outcome status.

---

## Task 13 — API/schema/store generation and full verification

### Generated contracts

Whenever API/storage sources changed:

```bash
npm run sqlc
npm run api
```

Verify generated diffs are intentional.

### Backend verification

```bash
cd backend
go test ./internal/domain
go test ./internal/service/intake
go test ./internal/service/outcome
go test ./internal/storage/sqlite/store
go test ./internal/daemon
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

### Root/frontend verification

```bash
npm run lint
npm run frontend:typecheck
npm run test:foundation
cd frontend
npm run typecheck
npm run build
npm test -- --runInBand
```

Use the actual frontend test command from `frontend/package.json` if it differs; do not invent a passing command.

### Static invariant searches

Search diffs/source for:

- hidden `Codex` default in routing/new-work paths;
- Outcome create/confirm calling execution spawn;
- Plan proposal calling execution spawn;
- model string selected without provider;
- approved Attempt re-reading Project worker/model to select execution;
- Outcome click handlers routing to session paths.

---

## Task 14 — Real-daemon MVP dogfood

This is the release gate, not optional polish.

Use an isolated profile/database and a disposable real Git repository.

### Scenario

Create the Outcome:

> Add a small visible README section that explains how to run this project locally, verify the command works, and leave the repository ready for review.

### Checkpoint A — Understand

Verify:

- intake persisted;
- optional IntelligenceRun visible/provenanced;
- material question works if produced;
- Contract review appears;
- **zero execution Attempts**;
- **zero execution AgentSessionRefs**.

### Checkpoint B — Decide & Authorize

Confirm Contract.

Verify:

- Outcome + immutable ContractRevision exist;
- Plan drafting runs/proposes;
- routing reason/provider/model visible;
- Plan can be reviewed before execution;
- still zero execution Attempts/sessions.

Approve Plan.

Verify:

- exact binding frozen;
- mutable Project preference change does not rewrite it;
- no candidate case produces Action Required and launches nothing.

### Checkpoint C — Act & Observe

Verify:

- Outcome opens Mission Control;
- scheduler starts only authorized WorkUnit;
- Attempt spawns exact provider/model semantics;
- terminal is drill-down;
- sidebar selection continues to open Outcome, not session;
- activity/progress survives renderer refresh.

### Checkpoint D — Prove & Close

Verify:

- actual repository change is inspectable;
- expected verification command/result is attached as evidence;
- criterion coverage is visible;
- session completion did not auto-accept;
- owner can accept or request more work.

### Checkpoint E — restart

Restart daemon/app while an Outcome is non-terminal.

Verify:

- canonical lineage recovers;
- no duplicate Attempt starts;
- live/unknown/terminal process state is reconciled truthfully;
- Outcome destination remains correct.

Capture logs/screenshots/test evidence for the PR.

---

## Task 15 — Review, documentation truth, and PR readiness

Before declaring the MVP complete:

1. use the repository's code-review workflow on the complete diff;
2. fix all correctness/authority/recovery issues;
3. re-run the relevant verification after fixes;
4. update `docs/STATUS.md` from target to implemented truth only for behavior actually verified;
5. consolidate any temporary amendment wording back into canonical product docs where safe;
6. keep ADR 0010/0011 as durable decisions;
7. update the PR body with:
   - architecture change;
   - deleted/gated AO authority paths;
   - retained runtime infrastructure;
   - intelligence adapter selected and why;
   - WT3 routing semantics;
   - verification evidence;
   - known MVP limitations (especially serialized execution if full scheduler not yet landed).

Do not merge automatically unless the user explicitly asks.

---

# Acceptance checklist

The MVP is not complete until all applicable items below are demonstrated:

- [ ] Project can be registered without immediately creating an orchestrator execution session for Outcome work.
- [ ] User can capture natural-language Outcome intent.
- [ ] Capture creates no execution Attempt/session.
- [ ] Contract intelligence produces a grounded proposal or material question.
- [ ] Offline/manual Contract floor remains available.
- [ ] Contract is editable and owner-confirmed.
- [ ] Contract confirmation creates no execution Attempt/session.
- [ ] Plan intelligence produces a reviewable Plan.
- [ ] Plan proposal produces no execution Attempt/session.
- [ ] Routing is provider-neutral, model-aware, deterministic, and explainable.
- [ ] No implicit Codex fallback exists.
- [ ] No valid candidate becomes Action Required and launches nothing.
- [ ] Plan approval is explicit.
- [ ] Approved WorkUnit binding is immutable.
- [ ] Attempt uses exact approved provider/model semantics.
- [ ] Project preference changes after approval do not reroute the Attempt.
- [ ] Mission Control is the post-approval Outcome destination.
- [ ] Clicking an Outcome never automatically opens a session route.
- [ ] Terminal/chat is explicit Attempt/session drill-down.
- [ ] Evidence maps to WorkUnit/Attempt and Contract criteria.
- [ ] Verification state is visible.
- [ ] Provider completion cannot accept Outcome.
- [ ] Owner/user creates final AcceptanceDecision.
- [ ] Restart/reload does not duplicate execution.
- [ ] Historical sessions remain readable.
- [ ] Unfinished Home is not presented as a complete primary beta surface.
- [ ] Full Go/frontend generation, tests, build, race/vet where feasible are recorded.

# After the MVP

Resume long-term kernel work in this order:

1. full direct-Outcome WorkUnit DAG;
2. WorkspaceLease and dependency scheduler from ADR 0008/0009;
3. truthful parallel Mission Graph;
4. structured SessionReceipt/WorkUnitReceipt continuity;
5. deeper provider role conformance;
6. governed external provider ingress;
7. richer Project Brief/Context and personal Waldo continuity;
8. learned routing/skill promotion only after deterministic baseline telemetry exists.

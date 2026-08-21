# Learning Experiment and Evaluation Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Evaluate one bounded learning candidate against a current baseline with locked held-in, hidden held-out, adversarial, cost, and safety measures.

**Architecture:** The daemon persists immutable campaign/variant/suite/run/result lineage and delegates isolated execution through existing worktree/runtime ports. Candidate workers may change only the declared artifact; evaluator, permissions, budgets, instrumentation, corpus, and promotion code remain outside their authority.

**Tech Stack:** Go, SQLite/goose/sqlc, existing workspace/runtime adapters, OpenAPI, React, deterministic and model-assisted evaluators

**Spec:** `docs/superpowers/specs/2026-08-21-waldo-learning-skill-evolution-design.md` sections 7 and L2

## Global Constraints

- L1 Experience Ledger gate must pass.
- One campaign has one hypothesis and one editable surface.
- Baseline and candidate use the same current suite/environment class.
- The proposer cannot read hidden held-out cases or modify evaluator/runtime authority.
- Missing evaluator results are unknown and block promotion.
- No experiment may push, merge, deploy, publish, message, pay, change permissions, or activate a skill.

---

### Task 1: Immutable campaign and evaluation contracts

**Files:**
- Create: `backend/internal/domain/experiment.go`, tests
- Create: `backend/internal/ports/experiment_store.go`, `experiment_executor.go`
- Create: `backend/internal/service/experiments/service.go`, tests
- Create: `backend/internal/storage/sqlite/migrations/0110_learning_experiments.sql`
- Create: `backend/internal/storage/sqlite/queries/experiments.sql`
- Create: `backend/internal/storage/sqlite/store/experiment_store.go`, tests
- Create: `backend/internal/httpd/controllers/experiments.go`, tests
- Modify/regenerate: DTO, API registry, sqlc, OpenAPI, frontend schema

**Interfaces:**
- Consumes: approved-for-experiment LearningCandidate revision
- Produces: `ExperimentCampaign`, `ExperimentVariant`, `EvaluationSuiteRevision`, immutable lifecycle

- [ ] **Step 1: Write failing immutability tests**

Cover exact baseline/candidate digest, declared editable path set, frozen suite, executor/evaluator identity, budgets, stop rules, revision conflict, idempotency, and denial of evaluator/permission/audit changes.

- [ ] **Step 2: Implement campaign contract**

```go
type CampaignBudget struct {
    WallClockSeconds, MaxAttempts, MaxConcurrent, MaxWorktrees int
    MaxTokens int64
    MaxCostMicros int64
}

type EditableSurface struct { Kind string; Paths []string; MaxBytes int64 }
```

- [ ] **Step 3: Regenerate, verify, and commit**

```bash
npm run sqlc && npm run api
cd backend && go test ./internal/domain ./internal/service/experiments ./internal/storage/sqlite/store ./internal/httpd/... -count=1
git add backend frontend/src/api/schema.ts
git commit -m "feat: add immutable learning campaigns"
```

### Task 2: Isolated candidate executor and budget enforcement

**Files:**
- Create: `backend/internal/service/experiments/runner.go`, `runner_test.go`
- Create: `backend/internal/adapters/experiment/worktree_executor.go`, tests
- Modify: daemon wiring through a new experiment-specific port
- Create: `test/fixtures/learning/experiments/*`

**Interfaces:**
- Consumes: campaign, variant, existing workspace/runtime ports
- Produces: immutable `EvaluationRun` with raw artifact references and cleanup receipt

- [ ] **Step 1: Write failing isolation tests**

Attempt candidate writes to baseline, suite, hidden corpus, permissions, audit, and out-of-scope paths; exceed wall clock, tokens, cost, worktree, and attempt budgets; simulate crash and cleanup failure.

- [ ] **Step 2: Implement generation-fenced execution**

```go
type Executor interface {
    Run(context.Context, RunRequest) (RunReceipt, error)
    Reconcile(context.Context, RunID) (RunReceipt, error)
    Cleanup(context.Context, RunID) (CleanupReceipt, error)
}
```

Create isolated worktrees under `~/.kennel`; never reset the user's branch. Store content-addressed candidate/results outside canonical SQLite payload columns and reference them by digest.

- [ ] **Step 3: Verify and commit**

```bash
cd backend && go test ./internal/service/experiments ./internal/adapters/experiment -run 'Isolation|Budget|Cleanup|Reconcile' -count=1
git add backend test/fixtures/learning
git commit -m "feat: run bounded learning experiments"
```

### Task 3: Held-out, adversarial, and no-regression evaluator

**Files:**
- Create: `backend/internal/service/experiments/evaluator.go`, tests
- Create: `backend/internal/domain/evaluation.go`, tests
- Extend: experiment queries/store/controller
- Create: `test/fixtures/learning/evaluation/{held-in,held-out,adversarial}/*`

**Interfaces:**
- Consumes: baseline and candidate run artifacts
- Produces: `EvaluationResult`, `PromotionReadiness` explanation; never PromotionDecision

- [ ] **Step 1: Write failing decision tests**

Cover held-in improvement with held-out regression, aggregate gain with a release-blocker regression, evaluator unavailable, same-model non-independence, exploit detection, excessive tokens/latency/cost, and material improvement beyond noise.

- [ ] **Step 2: Implement metric-vector comparison**

```go
type MetricResult struct {
    Name string
    Baseline, Candidate float64
    Direction string
    ReleaseBlocking bool
    MaterialDelta float64
}
```

Promotion-ready requires every release-blocker non-regressing, one intended material gain, adversarial pass, complete raw bindings, and truthful evaluator independence.

- [ ] **Step 3: Verify and commit**

```bash
cd backend && go test ./internal/domain ./internal/service/experiments -run 'HeldOut|NoRegression|Exploit|Readiness' -count=1
git add backend test/fixtures/learning/evaluation
git commit -m "feat: evaluate learning candidates against held-out work"
```

### Task 4: Experiment review UI and L2 gate

**Files:**
- Create: `frontend/src/renderer/components/learning/ExperimentDetail.tsx`, tests
- Create: `frontend/src/renderer/components/learning/EvaluationComparison.tsx`, tests
- Create: `backend/e2e/learning_experiment_test.go`
- Create: `frontend/e2e/learning-experiment.spec.ts`
- Create: `docs/evaluation/learning-experiment-gate.md`

**Interfaces:**
- Consumes: Tasks 1-3 APIs
- Produces: inspectable experiment result and L2 evidence; no activation

- [ ] **Step 1: Write failing UI/e2e tests**

Show hypothesis, editable surface, sources, baseline, variants, held-in/held-out/adversarial vectors, raw result links, failures, costs, executor/evaluator identity, stop/cleanup, and explicit `not independently evaluated` state.

- [ ] **Step 2: Implement review UI**

Never collapse the metric vector into a universal builder/productivity score. Promotion review remains disabled until L3 exists.

- [ ] **Step 3: Run gate and commit**

```bash
cd backend && go test ./e2e -run LearningExperiment -count=1
cd ../frontend && npm test -- --run ExperimentDetail EvaluationComparison && npx playwright test e2e/learning-experiment.spec.ts && npm run typecheck
npm run lint
git add backend/e2e frontend docs/evaluation
git commit -m "test: gate bounded learning experiments"
```

Record reproducibility, evaluator exploit detection, cleanup success, cost/latency, no-regression correctness, and reviewer comprehension. L3 stays blocked if candidates can mutate protected surfaces or hidden evaluation can be inferred from proposer inputs.

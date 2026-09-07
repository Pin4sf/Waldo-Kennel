# Learning Skill Registry and Provisional Activation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote one evaluated procedural skill into a daemon-owned, Project-scoped provisional binding with invocation receipts, monitoring, rollback, revoke, and deletion.

**Architecture:** SQLite owns stable skill identity, immutable revisions, evaluation evidence, lifecycle, bindings, and receipts. The daemon compiles an active binding into a versioned RunBrief reference and provider-specific materialization. Provider skill catalogs remain external observations and never become canonical automatically.

**Tech Stack:** Go, SQLite/goose/sqlc, provider adapter capability contracts, OpenAPI, React

**Spec:** `docs/superpowers/specs/2026-08-21-waldo-learning-skill-evolution-design.md` sections 8-9 and L3

## Global Constraints

- L1 and L2 gates must pass and current Outcome result signals must be trustworthy.
- First binding scope is one Project and one evaluated task family.
- Only the user promotes the exact revision/scope; no trace, model, or experiment promotes itself.
- Effective tools are the intersection of skill request, ContractRevision, PlanRevision, grants, adapter capabilities, worktree ownership, budget, and effect ceiling.
- No provider/repository/user-wide skill directory becomes canonical.
- Invocation never implies authority or Acceptance.
- Immediate suspend, rollback, revoke, and deletion must work during provider degradation and restart.

---

### Task 1: Canonical skill registry, revisions, and bindings

**Files:**
- Create: `backend/internal/domain/skill_registry.go`, tests
- Create: `backend/internal/ports/skill_registry_store.go`
- Create: `backend/internal/service/skillregistry/service.go`, tests
- Create: `backend/internal/storage/sqlite/migrations/0111_skill_registry.sql`
- Create: `backend/internal/storage/sqlite/queries/skill_registry.sql`
- Create: `backend/internal/storage/sqlite/store/skill_registry_store.go`, tests
- Create: `backend/internal/httpd/controllers/skill_registry.go`, tests
- Modify/regenerate: DTO, API registry, sqlc, OpenAPI, frontend schema

**Interfaces:**
- Consumes: promotion-ready candidate and exact EvaluationResult
- Produces: `SkillRecord`, immutable `SkillRevision`, `PromotionDecision`, `SkillBinding`

- [ ] **Step 1: Write failing lifecycle and metadata tests**

Cover provenance/license, trigger/exclusions, scope, tool manifest, data classes, confirmations, rollback, protected surfaces, evaluation identity, expiry, provisional/active/suspended/revoked/superseded/deleted states, idempotent decision, and no silent scope widening.

- [ ] **Step 2: Implement registry contracts**

```go
type PromotionAction string
const (
    PromotionTrial PromotionAction = "trial"
    PromotionActivate PromotionAction = "activate"
    PromotionSuspend PromotionAction = "suspend"
    PromotionRollback PromotionAction = "rollback"
    PromotionRevoke PromotionAction = "revoke"
    PromotionDelete PromotionAction = "delete"
)
```

PromotionDecision is immutable. Binding changes append decisions and revisions; they do not overwrite audit history.

- [ ] **Step 3: Regenerate, verify, and commit**

```bash
npm run sqlc && npm run api
cd backend && go test ./internal/domain ./internal/service/skillregistry ./internal/storage/sqlite/store ./internal/httpd/... -count=1
git add backend frontend/src/api/schema.ts
git commit -m "feat: add governed Waldo skill registry"
```

### Task 2: Provider-neutral RunBrief binding and invocation receipts

**Files:**
- Create: `backend/internal/service/skillregistry/compiler.go`, tests
- Create: `backend/internal/domain/invocation_receipt.go`, tests
- Modify: shared RunBrief contract from Work lane
- Modify: provider adapter capability profiles and admission tests
- Extend: skill registry queries/store/controller
- Preserve: `backend/internal/service/chat/skills.go` as external provider catalog

**Interfaces:**
- Consumes: Project binding, current RunBrief, provider adapter capability/authority
- Produces: `CompiledSkillRef`, provider materialization digest, `InvocationReceipt`, typed degradation

- [ ] **Step 1: Write failing authority and adapter tests**

Cover optional skill unsupported, required skill unsupported, requested tool denied, provider materialization drift, stale revision/generation, provider catalog name collision, restart, and invocation without Acceptance side effects.

- [ ] **Step 2: Implement compiler intersection**

```go
type CompileInput struct {
    RunBriefRevisionID, SkillRevisionID string
    AdapterCapabilities []string
    EffectiveTools []string
    PolicyGeneration int64
}
```

Do not write `.claude`, `.agents`, Codex, or repository skill directories. Adapter output is a rebuildable, generation-fenced artifact under `~/.kennel`.

- [ ] **Step 3: Verify and commit**

```bash
cd backend && go test ./internal/domain ./internal/service/skillregistry ./internal/adapters/... -run 'Skill|RunBrief|Invocation|Capability' -count=1
git add backend
git commit -m "feat: bind governed skills to agent attempts"
```

### Task 3: Promotion review, controls, and immediate rollback

**Files:**
- Create: `frontend/src/renderer/components/learning/PromotionReview.tsx`, tests
- Create: `frontend/src/renderer/components/skill-registry/SkillRegistry.tsx`, tests
- Create: `frontend/src/renderer/components/skill-registry/SkillDetail.tsx`, tests
- Modify: Settings & Control route
- Modify: Operator Inspector to show SkillRevision/InvocationReceipt

**Interfaces:**
- Consumes: registry/promotion/invocation APIs
- Produces: explicit `Try for this Project`, activate/suspend/rollback/revoke/delete/export controls

- [ ] **Step 1: Write failing control tests**

Promotion review must show exact diff, hypothesis, provenance, evaluation vectors, trigger, Project scope, exclusions, tools, data/privacy, protected surfaces, expiry, cost, failures, and rollback target. No global scope option appears.

- [ ] **Step 2: Implement provisional-first UX**

The first positive decision is `trial` for one Project. Active behavior shows a visible revision badge and last evaluation freshness. Inspector shows exact receipt/degradation; it never calls the skill “accepted work.”

- [ ] **Step 3: Verify and commit**

```bash
npm --prefix frontend test -- --run PromotionReview SkillRegistry SkillDetail
npm --prefix frontend run typecheck
git add frontend/src/renderer
git commit -m "feat: review and control learned project skills"
```

### Task 4: Effectiveness, drift, deletion, and L3 gate

**Files:**
- Create: `backend/internal/service/skillregistry/effectiveness.go`, tests
- Create: `backend/e2e/skill_registry_test.go`, `skill_deletion_test.go`
- Create: `frontend/e2e/project-skill-trial.spec.ts`
- Create: `docs/evaluation/project-skill-gate.md`

**Interfaces:**
- Consumes: invocation receipts and canonical Outcome results
- Produces: `EffectivenessObservation`, suspension proposal, L3 evidence

- [ ] **Step 1: Write failing lifecycle safety tests**

Cover real-work harm, counter-evidence, provider/model/dependency drift, stale evaluation, deleted source, replay race, rollback during active Attempts, revocation before next compile, and source-derived non-resurrection.

- [ ] **Step 2: Implement observation-only monitoring**

EffectivenessObservation may propose suspend/re-evaluate; it cannot change the binding. Source deletion immediately revokes directly attributable content. Generalized content enters `needs_revalidation` and requires an explicit independent adoption decision.

- [ ] **Step 3: Run the complete first-skill gate**

```bash
cd backend && go test ./internal/service/skillregistry ./e2e -run 'Skill|Deletion|Rollback|Drift' -count=1
cd ../frontend && npm test -- --run Skill && npx playwright test e2e/project-skill-trial.spec.ts && npm run typecheck
npm run lint
cd backend && go test -race ./...
```

- [ ] **Step 4: Record results and commit**

Compare with-skill versus no-skill quality, supervision minutes, interventions, retries, recovery failures, token/context overhead, latency, cost, false guidance, unauthorized effects, rollback time, and review burden. L4 orchestration-policy learning remains closed until this gate shows material benefit without unacceptable burden.

```bash
git add backend frontend/e2e docs/evaluation
git commit -m "test: gate the first evaluated Project skill"
```

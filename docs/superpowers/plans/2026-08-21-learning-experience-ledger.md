# Learning Experience Ledger Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build consented, Project-attributed, correctable LearningEpisodes and LearningCandidates in shadow mode without activating skills or changing responsibility truth.

**Architecture:** A daemon service projects permitted canonical Outcome/session facts into rebuildable episodes. SQLite owns grants, source references, candidate revisions, corrections, and deletion generations. Candidate mining is observer-only and routes proposals to distinct review domains.

**Tech Stack:** Go, SQLite/goose/sqlc, Chi/OpenAPI, React/TanStack Router, deterministic fixtures

**Spec:** `docs/superpowers/specs/2026-08-21-waldo-learning-skill-evolution-design.md` sections 3-6 and L1

## Global Constraints

- Requires current `ResponsibilitySpace`, Project, Outcome, Attempt, Evidence, Verification, and Acceptance identifiers; incomplete result facts remain labeled incomplete.
- No `LearningGrant` means no session-derived learning.
- Raw transcripts are excluded by default; chain-of-thought, secrets, raw health values, and unrelated files never enter.
- A LearningEpisode is a rebuildable projection, not Memory, responsibility, Evidence, or Acceptance.
- Low-confidence attribution remains `unattributed` and cannot influence evaluation.
- No candidate promotion, provider skill materialization, or production behavior change.

---

### Task 1: LearningGrant and source attribution

**Files:**
- Create: `backend/internal/domain/learning.go`, `learning_test.go`
- Create: `backend/internal/ports/learning_store.go`
- Create: `backend/internal/service/learning/grants.go`, `attribution.go`, tests
- Create: `backend/internal/storage/sqlite/migrations/0107_learning_grants_sources.sql`
- Create: `backend/internal/storage/sqlite/queries/learning.sql`
- Create: `backend/internal/storage/sqlite/store/learning_store.go`, tests
- Create: `backend/internal/httpd/controllers/learning.go`, tests
- Modify/regenerate: DTO, API registry, sqlc, OpenAPI, frontend schema

**Interfaces:**
- Consumes: Project/Outcome/Attempt/source identifiers
- Produces: `LearningGrant`, `LearningSourceRef`, `AttributionResult`

- [ ] **Step 1: Write failing source-policy tests**

Cover no-grant denial, excluded session/path/person/sensitivity, processing disclosure, date range, source generation, explicit identifiers outranking branch/path/time similarity, and unattributed material ambiguity.

- [ ] **Step 2: Implement deterministic attribution first**

```go
type AttributionResult struct {
    ProjectID domain.ProjectID
    OutcomeID *domain.OutcomeID
    Confidence string
    Reasons []string
    NeedsReview bool
}
```

Semantic similarity may add a reason but cannot override conflicting explicit identifiers.

- [ ] **Step 3: Regenerate, verify, and commit**

```bash
npm run sqlc && npm run api
cd backend && go test ./internal/domain ./internal/service/learning ./internal/storage/sqlite/store ./internal/httpd/... -run 'LearningGrant|Attribution' -count=1
git add backend frontend/src/api/schema.ts
git commit -m "feat: add governed learning sources"
```

### Task 2: Rebuildable LearningEpisode projection

**Files:**
- Create: `backend/internal/service/learning/episodes.go`, `episodes_test.go`
- Create: `backend/internal/storage/sqlite/migrations/0108_learning_episodes.sql`
- Modify: `backend/internal/storage/sqlite/queries/learning.sql`
- Modify: `backend/internal/storage/sqlite/store/learning_store.go`
- Create: `test/fixtures/learning/episodes/*.json`

**Interfaces:**
- Consumes: eligible `LearningSourceRef`s and canonical result facts
- Produces: immutable `LearningEpisodeRevision`, `LearningSignal`

- [ ] **Step 1: Write failing grouping tests**

Fixtures cover one task across sessions, multiple tasks in one session, subagent lineage, wrong repository label, retry/recovery, user intervention, missing result, deleted source, and idempotent rebuild.

- [ ] **Step 2: Implement projection contract**

```go
type LearningEpisodeRevision struct {
    ID, RevisionID string
    ProjectID domain.ProjectID
    SourceRefs []domain.LearningSourceRefID
    Intent, ResultState string
    Signals []LearningSignal
    AttributionConfidence string
    ProjectionVersion string
}
```

Episode rebuild appends a revision and advances generation; it never mutates Outcome/Attempt facts.

- [ ] **Step 3: Verify and commit**

```bash
cd backend && go test ./internal/service/learning ./internal/storage/sqlite/store -run 'Episode|Signal|Rebuild' -count=1
git add backend test/fixtures/learning
git commit -m "feat: project attributable learning episodes"
```

### Task 3: Candidate mining and routing

**Files:**
- Create: `backend/internal/domain/learning_candidate.go`, tests
- Create: `backend/internal/service/learning/candidates.go`, tests
- Create: `backend/internal/storage/sqlite/migrations/0109_learning_candidates.sql`
- Modify: learning query/store
- Extend: learning controller/API
- Create: `frontend/src/renderer/components/learning/ProjectLearning.tsx`, tests
- Modify: future/current Project route integration

**Interfaces:**
- Consumes: episode revisions and counter-evidence
- Produces: `LearningCandidate` routed to skill/context-rule/orchestration-policy/Memory/OpenLoop review

- [ ] **Step 1: Write failing addressability and routing tests**

Task difficulty, provider outage, ambiguous ownership, and unavailable capability are not skill defects. Repeated reusable procedure failures route to SkillCandidate; facts route to MemoryCandidate; unresolved obligations route to Open Loop review; topology patterns route to OrchestrationPolicyCandidate.

- [ ] **Step 2: Implement candidate envelope**

```go
type LearningCandidate struct {
    ID, RevisionID, Kind, Scope, Hypothesis string
    SourceSpans []LearningSourceSpan
    CounterEvidence []LearningSourceSpan
    Uncertainty string
    Applicability, Exclusions []string
    State string
}
```

- [ ] **Step 3: Add “What Waldo noticed” review**

Show scope, mechanism, provenance, counter-evidence, uncertainty, and correct/split/merge/exclude/reject/defer actions. Do not show a builder or productivity score.

- [ ] **Step 4: Regenerate, verify, and commit**

```bash
npm run sqlc && npm run api
cd backend && go test ./internal/domain ./internal/service/learning ./internal/httpd/... -run 'Candidate|Routing|Addressability' -count=1
cd ../frontend && npm test -- --run ProjectLearning && npm run typecheck
git add backend frontend
git commit -m "feat: add shadow learning candidates"
```

### Task 4: Correction, deletion, and shadow gate

**Files:**
- Create: `backend/e2e/learning_ledger_test.go`, `learning_deletion_test.go`
- Create: `frontend/e2e/project-learning-shadow.spec.ts`
- Create: `docs/evaluation/learning-experience-ledger-gate.md`

**Interfaces:**
- Consumes: Tasks 1-3
- Produces: L1 gate evidence; no active skill

- [ ] **Step 1: Add failure-injection tests**

Cover split/merge/exclude/regenerate, revoked grant, source deletion during projection, stale replay, wrong project, large resumable batch, restart, and derived-data non-resurrection.

- [ ] **Step 2: Run gates**

```bash
cd backend && go test ./e2e -run 'LearningLedger|LearningDeletion' -count=1
cd ../frontend && npx playwright test e2e/project-learning-shadow.spec.ts
npm run lint && npm run frontend:typecheck
```

- [ ] **Step 3: Record and commit evidence**

Record attribution precision, unattributed rate, candidate precision, review burden, provenance completeness, deletion latency, throughput, and storage. L2 stays blocked if responsibility/proof mutation or deletion resurrection occurs.

```bash
git add backend/e2e frontend/e2e docs/evaluation
git commit -m "test: gate the shadow Learning Experience Ledger"
```

# Kennel PR Convergence and Architecture Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Preserve and accept F0-F6, replace PR #11 with bounded post-foundation cleanup, and establish a clean base for the first approved Kennel v1 vertical slice.

**Architecture:** F0-F6 remains an indivisible prerequisite. PR #11 is not merged or rebased wholesale: accepted cleanup is reapplied on new branches from the accepted foundation, while its unapproved Outcome schema/store, premature provider deletion, CLI authority assumptions, and locale deletion are omitted. Product feature work begins only after the remaining gates in the v1 product architecture are approved.

**Tech Stack:** Git/GitHub, Go daemon and SQLite, Electron/React/TypeScript, generated OpenAPI/sqlc artifacts, npm foundation scripts

**Spec:** `docs/product/kennel-v1-product-architecture.md`

## Global Constraints

- Never place post-foundation cleanup or feature edits on `codex/foundation-gate-f0-f6`.
- Do not merge, deploy, publish, release, force-push, or rewrite public history without explicit authorization.
- Preserve user work and use a separate `codex/` branch/worktree for every implementation slice.
- The primary daemon listener remains unauthenticated and loopback-only.
- All application state remains under `~/.kennel`.
- Durable observable changes use SQLite-triggered `change_log` CDC.
- Already-merged migrations are immutable; new migrations are additive.
- The Electron renderer remains a thin daemon client.
- Generated OpenAPI and frontend TypeScript contracts travel together.
- Provider completion and verification never create user acceptance.

---

### Task 1: Accept the F0-F6 prerequisite without product scope

**Files:**
- Verify: `docs/foundation-acceptance-2026-08-18.md`
- Verify: `scripts/test-foundation.sh`
- Verify: `.github/workflows/ci.yml`
- Verify: `.github/workflows/security.yml`

**Interfaces:**
- Consumes: PR #1 at `codex/foundation-gate-f0-f6`
- Produces: one accepted post-foundation base commit for all later branches

- [ ] **Step 1: Confirm the PR head and clean checkout**

```bash
gh pr view 1 --repo Pin4sf/Waldo-Kennel --json state,isDraft,headRefName,headRefOid,baseRefName,mergeable,mergeStateStatus,statusCheckRollup
git status --short --branch
```

Expected: the checkout is clean; the PR head and expected base are explicit. Do not infer acceptance from `mergeable` alone.

- [ ] **Step 2: Install the normalized dependency sets**

```bash
npm run bootstrap
```

Expected: exit 0. Record audit warnings as known development-toolchain debt; do not run broad automatic audit fixes.

- [ ] **Step 3: Run the complete foundation gate**

```bash
npm run test:foundation
npm run audit:production
npm run lint
cd backend && go test -race ./...
```

Expected: every command exits 0. If the fake-agent timing test fails under whole-suite load, rerun it three times and fix or quarantine the deterministic timing assumption before accepting the foundation; do not label a failed full gate green.

- [ ] **Step 4: Record manual evidence while hosted Actions are unavailable**

Record command, timestamp, commit SHA, exit code, and intentional skips in the PR description or maintainer checklist. Do not claim GitHub checks exist when `statusCheckRollup` is empty.

- [ ] **Step 5: Obtain maintainer approval before merge**

Do not merge as part of this task unless the user separately authorizes it. The deliverable is a verified, reviewable foundation prerequisite.

### Task 2: Create the post-foundation cleanup branches

**Files:**
- Inspect: all 71 paths changed by PR #11
- Do not copy: `backend/internal/domain/outcome.go`
- Do not copy: `backend/internal/storage/sqlite/migrations/0099_outcome_control_plane.sql`
- Do not copy: `backend/internal/storage/sqlite/queries/outcomes.sql`
- Do not copy: `backend/internal/storage/sqlite/store/outcome_store.go`
- Do not copy: generated Outcome sqlc artifacts

**Interfaces:**
- Consumes: the accepted F0-F6 base and PR #11 as a donor diff only
- Produces: issue-sized post-foundation branches with no permanent Outcome schema

- [ ] **Step 1: Create an isolated branch from the accepted foundation**

```bash
git fetch origin --prune
task_worktree_root=$(mktemp -d /tmp/waldo-kennel-cleanup.XXXXXX)
git worktree add "$task_worktree_root/worktree" -b codex/remove-legacy-user-import ef0baffd7850702480cba84ae97e2790367eab12
```

Expected: a clean branch whose merge base is the accepted foundation, not the pre-foundation `main` used by PR #11.

- [ ] **Step 2: Capture the donor diff without applying it**

```bash
git diff --stat b1bcd37fd0911a86da5bb21c07ac3a0b9adecb3b...dbebb0956d2e039fefc77bb963c6ef9bc0c7e28b
git diff --name-status b1bcd37fd0911a86da5bb21c07ac3a0b9adecb3b...dbebb0956d2e039fefc77bb963c6ef9bc0c7e28b
```

Expected: all donor files are visible for classification. Do not cherry-pick the PR #11 commit.

- [ ] **Step 3: Classify every donor file**

Give each path exactly one disposition: `reapply`, `rewrite`, `defer`, or `drop`. Outcome persistence files are `drop`; locale deletion is `drop`; provider catalog deletion and public CLI narrowing are `defer` until their contracts are approved.

- [ ] **Step 4: Keep one issue per branch and PR**

Legacy-import removal, provider admission, and later CLI narrowing are separate review units. Do not recreate PR #11 as another mixed cleanup/feature commit.

### Task 3: Remove only the user-facing legacy import flow

**Files:**
- Delete: `backend/internal/cli/import.go`
- Delete: `backend/internal/cli/import_test.go`
- Delete: `backend/internal/httpd/controllers/imports.go`
- Delete: `backend/internal/httpd/controllers/imports_test.go`
- Delete: `backend/internal/legacyimport/`
- Delete: `backend/internal/service/importer/`
- Delete: `frontend/src/renderer/components/MigrationPopup.tsx`
- Delete: `frontend/src/renderer/components/MigrationPopup.test.tsx`
- Delete: `frontend/src/renderer/components/MigrationSection.tsx`
- Delete: `frontend/src/renderer/hooks/useMigrationOffer.ts`
- Modify: `backend/internal/cli/root.go`
- Modify: `backend/internal/httpd/api.go`
- Modify: `backend/internal/httpd/apispec/specgen/build.go`
- Modify: `backend/internal/httpd/controllers/dto.go`
- Modify: `frontend/src/renderer/main.tsx`
- Modify: `frontend/src/renderer/components/settings/GeneralSettingsSection.tsx`
- Modify: `backend/internal/skillassets/using-ao/SKILL.md`
- Modify: `backend/internal/skillassets/using-ao/references.md`
- Modify: `frontend/src/landing/content/docs/cli.mdx`
- Modify: `frontend/src/landing/content/docs/plugins/index.mdx`
- Regenerate: `backend/internal/httpd/apispec/openapi.yaml`
- Regenerate: `frontend/src/api/schema.ts`

**Interfaces:**
- Consumes: existing thin CLI and code-first HTTP contract
- Produces: no user-facing probing/import of legacy AO state while preserving developer import and documented source/project compatibility seams

- [ ] **Step 1: Write failing boundary assertions**

Add or update tests proving `kennel import` and import API operations are absent, the migration offer is not rendered, and active skill/landing documentation contains no `ao import` instruction. Preserve `devimport`, `.ao/attachments`, `.ao/launch.json`, `backend/cmd/ao`, Go-module, and provenance/sync seams.

- [ ] **Step 2: Run the focused tests and confirm failure**

```bash
cd backend
go test ./internal/cli ./internal/httpd/... ./internal/skillassets/...
cd ..
npm --prefix frontend test -- --run Migration
rg -n 'ao import|kennel import' backend/internal/skillassets frontend/src/landing/content/docs
```

Expected: at least one assertion or source scan fails before implementation because the current user-facing import flow exists.

- [ ] **Step 3: Remove the bounded implementation and references**

Delete only the listed user-facing import packages, routes, DTOs, UI, tests, and active instructions. Do not read, move, or delete any user legacy data.

- [ ] **Step 4: Regenerate API artifacts**

```bash
npm run api
```

Expected: OpenAPI and frontend schema no longer expose import operations and remain mutually generated.

- [ ] **Step 5: Run focused verification**

```bash
cd backend
go test ./internal/cli ./internal/httpd/... ./internal/skillassets/...
cd ..
npm --prefix frontend run typecheck
npm --prefix frontend test -- --run Migration
test -z "$(rg -l 'ao import|kennel import' backend/internal/skillassets frontend/src/landing/content/docs)"
```

Expected: every command exits 0.

- [ ] **Step 6: Commit the cleanup**

```bash
git add backend frontend
git commit -m "chore: remove legacy user import flow"
```

### Task 4: Separate provider admission from historical recovery

**Files:**
- Modify: `backend/internal/domain/harness.go`
- Modify: `backend/internal/domain/harness_test.go`
- Modify: `backend/internal/storage/sqlite/store/agent_switching_store.go`
- Test: `backend/internal/storage/sqlite/store/agent_switching_store_test.go`
- Modify after provider approval: `backend/internal/adapters/agent/registry/registry.go`
- Modify after provider approval: `backend/internal/adapters/reviewer/registry.go`
- Modify after provider approval: `frontend/src/renderer/lib/agent-select-options.ts`
- Modify after provider approval: `frontend/src/renderer/lib/reviewer-harnesses.ts`

**Interfaces:**
- Consumes: the approved provider capability/admission matrix
- Produces: separate recognition of historical provider identities and selectability for new work

- [ ] **Step 1: Stop if the provider matrix is not approved**

The new-work launch set is locked to Codex only. Do not treat that decision as permission to delete inherited provider identities or historical decoder paths. Stop until the Codex matrix and historical non-Codex recovery contract record authentication truth, Chat/TUI, model selection, continuation, cancellation, worktree behavior, evidence support, unattended/subscription constraints, cost visibility, deletion, and fallback.

- [ ] **Step 2: Write the historical recovery test**

Persist a session using one retired provider identity, load it successfully, expose it as unavailable for new work, and prove it can switch or migrate to an admitted provider without rewriting its original identity.

- [ ] **Step 3: Verify the test fails under PR #11 semantics**

```bash
cd backend
go test ./internal/domain ./internal/storage/sqlite/store -run 'Historical|Retired|Selectable' -count=1
```

Expected: failure demonstrates that one `IsKnown` predicate cannot represent both historical decoding and new-work admission.

- [ ] **Step 4: Introduce explicit predicates**

Use domain vocabulary equivalent to `IsRecognizedPersisted` and `IsSelectable`. Do not delete retired constants or historical decoder paths.

- [ ] **Step 5: Apply the approved launch matrix consistently**

Update backend registries, frontend options, reviewer availability, and readiness presentation together. A provider missing one capability must expose an honest degraded/admission result rather than implied parity.

- [ ] **Step 6: Run focused verification**

```bash
cd backend
go test ./internal/domain ./internal/adapters/agent/registry ./internal/adapters/reviewer ./internal/storage/sqlite/store
cd ../frontend
npm run typecheck
```

Expected: every command exits 0, including historical-row recovery.

- [ ] **Step 7: Commit the bounded provider change**

```bash
git add backend frontend
git commit -m "feat: separate provider admission from historical recovery"
```

### Task 5: Run the post-foundation cleanup gate

**Files:**
- Verify: all files changed by Tasks 3 and 4
- Verify: generated artifacts have no drift

**Interfaces:**
- Consumes: issue-sized cleanup commits
- Produces: reviewable PR candidates with no product ontology migration

- [ ] **Step 1: Inspect the final diff and prohibited scope**

```bash
git diff --check ef0baffd7850702480cba84ae97e2790367eab12...HEAD
git diff --name-status ef0baffd7850702480cba84ae97e2790367eab12...HEAD
test -z "$(git diff --name-only ef0baffd7850702480cba84ae97e2790367eab12...HEAD | rg 'outcome_control_plane|outcome_store|queries/outcomes|gen/outcomes')"
```

Expected: no whitespace errors and no PR #11 Outcome schema/store artifacts.

- [ ] **Step 2: Run the complete repository gate**

```bash
npm run test:foundation
npm run audit:production
npm run lint
cd backend && go test -race ./...
```

Expected: every command exits 0. Record exact local evidence if hosted Actions remain unavailable.

- [ ] **Step 3: Prepare PR descriptions without publishing**

Each description names the accepted foundation SHA, linked issue, exact omissions from PR #11, verification evidence, historical compatibility behavior, and why no Outcome migration is included. Opening or merging a PR requires separate user authorization.

### Task 6: Hold the feature gate before the first Outcome vertical slice

**Files:**
- Read: `docs/product/kennel-v1-product-architecture.md`
- Create after approval: one separate implementation plan for the first vertical slice

**Interfaces:**
- Consumes: approved Codex admission/historical recovery, RunBrief/admission, observability, evaluation, and effect-ceiling decisions
- Produces: a test-driven vertical-slice plan whose schema, service, API, UI, CDC, and user-visible acceptance behavior ship together

- [ ] **Step 1: Approve the remaining design gates**

Obtain explicit approval for gates 1-5 in the specification. Hosted attachment gate 6 may remain deferred for the fully local launch.

- [ ] **Step 2: Select one end-to-end dogfood Outcome**

Use the local-only Focus Ledger Outcome from the architecture handoff. The slice demonstrates contract definition, smallest sufficient execution topology, local authority, provider Attempt, evidence, verification, and owner acceptance without account, cloud sync, analytics, or deployment.

- [ ] **Step 3: Write a separate vertical-slice implementation plan**

Save it under `docs/superpowers/plans/` and enumerate exact domain, port, service, migration/query, CDC, controller/DTO, generated API, frontend, recovery, and evaluation files. Do not reuse PR #11's schema by default.

- [ ] **Step 4: Stop before feature implementation**

Feature execution requires the user's explicit approval of that vertical-slice plan.

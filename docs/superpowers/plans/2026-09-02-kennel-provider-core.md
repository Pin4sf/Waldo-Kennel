# Kennel Provider Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the AO-derived, Codex-gated provider stack with a five-provider Kennel core that discovers local readiness, supports explicit worker/orchestrator choices, routes without brand hardcoding, removes dead AO surfaces, and retires transcript-marker Outcome authority.

**Architecture:** One provider registry owns the shipped integrations and normalized capability metadata. The daemon probes readiness and performs role admission; React renders that projection rather than filtering providers locally. Project configuration stores explicit choices, automatic routing is transparent and capability-based, and provider-specific launch/auth quirks remain inside adapters.

**Tech Stack:** Go 1.25.7, SQLite/sqlc, Chi/OpenAPI, Electron, React 19, TypeScript, TanStack Query, Vitest/Playwright.

**Spec:** `docs/superpowers/specs/2026-09-02-kennel-provider-core-design.md`

## Global Constraints

- Active provider ids are exactly `codex`, `claude-code`, `opencode`, `cursor`, and `pi`.
- No historical AO/provider compatibility layer is required.
- No product service or renderer may define `provider == codex` as the new-work admission rule.
- No silent provider substitution is permitted.
- Deterministic safety, authorization, dependency, idempotency, recovery, evidence, and acceptance invariants remain deterministic.
- Provider/session completion must not close an Outcome.
- Current product identity is Kennel; active AO names are removed rather than aliased.

---

### Task 1: Collapse the provider vocabulary and registries

**Files:**
- Modify: `backend/internal/domain/harness.go`
- Modify: `backend/internal/domain/reviewerharness.go` or remove it if reviewer identity is folded into provider capability
- Modify: `backend/internal/adapters/registry.go`
- Modify: `backend/internal/adapters/agent/registry/registry.go`
- Modify: `backend/internal/adapters/reviewer/registry.go`
- Delete: inactive adapter directories under `backend/internal/adapters/agent/*` and `backend/internal/adapters/reviewer/*`
- Test: provider registry/domain tests

**Interfaces:**
- Produces: one five-provider vocabulary and normalized provider capabilities used by inventory, project admission, review, and routing.

- [ ] Write failing tests asserting the shipped worker registry contains only Codex, Claude Code, OpenCode, Cursor, and Pi.
- [ ] Run the focused Go tests and verify they fail against the 27-provider registry.
- [ ] Introduce normalized capabilities to the provider manifest and annotate the five adapters.
- [ ] Remove inactive adapters and duplicate reviewer-only provider vocabulary; reviewer support becomes a capability.
- [ ] Run focused registry/domain tests and verify green.
- [ ] Commit `refactor: reduce Kennel to five provider integrations`.

### Task 2: Make readiness role-aware and probe all five providers

**Files:**
- Modify: `backend/internal/service/agent/service.go`
- Modify: `backend/internal/httpd/apispec/openapi.yaml`
- Modify: DTO/controller files serving provider inventory
- Modify: `frontend/src/api/schema.ts` via generation
- Test: `backend/internal/service/agent/*_test.go`

**Interfaces:**
- Produces: `ProviderReadiness`/inventory response containing supported identity, installed state, auth state, capabilities, ready roles, and actionable reason.

- [ ] Write failing tests proving `Refresh` probes all five shipped providers and does not filter by a Codex-only predicate.
- [ ] Write readiness tests for not-installed, auth-required, auth-unknown-safe, and ready role cases.
- [ ] Implement normalized readiness projection while preserving bounded parallel probes, refresh coalescing, and spawn-time authoritative checks.
- [ ] Update OpenAPI and generated TypeScript schema.
- [ ] Run backend tests and frontend schema/type checks.
- [ ] Commit `feat: expose provider readiness and capabilities`.

### Task 3: Remove Codex defaults from project configuration

**Files:**
- Modify: `backend/internal/service/project/service.go`
- Modify: `backend/internal/daemon/daemon.go`
- Modify: `backend/internal/domain/projectconfig.go`
- Modify: project controllers/DTOs as required
- Test: project/daemon wiring tests

**Interfaces:**
- Consumes: provider readiness/capabilities from Task 2.
- Produces: explicit worker/orchestrator provider configuration with no Codex fallback.

- [ ] Write failing tests showing a ready Claude/OpenCode/Cursor/Pi provider can be configured for worker/orchestrator roles.
- [ ] Write a test proving an invalid/unready explicit provider is rejected instead of silently replaced by Codex.
- [ ] Remove `DefaultHarness: HarnessCodex` daemon wiring and service fallbacks.
- [ ] Validate role capability and support separately from local spawn readiness.
- [ ] Keep project-path, env, symlink, and worktree security validation unchanged.
- [ ] Run project and daemon tests.
- [ ] Commit `refactor: make project provider selection explicit`.

### Task 4: Replace frontend Codex filters with one provider picker model

**Files:**
- Modify: `frontend/src/renderer/lib/agent-select-options.ts`
- Modify: `frontend/src/renderer/components/CreateProjectAgentSheet.tsx`
- Modify: `frontend/src/renderer/components/ReviewerSelect.tsx`
- Modify: `frontend/src/renderer/components/SwitchAgentDialog.tsx`
- Modify: settings components using provider pickers
- Modify: i18n copy for install/auth/readiness states
- Test: corresponding Vitest component/lib tests

**Interfaces:**
- Consumes: daemon provider readiness projection.
- Produces: shared role-aware picker behavior for worker, orchestrator, reviewer, switch, and settings surfaces.

- [ ] Write failing tests proving the picker shows all five supported providers, disables unavailable ones, and shows reason text.
- [ ] Write tests proving multiple ready providers do not auto-collapse to Codex.
- [ ] Remove local `.filter(id === "codex")`, singleton Codex fallback arrays, and Codex reset/default code.
- [ ] Select only role-ready providers while keeping unavailable supported providers visible.
- [ ] Add refresh/recheck affordance and actionable status copy.
- [ ] Run frontend tests and typecheck.
- [ ] Commit `feat: make Kennel provider selection machine-aware`.

### Task 5: Replace static provider routing with capability admission

**Files:**
- Modify: `backend/internal/service/outcome/planning.go`
- Modify: session delegation/spawn admission services
- Create: focused routing policy module if needed
- Test: outcome planning/routing tests

**Interfaces:**
- Consumes: requested provider, WorkUnit requirements, provider readiness/capabilities, user/project preference.
- Produces: admitted explicit provider or transparent candidate result; no hidden brand priority.

- [ ] Write failing tests for explicit provider preservation, capability rejection, model incompatibility, and tied automatic candidates.
- [ ] Remove `Codex > Claude Code > OpenCode` priority ordering.
- [ ] Implement explicit admission with stable structured errors.
- [ ] Implement automatic routing over admitted candidates using declared requirements/preferences only.
- [ ] Return ambiguity instead of hidden brand tie-breaking.
- [ ] Run outcome/session tests.
- [ ] Commit `refactor: route work by capability instead of provider brand`.

### Task 6: Generalize switching, hooks, usage, and review behavior

**Files:**
- Modify: `backend/internal/session_manager/agent_switching.go`
- Modify: `backend/internal/cli/hooks.go`
- Modify: `backend/internal/service/usage/capabilities.go`
- Modify: reviewer service/resolver call sites
- Test: switching/hooks/usage/review tests

**Interfaces:**
- Consumes: provider manifest capabilities.
- Produces: capability checks instead of Codex/Claude switch statements.

- [ ] Add failing tests that capabilities, not hard-coded provider ids, determine switch/review/hook/usage eligibility.
- [ ] Replace provider-name switches where behavior can be represented as manifest capability.
- [ ] Leave true provider-specific parsing/translation inside the adapter package.
- [ ] Run focused tests.
- [ ] Commit `refactor: drive provider behavior from capabilities`.

### Task 7: Retire marker-based Outcome state authority

**Files:**
- Delete: `frontend/src/renderer/lib/outcome-coordination.ts` after call sites are migrated
- Modify/Delete: `OutcomeIntakePanel.tsx`, `OutcomeOrchestrationGraph.tsx`, and tests
- Modify: `SessionsBoard.tsx`, `TaskComposer.tsx`, and generated API consumers
- Add/complete daemon-owned Outcome/Contract/Plan command/read APIs required by current UI slice
- Test: backend Outcome lifecycle + frontend projection tests

**Interfaces:**
- Produces: renderer derives stage from typed daemon state only; model messages never become canonical through magic marker parsing.

- [ ] Write failing frontend test proving transcript text containing a marker cannot advance canonical Outcome state.
- [ ] Write backend lifecycle tests for explicit contract/plan mutation and acceptance boundaries.
- [ ] Replace marker commands with typed daemon commands and projections.
- [ ] Remove marker parser and donor graph/intake authority.
- [ ] Run Outcome/backend/frontend tests.
- [ ] Commit `refactor: move Outcome authority into Kennel daemon`.

### Task 8: Remove AO product baggage and rebrand active source

**Files:**
- Delete: `frontend/src/landing`
- Delete: `packages/mobile`
- Delete: `packages/cloud-client`
- Delete: `packages/ao*`
- Delete: obsolete AO-only docs/assets/scripts
- Rename: `backend/cmd/ao` -> `backend/cmd/kennel`
- Modify: `backend/go.mod` and every active Go import
- Modify: active comments, env/header names, `.ao/*` project artifacts, README/AGENTS/CONTEXT/CONTRIBUTING, root scripts/package config

**Interfaces:**
- Produces: one Kennel-branded active repository and build graph.

- [ ] Add a repository identity test/script that fails on active `github.com/aoagents/agent-orchestrator`, `backend/cmd/ao`, product-facing `AO`, `X-AO-*`, and `.ao/` references.
- [ ] Remove donor packages/surfaces from npm workspace/build scripts.
- [ ] Rename the Go module/import graph and CLI entrypoint.
- [ ] Rename current environment/header/artifact identifiers to Kennel equivalents.
- [ ] Update docs to describe only the supported Kennel product.
- [ ] Run identity/foundation tests.
- [ ] Commit `chore: remove AO product identity and donor surfaces`.

### Task 9: Full verification and cleanup gate

**Files:**
- Modify only files required to fix verification failures.

- [ ] Run `npm run bootstrap`.
- [ ] Run `npm run test:foundation`.
- [ ] Run `npm run audit:production`.
- [ ] Run backend Go tests/lint.
- [ ] Run frontend typecheck and tests.
- [ ] Run package identity checks.
- [ ] Search active tree for removed provider ids, Codex-only selection gates, AO identity, and Outcome marker constants.
- [ ] Review generated API diff and repository diff for accidental compatibility baggage.
- [ ] Commit any verification fixes as focused commits.

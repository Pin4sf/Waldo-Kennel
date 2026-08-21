# AO Legacy Retirement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove misleading Agent Orchestrator product identity and disconnected donor surfaces while preserving license provenance, historical data readability, and measured source/project compatibility seams.

**Architecture:** Apply expand–migrate–contract. First add Kennel/Waldo replacements and an explicit allowlist, then migrate active consumers, prove donor independence, and only then delete exact obsolete paths. Go module/source entrypoint and project-local `.ao` formats remain separate future migrations.

**Tech Stack:** Go, TypeScript/Electron/React, npm scripts, package identity checks, shell provenance checks

**Spec:** `docs/product/ao-legacy-retirement-audit.md`

## Global Constraints

- Keep `LICENSE`, `NOTICE`, `upstream.json`, `docs/upstream-provenance.md`, and `scripts/compare-upstream.sh`.
- Do not edit already-merged SQLite migrations or relabel historical provider/session identities.
- Keep `backend/cmd/ao`, the Go module path, sqlc imports, `.ao/attachments`, and `.ao/launch.json` in this plan.
- No AO npm package may be published. No Waldo/Kennel release is authorized.
- Every deletion task records its exact path list and checks active consumers before the destructive commit.
- Preserve unrelated user work and never use bulk destructive shell commands.

---

### Task 1: Replace active AO strings and visual identity

**Files:**
- Create: `frontend/assets/waldo-mark.svg`
- Create/update: `frontend/assets/icon.png`, `icon.icns`, `icon.ico`, `trayIconTemplate.png`, `trayIconTemplate@2x.png`
- Modify: `frontend/src/renderer/components/DaemonStartupLoader.tsx`, `Sidebar.tsx`
- Modify: `frontend/src/main.ts`, `frontend/src/main/*.ts`, `frontend/src/shared/file-annotations.ts`
- Modify: active `frontend/src/renderer/**/*.{ts,tsx}` product/diagnostic copy
- Modify: `frontend/src/renderer/i18n/*.json`
- Modify: active backend user-facing errors/prompts under `backend/internal/**`
- Modify: `frontend/src/renderer/lib/report-problem.ts`
- Create: `scripts/check-active-product-vocabulary.sh`
- Test: relevant existing frontend/backend tests and `frontend/scripts/assert-package-identity.mjs`

**Interfaces:**
- Consumes: canonical Waldo mark from `waldo-landing@962d022`, SHA-256 `40071ff75328a5cf74c504ee7fe8e53cf50ce8d858046720305f6576c910eef2`
- Produces: Kennel/Waldo active identity and explicit AO compatibility allowlist

- [ ] **Step 1: Write the failing vocabulary and asset checks**

The script scans active backend/frontend surfaces and permits AO only for module imports, historical compatibility constants, explicit migration messages, provenance, and test fixtures. It fails on `AO daemon`, `AO Cloud`, `AO feedback`, AO logo imports, AO release URLs, or `source: "AO"` in active product code.

```bash
bash scripts/check-active-product-vocabulary.sh
```

Expected: FAIL on the observed active AO strings and assets.

- [ ] **Step 2: Import the exact Waldo source asset**

Copy the reviewed SVG into `frontend/assets/waldo-mark.svg`, preserve its source commit/hash in a comment or adjacent asset manifest, and generate checked-in app/tray formats. The desktop label stays `Kennel`; accessible labels say `Waldo in Kennel` where the mark represents the agent.

- [ ] **Step 3: Rename active copy by ownership**

Use `Kennel` for app, daemon, workspace, runtime, browser, project, and feedback operations. Use `Waldo` only for the personal agent, responsibility, clarification, memory, learning, and proactive assistance. Preserve `Agent Orchestrator` only when explaining upstream or historical state.

- [ ] **Step 4: Verify**

```bash
bash scripts/check-active-product-vocabulary.sh
npm --prefix frontend test
npm --prefix frontend run typecheck
cd backend && go test ./...
npm --prefix frontend run package
npm --prefix frontend run package:identity
```

Visually inspect packaged icon, startup loader, sidebar at 16/20/32/64 px, light/dark themes, and tray template behavior.

- [ ] **Step 5: Commit**

```bash
git add frontend/assets frontend/src backend/internal scripts/check-active-product-vocabulary.sh
git commit -m "fix: replace active AO product identity"
```

### Task 2: Detach donor build and dependency consumers

**Files:**
- Modify: `scripts/bootstrap.sh`, `scripts/test-foundation.sh`, `scripts/audit-production.sh`
- Modify: `.github/dependabot.yml`
- Modify: root `package.json` only if donor scripts are referenced
- Create: `scripts/check-retired-donor-consumers.sh`
- Modify: `README.md`, `docs/STATUS.md`, `docs/development.md`, `docs/foundation-acceptance-2026-08-18.md`

**Interfaces:**
- Consumes: Task 1 current desktop/backend checks
- Produces: foundation matrix independent of `frontend/src/landing`, `packages/mobile`, and `packages/ao*`

- [ ] **Step 1: Add a failing consumer check**

Reject active build, audit, Dependabot, workflow, package, and current-doc references that require the three donor trees. Permit historical mentions in the audit and Git history.

- [ ] **Step 2: Remove donor jobs, not coverage**

Replace landing/mobile build checks with existing root/backend/frontend/product-ui/cloud-client checks. Keep cloud-client until Task 5 decides it separately.

- [ ] **Step 3: Verify the replacement matrix**

```bash
bash scripts/check-retired-donor-consumers.sh
npm run bootstrap
npm run test:foundation
npm run audit:production
npm run lint
npm run frontend:typecheck
```

- [ ] **Step 4: Commit**

```bash
git add scripts .github/dependabot.yml package.json README.md docs
git commit -m "chore: detach AO donor build consumers"
```

### Task 3: Remove disconnected donor products and obsolete media

**Files:**
- Delete: `frontend/src/landing/**`
- Delete: `packages/mobile/**`
- Delete: `packages/ao/**`, `packages/ao-darwin-arm64/**`, `packages/ao-darwin-x64/**`, `packages/ao-linux-x64/**`, `packages/ao-win32-x64/**`
- Delete: `translations/README.de.md`, `README.es.md`, `README.fr.md`, `README.ja.md`, `README.ko.md`, `README.pt-BR.md`, `README.zh-CN.md`
- Delete: `docs/ao-start-bootstrapper-and-npm-deprecation.md`
- Delete after reference proof: root `assets/ao-dashboard-preview.png`, `ao-logo.svg`, `first.png`, `image.png`, `second.png`, `third.png`, `tweet1.png`, `tweet2.png`, `tweet3.png`
- Delete after reference proof: obsolete `docs/assets/readme/**`
- Modify: any exact mock fixture path that names deleted media without reading it

**Interfaces:**
- Consumes: Task 2 zero-consumer proof and Task 1 replacement assets
- Produces: repository without disconnected AO landing/mobile/npm products

- [ ] **Step 1: Record exact deletion evidence and request confirmation**

```bash
git ls-files frontend/src/landing packages/mobile packages/ao packages/ao-* translations assets docs/assets/readme docs/ao-start-bootstrapper-and-npm-deprecation.md > /tmp/kennel-ao-retirement-paths.txt
wc -l /tmp/kennel-ao-retirement-paths.txt
```

Attach the path list and Task 2 green evidence to the issue. Obtain explicit confirmation for this destructive commit.

- [ ] **Step 2: Remove only confirmed paths with patch-based deletion**

Do not use recursive shell deletion. Preserve any file that gained a current consumer since the audit.

- [ ] **Step 3: Prove no broken references and run the full matrix**

```bash
bash scripts/check-retired-donor-consumers.sh
bash scripts/check-active-product-vocabulary.sh
rg -n 'frontend/src/landing|packages/mobile|packages/ao|ao-logo|ao-dashboard-preview' . --glob '!.git/**'
npm run bootstrap
npm run test:foundation
npm run audit:production
npm run lint
npm run frontend:typecheck
cd backend && go test -race ./...
```

The `rg` result contains only the retirement audit/plan and permitted provenance history.

- [ ] **Step 4: Commit**

Stage only the reviewed deletion list and necessary reference fixes, then commit:

```bash
git commit -m "chore: retire disconnected AO donor products"
```

### Task 4: Reconcile current technical documentation

**Files:**
- Modify: `docs/architecture.md`, `docs/stack.md`, `docs/backend-code-structure.md`
- Modify: `docs/harnesses/*.md`, `docs/posthog-cost-controls.md`
- Delete or archive after consumer proof: `docs/cloud-development.md`, `docs/cloud-refactor.md`
- Modify: `README.md`, `docs/README.md`, `docs/STATUS.md`, `docs/development.md`
- Preserve: `docs/adr/*`, `docs/upstream-provenance.md`, `docs/foundation-acceptance-2026-08-18.md`

**Interfaces:**
- Consumes: canonical product architecture and current code after Tasks 1-3
- Produces: one current Kennel chassis narrative plus clearly labeled historical records

- [ ] **Step 1: Add documentation classification headers**

Every retained technical doc says whether it is current Kennel behavior, upstream provenance, historical decision, or future design. Replace active AO subject wording with Kennel while retaining exact source paths and historical values in code examples.

- [ ] **Step 2: Remove stale cloud product promises**

Keep current `packages/cloud-client` technical facts only where tested. Remove hosted AO URLs, AO Cloud product claims, and private-repository workflow instructions that are not part of ADR 0006.

- [ ] **Step 3: Verify links, vocabulary, and current-state claims**

```bash
git diff --check
bash scripts/check-active-product-vocabulary.sh
rg -n 'AO Cloud|api\.aoagents\.dev|Agent Orchestrator is' README.md docs --glob '!docs/product/ao-legacy-retirement-audit.md' --glob '!docs/adr/**' --glob '!docs/upstream-provenance.md'
```

- [ ] **Step 4: Commit**

```bash
git add README.md docs
git commit -m "docs: make Kennel the current technical subject"
```

### Task 5: Decide remaining compatibility seams independently

**Files:**
- Inspect: `packages/cloud-client/**`, release/pod/updater helpers, LAN/mobile bridge backend
- Create: `docs/product/remaining-compatibility-seams.md`
- Do not modify: Go module path, `backend/cmd/ao`, `.ao/attachments`, `.ao/launch.json`

**Interfaces:**
- Consumes: Tasks 1-4 consumer inventory
- Produces: per-component retain/migrate/retire decisions and separate issue links

- [ ] **Step 1: Measure every remaining consumer**

Record build scripts, imports, tests, persisted data, runtime endpoints, downstream repositories, and accepted architecture use for each seam.

- [ ] **Step 2: Apply the decision rule**

Retain when a current accepted surface consumes it. Migrate when a replacement exists and compatibility can be additive. Retire only with zero-consumer proof. Open a separate issue for the Go module/path migration because it is a repository-wide mechanical change with upstream-sync cost.

- [ ] **Step 3: Verify final active vocabulary boundary**

```bash
bash scripts/check-active-product-vocabulary.sh
git diff --check
```

- [ ] **Step 4: Commit**

```bash
git add docs/product/remaining-compatibility-seams.md
git commit -m "docs: classify remaining AO compatibility seams"
```

## Rollback

Each task is independently revertible. Task 3 records the last donor-bearing commit and exact path list; Git history restores source-only donors. No runtime database or user state is deleted. A failure in package identity, current builds, historical database reads, or provenance comparison stops contraction and restores the last green task.

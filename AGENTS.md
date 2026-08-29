# AGENTS.md

Operational guidance for coding agents working in this repository. Keep changes small, match the current rewrite architecture, and prefer the documented daemon/API boundaries over behavior from the old TypeScript implementation.

## Repo layout

- `backend/` — AO-derived Go chassis: Cobra `kennel` CLI (source entrypoint `cmd/ao`), loopback HTTP daemon, services, SQLite storage, lifecycle/reaper, runtime/workspace/agent/tracker adapters, terminal mux, and tests.
- `frontend/` — Electron + React supervisor wired to the daemon via the generated typed client. Treat it as a thin supervisor/UI surface; do not move daemon logic into it.
- `docs/` — current architecture/status notes. Start here before changing lifecycle, CLI, agents, storage, or daemon behavior.
- `test/` — external smoke/e2e assets, including the CLI fresh-install container check.
- `.github/workflows/` — CI definitions. Mirror these commands locally when possible.

## Commands

From the repo root unless noted:

```bash
npm run lint                         # backend go test ./... + golangci-lint v2.12.2
npm run frontend:typecheck           # frontend TypeScript check
npm run sqlc                         # regenerate backend/internal/storage/sqlite/gen from queries/schema
npm run api                          # regenerate OpenAPI spec + frontend TS types (see API contract changes below)
npx @redwoodjs/agent-ci run --all    # local workflow validation; requires Docker socket
```

Backend-specific checks:

```bash
cd backend
go build ./...
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/ao start # source seam; reports and installs as kennel
```

Frontend-specific checks:

```bash
cd frontend
npm run typecheck
npm run build
```

When showing or demoing frontend changes, run `kennel preview [url]` from inside the session so the change renders in the desktop browser panel (the inspector rail's Browser tab); do not just describe it.

## Where to look first

- `README.md` — current run/config/test quickstart.
- `docs/README.md` — docs index.
- `docs/architecture.md` — backend mental model, package layout, lifecycle/session/service boundaries, and load-bearing rules.
- `docs/STATUS.md` — what is shipped on `main` today and what is still in flight.
- `docs/product/kennel-v1-product-architecture.md` — canonical Waldo Kennel product ontology, exact responsibility/execution/evidence lineages, custody, governance, and phase boundaries. Read it before changing Outcome, Home, Work, authority, evidence, verification, acceptance, or continuity semantics.
- `docs/product/kennel-v0-first-outcome-slice.md` — locked first vertical proof and its falsifiable acceptance/recovery contract.
- `docs/adr/0007-composed-outcomes.md` and `docs/superpowers/plans/2026-08-29-composed-outcomes-program.md` — the composed-Outcome ontology and its phased delivery. Read both before changing decomposition, contribution binding, proof roll-up, or acceptance.
- `docs/cli/README.md` — intended CLI shape: thin Cobra client over daemon HTTP, never direct storage/runtime access.
- `docs/plans/island-app-unification.md` — the implementation record for Kennel Island as a window of the desktop app's single Electron process. Read it before touching `packages/kennel-island/`, Island lifecycle/settings, or session deep-link handling.
- `CLAUDE.md` — compatibility pointer for Claude Code; it directs agents back to `AGENTS.md`.

For code entry points:

- CLI commands: `backend/internal/cli/*.go`; follow nearby command/test patterns before adding a new style.
- HTTP controllers and DTOs: `backend/internal/httpd/controllers/`.
- Service read/write boundaries: `backend/internal/service/`.
- Domain vocabulary: `backend/internal/domain/`.
- Port contracts: `backend/internal/ports/`.
- SQLite queries/migrations/store: `backend/internal/storage/sqlite/`.
- Generated sqlc code: `backend/internal/storage/sqlite/gen/`.

## Distribution

- The **desktop app** is the intended canonical, auto-updating install path. The foundation gate does not establish or publish a Kennel release; do not point users at inherited AO artifacts.
- **Do not publish the frozen AO npm packages.** `packages/ao*` are inherited compatibility artifacts, not Kennel's install path. The Kennel desktop release path targets `Pin4sf/Waldo-Kennel`, but no release is authorized by the foundation gate.
- **Exactly one publisher.** Only the designated release conductor runs a real publish, on any channel. Divergent artifacts from multiple publishers made the 28-29 Jul macOS incident unreadable. Use the fork dev loop for test builds. Full rule and rationale: `frontend/docs/desktop-release.md`, "Hard rule: exactly one publisher".
- **Verify macOS artifacts with `frontend/scripts/verify-mac-artifact.sh`, never by hand.** It extracts with `ditto -x -k` and runs `codesign --verify --deep --strict`, `spctl -a -vv -t exec`, `xcrun stapler validate`. Plain `unzip` breaks the seal and yields a convincing false failure; `spctl` without `-vv` prints nothing at all on success.
- **macOS ships both a `.zip` and a `.dmg`.** The dmg is first install only. The zip and `latest-mac.yml` must keep publishing forever: electron-updater cannot install an update from a dmg. macOS differential updates are permanently disabled (full download only); see issues #3151 and #3267.

## Coding conventions

- Keep every change surgical and directly tied to the task. Avoid drive-by cleanup, broad renames, formatting churn, speculative abstractions, and architectural refactors unless the task explicitly asks for them.
- Follow existing Go package boundaries. CLI code should call daemon HTTP routes through shared CLI client helpers; it should not open SQLite, spawn runtimes, or call adapters directly.
- Keep Cobra commands in the relevant command file and table-test them in the style of `backend/internal/cli/*_test.go`.
- Mirror existing response/request DTOs in the CLI instead of importing HTTP controller packages into CLI code, unless the package already establishes that dependency.
- Return usage errors as `usageError` so CLI misuse exits 2; runtime/daemon failures should exit 1.
- Preserve API error envelopes and request IDs when surfacing daemon errors.
- Use `context.Context` as the first argument for functions that do I/O or blocking work.
- Do not add abstractions for one-off use cases. Add helpers only when they remove duplication across real call sites.
- Tests should cover the user-visible behavior and boundary being changed: happy path, validation/missing args, daemon error envelopes, and any destructive confirmation path.

## Hard rules and boundaries

- The daemon's **primary (loopback) listener** stays bound to `127.0.0.1` and unauthenticated. Do not change its bind host or add auth to it.
- The daemon MAY run a **second, opt-in LAN listener** (the "Connect Mobile" feature) that binds `0.0.0.0` **only while explicitly enabled**, **only** behind the bearer-password `authMiddleware`, serving the app API but never the loopback-gated control routes (`/shutdown`, telemetry, mobile control). It is plaintext and home-network-only by deliberate decision — see `docs/adr/0001-lan-listener-for-mobile.md` and `CONTEXT.md`. Do not add any other network-facing bind.
- The CLI is a thin client. Do not port old in-process TypeScript CLI behavior that bypasses daemon HTTP routes.
- The current `OutcomeTask`/`completed` overlay is donor code, not the accepted product ontology. New product work must preserve `Outcome -> ContractRevision -> PlanRevision -> WorkUnit -> Attempt -> AgentSessionRef -> EvidenceItem -> VerificationRun -> AcceptanceDecision`; provider/session completion, commits, PRs, or checks cannot accept or close an Outcome.
- An Outcome may instead be **decomposed** into contributing Outcomes ([ADR 0007](docs/adr/0007-composed-outcomes.md)). A decomposed Outcome owns a `DecompositionRevision` in place of a `PlanRevision` and never starts an Attempt; each contributing Outcome runs the full lineage above and earns its own `AcceptanceDecision`. Composition is two levels deep, contribution is criterion-bound, and a child's authority ceiling never exceeds its parent's. Do not add a second decomposition mechanism: `PlanRevision` stays at one direct `WorkUnit`, and dependencies live between contributing Outcomes.
- Do not store derived/display session status. Status is derived from durable facts (`activity_state`, `is_terminated`, PR/check/comment facts) at service read time.
- Do not treat failed/unknown runtime probes as proof a session is dead.
- Do not force-delete dirty registered worktrees.
- Do not modify already-merged SQLite migrations. Add a new migration instead.
- Do not hand-edit `backend/internal/storage/sqlite/gen/*`; change `backend/internal/storage/sqlite/queries/*` or migrations and run `npm run sqlc`.
- SQLite change events come from DB triggers into `change_log`; do not add parallel manual CDC emission from store methods unless the architecture changes explicitly.
- Keep generated OpenAPI/API DTO drift in mind: controller response shapes live in `backend/internal/httpd/controllers/dto.go` and tests may assert CLI/HTTP wire compatibility.
- Do not add network calls to tests unless the package already has an integration/e2e pattern for them. Prefer `httptest`, fakes, and injected dependencies.
- Do not commit local run state, daemon data, temporary worktrees, build outputs, or credentials.
- All app state lives under `~/.kennel` only. The daemon's data dir, `running.json`, worktrees, and the Electron supervisor's `userData` (Chromium cache, cookies, local/session storage, crash dumps) must resolve under `~/.kennel` (overridable via `KENNEL_DATA_DIR`/`KENNEL_RUN_FILE`). Never write to or read from `~/Library/Application Support` or any other OS default app-data location. `main.ts` pins Electron's `userData` to `~/.kennel/electron`; do not remove that override or rely on Electron's default path.

## API contract changes

The daemon API is code-first. The OpenAPI spec and frontend TypeScript types are generated artifacts — edit the source, then regenerate.

**Source files to edit:**

- `backend/internal/httpd/controllers/dto.go` — request/response shapes.
- `backend/internal/httpd/apispec/specgen/build.go` — operation registry; add a `schemaNames` entry for any new named type.

**Regenerate after editing:**

```bash
npm run api          # runs api:spec then api:ts in sequence
```

This is equivalent to running:

```bash
npm run api:spec     # cd backend && go generate ./internal/httpd/apispec/...
npm run api:ts       # npx openapi-typescript@7.4.4 backend/internal/httpd/apispec/openapi.yaml -o frontend/src/api/schema.ts
```

**Verify:**

```bash
cd backend && go test ./internal/httpd/...    # spec drift + route/spec parity tests (does not cover schema.ts — that is checked by the api-drift CI job)
```

Commit `openapi.yaml` and `frontend/src/api/schema.ts` together with the Go changes. CI will regenerate both files and fail if the committed versions are out of date. The CLI hand-mirrored DTOs remain a deliberate manual boundary and are not generated.

## PR hygiene

- Product and documentation work branches from the latest `beta` and opens a PR back to `beta`, unless explicitly continuing an existing PR. `main` advances through a separately reviewed and tested `beta` -> `main` promotion PR; release/hotfix exceptions require explicit maintainer direction.
- Keep one issue per PR. If asked for separate work, create a separate branch and PR.
- Use conventional commit messages (`feat:`, `fix:`, `docs:`, `test:`, `chore:`).
- Explain intentional omissions in the PR body, especially when the TypeScript original had more behavior than the Go rewrite domain currently supports.
- Run the narrowest relevant tests first, then the repo/CI commands that match the touched area.

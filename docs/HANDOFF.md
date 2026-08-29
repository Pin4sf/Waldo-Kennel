# Kennel handoff

> **Superseded — historical note only (2026-08-22).**
> The canonical product definition is [`docs/product/kennel-v1-product-architecture.md`](product/kennel-v1-product-architecture.md),
> with the first implementation slice in [`kennel-v0-first-outcome-slice.md`](product/kennel-v0-first-outcome-slice.md).
> Two things below are actively wrong: this file treats `domain.Outcome`/`OutcomeTask`/`Completed`
> as the model to persist, but the accepted architecture names them **donor code, not the target**
> (see issue #21, "do not revive donor OutcomeTask/completed semantics"); and its snapshot predates
> `main` at `25fec2c`. Keep it for provenance; do not plan from it.

**Snapshot:** 2026-08-22 on `main` at `b1bcd37` (`docs: explain Kennel's outcome-first workflow`). The working tree was clean when this handoff was prepared. The configured remote is `git@github.com-personal:Pin4sf/Waldo-Kennel.git`.

## What we are building

Kennel is a local Electron workspace for directing coding agents toward a human-defined **Outcome**, rather than starting with an implementation task. The desired sequence is:

1. Open a project and have its orchestrator perform a read-only orientation.
2. Define an Outcome.
3. Resolve material ambiguity through focused questions.
4. Review deliverables, objective checks, constraints, and agent assignments.
5. Require explicit approval before implementation workers are created.
6. Supervise execution, evidence, review, and human acceptance on the board.

The top-level [README](../README.md) is the current product statement. It is the best source for product intent; most historical architecture/status documentation still uses the inherited “Agent Orchestrator” name and should not be read as Kennel product copy.

## Repository baseline

This repository was introduced in two commits:

| Commit | Meaning |
| --- | --- |
| `245ba4a` — `Kennel Remake initial build` | Imported the existing Agent Orchestrator codebase and introduced the initial Kennel outcome-flow implementation. It is a very large baseline commit (2,480 files, 574,490 insertions). |
| `b1bcd37` — `docs: explain Kennel's outcome-first workflow` | Replaced the inherited product README with Kennel’s outcome-first narrative. |

Treat the codebase as a mature AO foundation being deliberately repurposed, not as a greenfield app. The change so far is primarily a product/workflow layer over the existing daemon, sessions, worktrees, browser, PR, review, and terminal machinery.

## What is implemented now

### Kennel branding and entry experience

- Electron application name, tray/app menus, startup UI, update copy, and much of the renderer copy use **Kennel**.
- The primary empty-board and new-task experience asks the user to describe the result they want to achieve.
- On a Chat-capable project orchestrator, the frontend sends a read-only orientation request if the conversation is empty. The orchestrator is instructed to emit 2–4 `KENNEL_OUTCOME_SUGGESTION:` lines; these populate the new-outcome UI.
- The app retains the AO session model: a project has orchestrator and worker sessions, each with an agent harness, mode, workspace, activity, PR facts, and derived status.

### Outcome intake and plan review

The frontend creates an outcome by posting `outcome: true` to the existing `POST /api/v1/orchestrators/delegate` endpoint. It does **not** create an implementation worker at that point.

- The backend finds or resumes the project orchestrator, renames it to `Outcome: <brief>`, stages optional attachments into that orchestrator workspace, and sends the bounded intake prompt.
- The intake prompt requires model-produced JSON markers for questions and plans:
  - `KENNEL_OUTCOME_QUESTIONS_JSON:`
  - `KENNEL_OUTCOME_PLAN_JSON:`
  - `KENNEL_OUTCOME_PLAN_REVISION:`
  - `KENNEL_OUTCOME_PLAN_APPROVED:`
- The renderer parses those markers defensively and renders either a single-question flow or a plan-review panel.
- The plan review shows deliverables, agent assignment, objective completion checks, constraints, and an orchestrator → workers → outcome graph. The user can approve the exact plan or request a revision.
- Approval is explicit: a normal affirmative chat message is not considered approval. The frontend sends the plan with the approval marker.

### Board status support

- Outcome orchestrators can end an assistant update with `KENNEL_OUTCOME_STATUS: working|needs_you|reviewing|ready_to_merge`.
- Workers use the analogous `KENNEL_WORK_STATUS:` marker.
- The session service uses these only where terminal activity and SCM facts do not provide a stronger truth. Termination, active work, user-input state, and attributed PR facts remain authoritative.

## The most important gap

The durable Outcome model is **not wired into the runtime yet**.

`backend/internal/domain/outcome.go` defines `Outcome`, `OutcomeTask`, their states, and validation. `backend/internal/service/outcome/planning.go` validates an agent-proposed task graph, rejects cycles and unavailable/unknown harnesses, and deterministically assigns an available harness. However:

- there is no `outcomes` or `outcome_tasks` SQLite migration/table/query/store;
- no HTTP API exposes outcomes as first-class resources;
- no session-to-outcome durable association exists;
- approval is represented only in the orchestrator chat transcript/marker, not as a durable, structured approval record;
- after approval, the system asks the model to create workers through the existing agent/CLI mechanisms—Kennel does not yet independently persist, schedule, dependency-gate, or verify the planned task graph;
- the active UI is attached to an `Outcome:`-renamed orchestrator and is primarily shown while the project has no worker sessions. It is not yet a durable multi-worker outcome dashboard.

This is the natural next vertical slice: make Outcome the durable unit of truth while continuing to reuse sessions as execution activity/evidence beneath it.

## Suggested implementation sequence

Do these as small, independently reviewable slices; do not start by replacing the session/lifecycle subsystem.

1. **Persist the outcome contract.** Add new SQLite migrations (never edit existing ones) for outcomes, acceptance criteria/deliverables/checks, outcome tasks, dependencies, task/session links, plan revision, and explicit approval facts. Add queries, run `npm run sqlc`, and keep the durable schema focused on facts rather than display status.
2. **Add an outcome service and API.** Create/load/update an outcome and retrieve a stable read model. Update `backend/internal/httpd/controllers/dto.go`, register named schemas in `apispec/specgen/build.go`, then run `npm run api`. Preserve the CLI/daemon boundary—new CLI behavior must be a thin HTTP client.
3. **Make approval daemon-owned.** Validate the rendered/proposed plan through `service/outcome.BuildPlan`, check the locally available harnesses, record approval transactionally, and spawn only dependency-ready workers. Do not rely solely on the orchestrator honoring prompt text for this safety boundary.
4. **Connect workers and evidence.** Record the session assigned to every outcome task. Derive task/outcome presentation from durable task state plus existing session/PR/CI/review facts; do not store derived board state. Add a receipt/evidence shape for each completion check before declaring an outcome ready.
5. **Expand the UI.** Replace the empty-board-only intake rendering with an outcome detail/board that survives worker creation, project navigation, restart, and agent handoff. Let users inspect the accepted contract, tasks, dependencies, checks, evidence, blockers, and approval history.
6. **Harden orchestration.** Make retries, unavailable harnesses, worker failure, cancellation, plan revision after approval, and partial completion explicit product states. Preserve the existing one-controller/session and dirty-worktree safety rules.

## Important code map

| Concern | Primary locations | Notes |
| --- | --- | --- |
| Outcome domain and graph validation | `backend/internal/domain/outcome.go`, `backend/internal/service/outcome/planning.go` | Present but presently isolated from persistence/API. |
| Outcome submission and prompt gate | `backend/internal/service/session/delegation.go` | `DelegateTask` branches on `Outcome`; outcome mode uses the active project orchestrator rather than spawning a worker. |
| Prompt contract | `backend/internal/session_manager/prompt.go` | Contains the global orchestrator/worker Kennel marker instructions. Keep marker compatibility if changing this. |
| HTTP endpoint/DTO | `backend/internal/httpd/controllers/sessions.go`, `backend/internal/httpd/controllers/dto.go` | `POST /api/v1/orchestrators/delegate` carries the `outcome` boolean today. |
| Status derivation | `backend/internal/service/session/status.go` | Marker statuses are fallbacks, not the source of truth. |
| New-outcome composer | `frontend/src/renderer/components/TaskComposer.tsx` | Requests suggestions and submits outcome intake. |
| Coordination parser | `frontend/src/renderer/lib/outcome-coordination.ts` | Strictly parses marker JSON and builds response messages. |
| Plan/questions UI | `frontend/src/renderer/components/OutcomeIntakePanel.tsx`, `frontend/src/renderer/components/OutcomeOrchestrationGraph.tsx` | Current UI is conversation-driven. |
| Board integration | `frontend/src/renderer/components/SessionsBoard.tsx` | Decides when to show the outcome panel and sends user response markers. |
| Product UI primitives | `packages/product-ui/` | Shared presentational components used by desktop and related clients. |

## Foundation that should be retained

Kennel inherits a working local AO architecture. Preserve these boundaries:

- **Daemon:** Go loopback HTTP service; its primary listener stays unauthenticated on `127.0.0.1`. The opt-in authenticated LAN listener is the only supported non-loopback listener.
- **CLI:** a thin Cobra/HTTP client. Do not make CLI commands read SQLite, manage worktrees, or start harnesses directly.
- **Session:** one committed controller/interface mode at a time. Chat and TUI handoff must retain the controller-generation fencing and drain/interrupt rules.
- **Storage:** SQLite facts and trigger-generated `change_log` CDC. Do not hand-edit sqlc generated files or emit parallel manual CDC events.
- **Status:** derive display state from durable session, PR, CI, and review facts at read time. Do not turn board columns into stored status.
- **Worktrees:** workers receive isolated worktrees/branches; do not force-delete a dirty registered worktree.
- **API:** source is Go DTOs and API-spec registration; OpenAPI and frontend schema are generated artifacts that must be committed together.
- **App data:** all Kennel runtime data stays under `~/.kennel` (overridable via `KENNEL_DATA_DIR`/`KENNEL_RUN_FILE`), including Electron `userData`. The in-workspace `.ao/attachments` path is a separate compatibility seam decided independently by issue #37.

Read [architecture.md](architecture.md), [backend-code-structure.md](backend-code-structure.md), [AGENTS.md](../AGENTS.md), and the relevant LAN/reviewer ADR before changing their respective boundaries.

## Branding and documentation debt

The repository identity is not fully migrated. This is intentional unfinished work, not evidence that the product is still called AO.

- Code/module paths, API namespaces, CLI name (`ao`), storage/environment names, package name `@aoagents/product-ui`, remote historical URLs, updater/release configuration, and many tests still use Agent Orchestrator/AO.
- Documentation such as [STATUS.md](STATUS.md), [architecture.md](architecture.md), [development.md](development.md), [DESIGN.md](../DESIGN.md), and [CONTEXT.md](../CONTEXT.md) contains inherited product descriptions and sometimes older names such as ReverbCode. Preserve their technical guidance but do not copy their product positioning into new Kennel-facing content.
- The frontend metadata still points to historical AO homepage/repository URLs. Release identity, signing, updater feed, bundle identifiers, remote module paths, and user-data migration need a deliberate migration plan; do not casually search-and-replace them.
- The app version remains `0.10.3`, inherited from the source project.

Before a public release, create a dedicated, audited branding/distribution plan covering package/module identity, updater artifacts, migration of existing local data, telemetry naming, documentation, licenses/attribution, and macOS signing/notarization. Keep that work separate from Outcome persistence.

## Development and verification

Prerequisites are Go 1.25.7+ and Node 20.19+ (see [development.md](development.md)). The usual commands are:

```sh
# targeted Outcome checks
cd backend && go test ./internal/service/session ./internal/service/outcome
npm --prefix frontend test -- --run \
  src/renderer/lib/outcome-coordination.test.ts \
  src/renderer/components/OutcomeIntakePanel.test.tsx

# broad checks
npm run frontend:typecheck
npm run product-ui:check
npm run lint
```

The targeted backend and frontend Outcome tests above passed for this snapshot. No full-suite, packaging, mobile, or manual Electron acceptance run was performed as part of preparing this document.

When changing the API, run `npm run api` and commit both `backend/internal/httpd/apispec/openapi.yaml` and `frontend/src/api/schema.ts`. When changing queries/schema, add a migration, update queries, run `npm run sqlc`, and commit generated output. For visible frontend changes, use `ao preview [url]` from the relevant agent session to inspect the result in the desktop Browser panel.

## Handoff checklist for the next agent

1. Read this file, [README.md](../README.md), [AGENTS.md](../AGENTS.md), and the architecture docs relevant to the proposed change.
2. Confirm `git status --short` before modifying anything; preserve unrelated work.
3. State whether the task is Outcome-product work, AO-foundation work, or a deliberate branding/release migration. Keep those scopes separate.
4. For Outcome runtime work, start from the persistence/API slice above; do not encode more critical state in chat markers.
5. Run focused tests first, then the relevant project gate. Report exactly what was and was not verified.

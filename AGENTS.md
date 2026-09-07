# AGENTS.md

Operational authority for coding agents working in Waldo Kennel. Read this file before changing product ontology, daemon behavior, storage, provider adapters, scheduling, recovery, or Work UI.

## Canonical read order

For kernel/Work implementation, read in this order and stop when you have enough context for the task:

1. `AGENTS.md` — repository rules and non-negotiable boundaries.
2. `docs/product/kennel-v1-product-architecture.md` — canonical product/kernel ontology and user-facing hierarchy.
3. `docs/adr/0008-responsibility-composition-and-workunit-execution-dag.md` — responsibility decomposition vs execution decomposition.
4. `docs/adr/0009-workunit-scheduling-workspace-leases-and-effect-fencing.md` — scheduler, workspace custody, concurrency, effects, and recovery.
5. `docs/STATUS.md` — what is actually implemented on `beta` versus accepted target behavior.
6. `docs/product/kennel-build-program.md` and `docs/superpowers/plans/2026-09-04-kennel-builds-kennel.md` — implementation order and verification gates.
7. `docs/architecture.md` — chassis/package/lifecycle technical reference when needed.
8. `docs/research/2026-09-04-kernel-runtime-reference-index.md` — provider protocols and benchmark runtime patterns when touching adapters, scheduler, workspace, receipts, or external ingress.

The two current Work specifications are implementation companions:

- `docs/superpowers/specs/2026-08-25-work-control-plane-canonical-flow-design.md`
- `docs/superpowers/specs/2026-08-25-work-experience-screen-interaction-spec.md`

Older dated plans, handoffs, prototypes, Home/Memory research, and historical ADRs do **not** override the authority chain above. Load them only when a task explicitly targets that lane or the canonical docs link to them for provenance.

## Product invariant

> **The user manages Outcomes. Kennel manages the sessions required to make those Outcomes true.**

Provider sessions are execution machinery. They are never the durable user-owned responsibility.

The canonical Work lineage is:

```text
Project
├── ProjectBriefRevision*                 persistent context; never “done”
└── Outcome*                              finite responsibility
    └── ContractRevision
        ├── DecompositionRevision         when responsibility splits
        │   └── Contributing Outcome*     each owns its own Contract/Plan
        └── PlanRevision                  for a direct Outcome
            └── WorkUnit DAG
                └── Attempt*
                    └── AgentSessionRef
                        └── SessionReceipt
                └── WorkUnitReceipt
            └── EvidenceItem*
                └── VerificationRun*
                    └── AcceptanceDecision
```

An Outcome is exactly one of two v1 shapes:

- **Direct:** owns a `PlanRevision` whose execution topology is a bounded WorkUnit DAG.
- **Decomposed:** owns a `DecompositionRevision` and contributing Outcomes; the parent does not also own direct WorkUnits in v1.

The distinction is mandatory:

> **Create another Outcome when responsibility splits. Create another WorkUnit when execution splits.**

ADR 0008 supersedes ADR 0007 only where ADR 0007 treated Outcome composition and a WorkUnit DAG as competing mechanisms. ADR 0007 remains authoritative for criterion-bound contribution, authority narrowing, stale-parent handling, independent child proof, user-only acceptance, and the v1 composition-depth cap.

## Waldo, Kennel, and providers

The responsibility boundary is:

```text
Waldo proposes / interprets / recommends
              ↓
Kennel validates / schedules / records / enforces
              ↓
Providers execute
```

Models may propose contracts, plans, decompositions, context candidates, or next actions. Deterministic daemon policy decides whether a proposal is structurally valid, authorized, runnable, recoverable, and truthfully representable.

Never move authority, acceptance, dependency validation, idempotency, recovery fencing, effect reconciliation, or evidence binding into model prompts.

## Supported provider surface

PR #92 established exactly five first-class provider identities for new Kennel work:

- Codex
- Claude Code
- OpenCode
- Cursor
- Pi

Historical provider IDs may remain readable for migration/recovery compatibility, but they are not active product providers.

Provider admission is capability-derived and machine-aware. Do not add provider-name folklore such as “Claude for planning” or hidden fallback such as “if anything fails, use Codex.” An explicit user provider choice either admits truthfully or fails with a reason.

Current role capability may lag the five-provider identity surface. `docs/STATUS.md` is the source of truth for what has passed conformance **today**. New coordinator, reviewer, switch, resume, fork, approval, or external-ingress roles require provider-specific conformance tests before admission.

Across providers, “switch” means a new Attempt with an attributed continuation/handoff packet unless a provider-native protocol genuinely supports the requested continuation. Never claim hidden provider state was losslessly migrated.

## Workspaces, concurrency, and effects

The target scheduler is WorkUnit-scoped, not Project-serialized. ADR 0009 is authoritative.

Core rules:

- read/reason-only work may run concurrently;
- write-capable WorkUnits require isolated `WorkspaceLease`s;
- Git-backed Projects use worktrees as the v1 parallel-write isolation boundary;
- a new Project may initialize Git before parallel writes are enabled;
- a non-Git folder may support understanding/research/single-writer work, but must not pretend it has worktree isolation;
- Git integration/merge is a separate controlled boundary;
- consequential external effects such as PR mutation, deployment, sending, or external API writes are separately authorized and fenced;
- an unknown prior effect or ambiguous surviving Attempt blocks duplicate execution until reconciliation;
- cleanup failure leaves inspectable debris state; never delete unknown user work to make the UI look clean.

Do not render a Mission Graph that visually implies concurrency before the daemon can truthfully schedule that concurrency.

## User-facing Work hierarchy

The canonical navigation is:

```text
Waldo Island
    ↓
Global Work / Project Board or List       Outcomes
    ↓
Mission Control                           Contract | Graph
    ↓
WorkUnit                                  execution node
    ↓
Attempt / Session Inspector               technical escape hatch
```

Board/List show top-level active Outcomes by default, not sessions. Mission Control represents one selected Outcome. Session Inspector contains provider transcript/terminal/diff/browser/runtime detail and should not be required for ordinary supervision.

Only the user creates an `AcceptanceDecision`. Provider completion, process exit, commits, PRs, green checks, or verifier success may move an Outcome toward `Ready for Review`; none accepts it.

## Durable state and context rules

- SQLite in the daemon is the canonical writer for Work responsibility and execution facts.
- Frontend, Island, CLI, MCP/provider shims, and notifications are clients/projections of daemon state.
- Do not parse model prose or transcript markers to determine canonical Outcome/Plan/approval state.
- Agent-authored structured proposals call daemon APIs and pass the same validation as human-authored proposals.
- Provider-native subagents remain provider-native by default. Promote them to Kennel WorkUnits only when they need independent scheduling, authority, workspace/effect boundaries, retry semantics, artifacts, dependencies, or proof.
- Sessions produce bounded receipts. Do not grow one immortal Project transcript or replay every historical session into every prompt.
- Project learnings are provenance-bearing candidates until governed promotion; raw session text is not Project truth.
- External activity is explicitly `Governed`, `Observed`, or `Untracked`. Never fuzzy-auto-attach external work to an Outcome.
- `unknown` / `unconfirmed` is a valid runtime state. Absence of a process probe or provider event is not proof of completion or death.

## Repository layout

- `backend/` — Go daemon, services, domain, storage, runtime/workspace/provider adapters, CLI, recovery, and tests.
- `frontend/` — Electron + React supervisor using generated daemon contracts. Keep it thin; orchestration authority stays in the daemon.
- `docs/` — current authority, ADRs, specs, implementation program, and scoped future research.
- `packages/kennel-island/` — ambient projection of the same canonical daemon/event state.
- `test/` — external smoke/e2e assets.
- `.github/workflows/` — CI definitions.

### Code entry points

- domain vocabulary/invariants: `backend/internal/domain/`
- service read/write boundaries: `backend/internal/service/`
- ports: `backend/internal/ports/`
- provider adapters/registries: `backend/internal/adapters/`
- HTTP controllers/DTOs: `backend/internal/httpd/controllers/`
- SQLite migrations/queries/store: `backend/internal/storage/sqlite/`
- lifecycle/recovery/runtime: `backend/internal/lifecycle/` and relevant runtime packages
- frontend daemon client/projections: `frontend/src/`

## Required commands

From repository root unless noted:

```bash
npm run bootstrap
npm run lint
npm run frontend:typecheck
npm run sqlc
npm run api
npx @redwoodjs/agent-ci run --all
```

Backend checks:

```bash
cd backend
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

Frontend checks:

```bash
cd frontend
npm run typecheck
npm run build
```

For user-visible frontend work, also run the real-daemon preview path described in the repo docs and visually verify the changed flow. Do not claim a fixture-only screen proves daemon behavior.

## Hard engineering boundaries

- Primary daemon listener remains loopback `127.0.0.1`; preserve the separately governed opt-in LAN listener rules in ADR 0001.
- CLI is a thin daemon client. Do not open SQLite or spawn providers directly from CLI commands.
- Add SQLite migrations; never rewrite already-merged migrations.
- Edit sqlc source queries/schema, never generated `backend/internal/storage/sqlite/gen/*` by hand.
- CDC comes from SQLite triggers into `change_log`; do not invent a parallel manual event authority without an ADR.
- API source is code-first; regenerate OpenAPI and frontend TypeScript contracts together after DTO/route changes.
- Do not store display-only Outcome/session status when it can be derived from durable facts.
- Do not force-delete dirty registered worktrees.
- Do not create duplicate Attempts after restart/retry because a provider appears quiet.
- Do not treat verification as acceptance.
- Do not allow child/contributing authority to exceed the parent Contract ceiling.
- Do not let models bypass DAG cycle checks, capability checks, stale-revision checks, evidence requirements, idempotency, or effect fences.
- All application state remains under `~/.kennel` (or documented overrides), including Electron `userData`.

## API contract changes

Daemon API contracts are generated from source. When changing request/response shapes:

1. edit `backend/internal/httpd/controllers/dto.go` and operation/spec sources;
2. run `npm run api`;
3. commit generated `backend/internal/httpd/apispec/openapi.yaml` and `frontend/src/api/schema.ts` together;
4. run HTTP/spec parity tests.

When changing SQLite contracts, update migrations/queries and run `npm run sqlc`.

## PR and implementation discipline

- Product/kernel work branches from latest `beta` and targets `beta`.
- Use issue-sized or slice-sized PRs. Do not implement the entire kernel program as one uncontrolled patch.
- Before a slice, read `docs/STATUS.md` and the relevant ADR/spec; produce a current-code delta map.
- Run baseline tests before changing a load-bearing subsystem.
- Implement backend truth before frontend projections that depend on it.
- Verify narrow tests first, then repo-wide gates for touched areas.
- Stop rather than weakening an invariant or hiding a failing test to finish a long run.
- Use conventional commits.
- Document intentional omissions and provider capability gaps explicitly.

The dogfood objective is not “finish a Kanban.” It is to use Kennel to implement real Kennel work while the user interacts primarily with Outcomes and only opens provider Sessions when deep inspection is needed.

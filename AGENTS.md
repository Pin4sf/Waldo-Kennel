# AGENTS.md

Operational authority for coding agents working in Waldo Kennel. Read this file before changing product ontology, daemon behavior, storage, provider adapters, scheduling, recovery, routing, intelligence, or Work UI.

## Canonical read order

For kernel/Work implementation, read in this order and stop when you have enough context for the task:

1. `AGENTS.md` — repository rules and non-negotiable engineering boundaries.
2. `docs/adr/0010-outcome-first-control-plane-and-session-subordination.md` — Outcome-first authority and session subordination.
3. `docs/adr/0011-go-control-plane-and-non-authoritative-intelligence.md` — Go control plane and intelligence boundary.
4. `docs/adr/0012-waldo-reasons-with-the-owners-model.md` — Waldo reasons with the owner's model; there is no deterministic floor.
5. `docs/adr/0015-contract-bound-interactive-planning.md` — durable pre-execution planning conversation and authority boundary.
6. `docs/product/kennel-v1-product-architecture.md` — canonical product/kernel ontology and user-facing hierarchy.
7. `docs/product/2026-09-08-outcome-control-plane-mvp-reset.md` — current MVP architecture reset.
8. `docs/product/2026-09-08-pre-execution-ux-and-tech-debt-audit.md` — mandatory UX reuse / debt guardrails.
9. `docs/STATUS.md` — implemented runtime truth versus accepted target behavior.
10. `docs/adr/0008-responsibility-composition-and-workunit-execution-dag.md` — responsibility decomposition versus execution decomposition.
11. `docs/adr/0009-workunit-scheduling-workspace-leases-and-effect-fencing.md` — scheduler, workspace custody, concurrency, effects, and recovery.
12. `docs/superpowers/plans/2026-09-08-outcome-control-plane-mvp-reset.md` — current implementation order and verification gates.
13. `docs/architecture.md` and `docs/research/2026-09-04-kernel-runtime-reference-index.md` when touching lower-level chassis/provider/runtime details.

The current Work interaction specs remain implementation companions:

- `docs/superpowers/specs/2026-08-25-work-control-plane-canonical-flow-design.md`
- `docs/superpowers/specs/2026-08-25-work-experience-screen-interaction-spec.md`

Older dated plans, handoffs, prototypes, Home/Memory research, and historical ADRs do **not** override the authority chain above.

## Product invariant

> **The user manages Outcomes. Kennel manages the execution required to make those Outcomes true.**

Provider sessions are subordinate execution resources, never the durable user-owned responsibility.

Canonical lineage:

```text
Project
├── ProjectBriefRevision*                 persistent context; never “done”
└── Outcome*                              finite responsibility
    └── ContractRevision
        ├── PlanningSession*                 bounded pre-execution discussion
        │   └── PlanningTurn* / IntelligenceRun*
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

An Outcome is one of two v1 shapes:

- **Direct:** owns a `PlanRevision` whose execution topology is a bounded WorkUnit DAG.
- **Decomposed:** owns a `DecompositionRevision` and contributing Outcomes; the parent does not also own direct WorkUnits in v1.

> **Create another Outcome when responsibility splits. Create another WorkUnit when execution splits.**

Only the user creates the final `AcceptanceDecision`. Provider completion, process exit, commits, PRs, green checks, or verifier success may move an Outcome toward review; none accepts it.

## Production-slice engineering standard

The Outcome-control-plane MVP is the **first production vertical slice**, not a disposable proof-of-concept.

> **Reduce scope by limiting capabilities, not by introducing temporary domain models, fake authority, duplicate execution paths, or throwaway persistence.**

Production-shaped MVP reductions are encouraged:

- concurrency budget `1` behind a real scheduler boundary;
- one intelligence adapter behind a provider-neutral port;
- simple deterministic routing behind a real routing boundary;
- a bounded capability vocabulary;
- simple verification mechanisms behind canonical Evidence/Verification/Acceptance state.

Do **not** use MVP scope as justification for:

- a temporary task/session ontology that must later be replaced by Outcome/Plan/WorkUnit;
- a fake scheduler implemented as unrelated ad-hoc task loops when the canonical WorkUnit scheduler boundary already exists;
- provider-specific orchestration logic in Outcome services;
- duplicated storage writers for old/new semantics when one canonical writer can persist the current domain object;
- frontend-owned lifecycle truth;
- treating provider completion as Outcome completion;
- knowingly creating schema that must be replaced immediately after dogfood.

Before adding a shortcut, ask:

> **Does this reduce MVP scope, or postpone architectural correctness?**

Reducing scope is acceptable. Postponing architectural correctness requires an explicit documented reason and must not silently become the production path.

## Hardcode invariants, not mechanisms

Core/domain code must encode stable product and safety invariants. Provider versions, CLI flags, temporary scheduler limitations, UI wording, and unproven heuristics are implementation mechanisms and should remain behind ports/policy/adapters.

Examples:

- hard invariant: pre-authorization intelligence cannot create execution authority or unapproved persistent effects;
- flexible mechanism: Codex sandbox/approval flags or another provider's equivalent;
- hard invariant: an Attempt consumes the approved WorkUnit execution semantics;
- flexible mechanism: explicit model, provider default, or future provider-supported model-selection modes;
- hard invariant: WorkUnit dependencies are acyclic and respected;
- flexible mechanism: MVP serial execution versus later parallel scheduling;
- hard invariant: every required Contract criterion is covered by the compiled Plan;
- flexible mechanism: model-facing aliases versus internal stable criterion IDs;
- hard invariant: consequential external effects require explicit authority/fencing;
- flexible mechanism: provider-specific network/tool implementation.

Do not make domain behavior depend on an exact provider version. Runtime/provider conformance is capability-based. A version or binary fingerprint may be used only to cache/re-run conformance when an installation changes.

Do not encode provider-specific permission modes such as `PermissionModeAuto`/`Never` as product laws. Enforce the required effect boundary and let each adapter implement it truthfully.

Do not interpret “read-only intelligence” as “zero filesystem writes anywhere.” Provider-private scratch/cache state may be acceptable. The invariant is that pre-authorization intelligence cannot mutate the Project/worktree, canonical Kennel responsibility state, user data outside its allowed context, or external systems without authority.

Similarly, do not model intelligence as `network=false`. Inference/control traffic may be necessary; arbitrary tool-driven external effects are not implicitly authorized.

## Waldo, Kennel, intelligence, scheduler, router, and harnesses

Long-term responsibility boundaries must remain clear:

```text
Waldo / user / API
        ↓
Outcome authority
Contract → Plan → approval
        ↓
Kennel control plane
validate / route / schedule / reconcile / verify
        ↓
Harness adapters
Codex / Claude Code / OpenCode / Cursor / Pi / future providers
        ↓
Provider runtimes
```

Responsibilities:

- **Waldo / intelligence** proposes, interprets, recommends, explains.
- **Control plane** validates, versions, authorizes, records, reconciles, and enforces.
- **Router** chooses among already-normalized admissible execution candidates.
- **Scheduler** decides when an approved WorkUnit may run.
- **Harness adapter** translates normalized Kennel execution semantics into one provider/runtime and reports truthful capabilities/provenance.
- **Provider runtime** executes bounded authorized work.

Do not collapse these boundaries because one provider SDK makes it convenient.

### Intelligence providers

`IntelligenceProvider` must remain provider-neutral and non-authoritative.

- Do not type intelligence provenance as `AgentHarness`; direct APIs and future Waldo-hosted intelligence are not execution harnesses.
- Prefer an opaque intelligence-provider identifier plus explicit provenance/capability fields rather than a closed core enum that must change for every new adapter.
- Requested/effective provider/model provenance should be recorded when known; unknown is valid and must not be fabricated.
- Intelligence output is structured proposal material. It does not approve Plans, create Attempts, grant capabilities, accept Outcomes, or mutate canonical execution state.
- Provider-specific conformance belongs inside the adapter/conformance layer, not Outcome/Plan services.
- If the configured reasoning adapter cannot satisfy the required effect boundary, fail with an actionable reason. ADR 0012 removed the deterministic/manual proposal floor; any owner-selected adapter change must be explicit. Do not silently widen authority.
- ADR 0012 requires the owner's configured reasoning credential for model-backed intake/planning. It powers the intelligence plane only unless a future ADR explicitly changes that. Secrets never become canonical domain data.

### Harness providers

Current first-class execution-provider identities for new work are Codex, Claude Code, OpenCode, Cursor, and Pi. This is current product inventory, not a license to branch core policy on provider names.

Provider admission is capability-derived and machine-aware. Do not add provider folklore such as “Claude for planning,” “Codex fallback,” or provider-specific routing branches in control-plane policy.

A harness should evolve toward normalized support for identity/readiness, authentication state, roles, model semantics, execution capabilities, structured-control support, spawn/resume/cancel/observe, effective model, usage/cost, terminal state, and artifacts/evidence. The MVP does not need every capability, but new concerns should go behind harness/port boundaries rather than into Outcome services.

## Plan, WorkUnit, routing, authority, and scheduling

### Canonical Plan shape

Do not hardcode `PlanRevision` to exactly one WorkUnit merely because the MVP scheduler is serial. Canonical direct Plans should support a bounded WorkUnit dependency graph. The MVP may execute that graph with concurrency `1`.

Dependency correctness must not depend on JSON/list order. Validate graph references/cycles and derive a deterministic topological execution order. This lets the same canonical Plan support serial execution now and parallel scheduling later.

Bound model-generated Plan size to prevent pathological output, but treat exact limits (for example 16 WorkUnits) as named operational policy, not laws of Outcomes.

### Contract coverage

Every required Contract criterion must be represented in the compiled canonical Plan. The model-facing schema may use stable aliases such as `C1`, `C2`; the daemon maps them to internal criterion identity and rejects incomplete coverage.

### Capability authority

Least privilege is mandatory.

Do not grant every WorkUnit a universal read/write/exec trio by default. Derive the minimum capabilities required by WorkUnit semantics, Contract constraints, and deterministic Kennel policy, then intersect with the applicable authority ceiling.

Models may suggest required operations/capabilities, but model output is not canonical authority truth. The control plane deterministically derives/validates required capabilities, mandatory stop conditions, effect restrictions, and verification obligations.

Avoid using free-form UI text as canonical policy identity. Prefer stable semantic codes/types for policy and render human wording separately when that distinction becomes meaningful.

### Routing

Keep routing deterministic and explainable until real execution telemetry justifies more sophisticated scoring.

MVP default policy should be conceptually:

1. hard-gate role/readiness/capabilities/model semantics;
2. use an admissible explicit user preference when possible;
3. otherwise choose a deterministic admissible candidate and explain why the preference was unavailable;
4. if none are valid, return `NO_VALID_CANDIDATE` / Action Required and launch nothing.

Do not introduce magic score thresholds or invented quality weights without product evidence/telemetry. Soft scoring may be added later from measured success, verification quality, latency, cost, retry rate, task type, and user override signals.

Explicit preferred-model support is candidate/provider-local unless the Contract truly requires one global model identity. A preferred model name must never accidentally disqualify unrelated providers simply because their catalogs use different model names.

Routing recommendation and persisted WorkUnit execution binding must agree exactly before a Plan can be approved. Approval freezes the stored binding; Attempt admission never re-reads mutable Project provider/model preferences to reinterpret it.

`provider_default` is a valid exact **selection semantic** even when no concrete model name is frozen. Record the effective model at execution time when the provider reports it; never fabricate it.

### Scheduler

The scheduler boundary must be real even when MVP concurrency is `1`.

- scheduler chooses runnable approved WorkUnits;
- dependency and custody rules remain canonical;
- restart/retry must not duplicate Attempts;
- unknown/ambiguous liveness blocks unsafe duplicate execution;
- later WorkspaceLease/concurrency support should extend this boundary rather than replace the MVP task model.

Do not render fake parallelism before the daemon can truthfully enforce it.

## User-facing Work hierarchy and UX reuse

The normal Work experience remains:

```text
Enter → Understand → Decide & Authorize → Act & Observe → Prove & Close
```

These are UI projections of canonical daemon facts, not a second persisted lifecycle.

Normal navigation:

```text
Waldo Island
    ↓
Global Work / Project Board or List       Outcomes
    ↓
Mission Control                           one selected Outcome
    ↓
WorkUnit / Attempt                        execution detail
    ↓
Session Inspector / terminal              technical drill-down
```

Preserve and evolve the existing Work surfaces rather than creating parallel Contract/Plan/Mission wizards or another Work shell. `WorkShell` remains the persistent Work chrome. Mission Control is a projection, not a new domain table/state machine.

Board/List show Outcomes by default, not sessions. Session Inspector/provider transcript/terminal/diff/browser detail is an explicit technical escape hatch and should not be required for ordinary supervision.

Do not persist `current_stage = understand/act/prove` when the stage can be derived from Contract/Plan/Attempt/Evidence/Verification/Acceptance facts.

Preserve current design system, i18n, keyboard/focus behavior, loading/error/recovery states, and accessibility. Hide/disable controls that are not truthful rather than leaving dead or fake interactions.

## Durable state and persistence

- SQLite in the Go daemon is canonical for responsibility/execution facts.
- Frontend, Island, CLI, provider shims, and notifications are clients/projections.
- Do not parse model prose/transcript markers to determine canonical Outcome/Plan/approval state.
- `unknown` / `unconfirmed` is valid runtime truth; absence of a process probe/event is not proof of completion/death.
- Agent-authored structured proposals pass the same deterministic validation as human-authored proposals.
- Sessions produce bounded receipts; do not build one immortal Project transcript or replay all history into every prompt.
- External activity is explicitly `Governed`, `Observed`, or `Untracked`; never fuzzy-auto-attach it to an Outcome.

### Canonical writers

Prefer one production write path per canonical aggregate operation. When the domain evolves, evolve the canonical writer rather than permanently duplicating methods such as `CreateX` and `CreateXWithNewSemantics` that copy the same transaction.

Historical rows need read compatibility; that does not imply new production code needs a first-class legacy writer.

Raw SQL may be temporarily necessary while generated code cannot be refreshed, but do not leave duplicate raw-SQL and sqlc implementations as permanent competing sources of truth. Edit sqlc source queries/schema, regenerate, then converge on the canonical path.

### Migrations

- Add migrations; never rewrite migrations that have reached the shared `beta` release lineage.
- An unmerged feature-branch migration may be corrected before its first beta merge when necessary to avoid permanently shipping a bad schema, but once merged treat its number/content as immutable shipped history.
- Keep the migration ledger accurate in the same change.
- Preserve historical readability; do not synthesize new authority/provenance for old rows.
- Enforce important write-once/cross-row integrity in domain/service and schema where practical, but follow repository conventions rather than blindly introducing a new NULL/sentinel style.

## Testing standard

Test observable behavior and invariants, not implementation-shaped proxies.

Prefer names that identify the operation and behavior, for example:

```text
TestBindIntakeAnalysisRequestIntelligenceRun_RejectsRebind
TestStartAttempt_UsesApprovedExecutionBinding
TestRouteExecution_RejectsUnsupportedPreferredModelOnlyForThatCandidate
```

Table/subtests are preferred when several cases exercise one operation.

A test name must match the actual failure path it proves. For write-once behavior, create two otherwise-valid entities and prove the second bind fails because of the write-once fence—not because the second entity does not exist.

Avoid reflection tests that merely assert a struct lacks fields named `AttemptID`, `Secret`, `APIKey`, etc. They provide weak architectural/security guarantees and are easy to bypass accidentally. Prefer behavioral tests: canary secrets are not persisted/logged, terminal provenance cannot change, wrong-lineage runs cannot bind, unauthorized effects cannot occur.

For regression bugs, use real red-green evidence when possible: demonstrate the test fails with the fix removed and passes with the fix applied.

Do not add tests for getters/trivial implementation details unless they protect a meaningful contract, migration, concurrency, authority, compatibility, or regression boundary.

## Code quality and comments

Optimize for a small, legible systems codebase rather than translating every architecture paragraph into another type/helper/comment.

- Prefer the smallest abstraction that preserves the production boundary.
- Avoid duplicate domain models or storage paths that encode the same truth.
- Avoid speculative generic frameworks and policy DSLs before a second real use case exists.
- Avoid magic heuristics/constants; use named operational policy when a bound is required.
- Comments should explain **why**, especially authority ordering, concurrency/recovery hazards, migration history, provider quirks, and non-obvious safety boundaries.
- Do not narrate obvious getters/setters or repeat the same ADR sentence in every layer.
- Keep provider-specific flags/quirks inside provider adapters.
- Core services should read like product/control-plane policy, not CLI invocation code.

## Workspaces, concurrency, and effects

ADR 0009 remains authoritative for the full target scheduler.

- read/reason-only work may run concurrently when safely supported;
- write-capable WorkUnits require truthful isolation/custody;
- Git-backed Projects use worktrees as the v1 parallel-write isolation boundary;
- non-Git folders must not pretend they have worktree isolation;
- Git integration/merge is a separate controlled boundary;
- consequential external effects such as PR mutation, deployment, sending, or external API writes require separate authorization/fencing;
- an unknown prior effect or ambiguous surviving Attempt blocks duplicate execution until reconciliation;
- cleanup failure leaves inspectable debris; never delete unknown user work to make the UI clean.

## Repository layout

- `backend/` — Go daemon, services, domain, storage, runtime/workspace/provider adapters, CLI, recovery, and tests.
- `frontend/` — Electron + React supervisor using generated daemon contracts. Keep it thin; orchestration authority stays in the daemon.
- `docs/` — current authority, ADRs, specs, implementation plans, and scoped research.
- `packages/kennel-island/` — ambient projection of canonical daemon/event state.
- `test/` — external smoke/e2e assets.
- `.github/workflows/` — CI definitions.

Key code entry points:

- domain vocabulary/invariants: `backend/internal/domain/`
- service read/write boundaries: `backend/internal/service/`
- ports: `backend/internal/ports/`
- provider adapters/registries: `backend/internal/adapters/`
- HTTP controllers/DTOs: `backend/internal/httpd/controllers/`
- SQLite migrations/queries/store: `backend/internal/storage/sqlite/`
- lifecycle/recovery/runtime: `backend/internal/lifecycle/` and relevant runtime packages
- frontend daemon client/projections: `frontend/src/`

## Hard engineering boundaries

- Keep the Go daemon as canonical local control plane unless a future ADR explicitly changes it.
- Primary daemon listener remains loopback `127.0.0.1`; preserve the separately governed opt-in LAN listener rules in ADR 0001.
- CLI is a thin daemon client; do not open SQLite or spawn providers directly from CLI commands.
- CDC comes from SQLite triggers into `change_log`; do not invent a parallel manual event authority without an ADR.
- API source is code-first; regenerate OpenAPI/frontend TypeScript contracts together after DTO/route changes.
- Do not store display-only Outcome/session/stage state when it can be derived from durable facts.
- Do not force-delete dirty registered worktrees.
- Do not create duplicate Attempts after restart/retry because a provider appears quiet.
- Do not treat verification as acceptance.
- Do not allow child/contributing authority to exceed the parent Contract ceiling.
- Do not let models bypass graph-cycle checks, capability checks, stale-revision checks, evidence requirements, idempotency, routing admission, or effect fences.
- All application state remains under `~/.kennel` (or documented overrides), including Electron `userData`.

## API and generated contracts

When changing request/response shapes:

1. edit source DTO/operation/spec code;
2. run `npm run api`;
3. commit generated `backend/internal/httpd/apispec/openapi.yaml` and `frontend/src/api/schema.ts` together;
4. run HTTP/spec parity tests.

When changing SQLite contracts, update migrations/queries and run `npm run sqlc`. Never hand-edit `backend/internal/storage/sqlite/gen/*`.

## Required verification

From repository root unless noted:

```bash
npm run bootstrap
npm run lint
npm run frontend:typecheck
npm run sqlc
npm run api
npx @redwoodjs/agent-ci run --all
```

Backend:

```bash
cd backend
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

Frontend:

```bash
cd frontend
npm run typecheck
npm run build
```

For user-visible Work changes, run the real-daemon Electron/browser path against an isolated profile and disposable real repository. Fixture-only/static rendering is not runtime proof.

No completion, merge-readiness, test-pass, or bug-fix claim without fresh verification evidence. If the environment cannot run a required gate, report it as unverified rather than inferred.

## PR and implementation discipline

- Product/kernel work branches from latest `beta` and targets `beta`.
- Keep Outcome-control-plane work on its integration feature branch until schema/domain/backend/UI/runtime behavior is coherent and verified; do not use `beta` as a dumping ground for partially stabilized migrations.
- Before a slice, read `docs/STATUS.md` and the relevant ADR/spec and map current code to the target.
- Implement backend truth before frontend projections that depend on it.
- Verify narrow tests first, then repo-wide gates for touched areas.
- Stop rather than weakening an invariant or hiding a failing test to finish a long run.
- Use conventional commits.
- Document intentional omissions and provider capability gaps explicitly.
- Do not merge automatically; merge only after explicit request and current verification evidence.

The dogfood objective is not “finish a Kanban.” It is to use Kennel to implement real Kennel work while the user interacts primarily with Outcomes and only opens provider Sessions when deep inspection is needed.

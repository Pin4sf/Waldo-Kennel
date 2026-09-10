# Kennel status

**Current integration:** PR #101 fresh Outcome-first foundation, including the Wednesday L2–L4 code, direct WorkUnit graph, retention/proof corrections and fresh-start entry. Consult Git for the current beta SHA; the dated inventories below are historical evidence, not current remote-state assertions.
**Current objective:** finish the usable Outcome Continuity loop on this foundation.
**Team boundary:** [fresh Kennel beta and Island integration](product/2026-09-10-fresh-kennel-beta-boundary.md).
**Execution authority:** [completion handoff](superpowers/plans/2026-09-10-luna-kennel-work-completion.md).

## Fresh-start checkpoint — 2026-09-10

Work is the default root destination. Island startup is opt-in with `KENNEL_ENABLE_ISLAND=1`; its team should build on daemon Outcome projections. AO commit-author inference and old gitignore ownership adoption are removed; fresh profiles are the supported target.

Exact check scopes and outside-data read confinement are enforced. Proof finalization checks a committed append-only generation, including owner corrections, rather than evidence timestamps. Dependent WorkUnits fail closed with `UPSTREAM_MATERIALIZATION_UNAVAILABLE` until their inputs can actually be provisioned. This is a foundation checkpoint, not launch acceptance or a complete execution loop.

Remaining code: C-13 materialization, checks/evidence/run-intent/rework integration, full Board/Mission supervision, supplied-document wiring and durable delivery. Packaged and real-provider journeys remain separate acceptance gates.

## Implementation inventory — verified 2026-09-09

Fresh remote check: `origin/beta` is `9396c3844`. The Wednesday branch at `c83684c11` is five commits ahead, with no remote divergence. L2 (`0b867790c`), L3 (`684b9c0a9`), L4 (`170230bf0`) and follow-up fixes (`d617b4ee3`, `c83684c11`) exist locally but are not on beta. The original assignment scope does not make that code unimplemented; assess it by behavior and reuse it.

L2 settings/secrets/recovery and repository-focused L3 grounding/replan are implemented locally with remaining live/experience gates. L4 includes Plan cards, schedule API, daemon-selected Start and CDC invalidation, but the existing Mission graph renders decomposition rather than the direct WorkUnit DAG. Full Mission Graph and integrated Board/List navigation remain incomplete. L5 proof/continuation/delivery, L6 focused launch experience, and L7 rehearsal remain open. General non-repository Outcome behavior needs explicit verification/implementation; repository grounding alone does not establish it.

Use the [Work launch experience](product/2026-09-09-kennel-work-launch-experience.md) as target behavior. Complete gaps rather than rebuilding whole slices or resetting progress based on assignment history. Code presence, automated checks, live behavior and owner acceptance remain separate.

## Implemented foundation

Source and automated checks support these implementation claims; they are not end-to-end launch acceptance:

- Go loopback daemon, SQLite canonical writer, additive migrations, trigger CDC/SSE, generated HTTP/TypeScript contracts, thin CLI, Electron/React supervisor.
- Project/session lifecycle, Git worktree/runtime/recovery infrastructure and session/terminal/diff inspection.
- Durable Outcome → ContractRevision → PlanRevision → WorkUnit → Attempt → AgentSessionRef and criterion-bound Evidence/Verification/user Acceptance.
- Immutable Contract/Plan semantics, capability validation, Attempt admission fences and recovery facts.
- Five active execution-provider identities: Codex, Claude Code, OpenCode, Cursor and Pi. Identity is not role/capability conformance.
- PR99: preference-aware routing, persisted approved provider/model binding, historical-unbound rejection, provider-local model semantics, graph validation and serial scheduler decisions. L1a now carries the frozen binding through Manager.Spawn for TUI/Chat; live provider conformance remains unproved.
- PR99: IntelligenceRun storage (migration 0115), provider-neutral intelligence/LLM ports, direct Anthropic/OpenAI reasoning adapters, model-backed Contract and Plan proposals.
- Existing Understand/Decide/Act/Prove Work surfaces, Project Brief/conversation foundation and composed-Outcome storage remain available to evolve.

## Current policy: ADR0012

Waldo reasoning requires the owner's configured reasoning credential. Provider/model/effort are durable daemon settings and the secret is daemon-owned; `KENNEL_WALDO_*` environment values remain explicit development overrides with documented precedence in `daemon/waldo_reasoning.go`.

There is **no deterministic/offline proposal floor** and no hidden alternate-model fallback. Missing configuration must be recoverable setup failure. The old session-spawn intake/decomposition proposers were removed; model-backed decomposition proposal remains unavailable. Do not reconstruct those retired paths from old plans. Reasoning secrets must not enter canonical Work rows or logs.

ADRs 0010/0011/0012, product architecture and ADRs 0008/0009 govern the target. ADR0012 supersedes older fallback/key-optional wording. Migrations through 0115 are merged and immutable; use the next unused migration number for fixes.

## Confirmed remaining source gaps

| Area | Current gap | Plan slice |
|---|---|---|
| Exact model launch | Manager.Spawn now resolves the approved exact binding before readiness and TUI/chat launch; live provider conformance and restart/recovery semantics remain unproved | L7 |
| Runtime authority | Attempt spawn now carries an attributed normalized WorkUnit policy; Codex TUI/Chat require the exact supported capability set and `worktree/*` scope, then pin network, extra writable roots, and temp-root settings at the provider boundary. Live canary enforcement remains unproved | L7 |
| Reasoning readiness/recovery | Local settings/secret readiness, restart reconciliation, metrics and adapter HTTP seams are implemented; live provider conformance remains unproved | L2 |
| Grounding/replan | Bounded repository context, substantive proposal context, clarification semantics, explicit replan, and persisted assumptions/blockers are implemented; live grounded proposal evidence remains open | L3 |
| Plan/Mission UI | Candidate Plan cards/schedule API are present; complete Board/List/Mission Graph and real desktop journey remain unaccepted | L4 |
| Proof/continuation | UI Outcome-level proof does not satisfy scheduler WorkUnit proof scope; automated check/artifact collection and downstream workspace handoff need integration | L5 |
| Re-entry/navigation | Bounded prior-result context and Outcome-first normal entry paths require real journey verification and cleanup | L6 |
| Release | Packaged installation, live provider enforcement, restart, full proof/acceptance loop and measured performance remain unaccepted | L7 |

The daemon correctly rejects a supplied provider different from the approved binding. The frontend still sends Project preference, so this is a client integration defect, not evidence of silent daemon rerouting.

The serial scheduler currently uses a Project custody fence. Full WorkUnit WorkspaceLease parallel scheduling remains later work. Do not remove that fence merely to make a graph look concurrent.

## L1b capability and replay evidence

L1b is implemented in isolated worktree `codex/l1b-capability-replay` on dependency head `677c7612111f080ee447442ad61f8302f22d71da` (L0 plus L1a), with the enforcement correction at `d4cc51a3b` and the recovery correction in the current slice. `AttemptSpawnRequest` now carries an immutable `AttemptExecutionPolicy` built from the approved WorkUnit's required capabilities and only the matching Plan grants, with Outcome/Plan/WorkUnit/Contract and RunBrief attribution. The policy digest and packet are recorded in the v2 admission snapshot; historical snapshots without this field remain readable because no reader synthesizes policy for old rows.

Readiness and actual spawn share the same adapter-policy validation seam. A governed Attempt reduces mutable Project permission posture to a safe provider-neutral mode before the adapter boundary. The shared Codex validator now admits only the exact `worktree.read` or `worktree.read` + `worktree.write` + `worktree.exec` set, with exact `worktree/*` grants; restricted scopes and unknown/additional capabilities return typed `ATTEMPT_EXECUTION_POLICY_UNSUPPORTED` before launch. Codex TUI maps inspection to `--sandbox read-only` and modify-and-execute to `--sandbox workspace-write`, and the latter also pins `sandbox_workspace_write.network_access=false`, empty `writable_roots`, and excluded temp roots with `-c` overrides. Codex Chat sends the corresponding `approvalPolicy`/`sandbox` pair at thread start and repeats an explicit turn-level `sandboxPolicy` with network disabled, empty writable roots, and excluded temp roots, so permissive provider configuration cannot widen governed turns. No live canary/provider/model run was authorized or performed.

Replay now compares canonical Outcome/Plan/WorkUnit (and storage contract revision) identity on both the service fast path and the SQLite unique-key race path. Same-key identical starts return one Attempt; conflicting reuse returns `ATTEMPT_REQUEST_KEY_CONFLICT`; the existing Project fence remains unchanged.

Recovery now marks governed sessions with the immutable policy digest and resolves the frozen provider/model binding plus policy only from the durable Attempt session-reference admission snapshot. TUI native/fresh restore and Chat resume both receive that evidence; malformed, mismatched or missing evidence blocks governed recovery instead of rereading broadened Project permissions. Chat resume reapplies the governed thread and per-turn sandbox policy, including the frozen model selection. Ordinary legacy sessions without the governed marker remain readable.

Evidence: `go test ./...` exit 0; touched-package `go test -race ./internal/service/outcome ./internal/session_manager ./internal/daemon ./internal/storage/sqlite/store ./internal/adapters/agent/codex ./internal/adapters/chatdriver/codexappserver` exit 0; `go vet ./... && go build ./...` exit 0; `npm run lint` exit 0 with `0 issues`; `npm run sqlc` is clean; `git diff --check` is clean. Recovery regression coverage is in `backend/internal/session_manager/recovery_execution_test.go`; adapter restore/resume coverage is in the Codex adapter test files. Live canary evidence is blocked: Codex CLI `0.153.4` is authenticated and reports model `gpt-5.6-luna`, but `/Users/shivanshfulper/.local/bin/codex-code-mode-host` is missing, so model-generated terminal execution cannot reach a real command. The redacted doctor report is `/tmp/kennel-l1b-codex-doctor.json`; the TUI blocked probe is `/tmp/kennel-l1b-tui-live-blocked.log`. No SQL migration, generated API, frontend, deployment, or owner Acceptance was changed or claimed.

## L3 grounded proposals and explicit replan evidence

Candidate L3 code is present at `684b9c0a9`, based on L2 `0b867790c`; this is not accepted L3 completion. Contract and Plan intelligence requests now carry a bounded repository snapshot: registered root, Git revision/dirty state, applicable `AGENTS.md` instructions, selected small text files, Project Brief, discovered package check scripts, and a digest of the exact packet. Inspection is read-only, excludes ignored/dependency/build/secret trees, rejects binary and oversized files, skips symlinks, and is time/file/byte bounded. Discovered commands are presented as facts and are never executed during proposal generation.

Previous Contract proposals are serialized as substantive context rather than reduced to a title. Clarification answers remain clarification context and are appended to the proposal notes by the existing intake state machine; they no longer populate `TemporalCondition` unless a future explicit temporal field is supplied. Plan assumptions and blockers are durable immutable review data in migration 0117 and are included in the Plan API. Ordinary plan reload remains idempotent; `POST /outcomes/{outcomeId}/plans/replan` requires owner feedback and appends a new immutable proposal bound to the expected Contract revision.

GREEN evidence: repository-context exclusion/digest tests, clarification prompt tests, explicit replan service tests, plan-review SQLite round-trip tests, affected backend/controller tests, `npm run api`, `npm run frontend:typecheck`, and `git diff --check` pass. No live grounded Contract/Plan call was performed: live credentials and the code-mode host are absent, so real-provider grounded evidence and desktop journey acceptance remain blocked. L3 evidence logs are `/tmp/kennel-l3-go-grounded.log`, `/tmp/kennel-l3-api.log`, `/tmp/kennel-l3-frontend-typecheck.log` and `/tmp/kennel-l3-live-check.log`.

## L4 Plan and schedule projection evidence

Candidate L4 code is present at `170230bf0`, based on `684b9c0a9`; Board/List and Mission Graph acceptance remain open. The daemon now exposes read-only `GET /api/v1/outcomes/{outcomeId}/plans/{planId}/schedule`, mapping the existing derived scheduler view to the generated API with all WorkUnits, dependency blockers, criterion readiness, Attempt summaries, active Attempt and next runnable identity. Start admission accepts an omitted WorkUnit assertion so the daemon selects that next runnable unit; explicit legacy assertions remain validated. Schedule reads do not spawn or mutate execution state.

Plan review and Act & Observe now show every WorkUnit, criterion/dependency coverage, approved provider/model semantics, routing decisions, assumptions/blockers and daemon-derived schedule state. The normal frontend start request contains only the approved Plan identity and idempotency key; it no longer queries Project roles or sends a mutable harness. Preview data follows the same projection shape.

GREEN evidence: full backend tests, affected backend race tests (including migration recovery), `npm run lint` with `0 issues`, `npm run sqlc`, `npm run api`, frontend typecheck, and targeted Plan/Run/Mission tests (25/25). The full frontend suite reached 230/230 files and 2,776/2,782 tests, with 6 skips. The earlier Home-entry failure is not classified as inherited: follow-up comparison passed 6/6 on both the exact base and this branch, so the failure is unreproduced. Real daemon-backed desktop verification and live provider/model execution remain blocked by the missing `codex-code-mode-host` and absent configured credentials; no live provider call or packaged Electron acceptance is claimed. L4 evidence logs: `/tmp/kennel-l4-backend.log`, `/tmp/kennel-l4-go-full.log`, `/tmp/kennel-l4-go-race.log`, `/tmp/kennel-l4-lint.log`, `/tmp/kennel-l4-migration-fix.log`, `/tmp/kennel-l4-api.log`, `/tmp/kennel-l4-frontend-typecheck.log`, `/tmp/kennel-l4-frontend-tests.log`, `/tmp/kennel-l4-frontend-full.log`, `/tmp/kennel-l4-live-check.log`.

## 2026-09-09 follow-up corrections

Correction commit `d617b4ee3` closes the five review findings on the Wednesday branch. Reasoning credentials are now provider-bound in the daemon-owned file store; switching provider without a matching credential returns actionable `MISSING_CREDENTIAL`. Repository grounding explicitly excludes environment/credential/private-key/secret files, prunes excluded directories at every depth, distinguishes Git “not ignored” from Git failure, checks cancellation during traversal and reads, and stops at filesystem/candidate bounds. The bounded `RepositoryContextSnapshot` is included in the Contract input digest. Outcome CDC events and Plan/attempt/recovery/proof mutations invalidate the open schedule projection.

Evidence and exact acceptance rows are recorded in [the follow-up verification record](verification/2026-09-09-wednesday-followup.md). Full backend tests, affected race tests, full frontend tests, typecheck, lint, vet/build, generation parity and diff checks pass. The isolated daemon settings/repository/Outcome journey passed through Outcome creation and stopped Plan proposal at the missing OpenAI credential boundary; no live provider call or packaged Electron acceptance is claimed. L5 remains closed pending real provider-backed proof/artifact work.

## L0 baseline cleanup evidence

The four-file frontend RED reproduction failed 23 of 114 tests because fixtures still assumed hidden Codex/default-model selection, while current provider-neutral entry paths require explicit admitted selection. The corrected tests preserve explicit selection, keyboard submission, errors, legacy readability and role admission. Narrow GREEN: 114/114 passed. Full frontend GREEN: 230 files, 2775 passed, 6 skipped. Logs: `/tmp/kennel-l0-frontend-red.log`, `/tmp/kennel-l0-frontend-green.log`, `/tmp/kennel-l0-frontend-full.log`.

The Go lint baseline fell from 190 findings to zero unexplained findings. L1a wires the retained `prepareSpawnExecution` helper at the Manager.Spawn boundary and removes `normalizedExactModel` only after proving it had no callers; no lint rule was disabled or suppressed. The final full-lint output is `/tmp/kennel-l1a-lint-final.log`. The first `npm run lint` attempt exposed an L0-introduced stale generated-contract failure after the `AgentRoles` → `Roles` rename; `npm run api` regenerated both contracts and HTTP/spec parity now passes. The initial failing wrapper output is `/tmp/kennel-l0-lint-command-final.log`; regeneration output is `/tmp/kennel-l0-api-regenerate.log` and parity output is `/tmp/kennel-l0-api-parity.log`.

Full Go package tests pass after generated-contract repair; frontend typecheck passes (`/tmp/kennel-l1a-frontend-typecheck.log`). L0 changes no SQL schema or migration and does not change provider runtime behavior; generated OpenAPI/TypeScript contracts were updated to reflect the internal `Roles` name. L1a changes only request-local launch configuration: explicit approved models reach the adapter unchanged, provider-default semantics clear mutable Project model values, and persisted Project configuration remains unchanged. Manager-boundary TUI and Chat tests pass, including the provider-default case. Targeted tests, race tests and vet output are recorded in `/tmp/kennel-l1a-go-targeted.log`, `/tmp/kennel-l1a-go-race.log` and `/tmp/kennel-l1a-go-vet.log`; RED/GREEN launch regression output is in `/tmp/kennel-l1a-manager-red.log`, `/tmp/kennel-l1a-manager-green.log` and `/tmp/kennel-l1a-manager-green-expanded.log`.

## L2 reasoning configuration and recovery evidence

L2 is implemented on beta `9396c3844ee00cf2c350df0426f4224d33ef87de` in the isolated Wednesday milestone branch. Reasoning selection is durable and non-secret in `app_settings`; the API key is held by a daemon-owned provider-bound file secret store with `0700` directory and `0600` file permissions, atomic replacement, and no API/log exposure. Explicit development environment values remain an override for local launches, while packaged settings resolve persisted provider/model/effort and only the matching provider secret. Readiness distinguishes configured, key-present, and ready, with actionable missing-provider/credential errors.

Anthropic and OpenAI adapters now accept an injected HTTP client/base URL for conformance tests and disable implicit SDK retries at the daemon boundary. Adapter tests exercise request headers/body, structured output, malformed/refusal responses, authentication/rate-limit failures, timeout/cancellation behavior and retry count. `IntelligenceRun` records nullable provider usage and daemon duration without converting unknown usage to zero; a SQLite guard and monotonic store method prevent replacement. On daemon startup, requested/running runs are reconciled to an explicit interrupted terminal state before new work is accepted. Contract and Plan proposal adapters record start/terminal/metric/provenance facts and retain the existing no-Attempt proposal boundary.

Narrow GREEN evidence: `go test ./internal/daemon ./internal/service/settings ./internal/service/intake ./internal/service/intelligence ./internal/service/outcome ./internal/adapters/llm/... ./internal/secretstore ./internal/storage/sqlite/store ./internal/httpd/controllers` exit 0; `npm run frontend:typecheck`, `npm run sqlc`, `npm run api` and `git diff --check` exit 0. The first `npm run api` before `npm run bootstrap` was RED because `openapi-typescript` was unavailable; bootstrap installed the pinned dependencies and regeneration then passed. No live provider/model call was performed: no configured live credential/code-mode host was used, so provider conformance, real cancellation and packaged desktop readiness remain open. Evidence logs for this slice are `/tmp/kennel-l2-go-targeted.log`, `/tmp/kennel-l2-frontend-typecheck.log`, `/tmp/kennel-l2-generation.log` and `/tmp/kennel-l2-live-check.log`.

Fresh verification on this branch also passed `go test -race ./...` (exit 0), `npm run sqlc`, `npm run api`, and generated-contract parity (exit 0). This recovery slice's full backend and touched-package race gates are green. `npm run test:foundation` reached the shared cloud-client check after provenance and backend gates passed, then stopped at the pre-existing generated cloud schema drift (`Kennel` versus `AO`, exit 1); no unrelated cloud contract was changed.

## Current verification truth

## Luna Work completion branch — 2026-09-10

The isolated `codex/kennel-work-completion` branch adds the R1–R5 correction
slice, retained artifact bytes and deterministic multi-predecessor composition
primitives, a bounded governed-check runner, a secret-safe supplied-document
adapter, an owner-gated local export helper, and a reversible focused Work
shell. These are source/automated or adapter-level claims, not live-provider,
packaged, or owner-acceptance evidence. C-13 successor admission, full D run
intent/rework integration, document Outcome wiring, export HTTP/delivery
persistence, complete B1/B3 supervision, packaged startup suppression, and
live provider canaries remain open in the completion ledger.

The baseline record distinguishes pass/fail/not-run. The historical fresh run had **23 failures in 4 files**; L0 now has no frontend test failures. The full frontend suite is green at 230 files, 2776 passed and 6 skipped; frontend typecheck, HTTP/spec parity and full lint pass. Full backend tests and the L1b backend/adapters race gates pass. The earlier full `npm run lint` wrapper failure was caused by L0’s stale generated contracts and is repaired. The foundation wrapper remains blocked by the unrelated cloud-client generated-schema drift described above. L1a’s exact-binding regression is green at the Manager launch boundary for both TUI and Chat. L1b’s recovery and adapter policy enforcement are unit/integration-tested, but live TUI/Chat effect probes are blocked by the missing code-mode host; restart/recovery against a real provider, packaged Electron journey, and owner acceptance remain open. The macOS arm64 package build and package identity check pass, but were not launched. Do not treat the whole foundation or provider conformance gate as green.

No live-model, real-provider permission, packaged Electron journey, or owner-accepted Outcome is claimed by this documentation update. Green service tests do not establish those facts.

## Next work

Next is verification/integration of the existing Wednesday code and closure of concrete L2/L3 gaps, followed by the missing direct WorkUnit graph and complete L4 desktop journey. Do not restart L2/L3 from scratch. L5 proof/artifact/continuation/delivery follows the relevant verified dependencies; L6 and L7 remain open. Credentials and the execution host block different boundaries, not all local checks. Push/integration still requires authorization.

The release gate remains: real repo → grounded Contract → full Plan approval → exact bounded execution → retained artifacts/checks → understandable proof → owner acceptance/rework, including interruption and restart without duplicate execution.

# WT3 — Preference-aware provider/model routing and Outcome-first beta UX

**Base:** `origin/beta` at `afa1741a5c59145426e3f9676093a998bc839fc5` (verified 2026-09-07; no intervening beta commits)

**Branch:** `feat/wt3-routing-outcome-first`

## Goal

Make preference, recommendation, and execution authority distinct first-class concepts while preserving Kennel's durable Outcome hierarchy:

```text
Outcome
→ ContractRevision
→ PlanRevision
→ WorkUnit
→ Attempt
→ AgentSessionRef
→ EvidenceItem
→ VerificationRun
→ AcceptanceDecision
```

The user manages Outcomes. Kennel manages the sessions required to make them true. Provider/session completion never accepts an Outcome; only explicit user Acceptance does.

## Non-negotiable invariants

- Project and Outcome preferences are planning input, never execution authority.
- A routing recommendation is explainable but is not execution authority.
- Plan approval is the authority transition.
- A WorkUnit's approved provider/model binding is immutable.
- Attempts consume the approved binding and never route or silently substitute.
- There is no provider-brand fallback and no task→brand mapping.
- Missing coordinator selection never promotes a worker.
- Routing policy consumes normalized requirements/capability metadata, not provider IDs.
- A no-valid-candidate result is valid and blocks execution.
- Sessions remain subordinate execution traces; Board/List remain Outcome-first.

## Phase 0 — current beta architecture

### Project preferences

`domain.ProjectConfig` currently contains both:

1. `Worker` / `Orchestrator` `RoleOverride`s introduced/normalized by WT1; and
2. the older `AgentPreferences` (`DefaultWorker`, `Analyzer`, `Coordinator`, `Verifier`) used by mission-role resolution.

For direct Outcome execution, `service/outcome.projectWorkerProvider()` currently reads only `ProjectConfig.Worker.Harness`, while `ResolveMissionRoles` can still consume `AgentPreferences`.

**Decision:** do not create a third Project preference vocabulary. `ProjectConfig.Worker` and `ProjectConfig.Orchestrator` are the canonical mutable user-facing baseline. Older `AgentPreferences` are compatibility input only. Direct WT3 execution preference resolution uses `Worker`; coordinator resolution uses `Orchestrator`. Compatibility code may read legacy role values only when the direct role is absent and only when doing so is unambiguous; it must never manufacture a provider/model.

### Outcome / ContractRevision

`ContractRevision` currently stores goal, success criteria, review policy, constraints, non-goals, evidence policy, authority ceiling, etc., but no provider/model preference.

**Decision:** add an optional immutable Outcome execution preference to each ContractRevision. A new preference means a new ContractRevision; historical revisions stay unchanged. Pre-WT3 revisions deserialize as no Outcome-specific preference.

### Plan / WorkUnit

`ProposePlan` currently chooses the direct Project worker provider, creates the v0 WorkUnit with that provider, computes the RunBrief digest, and atomically inserts the plan + WorkUnit + immutable provider side row. Plan approval re-validates the proposed plan and transitions it to approved.

**Decision:** routing moves into Plan proposal/review. The proposed Plan contains the recommendation/final proposed candidate. If a user chooses another admissible candidate, create a new proposed PlanRevision rather than mutating the immutable binding row. Approval is the authority transition.

### Existing execution binding

Migration `0112_work_unit_provider_bindings.sql` creates a 1:1 immutable side relation containing provider. UPDATE and DELETE are rejected by triggers. `0113` is intake-analysis expiry. Both are shipped history and remain untouched.

**Decision:** migration `0114` extends this same binding relation with explicit model-selection authority and adds nullable persistence for Outcome preference and Plan routing provenance. Do not create a second provider source of truth. Historical 0112 rows remain readable as provider-bound/model-unbound truth.

### Attempt

`StartAttempt` currently reads the approved WorkUnit provider, rejects a mismatched compatibility `Harness` assertion, validates provider readiness, checks the RunBrief authorization digest, and spawns that exact provider. It does not reroute.

**Decision:** preserve that shape. Extend it to read/validate/pass the exact bound model-selection semantics. A bound provider/model becoming unavailable fails closed. Retry keeps the same WorkUnit binding; changing provider/model requires replanning and re-authorization.

### Model flow

`domain.AgentConfig.Model` is the current model string. Session spawn ultimately receives an `AgentConfig`; an empty model can currently fall back to project/provider behavior. The normalized model catalog exposes `id`, `label`, `provider`, `isDefault`, `selectionMode`, `allowCustom`, and source, which is enough for pickers but not routing.

Supported provider catalog reality:

- Codex has a static normalized catalog/default.
- Claude Code has static aliases but the active provider-default may intentionally be unknown.
- OpenCode, Cursor, and Pi discover models through their installed CLIs; discovery may be unavailable or empty.

Therefore WT3 cannot assume every supported provider always exposes a safely freezeable concrete current default.

### Provider role/capability metadata

Provider-specific code already knows supported providers and coordinator eligibility; current `IsSelectableAsCoordinator` makes Cursor and Pi worker-only. WorkUnit capability vocabulary currently contains verified execution effects such as worktree read/write/exec.

**Decision:** provider-specific components may emit normalized metadata. The router itself receives only normalized role/readiness/capability/model facts. Any provider-ID branching remains outside routing policy.

## Domain design

### 1. Outcome execution preference

Use a typed optional ContractRevision field. Repository naming may be refined during implementation, but semantics are fixed:

```text
ExecutionPreference (optional)
  provider: required when preference exists
  model selection:
    provider_default
    explicit
  model: required only for explicit
```

No preference object means "inherit Project baseline / no Outcome override".

A present Outcome preference always owns both provider and model semantics. This prevents cross-provider model leakage.

Semantics:

| Outcome fields | Meaning |
| --- | --- |
| provider empty, model empty | no Outcome preference object; inherit Project baseline |
| provider set, model empty in UI | normalize to provider + explicit `provider_default` |
| provider set, model set | provider + explicit model after provider/model validation |
| provider empty, model set | invalid/unresolved; never infer or guess provider |

Project worker normalization follows the same safety rule. A Project model is only paired with its explicit Project worker provider. No provider means no model preference.

Effective preference:

```text
Outcome preference present → exact Outcome preference
Outcome absent             → normalized Project worker preference
Project absent             → no preference
```

Example: Project Claude/Sonnet + Outcome Codex/default => Codex/provider-default, never Codex/Sonnet.

### 2. Execution binding

Extend the existing WorkUnit provider binding to represent:

```text
ExecutionBinding
  provider
  model selection:
    explicit
    provider_default
    historical_unbound (read compatibility only)
  model (for explicit)
```

`historical_unbound` is never produced by new WT3 Plan proposals; it only truthfully represents pre-WT3 WorkUnits.

If a provider exposes a concrete default that can safely be frozen, routing may recommend/bind that concrete model. If the adapter's launch contract genuinely supports only provider-default semantics, bind `provider_default` explicitly.

### 3. RunBrief authority

Add provider model-selection mode + model identity to the RunBrief authorization material. Changing explicit model X→Y or explicit→provider-default changes the authorization digest. Routing rationale is not execution-authority material.

### 4. Routing requirements

Create a typed provider-neutral representation derived from durable facts:

```text
RoutingRequirements
  role
  hard capabilities
  soft strengths/preferences
  execution constraints
  model requirements
```

Initial hard facts come from the ContractRevision, WorkUnit role/effects, capability grants, verification/evidence requirements when they map to a verified capability vocabulary, plus provider/model readiness. Do not invent unverifiable capabilities.

If semantic inference is added later, its output must be structured, vocabulary-constrained, validated, and advisory. It never directly authorizes a provider.

### 5. Normalized capability metadata

Keep facts distinct from quality/strength estimates.

```text
CapabilitySupport = supported | unsupported | unknown

ProviderRoutingMetadata
  roles
  readiness facts
  execution capabilities

ModelRoutingMetadata
  identity/default semantics
  factual capabilities
  optional strength signals
```

For mandatory requirements, `unknown` does not satisfy the gate.

WT3 should only populate metadata Kennel can actually verify today. Reliability, cost, latency, quota and historical success remain typed future extension points with no fabricated values.

### 6. Candidate pairs and hard gating

Generate valid provider/model pairs, including explicit provider-default candidates only where the adapter supports that launch contract.

Hard gates precede ranking:

- supported provider surface;
- provider installed/available for the requested role;
- authorization/profile readiness acceptable;
- coordinator/worker role eligibility;
- model/default/custom semantics valid;
- mandatory provider/model capabilities satisfied;
- policy constraints satisfied.

Zero candidates returns a typed `NO_VALID_CANDIDATE` result. No provider is synthesized.

### 7. Ranking

Rank only admissible candidates. Preference is a soft signal.

Policy rules:

- If admissible candidates are otherwise materially equivalent, favor the explicit user preference.
- A materially stronger capability fit may beat preference, with a structured reason.
- Hard requirements always beat preference.
- Provider/model names are never ranking features.
- Non-preference ties use a neutral deterministic candidate key or remain an explainable ranked tie; lexical provider ordering must not become hidden policy.

Router unit tests use fictional providers/models (`alpha`, `beta`, `gamma`) to prove brand neutrality and input-order independence.

### 8. Routing decision / provenance

Persist concise plan-bound structured provenance, not hidden model reasoning:

```text
RoutingDecision
  status
  policy version
  capability snapshot/version
  role
  effective preference
  requirements that mattered
  recommended candidate
  candidate evaluations / hard rejection codes
  concise structured reasons
```

This supports three UX states: preference honored, different recommendation, and no preference.

### 9. Authority policy

```text
Kennel derives requirements
→ filters/ranks candidates
→ proposes Plan with recommendation/final proposed binding
→ user reviews preference vs recommendation
→ user may choose another admissible candidate
→ a new proposed PlanRevision represents that choice
→ user approves Plan
→ binding becomes execution authority
→ Attempt executes exact binding
```

No second hidden authorization mechanism is introduced.

## Persistence design — migration 0114

Do not alter 0112/0113.

`0114` should, using SQLite-compatible additive changes:

1. add nullable `execution_preference_json` to `contract_revisions`;
2. add nullable `routing_decision_json` to `plan_revisions`;
3. add nullable model-selection columns to `work_unit_provider_bindings`, e.g. `model_selection` and `model`;
4. add INSERT validation for new model semantics while keeping historical rows valid;
5. preserve existing immutable UPDATE/DELETE triggers so provider and model authority cannot mutate.

New WT3 insert paths require a valid explicit/provider-default model selection. Existing rows with NULL selection remain historical model-unbound.

## TDD implementation sequence

Every production behavior follows Red → verify failing → Green → verify passing → Refactor. Because this execution harness cannot run local `git`/build commands against GitHub, RED/GREEN command verification must be performed through repository CI where available; no local result will be claimed.

### Slice A — preference domain + persistence

**RED tests first**

- ContractRevision validates provider/model preference combinations.
- Project Claude/Sonnet + Outcome none => Claude/Sonnet.
- Project Claude/Sonnet + Outcome Codex/default => Codex/provider-default; no Sonnet leak.
- no Project + no Outcome => no preference; no Codex.
- old ContractRevision persists/reads as no Outcome preference.
- revision 1 Claude/Sonnet remains unchanged after revision 2 Codex/default.
- restart/store round-trip preserves both revisions.

**GREEN**

Add typed domain preference, resolver, migration/storage serialization and HTTP request/response fields at canonical Contract intake/revision seams.

### Slice B — immutable model binding + RunBrief

**RED tests first**

- new provider/model binding round trips.
- historical provider-only binding reads as model-unbound.
- UPDATE/DELETE remains impossible.
- provider P/model X vs P/model Y produce different RunBrief authorization digests.
- provider-default vs explicit model are distinct digest material.

**GREEN**

Extend 0112 relation via 0114, storage binding API, WorkUnit projection, and RunBrief core/digest.

### Slice C — generic routing policy

**RED tests first with fictional providers**

- unauthorized preferred candidate filtered.
- mandatory capability missing/unknown filtered.
- worker-only candidate filtered for coordinator role.
- zero candidates => typed no-valid-candidate.
- preference wins equivalent admissible candidates.
- materially better soft capability candidate may beat preference with structured reason.
- same provider model A lacks required capability/model B satisfies => B recommended.
- candidate ordering/provider names do not alter policy result.

**GREEN**

Implement pure normalized candidate generation/gating/ranking and structured decision types with no provider-ID logic.

### Slice D — production metadata + Plan authority

**RED tests first**

- production metadata adapter emits role/readiness/model facts for exactly five supported providers without adding a brand fallback.
- ProposePlan uses effective preference only as routing input.
- recommendation is persisted with proposal.
- no valid candidate produces Action Required/no executable Plan state and no Attempt.
- user-selected admissible alternative becomes a new proposed PlanRevision.
- approval freezes exact provider/model binding.
- later Project preference or ContractRevision changes do not mutate approved Plan.

**GREEN**

Wire provider inventory/model catalogs into normalized metadata builder; derive v0 routing requirements from durable Plan/WorkUnit facts; route at proposal/review; persist decision provenance.

### Slice E — exact Attempt execution

**RED tests first**

- approved P/M launches P/M.
- conflicting provider assertion is rejected.
- conflicting legacy model assertion is rejected or removed at API boundary.
- explicit provider-default launches provider-default semantics intentionally.
- bound provider/model unavailable at launch => typed failure, no alternate provider.
- retry preserves the WorkUnit binding.

**GREEN**

Extend attempt spawn/readiness request with exact binding semantics and map it into session `AgentConfig` without re-routing or preference fallback.

### Slice F — role separation

**RED tests first**

- worker selection never fills coordinator by omission.
- normalized Cursor/Pi worker-only metadata cannot pass coordinator hard gate.
- no valid coordinator stays unassigned/no-valid-candidate.

**GREEN**

Keep coordinator preference/routing independent from Outcome worker execution preference.

### Slice G — Outcome-first beta UX cleanup

After core authority tests are green, implement in a separate commit series.

**RED frontend/projection tests first**

- one Outcome with multiple Attempts/Sessions => one Board card + one List row.
- sessions remain drill-down execution traces.
- unfinished Home is hidden/truthfully disabled as primary beta navigation.
- global empty Outcomes exposes `Define Outcome` and opens canonical intake.
- leave `Switch project` unchanged unless current source proves an accessibility defect.

## Phase 0 questions answered

- **Canonical Project worker preference:** `ProjectConfig.Worker` provider + worker `AgentConfig.Model`/model semantics after normalization.
- **Canonical Project coordinator preference:** `ProjectConfig.Orchestrator`, independent and optional.
- **`ProjectConfig.Worker` vs `AgentPreferences.DefaultWorker`:** the former is WT1's direct user-facing execution baseline; the latter is an older mission-role preference used by role resolution.
- **Two overlapping worker notions?** Yes. WT3 does not add a third. Direct Worker is canonical; legacy role preference is compatibility input only.
- **Where model identity flows today:** raw `AgentConfig.Model` through Project/session configuration into provider adapters; WorkUnit authority does not freeze it yet.
- **Does every provider always expose a concrete model catalog/default?** No. Codex/Claude have static normalized knowledge but Claude's active default may be opaque; OpenCode/Cursor/Pi depend on CLI discovery/readiness.
- **Catalog vs text/mode selection:** the five product providers are modeled through catalog/discovery for normal selection; text/manual fallback is an availability/picker affordance and must not become routing authority. Mode selection is not a provider-brand default mechanism.
- **Normalized provider facts today:** product support, installed/auth/profile readiness surfaces, worker/coordinator eligibility, and execution capability vocabulary such as worktree read/write/exec.
- **Facts currently adapter-local:** concrete model discovery/default behavior, provider CLI/runtime interface details, and some readiness/tool behavior.
- **What chooses `WorkUnit.Provider` today:** `ProposePlan` reads `ProjectConfig.Worker.Harness` directly.
- **When provider becomes immutable:** proposal insertion atomically writes the WorkUnit and immutable 0112 provider binding; Plan approval is the authority transition. WT3 preserves row immutability and expands it to model semantics.

## Verification gate

Before completion, run fresh evidence for backend unit/storage/provider/routing/Contract/Plan/Attempt/frontend/Board/List tests, migration sequencing, typecheck, lint, build, Electron dev/package builds where supported, real-daemon provider/model acceptance, and restart persistence. Any command unavailable in the harness is reported as unavailable rather than claimed.

Migration ledger must remain `0112 provider binding → 0113 intake expiry → 0114+ WT3` unless beta advances before merge.

## Review gate

Request review specifically for hidden provider/model defaults; worker→coordinator promotion; provider IDs or task→brand mappings inside policy; duplicate preference/binding sources; cross-provider model inheritance; stale preference mutating approved plans; runtime rerouting; unknown capability treated as supported; chain-of-thought persistence; migration conflicts; session-centric UI regressions; and scope creep into scheduler/WorkspaceLease/Home.

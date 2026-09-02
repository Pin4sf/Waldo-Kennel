# Kennel Provider Core and Product Cleanup Design

**Date:** 2026-09-02
**Status:** Approved implementation baseline

## Goal

Turn Waldo Kennel from an AO-derived, Codex-gated orchestrator into a small, provider-neutral Kennel product with five active local harnesses: **Codex, Claude Code, OpenCode, Cursor, and Pi**. Provider-specific behavior stays inside adapters; project, routing, UI, and Outcome logic consume normalized capabilities and readiness.

## Product principles

1. **Outcome is the durable responsibility; provider sessions are replaceable attempts.** Provider completion never means Outcome completion.
2. **No silent provider fallback.** If a user chooses Claude Code and it is unavailable, Kennel explains why and asks for another choice. It never silently starts Codex.
3. **Supported is not ready.** Kennel supports exactly five harnesses; each Mac/project gets a fresh readiness projection.
4. **Provider quirks are adapter concerns.** Product services and React must not contain `provider == codex` gates for ordinary selection or routing.
5. **Determinism is reserved for invariants.** Keep dependency-cycle checks, capability admission, permission ceilings, idempotency, recovery fencing, evidence validation, and explicit acceptance. Remove transcript-marker state machines and static provider-priority policy.
6. **No legacy compatibility burden.** Old AO harnesses, old AO npm packages, old mobile/cloud/landing donor surfaces, and historical provider decoding are not part of this refactor's supported product.
7. **Kennel identity everywhere.** Active source, environment variables, comments, docs, generated names, and package/module paths should use Kennel terminology rather than AO terminology.

## Active provider model

The build contains one provider registry with these ids:

- `codex`
- `claude-code`
- `opencode`
- `cursor`
- `pi`

Each manifest exposes normalized capabilities in addition to static identity. The initial capability vocabulary is intentionally small:

- `worker`: can execute a WorkUnit
- `orchestrator`: can run a planning/coordination session
- `reviewer`: can perform a code-review run
- `resume`: Kennel can reconnect or restore a managed session
- `steer`: Kennel can send a follow-up to a running managed session without relying on terminal scraping
- `model-selection`: provider exposes a model override/catalog surface
- `auth-probe`: Kennel can reliably distinguish authenticated vs unauthenticated
- `structured-events`: provider exposes hooks/app-server/plugin events that Kennel can normalize without transcript parsing

A provider need not implement every capability to be selectable. Role selection filters by the capability actually required.

## Readiness model

The daemon is the authority for provider readiness. The frontend renders it; it does not recreate policy.

`ProviderReadiness` contains:

- provider identity + display label
- `installed`
- `authStatus`: `authorized | unauthorized | unknown`
- normalized capabilities
- `readyRoles`: `worker`, `orchestrator`, and/or `reviewer`
- optional actionable problem code/message

Readiness rules:

- missing binary => visible, disabled, `Needs install`
- installed + explicit auth failure => visible, disabled, `Needs authentication`
- installed + auth unknown => selectable only when the adapter declares auth optional/unknown-safe (for example providers that can run local/free models); final spawn remains authoritative
- installed + role capability => selectable for that role
- any selection is re-admitted at spawn so stale inventory cannot bypass runtime validation

The UI always shows all five supported providers, ordered by readiness first and then display name. It does not use a hard-coded quality ranking.

## Project setup UX

Project registration stays simple:

1. User chooses a repository/workspace.
2. Kennel refreshes the five provider readiness probes in parallel.
3. Worker and orchestrator pickers show all five providers with status text.
4. A provider is selectable only when ready for that role.
5. If exactly one provider is ready for a role, Kennel may preselect it as a convenience. If multiple are ready, no hidden ranking is applied; use the user's existing project/global preference if one exists, otherwise leave the field explicit.
6. The project stores an explicit worker and orchestrator provider choice. It does not encode Codex as a project default.
7. Settings reuse the same picker and same readiness projection.

The picker must offer refresh/recheck and actionable install/auth guidance instead of failing later with a generic spawn error.

## Routing

Routing has two layers:

### Explicit routing

A WorkUnit may name a requested provider. Kennel validates support, role capability, readiness, project policy, and permission ceiling. It either admits that provider or returns a structured reason. It never substitutes a different provider.

### Automatic routing

Automatic routing is a separate explicit choice, not the empty-string alias for Codex. It considers only admitted candidates. v1 uses transparent deterministic scoring derived from declared requirements and user preferences; it does not hard-code provider brand priority.

Initial factors:

- role/capability match (required)
- project/user preferred provider (if set)
- requested model availability (if set)
- resume/structured-event support when the WorkUnit requires long-running supervision
- recent local readiness

If candidates tie, Kennel returns the candidate set for explicit selection rather than sorting by a hidden brand preference.

## Reviewer policy

Reviewer selection uses the same provider catalog. Remove the separate long reviewer vocabulary and frontend trust allowlists. Reviewer support is a provider capability.

Trust/permission warnings come from normalized capability/security metadata owned by the adapter/daemon, not hard-coded React sets.

## Outcome state

Delete the prototype transcript-marker authority:

- `KENNEL_OUTCOME_QUESTIONS_JSON`
- `KENNEL_OUTCOME_ANSWERS_JSON`
- `KENNEL_OUTCOME_PLAN_JSON`
- `KENNEL_OUTCOME_PLAN_REVISION`
- `KENNEL_OUTCOME_PLAN_APPROVED`

The renderer must never infer canonical Outcome stage by scanning assistant text. Structured model output is a proposal that enters a typed daemon command; the daemon validates and persists canonical state. The accepted product lineage remains:

`Outcome -> ContractRevision -> PlanRevision -> WorkUnit -> Attempt -> AgentSessionRef -> EvidenceItem -> VerificationRun -> AcceptanceDecision`

The existing `OutcomeTask/completed` donor model is removed/replaced as the new durable slice lands. Provider/session exit cannot write AcceptanceDecision.

## AO removal and repository cleanup

Remove from the active tree:

- all worker/reviewer adapters except the five supported providers
- `frontend/src/landing`
- `packages/mobile`
- `packages/cloud-client`
- frozen `packages/ao*`
- obsolete AO comparison/docs/assets that only describe Agent Orchestrator

Rename active source identity:

- Go module: `github.com/pin4sf/waldo-kennel/backend`
- CLI entrypoint: `backend/cmd/kennel`
- comments/copy: Kennel, never AO for current behavior
- environment/header names: `KENNEL_*` / `X-Kennel-*`
- temporary `.ao/*` project artifacts are replaced by `.kennel/*`

No migration for removed AO state is required.

## Error handling

Provider failures use stable machine-readable codes with user-facing guidance, for example:

- `PROVIDER_NOT_INSTALLED`
- `PROVIDER_AUTH_REQUIRED`
- `PROVIDER_ROLE_UNSUPPORTED`
- `PROVIDER_NOT_READY`
- `PROVIDER_MODEL_UNAVAILABLE`
- `PROVIDER_START_FAILED`

Refresh/probe failures are non-fatal to the app. A failed provider appears unavailable with its reason while other providers remain usable.

## Reliability and security

- Probe subprocesses remain timeout-bounded and refreshes rate-limited/coalesced.
- Spawn always repeats authoritative binary/config/admission checks.
- Worktree isolation and project path traversal guards remain.
- Permission policy is explicit per Attempt; adapters translate it to native provider semantics.
- Provider security characteristics are data, not frontend conditionals.
- No transcript parsing is permitted for product truth where a provider exposes structured events or daemon state.

## Testing gates

A change is not complete until tests prove:

1. only five providers exist in the shipped registry;
2. all five are probed by inventory refresh;
3. installed/auth states map to role readiness correctly;
4. project worker/orchestrator selection accepts every ready provider and rejects unavailable ones with structured errors;
5. no code path silently substitutes Codex;
6. automatic routing never uses a fixed provider brand priority;
7. frontend menus render the daemon catalog and statuses without Codex filters;
8. reviewer selection uses provider capability rather than a second hard-coded allowlist;
9. removed AO packages/surfaces are absent from build scripts;
10. active Go imports/module/CLI/package identity are Kennel-branded;
11. marker-based Outcome state is absent from renderer authority;
12. foundation, backend tests, frontend typecheck/tests, package identity, and security audit pass.

## Out of scope

This refactor does not add cloud execution, a Waldo backend, Gmail/Home continuity, cross-device sync, or new providers beyond the five listed above.
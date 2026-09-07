# ADR 0006: One durable Waldo with multiple governed presences

- Status: Accepted
- Date: 2026-08-21
- Scope: Final ecosystem identity, custody, desktop/mobile roles, and the evolution from Local Waldo Core
- Depends on: [ADR 0003](0003-local-first-waldo-core.md), [ADR 0004](0004-parallel-home-personal-agent-and-required-capture.md), and [ADR 0005](0005-governed-project-learning-and-skill-evolution.md)

## Context

Waldo's first useful proof is local-first: Kennel runs the canonical Waldo Core and execution runtime on one Mac. The intended ecosystem later includes durable continuity across desktop, mobile, phone/wearable capture, and hosted proactive execution. Treating each surface as an independent assistant or canonical memory store would split identity, corrections, authority, responsibilities, learning, and deletion.

Health-aware mobile support creates an especially sensitive boundary. Body and health context can improve planning and assistance, but it must not become a required product category, a hidden whole-person profile, or a source that silently flows into Work, Memory, or Learning.

## Decision

The mature Waldo ecosystem has one owner-scoped durable Waldo agent. It owns the canonical identity and the admitted continuity contracts that span surfaces:

- ResponsibilitySpace, Outcome, OpenLoop, ContractRevision, and ResponsibilityLink;
- authority, EffectIntent, Evidence, Verification, Acceptance, and audit lineage;
- admitted Memory and its provenance, correction, expiry, revocation, deletion, and non-resurrection generations;
- governed learning candidates, promoted SkillRevisions, bindings, and invocation receipts;
- presence admission, disclosure policy, and cross-surface Re-entry.

Kennel is Waldo's desktop Work and execution-supervision presence. The Waldo mobile app is the same agent's personal and Health-aware presence for Home, capture, briefing, mobile action, and permissioned body context. Phone, wearable, web, voice, terminal, MCP, and provider integrations are additional presences or adapters, never additional Waldo identities.

Health First is recommended, not required. The mobile app, Home, Work, capture, Outcomes, Open Loops, planning, verification, and closure remain useful without health access. Raw health observations do not automatically enter general Memory, Work context, Learning, responsibility, or proof. Every use requires purpose, scope, sensitivity, provenance, consent, retention, disclosure, correction, revoke, and deletion controls.

### Evolution from local-first

ADR 0003 remains the v0/v1 launch topology. A later hosted/durable attachment must preserve identity while changing canonical custody deliberately:

1. Local Waldo Core remains authoritative until the user approves an attachment and migration preview.
2. The attachment protocol assigns one canonical writer for each scope and epoch; active-active dual canonical writers are forbidden.
3. Desktop and mobile may keep encrypted, bounded offline caches and source queues. They are replicas and projections, not competing truth.
4. Sync carries ordered revisions, source/deletion generations, acknowledgements, conflicts, and explicit degradation.
5. Detach, export, revoke, account loss, offline operation, and deletion must have tested behavior before hosted custody can become active.
6. Workspace bytes, credentials, provider authentication, raw traces, and unselected sensitive artifacts remain device-local unless separately disclosed.

The exact hosted transport, service runtime, backup provider, and mobile implementation are intentionally not selected by this ADR. They require a later attachment protocol specification, health privacy threat model, and implementation plan.

## Consequences

### Benefits

- one identity and correction history across life and work;
- no duplicate personal profiles or device-specific assistant behavior;
- Health-aware planning can remain consented context instead of becoming Waldo's product category;
- Kennel can specialize in deep execution while mobile specializes in personal presence without splitting authority;
- deletion, revocation, and learning rollback have one governed lineage.

### Costs

- hosted attachment requires identity-preserving migration, offline conflict handling, encryption, and account recovery;
- health and bystander data require stricter consent, minimization, disclosure, and deletion proofs;
- mobile offline usefulness must coexist with a single canonical writer;
- availability of the durable agent becomes an ecosystem dependency after attachment.

## Rejected alternatives

1. **One assistant per device.** Rejected because identity, Memory, responsibility, and corrections would diverge.
2. **Peer-to-peer active-active canonical stores.** Rejected because authority, Acceptance, deletion, and audit conflicts cannot be resolved by last-write-wins.
3. **A health-only Waldo identity.** Rejected because Health First is a recommended wedge, not the product boundary or prerequisite.
4. **Upload all local history on sign-in.** Rejected because attachment and historical disclosure must be explicit and scoped.

## Implementation boundary

This ADR locks the final ecosystem direction. It does not authorize hosted attachment, mobile/wearable implementation, health-data processing, migration, deployment, release, or a second canonical writer. Those actions require their own approved specifications and plans after the local responsibility, capture, memory, and deletion contracts pass.

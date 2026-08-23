# Waldo Personal Agency Rail Modes — Incremental Design Checkpoint

**Date:** 2026-08-23

**Status:** Approved direction, implementation checkpoint

**Stacked baseline:** PR #50 / `codex/home-waldo-conversation-ui-design` at `820fddfe261aa7abddcad66ab01743eb3d42ee63`

**Scope:** Renderer-only, deterministic preview behavior inside the existing global `WaldoRail`

## Decision

The global rail remains one relationship with Waldo across Home and Work. This increment divides that relationship into two legible modes without adding another destination or agent identity:

- **Conversation** is where the user discusses, clarifies, redirects, returns to a topic episode, detaches context, and reviews structured observation, proposal, approval, and result cards.
- **Activity** is where the user inspects bounded delegated runs: goal, scope, delegated agent, permitted sources, ordered plan, authority, approvals, evidence, return path, and status.

Conversation and Activity are two projections of one Waldo relationship. Activity is not a roster, dashboard, or second agent. A specialist may appear only as the named executor of a bounded run.

## Interface job

The user should be able to answer three questions at a glance:

1. What relationship or topic am I in?
2. Is Waldo discussing, proposing, waiting, running, or returning evidence?
3. What became true, what remains unknown, and what—if anything—needs my authority?

The rail should therefore lead with state and next judgment, then reveal detail. It must not become a wall of transcript text.

## Preview information architecture

### Stable rail chrome

The existing header, Home/Work placement, Work Inspector return, narrow Back action, launcher, focus restoration, and canonical unconfigured state remain unchanged. The explicit preview boundary remains visible in both modes.

Below that boundary, preview-only controls provide:

1. A two-way `Conversation | Activity` mode switch.
2. A topic/episode selector with three deterministic local episodes.
3. A visible entry-state label: `Fresh`, `Contextual`, or `Returning`.
4. A pinned context chip that can be detached locally without changing the underlying Home or Work surface.

The selected mode, episode, detached context, expanded detail, and preview card acknowledgements are ephemeral renderer state. They are not conversation history, memory, responsibility, or run state.

### Conversation entry states

| State | Preview meaning | Default content |
| --- | --- | --- |
| Fresh | A new local episode with no transcript or attached object | Short welcome, suggested prompts, no plausible prior messages |
| Contextual | A local episode opened around the current Home or Work context | Pinned context, short exchange, observation and proposal cards |
| Returning | A local episode resumed around an earlier preview topic | Brief return summary and a result card; no claim of durable history |

Episode selection must be explicit. Context may be suggested by the current surface, but the user can detach it. Detaching changes the chip to `No attached context`; it never deletes, mutates, or disassociates a native object.

### Native structured cards

Conversation uses compact native cards instead of long prose:

- **Observation card:** `Noticed` is the observation state. `Candidate` is a separate disposition. It states that nothing was admitted to Memory or accepted as responsibility.
- **Proposal card:** `Proposal` and `Approval required` remain distinct. Opening review changes only local preview presentation. It cannot execute or save.
- **Result card:** `Result ready` describes returned material. `Outcome · Unknown` remains visible until a native verification/acceptance fact exists. Provider or run completion cannot become `Verified outcome`.

The UI may name `Verified outcome` only as a semantic boundary or as a future possible terminal state; no deterministic fixture in this slice claims one.

### Activity mode

Activity shows one bounded run card under Waldo, not a grid of agent identities. The collapsed reading path is:

`status -> goal -> current step -> approval/evidence/return summary -> inspect run`

Expanded detail exposes:

- goal and completion condition;
- bounded scope;
- delegated agent or specialist name and purpose;
- permitted sources;
- ordered plan steps with text labels and non-color-only states;
- authority explicitly denied or allowed;
- approval boundary;
- evidence produced so far;
- exact Home or Work return path; and
- `Running`, `Approval required`, `Result ready`, or `Unknown` status.

The preview run remains labeled `Preview run · not live or saved`. No progress percentage, avatar roster, personality, opaque confidence, or `Done` label is used.

## Semantic truth matrix

| Label | Means | Must not imply |
| --- | --- | --- |
| Noticed | Waldo surfaced an observation | admitted memory, priority, responsibility |
| Candidate | something may be reviewed | accepted truth or durable state |
| Proposal | a native command shape is available to inspect | execution |
| Approval required | an effect is blocked pending explicit authority | approval granted |
| Running | a bounded run is active in the preview story | outcome completion |
| Result ready | returned material is available for review | accepted or verified Outcome |
| Verified outcome | a native verification/acceptance fact exists | provider completion alone |
| Unknown | the product cannot prove the terminal state | failure or success |

## Visual and interaction rules

- Preserve the warm, editorial Home language and the 400–440px rail target.
- Use one strong reading path, restrained borders, and quiet state chips; avoid equal-weight nested cards.
- Mode switching is a compact segmented control with tab semantics and visible keyboard focus.
- Episode selection is a compact local control, not a sidebar or transcript archive.
- Keep first responses short; expansion belongs to card/run details, not whole-screen prose.
- No required animation. Existing rail entrance motion remains reduced-motion safe.
- Use text plus icon/shape for state. Never use color alone.
- The narrow layer keeps Back visible and does not create horizontal overflow.

## Canonical and preview boundary

Without `VITE_NO_ELECTRON=1` or `VITE_WALDO_UI_PREVIEW=1`, `WaldoRail` continues to show only the honest unconfigured surface. It renders no modes, topic episodes, plausible messages, observations, proposals, results, or runs.

With the explicit preview seam, every mode retains the `Interaction preview` boundary and states that no model or agent is running and nothing is sent or saved. Local controls have no daemon client, provider, runtime, persistence, mutation, or IPC path.

## Evidence reconciliation

Adopted from the old draft flow maps:

- fresh/contextual/returning entry states;
- detachable pinned context;
- persistent topics as an information-architecture concept;
- short-first expandable detail;
- structured in-place cards;
- observation/action separation and inspectable action reasoning.

Rejected or deferred:

- health as the relationship prerequisite;
- mascot mood or personality inference;
- proactive thread injection without consent;
- confidence-based hidden ordering;
- direct action buttons that silently execute;
- Patrol as proof of Outcome completion;
- Constellations or plausible long-term pattern intelligence;
- cross-surface persistence, provider connections, and agent execution.

## Before / after / why

| Before | After | Why |
| --- | --- | --- |
| One mixed preview stream combines discussion, proposal, and run detail | Conversation and Activity provide two explicit views of one relationship | Separates reasoning from orchestration without creating a second agent surface |
| One fixed preview exchange | Fresh, Contextual, and Returning local episode states | Makes entry behavior inspectable and testable |
| Current route context is visible but fixed | Context is pinned and detachable | Preserves user control over what Waldo may consider |
| Proposal and one activity trace carry most semantics | Observation, proposal/approval, result, and bounded-run cards each state their boundary | Prevents prose or provider progress from silently becoming truth |
| Run detail is partly visible in a long card | Short summary plus explicit inspectable detail | Improves scanning while preserving governance evidence |

## Acceptance boundary

This checkpoint is satisfied only when automated and rendered evidence proves mode switching, all three entry states, episode selection, context detachment, structured-card semantics, complete run inspection, keyboard focus, narrow layout, and canonical no-preview honesty. It still does not make Waldo conversation, activity, persistence, or execution functional.

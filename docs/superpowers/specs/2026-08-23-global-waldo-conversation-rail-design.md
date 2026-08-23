# Global Waldo Conversation Rail — Product and UI Design

**Date:** 2026-08-23
**Status:** Proposed for written review
**Target baseline:** `origin/beta` at `a9d2d4228ac6aa030720f551f0da10d0d1ca2596` (merged PR #47)
**Implementation authority:** None. This document authorizes neither product-code edits nor remote actions.

## Decision summary

Waldo should be reachable from every Home and Work surface through one shell-owned launcher and one right-side conversation rail. The rail is a relationship surface with the same durable Waldo, not a second Home dashboard, a provider-branded chat client, or a roster of autonomous identities.

Home and Work remain responsibility-space lenses. Waldo can explain context, help the user capture or correct something, propose a native command, resume work, or delegate a bounded task. The conversation itself never becomes canonical responsibility truth. Any durable responsibility or external effect must cross the existing typed daemon boundary and retain its own explicit authority and confirmation rules.

The first implementation slice, after this design and a separate implementation plan are approved, is a truthful UI shell only. It must not imply that conversation, providers, agent runs, or persistence exist before their daemon contracts exist.

## Why this belongs in Kennel

The user should not need to navigate to a special agent page before asking for help. A globally reachable Waldo makes the product feel like one continuous personal agent across Home and Work while preserving the native surfaces that make facts inspectable and actions governable.

The product hierarchy is:

1. Home and Work show inspectable responsibility and activity facts.
2. Waldo helps the user understand or act on those facts.
3. Native commands own durable mutations and effects.
4. Provider runtimes and subagents are replaceable executors behind Waldo, not the identity or source of truth.

## Product invariants

- There is one durable Waldo identity across Home and Work.
- Home and Work change the responsibility-space lens and available capabilities, not identity, custody, or canonical storage.
- The daemon and SQLite remain the only canonical writers.
- The renderer remains a thin projection and intent surface.
- Assistant prose, provider output, session completion, observation, suggestions, and UI rendering are never authoritative responsibility events.
- User statements outrank behavioral inference.
- Health, capture monitoring, hosted services, and provider connections are optional context or execution paths, never prerequisites for Home, Open Loops, or Quick Capture.
- Local responsibility truth remains useful and inspectable without an account or model connection.
- The rail must preserve exact route, selection, scroll, and keyboard-focus return when it opens and closes.
- Exactly one expanded Quick Capture remains on Home Today.

## User jobs

The global Waldo surface must make these jobs possible without turning chat into a hidden control plane:

- Ask Waldo to explain what is visible and why it matters.
- Ask a question that follows the user from Home to Work without losing the relationship.
- Capture a thought as a note or candidate without silently accepting responsibility.
- Correct context, provenance, or Waldo's interpretation.
- Ask Waldo to prepare an exact native command for review.
- Resume an existing Outcome, Work Unit, session, or agent run.
- Delegate a bounded task and see its progress, evidence, and return path.
- Mention an explicitly created specialist without changing Waldo's identity.
- Answer an approval request with the affected scope and consequence visible.
- Return to the exact product state from which Waldo was opened.

## Information architecture

### One global launcher

A compact Waldo launcher lives at the top right of the shared application shell. It is visible in both Home and Work and has a stable accessible label such as “Open Waldo.” It may show a small non-color-only state marker for `Needs you`, active work, or unavailable connection, but it must not become a notification counter for every background event.

The launcher opens the same Waldo relationship wherever the user is. Surface context is attached to a message explicitly; the application does not create a new Waldo identity per route.

### One shell-owned rail

The shell owns the rail's open state, active tab, ephemeral draft, and focus return target. Feature pages may contribute context, but they do not create their own duplicate conversation implementations.

On wide Home layouts, opening Waldo temporarily occupies the existing contextual-right-pane position. Closing Waldo restores the previous Home context panel, including its selection and scroll position. The adaptive brief remains the primary content.

On wide Work layouts, Waldo participates in the existing inspector region as a peer tab: `Waldo | Inspector`. The product must not show two competing right rails at once.

On narrow layouts, Waldo becomes a full-content layer beneath the application chrome. A visible Back action returns to the unchanged underlying surface. The layer must not squeeze the Home brief or Work board into an unusable sliver.

### Home Today composition

Quick Capture and Waldo conversation have different jobs:

- Quick Capture is the fast, local-first place to record a thought.
- Waldo conversation is the place to reason, clarify, prepare a command, or coordinate work.

Home Today retains exactly one expanded Quick Capture. Opening Waldo must not add another expanded capture composer, clone Quick Capture into the rail, or auto-focus a second text field unless the user explicitly invoked Waldo.

### Work composition

Work remains the inspectable surface for Outcomes, Work Units, sessions, agent activity, evidence, and verification. Waldo may point to or operate through those native objects, but conversation must not replace their detailed views.

## Invocation and keyboard behavior

- Default in-app shortcut: `Command-Shift-Space` on macOS and `Control-Shift-Space` on Windows/Linux.
- The shortcut must live in the existing configurable shortcut catalog.
- `Command-Backtick` is not used because Kennel Island already owns it.
- The top-right launcher is always available for discovery and pointer access.
- A future system-wide invocation is a separate, explicitly permissioned capability and is outside the initial slice.

Opening by launcher returns focus to the launcher when closed. Opening by keyboard returns focus to the previously focused valid element. If that element no longer exists, focus returns to the nearest stable route heading or primary control.

`Escape` closes the rail only when no modal approval or destructive confirmation is active and the message composer is not consuming Escape for its own bounded behavior.

## Conversation model

### One relationship, multiple episodes

The product presents one ongoing relationship with Waldo, but does not force the user into one unbounded transcript. Conversations are durable episodes or topics under the same Waldo identity. A user can continue the current episode, start a focused one, or later search and re-enter an earlier episode.

Context compaction may change the model's working packet. It must never be presented as a new identity or as lossless memory. Durable transcript facts, admitted memory, and working context remain distinct.

### Explicit message context

Each user message carries a visible context chip before send. Examples include:

- `Home · Today`
- `Home · Open Loop: renew renter's insurance`
- `Work · Outcome: ship onboarding repair`
- `Work · Work Unit: validate migration recovery`
- `Session: backend verification`
- `No attached context`

The current surface may be suggested automatically, but the user can detach or change it before sending. The chip identifies what Waldo may use for that message; it does not grant authority to mutate that object.

### Native proposal cards

When a request would create responsibility or cause an effect, Waldo returns a native proposal card instead of treating prose as execution. The card displays:

- operation name;
- affected object or scope;
- exact material arguments;
- provenance and source context;
- expected consequence;
- whether the action is reversible;
- required approval or confirmation;
- available actions such as `Confirm`, `Edit`, `Keep as note`, and `Dismiss`.

A proposal card is not canonical. A suggestion click may reveal or edit it, but only the explicit native confirmation action may invoke the relevant service. Where policy requires an approval plan, the plan is frozen before execution and the eventual execution is bound to that approved plan.

Quick Capture may create a note or candidate. Only explicit user confirmation or a direct, unambiguous user command may create a canonical Open Loop. No model output, provider/session completion, activity observation, suggestion click, or UI render may confirm, close, release, or transfer responsibility.

## Agent model

### Task-scoped agents are the default

Most delegation creates a bounded `AgentRun` associated with an Outcome, Work Unit, or explicit conversation request. A task-scoped agent has:

- a stated purpose and completion condition;
- a bounded capability and data scope;
- a selected execution connection;
- a budget and time boundary;
- an approval policy;
- inspectable activity, artifacts, and failures;
- an explicit return destination to Waldo and the associated Work object.

Task-scoped agents do not become permanent navigation items or new personal identities. Their results remain proposals or evidence until the owning native domain accepts them.

### Persistent specialists are explicit

A persistent `AgentProfile` exists only when the user explicitly creates or pins one. Its configuration must expose purpose, capabilities, data scopes, admitted memory scope, execution connection, budget, approval policy, and pause/revoke/export/delete controls.

Persistent specialists belong in Work or Settings, not as a roster competing with the Home brief. They may be mentioned from Waldo with a visible `@specialist` reference. Waldo remains the relationship surface, and the specialist's output returns through Waldo and the relevant Work object.

No specialist may silently mutate personal truth, responsibility state, memory admission, or another agent's authority.

## Provider and runtime boundary

The product should model connections by the capability they provide rather than embedding a provider's identity into Waldo.

Two conceptual connection classes are sufficient for planning:

- **Inference connection:** produces model responses from a bounded context packet.
- **Agent-runtime connection:** can run bounded tools or subagents under Kennel's approval and capability envelope.

Initial and later connection paths may include:

1. An installed, already authenticated agent runtime. Credentials remain owned by that runtime; Kennel receives only the supported bridge and observable run contract.
2. A user-supplied API key stored through a governed secret boundary, never renderer storage or logs.
3. A local model connection in a later slice.
4. A paid, managed Waldo connection in a later slice.

Provider authentication never grants tool authority by itself. OAuth is used only where the provider officially supports the intended delegated access; it is not simulated by scraping or importing session credentials. There is no silent fallback from one provider or runtime to another.

The renderer must never talk directly to model providers or agent runtimes.

## Monetization boundary

Monetization should sell convenience, reliability, and execution capacity—not custody of the user's responsibility graph or a second Waldo identity.

An initial local/free product can include Home, Open Loops, Quick Capture, native Work surfaces, and user-connected execution through an installed runtime or BYOK connection.

A later managed plan may include managed inference, durable background execution, higher concurrency, better availability, and governed persistent specialists. Cross-device or hosted continuity comes only after separate custody, export, revocation, privacy, and conflict-resolution gates.

Even with a managed plan:

- local SQLite responsibility truth remains canonical;
- the managed service receives minimized, purpose-bound context packets;
- account loss or plan cancellation does not erase local responsibility data;
- managed completion cannot close local responsibility;
- the user can inspect which connection performed a run.

## Truthful states

The rail needs explicit projections for:

- unconfigured;
- connecting;
- ready and idle;
- streaming a response;
- tool or subagent activity;
- waiting for approval;
- waiting for user input;
- rate limited;
- provider unavailable;
- offline;
- interrupted but resumable;
- unknown execution outcome;
- result ready.

Unavailable or unknown states must not masquerade as a friendly empty conversation or a completed action. Failure copy should state what is known, what is unknown, what may have happened, and the safest next action.

Deterministic conversation fixtures are permitted only in renderer tests and an explicitly labeled preview route or build mode. Canonical desktop routes must show truthful unconfigured/unavailable states until daemon facts exist; they must never display plausible fabricated chat, activity, or agent results.

## Proposed component ownership

These names describe responsibilities for planning; they do not require one file per name.

- `WaldoLauncher` — global entry point and small state marker.
- `WaldoRailHost` — shell integration, wide/narrow placement, tab participation, and focus restoration.
- `WaldoRail` — rail chrome, header, connection state, and conversation boundary.
- `WaldoConversation` — message and episode projection.
- `WaldoContextChip` — visible, editable message context.
- `WaldoProposalCard` — non-authoritative native-command proposal and confirmation entry point.
- `WaldoActivityCard` — bounded run status, evidence, failure, and return path.
- `WaldoConnectionState` — explicit unavailable, offline, rate-limit, and unknown states.

The Home components merged in PR #47 remain the source components. The existing Home context panel becomes a participant in the shell's right-region composition; no duplicate Home or Work screen is created.

Renderer-owned state is limited to ephemeral presentation concerns such as open/closed state, selected rail tab, unsent draft, and focus return. Durable conversations, messages, agent runs, approvals, and command results require daemon-owned services before they can appear as canonical facts.

## Layout, motion, and accessibility

- The visual language stays calm, warm, and editorial, consistent with merged Home Today.
- The adaptive brief remains visually primary; Waldo is available, not permanently demanding attention.
- The wide rail should be approximately 380–440 pixels where space permits, with the main surface retaining a tested minimum useful width.
- At narrow widths, use a full-content layer instead of compressing two panes.
- Opening and closing may use a short opacity/translation transition. Reduced-motion mode removes translation and uses an immediate or near-immediate state change.
- Keyboard invocation moves focus to the Waldo composer only after the rail is mounted and named to assistive technology.
- Closing restores focus deterministically.
- State, unread, and approval indicators use text or accessible labels in addition to color.
- Streaming content must not steal scroll position when the user has moved away from the newest message.
- A pending unsent draft prompts or persists ephemerally according to a tested rule; it is never mislabeled as durable conversation history.

## Future daemon data flow

```text
UI intent
  -> generated daemon client
  -> conversation service
  -> provider-neutral run port
  -> installed runtime / BYOK adapter / managed adapter

provider and runtime events
  -> durable conversation and agent facts
  -> SQLite trigger-backed change_log
  -> CDC invalidation
  -> rail projection

explicit proposal confirmation
  -> native responsibility or effect service
  -> durable domain fact
  -> trigger-backed change_log
  -> Home / Work / Waldo projection refresh
```

The future implementation must follow controller → service → port → SQLite store. It must use generated OpenAPI and TypeScript contracts, idempotent writes, explicit revision conflicts, and trigger-backed CDC. The renderer cannot create a parallel provider transport, event store, responsibility model, or manual CDC channel.

## Sequenced delivery

### Slice A — truthful UI shell and preview

- global top-right launcher;
- shell-owned rail placement in Home and Work;
- wide, narrow, keyboard, focus-return, and reduced-motion behavior;
- honest unconfigured/unavailable canonical state;
- deterministic proposal, activity, and conversation fixtures only in test/explicit preview;
- shortcut-catalog integration;
- reuse of merged Home components.

This slice includes no daemon API, model call, provider connection, agent execution, or persistence. It cannot claim conversation is functional.

### Slice B — durable conversation foundation

- daemon-owned conversation and message domain;
- service, port, SQLite store, migrations, trigger CDC, and generated contracts;
- restart durability, idempotent sends, interruption semantics, and read-back;
- provenance and context attachment.

### Slice C — governed connections

- provider-neutral inference/run port;
- installed authenticated runtime bridge and/or BYOK according to an approved threat model;
- secret custody, connection tests, rate-limit/offline states, and no silent fallback.

### Slice D — bounded agent runs and approvals

- task-scoped runs;
- capability and data scopes;
- frozen approval plans;
- activity, evidence, recovery, and return-to-Waldo behavior.

### Slice E — persistent specialists

- explicit creation and mention;
- purpose, scope, memory, connection, budget, pause, revoke, export, and delete controls;
- Work/Settings management surface.

### Slice F — managed Waldo plan

- only after local custody, hosted-context minimization, account separation, export, revocation, and conflict behavior are separately approved.

Issue #23 remains a separate dependency-bound lane for daemon-owned Personal Home, durable Quick Capture, and explicitly confirmed Open Loops. This design must neither invent a Home-specific responsibility identity nor bypass the shared contract from issue #21.

## Initial UI acceptance criteria

The Slice A implementation plan must include automated tests and visual checks proving:

1. The same launcher is available on Home and Work.
2. The configured shortcut opens and closes Waldo without conflicting with Island.
3. Home Today still has exactly one expanded Quick Capture.
4. Wide Home uses the contextual-right region and restores the previous context panel exactly.
5. Wide Work uses one right region with `Waldo | Inspector`, not two rails.
6. Narrow Waldo is a full-content layer with a visible Back action.
7. Route, selection, scroll, and focus restore after close.
8. Reduced-motion behavior contains no required translation animation.
9. Message context is visible, detachable, and editable before send in preview tests.
10. A proposal card cannot invoke a mutation through suggestion, render, or ordinary card selection.
11. Explicit preview fixtures cannot appear on canonical desktop routes.
12. Unconfigured, provider unavailable, rate limited, offline, interrupted, and unknown-outcome states are distinguishable.
13. Canonical UI never presents an agent run, response, or completion without daemon evidence.
14. The existing Home components and visual hierarchy are reused rather than duplicated.

## Risks and falsifiers

The design should be reconsidered if any of the following are true in testing:

- Users treat the rail as the primary dashboard and stop trusting native Home or Work facts.
- Users cannot distinguish Quick Capture from conversation.
- Users cannot tell a proposal from a canonical responsibility or completed effect.
- A provider or specialist appears to be a different Waldo identity.
- Connection setup obscures credential custody or silently expands tool authority.
- Opening the rail regularly destroys useful Home or Work context.
- The managed plan makes local data unusable, unexportable, or subordinate to an account.
- Narrow layouts turn the rail into an accidental modal trap.
- Agent activity is shown without scope, evidence, failure, or an exact return path.

## Explicit non-goals

This design and its initial UI slice do not implement:

- issue #23 backend persistence;
- capture monitoring, Omi, or Dayflow ingestion;
- durable Memory or memory admission;
- Daily Close persistence;
- Home-to-Work `ResponsibilityLink`;
- Paxel or AutoResearch;
- BYOK execution or provider OAuth;
- hosted, cloud, mobile, health, release, or billing behavior;
- onboarding preferences;
- persistent agent profiles;
- background agent execution;
- any push, PR, merge, deployment, release, publication, or GitHub mutation.

## Before / after / why

| Before | After | Why |
| --- | --- | --- |
| Waldo is implied by separate product surfaces | One stable top-right Waldo entry point across Home and Work | Makes the relationship continuous without merging the responsibility-space lenses |
| Home's right pane and Work's inspector own unrelated space | One shell-composed right region that can host Waldo or the native context | Prevents duplicate rails and preserves the existing information hierarchy |
| A chat request could be mistaken for execution | Material requests become inspectable native proposal cards | Keeps authority, arguments, consequence, and confirmation visible |
| “Subagent” risks becoming a roster of personalities | Bounded task runs are the default; persistent specialists are explicit | Preserves one Waldo identity and avoids permanent navigation clutter |
| Provider choice risks defining the product | Connections are replaceable inference/runtime capabilities behind Waldo | Protects portability, custody, and future monetization choices |
| A UI mock can look plausibly functional | Canonical routes show honest unavailable states; fixtures stay in tests/preview | Maintains the product's provenance and truth boundary |

## Review checkpoint

Written approval of this document is required before creating the Slice A implementation plan. The plan must then name exact files, red/green tests, visual checkpoints, and local conventional-commit boundaries. Production UI code must not begin from this document alone.

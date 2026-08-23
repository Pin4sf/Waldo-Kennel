# Waldo Product Surface Completion — Design Checkpoint

**Date:** 2026-08-23

**Status:** Approved incremental completion boundary after the Personal Agency rail checkpoint; implementation remains local

**Landed baseline:** `origin/beta` at `917dbe65c` (PR #52 squash merge; tree `795b9cf8d9fcd648b3957164bf87e518a5ab9f86`)

**Scope:** Kennel desktop renderer, product information architecture, and truthful preview seams

## Current

The stacked renderer now has a global Waldo relationship across Home and Work, a Home/Work switch, Home Today and continuity destinations, Conversation and Activity rail modes, contextual episodes, detachable context, structured semantic cards, bounded-run inspection, responsive behavior, keyboard access, and canonical no-preview honesty.

That checkpoint landed with a broken broad regression foundation: the fresh beta baseline had 33 passing and 35 failing Playwright cases despite the unit, typecheck, build, and package-identity gates passing. Local P0 diagnosis and TDD repair first restored all 69 existing Playwright cases; the completed preview surfaces now produce 74 passing cases. Source regressions were the centered Home/Work switch intercepting Work titlebar controls, the Inspector's 280px clamp violating the established 350px contract, chat-pane horizontal overflow, and the board's missing New terminal action. Harness regressions were the obsolete `window.ao` bridge key, missing onboarding completion, unserved agent-catalog requests, and terminal tests racing terminal-mux attach/replay. The remaining clusters were stale routes, fixture names, accessible roles, labels, or exact state assumptions; those assertions were updated only after the rendered product contract was inspected. Terminal retention also needed its lifecycle waits restored and its finite Chromium scroll-event ceiling clarified. The same full gate is now green. Physical public-UI/accessibility inspection now covers the wide Chat, preview send/source flow, Activity specialist workspace, and responsive inspector collapse; system-level VoiceOver, Full Keyboard Access, inactive-window, trackpad gesture, and Reduce Motion toggles remain an explicit follow-up rather than a claimed pass.

The remaining surface is split across two partially reconciled models:

- Home has `Today`, `Open Loops`, `Daily Close`, `Memory`, and `History`.
- Waldo conversation is globally available as a rail but is not an explicit Home destination.
- `History` is a continuity and supporting-evidence ledger, not a daily synthesis surface.
- Work can host multiple runtime agents, but Home does not yet explain how user-created specialists relate to Waldo.
- inherited AO presentation assets and copy remain in parts of the desktop even though Waldo is the product relationship and Kennel is its desktop harness.

The first local completion pass technically added Home Chat and a specialist builder, but the full-page Chat destination still reused the narrow Waldo rail as its entire layout. That made the route available without making it a legible place to resume a relationship, inspect the sources behind an answer, or hand bounded work to Activity. The specialist contract existed below Activity, but not yet as the Grok Bot-like selection, run, and evidence workspace the user approved. These are product-surface gaps, not backend gaps.

## Ideal

Home should answer four personal-agency questions:

1. **Today:** what deserves attention now?
2. **Chat:** what do I want to discuss, clarify, redirect, or ask Waldo to coordinate?
3. **Open Loops:** what responsibility remains unresolved?
4. **Insights:** what evidence-bounded pattern may help me act differently?

Daily Close and Memory remain review rituals. Continuity records remain inspectable evidence below Insights rather than competing as a top-level history archive.

Waldo remains the only global personal-agent relationship. A user-created specialist is an explicit, scoped `AgentProfile` coordinated by Waldo. It may be pinned as a shortcut or used in Work, but it must not appear as a peer identity that fragments Home into competing personal agents.

## Information architecture

### Home

| Destination | Primary job | Truth boundary |
| --- | --- | --- |
| Today | Orient and choose the next meaningful action | no invented urgency or readiness |
| Chat | Continue the same Waldo relationship shown in the global rail | no duplicate transcript or second Waldo state |
| Open Loops | Review accepted responsibilities and unknown outcomes | observations and proposals are not responsibilities |
| Insights | Review daily-use-derived candidates with sources and freshness | activity is evidence, not personality or intent |
| Daily Close | Consciously close, defer, or carry work | provider completion is not closure |
| Memory | Confirm, correct, or reject memory candidates | inference is never silently admitted |

### Insights

An Insight is a reviewable synthesis candidate, not a score. Its first reading path is:

`observation -> source window -> why it may matter -> suggested next judgment`

Each item exposes:

- source events and their freshness;
- whether the statement is `Noticed`, `Candidate`, `Confirmed`, `Corrected`, or `Dismissed`;
- what is directly observed versus inferred;
- known gaps and the synthesis/provider route;
- `Confirm`, `Correct`, `Dismiss`, and `Why this?` controls in preview;
- optional routing to an Open Loop, Memory proposal, or bounded experiment;
- the underlying continuity/evidence record.

No Insight may infer mood, personality, health status, commitment, or outcome from app usage. No opaque confidence score or surveillance-style time ranking is rendered.

### Chat and specialists

`Chat` is a full Home reading surface for the same Waldo conversation model as the rail. Opening or closing the rail changes presentation, not identity, topic ownership, or authority.

Specialists are subordinate, explicit profiles with a purpose, permitted sources/tools, authority ceiling, and revoke path. They appear in Work, Settings, and bounded Waldo delegation cards. A compact Home shortcut may be added only after the user creates or pins one. Waldo remains responsible for coordination and return-path presentation.

The approved full Chat composition has three responsive regions:

1. a compact episode rail for starting, finding, pinning, and resuming Waldo topics;
2. the primary conversation with a short statement of purpose, source-linked answers, inline proposals/approvals, and an attachment/voice-aware composer;
3. an optional context inspector showing attached sources, freshness, gaps, artifacts, and the consequence of detaching context.

At medium widths the optional inspector collapses before the transcript becomes cramped. At narrow widths the episode rail and inspector become labelled layers reached from the conversation; they never remain squeezed into three columns.

The approved Activity → Agents composition is not a second chat product. A compact specialist rail sits beside a selected run timeline and an optional evidence/authority inspector. Waldo is fixed as coordinator. Every specialist and run names its purpose, scope, sources/tools, authority, budget, completion condition, evidence expectation, current state, and return destination. Pause, resume, stop, revoke, retry, and recovery are explicit preview controls; none implies execution in this renderer-only phase.

### Dimension and Omi product consequence

The live first-party evidence resolves the requested “Dimensions” reference to **Dimension** (singular). Dimension officially wound down on May 20, 2026, so its still-live product pages are archival design evidence rather than proof of an available runtime. Dimension contributes the finite Morning Briefing rhythm, one primary agent relationship, searchable chat threads, source-retaining Catch Up, background Skills, and the inline `Accept / Reject / Edit / Respond` decision grammar.

Omi contributes the stronger continuity lineage: one conversation can ground a transcript, summary, insight, task candidate, and memory candidate; typed citations can return to Conversation, Memory, Task, Goal, screenshot, web, or an explicitly unavailable source. Omi's current macOS source also collapses its primary navigation when the full row cannot fit. Its specialist Apps can replace the visible chat persona, so Waldo adapts their capability disclosure but rejects persona switching as the relationship model.

Together with the installed-app benchmark, the approved synthesis is:

- **Today:** a finite Dimension-like orientation, not a dashboard score;
- **Chat:** an Omi-like query and continuity surface, but always the same Waldo relationship;
- **Insights / Records:** typed, openable lineage from source episode to correctable derived candidate;
- **Activity → Agents:** Grok Bot-like specialist discovery and conversation cards, governed by the Waldo purpose/authority/evidence/return contract;
- **Work:** execution, artifacts, evidence, verification, and acceptance lineage rather than a duplicate personal chat.

Omi-extracted tasks and memories remain candidates until the user confirms them. Dimension-style approvals cannot be globally disabled for consequential effects. Neither benchmark changes the renderer-only, no-provider, no-persistence boundary.

## Installed specialist benchmark decision

The local 2026-08-23 public-UI/accessibility audit of `/Applications/Grok Bot.app` is the primary specialist benchmark; the running Hermes desktop app is secondary. No application database, credentials, provider state, private API, network payload, DevTools state, message send, creation flow, or setting mutation was used.

- **Adopt from Grok Bot:** compact specialist discovery, explicit `New Bot` versus `New Channel`, cross-surface search, structured conversation cards, stale/offline recovery, attachment/voice affordances, and an inspectable computer/evidence entry.
- **Adapt from Grok Bot:** a bot becomes a preview-only scoped specialist under Waldo Activity; approvals must name action, scope, consequence, and recovery; secure connection input remains unavailable until native custody exists.
- **Adopt from Hermes:** explicit profile/session boundaries, searchable capability configuration, lifecycle copy such as “applies to new sessions,” safety controls, queue/steer/cancel grammar, and discoverable shortcuts.
- **Adapt from Hermes:** profiles require purpose, scope, authority ceiling, budget, evidence, revoke/pause, completion condition, and return destination. Skills, Tools, MCP, provider, model, and persona vocabulary remain inspectable configuration rather than Home navigation.
- **Reject from both:** peer personal-agent identities, plugin/capability sprawl as the everyday product, automatic durable memory, unlabeled progress counters, and configuration success presented as verified work.

## Completion criteria

1. The desktop shows Waldo product branding and Kennel harness wording; no user-visible AO asset or copy remains.
2. One development launch produces one main Electron app process. Auxiliary Island renderers and the daemon are labeled and diagnosed as parts of that app, not separate builds.
3. `Insights` replaces `History` in Home navigation and renders source-backed, correctable preview candidates; continuity records remain available underneath it.
4. `Chat` is an explicit Home destination that shares the global Waldo relationship and does not fork conversation state.
5. Specialists are user-created scoped profiles coordinated by Waldo, not a peer personal-agent roster.
6. Wide and narrow layouts preserve internal scrolling, no horizontal overflow, Back behavior, Work Inspector return, keyboard access, focus restoration, pointer reachability, context-menu actions, and reduced-motion behavior.
7. Without an explicit preview seam, canonical desktop shows no plausible messages, insights, runs, progress, or outcomes.
8. Renderer controls remain thin and local until durable API, persistence, provider, approval, run, evidence, and verification contracts exist.
9. Home Chat is usable as a destination: episodes, transcript, composer, and source/context inspection have a clear hierarchy at wide and narrow widths.
10. Activity → Agents shows specialist selection, bounded run state, authority, evidence, recovery, and return path while keeping Waldo visibly responsible for coordination.

The deterministic preview acceptance story is: ask “What changed in the pricing workshop and what still needs me?”, inspect a concise answer with source lineage, correct or detach context, approve only a bounded next step, follow the specialist in Activity, inspect evidence, and return to the original responsibility. In canonical mode the same route must disclose that no provider or durable transcript is connected and render no plausible answer or run.

## Gaps

- **Regression foundation:** the landed 33/35 gate is classified and restored; the completed local surface runs 73/0. The remaining P0 work is physical macOS interaction verification and any source issue that evidence reveals there.
- **macOS interaction grammar:** automated coverage now proves browser-style Back/Forward, draft-safe navigation, focus restoration, keyboard tab switching, pointer reachability, internal scrolling, responsive overflow, and Reduce Motion rendering. Native menus, secondary click, trackpad gestures, Full Keyboard Access, VoiceOver order/announcements, and inactive-window legibility still need physical verification after the desktop session is unlocked.
- **Responsive composition:** Playwright proves the supported 960px minimum and 1512px wide layouts for Home, Chat, Waldo Activity, Insights/Records, specialist creation, Work controls, Inspector width, scrolling, and horizontal overflow. Short, medium, and ultrawide physical window inspection remains outstanding.
- **Brand layer:** Figma mark, launcher, sidebar, startup, app, and tray assets are being reconciled; visible AO copy requires a catalog/runtime sweep.
- **Insights:** the local preview now supplies source-window candidates, observation/inference boundaries, known gaps, an explicit deterministic-fixture/no-provider disclosure, Confirm/Correct/Dismiss/Why review, an honest unconfigured state, and nested Records. No durable synthesis contract exists.
- **Chat:** Home now has a dedicated destination sharing Waldo mode, episode, attached-context state, and unsent draft with the global rail. No message transport or durable transcript exists.
- **Specialists:** Waldo Activity now has one preview-only governed creation/profile surface with purpose, explicit scope, sources/tools, authority ceiling, budget, completion, evidence, return destination, pause, and revoke. No durable `AgentProfile`, capability connection, execution, or pinning exists.
- **Runtime truth:** preview controls cannot send, save, approve, execute, or verify.
- **Persistence:** conversation episodes, attached context, insights, responsibilities, runs, approvals, evidence, and outcomes have no durable Personal Agency storage/API.
- **Connection:** there is no provider-neutral secure connection flow or Keychain-backed credential boundary.

## Anti-criterion

The product surface is not complete if it merely looks populated: no plausible daily Insight, chat history, specialist, progress, responsibility, or outcome may appear without a visible preview boundary or native evidence. API tokens must never be pasted into chat, stored in renderer state, or committed to the repository.

## Product decision

Proceed in this approved order: restore the responsive/macOS interaction foundation and broad regression gate; add dedicated Home Chat using the same Waldo relationship; replace visible History with truthful preview Insights plus Records; then add Grok Bot-informed, Hermes-governed preview specialists under Waldo Activity. Specialists are coordinated workers under Waldo, not a competing sidebar of personal-agent identities.

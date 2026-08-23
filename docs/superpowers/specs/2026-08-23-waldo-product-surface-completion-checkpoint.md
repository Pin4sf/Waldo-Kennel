# Waldo Product Surface Completion — Design Checkpoint

**Date:** 2026-08-23

**Status:** Proposed completion boundary after the Personal Agency rail checkpoint

**Scope:** Kennel desktop renderer, product information architecture, and truthful preview seams

## Current

The stacked renderer now has a global Waldo relationship across Home and Work, a Home/Work switch, Home Today and continuity destinations, Conversation and Activity rail modes, contextual episodes, detachable context, structured semantic cards, bounded-run inspection, responsive behavior, keyboard access, and canonical no-preview honesty.

The remaining surface is split across two partially reconciled models:

- Home has `Today`, `Open Loops`, `Daily Close`, `Memory`, and `History`.
- Waldo conversation is globally available as a rail but is not an explicit Home destination.
- `History` is a continuity and supporting-evidence ledger, not a daily synthesis surface.
- Work can host multiple runtime agents, but Home does not yet explain how user-created specialists relate to Waldo.
- inherited AO presentation assets and copy remain in parts of the desktop even though Waldo is the product relationship and Kennel is its desktop harness.

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
- `Confirm`, `Correct`, `Dismiss`, and `Why this?` controls in preview;
- optional routing to an Open Loop, Memory proposal, or bounded experiment;
- the underlying continuity/evidence record.

No Insight may infer mood, personality, health status, commitment, or outcome from app usage. No opaque confidence score or surveillance-style time ranking is rendered.

### Chat and specialists

`Chat` is a full Home reading surface for the same Waldo conversation model as the rail. Opening or closing the rail changes presentation, not identity, topic ownership, or authority.

Specialists are subordinate, explicit profiles with a purpose, permitted sources/tools, authority ceiling, and revoke path. They appear in Work, Settings, and bounded Waldo delegation cards. A compact Home shortcut may be added only after the user creates or pins one. Waldo remains responsible for coordination and return-path presentation.

## Completion criteria

1. The desktop shows Waldo product branding and Kennel harness wording; no user-visible AO asset or copy remains.
2. One development launch produces one main Electron app process. Auxiliary Island renderers and the daemon are labeled and diagnosed as parts of that app, not separate builds.
3. `Insights` replaces `History` in Home navigation and renders source-backed, correctable preview candidates; continuity records remain available underneath it.
4. `Chat` is an explicit Home destination that shares the global Waldo relationship and does not fork conversation state.
5. Specialists are user-created scoped profiles coordinated by Waldo, not a peer personal-agent roster.
6. Wide and narrow layouts preserve internal scrolling, no horizontal overflow, Back behavior, Work Inspector return, keyboard access, and focus restoration.
7. Without an explicit preview seam, canonical desktop shows no plausible messages, insights, runs, progress, or outcomes.
8. Renderer controls remain thin and local until durable API, persistence, provider, approval, run, evidence, and verification contracts exist.

## Gaps

- **Brand layer:** Figma mark, launcher, sidebar, startup, app, and tray assets are being reconciled; visible AO copy requires a catalog/runtime sweep.
- **Insights:** current History implementation stores evidence but has no synthesis candidate model, source window, correction workflow, or honest empty state.
- **Chat:** the rail exists, but Home has no dedicated reading destination or shared presentation contract.
- **Specialists:** Work has provider/runtime agents, but there is no Personal Agency `AgentProfile` contract, creation flow, authority ceiling, or revoke lifecycle.
- **Runtime truth:** preview controls cannot send, save, approve, execute, or verify.
- **Persistence:** conversation episodes, attached context, insights, responsibilities, runs, approvals, evidence, and outcomes have no durable Personal Agency storage/API.
- **Connection:** there is no provider-neutral secure connection flow or Keychain-backed credential boundary.

## Anti-criterion

The product surface is not complete if it merely looks populated: no plausible daily Insight, chat history, specialist, progress, responsibility, or outcome may appear without a visible preview boundary or native evidence. API tokens must never be pasted into chat, stored in renderer state, or committed to the repository.

## Product decision

Adopt the requested dedicated Home Chat and specialist capability with one adjustment: specialists are coordinated workers under Waldo, not a competing sidebar of personal-agent identities. This preserves the approved one-Waldo relationship while supporting Hermes/Grokbot-style custom capability profiles.

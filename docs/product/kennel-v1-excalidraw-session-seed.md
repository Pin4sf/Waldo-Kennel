# Waldo Kennel v0 dogfood and provider-neutral v1 Excalidraw session seed

- Purpose: team critique of the complete local Home + Work v0 dogfood and provider-neutral v1 seam
- Source of truth: [team review packet](kennel-v1-team-review-packet.md)
- Clickable reference: [low-fidelity prototype](kennel-v1-review-prototype.html)
- Status: facilitation seed, not final visual design
- Session length: 120 minutes

Keep the board grayscale. Put **Locked**, **Observed**, **Proposed**, or **Unknown** on every non-obvious claim. Use solid borders for Locked, an eye marker for Observed, dashed borders for Proposed, and a hatched corner for Unknown. Never use color as the only state signal.

## 1. Board layout

Build five left-to-right zones with horizontal lanes.

### Zones

1. **Enter** — onboarding, Home, connections, Project readiness.
2. **Understand** — Daily Snapshot, Communication Brief, Quick Capture, Outcome Define.
3. **Decide and authorize** — candidate confirmation, Mission Map, authority/effect preview.
4. **Act and observe** — v0 Codex Work Units, provider-adapter seam, draft effect, attention, Waiting, recovery.
5. **Prove and close** — Evidence, Verification, Acceptance, Open Loop closure, Daily Close, Re-entry.

### Lanes

1. **User responsibility** — what the user believes is open and which decision belongs to them.
2. **Home surface** — personal continuity, communication, Open Loops, Daily Snapshot.
3. **Work surface** — Outcomes, plans, Work Units, Evidence, Acceptance.
4. **Local Waldo Core** — contracts, authority, decisions, effects, proof, closure, lineage.
5. **Kennel Runtime** — Project, worktree, provider adapters, connector, capture, recovery.
6. **Evidence, privacy, and unknowns** — sources, disclosure, falsifiers, later boundaries.

## 2. Shared legend

| Shape | Meaning | Example |
| --- | --- | --- |
| User silhouette | Human authority | AcceptanceDecision, LoopDisposition |
| Rounded rectangle | Product surface/projection | Communication Brief, Daily Snapshot |
| Solid rectangle | Canonical object | Outcome, OpenLoop, WorkUnit |
| Document | Immutable revision | ContractRevision 2 |
| Gear | Runtime fact | provider adapter session, Gmail sync |
| Paperclip | Evidence/provenance | test result, thread reference |
| Shield | Capability/effect boundary | write grant, draft approval |
| Dashed arrow | Candidate/admission path | CommitmentCandidate to OpenLoop |
| Solid arrow | Canonical lineage | Outcome to ContractRevision |
| Dotted arrow | Later/optional | ContextEpisode to MemoryCandidate |

## 3. Primary product map

Place this at the top of the board:

```text
WALDO KENNEL DESKTOP

HOME                                  WORK                               SETTINGS
Today                                 Projects                           Spaces
Needs You / Action Required           Outcome Focus                      Codex
Open Loops / Waiting                  Define / Clarify                   Gmail
Communication                         Mission Map / Authority             Permissions
Daily Snapshot / Quick Capture        Run / Review / Accept               Privacy
Ready to Close / Re-entry             Close / Re-entry                   Desktop Context beta
```

Add a note: **One Waldo identity. Home and Work are projections over shared responsibility truth.**

## 4. Frame inventory

Each frame should fit on one laptop screen at normal zoom and have one obvious primary action.

### F01 — First run

- Local Personal Home created;
- optional “Add Work Project”;
- local custody statement;
- Gmail and Desktop Context shown as optional Later/Connect actions;
- primary: **Enter Home**.

States: first run, offline daemon, storage unavailable.

### F02 — Home / Today

- concise Today brief;
- Needs You and Action Required;
- Open Loops and Waiting;
- Communication candidates;
- current Outcomes and Ready to Close;
- primary: **Handle top item**.

Review: can the user understand the next useful intervention in five seconds?

### F03 — Quick Capture

- ordinary-language capture;
- choose note, Open Loop, or Outcome candidate;
- assign Personal Home or Work Project;
- show suspected duplicate;
- primary: **Preserve this**.

### F04 — Daily Snapshot

- what became true;
- what remains open;
- what changed since last visit;
- source confidence/provenance;
- corrections and Daily Close;
- primary: **Review what remains open**.

States: collecting, partial source, ready, corrected, closed for day.

### F05 — Communication connection

- connect one Gmail account;
- exact read/draft scope;
- model disclosure destination;
- retention and revoke;
- no auto-send/labels/archive/delete;
- primary: **Authorize selected scope**.

States: disconnected, authorizing, denied, syncing, ready, stale, revoked.

### F06 — Communication inbox

- only actionable candidates, not full inbox;
- Needs reply, commitment detected, waiting, follow-up, possibly resolved;
- sync health and freshness;
- primary: **Open brief**.

### F07 — Communication Brief

- actual ask in one sentence;
- who owes what;
- due/trigger;
- what happened;
- Waldo recommendation and why;
- source inspect;
- actions: dismiss, correct, Open Loop, Outcome, draft.

### F08 — Draft effect review

- exact To/Cc/subject/body/thread;
- why this response;
- disclosure/provenance;
- EffectIntent digest;
- primary: **Create Gmail draft**;
- user sends outside launch automation.

States: proposed, approved, created, edited, failed, unknown/reconciled, sent externally.

### F09 — Open Loop detail

- responsibility statement;
- owner;
- provenance/source;
- next review/trigger;
- closure condition;
- linked Outcome;
- actions: active, waiting, defer, transfer, release, close.

### F10 — Ready to Close

- what changed;
- why closure appears satisfied;
- remaining uncertainty;
- primary: **Close Loop**;
- alternatives: reopen, release, promote remaining work.

### F11 — Work Home

- Project selector/readiness;
- Outcome Focus;
- Active Outcomes;
- Needs You, Action Required, Waiting;
- Ready for Acceptance;
- sessions absent from primary hierarchy.

### F12 — Outcome Define

- Goal, Success, Review always;
- constraints, non-goals, stop conditions;
- Plan/Authority expand only when warranted;
- primary: **Clarify material gaps**.

### F13 — Adaptive clarification

- one irreducible question;
- recommendation and rationale;
- materially different choices;
- contract/authority/verification impact;
- primary: **Apply choice**.

### F14 — Mission Map

- direct one-WorkUnit variant and non-trivial graph;
- dependencies, evidence, verification, topology rationale;
- v0 Codex-only routing, visibly labeled as local dogfood rather than a v1 provider decision;
- provider-neutral RunBrief core plus adapter-compiled form; required versus optional capability profile;
- no silent provider fallback; later handoff is a new Attempt after adapter admission and any review required by a material change;
- smallest-sufficient direct default, with model-proposed topology checked by an inspectable deterministic policy;
- autonomy-preserving lease: `unconfirmed` is not dead, and fences guard only canonical writes/effects;
- multidimensional budget and recovery boundary;
- truthful evaluator tiers: deterministic, self-check, separate-session, cross-provider/model, owner walkthrough;
- primary: **Review authority**.

### F15 — Authority and effect preview

- read, write, execute, disclose, spend, effect rows;
- Project/worktree placement;
- expiry/revision binding;
- explicit exclusions;
- local commit if included;
- remote effects require later approval;
- primary: **Approve revision and run**.

### F16 — Run

- Outcome responsibility and current Work Unit;
- Attempt/fence/lease summary;
- Waldo's next safe action;
- Evidence progress;
- recovery receipt;
- primary: **Pause** or no action when healthy.

### F17 — Needs You

- problem;
- recommendation and why;
- consequence;
- primary choice, override, inspect;
- never a generic error.

### F18 — Action Required

- exact human-only action;
- location;
- reason;
- completion signal;
- resume behavior.

### F19 — Waiting

- dependency;
- owner/source;
- recheck condition;
- release behavior;
- revise/release, but no pressure to refresh.

### F20 — Evidence and Verification

- current ContractRevision and subject;
- one row per criterion;
- supporting and contradicting Evidence;
- verifier class and independence;
- failures/exceptions;
- primary: **Decide acceptance**.

### F21 — Acceptance

- contract summary;
- Evidence/Verification summary;
- exceptions and unknown effects;
- accept, rework, revise active, release;
- primary: **Accept Outcome**;
- user authority marker.

### F22 — Adaptive Close

- immutable close receipt;
- worktree/artifact disposition;
- remaining Open Loops;
- suggested successor;
- primary: **Finish close**.

### F23 — Re-entry

- predecessor/source;
- what became true;
- what remains open;
- exact workspace/thread return target;
- inherited Evidence references;
- new Outcome or reopened Open Loop;
- primary: **Continue from here**.

### F24 — Operator inspector

- Attempt, terminal, worktree, browser, trace, recovery;
- privacy warning for raw sources;
- facts never accept or close responsibility;
- contextual drawer, not top-level nav.

### F25 — Desktop Context consent

- capture toggle and visible pause;
- app exclusions and secret-app defaults;
- analysis route and provider disclosure;
- retention/storage cap/delete;
- “observations are untrusted” statement;
- primary: **Enable selected context**.

States: disabled, denied, paused, excluded app, storage cap, provider unavailable, delete pending.

### F26 — Context episode correction

- time range and apps;
- derived description;
- edit/dismiss/split/merge;
- explicit link to Snapshot or candidate;
- no automatic Evidence/OpenLoop/Memory/skill;
- primary: **Confirm selected use**.

### F27 — Settings & Control

- Responsibility Spaces and Projects;
- v0 Codex readiness, provider-adapter capability profiles, and historical providers;
- Gmail connection/sync/revoke;
- grants/effects/disclosure;
- retention/export/delete;
- skills/MCP/rules advanced;
- Desktop Context beta;
- hosted attachment Later.

## 5. Canonical lineage strips

### Work

```text
ResponsibilitySpace -> Project -> Outcome -> ContractRevision
-> PlanRevision / Mission Map -> CapabilityGrants -> WorkUnits
-> fenced Attempts -> Evidence -> Verification -> Ready for Acceptance
-> user Acceptance -> Adaptive Close -> successor/Re-entry
```

### Communication

```text
Gmail Connection -> ThreadRef -> Communication Brief
-> CommitmentCandidate -> user confirms OpenLoop or Outcome
-> optional Draft EffectIntent/Receipt -> Waiting
-> reply/change -> Ready to Close -> user LoopDisposition -> Re-entry
```

### Daily continuity

```text
trusted Kennel facts + confirmed OpenLoops + explicit notes
-> Daily Snapshot -> correction -> Today attention -> Daily Close -> Re-entry
```

### Optional context

```text
raw frame -> DesktopObservation -> ContextEpisode
-> explicit confirmation/link -> Snapshot input or CommitmentCandidate
-> MemoryCandidate later only
```

Cross out these invalid shortcuts:

- session done -> Outcome accepted;
- green check -> Acceptance;
- thread archived -> Open Loop closed;
- model extracted TODO -> canonical commitment;
- screenshot -> Evidence or Memory;
- draft created -> message sent;
- app activity -> productivity/personality claim.

## 6. State-machine mini-board

```text
Outcome
Draft -> Contracted -> Active -> Ready for Acceptance -> Accepted
                      |                 -> Reopened -> Active
                      -> Superseded / Released

OpenLoop
Open -> Active -> Waiting -> Ready to Close -> Closed
  |       -> Deferred         -> Reopened
  -> Released / Superseded / Transferred

Connection
Disconnected -> Authorizing -> Syncing -> Ready -> Stale
                -> Denied                         -> Revoked

Attempt
Queued -> Running -> Paused -> Succeeded / Failed / Cancelled
                  -> Lost -> Reconciled -> new Attempt or attention
```

## 7. Authority and custody frame

Draw a vertical boundary:

| Local Waldo Core | Kennel Runtime |
| --- | --- |
| ResponsibilitySpace, Outcome, OpenLoop | Files, repositories, worktrees |
| Contracts, plans, WorkUnits | Codex sessions/processes |
| Decisions, grants, EffectIntents | Credentials and connector I/O |
| Evidence metadata, Verification | Raw sources/artifacts/traces |
| Acceptance, LoopDisposition, lineage | Effect reconciliation/recovery facts |
| Attention, briefs, snapshots, Re-entry | Terminal/browser/capture custody |

Add beneath: **Hosted Waldo is absent at launch. Explicit future attachment chooses one semantic canonical writer; raw local custody remains Kennel.**

## 8. Failure-injection board

Create cards for:

1. Codex missing or unauthenticated;
2. protocol/capability mismatch;
3. stale fenced Attempt emits late result;
4. app restarts with dirty worktree;
5. deterministic check fails;
6. external effect result unknown;
7. Gmail refresh token revoked;
8. incremental sync history expired;
9. email contains prompt injection;
10. draft created but user edits/sends elsewhere;
11. communication source disappears;
12. Desktop Context permission denied;
13. excluded app accidentally captured;
14. storage cap reached;
15. user corrects a false Commitment Candidate.

For each, require: **contain, reconcile, user-visible truth, next action, retry owner, causal receipt**.

## 9. Dogfood board

Add scorecards for:

- supervision minutes vs direct Codex: target 30% lower;
- transcript reconstruction: at most 20%;
- attention precision: at least 80%;
- current Evidence/Verification coverage: 100% for accepted criteria;
- false-ready/reopen: at most 10%;
- injected recovery: at least 90% safe;
- unauthorized/duplicate effects: zero;
- Re-entry: median under 60 seconds;
- auto-created commitments or auto-sent mail: zero.

Ask: **What result makes us stop building this wedge?**

## 10. Ready-to-paste Excalidraw prompts

### Prompt A — complete UX flow

> Create a grayscale left-to-right UX flow for Waldo Kennel desktop. Use three primary destinations: Home, Work, Settings. Home includes Today, Quick Capture, Daily Snapshot, Communication Inbox, Communication Brief, Open Loop, Draft Review, Ready to Close, and Re-entry. Work includes Project readiness, Outcome Define, Clarify, Mission Map, Authority, Run, Needs You, Action Required, Waiting, Evidence, Acceptance, Adaptive Close, and Re-entry. Show Gmail and Desktop Context as optional bounded sources. Mark user-confirmed transitions and consequential-effect approvals. Never imply that sessions, checks, email activity, or screenshots create Acceptance or closure.

### Prompt B — ontology and governance

> Draw the Waldo Kennel ontology in grayscale. ResponsibilitySpace branches to WorkProject and PersonalHome. WorkProject contains Project and Outcome. Outcome owns ContractRevision, PlanRevision, WorkUnit, Attempt, EvidenceItem, VerificationRun, AcceptanceDecision, and SuccessorLink. PersonalHome contains OpenLoop, LoopDisposition, DailySnapshot, and explicit notes. Gmail SourceConnection produces CommunicationThreadRef and CommitmentCandidate, which requires user confirmation before OpenLoop or Outcome. EffectIntent precedes external I/O and EffectReceipt follows reconciliation. DesktopObservation and ContextEpisode are optional untrusted inputs; MemoryCandidate is later. Distinguish canonical objects, projections, runtime facts, and later candidates.

### Prompt C — local architecture

> Draw User to Electron to loopback Go daemon. Inside the daemon separate Local Waldo Core from Kennel Runtime around one SQLite canonical writer. Waldo owns responsibility semantics, authority, effects, Evidence metadata, Verification, Acceptance, closure, and lineage. Kennel owns files, worktrees, Codex, credentials, raw sources, connector I/O, browser/terminal, and recovery facts. Show optional direct Gmail and Desktop Context connections. Hosted Waldo is outside the launch boundary and may attach later without dual canonical writers.

## 11. Review agenda

### 0–15 minutes — Thesis and boundary

- Does one responsibility system coherently connect work and personal continuity?
- Is Gmail an adjacent proof rather than a second product?
- Is Desktop Context visibly optional and later?

### 15–35 minutes — Home

- Walk F01–F10.
- Remove anything that behaves like an activity dashboard or full inbox.
- Test false candidates, waiting, closure, and correction.

### 35–65 minutes — Work

- Walk F11–F23.
- Test simple versus non-trivial Outcome, authority, recovery, Evidence, and Acceptance.

### 65–80 minutes — Runtime and privacy

- Review F24–F27 and authority/custody boundary.
- Test connector/capture disclosure and revoke.

### 80–100 minutes — Failures and falsifiers

- Run the failure-injection board.
- Reject any recovery that hides unknown effects or needs transcript reconstruction.

### 100–115 minutes — Simplification

- Which object is actually a projection?
- Which screen can disappear?
- Which launch-beta item can remain internal without weakening the Work proof?

### 115–120 minutes — Record decisions

- Approved as-is;
- approved with named changes;
- blocked by one concrete unknown;
- deferred to later phase.

## 12. Decision capture template

```text
Decision:
Label: Locked | Observed | Proposed | Unknown
Problem it solves:
Chosen behavior:
Rejected alternative:
Ontology objects affected:
Screens/states affected:
Authority/privacy consequence:
Failure/recovery consequence:
Evidence or source:
Owner:
Follow-up issue/PR:
```

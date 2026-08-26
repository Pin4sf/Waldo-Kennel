# Work Experience: Screen and Interaction Specification

- **Status:** Approved product-behavior and interaction specification
- **Date:** 2026-08-25
- **Audience:** Product design, interaction design, content design, frontend, daemon/API, provider-adapter, evaluation, and QA agents
- **Canonical parent:** [Work Control Plane and Canonical User Flow](2026-08-25-work-control-plane-canonical-flow-design.md)
- **Delivery plan:** [Work Control-Plane Delivery](../plans/2026-08-25-work-control-plane-delivery.md)
- **Scope:** The complete desktop Work experience from first Project through accepted or reopened Outcome, including Project Waldo, adaptive intake, Mission planning, authorization, execution, agent-session inspection, proof, recovery, and governed learning
- **Implementation status:** Design contract, not a shipped-feature claim. Current product code represents only parts of this experience.
- **Authority:** This document fixes information architecture, responsibility boundaries, required states, and interaction behavior. It deliberately leaves final visual styling, exact dimensions, and illustration language to design exploration. It does not authorize release, deployment, ambient capture, provider access, or learning promotion.

For a team-facing, interactive explanation of this specification, open the self-contained [Work Experience Visual Guide](../../product/kennel-work-experience-visual-guide.html). The visual guide teaches the approved model; this written specification remains canonical for exact behavior and failure boundaries.

## 1. The job this experience must do

The user should be able to state a result, correct Waldo's understanding, approve a proposed way of achieving it, supervise several agents without becoming their dispatcher, and decide whether the result is actually complete.

The user must not need to:

- translate a goal into provider prompts;
- keep a coordinator model session permanently open;
- infer progress from terminals or transcripts;
- remember which agent owns which task;
- reconstruct scope after a provider session is lost or replaced;
- accept work merely because an agent says it is done;
- understand Kennel's entire ontology before creating useful work.

The experience changes delegation from **start an agent and watch a transcript** to **state what should become true, approve the mission, and supervise evidence-backed progress**.

## 2. Product model the interface must teach

```text
One Waldo identity
  -> Work
    -> Project
      -> persistent Project Waldo conversation
      -> Outcomes shown on Board or List
        -> Contract revisions
        -> Mission / Plan revisions
          -> Work Units
            -> Attempts
              -> Codex / DeepSeek / Claude Code sessions
        -> Evidence and Verification
        -> User Acceptance or Reopen
      -> Sessions, an operational cross-Outcome projection
      -> What Waldo noticed, governed learning candidates
```

The interface must consistently preserve these distinctions:

- **Outcome:** the durable result the user wants to make true. It is the card/row on the Project Board.
- **Mission:** the current authorized plan for pursuing one Outcome.
- **Work Unit:** a bounded responsibility inside the Mission.
- **Attempt:** one execution try for a Work Unit.
- **Agent session:** one provider-native Codex, DeepSeek, Claude Code, or future-harness session attached to an Attempt.
- **Project Waldo:** the persistent Project-scoped relationship that helps intake, coordinate decisions, and explain state. It is not one immortal LLM context and is not a provider console.

Never label an Outcome as an "Outcome session." A single Outcome may contain no agent session, one session, several concurrent sessions, or replacement sessions over time.

## 3. Before, after, and why

| Before | After | Why |
| --- | --- | --- |
| Project creation is hidden or only Scratch is visible. | A persistent **+** beside Projects and a row-level **+** on Project hover/focus expose Project and Outcome creation. | The next action must be discoverable without guessing at a dropdown. |
| A long empty Contract form is the first interaction. | A compact natural-language modal precedes an adaptive, pre-populated Contract proposal. | Users know the desired result before they know Kennel's schema. |
| Outcome, orchestration chat, and provider transcript blur together. | Board -> Outcome Mission Control -> Session Inspector creates explicit levels. | Users can supervise responsibility without losing access to operational detail. |
| A running provider looks like the source of truth. | Canonical Outcome, Plan, authority, Evidence, and acceptance remain daemon-owned; provider state is shown beneath them. | Sessions can fail or be replaced without losing the Mission. |
| User questions appear as raw terminal output. | Needs You cards, Waldo Island, and Decision Requests route answers at Outcome level. | The user answers the responsibility, and Kennel coordinates affected agents. |
| "Done" is inferred from session exit or a commit. | Criterion-bound Evidence, Verification, and explicit Accept/Reopen close the loop. | Completion becomes inspectable and owned by the user. |
| Agent history is either forgotten or replayed wholesale. | Bounded context packets, continuation receipts, and governed learning candidates preserve useful continuity with provenance. | Continuity does not require one ever-growing model context or silent memory. |

## 4. Global Work shell

### 4.1 Persistent regions

The desktop shell has four composable regions:

1. **Global rail** — Kennel identity, Home/Work switch, search, notifications, Settings, and window controls.
2. **Project rail** — Project selector, Project navigation, Outcome shortcuts, and the Project-scoped New Outcome action.
3. **Primary canvas** — Board/List, Outcome Mission Control, or a focused review surface.
4. **Context panel** — Project Waldo or Session Inspector. It may be closed, docked, or expanded without changing canonical state.

Wide layouts may show all four. Narrow layouts preserve the primary canvas and open the Project rail or context panel as overlays. Closing a panel never ends a conversation, Attempt, or session.

### 4.2 Global navigation

- **Home / Work pill:** Work remains selected throughout this specification. Selecting Home leaves Work state intact and opens the personal surface. Personal/general messages are not silently copied into Work.
- **Board / List:** switches only the current Project's Outcome projection. Selection, filters, and canonical state remain unchanged.
- **Search:** searches Projects, Outcomes, Work Units, sessions, artifacts, and admitted knowledge with type and Project labels.
- **Notifications:** opens durable attention events and Decision Requests; it is not a generic activity log.
- **Settings:** opens provider inventory, Project defaults, permissions, privacy, and advanced role preferences.
- **Define Outcome:** always binds to the visibly selected Project. If no real Project is selected, it first routes through Add Project.

### 4.3 Project rail

The **Projects** heading contains:

- disclosure control;
- persistent **+ Add Project** button;
- optional filter control.

Every Project row contains:

- folder/workspace icon;
- Project name;
- readiness or Action Required indicator when relevant;
- disclosure control for recent Outcomes;
- on hover and keyboard focus, **+ New Outcome**;
- overflow menu for Settings, Reveal Folder, Archive, or Remove Registration.

Expanded Project rows show a short, virtualized list of Outcomes ordered by attention and recency. Each item shows title, lifecycle/attention marker, and active-session count when non-zero. **Show all** opens the Project Board; it does not expand the rail indefinitely.

The row-level **+** has a stable focus target and tooltip: **New Outcome in {Project}**. It is not hover-only for keyboard or touch-equivalent access.

### 4.4 Truth and update behavior

- Canonical mutations show pending state until the daemon confirms them.
- Renderer-only selection, open panels, local draft text, and view preference may update immediately.
- Live CDC updates should preserve focus, scroll, open menus, and unsent drafts.
- A card may move columns after a derived attention fact changes, but the UI announces the change and provides **View update**; it does not animate the card away while the user is interacting with it.
- Status is derived from durable facts. The renderer must not persist display-only labels as Outcome truth.

## 5. End-to-end experience map

```text
Add Project
  -> choose default ready coding agent
  -> Project Board
  -> New Outcome natural-language modal
  -> analyze bounded Project context
  -> clarify only material ambiguity
  -> review and correct adaptive Contract
  -> confirm Outcome + ContractRevision 1
  -> inspect adaptive Mission proposal
  -> explicitly authorize PlanRevision
  -> Act & Observe in Outcome Mission Control
  -> answer Outcome-level Decision Requests
  -> inspect or take over individual sessions when desired
  -> review criterion-bound Evidence and Verification
  -> Accept, request rework, or Reopen
  -> review governed learning candidates separately
```

This is a lifecycle, not a mandatory linear wizard. Every confirmed Outcome is re-enterable from Board/List, and Mission Control emphasizes the current safe decision.

## 6. Screen W01 — First Work entry and no-Project state

### User job

Understand what Work is and add the first real repository/workspace.

### Visible composition

- normal Work shell;
- Project rail containing Scratch and **+ Add Project**;
- primary empty-state card headed **Bring a project into Kennel**;
- short explanation: **Choose a repository or workspace. Waldo will use its context to help define and coordinate Outcomes.**;
- primary **Add Project**;
- secondary **Use Scratch** for intentionally unbound exploration.

### Behavior and states

- Scratch never masquerades as a configured repository.
- If the daemon is unavailable, disable canonical creation and show **Reconnect to Kennel** with retry details; preserve no fake Project locally.
- If the filesystem picker is cancelled, return focus to Add Project with no toast.
- If existing Projects load slowly, show Project-row skeletons rather than the no-Project message.

## 7. Screen W02 — Add Project or Workspace

### Step 1: choose scope

The first sheet or modal offers:

- **Project** — one repository or working folder;
- **Workspace** — a parent folder containing several related repositories;
- current folder path after selection;
- **Choose folder**;
- **Continue** and **Cancel**.

Kennel detects repository roots and nested repositories, then shows what will be registered. It never scans unrelated folders without explicit selection.

### Step 2: identity and defaults

Required fields:

| Field | Behavior |
| --- | --- |
| Name | Suggested from folder; editable; non-empty and unique in visible Project scope. |
| Default coding agent | Preselect the lowest-risk ready provider, initially Codex when available. Inventory comes from the daemon. |
| Provider-specific configuration | Appears only when required. DeepSeek requires a valid `dsh` profile. Unsupported model overrides are hidden. |

Optional collapsed **Advanced Settings**:

- preferred analyzer, planner/coordinator, worker, integrator, and verifier roles;
- default model/profile where supported;
- concurrency and spend guidance;
- network and consequential-effect policy;
- workspace/worktree policy.

Advanced values are preferences subject to capability and policy validation, not permanent provider assignment.

### Readiness interaction

Selecting a provider starts an explicit readiness probe and displays one of:

- **Checking…** with non-looping progress;
- **Ready** with adapter and profile detail;
- **Action required** with the adapter's truthful detail and a direct fix;
- **Unavailable** when the binary is missing or capability is unsupported.

For DeepSeek, Project creation stays disabled until a profile is selected and composition succeeds. A real prompt-delivery test may be offered or required by the relevant admission gate. Kennel never silently substitutes Codex after DeepSeek is selected.

### Confirmation

**Add Project** creates the durable registration. On success, Kennel selects the new Project and opens its Board. If creation succeeds but selection fails, the Project remains in the rail and the UI explains how to open it; it does not recreate it.

### Failure states

- duplicate registration: link to existing Project;
- unreadable or moved folder: preserve entered settings and allow choose again;
- invalid nested workspace: show detected roots and let the user narrow scope;
- stale readiness result: recheck before confirmation;
- daemon conflict: reload served inventory and preserve compatible choices.

## 8. Screen W03 — Project Board and List

### User job

See every desired result, where attention is needed, what is running, and the next safe action.

### Project header

- Project name and repository/workspace identity;
- Board/List switch;
- saved filter and search controls;
- Project Waldo toggle;
- **+ Define Outcome**;
- optional summary: active Outcomes, Needs You, running sessions, Ready for Acceptance.

### Board projection

The initial projection may use these derived columns:

- **Draft / Needs Choice**;
- **Needs Input**;
- **In Progress / Waiting**;
- **Verify / Ready to Accept**;
- **Accepted** behind an optional completed filter.

Columns are projections over lifecycle and attention facts, not stored status enums. Design may merge sparse columns or use a focused attention view at narrow widths.

### Outcome card

Required card content, ordered by importance:

1. Outcome title;
2. plain-language next state or requested decision;
3. Project/repository and branch/workspace context when useful;
4. current lifecycle stage and latest verified progress;
5. active Work Units and session count;
6. provider/role chips only when they help explain running work;
7. blocker, recovery, or Decision Request;
8. Evidence coverage/progress without implying acceptance;
9. last meaningful change time.

Card actions:

- select/open Mission Control;
- answer or review the next Decision Request;
- open a specific active session through the count/agent chip;
- overflow for rename draft, archive view, or open lineage where permitted.

Provider terminal output must never dominate the Outcome card.

### List projection

List view presents the same Outcomes, filters, and next actions in sortable rows. Useful columns are Outcome, stage/attention, next action, active Work Units, sessions, Evidence, owner, and updated time. Board/List switching preserves the selected Outcome and filter query.

### Empty and degraded states

- no Outcomes: **What should become true in {Project}?** plus **Define Outcome**;
- filtered empty: explain the active filter and offer clear filters;
- Project Action Required: keep existing Outcomes visible and show the exact provider/workspace problem;
- offline: show last confirmed snapshot with a stale/offline label; disable canonical actions but preserve local drafts.

## 9. Surface W04 — Persistent Project Waldo

### User job

Discuss the Project, understand state, propose or correct Outcomes, and answer coordination questions without opening individual provider consoles.

### Identity and context

The panel header always shows:

- **Waldo**;
- Project chip, for example `kennel-design`;
- explicit context chip: **Project**, **Outcome: {title}**, or **Inspecting: {provider} · Attempt {n}**;
- context menu to return to Project or select another Outcome;
- close/expand control.

When no Outcome is selected, Waldo speaks from bounded Project context. When an Outcome is selected, it can explain and update proposals for that Outcome. When inspecting a session, Waldo may explain session state, but direct provider conversation remains in Session Inspector.

### Conversation behavior

- The visible conversation is a durable projection over bounded episodes, not one permanently running LLM session.
- Waldo may answer Project questions, propose an Outcome draft, request one material clarification, explain a Mission, summarize evidence, or route a Decision Request.
- A normal answer to **Needs You** attaches to the Outcome-level Decision Request. Kennel records it canonically and routes the necessary bounded update to affected Work Units.
- A message that would change Contract, Plan, permission, cost, or consequential effect becomes a visible proposal/decision. Chat text alone does not grant authority.
- Personal/general requests offer **Continue in Home**. Transfer requires confirmation and carries only approved content.

### Composer

- placeholder reflects context: **Ask about {Project}…** or **Reply about {Outcome}…**;
- attachment button exposes allowed Project artifacts and explicit screen/file attachments;
- send, stop-response, and retry controls;
- proposal chips such as **Turn this into an Outcome** or **Update Contract draft** appear only when semantically relevant;
- unsent text is retained per Project/context and clearly marked local when not daemon-confirmed.

### States

- no configured analyzer/provider: allow viewing durable conversation and manual Outcome entry; explain what requires configuration;
- analysis in progress: show the bounded task, not generic sparkles or indefinite thinking;
- stale response after a revision: label it and link to current truth;
- failed provider turn: preserve the user message, show retry/change-provider choices permitted by policy, and never fabricate a response.

## 10. Surface W05 — New Outcome modal

### User job

State the desired result in natural language with minimal ceremony.

### Exact first step

The compact modal contains:

- title **Define an Outcome**;
- selected Project chip with an explicit change action;
- one multiline field labelled **What would you like to make true?**;
- short example tied to the Project when safely available;
- primary **Analyze Outcome**;
- secondary **Cancel**.

The field accepts a sentence or several paragraphs. It must not demand Title, success criteria, permissions, agent choice, or tasks at this step.

### Submission behavior

- Enter inserts a line break; Command/Ctrl+Enter submits.
- Empty or whitespace-only input is rejected inline.
- Submission creates an IntakeSession and preserves the exact user statement.
- No Outcome or immutable Contract is created yet.
- Closing during analysis keeps a restorable draft/Intake and shows it on the Board as a non-Outcome intake draft only when the daemon confirms it.

### Classification outcomes

- clear or ambiguous Outcome -> continue to analysis/clarification;
- Project question -> offer to continue in Project Waldo without creating an Outcome;
- correction/follow-up -> offer related open Outcomes;
- personal/out-of-Project request -> offer explicit Home transfer or change Project;
- analyzer unavailable -> preserve intent and offer retry or **Draft manually**.

## 11. Screen W06 — Context analysis and material clarification

### Analysis state

Show a calm, bounded sequence such as:

- **Understanding the desired result**;
- **Checking relevant Project context**;
- **Drafting success and review conditions**.

An expandable **Context used** receipt lists sources, freshness, omissions, and explicit exclusions. Do not show fake percentage completion when the adapter cannot report it.

### Clarification state

Ask at most one active material question at a time. The screen shows:

- the original intent, still editable;
- one question and why it changes the result or authority;
- two or three mutually exclusive choices where possible, with a recommended option and trade-off;
- a free-text alternative;
- **Continue** and **Back to intent**.

Examples of material ambiguity include target platform, required definition of "today," production effect, acceptance owner, or whether two conflicting goals are in scope. Stylistic preferences that can be corrected in the Contract do not block progress.

If the user skips a non-blocking question, it becomes an explicit assumption in the Contract proposal. A question that changes consequential authority cannot be skipped.

## 12. Screen W07 — Adaptive Contract proposal

### User job

Correct Waldo's understanding of success before work or agent assignment begins.

### Page structure

1. stage header **Understand the Outcome**;
2. original intent and context receipt;
3. generated Contract form;
4. material assumptions/questions;
5. revision explanation;
6. confirmation actions.

### Stable Contract core

| Section | Required behavior |
| --- | --- |
| Title | Suggested short name; editable. |
| Desired result | Precise state that should become true; required. |
| Success criteria | One or more observable criteria; add, edit, reorder, or remove while keeping at least one. |
| Evidence expected | Attached to each criterion; may be suggested or manually specified. |
| Review and acceptance | How the result will be checked and who decides; required. |
| Constraints | Optional boundaries the work must respect. |
| Non-goals | Optional deliberate exclusions. |
| Authority ceiling | Initial bounds such as read/write, network, commit, PR, deploy, external messages, spend. Defaults to least privilege. |
| Stop and escalation | Conditions that require pause, recovery, or user decision. |
| Assumptions and clarifications | Material inferred statements, visibly attributed. |
| Time condition | Appears only when time language materially affects completion. |

### Adaptive facets

The analyzer may add typed, removable sections relevant to the Outcome, for example:

- software: target surfaces, compatibility, migrations, tests, performance;
- research: question, source standard, date boundary, citation/reproducibility;
- design: users, flows, states, device constraints, review artifacts;
- documentation: audience, canonical sources, publish location, approval;
- investigation: symptom, environment, reproduction, diagnosis-only boundary;
- operations: exact target, reversibility, approval, rollback, external effect.

Adaptive facets never replace the stable core or become arbitrary untyped model output.

### Editing and provenance

- Suggested fields are visually distinguishable from user-authored or confirmed text without relying on color alone.
- Each assumption exposes **Accept**, **Edit**, or **Remove**.
- **Re-analyze** explains which edits will be retained and creates a new proposal revision, not a silent overwrite.
- **Compare changes** appears after a regenerated or concurrently updated proposal.
- Agent/provider assignments do not appear here; they belong to the Mission.

### Actions

- **Confirm Outcome** atomically creates the Outcome and `ContractRevision 1`;
- **Save intake draft** when durable draft support is available;
- **Keep discussing with Waldo**;
- **Cancel intake** with explicit discard behavior.

After confirmation, the Outcome appears immediately on Board/List and the app advances to Mission proposal. Daemon failure leaves the proposal editable and unconfirmed.

## 13. Screen W08 — Mission proposal and explicit authorization

### User job

Understand how Kennel proposes to achieve the confirmed Outcome, inspect costs and risks, and authorize the exact Plan.

### Required composition

- Contract revision binding and change warning when stale;
- recommended topology with plain-language rationale;
- adaptive Mission graph;
- Work Unit list synchronized with graph selection;
- dependency and workspace/worktree ownership;
- role, provider, model/profile, and readiness for every Work Unit;
- evidence obligations and verifier independence;
- permissions, network, external effects, cost/budget, concurrency, stop, and recovery;
- alternatives when a meaningfully different safe topology exists;
- exact authorization summary.

### Mission graph

Nodes represent responsibilities or decisions, not decorative agents. Node types include:

- Contract/Outcome source;
- Work Unit;
- user Decision Request;
- integration/checkpoint;
- Verification;
- Acceptance.

Edges represent explicit dependencies or evidence flow. Selecting a node opens its details in place. The graph must remain legible as a simple line for small Outcomes and may branch for complex Outcomes; do not force every Mission into the same number of agents.

Example topologies:

- small: default worker -> deterministic check -> user acceptance;
- medium: implementer -> fresh verifier -> user acceptance;
- complex: planner/coordinator -> parallel isolated workers -> integrator -> verifier.

One harness can occupy several roles through separate sessions. Multiple providers are offered only when installed, ready, justified, and allowed. The Project default is preferred for small/medium work, not forced.

### Authorization

The primary action is **Authorize this Plan**. Immediately above it, summarize:

- agents and number of expected sessions;
- files/workspaces they may modify;
- network and external-effect permissions;
- commit, PR, deploy, or message permissions;
- budget/cost/concurrency guidance where facts exist;
- stop conditions and decisions that will return to the user.

Material edits create a new proposal revision and invalidate an earlier authorization. No provider starts before daemon confirmation. A separate **Ask Waldo to revise** action accepts natural-language corrections without granting execution authority.

## 14. Screen W09 — Outcome Mission Control: Act and Observe

### User job

Supervise the desired result, understand what is active or blocked, and intervene at the right level.

### Header

- Outcome title, Project, custody/owner;
- lifecycle stage and next safe action;
- Contract and Plan revision chips;
- pause/stop Mission control when supported;
- Project Waldo toggle;
- **Board** back action.

### Primary graph and state

The authorized Mission graph becomes the primary execution view. Nodes derive state from durable facts and may show:

- not started / waiting on dependency;
- admitted / starting / unconfirmed;
- active with last trustworthy activity;
- needs user / blocked;
- recovering / continued into replacement session;
- produced candidate Evidence;
- ready for or failed Verification;
- superseded Attempt.

Never use color alone. Every state has text/icon and an accessible description. Provider process exit does not mark the Outcome complete.

### Supporting regions

- **Next decision** — highest-priority Decision Request or safe action;
- **Work Units** — objective, owner role, Attempt history, session count, dependencies;
- **Timeline** — canonical decisions, revisions, starts, recoveries, evidence, and verification, not raw token-by-token output;
- **Evidence coverage** — criterion mapping and gaps;
- **Mission details** — authority, budget, worktrees, providers, recovery.

### Interaction

- selecting a Work Unit focuses it and opens its detail;
- selecting an Attempt/session opens Session Inspector;
- selecting a Decision Request opens the bounded response surface;
- changing scope or agent assignment proposes a new PlanRevision;
- routine live updates do not steal selection or scroll.

### Automatic session continuation receipt

When the daemon safely refreshes a provider context without material change, show a compact timeline receipt:

> Waldo refreshed this agent's working context. Scope, provider, workspace, authority, and budget did not change.

The detail reveals old/new session identity, checkpoint, continuation packet provenance, and fence state. User approval is required when provider, model, role, authority, cost, workspace, topology, or uncertain external effects would change.

## 15. Panel W10 — Work Unit and Attempt detail

### User job

Understand one responsibility and its execution history before inspecting a provider session.

Show:

- Work Unit objective and dependency inputs;
- role and provider assignment;
- frozen RunBrief digest and current revision bindings;
- worktree/workspace ownership;
- Evidence obligations and stop conditions;
- Attempt timeline, including failed, unconfirmed, superseded, and current Attempts;
- each attached AgentSessionRef and recovery/continuation relationship;
- **Open Session Inspector** for a selected session.

Retry is never a blind duplicate. The UI explains whether it will resume the same session, create a replacement session inside the same Attempt, or create a new Attempt/Plan revision and why.

## 16. Screen W11 — Individual Session Inspector

### User job

Inspect or directly intervene in one operational provider session without confusing it with the Outcome or Mission.

### Persistent identity header

- provider and native session identity;
- role, Work Unit, Attempt number;
- parent Outcome and Project links;
- model/profile where known;
- activity, termination, readiness, recovery, and connection facts;
- worktree and branch.

### Tabs or panes

- **Conversation / transcript** — provider-native structured chat when supported;
- **Terminal** — exact terminal surface where supported;
- **Browser** — preview/browser context where available;
- **Files / artifacts** — attributed changed files and produced artifacts;
- **RunBrief** — bounded instructions, authority, dependencies, and expected evidence;
- **History** — resume, recovery, continuation, and replacement lineage.

### Controls

- send tactical instruction;
- pause/resume when adapter semantics are trustworthy;
- request checkpoint;
- take over terminal/session;
- stop/cancel with consequence summary;
- return input routing to Waldo/Outcome;
- report a contradiction or scope problem.

Direct instruction is an advanced action. The composer says **Message this {provider} session**, not **Reply to Outcome**. Before sending, Kennel classifies whether the instruction is tactical within authority or material. Material changes open Contract/Plan/permission review rather than being delivered as unauthorized chat.

If the session is unconfirmed, disable instructions and effects until reconciliation. If a transcript is unavailable, show the known session facts and recovery options rather than an empty chat that implies no work occurred.

## 17. Surface W12 — Needs You and Waldo Island

### User job

Answer the smallest consequential question without opening every agent session.

A Decision Request contains:

- Outcome and Project;
- requesting Mission/Work Unit(s);
- the exact decision;
- why it is needed now;
- recommended answer and trade-off;
- scope of the answer;
- deadline or waiting effect when real;
- **Open Mission Control**.

The Board card, notification center, Project Waldo, and Island are projections of the same durable Decision Request. Answering in one resolves the others after daemon confirmation. The answer belongs to the Outcome-level decision, and Kennel routes it to every affected agent through updated RunBriefs or continuation packets.

The user may explicitly choose **Open individual session** for tactical context. The Island never exposes raw provider terminal output as the primary question and never claims the provider can directly widen authority.

## 18. Screen W13 — Prove and Close

### User job

Determine whether the desired result is supported by enough trustworthy evidence and decide what happens next.

### Criterion review

For every Contract criterion show:

- exact criterion text and Contract revision;
- expected Evidence;
- submitted EvidenceItems with source, subject revision, freshness, and producer;
- VerificationRun result and independence class;
- contradictions, gaps, or exceptions;
- user notes and walkthrough controls.

Verification classes are stated plainly: deterministic check, producer self-check, fresh same-provider review, cross-provider/model review, or owner walkthrough. Do not collapse them into an unexplained confidence score.

### Decisions

- **Accept Outcome** — creates the user AcceptanceDecision;
- **Request rework** — records correction/counter-evidence, keeps Outcome open, and proposes the required Work Unit/Attempt/Plan change;
- **Revise Outcome** — creates a new ContractRevision when the desired result changed;
- **Reopen** — available after acceptance and preserves earlier acceptance history;
- **Create successor Outcome** — for new scope rather than rewriting completed truth.

An agent saying done, a process exiting, tests passing, a commit, or a PR cannot trigger acceptance automatically.

## 19. Screen W14 — Sessions across Outcomes

### User job

Operate and diagnose active provider sessions across the selected Project without making sessions the primary product hierarchy.

The Sessions projection shows:

- provider/session identity;
- parent Outcome, Work Unit, and Attempt;
- role, worktree, activity, termination, recovery, and attention;
- current safe control;
- filters for active, needs attention, recovering, unconfirmed, and historical.

Selecting a row opens Session Inspector. Selecting the Outcome opens Mission Control. This view is operational and may be hidden behind a Project navigation item or inspector icon; Board remains the default Work landing.

## 20. Screen W15 — What Waldo noticed

### User job

Inspect, correct, dismiss, or approve useful Project learning without allowing silent behavioral drift.

Show attributed LearningEpisodes and candidates such as:

- recurring Project rule or correction;
- useful workflow or context-selection candidate;
- potential skill change;
- experiment/evaluation proposal;
- contradiction or stale knowledge.

Every candidate shows source Outcomes/sessions, provenance, scope, sensitivity, why it may help, counter-evidence, and proposed destination. Actions are **Correct**, **Dismiss**, **Keep as candidate**, **Evaluate**, or **Promote** only when the applicable gate allows it.

Raw transcripts do not become durable Memory automatically. Outcome acceptance is a result label, not proof that a particular procedure caused success. Home/user Memory is separate in purpose and consent and is not silently disclosed to Work.

## 21. Screen W16 — Project Settings and agent preferences

### User job

Configure future proposals and fix readiness without mutating running Missions.

### Agent section

- served provider inventory with readiness detail;
- **Default coding agent** for future Mission proposals;
- required profile/model controls by provider capability;
- test readiness / real prompt delivery where supported;
- optional Advanced role preferences for analyzer, planner/coordinator, executor, integrator, verifier, and recovery;
- truthful role eligibility and reason when unavailable;
- concurrency, budget, network, and effect defaults.

Changing these values affects future proposals. It does not rewrite an authorized Plan, swap a running session, or erase historical identity. A user may request a current Mission revision separately.

### Project and context section

- folder/workspace identity and repository roots;
- index/context freshness and exclusions;
- admitted Project knowledge and skills;
- conversation retention and deletion controls;
- learning/capture consent where implemented;
- remove registration versus delete Kennel-owned state, with exact consequence and recovery language.

## 22. Macro-interactions and re-entry

### Navigation rules

- Work always opens the last valid Project and view; otherwise the first-Project state.
- Opening an Outcome preserves its selected Mission node, stage, and context panel when safe.
- Browser-style Back returns Session Inspector -> Mission Control -> Project Board without ending execution.
- Deep links resolve Project -> Outcome -> Work Unit/Attempt -> session and explain missing or deleted ancestors.
- Project Waldo remains scoped to the selected Project even while switching Board/List.
- Switching Projects swaps conversation, filters, and drafts; it never sends one Project's unsent text to another.

### Background changes

- new attention appears as a badge and optional restrained notification;
- an active user edit is never overwritten by CDC;
- concurrent canonical changes create compare/reload state;
- stale Contract invalidates Plan authorization visibly;
- daemon restart restores the exact confirmed lineage and next safe action.

### Destructive or consequential actions

Stop, cancel, discard, remove, deploy, send externally, delete state, or revoke access always names:

- exact target;
- what continues running;
- whether effects can be reversed;
- what canonical history remains;
- whether another authorization will be required.

## 23. Micro-interaction contract

### Buttons and focus

- Hover and keyboard focus reveal the same contextual actions.
- The Project-row **+** uses a 44px-equivalent target even if its visual icon is compact.
- Primary actions use stable verb phrases: **Analyze Outcome**, **Confirm Outcome**, **Authorize this Plan**, **Accept Outcome**.
- Disable only when the user cannot safely proceed; include an adjacent reason or accessible description.
- Async submission changes the label to the concrete operation and prevents duplicate mutation.
- After modal close, return focus to the invoking **+** or Outcome card.

### Context chips

- Chips communicate scope, not decoration.
- A Project chip and Outcome chip remain readable and keyboard-selectable.
- Changing scope requires an explicit action and warns about unsent drafts.
- Provider/model chips appear only where operationally relevant.

### Graph behavior

- Keyboard traversal follows dependency order.
- Selection is persistent and mirrored in a textual Work Unit list.
- Hover may preview an edge path; click/focus owns selection.
- Panning/zooming never traps keyboard focus; **Fit Mission** restores the whole graph.
- A graph is optional presentation: all information and actions remain available in a list/details representation.

### Receipts, notifications, and motion

- Use inline receipts for saved revisions, authorization, recovery, context rollover, and routed decisions.
- Toasts are reserved for transient confirmation and always link to durable detail when consequential.
- Motion explains spatial change: panel open, node focus, card relocation, or successful binding.
- Avoid looping "AI thinking" effects, ornamental gradients, particle effects, and celebratory motion for routine work.
- Respect reduced motion; use immediate state changes and opacity only where needed.
- Do not animate attention indefinitely. Needs You remains discoverable through structure and labels.

### Copy

- Use Waldo for the persistent relationship and Kennel for the local control plane when the distinction matters.
- Name providers only for provider-specific choices or diagnostics.
- Prefer **Waldo needs your decision about…** to **The agent has a question**.
- Prefer **Waiting for verification** to **Almost done**.
- Prefer **Session ended; Outcome remains open** to **Task completed**.
- Never claim analyzed, remembered, verified, safe, or recovered without the corresponding daemon fact/receipt.

## 24. Responsive composition

### Wide desktop

- Global/Project rail remains visible.
- Board/List occupies the center.
- Project Waldo or Session Inspector docks right.
- Mission Control may use graph center + details right; Waldo becomes a collapsible tab.

### Medium desktop

- Project rail can collapse to icons/names.
- One context panel is visible at a time.
- Board columns horizontally scroll with clear column headings; List may be the recommended compact projection.

### Compact window

- Primary canvas occupies the window.
- Project rail and context panels become sheets.
- Mission graph defaults to fitted overview plus accessible Work Unit list.
- Essential Decision Request and primary action remain above the fold.

Resizing must not lose unsent text, selected nodes, terminal sessions, or running work.

## 25. Failure, empty, and recovery matrix

| Condition | Required experience |
| --- | --- |
| Daemon unavailable | Last confirmed snapshot may remain visible as stale; canonical mutations disabled; reconnect action and local draft preservation. |
| Analyzer unavailable | Preserve exact intent; retry, change admitted analyzer where allowed, or manually draft Contract; create no Outcome. |
| Provider not ready | Exact binary/profile/auth/capability detail; direct Settings action; no silent fallback. |
| Contract changed after Plan | Mark Plan stale; block authorization/start; propose regeneration with diff. |
| Start identity ambiguous | Show Attempt Unconfirmed; disable duplicate start/effects; reconcile first. |
| Session process disappears | Show last known trustworthy facts; attempt resume/recovery according to policy; do not infer death from failed probe alone. |
| Context rollover safe | Continue automatically with durable receipt and unchanged-scope summary. |
| Context rollover material/unsafe | Needs You decision with changes, unknowns, alternatives, and safest next action. |
| Conflicting evidence | Keep Outcome open; show contradiction at criterion; request verification or user exception. |
| Conversation send fails | Preserve unsent/sent-pending text, show retry, and never fabricate durable turn. |
| Project folder moved | Keep Project identity and Outcomes; block workspace operations; choose replacement path/reconcile. |
| Restart | Restore confirmed Intake, Outcome, revisions, Attempt/session lineage, drafts as designated, and next safe action. |

## 26. Backend truth required by each surface

| Surface | Daemon-owned facts and actions |
| --- | --- |
| Project setup | registration, roots, provider inventory, role capabilities, readiness, profile requirements, Project preferences |
| Board/List | Outcomes, current revisions, derived lifecycle/attention, Decision Requests, Work Units, sessions, Evidence coverage |
| Project Waldo | conversation episodes/turns, context scope, attachments, proposals, Decision routing, retrieval receipts |
| Intake | exact intent, classification, analyzer execution, proposal revisions, clarifications, confirmation idempotency |
| Contract | stable fields, typed facets, provenance, immutable revisions, conflicts |
| Mission proposal | role resolution, topology, PlanRevision, dependencies, worktrees, permissions, budgets, Evidence/Verification/stop/recovery |
| Mission Control | Work Units, Attempts, AgentSessionRefs, activity/termination facts, continuation/recovery, Decisions, Evidence |
| Session Inspector | provider-native identity, adapter capabilities, transcript/terminal/browser refs, worktree, RunBrief, controls, history |
| Prove and Close | criterion bindings, EvidenceItems, VerificationRuns, independence, AcceptanceDecision, Reopen/correction |
| Learning | attributed episodes/candidates, evaluation, scope, promotion/correction/deletion decisions |

The frontend remains a typed supervisor. It must not call models directly, compile hidden context, manufacture canonical status, or persist a parallel Outcome/session database.

## 27. Designer deliverables and review matrix

The complete design handoff should include:

### Core frames

1. first Work entry;
2. Add Project scope and provider readiness;
3. populated Board and equivalent List;
4. Project Waldo at Project and Outcome scopes;
5. New Outcome modal;
6. analyzing and material clarification;
7. adaptive Contract proposal with software and non-software examples;
8. small, medium, and complex Mission proposals;
9. authorization summary and stale-Plan state;
10. Mission Control active, Needs You, recovering, and proof states;
11. Work Unit/Attempt detail;
12. Codex and DeepSeek Session Inspector variants;
13. Island/Decision Request;
14. Evidence/Verification/Accept and rework/reopen;
15. cross-Outcome Sessions;
16. What Waldo noticed;
17. Project Settings/default agent;
18. offline, missing-provider, ambiguous-start, and restart recovery.

### Interaction prototypes

- Project hover/focus **+** -> modal -> Contract proposal;
- Board card -> Mission node -> Session Inspector -> back;
- Project Waldo context switch and Home redirect;
- one Outcome-level Decision Request answered from Island and reflected everywhere;
- Plan edit -> revised authorization;
- safe context rollover receipt versus material rollover decision;
- Evidence gap -> rework -> new Attempt -> acceptance;
- Board/List switch preserving selection and filters.

### Accessibility review

- full keyboard path for every flow;
- screen-reader names for graph nodes, states, context chips, counts, and icon buttons;
- no color-only state or evidence signal;
- focus restoration and live-region announcements for canonical updates;
- reduced-motion behavior;
- zoom and compact-window usability;
- terminal/transcript panes that do not trap navigation.

## 28. Explicit first-version boundaries

- Board is the default Work landing; an all-session terminal dashboard is not.
- One ready harness can execute the full first Mission through separate role sessions; multi-provider execution is additive.
- Project Waldo uses bounded provider turns; no permanent coordinator LLM process is required.
- Adaptive Contract facets are typed and reviewable; arbitrary generated schemas are not canonical.
- Session takeover is supported where adapter capabilities are truthful; unsupported controls remain hidden or unavailable with a reason.
- Learning remains candidate-first and cannot silently rewrite Missions, prompts, Memory, or skills.
- Home remains separate for now. Future Home/Work linking uses explicit responsibility/context links, not invisible context sharing.
- Final visual styling is intentionally open. The product hierarchy, states, labels, authority gates, and truth boundaries in this document are not optional decoration.

## 29. Acceptance of the Work experience

The experience is coherent only when a first-time user and an experienced operator can both answer:

1. Which Project am I in?
2. What result is this Outcome trying to make true?
3. What does Waldo currently need from me?
4. What Mission did I authorize and what may it do?
5. Which Work Units, Attempts, and provider sessions are active?
6. What changed, failed, recovered, or was replaced?
7. What Evidence supports each success criterion?
8. Who verified it and with what independence?
9. Has the user accepted it, requested rework, or reopened it?
10. What Project learning is merely proposed versus admitted?

The design is rejected if it visually collapses Outcome into session, hides Project creation, begins with a rigid blank form, treats Waldo chat as authority, makes the graph decorative, requires terminal reading for routine supervision, silently switches providers, loses state across restart, or lets provider completion stand in for user acceptance.

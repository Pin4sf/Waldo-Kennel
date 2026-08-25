# Work Control Plane and Canonical User Flow

- **Status:** Approved written specification
- **Date:** 2026-08-25
- **Target baseline:** `origin/beta` at `d461a125ca90b76893367cdbfbac092217fdf6f9`
- **Scope:** Work-side Project navigation, persistent Project conversation, adaptive Outcome intake, Mission planning, provider-role resolution, bounded session continuity, Mission Control, Session Inspector, proof, acceptance, and governed Project learning
- **Implementation status:** Partly represented by the merged preview and first three Outcome stages; durable conversation, adaptive intake, execution, proof, Mission Control, and learning remain issue-gated
- **Authority:** This specification records product and architecture decisions. It does not authorize merge, release, deployment, hosted attachment, ambient capture activation, or automatic learning promotion.

The companion [Work Experience Screen and Interaction Specification](2026-08-25-work-experience-screen-interaction-spec.md) translates this control-plane contract into the complete designer/implementation brief: global shell, every screen and form, visible states, transitions, micro-interactions, responsive composition, backend truth requirements, and design-review matrix.

## 1. Product decision

There is one Waldo identity. Kennel is Waldo's local desktop presence and execution control plane. Codex, DeepSeek Harness, Claude Code, and future providers are replaceable execution capabilities; none owns the Project, Outcome, Mission, Evidence, or acceptance truth.

Work is board-first:

```text
Project Board
  -> Outcome Mission Control
    -> Work Unit / Attempt
      -> individual Agent Session Inspector
```

The Project Board and List are Outcome projections. An Outcome is the durable result the user wants to make true, not a provider session. One Outcome may have no session yet, one direct session, multiple concurrent sessions, replacement Attempts, or several providers over time.

The canonical execution lineage remains:

```text
Outcome
  -> ContractRevision
  -> PlanRevision / Mission projection
  -> WorkUnit
  -> Attempt
  -> AgentSessionRef
  -> EvidenceItem
  -> VerificationRun
  -> user AcceptanceDecision or Reopen
```

## 2. Canonical Work user flow

### 2.1 Add and configure a Project

The user selects a repository, folder, or workspace and chooses one **Default coding agent**. Project creation performs a live readiness check and asks only for configuration required by the selected harness, such as a DeepSeek `dsh` profile. Separate analyzer, coordinator, worker, and verifier preferences remain optional Advanced Settings.

The selection is a Project preference, not a permanent LLM session. At Mission-planning time Kennel resolves actual roles from the intersection of Project preference, installed and ready adapters, role capabilities, Outcome requirements, cost/concurrency policy, and explicit user authority. The Mission review always exposes the resolved assignments. There is no silent provider fallback.

### 2.2 Start an Outcome

Project hover or keyboard focus exposes **New Outcome**. The compact modal contains only:

- selected Project;
- **What would you like to make true?**;
- **Analyze Outcome**;
- cancel.

Submitting begins an `IntakeSession`; it does not create an Outcome. The daemon first classifies whether the statement is a clear Outcome, an ambiguous Outcome, a Project question, a correction/follow-up, or something outside the selected Project. Project questions remain conversation. Out-of-scope personal requests may be explicitly redirected to Home. Nothing moves or shares context silently.

### 2.3 Compile bounded context

The daemon compiles the smallest Project-scoped packet required for the intake purpose:

1. current user statement and explicit edits;
2. selected Project identity, policy, configuration, and workspace snapshot;
3. high-signal repository rules and entry documents;
4. current related Outcomes and explicit lineage;
5. attributed prior-session summaries when already available;
6. admitted Project knowledge and promoted Project-scoped skills;
7. provenance, freshness, omissions, and a retrieval receipt.

The daemon does not replay every transcript or place all historical Outcomes into every prompt. User intent and current canonical revisions outrank every retrieved source and model inference.

### 2.4 Produce an adaptive Contract proposal

The Contract has a stable governable core and adaptive facets. The analyzer may vary presentation, examples, number of criteria, and relevant specialized facets; it may not invent an arbitrary canonical schema.

The stable core is:

```text
OutcomeContract
  identity and Project binding
  desired state
  observable success criteria
  criterion-bound Evidence expectations
  review and acceptance method
  constraints and non-goals
  initial authority ceiling
  stop and escalation conditions
  material assumptions and clarifications
  optional time condition
  adaptive typed facets
```

Adaptive facets may cover software implementation, research, design, documentation, investigation, evaluation, or consequential operations. Kennel asks at most one active material question at a time, and only when the answer changes meaning, scope, success, authority, cost, risk, Evidence, Verification, or responsibility placement.

The user may edit, re-analyze, relate the proposal to an earlier Outcome, dismiss it, or continue discussing it. Confirmation atomically creates the `Outcome` and immutable `ContractRevision 1`. Agent assignments are excluded from the Contract and belong in the Mission Plan.

### 2.5 Propose and authorize the Mission

Kennel proposes the smallest sufficient topology and explains why. A model may propose decomposition and role assignments, but deterministic daemon policy validates dependencies, overlapping ownership, adapter readiness, authority, Evidence, Verification, budget, cost visibility, stop, and recovery requirements.

Examples:

```text
Small:   one default-agent Attempt -> deterministic check -> owner review
Medium:  implementer Attempt -> fresh verifier Attempt -> owner review
Complex: planner/coordinator -> isolated parallel workers -> integrator -> verifier
```

Small and medium Outcomes normally reuse the Project's default harness, but independent roles use separate provider sessions and RunBriefs. Complex Outcomes may use multiple admitted providers when installed, ready, justified, and explicitly authorized. A single admitted harness may still execute multiple roles through separate sessions and worktrees.

The graph is adaptive across Outcomes but revisioned within one Outcome. A material topology, provider/model, authority, cost, workspace ownership, or dependency change creates a new `PlanRevision`; the graph never silently rewrites itself in place.

### 2.6 Execute and observe

Authorization binds the current Contract and Plan revisions. Kennel compiles a frozen provider-neutral RunBrief for every Work Unit, then an admitted adapter compiles the provider-specific form. Each write-capable Attempt owns one isolated worktree. Each Attempt receives one stable `AgentSessionRef` after provider-native identity is confirmed.

The Project Board summarizes Outcome responsibility. The separate Project Sessions view summarizes operational provider activity across Outcomes. Provider completion is an observation or candidate Evidence, never Outcome completion.

### 2.7 Verify and accept or reopen

Evidence is mapped to the exact criterion and subject revision. Verification declares its actual independence class: deterministic, producer self-check, fresh same-provider session, cross-provider/model, or owner walkthrough. Only the user creates `AcceptanceDecision`.

If the user says the result is incomplete, the Outcome remains open. Feedback becomes correction or counter-evidence and may require a replacement Attempt, revised Work Unit, new PlanRevision, or new ContractRevision. Acceptance may later be followed by an explicit Reopen decision or a successor Outcome.

## 3. Screen and interaction contract

### 3.1 Project Board and List

Each card or row represents one Outcome and may summarize:

- lifecycle stage and next safe action;
- Needs You, Action Required, Waiting, or Ready for Acceptance;
- active Work Units and session count;
- current provider/role assignments;
- blockers, recovery, and latest verified progress;
- criterion Evidence coverage.

Board columns are attention/lifecycle projections, not stored Outcome status and not provider-process columns.

### 3.2 Outcome Mission Control

Selecting an Outcome opens one adaptive, re-enterable workspace containing:

- Outcome header, Project binding, custody, and revision truth;
- Contract and clarification surface;
- current Plan/Mission graph and rationale;
- Work Units, dependencies, roles, Attempts, and session health;
- decisions, grants, budgets, stop, and recovery conditions;
- Evidence, Verification, Acceptance, Reopen, and lineage.

Before Contract confirmation the adaptive intake surface is primary. Before authorization the Plan review is primary. During execution the Mission graph is primary. During proof the criterion/Evidence view is primary. The graph remains available throughout but is not forced as the main screen when it would obscure the current decision.

### 3.3 Session Inspector

Selecting a Work Unit, Attempt, or agent node opens the individual Session Inspector with provider transcript, terminal/browser where available, worktree, activity, artifacts, instructions, recovery state, and pause/resume/cancel/takeover controls.

Normal user input attaches to the exact Outcome-level `DecisionRequest`; Kennel routes the canonical answer to every affected Work Unit/session. Direct session instruction is an explicit advanced action. Tactical instruction within authority may be delivered and recorded. A material scope, Plan, permission, or effect change raises the corresponding revision or approval boundary.

### 3.4 Persistent Project conversation

Work exposes one persistent Waldo conversation for the selected Project. It is a continuity projection over bounded conversation episodes and provider turns, not one immortal model context window and not canonical responsibility truth.

When an Outcome is selected, the visible context chip becomes, for example:

```text
Waldo · kennel-design
Context: Outcome "Build screenshot-ready dashboard"
```

When inspecting a session, Waldo may show `Inspecting: DeepSeek · Attempt 1`, but direct provider chat remains in the Session Inspector. Personal/general conversation belongs in Home. Moving a draft between Home and Work requires an explicit scope change and carries only the user-approved message/context.

## 4. Coordinator, role, and provider model

The always-running coordinator is the Kennel daemon control plane, not an LLM. It owns canonical state, context compilation, policy validation, routing, persistence, recovery, and attention projections.

An analyzer, planner, coordinator, executor, integrator, verifier, or recovery agent is a capability-based Mission role filled by one bounded provider Attempt. Codex may fill all roles when it is the only admitted harness. DeepSeek may fill worker roles after profile readiness. Later providers may fill additional roles only after their required capability tests pass.

Changing Project defaults affects future proposals. It does not rewrite an approved Plan, replace a running Attempt, or change historical provider identity.

## 5. Bounded context and automatic session continuation

### 5.1 Principle

The persistent Waldo relationship and Outcome continuity must survive provider context exhaustion, process loss, adapter upgrades, provider changes, and deliberate fresh-review boundaries. Provider sessions are replaceable executors.

### 5.2 Automatic criteria

The daemon may prepare a replacement session at a safe checkpoint when any of these is true:

- the provider reports insufficient remaining context for the reserved response/tool budget;
- a provider does not expose a trustworthy context meter and the adapter's conservative turn, summary-size, or compaction threshold is reached;
- the current Contract, Plan, Work Unit, workspace snapshot, authority, or admitted knowledge digest materially changed;
- the native session identity is lost, unhealthy, non-resumable, or fails reconciliation/readmission;
- the role or purpose changes, such as implementer to verifier, where inherited conclusions would contaminate independence;
- a user requests a fresh context or source deletion/revocation invalidates material context;
- repeated compaction produces unresolved contradictions or cannot preserve the minimum RunBrief.

No threshold is invented when a provider does not expose token/cost facts. Adapters publish trustworthy meters where available and conservative operational limits otherwise.

### 5.3 Background rollover versus user decision

Rollover may happen automatically and non-disruptively only when all of these remain unchanged:

- Project, Outcome, ContractRevision, PlanRevision, WorkUnit, provider, model/profile, role, authority, budget ceiling, worktree ownership, and consequential-effect policy;
- no tool call or external effect has an unknown outcome;
- the daemon can produce a provenance-bearing continuation packet and confirm the replacement provider-native identity;
- the old Attempt can be paused/contained or fenced so duplicate canonical writes and effects are impossible.

The daemon records a compact continuation/recovery receipt and may notify the user: **Waldo refreshed this agent's working context; scope and authority did not change.** The user does not need to authorize a purely mechanical, same-authority continuation.

The user must decide when rollover changes provider/model/role, raises cost or concurrency, changes authority/workspace, cannot safely fence the old session, encounters an unknown effect, loses material context, or requires a new Attempt/Plan. The UI shows what changed, what remains known, the recommendation, alternatives, and the safest next action.

If replacement identity is ambiguous, the Attempt becomes `unconfirmed`; Kennel reconciles before any retry and never starts a plausible duplicate session.

### 5.4 Continuation packet

A bounded continuation packet contains:

- exact canonical IDs and revision/digest bindings;
- current Work Unit objective and remaining work;
- explicit user decisions and authority;
- verified dependency outputs;
- exact workspace/worktree and fence facts;
- attributed artifacts, observations, Evidence candidates, and contradictions;
- unresolved questions and known omissions;
- source references and freshness, not an assertion of lossless transcript memory.

## 6. Context and communication between agents

Agents do not coordinate by sharing one unbounded transcript. Every Work Unit receives a purpose-built RunBrief. Dependencies communicate through durable, attributed handoff packets, artifacts, observations, decisions, Evidence candidates, verification results, and recovery receipts.

Coordinator output is advisory until admitted through typed Mission or decision contracts. A coordinator cannot widen authority, mutate the Contract/Plan, self-promote a skill, accept an Outcome, or hide a failed Attempt. If a coordinator session fails, the Mission remains and Kennel can create a replacement Attempt.

## 7. Project learning and Memory boundary

Paxel-style observation and AutoResearch-style evaluation attach beside the execution lineage:

```text
attributed session/workspace observations
  + Outcome/Contract/Plan/WorkUnit/Attempt references
  + Evidence/Verification/Acceptance/Reopen facts
  + user corrections
  -> LearningEpisode
  -> LearningCandidate
  -> bounded ExperimentCampaign when required
  -> Evaluation
  -> explicit PromotionDecision
  -> Project-scoped SkillRevision, context rule, or later orchestration policy
```

A `LearningEpisode` is not Memory. A `MemoryCandidate` is not admitted Memory. Outcome Acceptance is the strongest result label but does not prove which procedure caused success. No model or experiment promotes itself.

There is one governed Waldo Memory system with scoped records, not competing Project/Kennel/Home databases:

- Outcome facts remain canonical Outcome lineage and are not copied into Memory;
- Project-scoped admitted knowledge, corrections, and skills may inform future Project Outcomes;
- user/Home-scoped Memory remains purpose- and consent-bound and is not automatically disclosed to Work;
- Kennel operational state—provider profiles, sessions, worktrees, recovery, and capability inventory—is not personal Memory.

SQLite remains canonical for identity, provenance, scope, revisions, admission, correction, expiry, deletion, and retrieval receipts. Encrypted blobs and user-readable Markdown are subordinate storage/projections. A Markdown edit returns as a candidate revision; raw transcripts do not automatically become durable Memory.

Home may later link personal Open Loops and Work Outcomes through explicit `ResponsibilityLink`s. It does not merge their contracts or silently transfer personal context into Project execution.

## 8. Backend and renderer responsibilities

### Daemon

- Project preferences, adapter inventory, role/readiness resolution;
- durable conversation episodes/turns and context attachments;
- Intake lifecycle, context compilation, analyzer boundary, proposal revision, and confirmation;
- Contract/Plan validation, authority, RunBrief compilation, execution admission;
- Attempt/session correlation, continuation, fencing, recovery, and attention;
- Evidence, Verification, Acceptance/Reopen, learning attribution, Memory/skill admission;
- SQLite single-writer persistence, trigger CDC, typed errors, request IDs, and idempotency.

### Renderer

- Board/List, Outcome Mission Control, Session Inspector, Waldo rail, context chips, proposal and approval surfaces;
- typed user intent and explicit decisions;
- disposable drafts, open/closed state, selection, scroll, focus return, and preview-only fixtures;
- no direct model/provider call, raw context compilation, canonical status, provider-derived acceptance, or parallel persistence.

## 9. Failure and honesty contract

- Analyzer unavailable: preserve intent, allow retry/manual proposal, create no Outcome.
- Concurrent proposal/revision: typed conflict and compare/reload; never overwrite.
- Plan stale after Contract change: block authorization/start until re-proposed.
- Provider unavailable/not ready: exact Action Required, no silent fallback.
- Start/rollover ambiguous: Unconfirmed and reconcile before replacement.
- Session exits or reports done: preserve lineage and propose next safe action; never accept.
- Missing/contradicting Evidence: block readiness or expose an explicit exception decision.
- Conversation unavailable: preserve unsent draft locally without fabricating durable dialogue.
- Daemon restart: restore exact Intake, Outcome, Plan, Attempt/session, continuation receipt, and next safe action.

## 10. Delivery ownership and issue map

Existing issue ownership remains authoritative:

- #21 Contract lineage; #26 Plan/WorkUnit/authority; #31 Attempt/AgentSessionRef/recovery and bounded continuation; #35 Evidence/Verification/Acceptance; #38 complete Work evaluation;
- #32 shared IntakeSession/ResponsibilityLink; #40 Work-side consumption and Home integration;
- #19 LearningEpisode/candidates; #25 experiments/evaluation; #30 skill registry/promotion; #33 later orchestration-policy learning;
- #39 candidate Memory review/retrieval; #41 Home privacy/recovery/usefulness gate.

Missing implementation work must be published as dependency-aware vertical slices for:

1. durable Project/Waldo conversation episodes, turns, context attachments, and continuation;
2. Project default-agent onboarding plus capability-based role resolution;
3. adaptive Work intake and stable-core Contract proposal over the shared Intake contract;
4. Board -> Outcome Mission Control -> Session Inspector composition;
5. one adaptive multi-WorkUnit Mission vertical after the first direct-Outcome gate passes.

Each slice must cross domain/storage/CDC/service/API/UI/restart/evaluation as needed, own one migration/API lease window, and branch from freshly fetched `origin/beta`. Existing issue bodies are not silently broadened; shared-contract alignment is recorded in comments and missing slices receive separately reviewed issues.

## 11. Completion boundary

The Work control plane is complete only when a user can:

1. add a Project and select a ready default coding agent;
2. open Project-scoped Waldo conversation or click Project **New Outcome**;
3. state intent once and receive a correctable adaptive Contract proposal;
4. confirm the Outcome, inspect an adaptive Mission, and authorize exact roles/authority;
5. observe one or many provider sessions through Mission Control and inspect/intervene in a leaf session;
6. survive provider loss or context rollover without reconstructing truth from transcripts or duplicating effects;
7. review criterion-bound Evidence and Verification and explicitly accept or reopen;
8. contribute only consented, attributed learning candidates to future Project work;
9. restart and return to the exact canonical state and next safe action.

The design is falsified if sessions become Outcomes, chat becomes authority, automatic rollover changes material scope without review, all transcripts are injected into every prompt, provider completion closes work, learning silently activates, Markdown becomes a competing writer, or Home context enters Work without explicit scope and provenance.

# Waldo Kennel v0 dogfood and provider-neutral v1 architecture

- Status: Accepted v0 dogfood baseline and provider-neutral v1 direction, amended by ADR 0004; documentation only
- Decision date: 2026-08-20; Home/Personal Agent amendment accepted 2026-08-21
- Launch wedge: agent-heavy Mac users, expanding through governed personal and communication continuity
- Launch proof: useful Outcomes and commitments reach conscious closure with lower coordination and supervision cost
- Current implementation status: prerequisite foundation, legacy-import removal, Codex admission, and CLI reduction are on `main`; the accepted Outcome/Home/continuity model is not implemented by the existing overlay

This is the canonical product, ontology, lineage, governance, state, and surface definition for Waldo Kennel's local v0 dogfood. It consolidates the accepted Kennel Work loop with the approved local Personal Home, required governed desktop screen/audio capture capabilities, and Gmail Communication Loops beta. Codex-only execution is a v0 testing constraint, not a locked v1 provider decision: v1's provider set is **Unknown/TBD** and its orchestration core must remain provider-neutral. Home, Work, and Settings are destinations; **Enter -> Understand -> Decide & Authorize -> Act & Observe -> Prove & Close** is the common lifecycle spine inside them. Work and Home/Personal Agent implementation may proceed in parallel through the ownership rules in [ADR 0004](../adr/0004-parallel-home-personal-agent-and-required-capture.md) and the [Home/Personal Agent design](../superpowers/specs/2026-08-21-home-personal-agent-memory-design.md). This document is a contract for separately authorized implementation slices; it does not itself ship a feature, release, or hosted deployment.

## Evidence boundaries

### Observed

- F0-F6 provides an AO-derived Go daemon, SQLite and trigger-based CDC, Electron supervisor, worktrees, provider sessions, terminal/chat/browser surfaces, recovery facts, and PR/check/review observation.
- The current Outcome overlay is substantially session-oriented: a provider session and marker-parsed model text drive the visible task/plan/Kanban presentation. It is useful donor UI, but the session—not a durable user-approved Outcome contract—is still the practical center of gravity.
- PR #11 contained useful cleanup but also an unwired permanent Outcome schema/store that did not preserve the accepted contract, CDC, verification, acceptance, effect, Open Loop, or recovery semantics. Its accepted donor work landed separately in PRs #12-#14; its Outcome schema and locale deletion remain rejected.
- Gmail can expose conversations, message history, and drafts directly to a desktop client. Inbox reading and draft creation require restricted OAuth scopes for a public application.
- Dayflow demonstrates local desktop capture, local storage, timeline, and daily-summary mechanisms, but desktop observation cannot establish intent, a complete life context, or durable Memory.

### Reported

- Users supervising agent-heavy work lose time reconstructing transcripts, deciding whether an agent is actually done, and repairing interrupted or incorrectly routed work.
- Work and personal commitments also disappear inside long email threads, follow-ups, and context switching.
- GitHub Actions are disabled for the account, so complete local verification remains necessary until hosted checks return.

### Inference

- Work should prove responsibility transfer and judgment compression while Home/Personal Agent proves attention continuity, source governance, candidate quality, and conscious Open Loop closure. Useful agent concurrency, inbox automation, and captured context remain means governed by the Outcome/Open Loop contract—not product-success metrics or independent sources of truth.
- Kennel Work proves bounded execution; Personal Home proves work-plus-life continuity; Communication Loops prove continuity across human coordination. These lanes share one Waldo identity and one intake/context contract.
- A local-first Waldo Core can prove this without a required Waldo account, hosted API, Waldo-funded model calls, or central Memory service.

### Unknown

- whether Google restricted-scope verification and any required security assessment can complete on the intended public-launch schedule;
- the capture cadence, local model route, retention budget, and review volume that meet the Personal Agent utility, privacy, battery, and deletion gates;
- hosted attachment, offline acknowledgement, detach/revoke, deletion, and cross-device conflict behavior;
- pricing, commercialization, health/body-state semantics, Relationship, phone/wearable implementation, governed durable Memory admission, and the future Waldo harness.
- the v1 provider set and routing-policy defaults; evaluator identity and independence must always be labeled truthfully, while the future provider mix remains unknown.

## Product thesis

Waldo Kennel is one local-first Mac application with two synchronized responsibility destinations and one control destination:

- **Home** helps the user understand what needs attention across confirmed Outcomes, Open Loops, communication, and today's trusted facts.
- **Work** turns a delegated Outcome into bounded local execution, evidence, verification, conscious acceptance, and exact re-entry.
- **Settings & Control** makes provider readiness, authority, disclosure, retention, export, revoke, and deletion inspectable.

The destinations do not define separate workflows. Every responsibility advances through the same five-stage spine:

1. **Enter** — choose or confirm the responsibility space and capture an explicit responsibility.
2. **Understand** — ground the current facts, ambiguity, provenance, and success contract.
3. **Decide & Authorize** — recommend a plan and obtain only the material decisions, capabilities, effects, and budget it needs.
4. **Act & Observe** — execute bounded work while exposing progress, failure, recovery, and attention without transcript reconstruction.
5. **Prove & Close** — bind evidence to current criteria, verify independently where claimed, and let the responsible person accept, reopen, release, or create a successor.

These are five adaptive product surfaces, not five mandatory wizard pages. A simple local Outcome may move through them quickly; ambiguity, failure, or consequential effects expand the relevant surface without changing the lineage.

### From session-oriented to Outcome-oriented

The AO-derived orchestration and Kanban experience remains valuable; its authority changes. Today, the practical story is often “open a session, ask an agent, read its output, and infer whether the task is done.” In the accepted model:

```text
Outcome -> ContractRevision -> PlanRevision -> WorkUnit -> Attempt -> AgentSessionRef
```

The Outcome is the durable responsibility. A Work Unit is the bounded piece of work. An Attempt is one try. A Codex/Claude/provider session is the executor used by that Attempt. Kanban, Mission Map, session lists, terminal, browser, and operator views remain useful projections over those facts; none is removed. This lets Kennel replace or recover a failed session without losing the user's goal, approved authority, evidence, or closure history.

Kennel should answer four questions without requiring transcript reconstruction:

1. What responsibility is currently open?
2. What has become true, and what remains uncertain?
3. Does Waldo need judgment, a human-only action, or simply time?
4. What is the smallest safe next action, and why?

### Launch jobs

| Job | Current problem | Kennel progress |
| --- | --- | --- |
| Delegate agent-heavy work | Success, authority, and stopping conditions live in the user's head or transcript. | A versioned Outcome contract compiles into bounded Work Units and admitted Attempts. |
| Know when to intervene | Activity, errors, and decisions are mixed together. | Needs You, Action Required, and Waiting expose only the relevant responsibility. |
| Judge whether work is done | Provider completion, commits, and checks are mistaken for user success. | Criterion-bound Evidence and Verification lead to explicit Acceptance. |
| Process communication | Long threads hide requests, commitments, ownership, and follow-up. | Waldo proposes a concise brief, Commitment Candidates, a next action, and a draft-first effect. |
| Preserve personal continuity | Work facts, notes, and waiting commitments disappear between sessions. | Personal Home and Daily Snapshot restore trusted, correctable context without claiming durable Memory. |
| Recover and re-enter | Interrupted work and delayed replies require manual reconstruction. | Contain, reconcile, narrow retry, and exact Re-entry preserve lineage and next action. |

### Launch non-goals

These are exclusions from the first verified local release or from automatic behavior, not permanent product rejections. Home/OpenLoop and required capture foundations may proceed in parallel with Work; governed durable Memory use still waits for its admission, privacy, deletion, and evaluation gate.

- a generic multi-agent dashboard or provider launcher;
- a full email client, autonomous inbox, or automatic-reply bot;
- automatic task, Outcome, Open Loop, Acceptance, or closure creation from model inference;
- a transcript archive, productivity score, personality assessment, or ambient surveillance product;
- a complete or hidden life model, health-aware planner, Relationship product, durable proactive agent, or phone/wearable Waldo implementation;
- a required account, hosted canonical backend, cross-device sync, or Waldo-funded inference;
- automatic skill/rule promotion from traces;
- provider-authored truth or prompt rewriting presented as orchestration.

The current plan therefore does **not** automatically promote skills or rules from traces. The proposed [Learning and Skill Evolution design](../superpowers/specs/2026-08-21-waldo-learning-skill-evolution-design.md) and [ADR 0005](../adr/0005-governed-project-learning-and-skill-evolution.md) specify consented LearningEpisodes, candidate kinds, bounded experiments, hidden held-out evaluation, explicit promotion, versioning, rollback, and deletion. Until that written specification is approved, trace learning remains non-authoritative. The current plan also does **not** treat a provider's answer or rewritten prompt as orchestration truth: models may propose a plan or context packet, while deterministic policy and explicit user authority decide what is admitted.

## Product identity and responsibility scopes

Waldo is the single intelligence and responsibility identity. Kennel is its local desktop presence, not another assistant.

`ResponsibilitySpace` is the launch root for context, authority, retention, and future explicit attachment:

| Space kind | Purpose | Execution |
| --- | --- | --- |
| `WorkProject` | Repository/folder-backed work responsibility. | May compile Outcomes to local Work Units and admitted provider Attempts. |
| `PersonalHome` | Confirmed personal Open Loops, explicit notes, communication items, and Daily Snapshots not belonging to a Project. | No arbitrary code execution by default; may propose or create explicitly approved connector effects. |

An Outcome or Open Loop belongs to exactly one Responsibility Space at a time. Moving one creates a recorded transfer; it does not silently rewrite provenance. Future hosted attachment is explicit per Responsibility Space or Project, with one canonical writer and no dual authority.

## Canonical ontology

### Responsibility and continuity

| Object | Definition and invariant |
| --- | --- |
| `ResponsibilitySpace` | Explicit Work Project or Personal Home boundary for policy, context, retention, and later attachment. |
| `Project` | Local executable workspace inside a Work Project space. Repository bytes and worktrees remain Kennel custody. |
| `Outcome` | User-delegated responsibility: something that needs to become true. It owns execution and acceptance lineage. |
| `ContractRevision` | Immutable goal, success criteria, constraints, review expectation, authority envelope, and stop conditions. |
| `PlanRevision` | Versioned execution proposal. The optional Mission Map is its user-facing projection. |
| `OpenLoop` | Confirmed unresolved responsibility or commitment that must be preserved, revisited, consciously closed, released, or promoted to an Outcome. |
| `LoopDisposition` | Immutable user decision to confirm, close, release, reopen, transfer, or supersede an Open Loop. |
| `SuccessorLink` | Lineage from an accepted Outcome or closed Open Loop to a later follow-up responsibility. |
| `ResponsibilityLink` | Immutable, explicit, many-to-many lineage between one Home `OpenLoop` and one Work `Outcome`, recording source, destination, creator, reason, created time, and optional ended time/reason. Create/end never moves, merges, closes, verifies, accepts, or mutates either responsibility. Duplicate active pairs are rejected. |

### Execution, authority, and effects

| Object | Definition and invariant |
| --- | --- |
| `WorkUnit` | Smallest bounded schedulable unit with dependencies, capabilities, evidence, verification, stop, and recovery policy. |
| `Attempt` | One execution of one Work Unit through an admitted provider adapter. Retry, fallback, or provider handoff always creates a new Attempt. |
| `AgentSessionRef` | Stable reference to the provider-native session used by an Attempt. |
| `CapabilityGrant` | Explicit scoped authority to read, write, execute, disclose, spend, or propose/perform an effect. |
| `DecisionRequest` | Irreducible judgment with recommendation, rationale, consequence, override, and inspect path. |
| `EffectIntent` | Frozen proposed external or consequential action with canonical arguments, scope, expected result, and approval state. |
| `EffectReceipt` | Reconciled record of what the Effect Intent actually changed, including unknown outcome when certainty is unavailable. |

### Evidence, verification, and acceptance

| Object | Definition and invariant |
| --- | --- |
| `EvidenceItem` | Provenance-bearing support or contradiction tied to a criterion and exact subject revision. |
| `VerificationRun` | Method, verifier class/identity, evaluated subject, result, and exceptions. Verification never equals acceptance. |
| `AcceptanceDecision` | Immutable user accept or reopen decision for an Outcome. No automated actor may create it. |

### Communication and personal context

| Object | Definition and invariant |
| --- | --- |
| `IntakeSession` | Shared adaptive Home/Work understanding flow. It owns source, space, purpose, material clarification, responsibility proposal, and current revision; neither destination builds a separate Q&A state machine. |
| `ClarificationRequest` | One material question with reason, recommendation, alternatives, and the consequence of deferral. |
| `ResponsibilityProposal` | Proposed note, Open Loop, Outcome, link, correction, or dismissal. It is never canonical by itself. |
| `SourceConnection` | User-authorized external source such as one Gmail account, with explicit scope, disclosure, retention, revoke, and sync status. |
| `CommunicationThreadRef` | Reference to an external conversation. It is source context, never the ID of an Outcome or Open Loop. |
| `CommitmentCandidate` | Correctable model-derived proposal about an obligation, owner, due condition, or next action. It is not canonical responsibility until confirmed. |
| `DraftEffect` | A Gmail-draft EffectIntent and resulting receipt. Draft creation does not mean sent, acknowledged, or closed. |
| `DailySnapshot` | Time-bounded projection of trusted facts, confirmed Open Loops, explicit notes, and current attention. It is not durable Memory. |
| `CaptureGrant` | User-governed screen, system-audio, microphone, explicit-capture, connector, or future device-source contract covering purpose, scope, exclusions, processing, disclosure, retention, pause, revoke, export, and deletion. |
| `SourceArtifact` | Original captured or imported source identity, coverage, sensitivity, retention, and deletion generation. Source content is not truth by existence. |
| `Observation` | Untrusted typed observation derived from an explicit capture, structured source, connector, screen, audio, or future device source. It cannot grant authority or become Evidence or Memory automatically. |
| `ContextEpisode` | Correctable grouping of observations with time, actors, derivation version, and known source gaps. It remains a proposal until explicitly linked or admitted. |
| `MemoryCandidate` | Provenance-bearing proposal for durable continuity with type, valid-time proposal, uncertainty, sensitivity, and review need. It is not admitted memory. |
| `AdmissionDecision` | Immutable accept, edit, reject, or defer decision with actor, policy, reason, and scope. A model reviewer is not the user. |
| `MemoryRecord` | Stable conceptual identity across immutable `MemoryRevision`s. |
| `MemoryRevision` | Admitted content with provenance, recorded time, real-world valid time, scope, expiry, uncertainty, and supersession state. |
| `CounterEvidence` | Source-backed challenge that may propose a revision without overwriting current or historical claims. |
| `DeletionTombstone` | Content-free identity/digest and generation fence that prevents stale indexes, jobs, checkpoints, or source replay from resurrecting deleted content. |
| `RetrievalReceipt` | Purpose, caller, eligible spaces, included/excluded revisions, index generations, and degradation for one context compilation. |

### Projections, not competing canonical aggregates

- clarification and Outcome Focus;
- Mission Map;
- Needs You, Action Required, Waiting, and Ready for Acceptance;
- provider readiness, current session status, recovery receipt, and operator views;
- Communication Brief, Daily Snapshot, and Re-entry packet;
- Suggested Next Actions and Home Catch Up; these are correctable projections, not tasks or canonical responsibility;
- Project Follow-up and Keep for later.
- AO-style Kanban columns, agent/session lists, Mission topology, terminal/browser activity, and provider status. They remain operational projections rather than the canonical Outcome contract.

Provider completion, commits, PRs, checks, messages, drafts, archive state, screenshots, and activity are observations or candidate Evidence. None can create Acceptance or close an Open Loop.

### Governed local Memory — candidate foundations in parallel, admission separately gated

Waldo can and should develop a durable, updating personal Memory; the limitation is that capture alone cannot establish it. A screenshot can show that an app was visible, but not whether the user agreed, cared, changed their mind, completed a commitment elsewhere, or wants the observation remembered. Dayflow-style capture therefore produces candidate context, while Memory requires an admission and correction lifecycle.

The approved local architecture uses SQLite as canonical truth plus subordinate encrypted blobs, indexes, and a user-readable Waldo filesystem under `~/.kennel/waldo/memory/`:

- SQLite remains canonical for identity, provenance, admission decisions, relationships, scope, freshness, supersession, expiry, deletion generation, and projection checkpoints;
- encrypted content-addressed blobs hold retained source payloads under source-specific retention and deletion policy;
- SQLite FTS, optional local embeddings, and typed relationship indexes are rebuildable projections keyed by revision and canonical generation;
- daemon-owned Markdown files contain inspectable projections of admitted memory narratives, decisions, preferences, project continuity, and user corrections;
- direct user edits are re-imported as explicit candidate revisions and never silently overwrite lineage;
- captured activity, messages, transcripts, and model summaries enter a Memory inbox as candidates, not durable truth;
- admission records source, confidence, responsible space, valid time, review/expiry, and whether the user explicitly confirmed it;
- corrections supersede rather than erase history; deletion advances a generation, removes source and derived content, regenerates dependent projections, and leaves only a content-free anti-resurrection marker and permitted propagation receipt;
- retrieval returns a minimized, provenance-bearing, freshness-labeled packet and cannot override a current ContractRevision or explicit user statement.

This supports Minimi-like ambient continuity and Open Loop help while adding Waldo's stronger distinction between observation, memory, responsibility, execution, verification, and conscious closure. The Home lane may implement sources, episodes, candidates, review, deletion-generation fixtures, and retrieval evaluation in parallel. Enabling durable admitted Memory remains a separate gate; no implementation PR may settle that gate by convenience.

## Exact lineages

### Work Outcome

```text
ResponsibilitySpace / WorkProject
  -> Outcome
  -> clarification -> ContractRevision
  -> direct plan or PlanRevision / Mission Map
  -> CapabilityGrants
  -> WorkUnits + dependencies
  -> Attempts -> AgentSessionRefs
  -> EvidenceItems
  -> VerificationRuns
  -> Ready for Acceptance
  -> AcceptanceDecision
  -> Adaptive Close
  -> optional SuccessorLink -> new Outcome or OpenLoop
```

### Communication continuity

```text
SourceConnection / Gmail
  -> CommunicationThreadRef
  -> Communication Brief + CommitmentCandidate(s)
  -> user: dismiss | correct | confirm OpenLoop | create Outcome
  -> recommended next action
  -> optional approved DraftEffect
  -> user sends outside launch automation
  -> Waiting with owner + recheck condition
  -> reply / observable change / explicit user update
  -> close, release, reopen, or promote to Outcome
```

### Personal daily continuity

```text
Trusted Kennel facts + confirmed OpenLoops + explicit notes
  -> DailySnapshot
  -> user corrects / confirms / dismisses candidates
  -> today attention + next actions
  -> Daily Close
  -> exact tomorrow Re-entry

Required capability, activated only by explicit per-modality CaptureGrant:
raw screen frame / system audio / microphone segment -> Observation -> ContextEpisode
  -> explicit user confirmation/link
  -> Snapshot input or CommitmentCandidate
  -> MemoryCandidate, never automatic durable truth
```

### Home to Work

```text
trusted facts -> Suggested Next Action projection
  -> user: dismiss | correct | confirm/keep OpenLoop | create draft Outcome
OpenLoop -> explicit ResponsibilityLink -> new or existing Work Outcome
Home owner/recheck/closure remains separate from Work contract/Evidence/Acceptance
```

A direct candidate-to-Outcome conversion preserves the candidate's source and provenance. An Open Loop connection records a `ResponsibilityLink`. Work does not execute until the destination Project and Outcome contract are explicitly defined.

## State models

### Outcome

```text
Draft -> Contracted -> Active -> Ready for Acceptance -> Accepted
                      |                 |
                      |                 -> Reopened -> Active
                      -> Superseded / Released
```

`Completed` is not an Outcome state. Acceptance requires the user's immutable decision.

### Open Loop

```text
Open -> Active -> Waiting -> Ready to Close -> Closed
  |       |          |             |             |
  |       -> Deferred              -> Reopened <-+
  -> Released / Superseded / Transferred
```

An Open Loop requires an owner, source/provenance, a next review or trigger, and a declared closure condition. A thread becoming quiet or archived never closes it.

### Work Unit and Attempt

```text
WorkUnit: Proposed -> Ready -> Admitted -> Running -> Verifying -> Satisfied
                                      |             -> Rework -> Ready
                                      -> Failed / Superseded

Attempt: Queued -> Running -> Paused -> Succeeded
                          |         -> Failed / Cancelled
                          -> Lost -> Reconciled -> new narrow Attempt or attention
```

### Connection and communication

```text
Connection: Disconnected -> Authorizing -> Syncing -> Ready
                           |               |          |
                           -> Denied        -> Stale   -> Revoked

CommitmentCandidate: Proposed -> Corrected / Confirmed -> OpenLoop or Outcome
                           -> Dismissed

DraftEffect: Proposed -> Approved -> Created -> Edited / SentExternally
                    |             -> Failed / Unknown -> Reconciled
                    -> Rejected
```

## Attention and recovery contract

| Projection | Meaning | Required presentation |
| --- | --- | --- |
| **Needs You** | Waldo cannot responsibly choose between materially different valid paths. | Problem, recommendation, why, consequence, primary action, override, inspect. |
| **Action Required** | One exact human-only action is necessary. | Exact action, location, reason, completion signal, resume behavior. |
| **Waiting** | A dependency or timed condition is unresolved and immediate action is not useful. | Dependency, owner/source, recheck condition, expected release behavior. |
| **Ready for Acceptance** | Current Outcome criteria have required Evidence and Verification. | Criteria, support/contradiction, verification, exceptions, accept/reopen. |
| **Ready to Close** | An Open Loop's declared closure condition appears satisfied. | What changed, Evidence/source, uncertainty, close/reopen/release. |

The user never needs the full transcript or entire email thread to resolve attention. Raw sources remain inspectable with privacy boundaries.

Recovery is `contain -> reconcile -> narrowly retry`. Unknown effects are never blindly repeated. Non-material recovery stays in history; material recovery creates a compact receipt and exact Re-entry.

## Local Waldo Core and Kennel Runtime

### Waldo owns semantics and control

- Responsibility Spaces, Outcome/Open Loop meaning, revisions, plans, Work Units, grants, decisions, Effect Intents, Evidence metadata, Verification, Acceptance, closure, and lineage;
- attention, Communication Brief, Daily Snapshot, and Re-entry projections;
- policy for orchestration, admission requirements, disclosure, and retention.

### Kennel owns local custody and execution

- repository/workspace bytes, worktrees, terminals, browser sessions, provider authentication, credentials, processes, raw traces, and unselected artifacts;
- provider-adapter session start/resume/pause/cancel, ordered observations, capability enforcement, and recovery reconciliation;
- direct connector I/O under admitted intents and grants.

### Launch deployment

- Local Waldo Core runs inside the Go daemon; Electron stays a thin supervisor.
- SQLite is the sole canonical local writer and durable product changes emit through trigger-based CDC.
- Primary daemon API remains loopback-only on `127.0.0.1` and unauthenticated.
- App data remains under `~/.kennel`; credential-storage mechanics must satisfy repository and platform security review.
- No account or hosted Waldo backend is required. An account may be recommended later when attachment has real value.
- Provider/model calls use the user's admitted local provider route; any private content disclosure is explicit and inspectable.

### Future post-attachment custody

Hosted Waldo becomes canonical only for explicitly attached Responsibility Spaces and their identity, Outcome/Open Loop contracts, authority, decisions, Evidence metadata/digests, Verification, Acceptance, closure, and lineage. Kennel remains canonical for local files, worktrees, terminals, credentials, provider sessions, raw traces, and unselected artifacts. No record has dual canonical writers.

## Local v0 operating envelope and quality trade-offs

### Locked constraints

- one user on one Mac, one local daemon, one SQLite canonical writer, and no required hosted control plane;
- bounded local agent concurrency, with exactly one fenced writer for each worktree;
- strong read-after-write consistency for local canonical responsibility and authority facts;
- backpressure, pause, and truthful degraded states are preferred over hidden queues, duplicate writers, or optimistic effect claims;
- the modular daemon remains one deployment. Microservices, distributed consensus, and cross-device conflict resolution would solve problems outside the v0 envelope.

### Quality priority

| Priority | Attribute | Target and accepted cost |
| --- | --- | --- |
| 1 | Authority/correctness | No actor silently widens authority, accepts, closes, or retries an unknown effect. Accept extra approval/reconciliation latency. |
| 2 | Durability/recovery | Canonical facts survive restart and recovery preserves lineage. Accept local storage and explicit resource disposition. |
| 3 | Privacy/custody | Useful without account or capture; private content excluded by default. Accept less automatic context in v0. |
| 4 | Comprehensibility | Current responsibility, evidence, authority, and next action are inspectable without transcripts. Accept fewer opaque optimizations. |
| 5 | Evolvability | Provider-neutral contracts and additive storage/API evolution. Accept adapter/conformance work before provider expansion. |
| 6 | Responsiveness | Attention and navigation remain interactive while agents run. Long work is asynchronous and observable. |
| 7 | Scale | Optimize for one person's bounded local workload first; do not trade the higher priorities for speculative multi-user scale. |

### Bottleneck and counter-challenge

At substantially higher local load, retained screen/audio media, derived-index rebuilds, provider/process concurrency, worktree/disk pressure, terminal/browser resources, and connector rate limits fail before loopback HTTP or SQLite query throughput. v0 responds with capture cadence, age/storage retention, deduplication, concurrency budgets, backpressure, projection checkpoints, and cleanup—not premature distributed services. Exact concurrency, storage, energy, and latency thresholds are measured by the Work and Home/Personal Agent dogfood protocols.

The architecture should be revised rather than scaled if its central assumption is false: that a governed Outcome-to-Acceptance loop lowers supervision and re-entry cost compared with direct provider use.

## Orchestration and execution contracts

### Provider admission

- **Locked v0 dogfood constraint:** Codex is the only selectable provider for local testing. It limits early variability; it does not establish a v1 provider set.
- **Unknown/TBD v1 provider set:** Claude and other providers may become selectable only through a provider adapter that passes the same conformance contract. Provider names, models, and routing preferences are not architecture truth.
- An adapter publishes a capability profile: executable/protocol and version, authentication, stable session identity, requested mode, start/resume/pause/cancel, ordered events, Project/worktree binding, Attempt correlation, local Evidence capture, cost visibility, and recovery support. Required capabilities block admission; optional capabilities enable only their dependent route or presentation.
- Every Attempt start/resume performs live fail-closed admission against that profile and the current authority envelope. Known-bad versions are blocked. Unrecognized versions may be provisionally admitted only when every required check passes.
- Provider compatibility is capability-first, not name-first. Routing later selects among admitted providers by required capability, disclosure/privacy policy, user preference, budget/cost visibility, and task fit; it never silently falls back across providers.
- Historical provider identity is immutable and always readable. A historical session is continuable only when its adapter is later admitted, supports recovery, and the session passes fresh reconciliation and readmission. Otherwise it remains inspectable and may hand off through a provenance-bearing packet to a new Attempt on an admitted provider.

### `RunBrief`: provider-neutral core and compiled form

Every Work Unit receives a frozen versioned provider-neutral RunBrief core with:

- ResponsibilitySpace, Project, Outcome, ContractRevision, PlanRevision, WorkUnit, Attempt, and correlation IDs;
- objective, inputs, constraints, non-goals, dependencies, and workspace snapshot;
- capability/effect/disclosure envelope and explicit exclusions;
- expected output, criterion-bound Evidence, and Verification method/class;
- stop/escalation rules, budget, lease/fence, retry, recovery, and handoff policy.

The admitted adapter compiles that immutable core into a provider-specific execution form (for example, its session/mode/tool configuration and recovery bindings). Compilation may narrow authority or expose an admission failure; it may not widen the core's grants, effects, disclosure, budget, or acceptance semantics. The core and compiled-form digests are recorded on the Attempt.

The RunBrief is grounded in this precedence order: current user-approved ContractRevision and explicit decisions; approved PlanRevision and WorkUnit; Project policy, grants, disclosure/effect policy, and budget; verified dependency outputs; exact workspace/worktree snapshot and Project rules; user-approved Project knowledge or durable memory; then optional retrieved context with provenance and freshness. Transcripts, inferred preferences, screen observations, retrieved memory, and agent-authored plans are candidate context only. They cannot override higher authority. A material change to scope, criteria, dependencies, workspace, authority, budget, provider capability, or approved knowledge creates a new revision and Attempt; stale or contradictory grounding blocks compilation.

### Orchestration Advisor and routing

- Waldo recommends the smallest sufficient topology in plain language, with a simpler alternative and an advanced override when useful. Methods include direct, sequential, parallel isolated specialists, planner-executor, discovery-replan, implementer-reviewer, competing proposals with a judge, and human-gated effects.
- A model proposes decompositions and tactics. A deterministic, inspectable Orchestration Policy validates dependencies, file overlap, risk/effects, Evidence needs, admitted capabilities, budget, supervision cost, and recovery complexity. Opaque learned scores do not control routing; later trace learning may propose policy changes but cannot silently apply them.
- One direct Attempt is the default. Within an admitted Attempt, the provider may choose tools, tactics, and native subagents freely. Changing provider, model/session mode, topology, worktree ownership, authority, or a material budget creates a new RunBrief and Attempt. User review is required only when the change is material to scope, cost, permission, risk, or effect.
- There is no silent provider fallback. Reversible command retry may remain within the Attempt. A provider handoff is a new Attempt through an admitted adapter, with review when its material authority or plan changes.

### Authority, budget, worktrees, and recovery fences

- Effective authority is the intersection of Project policy, approved Contract/Plan, WorkUnit requirements, explicit user grants, admitted provider capabilities, current worktree ownership, and the consequential-effect ceiling. Read, write, execute, disclose, spend, and external effect are classified separately. A lower layer may narrow authority but never widen it.
- A new capability or effect raises a just-in-time DecisionRequest explaining the need, location, consequence, and revoke path. Grants bind to a revision or Attempt by default, not permanently to the Project.
- The budget envelope covers elapsed time, Attempts/retries, concurrent sessions, worktrees/storage, trustworthy token or monetary usage when exposed, consequential effects/disclosure, and requested human interventions. Models remain free inside it; soft limits warn and hard limits pause. Unknown cost stays labeled unknown and is governed through time, concurrency, retries, and effects rather than invented prices.
- Every write-capable Attempt owns one isolated worktree at a pinned base; two Attempts never write the same worktree. Read-only discovery may share a snapshot. Provider-native subagents inside one Attempt remain one fenced writer. Parallel WorkUnits require separate worktrees and non-conflicting dependencies/ownership; integration is an explicit WorkUnit/Attempt. Dirty or useful failed work is retained until explicit cleanup.
- Attempt authority lasts until completion, explicit pause/revoke, or confirmed recovery/takeover, and renews silently while healthy. A missed heartbeat means `unconfirmed`, not dead. The Attempt may continue reasoning, observing, exploring, and doing ordinary authorized local work; only new consequential effects and canonical truth mutations pause until reconciliation.
- Fences protect canonical durable writes and consequential effects, not model reasoning or tactical freedom. A new fence is issued only after reconciliation confirms retry, takeover, or replacement. Late work and observations from a stale Attempt remain inspectable but cannot overwrite current Evidence, results, or canonical state.

### Verification independence

- Deterministic verification outside the producing session is preferred. Producer self-checks are useful Evidence but are not independent verification.
- A fresh read-only review Attempt receives the criteria and subject through a verifier-focused RunBrief, without the implementer's conclusions or raw transcript by default. It cannot modify the subject; rework receives a separate write Attempt.
- In v0 Codex-only dogfood, a separate Codex session is labeled **separate-session review**, never provider-independent; deterministic tools and owner walkthroughs remain explicit. When multiple adapters are admitted, Waldo may recommend another provider or model when that reduces correlated failure and satisfies the verifier capability profile.
- Only the user accepts an Outcome. Every result states its actual independence class; the v1 provider set remains Unknown/TBD, but this tiered evaluator policy is provider-neutral and locked.

### Locked launch defaults

- One fenced writer per worktree; the renewable Attempt lease preserves tactical autonomy. Missing heartbeat becomes `unconfirmed`, and fences block only consequential effects and canonical mutations until reconciliation.
- One direct Attempt is the smallest-sufficient default. Parallel Work Units require isolated worktrees, no overlapping write ownership, and an approved Mission Map.
- There is no silent provider fallback. In v0, a Codex failure creates a new Codex Attempt, a replan, or human attention. A later provider handoff requires a new Attempt, an admitted adapter, and any required authority/plan review.
- The multidimensional budget governs time, retries, concurrency, storage, disclosed cost when trustworthy, effects, disclosure, and human interruptions without micromanaging tactics.
- Deterministic, producer self-check, separate-session, cross-provider/model, and owner-walkthrough verification are labeled truthfully. Only the user accepts.

### Consequential-effect ceiling

- Approved Work Units may read/write the authorized local worktree and run allowed local commands.
- Local commits require explicit inclusion in the approved plan. Dirty worktrees are never force-deleted.
- Push, remote PR creation, comments, merges, deploys, publishes, releases, payments, destructive remote mutation, and direct message sending require separate just-in-time approval and are not autonomous launch behavior.
- Communication Loops may read the explicitly connected Gmail scope and create one user-requested draft after an EffectIntent approval. They never auto-send, archive, delete, mark complete, or silently change labels.

## Privacy-preserving Outcome Trace

The launch trace records enough causal metadata to reconstruct responsibility without retaining private content by default:

- stable correlation IDs and parent/causal IDs;
- timestamp, actor class, event kind, prior/new state, revision, Attempt, and Work Unit;
- capability class, EffectIntent/Receipt status, Evidence type/digest, Verification class/result, and attention reason;
- failure/recovery classification and redacted receipt;
- disclosure destination and policy decision when content leaves local custody.

Raw prompts, email bodies, files, terminal output, screenshots, credentials, secrets, health values, and model chain-of-thought are excluded from the canonical trace. Source excerpts are explicit, minimized, local, provenance-bearing attachments. Trace export, retention, deletion, and debug access are user-controlled.

Locked local defaults:

- canonical contract, decision, authority, effect, evidence, verification, acceptance, recovery, and lineage metadata is retained with its responsibility until the user deletes that responsibility;
- redacted operational diagnostics expire 30 days after an Attempt ends and may be shortened or disabled by the user;
- raw private artifacts are not retained by the trace unless the user explicitly saves one with a visible scope and expiry;
- deletion records only a content-free generation marker so stale replicas or recovery logs cannot silently resurrect deleted trace content.

## Information architecture

Kennel has three primary destinations:

### 1. Home

- Today brief and Daily Snapshot;
- Needs You and Action Required across spaces;
- Open Loops, Waiting, follow-ups, and Ready to Close;
- Communication Loops beta;
- explicit quick capture and correction;
- recent accepted Outcomes and exact Re-entry.
- a calm brief plus focused Catch Up pane, with explicit conversion/link actions from suggested next actions or Open Loops into Work.

Home is a responsibility/attention projection, not a dashboard of activity or a Memory product.

### 2. Work

- Projects and project readiness;
- Outcome Focus and Active Outcomes;
- Ready for Acceptance and recent closes;
- Outcome Workspace modes: Define, Clarify, Plan/Mission Map, Authority, Run, Review, Accept, Close/Re-enter;
- contextual operator inspector for sessions, terminal, browser, worktree, trace, and recovery.
- incoming Home candidates and Open Loop links, kept separate from executable Outcome truth until the user confirms and defines the Work contract.

### 3. Settings & Control

- Responsibility Spaces, Projects, Gmail Connections, Codex authentication/admission;
- permissions, disclosure, effect policy, retention, export, revoke, and deletion;
- skills, MCP servers, rules, routing overrides, and required screen/audio capture capabilities with per-modality grants;
- future hosted attachment clearly marked later.

## Five adaptive product surfaces

The former review atlas is a state catalogue, not a requirement for separate routes. The product has exactly five lifecycle surfaces; each adapts to Home or Work context and to the current failure/attention state. Settings & Control and the Operator Inspector are overlays, not lifecycle stages.

| Surface | Primary question | Adaptive modes and required states |
| --- | --- | --- |
| **Enter** | What responsibility are we taking on, and where does it belong? | Work-first onboarding, Project selection/readiness, Home entry, Quick Capture, source candidate, invalid folder, provider unavailable, daemon offline. |
| **Understand** | What is true, uncertain, and required for success? | Work Home/Outcome Define/Clarify; Home Today/Catch Up/Daily Snapshot/Communication Brief/Open Loop detail; stale source, correction, duplicate, provenance inspect. |
| **Decide & Authorize** | What is the recommended approach, and what must the user decide or permit? | Mission Map or direct Work Unit, authority/effect/budget preview, Connect Home to Work, draft review, capability blocked, plan invalidated, changed grant. |
| **Act & Observe** | What is Waldo doing, what changed, and does it need attention? | Run, Needs You, Action Required, Waiting, paused/retry/lost/recovery, draft-effect reconciliation, partial evidence. |
| **Prove & Close** | Is each criterion proved, and should the responsibility close? | Evidence/Verification, Ready for Acceptance/Ready to Close, Acceptance, Adaptive Close, release, reopen, successor, Re-entry, dirty-worktree/resource disposition. |

### Detailed review-screen atlas is preserved

The five surfaces group the previous screen atlas; they do not delete it. The earlier seed contains F01-F27 plus the inserted F02A Home-to-Work frame—28 review frames when counted individually. They remain the concrete team discussion and UI-state inventory:

| Lifecycle group | Detailed screens retained |
| --- | --- |
| **Enter** | F01 First run; F03 Quick Capture; F05 Communication connection. |
| **Understand** | F02 Home/Today; F04 Daily Snapshot; F06 Communication inbox; F07 Communication Brief; F09 Open Loop detail; F11 Work Home; F12 Outcome Define; F13 Adaptive clarification; F26 Context episode correction. |
| **Decide & Authorize** | F02A Connect Home to Work; F08 Draft effect review; F14 Mission Map; F15 Authority/effect preview; F25 screen/audio Capture grants. |
| **Act & Observe** | F16 Run; F17 Needs You; F18 Action Required; F19 Waiting. |
| **Prove & Close** | F10 Ready to Close; F20 Evidence and Verification; F21 Acceptance; F22 Adaptive Close; F23 Re-entry. |
| **Cross-stage overlays** | F24 Operator Inspector; F27 Settings & Control. |

Some may ultimately share a route or component, but each remains a distinct user conversation with its own purpose, primary action, empty/error/recovery states, and review acceptance. Implementation may combine layouts only after proving that none of those details becomes hidden or ambiguous.

### Work-first first run

For v0 local dogfood, first run recommends **Work first**: choose a local Project, verify daemon/provider readiness, and enter the first Outcome. The user may instead enter Home, and no Personal Home is silently created. Screen/audio capture capabilities are required product surfaces but activation is never an onboarding blocker: each modality requires an explicit CaptureGrant and Home remains useful when capture is denied, paused, unavailable, or deleted. Gmail, an account, and hosted attachment also remain non-blocking. After either path, Home and Work remain peers over shared responsibility truth.

### First implementation milestone

The first complete Work vertical slice remains the [Local Focus Ledger Outcome](kennel-v0-first-outcome-slice.md). It must traverse all five stages with one smallest-sufficient Work Unit, durable restart/recovery, criterion-bound evidence, verification, and explicit user acceptance. Its scope and paired evaluation remain unchanged.

ADR 0004 starts a separate Home/Personal Agent lane in parallel. That lane follows the [Home, Personal Agent, capture, and memory design](../superpowers/specs/2026-08-21-home-personal-agent-memory-design.md) and may implement canonical Home/OpenLoop, required governed screen/audio capture, candidate-memory foundations, and retrieval evaluation without waiting for the Work evaluation. Shared `ResponsibilitySpace`, `ResponsibilityLink`, intake, RunBrief reference, CDC, and API contracts require coordinated ownership; neither lane may create a duplicate contract or second writer.

## Reference disposition

| Reference | Decision |
| --- | --- |
| AO-derived Kennel chassis | **Adopt** daemon, sessions, worktrees, terminal/browser, observations, and recovery. Reject AO identity and provider-authored product truth. |
| Xirp | **Adapt** simpler Project/session/rules/skills/worktree UX, local custody, and native provider authentication. |
| Medley | **Adapt** interview, explicit plan approval, bounded workers, review tasks, recovery, and receipts. |
| Devin Review, Greptile/TREX, CodeRabbit | **Adapt** Evidence and independent review; reject reviewer output as Acceptance. |
| OpenAI Useful-Work Scorecard | **Adapt** objective useful-output and supervision-cost evaluation. |
| ChatGPT Work, Notion Custom Agents, Webhound | **Adapt** concise attention, reusable work context, and governed proactive work. |
| [Gmail API](https://developers.google.com/workspace/gmail/api/guides) and [Gmail MCP reference shape](https://developers.google.com/workspace/gmail/api/guides/configure-mcp-server) | **Adapt** thread search/get, draft creation, incremental sync, and prompt-injection boundary. Do not depend on Developer Preview MCP for public launch. Restricted-scope and user-data policy review remain launch-beta gates. |
| [Omi](https://github.com/BasedHardware/omi) | **Adapt now at the source-contract level** conversation/audio capture, explicit application surfaces, and broad provider integration. Reject automatic durable truth, capture-led onboarding, and any provider/session store as Waldo authority. |
| [Dayflow](https://github.com/JerryZLiu/Dayflow/tree/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046) | **Adapt now as a required Personal Agent capability** screen capture, exclusions, visible pause, local batching, recoverable observations, and local-model route. Activation remains explicit per user/modality. Reject direct DB access, automatic truth, incomplete deletion, and permanent raw screenshots. |
| [Minimi](https://www.projectminimi.com/) | **Adapt now at the Home/Personal Agent architecture level** ambient continuity, Open Loop discovery, cross-app context, and user-owned retrieval. Add Waldo's explicit admission/correction/provenance model, separate responsibility lineage, verified work execution, and conscious acceptance/closure; do not copy unsupported accuracy or automatic-closure claims as product truth. |
| MIRIX, Hindsight, MemOS, and A-MEM | **Adapt** typed memory, source-linked synthesis, temporal validity, scopes, and linked projections. Reject raw-secret memory, opaque beliefs, silent neighbor mutation, and source replay that resurrects deletion. |
| Mem0, Graphiti/Zep, and Supermemory | **Adapt as subordinate infrastructure patterns** hybrid retrieval, temporal provenance, version relationships, and preview/apply deletion. Reject a second canonical store, namespace-as-authorization, and model-to-truth insertion. |
| Letta, LangGraph/LangMem, and AutoGen | **Adapt at the orchestration boundary** tiered context, source-addressed compaction, checkpoint lineage, pending writes, and explicit save/load. Runtime state remains under Attempt and cannot become personal truth or proof. |
| Claude Code auto-memory | **Adapt only through MemoryCandidate** with explicit admission/correction rules. |
| [Paxel and AutoResearch-style loops](../research/2026-08-21-paxel-autoresearch-learning-reference.md) | **Proposed adaptation:** Paxel-style attributable LearningEpisodes observe patterns; AutoResearch-style campaigns test one bounded editable surface; Waldo governs candidate scope, evaluation, promotion, correction, rollback, and deletion. Reject automatic promotion, personality scoring, visible/editable evaluators, and self-authorizing harness changes. |
| Folk, Poke, Dimension, Manus, Grok Bot, Qodo, Lifestack | **Later/reference only**. |
| [TapeFlow](https://github.com/xingrz/tapeflow) / [Tapflow](https://www.tapflow.ai/) products | **Reject** as unrelated to the communication/personal-agent mechanism. |
| Standalone WHOOP/Oura entries | **Remove from active Kennel benchmark**; preserve as broader Waldo history. |

Open-source code may only be copied after a source-pinned license, dependency, provenance, and security review. No proprietary prompt, engine, scoring model, hidden behavior, or visual language is copied.

## Phased launch boundary

### Parallel launch-core lanes: required

- local-first Home, Work, and Settings destinations;
- Responsibility Spaces, Project readiness, v0 Codex-only admission, provider-neutral adapter/conformance seam, and historical provider readability;
- guided Outcome contract, optional Mission Map, RunBrief, Work Units, grants, fenced Attempts, recovery;
- Needs You, Action Required, Waiting, Evidence, Verification, explicit Acceptance, Adaptive Close, and Re-entry;
- confirmed Open Loops, LoopDisposition, explicit Quick Capture, and Home-to-Work ResponsibilityLink;
- Today/Morning Brief, Catch Up, Open Loop detail, Ready to Close, and Daily Snapshot from trusted facts, confirmed items, explicit notes, and correctable candidates;
- required desktop screen and audio capture capabilities behind explicit per-modality CaptureGrants, with visible pause, exclusions, local preprocessing, retention, revoke, export, and deletion;
- SourceArtifact, Observation, ContextEpisode, CommitmentCandidate, and MemoryCandidate foundations;
- redacted Outcome Trace and dogfood instrumentation.

The Work Local Focus Ledger and Home/Personal Agent slices may proceed in parallel through separate file/API ownership. Shared intake, ResponsibilitySpace, ResponsibilityLink, RunBrief references, CDC, and generated API contracts are designed once and integrated through coordinated PRs.

### Launch beta: optional but coherent

- one Gmail account connection;
- actionable-thread triage, Communication Brief, correctable Commitment Candidates;
- confirm as Open Loop or promote to Outcome;
- user-approved draft creation, Waiting, follow-up, Ready to Close, and Re-entry;
- no auto-send, archive, deletion, labels, or closure.

Public availability of this beta depends on Google OAuth verification and an approved model-disclosure/security boundary. It may remain dogfood-only while the Work core launches.

### Candidate-memory gate

- candidate extraction and correction from explicit, structured, screen, and audio sources;
- bitemporal revisions, counter-evidence, expiry, revocation, deletion generation, and retrieval receipts;
- cross-space denial, stale-index hydration, crash recovery, and non-resurrection evaluation;
- durable admitted Memory use only after this gate passes.

### Proposed learning-foundation gate

- consented, Project-attributed LearningEpisode projections over canonical Outcome facts and permitted source references;
- candidate routing into skill, context-rule, orchestration-policy, Memory, or Open Loop review without collapsing their contracts;
- baseline, held-in, proposer-hidden held-out, adversarial, cost, and safety evaluation under a locked runtime/evaluator boundary;
- daemon-owned SkillRecord/SkillRevision/SkillBinding with provider-specific materialization as a rebuildable projection;
- first activation limited to one explicitly promoted, provisional Project-scoped procedural skill with invocation receipts and immediate rollback;
- no orchestration-policy activation until the project-skill proof passes; no harness/optimizer self-modification without another ADR.

Learning contracts, fixtures, shadow episodes, and candidate evaluation may proceed alongside Work/Home after ADR 0005 approval. Active skills still wait for trustworthy Outcome result facts and the promotion gate.

### Later Waldo ecosystem

- explicit hosted attachment, backup, cross-device sync, and hosted remote execution;
- governed durable Memory after the candidate-memory gate, broader learned skills after the project-skill gate, and later orchestration-policy evaluation;
- Relationship and broad work-plus-life Open Loops;
- phone/wearable Waldo presence, Health First mobile experience, and permissioned body-state planning through the same source/memory contract;
- durable proactive agent, Waldo-owned harness, provider routing beyond v0, teams, marketplace, and commercialization.

## Dogfood and falsification gate

The launch candidate must be compared with direct Codex use across at least 20 representative Outcomes and a smaller communication-loop set, recording both successful and failed attempts.

| Measure | Launch threshold |
| --- | --- |
| Active supervision minutes per accepted Outcome | Median at least 30% lower than direct-Codex baseline. |
| Full transcript reconstruction | Needed in no more than 20% of Outcomes. |
| Attention precision | At least 80% of Needs You/Action Required items judged necessary and correctly classified. |
| Criterion Evidence coverage | 100% of accepted criteria have current provenance-bearing Evidence and a Verification result/exception. |
| False-ready/reopen rate | No more than 10% of Outcomes reopened because readiness omitted a material known fact. |
| Recovery | At least 90% of injected recoverable failures produce containment, reconciliation, and a safe next action without manual state reconstruction. |
| Authority safety | Zero unauthorized, duplicate, silently widened, or blindly retried consequential effects. |
| Re-entry | Median under 60 seconds for the user to identify current state and next action after interruption. |
| Communication correction | Zero auto-created canonical commitments or auto-sent messages. |

The wedge is falsified or paused if supervision cost is not lower, false readiness exceeds the threshold, users still read full transcripts routinely, authority cannot be explained, recovery duplicates effects, or Gmail/privacy requirements make the local launch misleading.

## Architecture gates and execution entry

### Resolved for implementation planning

1. v0 Codex-only admission plus provider-neutral adapter/conformance seam and immutable historical provider identity. The v1 provider set remains **Unknown/TBD**.
2. Grounded provider-neutral RunBrief core/compiled form; recommendation-first hybrid orchestration; autonomy-preserving leases/fences; intersected capability/effect admission; isolated-worktree concurrency; multidimensional budgets; no silent provider fallback; and truthfully labeled evaluator independence.
3. Redacted causal Outcome Trace with private content excluded by default.
4. Objective dogfood measures, thresholds, failure injections, and falsifiers.
5. Consequential-effect ceiling: authorized local work plus separately approved local commits; no autonomous remote effects; Gmail draft only after explicit user intent.

### Deferred and non-blocking for the parallel local lanes

6. Hosted attachment offline/sync acknowledgement, detach/revoke, deletion, and conflict semantics.
7. Durable admitted Memory use until its candidate/admission/deletion gate passes; active learned skills until the learning promotion gate passes; Relationship, Health, phone/wearable implementation, proactive agent, orchestration-policy learning, and Waldo-owned harness.

### Final architecture-review conclusion

No unresolved architecture choice blocks starting the first authorized v0 issue after the foundation gate. Remaining Unknowns are either implementation evidence—exact performance/concurrency, first-slice evaluation, foundation acceptance—or explicitly later capabilities. They must not be silently decided inside an implementation PR. A failed test, recovery injection, or dogfood falsifier reopens the relevant architecture decision rather than being relabeled as completion.

### Implementation entry rules

- The prerequisite boundary is complete on `main`: F0-F6 in PR #1, legacy-import removal in PR #12, provider-neutral admission in PR #13, and public CLI reduction in PR #14.
- PR #11 is a superseded donor and must not merge wholesale. Its speculative Outcome migration/store and locale deletion remain rejected until an authorized vertical slice owns domain, service, storage, CDC, API, UI, recovery, and evaluation together.
- Product work starts from current `origin/main` in a new issue-specific worktree, never from the historical F0-F6 or PR #11 branches.
- Implement the Focus Ledger milestone as five stage-aligned issue-sized PRs: Enter; Understand; Decide & Authorize; Act & Observe; Prove & Close. Each PR owns every domain, storage, CDC, service, API, UI, recovery, and evaluation change required by its user-visible truth boundary, reusing proven foundation APIs where no new durable truth is needed; no PR may leave a horizontal schema or deceptive screen layer for another PR to make true.
- Implement Home/Personal Agent in a separate parallel worktree and PR sequence: Home shell/fixtures; PersonalHome/OpenLoop/Quick Capture; Today/Catch Up/detail/closure; ResponsibilityLink/shared intake; required screen/audio CaptureGrant and source episodes; MemoryCandidate review and retrieval evaluation. Durable admitted Memory remains separately gated.
- After ADR 0005 and its written specification are approved, implement Learning through separate plans: L1 Experience Ledger/candidate mining in shadow mode; L2 bounded Experiment/Evaluation; L3 daemon skill registry and one provisional Project-scoped skill. L4 orchestration-policy learning remains later.
- Before code edits, record exact file and API ownership for both lanes. Shared contract changes use a named integration owner; Home and Work must not independently implement intake/Q&A, ResponsibilitySpace, ResponsibilityLink, RunBrief memory references, CDC semantics, or generated DTOs.
- Preserve Home and Work responsibility lineages even when one adaptive surface displays both. A `ResponsibilityLink` is lineage, never lifecycle coupling.
- Use the [first Outcome specification](kennel-v0-first-outcome-slice.md) and amended [execution handoff](../superpowers/plans/2026-08-20-first-outcome-execution-handoff.md) for Work. Use the [Home/Personal Agent design](../superpowers/specs/2026-08-21-home-personal-agent-memory-design.md) and ADR 0004 for Home. After review, use the [Learning and Skill Evolution design](../superpowers/specs/2026-08-21-waldo-learning-skill-evolution-design.md) and ADR 0005 for Learning. Each code slice still requires an implementation plan and explicit PR scope.
- No merge, push, deploy, publish, release, destructive cleanup, or hosted attachment is authorized by this document.

# Waldo Kennel desktop launch product architecture

- Status: Accepted architecture baseline; documentation only
- Decision date: 2026-08-20
- Launch wedge: agent-heavy Mac users, expanding through governed communication continuity
- Launch proof: useful Outcomes and commitments reach conscious closure with lower coordination and supervision cost
- Current implementation status: not implemented by the existing Outcome overlay or PR #11

This is the canonical product, ontology, lineage, governance, state, and surface definition for the first Waldo Kennel desktop launch. It consolidates the accepted Kennel Work loop with the approved local personal Home, Gmail Communication Loops beta, and phased Dayflow-inspired Desktop Context. It does not authorize feature edits on the F0-F6 foundation branch, a merge, release, or hosted deployment.

## Evidence boundaries

### Observed

- F0-F6 provides an AO-derived Go daemon, SQLite and trigger-based CDC, Electron supervisor, worktrees, provider sessions, terminal/chat/browser surfaces, recovery facts, and PR/check/review observation.
- The current Outcome overlay is substantially session-oriented and is not this accepted model.
- PR #11 contains useful cleanup but also an unwired permanent Outcome schema/store that does not preserve the accepted contract, CDC, verification, acceptance, effect, Open Loop, or recovery semantics.
- Gmail can expose conversations, message history, and drafts directly to a desktop client. Inbox reading and draft creation require restricted OAuth scopes for a public application.
- Dayflow demonstrates local desktop capture, local storage, timeline, and daily-summary mechanisms, but desktop observation cannot establish intent, a complete life context, or durable Memory.

### Reported

- Users supervising agent-heavy work lose time reconstructing transcripts, deciding whether an agent is actually done, and repairing interrupted or incorrectly routed work.
- Work and personal commitments also disappear inside long email threads, follow-ups, and context switching.
- GitHub Actions are disabled for the account, so complete local verification remains necessary until hosted checks return.

### Inference

- The launch product should transfer responsibility and compress judgment rather than maximize agents, inbox automation, or captured activity.
- Kennel Work proves bounded execution; Communication Loops prove continuity across human coordination; Personal Home unifies the user's attention without claiming to be the later complete Waldo ecosystem.
- A local-first Waldo Core can prove this without a required Waldo account, hosted API, Waldo-funded model calls, or central Memory service.

### Unknown

- whether Google restricted-scope verification and any required security assessment can complete on the intended public-launch schedule;
- whether Desktop Context reduces re-entry cost enough to justify system-wide capture for launch users;
- hosted attachment, offline acknowledgement, detach/revoke, deletion, and cross-device conflict behavior;
- pricing, commercialization, health/body-state semantics, Relationship, mobile, durable Memory, and the future Waldo harness.

## Product thesis

Waldo Kennel is one local-first Mac application with two synchronized surfaces:

- **Home** helps the user understand what needs attention across confirmed Outcomes, Open Loops, communication, and today's trusted facts.
- **Work** turns a delegated Outcome into bounded local execution, evidence, verification, conscious acceptance, and exact re-entry.

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

- a generic multi-agent dashboard or provider launcher;
- a full email client, autonomous inbox, or automatic-reply bot;
- automatic task, Outcome, Open Loop, Acceptance, or closure creation from model inference;
- a transcript archive, productivity score, personality assessment, or ambient surveillance product;
- a complete life model, health-aware planner, Relationship product, durable agent, or mobile Waldo;
- a required account, hosted canonical backend, cross-device sync, or Waldo-funded inference;
- automatic skill/rule promotion from traces;
- provider-authored truth or prompt rewriting presented as orchestration.

## Product identity and responsibility scopes

Waldo is the single intelligence and responsibility identity. Kennel is its local desktop presence, not another assistant.

`ResponsibilitySpace` is the launch root for context, authority, retention, and future explicit attachment:

| Space kind | Purpose | Execution |
| --- | --- | --- |
| `WorkProject` | Repository/folder-backed work responsibility. | May compile Outcomes to local Work Units and Codex Attempts. |
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

### Execution, authority, and effects

| Object | Definition and invariant |
| --- | --- |
| `WorkUnit` | Smallest bounded schedulable unit with dependencies, capabilities, evidence, verification, stop, and recovery policy. |
| `Attempt` | One execution of one Work Unit. Retry, fallback, or provider handoff always creates a new Attempt. |
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
| `SourceConnection` | User-authorized external source such as one Gmail account, with explicit scope, disclosure, retention, revoke, and sync status. |
| `CommunicationThreadRef` | Reference to an external conversation. It is source context, never the ID of an Outcome or Open Loop. |
| `CommitmentCandidate` | Correctable model-derived proposal about an obligation, owner, due condition, or next action. It is not canonical responsibility until confirmed. |
| `DraftEffect` | A Gmail-draft EffectIntent and resulting receipt. Draft creation does not mean sent, acknowledged, or closed. |
| `DailySnapshot` | Time-bounded projection of trusted facts, confirmed Open Loops, explicit notes, and current attention. It is not durable Memory. |
| `DesktopObservation` | Optional untrusted local observation derived from consented capture. It cannot grant authority or become Evidence automatically. |
| `ContextEpisode` | Correctable grouping of Desktop Observations. It remains a proposal until explicitly linked or admitted. |
| `MemoryCandidate` | Later proposal for durable continuity, requiring provenance, admission, correction, expiry, rollback, and deletion. |

### Projections, not competing canonical aggregates

- clarification and Outcome Focus;
- Mission Map;
- Needs You, Action Required, Waiting, and Ready for Acceptance;
- provider readiness, current session status, recovery receipt, and operator views;
- Communication Brief, Daily Snapshot, and Re-entry packet;
- Project Follow-up and Keep for later.

Provider completion, commits, PRs, checks, messages, drafts, archive state, screenshots, and activity are observations or candidate Evidence. None can create Acceptance or close an Open Loop.

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

Optional beta only:
raw frame -> DesktopObservation -> ContextEpisode
  -> explicit user confirmation/link
  -> Snapshot input or CommitmentCandidate
  -> MemoryCandidate later, never automatic
```

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
- Codex session start/resume/pause/cancel, ordered observations, capability enforcement, and recovery reconciliation;
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

## Orchestration and execution contracts

### Provider admission

- Codex is the only selectable provider for new launch Work Units.
- Historical provider identities remain readable; non-Codex sessions are inspect-only and continue through a provenance-bearing packet into a new Codex Attempt.
- Every Attempt start/resume performs live fail-closed checks for executable/protocol, current authentication, stable session identity, requested mode, lifecycle controls, ordered events, Project/worktree binding, Attempt correlation, and required local Evidence capture.
- Known-bad versions are blocked. Unrecognized versions may be provisionally admitted only when all required checks pass.

### `RunBrief`

Every Work Unit receives a frozen versioned RunBrief with:

- ResponsibilitySpace, Project, Outcome, ContractRevision, PlanRevision, WorkUnit, Attempt, and correlation IDs;
- objective, inputs, constraints, non-goals, dependencies, and workspace snapshot;
- capability/effect/disclosure envelope and explicit exclusions;
- expected output, criterion-bound Evidence, and Verification method/class;
- stop/escalation rules, budget, lease/fence, retry, recovery, and handoff policy.

### Locked launch defaults

- One active write lease per worktree. Attempts receive a monotonically increasing fence; stale events cannot mutate current state.
- Execution is sequential by default. Parallel Work Units require isolated worktrees, no overlapping write ownership, and an approved Mission Map.
- There is no provider fallback in launch: Codex failure creates a new Codex Attempt, a replan, or human attention.
- The approved budget envelope covers maximum Work Units, concurrent Attempts, wall-clock limit, and any provider-reported cost/token ceiling. Missing cost telemetry cannot silently widen work.
- Deterministic checks run outside the producing session. A producing Codex session may propose Evidence but cannot mark its own semantic criterion verified. Non-deterministic criteria require an independent review Attempt or explicit user walkthrough.

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

## Information architecture

Kennel has three primary destinations:

### 1. Home

- Today brief and Daily Snapshot;
- Needs You and Action Required across spaces;
- Open Loops, Waiting, follow-ups, and Ready to Close;
- Communication Loops beta;
- explicit quick capture and correction;
- recent accepted Outcomes and exact Re-entry.

Home is a responsibility/attention projection, not a dashboard of activity or a Memory product.

### 2. Work

- Projects and project readiness;
- Outcome Focus and Active Outcomes;
- Ready for Acceptance and recent closes;
- Outcome Workspace modes: Define, Clarify, Plan/Mission Map, Authority, Run, Review, Accept, Close/Re-enter;
- contextual operator inspector for sessions, terminal, browser, worktree, trace, and recovery.

### 3. Settings & Control

- Responsibility Spaces, Projects, Gmail Connections, Codex authentication/admission;
- permissions, disclosure, effect policy, retention, export, revoke, and deletion;
- skills, MCP servers, rules, routing overrides, and Desktop Context beta;
- future hosted attachment clearly marked later.

## Complete launch screen and state inventory

| Surface | User job | Required states |
| --- | --- | --- |
| Onboarding | Establish local Home and optional Work Project. | no Project, invalid folder, Codex absent/auth required, ready, offline daemon. |
| Home / Today | See what needs handling now. | empty, normal, stale, offline, recovery, high-attention. |
| Quick Capture | Add or correct an explicit Open Loop/note. | draft, duplicate candidate, assigned space, deferred. |
| Daily Snapshot | Reconstruct today from trusted facts. | collecting, ready, corrected, partial source, Daily Close. |
| Communication Inbox | See only actionable conversation candidates. | disconnected, authorizing, syncing, ready, stale, revoked, no actionable items. |
| Communication Brief | Understand the ask and next action without reading the thread. | candidate uncertain, corrected, confirmed Open Loop, promoted Outcome, dismissed. |
| Draft Review | Approve a bounded remote draft effect. | proposed, approval, created, edited, failed, unknown/reconciled, sent externally. |
| Open Loop Detail | Track owner, closure condition, next review, provenance, and linked Outcomes. | active, waiting, deferred, ready to close, closed, released, reopened, transferred. |
| Work Home | Find Project Outcomes and next intervention. | empty, active, Needs You, Action Required, Waiting, Ready for Acceptance. |
| Outcome Define/Clarify | Establish an inspectable contract. | vague success, conflict, revision, defer. |
| Mission Map/Authority | Understand plan, topology, Evidence, budget, placement, and effects. | direct unit, non-trivial graph, invalidated, capability blocked, changed grant. |
| Run | Track responsibility by Work Unit. | queued, running, paused, retry, lost, recovery, partial Evidence. |
| Evidence/Verification | Judge each current criterion. | missing/stale/contradicting Evidence, failed check, verifier conflict, exception. |
| Acceptance/Adaptive Close | Accept, reopen, release, retain resources, or create successor. | accept, rework, revised active, dirty worktree, unresolved Open Loop. |
| Re-entry | Continue with minimum exact context. | Outcome successor, Open Loop reopened, source unavailable, historical provider unavailable. |
| Desktop Context beta | Consent to optional ambient context and correct episodes. | disabled, permission denied, paused, excluded app, storage cap, local/provider disclosure, delete. |
| Settings & Control | Inspect and revoke authority. | provider incompatible, connection revoked, retention/export/delete, attachment unavailable. |

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
| [Dayflow](https://github.com/JerryZLiu/Dayflow/tree/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046) | **Adapt after core stability** optional capture consent, exclusions, local timeline, Daily/Weekly projections, and local-model route. Reject default-on capture, direct DB access, automatic truth, and permanent raw screenshots. |
| Claude Code auto-memory | **Later** project continuity with explicit admission/correction rules. |
| Paxel and AutoResearch-style loops | **Later** consented trace learning, candidate skills, evaluation, promotion, correction, rollback, and deletion. |
| Folk, Poke, Dimension, Manus, Grok Bot, Qodo, Lifestack | **Later/reference only**. |
| [TapeFlow](https://github.com/xingrz/tapeflow) / [Tapflow](https://www.tapflow.ai/) products | **Reject** as unrelated to the communication/personal-agent mechanism. |
| Standalone WHOOP/Oura and minimi entries | **Remove from active Kennel benchmark**; preserve as broader Waldo history. |

Open-source code may only be copied after a source-pinned license, dependency, provenance, and security review. No proprietary prompt, engine, scoring model, hidden behavior, or visual language is copied.

## Phased launch boundary

### Launch core: required

- local-first Home, Work, and Settings destinations;
- Responsibility Spaces, Project readiness, Codex-only admission, and historical provider readability;
- guided Outcome contract, optional Mission Map, RunBrief, Work Units, grants, fenced Attempts, recovery;
- Needs You, Action Required, Waiting, Evidence, Verification, explicit Acceptance, Adaptive Close, and Re-entry;
- confirmed Open Loops and explicit Quick Capture;
- Daily Snapshot from trusted Kennel facts, confirmed items, and explicit notes;
- redacted Outcome Trace and dogfood instrumentation.

### Launch beta: optional but coherent

- one Gmail account connection;
- actionable-thread triage, Communication Brief, correctable Commitment Candidates;
- confirm as Open Loop or promote to Outcome;
- user-approved draft creation, Waiting, follow-up, Ready to Close, and Re-entry;
- no auto-send, archive, deletion, labels, or closure.

Public availability of this beta depends on Google OAuth verification and an approved model-disclosure/security boundary. It may remain dogfood-only while the Work core launches.

### Launch+1 beta

- Dayflow-inspired Desktop Context with separate capture, analysis, provider-disclosure, retention, and deletion consent;
- DesktopObservation and ContextEpisode correction;
- optional Snapshot linkage only after confirmation.

### Later Waldo ecosystem

- explicit hosted attachment, backup, cross-device sync, and hosted remote execution;
- durable Memory and governed trace-learning/skill promotion;
- Relationship and broad work-plus-life Open Loops;
- Health First mobile experience and permissioned body-state planning;
- durable proactive agent, Waldo-owned harness, broader providers, teams, marketplace, and commercialization.

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

1. Codex-only new-work admission and historical provider recovery.
2. RunBrief semantic fields, one-write-lease/fence rule, sequential-by-default concurrency, no provider fallback, explicit budget envelope, and evaluator classes.
3. Redacted causal Outcome Trace with private content excluded by default.
4. Objective dogfood measures, thresholds, failure injections, and falsifiers.
5. Consequential-effect ceiling: authorized local work plus separately approved local commits; no autonomous remote effects; Gmail draft only after explicit user intent.

### Deferred and non-blocking for fully local launch

6. Hosted attachment offline/sync acknowledgement, detach/revoke, deletion, and conflict semantics.
7. Durable Memory admission, Relationship, Health, mobile, proactive agent, trace learning, and Waldo-owned harness.

### Implementation entry rules

- PR #1/F0-F6 must first be accepted on its existing foundation boundary from a complete green local gate.
- PR #11 must not merge wholesale. Rebase/re-extract issue-sized cleanup after foundation; remove its speculative Outcome migration/store until a vertical slice owns domain, service, storage, CDC, API, UI, recovery, and evaluation together.
- Product work occurs only on post-foundation feature worktrees, never on the F0-F6 branch.
- Build in vertical slices: truthful local facts and projections first, then the Outcome loop, then communication beta, then optional Desktop Context.
- No merge, push, deploy, publish, release, destructive cleanup, or hosted attachment is authorized by this document.

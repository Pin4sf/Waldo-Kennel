# Kennel v1 product architecture

- Status: Accepted design baseline; not implemented
- Decision date: 2026-08-20
- Launch wedge: Kennel-first for agent-heavy Mac users
- Launch proof: Outcome to verified acceptance with lower coordination and supervision cost

This document records the product, ontology, state, user-flow, and work-surface decisions accepted during the 18-20 August 2026 architecture session. It is the design gate for post-foundation work. It does not claim that the current prototype or PR #11 implements these decisions.

## Evidence boundaries

### Observed

- F0-F6 provides an AO-derived Go daemon, SQLite and trigger-based CDC, Electron supervisor, worktrees, provider sessions, terminal/chat/browser surfaces, recovery, and PR/check/review observation.
- The current Outcome overlay is a prototype based substantially on provider-session coordination; it is not the accepted product model.
- PR #11 removes substantial inherited surface area but also adds an unwired permanent Outcome schema and store that do not preserve the accepted contract, verification, acceptance, CDC, or recovery semantics.

### Reported

- GitHub Actions are currently disabled for the account, so PR acceptance may require the repository's complete local verification matrix until hosted checks return.
- Users supervising multiple coding agents lose time reconstructing transcripts, deciding whether an agent is actually done, and repairing interrupted or incorrectly routed work.

### Inference

- The first product must improve responsibility transfer and judgment compression, not maximize agent count or expose a larger provider dashboard.
- A local-first Waldo Core reduces launch dependencies while retaining a clean future hosted boundary.

### Unknown

- exact provider launch set and provider-by-provider unattended/subscription compatibility;
- routing thresholds, fallback policy, concurrency ceilings, and evaluator independence rules;
- offline attachment acknowledgement, conflict, detach/revoke, and deletion semantics;
- the objective dogfood thresholds required to call the wedge proved;
- whether and when a hosted account provides enough concrete value to recommend it.

## Product thesis and non-goals

Kennel helps a user delegate a responsibility as an Outcome, understand and authorize the proposed work, let local provider agents execute bounded Work Units, receive only irreducible interventions, inspect criterion-bound evidence and verification, and consciously accept or reopen the result.

Launch is not:

- a generic multi-agent dashboard;
- a provider launcher or lowest-common-denominator chat wrapper;
- prompt rewriting presented as orchestration;
- a transcript archive, activity score, or automatic claim that more agents mean more useful output;
- the Memory, Relationship, or broader life-plus-work Waldo product;
- a hosted synchronization product or required account funnel.

Memory is a supporting continuity capability after the core loop proves demand. Relationship and broader life-plus-work Waldo are planned in a separate architecture session.

## Canonical ontology

| Object | Definition and invariant |
| --- | --- |
| `Project` | Local workspace and policy boundary. A future hosted attachment occurs at this boundary. |
| `Outcome` | The responsibility delegated by the user. It owns lineage, not a provider session or a PR. |
| `ContractRevision` | Immutable version of goal, success criteria, constraints, review expectation, authority envelope, and stop conditions. |
| `PlanRevision` | Versioned execution proposal. Its understandable projection is the optional **Mission Map**. It is not a second responsibility object. |
| `WorkUnit` | Smallest bounded schedulable unit with dependencies, required capabilities, expected evidence, verification method, stop conditions, and recovery policy. |
| `Attempt` | One execution of a Work Unit. Retries and provider switches create new Attempts rather than rewriting history. |
| `AgentSessionRef` | Reference to the provider-native Codex, Claude, or other admitted session used by an Attempt. |
| `CapabilityGrant` | Explicit, scoped authority to read, write, execute, disclose, spend, or create a consequential effect. |
| `DecisionRequest` | Irreducible judgment requiring the user, including Waldo's recommendation, rationale, consequence, override, and inspect path. |
| `EvidenceItem` | Provenance-bearing artifact or observation tied to one ContractRevision criterion and the subject it evaluates. |
| `VerificationRun` | Method, verifier identity/independence, subject revision, result, and exceptions. Verification does not equal acceptance. |
| `AcceptanceDecision` | User-owned decision to accept or reopen. Accepted history is immutable. |
| `SuccessorLink` | Lineage from an accepted Outcome to a later Outcome created through re-entry or follow-up. |

The following are projections, interactions, or display language rather than independent canonical aggregates:

- clarification;
- Mission Map;
- Needs You, Action Required, Waiting, and Ready for Acceptance;
- current session status;
- Recovery receipt;
- Re-entry packet;
- Project Follow-up / Keep for later.

Provider completion, commits, PRs, checks, artifacts, and process exit are observations or candidate evidence. None can create an `AcceptanceDecision`.

## Product flow and lineage

```text
Project
  -> Outcome
  -> clarification produces ContractRevision
  -> simple direct plan or optional PlanRevision / Mission Map
  -> approved CapabilityGrants
  -> WorkUnits and dependencies
  -> Attempts using provider AgentSessions
  -> EvidenceItems
  -> VerificationRuns
  -> Ready for Acceptance
  -> AcceptanceDecision
  -> Adaptive Close
  -> optional Project Follow-up / Successor Outcome
```

### Contract and change behavior

- Before execution, edits create a new ContractRevision and corresponding plan revision.
- During execution, a material contract change creates a revision and invalidates affected plans, Work Units, evidence, or verification; it never silently mutates authorized work.
- Low-risk reversible execution mechanics may be recompiled automatically within the approved authority envelope.
- Meaningful decomposition, material cost, new permissions, external effects, or changed scope/success criteria require user review.
- After acceptance, historical contract, decisions, evidence, and verification are immutable. Re-entry creates a successor Outcome.

### Orchestration and prompt compilation

Waldo selects the smallest sufficient topology from the Outcome's uncertainty, coupling, risk, capabilities, evidence needs, cost, latency, and reversibility. Simple Outcomes may compile directly to one Work Unit. Non-trivial Outcomes may show a Mission Map and a more detailed compiled execution graph.

Every Work Unit receives a versioned provider-specific `RunBrief` containing:

- parent Outcome and ContractRevision identifiers;
- exact objective, inputs, constraints, and non-goals;
- dependency and workspace snapshot;
- capability/effect envelope and disclosure boundary;
- expected outputs and criterion-bound evidence;
- verification method and evaluator boundary;
- stop/escalation conditions;
- lease/fence, retry, recovery, and handoff rules.

Kennel admits and executes only the locally authorized graph. Provider-native strengths remain available behind a shared capability and event contract; routing must not pretend that all providers have identical behavior.

## State model

Durable facts and explicit decisions are stored; attention labels are derived.

### Outcome responsibility

```text
Draft -> Contracted -> Active -> Ready for Acceptance -> Accepted
                      |                 |
                      |                 -> Reopened -> Active
                      -> Superseded / Released
```

`Accepted` requires an immutable user AcceptanceDecision. `Completed` is not a launch Outcome state because execution completion and user acceptance are different facts.

### Work Unit

```text
Proposed -> Ready -> Admitted -> Running -> Verifying -> Satisfied
                                  |             |
                                  |             -> Rework -> Ready
                                  -> Failed / Superseded
```

Waiting, Action Required, and Needs You are derived from durable dependency, capability, failure, and decision facts.

### Attempt

```text
Queued -> Running -> Paused -> Succeeded
                  |         -> Failed
                  |         -> Cancelled
                  -> Lost -> Reconciled -> narrow retry or human attention
```

Unknown external effects are reconciled and never blindly retried.

## Attention and recovery contract

| Projection | Meaning | Required presentation |
| --- | --- | --- |
| **Needs You** | Waldo cannot responsibly choose between materially different valid paths. | Current problem, recommendation, why, consequence, primary action, override, and inspect path. |
| **Action Required** | One exact human-only action is necessary, such as sign-in, scoped permission, or an external step. | Exact action, location, reason, completion signal, and resume behavior. |
| **Waiting** | A dependency or timed condition is unresolved and no immediate user action is useful. | Dependency, owner/source, recheck condition, and expected release behavior. |

The user must never need to read the full transcript to understand one of these states. Raw sessions remain inspectable.

Recovery is `contain -> reconcile -> narrowly retry`. Non-material recovery stays in history. Material recovery appears as a compact receipt. If recovery exposes human responsibility, it becomes Action Required or Needs You.

## Work surface

Kennel has three primary destinations rather than a large collection of top-level screens.

### 1. Work Home

- Outcome Focus as the entry point;
- Active Outcomes;
- Needs You;
- Action Required;
- Waiting;
- Ready for Acceptance;
- recently accepted Outcomes and Project Follow-up.

Provider sessions are subordinate operational detail rather than the home-level organizing object.

### 2. Outcome Workspace

One progressive surface with these modes:

1. **Define** — Goal, Success, and Review always; Plan and Authority only when warranted.
2. **Plan** — optional Mission Map, dependencies, roles, routing rationale, cost/risk, evidence, and authority preview.
3. **Run** — Work Units, Attempts, concise current state, and Waldo's next-best action.
4. **Review** — evidence and verification grouped by success criterion, including exceptions and failed checks.
5. **Accept / Re-enter** — accept, request rework, revise while active, release, or create a successor Outcome after acceptance.

Terminal, transcript, worktree, browser, provider selection, and low-level recovery live in an inspectable operator drawer. Xirp's simpler local-project/session/rules/skills/worktree interaction is a UX reference, but Outcome responsibility remains primary.

### 3. Settings and Control

- Projects and workspaces;
- providers, models, authentication, readiness, and advanced routing overrides;
- rules, skills, MCP servers, and capability defaults;
- permissions, disclosure, and consequential-effect policy;
- local data, export, revoke, and future explicit hosted attachment.

Infrastructure terminology stays hidden by default but remains inspectable where it affects permission, placement, failure, recovery, or revoke.

## Reference disposition

| Reference | Launch use |
| --- | --- |
| AO-derived Kennel chassis | **Adopt** daemon, sessions, worktrees, terminal, browser, observations, and recovery; do not adopt AO identity or provider-authored product truth. |
| Xirp | **Adapt** simpler projects, sessions, rules, skills, worktree UX, native provider auth, and local custody. |
| Medley | **Adapt** interview, explicit plan contract, approval, bounded workers, review tasks, recovery, and receipts. Reject opaque authority and same-working-tree assumptions. |
| Devin Review, Greptile/TREX, CodeRabbit | **Adapt** evidence and independent review mechanisms. Reject reviewer output as user acceptance. |
| OpenAI Useful-Work Scorecard | **Adapt** objective useful-output and supervision-cost evaluation. |
| ChatGPT Work, Notion Custom Agents, Webhound | **Adapt** concise attention, reusable work context, and governed proactive work where source evidence supports it. |
| Claude Code auto-memory | **Adapt later** project-scoped continuity after explicit admission/correction rules. |
| Paxel and AutoResearch-style loops | **Later** consented trace learning, candidate procedures/skills, evaluation, promotion, versioning, correction, rollback, and deletion. |
| Folk, Poke, Dimension, Manus, Grok Bot, Qodo, Lifestack | **Later/reference only**; they do not define the Kennel Work launch loop. |
| Standalone WHOOP/Oura and minimi entries | **Remove from active Kennel benchmark**; retain only as broader Waldo source history. |

No proprietary prompt, algorithm, engine, scoring model, archetype, or visual language is copied. Open-source mechanisms require source-pinned license and security review before adoption.

## Launch feature boundary

### Required for the first proof

- local Project and provider readiness;
- guided Outcome definition and versioned contract;
- direct plan or optional Mission Map;
- compiled Work Units, RunBriefs, capabilities, and local admission;
- provider Attempts with worktree/recovery custody;
- Needs You, Action Required, and Waiting;
- criterion-bound evidence and verification;
- explicit acceptance, immutable close, and successor re-entry;
- redacted Outcome Trace and dogfood instrumentation.

### Deliberately later

- hosted attachment and cross-device sync;
- Waldo-funded inference or hosted remote execution;
- Memory as a product surface, Relationship, and life-plus-work Open Loops;
- Paxel/AutoResearch learning and automatic skill promotion;
- broad provider catalog, mobile product, teams, marketplace, and commercialization.

## Remaining architecture gates before feature execution

1. Define the exact launch provider capability/admission matrix and historical-session recovery contract.
2. Lock `RunBrief`, lease/fence, routing/fallback, cost/budget, capability/effect, worktree/concurrency, and evaluator-independence contracts.
3. Lock the redacted Outcome Trace and privacy-preserving observability event model.
4. Lock objective dogfood measures, thresholds, failure injections, and falsifiers.
5. Decide the v1 consequential-effect ceiling, including whether Kennel may create only local changes, draft PRs, or other external effects before a distinct approval.
6. Resolve offline hosted-attachment, detach/revoke, and deletion semantics before implementing attachment; these do not block the fully local launch.

No permanent Outcome migration or feature implementation should begin until gates 1-5 are approved. Gate 6 is required only before hosted attachment implementation.

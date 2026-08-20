# Kennel v1 Excalidraw review session seed

- Purpose: collaboratively reconstruct and critique the Kennel v1 experience in Excalidraw
- Source of truth: [team architecture review packet](kennel-v1-team-review-packet.md)
- Clickable reference: [low-fidelity prototype](kennel-v1-review-prototype.html)
- Status: facilitation seed; not a final UI specification
- Session length: 90 minutes

This seed is deliberately tool-neutral Markdown so a founder, designer, or engineer can recreate the board together rather than importing a polished diagram that discourages correction. Keep the board grayscale. Use solid borders for **Locked**, a small eye marker for **Observed**, dashed borders for **Proposed**, and hatched corners for **Unknown**. Put the label on every decision; do not rely on color.

## 1. Board layout

Use a left-to-right canvas with five horizontal lanes:

1. **User responsibility** — what the user believes they delegated and what requires their judgment.
2. **Product surface** — Work Home, Outcome Workspace, attention, review, and close.
3. **Local Waldo Core** — contract, plan, authority, evidence, verification, acceptance, and lineage.
4. **Kennel Runtime** — Codex Attempt, worktree, process, terminal/browser, capability enforcement, and recovery.
5. **Evidence and unknowns** — observed facts, proposed contracts, questions, and falsifiers.

Place a thin vertical divider after each phase: **Ready**, **Define**, **Authorize**, **Run**, **Review**, **Close**, and **Re-enter**.

## 2. Shared legend

Create these reusable sticky styles in the top-left:

| Style | Meaning | Example |
| --- | --- | --- |
| Solid rectangle | Locked decision | “Codex only for new v1 work” |
| Rectangle with eye icon | Observed fact | “Current Outcome overlay depends on provider sessions” |
| Dashed rectangle | Proposed contract | “Authoritative preflight before every Attempt” |
| Hatched corner | Unknown | “May v1 open a pull request?” |
| User silhouette | Human authority | AcceptanceDecision |
| Small document | Immutable revision | ContractRevision 2 |
| Small gear | Runtime fact | Codex process, worktree, check |
| Paperclip | Evidence reference | test result, walkthrough, artifact digest |

## 3. Frame inventory

Create the following numbered frames. Each frame should fit on one laptop screen at normal zoom and contain one obvious primary action.

### F01 — Onboarding and Project selection

**Question:** Can a user start without understanding infrastructure?

Elements:

- Project name and local-folder picker;
- local custody statement;
- Codex readiness block;
- “Codex only” locked label;
- historical Codex: reconcile and readmit or create a new Attempt;
- historical non-Codex: readable, inspect-only, and continued only through a provenance-bearing packet to a new Codex Attempt;
- Action Required variant for missing installation or authentication;
- primary action: **Use this Project**.

Review prompts:

- What must be explained before folder access?
- Does readiness come from the actual execution boundary?
- Is any account, cloud, API-key, model, MCP, or CLI detail unnecessarily exposed?

### F02 — Work Home

**Question:** Can the user find the next useful intervention in five seconds?

Elements:

- Outcome Focus input;
- Active Outcomes;
- Needs You;
- Action Required;
- Waiting;
- Ready for Acceptance;
- recently accepted and Project Follow-up;
- sessions absent from primary hierarchy.

Review prompts:

- Does any card report activity instead of responsibility?
- Are Needs You and Action Required visibly different without color?
- Is there one clear place to state a new Outcome?

### F03 — Outcome Define

**Question:** Is the contract understandable without architecture vocabulary?

Elements:

- Goal;
- Success criteria;
- Review expectation;
- constraints and non-goals;
- stop conditions;
- optional Plan and Authority sections only when warranted;
- primary action: **Clarify material gaps**.

### F04 — Adaptive clarification

**Question:** Is this question irreducible and is its consequence explicit?

Elements:

- one question;
- Waldo recommendation and rationale;
- materially different choices;
- contract/authority/verification impact preview;
- defer and inspect paths.

### F05 — Mission Map

**Question:** Is this the smallest sufficient topology?

Elements:

- direct single-Work-Unit variant;
- sequential three-unit Focus Ledger variant;
- dependencies and current unit;
- expected evidence per unit;
- Codex-only routing;
- topology rationale;
- unknown badges for lease/fence, fallback, budget, concurrency, and evaluator independence.

### F06 — Authority Preview

**Question:** Can the user explain what execution may do and where?

Elements:

- separate read, write, execute, disclose, spend, and external-effect rows;
- Project and worktree placement;
- expiration/revision binding;
- explicit exclusions;
- revoke behavior;
- hatched consequential-effect ceiling;
- primary action: **Approve this revision and grants**.

### F07 — Run

**Question:** Does progress describe the responsibility rather than transcript volume?

Elements:

- Outcome and current Work Unit;
- Attempt identity and Codex session reference;
- evidence progress by criterion;
- Waldo's next action;
- Pause and inspector actions;
- compact recovery receipt;
- lost/reconciled/new-Attempt variant.

### F08 — Needs You

**Question:** Is human judgment genuinely required?

Elements:

- current problem;
- recommendation and why;
- consequence of each path;
- primary choice, override, inspect, and defer;
- revision invalidation note.

### F09 — Action Required

**Question:** Is there exactly one human-only action?

Elements:

- exact action and location;
- reason it cannot be delegated;
- completion signal;
- what Kennel validates next;
- resume behavior;
- failed/retry variant.

### F10 — Waiting

**Question:** Is it clear why action now would not help?

Elements:

- dependency and owner/source;
- current durable fact;
- recheck condition;
- release behavior;
- inspect, revise, and release actions;
- timeout/dependency-failure variant.

### F11 — Evidence and Verification

**Question:** Can the user judge every criterion without reading a transcript?

Elements:

- ContractRevision and subject snapshot;
- criterion rows;
- supporting and contradicting evidence;
- provenance and raw-inspect link;
- VerificationRun method, identity/independence, result, and exceptions;
- missing, stale, failed, and conflict variants.

### F12 — Acceptance

**Question:** Does the surface preserve conscious human closure?

Elements:

- contract summary;
- verification summary and exceptions;
- Accept Outcome;
- Request rework;
- Revise while active;
- Release without accepting;
- explicit statement that no automated event can accept.

### F13 — Adaptive Close

**Question:** What should be consciously released or kept?

Elements:

- immutable close receipt;
- local resource disposition;
- Keep for later;
- follow-up suggestion;
- dirty-worktree and retained-artifact variants.

### F14 — Re-entry and successor

**Question:** Can follow-up begin without mutating accepted history?

Elements:

- predecessor link;
- accepted facts;
- unresolved exception/open loop;
- context inheritance choices;
- new Outcome contract draft;
- primary action: **Define successor**.

### F15 — Operator inspector

**Question:** Are operational details available without becoming product truth?

Elements:

- Attempt and `AgentSessionRef`;
- terminal/chat;
- worktree and file ownership;
- browser;
- capability snapshot;
- redacted trace;
- recovery facts;
- persistent reminder: inspector facts cannot accept an Outcome.

### F16 — Settings and Control

**Question:** Can advanced control be found without burdening the primary flow?

Elements:

- Projects and paths;
- Codex authentication/readiness;
- historical provider identities;
- rules, skills, MCP servers, and advanced routing override;
- permission and disclosure defaults;
- effect ceiling;
- local export and revoke;
- hosted attachment visibly later.

## 4. Primary flow arrows

Draw these solid arrows:

```text
F01 Onboarding
 -> F02 Work Home
 -> F03 Define
 -> F04 Clarify (zero or more)
 -> F05 Mission Map (optional for simple work)
 -> F06 Authority Preview
 -> F07 Run
 -> F11 Evidence & Verification
 -> F12 Acceptance
 -> F13 Adaptive Close
 -> F14 Re-entry (optional successor)
```

Draw these conditional arrows:

- F01 -> F09 when Codex is unavailable or unauthenticated;
- F07 -> F08 for irreducible judgment;
- F07 -> F09 for a human-only action;
- F07 -> F10 for a passive dependency;
- F07 -> F15 whenever the user inspects operational detail;
- F11 -> F07 when verification requires rework;
- F12 -> F07 when the user requests rework;
- F03/F05/F06 -> earlier revision when a material change occurs;
- F07 -> F07 through a new Attempt after contain/reconcile/narrow retry;
- F02/F07/F11 -> F16 for settings or control, then back to the originating context.

Use a dashed arrow for every flow whose exact contract is unapproved.

## 5. Canonical lineage strip

Across the bottom of the board, draw one continuous lineage strip:

```text
Project
 -> Outcome
 -> ContractRevision
 -> PlanRevision
 -> WorkUnit
 -> CapabilityGrant
 -> Attempt
 -> AgentSessionRef
 -> EvidenceItem
 -> VerificationRun
 -> AcceptanceDecision
 -> SuccessorLink -> new Outcome
```

Place these projections above the object that derives them:

- clarification above ContractRevision;
- Mission Map above PlanRevision;
- Needs You above DecisionRequest;
- Action Required and Waiting above durable capability/dependency facts;
- Ready for Acceptance above EvidenceItem + VerificationRun;
- recovery receipt above Attempt/recovery facts;
- re-entry packet above AcceptanceDecision + SuccessorLink.

Add one red note: **No projection is a competing canonical writer.**

## 6. Architecture frame

Create one system frame with four boxes:

```text
Electron UI
    <-> loopback daemon API
        -> Local Waldo Core <-> local SQLite
        -> Kennel Runtime   <-> local SQLite
             -> Codex AgentSessions
             -> Project/worktree/terminal/browser
```

Arrow labels:

- Waldo Core -> Kennel Runtime: authorized graph, frozen grants, RunBrief;
- Kennel Runtime -> Waldo Core: observations, effects, candidate evidence, recovery facts;
- User -> Waldo Core: contract, authority decisions, acceptance;
- Inspector -> User: operational truth, never acceptance.

Put a lock above SQLite: **sole canonical writer in v1**. Put a dashed cloud outside the frame: **hosted attachment later; no dual writer**.

## 7. Failure-injection mini-board

Create eight small cards. Each card must answer “contain, reconcile, resume/escalate.”

1. Codex missing;
2. stale authentication report;
3. capability mismatch;
4. process lost after app restart;
5. TUI/chat transition delivery unknown;
6. dirty or overlapping worktree;
7. verification failure;
8. external effect outcome unknown.

Add a ninth solid card: **historical provider session** — Codex requires exact binding, reconciliation, and fresh admission; non-Codex is inspect-only and hands off explicitly selected recovery context to a new Codex Attempt.

## 8. Ready-to-paste Excalidraw prompts

Use these prompts only to establish rough geometry; the team should correct every generated label against the review packet.

### Prompt A — end-to-end product flow

> Create a grayscale low-fidelity desktop product flow for Kennel. Arrange 16 numbered frames left to right: Onboarding, Work Home, Outcome Define, Clarify, Mission Map, Authority Preview, Run, Needs You, Action Required, Waiting, Evidence and Verification, Acceptance, Adaptive Close, Re-entry, Operator Inspector, Settings and Control. Use solid borders for locked decisions, dashed borders for proposed details, and hatched corners for unknowns. Keep Outcome as the primary hierarchy and provider sessions inside the inspector. Show solid primary-flow arrows and dashed exception/recovery arrows. Do not add color, gradients, mascots, analytics dashboards, or decorative cards.

### Prompt B — ontology and lineage

> Draw one exact left-to-right lineage: Project to Outcome to ContractRevision to optional PlanRevision to WorkUnit to CapabilityGrant to Attempt to AgentSessionRef to EvidenceItem to VerificationRun to AcceptanceDecision to SuccessorLink and a new Outcome. Place clarification, Mission Map, Needs You, Action Required, Waiting, Ready for Acceptance, recovery receipt, and re-entry packet above the facts they project. Mark every projection as derived and add a rule that only the user creates AcceptanceDecision.

### Prompt C — local architecture

> Draw a local-first architecture with Electron as a thin UI over a loopback daemon API. Inside the daemon, separate Local Waldo Core from Kennel Runtime. Both persist through daemon service boundaries to local SQLite, the sole canonical writer. Local Waldo Core owns Outcome, contract, plan, authority, evidence metadata, verification, acceptance, and lineage. Kennel Runtime owns Projects, worktrees, Codex sessions, processes, terminals, browser, raw traces, and recovery facts. Show authorized graph and grants flowing to runtime and observations/candidate evidence flowing back. Place hosted attachment outside as a future dashed box and forbid dual writers.

## 9. Unresolved-decision parking lot

Move **Codex admission and historical recovery** to the resolved strip: per-Attempt fail-closed admission, required/optional capability separation, capability-first compatibility, and asymmetric historical recovery are locked.

Put these four unresolved cards in review order:

2. **RunBrief and orchestration** — schema, leases/fences, routing/fallback, budget, capabilities/effects, worktree concurrency, evaluator independence.
3. **Redacted Outcome Trace** — event fields, correlation, minimization, retention, export, deletion, and debugging access.
4. **Dogfood proof** — tasks, objective measures, supervision accounting, thresholds, failure injection, and falsifiers.
5. **Consequential-effect ceiling** — local edits, commits, draft/open PRs, network, spend, and just-in-time approval.

Hosted attachment offline/sync/detach/revoke/delete goes in a separate **Deferred** area, not the launch-blocking queue.

## 10. Review agenda

### 0-10 minutes — Align on the promise

- Read the product promise and launch wedge aloud.
- Confirm that the proof is lower supervision cost and verified acceptance.
- Remove any frame that primarily celebrates provider activity.

### 10-25 minutes — Walk the happy path

- Traverse F01 -> F14 without opening the inspector.
- Mark any place where the user needs hidden infrastructure knowledge.
- Confirm one primary action and one reading path per frame.

### 25-40 minutes — Authority and truth audit

- Follow every canonical entity on the lineage strip.
- Challenge any projection being persisted as product truth.
- Identify every action that can widen scope, cost, permission, disclosure, or effect.

### 40-55 minutes — Failure and recovery

- Inject three failure cards into F07.
- Verify contain -> reconcile -> narrow retry.
- Confirm that unknown effects never retry blindly and stale Attempts cannot write current truth.

### 55-70 minutes — Evidence and acceptance

- Review one satisfied criterion, one failed check, and one exception.
- Confirm the subject revision and verifier boundary are visible.
- Confirm that only the user can accept or reopen.

### 70-85 minutes — Open gates

- Take the four unresolved decision cards in order.
- Record one explicit decision or one named owner/evidence need per card.
- Do not convert a proposal into Locked merely because it is drawn cleanly.

### 85-90 minutes — Close

- List contradictions and missing states.
- Name the smallest next decision, not the first implementation task.
- Reconfirm that documentation approval does not authorize product implementation.

## 11. Capture template

For each team annotation, write:

```text
Frame:
Label: Locked / Observed / Inference / Proposed / Unknown
Problem or contradiction:
Why it matters to user responsibility:
Suggested decision:
Evidence needed:
Owner:
```

At session end, copy approved decisions into the architecture packet and leave every other card visibly Proposed or Unknown.

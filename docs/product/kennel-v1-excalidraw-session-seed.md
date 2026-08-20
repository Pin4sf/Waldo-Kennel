# Waldo Kennel five-stage Excalidraw session seed

- Status: final convergence board seed; documentation only
- Decision date: 2026-08-20
- Canonical parent: [Waldo Kennel product architecture](kennel-v1-product-architecture.md)
- Interactive companion: [five-stage review prototype](kennel-v1-review-prototype.html)
- First proof: [Local Focus Ledger vertical slice](kennel-v0-first-outcome-slice.md)

Use this seed to recreate the product architecture collaboratively. It intentionally replaces the former 27-frame screen atlas with five adaptive product frames. Home, Work, and Settings remain destinations; they are not separate lifecycles.

## 1. Board grammar

Draw one horizontal spine:

```text
ENTER -> UNDERSTAND -> DECIDE & AUTHORIZE -> ACT & OBSERVE -> PROVE & CLOSE
```

Use four horizontal lanes across all five frames:

1. **Human** — intent, correction, judgment, authority, acceptance, closure.
2. **Waldo** — grounding, recommendation, orchestration, attention, proof assembly.
3. **Kennel** — local custody, canonical mutations, worktrees, provider adapter, recovery.
4. **Evidence boundary** — observed facts, inferred candidates, verified evidence, explicit decisions.

Use grayscale only:

- solid black border: **Locked canonical object or transition**;
- gray fill: **Observed fact or projection**;
- dashed border: **Proposed/later**;
- diagonal hatch: **Unknown**;
- double border: **explicit user decision or consequential-effect approval**;
- red text is prohibited; write failure labels explicitly instead.

Put a persistent destination rail above the spine:

```text
HOME | WORK | SETTINGS & CONTROL
```

Add this note: **Destinations choose responsibility context. The five stages govern progress. Settings and Operator Inspector are overlays.**

## 2. The five adaptive frames

### F1 — Enter

Primary question: **What responsibility are we taking on, and where does it belong?**

Show:

- Work-first v0 first run as the recommended path;
- local Project picker and readiness;
- natural-language Outcome entry;
- alternative “Start with Home” path with no silently created Personal Home;
- Quick Capture and source candidate as Home modes;
- daemon offline, invalid folder, and Codex unavailable states;
- Gmail, Desktop Context, account, hosted attachment, and broad providers as non-blocking later/optional cards.

Required decision:

```text
choose ResponsibilitySpace -> create draft responsibility -> continue to Understand
```

### F2 — Understand

Primary question: **What is true, uncertain, and required for success?**

Work modes:

- Work Home;
- Outcome Define;
- one material clarification at a time;
- current Goal, Success, Review, constraints, stop, and provenance;
- immutable ContractRevision.

Home modes:

- Morning Brief / Today;
- focused Catch Up;
- Daily Snapshot;
- Communication Brief;
- Open Loop detail;
- correct, dismiss, confirm, defer, inspect provenance.

Put the grounding precedence beside the frame:

```text
approved contract/decisions
  -> approved plan/work unit
  -> project policy/grants/effect/budget
  -> verified dependency/workspace facts
  -> approved knowledge
  -> optional candidate context with provenance/freshness
```

Mark transcript text, inferred preferences, source messages, screen observations, and retrieved memory as candidate context. Material contradiction or staleness blocks compilation.

### F3 — Decide & Authorize

Primary question: **What is the recommended approach, and what must the user decide or permit?**

Show:

- Orchestration Advisor recommendation;
- deterministic Orchestration Policy validation;
- direct one-WorkUnit default and expandable Mission Map;
- capability, effect, budget, placement, disclosure, stop, and recovery preview;
- one fenced writer per worktree;
- no silent provider fallback;
- agent freedom inside the approved envelope;
- only material judgment becomes Needs You.

Home -> Work mode:

```text
Suggested Next Action candidate
  -> dismiss | correct | keep/confirm OpenLoop | draft new Outcome | choose existing Outcome
OpenLoop -> explicit ResponsibilityLink -> Outcome
```

Draw `ResponsibilityLink` as an immutable many-to-many lineage edge with source/destination, creator/reason, created time, and optional ended time/reason. Reject duplicate active pairs. Add a bold invariant: **create/end changes neither lifecycle**.

### F4 — Act & Observe

Primary question: **What is Waldo doing, what changed, and where is attention useful?**

Show:

- grounded RunBrief core -> provider adapter compiled form;
- fresh provider admission at Attempt start/resume;
- WorkUnit -> Attempt -> AgentSessionRef;
- activity and current safe next action, not transcript volume;
- Needs You, Action Required, and Waiting as adaptive modes;
- failure containment, lease/fence, reconciliation, replacement Attempt;
- EffectIntent -> I/O -> EffectReceipt for any consequential effect;
- Operator Inspector as a side overlay with session, terminal, worktree, trace, and recovery facts.

Mark process exit, provider completion, commits, checks, drafts, and messages as observations only.

### F5 — Prove & Close

Primary question: **Does current evidence support current success, and should the responsibility close?**

Show:

- criterion -> current EvidenceItem -> VerificationRun;
- verifier class and independence label;
- missing, stale, contradictory, failed, exception, and conflict states;
- Ready for Acceptance for Outcome and Ready to Close for Open Loop as distinct projections;
- explicit AcceptanceDecision or LoopDisposition;
- request rework, revise, release, reopen, retain resource, or create successor;
- immutable history and exact Re-entry packet.

Add a bold invariant: **Verification never creates Acceptance; silence, archive, or inactivity never closes an Open Loop.**

## 3. Canonical lineage strips

### Work

```text
Work ResponsibilitySpace -> Project -> Outcome -> ContractRevision
  -> PlanRevision -> WorkUnit -> Attempt -> AgentSessionRef
  -> EvidenceItem -> VerificationRun -> AcceptanceDecision
  -> optional SuccessorLink
```

### Home

```text
PersonalHome -> candidate projection -> explicit user confirmation -> OpenLoop
  -> owner + recheck + closure condition -> LoopDisposition
  -> optional SuccessorLink
```

### Home to Work

```text
OpenLoop --ResponsibilityLink--> Outcome
Home owner/recheck/closure || Work contract/evidence/verification/acceptance
```

### Effects

```text
approved plan -> scoped CapabilityGrant -> EffectIntent
  -> consequential I/O -> EffectReceipt -> reconcile unknown outcome
```

## 4. First complete Outcome overlay

Place the Focus Ledger example directly underneath the five frames:

| Stage | Focus Ledger proof |
| --- | --- |
| Enter | Select local Project and state the positive whole-minute focus-block Outcome. |
| Understand | Freeze local-day semantics and three observable success criteria. |
| Decide & Authorize | Approve one WorkUnit, one isolated worktree, local checks, no remote effects. |
| Act & Observe | Admit one Codex Attempt, contain/reconcile failure, create replacement Attempt only when required. |
| Prove & Close | Bind current evidence to all three criteria, verify, then record explicit user Acceptance or reopen. |

Mark this as the first implementation milestone. The paired direct-Codex trials are an architecture signal; the 20-Outcome comparison remains the launch evaluation.

## 5. Failure-injection board

Create eight cards, each with trigger -> containment -> reconciliation -> next safe action -> falsifier:

1. daemon restart during an active Attempt;
2. provider session disappears or is inaccessible;
3. late event arrives from a stale fence;
4. worktree is dirty during close;
5. Evidence belongs to an older ContractRevision;
6. Verification contradicts provider completion;
7. authority or required capability changes before resume;
8. duplicate/unknown consequential effect outcome.

The product fails the first-slice gate if it duplicates effects, accepts automatically, rewrites lineage, requires raw transcript reconstruction, or cannot state a safe next action.

## 6. Privacy and retention board

Draw two concentric boundaries:

- **Waldo semantics/control:** responsibility, contract, plan, grants, trace metadata, evidence references, verification, acceptance, closure.
- **Kennel local custody/execution:** SQLite, worktrees, raw local artifacts, provider sessions, terminal/browser, recovery.

Place these locked defaults beside them:

- canonical control/lineage metadata follows the responsibility until user deletion;
- redacted operational diagnostics expire 30 days after Attempt end and may be shortened or disabled;
- raw prompts, files, terminal output, messages, screenshots, secrets, health values, and chain-of-thought are excluded unless explicitly saved with scope and expiry;
- deletion leaves only a content-free anti-resurrection generation marker.

Hosted attachment remains a dashed later boundary with one canonical writer; do not draw dual authority.

## 7. Ready-to-paste Excalidraw prompts

### Prompt A — product surface board

> Create a grayscale desktop-product board for Waldo Kennel. Put Home, Work, and Settings & Control in a top destination rail. Under it draw exactly five adaptive frames left to right: Enter, Understand, Decide & Authorize, Act & Observe, Prove & Close. Use Human, Waldo, Kennel, and Evidence-boundary lanes. Show Work-first onboarding with Home as an available alternative; one Focus Ledger Outcome across all five stages; Needs You, Action Required, Waiting, evidence, acceptance, recovery, and re-entry as modes rather than separate screens. Settings and Operator Inspector are overlays. Never imply provider completion, checks, messages, drafts, or screenshots create acceptance or closure.

### Prompt B — Home to Work lineage

> Draw a calm Home Morning Brief and focused Catch Up inside Understand. A correctable Suggested Next Action can be dismissed, corrected, kept as a Home Open Loop, converted to a draft Work Outcome, or linked to an existing Work Outcome in Decide & Authorize. Show an immutable many-to-many ResponsibilityLink with provenance and optional end metadata. Keep Home owner/recheck/closure separate from Work contract/evidence/verification/acceptance. Linking never changes either lifecycle.

### Prompt C — execution and recovery

> Draw the provider-neutral RunBrief core compiled by a v0 Codex adapter, with fresh per-Attempt admission, intersected authority, one fenced worktree writer, replacement Attempts on retry, no silent provider fallback, effect intents/receipts, and truthful evaluator labels. The agent has tactical freedom inside the envelope. Show failure containment, reconciliation, and the next safe action without making process exit or provider completion authoritative.

## 8. Review agenda

### 0–10 minutes — Spine and first-run

- Are three destinations and five surfaces unambiguous?
- Can Work start without Home, connectors, account, or hosted state?

### 10–30 minutes — First Outcome

- Can Focus Ledger traverse all five stages without hidden horizontal dependencies?
- Is one WorkUnit the smallest sufficient topology?

### 30–45 minutes — Home to Work

- Does the relation preserve provenance and two responsibility lineages?
- Is any candidate silently promoted to responsibility?

### 45–65 minutes — Authority, recovery, and proof

- Does the agent remain free inside a bounded envelope?
- Can every failure end in containment, reconciliation, and a safe next action?
- Can any non-user actor accept or close?

### 65–80 minutes — Privacy and simplification

- Are trace defaults understandable and content-minimized?
- Is every former “screen” either a mode of one surface, an overlay, or removable?

### 80–90 minutes — Record execution decision

Record only:

```text
Decision:
Label: Locked | Observed | Inference | Proposed | Unknown
Owner:
Affected stage:
Domain/API/UI consequence:
Acceptance evidence:
Falsifier:
Rollback/removal rule:
```

The review ends with one of: **authorize the first issue**, **revise the first-slice contract**, or **stop/falsified**. This seed does not authorize implementation, merge, push, deploy, publish, or release.

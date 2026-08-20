# Kennel v0 first complete Outcome slice

- Status: **Locked implementation target; documentation only**
- Date: 2026-08-20
- Canonical parent: [Waldo Kennel product architecture](kennel-v1-product-architecture.md)
- Execution handoff: [first Outcome execution handoff](../superpowers/plans/2026-08-20-first-outcome-execution-handoff.md)
- Implementation status: not shipped and not authorized by this document

## Decision boundary

### Locked

- The first product proof is one repository-backed Work Outcome traversing **Enter → Understand → Decide & Authorize → Act & Observe → Prove & Close**.
- First run supports Work-first entry. Personal Home is available but is not created or required before Work.
- The test Outcome is **Local Focus Ledger** and uses one smallest-sufficient Work Unit unless a failure creates a replacement Attempt.
- v0 dogfood admits Codex only through the provider-neutral adapter contract. This does not choose the v1 provider set.
- Work is local-only: selected Project, isolated worktree, allowed local commands, no network effects, no push/PR/deploy/publish/release, and no automatic commit.
- Provider completion never means Outcome success. Criterion-bound Evidence and Verification precede an explicit owner AcceptanceDecision.
- The slice is incomplete unless restart, stale-attempt, failed-verification, reopen, and dirty-worktree paths preserve truthful state.

### Observed

- The chassis already provides Project registration, worktrees, Codex sessions, local process observation, SQLite, trigger-based CDC, loopback APIs, and Electron routes.
- The inherited `domain.Outcome`, `OutcomeTask`, session-oriented orchestration service, marker-parsed UI, and `Completed` state do not implement this contract and are donor code, not the target model.

### Unknown outside this slice

- v1 provider selection, hosted attachment, Home/Open Loop persistence, Gmail, Desktop Context, durable Memory, Health, Relationship, mobile, and commercialization.

## Outcome fixture

### User statement

> Build a local Focus Ledger that lets me record one focus block, see the total protected minutes for the Mac's current local calendar day, and retain the entry after a normal app restart.

### Frozen ContractRevision 1

| Field | Value |
| --- | --- |
| Goal | A user can record and review today's protected focus time locally. |
| Criterion 1 | Entering a positive whole-minute duration and optional note creates one local focus block. |
| Criterion 2 | Today's total equals the sum of focus blocks whose start belongs to the Mac's current local calendar day. |
| Criterion 3 | A normal application restart preserves the entry and total. |
| Review | Deterministic checks for validation, date boundary, aggregation, and persistence; owner walkthrough for the visible flow. |
| Constraints | Local-only; no account, analytics, cloud sync, remote I/O, background capture, or hidden scoring. |
| Non-goals | Timers, reminders, calendar integration, health interpretation, productivity judgment, multi-device sync, and deployment. |
| Stop | Stop before an unapproved dependency, remote effect, write outside the isolated worktree, or contradictory Project policy. |

The only material clarification is the meaning of “today.” The locked answer is the Mac's local calendar day, resetting at local midnight. A later edit creates ContractRevision 2 and invalidates plans, grants, Evidence, and Verification bound to revision 1.

## Smallest sufficient plan

The Orchestration Advisor recommends one direct Work Unit:

```text
WU-1 Build and prove Local Focus Ledger
  inputs: ContractRevision 1, pinned Project/base/worktree snapshot
  output: working local feature in the isolated worktree
  evidence: named deterministic test results + restart walkthrough reference
  verification: deterministic outside producer session + owner walkthrough
  authority: local read/write/execute only
  recovery: contain -> reconcile -> replacement Attempt when necessary
```

The model may choose implementation tactics and local tools inside the RunBrief. It may use provider-native subagents as one fenced writer. It may not silently introduce parallel worktrees, another provider, remote I/O, commits, or broader authority.

## Five-stage acceptance contract

| Stage/surface | Required behavior | Durable truth | Completion evidence |
| --- | --- | --- | --- |
| **Enter** | User chooses **Start with Work**, selects/registers a local Project, and sees current Codex readiness without creating Home. | Project identity/readiness and selected entry destination. | Valid Project opens Understand; invalid folder, offline daemon, and missing Codex are truthful. |
| **Understand** | User states the Outcome, sees Goal/Success/Review, answers only the material “today” clarification, and creates ContractRevision 1. | ResponsibilitySpace, Outcome, immutable ContractRevision, causal event. | Restart reproduces the exact contract from SQLite without transcript parsing. |
| **Decide & Authorize** | Waldo recommends one direct Work Unit and previews worktree, capabilities, budget, Evidence, Verification, stop, and exclusions. User approves the revision-bound authority. | PlanRevision, WorkUnit, CapabilityGrant, provider-neutral RunBrief core digest. | Altered scope/authority invalidates the compiled brief; unavailable required capability fails closed. |
| **Act & Observe** | Kennel admits Codex, creates the isolated worktree, records Attempt/session identity, and presents next action plus Needs You/Action Required/Waiting only when warranted. | Attempt, AgentSessionRef, fence/lease facts, ordered observations, recovery receipt. | Healthy run needs no transcript; injected loss reconciles before a replacement Attempt; stale events cannot mutate current truth. |
| **Prove & Close** | Evidence is grouped by current criterion; verification declares its actual independence; owner accepts/reopens; close resolves worktree and successor. | EvidenceItem, VerificationRun, AcceptanceDecision, resource disposition, SuccessorLink/Re-entry. | No Acceptance can be created by provider/check; reopen returns to active lineage; restart preserves accepted history and next action. |

## Minimum canonical model

The first milestone persists only:

- `ResponsibilitySpace` of kind `WorkProject`;
- `Outcome` without a `completed` state;
- immutable `ContractRevision`;
- immutable `PlanRevision` and one `WorkUnit`;
- scoped `CapabilityGrant`;
- provider-neutral RunBrief core digest and adapter-compiled digest;
- `Attempt` and `AgentSessionRef` with fence/recovery lineage;
- `EvidenceItem`, `VerificationRun`, and immutable `AcceptanceDecision`;
- resource disposition and optional `SuccessorLink`;
- metadata-first causal events emitted through SQLite-triggered `change_log`.

Do not add PersonalHome, OpenLoop, ResponsibilityLink, SourceConnection, Gmail, DesktopObservation, Memory, hosted IDs, or generic multi-provider routing to this first milestone.

## Evaluation protocol

### A. Contract conformance gate

Run one normal lifecycle and the required failure cases below against a fresh temporary Project. The gate passes only when:

- every canonical transition is written through daemon service boundaries and replayed after restart;
- every renderer mutation uses generated API types and the daemon API;
- every accepted criterion has current Evidence and a Verification result or explicit exception;
- no provider message, process exit, check, or display label creates Acceptance;
- no unauthorized or duplicate effect occurs;
- the owner can identify current stage, state, and next action without opening the transcript.

### B. Failure injection matrix

| Injection | Required result |
| --- | --- |
| Daemon restart after ContractRevision | Exact contract and Understand state replay; no duplicate revision. |
| Required Codex capability missing | Action Required; no Attempt or worktree writer admitted. |
| Provider process disappears mid-Work Unit | Attempt becomes `unconfirmed`; reconcile before replacement; local work retained. |
| Stale Attempt reports success after replacement | Observation remains inspectable but cannot change current Evidence, Verification, or Outcome state. |
| Deterministic verification fails criterion 2 | Outcome remains active; affected Work Unit returns to rework; Acceptance unavailable. |
| Owner reopens after Acceptance | Immutable prior AcceptanceDecision remains; new active revision/lineage is explicit. |
| Worktree is dirty at close | No force deletion; user chooses retain or later cleanup. |
| App restarts after Acceptance | Accepted contract, Evidence summary, exception, resource disposition, and Re-entry replay exactly. |

### C. First-slice dogfood comparison

Before expanding the model, run five paired trials using equivalent small local feature Outcomes:

1. direct Codex with the same user statement and Project;
2. Kennel through the five-stage lifecycle.

Record active supervision minutes, transcript opens, material interventions, time to identify state after interruption, false-ready/reopen, and unauthorized/duplicate effects. Five trials are an implementation signal, not the 20-Outcome launch gate.

| First-slice signal | Required result |
| --- | --- |
| Lifecycle completion | 5/5 Kennel trials reach an explicit owner disposition or an explained falsifier. |
| Evidence coverage | 100% of accepted criteria have current Evidence and Verification/exception. |
| Transcript reconstruction | No more than 1 of 5 Kennel trials requires the full transcript to decide state or acceptance. |
| Recovery | Every injected recoverable failure contains, reconciles, and exposes a safe next action without duplicate execution/effect. |
| Authority safety | Zero unauthorized, silently widened, or blindly retried effects. |
| Re-entry | Median under 60 seconds to name current stage, material state, and next action. |
| Supervision | Report the matched difference honestly; do not apply the 30% launch claim until the 20-Outcome protocol. |

### D. Falsifiers

Stop and revise the architecture before adding Home or connectors if any of these occurs:

- the complete lifecycle still depends on parsing provider prose or rebuilding a transcript;
- the renderer becomes a canonical writer;
- `completed`, process exit, or green checks can bypass owner Acceptance;
- recovery can duplicate a writer or consequential effect;
- a material contract/authority change reuses the same RunBrief/Attempt;
- the five surfaces require separate competing state machines rather than projections over the same durable facts;
- an issue-sized implementation cannot preserve domain, storage, CDC, API, UI, recovery, and evaluation ownership together.

## Completion definition

The first Outcome milestone is complete only after all five stage-aligned PRs are accepted in order, the contract conformance gate passes, the failure matrix is recorded, and five paired dogfood trials are reported. It does not authorize launch, Home, Gmail, Desktop Context, provider expansion, merge, push, deploy, publish, or release.

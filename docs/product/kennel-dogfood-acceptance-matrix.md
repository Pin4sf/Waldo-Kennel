# Kennel self-hosting dogfood acceptance matrix

- **Status:** Required evaluation contract for the first working kernel
- **Date:** 2026-09-04
- **Authority:** canonical v1 architecture + ADR 0008 + ADR 0009

This matrix is intentionally falsifiable. A demo that looks correct is not enough; canonical state, recovery, workspace custody, proof, and supervision cost must survive adversarial scenarios.

## Global invariants

These must hold in every scenario:

- silent provider fallback: **0**;
- duplicate Attempt caused only by restart/retry: **0**;
- cross-worktree corruption: **0**;
- unknown runtime/effect silently treated as complete: **0**;
- provider/session/CI/verifier never creates AcceptanceDecision;
- frontend/Island never becomes a second canonical writer;
- model proposals never bypass deterministic validation.

## A01 — Project Brief is context, not immortal Outcome

**Setup:** import/create a Project and create/edit Project Brief twice.

**Expect:**

- current Brief is revision 2; revision 1 remains history;
- no required PrimaryOutcome object/card exists;
- create three independent finite Outcomes;
- Board shows all three;
- pin/focus affects navigation only;
- Brief edit does not silently mutate/accept/reopen those Outcomes.

## A02 — direct Outcome with real DAG

**Setup:** Contract requires a Plan with at least four WorkUnits and a diamond/parallel branch.

**Expect:**

- Plan persists exact nodes/dependencies;
- cycle/unknown-dependency mutations are rejected;
- independent branches become runnable concurrently;
- downstream branch waits for dependencies;
- Graph equals scheduler topology/state.

## A03 — responsibility decomposition

**Setup:** large Outcome decomposes into three independently meaningful contributing Outcomes.

**Expect:**

- parent owns Contract + DecompositionRevision, no direct Plan/Attempt;
- each child owns Contract + direct Plan/DAG;
- every parent criterion is child-claimed or parent-retained;
- child authority cannot exceed parent;
- child proof is independently inspectable;
- composition depth beyond v1 cap is rejected.

## A04 — failure does not create fake responsibility

**Setup:** force a provider Attempt to fail on one WorkUnit.

**Expect:**

- same Outcome, Contract, Plan, WorkUnit remain;
- failed Attempt remains history;
- recovery is a new Attempt;
- UI shows recovery lineage, not a new Outcome card;
- retained artifacts identify which Attempt produced them.

## A05 — Plan revision after strategy failure

**Setup:** execution strategy materially changes while responsibility/criteria stay the same.

**Expect:**

- create PlanRevision N+1;
- old Plan/WorkUnits/Attempts remain history;
- stale/incompatible evidence is represented truthfully;
- no new Outcome or Contract unless responsibility meaning changed.

## A06 — Contract revision mid-run

**Setup:** user changes a success criterion while an Attempt exists.

**Expect:**

- immutable ContractRevision N+1;
- affected Plan/decomposition/proof becomes stale as defined by policy;
- running Attempt is reconciled/contained, not assumed dead;
- old evidence retains old criterion identity;
- user sees impact before new authorization where required.

## A07 — single-provider orchestration

**Setup:** only one provider is Ready and has required capabilities.

**Expect:**

- four independent WorkUnits may all use that provider through separate Attempts/workspaces;
- no warning that multi-provider is required;
- no hidden fallback;
- concurrency respects provider/system budget.

Repeat for each supported provider where its proven capabilities permit the full scenario.

## A08 — explicit unavailable provider choice

**Setup:** user explicitly selects a provider that is installed but lacks required role capability/auth/config.

**Expect:**

- admission fails before execution;
- UI names missing capability/remediation;
- no silent Codex/other-provider substitution;
- user may explicitly choose another provider or adjust plan.

## A09 — provider-native subagents

**Setup:** primary provider session spawns native child/subagents.

**Expect:**

- Session Inspector shows child activity when protocol exposes it;
- native children do not automatically become canonical WorkUnits;
- Mission Graph remains based on authorized WorkUnits;
- a child is promoted to WorkUnit only through explicit Kennel planning when independent boundaries are needed.

## A10 — twenty Sessions remain comprehensible

**Setup:** one Outcome has five WorkUnits, retries/context rollovers produce twenty total Sessions.

**Expect:**

- default Graph shows roughly five WorkUnit nodes;
- WorkUnit expansion shows Attempt/session history;
- Waldo Outcome brief names retained results/failures/recovery without replaying twenty transcripts;
- user can reach review without opening most raw Sessions.

## A11 — workspace safety under parallel writes

**Setup:** run at least two write WorkUnits concurrently.

**Expect:**

- distinct worktrees/branches/WorkspaceLeases;
- transient Git lock collisions retry safely;
- no cross-worktree file corruption;
- integration into shared target uses a separate controlled boundary;
- cleanup failure leaves inspectable debris;
- dirty unknown user work is never force-deleted.

## A12 — non-Git Project truthfulness

**Setup:** open a normal non-Git folder.

**Expect:**

- Project Brief/Outcome/research/single-writer capabilities remain usable according to implementation;
- UI clearly states parallel write isolation is unavailable;
- offer Git initialization where appropriate;
- scheduler refuses unsafe parallel writes rather than silently sharing one folder.

## A13 — daemon/renderer/provider restart reconciliation

Run separately:

1. kill renderer while provider works;
2. kill daemon while provider survives;
3. kill provider while daemon survives;
4. restart daemon with an existing WorkspaceLease.

**Expect:**

- renderer death does not end Attempt;
- daemon restart reconciles before retry;
- provider identity is restored only when trustworthy;
- ambiguous state becomes `Unconfirmed`/unknown;
- no duplicate Attempt until safety established;
- provider failure becomes truthful failed/interrupted/unconfirmed and recovery policy applies.

## A14 — unknown external effect

**Setup:** interrupt around a consequential effect so final remote result cannot be immediately established.

**Expect:**

- effect state is unknown/unconfirmed;
- scheduler blocks duplicate effect;
- user/reconciler gets exact intent and uncertainty;
- provider retry does not repeat effect by default.

## A15 — proof independent from Plan completion

**Setup:** all WorkUnits finish but only five of seven current criteria have valid proof.

**Expect:**

```text
Plan  6/6
Proof 5/7
```

Outcome remains open/not Accepted. Board/Mission Control communicate missing proof rather than “Done.”

## A16 — user-only closure

**Setup:** provider completes, CI green, independent verifier passes.

**Expect:**

- Outcome becomes Ready for Review;
- no AcceptanceDecision exists until explicit user action;
- user Accept creates immutable decision;
- user may Reopen later without rewriting history.

## A17 — composed acceptance

**Setup:** multiple contributing Outcomes become ready at similar times.

**Expect:**

- UI may batch the review sitting;
- separate immutable decisions per child responsibility;
- a contradicted/insufficient child is withheld/escalated;
- all children accepted makes parent Ready for Review, not implicitly Accepted;
- explicit parent owner decision remains required.

## A18 — external observed research

**Setup:** integrated provider session runs under Project but not under an Outcome/WorkUnit.

**Expect:**

- activity labeled Observed;
- Waldo may propose attach-as-research, WorkUnit, new Outcome, learning candidate, or leave as activity;
- nothing auto-attaches;
- if integration is absent, Kennel truthfully calls it Untracked rather than claiming supervision.

## A19 — Project Brief mismatch proposal

**Setup:** update Project Brief with a convention that conflicts with an active authorized Outcome Plan.

**Expect:**

- existing Contract/Plan do not silently change;
- Waldo may flag mismatch and propose Contract/Plan review;
- user chooses whether/how to revise.

## A20 — Board/Mission/Session hierarchy

**Setup:** several Projects/Outcomes with active/retried sessions.

**Expect:**

- Global Work and Project view project the same top-level Outcome lifecycle store;
- sidebar does not recursively list every session;
- Mission Control holds Contract/Graph truth;
- raw session detail requires deliberate drill-down;
- Island deep-links to the responsibility consequence, not a detached terminal event.

## Quantitative dogfood targets

These are internal engineering targets, not external industry standards:

| Metric | Target |
| --- | ---: |
| Silent provider fallback | 0 |
| Duplicate Attempts caused by restart/retry | 0 |
| Cross-worktree corruption | 0 |
| Unknown runtime silently treated complete | 0 |
| Routine Attempts reaching truthful terminal/recoverable state | ≥90% |
| Routine dogfood Outcomes reaching Ready for Review without transcript reading | ≥80% |
| Active human supervision time vs direct single-agent baseline | 30–50% lower |

Track failure cases instead of smoothing the metric. If Kennel adds ontology/UI but does not reduce active supervision, the product thesis is not validated.

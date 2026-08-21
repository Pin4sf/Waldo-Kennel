# ADR 0005: Governed project learning and skill evolution

- Status: Proposed; awaiting written-specification review
- Date: 2026-08-21
- Scope: Session-derived learning, bounded experiments, skill registry, promotion, and orchestration-policy boundary
- Depends on: [ADR 0003](0003-local-first-waldo-core.md), [ADR 0004](0004-parallel-home-personal-agent-and-required-capture.md), and canonical Outcome/Evidence/Acceptance contracts

## Context

Kennel already observes agent sessions, tools, worktrees, and runtime activity. The accepted architecture adds canonical Outcome, Attempt, Evidence, Verification, Acceptance, Home, Open Loop, capture, and candidate-memory facts. Those sources can reveal repeated project procedures, context omissions, orchestration failures, and useful interventions.

Session analysis alone cannot prove a reusable skill. Agent self-reports are optimistic, historical replays are causally weak, project attribution can be ambiguous, and any optimizer will exploit gaps in a visible or editable evaluator. Automatic trace-to-skill promotion would turn private activity and model interpretation into executable behavior without adequate authority.

Paxel supplies a useful clean-room reference for attributable cross-session episode analysis. AutoResearch supplies a useful open-source reference for bounded empirical search over one declared editable surface with a fixed evaluator and durable result log.

## Decision

Waldo will use a three-part learning architecture:

1. **Observer:** consented, minimized Paxel-style project/session normalization produces provenance-bearing `LearningEpisode` projections and `LearningCandidate`s.
2. **Experimenter:** AutoResearch-style `ExperimentCampaign`s compare bounded candidate revisions against a baseline, held-in tasks, held-out tasks, and adversarial pressure under a locked evaluator and runtime budget.
3. **Governor:** the Kennel daemon and user own candidate admission, scope, activation, correction, rollback, revocation, deletion, and audit. No model or experiment promotes itself.

The first active learning proof is one project-scoped procedural skill. Context rules and orchestration-policy candidates follow only after that proof. Harness-code and optimizer self-modification remain later architecture gates.

### Truth and authority

- SQLite in the daemon is the sole canonical writer for grants, candidates, campaigns, evaluations, decisions, revisions, bindings, invocations, and deletion generations.
- Agent sessions, raw transcripts, commits, PRs, tool calls, activity, and model summaries are sources or observations, not skill truth.
- Outcome Acceptance is the strongest result label but does not prove which procedure caused success.
- Skills advise or structure execution. Effective authority remains the intersection of the approved contract, plan, grants, adapter capabilities, worktree ownership, budgets, and consequential-effect ceiling.
- Evaluator source, permission enforcement, budgets, instrumentation, and promotion code remain outside every editable surface.

### Phase boundary

Learning contracts, synthetic fixtures, shadow `LearningEpisode`s, candidates, and evaluation tooling may proceed in a separate lane in parallel with Work and Home after shared source/reference contracts are ratified.

No learned skill may become active until:

- the relevant Outcome/Evidence/Verification/Acceptance signals are implemented and trustworthy;
- a current no-skill baseline exists;
- held-in and proposer-hidden held-out evaluation show no regression and at least one material improvement;
- tool, privacy, protected-surface, rollback, and deletion contracts pass;
- the user explicitly promotes the exact revision for an exact scope.

### Scope and generalization

The first promotion scope is one Project. Generalization to a task family, ResponsibilitySpace, or user-global scope requires a new evaluation corpus and explicit promotion decision. Team, marketplace, hosted, health-derived, and cross-user learning are out of scope.

## Consequences

### Benefits

- Kennel can compound project knowledge without treating every session as memory or every success as a skill.
- Negative experiments and corrections become reusable counter-evidence.
- Orchestration can improve from verified work while deterministic policy remains authoritative.
- Every active skill remains explainable, scoped, versioned, removable, and independently evaluable.

### Costs

- reliable learning depends on canonical result labels and fresh evaluation, not merely abundant transcripts;
- evaluation consumes time, model usage, worktrees, storage, and reviewer attention;
- candidate and source deletion must propagate through experiment archives and materialized skills;
- model, repository, dependency, or evaluator changes can invalidate prior effectiveness evidence.

## Rejected alternatives

1. **Automatically extract and install skills from successful sessions.** Rejected because success is not causal attribution and execution authority would be silently widened.
2. **Use an opaque project or builder score to route work.** Rejected because it hides uncertainty, task difficulty, model effects, and user correction.
3. **Let candidates edit their evaluator or permission layer.** Rejected because the optimizer would optimize the measurement boundary rather than user value.
4. **Begin with harness self-modification.** Rejected because a project-scoped procedural skill is smaller, safer, and easier to falsify.
5. **Make provider or repository skill folders canonical.** Rejected because they create competing writers and cannot represent admission, evaluation, rollback, and deletion consistently.

## Implementation boundary

The detailed contract, ownership matrix, evaluation gate, and phased subprojects live in [Waldo Learning and Skill Evolution design](../superpowers/specs/2026-08-21-waldo-learning-skill-evolution-design.md).

This ADR does not authorize code, active learned skills, automatic promotion, global policy changes, push, merge, deployment, release, or publication. Its status becomes Accepted only after the written specification is reviewed and approved.

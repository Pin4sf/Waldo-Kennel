# Paxel and AutoResearch learning reference for Waldo Kennel

- **Status:** architecture research; no implementation authority
- **Research date:** 2026-08-21
- **Scope:** consented project-session analysis, bounded experimentation, evaluated skill evolution, and orchestration learning
- **Waldo decision:** adapt the mechanisms through the Kennel daemon; do not copy private Paxel implementation or make AutoResearch-style loops self-authorizing

## Sources and evidence boundary

### Paxel

The source baseline is the clean-room dossier at `waldo-brain/03-References/research/paxel-clean-room-research-dossier-2026-07-23.md`. Its inspected public bootstrap snapshot recorded script SHA-256 `4b6837f45e106973ec6cfd2164e8dac5647bb1b4935de0a0e40914e15a9ac4ae`, resolved container digest `sha256:900858b22f88a71b5c3c1029b8e50d3bab1e175af381bca65c7a11c8ac98e22b`, and declared source revision `bc061503de05b12f65832c540d09c14b183ceaf6`. The mutable public script had already drifted by the dossier's 30 July refresh. Paxel has no public source repository suitable for copying; its private prompts, scoring, aggregation, archetypes, visual language, and algorithms remain out of scope.

First-party sources: [Paxel data handling](https://paxel.ycombinator.com/data-handling), [privacy](https://paxel.ycombinator.com/privacy), and [public bootstrap](https://paxel.ycombinator.com/upload.sh).

### AutoResearch

The official source is [karpathy/autoresearch](https://github.com/karpathy/autoresearch), inspected at [`228791fb499afffb54b46200aca536f79142f117`](https://github.com/karpathy/autoresearch/commit/228791fb499afffb54b46200aca536f79142f117). The repository is MIT licensed.

The observed reference loop keeps preparation and evaluation fixed, permits one declared file to change, runs every experiment under a fixed wall-clock budget, records results on disk, and keeps or discards a candidate based on the measured result. The default program is intentionally autonomous and single-metric; those are reference properties, not Waldo requirements.

## Evidence labels

- **Observed:** present in a pinned public source or the clean-room authorized run record.
- **Reported:** stated by a first-party source but not independently reproduced here.
- **Inference:** conclusion from observed mechanisms.
- **Proposed:** Waldo design choice.
- **Unknown:** requires implementation evidence.

## Paxel mechanism

**Observed:** Paxel discovers local agent-session stores, resolves sessions to repositories, normalizes heterogeneous transcripts, creates per-session narratives and deterministic signals, groups related sessions and commits into multi-session work episodes, links decisions to outcomes, and produces report/profile projections.

**Observed:** project attribution is imperfect. One authorized aggregate run labeled seven repositories as one. Large derived uploads also produced HTTP 502 failures. A Waldo implementation therefore needs confidence-bearing attribution plus resumable, content-addressed, idempotent ingestion rather than trusting a directory name or one synchronous payload.

**Inference:** the valuable unit is not a session. A task can span several sessions; one session can contain several tasks. Project identity, Outcome/Attempt links, issue/PR identifiers, branches, files, time, and parent/subagent lineage are stronger grouping inputs than transcript semantics alone.

**Reject:** builder scores, personality/archetype inference, activity-volume ranking, high-stakes assessment, or any conclusion that provider completion, a commit, or a quiet session proves human success.

## AutoResearch mechanism

**Observed:** the reference sharply separates the editable artifact from fixed preparation/evaluation, establishes a baseline, runs bounded experiments, persists a result ledger, and preserves a simple keep/discard decision.

**Inference:** the transferable mechanism is constrained empirical search. Waldo does not need model-training infrastructure to use it; it needs a declared editable surface, a locked evaluator, reproducible fixtures, a budget, lineage, and a promotion authority outside the loop.

**Reject:** a visible or editable evaluator, one scalar score for broad human work, destructive branch reset as a product lifecycle, indefinite experimentation without an explicit budget, same-model rationale as acceptance, and automatic production promotion.

## Waldo synthesis

```text
consented project/session sources + canonical Outcome facts
  -> attributable LearningEpisode projection
  -> repeated mechanism or failure cluster
  -> LearningCandidate
  -> bounded ExperimentCampaign
  -> baseline + held-in + held-out EvaluationRuns
  -> user PromotionDecision
  -> provisional SkillRevision and scoped SkillBinding
  -> InvocationReceipts + EffectivenessObservations
  -> retain, revise, suspend, roll back, revoke, or delete
```

Paxel-style analysis is the observer and hypothesis generator. AutoResearch-style experimentation is the bounded search and comparison engine. Waldo is the governor: it owns consent, source policy, scope, evaluation independence, activation, rollback, and deletion.

Historical replay may mine candidates and construct fixtures. It cannot prove causal improvement because model versions, repositories, dependencies, and runtime conditions drift. Promotion evidence requires fresh paired execution or a deterministic task simulator that represents the intended scope.

## Adopt / Adapt / Reject

### Adopt

- source adapters plus canonical normalization;
- explicit project attribution with confidence and an unattributed queue;
- evidence-linked cross-session episodes;
- disk-durable experiment lineage and negative results;
- one declared editable surface per campaign;
- baseline-before-change and fixed evaluation contracts;
- small, inspectable candidate diffs.

### Adapt

- Paxel profile aggregation into user-owned, correctable learning candidates rather than identity claims;
- behavioral signals into mechanism-level observations linked to Outcome truth;
- AutoResearch keep/discard into provisional, active, suspended, revoked, superseded, and deleted lifecycle states;
- one metric into a Pareto comparison over quality, supervision, cost, latency, safety, and context overhead;
- direct file mutation into isolated candidate artifacts compiled by the daemon;
- autonomous continuation into explicit campaign budgets and stop conditions.

### Reject

- automatic skill or policy promotion from traces;
- raw transcript or health-data ingestion as a default;
- hidden scoring or opaque learned routing;
- cross-project generalization without evaluation and user approval;
- provider skill stores, repository skill directories, or UI caches as canonical truth;
- skills granting authority, widening tool permissions, or changing Acceptance;
- candidate access to evaluator implementation, runtime ACLs, budgets, audit controls, or promotion code.

## Unknowns for implementation evidence

- the minimum independent episode count that produces useful candidates without flooding review;
- which deterministic signals best identify addressable procedure failures;
- how many task families can support a meaningful held-out split during early dogfood;
- the cost and latency of repeated candidate evaluation on a single Mac;
- which provider adapters can bind a daemon-owned skill revision without mutating user/repository skill sources;
- how often model or dependency drift requires re-evaluation;
- whether project-scoped learned skills reduce supervision enough to justify their review burden.

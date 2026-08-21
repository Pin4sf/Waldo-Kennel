# Waldo Learning and Skill Evolution design

- **Status:** Approved conceptual direction; written specification for review
- **Date:** 2026-08-21
- **Scope:** consented project-session analysis, learning candidates, bounded experiments, governed runtime skills, and later orchestration-policy learning
- **Implementation status:** Not shipped; this document does not authorize product code, promotion, merge, push, release, or deployment
- **Decision record:** [Proposed ADR 0005](../../adr/0005-governed-project-learning-and-skill-evolution.md)
- **Research:** [Paxel and AutoResearch learning reference](../../research/2026-08-21-paxel-autoresearch-learning-reference.md)

## 1. Positioning

Waldo should improve from work without turning private activity into a score or letting a model rewrite its own rules.

The learning system has three roles:

> Paxel-style analysis observes attributable project episodes and proposes hypotheses. AutoResearch-style campaigns test bounded variants. Waldo governs what may become active.

Learning is not Memory, responsibility, runtime state, or proof. A learned skill may help an Attempt, but it cannot create an Outcome, grant authority, manufacture Evidence, verify work, accept an Outcome, or close an Open Loop.

## 2. Approved conceptual decisions

1. Project/session learning is a separate lane over one daemon-owned canonical model.
2. Analysis uses consented, minimized source material and canonical Outcome facts. Raw transcripts are not ingested by default.
3. A session is a source boundary, not the unit of learning. Related activity is grouped into attributable `LearningEpisode` projections.
4. Every model-derived learning is a candidate. Nothing becomes an active skill, context rule, or orchestration policy automatically.
5. Each experiment changes one declared surface under a fixed budget. Evaluators, instrumentation, permissions, promotion logic, and protected surfaces are locked outside the loop.
6. Promotion requires current baseline, held-in and proposer-hidden held-out evaluation, no regression, material improvement, complete provenance/tool/privacy contracts, and explicit user approval.
7. The first proof is one Project-scoped procedural skill. Broader scope requires fresh evaluation and approval.
8. SQLite in the Kennel daemon is the sole canonical writer. Candidate worktrees, logs, skill files, indexes, and provider materializations are subordinate artifacts or projections.
9. Existing provider-reported chat skills remain external catalog entries. They are not silently imported into the Waldo registry.
10. Orchestration-policy learning follows the project-skill proof. Harness-code and optimizer self-modification remain later gates.

## 3. Build and do-not-build boundary

### Build across the first learning program

- per-Project `LearningGrant` and visible eligible-source controls;
- versioned source normalization and confidence-bearing Project attribution;
- derived `LearningEpisode` projection from canonical facts and permitted trace references;
- mechanism-level pattern and failure clustering;
- `LearningCandidate` routing into skill, context-rule, orchestration-policy, Memory, or Open Loop review paths;
- one project-scoped procedural `SkillCandidate` path;
- baseline, held-in, held-out, and adversarial evaluation fixtures;
- isolated, budgeted `ExperimentCampaign`s and immutable results;
- daemon-owned skill registry, revision, binding, invocation, and effectiveness contracts;
- user promotion, edit, defer, reject, suspend, roll back, revoke, export, and delete;
- Project and Settings/Control review surfaces;
- correction, source deletion, executor-version drift, and re-evaluation behavior.

### Do not build in the first learning program

- personality, intelligence, employability, morality, productivity, or health scoring;
- automatic skill/rule/policy promotion;
- raw health data in learning sources, traces, skills, or eval corpora;
- cross-user, team, marketplace, or public leaderboards;
- global learned routing from one Project;
- autonomous modification of AGENTS files, evaluator source, permissions, budgets, audit controls, release workflows, or protected governance;
- provider-authored skill catalogs as canonical registry truth;
- indefinite research campaigns or destructive branch resets;
- harness-code, optimizer-code, or evaluator self-modification;
- automatic remote effects, push, PR, merge, deploy, publish, release, message send, or payment.

## 4. Source and privacy contract

### `LearningGrant`

Learning requires a durable grant separate from ordinary execution authority. It records:

- Project or ResponsibilitySpace scope;
- eligible source classes and date range;
- whether minimized transcript excerpts may be inspected;
- excluded sessions, people, branches, paths, applications, and sensitivity classes;
- local and provider processing routes and disclosures;
- raw, derived, experiment, and registry retention;
- allowed candidate kinds;
- pause, revoke, export, and deletion behavior;
- current policy generation.

No grant means no session-derived learning. Canonical Work facts may support ordinary product evaluation, but their reuse for candidate mining remains visible and purpose-bound.

### Eligible inputs

Highest-trust inputs are current ContractRevision, PlanRevision, WorkUnit, Attempt, explicit user decisions, criterion Evidence, VerificationRun, AcceptanceDecision, recovery receipts, and actual test/build results.

Lower-trust inputs include provider messages, tool calls, terminal/activity metadata, commits, PRs, review comments, model summaries, and permitted transcript excerpts. They remain source observations.

Chain-of-thought, credentials, secrets, raw health values, unrelated files, and excluded third-party content never enter the learning corpus.

### Attribution

Attribution prefers explicit Outcome, Attempt, WorkUnit, Project, repository remote, worktree, issue/PR, parent-session, and user-selected identifiers. Branch, path, time, file overlap, and semantic similarity may support attribution but cannot silently resolve a material ambiguity.

Low-confidence sources enter an `unattributed` review queue and cannot influence promotion evaluation until corrected.

## 5. Learning data model

| Object | Responsibility and invariant |
| --- | --- |
| `LearningGrant` | Purpose, source, scope, processing, retention, and deletion authority for learning. |
| `LearningSourceRef` | Reference to an eligible canonical fact or permitted private artifact with source generation and attribution confidence. |
| `LearningEpisode` | Rebuildable, versioned projection grouping related intent, attempts, decisions, outcomes, interventions, and gaps. It is not Memory or responsibility. |
| `LearningSignal` | Deterministic or bounded semantic observation such as retry, intervention, missing prerequisite, recovery, test result, or context reopen. |
| `LearningCandidate` | Stable proposal envelope with kind, scope, hypothesis, source spans, counter-evidence, uncertainty, applicability, and review state. |
| `SkillCandidateRevision` | Immutable proposed workflow content, trigger, exclusions, tool requests, protected surfaces, and expected effect. |
| `ContextRuleCandidateRevision` | Proposed itemized RunBrief/context delta; it cannot rewrite the whole context artifact. |
| `OrchestrationPolicyCandidateRevision` | Proposed deterministic-policy change; never an opaque learned score. |
| `ExperimentCampaign` | Baseline, editable surface, budgets, evaluator suite, executor identity, stop rules, and lineage for one hypothesis. |
| `ExperimentVariant` | Immutable candidate artifact and parent/diff lineage. |
| `EvaluationSuiteRevision` | Frozen held-in, held-out, adversarial, and cost/latency contract. The proposer cannot read hidden cases or edit the suite. |
| `EvaluationRun` | One reproducible execution with exact candidate, environment, executor, seed where supported, and raw result references. |
| `EvaluationResult` | Metric vector, failures, exploit detections, confidence, and comparison to the named baseline. |
| `PromotionDecision` | Immutable user decision to edit, reject, defer, trial, activate, suspend, roll back, revoke, or delete an exact revision and scope. |
| `SkillRecord` | Stable Waldo registry identity across immutable SkillRevisions. |
| `SkillRevision` | Content, provenance, trigger, scope, tool manifest, protected surfaces, evaluation evidence, expiry, and source dependencies. |
| `SkillBinding` | Explicit binding of one active/provisional revision to a Project, task family, ResponsibilitySpace, or later user-global scope. |
| `InvocationReceipt` | Attempt, RunBrief, skill revision, trigger, effective authority, provider compilation, result, and degradation for one invocation. |
| `EffectivenessObservation` | Post-promotion result or counter-evidence linked to canonical Outcome facts; it proposes lifecycle changes but cannot apply them. |

### Required separations

- A personal/project fact routes to `MemoryCandidate`, not SkillCandidate.
- An unresolved obligation routes to `CommitmentCandidate` or Open Loop review, not SkillCandidate.
- Runtime recovery remains under Attempt.
- Outcome truth remains under Evidence, Verification, and Acceptance.
- A reusable procedure routes to SkillCandidate.
- A RunBrief selection improvement routes to ContextRuleCandidate.
- A topology/dependency/risk-validator improvement routes to OrchestrationPolicyCandidate.

## 6. Candidate lifecycle

```text
LearningCandidate:
  proposed -> triaged -> approved_for_experiment -> experimenting
           -> evaluated -> promotion_ready
           -> rejected | deferred | withdrawn | expired | invalidated

SkillRevision:
  draft -> provisional -> active
        -> suspended | superseded | revoked | deleted
```

Candidate generation applies an addressability filter. Task difficulty, unavailable provider capability, external outage, or ambiguous ownership is not automatically a skill defect. Clusters use verifier-level cause, causal contribution, and reusable mechanism—not error strings alone.

User correction appends a new candidate revision and preserves counter-evidence. Rejected and failed variants remain searchable within retention because preventing rediscovery is part of the value.

## 7. Experiment architecture

### Campaign contract

Every campaign freezes:

- one hypothesis and one editable surface;
- baseline artifact and version;
- candidate type and permitted diff surface;
- executor provider/model/runtime and dependency snapshot;
- held-in, hidden held-out, adversarial, and regression suite revision;
- quality, supervision, latency, cost/token, privacy, and safety measures;
- repetitions or deterministic seed policy;
- time, Attempt, concurrency, worktree, storage, model-usage, and human-intervention budgets;
- stop, crash, exploit, invalidation, and cleanup rules.

Candidate execution runs in isolated worktrees or an equivalent sandbox. It cannot mutate the baseline, evaluator, source corpus, permissions, audit log, or promotion implementation.

### Evaluation rule

A candidate becomes promotion-ready only when:

1. the baseline and candidate both run against the same current suite/environment class;
2. neither held-in nor held-out regresses on a release-blocking measure;
3. at least one intended measure improves materially beyond expected noise;
4. adversarial misuse, protected-surface, authority, and privacy tests pass;
5. token/context overhead, latency, and cost remain within the declared envelope;
6. detected evaluator gaming is treated as failure;
7. results are bound to raw execution artifacts and the exact submitted revision;
8. a separate evaluator is used where deterministic evaluation is insufficient, with identity/independence labeled truthfully;
9. the user reviews the exact diff and evidence.

The same model may propose and execute when necessary, but it cannot be represented as independent evaluation. A model-written summary never substitutes for raw results.

### Evaluation measures

- criterion Evidence and Verification completeness;
- accepted/reopened Outcome rate;
- supervision minutes and material user interventions;
- retries, recovery failures, duplicate effects, and invalid transitions;
- transcript opens and time to correct re-entry;
- tool or command failure rate;
- false-positive guidance and unnecessary steps;
- token/context overhead, elapsed time, storage, and disclosed cost;
- unauthorized effects, protected-file edits, privacy leakage, and evaluator exploits.

No universal builder or productivity score is stored or displayed. Results remain a metric vector with scope and uncertainty.

## 8. Registry and runtime activation

### Registry fields

Every SkillRevision records at minimum:

- stable id, display name, version, trigger-oriented description, and invocation mode;
- provenance (`user`, `waldo_observed`, `external_adapted`, or system), source revisions, source dependencies, and license where applicable;
- exact adoption scope and applicability exclusions;
- required tools/connectors/scopes, operation kinds, data classes, confirmations, idempotency, rollback, timeout/retry, test doubles, and availability probes;
- side-effect level, network access, writes allowed, privacy tier, sanitizer policy, and health-data prohibition;
- protected surfaces and hard rejects;
- evaluation suite, baseline, with-skill results, token overhead, executor/evaluator versions, known failures, and expiry/re-evaluation conditions;
- provisional/active/suspended/revoked/superseded/deleted lifecycle and audit lineage.

### Materialization

SQLite owns registry identity, lifecycle, bindings, source dependencies, evaluation evidence, and content digest. Skill payloads are encrypted or content-addressed subordinate blobs under `~/.kennel`. Provider-specific materializations are rebuildable and generation-fenced.

The daemon compiles active bindings into a versioned RunBrief reference and provider-specific form. It does not silently write user-wide or repository `.claude`, `.agents`, Codex, or other provider skill directories. If an adapter cannot bind the revision without widening authority or creating a competing source of truth, admission fails for that skill and the Attempt continues without it only when the skill was optional.

Existing `/api/v1/sessions/{sessionId}/conversation/skills` remains a live provider catalog. A provider skill becomes a Waldo registry candidate only through explicit import/review with source, license, tool, privacy, and evaluation records.

### Authority

A skill declares desired tools; the runtime decides allowed tools. Effective authority never exceeds the active ContractRevision, PlanRevision, grants, adapter capabilities, worktree ownership, budget, and effect ceiling. A skill cannot authorize itself or interpret invocation as consent.

## 9. Correction, rollback, revocation, and deletion

- A bad result creates counter-evidence and may propose suspension; it cannot silently rewrite the active revision.
- Rollback is a new PromotionDecision that rebinds the last accepted compatible revision or removes the binding.
- Revocation immediately prevents compilation and invocation while permitted audit lineage remains.
- Source deletion advances a generation, invalidates dependent candidates/campaigns, removes derived excerpts and materializations, and blocks stale replay.
- An active revision containing direct or uniquely attributable deleted content is revoked immediately.
- A generalized procedure with no retained source content enters `needs_revalidation`; the user may explicitly retain it as an independently adopted revision after review.
- Executor, provider, dependency, standards-manifest, or evaluator drift can suspend effectiveness claims and require re-evaluation.
- Deletion removes payload, indexes, caches, experiment copies, provider materializations, and dependent summaries; only content-free tombstones and permitted receipts remain.

## 10. User experience

Learning is not a fourth primary destination. It appears where the decision belongs:

- **Project / Work:** “What Waldo noticed” shows correctable episode patterns and project-scoped candidates.
- **Experiment detail:** hypothesis, source episodes, baseline, variants, held-in/held-out results, failures, cost, and recommendation.
- **Promotion review:** exact skill diff, trigger, scope, exclusions, tools, protected surfaces, expiry, rollback, and “Try for this Project.”
- **Settings & Control / Learning:** grants, active/provisional skills, source coverage, provider materialization, evaluation freshness, invocations, suspend/revoke/delete/export.
- **Operator Inspector:** exact SkillRevision and InvocationReceipt used by an Attempt, including degradation or denial.

The system explains why a candidate exists and what evidence would falsify it. It does not shame the user, maximize skill count, or imply that more automation is inherently better.

## 11. Lane and file/API ownership

| Area | Work owns | Home/Personal Agent owns | Learning owns | Shared integration rule |
| --- | --- | --- | --- | --- |
| Domain | Outcome through Acceptance, Attempt, AgentSessionRef, Work trace facts | Home/OpenLoop, capture, Observation/ContextEpisode, memory candidates and admitted memory | LearningGrant, LearningEpisode projection, candidates, campaigns, registry, bindings, invocations | ResponsibilitySpace, RunBrief refs, deletion generations, source refs require coordinated review |
| Services | execution, recovery, evidence, verification, acceptance | Home, capture, candidate-memory admission/retrieval | learning ingestion, candidate mining, experiments, skill registry | Context compilation and provider admission are implemented once |
| Storage | Work lineage queries/migrations | Home/source/memory queries/migrations | learning, experiment, registry queries/migrations | Additive migrations allocated by integration owner; SQLite remains sole writer |
| API | Outcome/WorkUnit/Attempt/Verification operations | Home/OpenLoop/capture/memory operations | learning/candidate/experiment/registry operations | DTO/spec/generated types and CDC land together through one owner |
| Frontend | Work destination | Home and capture/memory review | Project learning, experiment, skill registry controls | Shared shell, provenance, authority, and inspector components have one owner per PR |

### Concrete reservations

Learning reserves:

- `backend/internal/domain/{learning,learning_episode,learning_candidate,experiment,skill_registry}*.go`;
- `backend/internal/service/{learning,experiments,skillregistry}/**`;
- SQLite query files `learning.sql`, `learning_candidates.sql`, `experiments.sql`, and `skill_registry.sql`;
- `/api/v1/learning-grants/**`, `/api/v1/learning-episodes/**`, `/api/v1/learning-candidates/**`, `/api/v1/experiment-campaigns/**`, `/api/v1/skill-registry/**`, and `/api/v1/skill-bindings/**`;
- `frontend/src/renderer/components/{learning,skill-registry}/**` and future Project/Settings learning route modules.

Shared integration owns:

- RunBrief skill/context references;
- provider adapter skill-binding capability and existing provider skill catalog;
- `backend/internal/httpd/controllers/dto.go`, spec registry, route registration, CDC schemas, OpenAPI, and generated frontend types;
- shared Project/ResponsibilitySpace/deletion-generation references;
- exact SQLite migration-number allocation.

Learning must not repurpose `backend/internal/service/chat/skills.go`; it remains the live external-provider skill catalog.

## 12. Failure behavior

| Failure | Required behavior |
| --- | --- |
| Project attribution ambiguous | Keep source unattributed; block promotion use until corrected. |
| Source denied, revoked, or deleted | Stop new processing, advance generation, invalidate dependent work, scrub derivatives. |
| Episode grouping wrong | User can split, merge, exclude, and regenerate without changing responsibility truth. |
| Candidate is duplicate | Link/merge evidence into the existing candidate; do not create skill clutter. |
| Experiment crashes or times out | Record failure against the exact variant; clean isolated resources; preserve lineage. |
| Evaluator unavailable | Report unknown and block promotion; never treat missing evaluation as pass. |
| Candidate tries protected changes | Fail the variant and record an exploit/safety result. |
| Held-in improves, held-out regresses | Reject promotion even if aggregate score rises. |
| Provider cannot bind skill | Show typed degradation; do not mutate provider/user skill stores as fallback. |
| Active skill harms real work | Preserve Outcome truth, propose suspension, allow immediate user rollback. |
| Model/provider version changes | Mark evidence stale for affected scope and require re-evaluation. |
| Deletion/replay races | Canonical generation rejects stale persistence, materialization, and invocation. |

## 13. Implementation subprojects and gates

The architecture intentionally decomposes into separate plans.

### L1 — Experience Ledger and candidate mining

- synthetic fixtures and source-normalization contract;
- LearningGrant and attribution;
- read-only LearningEpisode projection over canonical facts and permitted source refs;
- correction/exclusion/deletion generation;
- project-scoped candidate review in shadow mode.

**Gate:** zero responsibility/proof mutation, provenance complete, attribution failures visible, deletion non-resurrection passes.

### L2 — Experiment and evaluation engine

- immutable campaign/variant/suite/run/result contracts;
- isolated candidate worktrees;
- baseline, held-in, hidden held-out, adversarial, cost, and safety evaluation;
- budgets, stop/cleanup, exploit detection, and raw-result binding.

**Gate:** evaluator and permission surfaces demonstrably immutable; no-regression acceptance works under failure injection.

### L3 — Skill registry and provisional activation

- registry metadata, tool manifests, source lineage, revisions, and bindings;
- provider-neutral RunBrief binding plus adapter conformance;
- promotion review, provisional Project activation, invocation receipts, monitoring, rollback, revoke, and delete;
- first evaluated project-scoped procedural skill.

**Gate:** current Outcome result signals, successful held-in/held-out evaluation, explicit promotion, typed provider binding, and immediate rollback.

### L4 — Orchestration-policy learning, later

Only after L3 demonstrates useful skills with acceptable review burden may Waldo evaluate ContextRuleCandidate and OrchestrationPolicyCandidate. Learned changes remain inspectable deterministic revisions. Harness and optimizer self-modification require another ADR.

## 14. GitHub planning projection

After written-spec approval, GitHub should use parallel milestones:

1. **v0 Work — First Verified Outcome**
2. **v0 Home — Personal Agent Foundations**
3. **v0 Learning — First Evaluated Project Skill**
4. **Later Gates — Durable Memory, Mobile, Hosted**

Work issues follow the five accepted lifecycle stages. Home issues follow the seven vertical slices in the Home/Personal Agent design. Learning issues follow L1-L3; L4 remains a later-gate issue rather than an implementation milestone.

Existing issues #2-#10 predate the accepted ontology. After replacement issues exist, completed foundation issues should close with their merge evidence and incompatible donor issues should close as superseded with links—never be silently rewritten as if their original contract shipped.

## 15. Falsifiers and evolution

Pause or revise the learning architecture if:

- candidates are mostly non-addressable task difficulty or attribution noise;
- review burden exceeds the supervision saved;
- held-out gains do not survive current-project dogfood;
- skills routinely become stale after model/dependency changes;
- reliable provider binding requires mutating user/repository skill truth;
- deletion cannot prevent source-derived resurrection;
- promotion pressure encourages gaming, activity maximization, or false Outcome readiness.

At 100× session history, source parsing, episode regeneration, experiment compute, and index rebuilds fail before SQLite transaction throughput. Use bounded retention, incremental checkpoints, content-addressed artifacts, resumable jobs, and project/task-family partitions before distributed infrastructure.

The evolution path is:

```text
shadow LearningEpisodes and candidates
  -> bounded experiments with locked evaluation
  -> one provisional Project skill
  -> measured active Project skills
  -> task-family/context-rule evaluation
  -> deterministic orchestration-policy revisions
  -> separate future gate for harness evolution
```

# Waldo Kennel build program

- **Status:** Architecture and issue planning complete; issue-scoped beta implementation is in progress, while this document remains coordination authority rather than implementation authorization
- **Date:** 2026-08-25
- **Current base:** refresh `origin/beta` before every implementation issue; historical planning bases are not execution targets

## Integration branch policy

- `main` is the tested promotion branch and remains the default/release source.
- `beta` is the shared integration branch. Contributors do not push commits directly to it; each assigned issue uses a focused branch created from the latest `origin/beta` and a PR back to `beta`.
- The integration owner sequences PRs that touch shared routes, DTO/spec generation, migrations, RunBrief contracts, or product vocabulary. A PR refreshes from current `beta` before final verification.
- After the milestone acceptance matrix passes on integrated `beta`, a maintainer opens a separate `beta` -> `main` promotion PR. Promotion does not itself authorize publication, release, or deployment.
- Emergency hotfixes or release-only branches that bypass `beta` require explicit maintainer authorization and a reconciliation PR back into `beta`.

## Canonical inputs

| Lane | Specification | Implementation plan | Gate |
| --- | --- | --- | --- |
| Foundation identity | [AO retirement audit](ao-legacy-retirement-audit.md) | [AO retirement plan](../superpowers/plans/2026-08-21-ao-legacy-retirement.md) | active product vocabulary, donor independence, package identity, provenance preserved |
| Work | [First Outcome slice](kennel-v0-first-outcome-slice.md) | [First Outcome handoff](../superpowers/plans/2026-08-20-first-outcome-execution-handoff.md) | one real Focus Ledger Outcome through explicit Acceptance and Re-entry |
| Work control plane | [Canonical Work flow](../superpowers/specs/2026-08-25-work-control-plane-canonical-flow-design.md) + [screen/interaction spec](../superpowers/specs/2026-08-25-work-experience-screen-interaction-spec.md) | [Work control-plane delivery](../superpowers/plans/2026-08-25-work-control-plane-delivery.md) | Board -> Outcome Mission Control -> Session Inspector, durable Project conversation, adaptive Contract, bounded continuation, and one evaluated adaptive Mission. Its "adaptive multi-Work-Unit Mission" slice is superseded by [ADR 0007](../adr/0007-composed-outcomes.md): `PlanRevision` stays at one direct Work Unit and the contributing-Outcome layer carries the topology |
| Home/Personal Agent | [Home design](../superpowers/specs/2026-08-21-home-personal-agent-memory-design.md) | [Home foundations plan](../superpowers/plans/2026-08-21-home-personal-agent-foundations.md) | useful Home, explicit closure, governed capture, candidate-only memory, deletion non-resurrection |
| Learning L1 | [Learning design](../superpowers/specs/2026-08-21-waldo-learning-skill-evolution-design.md) | [Experience Ledger plan](../superpowers/plans/2026-08-21-learning-experience-ledger.md) | attributable shadow episodes/candidates with no responsibility/proof mutation |
| Learning L2 | same | [Experiment/Evaluation plan](../superpowers/plans/2026-08-21-learning-experiment-evaluation.md) | locked evaluator, hidden held-out no-regression, isolation and cleanup |
| Learning L3 | same | [Skill Registry plan](../superpowers/plans/2026-08-21-learning-skill-registry.md) | one explicitly promoted provisional Project skill with receipts and rollback |
| Composed Outcomes | [ADR 0007](../adr/0007-composed-outcomes.md) | [Composed Outcomes program](../superpowers/plans/2026-08-29-composed-outcomes-program.md) | one Project-level Outcome decomposed into independently governed contributing Outcomes, with criterion-bound roll-up and no rise in supervision cost |
| Later ecosystem | [ADR 0006](../adr/0006-one-durable-waldo-multiple-governed-presences.md) | Not implementation-ready by design | local gates, attachment protocol, health threat model, custody migration, and mobile specification |

## Dependency and parallelism

```text
AO identity replacement ───────────────┐
                                      ├─> all product surfaces use Kennel/Waldo truth
Work A Enter -> B Contract -> C Plan -> D Attempt -> E Proof/Acceptance
       │           │                         │             │
       │           ├─> Home OpenLoop ───────┼─> Home-to-Work link
       │           └─> shared intake ───────┘
       └─> Home fixture shell
Home OpenLoop -> Home flows -> Capture/source plane -> MemoryCandidate gate
Work result facts + Home/source refs -> Learning L1 -> L2 -> L3
Work D/E + shared Intake + durable Project conversation -> Work Mission Control
Work Mission Control + first-slice evaluation -> composed Outcomes (ADR 0007)
Home durable-memory gate + L3 evidence + attachment/health specs -> ADR 0006 ecosystem work
```

AO donor detachment can run beside Work/Home. The destructive donor removal waits for its exact consumer proof and confirmation. Home shell can run immediately; Home persistence waits for shared ResponsibilitySpace. Learning L1 waits for current canonical identifiers and enough trustworthy result facts to label incomplete/verified outcomes honestly. L2 waits for L1; L3 waits for L2 and Work proof semantics.

## Shared ownership

| Shared surface | Integration owner | Consumers |
| --- | --- | --- |
| `ResponsibilitySpace` and IDs | Work Contract PR | Home, Learning |
| Shared `IntakeSession` state machine | Home Task 4, reviewed by Work owner | Home and Work controllers |
| `ResponsibilityLink` | Home Task 4 | Work read projection |
| Outcome/Attempt/Evidence/Verification/Acceptance facts | Work | Home projection, Learning sources |
| `DecompositionRevision`, `ContributionLink`, `ContributionDependency` | Composed Outcomes vertical | Board/Mission Control, attention roll-up, Learning attribution |
| Source/deletion generations | Home capture/memory | Learning and future attachment |
| RunBrief skill/context references | Work integration owner | Home retrieval, Learning L3, provider adapters |
| Project Waldo conversation and continuation receipts | new Conversation Runtime vertical | Work intake, Mission Control, later Home relationship surface |
| Project default-agent preference and role resolution | new Project role-resolution vertical | Intake analyzer, Mission planner, Session Inspector |
| Work adaptive intake composition | shared Intake owner #32 plus Work consumer | Outcome Contract and Mission Control |
| SQLite migration allocation | named integration owner per merge window | every lane |
| DTO/spec/routes/OpenAPI/generated TS | named integration owner per shared PR | daemon, CLI, desktop |
| Product vocabulary/brand allowlist | AO retirement Task 1 | all lanes |

One PR owns a shared file at a time. Parallel branches do not independently edit DTO registry, route registration, migration numbers, generated API files, or shared RunBrief contracts.

Current route integration convention: `/work` is the Work-first Enter destination, `/home` is the Personal Home Today destination, and the global mode control remembers the last meaningful route within each lane. The inherited `/` orchestration board remains a valid remembered Work route after it has been visited. Generated `routeTree.gen.ts` conflicts are resolved from the route source files and regenerated; neither lane selects its generated side wholesale.

Migrations `0099-0105` are present on the 2026-08-29 `beta` baseline; the next free number is `0106`. Older numeric reservations for Home and Learning are historical coordination evidence, not live claims. Every unmerged vertical must inspect current `origin/beta`, claim the next unused migration in its issue/PR lease before editing schema, and renumber its own unmerged work when another integration lands first.

## Milestones

1. **v0 Foundation — Kennel Identity & AO Retirement**
2. **v0 Work — First Verified Outcome**
3. **v0 Home — Personal Agent Foundations**
4. **v0 Learning — First Evaluated Project Skill**
5. **Later Gates — Durable Memory, Mobile, Hosted**

The first four milestones contain issue-sized implementation work. The fifth contains architecture/evaluation gates only; it is not a promise that durable hosted Waldo, the Health-aware mobile app, or orchestration-policy learning can start before their prerequisites pass.

### GitHub execution map

| Milestone | Canonical issues |
| --- | --- |
| Foundation | [#16 active identity](https://github.com/Pin4sf/Waldo-Kennel/issues/16), [#22 donor consumers](https://github.com/Pin4sf/Waldo-Kennel/issues/22), [#27 bounded donor removal](https://github.com/Pin4sf/Waldo-Kennel/issues/27), [#34 docs](https://github.com/Pin4sf/Waldo-Kennel/issues/34), [#37 compatibility seams](https://github.com/Pin4sf/Waldo-Kennel/issues/37) |
| Work | [#17 Enter](https://github.com/Pin4sf/Waldo-Kennel/issues/17), [#21 contract](https://github.com/Pin4sf/Waldo-Kennel/issues/21), [#26 authority](https://github.com/Pin4sf/Waldo-Kennel/issues/26), [#31 execution](https://github.com/Pin4sf/Waldo-Kennel/issues/31), [#35 proof and acceptance](https://github.com/Pin4sf/Waldo-Kennel/issues/35), [#38 evaluation](https://github.com/Pin4sf/Waldo-Kennel/issues/38), [#40 Home integration](https://github.com/Pin4sf/Waldo-Kennel/issues/40) |
| Home | [#18 shell](https://github.com/Pin4sf/Waldo-Kennel/issues/18), [#23 capture and Open Loops](https://github.com/Pin4sf/Waldo-Kennel/issues/23), [#29 Home projections](https://github.com/Pin4sf/Waldo-Kennel/issues/29), [#32 shared intake](https://github.com/Pin4sf/Waldo-Kennel/issues/32), [#36 governed capture](https://github.com/Pin4sf/Waldo-Kennel/issues/36), [#39 MemoryCandidate review](https://github.com/Pin4sf/Waldo-Kennel/issues/39), [#41 gate](https://github.com/Pin4sf/Waldo-Kennel/issues/41) |
| Learning | [#19 L1 Experience Ledger](https://github.com/Pin4sf/Waldo-Kennel/issues/19), [#25 L2 experiments](https://github.com/Pin4sf/Waldo-Kennel/issues/25), [#30 L3 skill registry](https://github.com/Pin4sf/Waldo-Kennel/issues/30) |
| Later gates | [#20 durable Memory](https://github.com/Pin4sf/Waldo-Kennel/issues/20), [#24 hosted attachment](https://github.com/Pin4sf/Waldo-Kennel/issues/24), [#28 Health-aware mobile](https://github.com/Pin4sf/Waldo-Kennel/issues/28), [#33 L4 policy learning](https://github.com/Pin4sf/Waldo-Kennel/issues/33) |

Foundation, Work, and Home use one issue per implementation-plan task. Learning uses one issue per gated L1/L2/L3 plan, with that plan's four tasks serving as the issue checklist. The later milestone contains gates, not implementation promises. Closed AO-era issues [#2-#10](https://github.com/Pin4sf/Waldo-Kennel/issues?q=is%3Aissue+is%3Aclosed+number%3A2..10) retain their completion or supersession comments rather than being rewritten.

The 2026-08-25 issue audit found the canonical Outcome and learning lineages covered by the existing issues above. The Work control-plane specification requires separately reviewed vertical issues for durable Project conversation, Project agent-role resolution, adaptive Work intake composition, Outcome Mission Control/Session Inspector, and a later evaluated adaptive multi-WorkUnit Mission. That last slice is superseded by [ADR 0007](../adr/0007-composed-outcomes.md), which puts the topology in the contributing-Outcome layer instead; the Composed Outcomes program replaces it. The remaining slices must not be smuggled into #31, #32, or #40 without an explicit issue-scope decision.

## Completion definition

Planning is complete when canonical docs link to these plans, ADRs 0003-0006 are consistent, GitHub issues map to every authorized plan or plan task at the granularity above, old issues are closed with completion/supersession evidence, and no plan represents a prototype or competitor feature as shipped Waldo behavior.

Implementation begins only from an authorized issue in an issue-specific worktree. No document authorizes merge, push, deployment, publication, release, health-data processing, hosted attachment, or destructive donor deletion without its named gate.

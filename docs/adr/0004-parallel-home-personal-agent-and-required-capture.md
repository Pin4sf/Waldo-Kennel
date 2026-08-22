# ADR 0004: Parallel Home and Personal Agent lane with required governed capture

- Status: Accepted
- Date: 2026-08-21
- Scope: Phase sequencing, capture commitment, memory boundary, and Work/Home ownership
- Amends: [ADR 0003](0003-local-first-waldo-core.md) and the sequencing sections of the accepted v0 architecture
- Final presence model: [ADR 0006](0006-one-durable-waldo-multiple-governed-presences.md)

## Context

The accepted 20 August architecture required the Local Focus Ledger Work slice to complete and be evaluated before Home/OpenLoop persistence expanded. Desktop Context was a separately consented launch+1 beta, and durable Memory remained later.

Subsequent primary-source research across Omi, Dayflow, Project Minimi, MIRIX, Hindsight, Mem0, Graphiti, Supermemory, Letta, LangGraph/LangMem, MemOS, A-MEM, and AutoGen established two product requirements:

1. Waldo cannot become a strong work-and-life personal agent from explicit notes and repository events alone. Desktop screen and audio context are required capabilities, while future physical-world parity requires a phone or wearable presence.
2. Home/OpenLoop and candidate-memory learning must shape the shared responsibility, intake, provenance, and context contracts while the Work Outcome spine is being implemented. Waiting to begin all Home persistence until after Work evaluation would delay discovery of cross-lane contract errors.

Parallel implementation increases collision and truth-ownership risk. It must not create duplicate clarification systems, a UI-owned Home store, a second SQLite writer, or automatic durable memory from capture.

## Decision

### Parallel lanes

Work and Home/Personal Agent implementation may proceed in parallel in separate issue-specific worktrees. Under the current integration policy, each issue branch starts from the latest `origin/beta` and opens its PR back to `beta`; tested promotion to `main` remains separate.

The Work lane retains the locked Local Focus Ledger scope and owns:

- Outcome, ContractRevision, PlanRevision, WorkUnit, Attempt, AgentSessionRef;
- execution authority, runtime recovery, Evidence, Verification, and Acceptance;
- Work destination projections and Work-specific evaluation.

The Home/Personal Agent lane owns:

- PersonalHome, OpenLoop, LoopDisposition, Quick Capture, Today/Morning Brief, Catch Up, Open Loop detail, and Ready to Close;
- CaptureGrant, SourceArtifact, Observation, ContextEpisode, CommitmentCandidate, and MemoryCandidate;
- candidate review, correction, expiry, revocation, deletion-generation behavior, and purpose-bound retrieval;
- required desktop screen and audio capture capabilities;
- Home-specific evaluation.

Shared contracts are designed once and require coordinated ownership:

- ResponsibilitySpace and ResponsibilityLink;
- IntakeSession, IntakeTurn, ClarificationRequest, ResponsibilityProposal, PlanProposal, and AuthorityProposal;
- Home-to-Work source/provenance references;
- RunBrief references to admitted memory and candidate context;
- daemon/SQLite ownership, CDC, OpenAPI, generated frontend types, and shared UI components.

### Required capability, governed activation

Desktop visual capture and desktop audio capture are required Personal Agent product capabilities. They are not optional roadmap ideas or a launch+1-only research beta.

Activation remains optional per user and modality. No capture begins without a visible, revocable `CaptureGrant` defining purpose, applications/people/spaces, processing route, disclosure, retention, exclusions, pause, export, and deletion. Home and Work remain usable when capture is denied, paused, unavailable, or deleted.

### Memory boundary

Capture produces sources, observations, episodes, and candidates. It does not produce durable truth, responsibility, authority, Evidence, Verification, Acceptance, or closure automatically.

The Home lane may implement `MemoryCandidate`, admission-review contracts, deletion-generation fencing, and retrieval evaluation in parallel. Durable admitted `MemoryRevision` use remains behind a separate approval gate requiring:

- provenance and bitemporal revision semantics;
- correction and counter-evidence;
- expiry, revocation, deletion, and non-resurrection;
- privacy threat model and cross-space isolation;
- benchmark and Waldo-specific evaluation acceptance.

### One Waldo across devices

A later Health-aware mobile, phone, or wearable presence extends Waldo's source coverage through the same durable identity, CaptureGrant, ResponsibilitySpace, admission, retrieval, and deletion contracts. It does not create a second assistant, a second canonical writer, or an independently evolving personal profile. Health First remains recommended, not required.

## Consequences

### Benefits

- Home, Work, and the shared intake contract are tested against each other before either hardens incorrectly.
- Screen and audio continuity are treated as first-class Personal Agent requirements.
- The team can evaluate candidate precision, privacy, correction, and deletion using real source shapes before enabling durable memory.
- Work remains protected from Home schema creep and keeps its falsifiable Outcome milestone.

### Costs

- parallel lanes require explicit file/API ownership and coordinated shared-contract PRs;
- capture adds privacy, storage, energy, model-cost, bystander, and deletion obligations;
- Home persistence can no longer assume the Work schema will be complete first;
- the product must support useful degraded behavior when required capture capabilities are not activated.

### Rejected alternatives

1. **Keep the strict sequential boundary.** Rejected because it delays learning about the shared responsibility, intake, and context contracts.
2. **Start all Home, capture, and durable Memory persistence at once.** Rejected because it would collapse sources, candidates, and truth before admission/deletion evaluation.
3. **Make capture default-on because the capability is required.** Rejected because product commitment does not override consent, privacy, bystander, or disclosure boundaries.

## Implementation boundary

The detailed contract and ownership matrix live in [Home, Personal Agent, capture, and memory design](../superpowers/specs/2026-08-21-home-personal-agent-memory-design.md).

This ADR authorizes architecture and implementation planning for the parallel Home/Personal Agent lane. It does not by itself authorize a code PR, merge, push, release, deployment, hosted attachment, mobile/wearable implementation, or automatic durable memory admission.

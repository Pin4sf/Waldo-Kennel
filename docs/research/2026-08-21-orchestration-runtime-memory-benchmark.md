# Orchestration/runtime memory benchmark for Waldo Kennel

**Date:** 2026-08-21

**Status:** Architecture research; no implementation authorization

> **Architecture disposition:** ADR 0004 subsequently starts a separately owned Home/Personal Agent lane in parallel with Work. The runtime/checkpoint, authority, and proof boundaries in this report remain unchanged; durable admitted Memory is still separately gated.

**Scope:** Letta Code/MemGPT, LangGraph, LangMem, and AutoGen as references for recoverable agent execution and memory-enabled orchestration

**Out of scope:** selecting a hosted memory vendor, changing the accepted Outcome ontology, implementing persistence, or presenting any reference behavior as shipped Waldo functionality

## Executive conclusion

The systems reviewed do not reveal one database or framework that should become “Waldo memory.” They reveal several different state problems that are often given the same name:

1. **Working context** is the bounded material presently visible to a model.
2. **Runtime/checkpoint state** makes an interrupted attempt recoverable.
3. **Durable knowledge** is governed user, project, or procedural knowledge reusable across attempts.
4. **Canonical responsibility state** records what was agreed, planned, attempted, and accepted.
5. **Proof state** records evidence, verification, and the user's acceptance decision.

Waldo should keep all five separate. The daemon and SQLite remain the sole canonical writer. A provider or orchestration framework may contribute an attempt-scoped checkpoint payload and opaque `AgentSessionRef`, but it must not own Waldo's identity, durable memory, `OpenLoop`, `Outcome`, `EvidenceItem`, `VerificationRun`, or `AcceptanceDecision`.

The strongest combined pattern is:

- **Adopt from LangGraph:** explicit checkpoint boundaries, immutable checkpoint lineage, pending writes, interrupts, deterministic replay boundaries, and the requirement that effects be idempotent.
- **Adapt from Letta Code/MemFS:** inspectable version history, tiered context, raw history separate from compacted summaries, and isolated background consolidation—but make every durable-memory or procedure change a proposal subject to Waldo admission.
- **Adapt from LangMem:** semantic/episodic/procedural distinctions, hot-path versus background formation, profile versus collection representations, and compaction bookkeeping—but do not let a model directly mutate admitted memory.
- **Adopt narrowly from AutoGen:** explicit save/load contracts, bounded model context, and warnings against concurrent use of stateful agents. Do not adopt AutoGen as a new dependency: the project is now in maintenance mode.

The recommended Waldo contract is therefore **a fenced runtime checkpoint envelope plus an effect journal**, not a second general-purpose “memory store.” Durable personal or project knowledge continues through `MemoryCandidate` admission. Retrieval helps compile a purpose-bound, versioned `RunBrief`; it never grants authority and never proves completion.

## Evidence discipline

This document uses these labels:

- **Observed** — directly present in an official repository, code, test, or current official documentation at the pinned snapshot.
- **Reported** — claimed by an official project page or primary paper, but not independently reproduced here.
- **Inference** — a conclusion drawn from observed mechanisms; it is not directly claimed by the source.
- **Proposed** — a Waldo/Kennel architecture decision offered for approval; it is not shipped behavior.
- **Unknown** — not established by the inspected evidence.

Repository behavior is not assumed to establish hosted-service behavior. A paper result is not assumed to be implemented in the inspected current repository. A component version is not treated as a release unless a matching release tag was established.

## Source snapshot and license ledger

| System | Inspected primary snapshot | Version/release evidence | License evidence | Qualification |
|---|---|---|---|---|
| Letta Code / MemFS | [`letta-ai/letta-code@4f2d0d1`](https://github.com/letta-ai/letta-code/tree/4f2d0d13496117e8fbf584aa24bea595c464f11b), committed 2026-08-20 | Root package declares `0.30.28`. No matching `v0.30.28` repository tag was established in the inspected clone. | [`Apache-2.0`](https://github.com/letta-ai/letta-code/blob/4f2d0d13496117e8fbf584aa24bea595c464f11b/LICENSE) | **Observed.** The older [`letta-ai/letta`](https://github.com/letta-ai/letta) repository now points to this codebase and describes its former V1 server as archived; current implementation claims below refer to `letta-code`, not the retired server. |
| MemGPT paper | [arXiv:2310.08560v2](https://arxiv.org/abs/2310.08560v2), revised 2024-02-12 | Paper revision, not a software release | arXiv paper terms | **Reported paper architecture/results only.** |
| Sleep-time compute paper | [arXiv:2504.13171v1](https://arxiv.org/abs/2504.13171v1), submitted 2025-04-17 | Paper revision, not a software release | arXiv paper terms | **Reported experimental results only.** |
| LangGraph | [`langchain-ai/langgraph@f09cfe8`](https://github.com/langchain-ai/langgraph/tree/f09cfe8ffc1eeffd68f4b628ed69c30f7cad229f), committed 2026-08-20 | Monorepo packages declare LangGraph `1.2.11`, checkpoint `4.2.0`, SQLite saver `3.1.1`, and Postgres saver `3.1.2`. A `1.2.11` tag exists, but it does not identify the inspected monorepo head. | [`MIT`](https://github.com/langchain-ai/langgraph/blob/f09cfe8ffc1eeffd68f4b628ed69c30f7cad229f/LICENSE) | **Observed.** Pin the commit and per-package versions together. |
| LangMem | [`langchain-ai/langmem@29cbe41`](https://github.com/langchain-ai/langmem/tree/29cbe41e58528f92e9efa773c12e15c47be3808c), committed 2026-08-10 | Package declares `0.0.30`; no matching release tag was established in the inspected clone. | [`MIT`](https://github.com/langchain-ai/langmem/blob/29cbe41e58528f92e9efa773c12e15c47be3808c/LICENSE) | **Observed.** |
| AutoGen | [`microsoft/autogen@027ecf0`](https://github.com/microsoft/autogen/tree/027ecf0a379bcc1d09956d46d12d44a3ad9cee14), committed 2026-04-06 | `autogen-core`, `autogen-agentchat`, and `autogen-ext` declare `0.7.5`; repository tag `python-v0.7.5` exists at a different commit. | Code: [`MIT`](https://github.com/microsoft/autogen/blob/027ecf0a379bcc1d09956d46d12d44a3ad9cee14/LICENSE-CODE); docs/content: CC BY 4.0 | **Observed.** The current README says the project receives no new features, is community-managed, and recommends Microsoft Agent Framework for new projects. It is a reference, not a dependency recommendation. |

## The boundary Waldo must preserve

### State classes are not interchangeable

| State class | Purpose | Lifetime and owner | May it authorize action? | May it prove or close work? |
|---|---|---|---|---|
| Model working context | Supply the next inference with a bounded view | One model call or short run; runtime-owned projection | No | No |
| Runtime checkpoint | Resume one `Attempt` safely | Attempt-scoped; daemon-owned envelope with provider payload/reference | No; it can only restore already granted authority after revalidation | No |
| Transcript/event log | Reconstruct what occurred | Append-only, policy-retained daemon record and external references | No | It may be a source for candidate evidence, not proof by itself |
| Durable knowledge | Reuse admitted user/project facts, preferences, episodes, or procedures | `MemoryCandidate` → governed admitted revision | No; it can inform a proposal or `RunBrief` | No |
| Canonical responsibility state | Preserve commitments and execution lineage | `ResponsibilitySpace`, `Outcome`, revisions, `WorkUnit`, `Attempt`, `AgentSessionRef` | Only explicit contract/grant revisions define authority | Only through the separate acceptance lineage |
| Proof state | Establish what result exists and whether it meets the contract | `EvidenceItem` → `VerificationRun` → `AcceptanceDecision` | No new authority | Yes, and only the user acceptance decision closes the Outcome |
| Compacted summary | Reduce context cost while retaining trace references | Regenerable projection | No | No |

**Proposed:** “Runtime memory” should mean only the attempt-scoped recovery envelope and its bounded working projection. Calling durable personal knowledge, orchestration checkpoints, and proof one memory layer would obscure authority and make deletion, correction, and recovery unsafe.

### Existing Waldo invariants

This benchmark preserves the accepted lineage:

`Outcome → ContractRevision → PlanRevision → WorkUnit → Attempt → AgentSessionRef`

and the independent proof lineage:

`EvidenceItem → VerificationRun → AcceptanceDecision`

A provider session completing, an orchestrator node returning, a checkpoint being durable, a tool succeeding, a commit existing, or an agent saying “done” cannot accept or close an `Outcome`. Runtime state can point to proof candidates, but cannot contain the canonical acceptance decision.

## Reference 1: Letta Code, MemFS, and the MemGPT lineage

### What is implemented now

**Observed:** The current Letta Code README describes agents that can rewrite memory, skills, prompts, and other harness configuration. Its MemFS stores context in a real git repository. The official [MemFS documentation](https://docs.letta.com/concepts/memfs) says `system/` files are always placed in the system prompt, other files remain out of context, and the directory tree remains visible. Search is file-oriented by default rather than automatically vector-indexed.

**Observed:** MemFS changes are git changes: edits are local until committed and, for configured remotes, pushed. The current memory patch implementation requires a clean repository, applies the patch, and commits an update with an agent author and reason ([`memory-apply-patch.ts`](https://github.com/letta-ai/letta-code/blob/4f2d0d13496117e8fbf584aa24bea595c464f11b/src/tools/impl/memory-apply-patch.ts)). This provides inspectable history and a concrete conflict boundary.

**Observed:** The built-in shared-memory skill describes a separate cloud-hosted git repository mounted by more than one agent. Each mount is walled off, the harness does not automatically push shared changes, and non-fast-forward conflicts require synchronization and rebase ([`managing-shared-memory/SKILL.md`](https://github.com/letta-ai/letta-code/blob/4f2d0d13496117e8fbf584aa24bea595c464f11b/src/skills/builtin/managing-shared-memory/SKILL.md)). MemFS documentation also describes worktrees for isolated concurrent agent edits.

**Observed:** Local compaction produces model-generated full or sliding summaries, requests retention of exact identifiers, truncates tool results, and places explicit word caps on summaries ([`compaction.ts`](https://github.com/letta-ai/letta-code/blob/4f2d0d13496117e8fbf584aa24bea595c464f11b/src/backend/local/compaction.ts)). The inspected tests preserve raw local transcript search independently of the compacted prompt projection. Summary and source trace are therefore distinct artifacts.

**Observed:** Letta Code includes useful local recovery primitives: cross-process file locking with stale-lock recovery ([`file-lock.ts`](https://github.com/letta-ai/letta-code/blob/4f2d0d13496117e8fbf584aa24bea595c464f11b/src/utils/file-lock.ts)), turn leases and recovery logic, bounded queues, and credential-redacting git synchronization. Its message-channel idempotency helper suppresses an in-flight duplicate and an immediately adjacent repeated successful message in the same process; it is not a durable, general external-effect idempotency journal ([`message-channel-idempotency.ts`](https://github.com/letta-ai/letta-code/blob/4f2d0d13496117e8fbf584aa24bea595c464f11b/src/channels/message-channel-idempotency.ts)).

**Observed:** Current [memory documentation](https://docs.letta.com/configuration/memory) describes “dreaming” background subagents that consolidate memory after a step threshold or compaction. An optional configuration lets another background conversation review proposed changes before applying them. This is agent review, not explicit user approval.

### What the papers report

**Reported:** The MemGPT paper frames long-context agents as an operating-system-style hierarchy: limited in-context “main memory,” larger external storage, and tool/interrupt mechanisms that move information between tiers. It reports experiments on long conversations and document analysis. The paper does not establish Waldo-grade admission, user correction, revocation, deletion, or acceptance semantics.

**Reported:** The sleep-time compute paper evaluates offline/pre-query computation and reports lower test-time compute plus accuracy gains on selected stateful reasoning tasks. It supports the value of background preparation, not autonomous promotion of inferred personal facts or procedures to trusted truth.

### Product and engineering assessment

**Inference:** Git provides strong inspection, history, and branching semantics for human-readable memory projections. It does not by itself define which statement is true, whether a user approved it, which source it came from, when it expires, or whether every remote copy was deleted.

**Inference:** “Always in prompt” is an allocation policy, not a trust policy. A wrong or stale inferred persona file placed in `system/` would receive disproportionate influence.

**Inference:** Worktree isolation prevents file-level interference but does not solve semantic conflicts: two background workers can still propose contradictory user facts, preferences, or procedures.

**Unknown:** This review did not establish end-to-end consistency guarantees of Letta's hosted control plane, all provider-side data copies, deletion propagation, or production fault behavior. Local source mechanisms must not be presented as hosted guarantees.

### Adopt / Adapt / Reject

**Adopt**

- Inspectable version history for human-readable context projections.
- A small always-loaded tier plus a larger searchable tier.
- Raw trace retained separately from lossy compacted summaries.
- Isolated branches/workspaces for concurrent background synthesis.
- Turn leases/fences and explicit stale-worker recovery.

**Adapt**

- Treat a MemFS-like tree as an export/projection generated from daemon truth, never as a second canonical writer.
- Let background “dreaming” create `MemoryCandidate` or `ProceduralCandidate` records with source lineage; admission remains a Waldo policy/user decision.
- Replace manual git merge as a product consistency mechanism with daemon transactions, expected revision/CAS, and explicit conflict records.
- Keep only current contracts, grants, policy, and other explicitly approved material in the always-loaded tier. Retrieved personal memory remains source-visible, purpose-bound, and removable.

**Reject**

- Autonomous rewrite of admitted personal memory, trusted prompts, skills, or harness policy by the acting agent.
- Treating background agent review as user approval.
- Making a remote git repository, Letta agent identity, or provider session the owner of Waldo identity or canonical state.
- Using a committed memory file as evidence that its contents are true.

## Reference 2: LangGraph checkpointing and replay

### What is implemented and documented

**Observed:** LangGraph's [persistence documentation](https://docs.langchain.com/oss/python/langgraph/persistence) separates a checkpointer from a store. A checkpointer records graph-state snapshots within a thread and supports fault tolerance, human interrupts, state history, replay, and fork/time-travel. A store is an application-defined cross-thread long-term memory. The distinction is useful: checkpoint persistence is not automatically durable personal memory.

**Observed:** A checkpoint contains channel values, per-channel versions, versions seen by nodes, updated channels, configuration, metadata, and pending writes. `BaseCheckpointSaver` exposes get/list/put/put-writes/delete-thread operations and version allocation ([checkpoint README](https://github.com/langchain-ai/langgraph/blob/f09cfe8ffc1eeffd68f4b628ed69c30f7cad229f/libs/checkpoint/README.md), [`BaseCheckpointSaver`](https://github.com/langchain-ai/langgraph/blob/f09cfe8ffc1eeffd68f4b628ed69c30f7cad229f/libs/checkpoint/langgraph/checkpoint/base/__init__.py)). State updates create a new checkpoint rather than mutating an old checkpoint.

**Observed:** Pending writes persist successful sibling-node outputs when another node in the same superstep fails. On resume, the completed outputs need not be recomputed. This is a precise recovery mechanism, but it covers graph writes, not arbitrary unjournaled external effects.

**Observed:** The [functional API documentation](https://docs.langchain.com/oss/python/langgraph/functional-api) states that resume occurs from a checkpoint boundary rather than the exact instruction pointer. Completed task/subgraph results can be restored, while unfinished tasks can run again. Side effects and non-deterministic operations should be wrapped in tasks, and callers should use idempotency keys or check for an existing result before repeating them.

**Observed:** Replaying after a checkpoint can re-execute later LLM, API, and interrupt operations. Therefore “checkpoint exists” does not mean “every effect after it executes exactly once.”

**Observed:** The SQLite saver uses WAL plus a process/thread lock and describes itself as a lightweight synchronous implementation that does not scale to multiple threads ([SQLite saver](https://github.com/langchain-ai/langgraph/blob/f09cfe8ffc1eeffd68f4b628ed69c30f7cad229f/libs/checkpoint-sqlite/langgraph/checkpoint/sqlite/__init__.py)). It is not a reason to add a second SQLite writer inside Kennel.

**Observed:** The checkpoint README warns that the default serializer can deserialize arbitrary Python types and documents strict MessagePack/allowlist controls. A recoverable envelope is therefore also a code-execution and schema-compatibility boundary, not merely a storage concern.

### Product and engineering assessment

**Inference:** LangGraph supplies the clearest runtime-state model in this benchmark: stable thread/checkpoint identities, immutable history, pending writes, and explicit replay. Its concepts can be implemented without adopting the Python graph runtime.

**Inference:** Framework-level thread identity is narrower than Waldo identity. One `Outcome` can require several `WorkUnit`s, `Attempt`s, providers, and replacement sessions. A LangGraph thread cannot become the user-visible responsibility object.

**Inference:** Pending writes approximate exactly-once graph-node computation, not exactly-once side effects. External messages, purchases, pushes, deployments, issue edits, or destructive file operations require Waldo-owned effect intents and receipts.

**Unknown:** The inspected documentation does not establish that an arbitrary user-supplied graph and every external adapter become safely replayable simply by enabling a checkpointer. Correct task boundaries and idempotency remain application responsibilities.

### Adopt / Adapt / Reject

**Adopt**

- Explicit `thread/run/checkpoint` identities and parent checkpoint lineage.
- Immutable checkpoint history, forks, and explicit interrupts.
- Pending-write persistence so successful independent computations are not needlessly repeated.
- Strict serialization allowlists, schema versions, and migration handling.
- The rule that non-determinism and side effects require explicit replay boundaries and idempotency.

**Adapt**

- Map thread/run/checkpoint semantics into `WorkUnit → Attempt → AgentSessionRef → RuntimeCheckpoint`, with Waldo IDs primary and provider IDs secondary.
- Persist checkpoints through the daemon's single-writer SQLite contract; provider checkpointers are imported/exported payloads or references, not parallel canonical stores.
- Revalidate `ContractRevision`, grants, capability, budget, workspace generation, revocation, and deletion state before resuming a checkpoint.
- Convert pending writes into Waldo effect intents/results with stable keys and reconciliation states.

**Reject**

- Adding a graph runtime and a second checkpoint database before a demonstrated execution need.
- Treating node completion, a checkpoint, or replay success as Evidence, Verification, Acceptance, or Outcome closure.
- Blindly retrying an unfinished node whose external side effect has unknown disposition.
- Deserializing provider checkpoint objects without a restrictive type/schema boundary.

## Reference 3: LangMem long-term formation and compaction

### What is implemented and documented

**Observed:** LangMem's [conceptual guide](https://github.com/langchain-ai/langmem/blob/29cbe41e58528f92e9efa773c12e15c47be3808c/docs/docs/concepts/conceptual_guide.md) organizes long-term memory as semantic facts, episodic experiences, and procedural rules. It distinguishes semantic “profile” records from collections of independent records, and active/hot-path formation from background consolidation. Namespaces may encode organization, user, and application scope.

**Observed:** LangMem's manager asks a model to extract, consolidate, update, and remove semantic, episodic, and procedural memory, including inferred/generalized content and confidence. The store-backed manager retrieves a bounded set, lets the model synthesize updates, then executes store puts/deletes ([`extraction.py`](https://github.com/langchain-ai/langmem/blob/29cbe41e58528f92e9efa773c12e15c47be3808c/src/langmem/knowledge/extraction.py)).

**Observed:** Direct memory tools permit a model to create, update, and delete UUID-addressed records within a namespace and encourage proactive storage of user preferences/context ([`tools.py`](https://github.com/langchain-ai/langmem/blob/29cbe41e58528f92e9efa773c12e15c47be3808c/src/langmem/knowledge/tools.py)). The mechanism does not impose Waldo's provenance, admission, counter-evidence, expiry, revocation, or user-decision fields.

**Observed:** The short-term summarizer records which message IDs are covered, rejects attempts to summarize already-covered messages, keeps tool-call/result pairs coherent, and enforces token budgets ([`summarization.py`](https://github.com/langchain-ai/langmem/blob/29cbe41e58528f92e9efa773c12e15c47be3808c/src/langmem/short_term/summarization.py)). This is materially safer than an untraceable free-text summary, but it remains a lossy projection.

**Reported but unestablished:** A store-manager docstring describes a “versioned history of all changes.” The inspected manager implementation performs key-based store puts and deletes, while the common store item exposes creation/update timestamps. This review did not establish a framework-imposed immutable revision history for every memory change.

### Product and engineering assessment

**Inference:** The cognitive taxonomy helps decide representation and retrieval, but it does not define trust. “The user likes X,” “X happened,” and “when doing X, use procedure Y” need different correction and evaluation rules even if all are searchable embeddings.

**Inference:** A profile is compact but has a large overwrite blast radius; a collection preserves more granular provenance but creates retrieval/deduplication work. Waldo should use small revisioned claims/episodes as canonical records and compile profiles as regenerable projections.

**Inference:** The manager's direct model-driven upsert/delete path is appropriate for experimental application memory, not canonical user truth. Without expected revisions or a daemon admission boundary, concurrent foreground and background consolidators may conflict according to backend behavior.

**Unknown:** This review did not establish independent long-horizon benchmarks for correction safety, concurrent consolidation, deletion propagation, or false-memory admission in the inspected `0.0.30` implementation.

### Adopt / Adapt / Reject

**Adopt**

- Semantic, episodic, and procedural distinctions.
- Profile versus collection as representation alternatives.
- Hot-path capture separated from background consolidation.
- Namespace/scope as a required retrieval dimension.
- Summary coverage bookkeeping and preservation of tool-call/result structure.

**Adapt**

- Every extractor result becomes a `MemoryCandidate` or `ProceduralCandidate`; it does not become durable truth directly.
- Add immutable source references, subject/scope, observed-versus-inferred status, confidence and uncertainty, counter-evidence links, admission decision, correction/supersession, expiry, revocation, deletion generation, and projection version.
- Use expected revision/CAS and stable operation IDs for consolidation. Simultaneous foreground and background jobs may append candidates but cannot race to overwrite admitted state.
- Treat procedural optimization as a versioned change proposal requiring evaluation and policy/user approval before it can alter a trusted prompt or skill.
- Inject retrieved memory through the RunBrief compiler with reason, source, age, and trust visible—not through arbitrary mutation of a model's system context.

**Reject**

- Autonomous create/update/delete of admitted memory by the acting model.
- Inferring a personality profile and giving it higher authority than user statements.
- Silent prompt or procedure optimization in production.
- Treating a generic vector/BaseStore record as canonical truth or as proof of work.
- Claiming immutable version history where the selected backing store and schema do not demonstrably provide it.

## Reference 4: AutoGen as a stateful multi-agent runtime

### What is implemented and documented

**Observed:** AutoGen defines memory as a protocol with `add`, `query`, `update_context`, `clear`, and `close`; storage and retrieval behavior belong to the implementation ([`_base_memory.py`](https://github.com/microsoft/autogen/blob/027ecf0a379bcc1d09956d46d12d44a3ad9cee14/python/packages/autogen-core/src/autogen_core/memory/_base_memory.py)). This is an extension seam, not a governance model.

**Observed:** `ListMemory` is an in-process chronological list. It returns all entries and appends the memory material to model context as a system message ([`_list_memory.py`](https://github.com/microsoft/autogen/blob/027ecf0a379bcc1d09956d46d12d44a3ad9cee14/python/packages/autogen-core/src/autogen_core/memory/_list_memory.py)). It offers neither durable persistence nor selective, provenance-aware retrieval.

**Observed:** `AssistantAgent` invokes each configured memory's context update before model inference. Its saved state contains model-context state, while the memory component is configured separately. Agent runtime state and long-term memory are therefore distinct mechanisms.

**Observed:** Stateful agents and teams expose save/load operations. A group-chat state can include participant states, message thread, current turn, and speaker-selection state ([`_base_group_chat.py`](https://github.com/microsoft/autogen/blob/027ecf0a379bcc1d09956d46d12d44a3ad9cee14/python/packages/autogen-agentchat/src/autogen_agentchat/teams/_group_chat/_base_group_chat.py)). The code warns that saving while a team runs may produce inconsistent state and that cancellation can leave state inconsistent.

**Observed:** Agent-as-tool and team-as-tool wrappers warn that parallel tool calls must be disabled because the wrapped component is stateful. In contrast, an assistant may execute multiple ordinary tool calls concurrently. Arbitrary tool effects therefore still require adapter-level concurrency and idempotency controls.

**Observed:** The single-threaded runtime can stop after the current message and discard queued messages; runtime/component save state does not thereby become a complete durable effect journal. Human-in-the-loop examples explicitly leave checkpoint placement to the application.

**Observed:** AutoGen's current official README marks it maintenance-only and recommends Microsoft Agent Framework for new projects. The reference remains useful, but adopting it now would create migration and maintenance risk.

### Product and engineering assessment

**Inference:** Explicit serializable state is valuable, but “save state” is only safe at a quiescent or application-defined checkpoint boundary. A byte-valid snapshot taken while effects are in flight can still be semantically inconsistent.

**Inference:** A bounded chat buffer is useful context control, not memory governance. Appending all memories into a system message collapses relevance, trust, and authority.

**Unknown:** This review did not establish cross-version compatibility of arbitrary component state or end-to-end exactly-once recovery guarantees for the multi-agent runtime.

### Adopt / Adapt / Reject

**Adopt**

- Explicit save/load contracts for stateful components.
- Bounded short-term model contexts.
- Clear documentation that stateful agents/teams cannot be invoked concurrently without coordination.
- External persistence as an application-owned seam.

**Adapt**

- Save provider/component state as an opaque, schema-versioned payload or external reference under a Waldo checkpoint; validate provider, version, hash, sensitivity, and restoration capability before use.
- Checkpoint only at a quiescent boundary, or separately journal every in-flight effect and queue item.
- Fence one active writer per `Attempt`; parallel `WorkUnit`s receive distinct attempts, sessions, and workspaces.
- Reconstruct from canonical facts and start a replacement attempt when an opaque payload is incompatible or untrusted.

**Reject**

- AutoGen as a new Kennel runtime dependency while it is in maintenance mode.
- `ListMemory`-style injection of every stored item as a system message.
- Saving a live, mutating team and assuming the snapshot is consistent.
- Treating team stop/termination or agent state as Outcome completion.
- Unguarded parallel execution of external-effect tools.

## Cross-system comparison

| Concern | Letta Code / MemFS | LangGraph | LangMem | AutoGen | Waldo decision |
|---|---|---|---|---|---|
| Working state | Filesystem-backed context tiers plus live transcript | Graph channels/state | Model messages plus selected store memories | Model context and agent/team state | Compile a bounded, versioned `RunBrief`; working projection is disposable |
| Checkpoint state | Local transcripts, leases, compaction, git state | First-class immutable checkpoints and pending writes | Not its primary responsibility | Component/team save/load | Daemon-owned `RuntimeCheckpoint` per Attempt; provider payload subordinate |
| Long-term taxonomy | File/tree chosen by agent/user | Cross-thread store is application-defined | Semantic, episodic, procedural | Memory protocol is implementation-defined | Governed candidates and typed admitted revisions; separate projections |
| Consolidation | Agent/background “dreaming,” optionally agent-reviewed | Application graph nodes | Foreground tools or background manager | Application-defined | Background jobs propose only; admission/evaluation is separate |
| Private/shared scope | Agent-private MemFS; separate shared git repo | Thread IDs plus store namespaces | Arbitrary namespaces | Per configured component/team | `ResponsibilitySpace` and user/project scope; no provider-global identity |
| Compaction | Full/sliding summary; raw trace remains searchable locally | State reduction/channels; checkpoints retained by saver policy | Covered-message tracking and token budget | Buffered context keeps recent messages | Summary is regenerable, source-addressed, and never authority/proof |
| Retrieval injection | Always-loaded system files plus agent file search | Application injects store results into graph state | Tools/managers inject selected memories | `update_context` mutates model context | One RunBrief compiler with fixed precedence, provenance, and budget |
| Concurrency | Worktrees, locks, leases; git conflict handling | Supersteps, channel versions, pending writes | Store implementation/caller coordinates | Stateful agents/teams warn against parallel calls | One fenced writer per Attempt; append candidates; CAS projections |
| Idempotency | Narrow duplicate-message suppression; leases | Application must make effects idempotent | Stable keys possible, not imposed as effect journal | Application/tool responsibility | Stable `EffectIntent` key plus durable disposition/receipt |
| Recovery | Local recovery mechanisms and git history | Resume/replay/fork from checkpoint boundary | Re-run manager over store state | Load saved component/team state | Revalidate authority and reconcile effects before resume |
| Failure boundary | File/git state can conflict; hosted behavior unestablished | Incomplete tasks can execute again | Model can make wrong upsert/delete | Live save/cancel can be inconsistent | Unknown effect blocks; stale worker fenced; invalid payload starts replacement attempt |
| Proof/acceptance | Not a native separation | Not a native Outcome acceptance model | Not a native separation | Termination conditions are runtime concepts | Evidence, verification, acceptance remain canonical and separate |

## Proposed Waldo runtime-memory architecture

This section is a design proposal for approval, not an implementation claim.

### 1. Five planes, one canonical writer

```text
Responsibility plane
  ResponsibilitySpace
    └─ Outcome → ContractRevision → PlanRevision → WorkUnit → Attempt
                                                        └─ AgentSessionRef

Execution recovery plane
  RunBrief → RuntimeCheckpoint → EffectIntent → EffectReceipt
                           │                    └─ unknown / succeeded / failed / reconciled
                           └─ parent checkpoint / lease epoch / provider payload ref

Knowledge plane
  SourceObservation → MemoryCandidate → AdmissionDecision → AdmittedMemoryRevision
                                                └─ correction / counter-evidence / expiry /
                                                   revocation / deletion generation

Proof plane
  EvidenceItem → VerificationRun → AcceptanceDecision

Projection plane
  RunBrief / compacted summary / search index / human-readable memory tree
  (all regenerable; none canonical)
```

**Proposed:** The daemon owns transactions across canonical responsibility, recovery, knowledge, and proof records. Search indexes, summaries, provider threads, and git-style context trees are projections or external references. The frontend never owns canonical truth.

### 2. `RuntimeCheckpoint` contract

The minimum useful checkpoint record should include:

- Waldo IDs: `runtime_checkpoint_id`, `attempt_id`, `work_unit_id`, `agent_session_ref_id`.
- Provider identity: adapter kind, provider session/thread/run/checkpoint IDs, opaque payload reference, payload hash, serialization format, and schema/provider version.
- Lineage: parent checkpoint ID, monotonic attempt sequence, creation reason, checkpoint boundary, writer identity, created time.
- Frozen inputs: versioned `RunBrief` ID/hash, `ContractRevision`, `PlanRevision`, workspace generation/ref, policy revision, grant set/revision, capability set, budget reservation/revision, and relevant deletion/revocation generation.
- Runtime cursor: completed steps/tasks, pending steps, interrupt/approval wait, compacted-summary ref, raw trace range, and resumability classification.
- Effect reconciliation: pending `EffectIntent` IDs, completed receipt IDs, unknown-disposition effects, tool-call/result references, and adapter reconciliation cursor.
- Concurrency: lease ID, fence epoch, owner, acquisition/expiry/heartbeat facts, and superseding checkpoint/recovery decision.
- Data governance: sensitivity class, encrypted payload/ref, retention/expiry, redaction state, export/deletion status, and whether source content is reconstructable.

Do not duplicate secrets or large raw tool outputs in every checkpoint. Store governed references and hashes where recovery permits it. Provider payloads must be treated as untrusted, version-sensitive data.

### 3. Checkpoint boundaries

Create a checkpoint or durable journal boundary:

1. after the RunBrief and authority snapshot are frozen;
2. immediately before an external or irreversible effect intent is dispatched;
3. immediately after its durable receipt or reconciliation result is recorded;
4. before and after an approval/clarification interrupt;
5. after a WorkUnit-level atomic computation whose replay is expensive or non-deterministic;
6. before compaction, handoff, provider shutdown, or lease transfer;
7. when the adapter reports a recoverable provider checkpoint.

Saving after every token is unnecessary. Saving only at session end is unsafe. The checkpoint cadence follows effect and authority boundaries, not chat-message count alone.

### 4. Recovery algorithm

**Proposed:** Recovery is a decision procedure, not “load bytes and continue.”

1. Load the canonical `Attempt`, its latest valid checkpoint lineage, and current contract/policy/grant/deletion facts.
2. Acquire the attempt lease and new fence epoch. A stale worker cannot publish events, receipts, evidence candidates, or checkpoints after replacement.
3. Validate checkpoint hash, serializer allowlist, adapter compatibility, provider availability, workspace identity/generation, and source retention.
4. Compare frozen authority against current authority. A revocation, changed contract, expired grant, depleted budget, deleted memory generation, or materially changed workspace blocks silent resume.
5. Reconcile every external effect by stable key and provider receipt. Never retry an effect with unknown disposition merely because the model task was unfinished.
6. Classify the next action as `safe_resume`, `replay_read_only`, `requires_reconciliation`, `requires_user`, `handoff`, or `replacement_attempt`.
7. Compile a fresh RunBrief when the canonical facts materially changed. Preserve the old brief and checkpoint for audit; do not mutate history.
8. On completion, record candidate evidence separately. Verification and user acceptance remain subsequent decisions.

### 5. Effect journal and the exactly-once illusion

No inspected system turns arbitrary external effects into true exactly-once operations. Waldo should assume at-least-once execution and build idempotency/reconciliation explicitly.

**Proposed `EffectIntent`:** stable effect ID/idempotency key, attempt/work-unit identity, adapter/action, normalized target and request hash, authority/grant revision, precondition, risk/destructive class, created/dispatch times, and current disposition.

**Proposed `EffectReceipt`:** provider operation/request ID, observed outcome, response hash/reference, reconciliation source/time, retryability, and any ambiguity. Valid dispositions should include at least `prepared`, `dispatched`, `succeeded`, `failed`, `cancelled_before_dispatch`, `unknown`, and `reconciled`.

An `unknown` message-send, purchase, publish, deployment, issue mutation, or destructive filesystem operation must pause for adapter reconciliation or user decision. An idempotent read may be replayed under policy. A local pure computation may be recomputed if its inputs and tool version are pinned.

### 6. Concurrency model

- One active leased/fenced writer owns an `Attempt` at a time.
- Parallel independent `WorkUnit`s use separate attempts, provider sessions, and workspaces/worktrees.
- Background memory and compaction workers append candidates or create new projection revisions; they never overwrite admitted records in place.
- Projection promotion uses expected revision/CAS and idempotent job keys.
- Cross-agent exchange occurs through canonical artifact/evidence/dependency references, not a shared mutable scratchpad.
- Shared project knowledge is scoped through `ResponsibilitySpace` and explicit access policy. Agent-private scratch is attempt-scoped and expires by policy.
- SQLite transactions and the daemon's single-writer ownership remain the final serialization boundary; framework locks or provider threads cannot replace it.

### 7. Compaction contract

A compacted summary should record:

- source transcript/event range and immutable source IDs;
- covered and intentionally omitted IDs;
- unresolved tool-call/result pairs and effect dispositions;
- contract, decision, entity, file, artifact, and checkpoint identifiers preserved exactly;
- compactor model/tool/prompt version, input hash, output hash, created time;
- confidence/known gaps and sensitivity/redaction policy;
- the prior summary it supersedes.

The raw trace remains separately policy-retained and searchable where consent allows. A summary is a regenerable context projection: it cannot become admitted personal memory, evidence, verification, or acceptance merely because it was produced during a successful run.

### 8. Retrieval and RunBrief injection

**Proposed fixed precedence:**

1. exact current contract and explicit user decisions;
2. current plan, WorkUnit, acceptance criteria, and known blockers;
3. current policy, grants, capabilities, budget, and destructive-action limits;
4. verified dependency outputs and workspace/repository rules;
5. approved project knowledge and admitted durable memory relevant to purpose;
6. optional retrieved episodes, summaries, and provider context, clearly marked and bounded.

Each retrieved item needs source/provenance, scope, admission/trust state, age/expiry, uncertainty, retrieval reason, and token allocation. User statements and current contract facts outrank behavioral inference. A retrieved instruction from an untrusted document remains data, not an instruction. The RunBrief compiler—not each Home/Work UI and not each provider—owns this logic once for all lanes.

### 9. Background consolidation

Letta and LangMem both show why background consolidation is attractive; neither justifies silently changing Waldo's durable truth.

**Proposed background job lifecycle:**

`eligible source range → extraction job → candidate set → contradiction/deduplication analysis → admission queue → admitted revision or rejection → projection regeneration`

- Semantic and episodic extractors may propose claims with source spans and uncertainty.
- Procedure/prompt optimizers may propose a new revision with evaluation cases and rollback link.
- Contradictions create counter-evidence links and review work; they do not silently overwrite history.
- Deletion/revocation removes the candidate/revision from future retrieval and advances a deletion generation so stale checkpoints cannot reintroduce it.
- A background model or second “reviewing” model is not the user. Model consensus may affect prioritization/confidence, not approval.

### 10. Shared versus private runtime state

| Scope | Allowed content | Promotion path | Default retention |
|---|---|---|---|
| Model-call scratch | Intermediate reasoning/context needed for one call | None; discard unless an explicit artifact/candidate is created | Ephemeral |
| Attempt-private runtime | Checkpoint, queue/cursor, raw provider refs, compacted trace, effect journal | Candidate evidence or memory via explicit extraction | Attempt/recovery policy |
| WorkUnit-shared dependency | Verified artifact/result references required by sibling attempts | Through dependency/evidence service | Outcome/audit policy |
| ResponsibilitySpace knowledge | Admitted project facts, decisions, procedures | `MemoryCandidate`/procedure admission with scope and revision | Governed until expiry/revocation/deletion |
| User personal knowledge | Admitted personal facts/preferences/episodes | Explicit user/policy admission, correction, release | User-governed |
| Provider/framework state | Opaque thread/checkpoint payload | Never promoted wholesale; extract only source-linked candidates | Minimum necessary; deletable/exportable |

No “shared agent memory” bucket should become an implicit authority channel. When one agent learns something another needs, the handoff should reference a canonical dependency, admitted knowledge revision, artifact, or evidence candidate.

## Failure and recovery matrix

| Failure | Required durable facts | Safe behavior | Unsafe shortcut |
|---|---|---|---|
| Process dies before effect dispatch | Prepared intent, no dispatch receipt | Revalidate authority, dispatch once with same stable key | Generate a new unlinked tool call |
| Process dies after dispatch but before receipt | Dispatched intent with provider/idempotency key | Reconcile provider; pause if disposition remains unknown | Blind retry |
| Tool returns but checkpoint write fails | Provider receipt discoverable by stable key | Reconcile, append receipt, then advance checkpoint | Assume failure from missing checkpoint |
| Parallel worker resumes stale attempt | Lease/fence epochs | Reject stale writes/events/receipts | Last writer wins |
| Contract/grant revoked while sleeping | Frozen and current revisions | Require fresh RunBrief and user/policy decision | Resume old authority |
| Provider payload incompatible after upgrade | Provider/schema version and hash | Migrate with tested converter or start replacement attempt from canonical facts | Deserialize arbitrary legacy objects |
| Summary omits an unresolved effect | Source coverage and effect journal are separate | Effect journal controls recovery; regenerate summary | Trust summary as full state |
| Background memory conflicts with admitted claim | Source links, expected revision, counter-evidence | Append candidate/conflict for review | Silent overwrite |
| User deletes/revokes memory while attempt paused | Deletion generation in checkpoint/current state | Block stale retrieval and recompile context | Restore deleted claim from checkpoint |
| Provider session terminates | Attempt/session facts and latest checkpoint | Classify failure/handoff/replacement; preserve evidence candidates | Mark Outcome complete |
| Verification fails after technically successful run | Evidence and Verification records | Plan revision or new attempt; user decides next step | Rewrite memory to say work succeeded |

## Evaluation contract before production use

No reference system's benchmark substitutes for Waldo evaluation. Before enabling automatic resume or background consolidation, test:

### Recovery correctness

- Crash injection before dispatch, after dispatch, after provider success, before receipt persistence, after receipt persistence, during compaction, and during checkpoint migration.
- Duplicate delivery and stale-worker fencing.
- Unknown-effect reconciliation for reversible and irreversible adapters.
- Contract/grant/budget/revocation changes while paused.
- Provider loss, payload corruption, incompatible schema, and missing raw trace.

### Memory safety

- False candidate, ambiguous source, contradiction, user correction, supersession, expiry, revocation, deletion, and attempted regeneration from a stale checkpoint.
- Concurrent foreground/background candidate creation and projection promotion.
- Cross-user, cross-space, and cross-attempt isolation.
- Prompt injection inside retrieved files and memories.
- Reproduction of provenance and correction history after compaction.

### Long-horizon utility

- Retrieval precision/recall for current contracts, stable user facts, episodes, and procedures separately.
- Stale-memory harm rate and authority-confusion rate, not only answer accuracy.
- Context cost, latency, and usefulness of summaries versus raw trace.
- Recovery success without duplicate effects.
- User ability to inspect, correct, revoke, delete, and understand why an item was used.

### Proof-boundary checks

- Provider completion never creates an `AcceptanceDecision`.
- A checkpoint, memory record, summary, or tool receipt cannot masquerade as verified evidence.
- Only accepted authority can dispatch effects; retrieved memory cannot expand grants.
- Only the user acceptance action closes an Outcome.

## Consolidated Adopt / Adapt / Reject decision

### Adopt now at the contract level

- Attempt-scoped immutable checkpoint lineage with parent IDs.
- Stable effect intents, durable receipts, reconciliation, leases, and fence epochs.
- Explicit serializer/schema/provider versions and restrictive deserialization.
- RunBrief hash plus frozen contract/grant/capability/budget/workspace revisions in every resumable checkpoint.
- Separate raw trace, compacted summary, durable memory, canonical responsibility state, and proof.
- Source-addressed compaction with covered-message bookkeeping.
- Candidate-only background consolidation.

### Adapt after the shared contract stabilizes

- Git/MemFS-style human-readable projections for inspection and export.
- Provider-native checkpoints or thread state as opaque subordinate payloads.
- Semantic/episodic/procedural extractors and profile/collection projections.
- Framework-specific pause/resume, pending-write, and time-travel capabilities behind a stable adapter.
- Procedural self-improvement only as versioned, evaluated, reversible proposals.

### Reject for the initial Waldo slice

- A framework-owned general memory store as canonical truth.
- A second database writer or frontend-owned execution state.
- Autonomous model mutation/deletion of admitted personal or project memory.
- Shared mutable multi-agent scratch as cross-agent truth.
- Automatic persona inference in the highest-priority context tier.
- Blind replay, live-state snapshot assumptions, or framework termination as Outcome closure.
- Adopting maintenance-mode AutoGen as the core runtime.

## Smallest useful architectural decision

The smallest useful orchestration step is to approve a provider-neutral **`RuntimeCheckpoint` + `EffectIntent` + `EffectReceipt` contract** under `Attempt`, together with a recovery state machine and RunBrief revalidation rule. This remains Work-owned while the separately approved Home/OpenLoop lane proceeds in parallel; it does not trigger durable Memory implementation.

Durable memory remains behind its separate admission/evaluation gate. Background consolidation may be prototyped only as candidate generation against fixed test corpora until correction, counter-evidence, expiry, revocation, deletion, and stale-checkpoint suppression pass evaluation.

ADR 0004 explicitly changes the prior phase boundary: canonical Home/OpenLoop persistence may proceed in parallel after the necessary shared responsibility/intake contracts are ratified. Admitted durable Memory still comes later through its separate gate.

## Open questions to resolve before implementation approval

1. Which exact external effects must the first Work slice recover: filesystem edits only, or also git hosting, issue trackers, messages, releases, and deployments?
2. What is the minimum provider adapter state that cannot be reconstructed from Waldo facts and raw transcript?
3. Which checkpoint payload fields require encryption, and which must never be persisted?
4. What is the retention/deletion contract for raw traces versus summaries, receipts, and audit facts?
5. Does a replacement provider session continue the same `Attempt`, or does provider replacement always create a new `Attempt` linked by a recovery decision?
6. Which memory candidate classes can policy-admit without a user gesture, if any? The safe default is none for identity, relationships, health, finances, permissions, or behavioral inferences.
7. What adapter-specific reconciliation APIs exist for every effectful tool in the first slice?
8. What exact facts must remain available to audit a deleted memory without retaining the deleted content itself?

## Primary sources

### Letta / MemGPT

- [Letta Code pinned repository](https://github.com/letta-ai/letta-code/tree/4f2d0d13496117e8fbf584aa24bea595c464f11b)
- [Letta Code README](https://github.com/letta-ai/letta-code/blob/4f2d0d13496117e8fbf584aa24bea595c464f11b/README.md)
- [MemFS official documentation](https://docs.letta.com/concepts/memfs)
- [Memory official documentation](https://docs.letta.com/configuration/memory)
- [Current memory patch implementation](https://github.com/letta-ai/letta-code/blob/4f2d0d13496117e8fbf584aa24bea595c464f11b/src/tools/impl/memory-apply-patch.ts)
- [Current local compaction implementation](https://github.com/letta-ai/letta-code/blob/4f2d0d13496117e8fbf584aa24bea595c464f11b/src/backend/local/compaction.ts)
- [Current shared-memory skill](https://github.com/letta-ai/letta-code/blob/4f2d0d13496117e8fbf584aa24bea595c464f11b/src/skills/builtin/managing-shared-memory/SKILL.md)
- [MemGPT: Towards LLMs as Operating Systems](https://arxiv.org/abs/2310.08560v2)
- [Sleep-time Compute](https://arxiv.org/abs/2504.13171v1)

### LangGraph / LangMem

- [LangGraph pinned repository](https://github.com/langchain-ai/langgraph/tree/f09cfe8ffc1eeffd68f4b628ed69c30f7cad229f)
- [LangGraph checkpoint library](https://github.com/langchain-ai/langgraph/blob/f09cfe8ffc1eeffd68f4b628ed69c30f7cad229f/libs/checkpoint/README.md)
- [LangGraph persistence documentation](https://docs.langchain.com/oss/python/langgraph/persistence)
- [LangGraph functional API and replay documentation](https://docs.langchain.com/oss/python/langgraph/functional-api)
- [LangGraph SQLite saver](https://github.com/langchain-ai/langgraph/blob/f09cfe8ffc1eeffd68f4b628ed69c30f7cad229f/libs/checkpoint-sqlite/langgraph/checkpoint/sqlite/__init__.py)
- [LangMem pinned repository](https://github.com/langchain-ai/langmem/tree/29cbe41e58528f92e9efa773c12e15c47be3808c)
- [LangMem conceptual guide](https://github.com/langchain-ai/langmem/blob/29cbe41e58528f92e9efa773c12e15c47be3808c/docs/docs/concepts/conceptual_guide.md)
- [LangMem memory manager](https://github.com/langchain-ai/langmem/blob/29cbe41e58528f92e9efa773c12e15c47be3808c/src/langmem/knowledge/extraction.py)
- [LangMem direct memory tools](https://github.com/langchain-ai/langmem/blob/29cbe41e58528f92e9efa773c12e15c47be3808c/src/langmem/knowledge/tools.py)
- [LangMem short-term summarizer](https://github.com/langchain-ai/langmem/blob/29cbe41e58528f92e9efa773c12e15c47be3808c/src/langmem/short_term/summarization.py)

### AutoGen

- [AutoGen pinned repository and maintenance notice](https://github.com/microsoft/autogen/tree/027ecf0a379bcc1d09956d46d12d44a3ad9cee14)
- [AutoGen memory protocol](https://github.com/microsoft/autogen/blob/027ecf0a379bcc1d09956d46d12d44a3ad9cee14/python/packages/autogen-core/src/autogen_core/memory/_base_memory.py)
- [AutoGen `ListMemory`](https://github.com/microsoft/autogen/blob/027ecf0a379bcc1d09956d46d12d44a3ad9cee14/python/packages/autogen-core/src/autogen_core/memory/_list_memory.py)
- [AutoGen group-chat state implementation](https://github.com/microsoft/autogen/blob/027ecf0a379bcc1d09956d46d12d44a3ad9cee14/python/packages/autogen-agentchat/src/autogen_agentchat/teams/_group_chat/_base_group_chat.py)
- [AutoGen code license](https://github.com/microsoft/autogen/blob/027ecf0a379bcc1d09956d46d12d44a3ad9cee14/LICENSE-CODE)

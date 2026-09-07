# Agent memory infrastructure benchmark for Waldo Kennel

- **Status:** architecture research; no implementation authority
- **Research date:** 2026-08-21
- **Scope:** Mem0, Zep/Graphiti, and Supermemory, translated into a local-first work-and-life memory architecture for Waldo Kennel
- **Phase boundary:** ADR 0004 now permits separately owned Home/OpenLoop persistence in parallel with the unchanged first-Outcome Work sequence; this document does not authorize durable admitted Memory or a second database

> **Architecture disposition:** ADR 0004 subsequently authorizes a separate Home/Personal Agent lane in parallel with Work and makes governed desktop screen/audio capture required capabilities. This report's evidence and memory-safety conclusions remain current; its sequential phase recommendation is superseded. Durable admitted Memory remains separately gated.

## Executive finding

There is no production-ready memory package that Waldo should adopt as its canonical memory layer.

The strongest design is a **governed memory ledger in Kennel's daemon-owned SQLite**, with model-generated extraction treated only as a proposal:

1. Sources and observations produce `MemoryCandidate`s.
2. Policy and, where required, the user admit a candidate as a versioned `MemoryRecord`/`MemoryRevision`.
3. Lexical, vector, and relationship indexes are disposable projections of admitted revisions, never independent truth.
4. Retrieval rehydrates canonical records, applies owner/space/purpose policy, excludes stale or revoked generations, and returns a provenance-bearing, uncertainty-bearing context packet.
5. Corrections supersede rather than overwrite; counter-evidence remains inspectable.
6. Deletion removes content and advances a content-free anti-resurrection generation before asynchronous index cleanup.
7. Memory can inform an `OpenLoop` or an `Outcome`, but it cannot create, accept, complete, or close either one.

The three systems contribute different useful mechanisms:

- **Graphiti:** adapt its episodic-source/fact/entity separation, temporal validity, provenance edges, and hybrid graph retrieval.
- **Supermemory:** adapt its inferred-memory review queue, version relationships, and preview-then-apply destructive operations.
- **Mem0:** adapt its scoped retrieval plumbing, modular embedding/reranking, over-fetching, and graceful retrieval degradation.

Their unsafe defaults are as important as their strengths. Mem0's current open-source v3 extraction path is additive and permissive; Graphiti can let model resolution and graph infrastructure become de facto truth; Supermemory's visible repository does not expose the core engine implementation at the inspected revision, while its reviewable inferences still participate in retrieval at reduced weight. None provides Waldo's required admission, revocation, deletion, authority, and cross-work/life policy boundary as one inspectable local transaction.

## Evidence discipline

This report uses the following labels:

- **Observed:** directly verified in a pinned official repository, release, schema, or source file.
- **Reported:** stated by an official project document or project-authored paper, but not independently verified here.
- **Inference:** a conclusion drawn from observed or reported mechanisms.
- **Proposed:** a Waldo design recommendation, not shipped behavior.
- **Unknown:** not established by the inspected primary sources.

Project-authored benchmark results are **Reported**, not proof of comparative superiority. They are useful for learning evaluation shapes and failure categories, not for selecting Waldo's architecture by leaderboard position.

## Inspected snapshots and licenses

| System | Inspected revision/release | Verified license | Source boundary |
|---|---|---|---|
| Mem0 | repository commit [`feb12852c0789a1f1182b05ee0dbc386037b012f`](https://github.com/mem0ai/mem0/tree/feb12852c0789a1f1182b05ee0dbc386037b012f) from 2026-08-21; Python package `2.0.18`; [release `v2.0.18`](https://github.com/mem0ai/mem0/releases/tag/v2.0.18) published 2026-08-11 | [Apache-2.0](https://github.com/mem0ai/mem0/blob/feb12852c0789a1f1182b05ee0dbc386037b012f/LICENSE) | Open-source library inspected. Its README explicitly says reported hosted-platform benchmark results include proprietary optimizations not present in the open-source package. |
| Graphiti / Zep | Graphiti commit [`993e081a6d7948a0d8851c12a5fbdbeb49fed862`](https://github.com/getzep/graphiti/tree/993e081a6d7948a0d8851c12a5fbdbeb49fed862) from 2026-08-20; package/release [`0.29.3`](https://github.com/getzep/graphiti/releases/tag/v0.29.3) | [Apache-2.0](https://github.com/getzep/graphiti/blob/993e081a6d7948a0d8851c12a5fbdbeb49fed862/LICENSE) | Graphiti open-source library inspected. Managed Zep is a separate product with proprietary Context Graph Engine behavior; its product claims are Reported. |
| Supermemory | repository commit [`34876664810a43a55954a0a83571662a3bd333b8`](https://github.com/supermemoryai/supermemory/tree/34876664810a43a55954a0a83571662a3bd333b8) from 2026-08-20; local server [release `server-v0.0.8`](https://github.com/supermemoryai/supermemory/releases/tag/server-v0.0.8) published 2026-08-17 | [MIT](https://github.com/supermemoryai/supermemory/blob/34876664810a43a55954a0a83571662a3bd333b8/LICENSE) for repository contents | The inspected repository exposes docs, SDK/API types, UI, and integrations, but not the local server's core memory-engine implementation. Engine internals and transaction guarantees are therefore Reported or Unknown despite the binary being distributed from the official repository. |

## Locked Waldo constraints

The benchmark is evaluated against Kennel's accepted architecture, not against a generic cloud-agent design:

- The daemon is the sole canonical writer; the renderer and clients do not own truth.
- SQLite is canonical. Auxiliary indexes must be rebuildable projections.
- A captured observation or model summary becomes a `MemoryCandidate`, never automatic durable truth.
- `MemoryCandidate`, `OpenLoop`, `Outcome`, `AgentSessionRef`, `Evidence`, `Verification`, and `Acceptance` remain distinct.
- User statements outrank behavioral inference, but corrections preserve inspectable lineage.
- Personal/work separation is policy, not merely a metadata tag.
- Retrieval is purpose-bound, minimized, attributable, fresh, and revocable.
- Deletion must prevent regeneration or resurrection from a stale index, cache, summary, or source-derived job.
- Current accepted sequencing allows the Local Focus Ledger Work slice and separately owned persistent Home/OpenLoop slice to proceed in parallel after shared contracts are ratified; durable Memory remains behind a later admission and evaluation gate.

## Comparative mechanism matrix

| Concern | Mem0 | Graphiti / Zep | Supermemory | Waldo implication |
|---|---|---|---|---|
| Canonical object | A vector-store memory payload plus separate SQLite history | Episodes, entity nodes, and fact edges in a graph | Documents plus versioned memory entries | Use explicit SQLite source, candidate, record, revision, provenance, and decision tables. |
| Time/version semantics | `created_at`, `updated_at`, optional expiry; manual update mutates the same vector ID | Ingestion time plus fact `valid_at`, `invalid_at`, and processing-time expiry | `version`, `isLatest`, parent/root IDs; update/extend/derive relations; optional expiry | Adapt Graphiti's event-time vs system-time distinction and Supermemory's immutable revision lineage. |
| Ingestion | LLM extraction from recent messages plus retrieved memories | Sequential episode ingestion, node/edge extraction, resolution, invalidation, persistence | Document indexing and later graph-memory “dreaming” are distinct stages | Separate source durability, extraction, review, admission, and projection readiness. |
| Dedup/consolidation | Exact content hash plus retrieved semantic context; current v3 path is ADD-only | Deterministic and LLM-assisted node/edge resolution; contradictory facts can invalidate older edges | Updates/extends/derives relations and current-version selection | Dedup candidates; do not consolidate admitted truth through an unreviewed model operation. |
| Retrieval | Semantic over-fetch, optional keyword/BM25, entity boosts, optional reranker | BM25, cosine, graph traversal, RRF, MMR, cross-encoder recipes | Hybrid memory/document search, threshold, optional rerank and query rewrite | Use multi-stage retrieval, but rehydrate canonical rows and reapply policy after every approximate index. |
| Graph relationships | Lightweight entities linked to memory IDs | First-class nodes, facts, episodes, communities, provenance | Memory parent/child update/extend/derive relations | Start with typed SQLite relations; add a graph engine only as a disposable projection if evaluation proves value. |
| Scope | Requires at least one `user_id`, `agent_id`, or `run_id`; filter-based | `group_id` partition; managed Zep describes a per-user graph | `containerTag` namespace and reported API-key scopes | Namespace strings are not authorization. Enforce owner, space, purpose, and sensitivity in the daemon. |
| Correction | Explicit update to the same memory ID; history records old/new content | New fact can temporally invalidate an earlier fact | New version with old revision no longer latest | Preserve immutable revisions, counter-evidence, and explicit admission/correction decisions. |
| Deletion | Hard delete vector; history records old content and delete event | Hard-delete APIs for episode/node/edge/group | “Forget” is a soft delete; batch forget can be previewed | Content-free tombstone + generation fence + cascade/rebuild receipts; neither old-content history nor indefinite soft retention meets Waldo deletion. |
| Provenance | Metadata and separate history; entity links to memory IDs | Fact edge retains source episode IDs | Associated documents, source counts, relation lineage | Make provenance mandatory and queryable on every admitted revision and retrieval result. |
| Observability | PostHog telemetry enabled by default; logging around fallbacks | Optional OpenTelemetry plus anonymous PostHog enabled by default | API/status fields; core internal instrumentation not inspectable | Local, privacy-preserving operational metrics and structured audit by default; external telemetry opt-in only. |
| Failure behavior | Several best-effort fallbacks; vector/history writes can diverge | Ingestion errors generally propagate; timestamp extraction may warn and continue; transactionality varies by graph driver | Pipeline statuses and review APIs are documented; core transaction/failure behavior Unknown | Model failure must not look like “no memory”; projection lag and partial extraction must be explicit durable states. |
| Evaluation | Project-authored paper and benchmark repository; hosted/OSS results differ | Project-authored paper and managed-product claims | MemoryBench harness and vendor-reported LongMemEval results | Borrow reproducible pipelines and failure inspection; require Waldo-specific privacy, correction, deletion, and authority tests. |

## Mem0

### Observed

Mem0 initializes an embedder, vector store, LLM, SQLite history manager, optional reranker, and a lazily created entity collection. An add or search is scoped by one or more of `user_id`, `agent_id`, and `run_id`; an unscoped call is rejected. These fields constrain retrieval but are not an authorization system. See the [memory implementation](https://github.com/mem0ai/mem0/blob/feb12852c0789a1f1182b05ee0dbc386037b012f/mem0/memory/main.py#L161-L328).

The current open-source v3 add path:

1. Reads a recent message window and retrieves existing scoped memories.
2. Prompts an LLM for **ADD-only** extracted memory statements.
3. Embeds candidates in a batch, with individual fallback.
4. Removes exact duplicates by content hash among retrieved and current candidates.
5. Inserts vector records.
6. Appends separate SQLite history rows.
7. Performs entity extraction/linking as a best-effort nonfatal step.

This behavior is visible in [`main.py`](https://github.com/mem0ai/mem0/blob/feb12852c0789a1f1182b05ee0dbc386037b012f/mem0/memory/main.py#L879-L1198). The extraction prompt explicitly says the operation is only ADD, considers user and assistant content, uses existing memories for deduplication/linking, and encourages extraction when uncertain; it can turn assistant recommendations or content from shared documents into candidate “memories.” See the pinned [prompt](https://github.com/mem0ai/mem0/blob/feb12852c0789a1f1182b05ee0dbc386037b012f/mem0/configs/prompts.py#L468-L620).

Search performs semantic over-fetching, optionally combines store-native lexical/BM25 results and entity boosts, and can run a configured reranker. Unsupported keyword search degrades to semantic search; reranker failure logs and returns the pre-reranked result. Expired items are hidden by default rather than necessarily removed. See [search and ranking](https://github.com/mem0ai/mem0/blob/feb12852c0789a1f1182b05ee0dbc386037b012f/mem0/memory/main.py#L1379-L1813).

Manual update preserves the vector record ID and creation time while replacing memory content, hash, and embedding, then appends a history row. Manual delete removes the vector record and writes a deletion history row containing the prior content. Vector and history writes are not one database transaction. Batch failures fall back to per-item writes, improving availability but permitting partial success and projection/history divergence. See [update/delete behavior](https://github.com/mem0ai/mem0/blob/feb12852c0789a1f1182b05ee0dbc386037b012f/mem0/memory/main.py#L1815-L2160) and the [SQLite history schema](https://github.com/mem0ai/mem0/blob/feb12852c0789a1f1182b05ee0dbc386037b012f/mem0/memory/storage.py#L11-L180).

Mem0 emits PostHog telemetry by default unless disabled. Telemetry failures do not block normal behavior. See [telemetry implementation](https://github.com/mem0ai/mem0/blob/feb12852c0789a1f1182b05ee0dbc386037b012f/mem0/memory/telemetry.py#L15-L240).

### Reported

The project-authored [Mem0 paper](https://arxiv.org/abs/2504.19413) reports long-term-memory quality, latency, and token improvements. The project's own README warns that hosted-platform benchmark results include proprietary optimizations absent from the open-source implementation; this makes hosted results unsuitable as proof that the inspected library behaves equivalently. See the [README warning](https://github.com/mem0ai/mem0/blob/feb12852c0789a1f1182b05ee0dbc386037b012f/README.md#L54-L63) and the official [memory-benchmarks repository](https://github.com/mem0ai/memory-benchmarks).

### Inference

- The current ADD-only extraction path is better understood as candidate generation than truth maintenance. It does not by itself provide correction, supersession, counter-evidence, or durable temporal validity.
- Exact content hashes prevent identical additions but do not solve semantic duplicates, conflicting facts, changing preferences, or source-quality differences.
- Because the vector write and history write are independent, SQLite history cannot guarantee a complete audit of what became retrievable.
- Filtering by an identity field can prevent accidental mixing in a correct caller, but it cannot enforce a privacy boundary against a buggy or compromised caller.
- Treating assistant output as extractable memory can convert a suggestion, hallucination, or plan draft into an apparent user fact.

### Adopt / Adapt / Reject

**Adopt**

- Semantic over-fetch before final ranking.
- Lexical fallback when embeddings or reranking are unavailable.
- Explicit scope required for every add/search operation.
- Failure-tolerant reranking that does not make all recall unavailable.

**Adapt**

- Use its modular embedding/vector/reranker shape only behind Kennel projection ports.
- Map identity filters to daemon-enforced owner, `ResponsibilitySpace`, purpose, and sensitivity policy.
- Run extraction into `MemoryCandidate`, with source role and provenance preserved; never insert directly into durable Memory.
- Preserve explicit states for “extraction produced zero candidates” versus “extraction failed or was incomplete.”

**Reject**

- Direct LLM-to-durable-memory insertion.
- “When in doubt, extract” as an admission policy.
- Treating assistant-generated recommendations as user facts.
- A vector store plus separate history as dual canonical state.
- Hard deletion while retaining deleted content in ordinary history.
- Default-on external telemetry for private work-and-life memory.
- A separate default data root outside `~/.kennel`.

### Unknown

- The consistency and deletion guarantees of every supported third-party vector backend.
- How hosted proprietary consolidation differs from the inspected open-source v3 path.
- Whether hosted results remain reproducible under Waldo's local models, privacy constraints, and cross-space policy checks.

## Graphiti and managed Zep

### Observed: Graphiti

Graphiti models three important layers:

- `EpisodicNode`: raw source content with source description, reference/valid time, and edges to extracted entities.
- `EntityNode`: a resolved person/place/object/concept with a summary and typed attributes.
- `EntityEdge`: an extracted fact or relationship, source episode IDs, embedding, and temporal fields including `valid_at`, `invalid_at`, and `expired_at`.

See the pinned [node model](https://github.com/getzep/graphiti/blob/993e081a6d7948a0d8851c12a5fbdbeb49fed862/graphiti_core/nodes.py#L93-L359) and [edge model](https://github.com/getzep/graphiti/blob/993e081a6d7948a0d8851c12a5fbdbeb49fed862/graphiti_core/edges.py#L49-L375). This separates the episode that was observed from the fact inferred from it and records both ingestion time and event-validity time.

Episode ingestion resolves nodes and edges against earlier graph state, derives temporal information, invalidates contradictions, writes the graph, and optionally updates communities. Calls are partitioned by `group_id`; documentation recommends sequential ingestion per group. Exceptions are captured in spans and re-raised. See [`add_episode`](https://github.com/getzep/graphiti/blob/993e081a6d7948a0d8851c12a5fbdbeb49fed862/graphiti_core/graphiti.py#L980-L1228).

Node resolution combines exact normalization, semantic candidates, deterministic guards, and LLM decisions. Defensive checks reject missing or malformed duplicate identifiers rather than blindly merging. See [node operations](https://github.com/getzep/graphiti/blob/993e081a6d7948a0d8851c12a5fbdbeb49fed862/graphiti_core/utils/maintenance/node_operations.py#L336-L650).

Graph writes use the selected driver's write-session abstraction. The Neo4j path can provide a transaction, while some driver implementations do not offer equivalent native transaction semantics; the library's consistency therefore depends on the backend. See [bulk graph writes](https://github.com/getzep/graphiti/blob/993e081a6d7948a0d8851c12a5fbdbeb49fed862/graphiti_core/utils/bulk_utils.py#L128-L260).

Search recipes combine BM25, cosine similarity, graph traversal, reciprocal-rank fusion, MMR diversity, cross-encoder reranking, node distance, and episode mentions. See [search configurations](https://github.com/getzep/graphiti/blob/993e081a6d7948a0d8851c12a5fbdbeb49fed862/graphiti_core/search/search_config_recipes.py#L33-L145) and [edge-search execution](https://github.com/getzep/graphiti/blob/993e081a6d7948a0d8851c12a5fbdbeb49fed862/graphiti_core/search/search.py#L253-L461).

Graphiti supports removal by episode, node, edge, and group. Removal is destructive; the inspected episode-removal method has no anti-resurrection tombstone or canonical generation fence. See [`remove_episode`](https://github.com/getzep/graphiti/blob/993e081a6d7948a0d8851c12a5fbdbeb49fed862/graphiti_core/graphiti.py#L1765-L1793).

Graphiti offers optional OpenTelemetry tracing and anonymous PostHog usage telemetry enabled by default unless configured otherwise. See [tracing](https://github.com/getzep/graphiti/blob/993e081a6d7948a0d8851c12a5fbdbeb49fed862/graphiti_core/tracer.py#L126-L193) and [telemetry](https://github.com/getzep/graphiti/blob/993e081a6d7948a0d8851c12a5fbdbeb49fed862/graphiti_core/telemetry/telemetry.py#L15-L117).

### Reported: Graphiti and Zep

The Graphiti README describes incremental temporal knowledge-graph construction, raw-episode provenance, temporal invalidation, and hybrid search; it also says Graphiti users must provide their own graph database and surrounding user, conversation, security, and operations layer. Managed Zep uses a separate proprietary Context Graph Engine. See the [Graphiti overview and product distinction](https://github.com/getzep/graphiti/blob/993e081a6d7948a0d8851c12a5fbdbeb49fed862/README.md#L41-L140).

Zep's official documentation reports a per-user graph across threads, facts with time-scoped relationships, entities, raw episodes, thread summaries, observations, and user summaries. It reports that deleting a user removes the user's threads and artifacts. These managed-product behaviors were not independently verified in source. See [Zep concepts](https://help.getzep.com/concepts), [user graphs](https://help.getzep.com/users-and-user-graphs), [context types](https://help.getzep.com/context-types), and [graph creation](https://help.getzep.com/how-graph-creation-works).

The project-authored [Zep paper](https://arxiv.org/abs/2501.13956) reports retrieval and response-quality results for its temporal graph approach. Those results are vendor-authored evidence, not independent proof for Waldo.

### Inference

- Graphiti offers the most useful inspected reference for separating a source episode from a temporally qualified derived fact.
- `created_at` plus `valid_at`/`invalid_at` approximates the two questions Waldo must answer: “When did Waldo learn this?” and “For what real-world interval was it claimed to be true?”
- Model-driven contradiction resolution is valuable as a proposal generator but unsafe as Waldo's canonical admission decision.
- A graph database becomes a second source of truth if Waldo cannot regenerate it deterministically from SQLite or fence it by canonical generation.
- A `group_id` is a partitioning aid, not sufficient proof that Home, Work, health, and third-party contexts cannot leak into each other.

### Adopt / Adapt / Reject

**Adopt**

- Source episode, derived fact, entity, and provenance separation.
- Valid-time and invalid-time semantics in addition to ingestion/system time.
- Supersession/invalidation instead of silently overwriting an old assertion.
- Hybrid lexical, semantic, graph, and diversity-aware retrieval recipes.
- Typed entity/relationship vocabularies and defensive validation of model-produced identifiers.
- Privacy-safe local spans for extraction, resolution, indexing, and retrieval stages.

**Adapt**

- Store typed relationship edges in canonical SQLite tables first. A graph service may later be a rebuildable index, not a writer of canonical facts.
- Treat episode extraction, entity resolution, and contradiction detection as `MemoryCandidate` or counter-evidence proposals requiring Waldo admission policy.
- Map `group_id` to a daemon-authorized owner/space/purpose scope; do not trust a caller-provided string by itself.
- Record temporal extraction uncertainty and parse failures explicitly rather than quietly leaving dates absent.
- Make every retrieved relationship point back to admitted revisions and their source evidence.

**Reject**

- Raw episodes, transcripts, or summaries as “ground truth.” They are sources with varying authority.
- LLM-driven contradiction resolution as an automatic durable update.
- A required external graph database or graph writer in the first local implementation.
- Hard deletion without an anti-resurrection marker and derived-data cascade receipt.
- Default cloud-model and external-telemetry assumptions for private personal memory.
- Automatic relationship inference across work/life or sensitive-space boundaries.

### Unknown

- Managed Zep's internal admission, correction, deletion-audit, and transaction mechanisms.
- Exact consistency equivalence across Graphiti's supported graph drivers.
- Whether deletion of one source fully retracts or recomputes every shared derived fact and community summary.
- How often temporal extraction failure produces a timeless fact that later outranks correct current information.

## Supermemory

Supermemory adds a distinct mechanism to this comparison: a product-level review workflow for inferred memories and a bounded preview/apply workflow for forgetting. Those mechanisms are more relevant to Waldo than another vector-store implementation.

### Observed in the public repository

The exposed `MemoryEntry` schema includes:

- `version`, `isLatest`, `parentMemoryId`, and `rootMemoryId`;
- relation types `updates`, `extends`, and `derives`;
- `isInference`, `isForgotten`, and `isStatic`;
- `forgetAfter` and `forgetReason`;
- source counts, embedding/model metadata, timestamps, and arbitrary metadata.

See the pinned [validation schema](https://github.com/supermemoryai/supermemory/blob/34876664810a43a55954a0a83571662a3bd333b8/packages/validation/schemas.ts#L239-L278). Search response types expose version, similarity, parent/child context relations, metadata, and associated documents. See the [search API types](https://github.com/supermemoryai/supermemory/blob/34876664810a43a55954a0a83571662a3bd333b8/packages/validation/api.ts#L680-L777).

The inspected repository does **not** expose the implementation of the memory engine or local server. The MIT license verifies the visible repository contents; it does not make uninspected server internals observable. The local server is distributed as an official [binary release](https://github.com/supermemoryai/supermemory/releases/tag/server-v0.0.8).

### Reported

Official documentation distinguishes raw documents from extracted memories and represents memory changes as update, extend, or derive relations. It says latest-version retrieval keeps current facts while retaining history. See [graph memory](https://github.com/supermemoryai/supermemory/blob/34876664810a43a55954a0a83571662a3bd333b8/apps/docs/concepts/graph-memory.mdx#L41-L174).

The documented ingestion pipeline separates source indexing (`queued`, `extracting`, `chunking`, `embedding`, `indexing`, `done`) from later memory-graph extraction called “dreaming.” A document being searchable does not mean memory extraction is complete. A stable `customId` can identify an evolving source. See [how ingestion works](https://github.com/supermemoryai/supermemory/blob/34876664810a43a55954a0a83571662a3bd333b8/apps/docs/concepts/how-it-works.mdx#L74-L165).

Search can combine memories and document chunks, apply thresholds, rewrite queries, and optionally rerank. Forgotten and expired items are excluded by default. See [search documentation](https://github.com/supermemoryai/supermemory/blob/34876664810a43a55954a0a83571662a3bd333b8/apps/docs/recall/search.mdx#L8-L215).

Memory operations include:

- direct creation of an exact fact;
- versioned update;
- forgetting a single memory while retaining it in storage but excluding it from search;
- semantic bulk forget with a dry-run preview, threshold, maximum count, and a returned exact candidate set that can be used for the apply step;
- operation scoping by `containerTag` and a `forgetBatchId` receipt.

See [memory operations](https://github.com/supermemoryai/supermemory/blob/34876664810a43a55954a0a83571662a3bd333b8/apps/docs/recall/memory-operations.mdx#L15-L340).

Inferred memories have `isInference=true`, are downweighted, and enter a review queue. Approval removes the inference flag; decline marks the item forgotten; undo can return it to inferred/unreviewed state. The queue is scoped and excludes expired, forgotten, and already reviewed items. See [memory review](https://github.com/supermemoryai/supermemory/blob/34876664810a43a55954a0a83571662a3bd333b8/apps/docs/recall/memory-review.mdx#L8-L202).

`containerTag` is documented as the primary namespace and vector isolation boundary, with API-key read/write scopes. See [container tags](https://github.com/supermemoryai/supermemory/blob/34876664810a43a55954a0a83571662a3bd333b8/apps/docs/concepts/container-tags.mdx#L8-L127). The local server is reported to use local embeddings, support offline local models, and store state under a local directory. See [self-hosting overview](https://github.com/supermemoryai/supermemory/blob/34876664810a43a55954a0a83571662a3bd333b8/apps/docs/self-hosting/overview.mdx#L8-L74).

Supermemory's benchmark statements and [LongMemEval research page](https://supermemory.ai/research/longmembench/) are project-authored. The official [MemoryBench repository](https://github.com/supermemoryai/memorybench) is useful for its checkpointed ingest/index/search/answer/evaluate workflow and for reporting accuracy, latency, and token use separately, but its vendor comparisons are not independent architecture proof.

### Inference

- Separating “source searchable” from “memory extracted and reviewed” is a strong correction to the common one-stage ingestion mistake.
- Previewing a destructive semantic selection and then applying the exact reviewed IDs is safer than rerunning the query during deletion.
- A review queue is close to Waldo's `MemoryCandidate` concept, but downweighting unreviewed inferences is insufficient: an inference that appears in an agent prompt can still cause action.
- Soft forgetting is useful for reversible UI behavior but does not satisfy a right-to-delete or anti-resurrection guarantee because content remains present.
- `derives` is the riskiest relation for Waldo: a pattern-derived assertion must remain a candidate, regardless of confidence or source count.
- A binary memory server would become a second writer/state root unless Kennel treated it only as a rebuildable projection.

### Adopt / Adapt / Reject

**Adopt**

- Separate source durability, source-search readiness, candidate extraction, review/admission, and projection readiness.
- Stable source identities for evolving conversations/documents.
- Immutable version lineage with explicit update/extend relations.
- An inference-review inbox with approve, correct, decline, and inspect-source actions.
- Dry-run destructive selection followed by apply-to-the-exact-reviewed-ID-set.
- Batch receipts and bounded maximums for bulk revocation/deletion.
- Checkpointed evaluation with per-stage failure inspection and separate quality/cost/latency measures.

**Adapt**

- Keep unreviewed inferred candidates entirely outside trusted agent retrieval, rather than merely downweighting them.
- Replace generic approve/decline/undo with immutable admission, correction, revocation, and re-admission decisions.
- Keep a static/dynamic “profile” only as a regenerable, purpose-specific projection; never inject a global profile indiscriminately.
- Enforce owner/space/purpose/sensitivity in daemon policy in addition to any namespace tag.
- Implement preview/apply receipts in canonical SQLite so a later model query cannot alter the deletion target set.

**Reject**

- Soft forgetting as the only deletion behavior.
- Automatic `derives` assertions becoming durable Memory.
- Allowing unreviewed inferences into normal prompt context.
- An externally managed binary as the canonical local memory store.
- A whole-person profile automatically injected into every Work or Home request.
- Benchmark rankings as proof of privacy, correction, deletion, or orchestration fitness.

### Unknown

- The local server's internal extraction, consolidation, index, transaction, and crash-recovery implementation.
- Whether deletion cascades to embeddings, caches, summaries, backups, and derived graph nodes.
- How API-key and `containerTag` isolation are enforced inside the uninspected engine.
- Whether reported version and forget operations are atomic with every search projection.

## Proposed Waldo memory architecture

### 1. Canonical object model

The daemon-owned SQLite ledger should represent the following concepts explicitly:

```text
SourceArtifact / Observation
        |
        | extraction proposes; never admits
        v
MemoryCandidate --------------------> CandidateEvidence
        |
        | AdmissionDecision
        v
MemoryRecord --> MemoryRevision --> ProvenanceEdge --> SourceArtifact
                     |    |
                     |    +--> CounterEvidence / CorrectionDecision
                     |
                     +--> Revocation / Expiry / DeletionTombstone

MemoryRevision --> disposable FTS/vector/relation projections
                         |
                         +--> RetrievalReceipt --> purpose-bound context packet
```

**Proposed object responsibilities**

| Object | Responsibility |
|---|---|
| `SourceArtifact` | Identifies the original user statement, imported item, event, transcript segment, outcome fact, or other source. Holds capture time, source owner, space, permissions, and retention state. |
| `MemoryCandidate` | A proposed assertion with normalized subject/predicate/value or narrative content, source spans, extraction method/model, confidence, uncertainty, sensitivity, valid-time proposal, and review need. It is not available to trusted retrieval. |
| `CandidateEvidence` | Supports or contradicts a candidate. Multiple sources do not automatically make a candidate true. |
| `AdmissionDecision` | Records who/what policy admitted, corrected, rejected, or deferred a candidate and why. Sensitive, inferred, identity-shaping, and cross-space assertions require explicit user admission. |
| `MemoryRecord` | Stable identity for a conceptual memory across revisions. |
| `MemoryRevision` | Immutable admitted content with system-time interval, real-world valid-time interval, source strength, confidence/uncertainty, freshness, expiry, and revision number. One revision may be current; history is preserved until deletion policy removes content. |
| `ProvenanceEdge` | Links a revision to exact source spans and derivation steps. |
| `CounterEvidence` | A durable challenge to the current revision. It may propose a correction but does not silently overwrite. |
| `Revocation` | Makes a revision unavailable for use while preserving the permitted audit lineage. |
| `DeletionTombstone` | Content-free marker containing stable identity/hash, scope, deletion generation, time, and reason class. It prevents stale jobs or source reprocessing from recreating deleted content. |
| `ProjectionCheckpoint` | Records which canonical generation an FTS/vector/relation index has processed. |
| `RetrievalReceipt` | Records purpose, caller, authorized spaces, query transformation, candidates considered, canonical revisions disclosed, exclusions, uncertainty, index generations, and model/ranker versions. Content retention follows audit policy. |

### 2. Keep product concepts separate

Memory is context, not authority:

```text
MemoryCandidate --may propose--> OpenLoopCandidate
OpenLoopCandidate --explicit confirmation--> OpenLoop

MemoryRevision --may inform--> Outcome / ContractRevision / RunBrief

Outcome
  -> ContractRevision
  -> PlanRevision
  -> WorkUnit
  -> Attempt
  -> AgentSessionRef
  -> EvidenceItem
  -> VerificationRun
  -> AcceptanceDecision
```

- A `MemoryCandidate` is not an `OpenLoop`.
- An admitted memory that “the user intends to do X” does not authorize X.
- An `OpenLoop` does not become an `Outcome` without explicit transition.
- A remembered commit, check, or provider-session result is not `EvidenceItem` until attached through the evidence contract.
- Memory retrieval cannot produce `VerificationRun` or `AcceptanceDecision`.
- Revoking a memory does not silently rewrite an Outcome contract; it can raise a review signal.

### 3. Admission and revision lifecycle

```text
candidate:
  proposed -> needs_review -> admitted
            -> rejected
            -> expired
            -> withdrawn

admitted revision:
  current -> superseded
          -> disputed
          -> revoked
          -> expired
          -> deleted
```

**Proposed admission rules**

1. An explicit current user statement is the strongest personal source, but its scope and time still matter.
2. A deterministic fact from a user-authorized source can be admitted under a narrow pre-approved rule; the rule and parser version are provenance.
3. A third-party statement remains attributed to that party.
4. A model summary, behavioral pattern, sentiment, inferred relationship, health interpretation, personality label, or predicted intention always remains a candidate until explicit admission.
5. Assistant proposals and generated plans are never memories about the user merely because the assistant said them.
6. Contradiction creates counter-evidence and a proposed revision. The latest ingestion is not automatically the latest truth.
7. Corrections supersede the current revision with immutable lineage. Retrieval uses the current authorized revision and can disclose uncertainty when facts remain disputed.
8. Expiry prevents use; regeneration requires re-evaluation against the source and current admission policy.
9. Revocation prevents use and marks every projection stale for that record generation.
10. Deletion first writes the content-free tombstone and advances the generation transactionally, then scrubs content and rebuilds projections. A stale projection result must fail canonical hydration.

### 4. Single-writer storage and projection design

SQLite remains the only canonical store and the daemon remains the only writer.

Recommended initial projection stack:

- SQLite FTS for exact names, phrases, tasks, source titles, and lexical fallback.
- An embedding table or local vector extension only as a derived index keyed by `memory_revision_id` and canonical generation.
- Typed SQLite relationship tables for entities and admitted relations.
- Optional local reranker behind a bounded interface.
- No graph database until Waldo-specific evaluation shows that SQLite relations plus hybrid retrieval fail a concrete user outcome.

Every projection row carries:

- canonical revision ID;
- owner and authorized space key;
- sensitivity class;
- projection schema/model version;
- canonical record generation;
- indexed time;
- source/admission policy version.

The daemon publishes projection jobs only after the canonical transaction commits. A failed job is retryable. It may reduce recall, but it cannot expose a revoked/deleted item because all results are rehydrated from current canonical rows.

### 5. Retrieval pipeline

```text
request + purpose + authority
        |
        v
policy-derived eligible spaces and sensitivity classes
        |
        v
lexical + semantic + typed-relation candidate retrieval
        |
        v
canonical SQLite hydration and generation check
        |
        v
filter revoked / deleted / expired / superseded / unauthorized
        |
        v
hybrid rank + freshness + source strength + diversity
        |
        v
minimal provenance-bearing context packet + RetrievalReceipt
```

**Proposed ranking inputs**

- lexical relevance;
- semantic relevance;
- typed entity/relation match;
- purpose and current Outcome/ResponsibilitySpace relevance;
- currentness and real-world valid time;
- source strength and admission type;
- uncertainty/dispute penalty;
- diversity/MMR to avoid repeated near-duplicates;
- explicit user pin or exclusion;
- sensitivity minimization.

The context packet should expose, at minimum, revision ID, concise content, owner/space, valid-time/freshness, uncertainty, source label, and correction affordance. The model never receives an unattributed profile dump.

### 6. Failure behavior

| Failure | Required Waldo behavior |
|---|---|
| Extraction model unavailable or malformed | Mark extraction job failed/incomplete; do not report “no memories found.” Retry is explicit and idempotent. |
| Some source segments fail | Persist coverage ranges and incomplete status. Never imply full source review. |
| Embedding/indexing fails | Keep canonical admission intact; retrieve lexically; expose projection lag; queue rebuild. |
| Reranker fails | Return policy-filtered base ranking and record degradation in the receipt. |
| Relation extraction fails | Preserve the admitted narrative revision; omit proposed relation and record the failure. |
| Conflicting facts | Preserve both source lineages, mark disputed, and request correction when consequential. |
| Projection is behind | Hydration rejects stale generations. A miss may reduce recall; stale content must never be disclosed. |
| Correction arrives during retrieval | Canonical hydration sees the new generation; prior projection candidate is dropped. |
| Delete/revoke requested | Commit generation fence/tombstone first; asynchronously scrub projections, caches, summaries, source-derived jobs, and permitted backups; issue a deletion receipt. |
| Daemon crashes mid-projection | Resume from durable job/checkpoint. Canonical transaction remains authoritative. |
| Cross-space query or model prompt attempts expansion | Deny by policy and audit the denied purpose/space request. |
| External model/provider unavailable | Home and Work remain usable from admitted canonical facts and lexical retrieval; no provider outage changes truth. |

### 7. Correction, revocation, deletion, and regeneration

Correction and deletion are different operations:

- **Correct:** create a new revision, supersede the prior one, retain allowed provenance and counter-evidence.
- **Revoke:** stop use immediately while retaining the audit facts allowed by policy.
- **Delete:** remove memory and source-derived content, retain only a content-free tombstone and operational deletion receipt.
- **Regenerate:** re-run derivation only from still-authorized sources and only after checking the tombstone/generation. Regeneration produces candidates, not admitted truth.

A deletion preview should freeze exact target IDs and derived-dependency IDs, following the useful Supermemory preview/apply shape. The apply step must reference that reviewed set; it must not rerun a semantic query that could select different memories. The receipt should report successfully removed, pending projection cleanup, already absent, and policy-retained content-free markers.

### 8. Observability and audit

Default observability is local and privacy-minimized:

- counts and latency by pipeline stage;
- candidate/admission/rejection rates by source class, not raw content;
- projection lag and rebuild state;
- retrieval empty/degraded/denied reasons;
- correction propagation time;
- delete propagation and stale-index rejection counts;
- model/parser/schema versions;
- explicit user-visible provenance and decision history.

External telemetry is opt-in and must never include raw memory, source text, entity names, embeddings, prompts, or stable personal identifiers. Operational traces and user-facing audit are distinct: a trace diagnoses the system; an audit explains why a fact was admitted, used, corrected, revoked, or deleted.

## Evaluation gates for “SOTA” Waldo memory

“SOTA” should mean the best measured behavior for Waldo's actual work-and-life contract, not the best vendor benchmark score.

### Component measures

- Extraction precision/recall, with assistant suggestions and quoted third-party claims as hard negatives.
- Candidate deduplication without merging distinct time intervals or people.
- Temporal parsing accuracy and uncertainty calibration.
- Admission precision: how often durable Memory is genuinely authorized and useful.
- Correction selection and propagation latency.
- Provenance completeness and source-span accuracy.
- Retrieval recall, nDCG/MRR, grounded-answer accuracy, latency, and token cost reported separately.
- Cross-space and sensitive-purpose leak rate; target must be zero in adversarial tests.
- Deletion non-resurrection after restart, failed cleanup, stale index, source re-import, summary regeneration, and model retry.
- False `OpenLoop` suggestion rate and false Outcome/closure authority rate.
- User effort for review, correction, release, and deletion.
- Re-entry time: how quickly Waldo helps the user resume the right personal or work context without reconstructing it.

### Waldo scenario suite

Public long-memory benchmarks may be included as compatibility baselines, but release gates require local scenarios such as:

1. A preference changes twice with explicit valid dates.
2. The user corrects a name the assistant previously guessed.
3. A meeting participant speculates about the user; Waldo attributes rather than adopts it.
4. A generated plan mentions a task the user never authorized.
5. A Home observation suggests a Work open loop; the user declines the link.
6. A Work Outcome references personal context under a narrow one-run permission.
7. Health capture is absent, declined, revoked, or deleted while Home and Work continue functioning.
8. A source is deleted while embeddings and summaries are intentionally stale.
9. The daemon restarts between canonical admission and projection indexing.
10. Two people or projects share names; dedup must not merge them.
11. A current user statement conflicts with an old high-similarity memory.
12. A model returns malformed entity IDs, dates, or deletion targets.
13. An index/model upgrade rebuilds all projections without changing canonical truth.
14. A retrieved memory helps construct a `RunBrief` but cannot manufacture Evidence or Acceptance.

Evaluation runs should be checkpointed by ingestion, extraction, admission, projection, retrieval, answer/use, correction, and deletion stages. Failures must be inspectable per stage. Report quality, latency, compute/token cost, privacy violations, and user-review burden separately; do not collapse them into one score.

## Adopt / Adapt / Reject summary

### Adopt

- Graphiti's source episode vs temporal fact vs entity distinction.
- Graphiti's valid/invalid time and provenance-preserving invalidation.
- Supermemory's explicit inferred-memory review and exact preview/apply destructive set.
- Supermemory's distinct source-search and memory-extraction readiness.
- Mem0/Graphiti hybrid over-fetch, lexical/semantic/entity/graph ranking, reranking, and diversity patterns.
- Versioned immutable revisions, source lineage, local observability, and staged evaluation.

### Adapt

- All extraction becomes `MemoryCandidate` generation.
- All identity/group/container filters become daemon-enforced owner/space/purpose/sensitivity policy.
- Graph/vector systems become rebuildable projections of SQLite.
- Inferred facts stay out of trusted retrieval until admitted, rather than being merely downweighted.
- Soft forget becomes immediate revocation plus governed hard deletion when requested.
- Profiles become purpose-specific, regenerable views with provenance, not global personality blobs.
- Relation and contradiction models propose; an admission decision governs durable change.

### Reject

- A second canonical store or second writer.
- Automatic durable memory from conversations, screenshots, ambient audio, summaries, assistant messages, or behavioral inference.
- Embeddings, entity graphs, summaries, or model confidence as truth.
- Latest-ingested-wins correction.
- Namespace strings as sufficient authorization.
- Hard delete without cascade/generation protection, and soft delete as a deletion guarantee.
- Automatic profile injection across Home and Work.
- Memory creating an `OpenLoop`, authorizing an `Outcome`, or accepting completed work.
- Default external telemetry and vendor benchmark claims as proof.

## Recommended phase sequence

This research alone did not justify overturning the prior phase boundary. ADR 0004 subsequently records the explicit user-approved change.

1. **Now, across the parallel lanes:** ratify the shared object vocabulary, admission/revision/deletion invariants, retrieval receipt, projection interface, and Waldo-specific evaluation suite.
2. **Home vertical slice in parallel with Work:** implement the smallest persistent Home/OpenLoop slice against the shared daemon contracts, with no duplicate intake/Q&A or UI-owned truth.
3. **Separate Memory gate:** run candidate-only extraction and retrieval experiments against fixtures; demonstrate correction, cross-space denial, crash recovery, deletion non-resurrection, and useful re-entry before enabling durable admitted Memory.
4. **Only if evaluation requires it:** add a vector extension or graph projection. Keep SQLite canonical and require deterministic rebuild and generation fencing.

ADR 0004 supplies the explicit approval for parallel Home/OpenLoop persistence. Enabling durable admitted Memory before its remaining gate would still require a separate architecture decision; it must not be introduced as an implementation detail of a memory library.

## Open questions before implementation

- Which memory classes may be admitted automatically from deterministic sources, if any?
- Which sensitivity classes always require explicit user review?
- What is the precise deletion retention policy for content-free tombstones and operational receipts?
- May encrypted backups retain deleted content for a bounded interval, and how is that disclosed and verified?
- What purpose taxonomy controls Home-to-Work disclosure and one-run permissions?
- What local embedding/reranking models meet privacy, latency, package-size, and hardware targets?
- What review-volume ceiling makes a candidate inbox useful rather than burdensome?
- What confidence representation is understandable enough to help a user correct Waldo without implying false numeric precision?
- Which typed relationships materially improve Waldo scenarios beyond SQLite FTS and vector retrieval?
- What evidence threshold would justify an external graph projection without creating a second authority?

## Bottom line

Waldo's advantage should not be “remember more.” It should be **remember only what is attributable, admitted, correctable, purpose-appropriate, expirable, revocable, deletable, and unable to manufacture authority**.

Mem0, Graphiti, and Supermemory each demonstrate useful machinery, but the state-of-the-art Waldo design is the governed composition around that machinery: a daemon-owned SQLite memory ledger; candidate-first admission; temporal revisions and counter-evidence; disposable hybrid indexes; policy-hydrated retrieval; exact correction/deletion receipts; and evaluation that includes privacy and non-resurrection alongside recall.

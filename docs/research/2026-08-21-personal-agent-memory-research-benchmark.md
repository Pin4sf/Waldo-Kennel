# Personal-agent memory research benchmark

**Date:** 2026-08-21

**Status:** Architecture research adopted selectively by ADR 0004; no implementation authority

> **Architecture disposition:** The user approved the governed typed-ledger direction, required screen/audio capture capabilities, and parallel Home/Work lanes in ADR 0004. Candidate-first admission, deletion, scope, and evaluation requirements remain binding; durable admitted Memory still requires its separate gate.

**Scope:** Long-lived personal memory for Waldo across life, work, and orchestration

**Implementation status:** No capability described as **Proposed** is shipped by Waldo

## Executive conclusion

The strongest design for Waldo is not a copy of any one memory framework. It is a governed, typed claim ledger with immutable source and revision lineage, bitemporal validity, explicit admission, evidence-backed derived projections, hybrid retrieval, and purpose-bound context compilation.

The core boundary is:

> Capture is a source, not memory. Memory is context, not authority. Responsibility and proof are not memory.

MIRIX contributes the clearest cognitive taxonomy and practical multimodal routing. Hindsight contributes the strongest inspected model for source-linked synthesis, temporal retrieval, soft invalidation, and regeneration. MemOS contributes useful resource, isolation, lifecycle, and read/write-scope abstractions. A-MEM contributes useful linked-note and neighborhood-discovery ideas, but its silent neighbor rewriting is unsuitable for user-owned durable truth.

For Waldo, those ideas should be combined only behind these invariants:

- observations and summaries first become `MemoryCandidate`s, never automatic durable truth;
- admitted claims remain correctable, revocable, expirable, and traceable to sources;
- a correction appends a new revision and suppresses the old one; it does not silently rewrite history;
- deletion includes an anti-resurrection marker that survives source reprocessing;
- retrieval returns a provenance-bearing, freshness-aware context packet, not a free-floating answer;
- `OpenLoop`, `Outcome`, `ContractRevision`, `AgentSessionRef`, `EvidenceItem`, `VerificationRun`, and `AcceptanceDecision` remain separate canonical objects;
- provider/session completion, memory confidence, and recalled statements cannot authorize work, verify it, accept it, or close an Outcome;
- SQLite remains the daemon-owned canonical store and single writer; search indexes and Markdown are disposable projections;
- credentials and secret values are not stored as ordinary memory. Waldo stores an access-controlled reference to an OS-managed secret when needed.

The recommended program order remains the accepted one: conduct memory research and contract design in parallel with the Work lane; stabilize the canonical Outcome spine; evaluate the first Work slice; then gate Home/OpenLoop persistence; and put durable admitted memory behind a separate architecture, privacy, deletion, and evaluation gate.

## Evidence discipline

This document uses these labels consistently:

- **Observed:** directly inspected in the pinned official repository, code, schema, documentation, or published paper.
- **Reported:** claimed by the authors or vendor, including benchmark results not independently reproduced here.
- **Inference:** a conclusion drawn from observed evidence; plausible, but not directly established by the source.
- **Proposed:** a Waldo design recommendation; not current Waldo functionality.
- **Unknown:** not established by the inspected primary sources.

Benchmark numbers are not directly comparable across systems. They vary by dataset revision, excluded categories, base model, retrieval budget, judge, prompting, number of runs, and whether the authors evaluated their own system.

## Repository and publication snapshot

| System | Primary publication | Inspected official implementation | Published version context | License | Confidence boundary |
|---|---|---|---|---|---|
| MIRIX | [arXiv:2507.07957v1](https://arxiv.org/html/2507.07957v1), 2025-07-10, CC BY 4.0 | [`Mirix-AI/MIRIX@8cb06a6`](https://github.com/Mirix-AI/MIRIX/tree/8cb06a62bbb7c478beb33dd4f2815696a72df482) | Main was ahead of the observed `v0.1.6` tag | [Apache-2.0](https://github.com/Mirix-AI/MIRIX/blob/8cb06a62bbb7c478beb33dd4f2815696a72df482/LICENSE) | Taxonomy and code paths are inspectable; benchmark and privacy claims remain author-reported |
| MemOS | [arXiv:2507.03724v4](https://arxiv.org/html/2507.03724v4), revised 2025-12-03 | [`MemTensor/MemOS@be68e2f`](https://github.com/MemTensor/MemOS/tree/be68e2fb5370866bd5e2b188bb3d22bd13b49e09) | Commit describes development version `v2.0.31`; observed latest tag was `v2.0.30` | [Apache-2.0](https://github.com/MemTensor/MemOS/blob/be68e2fb5370866bd5e2b188bb3d22bd13b49e09/LICENSE) | Broad platform and lifecycle claims exceed the coherent governance paths established by this inspection |
| A-MEM | [arXiv:2502.12110v11](https://arxiv.org/html/2502.12110v11), NeurIPS 2025 | [`agiresearch/A-mem@ceffb86`](https://github.com/agiresearch/A-mem/tree/ceffb860f0712bbae97b184d440df62bc910ca8d) | No release tag observed | [MIT](https://github.com/agiresearch/A-mem/blob/ceffb860f0712bbae97b184d440df62bc910ca8d/LICENSE) | The package demonstrates the note mechanism; its README points to a different repository for paper evaluation |
| Hindsight | [arXiv:2512.12818v1](https://arxiv.org/abs/2512.12818), [ACL 2026 demo paper](https://aclanthology.org/2026.acl-demo.27/) | [`vectorize-io/hindsight@6ff6dc6`](https://github.com/vectorize-io/hindsight/tree/6ff6dc692ea588067aa5e7235e80640c6a842ba6) | Main was ahead of the observed `v0.9.1` tag | [MIT](https://github.com/vectorize-io/hindsight/blob/6ff6dc692ea588067aa5e7235e80640c6a842ba6/LICENSE) | Strongest inspected source/synthesis/curation mechanisms; still a service architecture, not Waldo's local authority model |

The commit pins make this benchmark reproducible. They are not recommendations to vendor or copy an entire repository. License compatibility is only a source-level observation, not a legal or dependency-security review.

## What Waldo actually needs to remember

A personal agent spanning work and life needs several memory behaviors, but they should not all share one authority model.

| Need | Durable form | Admission rule | Typical expiry | Must never imply |
|---|---|---|---|---|
| Explicit identity and boundaries | user-authored core claim | explicit statement or confirmation | until revoked or reviewed | personality inference is fact |
| Event continuity | episodic claim plus source links | candidate by default; admit when useful and sufficiently grounded | domain-dependent | an event created a commitment |
| People, places, concepts, relationships | semantic claim graph | source-backed admission; conflicting claims can coexist by valid time | review on contradiction or staleness | inferred relationship is authoritative |
| Preferences | scoped preference claim | explicit preference outranks behavior; inferred preferences remain candidates | short review horizon for inferred claims | preference authorizes consequential action |
| Decisions and rationale | decision record projection | user or accepted contract source | until superseded; retain lineage | recalled rationale is current authority |
| Documents, transcripts, media | resource/source object | import/capture grant and retention policy | source-specific | source contents are all true or admitted |
| Reusable workflow knowledge | procedural candidate or approved skill revision | evaluation and approval separate from memory admission | until superseded or failed | observed behavior is a safe procedure |
| Future responsibility | `OpenLoop`, `Outcome`, or candidate for either | explicit confirmation/contract | until closure/release | memory alone creates obligation |
| Execution proof | `EvidenceItem` and `VerificationRun` | verification contract | retention policy | a recollection proves completion |
| Current prompt context | ephemeral retrieval packet | purpose-bound compilation | end of run/session | retrieved context becomes durable truth |

This separation is the most important difference between a helpful memory system and an ungoverned personal dossier.

## System dossiers

### 1. MIRIX: cognitive taxonomy and multimodal routing

Pinned implementation evidence inspected: [README](https://github.com/Mirix-AI/MIRIX/blob/8cb06a62bbb7c478beb33dd4f2815696a72df482/README.md), [episodic schema](https://github.com/Mirix-AI/MIRIX/blob/8cb06a62bbb7c478beb33dd4f2815696a72df482/mirix/schemas/episodic_memory.py), [procedural schema](https://github.com/Mirix-AI/MIRIX/blob/8cb06a62bbb7c478beb33dd4f2815696a72df482/mirix/schemas/procedural_memory.py), [Knowledge Vault schema](https://github.com/Mirix-AI/MIRIX/blob/8cb06a62bbb7c478beb33dd4f2815696a72df482/mirix/schemas/knowledge_vault.py), and [auto-dream experience prompt](https://github.com/Mirix-AI/MIRIX/blob/8cb06a62bbb7c478beb33dd4f2815696a72df482/mirix/prompts/system/base/auto_dream_agent/experience.txt).

#### Observed

- The paper defines six memory types: Core, Episodic, Semantic, Procedural, Resource, and Knowledge Vault. The design uses a meta-memory manager plus specialized memory agents.
- Core memory is continuously visible and compacted when near capacity. Episodic records include an actor and time; semantic records represent concepts, entities, and relationships; procedural records hold workflows, guides, and scripts; resource memory retains documents, transcripts, and multimodal material.
- The implementation has separate schemas and managers for these types. Episodic records distinguish occurrence time from storage/update timestamps. Procedural records include triggers, examples, semantic versions, and tags.
- Retrieval combines embedding, BM25, and string matching, with topic generation before per-type retrieval. Current constants and code include hybrid ranking and session-tag routing.
- The repository contains create, update, and delete paths, soft-delete preferences, retained-session bounds, and an `auto_dream` consolidation path.
- The consolidation prompt instructs the model to merge duplicates, resolve conflicts conservatively, prefer more recent or detailed information when justified, and retain both claims with a discrepancy when uncertainty remains.
- The paper's desktop experiment samples screenshots, removes near-duplicates, groups unique frames, and sends batches to a multimodal model. This establishes an ingestion pattern, not a trustworthy durable-memory policy.
- Knowledge Vault records contain secret values and sensitivity metadata. The inspected schema did not establish a general encrypted, revisioned, revocable secret-management substrate.

#### Reported

- On LoCoMo, MIRIX reports 85.38% after excluding the adversarial/unanswerable category. MIRIX/full-context results were averaged across multiple runs while some baselines were run once.
- On the authors' ScreenshotVQA evaluation, MIRIX reports a 35% accuracy improvement over RAG and 99.9% storage reduction. The dataset was small and private: 87 questions across three users with large screenshot histories, judged with an LLM.
- The paper discusses encryption, fine-grained permissions, and decentralized storage as a future privacy direction. It is not evidence that those protections are complete in the inspected implementation.

#### Inference

- Typed memory routing is useful because different content needs different prompts, retention, retrieval, and review rules.
- The six-type taxonomy is a better UX and policy starting point than a single vector store, but it is not sufficient governance. Its record schemas lack a uniform immutable revision chain, admission decision, counter-evidence relation, expiry, revocation, and anti-resurrection generation.
- High-frequency screenshot capture can create continuity, but it also creates the highest-risk source stream. Similarity filtering and summarization reduce storage, not privacy or truth risk.

#### Unknown

- Whether current privacy controls cover provider copies, backups, derived summaries, indexes, and deletion propagation end-to-end.
- Whether claimed conflict handling is reliable across long-running, adversarial, or highly sensitive personal data.
- Whether benchmark performance transfers to a single-writer local-first desktop agent with strict admission and source-gap disclosure.

#### Adopt / Adapt / Reject

- **Adopt:** typed routing for episodic, semantic, resource, and procedural candidates; occurred-at versus recorded-at time; hybrid retrieval; source tags; conservative discrepancy retention.
- **Adapt:** keep Core limited to explicit, high-confidence, user-controlled claims; make consolidation append a projection revision instead of rewriting prior facts; make multimodal ingestion source- and purpose-granted, local-first, and visible.
- **Reject:** storing raw credentials or API keys as ordinary memory; assuming a screenshot summary is admitted truth; always-visible inferred personality; automatic promotion from capture to durable memory; treating paper privacy aspirations as implemented guarantees.

### 2. Hindsight: evidence-backed synthesis and temporal recall

Pinned implementation evidence inspected: [README](https://github.com/vectorize-io/hindsight/blob/6ff6dc692ea588067aa5e7235e80640c6a842ba6/README.md), [retain documentation](https://github.com/vectorize-io/hindsight/blob/6ff6dc692ea588067aa5e7235e80640c6a842ba6/hindsight-docs/docs/developer/retain.md), [observations documentation](https://github.com/vectorize-io/hindsight/blob/6ff6dc692ea588067aa5e7235e80640c6a842ba6/hindsight-docs/docs/developer/observations.mdx), [memory curation API](https://github.com/vectorize-io/hindsight/blob/6ff6dc692ea588067aa5e7235e80640c6a842ba6/hindsight-docs/docs/developer/api/memories.mdx), and [memory-defense documentation](https://github.com/vectorize-io/hindsight/blob/6ff6dc692ea588067aa5e7235e80640c6a842ba6/hindsight-docs/docs/developer/memory-defense/index.md).

#### Observed

- Hindsight separates four logical networks: world facts, the agent's own experiences, synthesized observations, and evolving mental models or beliefs.
- A memory bank isolates a user, agent, or project. Retention extracts facts, entities, temporal and causal relations. The caller must provide context that identifies the speaker so that the system can distinguish world facts from agent experience.
- Stored records include source document/chunk identity, tags, metadata, proof count, event/occurrence intervals, mentioned time, create/update time, source memory IDs, and graph edges.
- Recall combines semantic, BM25, graph, and temporal strategies, then fuses them. Ranking can include recency, temporal proximity, and proof count.
- Synthesized observations carry source IDs and evidence excerpts. New evidence can strengthen or refine an observation; near-duplicate observations are consolidated.
- Curation supports direct editing of raw world/experience facts, soft invalidation, restoration, and regeneration of derived observations. Synthesized observations are regenerated rather than directly edited.
- The curation documentation states that source documents remain the source of truth and that reprocessing a document resets curation derived from the original text.
- Memory-defense filtering and audit logging exist but are disabled by default in inspected documentation/configuration. The defense is not retroactive to previously retained content.
- Full bank deletion exists. The standard deployment uses a service/database architecture, including Postgres options, rather than Waldo's daemon-owned SQLite authority.

#### Reported

- The paper reports that the memory system raises one 20B model's score from 39 to 83.6 and reports 91.4 on LongMemEval and up to 89.61 on LoCoMo with larger models.
- The authors position the four-network design and retain/recall/reflect operations as a general long-term-memory substrate.

#### Inference

- Hindsight's source-linked observations are the closest of the inspected systems to the derived-projection model Waldo needs.
- Distinguishing occurrence time from learning/mention time is essential for corrections such as “I moved in June, but told Waldo in August.”
- Resetting curation on source reprocessing creates a direct resurrection risk. Waldo needs a correction/deletion overlay and source-generation fence that wins over replay.
- “Belief” and “mental model” are risky product terms for a personal agent. They invite inferred personality and opaque conclusions to look authoritative. Waldo should expose source-backed summaries or projections, with uncertainty and edit paths.

#### Unknown

- The completeness of deletion across backups, queues, embeddings, generated observations, provider caches, and exported copies.
- Independent benchmark reproduction and sensitivity to model/provider choice.
- The behavior of conflicting evidence across long periods when multiple sources have different reliability and disclosure scopes.

#### Adopt / Adapt / Reject

- **Adopt:** source-linked derived observations; evidence/proof counts as retrieval signals, not truth scores; bitemporal fields; hybrid semantic/lexical/graph/temporal retrieval; soft invalidation with inspectable history; dependency-aware regeneration.
- **Adapt:** use bank-like `ResponsibilitySpace` and disclosure scopes within SQLite rather than an external canonical service; rename beliefs/mental models to derived projections; enable relevant audit and sensitive-source defenses by default; make reprocessing honor correction/deletion generations.
- **Reject:** default automatic retention from every conversation; generated beliefs as canonical truth; disposition/personality shaping from behavior; curation reset that resurrects corrected/deleted data; disabled-by-default safety for sensitive domains; using the external memory service as a second source of truth.

### 3. MemOS: resource abstraction, scope, and lifecycle

Pinned implementation evidence inspected: [README](https://github.com/MemTensor/MemOS/blob/be68e2fb5370866bd5e2b188bb3d22bd13b49e09/README.md), [MemOS introduction](https://github.com/MemTensor/MemOS/blob/be68e2fb5370866bd5e2b188bb3d22bd13b49e09/docs/en/open_source/home/memos_intro.md), [general memory/cube types](https://github.com/MemTensor/MemOS/blob/be68e2fb5370866bd5e2b188bb3d22bd13b49e09/src/memos/types/general_types.py), [API product models and cube scopes](https://github.com/MemTensor/MemOS/blob/be68e2fb5370866bd5e2b188bb3d22bd13b49e09/src/memos/api/product_models.py), and [tree textual-memory documentation](https://github.com/MemTensor/MemOS/blob/be68e2fb5370866bd5e2b188bb3d22bd13b49e09/docs/en/open_source/modules/memories/tree_textual_memory.md).

#### Observed

- The paper frames memory as an operating-system-managed resource and distinguishes plaintext, activation, and parametric memory.
- The repository exposes add, retrieve, edit, and delete operations; textual, activation/KV, parametric/LoRA, and preference memory configurations; user/cube identity; and explicit readable versus writable cube IDs.
- The current platform supports text, images, tool traces, persona material, graph-backed retrieval, multimodal readers, multiple memory cubes, and asynchronous scheduling.
- Tree-based textual-memory documentation includes hierarchy, embeddings, tags, entities, source metadata, and activated/archived/deleted states.
- The project provides cloud, self-hosted, and local/plugin configurations. Some configurations depend on Neo4j and Qdrant; the local plugin can use SQLite.
- Documentation describes generated, activated, merged, archived, and frozen lifecycle states, plus provenance, versioning, audit, time-machine, governance, redaction, and compliance concepts.
- The inspected code clearly establishes cube, scheduler, textual-memory, and backend mechanisms. This inspection did not establish one coherent, end-to-end code path for all documented governance, rollback, and compliance claims.

#### Reported

- MemOS reports results including 88.83 on LoCoMo and 89.20 on LongMemEval through its OmniMemEval evaluation work.
- The paper reports that composable memory cubes can be migrated, shared, and fused across agents and systems.

#### Inference

- Readable and writable scopes are valuable for purpose limitation: a run may read a bounded set of memories without gaining authority to modify them.
- A portable memory resource is useful only if canonical ownership, deletion, correction, and conflict rules remain local and inspectable.
- Activation/KV and parametric/LoRA memory are poor first choices for personal truth because they are difficult to inspect, surgically correct, expire, export, or prove deleted.

#### Unknown

- Which governance and rollback guarantees apply consistently across every textual, activation, and parametric backend.
- Whether deletion of parametric or activation memory can meet Waldo's user-facing non-resurrection standard.
- Independent comparability of the reported evaluation scores.

#### Adopt / Adapt / Reject

- **Adopt:** first-class memory resources; explicit read versus write scopes; async work for derived indexes and projections; source/status metadata; isolated domains with controlled sharing.
- **Adapt:** make `ResponsibilitySpace` the canonical scope rather than portable cubes; keep SQLite authoritative and external graph/vector indexes disposable; expose natural-language correction only as a request that resolves to explicit revisions.
- **Reject:** multi-cube federation as a second authority; Neo4j/Qdrant as mandatory local dependencies; initial use of KV-cache or parametric memory for user facts/preferences; claims of deletion or rollback that cannot be inspected at the logical record level.

### 4. A-MEM: linked notes and agentic evolution

Pinned implementation evidence inspected: [README and evaluation-repository boundary](https://github.com/agiresearch/A-mem/blob/ceffb860f0712bbae97b184d440df62bc910ca8d/README.md), [memory system and note model](https://github.com/agiresearch/A-mem/blob/ceffb860f0712bbae97b184d440df62bc910ca8d/agentic_memory/memory_system.py), and [retrieval implementation](https://github.com/agiresearch/A-mem/blob/ceffb860f0712bbae97b184d440df62bc910ca8d/agentic_memory/retrievers.py).

#### Observed

- The paper proposes a Zettelkasten-inspired note model with content, contextual descriptions, keywords, tags, links, and automatic memory evolution.
- The package's `MemoryNote` includes content, ID, keywords, links, retrieval count, timestamps, context, category, tags, and an evolution-history field.
- New notes are embedded into a persistent Chroma collection. The model can decide to strengthen or update neighboring notes during insertion.
- Update replaces an indexed document; delete removes it. The package uses an in-memory dictionary alongside Chroma during a running process.
- The inspected mutation path can revise neighboring context and tags. It does not establish a uniform source-provenance model, immutable claim revisions, admission actor, temporal validity, sensitivity, counter-evidence, expiry, revocation, or anti-resurrection behavior.
- The official README points to a separate repository for evaluation reproduction, so the small package alone does not establish the paper's benchmark results.

#### Reported

- The paper reports improvements across long-term-memory benchmarks and six foundation models.
- The authors argue that automatic linking and evolution produce a more coherent, adaptable knowledge network than static retrieval.

#### Inference

- Automatic link suggestions can help Waldo reveal related episodes, entities, and decisions.
- Silent neighbor mutation is unsafe for durable personal memory: one low-quality or malicious source can cause second-order changes to previously admitted claims.
- A note graph is best treated as a derived projection over canonical claims, not the canonical store itself.

#### Unknown

- How the package behaves under high-volume multimodal ingestion, concurrent writers, adversarial content, long-running corrections, or strict deletion.
- Whether evolution history is sufficiently populated and durable to reconstruct every model-authored mutation.

#### Adopt / Adapt / Reject

- **Adopt:** atomic note-like projections; related-memory suggestions; retrieval telemetry as an optimization signal, not a truth signal.
- **Adapt:** generate links and neighbor-update proposals without mutating admitted claims; store accepted relationship changes as new revisions with provenance; rebuild the graph from SQLite facts.
- **Reject:** silent model-authored evolution of prior memory; Chroma plus process memory as canonical personal truth; automatic reinforcement based on retrieval frequency; using this package as Waldo's governance substrate.

## Cross-system synthesis

| Concern | MIRIX | Hindsight | MemOS | A-MEM | Waldo decision |
|---|---|---|---|---|---|
| Cognitive taxonomy | strongest explicit six-type taxonomy | world/experience/observation/belief networks | storage-form/resource taxonomy | note graph | use a Waldo-specific semantic taxonomy; do not equate storage form with truth class |
| Multimodal ingestion | strongest concrete screen/text/image/voice path | documents and contextual retain pipeline | broad multimodal reader/platform support | primarily textual notes | all modalities enter as governed sources and observations |
| Consolidation | auto-dream and specialist agents | evidence-backed observations and regeneration | scheduler/lifecycle concepts | automatic note evolution | append source-linked projection revisions; no silent fact rewriting |
| Contradiction | prompt-level conservative merge/discrepancy | evidence can refine observations | natural-language correction and lifecycle claims | neighbor evolution | preserve competing claims by valid time and source; explicit supersession/counter-evidence |
| Forgetting | retention bounds and soft-delete preference | invalidate/restore and bank deletion | archive/delete/freeze concepts | hard delete | expiry, revocation, deletion generation, projection/index cascade, non-resurrection tests |
| Retrieval | hybrid typed retrieval | strongest semantic/lexical/graph/temporal mix | tree/graph/vector backends | vector plus links | staged hybrid candidate generation, canonical hydration, conflict/freshness checks, bounded context |
| Ownership/privacy | local-use claims; future privacy direction | isolated banks; optional defense/audit | deployment and cube isolation options | limited governance | local daemon authority, per-source grants, read/write/disclosure scopes, access audit, export/delete |
| Evaluation | LoCoMo plus private ScreenshotVQA | LoCoMo/LongMemEval results | OmniMemEval-reported suite | separate evaluation repo | public benchmark subset plus Waldo authority, privacy, deletion, and capture-gap tests |

## Proposed Waldo memory architecture

Everything in this section is **Proposed**, not shipped.

### 1. Four planes, one canonical writer

```text
Source plane        raw user input, files, messages, screenshots, audio, app events
Claim plane         observations, candidates, admission decisions, claim revisions
Projection plane    episodes, entity summaries, timelines, related-note graph, daily summaries
Context plane       purpose-bound retrieval packets assembled for Home or Work runs

                         daemon-owned SQLite authority
                                   |
                  +----------------+----------------+
                  |                                 |
       encrypted/content-addressed blobs      disposable indexes
                  |                         FTS/vector/graph/cache
                  +----------------+----------------+
                                   |
                       daemon-owned Markdown export
```

- **Source plane:** stores or references what was captured, under a `CaptureGrant`, retention policy, sensitivity, and source generation. Raw sources may be short-lived even when a narrow derived claim survives.
- **Claim plane:** holds the smallest contestable assertion and its provenance. This is where correction, counter-evidence, admission, expiry, revocation, and deletion operate.
- **Projection plane:** builds human-usable continuity from claims. Every projection is derived, revisioned, regenerable, and non-authoritative.
- **Context plane:** compiles only what a particular run may see. It applies authority, purpose, freshness, sensitivity, and token-budget rules after retrieval.

SQLite owns canonical identity, lineage, state, and scope. Blob storage owns encrypted-at-rest payloads where needed. FTS/vector/graph stores accelerate discovery but cannot make a record live, admitted, readable, or current. The daemon hydrates every retrieved ID through SQLite before use.

### 2. Write path and admission state machine

```text
CaptureGrant
    -> RawSourceRef
    -> Observation
    -> MemoryCandidate
    -> AdmissionDecision
    -> AdmittedMemoryRevision
    -> DerivedProjectionRevision
```

Recommended candidate states:

```text
untrusted_observation -> candidate -> pending_review -> admitted
                                  \-> rejected
candidate/pending_review          -> expired
admitted                          -> superseded | revoked | deleted
```

Rules:

1. A capture event creates a source reference and, at most, an observation.
2. Extraction can propose atomic `MemoryCandidate`s. A candidate records the model/prompt/version that proposed it.
3. Admission policy considers source type, sensitivity, uncertainty, expected usefulness, expiry, review burden, and whether the claim was explicit or inferred.
4. Explicit “remember this” can expedite admission, but still creates a visible admission record and scope. It is not an invisible bypass.
5. User statements and corrections outrank behavioral inference. Contradictory user statements are reconciled by valid time or left visibly disputed; the newest statement is not blindly assumed universally true.
6. Highly sensitive, identity-defining, health, financial, or relationship inferences require confirmation or remain candidates with short expiry.
7. No admission path creates an `OpenLoop`, `Outcome`, authority grant, evidence item, verification, or acceptance decision.

### 3. Canonical claim contract

An admitted memory should be an atomic, revisioned claim, not an unconstrained paragraph. A proposed minimum contract is:

| Field group | Required content |
|---|---|
| Identity | stable claim ID, revision ID, memory kind, `ResponsibilitySpace` |
| Assertion | subject, predicate, value or structured payload; explicit versus inferred |
| Provenance | source IDs, content digests, minimal supporting excerpts, assertion actor, capture mechanism |
| Time | occurred/observed time, valid-from/to, recorded time, admitted time, review/expiry time |
| Derivation | extractor/model/prompt/version, uncertainty, confidence basis, derivation dependencies |
| Governance | admission actor/reason, confirmation level, sensitivity, purpose, readable/writable/disclosure scopes |
| Revision | supersedes, corrected-by, counter-evidence IDs, conflict set, current-state reason |
| Deletion | revocation/deletion state, deletion generation, content-free tombstone, propagation receipts |
| Retrieval | canonical status, freshness, importance/usefulness policy, last-access audit reference |

“Confidence” is not one magic probability. Waldo should expose why a claim is uncertain: inferred speaker, weak source, stale information, conflicting evidence, imprecise time, or model extraction uncertainty.

### 4. Waldo memory types

The cognitive taxonomy should be semantic and user-legible. It should not leak backend-specific representations such as KV cache or LoRA.

| Type | Content | Admission default | Retrieval behavior |
|---|---|---|---|
| Core boundary | explicit identity, standing boundary, stable preference | confirmation or explicit statement | small, always inspectable, never silently inferred |
| Episodic | an event or interaction with actors and time | candidate; promote when useful and grounded | temporal plus entity retrieval; show source gap |
| Semantic | entity, concept, relationship, fact | source-backed candidate; valid-time aware | lexical/vector/entity retrieval; conflict-aware |
| Preference | scoped likes, defaults, communication or work style | explicit > inferred; inferred expires quickly | only in matching purpose/domain |
| Decision/continuity | decision, rationale, supersession, unresolved question | accepted decision/contract source | prefer newest valid revision, preserve rationale lineage |
| Resource | source document, transcript, image/audio/video segment, imported archive | capture/import grant | chunk retrieval with source and retention controls |
| Procedural candidate | observed or proposed way to perform a task | evaluation/approval required before skill use | retrieval may propose a procedure; cannot silently execute it |

Two useful things are deliberately not memory types:

- prospective responsibilities belong in `OpenLoop` or `Outcome`, possibly preceded by a `CommitmentCandidate`;
- verification and proof belong in `EvidenceItem`, `VerificationRun`, and `AcceptanceDecision`.

### 5. Consolidation without silent rewriting

The useful part of “reflection,” “dreaming,” or “memory evolution” is synthesis. The unsafe part is mutation without a governance trail.

Proposed consolidation rules:

- append evidence to a claim or create a new revision; never overwrite the original source assertion;
- allow multiple incompatible claims in one conflict set, each scoped by source and valid time;
- derive entity summaries, episode clusters, timelines, and recurring-pattern suggestions as `MemoryProjectionRevision`s;
- require every projection sentence or atomic field to cite supporting claim IDs;
- preserve counter-evidence and indicate what would change the conclusion;
- make projections disposable and deterministically regenerable from currently allowed claims;
- run background consolidation only through the daemon's job ownership and transaction rules;
- never allow a projection to be the sole source for a new admitted memory without retaining its underlying claim lineage;
- never convert inferred patterns such as “you avoid difficult conversations” into Core memory without user confirmation.

### 6. Retrieval and context compilation

Retrieval should be a staged policy pipeline:

```text
run purpose + ResponsibilitySpace + authority
    -> candidate generation: FTS + vector + entity/graph + temporal
    -> rank/fuse: relevance + valid time + freshness + source quality + diversity
    -> canonical hydration: state + deletion generation + scopes + conflicts
    -> sensitive-data and disclosure policy
    -> evidence/source-gap check
    -> token-budgeted context packet
    -> RunBrief segment with provenance and uncertainty
```

Required properties:

- Hybrid retrieval is preferred over embedding-only search.
- Recency is one signal, not truth. A recent inference cannot displace an older explicit statement without a valid correction.
- Retrieval frequency may optimize caching; it must not increase factual authority.
- Conflicting claims should be returned together when material to the run.
- The packet records query purpose, included memory/revision IDs, excluded/conflicting items, freshness, compilation time, and policy version.
- The packet is ephemeral. The model's use of it does not automatically create a new memory.
- Memory is placed below the current `ContractRevision` and explicit authority in orchestration context.
- A missing source interval is stated as unknown. Waldo must not convert capture gaps into “nothing happened.”

### 7. Correction, counter-evidence, expiry, and revocation

Correction is a first-class user action and a conversational capability.

Examples:

- “That was not Kashish; it was Kshitij” creates a correcting revision, updates dependent projections, and retains the old assertion as superseded.
- “I used to prefer morning workouts; now I prefer evenings” preserves two valid-time intervals instead of declaring the old claim false for all time.
- “Do not use that conversation for Work” changes disclosure/purpose scope and regenerates affected context projections.
- “Forget this” revokes or deletes according to the user's requested semantics and displays propagation state.

Expiry should be policy-driven. Inferred preferences, temporary locations, and volatile project details need shorter review horizons than explicit identity boundaries or accepted decision history. Expired claims remain unavailable to normal retrieval unless a retention rule explicitly preserves a content-free audit reference.

Revocation differs from deletion:

- revocation makes a claim unavailable for future use while retaining inspectable lineage where policy permits;
- deletion removes content and derived material, leaving only the minimum content-free anti-resurrection and propagation receipt necessary to honor the deletion.

### 8. Deletion and non-resurrection

Deletion is successful only when later imports, retries, or source reprocessing cannot silently recreate the deleted claim.

Proposed sequence:

1. Commit a content-free tombstone containing object identity/digest, scope, deletion generation, and policy reason category.
2. Make canonical reads reject older generations immediately.
3. Remove or cryptographically discard raw payloads and affected blob keys as policy requires.
4. Delete search embeddings, FTS rows, graph edges, caches, and projections.
5. Regenerate dependent summaries and retrieval packets.
6. Propagate deletion to provider-side files/caches when an integration permits it; otherwise disclose the unresolved external copy.
7. Record content-free receipts for each propagation target and retry state.
8. Test source replay against the deletion generation before declaring completion.

Corrections need the same generation fence. Otherwise the Hindsight-style “reprocess from source” behavior can restore a fact the user has already corrected.

### 9. User ownership and privacy

Every continuous or imported source needs a visible `CaptureGrant` with:

- source and modality;
- apps, people, spaces, and time windows included or excluded;
- purpose and allowed destination spaces;
- local retention and raw-versus-derived retention;
- provider processing and geographic/third-party disclosure, if any;
- sensitivity class and review mode;
- pause, revoke, export, and delete controls.

Additional requirements:

- local processing is the default for capture segmentation, deduplication, and redaction when feasible;
- no screenshot, transcript, or audio stream is continuous by implication; capture state and gaps remain visible;
- bystander and third-party content receives stricter retention/disclosure rules than direct user input;
- health/body data remains permissioned context, not a prerequisite or product identity;
- secrets remain in Keychain or an equivalent OS secret manager; memory stores only the reference and permitted purpose;
- audit logs record reads, writes, admissions, corrections, disclosures, and deletions without duplicating sensitive content;
- users can inspect “why Waldo recalled this,” correct it, narrow its scope, revoke it, delete it, and see propagation status;
- export includes canonical claims, sources the user may export, revision lineage, and machine-readable scopes—not merely generated summaries.

## Multimodal life-and-work capture implications

MIRIX demonstrates that high-frequency screen capture can be reduced into searchable episodes. That is an implementation nudge, not the correct Waldo product boundary.

For Waldo, capture should be layered:

```text
device/app signal
    -> consent and exclusion gate
    -> local segmentation/dedup/redaction
    -> short-lived source segment
    -> untrusted observation
    -> user-visible episode candidate
    -> memory/commitment/Outcome proposal through separate admission paths
```

The same pipeline applies to screen, audio, calendar, messages, files, browser, terminal, and explicit Quick Capture, but the policies differ. Explicit Quick Capture has the strongest user intent. Imported documents have strong source fidelity but not necessarily truth. Ambient screen/audio has high continuity value but the weakest consent, speaker, and completeness guarantees.

No capture mode should infer that:

- a mentioned task is the user's responsibility;
- a visible status is current or accurate;
- a meeting statement is an accepted decision;
- a completed-looking artifact satisfies acceptance;
- silence or an unrecorded interval proves inactivity;
- repeated behavior is a durable preference.

Home may surface an episode, candidate Open Loop, or “ready to confirm” suggestion. Work consumes only confirmed responsibilities and accepted contracts. Both use the same candidate/admission primitives; they must not build duplicate Q&A or memory systems.

## Evaluation program

### Public benchmark coverage

- [LongMemEval](https://arxiv.org/abs/2410.10813) evaluates information extraction, multi-session reasoning, temporal reasoning, knowledge updates, and abstention over 500 questions. It is a useful baseline for personal conversational continuity.
- [LoCoMo](https://aclanthology.org/2024.acl-long.747/) evaluates long-conversation QA, event summarization, and multimodal dialogue generation. Results must report dataset version and any excluded categories; MIRIX's reported result excludes adversarial/unanswerable questions.
- [MemBench](https://aclanthology.org/2025.findings-acl.989/) separates factual from reflective memory and participation from observation, and includes effectiveness, efficiency, and capacity dimensions.
- [LongMemEval-V2](https://arxiv.org/abs/2605.12493) expands to long-running agent trajectories and evaluates static state, dynamic state tracking, workflow knowledge, environment gotchas, and premise awareness. It is especially relevant to Waldo's Work and procedural-memory lane.

### Waldo-specific gates

Public QA accuracy is necessary but far from sufficient. The durable-memory gate should include:

| Gate | Example measure |
|---|---|
| Candidate precision | fraction of proposed claims that users consider worth review/admission |
| Provenance completeness | every retrieved atomic claim resolves to allowed source lineage |
| Temporal correctness | valid-time and recorded-time questions, including retroactive corrections |
| Contradiction behavior | conflicts surfaced; no unsupported newest-wins behavior |
| Abstention and source gaps | says unknown when capture or evidence is incomplete |
| Correction propagation | dependent summaries and packets update within bounded time |
| Deletion non-resurrection | deleted/corrected content does not return after replay, reindex, restore, or regeneration |
| Scope isolation | zero cross-`ResponsibilitySpace`, person, purpose, or health-context leakage |
| Authority poisoning | memory cannot grant tools, expand a contract, create responsibility, or accept an Outcome |
| Multimodal attribution | speaker/app/window/time uncertainty retained through extraction |
| Procedural safety | recalled procedure remains a proposal until evaluated/approved |
| Long-horizon retrieval | answer quality and provenance over months of events and changing facts |
| Efficiency | p50/p95 ingestion, consolidation, retrieval latency; token, storage, energy, and model cost |
| User burden | review interruptions, correction time, false reminders, and trust recovery after error |
| Adversarial resilience | embedded instructions in sources cannot alter memory policy or orchestration authority |

Every evaluation result should pin model/provider, prompt/policy version, embedding/index version, benchmark revision, retrieval budget, run count, and judge method. Quality must be measured both before and after corrections/deletions, not only on a clean append-only corpus.

## Adopt / Adapt / Reject summary for Waldo

### Adopt

- MIRIX's legible cognitive types, typed managers, temporal episodic fields, multimodal segmentation idea, and hybrid retrieval.
- Hindsight's source-linked observations, bitemporal recall, evidence-backed regeneration, soft invalidation, and multi-strategy retrieval.
- MemOS's explicit read/write scope separation, first-class resource framing, asynchronous derived work, and domain isolation.
- A-MEM's atomic linked-note projections and related-memory suggestions.
- A benchmark program covering updates, temporal reasoning, abstention, reflective synthesis, work trajectories, efficiency, and capacity.

### Adapt

- Replace “beliefs,” “dreams,” and silent “evolution” with revisioned, source-backed projections.
- Replace external banks/cubes as authority with daemon-owned `ResponsibilitySpace` scoping.
- Replace auto-retain with source-specific capture grants and explicit admission policy.
- Replace hard overwrites with immutable claim revisions and counter-evidence.
- Replace embedding-first authority with SQLite hydration and deletion-generation checks.
- Replace generic confidence scores with inspectable uncertainty causes and confirmation levels.
- Replace raw-secret memory with OS-secret references.
- Treat procedural memory as a candidate for an approved skill/procedure, not an executable truth.

### Reject

- A single vector database as canonical personal memory.
- Automatic durable memory from conversations, screen/audio, behavior, or model summaries.
- Personality/disposition scoring or inferred identity as hidden Core memory.
- Silent mutation of neighboring or previously admitted memories.
- “Latest wins” contradiction resolution without valid-time and source reasoning.
- Reprocessing that restores revoked, corrected, or deleted information.
- Provider, activation/KV, or parametric memory as the first canonical store for personal claims.
- Memory as responsibility, authority, evidence, verification, acceptance, or closure.
- Benchmark scores or competitor architecture as proof of shipped Waldo behavior.

## Recommended smallest next decision

Approve or revise the architecture contract before implementation. The smallest design commitment worth making now is:

1. `MemoryCandidate` is a provenance-bearing proposal, not durable truth.
2. Admission, correction, counter-evidence, expiry, revocation, deletion generation, and access audit are first-class contracts.
3. SQLite is the sole canonical writer; blobs and indexes are subordinate; Markdown is an export/projection.
4. Retrieval produces a purpose-bound, provenance-bearing packet below the active contract and authority.
5. `OpenLoop`, `Outcome`, `AgentSessionRef`, evidence, verification, and acceptance stay separate.
6. Durable admitted Memory waits for its own approved schema, threat model, deletion design, and evaluation gate; Home/OpenLoop and candidate foundations may proceed in parallel with the Work Outcome spine.

ADR 0004 records the explicit change allowing persistent Home/OpenLoop work in parallel and carries its migration, ownership, and evaluation consequences. It does not approve automatic or durable memory admission.

## Unresolved questions for design approval

- Which memory kinds, if any, may be admitted automatically, under what confidence and expiry, and in which `ResponsibilitySpace`?
- Is raw ambient capture retained at all, or is only a locally derived, user-reviewable episode retained? What are the default time limits by modality?
- What exact semantics does the user expect from “forget,” “delete,” “do not use,” and “that is wrong”?
- Which content must be encrypted with per-domain keys, and what cryptographic erasure guarantees are feasible on backups?
- Which integrations can prove provider-side deletion, and how should unresolved external copies be disclosed?
- When may an inferred preference influence low-risk presentation defaults, even if it is not admitted as durable memory?
- Who can approve a procedural candidate as an executable skill, and what evaluation/rollback contract is required?
- What minimum audit detail remains useful without reproducing sensitive content?
- Which public benchmark subsets and private work+life scenarios form the durable-memory launch gate?

Until these are decided, schema or API work would prematurely encode policy.

## Primary sources

### Systems

- MIRIX paper: [Wang et al., “MIRIX: Multi-Agent Memory System for LLM-Based Agents,” arXiv:2507.07957v1](https://arxiv.org/html/2507.07957v1).
- MIRIX implementation: [`Mirix-AI/MIRIX@8cb06a62bbb7c478beb33dd4f2815696a72df482`](https://github.com/Mirix-AI/MIRIX/tree/8cb06a62bbb7c478beb33dd4f2815696a72df482).
- MemOS paper: [Kang et al., “MemOS: A Memory OS for AI System,” arXiv:2507.03724v4](https://arxiv.org/html/2507.03724v4).
- MemOS implementation: [`MemTensor/MemOS@be68e2fb5370866bd5e2b188bb3d22bd13b49e09`](https://github.com/MemTensor/MemOS/tree/be68e2fb5370866bd5e2b188bb3d22bd13b49e09).
- A-MEM paper: [Xu et al., “A-MEM: Agentic Memory for LLM Agents,” arXiv:2502.12110v11](https://arxiv.org/html/2502.12110v11).
- A-MEM implementation: [`agiresearch/A-mem@ceffb860f0712bbae97b184d440df62bc910ca8d`](https://github.com/agiresearch/A-mem/tree/ceffb860f0712bbae97b184d440df62bc910ca8d).
- Hindsight paper: [Latimer et al., “Hindsight is 20/20: Building Agent Memory that Retains, Recalls, and Reflects,” arXiv:2512.12818v1](https://arxiv.org/abs/2512.12818).
- Hindsight ACL demo: [Latimer et al., ACL 2026 System Demonstrations](https://aclanthology.org/2026.acl-demo.27/).
- Hindsight implementation: [`vectorize-io/hindsight@6ff6dc692ea588067aa5e7235e80640c6a842ba6`](https://github.com/vectorize-io/hindsight/tree/6ff6dc692ea588067aa5e7235e80640c6a842ba6).

### Evaluations

- [Wu et al., “LongMemEval: Benchmarking Chat Assistants on Long-Term Interactive Memory,” ICLR 2025](https://arxiv.org/abs/2410.10813).
- [Maharana et al., “Evaluating Very Long-Term Conversational Memory of LLM Agents,” ACL 2024](https://aclanthology.org/2024.acl-long.747/).
- [Tan et al., “MemBench: Towards More Comprehensive Evaluation on the Memory of LLM-based Agents,” Findings of ACL 2025](https://aclanthology.org/2025.findings-acl.989/).
- [LongMemEval-V2, arXiv:2605.12493v1](https://arxiv.org/abs/2605.12493).

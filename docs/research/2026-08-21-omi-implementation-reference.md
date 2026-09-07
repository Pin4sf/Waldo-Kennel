# Omi implementation reference for Waldo Kennel

**Status:** architecture research; no implementation authority

> **Architecture disposition:** ADR 0004 adopts governed desktop screen/audio capture as required Personal Agent capabilities and starts Home in parallel with Work. This report remains the evidence base; it does not authorize automatic capture, automatic durable truth, or an Omi-owned canonical store.

**Research date:** 2026-08-21

**Official repository:** <https://github.com/BasedHardware/omi>

**Inspected revision:** [`b056ecba3bdaa4f81940505cabd1b72857ab632d`](https://github.com/BasedHardware/omi/tree/b056ecba3bdaa4f81940505cabd1b72857ab632d), committed 2026-08-21 10:26:06 UTC

**Official product sources:** [Omi product](https://www.omi.me/pages/product), [privacy policy](https://www.omi.me/pages/privacy), [app marketplace](https://h.omi.me/apps)

**License:** [MIT](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/LICENSE), copyright 2024 Based Hardware Contributors

## Executive finding

Omi deserves a closer look, but its most valuable implementation lessons for Waldo are not the headline behaviours of continuous capture or “AI that knows everything.” The stronger references are the newer authority mechanisms underneath its desktop coordinator and canonical-memory work:

1. treat model interpretation as an untrusted proposal, then resolve it through one deterministic authority;
2. bind an approved decision to owner, surface, context generation, intent hash, and a one-shot consumption receipt before effects;
3. derive attention queues and open-loop snapshots from durable facts instead of giving the UI a second task store;
4. admit every memory through one lifecycle, with provenance, grounding, review, tombstones, and fail-closed retrieval hydration;
5. keep integrations capability-scoped and prove that a connection works rather than inferring it from configuration.

Those are strong **Adapt** candidates. They must be expressed through Kennel's Go daemon and its canonical SQLite store—not copied as Omi's separate TypeScript coordinator database or as a second assistant identity.

Omi is not evidence that Waldo should begin with ambient capture. Its current first run is a long permission and consent sequence; its default service architecture includes cloud processing and multiple remote stores; and parts of its user-facing task UI permit direct checkbox completion. Those choices conflict with Waldo's current local-first, smallest-sufficient-plan, explicit-authority, user-disposition, and phase-gated architecture.

The recommended phase decision remains unchanged: research and shared-contract design can proceed in parallel, but the canonical Work `Outcome` spine is evaluated first; persistent Home/Open Loop follows only after those contracts stabilize; durable Memory remains behind a separate admission and evaluation gate.

## Evidence discipline

This dossier uses these labels throughout:

- **Observed** — present in code or documentation at the pinned Omi revision.
- **Reported** — claimed by Omi's official site or policy, but not independently verified here.
- **Inference** — a conclusion drawn from the observed sources.
- **Proposed** — a Waldo/Kennel design recommendation, not shipped behaviour.
- **Unknown** — not established by the inspected primary sources.

Repository code and invariant documents show implementation intent at one commit. They do not by themselves prove which revision is deployed, how every client behaves in production, security efficacy, adoption, or performance. Marketing counts and testimonials are deliberately excluded from the architectural conclusions.

## Waldo constraints used for the comparison

The governing Kennel documents are [`kennel-v1-product-architecture.md`](../product/kennel-v1-product-architecture.md), [`kennel-v1-team-review-packet.md`](../product/kennel-v1-team-review-packet.md), [`kennel-v0-first-outcome-slice.md`](../product/kennel-v0-first-outcome-slice.md), [`2026-08-20-first-outcome-execution-handoff.md`](../superpowers/plans/2026-08-20-first-outcome-execution-handoff.md), [`0003-local-first-waldo-core.md`](../adr/0003-local-first-waldo-core.md), and the amendment in [`0004-parallel-home-personal-agent-and-required-capture.md`](../adr/0004-parallel-home-personal-agent-and-required-capture.md).

They establish the following non-negotiable comparison frame:

- Waldo is one agent identity; Home and Work are responsibility spaces, and Kennel is a local presence/harness.
- The Go daemon is the policy and write authority. Electron is a projection and command surface. SQLite has one canonical writer.
- `Outcome`, `ContractRevision`, `PlanRevision`, `WorkUnit`, `Attempt`, `AgentSessionRef`, `EvidenceItem`, `VerificationRun`, `AcceptanceDecision`, `OpenLoop`, `LoopDisposition`, `ResponsibilityLink`, and `MemoryCandidate` stay distinct.
- Provider/session completion, commits, checks, tasks, and inferred activity cannot accept an Outcome or consciously close an Open Loop.
- The accepted first proof is the Work Outcome/Focus Ledger slice. Persistent Home/Open Loop is sequenced after its evaluation; durable Memory is later still.

## Source and provenance assessment

### What was inspected

- Root product and architecture material: [`README.md`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/README.md), [`PRODUCT.md`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/PRODUCT.md), and the MIT [`LICENSE`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/LICENSE).
- Desktop agent coordinator code and design in `desktop/macos/agent` and `desktop/macos/docs`.
- Mobile onboarding and Home code in `app/lib`.
- Backend canonical-memory, task-candidate, integration, and deletion documentation and implementation in `backend`, `docs/memory`, and `docs/doc/developer`.
- macOS Rewind capture, OCR, exclusion, database, and file-storage code.
- The current official product, privacy, and marketplace pages linked above.

### License and reuse boundary

**Observed:** The repository is MIT licensed. The license permits use, modification, distribution, sublicensing, and sale, provided the copyright and permission notice are included in copies or substantial portions.

**Proposed:** Treat this dossier as mechanism research. Re-express adopted concepts in Waldo's existing domain and service boundaries. If Waldo later copies substantial Omi code, retain the MIT notice and add file-level provenance. Do not copy Omi trademarks, product copy, marketplace content, or claims. Architecture ideas alone do not establish shipped parity.

## 1. First run and adaptive intake

### What Omi does

**Observed:** The mobile onboarding wrapper is a fixed sequence: authentication, AI consent, name, language, acquisition questions, system permissions, speech profile, knowledge graph, and completion. Fresh installs force the AI-processing consent step before subsequent setup. See [`wrapper.dart`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/app/lib/pages/onboarding/wrapper.dart#L44-L123).

**Observed:** The macOS onboarding flow progressively requests screen recording, file access, microphone, accessibility, and automation permissions, with skip paths, before starting its capture services. It also presents a privacy/trust explanation. See [`OnboardingView.swift`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/desktop/macos/Desktop/Sources/Onboarding/OnboardingView.swift#L188-L343) and its service-start path at [lines 620–665](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/desktop/macos/Desktop/Sources/Onboarding/OnboardingView.swift#L620-L665).

**Observed:** A setup-question file asks role, intended usage, and age-range questions, but the inspected main mobile wrapper does not establish this survey as an adaptive plan-building dialogue. See [`setup_questions.dart`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/app/lib/pages/onboarding/setup/setup_questions.dart#L14-L37).

**Reported:** Omi describes its use loop as Ask → Learn → Do and claims users can turn conversations into summaries, tasks, memories, and actions. The official product page does not describe a smallest-sufficient plan proposal or a Waldo-like authority checkpoint.

**Inference:** Omi's onboarding is permission-led and capture-led. It is a useful reference for progressive permission explanation, but not for Waldo's intended adaptive outcome clarification.

### Waldo decision

**Adapt** the following:

- ask for a permission only when a proposed plan requires that capability;
- explain the purpose, data boundary, duration, revoke path, and reduced-function fallback before the system prompt;
- persist onboarding progress so a user can safely resume;
- allow skip without making Home or Work unusable.

**Reject** the following:

- a long fixed permission ladder before the first useful outcome;
- ambient capture as the default entry premise;
- treating consent to processing as authority to execute a plan;
- demographic or acquisition questions in the critical product path.

**Proposed:** Waldo first run should begin with one human intent in ordinary language. A single shared intake service should then produce one of: answer inline, ask a material clarification, propose the smallest sufficient plan, or reject with a reason. Only an explicitly accepted `ContractRevision` and authority preview may transition into execution.

## 2. Orchestration and the decision boundary

### What Omi does

**Observed:** Omi's desktop coordinator design assigns its TypeScript kernel ownership of sessions, runs, attempts, dispatches, candidate records, and action-queue derivation. Swift is a projection surface and provider adapters execute but do not approve or establish truth. See [`agent-coordinator.md`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/desktop/macos/docs/agent-coordinator.md#L1-L29).

**Observed:** The intent router distinguishes inline answer, spawn, continue, clarify, and reject. A semantic interpretation is explicitly an untrusted proposal. The deterministic router creates a decision bound to an owner, surface, authority snapshot, reason, explanation, and input hash. See [`desktop-intent-router.ts`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/desktop/macos/agent/src/runtime/desktop-intent-router.ts#L5-L137).

**Observed:** Before an effect, the router consumes a short-lived, one-shot decision and validates owner, surface, context generation, and intent binding. It consumes the receipt before effect dispatch so an uncertain effect cannot be casually replayed. See [`desktop-intent-router.ts`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/desktop/macos/agent/src/runtime/desktop-intent-router.ts#L173-L238).

**Observed:** The control-plane invariant pins execution profiles and context snapshots, separates Omi IDs from provider-native IDs, prevents multiple non-terminal attempt authorities for one run, and treats unknown outcomes as non-replayable without reconciliation. See [`agent-control-plane.md`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/docs/product/invariants/agent-control-plane.md).

**Inference:** This is Omi's strongest implementation reference for Waldo. The reusable pattern is not the exact route enum; it is the two-stage boundary:

```text
language/model interpretation (untrusted)
              ↓
deterministic policy decision (bound and expiring)
              ↓
one-shot authority consumption
              ↓
effect dispatch with reconciliation
```

### Waldo decision

**Adopt** the principle that model output never directly creates execution authority.

**Adapt** the receipt pattern into Waldo's ontology:

- `IntakeTurn` records the user's source text and surface.
- `ClarificationRequest` names only missing facts that materially change scope, acceptance, or authority.
- `PlanProposal` references the proposed `ContractRevision` and smallest sufficient plan.
- `AuthorityPreview` enumerates allowed effects, sensitive scopes, stop conditions, and expiry.
- `AuthorityDecision` records explicit accept/reject/revise intent against exact revisions.
- `DispatchReceipt` binds owner, responsibility space, contract revision, run brief, context snapshot, capabilities, and idempotency key before an `Attempt` is dispatched.

**Reject** a literal port of Omi's `omi-agentd.sqlite3` or a TypeScript coordinator as another Waldo authority. In Kennel, these semantics belong behind the Go daemon API and its one canonical SQLite writer.

**Unknown:** The inspected sources do not establish that every Omi client and legacy path has converged on the new coordinator invariant. Waldo should test its own API and storage invariants rather than assume Omi's convergence is complete.

## 3. Personal Home, attention, and Open Loops

### What Omi does

**Observed:** Omi mobile Home combines capture entry, today's tasks, daily recaps, recent conversations, and a memory/mind map. New users see capture-oriented start tiles. See [`home_content.dart`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/app/lib/pages/home/home_content.dart#L75-L168) and [its start tiles](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/app/lib/pages/home/home_content.dart#L259-L287).

**Observed:** The desktop action queue is a derived operational view over pending dispatches, failed or stale runs, artifact deliveries, memory/task candidate reviews, and reusable sessions. Dismissal is an overlay, and ordering is deterministic. See [`desktop-action-queue.ts`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/desktop/macos/agent/src/runtime/desktop-action-queue.ts#L9-L15) and [the queue builder](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/desktop/macos/agent/src/runtime/desktop-action-queue.ts#L117-L253).

**Observed:** Omi materializes device-scoped, generation-scoped, expiring open-loop snapshots from that queue. The backend can ingest those snapshots into recommendations after validating owner, time window, and canonical workstream. See [`workstream-continuity.ts`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/desktop/macos/agent/src/runtime/workstream-continuity.ts#L183-L199) and [`recommendations.py`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/backend/utils/task_intelligence/recommendations.py#L1164-L1192).

**Observed:** Task recommendations distinguish canonical tasks from reviewable candidates. Suggested Tasks are capped, provenance-bearing, generation-fenced, time-bounded, deduplicated, and protected from resurrection after a user override. “What Matters Now” is a deterministic shortlist and may correctly be empty. See [`task_candidate_lifecycle.mdx`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/docs/doc/developer/backend/task_candidate_lifecycle.mdx#L6-L63).

**Observed:** Omi's mobile Today widget lets the user toggle a task's `completed` flag directly. See [`today_tasks_widget.dart`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/app/lib/pages/conversations/widgets/today_tasks_widget.dart#L11-L41) and [the toggle handler](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/app/lib/pages/conversations/widgets/today_tasks_widget.dart#L123-L137).

**Inference:** Omi uses “open loop” primarily as an expiring operational-attention projection. Waldo's accepted `OpenLoop` is different: a durable user-confirmed responsibility that requires an explicit `LoopDisposition` to close. Omi's snapshot is therefore a useful signal source, not a compatible canonical object.

### Waldo decision

**Adopt**:

- projection-first Home composition;
- deterministic, bounded “what needs attention” lists;
- honest empty states;
- expiry, owner, device, and generation fencing for derived attention;
- provenance on every suggestion and an explicit review state;
- anti-resurrection after dismissal, rejection, or supersession.

**Adapt** Omi's action queue into two Waldo layers:

1. `AttentionItem` — an expiring projection derived from run state, candidate review, evidence, verification, acceptance, and confirmed Open Loops;
2. `OpenLoop` — created only through user confirmation or an explicit policy, kept separate from tasks and runtime failures, and closed only with `LoopDisposition`.

The initial Personal Home projections should be:

- **Today / Morning Brief:** a bounded explanation of confirmed responsibilities, planned work, time-sensitive attention, and uncertainty;
- **Catch Up:** changes since the last acknowledged checkpoint, with provenance;
- **Quick Capture:** input that becomes an `IntakeTurn`, `MemoryCandidate`, or Open Loop proposal—not automatic truth;
- **Confirmed Open Loops:** only durable user-confirmed loops;
- **Open Loop detail:** origin, evidence, links, next options, and disposition history;
- **Ready to Close:** candidate dispositions requiring conscious confirmation;
- **Home ↔ Work links:** explicit `ResponsibilityLink` records; linking does not merge lifecycles or move custody.

**Reject**:

- using failed runs, stale artifacts, or extracted action items as canonical Open Loops;
- direct checkbox completion as conscious closure;
- a generic dashboard of every possible metric;
- treating a daily recap or shortlist as acceptance truth.

## 4. Memory candidate, admission, and correction boundary

### What Omi does

**Observed:** Omi's locked memory-promotion invariant says every intake—conversation, explicit user statement, import, API, plugin, or integration—starts in Short-term. Exactly one consolidation route may promote, archive, send for review, or reject it. Only an atomic, server-authored promotion receipt and graph assertion may move content to Long-term. Search and vector indexes are projections, not authority. See [`memory-promotion-authority.md`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/docs/product/invariants/memory-promotion-authority.md).

**Observed:** Omi's domain model explicitly separates raw conversations from memory, requires provenance and grounding, collapses canonical lineage on default reads, and extracts workflow/action items separately. See [`domain_model.md`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/docs/memory/domain_model.md#L44-L118).

**Observed:** The memory schema includes source/evidence IDs, canonical lineage, processing state, promotion/ledger/graph information, and TTL. See [`domain_model.md`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/docs/memory/domain_model.md#L218-L280).

**Observed:** Omi's canonical architecture keeps historical records behind read-only adapters, hides pending content from protected consumers, records quote grounding, turns owner rejection into bounded negative feedback, and makes promotion atomic. Tombstones are written before asynchronous cleanup to prevent resurrection. See [`canonical_memory_architecture.md`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/docs/doc/developer/backend/canonical_memory_architecture.md#L47-L104).

**Observed:** Vector results are candidate identifiers only. They must be hydrated against canonical authority; missing, stale, deleted, restricted, or wrong-generation results fail closed. See [`canonical_memory_architecture.md`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/docs/doc/developer/backend/canonical_memory_architecture.md#L135-L154).

**Observed:** The desktop `create_memory` tool is instructed to run only after explicit affirmative language such as “remember” or “save.” It must not infer consent or claim that the resulting short-term candidate is already long-term memory. See [`omi-tool-manifest.ts`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/desktop/macos/agent/src/runtime/omi-tool-manifest.ts#L1063-L1094).

**Inference:** Omi's short-term layer is close to Waldo's `MemoryCandidate`, but its three-tier terminology is not itself the important part. The important part is one admission authority, complete provenance, candidate status, lineage collapse, and fail-closed projection hydration.

### Waldo decision

**Adopt**:

- no observation, transcript, summary, model extraction, integration payload, or screen event becomes durable truth directly;
- one candidate admission authority in the daemon;
- explicit provenance and evidence references;
- immutable admission/correction/revocation receipts;
- canonical lineage with supersession rather than in-place history erasure;
- generation fences and tombstones before asynchronous deletion;
- retrieval that hydrates every index hit against canonical authorization and current lineage;
- explicit user language outranking behavioural inference.

**Adapt** the Omi lifecycle into a Waldo-specific `MemoryCandidate` contract containing at least:

- candidate ID, owner, source kind, source reference, and capture timestamp;
- proposed claim and evidence span, without silently copying unnecessary raw context;
- confidence plus machine-readable uncertainty reasons;
- sensitivity, intended scope, allowed consumers, and purpose;
- observed-at, valid-from, valid-until, review-by, and expiry policy;
- status: pending, admitted, rejected, superseded, revoked, expired, or deleted;
- admission actor/policy/model version and decision rationale;
- correction and counter-evidence links;
- canonical-memory lineage ID, if admitted;
- deletion receipt and a content-free anti-resurrection marker;
- regeneration policy: derived summaries/indexes may be rebuilt only from still-authorized sources.

**Reject**:

- automatic durable memory from conversation or screen capture;
- a vector database, knowledge graph, prompt, or UI cache as truth;
- confidence as a substitute for admission;
- TTL as silent deletion without an auditable disposition;
- memory admission that also creates an Open Loop, Outcome, Evidence, Verification, or Acceptance record;
- re-importing deleted content from stale devices, indexes, backups, or integrations.

**Unknown:** Omi's docs explicitly acknowledge staged convergence and gated gaps in some deletion-cascade paths. This dossier therefore treats its canonical memory material as a valuable design reference, not proof of complete production enforcement.

## 5. Storage, local custody, retrieval, and deletion

### What Omi does

**Observed:** The root architecture connects clients to a Python backend using Firebase/Firestore, Redis, speech-to-text providers, and LLM services. The macOS quick start connects to Omi's cloud backend unless a developer runs the backend separately. See [`README.md`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/README.md).

**Observed:** Omi's macOS Rewind subsystem stores a local GRDB database and screenshot/video files. It performs OCR through Apple's Vision framework, saves screenshot frames locally, and records extracted text and embeddings in its local database. See [`RewindDatabase.swift`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/desktop/macos/Desktop/Sources/Rewind/Core/RewindDatabase.swift#L1-L167), [`RewindStorage.swift`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/desktop/macos/Desktop/Sources/Rewind/Core/RewindStorage.swift#L18-L161), and [`RewindOCRService.swift`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/desktop/macos/Desktop/Sources/Rewind/Core/RewindOCRService.swift#L104-L139).

**Observed:** Capture exclusion is generation-fenced and waits for in-flight persistence boundaries so excluded application frames cannot cross into storage after a policy change. See [`RewindCaptureExclusion.swift`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/desktop/macos/Desktop/Sources/Rewind/Core/RewindCaptureExclusion.swift#L70-L166).

**Reported:** Omi's product page says conversations may be stored locally or in the cloud, offers manual local mode, export, and deletion controls, and claims encryption in transit and at rest. Its privacy policy is more specific: service data may be stored in Firebase/Google Cloud, audio in Cloud Storage, embeddings in Pinecone, and short-lived cache data in Redis; local phone caches are not described as canonical custody. The policy also says conversation deletion removes its transcript, associated data, and stored audio.

**Inference:** “Omi is local-first” is too broad. It has a valuable locally stored desktop screen-history subsystem and reported local/on-phone modes, while its wider account, memory, conversation, search, and integration architecture can use cloud systems. Waldo should adopt particular local mechanisms, not Omi's overall custody topology.

### Waldo decision

**Adopt**:

- local OCR/embedding where technically suitable and explicitly authorized;
- capture exclusion checked again at the final persistence boundary;
- owner and policy generations so in-flight work cannot write after revoke/account switch;
- deletion tombstones and cleanup jobs separated but fenced;
- retrieval indexes as disposable projections that hydrate through authority;
- explicit retention, pause, revoke, export, and deletion controls.

**Adapt:** All Waldo canonical state remains in the Kennel daemon's SQLite ownership boundary under `~/.kennel`. Any future large local blobs should be referenced by canonical records and governed by the same daemon authority, retention policy, audit, and deletion receipt. Local model and index processes may compute projections; they may not become additional writers of product truth.

**Reject**:

- Firebase, Pinecone, Redis, or a per-feature database as necessary Waldo architecture;
- a second desktop coordinator database;
- “stored locally” as a blanket privacy statement when any derived content, prompt, embedding, telemetry, or integration payload leaves the device;
- capture that begins before exclusion and revoke paths are proven at the persistence boundary.

## 6. Agents, apps, plugins, and integrations

### What Omi does

**Observed:** Omi apps can modify prompts, process memories, receive realtime transcript or audio events, expose chat tools, call webhooks, and send notifications. See [`Introduction.mdx`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/docs/doc/developer/apps/Introduction.mdx#L7-L140), [`Integrations.mdx`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/docs/doc/developer/apps/Integrations.mdx#L7-L69), and [`Notifications.mdx`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/docs/doc/developer/apps/Notifications.mdx#L158-L208).

**Observed:** A memory-creation webhook can receive broad conversation data, including transcript, summary, action items, and metadata. Realtime webhooks have delivery and deduplication guidance.

**Observed:** Omi's integrations invariant prefers functional probes over static “Connected” labels and requires tokens and personal data to be stripped from diagnostic traces. See [`integrations.md`](https://github.com/BasedHardware/omi/blob/b056ecba3bdaa4f81940505cabd1b72857ab632d/docs/product/invariants/integrations.md).

**Reported:** The official marketplace spans productivity, conversation insight, health, relationship, persona, and integration apps. Marketplace presence does not prove that an app's claims, security, or data handling have been independently verified.

### Waldo decision

**Adopt**:

- an inspectable capability manifest for every provider, app, agent, and integration;
- functional connection probes with current status, last checked time, and a clear repair/revoke action;
- bounded input/output schemas, delivery IDs, idempotency keys, retries, and dead-letter visibility;
- secret and personal-data redaction in traces;
- provenance on every external result admitted into Waldo.

**Adapt:** Apps may propose actions, candidates, or evidence, but the daemon must validate capability, owner, responsibility space, contract revision, and authority receipt before an effect. Home and Work consume the same intake/Q&A and authority contracts. An app install is not consent to all event classes; disclosure should be event-specific and payload-minimized.

**Reject**:

- full conversation/transcript delivery as a default integration payload;
- prompt text as the only security boundary;
- plugin installation as authority to mutate canonical memory, Outcomes, Open Loops, evidence, or acceptance;
- branding each provider or specialist agent as another assistant identity;
- static “connected” state without a functional probe.

## 7. Operational trade-offs

| Mechanism | Omi benefit | Omi/Waldo risk | Waldo decision |
|---|---|---|---|
| Continuous audio/screen capture | Rich recall and proactive context | Consent, bystander data, storage growth, false extraction, security, and high first-run friction | **Adapt as required capability**, never default-on premise; per-modality grants, visible state, bounded retention, and revocation |
| Fixed permission onboarding | Predictable setup coverage | Front-loads trust cost before value; permissions are broader than the first outcome | **Adapt** to just-in-time purpose-bound grants |
| Model proposal + deterministic router | Flexible language with reproducible policy | Router can become a hidden second product ontology | **Adopt pattern**, map to Waldo contracts |
| One-shot decision receipt | Replay protection and clear effect boundary | Requires careful reconciliation after unknown outcomes | **Adopt** |
| Derived action queue | One attention view without duplicating source state | Easy to confuse operational alerts with human responsibilities | **Adapt** as `AttentionItem`, never canonical `OpenLoop` |
| Canonical memory promotion | Strong provenance, correction, and anti-resurrection | Considerable migration and operational complexity | **Adopt later**, behind Memory gate |
| Vector hydration through authority | Prevents stale/deleted index results from leaking | Extra read latency and dependency on canonical availability | **Adopt**, fail closed |
| Separate TypeScript coordinator DB | Fast local autonomy for Omi desktop | Violates Kennel's Go daemon/SQLite single-writer boundary | **Reject** |
| Broad webhooks/app marketplace | Extensibility and ecosystem leverage | Data exfiltration, unclear scopes, third-party quality variance | **Adapt** with capability/payload governance |
| Direct task completion | Fast familiar UI | Collapses task state, evidence, acceptance, and conscious loop disposition | **Reject** |

## 8. Concrete architecture nudges for Waldo

These are **Proposed**, not approved implementation tasks.

### 8.1 One shared intake contract

Define the intake/Q&A contract once in the daemon and consume it from both Home and Work:

```text
IntakeTurn
  -> InterpretationProposal (untrusted)
  -> ClarificationRequest | PlanProposal | InlineAnswer | Rejection
  -> AuthorityPreview
  -> AuthorityDecision
  -> DispatchReceipt
  -> RunBrief / Attempt
```

No Home-specific Q&A engine, no Work-specific router, and no UI-owned conversation truth.

### 8.2 Derived attention without premature persistence

After the Work spine exists, define a pure read model that derives `AttentionItem` records from existing Outcome/Attempt/Evidence/Verification/Acceptance facts. The initial Home UI may project those items without introducing canonical Open Loop or Memory tables. That creates a safe place to evaluate usefulness, provenance, ranking, and empty states before custody expands.

### 8.3 Open Loop confirmation membrane

When persistent Home is approved, require an explicit transition from an attention suggestion to `OpenLoop`. Store the source proposal and confirmation receipt. Keep operational status and user disposition separate. “Ready to Close” should propose a `LoopDisposition`; it must never silently write one.

### 8.4 Memory admission membrane

When durable Memory is separately approved, implement `MemoryCandidate` first, not “memory extraction.” Add lineage, counter-evidence, correction, expiry, revoke, deletion, regeneration, and audit tests before enabling any ambient source. Indexing comes after canonical admission and deletion semantics, not before.

### 8.5 Purpose-bound context packets

Build context from named sources and a declared purpose. Record the source IDs, policy generation, redaction summary, expiry, and access event. Prefer the smallest packet sufficient for the proposed plan. A provider receives only the material authorized for that attempt.

### 8.6 Reconciliation before retry

Bind effects to idempotency and authority receipts. If dispatch or an external action has an unknown outcome, reconcile before retry. Provider completion yields an `AgentSessionRef`/attempt fact and possible evidence—not Outcome acceptance or Open Loop closure.

## 9. Shared contract and ownership proposal

This matrix is for design coordination only; it does not assign code files or authorize implementation.

| Area | Canonical owner | Home/Work consumption rule |
|---|---|---|
| `ResponsibilitySpace`, `Outcome`, `ContractRevision`, `PlanRevision` | Kennel daemon, Work spine | Home reads projections and explicit links; it does not fork these types |
| `WorkUnit`, `Attempt`, `AgentSessionRef`, `RunBrief` | Kennel daemon orchestration | Both surfaces show the same records under their responsibility context |
| `EvidenceItem`, `VerificationRun`, `AcceptanceDecision` | Kennel daemon outcome lineage | Home cannot infer acceptance from activity or provider completion |
| Intake, clarification, plan proposal, authority decision, dispatch receipt | One shared daemon service/API | Home and Work are clients of one contract |
| `OpenLoop`, `LoopDisposition`, `ResponsibilityLink` | Parallel daemon Home/responsibility service | Persist after the necessary shared contracts are ratified under ADR 0004 |
| `MemoryCandidate` and admitted memory lineage | Future daemon Memory service | Separate admission/evaluation gate; no screen/transcript auto-truth |
| Attention, brief, catch-up, “ready to close” | Derived read models | UI displays projections; it never owns canonical truth |

The non-overlapping file/API ownership matrix is now recorded in the [Home/Personal Agent design](../superpowers/specs/2026-08-21-home-personal-agent-memory-design.md); each code plan must claim its exact subset before editing.

## 10. Phase-boundary recommendation

The earlier research presented three viable strategies:

1. **Recommended — parallel research/design, sequential custody.** Work implements and evaluates the canonical Outcome spine. Home designs and may later project read-only attention from stable facts. Persistent Open Loop waits for shared contracts; durable Memory waits for a separate gate. This maximizes learning without creating competing authorities.
2. **Strict sequence.** Freeze Home UI and data design until the Work slice is evaluated. This minimizes rework but delays learning about Home comprehension and usefulness.
3. **Explicit architecture change.** Persist Home/Open Loop or Memory in parallel with Work. This offers faster breadth but increases migration, ontology, and single-writer risk. It must be recorded as an ADR change that intentionally overturns the accepted phase boundary; it must not happen through incidental implementation.

ADR 0004 selects strategy 3 for Home/OpenLoop and candidate foundations, with an explicit ownership matrix and a single daemon writer. It does not select parallel durable Memory admission. Omi's historical store convergence remains evidence for preserving Waldo's staged custody and candidate/admission membrane.

## 11. Unknowns and questions that must remain open

- Which Omi invariant revisions are deployed across mobile, web, backend, and desktop is **Unknown**.
- Production reliability, latency, extraction precision, and deletion completion are **Unknown**; no benchmarks were accepted from marketing claims.
- Whether all integrations consistently enforce least-privilege event scopes is **Unknown**.
- The exact completeness of Omi's historical-memory migrations and deletion cascades is **Unknown**; its docs describe staged/gated work.
- Whether Omi's manual local mode covers every derived prompt, embedding, recap, memory, telemetry, and integration path is **Unknown**. The official privacy policy describes substantial cloud storage and processing.
- Omi's terminology—memory, task, open loop, outcome, agent—must not be assumed semantically equivalent to Waldo's accepted ontology.

## 12. Final Adopt / Adapt / Reject register

### Adopt

- model interpretation as untrusted proposal;
- deterministic policy owner and one-shot, snapshot-bound decision receipt;
- effect idempotency plus reconciliation before retry;
- derived, bounded attention views with honest empty states;
- one memory admission authority, canonical lineage, correction, tombstones, and anti-resurrection;
- retrieval hydration through canonical authority;
- generation fences at capture, persistence, retrieval, and deletion boundaries;
- capability manifests, functional probes, redacted traces, and provenance-bearing external results.

### Adapt

- Omi Short-term → Waldo `MemoryCandidate`, with explicit admission/correction/revoke semantics;
- Omi action queue/open-loop snapshots → Waldo `AttentionItem` projections, not durable responsibility;
- Omi permission progression → plan-triggered, purpose-bound, revocable authority grants;
- Omi coordinator decision pattern → one shared Home/Work intake and authority contract in the Go daemon;
- Omi local Rewind mechanics → optional later local capture capability, only after exclusion/deletion gates;
- Omi apps/webhooks → event-specific, payload-minimized integrations under daemon capability policy.

### Reject

- continuous capture or broad permissions as first-run premise;
- separate assistant identities or a second coordinator/store authority;
- cloud dependencies or per-feature stores as Waldo's canonical core;
- auto-admitting transcripts, summaries, screen observations, or model inferences as durable truth;
- direct task checkbox completion as Outcome acceptance or Open Loop closure;
- full-conversation webhook delivery by default;
- indexes, graphs, prompts, provider state, or UI caches as canonical truth;
- implementing persistent Home/Open Loop outside ADR 0004's ownership/shared-contract rules, or durable Memory before its separate gate is explicitly approved.

## Conclusion

The proper Omi lesson is governance through implementation detail. Its newer code makes route decisions consumable once, keeps candidate records reviewable, derives attention instead of duplicating it, hydrates search through authority, and uses tombstones and generations against resurrection. Waldo should adopt those disciplines while preserving its own sharper separations: responsibility is not activity, an Open Loop is not a runtime alert, memory is not an extraction, evidence is not verification, and completion is not acceptance.

No code, schema, API, persistence, release, or deployment change is authorized by this research.

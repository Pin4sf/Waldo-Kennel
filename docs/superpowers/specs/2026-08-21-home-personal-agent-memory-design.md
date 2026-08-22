# Waldo Kennel Home, Personal Agent, capture, and memory design

- **Status:** Approved written specification
- **Date:** 2026-08-21
- **Scope:** Personal Home, Open Loops, life-and-work capture, memory candidates, retrieval, and the shared boundary with Work orchestration
- **Implementation status:** Not shipped; this document does not authorize merge, push, release, or deployment
- **Decision record:** [ADR 0004](../../adr/0004-parallel-home-personal-agent-and-required-capture.md)

## 1. Positioning

Waldo is one user-owned personal agent spanning life and work. Kennel is its local Mac presence and governed execution harness, not a second assistant identity.

Home reduces the cost of remembering, reconstructing, and consciously closing responsibilities. Work turns an explicitly contracted Outcome into bounded execution, Evidence, Verification, and user Acceptance. Both consume one shared intake and context contract, but preserve separate responsibility and proof lineages.

The memory invariant is:

> Capture is a source. Memory is context. Responsibility, authority, runtime state, and proof remain separate.

## 2. Approved decisions

1. Personal Home and Work are parallel implementation lanes over one daemon-owned canonical model.
2. Desktop screen capture and desktop audio capture are required Personal Agent capabilities. They are not optional roadmap ideas.
3. Required capability does not mean forced activation. Capture starts only through an explicit `CaptureGrant` and always provides visible state, pause, exclusions, retention, revoke, export, and deletion.
4. A later Health-aware mobile, phone, or wearable presence extends Waldo into physical-world life capture through the same durable identity, source, admission, scope, and deletion contracts. It does not create another assistant or another canonical memory store; Health First remains recommended, not required.
5. Observations, OCR, transcripts, summaries, and inferred patterns become candidates. They never become durable personal truth automatically.
6. SQLite in the Kennel daemon is the sole canonical writer. Blobs, full-text indexes, embeddings, relationship indexes, Markdown, and provider state are subordinate projections or references.
7. `MemoryCandidate`, `OpenLoop`, `Outcome`, `AgentSessionRef`, `EvidenceItem`, `VerificationRun`, and `AcceptanceDecision` stay distinct.
8. The Work Local Focus Ledger keeps its original scope and evaluation contract. Parallel Home work does not add Home concepts to Work migrations or APIs.
9. Durable admitted Memory remains behind a separate admission, privacy, deletion, threat-model, and evaluation gate. The Home lane may implement observations, episodes, candidates, review, and retrieval fixtures before that gate.
10. The provisional Home v0 has five stable destinations: Today, Open Loops, Memory, Daily Close, and History. Catch Up is a contextual review flow launched from Today, not a sixth destination.
11. This information architecture is a buildable prototype contract, not a permanent visual design. Product use may redesign navigation, layout, and grouping without changing canonical responsibility, provenance, admission, or authority boundaries.
12. Quick Capture and governed ambient capture are shared Waldo capabilities, not Today-owned behavior. Today presents the expanded explicit composer; enabled ambient capture continues independently of the open destination.
13. Daily Close closes a review interval, not a responsibility. History is a derived continuity projection, not a second event store or a product rendering of raw `change_log` rows.

## 3. Product scope

### Build in the Home and Personal Agent lane

- first-run choice between Home and Work without creating either silently;
- shared adaptive intake and clarification;
- Today / Morning Brief;
- Catch Up;
- Quick Capture;
- confirmed Open Loops and Open Loop detail;
- Daily Close and exact tomorrow re-entry;
- continuity-first History with optional activity and audit drill-down;
- Waiting, Ready to Close, close, release, reopen, and transfer dispositions;
- explicit Home-to-Work linking;
- screen and audio capture controls and ingestion;
- correctable Context Episodes;
- provenance-bearing Commitment and Memory Candidates;
- candidate review, correction, rejection, expiry, revocation, and deletion behavior;
- purpose-bound retrieval packets for Home and Work;
- local evaluation and failure injection.

### Do not build in this lane yet

- automatic durable admission from ambient capture;
- automatic Open Loop, Outcome, Acceptance, or closure creation;
- autonomous message sending, purchasing, publishing, deploying, or other remote effects;
- personality scoring, productivity scoring, or a hidden whole-person profile;
- raw health measurements in general memory;
- hosted synchronization or dual canonical writers;
- mobile or wearable implementation before the desktop source contract passes its privacy and deletion gates;
- automatic skill promotion from traces.

## 4. User experience

### Provisional v0 information architecture

The first working Home prototype optimizes for learning rather than visual finality. It must make the complete loop usable and truthful, while keeping layout, styling, density, and grouping easy to replace after dogfood.

The desktop Home shell provides:

- a stable Home/Work destination switch;
- direct navigation to Today, Open Loops, Memory, Daily Close, and History;
- a persistent Quick Capture action;
- one main content region;
- one shared source/provenance inspector opened only when the user requests detail;
- explicit empty, partial, capture-off, stale, and offline states shared across the five destinations.

`/home` opens Today. The other stable destinations use `/home/open-loops`, `/home/memory`, `/home/daily-close`, and `/home/history`. Catch Up is restartable Today state rather than permanent navigation. On desktop v0, navigation stays in the adaptive sidebar; a floating mobile-style bottom bar is not part of this slice.

The shell does not add widgets, personalization, scoring, a graph canvas, a Kanban board, or user-configurable dashboards. Those would add structure before real use demonstrates a need.

### Five destination contracts

| Destination | Primary question | Minimum v0 content | Explicit exclusion |
| --- | --- | --- | --- |
| **Today** | What matters now? | Expanded Quick Capture, concise brief, material Catch Up entry, Open Loops needing attention, recent Outcome facts, source freshness/gaps, and Daily Close entry. | Raw activity feed, every responsibility, or background-agent controls. |
| **Open Loops** | What confirmed responsibility remains unresolved? | Searchable list, Waiting and Ready to Close views, owner, meaning, trigger, recheck condition, provenance, related Work Outcomes, and explicit dispositions. | Inferred TODOs or provider/session completion presented as closure. |
| **Memory** | What context is Waldo proposing or permitted to retain? | Candidate review queue, source spans, uncertainty/sensitivity, correct, reject, defer, and admitted-revision state only when the durable Memory gate has passed. | An automatic profile, personality score, or ambient observations presented as admitted truth. |
| **Daily Close** | What became true, what remains open, and where should tomorrow resume? | Activity-backed reconstruction, known gaps, source inspection, explicit corrections, unresolved responsibility references, optional note, and selected re-entry. | Inbox-zero pressure, streaks, productivity scores, or implicit Open Loop/Outcome closure. |
| **History** | How did the current situation become true? | Continuity timeline of decisions, corrections, candidate decisions, Open Loop dispositions, Daily Close receipts, responsibility links, and re-entry. Activity and audit are optional filters or drill-down. | Raw CDC/change-log rows as user-facing history or a duplicate canonical event store. |

Selecting an item on any destination opens the same provenance inspector. Closing the inspector returns focus and scroll position to the exact originating item.

### Natural flow

The design is direct navigation, not a wizard:

```text
Launch -> Today
  -> Quick Capture at any time
  -> Catch Up only when a material candidate needs judgment
  -> confirm Open Loop, keep note, correct, dismiss, defer, or draft/link Work

During the day -> Open Loops and Memory remain directly accessible

End of day -> Daily Close
  -> review activity-backed reconstruction and known gaps
  -> append corrections or native responsibility dispositions
  -> choose exact tomorrow re-entry

Whenever explanation is needed -> History -> shared provenance inspector
```

The user may skip Catch Up, open Daily Close at any time, or navigate directly to any stable destination. An empty Catch Up queue is a successful state and does not leave an empty permanent screen.

### Capture placement

Capture has two distinct paths:

1. **Explicit Quick Capture:** a persistent Home-shell action and expanded composer on Today. Text is the v0 input. The command enters shared adaptive intake and may remain a note or propose an Open Loop, correction, dismissal, or draft Work Outcome. The UI preserves unsent input across route changes and recoverable failures.
2. **Governed ambient capture:** independently granted screen, audio, application, or structured-source ingestion. It continues independently of the open Home destination, exposes active/paused/denied/stale/failed state, and never treats an unobserved interval as proof that nothing happened.

Today, Catch Up, Daily Close, Memory, and History consume provenance-bearing projections from the capture/source plane. They do not directly write raw observations or bypass the daemon. When ambient capture is off, explicit capture, confirmed responsibilities, and trusted Work facts keep Home useful.

### First run and shared intake

The first screen asks what the user needs help with now and offers Home or Work. Work remains the recommended dogfood entry, but Home is a complete peer and may be selected immediately.

Both destinations use one adaptive intake service:

1. accept the user's statement or source candidate;
2. determine the `ResponsibilitySpace` and proposed responsibility kind;
3. ask only a material clarification that changes meaning, ownership, success, timing, authority, or disposition;
4. present the smallest sufficient proposal;
5. obtain explicit confirmation and any required authority;
6. create the appropriate canonical object or dismiss/defer without inventing responsibility.

Home may produce an explicit note, confirmed `OpenLoop`, draft Work `Outcome`, correction, or dismissal. Work may produce a draft `Outcome` and `ContractRevision`. The UI and models do not implement separate Q&A systems.

### Today / Morning Brief

Today is a calm projection over trusted current facts:

- confirmed Open Loops needing attention;
- Waiting items whose recheck condition is met;
- Ready to Close suggestions;
- active and recently accepted Outcomes;
- explicit notes and Quick Captures;
- correctable candidate episodes or commitments;
- source freshness and known capture gaps.

Activity volume, screen time, or model confidence does not determine priority by itself.

### Catch Up

Catch Up processes one material item at a time. Each card exposes source, uncertainty, why it matters now, and the smallest useful decision:

- dismiss;
- correct;
- keep as a note;
- confirm an Open Loop;
- connect to existing Work;
- create a draft Work Outcome;
- defer with a recheck condition.

### Quick Capture

Quick Capture accepts text first and may later accept voice, image, file, or shared content. Explicit user intent makes it the strongest candidate source. It still asks for confirmation before creating an Open Loop or Outcome unless the user issued an unambiguous direct command.

### Daily Close and re-entry

Daily Close combines a time-bounded `DailyCloseProjection` with optional source inspection. The projection contains trusted facts, confirmed responsibilities, explicit notes, activity episodes, capture gaps, and proposed changes. It may suggest that an Open Loop is Ready to Close, but the user must create the native `LoopDisposition`; completing Daily Close cannot create that disposition implicitly.

Finishing the review appends an immutable `DailyCloseReceipt` containing the local date, reviewed-through watermark, acknowledged gaps, optional user note, and references selected for tomorrow's re-entry. The receipt does not copy or mutate the referenced Open Loops or Outcomes. Re-entry resolves those references against their current canonical state and discloses missing, superseded, released, or unavailable sources.

### History

History consumes a derived `ContinuityEvent` read model assembled from native facts. Its default ordering favors user decisions and continuity changes over activity volume. Supported event families are explicit captures, corrections, candidate decisions, Open Loop dispositions, ResponsibilityLink changes, Outcome facts permitted in Home, Daily Close receipts, and re-entry.

Activity episodes and audit metadata are filters or drill-downs. The read model remains rebuildable and does not become a new authority source. A displayed event always links back to its canonical object and provenance when available.

### Open Loop detail and closure

An Open Loop shows owner, meaning, provenance, current state, trigger, recheck condition, related responsibilities, and disposition history. Waldo may propose Ready to Close from source changes; only the user closes, releases, reopens, transfers, or supersedes it.

### Home to Work

An Open Loop may be connected to a new or existing Work Outcome through an immutable `ResponsibilityLink`. The link records provenance, reason, creator, created time, and optional ended time/reason. It never moves, merges, verifies, accepts, or closes either side.

Work execution begins only after the destination Project, Outcome contract, plan, and authority are defined through the shared intake contract.

## 5. Required capture system

### Capture tiers

Waldo uses a selective hybrid rather than one undifferentiated recording stream:

1. **Explicit capture:** Quick Capture, user-selected text, voice note, file, or screenshot.
2. **Structured local facts:** Outcome events, application integrations, files, Git, terminal, browser, calendar, and communication metadata.
3. **Desktop visual context:** periodic or event-triggered screen frames, OCR, deduplication, and local episode segmentation.
4. **Desktop audio context:** microphone and system audio for meetings, calls, conversations, and other granted windows.
5. **Future physical-world presence:** phone or wearable capture using the same Waldo source contract.

Screen and audio support are required for the Personal Agent milestone. Each modality is activated independently and may be paused or revoked independently.

### `CaptureGrant`

Every capture source requires a durable grant containing:

- modality and device;
- included and excluded applications, people, spaces, and time windows;
- purpose and allowed destination `ResponsibilitySpace`s;
- local-versus-provider processing route and disclosure;
- raw and derived retention limits;
- sensitivity class and bystander policy;
- visible active, paused, denied, stale, and failed states;
- revoke, export, and deletion behavior.

Password managers, authenticators, private browsing surfaces, and user-defined excluded applications are blocked before capture where the platform permits it. Capture gaps remain explicit; Waldo never infers that nothing happened during an unobserved interval.

### Ingestion

```text
CaptureGrant
  -> SourceArtifact / SourceSegment
  -> local deduplication, segmentation, and redaction
  -> Observation
  -> ContextEpisode
  -> CommitmentCandidate and/or MemoryCandidate
  -> explicit responsibility confirmation or memory admission
```

Raw screen and audio material is short-lived by default. Retention is defined by age and storage budget. Derived text does not survive source deletion merely because it is smaller.

## 6. Canonical contracts

### Shared adaptive intake

| Object | Responsibility |
| --- | --- |
| `IntakeSession` | One Home or Work understanding flow with purpose, source, space, state, and current revision. |
| `IntakeTurn` | User statement, Waldo question/proposal, or user response with actor and provenance. |
| `ClarificationRequest` | One material question with reason, recommendation, alternatives, and consequence of deferral. |
| `ResponsibilityProposal` | Proposed note, Open Loop, Outcome, link, correction, or dismissal; never canonical by itself. |
| `PlanProposal` | Smallest sufficient direct action or Work plan. |
| `AuthorityProposal` | Exact capabilities, effects, disclosure, budget, and expiry requested before action. |

Home and Work controllers consume this contract. Neither owns a separate transcript parser, question generator, or proposal state machine.

### Home and source objects

| Object | Responsibility |
| --- | --- |
| `PersonalHome` | Home `ResponsibilitySpace` and policy root. |
| `OpenLoop` | Confirmed unresolved responsibility with owner, trigger, recheck, and lifecycle. |
| `LoopDisposition` | Immutable confirm, close, release, reopen, transfer, or supersede decision. |
| `ResponsibilityLink` | Explicit immutable many-to-many Home/Work lineage. |
| `CaptureGrant` | User-governed permission and retention contract for a source modality. |
| `SourceArtifact` | Original imported or captured source identity and retention state. |
| `Observation` | Untrusted typed fact derived from a source segment. |
| `ContextEpisode` | Correctable grouping of observations with time, actors, source gaps, and derivation version. |
| `DailyCloseProjection` | Rebuildable, time-bounded read model for review; never canonical responsibility truth. |
| `DailyCloseReceipt` | Immutable record of the reviewed-through point, acknowledged gaps, optional note, and selected re-entry references; never a closure disposition. |
| `ContinuityEvent` | Rebuildable History read model derived from canonical facts and source lineage. |

### Memory objects

| Object | Responsibility |
| --- | --- |
| `MemoryCandidate` | Proposed claim with source spans, type, valid-time proposal, uncertainty, sensitivity, and review need. |
| `AdmissionDecision` | Immutable accept, edit, reject, or defer decision with actor, policy, and reason. |
| `MemoryRecord` | Stable identity for one conceptual memory across revisions. |
| `MemoryRevision` | Immutable admitted content with valid time, recorded time, scope, expiry, and provenance. |
| `CounterEvidence` | Source-backed challenge that may propose a correction without overwriting. |
| `Revocation` | Immediate prohibition on future use while permitted audit lineage remains. |
| `DeletionTombstone` | Content-free identity/digest and generation fence preventing stale resurrection. |
| `RetrievalReceipt` | Purpose, caller, eligible spaces, included/excluded revisions, generations, and degradation. |

### Memory types

- explicit core boundaries and stable preferences;
- episodic events;
- semantic facts, entities, and relationships;
- scoped preferences;
- decisions, rationale, and supersession;
- source resources such as documents, transcripts, images, and audio;
- procedural candidates that require separate evaluation before becoming executable skills.

Prospective responsibility belongs to Open Loop or Outcome. Runtime checkpoints belong to Attempt. Proof belongs to Evidence, Verification, and Acceptance.

## 7. Admission and lifecycle

### Admission defaults

- An explicit “remember this” command may directly create an admitted revision after scope and sensitivity validation.
- Accepted canonical product facts remain in their native domain and may be projected into context; they are not duplicated as personal memory.
- Ordinary user statements become high-priority candidates unless an approved narrow policy admits that class.
- Ambient observations, third-party statements, assistant output, summaries, sentiment, behavior patterns, relationship inference, identity inference, health interpretation, and predicted intention always require review.
- A model or second reviewing model is not the user.

### State

```text
MemoryCandidate:
  proposed -> needs_review -> admitted
            -> edited -> admitted
            -> rejected | deferred | expired | withdrawn

MemoryRevision:
  current -> superseded | disputed | expired | revoked | deleted
```

Corrections append revisions and preserve valid-time history. Recent ingestion does not automatically outrank an older explicit statement. Conflicts remain attributable and are surfaced when consequential.

### Deletion

Deletion is complete only when stale sources, jobs, indexes, summaries, checkpoints, and imports cannot recreate the deleted content:

1. write the content-free tombstone and advance the canonical generation;
2. make normal reads reject older generations immediately;
3. remove or cryptographically discard raw payloads and governed keys;
4. remove FTS, embedding, relation, cache, and projection entries;
5. regenerate dependent summaries and packets;
6. propagate deletion to provider copies when supported and disclose unresolved copies;
7. retain only content-free propagation receipts permitted by policy;
8. test replay against the deletion generation.

## 8. Storage and retrieval

SQLite is canonical and the daemon is the only writer. Recommended subordinate storage:

- encrypted content-addressed blobs for retained sources;
- SQLite FTS for lexical retrieval;
- a local embedding projection keyed by revision and generation when evaluation justifies it;
- typed SQLite relationship tables;
- daemon-generated Markdown for inspection and export.

A separate graph database is not part of the initial slice. It may be added only as a rebuildable projection when multi-hop or temporal evaluation demonstrates a material deficit.

Retrieval follows:

```text
purpose + caller + authority
  -> eligible people, spaces, source classes, and sensitivity
  -> lexical + semantic + temporal + typed-relation candidates
  -> canonical SQLite hydration and generation check
  -> remove unauthorized, stale, superseded, expired, revoked, or deleted items
  -> rank for purpose, relevance, validity, source strength, uncertainty, and diversity
  -> minimized provenance-bearing RetrievalPacket + RetrievalReceipt
```

The context precedence is: current user statement; current contract and decisions; current grants and policy; verified dependency facts; admitted purpose-relevant memory; then explicitly marked candidate context. Retrieved text cannot grant authority or override a contract.

## 9. Work and orchestration boundary

Each Work `Attempt` receives a versioned `RunBrief`. Personal memory is read-only context to the acting agent. The agent may append a `MemoryCandidate` or procedural candidate through the daemon API; it cannot edit an admitted revision, prompt policy, or skill directly.

Runtime recovery uses `RuntimeCheckpoint`, `EffectIntent`, and `EffectReceipt` under `Attempt`. Checkpoints, transcripts, compacted summaries, provider completion, commits, and checks do not become durable personal memory or proof by implication.

## 10. Lane and file ownership

Before code begins, the Work and Home lanes claim non-overlapping files and API operations. Shared contracts are changed only through a named integration owner or a deliberately coordinated contract PR.

| Area | Work lane owns | Home / Personal Agent lane owns | Shared integration rule |
| --- | --- | --- | --- |
| Domain | Outcome, ContractRevision, PlanRevision, WorkUnit, Attempt, Evidence, Verification, Acceptance | PersonalHome, OpenLoop, LoopDisposition, DailyCloseReceipt, CaptureGrant, SourceArtifact, Observation, ContextEpisode, memory objects | ResponsibilitySpace, ResponsibilityLink, intake contracts, RunBrief references require coordinated review |
| Services | Outcome lifecycle, execution, verification, acceptance, recovery | Home lifecycle, capture, candidate admission/review, retrieval | Shared intake and context compilation are implemented once in a neutral service package |
| Storage | Work lineage migrations and queries | Home, Daily Close receipt, source, episode, candidate, memory migrations and queries | Separate additive migrations; neither edits the other's merged migration |
| HTTP API | Work Outcome and Attempt operations | Home, Open Loop, capture, candidate, memory operations | Shared DTOs and routes land through one contract PR and regenerate OpenAPI/types together |
| Frontend | Work destination and execution/review projections | Home destination, Today, Catch Up, Open Loops, Memory, Daily Close, History, capture controls, candidate review | Shared shell, intake components, provenance, and authority UI have one owner per PR |
| Evaluation | Outcome conformance, recovery, effect and proof gates | candidate precision, capture/privacy, correction, deletion, retrieval, Home utility | Cross-lane Home-to-Work and RunBrief tests are integration-owned |

No frontend component, provider adapter, capture worker, or memory library owns canonical truth.

### Concrete file and API reservations

The implementation plan may split files more finely, but it must preserve these reserved prefixes and touchpoints:

| Owner | Reserved backend files/packages | Reserved API prefix | Reserved frontend files/directories |
| --- | --- | --- | --- |
| Work | `backend/internal/domain/{outcome,contract,plan,workunit,attempt,evidence,verification,acceptance}*.go`; `backend/internal/service/outcome/**` and future Work execution/proof packages; matching Work query files | `/api/v1/outcomes/**`, `/api/v1/work-units/**`, `/api/v1/attempts/**`, `/api/v1/verification-runs/**` | future `frontend/src/renderer/routes/work/**` and `components/work/**`; existing donor Outcome components only when claimed by the Work plan |
| Home / Personal Agent | `backend/internal/domain/{home,openloop,daily_close,continuity,capture,source,observation,memory}*.go`; `backend/internal/service/{home,capture,memory}/**`; matching `home.sql`, `open_loops.sql`, `daily_close.sql`, `capture.sql`, `observations.sql`, and `memory.sql` query files | `/api/v1/home/**`, `/api/v1/open-loops/**`, `/api/v1/capture-grants/**`, `/api/v1/sources/**`, `/api/v1/observations/**`, `/api/v1/memory-candidates/**`, `/api/v1/memory/**` | future `frontend/src/renderer/routes/home/**`, `components/home/**`, `components/capture/**`, `components/memory/**`; OS capture bridge only under a separately claimed `frontend/src/main/capture/**` |
| Shared integration owner | `backend/internal/domain/{responsibility,intake,responsibility_link,run_brief_ref}*.go`; `backend/internal/service/intake/**`; `backend/internal/httpd/controllers/dto.go`; `backend/internal/httpd/apispec/specgen/build.go`; route registration; shared CDC semantics | `/api/v1/responsibility-spaces/**`, `/api/v1/intake/**`, `/api/v1/responsibility-links/**`; shared schemas referenced by both lane prefixes | `frontend/src/renderer/components/IntakeFields.tsx`, `OutcomeIntakePanel.tsx`, future `components/intake/**` and `components/responsibility/**`; `frontend/src/api/schema.ts` |

The integration owner also coordinates generated `backend/internal/httpd/apispec/openapi.yaml` and `frontend/src/api/schema.ts`; both generated artifacts land with their source DTO/controller change. Parallel branches must not hand-edit either artifact or resolve generated conflicts by choosing one side.

SQLite migration numbers are assigned by the integration owner before a code slice begins. Each lane uses a separate additive migration and query file; neither lane reserves a number speculatively, edits an already merged migration, or shares one migration merely to avoid coordination.

## 11. Parallel execution sequence

The two lanes may proceed concurrently, with explicit dependency gates:

### Shared contract track

1. Ratify `ResponsibilitySpace`, `ResponsibilityLink`, shared intake DTOs, source references, and RunBrief memory references.
2. Land neutral domain and API contracts through coordinated PRs.
3. Maintain route/spec parity and one daemon writer.

### Work track

Continue the Local Focus Ledger five-stage Outcome milestone unchanged. It must not add speculative Home or Memory tables to make its own screens work.

### Home and Personal Agent track

1. Home shell plus read-only fixture projections.
2. PersonalHome, OpenLoop, LoopDisposition, and Quick Capture vertical slice.
3. Today, contextual Catch Up, Open Loop detail, Ready to Close, Daily Close/receipt, continuity History, and restart/re-entry.
4. ResponsibilityLink and shared-intake Home-to-Work integration.
5. CaptureGrant and required desktop screen/audio ingestion into SourceArtifact, Observation, and ContextEpisode.
6. MemoryCandidate review, correction, expiry, revocation, deletion-generation fixtures, and purpose-bound retrieval.
7. Durable Memory admission only after its safety and evaluation gate passes.

Each step owns domain, storage, CDC, service, API, UI, recovery, and evaluation for its user-visible truth boundary. Parallel work does not authorize duplicate Q&A, a second SQLite writer, or UI-only persistence.

## 12. Failure and privacy behavior

| Failure | Required behavior |
| --- | --- |
| Capture denied or paused | Home remains useful; source state and gaps are visible without pressure. |
| Capture worker crashes | Durable ingestion state distinguishes incomplete from empty; retries are idempotent. |
| OCR/transcription/model fails | Preserve source coverage and failure; do not report that nothing happened. |
| Source is sensitive or excluded | Block before capture where possible; otherwise redact/delete before downstream processing. |
| Candidate conflicts with admitted memory | Preserve both lineages and request correction when material. |
| Indexing fails | Canonical admission remains; lexical retrieval and explicit degradation continue. |
| Retrieval returns stale data | Canonical hydration rejects stale generations. |
| User revokes or deletes | Stop use immediately, advance generation, scrub every derived layer, and report propagation. |
| Work requests Home context | Apply purpose and space policy; require one-run or standing disclosure grant where needed. |
| Provider session ends | Preserve Attempt state; never close an Outcome or Open Loop automatically. |
| Daily Close has partial or stale sources | Permit a review with acknowledged gaps; never present the interval as complete. |
| A History source is unavailable | Preserve the continuity event and disclose that its source cannot currently be inspected. |

## 13. Evaluation gates

The Home and Personal Agent lane measures:

- all five stable routes remain directly navigable and preserve active navigation state;
- Catch Up opens from and returns to the exact Today context without becoming permanent navigation;
- Quick Capture remains available across Home routes and preserves an unsent draft across recoverable navigation and failures;
- every destination renders truthful populated, empty, partial/capture-off, and offline states without treating unavailable data as empty;
- the shared provenance inspector returns focus and scroll position to the originating item;
- finishing Daily Close cannot close, release, accept, or otherwise mutate a referenced responsibility without its native explicit command;
- History rebuilds from canonical facts and never requires a second event store;
- candidate precision and user review burden;
- provenance completeness and source-span accuracy;
- temporal updates and contradiction behavior;
- correction propagation latency;
- deletion non-resurrection after restart, replay, reindex, regeneration, and backup restore;
- zero cross-person, cross-purpose, cross-space, and health-context leakage;
- zero memory-created authority, responsibility, Evidence, Verification, or Acceptance;
- capture coverage with visible gaps and accurate modality attribution;
- median time to regain personal or work context;
- false Open Loop and false Ready to Close rates;
- local storage, energy, model cost, and p50/p95 ingestion/retrieval latency.

Public long-memory benchmarks may be compatibility baselines. Waldo release gates use work-and-life fixtures containing corrections, ambiguous speakers, third-party claims, deleted sources, changing preferences, capture gaps, prompt injection, provider failure, and Home-to-Work disclosure.

## 14. Costs and evolution

Required capture increases privacy risk, storage pressure, model cost, battery use, and bystander responsibility. The architecture pays that cost through explicit grants, local preprocessing, short raw retention, source gaps, bounded disclosure, and deletion verification.

Parallel Home and Work execution shortens product learning but adds coordination risk. The file/API ownership matrix, coordinated shared-contract PRs, and prohibition on duplicate intake or writers are the chosen controls.

At 100× personal history, retained media and derived-index rebuild time fail before SQLite transactional throughput. Retention, deduplication, compaction, incremental projection checkpoints, and storage budgets precede any distributed or hosted memory architecture.

The evolution path is:

```text
parallel Work + Home foundations
  -> required desktop screen/audio source plane
  -> candidate-only memory and retrieval evaluation
  -> governed durable local Memory
  -> explicit identity-preserving durable Waldo attachment
  -> Health-aware mobile/phone/wearable presence on that same durable agent
```

Every later presence consumes the same canonical contracts and preserves one Waldo identity. ADR 0006 governs the final ecosystem: Kennel remains the desktop Work/execution presence, the Health-aware mobile app is the personal presence, Health stays recommended rather than required, and offline caches never become competing canonical writers.

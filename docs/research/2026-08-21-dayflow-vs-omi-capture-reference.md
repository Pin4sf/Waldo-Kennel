# Dayflow versus Omi capture reference for Waldo Kennel

**Status:** architecture research; no implementation authority

> **Architecture disposition:** ADR 0004 adopts desktop screen and audio capture as required Personal Agent capabilities with explicit per-modality activation. This report remains the evidence base; “optional” references describe user activation or the earlier phase plan, not the current product commitment.

**Research date:** 2026-08-21

**Dayflow official repository:** <https://github.com/JerryZLiu/Dayflow>

**Dayflow inspected revision:** current `main` and `v2.1.1`, [`86f5288d7267c9c8da1ab47e9c25ad1cd0b80046`](https://github.com/JerryZLiu/Dayflow/tree/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046), committed 2026-08-19 08:54:48 UTC

**Dayflow official product sources:** [product README](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/README.md), [privacy policy](https://www.dayflow.so/privacy/)

**Omi comparison source:** [`2026-08-21-omi-implementation-reference.md`](2026-08-21-omi-implementation-reference.md), pinned there to `b056ecba3bdaa4f81940505cabd1b72857ab632d`

**License:** Dayflow is [MIT licensed](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/LICENSE), copyright 2025 Jerry Liu.

## Executive finding

Waldo should not choose between a “Dayflow style” and an “Omi style” as one capture architecture.

- **Dayflow is the stronger screen-to-episode reference.** Its implemented core periodically captures the active Mac display, records local raw frames and idle metadata, groups frames into bounded batches, and derives a correctable work timeline. It does not establish a whole-life memory layer, intention, commitment, authority, evidence, or acceptance.
- **Omi is the broader life-capture reference.** It combines conversation/audio, mobile or wearable capture, desktop Rewind, integrations, and a cloud-capable memory system. Its valuable Waldo lessons are its capture exclusion fence and newer candidate/admission/authority mechanisms—not ambient capture as the product premise.
- **Waldo needs a layered hybrid:** trusted Kennel work facts and explicit Quick Capture, required governed Dayflow-like desktop context, and required governed Omi-like audio capture. Each modality still requires explicit activation and may begin with bounded meeting/call windows. Durable Memory remains behind a separate admission, correction, deletion, and evaluation gate.

The Minimi-like product outcome is not “record everything.” It is being able to recover the right context, notice a likely unresolved responsibility, and help the user re-enter or consciously close it without silently turning observation into truth. No public evidence currently establishes Minimi's claimed quality as a reproducible benchmark, so Waldo should evaluate that outcome directly rather than claim feature parity.

## Evidence discipline

Labels used throughout:

- **Observed** — present in source code or repository documentation at the pinned revision.
- **Reported** — claimed by an official product or privacy source but not verified in the inspected client source.
- **Inference** — a conclusion drawn from observed sources.
- **Proposed** — a Waldo/Kennel design recommendation, not shipped behaviour.
- **Unknown** — not established by the inspected primary sources.

The Dayflow repository is a client implementation snapshot. It cannot establish which binary is deployed, the behaviour of the closed Dayflow cloud backend, real-world accuracy, deletion from provider systems, or production reliability. Omi facts below are intentionally limited to the already completed pinned dossier rather than duplicating it.

## 1. What Dayflow actually captures

### 1.1 Screen substrate

**Observed:** Dayflow uses `SCScreenshotManager`, not a continuous video stream. It captures one JPEG of the selected display every 10 seconds by default, scales it to approximately 1080p, uses JPEG quality `0.85`, and includes the cursor. The selected display follows the display under the mouse with a low-frequency poll and debounce. See [`ScreenRecorder.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/ScreenRecorder.swift#L18-L37), [capture and persistence](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/ScreenRecorder.swift#L345-L469), and [`ActiveDisplayTracker.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/ActiveDisplayTracker.swift#L25-L35).

**Observed:** Each screenshot stores capture time, file path, file size, and seconds since hardware input. Dayflow stores the JPEG on disk and a row in SQLite. See [`StorageManager+Screenshots.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/StorageManager%2BScreenshots.swift#L8-L35).

**Observed:** The inspected capture path does not use Accessibility APIs, window-title APIs, or semantic app integrations. `NSWorkspace.frontmostApplication` is used to enforce the privacy blocklist. For activity interpretation, model prompts infer the active app from visible pixels or locally derived OCR hints. See [`RecordingPrivacyPreferences.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/RecordingPrivacyPreferences.swift#L105-L153) and the [Gemini transcription prompt](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/GeminiDirectProvider%2BTranscription.swift#L35-L54).

**Observed:** The README describes macOS's “Screen & System Audio Recording” permission, but the inspected recorder persists screenshots and idle metadata; it does not implement microphone, conversation, or system-audio recording. See the [README requirements](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/README.md#requirements) and [`ScreenRecorder.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/ScreenRecorder.swift#L1-L16).

**Inference:** Dayflow observes visible computer activity well enough to propose work episodes. It cannot see off-screen conversations, physical-world commitments, why the user did something, whether an observed draft was sent, or whether the user later changed their mind.

### 1.2 Ingestion volume

**Observed:** At the default 10-second interval, capture produces 6 frames/minute, 360 frames/hour, or approximately 2,880 frames during an eight-hour workday. A standard analysis batch spans 15 minutes—about 90 frames—and is split when the screenshot gap exceeds two minutes. Recent card generation uses a 45-minute lookback. See [`ScreenshotConfig`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/ScreenRecorder.swift#L20-L27), [`BatchingConfig`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/LLMTypes.swift#L154-L163), and [batch construction](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Analysis/AnalysisManager.swift#L602-L679).

**Observed:** Provider input volume differs by route:

- Gemini composites all screenshots in a 15-minute batch into a compressed 90-second video and uploads it. See [`GeminiDirectProvider+Transcription.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/GeminiDirectProvider%2BTranscription.swift#L630-L715).
- The Dayflow backend route base64-encodes every loadable screenshot and posts the batch to `/v1/dayflow/transcribe`. See [`DayflowBackendProvider.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/DayflowBackendProvider.swift#L399-L472).
- Codex samples approximately 15 frames and passes their local paths to the CLI. See [`CodexProvider+Transcription.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/CodexProvider%2BTranscription.swift#L18-L58).
- Claude selects at most 15 frames, renders one contact sheet, and adds bounded Apple Vision OCR as fallible supporting evidence. See [`ClaudeTranscriptionInputBuilder.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/ClaudeTranscriptionInputBuilder.swift#L178-L214) and [input preparation](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/ClaudeTranscriptionInputBuilder.swift#L312-L381).

**Unknown:** The repository contains no representative benchmark for daily raw bytes, OCR/model cost, energy use, missed transitions, false episode boundaries, or the sensitivity/quality trade-off across sampling intervals. Frame count is knowable; real storage and inference volume depend on screen entropy, route, model, and working hours.

## 2. Pause, exclusions, revoke, retention, and deletion

### 2.1 Capture control

**Observed:** After onboarding, Dayflow restores the saved recording preference and defaults it to on if no preference exists. See [`AppDelegate.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/App/AppDelegate.swift#L88-L147).

**Observed:** The user can pause for 15 minutes, 30 minutes, one hour, or indefinitely. Timed pauses auto-resume; indefinite pause remains off. Sleep, lock, and screensaver events pause capture and may resume it after wake/unlock when the user preference still allows recording. See [`PauseManager.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/App/PauseManager.swift#L13-L38), [pause/resume effects](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/App/PauseManager.swift#L92-L157), and [system-event handling](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/ScreenRecorder.swift#L573-L713).

**Observed:** Missing or revoked Screen Recording permission stops the timer, clears cached capture content, marks recording off, and posts a notice. See [`ScreenRecorder.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/ScreenRecorder.swift#L509-L531).

**Observed:** Users can block applications by bundle identifier. A blocked foreground app produces a local privacy placeholder instead of its pixels, and other blocked running applications are excluded through `SCContentFilter`. Dayflow seeds common password managers and authenticators into the default blocklist. See [`RecordingPrivacyPreferences.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/RecordingPrivacyPreferences.swift#L19-L60) and [the final capture check](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/ScreenRecorder.swift#L361-L415).

**Inference:** Dayflow has strong user-facing controls for a work journal, but the inspected blocklist update path does not expose an Omi-style generation fence that waits for in-flight persistence before acknowledging a policy change. Waldo should preserve the approachable blocklist UI while adapting Omi's stricter persistence-boundary fence.

### 2.2 Local custody and raw retention

**Observed:** Dayflow stores recordings under `~/Library/Application Support/Dayflow/recordings`, SQLite at `~/Library/Application Support/Dayflow/chunks.sqlite`, and database backups in the adjacent `backups` directory. It uses SQLite WAL, a five-minute checkpoint schedule, daily backups, and keeps the three newest backups. See [`StorageManager.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/StorageManager.swift#L179-L270) and [`StorageManager+Maintenance.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/StorageManager%2BMaintenance.swift#L225-L315).

**Observed:** Raw screenshot retention is size-based, not time-based. The default recording cap is 10 GB. An hourly purge marks the oldest active screenshot rows deleted before removing files, in batches of 500, until usage is below the cap. Derived timeline-card text is deliberately preserved when raw recordings are purged. See [`StoragePreferences.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/StoragePreferences.swift#L3-L30), [purge ordering](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/StorageManager%2BMaintenance.swift#L407-L520), and [`StorageSettingsViewModel.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Views/UI/Settings/StorageSettingsViewModel.swift#L94-L103).

**Observed:** SQLite also stores LLM request and response bodies. A launch-time maintenance pass truncates fields larger than 64 KiB but retains the prefix rather than omitting private model I/O. See [`StorageModels.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/StorageModels.swift#L156-L184) and [`StorageManager+Maintenance.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/StorageManager%2BMaintenance.swift#L328-L404).

**Observed:** Deleting an individual activity soft-deletes its timeline card, removes overlapping observations, and removes a referenced timelapse. It does not delete the raw screenshots in that time range; those remain governed by the recording cap. See [`StorageManager+TimelineCards.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/StorageManager%2BTimelineCards.swift#L138-L198).

**Reported:** Dayflow's privacy policy says local processing leaves screenshots on-device; BYO mode sends screenshots directly to the selected provider; cloud mode sends screenshots to Dayflow, deletes them after processing, and retains derived Activity Data for sync. It reports individual entry deletion, account deletion from active systems within seven days, export, analytics opt-out, and mode switching. The public client repository does not verify the server-side deletion path. See the [official privacy policy](https://www.dayflow.so/privacy/).

**Unknown:** Cascade deletion across local SQLite backups, LLM request/response logs, configured provider retention, temporary files after crash, cloud backups, and regenerated summaries is not established by the inspected sources. No app-layer encryption for local JPEGs or SQLite was identified in the inspected storage path. “Delete timeline entry” is not equivalent to “erase every source and derivative.”

## 3. OCR, summarization, and episode derivation

**Observed:** Dayflow's main pipeline is:

```text
periodic screenshot + hardware-idle seconds
  -> local JPEG + screenshot row
  -> 15-minute analysis batch
  -> provider-generated timestamped observations
  -> provider-generated activity cards
  -> local timeline / daily / weekly projections
```

The analysis job checks every minute. It discards the newest incomplete batch, handles sufficiently idle batches through a deterministic shortcut, and processes complete batches through the chosen provider. See [`AnalysisManager.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Analysis/AnalysisManager.swift#L42-L74) and [screenshot batching](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Analysis/AnalysisManager.swift#L602-L679).

**Observed:** Activity cards contain start/end time, category, subcategory, title, summary, detailed summary, distractions, app/site hints, source batch, and an optional backup-provider marker. See [`StorageModels.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/StorageModels.swift#L101-L153).

**Observed:** Card generation is a sliding rewrite, not an append-only event interpretation. A serialized generation gate reads up to 45 minutes of recent observations and existing cards, asks the provider to regenerate a connected range, then atomically replaces the cards in that range. Users may edit a card's title and category or delete it. See [`LLMService.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/LLMService.swift#L834-L959) and [card edits](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/StorageManager%2BTimelineCards.swift#L200-L227).

**Observed:** Apple Vision OCR is implemented as supporting evidence for the Claude route and a separate distraction feature. For Claude, a maximum of 15 sampled frames becomes one contact sheet; OCR is bounded and explicitly described to the model as fallible, with pixels treated as primary. It is not a canonical full-text index over every frame. See [`ClaudeTranscriptionInputBuilder.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/ClaudeTranscriptionInputBuilder.swift#L178-L281) and [the Claude prompt](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/ClaudeProvider%2BTranscription.swift#L206-L220).

**Inference:** The high-value Dayflow mechanism is bounded episode proposal with recent-context rewriting. The dangerous translation would be to let a rewritten card become a durable Waldo memory, commitment, Outcome, or proof. For Waldo it can only be a correctable candidate-context projection with source-frame lineage.

## 4. Provider routes and failure behaviour

**Observed:** Timeline routing supports Dayflow cloud, direct Gemini, ChatGPT/Codex CLI, Claude CLI, OpenAI-compatible endpoints, and local Ollama/LM Studio. Routing persists one primary and optional distinct secondary provider. See [`LLMTypes.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/LLMTypes.swift#L84-L118) and [`LLMProviderRouting.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/LLMProviderRouting.swift#L3-L19).

**Observed:** “Local CLI” describes local invocation, not necessarily on-device inference. Codex and Claude routes pass images or a contact sheet through those provider CLIs. The only clearly on-device analysis route in the inspected client is the local Ollama/LM Studio-compatible route. The README correctly says cloud-provider activity data leaves the machine. See [Dayflow privacy documentation](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/README.md#privacy).

**Observed:** Provider initialization and each of transcription/card generation can fall back to the configured secondary provider. Gemini has an additional Gemma fallback. The effective provider and fallback use are recorded in analytics and on generated cards. See [`LLMService.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/LLMService.swift#L200-L313) and [batch fallback](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/LLMService.swift#L719-L807).

**Observed:** Dayflow distinguishes no screenshots, short batches, zero observations, provider/configuration failures, and successful analysis. Failures are classified into actionable provider/account states or non-actionable transient/model-flaky/unknown states. A failed batch is durable and gets a visible System error card; recordings remain available for retry. See [`AnalysisManager.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Analysis/AnalysisManager.swift#L399-L452), [`TimelineFailureClassifier.swift`](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/TimelineFailureClassifier.swift#L12-L32), and [failure-card creation](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/LLMService.swift#L995-L1079).

**Adapt:** Waldo should expose capture coverage, analysis coverage, provider destination, backup-provider use, and unprocessed gaps as first-class projection facts. A fallback route must not silently disclose private capture to a provider the user did not authorize for that source and purpose.

## 5. Dayflow versus Omi as Waldo substrates

| Dimension | Dayflow at pinned revision | Omi at pinned dossier revision | Waldo decision |
| --- | --- | --- | --- |
| Primary capture | Periodic active-display screenshots and input-idle metadata | Conversation/audio ecosystem plus local desktop Rewind screen history | Use separate, purpose-bound source grants; no universal capture switch |
| Best fit | Work reconstruction and screen-derived episodes | Broader life recall, conversations, integrations, and candidate memory | Required screen/audio capability behind independent CaptureGrants; start audio with bounded granted windows and expand only through privacy evidence |
| OCR | Apple Vision only in selected paths; no canonical all-frame OCR index | Rewind locally stores OCR text and embeddings | Adapt local OCR as a disposable projection with source-frame lineage |
| Raw custody | Local JPEGs/SQLite by default; optional BYO/cloud egress | Local Rewind subsystem, but wider product can use cloud stores and providers | Keep all canonical state and raw local blobs under `~/.kennel`; disclose every egress route |
| Derivation | 15-minute batches, 45-minute rewrite window, model-generated activity cards | Searchable screen history plus conversation summaries/memories and newer reviewable candidates | Derive candidate context; never let a summary establish truth |
| Pause/exclude | Timed/indefinite pause, per-app blocklist, permission revoke handling | Progressive permissions and generation-fenced capture exclusion | Adopt Dayflow's approachable control surface and Omi's persistence fence |
| Retention | 10 GB raw cap by default; oldest frames purged; card text persists | Mixed local/cloud retention; exact coverage varies by subsystem | Source-specific TTL and cap; explicit cascade-delete receipt |
| Failure | Durable batch states, visible gaps/error cards, retry, provider fallback | Broader distributed/cloud failure surface | Honest partial coverage; no inferred continuity across gaps |
| Memory authority | No durable-memory admission model | Candidate, promotion, lineage, tombstone, and fail-closed hydration mechanisms | Adapt Omi's authority pattern into Kennel's daemon; do not copy its separate stores |
| Responsibility truth | Activity cards can be edited/deleted but are still journal records | Some task/candidate/open-loop mechanisms; mobile task checkbox can directly complete | Neither source may create/close OpenLoop or accept Outcome automatically |

The Omi statements in this table are summarized from the [pinned Omi dossier](2026-08-21-omi-implementation-reference.md), especially its storage, exclusion, candidate-admission, and operational-trade-off sections.

## 6. Proposed Waldo capture contract

### 6.1 Capture is below Memory and responsibility

**Proposed:** Implement one daemon-owned capture pipeline with source-specific adapters, not separate Dayflow, meeting, Home, and Work truth stores:

```text
CaptureGrant
  -> SourceEvent / local RawCaptureRef
  -> untrusted observation
  -> correctable candidate-context projection
       -> MemoryCandidate (optional, separate admission)
       -> CommitmentCandidate (optional, user review)

CommitmentCandidate
  -> dismiss | correct | confirm OpenLoop | create draft Outcome

Outcome
  -> ContractRevision -> PlanRevision -> WorkUnit -> Attempt -> AgentSessionRef
  -> EvidenceItem -> VerificationRun -> AcceptanceDecision
```

Rules:

1. A raw frame, OCR fragment, audio segment, transcript, summary, model card, provider session, commit, or check is never durable user truth by itself.
2. `MemoryCandidate`, `OpenLoop`, `Outcome`, `AgentSessionRef`, `EvidenceItem`, `VerificationRun`, and `AcceptanceDecision` remain separate records and lifecycles.
3. A capture-derived image may be proposed as candidate Evidence, but capture cannot register, verify, or accept it automatically.
4. A likely commitment remains a `CommitmentCandidate` until the user corrects or confirms it. Closing a confirmed `OpenLoop` still requires an explicit `LoopDisposition`.
5. Memory admission never closes an Open Loop, accepts an Outcome, or grants execution authority.
6. Home and Work consume the same intake/candidate contracts. Neither owns a duplicate Q&A, capture, or memory system.

### 6.2 One local custody and policy authority

**Proposed:** The Go daemon remains the only canonical writer. Electron may request capture grants, pause, exclusions, review, correction, promotion, revocation, and deletion, but it cannot write canonical capture or memory truth.

- Canonical metadata, grants, provenance, generations, candidate state, correction lineage, retention policy, and deletion receipts live in daemon-owned SQLite.
- Raw screenshots/audio are encrypted local blobs under `~/.kennel`, referenced by daemon records; they are not a second canonical database.
- OCR, transcription, embeddings, and model summaries are projections. A local index returns candidate IDs that must be hydrated against current daemon authority and generation.
- Every provider route binds source, purpose, minimum payload, retention expectation, allowed backup routes, expiry, and an inspectable disclosure receipt.
- Pause, exclusion, revoke, and delete increment a source-policy generation. In-flight capture or model output from an older generation fails closed before persistence or retrieval.
- Deletion covers raw blob, transcript/OCR, summary, embeddings/index entries, caches, authorized provider deletion where available, and regeneration eligibility. A content-free anti-resurrection marker may remain.

### 6.3 Separate capture modalities

**Proposed:** Waldo needs four separately governed modalities. Screen and audio are required product capabilities, while every modality remains independently consented:

1. **Trusted work facts and explicit capture:** Kennel's own Outcomes, revisions, WorkUnits, Attempts, judgments, Evidence, Verification, Acceptance, explicit notes, and Quick Capture. No ambient permission is required.
2. **Desktop visual context — required capability, governed activation:** Dayflow-like periodic or event-adaptive screen sampling during user-chosen windows, local OCR, explicit app/site exclusions, visible pause, raw-budget indicator, and correctable episode proposals. This improves work reconstruction and re-entry.
3. **Desktop audio context — required capability, governed activation:** begin with user-started or calendar-confirmed microphone/system-audio sessions with a persistent indicator, separate mic/system-audio grants, local voice activity detection where practical, disclosed transcription route, bystander-aware controls, and immediate delete. This supplies off-screen life/work context without making invisible 24/7 audio the premise.
4. **Source connectors — later:** calendar, communication, files, health, and other permissioned sources, each with least-data scope and source-specific revoke/delete. Connectors provide candidate context, never universal authority.

There should be no single “remember everything” toggle. Each source has its own capture, analysis, retention, provider, and Memory-admission controls.

## 7. Phased recommendation

### Phase 0 — parallel foundations

Keep the accepted first Outcome/Local Focus Ledger slice unchanged while starting the separately owned Home/Personal Agent lane. Use trusted Kennel facts and explicit input in Work; ratify shared responsibility, intake, provenance, and context references before either lane hardens them independently.

### Phase 1 — capture contract and synthetic harness

Design and test `CaptureGrant`, source-policy generations, raw-blob references, candidate provenance, exclusion/pause/revoke/delete receipts, provider disclosure, and gap projection using synthetic frames and transcripts. Personal Home/OpenLoop may persist through its separate vertical slice; durable admitted Memory may not be enabled merely to test capture.

### Phase 2 — Desktop Context dogfood

Add the required, explicitly activated Mac screen adapter after the necessary shared source and responsibility contracts are ratified:

- start with explicit capture windows rather than default-on all-day recording;
- capture one active display, local input-idle metadata, default secret-app exclusions, and local OCR;
- use adaptive sampling or a measured interval instead of copying Dayflow's 10-second constant blindly;
- generate bounded candidate episodes with source coverage and uncertainty;
- set a finite raw cap and expiry visible before enabling capture;
- keep Home useful when capture is denied, paused, partially processed, or deleted.

Success is lower re-entry time and useful correction-bounded episodes—not screenshots collected or timeline fullness.

### Phase 3 — Home/Open Loop projection

Allow confirmed Open Loops, Morning Brief, Catch Up, and Ready to Close to consume trusted facts plus reviewed candidate context. Captured activity may suggest a commitment or closure, but the user confirms the Open Loop and its disposition. Home-to-Work uses an explicit `ResponsibilityLink`; it does not merge lifecycles.

### Phase 4 — governed desktop audio dogfood

Implement the required microphone/system-audio capability through explicit meeting/conversation grants after its privacy and deletion/revocation tests pass. Compare local and approved cloud transcription on usefulness, latency, energy, cost, bystander risk, correction burden, and deletion coverage. Expand beyond bounded windows only through evidence; do not start with continuous wearable audio.

### Phase 5 — durable Memory gate

Only after candidate-quality and privacy evaluation should Waldo admit durable Memory. Admission must include provenance, grounding, user statements outranking inference, confidence/uncertainty, correction and counter-evidence, supersession, expiry, revocation, deletion, regeneration rules, audit, and fail-closed retrieval hydration.

## 8. Adopt / Adapt / Reject

### Adopt

- Dayflow's periodic screenshot mechanism as the reference for Waldo's required, explicitly activated desktop visual-capture capability;
- active-display selection and local hardware-idle signal;
- bounded batching, recent-context episode rewriting, honest incomplete-batch handling, and visible analysis failures;
- local raw custody, storage budgets, oldest-first cleanup, and database durability practices;
- approachable timed/indefinite pause and per-app blocklist controls;
- Omi's candidate/admission membrane, final persistence exclusion fence, tombstones, and fail-closed retrieval hydration.

### Adapt

- Dayflow activity cards -> provenance-bearing, correctable candidate-context projections, never Memory or responsibility truth;
- Dayflow provider fallback -> purpose-bound backup routes explicitly approved per capture source;
- Dayflow OCR -> local disposable projection with coverage and confidence, not a canonical semantic transcript;
- Dayflow raw storage -> encrypted blobs under `~/.kennel` governed by daemon records, expiry, generation, and cascade deletion;
- Omi conversation capture -> a required desktop audio capability that begins with explicit sessions and source-specific grants rather than ambient capture as onboarding;
- Omi memory mechanisms -> Kennel Go daemon and one SQLite writer, not Omi's backend topology or a second coordinator database.

### Reject

- default-on ambient capture after onboarding;
- one global permission bundle for screen, accessibility, microphone, system audio, automation, files, and integrations;
- continuous audio or wearable capture as the initial product premise;
- permanent raw screenshots or recordings;
- “local” claims that omit cloud model, CLI, telemetry, embedding, backup, or fallback routes;
- silent fallback to another provider for sensitive screen/audio data;
- screen activity, OCR, transcripts, or summaries automatically becoming Memory;
- inferred TODOs automatically becoming Open Loops or Outcomes;
- activity disappearance, provider completion, commits, PRs, checks, or model confidence closing an Open Loop or accepting an Outcome;
- UI-owned canonical capture, memory, or responsibility state;
- deletion that removes only a visible card while leaving unknown source and derivative copies.

## 9. Unknowns that require evaluation before implementation

- Which capture interval or event-adaptive policy provides useful episode boundaries at acceptable energy, disk, and inference cost on target Macs?
- Which apps/sites need safe defaults beyond password managers, and can web incognito/private tabs be excluded reliably without Accessibility or browser integrations?
- What local OCR/transcription models meet Waldo's quality and hardware budgets?
- What raw screenshot/audio expiry and storage cap users understand and trust? Exact defaults should follow measured dogfood rather than Dayflow's 10 GB constant.
- Can a deletion receipt prove cleanup across SQLite backups, temporary files, local indexes, model caches, and every approved remote provider?
- What capture-gap semantics prevent a missing period from being summarized as “nothing happened”?
- What precision and correction-burden thresholds are required before commitment suggestions enter Home?
- Which granted audio windows improve re-entry and Open Loop recall enough to justify their bystander and privacy risk?
- Which capture-derived source excerpts, if any, may be retained as candidate Evidence, and for how long?

Until those questions are answered, Waldo should describe desktop screen/audio as required but unshipped governed capabilities, and durable Memory as separately gated—not as shipped Minimi parity.

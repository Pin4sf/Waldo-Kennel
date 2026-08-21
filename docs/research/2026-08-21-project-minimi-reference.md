# Project Minimi reference dossier for Waldo Kennel

- Status: architecture research only; no implementation authorization
- Research date: 2026-08-21
- Product studied: [Project Minimi](https://www.projectminimi.com/)
- Operator identified by first-party sources: Shram Intelligence, Inc.
- Waldo baseline: [accepted product architecture](../product/kennel-v1-product-architecture.md), [team review packet](../product/kennel-v1-team-review-packet.md), and [Local Focus Ledger slice](../product/kennel-v0-first-outcome-slice.md)
- Source-code status: no official public repository is linked or was found; no source commit can be pinned

> **Architecture disposition:** ADR 0004 adopts Minimi-like continuity as a current Home/Personal Agent architecture input, requires governed desktop screen/audio capture capabilities, and keeps automatic truth and automatic closure rejected.

## Executive conclusion

Minimi is a useful product reference because it turns ambient capture into present-tense help: identify commitments, retrieve prior context into an existing LLM, and interrupt only when something appears actionable. Its founder explicitly argues that a memory archive is infrastructure rather than the product; the recurring product is selective capture, opinionated retrieval, useful timing, and knowing what to forget. That is strongly aligned with Waldo's Home thesis.

The most important lesson is also the main reason not to copy Minimi's apparent product contract. First-party materials describe screen/browser/app and microphone/system-audio capture, local durable memory, cloud-assisted embeddings/action-item extraction/transcription, MCP retrieval into a chosen model, and automatic open-loop resolution. They do not publish a source-level contract for provenance, memory admission, confidence, correction propagation, counter-evidence, expiry, revocation, deletion cascades, or the evidence required for automatic closure.

For Waldo Kennel:

- **Adopt** the quiet tray-like presence, low-friction recall, action-oriented retrieval, and the principle that memory should earn attention by helping with current or future responsibility.
- **Adapt** ambient capture into untrusted observations and `MemoryCandidate`s; require provenance-bearing retrieval, explicit disclosure boundaries, correctable Open Loop candidates, and owner-controlled admission and closure.
- **Reject** automatic promotion from capture to durable truth, opaque automatic closure, "remembers everything" framing, and any claim that data never leaves the device when cloud inference or a connected LLM processes it.

This research alone did not justify changing the prior phase boundary. The subsequent user decision recorded in ADR 0004 does: canonical Home/Open Loop persistence may now proceed in a separately owned lane parallel with Work after the necessary shared contracts are ratified. Durable Memory remains behind a separate admission, privacy, deletion, and evaluation gate.

## Evidence labels

| Label | Meaning in this dossier |
| --- | --- |
| **Observed** | Directly visible in a first-party page or in non-executing outer metadata of the official Mac distribution. |
| **Reported** | A first-party statement about product behavior that was not independently reproduced. |
| **Inference** | A bounded interpretation supported by Observed or Reported evidence. |
| **Proposed** | A Waldo-specific design recommendation, not a Minimi or shipped Waldo capability. |
| **Unknown** | A material behavior not established by public first-party evidence. |

Marketing statements are kept as **Reported** even when they appear on an official page. A product page saying that something is local, accurate, automatic, or supported does not by itself establish the implementation or its failure behavior.

## Identity, ownership, and source boundary

### Observed

- The official homepage and privacy policy identify the product as `minimi` and the company as Shram Intelligence, Inc. The [About page](https://www.projectminimi.com/about) says the team first built `mini-me` as an ambient-memory side project and names Jay, Vineet, and Ojasvika.
- The official download endpoint serves a signed macOS disk image. The artifact inspected without launching it was Minimi `1.0.73`, signed by `Developer ID Application: SHRAM INSIGHTS PRIVATE LIMITED (N6ZXNHU594)` and notarized by Apple.
- Artifact pin for this research snapshot:
  - mutable first-party download: [projectminimi.com/downloading](https://www.projectminimi.com/downloading)
  - observed object last-modified time: 2026-08-20 19:50:02 GMT
  - SHA-256: `5a54f1b05a98efd52f2377eeea5b111d8fd1124d682e2041ddfcf03dcb0692dc`
  - bundle identifier: `com.minimi.app`
- The official site links its homepage, About page, MCP page, legal pages, company social accounts, and download. It does not link a source repository or developer documentation.

### Source-code and clean-room boundary

No official public source repository was found, so there is no commit to pin and no licensed source to adopt. The [terms](https://www.projectminimi.com/terms-of-service) prohibit unauthorized reverse engineering. This dossier therefore does not use or reproduce proprietary prompts, algorithms, hidden behavior, internal schemas, scoring, or visual language. Outer application metadata is used only to establish that a current signed product artifact exists and declares the relevant macOS permission classes.

An unlinked, private, or unindexed repository may exist. Its existence, ownership, license, and relation to the shipped binary are **Unknown**.

## Primary source register

| Priority | Source | What it supports | Limits |
| --- | --- | --- | --- |
| 1 | [Privacy policy](https://www.projectminimi.com/privacy-policy), updated 2026-08-12 | Operator identity; capture inputs; on-device storage claim; cloud processors; MCP disclosure; analytics; deletion and correction statements. | Policy statements, not execution traces or a technical specification. |
| 2 | [MCP product page](https://www.projectminimi.com/mcp) | First-run connector flow; retrieval jobs; local-vector claim; Gemini embeddings; backend relay; BEAM claim. | Marketing page; broad compatibility and accuracy claims are unverified. |
| 3 | [Homepage](https://www.projectminimi.com/) | Open-loop positioning; action-item discovery; claimed automatic resolution; no-integration positioning. | Does not define detection, evidence, correction, or closure contracts. |
| 4 | [Founder launch article](https://www.linkedin.com/pulse/your-memory-layer-feature-stop-pretending-its-company-jay-gadekar-n31rc) | First-party product strategy: memory as infrastructure, present/future use, selective signals, interruption, forgetting. | Founder argument and benchmark claim, not independent validation. |
| 5 | [About](https://www.projectminimi.com/about) | Product origin and named team. | Narrative, not technical evidence. |
| 6 | [Terms](https://www.projectminimi.com/terms-of-service) and [Limited Use Disclosure](https://www.projectminimi.com/limited-use-disclosure) | Usage boundary, beta caveat, third-party applications, Google API Limited Use statement. | Terms contain generic "cloud-based work management" language that does not precisely describe the memory product. |
| 7 | Official signed Mac artifact pinned above | Existence, version, signature, notarization, tray/background application mode, and declared accessibility/microphone/system-audio permissions. | Not executed; internal proprietary behavior was excluded from this research. |

## What the first-party evidence establishes

### 1. Product job and interaction model

**Reported**

- Minimi positions itself as a personal memory that finds high-value action items, traces decisions, and resolves closed loops without manual effort.
- It positions memory around questions such as who sent something, what the user promised, what was decided, where the user left off, what changed in their thinking, and what deserves focus now.
- It claims to work across apps without per-app integrations.
- The founder's product argument is that a passive searchable archive has weak recurring use. Value comes from deciding which signals matter, which to ignore, when to interrupt, when to stay silent, and what to forget.

**Inference**

Minimi's actual product wedge is not generic recall. It is converting ambient context into a small attention surface and making that context available inside an LLM the user already uses. This is the strongest relevance to Waldo Home.

The "without integrations" statement most plausibly means capture through operating-system accessibility/screen/audio surfaces rather than first-party semantic APIs for every application. It should not be translated into a Waldo claim of source completeness or semantic equivalence.

### 2. Observation ingestion

**Reported**

The privacy policy says that, with prior permission, Minimi can capture:

- screen content;
- browser activity;
- interactions in supported applications through macOS accessibility permission;
- microphone audio; and
- system audio for meetings and conversations.

The official Mac artifact declares accessibility, microphone, and system-audio usage descriptions consistent with those reported inputs.

**Unknown**

- sampling cadence and event triggers;
- whether capture is pixels, accessibility-tree text, browser DOM, application APIs, or a mixture for each source;
- foreground/background and idle behavior;
- source-specific completeness and failure detection;
- deduplication, ordering, redaction, and prompt-injection handling;
- per-source retention and raw-source expiry;
- whether per-source exclusions exist and, if so, fail closed under every capture path.

### 3. Summarization, decisions, and action items

**Reported**

- Relevant captured content is sent transiently to Minimi's AI inference layer for responses, action-item extraction, and decision tracing.
- The privacy policy names OpenAI ChatGPT hosted on Azure for surfacing action items and decision tracing, Gemini for embeddings, and Deepgram for transcription.
- The homepage says Minimi finds high-value action items and automatically resolves closed loops by tracing decisions.

**Unknown**

- the shape of a source observation, summary, decision, commitment, or action item;
- whether summaries are versioned or regenerated when sources change;
- whether an action item records owner, due condition, next review, closure condition, confidence, or contradictory evidence;
- what event is sufficient to declare a loop resolved;
- whether a user must confirm, may reverse, or can inspect an automatic resolution;
- how stale or deleted source content affects derived summaries and action items.

### 4. Profile and identity

**Reported**

- The MCP flow begins with downloading the app and signing in.
- The service may collect an account email and username; billing information is handled by payment providers.
- Minimi is presented as the user's memory connected to whichever LLM they choose, not as a new model identity.

**Unknown**

- whether there is a structured user profile beyond account fields;
- which preferences or identity claims are inferred from behavior;
- whether the user can inspect, correct, scope, expire, or delete inferred profile claims;
- whether one person can separate work, personal, organization, or source-specific contexts;
- how account deletion interacts with locally stored memory and existing MCP credentials.

### 5. Retrieval and MCP

**Reported**

- First run for MCP is: install and sign in; copy an MCP link into an LLM connector; start a new chat and query memory.
- Memories are represented as embeddings in a local vector database on the Mac.
- Querying through MCP sends a relevant retrieved slice to the selected LLM, whose data handling is governed by the user's agreement with that provider.
- The product claims compatibility with ChatGPT, Claude, Gemini, and other LLMs.

**Unknown**

- the MCP tools/resources and their input/output schemas;
- authentication, credential rotation, scope, revoke, and per-client authorization;
- whether retrieval returns citations, source excerpts, timestamps, uncertainty, or freshness;
- ranker, filters, temporal reasoning, multi-hop retrieval, and context-budget behavior;
- how retrieval distinguishes explicit user statements from inferred summaries;
- whether connected agents can query all memory or only a purpose-bound subset;
- auditability of what left the device and why.

### 6. Local-first and privacy

**Reported**

- Durable "On-Device Memory" is stored on the user's device and not in Minimi's cloud database.
- Relevant content still leaves the device transiently: Minimi's backend relays content for Gemini embeddings, OpenAI action-item/decision processing, and Deepgram transcription.
- A retrieved memory slice is disclosed to the LLM the user connects through MCP.
- Product analytics may collect event-level feature use, onboarding steps, IP address, and diagnostics, but the policy says the on-device memory is not sent to the analytics provider.
- The policy says provider processing is not used for model training and is retained only as necessary for the response, with a possible limited abuse-monitoring window governed by data-processing agreements.

**Conflict**

The homepage says there is no cloud database, while the MCP page also uses stronger language that "nothing is uploaded to the cloud." The same MCP page and the privacy policy explicitly describe backend relay and cloud inference. The defensible description is:

> Local durable storage with cloud-assisted processing and user-authorized disclosure to a connected LLM.

It is not fully local computation. Waldo should make the storage, processing, and disclosure boundaries separate and visible instead of compressing them into a single "local" claim.

### 7. Correction, deletion, expiry, and revocation

**Reported**

- Users can disable screen/audio access through device settings.
- The policy says on-device memory can be deleted within the app or removed by uninstalling it.
- The GDPR section says users can edit or delete on-device content within the app.
- Users can request access, correction, or deletion of account/usage data held by the company or analytics provider.

**Unknown**

- whether editing memory creates a revision or silently overwrites it;
- whether corrections propagate to embeddings, summaries, action items, caches, and MCP results;
- whether deletion cascades to all derived content and local indexes;
- whether deletion prevents stale backup/recovery artifacts from resurrecting content;
- whether there is item-, source-, time-range-, scope-, or responsibility-level deletion;
- expiry, review, forgetting, revocation-generation, and audit behavior;
- export format and portability.

## Actual implementation versus product claim

| Capability | Evidence status | Safe conclusion |
| --- | --- | --- |
| A current Mac application exists | **Observed** signed/notarized artifact | Product is more than a landing-page concept. Runtime reliability and feature completeness were not tested. |
| Tray/background presence | **Observed** outer bundle metadata; **Reported** MCP instructions | A quiet persistent presence is an intentional interaction model. |
| Accessibility, microphone, and system-audio capture | **Observed** declared permission descriptions; **Reported** policy behavior | The distribution is prepared to request those permissions. No claim is made here about capture accuracy or completeness. |
| Local vector memory | **Reported** | Treat as first-party architecture claim, not independently verified storage behavior. |
| Cloud embeddings, action-item processing, and transcription | **Reported** in the privacy policy | Any Waldo adaptation needs explicit egress manifests and provider-purpose labels. |
| MCP retrieval into arbitrary LLMs | **Reported** | The connector shape is strategically useful; universality, auth, and scope remain unverified. |
| Open-loop discovery | **Reported** | Useful product reference, but no public precision/recall or correction evidence exists. |
| Automatic loop closure | **Reported** only | Do not treat it as safe or authoritative. Closure evidence and reversibility are Unknown. |
| 54% BEAM vs 36% LIGHT / "50% more accurate" | **Reported** only | No public evaluation code, dataset configuration, run artifact, or source commit was found. Do not use as comparative proof. |
| Editing/deleting memory | **Reported** by policy | Exact cascade, regeneration, audit, and anti-resurrection behavior remain Unknown. |
| Open-source or reusable implementation | **Unknown**, with no official repo linked | No source may be copied or adapted. Use only clean-room product/contract lessons. |

## Adopt / Adapt / Reject for Waldo Kennel

### Adopt

1. **Memory must do present-tense work.** Home should surface a Morning Brief, Catch Up item, useful recall, or a responsibility at risk—not ask users to browse an archive.
2. **Quiet presence with selective interruption.** A tray/ambient presence can observe within explicit permission while Home opens only around useful attention, capture, correction, or closure.
3. **Fast time to first proof.** Minimi's three-step MCP story gives the user an immediate recall test. Waldo should similarly reach one useful, inspectable result before requesting broad optional sources.
4. **Permission state must stay visible.** Screen and audio permissions should be understandable from onboarding and Settings, not disappear behind OS permission toggles after first run.
5. **Model-independent retrieval seam.** Memory retrieval should be consumable by the user's chosen admitted model or AgentSession through a stable provider-neutral contract.
6. **Forgetting is product judgment.** Review, expiry, supersession, release, and deletion matter as much as capture and retrieval.

### Adapt

1. **Ambient capture becomes candidate context.** Screen, app, browser, message, meeting, or audio-derived content enters as a provenance-bearing observation. A model summary becomes a `MemoryCandidate`, never durable truth by default.
2. **"Action item" becomes a candidate, then a confirmed Open Loop.** Detection may populate Catch Up, but canonical `OpenLoop` creation requires confirmation or an explicit user-authored capture.
3. **Automatic closure becomes Ready to Close.** A detected source change may propose closure with provenance and uncertainty. Only a user `LoopDisposition` closes, releases, reopens, transfers, or supersedes the Open Loop.
4. **Retrieval becomes a minimized evidence packet.** Every returned memory slice should include source, observed/valid time, admission state, freshness, confidence/uncertainty, correction lineage, and disclosure destination. It must remain below current user statements and `ContractRevision` in RunBrief precedence.
5. **Local-first becomes three explicit claims.** Show separately where durable bytes live, which processor receives transient content, and which agent/model receives retrieved context. Each boundary needs purpose, scope, retention, revoke, and audit.
6. **Cross-app continuity becomes source-aware continuity.** Broad capture is useful, but source-specific parsers/connectors should declare what they can observe and when the source is partial, stale, excluded, or unavailable.
7. **Identity context becomes correctable claims.** Role, preference, relationship, and routine inferences should remain scoped candidates. Explicit user statements outrank behavioral inference.
8. **Add source exclusions before ambient capture.** Application, domain, source, and time-bound exclusions are a Waldo safety requirement inferred from the breadth of Minimi's reported capture, not an observed Minimi mechanism.

### Reject

1. **Automatic durable memory from observation or summary.** Capture does not establish consent, importance, truth, or continuing validity.
2. **Opaque automatic closure.** A quiet thread, changed wording, calendar event, completed session, model conclusion, or missing observation cannot close an Open Loop.
3. **One undifferentiated action-item/task model.** Preserve `MemoryCandidate`, `OpenLoop`, `Outcome`, `AgentSessionRef`, `EvidenceItem`, `VerificationRun`, and `AcceptanceDecision` as separate lineages.
4. **"Everything" or "never misses" claims.** Capture is partial and failure-prone; product language and UI must show source health and uncertainty.
5. **"Nothing leaves the device" when inference is remote.** Local durable custody is valuable, but it does not erase transient cloud disclosure.
6. **Global MCP access to all memory.** Agents should receive the minimum purpose-bound context allowed for a specific responsibility and attempt.
7. **Accuracy claims without reproducible artifacts.** Benchmark results may guide questions, not establish Waldo quality or competitor superiority.

## Proposed Waldo memory boundary

This section is **Proposed** and remains behind the later Memory architecture gate.

### Candidate lineage

```text
permissioned source
  -> untrusted observation
  -> optional correctable episode/summary
  -> MemoryCandidate
  -> admit | reject | expire | revoke | delete
  -> admitted memory revision (later object name TBD)
  -> provenance-bearing minimized retrieval
```

An observation or candidate may also suggest an Open Loop, but the responsibility lineage stays separate:

```text
observation / MemoryCandidate
  -> Suggested Next Action projection
  -> user dismisses | corrects | confirms OpenLoop | drafts Outcome

OpenLoop
  -> explicit ResponsibilityLink
  -> Work Outcome with its own ContractRevision
  -> WorkUnit -> Attempt -> AgentSessionRef
  -> EvidenceItem -> VerificationRun -> AcceptanceDecision
```

Neither memory admission nor Open Loop closure proves an Outcome. Outcome Acceptance does not automatically delete, validate, or close a related memory or Open Loop.

### Minimum `MemoryCandidate` contract

The later design gate should require at least:

| Field group | Required information |
| --- | --- |
| Identity | candidate ID, owner, responsible `ResponsibilitySpace`, candidate kind, schema/version |
| Provenance | source reference, minimized source excerpt or digest, capture method, extractor/model identity and version, causal IDs |
| Claim | candidate text or structured assertion, subject, scope, valid time, observed time |
| Epistemics | confidence plus basis, uncertainty, contradictions, supporting and counter-evidence references |
| Governance | proposed purpose, audiences, disclosure class, retention, review time, expiry, revoke path |
| Admission | proposed/admitted/rejected/expired/revoked/deleted, actor, reason, timestamp, confirmation level |
| Correction | prior revision, superseding revision, user correction, changed fields, regeneration status |
| Deletion | deletion generation, derived-index cleanup status, anti-resurrection marker, audit receipt without deleted content |

Confidence must not be a decorative model score. It needs a readable basis such as source directness, corroboration, staleness, and contradiction. User statements and corrections outrank inferred confidence.

### Admission and correction behavior

1. Explicit user-authored facts may be eligible for streamlined admission, but their scope and retention still remain visible.
2. Ambient/model-derived candidates stay in a reviewable inbox or are used ephemerally; they do not silently become durable memory.
3. Admission records the exact candidate revision, source, purpose, scope, valid time, review/expiry, and confirming actor.
4. Corrections supersede prior revisions and preserve counter-evidence. Retrieval prefers the current correction while keeping the causal audit inspectable.
5. When a source or correction changes, derived summaries and indexes regenerate from current admissible inputs; stale derivatives stay excluded from retrieval until reconciliation completes.
6. Revocation immediately blocks new retrieval/disclosure even if physical deletion or index compaction is still reconciling.
7. Deletion removes content and derivatives and leaves only a content-free generation marker so stale caches, backups, or recovery logs cannot resurrect it.

### Retrieval behavior

A Waldo retrieval response should not be a free-floating paragraph. It should return:

- the minimum relevant admitted claims;
- source and correction lineage;
- observed time, valid time, freshness, and expiry;
- uncertainty and counter-evidence;
- the requesting responsibility/Attempt and reason for access;
- the external disclosure destination, if any; and
- omissions or source-health limitations material to the answer.

Retrieval is candidate context. It cannot override an explicit current user statement, `ContractRevision`, authority decision, or verified dependency output. It cannot create `EvidenceItem`, `VerificationRun`, `AcceptanceDecision`, or `LoopDisposition` merely because a semantic match was returned.

## Proposed Open Loop discovery and closure contract

### Discovery

Minimi's strongest visible behavior is converting ambient context into action items. Waldo should adapt that into a precision-oriented candidate pipeline:

1. identify a possible unresolved commitment or responsibility;
2. show the minimized source, inferred owner, trigger/due condition, proposed next review, closure condition, and uncertainty;
3. let the user dismiss, correct, keep as ephemeral context, confirm an `OpenLoop`, or draft/link a Work `Outcome`;
4. record false positives and corrections as evaluation/counter-evidence rather than silently forgetting them.

### Closure

A source event may propose `Ready to Close` only when it is tied to the Open Loop's declared closure condition. The projection must show:

- what changed;
- the supporting and contradicting source facts;
- uncertainty and source freshness;
- what remains unresolved;
- close, release, reopen, transfer, or inspect-source actions.

Only the user creates the immutable `LoopDisposition`. A model, connector, communication thread, calendar event, inactivity period, agent session, check, or accepted Work Outcome cannot close the Open Loop.

## First-run lessons for the shared Home/Work intake

Minimi's public first-run story is unusually compact: install, sign in, connect the MCP URL, ask a recall question. Waldo should preserve that sense of momentum while keeping a different authority contract.

### Proposed Waldo adaptation

1. Explain local custody and the one Waldo identity in one sentence.
2. Offer **Start with Work** and **Start with Home**, retaining Work-first as the recommended v0 path.
3. Use the shared adaptive orchestrator to ask only the one material clarification that changes the initial contract.
4. Propose the smallest sufficient next step:
   - Work: one Outcome contract and smallest-sufficient Work Unit;
   - Home: one explicit capture or one trusted-fact Catch Up item, without ambient capture as a prerequisite.
5. Preview placement, sources, disclosure, authority, and stop conditions; obtain explicit authorization.
6. Transition clearly into execution or observation and state what Waldo will return with.
7. Request accessibility, microphone, system audio, Gmail, or other source permissions only when the chosen step needs them. Each remains independently revocable.

The shared orchestrator owns clarification and plan proposal for both destinations. Home must not build a second Q&A engine or a second assistant identity.

## Evaluation gates before a durable Memory slice

The later Memory gate should require a source-backed prototype and measured results for:

| Measure | Minimum question |
| --- | --- |
| Candidate precision | Do users confirm candidates often enough that Catch Up reduces rather than creates attention cost? |
| Correction integrity | Does a correction reliably suppress stale claims across summaries, indexes, retrieval, and future candidates? |
| Counter-evidence | Can retrieval present a contradiction instead of selecting the most convenient prior claim? |
| Freshness/expiry | Do stale claims stop grounding action at the intended time without deleting useful history prematurely? |
| Revocation | Does revoked content immediately stop appearing in RunBriefs, MCP responses, logs, caches, and projections? |
| Deletion | Can content and derivatives be removed without stale recovery resurrecting them? |
| Disclosure audit | Can the user see what memory slice went to which processor/model, for which purpose? |
| Open Loop safety | Are there zero automatic canonical Open Loops and zero automatic `LoopDisposition`s? |
| Responsibility separation | Can no memory or session event bypass Outcome Evidence, Verification, or owner Acceptance? |
| Re-entry value | Does admitted memory reduce the time needed to understand state and next action without requiring raw transcript review? |

## Phase-boundary decision

ADR 0004 now records the explicit architecture change that this dossier previously required:

- **Now:** implement the separately owned Home/Open Loop lane, `MemoryCandidate` contract and evaluation foundations, and governed capture around the shared responsibility/intake contracts and the approved API/file ownership matrix.
- **In parallel with Work:** persist `PersonalHome`, `OpenLoop`, `LoopDisposition`, Daily Snapshot, Quick Capture, and `ResponsibilityLink` through daemon/SQLite single-writer boundaries without adding those objects to the Work slice.
- **After a separate Memory admission/privacy/evaluation decision:** implement durable memory storage, correction, expiry, revocation, deletion, regeneration, and retrieval.

Minimi did not supply evidence strong enough to overturn sequencing silently; ADR 0004 and the Home/Personal Agent design now make the change explicit and carry the ownership, migration, privacy, and evaluation consequences.

## Open questions to carry into the architecture review

1. Which exact observation types are worth admitting for the first Home proof, and which remain ephemeral?
2. What is the narrowest useful `MemoryCandidate` kind: explicit preference, decision, project continuity, or commitment context?
3. Can the first Home slice prove Open Loop value using trusted Kennel facts and explicit capture before any ambient source permission?
4. What source health and partial-capture states must Catch Up expose?
5. What is the canonical admitted-memory object called, and is an inspectable Markdown layer still required alongside SQLite?
6. Which processor routes are acceptable for embeddings, summarization, and transcription, and can each be local, user-provider, or disabled?
7. How are corrections and deletions propagated atomically enough to prevent stale retrieval while expensive regeneration is pending?
8. What evidence can make an Open Loop `Ready to Close` without permitting automatic closure?

## Final disposition

Project Minimi deserves a proper place in the Waldo reference set, but primarily as a product-mechanism reference:

> Ambient context becomes valuable when it returns as selective, timely help with present and future responsibility.

Waldo's differentiation is the governed boundary Minimi's public materials do not establish: observation is not memory; memory is not responsibility; responsibility is not execution; execution is not evidence; verification is not acceptance; and no inferred change is conscious closure.

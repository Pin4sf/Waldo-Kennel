# Kennel status

As of `main` after PRs #1 and #12-#14 on 2026-08-21, Kennel has a working AO-derived coding-agent orchestration chassis with isolated installed identity and state, a reduced public CLI surface, and a provider-neutral admission boundary that selects Codex for fresh v0 work while preserving historical provider compatibility. It does not yet implement the accepted Home, Work, Outcome, Open Loop, communication, Daily Snapshot, or personal-continuity architecture that will define the Waldo product.

## Shipped in the current chassis

### Backend and CLI

- A Go daemon bound to `127.0.0.1`, with health/readiness/control endpoints and an opt-in authenticated home-LAN listener.
- SQLite persistence, additive migrations, trigger-based change-data capture, SSE invalidation/replay, and durable session/chat facts.
- A thin `kennel` Cobra CLI that uses daemon HTTP rather than opening storage or adapters directly.
- A deliberately narrow root help surface; advanced/runtime commands remain directly callable for operators and compatibility.
- Project and session lifecycle, native chat and terminal interfaces, worktree management, recovery, PR/check/review observation, terminal mux, browser preview/control, and a broad inherited provider-adapter catalog.
- Codex-first admission for fresh v0 work/delegation/switch targets, now joined by the DeepSeek Harness (`dsh`) adapter for worker sessions; reviewer targets stay Codex-only, and historical provider identities and recovery reads are preserved.
- Generated OpenAPI and frontend TypeScript contracts with drift checks.

### Desktop supervisor

- Electron + React 19, a generated daemon client, project/session views, terminal and native chat surfaces, notification and PR context, browser preview/DevTools, and settings.
- A prototype Outcome clarification/planning/Kanban overlay. It is present and tested, but it is not the accepted Mission or personal-agent model.
- Kennel-owned packaging: `Kennel.app`, `in.heywaldo.kennel`, `kennel`, `kennel-app`, `~/.kennel`, `kennel-updater`, and `Pin4sf/Waldo-Kennel`.

### Foundation controls

- Pinned AO provenance and a non-destructive synchronization procedure.
- Compatibility reconciliation for the colliding migration 0098 ledgers.
- Reproducible Node/npm selection and a multi-package bootstrap.
- Local foundation tests, Go lint/race checks, production dependency audit, macOS packaged-identity assertion, Dependabot, and GitHub secret scanning. Hosted CI/security workflows are intentionally deferred.
- Zero known production npm vulnerabilities across the audited package sets at the dated foundation run. Inherited development toolchain advisories remain documented in the [acceptance record](foundation-acceptance-2026-08-18.md).

## Beta integration work, not shipped on `main`

- `beta` now carries the first three Outcome lifecycle stages on canonical lineage: Enter (PR #46), Understand with immutable ContractRevisions and marker-parsed intake retired (PRs #51, #55), and Decide & Authorize with immutable PlanRevision, one direct WorkUnit, scoped capability grants, RunBrief core digest, and owner-gated approval failing closed on stale contracts or narrowed authority (PR #58; migrations 0099–0100). Execution (#31), proof/close (#35), and evaluation (#38) remain open.
- Issue #18 adds the peer `/home` destination and contextual Home routes for Today, Open Loops, Daily Close, Memory Review, and History. These screens use deterministic renderer fixtures and local preview interactions; they do not persist a `PersonalHome`, admit Memory, create Work Outcomes, close responsibilities, or run a Personal Agent.
- The first issue #23 increment establishes the confirmed `OpenLoop` and immutable `LoopDisposition` domain plus a storage-independent Home service/port contract. Quick Capture can preserve an explicit note or non-canonical candidate; only explicit confirmation or an unambiguous direct user command can create an Open Loop, and provider/session completion cannot create lifecycle dispositions. Creation binds the Open Loop and its confirmation atomically, while idempotency replay/conflict and optimistic revision conflict behavior are covered through a transactional fake store. This increment does not yet persist `PersonalHome`, captures, Open Loops, or dispositions; add SQLite/trigger CDC, daemon API/generated contracts, restart durability, and thin UI only after the shared-file lease is released or transferred.
- The global Home/Work mode control treats `/work` as the initial Work destination, remembers an already-visited Work project/session/board route, and remembers the last Home route independently. Work project/session navigation is not rendered inside Home.
- The Work-first “Set up Home” choice navigates to the real `/home` surface without creating either responsibility space. Onboarding default persistence remains deferred to a later preferences/daemon contract.
- This beta UI work does not advance the capture, source, Home persistence, ResponsibilityLink, durable Memory, Paxel, AutoResearch, BYOK, or agent/harness gates described below.

## Present but not a current product promise

- `frontend/src/landing` is an AO marketing donor retained for build coverage; it is not the desktop launch surface.
- `packages/mobile` is an Expo donor, not a currently claimed Kennel mobile product.
- `packages/cloud-client` is a tested compatibility package, not proof of a deployed Kennel cloud service.
- Frozen `packages/ao*`, release, pod, and updater helpers remain for controlled compatibility/migration. This foundation does not publish them.
- The Go module, `backend/cmd/ao`, and some internal AO vocabulary remain deliberate source synchronization seams.

## Not shipped

- Mission as Waldo's governing unit of intent.
- A user-owned personal-agent identity or authority model.
- Personal memory admission, correction, provenance, conscious closure, or release semantics.
- Waldo verification/acceptance contracts beyond the inherited coding-work evidence model.
- Xirp, Medley, Paxel, or other named integrations.
- Implementation of the accepted Outcome contract; the current `OutcomeTask`/`completed` overlay remains donor code and is not the final product model.
- Shared Git ancestry with AO. Repairing ancestry would rewrite published history and requires separate explicit approval.

## Verification

From a normalized checkout:

```sh
npm run bootstrap
npm run test:foundation
npm run audit:production
npm run lint
cd backend && go test -race ./...
```

Package identity is verified without launching the application:

```sh
npm --prefix frontend run package
npm --prefix frontend run package:identity
```

## Accepted post-foundation desktop launch design, not shipped

The 18-20 August product-architecture session accepted one local-first Waldo Kennel desktop with three destinations—**Home**, **Work**, and **Settings & Control**—and one common five-stage lifecycle: **Enter -> Understand -> Decide & Authorize -> Act & Observe -> Prove & Close**. The stages organize the complete F01-F27 plus F02A detailed screen/state atlas; they do not replace those screens or force them into mandatory wizard steps. Settings and Operator Inspector are cross-stage overlays. Waldo's responsibility/control semantics run inside the Kennel daemon; local SQLite is the sole canonical writer; the launch does not require an account, hosted backend, or Waldo-funded model API. A `ResponsibilitySpace` separates repository-backed Work Projects from Personal Home without creating a second assistant identity.

The launch core is Outcome-to-verified-Acceptance, confirmed Open Loops, trusted Daily Snapshot, concise attention, exact Re-entry, and required governed desktop screen/audio capture capabilities for the Personal Agent lane. Required capability does not mean forced recording: each modality requires an explicit `CaptureGrant`, and Home remains useful when capture is denied, paused, unavailable, or deleted. One-account Gmail Communication Loops remains an optional draft-only beta. Durable admitted Memory remains behind a separate admission/privacy/deletion/evaluation gate; Relationship, Health, phone/wearable implementation, hosted attachment, proactive agent, and Waldo-owned harness remain later. The accepted ontology and amended phase boundary are recorded in [Waldo Kennel desktop launch architecture](product/kennel-v1-product-architecture.md), [Home/Personal Agent design](superpowers/specs/2026-08-21-home-personal-agent-memory-design.md), [ADR 0003](adr/0003-local-first-waldo-core.md), and [ADR 0004](adr/0004-parallel-home-personal-agent-and-required-capture.md).

The current **v0 local dogfood** provider constraint is Codex-only, so the team can test the end-to-end responsibility loop with less provider variability. It is not a locked v1 provider decision: v1's provider set is TBD and the core must stay provider-neutral. Current code fails closed for fresh session/review/delegation/switch selection while preserving historical identities and recovery reads. The accepted Outcome architecture extends that boundary to every Attempt start or resume, with required versus optional adapter capabilities and capability-first compatibility. A historical session becomes continuable only after its adapter is admitted, supports recovery, and passes fresh reconciliation/readmission; otherwise it remains inspectable and can hand off through a provenance-bearing packet to a new Attempt on an admitted provider.

The approved orchestration contract is recommendation-first rather than a rigid recipe: a model proposes the smallest sufficient topology and an inspectable deterministic policy validates authority, dependencies, overlap, risk, Evidence, capability, budget, and recovery constraints. RunBrief grounding follows approved user intent and verified facts before candidate context. Attempts retain tactical freedom inside an intersected authority and multidimensional budget. Leases renew silently; missing heartbeat is `unconfirmed`, not dead; fences guard canonical writes and consequential effects rather than reasoning or ordinary exploration. Verification truthfully distinguishes deterministic, producer self-check, separate-session, cross-provider/model, and owner-walkthrough evidence, and only the user accepts.

The approved [Learning and Skill Evolution design](superpowers/specs/2026-08-21-waldo-learning-skill-evolution-design.md) and [ADR 0005](adr/0005-governed-project-learning-and-skill-evolution.md) specify the Paxel/AutoResearch boundary: consented Project-attributed LearningEpisodes propose candidates; bounded campaigns evaluate one editable surface against baseline, held-in, hidden held-out, and adversarial work; only the user may promote an exact SkillRevision and scope. Learning foundations may run in shadow alongside Work/Home, but no skill activates until trustworthy Outcome result facts, evaluation, tool/privacy contracts, and rollback pass. These accepted documents are architecture authority, not shipped behavior.

[ADR 0006](adr/0006-one-durable-waldo-multiple-governed-presences.md) locks the final ecosystem direction without changing the local launch: one owner-scoped durable Waldo agent eventually spans Kennel desktop and a Health-aware Waldo mobile presence. Health First remains recommended, not required. Desktop and mobile are governed presences with bounded offline caches, never separate assistant identities or canonical memory stores. Hosted attachment, mobile implementation, health processing, and custody migration remain later specifications and gates.

v0 onboarding recommends Work first: select a local Project and define the first Outcome. Home remains an available peer and never blocks Work. Capture activation, Gmail, an account, and hosted attachment are non-blocking. Home has a calm Morning Brief and focused Catch Up flow. Suggested Next Actions remain correctable projections. The user may keep or confirm an Open Loop, create a draft Outcome in a selected Work Project, or explicitly link an Open Loop to an existing Outcome. The immutable many-to-many `ResponsibilityLink` preserves provenance and never transfers, merges, closes, verifies, accepts, or mutates either responsibility; Work still requires its own contract and authority before execution.

A [team architecture review packet](product/kennel-v1-team-review-packet.md), [clickable five-stage prototype](product/kennel-v1-review-prototype.html), and [Excalidraw session seed](product/kennel-v1-excalidraw-session-seed.md) make the accepted direction, complete detailed screen atlas, adaptive modes, lineages, failures, falsifiers, and phase boundaries reviewable. These are documentation artifacts, not shipped product surfaces.

These documents do not make the prototype Outcome overlay, PR #11's rejected Outcome schema, or any Home/Mission/Open Loop/capture/memory/communication/verification/acceptance/learning/mobile/health feature shipped. The prerequisite sequence is complete: F0-F6 landed in PR #1, legacy import removal in PR #12, provider-neutral admission in PR #13, and CLI reduction in PR #14. PR #11 is closed unmerged as a superseded donor. The [Local Focus Ledger Outcome](product/kennel-v0-first-outcome-slice.md) remains the first complete Work milestone. ADR 0004 permits a separate Home/Personal Agent lane; ADR 0005 permits shadow Learning foundations; active skills and durable admitted Memory retain separate gates. The [AO retirement audit](product/ao-legacy-retirement-audit.md) identifies active AO branding and disconnected donor surfaces that must be retired through its own plan. Each code slice still requires its own issue/PR authority.

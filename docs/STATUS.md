# Kennel status

As of the F0-F6 foundation branch on 2026-08-18, Kennel has a working AO-derived coding-agent orchestration chassis with isolated installed identity and state. It does not yet implement the accepted Home, Work, Outcome, Open Loop, communication, Daily Snapshot, or personal-continuity architecture that will define the Waldo product.

## Shipped in the current chassis

### Backend and CLI

- A Go daemon bound to `127.0.0.1`, with health/readiness/control endpoints and an opt-in authenticated home-LAN listener.
- SQLite persistence, additive migrations, trigger-based change-data capture, SSE invalidation/replay, and durable session/chat facts.
- A thin `kennel` Cobra CLI that uses daemon HTTP rather than opening storage or adapters directly.
- Project and session lifecycle, native chat and terminal interfaces, worktree management, recovery, PR/check/review observation, terminal mux, browser preview/control, and a broad inherited provider-adapter catalog.
- Generated OpenAPI and frontend TypeScript contracts with drift checks.

### Desktop supervisor

- Electron + React 19, a generated daemon client, project/session views, terminal and native chat surfaces, notification and PR context, browser preview/DevTools, and settings.
- A prototype Outcome clarification/planning/Kanban overlay. It is present and tested, but it is not the accepted Mission or personal-agent model.
- Kennel-owned packaging: `Kennel.app`, `in.heywaldo.kennel`, `kennel`, `kennel-app`, `~/.kennel`, `kennel-updater`, and `Pin4sf/Waldo-Kennel`.

### Foundation controls

- Pinned AO provenance and a non-destructive synchronization procedure.
- Compatibility reconciliation for the colliding migration 0098 ledgers.
- Reproducible Node/npm selection and a multi-package bootstrap.
- Foundation CI, Go lint/race checks, production dependency audit, macOS packaged-identity assertion, Gitleaks, govulncheck, pinned Actions, and Dependabot.
- Zero known production npm vulnerabilities across the audited package sets at the dated foundation run. Inherited development toolchain advisories remain documented in the [acceptance record](foundation-acceptance-2026-08-18.md).

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
- A decision that the current Outcome overlay is the final product model.
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

The launch core is Outcome-to-verified-Acceptance, confirmed Open Loops, trusted Daily Snapshot, concise attention, and exact Re-entry. One-account Gmail Communication Loops is an optional draft-only beta; Dayflow-inspired Desktop Context is a separately consented launch+1 beta. Durable Memory, Relationship, Health/mobile, hosted attachment, proactive agent, and Waldo-owned harness remain later. The accepted ontology, lineages, attention/recovery contract, three-destination surface, custody boundary, reference disposition, launch defaults, dogfood gate, and phased boundary are recorded in [Waldo Kennel desktop launch architecture](product/kennel-v1-product-architecture.md). The deployment decision is recorded in [ADR 0003](adr/0003-local-first-waldo-core.md).

The current **v0 local dogfood** provider constraint is Codex-only, so the team can test the end-to-end responsibility loop with less provider variability. It is not a locked v1 provider decision: v1's provider set is TBD and the core must stay provider-neutral. Admission is live, fail-closed, and scoped to every Attempt start or resume, with required versus optional adapter capabilities and capability-first compatibility. Historical provider identities remain immutable and readable; a historical session becomes continuable only after its adapter is admitted, supports recovery, and passes fresh reconciliation/readmission. Otherwise it remains inspectable and can hand off through a provenance-bearing packet to a new Attempt on an admitted provider.

The approved orchestration contract is recommendation-first rather than a rigid recipe: a model proposes the smallest sufficient topology and an inspectable deterministic policy validates authority, dependencies, overlap, risk, Evidence, capability, budget, and recovery constraints. RunBrief grounding follows approved user intent and verified facts before candidate context. Attempts retain tactical freedom inside an intersected authority and multidimensional budget. Leases renew silently; missing heartbeat is `unconfirmed`, not dead; fences guard canonical writes and consequential effects rather than reasoning or ordinary exploration. Verification truthfully distinguishes deterministic, producer self-check, separate-session, cross-provider/model, and owner-walkthrough evidence, and only the user accepts.

v0 onboarding recommends Work first: select a local Project and define the first Outcome. Home remains an available peer and never blocks Work; Gmail, Desktop Context, an account, and hosted attachment are optional. Home has a calm Morning Brief and focused Catch Up flow. Suggested Next Actions remain correctable projections. The user may keep or confirm an Open Loop, create a draft Outcome in a selected Work Project, or explicitly link an Open Loop to an existing Outcome. The immutable many-to-many `ResponsibilityLink` preserves provenance and never transfers, merges, closes, verifies, accepts, or mutates either responsibility; Work still requires its own contract and authority before execution.

A [team architecture review packet](product/kennel-v1-team-review-packet.md), [clickable five-stage prototype](product/kennel-v1-review-prototype.html), and [Excalidraw session seed](product/kennel-v1-excalidraw-session-seed.md) make the accepted direction, complete detailed screen atlas, adaptive modes, lineages, failures, falsifiers, and phase boundaries reviewable. These are documentation artifacts, not shipped product surfaces.

These documents do not make the prototype Outcome overlay, PR #11's Outcome schema, or any Home/Mission/Open Loop/communication/verification/acceptance feature shipped. The implementation sequence begins with accepting F0-F6 and replacing PR #11 with bounded post-foundation cleanup according to the [PR convergence and architecture gate plan](superpowers/plans/2026-08-20-pr-convergence-and-architecture-gate.md). The first complete milestone is the [Local Focus Ledger Outcome](product/kennel-v0-first-outcome-slice.md), delivered as five stage-aligned, issue-sized end-to-end PRs and evaluated before Home persistence expands. Exact new-session ownership and commands are in the [First Outcome execution handoff](superpowers/plans/2026-08-20-first-outcome-execution-handoff.md). No implementation is authorized by these documents.

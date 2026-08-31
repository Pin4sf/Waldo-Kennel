# Work lane execution handoff — build the Work backend & agent orchestration

- **Date:** 2026-08-24
- **Status:** Execution handoff for a fresh agent session. Planning was completed and approved in a prior session; this file is its durable record for the Work lane.
- **Lane assignment:** THIS session owns **Lane W (Work)** — Work backend integration + agent orchestration finalization. A separate Codex session owns Lane H (Home). Push/PR requires explicit per-issue user authorization. Do not implement Home lane items.

## 1. Mission

Take Waldo Kennel's Work lane from "Decide & Authorize shipped" to a complete, evaluated five-stage Outcome loop plus the first governed multi-agent (custom agents / team / per-agent computer) vertical slice — the Work-side half of the bounded alpha.

Alpha demo loop being built toward (user-approved):

```text
capture -> confirmed OpenLoop -> linked Outcome -> approved bounded plan
  -> governed agents (one receiving an isolated environment when needed)
  -> evidence-bearing recovery-capable run -> conscious owner acceptance
```

## 2. Verified repo state (all verified 2026-08-24 on `origin/beta`)

- Beta tip: **`b57413b86`** = `docs: record Work-lane progress through Decide & Authorize (#59)`. No open PRs target beta.
- Shipped lineage on beta: Enter (#46) → Understand w/ immutable ContractRevisions (#51/#55) → Decide & Authorize w/ PlanRevision, one direct WorkUnit, scoped CapabilityGrants, RunBrief core digest, owner-gated approval failing closed (#58; migrations **0099–0100**).
- Migration ledger: **0101 reserved for #31, 0102 reserved for #35**. Later issues take the next unused number at implementation start (reservations, not permanent assignments; never reserve speculatively).
- **Issue #48 is already fixed**: PR #49 (`43ab1dd13` "fix: keep the island deadline test alive on Node 22") is merged and an ancestor of tip. Remaining work is ledger cleanup only: re-run `npm run test:foundation` on pinned Node 22, then close #48 with evidence (GitHub write → needs authorization).
- Adapter seam: `backend/internal/adapters/agent/` holds all provider adapters; **`grok/` is the declared template** for the DeepSeek Harness adapter; `codex` is the only v0-admitted provider.
- Domain fact: `backend/internal/domain/responsibility.go` — `ResponsibilitySpaceKind.Valid()` accepts only `"WorkProject"` today. `PersonalHome` arrives with Home issue #23 (Codex lane).
- Chat machinery exists at `backend/internal/service/chat/` (streaming/approvals/interrupt/resume) tied to Work SessionID/ProjectID — the Conversation Runtime (Codex) must reuse it behind a new seam, not fork it.
- Repo checkout gotcha: local sparse-checkout has stale patterns from another repo (`apps/*`, `hermes_cli/*`, …) so `backend/` is NOT materialized. Fix once (see §7), by whichever lane starts first; do not redo.

## 3. Decisions locked this session (treat as binding)

1. **AgentRun subject model** (replaces "every execution is an Attempt"):
   ```text
   AgentRun.subject ∈ { ConversationRequest | OpenLoop | Outcome | WorkUnit }
   subject = WorkUnit    -> AgentRun -> Attempt -> AgentSessionRef
                            -> EvidenceItem -> VerificationRun -> AcceptanceDecision
   subject = anything else -> AgentRun -> artifacts / proposals / observations
                              (provenance-bearing, candidate context ONLY)
   ```
2. **Three asymmetries** (go into ADR 0007):
   - Execution: `PersonalHome` keeps its locked default of no arbitrary code execution; any effect beyond proposals crosses typed grants / EffectIntent regardless of subject. Computers (`ExecutionEnvironment`) are allocatable by grant to ANY subject; **durable hosted compute stays behind the ADR 0006 attachment gate**.
   - Proof: only WorkUnit-subject runs bind canonical EvidenceItem/VerificationRun/AcceptanceDecision. Everything else yields inspectable non-authoritative artifacts. Acceptance exists only on Outcomes. Team transcripts are non-authoritative in both branches.
   - Identity: specialists/teams never become Waldo identities or own the user return path (rail spec invariant).
3. **ADR 0007 scope** (to be authored): `AgentProfile`, `AgentTeam`, `ExecutionEnvironment` (none | ephemeral local | durable local worktree [= today's Attempt behavior, already locked — do not reinvent] | ephemeral hosted | durable hosted), plus invariants: fenced writer per environment, effective authority = intersection across members, budget envelope extends to compute cost/storage, custody/deletion/snapshot semantics for durable environments.
4. **Coordination mechanics** (formally amends #53's odd/even stagger):
   - Shared-file ownership LEASE: one active PR owns migrations ledger + `dto.go` + controllers `api.go` + `specgen/build.go` + `schema.ts` + `routeTree.gen.ts`; ownership recorded on the issue; transfers after merge/handoff; generated files regenerated only by the lease owner.
   - Refresh policy: branch from latest `origin/beta` at start; refresh after relevant shared-contract merges; always refresh before final verification. No churn-rebasing after unrelated merges.
   - One implementer + one reviewer per PR. Cross-lane seams: #40 (implementer Codex/reviewer Work) and provider-environment port (implementer Work/reviewer Codex).
5. **Sequencing**: DeepSeek adapter is NON-BLOCKING for the critical path but FIRST in the parallel queue (dogfood meta-experiment wants DSH as executor; #38 gains truthful cross-provider verification tier). Capture (#36)/memory (#39) move post-alpha; Daily Close (#44)/History (#45) trail alpha with labeled pre-gate states; **#41 remains the formal Home gate** — no silent relabeling.
6. **Alpha keeps all eleven gates** from the research doc (§5 reading list), including profile/team/environment visibility.
7. Orchestration vertical slice start condition: **#31 + #35 merged + conversation runtime Slice B + ADR 0007 accepted.**

## 4. Lane W execution order

| Step | Content | Notes |
| --- | --- | --- |
| W0 | Env repairs ONCE (sparse-checkout widen to include `backend/*`; reconcile untracked `docs/research/*` vs tracked same-name files before ever checking out beta); verify `test:foundation` green on pinned Node 22 via nvm; CI mode still user-pending — default to committing minimal workflow + local verify script as binding gate | Nothing branches before this |
| W1 | Issue **#60**: branch `work/deepseek-harness-adapter`: `backend/internal/adapters/agent/deepseekharness/` off `grok/` template + `dsh/waldo-profile` capability profile + lifecycle e2e scaffold with failure-injection fixtures | Parallel, non-blocking; registry hookup only, no lease files; delegable to @Developerr86 independently of the spine |
| W2a | Branch `work/31-act-observe`: migration 0101; WorkUnit/Attempt/AgentSessionRef persistence; lease/fence facts; ordered observations; recovery (contain→reconcile→narrow retry); truthful unknown-outcome; Act & Observe surfaces one stage behind DTOs | Start condition met (#26 merged). Lease holder while PR open |
| W2b | Branch `work/35-prove-close`: migration 0102; EvidenceItem/VerificationRun/AcceptanceDecision as distinct records; independence classes labeled truthfully (deterministic / producer self-check / separate-session / cross-provider / owner walkthrough); reopen lineage; resource disposition; re-entry | After W2a |
| W3 | `#38` evaluation protocol on one real Outcome (Local Focus Ledger fixture); summarize accumulated meta-experiment instruments; record continue/revise/stop verdict | Verdict gates #40 |
| W4 | During spine (parallel authoring): draft ADR 0007 text + domain types for AgentProfile/AgentTeam/ExecutionEnvironment. Implementation integration waits for dependencies; design does not | Zero dead time after #35 |
| W5 | Orchestration vertical slice: persist one user-created AgentProfile; one AgentTeam (Waldo + profile + optional task-scoped member); allocate isolated environment only to the member needing computer execution; show plan/handoffs/member state/effective permission/budget/evidence contribution/return path | After start condition in §3.7 |

Meta-experiment (pending explicit user go): register Waldo-Kennel as a Project; file hygiene slice + adapter work as the first real Outcomes once docs authority lands; they park truthfully at Authorized until the executor exists; instruments accumulate continuously for #38.

## 5. Required reading (from latest `origin/beta`, in order)

1. `AGENTS.md` — skip-worktree flag means it may be absent on disk; read via `git show HEAD:AGENTS.md`
2. `docs/product/kennel-v1-product-architecture.md` — canonical ontology, exact lineages, RunBrief, authority fences
3. `docs/product/kennel-v0-first-outcome-slice.md` — locked fixture, acceptance contract, failure matrix, falsifiers
4. `docs/superpowers/specs/2026-08-21-home-personal-agent-memory-design.md` — §9–11 ownership matrix and reservations
5. `docs/superpowers/specs/2026-08-23-global-waldo-conversation-rail-design.md` — Slices B–E; proposal cards; connection classes
6. `docs/superpowers/plans/2026-08-23-work-lane-parallel-execution.md` — Advisor v2 substrate, dogfood meta-experiment, instruments
7. `docs/research/2026-08-24-instinct-skydive-orca-agent-reach-launch-signals.md` — launch sequence, alpha gates 1–11, ExecutionEnvironment taxonomy (evidence, not authority until ADR 0007 lands)
8. `docs/STATUS.md` · ADRs `0003`, `0004`, `0005`, `0006`
9. Issue bodies: **#31, #35, #38, #40** (execution coordination blocks each)

## 6. Hard truths (non-negotiable)

- Renderer is thin; SQLite daemon is sole canonical writer; trigger-based CDC only.
- No transcript parsing in any canonical path; derived stage labels computed at read time, never stored.
- Provider completion/checks/PRs can NEVER accept an Outcome — owner AcceptanceDecision only.
- Migrations are append-only; ledger entry ships with the file; never edit merged ones.
- One issue = one branch = one PR from latest `origin/beta`.
- Effective authority is an intersection; lower layers narrow, never widen. Fences guard canonical writes and consequential effects, not reasoning. Missing heartbeat = `unconfirmed`, not dead.
- Local commits require explicit plan inclusion; push/PR/deploy are separately approved effects.
- Every environment/worktree: exactly one fenced writer. Dirty/useful-failed work retained until explicit cleanup.
- Budget envelope covers time, retries, concurrency, storage, compute cost, effects, disclosure, interruptions.

## 7. Environment gotchas

- System node is **v26.4.0** → breaks forge nested npm (EALLOWSCRIPTS). Repo pins **22.23.2** (`.nvmrc`). Use nvm; never bare `node`/`npm` for repo scripts.
- Working dev stack that works today: `go build -o .gocache/kennel-dev ./cmd/kennel` then `KENNEL_DATA_DIR=$HOME/.kennel-demo26 .gocache/kennel-dev daemon` (:3031) + `VITE_NO_ELECTRON=1 VITE_KENNEL_API_BASE_URL=http://127.0.0.1:3031 npm run dev:web` (:5173). Stale processes may hold ports — kill via lsof.
- Sparse checkout excludes `backend/` (stale patterns). Fix: `git sparse-checkout disable` (or set patterns incl. `backend/*`) AFTER reconciling untracked-vs-tracked collisions (`docs/research/*.md` exist both untracked locally and tracked on beta — diff first).
- `web_search` tool broken in DSH sessions (API key error); direct `curl` works fine for research.
- Vitest gotcha: openapi-fetch passes URL templates (`{outcomeId}`) to mocks — match templates, not concrete ids.
- API contract changes: edit `dto.go` + `specgen/build.go`, run `npm run api`, commit `openapi.yaml` + `schema.ts` together; sqlc via `npm run sqlc`; never hand-edit gen dirs.

## 8. Pending user authorizations (do NOT act without them)

1. Docs PR to beta: commit research note + ADR 0007 draft + build-program update (anchored to a new dedicated issue).
2. GitHub writes: create "Conversation Runtime" issue (Codex-owned; Slice B/C of rail spec; BYOK threat-model approval named as in-issue gate) and "Orchestration Vertical Slice" issue (Work-owned); post #48 closure comment with fresh verification evidence.
3. CI mode choice (local-first recommended default).
4. Meta-experiment filing start.
5. Per-issue push/PR authorization — required every time, forever.

## 9. Definition of done for Lane W

- One real Outcome traverses Enter → Understand → Decide & Authorize → Act & Observe → Prove & Close on a real repository with a real harness, restart/failure paths truthful (#31/#35 merged).
- #38 protocol executed; failure matrix recorded; verdict delivered.
- Integration-ready state for #40 (Work-side consumption points documented).
- ADR 0007 accepted; orchestration vertical slice demonstrable: one profile, one team, need-based environment, member-level evidence, single coherent result through Waldo.
- All eleven alpha gates that touch Work passing or explicitly labeled.

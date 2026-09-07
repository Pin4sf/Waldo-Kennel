# Model-authored Contract proposals for Outcome intake — plan

- **Date:** 2026-08-30
- **Status:** Proposed. Not started. Written after the investigation recorded in §1, and after the review-screen rework it depends on landed on `claude/outcome-intake-ux`.
- **Design authority:** `docs/adr/0007-composed-outcomes.md` for the callback pattern this reuses, and the hard rule at `AGENTS.md:108` — *"Agent-authored proposals (decomposition today, **intake later**) reach the daemon by callback, never by parsing model text, and pass the same validation as a hand-authored one."* This plan is the "intake later" half of that sentence.

## 1. What is true today (verified, not assumed)

- `backend/internal/service/intake/analyzer.go` holds the **only** implementation of `ports.IntakeAnalyzer`, and `backend/internal/daemon/daemon.go:427` wires it unconditionally. There is no dormant model-backed analyzer anywhere in the tree.
- `RuleBasedAnalyzer.Analyze` performs no model call and no I/O. It produces: the statement's **first eight words** as the title, the statement verbatim as the desired state, **one hardcoded criterion** identical for every Outcome, a hardcoded review method, a facet chosen by substring match, and a single clarifying question gated on the regex `today|tonight|this morning|this evening`. Its own doc comment says it makes a proposal *"without claiming model analysis."*
- The seam it should follow already exists and ships: `ports.DecompositionProposer` returns a **ticket**, not a proposal, and its doc comment states the intent outright — *"This mirrors IntakeAnalyzer deliberately. When intake's rule-based analyzer gains a model-backed implementation it should follow this same shape rather than inventing a second seam."*
- `backend/internal/daemon/decomposition_proposer.go` is the working reference: resolve the project's **analyzer** mission role, fail closed through `ProfileReadinessForSpawn` with no provider fallback, spawn one `KindWorker` session carrying an explicit JSON brief, return the session id.

## 2. The one thing that makes this not a drop-in

`IntakeAnalyzer.Analyze` is **synchronous** — it returns the proposal inline. The daemon makes no synchronous model call anywhere, by design. So the port itself has to move to the ticket shape before a model can author anything.

That is not the only consequence. `Service.RecoverInterruptedAnalyses` sweeps **every** intake left in `analyzing` to `analysis_failed` at daemon startup. That is correct for an in-process call that cannot outlive the process, and **wrong** for an agent analysis that legitimately spans minutes and restarts. Reusing `analyzing` would make every daemon restart kill in-flight intake analysis.

## 3. Where this deliberately diverges from decomposition

Decomposition **fails closed**: no analyzer harness ready means no proposal, which is right because decomposing is an explicit thing an owner asked for and can decline to do.

Intake is the **entry point to the entire product**. Failing closed there would mean a person with no authorized agent cannot create an Outcome at all. So:

> **The rule-based analyzer stays, as the floor.** Model analysis is attempted when an analyst role resolves and is ready; when it is not ready, times out, or returns something the daemon refuses, intake **degrades to the rule-based proposal** and says so. It never blocks Outcome creation.

This is the one place this plan does not simply mirror ADR 0007, and the reason should stay in the code.

## 4. Phases

### Phase 0 — Fix the downstream read-path bug first (prerequisite)

`GET /api/v1/outcomes/{id}` returns `authorityCeiling` all-false and omits `stopConditions` for every Outcome created through intake. Verified 2026-08-30: the values **are** persisted correctly in `contract_revision_intake_core` (checked in SQLite), but only the *intake snapshot* read path hydrates that side table (`storage/sqlite/store/intake_store.go:395`). The outcome read path does not join it.

This is pre-existing and unrelated to model analysis, but it must be fixed **first**: a model that carefully narrows an authority ceiling is pointless while everything downstream of confirmation reads that ceiling as all-false. Fixing it is also what makes Phase 4 verifiable.

**Done when:** an Outcome confirmed from an intake reports the same ceiling and stop conditions over the API that the database holds.

### Phase 1 — Reshape the port

Change `ports.IntakeAnalyzer` to mirror `DecompositionProposer`:

```go
type IntakeAnalysisTicket struct {
    SessionID string // bounded session started to answer, when one was
    Detail    string // what was started, in the owner's terms
    // Inline is the proposal when the analyzer answered synchronously — the
    // rule-based floor does. An agent-backed analyzer leaves it nil.
    Inline *IntakeAnalysisResult
}
```

Keeping `Inline` is what lets the rule-based analyzer remain a legitimate implementation of the same port instead of becoming a special case the service branches on.

**Done when:** `RuleBasedAnalyzer` satisfies the new port and every existing intake test passes unchanged in behaviour.

### Phase 2 — Durable request + callback

Mirror `decomposition_requests` (migration 0109) as `intake_analysis_requests`, migration **0110**:

- `id`, `intake_id`, `expected_proposal_revision` (freezes what was asked about — a proposal answering a superseded revision is refused, not rebound), `status`, `callback_token_digest` (SHA-256; the token is never stored), `session_id`, `expires_at`, `raw_proposal`, `refusal_reason`.
- New intake status `awaiting_analyst`, distinct from `analyzing`, and **excluded** from `RecoverInterruptedAnalyses`'s sweep. Expiry is what ends it, not process lifetime.
- `POST /api/v1/intake-analysis-requests/{id}/proposal` — the callback. Single-use, expiring, and validated by **exactly** `domain.OutcomeContractProposal.Validate`, the same gate a hand-authored revision passes.
- A refused draft is **retained** and inspectable, per ADR 0007's precedent.

**The token is scoping, not authentication.** The loopback listener is unauthenticated by deliberate decision; the token stops a confused or retrying agent answering for the wrong intake or twice. Do not describe it as auth. (`AGENTS.md:108`.)

### Phase 3 — The agent-backed analyzer

`backend/internal/daemon/intake_analyzer.go`, closely following `decomposition_proposer.go`:

- Resolve the project's **analyzer** role via `domain.ResolveMissionRoles`; probe with `ProfileReadinessForSpawn`; **no provider fallback**.
- Not ready → return the rule-based result inline (§3), not an error.
- Spawn one `KindWorker` session named `"Propose Contract: <statement excerpt>"`.
- The brief states, explicitly rather than conversationally: the statement, the project, the exact callback URL and token, the full JSON shape including every field the review screen now edits, the facet-kind enum, the authority flags in escalating order with the instruction to request the **least** that could work, and the one-material-question rule.
- **The agent reads the repository before proposing.** This is the entire point: criteria grounded in what the project actually is, not in the statement's first eight words. Decomposition already proved an agent will do this well — it grouped six Mesa criteria into two contributors rather than one-per-criterion.

### Phase 4 — Renderer

The review screen already renders and edits every proposal field (landed on this branch), so the remaining work is the waiting state:

- `awaiting_analyst` renders as real progress with the harness named, not the current indefinite "Understanding the Outcome".
- Escape hatches that always exist: **use the rule-based proposal instead**, and **release the intake**. A person must never be stuck behind an agent that will not answer.
- A degraded (rule-based) proposal is **labelled as such** in the review screen. A person deciding whether to hand-write criteria deserves to know nothing analyzed them.
- A refused draft is reopenable for correction, mirroring the decomposition editor's refused-draft path.

### Phase 5 — Verification

Unit and contract tests are necessary but not sufficient here; the thing being built is judgement quality.

- Same bar decomposition was held to: run it against a **real project** (Mesa), and confirm the agent read the repo and produced criteria that are specific to it and independently checkable — not a reworded statement.
- Verify the refusal path live: an invalid proposal is refused, retained, and correctable.
- Verify the degradation path: with no analyst harness ready, intake still completes on the rule-based floor and says it did.
- Verify a daemon restart mid-analysis does **not** kill an in-flight request (the §2 hazard).

## 5. Risks

| Risk | Handling |
| --- | --- |
| Intake becomes slow and gated on an agent | The rule-based floor stays reachable at all times, and the waiting state offers it explicitly (Phase 4). |
| A restart kills in-flight analysis | `awaiting_analyst` is excluded from the startup sweep; expiry ends it instead (§2, Phase 2). |
| An agent proposes a wide authority ceiling | Validated like any proposal, and the owner sees the real eight flags and can narrow them before confirming — which is what the review-screen rework already delivers. |
| Two proposal seams drift apart | The port comment forbids a second seam; this reshapes the existing one rather than adding a parallel path. |
| The model's proposal is *worse* than the rule-based one | This is the real risk and it is unmeasured. Phase 5 is a gate, not a formality — if the agent's criteria are not better than the hardcoded sentence, this feature is not worth its latency. |

## 6. Explicitly out of scope

- Multi-turn clarification. The one-material-question limit stays; an agent that wants a conversation is out of contract.
- Trusting an agent-authored proposal more than a hand-authored one. There is no trusted-proposer path, here or anywhere.
- Changing what a `ContractRevision` is. This changes who drafts the proposal, not what confirmation compiles.

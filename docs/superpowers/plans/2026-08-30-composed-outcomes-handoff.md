# Composed Outcomes handoff — post-merge, next session

- **Date:** 2026-08-30
- **Status:** ADR 0007 (Composed Outcomes) Phases 0–5 are shipped and merged. This is a plain continuation handoff, not a new plan — the next session should read it, then decide what to work on next.
- **What changed since the plan was written:** the [program plan](2026-08-29-composed-outcomes-program.md) and [ADR 0007](../../adr/0007-composed-outcomes.md) are the design authority; this file is only the "where things stand right now" snapshot. Read the plan for *why* things are the way they are — this file exists so a fresh session doesn't have to reconstruct *what already happened* from git log.

## 1. Verified repo state (verified 2026-08-30 on `origin/beta`)

- **`beta` tip: `5847b21`** = `Merge pull request #85 from Pin4sf/claude/composed-outcomes`. Local `beta` matches `origin/beta` exactly (0 ahead, 0 behind). Working tree clean.
- [PR #85](https://github.com/Pin4sf/Waldo-Kennel/pull/85) is merged: 27 commits, linear (no merge commits — see §5 below for why that mattered), 111 files, +18,608/−247.
- Composed Outcomes (migrations **0106–0109**), the agent-authored decomposition callback, Mission Control, and the sidebar/overview navigation wiring are all live on `beta` right now — not staged, not pending review.
- `docs/STATUS.md` reflects this (it previously said composed Outcomes were "accepted and unimplemented" — that line was stale and is now corrected).
- **Unrelated pre-existing state, do not touch as part of this work:**
  - A stash: `stash@{0}: On issue-17-work-first-shell: notch-height-offset calibration WIP`. Not from this branch.
  - Sibling worktrees `kennel-issue-17`, `kennel-issue-21`, `kennel-issue-48` and two prunable worktrees under `/private/tmp/...` for issues #16/#17. Pre-existing, unrelated to Composed Outcomes.

## 2. What is actually implemented (verified, not aspirational)

Read `docs/STATUS.md` (the "Composed Outcomes" bullets) for the authoritative list. Summary:

- An Outcome is **direct** or **decomposed**, derived from whether contributors exist — never a stored flag.
- Contribution is criterion-bound: every parent criterion is claimed by a contributor or explicitly parent-retained, enforced fail-closed at authorization.
- A contributor's authority ceiling is intersected with its parent's (narrows only).
- Sibling dependencies are declared, cycle-checked, and gate a contributor's first Attempt unless waived.
- Proof rolls up to the parent; acceptance is **batched interaction, unbatched authority** — one sitting, but each `AcceptanceDecision` stays separate and immutable. A contributor verified only by its own producer's self-check cannot enter the batch.
- **The model can author the proposal.** The daemon has no synchronous model call anywhere, so this runs as: owner asks → durable `DecompositionRequest` + single-use expiring token → spawned analyzer session → agent `POST`s its proposal back with the token → same validation gates as a hand-authored proposal, no exceptions.
  - **The token is scoping, not authentication.** The loopback listener is unauthenticated by design; the token stops a *confused* agent posting for the wrong Outcome or twice, not a hostile local process. This is a hard rule in `AGENTS.md` — do not describe it as auth, do not build anything that relies on it as auth.
  - A refused proposal is retained as an inspectable draft, not discarded.
- Renderer: Outcome Mission Control (`stage=decompose`), a decomposition editor, the ask/pending/refused-draft flow, and shared navigation — the sidebar and the Outcomes overview both derive where an Outcome row leads from one shared function (`frontend/src/renderer/lib/outcome-tree.ts`), so they cannot disagree with each other.
- Bundled into the same branch by explicit user decision (not given their own branch): OpenCode admitted across every mission role with verified continuation capability, and `dev:web:live` (drive the renderer against a real daemon from a browser, not just Electron).

**Verified end to end against a real project (Mesa), not just unit tests**: the agent read the repo, grouped six criteria into two contributors (not the mechanical one-per-criterion default), declared a real dependency, and gave an actual rationale. The refusal path was also verified live — a second Mesa Outcome still holds its refused draft.

## 3. What is explicitly NOT done — read this before assuming otherwise

From `docs/STATUS.md` and the plan's Phase 6 section, in priority order:

1. **Contributors do not run in parallel.** The Attempt fence is `project:<projectID>`, so contributing Outcomes serialize against each other and against everything else in the project. This was found during Phase 3 and deliberately **not resolved inside an implementation phase** — widening a safety mechanism to unblock a feature defeats the point of the safety mechanism. It is an open decision, not a bug.
2. **The Mission graph does not exist.** Mission Control renders an ordered list, for the same reason as #1: drawing a graph would show concurrency the daemon currently refuses to allow.
3. **Phase 6 (the evaluation gate) has not run.** It measures median active supervision minutes per accepted Outcome against direct-Codex use and flat single-Outcome Kennel use. Until it runs, whether composition actually reduces supervision cost — the entire point of the feature — is **unproven**. Everything shipped so far is machinery that makes the measurement possible, not evidence it will come out favorably.
4. **The independence bar is a fixed floor** (independent of the producer), not per-contract, because `ContractRevision.Review` is free text and cannot express a required verification strength today.
5. **Proposer sessions are not reaped.** An analyzer session spawned to propose a decomposition stays in the project's session list after it answers. Two such sessions (`mesa-2`, `mesa-3`) are currently sitting in the live Mesa project from earlier verification runs.
6. **Nothing points at Mission Control by default** for an Outcome that has never been decomposed — you have to hover a row to find the action. This was an intentional design call (no fact exists to derive a destination from until authorization happens), not an oversight, but it does mean the affordance is discoverable only on hover/focus.

## 4. What is running right now on this machine

Started this session, still live at time of writing:

- A dev Electron instance of Kennel (`npm run dev` from `frontend/`, launched via the pinned Node 22 toolchain — see §6), backed by its own dev daemon on `127.0.0.1:3032`.
- Data dir: `~/.kennel/dev/data`. Electron `userData`: `~/.kennel/dev/electron`. Both correctly scoped under `~/.kennel` per the hard rule in `AGENTS.md` / `CLAUDE.md`.
- It has the real `mesa` and `scratch` projects loaded, including the two live Mesa Outcomes used for Phase 2b verification (one with a fulfilled 2-contributor proposal awaiting authorization, one with a refused/retained draft).
- **If this is still running when the next session starts:** it's disposable dev state, safe to leave running, restart, or kill (`pkill -f electron-forge` / `lsof -ti tcp:3032 | xargs kill`) without asking. If it has died naturally between sessions, that's expected — nothing durable depends on this specific process staying alive; the durable state is in `~/.kennel/dev/data`'s SQLite file.

## 5. One thing worth knowing before touching git history on this branch again

PR #85 was rebased onto `beta` *after* merge review to produce linear history, at the user's explicit request. The gate used was **tree-hash identity** (pre-rebase and post-rebase tips had the exact same tree hash, `30ba2699`), not "the diff looked fine." The one real hazard: the merge commit being linearized carried a **hand-authored** conflict resolution (Mission Control's route vs. PR #84's WorkShell rebuild), and replaying it as a rebase caused the exact same file (`_shell.work.tsx`) to **auto-merge cleanly but wrongly** a second time — silently reintroducing the bug the original merge fixed. It had to be resolved by hand again, matching the original reconciliation exactly, not accepted from git.

**Lesson for next time a merge commit needs linearizing on this repo:** if the commit you're dropping has a non-trivial commit message explaining *why* it resolved something a particular way, assume a rebase will get that same spot wrong, check the auto-merge result against what the original resolution said, and use full tree-hash identity as the pass/fail gate — not just "no conflict markers."

## 6. Environment notes (repeated from other handoffs because they keep being needed)

- System `node` may not match the pin. Use `/opt/homebrew/opt/node@22/bin` (Node 22.23.2) explicitly for anything under `frontend/` — `PATH="/opt/homebrew/opt/node@22/bin:$PATH" npm run ...`.
- `go` at `/opt/homebrew/bin/go` (1.26.6, darwin/arm64) builds the daemon fine as-is.
- `dev:web:live` (added this branch) drives the renderer against a real daemon from a plain browser: `KENNEL_DEV_API_TARGET=http://127.0.0.1:<port> npm --prefix frontend run dev:web:live`. Known limitation, pre-existing and unrelated to this branch: a hard page load of `/work?...` does not mount that route in browser preview — navigate to it by clicking through the app instead.
- `kennel preview` (the `CLAUDE.md`-documented way to show frontend changes) requires a Kennel-managed session (`KENNEL_SESSION_ID` set); a plain terminal does not have this, hence `dev:web:live` as the fallback for browser-based verification.

## 7. Plausible next steps (not decided — pick up with the user)

Roughly in the order they were surfaced as gaps, not a mandate to do all of them:

1. **The attempt fence decision** (§3.1) — this blocks parallel contributors, a concurrency budget, and the Mission graph simultaneously. Probably the highest-leverage next conversation, but it's a real architecture decision, not an implementation detail — do not just widen the fence to unblock something without going back to the user first, per the plan's own risk table.
2. **Phase 6** — run the evaluation protocol. This is the one that actually tells you whether the feature works as intended.
3. Reap proposer sessions (`mesa-2`, `mesa-3`) that are stuck live from verification, or decide that's fine to leave alone.
4. Per-contract independence bar, if the fixed floor turns out to be too coarse in practice.
5. Anything the user brings that isn't on this list — this file is a snapshot, not a backlog.

## 8. Required reading, in order, for a fresh session

1. `AGENTS.md` — hard rules, including the two added by this branch (the composition ontology rule and the callback-token scoping rule).
2. `docs/adr/0007-composed-outcomes.md` — the accepted decision.
3. `docs/superpowers/plans/2026-08-29-composed-outcomes-program.md` — full phase-by-phase delivery record, including Phase 2b (the agent-authored proposal) and the "what is open when this program lands" list this handoff summarizes in §3.
4. `docs/STATUS.md` — current authoritative shipped/not-shipped state.
5. This file, last, for the parts of state that don't belong in permanent docs (running processes, unrelated leftovers, the rebase lesson).

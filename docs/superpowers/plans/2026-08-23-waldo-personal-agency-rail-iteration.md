# Waldo Personal Agency Rail Modes — Executable Plan

**Goal:** Evolve the existing PR #50 `WaldoRail` into a preview-only Personal Agency surface with explicit Conversation and Activity modes, three entry states, topic/context control, structured semantic cards, and inspectable bounded-run detail while preserving every canonical truth and shell behavior.

**Architecture:** Keep one `WaldoRail` and one shell owner. Add only deterministic renderer state and localized fixture presentation inside the existing preview branch. Canonical desktop continues down the existing unconfigured branch before any preview controls or content render.

**Spec:** `docs/superpowers/specs/2026-08-23-waldo-personal-agency-modes-checkpoint.md`

## Constraints

- Stack locally on PR #50 tip in the existing worktree and branch.
- No push, PR mutation, merge, rebase, deploy, release, publication, API, daemon, migration, persistence, provider, runtime, or generated-contract work.
- Preserve Home Today, one expanded Quick Capture, Home/Work pill, dynamic Home navigation, Work Inspector return, narrow layer, configured shortcut, focus restoration, and explicit preview predicate.
- Run every behavior test red before production edits.
- Keep all preview actions local and explicitly non-authoritative.
- Use localized catalog keys for user-facing chrome and content.

## Task 1 — Prove the Conversation model red

**Files:**
- Modify `frontend/src/renderer/components/waldo/WaldoRail.test.tsx`

Add independent behavior tests that fail because the current rail lacks:

1. `Conversation | Activity` tabs with Conversation selected initially.
2. `Fresh`, `Contextual`, and `Returning` episode selection.
3. Fresh suggested prompts with no plausible prior exchange.
4. Contextual pinned context and a detachable `No attached context` state.
5. Returning short-first summary with explicit expansion.
6. Separate `Noticed`, `Candidate`, `Proposal`, `Approval required`, `Result ready`, and `Outcome · Unknown` semantics.

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/waldo/WaldoRail.test.tsx
```

Confirm the assertions fail for missing behavior, not setup errors.

## Task 2 — Implement the minimal Conversation model green

**Files:**
- Modify `frontend/src/renderer/components/waldo/WaldoRail.tsx`
- Modify `frontend/src/renderer/i18n/{en,zh-CN,ja,ko,es,fr,de,pt-BR}.json`
- Modify `frontend/src/renderer/styles.css` only where semantic component classes improve responsive layout

Implement local preview types for mode and episode. Render the stable preview boundary first, then mode tabs, episode selector, entry-state chip, detachable context, and the entry-specific structured content. Keep proposal review local. Fresh state must not render prior user/Waldo messages.

Run the focused rail and localization tests until green:

```bash
npm --prefix frontend test -- --run src/renderer/components/waldo/WaldoRail.test.tsx src/renderer/i18n/instance.test.ts src/renderer/i18n/renderer-coverage.test.ts
```

Refactor only after green. Then create the local checkpoint:

```bash
git add docs/superpowers/specs/2026-08-23-waldo-personal-agency-modes-checkpoint.md docs/superpowers/plans/2026-08-23-waldo-personal-agency-rail-iteration.md frontend/src/renderer/components/waldo/WaldoRail.tsx frontend/src/renderer/components/waldo/WaldoRail.test.tsx frontend/src/renderer/i18n/*.json frontend/src/renderer/styles.css
git diff --cached --check
git commit -m "feat: add Waldo conversation episodes"
```

## Task 3 — Prove Activity inspection red

**Files:**
- Modify `frontend/src/renderer/components/waldo/WaldoRail.test.tsx`

Add tests that switch to Activity and prove:

1. Conversation cards are replaced by one bounded-run surface.
2. The collapsed run exposes status, goal, current step, approval, evidence, and return summary.
3. `Inspect run` expands goal, completion condition, scope, delegated agent, permitted sources, ordered plan, authority, approvals, evidence, return path, and terminal truth.
4. The result remains `Result ready` with `Outcome · Unknown`; no `Done` or `Verified outcome` claim appears.
5. The specialist is described as a bounded delegation under Waldo, not as another relationship or navigation roster.

Run the focused test and confirm RED.

## Task 4 — Implement Activity green and preserve shell contracts

**Files:**
- Modify `frontend/src/renderer/components/waldo/WaldoRail.tsx`
- Modify locale catalogs
- Modify `frontend/src/renderer/styles.css` only if the real rail needs a layout correction

Implement the short-first bounded-run card and local expand/collapse behavior. Reuse the existing run step treatment, but make every required field inspectable and every state non-color-only. Do not add provider controls or executable approvals.

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/waldo/WaldoRail.test.tsx src/renderer/components/waldo/WaldoRailContext.test.tsx src/renderer/components/waldo/WaldoShellRail.test.tsx src/renderer/components/waldo/WaldoChromeInteraction.test.tsx src/renderer/components/home/HomeShell.test.tsx src/shared/shortcuts.test.ts
```

## Task 5 — Add and prove integrated preview journeys

**Files:**
- Modify `frontend/e2e/waldo-home-ui.spec.ts`

Write failing E2E journeys for:

- Conversation/Activity switching and mode keyboard semantics;
- contextual episode selection and context detachment;
- returning expansion and result/unknown boundary;
- Activity run inspection;
- canonical route absence of preview controls/content;
- narrow Back, no horizontal overflow, and focus restoration.

Make only test-proven corrections. Then run narrow and broad gates:

```bash
npm --prefix frontend test -- --run src/renderer/components/waldo/WaldoRail.test.tsx src/renderer/components/waldo/WaldoRailContext.test.tsx src/renderer/components/waldo/WaldoShellRail.test.tsx src/renderer/components/waldo/WaldoChromeInteraction.test.tsx src/renderer/components/home/HomeShell.test.tsx src/renderer/components/home/HomeNavigation.test.tsx src/shared/shortcuts.test.ts
npm --prefix frontend test -- --run src/renderer/components/home/HomeOpenLoops.test.tsx src/renderer/components/home/HomeDailyClose.test.tsx src/renderer/components/home/HomeMemoryReview.test.tsx src/renderer/components/home/HomeHistory.test.tsx src/renderer/i18n/instance.test.ts src/renderer/i18n/renderer-coverage.test.ts
npm --prefix frontend run typecheck
npm --prefix frontend run typecheck:e2e
npm --prefix frontend run test:e2e -- home-work-mode.spec.ts home-today-layout.spec.ts home-personal-agent-flows.spec.ts waldo-home-ui.spec.ts
npm --prefix frontend run build
git diff --check
```

## Task 6 — Inspect actual Electron wide, narrow, and canonical truth

Run the actual Electron app with `VITE_WALDO_UI_PREVIEW=1`, use `kennel preview` from inside the session, and capture wide (`1512x982`) and narrow (`720x760`) evidence for Conversation states, Activity inspection, Home context replacement/restoration, Work Inspector return, focus, and overflow. Launch again without the Waldo preview variable and prove no plausible modes, topics, messages, cards, or runs appear.

Store images only under `/tmp/waldo-personal-agency-rail-review-2026-08-23/`.

Review the full stacked diff and create the final local checkpoint only after all current gates and visual evidence support it:

```bash
git add frontend/e2e/waldo-home-ui.spec.ts frontend/src/renderer/components/waldo/WaldoRail.tsx frontend/src/renderer/components/waldo/WaldoRail.test.tsx frontend/src/renderer/i18n/*.json frontend/src/renderer/styles.css
git diff --cached --check
git commit -m "test: verify Waldo Personal Agency modes"
```

Do not stage screenshots, local runtime state, generated artifacts, research assets, or unrelated files.

## Completion boundary

Completion means the renderer preview proves the interaction and semantic contract while canonical desktop remains honestly unconfigured. It does not mean conversation, agent orchestration, provider connection, persistence, responsibility mutation, or Outcome verification exists.

# Waldo Product Surface Completion — Executable Plan

**Goal:** Finish the Kennel Personal Agency renderer surface before connecting the real Waldo agent, preserving truthful state and one global Waldo relationship.

**Spec:** `docs/superpowers/specs/2026-08-23-waldo-product-surface-completion-checkpoint.md`

**Baseline:** feature branch rebased onto `origin/beta` at `0ae292745`; PR #52 and PR #54 are landed. The user has authorized pushing this feature branch and opening a PR against `beta`; merge, release, deployment, provider/backend wiring, and persistence remain unauthorized.

## Phase 0 — Restore the interaction and regression foundation

### 0A — Classify the landed gate before changing assertions

1. Re-run the full Playwright gate from the landed baseline and retain the failure list.
2. Classify each of the 35 reported failures as product/source regression, fixture or daemon precondition, stale product assumption, or cascading failure. Do not weaken an assertion merely to make the gate green.
3. Restore the broad gate or document every remaining non-product precondition with a focused reproducible command and owner. The product-surface work is not complete while a source regression is hidden in the baseline.

**Local checkpoint (2026-08-23): complete.** The landed 33 pass / 35 fail baseline was restored to 69 pass / 0 fail before new destinations; the completed preview surface now runs 75 pass / 0 fail. Source fixes cover mode-switch hit testing/layout, the 350px Inspector contract, chat overflow containment, and board New terminal routing. Harness fixes cover the renamed public preload bridge, onboarding state, agent-catalog routes, and mux/replay readiness. Obsolete demo routes, fixture names, roles, labels, and exact preview-state assumptions were updated after product inspection. Retained-terminal tests now wait for actual mux/replay readiness; the bounded reveal sequence permits at most four Chromium scroll events while still proving one open writer, no parked resize, and a final viewport at the latest output.

### 0B — Pointer, pane width, and responsive composition (TDD)

1. Add a failing public-interaction test proving Work titlebar controls such as New terminal and Switch agent remain pointer-reachable while the Home/Work switch is present. Fix the switch's layout/hit region rather than routing around the control.
2. Add a failing resize test that states the Inspector contract at supported desktop widths. Reconcile the 280px implementation and the established 350px E2E expectation with one responsive rule: preserve readable Inspector content when the center can support it, and use an explicit compact/full-layer composition when it cannot.
3. Add failing minimum-, medium-, wide-, ultrawide-, and short-window tests for Work center readability with Waldo open, internal scrolling, topic overflow, state-aware launcher labels, hidden-pane inertness, Back versus Close, focus restoration, and no horizontal document overflow.
4. Keep the renderer thin: layout state may be local/provider-owned, but no runtime authority, provider connection, or persistence belongs in these fixes.

### 0C — macOS interaction grammar (TDD)

1. Test and implement native app-menu commands for the visible Home/Waldo actions, with discoverable shortcuts and disabled states where an action is unavailable.
2. Test pointer secondary-click menus for meaningful Home/Waldo objects, browser-style Back/Forward including standard mouse/trackpad navigation events, and draft-safe navigation that never silently loses unsent text.
3. Test deterministic focus entry/order/restoration, visible focus, inactive-window legibility, accessible names/status announcements, Full Keyboard Access, VoiceOver reading order, and `prefers-reduced-motion` behavior.
4. Preserve native browser/text-edit behavior where it already provides the correct macOS contract; do not shadow it with renderer-only imitation.

**Status (2026-08-23): open beyond the automated renderer subset.** Browser-style titlebar history, draft-safe shared state, roving tab focus, compact-layer focus entry/Escape/restoration, pointer reachability, and Reduce Motion CSS have automated coverage. Native Home/Waldo app-menu commands, secondary-click object menus, standard mouse/trackpad navigation events, Full Keyboard Access, VoiceOver reading/announcements, and inactive-window inspection are not implemented or physically verified in this checkpoint and must not be represented as complete.

### 0D — Gate and checkpoint

Run focused unit/E2E tests, the full Playwright gate, typechecks, build/package identity, `git diff --check`, and physical Electron inspection. Create one local conventional commit only after the source-owned regression gate is green or the remaining external preconditions are evidence-bounded.

## Phase 1 — Add a dedicated Home Chat presentation

### RED

Prove Home exposes `Chat` and that:

- Home and the rail present the same Waldo relationship, episode, mode, attached context, and draft rather than duplicate state;
- rail-to-Chat and Chat-to-rail transitions preserve topic ownership and restore focus;
- canonical unconfigured mode contains no plausible transcript or successful action;
- Work return, narrow navigation, keyboard access, and draft-safe Back behavior remain correct.
- the destination has distinct episode, conversation, and source/context regions at wide width, then collapses optional context before degrading the conversation at medium/narrow widths;
- the deterministic preview question “What changed in the pricing workshop and what still needs me?” produces one local-fixture reply with typed source links and an explicit no-provider/no-save disclosure;
- correcting or detaching context is visible and reversible locally, and no preview decision creates a responsibility, memory, effect, run, or accepted outcome.

### GREEN

Extract a shared thin Waldo conversation presentation/state contract from the existing rail. Home gets a dedicated responsive composition: episode rail, readable transcript, source-aware composer, and optional context/evidence inspector. Home may provide more reading room, but it must not create a second message store, a second Waldo identity, a provider client, or runtime authority.

### VERIFY

Run Today-to-Chat, rail-to-Chat, Work return, keyboard, focus, canonical-truth, responsive, full Playwright, typecheck, build, and physical Electron checks. Commit locally.

**Local checkpoint (2026-08-23): complete for the local preview boundary.** `Chat` is a primary Home destination backed by the same `WaldoRailContext` conversation state as the global rail, including the mounted preview exchange rather than only mode/draft metadata. Wide presentation now separates resumable topics, the readable transcript/composer, and source/freshness/gap context; medium and narrow presentation collapse optional context behind an accessible details dialog with focus entry, Escape, trapping, and trigger restoration. The deterministic `Send preview` fixture clears the composer, reveals the latest answer, opens its exact local source Record, and can route a bounded review to Activity. Canonical mode still exposes no plausible transcript, Send action, provider, message transport, or persistence.

## Phase 2 — Replace History with evidence-backed Insights plus Records

### RED

Add Home navigation and surface tests proving:

- `Insights` replaces the visible `History` label while preserving `/home/history` as a compatibility alias or redirect;
- an honestly empty canonical surface contains no synthesized claims;
- preview Insights expose observed facts, inference boundaries, known gaps, source window/freshness, provider/synthesis disclosure, and status;
- `Confirm`, `Correct`, `Dismiss`, and `Why this?` update only local preview state;
- `Records` retains inspectable continuity and supporting activity without becoming an Insight claim;
- no mood, personality, health, confidence score, commitment, or verified outcome is inferred.

### GREEN

Refactor the existing Home History fixture into an Insights presentation with a `Records` subview. Keep the fixture deterministic and explicitly preview-only. Update Daily Close return links, route compatibility, and localized navigation copy. Do not add telemetry collection or persistence.

### VERIFY

Run Home navigation, Insights, Records, Daily Close, shell, i18n, canonical-truth, full Playwright, typecheck, build, and physical Electron wide/narrow checks. Commit locally.

**Local checkpoint (2026-08-23): automated complete.** Visible `History` became `Insights` while `/home/history` remains the compatibility route. Deterministic candidates expose direct observation, source window/freshness, inference boundary, known gaps, an explicit no-model/no-provider synthesis route, Why, Confirm, Correct, and Dismiss; Records retains continuity and supporting evidence below interpretation. The unconfigured component renders no candidate or record. Unit, localization, Daily Close, navigation, and Playwright coverage pass.

## Phase 3 — Model specialist profiles without a peer-agent roster

This preview-only phase is approved after Phases 0–2. Grok Bot is the primary discovery/card benchmark; Hermes is the secondary profile/capability/governance benchmark.

### RED

Prove Activity can open a preview-only specialist profile/creation surface with purpose, explicit scope, sources/tools, authority ceiling, budget, completion condition, pause/revoke, evidence expectation, and Waldo return destination. Prove no specialist becomes another Waldo, no default roster is fabricated, canonical mode stays empty/unconfigured, and no UI action executes, connects, saves, or persists.

Also prove the preview Activity → Agents workspace has a compact specialist rail, selected run timeline, and evidence/authority inspector; Waldo remains labelled as coordinator; approvals offer `Accept`, `Edit`, `Reject`, and `Respond`; and pause, resume, stop, retry, recovery, revoke, and return controls change only deterministic local preview state.

### GREEN

Expose compact specialist creation under Waldo Activity and make the selected specialist/run legible in a dedicated responsive workspace. Use structured governed cards; do not introduce a Home roster or fake searchable capability catalog. A deterministic preview specialist may appear only behind the visible preview seam to prove selection, run, evidence, and recovery behavior. Keep Skills/Tools/MCP, provider, model, and persona unavailable until a truthful native source exists. Any future run remains bounded delegation returned through Waldo—not a second personal-agent transcript.

### VERIFY

Run profile contract, Activity, Home non-roster, canonical-truth, accessibility, responsive, full Playwright, typecheck, build, and physical Electron checks. Commit locally.

**Local checkpoint (2026-08-23): complete for the local preview boundary.** Waldo Activity alone exposes Waldo as coordinator, a compact specialist rail, the selected run timeline, inline `Accept / Edit / Reject / Respond` review, and the source/authority/evidence inspector. The Research specialist names purpose, explicit scope, permitted sources/tools, authority ceiling, budget, completion condition, evidence expectation, and Waldo return destination. Accept, pause, resume, stop, retry/recovery, and return now follow one coherent mounted-preview state path; unavailable Home creation is disabled and truthfully described. The fuller creation/pause/revoke profile remains in the global Waldo Activity preview. Nothing is connected, run, or saved.

## Verification checkpoint — 2026-08-23

- unit: 211 files, 2632 passed, 6 skipped;
- typecheck and typecheck:e2e: pass;
- full Playwright: 75 passed, 0 failed;
- build and package identity: pass for `in.heywaldo.kennel`;
- physical Electron: the local preview renderer and daemon launched successfully. Public UI/accessibility inspection covered the wide three-region Chat, deterministic send/source flow, Activity specialist/run/authority composition, medium-width inspector collapse, and the final 960px-minimum `Open context details` layer with its `Back to conversation` return. The first inspection exposed missing compact details access and weak submit/scroll feedback; both were corrected before the final narrow pass. Standard pointer and accessibility-tree behavior were inspected without accessing credentials or private app state. System-level VoiceOver, Full Keyboard Access, inactive-window, trackpad gesture, and Reduce Motion toggles remain explicitly open;

## Phase 4 — Establish the real Personal Agency backend boundary

After the renderer surface and schemas are accepted:

1. define durable conversation, episode, context attachment, Insight candidate, responsibility, run, approval, evidence, and Outcome facts;
2. define daemon DTO/API and persistence with migrations and generated contracts;
3. add secure provider-neutral connection setup using OS Keychain or equivalent secret storage;
4. connect streaming discussion without granting execution authority;
5. add bounded delegated runs, explicit approvals, evidence ingestion, return paths, and native Outcome verification;
6. add recovery, revoke, audit, and unknown-state behavior before proactive orchestration.

The user should not provide an API token until the secure credential flow exists. A token is never accepted through chat, renderer fixtures, source files, logs, or test data.

## Required gates for surface completion

```bash
npm --prefix frontend test -- --run \
  src/renderer/components/waldo/WaldoRail.test.tsx \
  src/renderer/components/waldo/WaldoRailContext.test.tsx \
  src/renderer/components/waldo/WaldoShellRail.test.tsx \
  src/renderer/components/waldo/WaldoChromeInteraction.test.tsx \
  src/renderer/components/home/HomeShell.test.tsx \
  src/renderer/components/home/HomeNavigation.test.tsx \
  src/renderer/i18n/instance.test.ts \
  src/renderer/i18n/renderer-coverage.test.ts
npm --prefix frontend run typecheck
npm --prefix frontend run typecheck:e2e
npm --prefix frontend run test:e2e -- waldo-home-ui.spec.ts home-personal-agent-flows.spec.ts home-work-mode.spec.ts
npm --prefix frontend run test:e2e
npm --prefix frontend run build
npm --prefix frontend run package:identity
git diff --check
```

Actual Electron must then be inspected at short, minimum supported, medium, wide, and ultrawide dimensions in preview and canonical modes. Completion requires screenshots and accessibility/DOM evidence for pointer reachability, secondary click, trackpad/mouse Back, menus/shortcuts, scrolling, overflow, focus, Full Keyboard Access, VoiceOver order and announcements, inactive-window appearance, Reduce Motion, truth labels, and the absence of plausible canonical data.

## Integration boundary

This branch is rebased onto current `origin/beta`, and the user has explicitly authorized pushing it and opening a PR against `beta`. Do not merge, deploy, publish, migrate, release, or connect a provider/backend without new authorization.

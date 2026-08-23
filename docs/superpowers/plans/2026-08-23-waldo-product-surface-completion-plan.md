# Waldo Product Surface Completion — Executable Plan

**Goal:** Finish the Kennel Personal Agency renderer surface before connecting the real Waldo agent, preserving truthful state and one global Waldo relationship.

**Spec:** `docs/superpowers/specs/2026-08-23-waldo-product-surface-completion-checkpoint.md`

## Phase 0 — Finish the visible shell checkpoint

1. Replace inherited AO desktop marks with the approved Figma Waldo paw in the sidebar, startup, global launcher, rail, app icon, and tray assets.
2. Replace user-visible AO wording with Waldo for the product/account and Kennel for the desktop harness/daemon.
3. Preserve internal AO compatibility identifiers, headers, paths, and generated contracts until a separately tested migration exists.
4. Prove the wide Home rail stays within 440px, owns vertical scrolling, and remains usable at the viewport bottom.
5. Run focused unit tests, the Waldo Home E2E suite, typechecks, build/package, and actual Electron wide/narrow inspection.
6. Create one local conventional commit. Do not push or integrate.

## Phase 1 — Replace History with evidence-backed Insights

### RED

Add Home navigation and surface tests proving:

- `Insights` replaces the visible `History` label while preserving the route as a compatibility redirect or alias;
- an honestly empty canonical surface contains no synthesized claims;
- preview Insights expose observed facts, inference boundaries, source window/freshness, and status;
- `Confirm`, `Correct`, `Dismiss`, and `Why this?` update only local preview state;
- the underlying continuity and supporting-activity records remain inspectable;
- no mood, personality, health, confidence score, or verified outcome is inferred.

### GREEN

Refactor the existing Home History fixture into an Insights presentation with a `Records` subview. Keep the fixture deterministic and explicitly preview-only. Update Daily Close return links and localized navigation copy. Do not add telemetry collection or persistence.

### VERIFY

Run Home navigation, Insights, Daily Close, shell, i18n, type, E2E, build, and actual Electron wide/narrow checks. Commit locally.

## Phase 2 — Add a dedicated Home Chat presentation

### RED

Prove Home exposes `Chat`, opens the same Waldo relationship, preserves topic and context presentation across rail/full-page transitions, restores focus, and remains canonical-unconfigured without preview.

### GREEN

Extract a thin shared Waldo conversation presentation from the existing rail. The Home destination may provide more reading room, but mode, episode, card semantics, and authority boundaries remain one state model. Do not create a second message store or provider client.

### VERIFY

Test Today-to-Chat, rail-to-Chat, Work return, keyboard, narrow navigation, canonical truth, and build. Commit locally.

## Phase 3 — Model specialist profiles without a peer-agent roster

This phase requires confirmation of the proposed IA before implementation.

Define a preview-only `AgentProfile` with purpose, scope, tools/sources, authority ceiling, owner, revoke action, and Waldo return path. Expose creation/management in Work or Settings. Show specialists in Home only when explicitly created or pinned, and show runtime use as a bounded delegation under Waldo.

Prove no specialist becomes another Waldo, no default roster is fabricated, and no UI action executes or persists.

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
npm --prefix frontend run test:e2e -- waldo-home-ui.spec.ts home-personal-agent-flows.spec.ts
npm --prefix frontend run build
git diff --check
```

Actual Electron must then be inspected at wide and narrow dimensions in preview and canonical modes. Completion requires screenshots and DOM evidence for scrolling, overflow, focus, keyboard access, truth labels, and the absence of plausible canonical data.

## Integration boundary

All work remains local and stacked. Do not push, open or update a PR, merge, deploy, publish, or rebase. After PR #50 lands, refresh the implementation branch onto current `origin/beta` only with explicit user authorization.

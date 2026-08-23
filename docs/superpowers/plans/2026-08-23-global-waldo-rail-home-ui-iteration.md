# Global Waldo Rail and Home UI Iteration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a globally reachable, truthful Waldo conversation rail with a Dimension-inspired bounded-agent activity preview, and make the already-built supporting Home screens discoverable without changing canonical responsibility state.

**Architecture:** Keep this slice inside the Electron renderer and shared shortcut catalog. A shell-owned Waldo provider owns ephemeral open state, focus return, and preview selection; Home reuses its existing contextual-right region while Work receives one temporary right rail that restores the existing Inspector. Canonical desktop routes render an honest unconfigured state, while deterministic conversation and agent activity exist only under explicit local preview seams.

**Tech Stack:** Electron 33, React 19, TypeScript 5.6, TanStack Router, Zustand shortcut overrides, existing Kennel tokens and CSS, Vitest, Testing Library, Playwright.

**Spec:** `docs/superpowers/specs/2026-08-23-global-waldo-conversation-rail-design.md`

**Research basis:** `docs/research/dimensions-personal-agency-visual-audit-2026-08-22.md`

## Global Constraints

- Work only in `/Users/shivanshfulper/.codex/worktrees/home-waldo-conversation-ui-design` on `codex/home-waldo-conversation-ui-design`.
- Keep all work local: no push, PR, merge, deploy, release, publication, or GitHub mutation.
- Issue #21 remains absent from `beta`; do not implement issue #23 persistence or invent a Home-specific responsibility identity.
- Reuse the Home components merged in PR #47. Do not create duplicate Today, Open Loops, Daily Close, Memory Review, or History screens.
- Keep the Home/Work pill and exactly one expanded Quick Capture on Today.
- Today and Open Loops remain the primary Home destinations. Daily Close, Memory Review, and History are a visually subordinate continuity group.
- Canonical Electron routes show an unconfigured/unavailable Waldo until daemon contracts exist.
- Deterministic chat, proposal, and agent traces may render only when `VITE_NO_ELECTRON=1` or the local-only `VITE_WALDO_UI_PREVIEW=1` development seam is present.
- The preview seam must not change daemon data, call a model/provider/runtime, create an Outcome/Open Loop, send an effect, or persist a conversation.
- Every proposal and activity preview must state that it is not live and not saved.
- No LLM output, activity event, agent completion, suggestion click, or UI render may confirm, close, release, or transfer responsibility.
- Keep renderer state ephemeral. Do not add APIs, migrations, SQLite access, manual CDC, generated-contract edits, secret handling, provider OAuth, BYOK, or agent execution.
- Use the existing localization catalog; do not add hardcoded English JSX chrome.
- Preserve visible focus, exact focus restoration, reduced motion, narrow full-layer behavior, and no horizontal overflow.
- Run each test red before writing the corresponding production code.

## File map

### New files

- `frontend/src/renderer/components/waldo/WaldoRailContext.tsx` — ephemeral shell-owned rail state, invocation origin, and deterministic focus restoration.
- `frontend/src/renderer/components/waldo/WaldoLauncher.tsx` — global top-right launcher using the provider.
- `frontend/src/renderer/components/waldo/WaldoRail.tsx` — truthful unconfigured state plus explicitly gated conversation, proposal, and bounded-agent activity preview.
- `frontend/src/renderer/components/waldo/WaldoShellRail.tsx` — Work/supporting-route placement and Inspector-return affordance.
- `frontend/src/renderer/components/waldo/WaldoRail.test.tsx` — truth, proposal-authority, and agent-trace behavior.
- `frontend/src/renderer/components/waldo/WaldoRailContext.test.tsx` — open/close, keyboard-safe toggling, and focus return.
- `frontend/src/renderer/components/home/HomeNavigation.test.tsx` — primary versus supporting Home navigation behavior.
- `frontend/e2e/waldo-home-ui.spec.ts` — wide/narrow shell, Home-screen access, and preview-boundary journeys.

### Modified files

- `frontend/src/shared/shortcuts.ts` and `frontend/src/shared/shortcuts.test.ts` — add the customizable `toggle-waldo` command with `Command-Shift-Space` / `Control-Shift-Space` defaults.
- `frontend/src/renderer/i18n/key-maps.ts` — type the new shortcut label.
- `frontend/src/renderer/i18n/{en,zh-CN,ja,ko,es,fr,de,pt-BR}.json` — add Waldo rail and Home continuity navigation copy.
- `frontend/src/renderer/lib/preview-mode.ts` — expose a local-only Waldo UI preview predicate.
- `frontend/src/renderer/routes/_shell.tsx` — mount the provider, launcher, keyboard handler, and non-Home shell rail; suppress and restore a session Inspector while Waldo owns the right region.
- `frontend/src/renderer/components/home/HomeShell.tsx` — replace Today’s context pane with Waldo while open and host the full layer on supporting Home routes.
- `frontend/src/renderer/components/home/HomeNavigation.tsx` — expose the completed continuity routes as a secondary group.
- `frontend/src/renderer/components/home/HomeShell.test.tsx` — verify Today composition, one capture, and Home context restoration.
- `frontend/src/renderer/styles.css` — rail sizing, wide/narrow placement, motion, and Home right-region composition.
- `frontend/tsconfig.e2e.json` only if the new Playwright spec reveals a real inclusion gap.

---

### Task 1: Add the global Waldo shortcut contract

**Files:**
- Modify: `frontend/src/shared/shortcuts.ts`
- Modify: `frontend/src/shared/shortcuts.test.ts`
- Modify: `frontend/src/renderer/i18n/key-maps.ts`
- Modify: `frontend/src/renderer/i18n/en.json`
- Modify: `frontend/src/renderer/i18n/zh-CN.json`
- Modify: `frontend/src/renderer/i18n/ja.json`
- Modify: `frontend/src/renderer/i18n/ko.json`
- Modify: `frontend/src/renderer/i18n/es.json`
- Modify: `frontend/src/renderer/i18n/fr.json`
- Modify: `frontend/src/renderer/i18n/de.json`
- Modify: `frontend/src/renderer/i18n/pt-BR.json`

**Interfaces:**
- Consumes: `AppShortcutId`, `APP_SHORTCUTS`, `defaultShortcutBindings`, and renderer override matching.
- Produces: `AppShortcutId = ... | "toggle-waldo"` with a customizable default used by the shell keyboard handler and shortcut settings.

- [ ] **Step 1: Write the failing shortcut tests**

Add to `frontend/src/shared/shortcuts.test.ts`:

```ts
it("opens Waldo with Command-Shift-Space or Control-Shift-Space", () => {
  expect(matchesAppShortcut("toggle-waldo", chord({ key: " ", code: "Space", meta: true, shift: true }), true)).toBe(true);
  expect(matchesAppShortcut("toggle-waldo", chord({ key: " ", code: "Space", ctrl: true, shift: true }), false)).toBe(true);
});

it("does not reuse Island or terminal shortcuts for Waldo", () => {
  expect(matchesAppShortcut("toggle-waldo", chord({ key: "`", code: "Backquote", meta: true }), true)).toBe(false);
  expect(matchesAppShortcut("toggle-waldo", chord({ key: "t", meta: true, shift: true }), true)).toBe(false);
});
```

Extend the shortcut-catalog test to assert the `toggle-waldo` label exists and remains customizable.

- [ ] **Step 2: Run the test and verify RED**

Run:

```bash
npm --prefix frontend test -- --run src/shared/shortcuts.test.ts
```

Expected: FAIL because `toggle-waldo` is not assignable to `AppShortcutId` and has no default binding.

- [ ] **Step 3: Add the minimal shortcut definition**

Add `toggle-waldo` to `AppShortcutId`, the General catalog, and the binding switch:

```ts
case "toggle-waldo":
  return [isMac
    ? binding(" ", { code: "Space", meta: true, shift: true })
    : binding(" ", { code: "Space", ctrl: true, shift: true })];
```

Add `"toggle-waldo": "shortcut.toggle-waldo"` to `shortcutLabelKeys`. Add the translated label “Open Waldo” to every locale catalog so catalog-key parity remains exact.

- [ ] **Step 4: Run focused shortcut and localization tests**

Run:

```bash
npm --prefix frontend test -- --run src/shared/shortcuts.test.ts src/renderer/i18n/instance.test.ts
```

Expected: PASS with no missing shortcut default or locale key.

- [ ] **Step 5: Commit the shortcut checkpoint**

```bash
git add frontend/src/shared/shortcuts.ts frontend/src/shared/shortcuts.test.ts frontend/src/renderer/i18n/key-maps.ts frontend/src/renderer/i18n/*.json
git diff --cached --check
git commit -m "feat: add global Waldo shortcut"
```

---

### Task 2: Build the truthful Waldo rail and bounded-agent preview

**Files:**
- Create: `frontend/src/renderer/components/waldo/WaldoRail.tsx`
- Create: `frontend/src/renderer/components/waldo/WaldoRail.test.tsx`
- Modify: `frontend/src/renderer/lib/preview-mode.ts`
- Modify: `frontend/src/renderer/i18n/{en,zh-CN,ja,ko,es,fr,de,pt-BR}.json`

**Interfaces:**
- Consumes: `contextLabel: string`, `previewEnabled: boolean`, `onClose(): void`, and optional `onReturnToInspector(): void`.
- Produces: one `WaldoRail` that renders either an honest unconfigured state or an explicitly labeled local preview. It owns only local preview selection/status.

- [ ] **Step 1: Write failing truth-boundary tests**

Create `WaldoRail.test.tsx` with these behaviors:

```tsx
it("shows no plausible conversation when Waldo is unconfigured", () => {
  render(<WaldoRail contextLabel="Home · Today" previewEnabled={false} onClose={vi.fn()} />);
  expect(screen.getByRole("heading", { name: "Waldo isn't connected yet" })).toBeInTheDocument();
  expect(screen.getByText("Home and Work remain available without a model connection.")).toBeInTheDocument();
  expect(screen.queryByRole("textbox", { name: "Message Waldo" })).not.toBeInTheDocument();
  expect(screen.queryByText("Preparing your briefing")).not.toBeInTheDocument();
});

it("labels the deterministic agent surface as a non-live preview", () => {
  render(<WaldoRail contextLabel="Home · Today" previewEnabled onClose={vi.fn()} />);
  expect(screen.getByRole("status", { name: "Waldo preview boundary" })).toHaveTextContent("Interaction preview");
  expect(screen.getByText("No model or agent is running")).toBeInTheDocument();
  expect(screen.getByText("Home · Today")).toBeInTheDocument();
  expect(screen.getByRole("region", { name: "Agent activity preview" })).toHaveTextContent("Waiting for your review");
});
```

Add one test that clicks the proposal’s review action and asserts the local status says `Nothing was created, sent, or saved`. Add one test that the activity trace exposes request, bounded completion condition, permitted sources, ordered steps, approval boundary, evidence, and `Returns to Work · no Outcome created`.

- [ ] **Step 2: Run the rail test and verify RED**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/waldo/WaldoRail.test.tsx
```

Expected: FAIL because `WaldoRail` does not exist.

- [ ] **Step 3: Add the preview predicate**

In `preview-mode.ts` add:

```ts
export const usesWaldoUiPreview =
  import.meta.env.VITE_NO_ELECTRON === "1" ||
  import.meta.env.VITE_WALDO_UI_PREVIEW === "1";
```

This variable controls presentation fixtures only. It must not alter `usesPreviewWorkspaceData`, daemon selection, or API behavior.

- [ ] **Step 4: Implement the minimum Waldo rail**

Use three vertical regions:

1. Header: Waldo identity, visible context chip, connection/preview state, Close.
2. Scroll body: one short preview exchange, one native proposal card, and one Dimension-inspired activity trace.
3. Footer: canonical unconfigured explanation or disabled preview composer labeled `Preview composer — messages are not sent`.

The agent activity trace uses literal local preview data with these steps:

```ts
const previewSteps = [
  { label: "Read the attached Home context", state: "evidenced" },
  { label: "Gather the permitted Work sources", state: "evidenced" },
  { label: "Prepare the meeting brief", state: "active" },
  { label: "Wait before any outward action", state: "blocked" },
] as const;
```

Render state labels as text plus icons/shapes. Do not use avatar rosters, percent-complete scores, or a `Done` state.

- [ ] **Step 5: Add localized copy to all catalogs**

Add a `waldo.rail.*` family covering launcher, close, unconfigured state, preview boundary, context, proposal boundary, activity labels, evidence/return path, Inspector tab, disabled composer, and narrow Back. Translate every key in all eight catalogs; do not use English fallback as the implementation.

- [ ] **Step 6: Run rail and localization coverage tests**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/waldo/WaldoRail.test.tsx src/renderer/i18n/instance.test.ts src/renderer/i18n/renderer-coverage.test.ts
```

Expected: PASS and no hardcoded English JSX chrome.

- [ ] **Step 7: Commit the rail component checkpoint**

```bash
git add frontend/src/renderer/components/waldo/WaldoRail.tsx frontend/src/renderer/components/waldo/WaldoRail.test.tsx frontend/src/renderer/lib/preview-mode.ts frontend/src/renderer/i18n/*.json
git diff --cached --check
git commit -m "feat: add truthful Waldo activity rail"
```

---

### Task 3: Give the shell one Waldo owner with exact focus return

**Files:**
- Create: `frontend/src/renderer/components/waldo/WaldoRailContext.tsx`
- Create: `frontend/src/renderer/components/waldo/WaldoRailContext.test.tsx`
- Create: `frontend/src/renderer/components/waldo/WaldoLauncher.tsx`
- Create: `frontend/src/renderer/components/waldo/WaldoShellRail.tsx`
- Modify: `frontend/src/renderer/routes/_shell.tsx`
- Modify: `frontend/src/renderer/components/home/HomeShell.tsx`
- Modify: `frontend/src/renderer/components/home/HomeShell.test.tsx`
- Modify: `frontend/src/renderer/styles.css`

**Interfaces:**
- Consumes: `matchesRendererShortcut("toggle-waldo", event)`, route context, `usesWaldoUiPreview`, and the current session Inspector state.
- Produces: `useWaldoRail()` with `{ isOpen, open(origin?), close(), toggle(origin?), launcherRef }`; one launcher; Home-context replacement; one Work/supporting-route rail.

- [ ] **Step 1: Write failing provider focus tests**

Create a test harness under `WaldoRailProvider` with an origin button, launcher, and conditional rail. Verify:

```tsx
await user.click(screen.getByRole("button", { name: "Open from context" }));
expect(screen.getByRole("region", { name: "Waldo" })).toBeInTheDocument();
await user.click(screen.getByRole("button", { name: "Close Waldo" }));
expect(screen.getByRole("button", { name: "Open from context" })).toHaveFocus();
```

Add cases for keyboard toggle, disconnected origin falling back to the launcher, and Escape not closing when a child marks an approval dialog active.

- [ ] **Step 2: Run provider tests and verify RED**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/waldo/WaldoRailContext.test.tsx
```

Expected: FAIL because the provider does not exist.

- [ ] **Step 3: Implement the provider and launcher**

The provider stores only:

```ts
type WaldoRailState = {
  isOpen: boolean;
  invocationOrigin: HTMLElement | null;
};
```

`open` captures the supplied origin or `document.activeElement` when it is an `HTMLElement`. `close` updates state first, then restores focus in `requestAnimationFrame` to a connected origin, otherwise to `launcherRef.current`. The launcher is a compact icon button with `aria-expanded`, `aria-controls="waldo-rail"`, and the configured shortcut in its tooltip/title.

- [ ] **Step 4: Add the shell wrapper and shortcut handler**

Change the route component to mount a provider without moving the existing shell logic:

```tsx
function ShellRoute() {
  return (
    <WaldoRailProvider>
      <ShellLayout />
    </WaldoRailProvider>
  );
}
```

Inside `ShellLayout`, handle `toggle-waldo` before sidebar/project shortcuts. Mount the launcher at the top right of `<main>` while leaving the Home/Work pill centered.

For non-Home routes, `WaldoShellRail` occupies the right edge. On a session route, record whether the session Inspector was open, close it while Waldo owns the right region, label the header tabs `Waldo | Inspector`, and restore the prior Inspector state when Waldo closes or the Inspector tab is selected.

- [ ] **Step 5: Write failing Home-composition tests**

Wrap `HomeShell` in the provider and verify:

```tsx
expect(screen.getByRole("heading", { name: "Catch Up" })).toBeInTheDocument();
await user.click(screen.getByRole("button", { name: "Open Waldo" }));
expect(screen.getByRole("region", { name: "Waldo" })).toBeInTheDocument();
expect(screen.queryByRole("heading", { name: "Catch Up" })).not.toBeInTheDocument();
expect(screen.getAllByRole("textbox", { name: "Quick Capture" })).toHaveLength(1);
await user.click(screen.getByRole("button", { name: "Close Waldo" }));
expect(screen.getByRole("heading", { name: "Catch Up" })).toBeInTheDocument();
```

Add a supporting-route case showing Waldo as an overlaid full rail while the destination remains mounted underneath.

- [ ] **Step 6: Run Home tests and verify RED**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/home/HomeShell.test.tsx
```

Expected: FAIL because Home does not consume the rail provider.

- [ ] **Step 7: Integrate Home and add responsive CSS**

On Today, render `WaldoRail` in the existing `.home-today-context-pane` when open; otherwise render `HomeContextPanel`. On supporting Home routes, render a `.waldo-home-layer` inside the existing Home surface.

At center-panel widths below 960px, the open Waldo region becomes an inset full-content layer with a visible Back control. At or above 960px, Today keeps the merged 3:2 brief/right-region composition. Work/supporting routes use a 400–440px right rail where possible. Under reduced motion, remove translation and transition delay.

- [ ] **Step 8: Run the focused shell tests**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/waldo/WaldoRailContext.test.tsx src/renderer/components/waldo/WaldoRail.test.tsx src/renderer/components/home/HomeShell.test.tsx src/shared/shortcuts.test.ts
```

Expected: PASS with exact focus return and one Today capture.

- [ ] **Step 9: Commit the shell-integration checkpoint**

```bash
git add frontend/src/renderer/components/waldo frontend/src/renderer/routes/_shell.tsx frontend/src/renderer/components/home/HomeShell.tsx frontend/src/renderer/components/home/HomeShell.test.tsx frontend/src/renderer/styles.css
git diff --cached --check
git commit -m "feat: integrate Waldo across Home and Work"
```

---

### Task 4: Finalize discoverability of the completed Home screens

**Files:**
- Create: `frontend/src/renderer/components/home/HomeNavigation.test.tsx`
- Modify: `frontend/src/renderer/components/home/HomeNavigation.tsx`
- Modify: `frontend/src/renderer/i18n/{en,zh-CN,ja,ko,es,fr,de,pt-BR}.json`
- Modify: `frontend/src/renderer/styles.css` only if group spacing needs a semantic class.

**Interfaces:**
- Consumes: existing `HomeDestination` routes and `variant="sidebar"`.
- Produces: primary `Today` / `Open Loops` navigation plus a subordinate `Review and continuity` group containing Daily Close, Memory Review, and History.

- [ ] **Step 1: Write the failing navigation test**

Create `HomeNavigation.test.tsx`:

```tsx
render(<HomeNavigation destination="memory" variant="sidebar" />);

const primary = screen.getByRole("group", { name: "Primary Home destinations" });
expect(within(primary).getByRole("link", { name: "Today" })).toBeInTheDocument();
expect(within(primary).getByRole("link", { name: "Open Loops" })).toBeInTheDocument();

const continuity = screen.getByRole("group", { name: "Review and continuity" });
expect(within(continuity).getByRole("link", { name: "Daily Close" })).toHaveAttribute("href", "#/home/daily-close");
expect(within(continuity).getByRole("link", { name: "Memory Review" })).toHaveAttribute("aria-current", "page");
expect(within(continuity).getByRole("link", { name: "History" })).toHaveAttribute("href", "#/home/history");
```

Add a panel-variant case proving the secondary destinations do not become an equal horizontal tab bar.

- [ ] **Step 2: Run the navigation test and verify RED**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/home/HomeNavigation.test.tsx
```

Expected: FAIL because only Today and Open Loops are rendered.

- [ ] **Step 3: Implement grouped navigation without new screens**

Keep Today and Open Loops in the current primary list. For the expanded sidebar variant only, render a small text label and secondary links for `daily_close`, `memory`, and `history`. Reuse the existing routes and components. Do not create a generic Home dashboard, separate Catch Up route, Automations route, or Agent roster.

- [ ] **Step 4: Add localized group and route labels**

Add exact keys for `home.visual.navigation.primary`, `home.visual.navigation.continuity`, `home.visual.navigation.dailyClose`, `home.visual.navigation.memory`, and `home.visual.navigation.history` to every locale.

- [ ] **Step 5: Run Home navigation and destination tests**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/home/HomeNavigation.test.tsx src/renderer/components/home/HomeShell.test.tsx src/renderer/components/home/HomeDailyClose.test.tsx src/renderer/components/home/HomeMemoryReview.test.tsx src/renderer/components/home/HomeHistory.test.tsx
```

Expected: PASS; all supporting routes reuse their existing components and render no Quick Capture.

- [ ] **Step 6: Commit the Home discoverability checkpoint**

```bash
git add frontend/src/renderer/components/home/HomeNavigation.tsx frontend/src/renderer/components/home/HomeNavigation.test.tsx frontend/src/renderer/i18n/*.json frontend/src/renderer/styles.css
git diff --cached --check
git commit -m "feat: expose Home continuity screens"
```

---

### Task 5: Verify renderer behavior and visually inspect Electron

**Files:**
- Create: `frontend/e2e/waldo-home-ui.spec.ts`
- Modify: `frontend/tsconfig.e2e.json` only for a demonstrated inclusion gap.

**Interfaces:**
- Consumes: explicit local preview seams, Home routes, global shell, and configured shortcut.
- Produces: repeatable wide/narrow/focus/truth-boundary evidence and local review images.

- [ ] **Step 1: Write failing end-to-end journeys**

Cover:

1. Home Today opens Waldo from the top-right launcher, keeps one Quick Capture, replaces Catch Up, and restores Catch Up and launcher focus on close.
2. `Control-Shift-Space` or `Meta-Shift-Space` toggles the same rail according to the Playwright platform.
3. The explicit renderer preview shows context, proposal boundary, bounded steps, approval gate, evidence, and return path.
4. A proposal click changes only local preview status and never shows saved/created/executed copy.
5. Work opens one Waldo right region and returns to Inspector without two visible rails.
6. At `720x760`, Waldo becomes a full-content layer with Back, no horizontal overflow, and exact focus return.
7. Home sidebar reaches Daily Close, Memory Review, and History while Today/Open Loops remain primary.
8. Supporting routes contain no expanded Quick Capture.

- [ ] **Step 2: Run the E2E spec and verify RED**

Run:

```bash
npm --prefix frontend run test:e2e -- waldo-home-ui.spec.ts
```

Expected: FAIL until shell placement, preview state, and grouped navigation are complete.

- [ ] **Step 3: Make only test-proven accessibility/layout corrections**

Fix violations demonstrated by the E2E run: unique accessible names, focus order, route preservation, overflow, or hidden competing rails. Do not add decorative animation, a second composer, agent avatars, or new dependencies.

- [ ] **Step 4: Run narrow then full frontend gates**

Run:

```bash
npm --prefix frontend test -- --run src/shared/shortcuts.test.ts src/renderer/components/waldo/WaldoRailContext.test.tsx src/renderer/components/waldo/WaldoRail.test.tsx src/renderer/components/home/HomeNavigation.test.tsx src/renderer/components/home/HomeShell.test.tsx
npm --prefix frontend test -- --run src/renderer/components/home/HomeOpenLoops.test.tsx src/renderer/components/home/HomeDailyClose.test.tsx src/renderer/components/home/HomeMemoryReview.test.tsx src/renderer/components/home/HomeHistory.test.tsx src/renderer/i18n/instance.test.ts src/renderer/i18n/renderer-coverage.test.ts
npm --prefix frontend run typecheck
npm --prefix frontend run typecheck:e2e
npm --prefix frontend run test:e2e -- home-work-mode.spec.ts home-today-layout.spec.ts home-personal-agent-flows.spec.ts waldo-home-ui.spec.ts
npm --prefix frontend run build
git diff --check
```

Expected: all commands exit 0. Record any pre-existing environment or packaging failure exactly and do not weaken a gate.

- [ ] **Step 5: Inspect the actual Electron app wide and narrow**

Start the development app with the explicit local visual seam:

```bash
VITE_WALDO_UI_PREVIEW=1 npm --prefix frontend run dev
```

Use the repository-required `kennel preview` path from inside the session. Inspect at `1512x982` and `720x760`:

- Today with Catch Up, Waldo open, and Catch Up restored;
- Open Loops and each supporting continuity destination;
- Work with Waldo and Inspector return;
- preview proposal and agent activity trace;
- canonical unconfigured state in a second launch without `VITE_WALDO_UI_PREVIEW`;
- visible focus, reduced motion, and no overflow.

Save review images under `/tmp/waldo-global-rail-visual-review-2026-08-23/`. Do not commit them.

- [ ] **Step 6: Review the complete diff and commit the verification checkpoint**

```bash
git status --short
git diff origin/beta...HEAD --stat
git diff origin/beta...HEAD --check
git add frontend/e2e/waldo-home-ui.spec.ts frontend/tsconfig.e2e.json
git diff --cached --name-only
git commit -m "test: verify Waldo Home UI flows"
```

Stage `frontend/tsconfig.e2e.json` only if it changed for a proven typecheck reason. Confirm no daemon, migration, generated API, research asset, credential, screenshot, or user-owned file is staged.

## Completion boundary

This UI iteration is complete only when the actual Electron app has been inspected at wide and narrow sizes and the canonical no-preview launch shows no plausible conversation or agent activity. It still does not make Waldo conversation functional, persist Home, or unblock issue #23. Durable capture → confirm → restart → disposition → read-back remains owned by the later daemon implementation after issue #21 lands on `beta`.

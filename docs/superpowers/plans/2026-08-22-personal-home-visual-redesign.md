# Personal Home Visual Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the generic Personal Home fixture lists with the approved adaptive brief, global Home/Work pill, contextual capture, focused Catch Up/provenance review, and continuity-led Daily Close/History experience.

**Architecture:** Keep the renderer fixture-only. A global shell component owns mode navigation; focused Home components consume richer typed fixtures and emit only local preview interactions. Existing daemon, API, storage, Work runtime, project/session, and canonical responsibility boundaries remain untouched.

**Tech Stack:** Electron renderer, React 19, TypeScript, TanStack Router, Tailwind utility classes and existing Kennel tokens, Vitest, Testing Library.

**Spec:** `docs/superpowers/specs/2026-08-22-personal-home-visual-redesign.md`

## Global Constraints

- Work only in `/Users/shivanshfulper/.codex/worktrees/issue-18-home-v0-routes/Waldo-Kennel` on `codex/issue-18-home-v0-routes`.
- Keep all work local: no push, PR, merge, deploy, release, or GitHub mutation.
- This plan is frontend-only; do not add daemon routes, DTOs, migrations, persistence, capture workers, memory admission, agent execution, or harness wiring.
- Preserve truthful `ready`, `partial`, `capture_off`, `stale`, and `offline` fixture states.
- Preserve exact focus and scroll return from Catch Up and provenance.
- Reuse existing tokens and component conventions; add no dependency.
- Do not present preview interactions as saved, canonical, accepted, verified, executed, or closed.

---

### Task 1: Global Home/Work mode switch and contextual sidebar

**Files:**
- Create: `frontend/src/renderer/components/HomeWorkModeSwitch.tsx`
- Create: `frontend/src/renderer/components/HomeWorkModeSwitch.test.tsx`
- Modify: `frontend/src/renderer/routes/_shell.tsx`
- Modify: `frontend/src/renderer/components/Sidebar.tsx`
- Modify: `frontend/src/renderer/components/Sidebar.test.tsx`
- Modify: `frontend/src/renderer/components/home/HomeNavigation.tsx`

**Interfaces:**
- Consumes: TanStack Router pathname/history and existing Home destination routes.
- Produces: `HomeWorkModeSwitch`, a shell-level segmented navigation control that retains the last observed Home and Work paths for the lifetime of the shell.

- [ ] **Step 1: Write failing mode-switch tests**

```tsx
expect(screen.getByRole("navigation", { name: "Waldo mode" })).toBeInTheDocument();
expect(screen.getByRole("button", { name: "Home" })).toHaveAttribute("aria-pressed", "true");
await user.click(screen.getByRole("button", { name: "Work" }));
expect(historyPush).toHaveBeenCalledWith("/");
```

Add a rerender case proving `/home/open-loops -> /projects/proj-1 -> Home` returns to `/home/open-loops`.

- [ ] **Step 2: Run the focused tests and confirm failure**

Run: `npm --prefix frontend test -- --run src/renderer/components/HomeWorkModeSwitch.test.tsx src/renderer/components/Sidebar.test.tsx`

Expected: FAIL because the global switch does not exist and the sidebar still renders duplicate Home/Work rows and five equal Home destinations.

- [ ] **Step 3: Implement the global control**

Implement `HomeWorkModeSwitch` with:

```tsx
<nav aria-label="Waldo mode">
  <button aria-pressed={mode === "home"} type="button">Home</button>
  <button aria-pressed={mode === "work"} type="button">Work</button>
</nav>
```

Track the last matching Home and Work pathname in refs while the shell remains mounted. Use `router.history.push(path)` so exact dynamic Project/session paths can be restored without weakening route types. Mount the component inside the shell's main content frame and apply `WebkitAppRegion: "no-drag"` on macOS.

Remove the duplicate generic Home/Work sidebar rows. When Home is active, render only Today and Open Loops as primary sidebar destinations; preserve the other routes as addressable contextual/detail screens.

- [ ] **Step 4: Run focused tests**

Run: `npm --prefix frontend test -- --run src/renderer/components/HomeWorkModeSwitch.test.tsx src/renderer/components/Sidebar.test.tsx`

Expected: PASS.

### Task 2: Rich fixture contract and adaptive Today surface

**Files:**
- Modify: `frontend/src/renderer/lib/home-fixture.ts`
- Create: `frontend/src/renderer/components/home/HomeBrief.tsx`
- Create: `frontend/src/renderer/components/home/HomeQuickCapture.tsx`
- Create: `frontend/src/renderer/components/home/HomeAttentionSummary.tsx`
- Modify: `frontend/src/renderer/components/home/HomeShell.tsx`
- Modify: `frontend/src/renderer/components/home/HomeShell.test.tsx`

**Interfaces:**
- Consumes: `HomeFixtureState` with preview-only source, availability, brief, re-entry, attention, waiting, closure, capture, and continuity fields.
- Produces: editorial Today composition and one expanded capture field only for `destination === "today"`.

- [ ] **Step 1: Write failing adaptive-brief tests**

```tsx
expect(screen.getByRole("heading", { name: "What matters now" })).toBeInTheDocument();
expect(screen.getByText("Prepare the revised deck; do not send it yet.")).toBeInTheDocument();
expect(screen.getByText("Meeting audio unavailable from 3:10–3:24 PM")).toBeInTheDocument();
expect(screen.getAllByRole("textbox", { name: "Quick Capture" })).toHaveLength(1);
```

Add cases proving non-Today destinations contain no expanded Quick Capture textbox and degraded availability does not render as an empty state.

- [ ] **Step 2: Run the focused Home test and confirm failure**

Run: `npm --prefix frontend test -- --run src/renderer/components/home/HomeShell.test.tsx`

Expected: FAIL because the existing shell renders placeholder cards and a toggleable preview block.

- [ ] **Step 3: Implement fixture types and focused components**

Define concrete preview types rather than unstructured display cards:

```ts
type HomeCaptureState = "off" | "active" | "partial" | "paused" | "blocked";
type HomeAttentionItem = {
  id: string;
  statement: string;
  proposedMeaning: string;
  sourceSummary: string;
  sourceGap?: string;
};
```

Render date/status, Resume, evidence-bounded Brief, Needs You, Waiting, Ready to Close, and a single labelled text input. Preserve draft text in module-local session state keyed to the Home shell; copy must state `Architecture preview — nothing is saved`.

Use borders and editorial bands for grouping; use one raised decision plane rather than an equal card grid.

- [ ] **Step 4: Run focused Home tests**

Run: `npm --prefix frontend test -- --run src/renderer/components/home/HomeShell.test.tsx`

Expected: PASS.

### Task 3: Focused Catch Up and provenance split pane

**Files:**
- Create: `frontend/src/renderer/components/home/HomeCatchUp.tsx`
- Modify: `frontend/src/renderer/components/home/ProvenanceInspector.tsx`
- Delete: `frontend/src/renderer/components/home/HomeScreenFixture.tsx`
- Modify: `frontend/src/renderer/components/home/HomeShell.tsx`
- Modify: `frontend/src/renderer/components/home/HomeShell.test.tsx`

**Interfaces:**
- Consumes: the first `HomeAttentionItem`, scroll-container ref, and return-control ref.
- Produces: a local `today | catch_up` view state, wide split-pane provenance, narrow replacement provenance, and exact return behavior.

- [ ] **Step 1: Write failing scenario tests**

```tsx
await user.click(screen.getByRole("button", { name: "Review the deck follow-up" }));
expect(screen.getByRole("heading", { name: "Catch Up" })).toBeInTheDocument();
expect(screen.getByText("I'll send Ashish the revised deck tomorrow.")).toBeInTheDocument();
expect(screen.getByText("Prepare the revised deck for Ashish; do not send it.")).toBeInTheDocument();
await user.click(screen.getByRole("button", { name: "Continue in Work" }));
expect(screen.getByText(/preview only/i)).toBeInTheDocument();
```

Keep the current assertions for Escape, originating focus, and exact `scrollTop` restoration.

- [ ] **Step 2: Run the focused test and confirm failure**

Run: `npm --prefix frontend test -- --run src/renderer/components/home/HomeShell.test.tsx`

Expected: FAIL because Catch Up has no real scenario or split-pane source view.

- [ ] **Step 3: Implement focused review**

Build the scenario with distinct labels for `User statement`, `Waldo proposal`, and `Known gap`. Provide preview actions: Correct, Keep as note, Confirm Open Loop, Defer, Dismiss, and Continue in Work. Continue in Work opens a truthful local boundary preview; it does not navigate as though an Outcome exists.

Use `lg:grid-cols-[minmax(0,1fr)_minmax(18rem,0.72fr)]` for wide review and conditional single-pane replacement below `lg`. Respect reduced motion with `motion-reduce:transition-none`.

- [ ] **Step 4: Run focused Home tests**

Run: `npm --prefix frontend test -- --run src/renderer/components/home/HomeShell.test.tsx`

Expected: PASS.

### Task 4: Open Loops, Daily Close, Memory Review, and continuity History

**Files:**
- Create: `frontend/src/renderer/components/home/HomeDestinationView.tsx`
- Modify: `frontend/src/renderer/components/home/HomeShell.tsx`
- Modify: `frontend/src/renderer/components/home/HomeShell.test.tsx`
- Modify: `frontend/src/renderer/routes/_shell.home_.open-loops.tsx`
- Modify: `frontend/src/renderer/routes/_shell.home_.memory.tsx`
- Modify: `frontend/src/renderer/routes/_shell.home_.daily-close.tsx`
- Modify: `frontend/src/renderer/routes/_shell.home_.history.tsx`

**Interfaces:**
- Consumes: fixture destination, availability, responsibility examples, and continuity events.
- Produces: destination-specific editorial views with contextual actions and no repeated expanded capture composer.

- [ ] **Step 1: Write failing destination tests**

```tsx
expect(screen.getByRole("heading", { name: "Open Loops" })).toBeInTheDocument();
expect(screen.getByText("What remains unresolved")).toBeInTheDocument();
expect(screen.queryByRole("textbox", { name: "Quick Capture" })).not.toBeInTheDocument();
expect(screen.getByText("A review never closes a responsibility by itself.")).toBeInTheDocument();
```

Add History assertions for correction, Work link, and re-entry events, plus Memory assertions that retained context remains proposed and correctable.

- [ ] **Step 2: Run the focused test and confirm failure**

Run: `npm --prefix frontend test -- --run src/renderer/components/home/HomeShell.test.tsx`

Expected: FAIL because destinations still share one generic card.

- [ ] **Step 3: Implement destination-specific views**

Open Loops shows meaning, owner, source, recheck, and explicit status. Daily Close presents `What became true`, `What remains unresolved`, source gaps, and local preview choices without mutating anything. History renders a temporal continuity spine of user decisions and derived links, with activity subordinate. Memory states that durable Memory is unavailable and renders a proposal/correction review rather than a profile.

- [ ] **Step 4: Run focused Home tests**

Run: `npm --prefix frontend test -- --run src/renderer/components/home/HomeShell.test.tsx`

Expected: PASS.

### Task 5: Integration, accessibility, and rendered evidence

**Files:**
- Modify: `frontend/src/renderer/__tests__/integration/home-entry.test.tsx`
- Modify: `frontend/src/renderer/components/Sidebar.test.tsx`
- Modify: `frontend/src/renderer/components/HomeWorkModeSwitch.test.tsx`
- Modify: `frontend/src/renderer/components/home/HomeShell.test.tsx`
- Modify: `frontend/src/renderer/i18n/en.json`
- Modify: other locale files only if existing tests require key parity

**Interfaces:**
- Consumes: completed visual components and routes.
- Produces: verified route behavior, keyboard/focus coverage, responsive screenshots, and no source-only visual claim.

- [ ] **Step 1: Add route and regression assertions**

Test Home/Work switching, contextual navigation, direct access to retained detail routes, exact return, one-composer rule, and preservation of Work project/session selection behavior.

- [ ] **Step 2: Run focused frontend tests**

Run: `npm --prefix frontend test -- --run src/renderer/components/HomeWorkModeSwitch.test.tsx src/renderer/components/Sidebar.test.tsx src/renderer/components/home/HomeShell.test.tsx src/renderer/__tests__/integration/home-entry.test.tsx`

Expected: PASS.

- [ ] **Step 3: Run typecheck and build**

Run: `npm run frontend:typecheck`

Expected: exit 0.

Run: `npm --prefix frontend run build:product-ui`

Expected: exit 0.

- [ ] **Step 4: Review the real renderer**

Run: `npm --prefix frontend run dev:web -- --host 127.0.0.1 --port 4173`, then open `http://127.0.0.1:4173/#/home` using `kennel preview` from the session when available.

Capture wide and narrow screenshots. Verify text fit, Home/Work pill focus/selected states, Today hierarchy, split provenance, direct route rendering, reduced-motion behavior, and no console errors.

- [ ] **Step 5: Run final local verification**

Run: `npm --prefix frontend test -- --run`

Expected: all frontend tests pass.

Run: `git diff --check`

Expected: no output.

No push, PR, merge, deployment, release, or GitHub mutation follows this task.

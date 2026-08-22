# Personal Home Complete Visual Flows Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the approved Home Today aesthetic into a coherent, high-utility Personal Home experience: adaptive day chapters and contextual review on Today, responsibility detail in Open Loops, explicit Closure and Daily Close, governed Memory Review, and continuity-first History.

**Architecture:** Keep this pass entirely inside the Electron renderer and backed by deterministic fixtures. Separate the editorial `HomeDayPhase` from the event-driven `HomeContextFlow`, then let dedicated screen components consume one typed `HomeFixtureState`. Today remains the primary adaptive surface; Open Loops remains the only other primary Home destination; Closure, Memory Review, and History remain contextual/supporting routes. Components may change local preview state, but no interaction creates canonical responsibility, memory, Work execution, acceptance, or closure.

**Tech Stack:** Electron, React 19, TypeScript, TanStack Router, existing Kennel tokens and CSS/Tailwind utilities, Vitest, Testing Library, Playwright.

**Spec:** `docs/superpowers/specs/2026-08-22-personal-home-visual-redesign.md`

**Research basis:** `docs/research/dimensions-personal-agency-visual-audit-2026-08-22.md`

## Global Constraints

- Work only in `/Users/shivanshfulper/.codex/worktrees/issue-18-home-v0-routes/Waldo-Kennel` on `codex/issue-18-home-v0-routes`.
- Keep all work local: no push, PR, merge, deploy, release, or GitHub mutation.
- Do not stage or commit `docs/research/assets/`; it is the untracked Dimension evidence library.
- Preserve the current Home/Work pill, its last-route behavior, and the reduced Home sidebar. Do not redesign Work in this pass.
- Keep the expanded Quick Capture composer exactly once on Today. Supporting screens use scoped actions such as `Add context`, `Correct this`, and `Add close note`.
- This is a visual fixture slice. Do not add daemon APIs, storage, migrations, capture workers, BYOK/provider calls, memory admission, agent execution, Paxel, Autoresearch, learning-kernel, or harness wiring.
- Never present fixture actions as saved, sent, accepted, verified, remembered, executed, linked to Work, or closed.
- Preserve explicit `ready`, `partial`, `capture_off`, `stale`, and `offline` states. Missing sources are gaps, never proof that nothing changed.
- A user statement outranks Waldo inference. Show statement, proposal, source, freshness, and gap as different fields.
- Clock time selects a default chapter only. It never starts Closure, changes responsibility, or creates a Daily Close receipt.
- Use editorial bands, dividers, compact rows, and one focused decision plane. Avoid equal card grids, generic analytics, urgency scores, streaks, and productivity scoring.
- Keep all controls keyboard accessible, preserve focus on pane transitions, support reduced motion, and validate wide and narrow layouts.

### Integration authorization update · 2026-08-23

The local-only constraint above governed visual implementation. After that work passed its local gate, the user explicitly authorized rebasing this issue branch onto current `origin/beta`, updating required documentation, pushing the issue branch, and opening a PR to `beta`. That authorization does not permit direct pushes to `beta` or `main`, merge, deployment, publication, or release.

---

### Task 1: Add the shared day-phase and contextual-flow fixture contract

**Files:**
- Create: `frontend/src/renderer/lib/home-day-phase.ts`
- Create: `frontend/src/renderer/lib/home-day-phase.test.ts`
- Modify: `frontend/src/renderer/lib/home-fixture.ts`
- Modify: `frontend/src/renderer/components/home/HomeShell.test.tsx`

**Interfaces:**
- Consumes: a local `Date`, destination, availability, and optional deterministic fixture overrides.
- Produces: `HomeDayPhase`, `HomeContextFlow`, phase-specific presentation copy, and richer typed fixture collections for upcoming commitment, closure review, memory candidates, and continuity events.

- [x] **Step 1: Write failing phase-boundary tests**

```ts
expect(resolveHomeDayPhase(new Date(2026, 7, 22, 5, 0, 0))).toBe("morning");
expect(resolveHomeDayPhase(new Date(2026, 7, 22, 11, 59, 59))).toBe("morning");
expect(resolveHomeDayPhase(new Date(2026, 7, 22, 12, 0, 0))).toBe("afternoon");
expect(resolveHomeDayPhase(new Date(2026, 7, 22, 16, 59, 59))).toBe("afternoon");
expect(resolveHomeDayPhase(new Date(2026, 7, 22, 17, 0, 0))).toBe("evening");
expect(resolveHomeDayPhase(new Date(2026, 7, 23, 4, 59, 59))).toBe("evening");
```

Add assertions that an explicit `contextFlow` override survives independently of phase, and that `homeFixture("today")` remains deterministically morning for existing component tests.

- [x] **Step 2: Run the focused tests and confirm failure**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/lib/home-day-phase.test.ts src/renderer/components/home/HomeShell.test.tsx
```

Expected: FAIL because `HomeDayPhase`, `HomeContextFlow`, and the deterministic resolver do not exist.

- [x] **Step 3: Implement pure phase resolution and typed presentation**

```ts
export type HomeDayPhase = "morning" | "afternoon" | "evening";

export type HomeContextFlow =
  | "catch_up"
  | "before_next"
  | "plans_changed"
  | "evening_review"
  | "quiet_focus";

export type HomePhasePresentation = {
  greeting: string;
  briefLabel: string;
  attentionSummary: string;
  primaryListLabel: string;
  suggestionLabel: string;
  finishLabel: string;
  capturePlaceholder: string;
};

export function resolveHomeDayPhase(now: Date): HomeDayPhase {
  const hour = now.getHours();
  if (hour >= 5 && hour < 12) return "morning";
  if (hour >= 12 && hour < 17) return "afternoon";
  return "evening";
}
```

Extend `HomeFixtureState` with `dayPhase`, `contextFlow`, `presentation`, `nextThing`, `closureReview`, and `memoryCandidates`. Keep the current two-argument `homeFixture(destination, availability)` call compatible; add a third options argument for tests and routes.

Morning fixture copy orients around overnight changes and confirmed responsibilities. Afternoon copy says what became true, what changed, and what is still realistic. Evening copy separates evidence, unresolved responsibility, and an invitation to start Closure.

- [x] **Step 4: Run focused tests**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/lib/home-day-phase.test.ts src/renderer/components/home/HomeShell.test.tsx
```

Expected: PASS.

- [x] **Step 5: Commit the contract locally**

```bash
git add frontend/src/renderer/lib/home-day-phase.ts frontend/src/renderer/lib/home-day-phase.test.ts frontend/src/renderer/lib/home-fixture.ts frontend/src/renderer/components/home/HomeShell.test.tsx
git commit -m "feat: model adaptive Personal Home day phases"
```

---

### Task 2: Make Today adaptive without losing the approved composition

**Files:**
- Create: `frontend/src/renderer/components/home/HomeContextPanel.tsx`
- Create: `frontend/src/renderer/components/home/HomeBeforeNext.tsx`
- Create: `frontend/src/renderer/components/home/HomePlansChanged.tsx`
- Create: `frontend/src/renderer/components/home/HomeEveningReview.tsx`
- Create: `frontend/src/renderer/components/home/HomeQuietFocus.tsx`
- Modify: `frontend/src/renderer/components/home/HomeBrief.tsx`
- Modify: `frontend/src/renderer/components/home/HomeCatchUp.tsx`
- Modify: `frontend/src/renderer/components/home/HomeQuickCapture.tsx`
- Modify: `frontend/src/renderer/components/home/HomeShell.tsx`
- Modify: `frontend/src/renderer/routes/_shell.home.tsx`
- Modify: `frontend/src/renderer/components/home/HomeShell.test.tsx`
- Modify: `frontend/src/renderer/__tests__/integration/home-entry.test.tsx`
- Modify: `frontend/src/renderer/styles.css`

**Interfaces:**
- Consumes: `fixture.presentation`, `fixture.contextFlow`, and deterministic route phase.
- Produces: one consistent Today chassis whose left editorial chapter and right contextual review change independently.

- [x] **Step 1: Write failing adaptive Today tests**

```tsx
render(<HomeShell fixture={homeFixture("today", "ready", { dayPhase: "afternoon", contextFlow: "before_next" })} />);
expect(screen.getByRole("heading", { name: "Good afternoon, Shivansh." })).toBeInTheDocument();
expect(screen.getByRole("heading", { name: "Afternoon brief" })).toBeInTheDocument();
expect(screen.getByRole("heading", { name: "Before your next thing" })).toBeInTheDocument();
expect(screen.getAllByRole("textbox", { name: "Quick Capture" })).toHaveLength(1);
```

Add separate cases for:

- morning + Catch Up;
- afternoon + Before your next thing;
- afternoon + Plans changed;
- evening + Evening review, with `Start Closure` linking to `#/home/daily-close`;
- quiet focus, with no artificial Needs You queue;
- `capture_off`, `partial`, `stale`, and `offline` copy;
- `Correct`, provenance, and Continue in Work boundaries remaining preview-only.

- [x] **Step 2: Run the focused tests and confirm failure**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/home/HomeShell.test.tsx src/renderer/__tests__/integration/home-entry.test.tsx
```

Expected: FAIL because Today is hard-coded to Morning Brief and Catch Up.

- [x] **Step 3: Read all Today labels from the fixture presentation**

Remove hard-coded `Good morning`, `Morning brief`, and `Finish morning brief` strings from `HomeBrief` and `HomeCatchUp`. Keep the same type scale, wide split, date band, list rhythm, and bottom capture anchor that the user approved.

`HomeQuickCapture` receives the phase-specific placeholder:

```tsx
<HomeQuickCapture placeholder={fixture.presentation.capturePlaceholder} />
```

Use:

- Morning: `Capture a thought…`
- Afternoon: `Note what changed…`
- Evening: `Leave context for tomorrow…`

- [x] **Step 4: Implement a contextual right-pane dispatcher**

```tsx
export function HomeContextPanel(props: HomeContextPanelProps) {
  switch (props.fixture.contextFlow) {
    case "catch_up": return <HomeCatchUp {...props} />;
    case "before_next": return <HomeBeforeNext {...props} />;
    case "plans_changed": return <HomePlansChanged {...props} />;
    case "evening_review": return <HomeEveningReview {...props} />;
    case "quiet_focus": return <HomeQuietFocus {...props} />;
  }
}
```

`HomeBeforeNext` shows the upcoming commitment, exact promises, open questions, source strength, and a `Prepare in Work` preview. `HomePlansChanged` shows the superseded assumption, new fact, and explicit `Keep`, `Defer`, or `Release` proposals. `HomeEveningReview` separates `What became true` from `Still unresolved` and invites—but never starts—Closure. `HomeQuietFocus` is a calm, compact state with a single Resume action.

- [x] **Step 5: Make the live route choose a phase once**

In `_shell.home.tsx`, freeze `new Date()` with a lazy `useState` initializer so rerenders do not change chapter underneath the user, then choose the default context mapping:

```ts
const defaults: Record<HomeDayPhase, HomeContextFlow> = {
  morning: "catch_up",
  afternoon: "before_next",
  evening: "evening_review",
};

const [openedAt] = useState(() => new Date());
const dayPhase = resolveHomeDayPhase(openedAt);
```

Do not let individual components call `new Date()` or recalculate while mounted. Later daemon projection can replace this route-owned default without rewriting the views.

- [x] **Step 6: Run focused tests**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/home/HomeShell.test.tsx src/renderer/__tests__/integration/home-entry.test.tsx
```

Expected: PASS.

- [x] **Step 7: Commit the adaptive Today slice locally**

```bash
git add frontend/src/renderer/components/home/HomeContextPanel.tsx frontend/src/renderer/components/home/HomeBeforeNext.tsx frontend/src/renderer/components/home/HomePlansChanged.tsx frontend/src/renderer/components/home/HomeEveningReview.tsx frontend/src/renderer/components/home/HomeQuietFocus.tsx frontend/src/renderer/components/home/HomeBrief.tsx frontend/src/renderer/components/home/HomeCatchUp.tsx frontend/src/renderer/components/home/HomeQuickCapture.tsx frontend/src/renderer/components/home/HomeShell.tsx frontend/src/renderer/components/home/HomeShell.test.tsx frontend/src/renderer/routes/_shell.home.tsx frontend/src/renderer/__tests__/integration/home-entry.test.tsx frontend/src/renderer/styles.css
git commit -m "feat: adapt Personal Home across the day"
```

---

### Task 3: Turn Open Loops into a responsibility desk

**Files:**
- Create: `frontend/src/renderer/components/home/HomeOpenLoops.tsx`
- Create: `frontend/src/renderer/components/home/HomeOpenLoops.test.tsx`
- Modify: `frontend/src/renderer/components/home/HomeDestinationView.tsx`
- Modify: `frontend/src/renderer/lib/home-fixture.ts`
- Modify: `frontend/src/renderer/styles.css`

**Interfaces:**
- Consumes: confirmed fixture Open Loops grouped as `attention`, `waiting`, and `ready_to_close`.
- Produces: a wide list/detail responsibility desk and a stacked narrow layout with local selection only.

- [x] **Step 1: Write failing responsibility-detail tests**

```tsx
render(<HomeShell destination="open_loops" fixture={homeFixture("open_loops")} />);
expect(screen.getByRole("heading", { name: "Open Loops" })).toBeInTheDocument();
expect(screen.getByRole("tab", { name: "Needs attention 1" })).toHaveAttribute("aria-selected", "true");
await user.click(screen.getByRole("button", { name: "Deck follow-up" }));
expect(screen.getByText("Prepare the revised deck for Ashish; do not send it yet.")).toBeInTheDocument();
expect(screen.getByText("User-confirmed")).toBeInTheDocument();
expect(screen.getByText("Recheck tomorrow morning")).toBeInTheDocument();
```

Add assertions that `Continue in Work` reveals `No Work Outcome or responsibility link has been created`, and that `Correct this`/`Add context` remain local preview controls.

- [x] **Step 2: Run the focused test and confirm failure**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/home/HomeOpenLoops.test.tsx
```

Expected: FAIL because Open Loops is a generic flat list with no selection or detail plane.

- [x] **Step 3: Enrich the fixture and implement the desk**

Extend `HomeOpenLoopFixture` with:

```ts
type HomeOpenLoopState = "attention" | "waiting" | "ready_to_close";

type HomeOpenLoopFixture = {
  id: string;
  label: string;
  meaning: string;
  owner: string;
  state: HomeOpenLoopState;
  trigger: string;
  recheck: string;
  sourceStrength: string;
  sourceSummary: string;
  lastConfirmedAt: string;
};
```

Render a compact state strip above a two-pane layout. The left side is a restrained responsibility index, not a task board. The right side explains meaning, ownership, trigger/recheck, provenance, known gaps, and available preview actions. Selection is local React state and does not alter the fixture.

At narrow widths, selecting a row replaces the index with detail and exposes a labelled `Back to Open Loops` control that restores focus to the selected row.

- [x] **Step 4: Run focused tests**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/home/HomeOpenLoops.test.tsx
```

Expected: PASS.

- [x] **Step 5: Commit the Open Loops slice locally**

```bash
git add frontend/src/renderer/components/home/HomeOpenLoops.tsx frontend/src/renderer/components/home/HomeOpenLoops.test.tsx frontend/src/renderer/components/home/HomeDestinationView.tsx frontend/src/renderer/lib/home-fixture.ts frontend/src/renderer/styles.css
git commit -m "feat: add Personal Home responsibility detail"
```

---

### Task 4: Build explicit Closure and a truthful Daily Close preview receipt

**Files:**
- Create: `frontend/src/renderer/components/home/HomeDailyClose.tsx`
- Create: `frontend/src/renderer/components/home/HomeDailyClose.test.tsx`
- Modify: `frontend/src/renderer/components/home/HomeDestinationView.tsx`
- Modify: `frontend/src/renderer/lib/home-fixture.ts`
- Modify: `frontend/src/renderer/styles.css`

**Interfaces:**
- Consumes: evidence-backed `becameTrue`, unresolved responsibilities, source gaps, and tomorrow re-entry choices.
- Produces: a local `review -> preview_receipt` flow; no fixture or canonical state mutation.

- [x] **Step 1: Write failing Closure tests**

```tsx
render(<HomeShell destination="daily_close" fixture={homeFixture("daily_close")} />);
expect(screen.getByRole("heading", { name: "Close the day deliberately" })).toBeInTheDocument();
expect(screen.getByRole("heading", { name: "What became true" })).toBeInTheDocument();
expect(screen.getByRole("heading", { name: "What remains unresolved" })).toBeInTheDocument();
expect(screen.getByText("Meeting audio was unavailable from 3:10–3:24 PM.")).toBeInTheDocument();
```

Exercise local disposition controls, then:

```tsx
await user.click(screen.getByRole("button", { name: "Review complete — preview Daily Close" }));
expect(screen.getByRole("heading", { name: "Daily Close preview" })).toBeInTheDocument();
expect(screen.getByText("Preview receipt — nothing was saved or closed")).toBeInTheDocument();
expect(screen.getByText("Resume from the corrected deck follow-up")).toBeInTheDocument();
```

- [x] **Step 2: Run the focused test and confirm failure**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/home/HomeDailyClose.test.tsx
```

Expected: FAIL because Daily Close is a static two-column fixture without a guided review or receipt stage.

- [x] **Step 3: Implement the guided review**

Use a small local state machine:

```ts
type DailyCloseStage = "review" | "preview_receipt";
type PreviewDisposition = "still_open" | "defer" | "release" | "resume_tomorrow";
```

The review must show:

- evidence-backed changes;
- unresolved responsibilities;
- known source gaps;
- one selected local disposition for each unresolved item;
- a scoped `Add close note` action;
- an explicit tomorrow re-entry choice.

Changing a disposition only alters local component state. The receipt restates the reviewed facts, selected preview dispositions, source gaps, and re-entry, with links back to Today and History. It never uses celebration confetti, a score, a streak, `All done`, or language implying canonical closure.

- [x] **Step 4: Run focused tests**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/home/HomeDailyClose.test.tsx
```

Expected: PASS.

- [x] **Step 5: Commit the Closure slice locally**

```bash
git add frontend/src/renderer/components/home/HomeDailyClose.tsx frontend/src/renderer/components/home/HomeDailyClose.test.tsx frontend/src/renderer/components/home/HomeDestinationView.tsx frontend/src/renderer/lib/home-fixture.ts frontend/src/renderer/styles.css
git commit -m "feat: add deliberate Personal Home closure flow"
```

---

### Task 5: Build Memory Review as a governed candidate queue

**Files:**
- Create: `frontend/src/renderer/components/home/HomeMemoryReview.tsx`
- Create: `frontend/src/renderer/components/home/HomeMemoryReview.test.tsx`
- Modify: `frontend/src/renderer/components/home/HomeDestinationView.tsx`
- Modify: `frontend/src/renderer/lib/home-fixture.ts`
- Modify: `frontend/src/renderer/styles.css`

**Interfaces:**
- Consumes: fixture `HomeMemoryCandidate` records.
- Produces: an inspectable candidate/detail review with local `correct`, `reject`, and `defer` outcomes; no durable memory admission.

- [x] **Step 1: Write failing candidate-boundary tests**

```tsx
render(<HomeShell destination="memory" fixture={homeFixture("memory")} />);
expect(screen.getByRole("heading", { name: "Memory Review" })).toBeInTheDocument();
expect(screen.getByText("Candidate — not memory")).toBeInTheDocument();
expect(screen.getByText("Ashish should receive the deck only after the revision is reviewed.")).toBeInTheDocument();
expect(screen.getByText("Valid until the deck is sent or this instruction is corrected")).toBeInTheDocument();
expect(screen.getByText("Source contains a 14 minute audio gap")).toBeInTheDocument();
```

After clicking Reject, assert `Rejected in this preview — no durable memory changed`. Add matching local outcomes for Correct and Defer.

- [x] **Step 2: Run the focused test and confirm failure**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/home/HomeMemoryReview.test.tsx
```

Expected: FAIL because Memory Review is a single static text block.

- [x] **Step 3: Implement the governed queue**

```ts
type HomeMemoryCandidate = {
  id: string;
  statement: string;
  sourceSummary: string;
  sourceGap?: string;
  validUntil: string;
  sensitivity: "ordinary" | "sensitive";
  uncertainty: string;
};
```

Use the same wide index/detail composition as Open Loops, but keep visual semantics distinct: a memory candidate is context under review, not responsibility. Show origin, validity interval, sensitivity, uncertainty, and source gap before actions. Do not include a functional `Remember this` control in this fixture slice; durable admission belongs to the later backend/memory contract.

- [x] **Step 4: Run focused tests**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/home/HomeMemoryReview.test.tsx
```

Expected: PASS.

- [x] **Step 5: Commit the Memory Review slice locally**

```bash
git add frontend/src/renderer/components/home/HomeMemoryReview.tsx frontend/src/renderer/components/home/HomeMemoryReview.test.tsx frontend/src/renderer/components/home/HomeDestinationView.tsx frontend/src/renderer/lib/home-fixture.ts frontend/src/renderer/styles.css
git commit -m "feat: add governed Personal Home memory review"
```

---

### Task 6: Make History a continuity record with subordinate activity evidence

**Files:**
- Create: `frontend/src/renderer/components/home/HomeHistory.tsx`
- Create: `frontend/src/renderer/components/home/HomeHistory.test.tsx`
- Modify: `frontend/src/renderer/components/home/HomeDestinationView.tsx`
- Modify: `frontend/src/renderer/lib/home-fixture.ts`
- Modify: `frontend/src/renderer/styles.css`

**Interfaces:**
- Consumes: typed continuity events and their source evidence.
- Produces: a temporal continuity spine, event detail inspector, and an optional supporting-activity layer.

- [x] **Step 1: Write failing continuity tests**

```tsx
render(<HomeShell destination="history" fixture={homeFixture("history")} />);
expect(screen.getByRole("heading", { name: "History" })).toBeInTheDocument();
expect(screen.getByText("Continuity, not activity volume")).toBeInTheDocument();
expect(screen.getByText("User correction recorded")).toBeInTheDocument();
expect(screen.getByText("Work link proposed")).toBeInTheDocument();
expect(screen.getByText("Tomorrow re-entry selected")).toBeInTheDocument();
```

Select an event and assert its statement, source, and architecture-preview boundary appear. Switch to `Supporting activity` and assert activity is labelled evidence rather than productivity.

- [x] **Step 2: Run the focused test and confirm failure**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/home/HomeHistory.test.tsx
```

Expected: FAIL because History has only a static timeline and query-string link.

- [x] **Step 3: Enrich continuity events and implement History**

```ts
type HomeContinuityEventKind =
  | "observation"
  | "correction"
  | "work_link_preview"
  | "close_receipt_preview"
  | "reentry";

type HomeContinuityEvent = {
  id: string;
  kind: HomeContinuityEventKind;
  time: string;
  title: string;
  detail: string;
  sourceSummary: string;
  boundary: string;
};
```

Render the event spine as the primary story and a single selected event detail plane beside it on wide screens. `Supporting activity` may show source observations in chronological order, but must not show hours, app rankings, streaks, attention scores, or behavioral personality conclusions.

Use labels and icons/shapes in addition to color. On narrow layouts, detail replaces the spine and returns exact focus to the selected event.

- [x] **Step 4: Run focused tests**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/components/home/HomeHistory.test.tsx
```

Expected: PASS.

- [x] **Step 5: Commit the History slice locally**

```bash
git add frontend/src/renderer/components/home/HomeHistory.tsx frontend/src/renderer/components/home/HomeHistory.test.tsx frontend/src/renderer/components/home/HomeDestinationView.tsx frontend/src/renderer/lib/home-fixture.ts frontend/src/renderer/styles.css
git commit -m "feat: add continuity-first Personal Home history"
```

---

### Task 7: Verify the complete Home flow in renderer and Electron

**Files:**
- Modify: `frontend/e2e/home-today-layout.spec.ts`
- Create: `frontend/e2e/home-personal-agent-flows.spec.ts`
- Modify: `frontend/tsconfig.e2e.json` only if the new spec exposes a real typecheck gap
- Modify: `docs/superpowers/plans/2026-08-22-personal-home-complete-visual-flows.md` to record final checked steps

**Interfaces:**
- Consumes: all Home routes and Today context states through the browser renderer.
- Produces: deterministic navigation, layout, keyboard, accessibility, and boundary verification; no remote effects.

- [x] **Step 1: Write failing end-to-end flow coverage**

Cover these journeys:

1. `Home -> Today morning -> Catch Up -> provenance -> exact return`.
2. deterministic afternoon Today -> Before your next thing.
3. deterministic evening Today -> Start Closure -> review -> preview receipt -> History.
4. `Open Loops -> select responsibility -> Continue in Work preview -> return`.
5. `Memory Review -> reject candidate locally`.
6. Home/Work pill returns to the last meaningful route in each mode.
7. Today has exactly one expanded Quick Capture input; supporting routes have none.
8. narrow viewport preserves labelled navigation, detail return, and no horizontal overflow.

Use a renderer-only query seam such as `?homePhase=afternoon&homeContext=before_next` only in `VITE_NO_ELECTRON` development/test mode. Do not expose test overrides in the packaged Electron runtime.

- [x] **Step 2: Run the new e2e spec and confirm failure**

Run:

```bash
npm --prefix frontend run test:e2e -- home-today-layout.spec.ts home-personal-agent-flows.spec.ts
```

Expected: FAIL until route phase seams, destination flows, and narrow exact-return behavior are complete.

- [x] **Step 3: Complete accessibility and layout polish**

Verify and fix:

- visible focus for every action;
- `aria-current`, `aria-selected`, and status announcements;
- heading order and unique accessible names;
- exact focus return from provenance and narrow detail panes;
- no action depends on hover;
- immediate reduced-motion behavior;
- no overflow at `900x760` and `1512x982`;
- one dominant plane per screen;
- no repeated generic Home header competing with the screen title.

Do not add decorative animation unless it explains a pane transition. Do not add a new UI dependency.

- [x] **Step 4: Run focused and full verification**

Run:

```bash
npm --prefix frontend test -- --run src/renderer/lib/home-day-phase.test.ts src/renderer/components/home/HomeShell.test.tsx src/renderer/components/home/HomeOpenLoops.test.tsx src/renderer/components/home/HomeDailyClose.test.tsx src/renderer/components/home/HomeMemoryReview.test.tsx src/renderer/components/home/HomeHistory.test.tsx src/renderer/__tests__/integration/home-entry.test.tsx src/renderer/components/HomeWorkModeSwitch.test.tsx
npm --prefix frontend run typecheck
npm --prefix frontend run typecheck:e2e
npm --prefix frontend run test:e2e -- home-work-mode.spec.ts home-today-layout.spec.ts home-personal-agent-flows.spec.ts
npm --prefix frontend run build
git diff --check
```

Expected: all commands PASS. If `npm --prefix frontend run build` exposes an unrelated existing packaging/signing failure, record the exact command and error; do not weaken build checks or claim the build passed.

- [x] **Step 5: Review the local Electron build visually**

Start the local development app from the worktree:

```bash
npm --prefix frontend run dev
```

Use `kennel preview` from inside the running session for each Home route, per repository guidance. Inspect Today in morning, afternoon, and evening fixtures plus Open Loops, Closure/Daily Close, Memory Review, and History at wide and narrow widths. Confirm the Home/Work pill remains in global chrome and Work project/session navigation does not leak into Home.

Capture review images under `/tmp/waldo-home-visual-review-2026-08-22/`; do not commit them unless the user explicitly asks.

- [x] **Step 6: Run a final diff review and commit only the intended code**

```bash
git status --short
git diff --stat HEAD
git diff --check
git add frontend/e2e/home-today-layout.spec.ts frontend/e2e/home-personal-agent-flows.spec.ts docs/superpowers/plans/2026-08-22-personal-home-complete-visual-flows.md
git commit -m "test: verify complete Personal Home visual flows"
```

Before committing, inspect the staged list with `git diff --cached --name-only` and confirm `docs/research/assets/` is absent.

### Completion record · 2026-08-23

- Focused Vitest gate: 8 files, 52 tests passed.
- Renderer and e2e TypeScript checks passed.
- Playwright gate: 11 Home/Work and personal-agent journeys passed, including exact narrow-detail focus return and horizontal-overflow checks.
- `npm --prefix frontend run build` packaged the local arm64 Electron app successfully.
- Visual checkpoints covered Today and the supporting Home routes at `1512x982`, plus Open Loops at `720x760`. Review images remain local under `/tmp/waldo-home-visual-review-2026-08-22/`.
- The local Electron development build was restarted from this worktree after verification. No backend, agent, harness, provider, memory persistence, or remote behavior was added.
- The branch was subsequently rebased onto `origin/beta` at `8ebddf3fa`. The route tree preserves both `/work` and `/home/*`; Work-first “Set up Home” now enters `/home`, and the Home/Work mode control uses `/work` as its initial Work return while retaining later meaningful Work routes.

## Final Visual Acceptance Matrix

| Surface | Primary job | Required wide composition | Required narrow behavior | Truth boundary |
| --- | --- | --- | --- | --- |
| Today · Morning | Orient after absence | Editorial brief + focused Catch Up | Stacked brief then review | Candidate is not responsibility |
| Today · Afternoon | Recalibrate before next commitment | Editorial recalibration + meeting/prep context | One readable sequence | Advice is not fact or execution |
| Today · Evening | Transition toward closure | Day evidence + unresolved review | Stacked review + explicit Closure link | Clock does not close the day |
| Open Loops | Understand unresolved responsibility | Responsibility index + selected detail | Detail replacement + exact return | Work handoff is preview only |
| Closure / Daily Close | Reconcile and choose re-entry | Guided review then receipt | One decision at a time | Review changes no canonical state |
| Memory Review | Govern proposed durable context | Candidate index + claim/source detail | Detail replacement + exact return | Candidate is not memory |
| History | Reconstruct how current truth formed | Continuity spine + selected evidence | Event detail + exact return | Activity is supporting evidence only |

## Deferred Backend and Agent Work

After visual approval, create a separate implementation plan for:

- daemon-owned Home day projection and checkpoint events;
- capture grants, source freshness, gaps, pause, and revoke;
- BYOK provider selection and minimum-necessary retrieval packets;
- candidate extraction and explicit Open Loop confirmation;
- Home-to-Work responsibility contracts and AgentSessions;
- Closure dispositions and canonical Daily Close receipts;
- MemoryCandidate admission, correction, expiry, deletion, and retrieval;
- shared scoped learning kernel for Home, Work, and plugin sessions;
- Paxel/session learning and Autoresearch integration in Work/plugin scopes;
- observability, replay fixtures, privacy tests, evals, and adversarial harness coverage.

None of those capabilities may be inferred from completion of this visual plan.

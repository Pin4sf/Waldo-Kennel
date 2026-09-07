# Kennel Island + desktop app — unification plan

> Status: implemented in the working tree; automated tests and macOS packaging pass.
> Native interaction/screenshots on a real notched Mac remain the final manual gate.
> Written 2026-08-22. Decisions in "Decisions taken" were made by the product
> owner in session; do not re-open them without asking.

## Implementation record — 2026-08-22

The production architecture below is now present: one Electron host owns the
workspace, Island panel, and Island settings window. The Island uses the host's
already-verified daemon connection, receives immediate invalidations from both
daemon event streams, stays available offline for media/local utilities, and
routes a double-click or `kennel-app://session/...` intent into the main app.

Implemented recovery paths are the in-Island **Hide** action, global
`Command+backquote`, the app Settings switch, and the Dock menu. Closing the main
window leaves the Island alive; normal Kennel Quit closes the unified process.
The Island remains gated to notched Macs for this release.

Two explicit implementation notes:

- `packages/kennel-island/` is the only build/CI source. The repo-sibling
  `kennel-island/` was not deleted because it had drifted by implementation time
  and contains a sibling-only `.claude/launch.json`; deleting it would discard
  unverified local work. It is not referenced by production wiring.
- No legacy settings are read from macOS Application Support or `~/.ao`. That
  would violate this repository's hard state boundary. Unified Island settings
  start from safe defaults and persist only beneath Kennel's canonical
  `~/.kennel` state tree.

Automated verification completed:

- frontend TypeScript check;
- 2,516 passing frontend tests (6 skipped);
- 219 passing Island desktop/renderer tests;
- Forge production package for macOS arm64, including both renderer targets and
  all three preloads;
- packaged `kennel-notch` and `kennel-haptics` helpers verified as universal
  `arm64 + x86_64` binaries.

Still manual: launch the unified dev app on a notched Mac with the real daemon,
exercise Hide/restore, offline media, immediate approval/session refresh,
double-click routing, main-window close/reopen, fullscreen visibility, and
capture quiet/running/paused/blocked screenshots.

## What we are building

One product, one install, one launch. The Kennel desktop app does the
orchestration; the island is the ambient supervision surface at the notch, for
when the person is doing something else. Today they are two unrelated Electron
apps that cannot even see each other. This plan closes that.

### Decisions taken

1. **Integrate the code.** Not a spec exercise — the end state is a running pair.
2. **Both launch together as a single unit.** The island is not a thing the user
   starts separately. The app owns the process; the island is a window of it.
3. **Double-clicking a session component focuses that session in the app.** The
   main app window comes forward on that session. Do not add an "Open session"
   button; single-click actions inside the island keep their existing approval,
   input, steering, and expansion behaviour.
4. **The island keeps its visual tuning, not a separate design system.** Per the
   owner: it is the same system, tuned for the notch — reduced text hierarchy,
   simpler composition, `#000000` background at all times so the body reads as
   continuous with the physical camera housing. Components are otherwise the
   same. `DESIGN.md`'s "Kennel Island is excluded" line should be reworded to say
   this, rather than implying a second design language.
5. **Island settings stay in the island.** Notch, hover, gesture, media,
   appearance, calibration, and demo controls remain on the island-owned
   settings surface. The app's Settings gets an Island entry for app-level
   visibility and for opening the detailed island settings; it does not duplicate
   those controls.
6. **State changes are immediate.** Approvals and session changes should arrive
   from the daemon event stream, with polling only as recovery rather than the
   primary five-second update path.
7. **The island remains useful without the daemon.** A disconnect must not hide
   or destroy it: media and local utilities keep working, the Kennel portion
   shows its offline/degraded state, and the connection rebinds automatically.
8. **Hiding is not quitting.** The island has a Hide action. It can be restored
   with the global `Command+backquote` shortcut, the app's Island setting, or a
   discoverable Dock-menu action. Closing the main window leaves the island
   running; quitting the unified Kennel process still terminates both surfaces.
9. **The island stays above everything.** Preserve its current visibility over
   fullscreen apps, video, presentations, and screen sharing.

### Non-goals

- Rebuilding the island's visual language. It is done; leave it alone.
- Moving orchestration logic into the island. It reads state and resolves
  approvals; it does not own sessions.
- Windows/Linux island. The island is macOS-only by construction (notch geometry,
  Swift helpers). On other platforms the app ships without it for this build,
  while the renderer/model boundary remains portable for a later non-Mac design.

## Pre-implementation state (archived for context)

At planning time these were two Electron apps that shared a name and nothing
else. This table explains the drift the implementation removed; it is not the
current production shape.

| | Desktop app | Island |
| --- | --- | --- |
| Location | `frontend/` | `packages/kennel-island/` (plus a then-identical repo-sibling copy) |
| Electron | `^33.0.0` (`frontend/package.json:56`) | `^43.4.1` (`packages/kennel-island/package.json:28`) |
| Build | electron-forge + Vite plugin | plain Vite + electron-builder |
| Run file | `~/.kennel/running.json`, `KENNEL_RUN_FILE` (`frontend/src/main.ts:552`, `frontend/src/shared/daemon-discovery.ts:117`) | `~/.ao/running.json`, `AO_RUN_FILE` (`packages/kennel-island/desktop/runtime-helpers.mjs:44`, `:49`) |
| Daemon identity | `kennel-daemon` (`frontend/src/shared/daemon-attach.ts:26`) | `agent-orchestrator-daemon` (`packages/kennel-island/desktop/kennel-service.mjs:5`) |
| App marker | writes `~/.kennel/app-state.json` (`frontend/src/main.ts:1989`) | reads `~/.ao/app-state.json` (`runtime-helpers.mjs:56`) |

### The consequence

**The island could not attach to a running Kennel at planning time.** Two
independent breaks existed, either one sufficient:

1. It reads `~/.ao/running.json`, which nothing writes any more.
2. Even given the right file, `/readyz` returns `service: "kennel-daemon"` and
   the island's identity check (`kennel-service.mjs:521`) rejects it as
   `DAEMON_IDENTITY_MISMATCH`.

Both are renames left behind by the agent-orchestrator → Kennel migration. The
island has been running against fixtures.

### What already exists and should be reused

- The app registers a deep-link scheme, `kennel-app://`
  (`frontend/src/shared/product-identity.ts:4`), handled on `open-url`
  (`frontend/src/main.ts:1925`) and `second-instance` (`:1930`) for WorkOS auth
  callbacks. Session deep-linking is an extension of this path, not new
  infrastructure.
- The island already creates a second `BrowserWindow` for its own settings
  (`desktop/main.mjs:670`), so a multi-window main process is not new to it.
- The daemon's local control plane is loopback HTTP with **no auth token**, so
  sharing a connection across windows costs nothing.
- `packages/kennel-island` and the sibling `kennel-island/` were byte-identical
  during the planning audit (excluding `node_modules`, `dist`, `release`). They
  later drifted, so the sibling was preserved rather than destructively removed.

## Target architecture

**One Electron process. The island is a second renderer of the desktop app.**

```
Kennel.app
├── main process (frontend/src/main.ts)
│   ├── daemon supervisor + attach          ← already there
│   ├── main_window   → orchestration UI    ← already there
│   ├── island_window → notch surface       ← NEW: panel window, macOS only
│   └── island IPC + media/notch/haptics    ← ported from desktop/*.mjs
└── Resources/
    ├── daemon, agent-browser, acp-runtime  ← already there
    └── helpers/kennel-notch, kennel-haptics ← NEW: Swift helpers, macOS only
```

Why one process rather than the app spawning the island binary:

- The run-file drift class of bug disappears by construction. The island stops
  discovering a daemon; it is handed the connection the app already holds.
- Session focus becomes an in-process IPC call rather than a URL round-trip
  through the OS.
- One thing to sign, notarize, update, and crash-report.
- The island's own `kennel-service.mjs` (752 lines of daemon HTTP client) becomes
  a thin adapter over the trusted daemon base URL the app owns. Preserve its
  runtime validation; the existing OpenAPI client is renderer-scoped, not a
  main-process dependency.

The cost is that the island must run on **Electron 33**, not 43. Audit its main
process for APIs newer than 33 before porting (`setHiddenInMissionControl`,
`setWindowButtonVisibility`, `type: "panel"` are all older than 33 and fine —
check the rest). If something genuinely requires 43, raise it rather than
bumping the app's Electron as a side effect.

## Phases

Each phase lands on its own branch off `beta` and is independently shippable.
Do not start a phase before the previous one is merged — phase 2 rewrites files
phase 1 touches.

---

### Phase 0 — one copy of the island

**Implementation status:** canonical build/CI ownership is complete. The sibling
copy is intentionally retained but excluded from production for the preservation
reason in the implementation record above.

**Why first:** every later phase edits island files. Two copies means editing the
wrong one.

- Delete the repo-sibling `kennel-island/` directory. `packages/kennel-island/`
  is the truth. (Confirm the diff is still clean before deleting.)
- Add `packages/kennel-island` to the root workspace/scripts so its tests run in
  CI: `npm run test:island` from the repo root, wired into the same job as the
  frontend checks.
- Update `DESIGN.md:17` and `CLAUDE.md:23`: drop the "keeps its own tuned system"
  wording and replace with the tuning description in "Decisions taken" §4 above.
  Delete the reference to the now-gone sibling path.

**Done when:** one island tree exists, its 14 test files run in CI, and no doc
points at a path that does not exist.

---

### Phase 1 — make the island attach (standalone, still two apps)

**Implementation status:** runtime identity and adapter contracts are complete
and covered by tests. The standalone Electron host was subsequently retired, so
the historical standalone screenshot gate is replaced by the unified native QA
gate above.

**Why before the merge:** it is a small, verifiable change that proves the island
renders live Kennel data. Doing it after the window port would mean debugging two
new things at once.

- `desktop/runtime-helpers.mjs:44` — `AO_RUN_FILE` → `KENNEL_RUN_FILE`.
- `desktop/runtime-helpers.mjs:49` — `.ao` → `.kennel`. Dev subdir stays `dev`,
  which already matches the app (`frontend/src/main.ts:553`).
- `desktop/kennel-service.mjs:5` — `agent-orchestrator-daemon` → `kennel-daemon`.
  Better: import the constant rather than restating it, once phase 2 makes that
  possible; for now a comment pointing at
  `frontend/src/shared/daemon-attach.ts:26` is enough.
- `desktop/kennel-service.mjs:448` — same `.ao` → `.kennel` on the fallback path.
- Update every affected test: `runtime-helpers.test.mjs`,
  `kennel-service.test.mjs`, `packaging.test.mjs`.
- Update `packages/kennel-island/README.md` — it documents `~/.ao` and
  `AO_RUN_FILE` in the "Run the desktop app" section.

**Done when:** with Kennel running, `npm run dev:desktop` in
`packages/kennel-island` shows real sessions in the notch — chip, badge count,
peek bar — and an approval raised in the app appears there. Screenshot it.

**Watch for:** this is the first time the island's projection layer
(`src/island/live-adapter.ts`, 1,476 lines) meets real daemon payloads rather
than fixtures. Expect shape mismatches in session status and activity state
mapping. Fix them in the adapter, not by loosening the contract.

---

### Phase 2 — the island becomes a window of the app

**Implementation status:** complete in code and verified in a packaged macOS
application. Native interaction QA remains outstanding.

The main body of work.

**Build wiring**

- Add a renderer entry to `frontend/forge.config.ts:255`:
  `renderer: [{ name: "main_window", … }, { name: "island_window", … }]`, with
  its own Vite config pointing at the island's `index.html`.
- Add the island preload to the `build` array (`forge.config.ts:250-254`)
  alongside `src/preload.ts` and `src/annotate-preload.ts`.
- Keep the renderer, assets, and renderer tests in `packages/kennel-island` as a
  shared production renderer plus browser-only visual lab. The desktop app builds
  it as its `island_window` entry; the package no longer produces or launches a
  second Electron application after this phase.

**Main process port**

Port `packages/kennel-island/desktop/*.mjs` to TypeScript under
`frontend/src/main/island/`, keeping module boundaries:

| From | To | Notes |
| --- | --- | --- |
| `notch-geometry.mjs`, `notch-measure.mjs` | `island/notch.ts` | pure, port as-is |
| `media-activity.mjs`, `media-artwork.mjs` | `island/media.ts` | pure-ish, port as-is |
| `haptics.mjs` | `island/haptics.ts` | port as-is |
| `settings.mjs` | `island/settings.ts` | keep the island-owned schema and settings UI; persist under the unified app's `~/.kennel` state only |
| `kennel-service.mjs` | `island/service.ts` | **shrink**: drop `attachDaemon()` and the run-file read entirely; accept the trusted daemon base URL/port from the app while preserving validation |
| `main.mjs` window/IPC/settings half | `island/window.ts` | window creation, IPC registration, detailed island settings window, visibility, and local utilities |
| `runtime-helpers.mjs` | delete | run-file discovery is the app's job now |
| `preload.cjs` | `frontend/src/island-preload.ts` | keep the channel names (`desktop/preload.cjs:3-25`) so the renderer is untouched |

- Swift helpers: run `packages/kennel-island/scripts/build-helpers.mjs` from the
  frontend's `prepackage`/`premake` chain and ship the two binaries via
  `packagerConfig.extraResource` (`forge.config.ts:59`). Keep the existing
  graceful degradation — no Swift toolchain means no haptics and a menu-bar-derived
  notch, never a build failure.
- Gate everything on `process.platform === "darwin"`. No island window, no media
  polling, no helpers on other platforms.

**Lifecycle**

- Create the island window during app readiness after settings and notch
  measurement. Do not wait for the daemon: media and local utilities remain
  useful while Kennel is connecting or offline.
- A daemon loss or port change keeps the window and last safe snapshot visible,
  marks Kennel state offline/degraded, and rebinds automatically. Drive normal
  updates from the daemon event stream; retain a slow polling fallback.
- Closing the main window leaves the ambient island running. Explicitly quitting
  the unified Kennel process closes both surfaces; there is no island-specific
  Quit action.
- Keep `LSUIElement` **off**. The app has a dock icon; the island window is
  `type: "panel"` + `skipTaskbar: true` and does not need the accessory role
  (`desktop/main.mjs:807-838`).
- Preserve the system-wide summon shortcut (`Command+backquote`) and extend the
  app shortcut infrastructure to register it globally with conflict handling.
  It toggles visibility even while Kennel is unfocused.
- Add a Hide action to the island, a Show/Hide control in app Settings, and a
  Show/Hide Island Dock-menu entry. Teach the shortcut in the hide confirmation
  so a person cannot strand the surface by accident.

**Done when:** `npm run dev` in `frontend/` brings up both the app window and the
notch island, from one process. `packages/kennel-island`'s standalone
electron-builder packaging is deleted or explicitly demoted to a dev-only visual
lab (`npm run dev` browser mode) — say which in the PR.

---

### Phase 3 — double-clicking a session focuses it

**Implementation status:** complete in code with typed renderer events,
allowlisted validation, main-window recreation/focus, and external deep-link
parsing tests. Native double-click QA remains outstanding.

- Replace `openKennelFromMarker` (`desktop/main.mjs:599`) — with one process
   there is no bundle to open. The island IPC handler focuses the main window and
   routes it.
- Add the navigation gesture to the session component itself: double-clicking a
  session row/card routes it. Do not add an "Open session" button. Nested
  single-click actions continue to resolve approvals, input, steering, and
  expansion inside the island.
- Add a main→renderer route message, e.g. `kennel:focus-session` carrying
  `{ projectId, sessionId }`; the renderer's router navigates to that session.
  Validate the ids against the current snapshot before routing; do not trust a
  renderer-supplied id straight into navigation.
- Also accept the same intent over the existing `kennel-app://` scheme
  (`frontend/src/main.ts:1925`) so a session link works from outside the app —
  a notification, a terminal, a browser. Reuse the auth-callback plumbing;
  namespace the path (`kennel-app://session/<projectId>/<sessionId>`) so the
  cloud handler and this one cannot be confused.
- Remove `desktop/runtime-helpers.mjs`'s `resolveKennelAppPath` and the
  `app-state.json` read. The marker stays — `kennel start` still uses it — but
  the island no longer needs it.

**Done when:** double-clicking a session component on the island brings the app
forward on that session's view from background or from a closed main window, and
a `kennel-app://session/…` URL does the same. No new visible navigation button is
introduced.

---

### Phase 4 — coordinated settings and visibility

**Implementation status:** complete in code. Legacy settings migration was
intentionally omitted because reading the old macOS Application Support location
would violate the repository state boundary; see the implementation record.

- Keep every detailed island preference on the island-owned settings surface:
  notch, hover/peek, gestures, media, appearance, calibration, haptics, and demo
  mode.
- Add an "Island" entry to the app's Settings for app-level visibility and an
  action that reveals and opens the detailed island settings. Do not duplicate
  the detailed controls in the app.
- Default visible only on macOS when a notch is detected. This build exposes no
  island on notchless Macs or other platforms.
- Hiding destroys no process and does not disable local utilities permanently;
  it hides the panel until restored by `Command+backquote`, app Settings, or the
  Dock menu. Persist the visibility choice across launches.
- Migrate any existing island settings file into the unified `~/.kennel` state
  on first run, then stop writing the legacy file.

**Done when:** detailed settings remain reachable from the island, the app can
show a hidden island, the global shortcut and Dock menu can always restore it,
and visibility plus detailed preferences survive restart.

## Risks

- **The projection layer meets real data for the first time in phase 1.** Budget
  for it. `live-adapter.ts` and `kennel-projection.ts` encode assumptions about
  session status and activity vocabulary that fixtures never challenged.
- **Electron 33 vs 43.** Audit before porting, do not bump the app's Electron to
  suit the island.
- **The island window is always-on-top over every space, including fullscreen.**
  A crash loop or a stuck interactive region is far more intrusive here than a
  normal window. Keep `setIslandInteractive` (`desktop/main.mjs:352`) faithful in
  the port — a window that stays click-through-false eats clicks meant for the
  menu bar.
- **Sandbox and CSP.** The island renderer runs with `sandbox: true` and a strict
  navigation policy (`desktop/main.mjs:783-800`). The app's renderer config is
  its own; make sure the island entry does not inherit something looser.

## Verification

Primary automated commands, from the repo root:

```bash
npm run frontend:typecheck && npm --prefix packages/kennel-island run test
```

The full frontend suite needs permission to bind its local Unix and loopback
sockets in constrained environments. Packaging is verified with Electron Forge.

Since this is a native visual surface, final review still requires the real
unified app on a notched Mac plus screenshots of quiet, running, paused, and
blocked states. Browser-only `kennel preview` validates the renderer lab but
cannot prove panel level, global shortcut, Dock lifecycle, or notch alignment.

# Kennel Island

A native macOS notch companion for Kennel. The Figma file defines the visual language; the shipped
interface is React/CSS and every production action is backed by Kennel's local daemon.

The package lives inside the Kennel repository and is built by the main desktop app as its Island
renderer. One Kennel process owns the normal workspace window, the notch panel, and the detailed
Island settings window; the Island never starts, stops, or owns the daemon.

## Run the visual lab

```bash
npm install
npm run dev
```

This browser-only lab exposes the visual scenarios and demo data. To run the real Island and Kennel
together, use `npm run dev` from `frontend/`; the unified main process hands its already-verified
daemon connection to the Island.

The global summon shortcut is `Command + backquote`. A hidden Island can also be restored from the
main app's Settings or the Kennel Dock menu. This build enables the Island only on notched Macs.

The browser-only visual state lab remains available with `npm run dev`. Browser scenarios are
explicitly demo data; Electron always uses the live adapter.

## Resting states

At rest the island is a header strip straddling the notch. It reports one thing
at a time, in descending urgency:

| State | Chip | Badge | Meaning |
| --- | --- | --- | --- |
| Quiet | none | none | Nothing running, nothing playing. The body shrinks to the exact size of the camera housing, where black on black leaves the notch looking like plain hardware. |
| Peek | none | none | The pointer has settled on a quiet housing. The body swells a few points on every side with a softer corner — an acknowledgement, not a panel. |
| Media | none | none | Audio is leaving the Mac. Pet and waveform only. |
| Running | blue | blue ring | Sessions working, wanting nothing. |
| Paused | yellow | yellow ring | A turn ended and the session is waiting on you. Nothing is gated. |
| Blocked | orange | orange ring | A session stopped at a permission request, a question, or a choice. |

The badge counts sessions in the presence being shown, not sessions overall. A
session appears in exactly one presence — the most urgent it qualifies for — so
the counts never double-count.

When several presences are live the island shows the most urgent, then rotates
every 3.5 seconds so a queue of approvals never hides the sessions still working
behind it. Rotation holds while the pointer is on the island, and restarts from
the top whenever a new presence appears, so an approval takes the chip
immediately rather than waiting its turn.

Hovering grows a second bar beneath the strip, joined to it by the same concave
fillets that join the strip to the menu bar. Clicking anywhere opens the work
queue.

### The peek bar

The bar is bounded by the **midlines of the outermost chips** — the vertical
centre of the leftmost to the vertical centre of the rightmost — and never by
the strip's edge. A bar that reached the edge would put its own rounded corner
directly under a chip's, and two curves a few points apart read as a mistake.
Stopping at the midlines leaves each chip half overhanging the bar, which is
what makes the pair look milled from one piece.

Within that bound the bar is sized by its content and nothing else. A short
label gets a short bar; only content that cannot fit the bound scrolls. That is
why a track change under a resting cursor resizes the bar rather than starting a
marquee that was not needed.

**Some things do not travel.** In a session the agent glyph, the completion
tick, and the diff are pinned, and the title and branch scroll behind them. They
are the parts read as symbols rather than as words: a tick that slid past would
have to be waited for, and a diff that slid past would be unreadable at any
speed. Each pin sits over a gradient of the bar's own background, so the text
passing underneath never competes with it for contrast.

### Hover zones

The strip carries two kinds of chip, and hovering one says something about that
kind — sessions on the status chip and the count badge, media on the album art
and the waveform. Asking a pointer to hit a 24pt chip would make that a game, so
the strip is divided into full-height zones instead: every point belongs to
whichever kind is nearest, and the boundary falls in the middle of the gap
between two chips of different kinds.

Chips of the same kind on opposite sides of the housing merge into one zone
spanning it, which is why the album art and the waveform — with the camera
housing between them — are one media zone rather than two.

Hovering a subject with nothing to say falls through rather than opening an
empty bar. And when the pointer parks for ten seconds with both a session and
media live, the bar offers the other one, then alternates every six seconds;
any real movement stands the rotation down, because a pointer that is moving is
still choosing.

### The peek

A quiet island is exactly the housing, so there is nothing to hover. The peek is
how it answers anyway: the body swells a few points on every side, its lower
corners soften, and a Force Touch trackpad gets a single `alignment` tap. It is
deliberately not a panel — an accidental pass over the notch should cost nothing.

It is gated on a dwell rather than on hover, because the notch sits on the route
every pointer takes to the menu bar. Without the delay the island would answer
dozens of times an hour to someone who was going somewhere else. Width, height,
delay, and the haptic all live under Settings → Hover, and **Open on hover**
replaces the peek with the full panel for anyone who would rather skip the step.

An awake island never peeks. It is already showing a strip, and swelling that
strip on hover would be a second, competing answer to the same question.

Media never takes the chip and never joins the rotation: it is not a session,
and it has no count. It shows as the waveform, and its ticker appears only when
no session presence is competing for the bar.

### Media

Two questions, answered separately, because macOS no longer answers them
together.

*Is audio playing* comes from power assertions, which name every process holding
the output device awake. Public, permission-free, and equally true for Music,
Spotify, and a video in any browser. A call is excluded: a conversation opens
the microphone alongside the speakers, and an assertion naming both directions
is a call rather than media.

*What is playing* is best effort. `MRMediaRemoteGetNowPlayingInfo` — the API
that would answer this for every source — has required the private
`com.apple.mediaremote.now-playing-info` entitlement since macOS 15.4, and Apple
does not grant it to third parties. Music and Spotify still answer over Apple
events, so those two are asked and nobody else; each asks for Automation consent
the first time. A browser therefore shows the waveform and a generic label. The
island never invents a title it was not told.

Sampling runs in the main process on a 2.5 second interval and reaches the
renderer as a bounded `{ playing, track }` value. Media is host state rather than
Kennel state, so it travels beside the island model instead of inside a snapshot
that is otherwise entirely server-derived.

*What it looks like* is the artwork, and the two players differ again. Music
hands over the bytes, but only through an Apple event returning raw data that
`osascript` cannot print, so the script writes them to a file we name and the
main process reads it back and deletes it. Spotify hands over a URL on its own
CDN instead, so its artwork costs one HTTPS GET — the only outbound request this
app makes. That request is allowlisted to `i.scdn.co`, capped at 2 MB, checked
for a real image content type, and switched off by **Settings → Media → Fetch
album art**. Artwork is resolved once per track rather than once per poll, and
crosses the bridge as a `data:` URI: the renderer's CSP allows `img-src 'self'
data:` and nothing else, and a canvas reading a data URI is never tainted, which
is what lets the peek's accent colour be sampled from it. A cover with no usable
colour — anything black and white — yields no accent rather than a grey one.

*The waveform is generated, not measured.* Reading the machine's actual output
level needs either a virtual audio device the user installs or a ScreenCaptureKit
audio tap with its own permission prompt, and neither is worth a prompt for a
decoration. The four bars run on staggered loops of different lengths, so they
never repeat together.

## The stage

The unified app owns one Island stage: a fixed transparent panel pinned flush with the top of the built-in
display and centred on it, so the notch sits at the centre of the stage. The stage is never resized
when the island changes surface. An OS-level resize cannot be animated and always trails the content
that triggered it, so the island morphs inside the stage instead, and the stage stays large enough
for the widest surface it will ever hold.

The stage ignores the mouse by default, which leaves the menu bar and everything beneath it usable.
Electron keeps forwarding mouse movement while ignoring, so the renderer can see the pointer reach
the island and ask for the clicks back; it hands them straight back when the pointer leaves, or when
the island stops holding focus.

The island body is drawn in CSS — a square-topped, round-bottomed black shape with a concave fillet
on each top corner that curves it into the menu bar. Nothing about the chrome is an image, because a
picture cannot change width with its content, match a measured notch, or animate between states. The
exported Figma frame shells are kept for reference under `design/figma-shells/` and are not built.

### Notch geometry

The housing is measured, not guessed. `desktop/helpers/notch-metrics.swift` compiles to a small
helper that reads `NSScreen.auxiliaryTopLeftArea` and its right-hand twin — the two strips of menu
bar either side of the camera housing — and prints the gap between them. That gap is the notch, in
points, exactly. Electron bridges neither property, and a native addon would put a compiled binary
in the dependency tree and a rebuild in every install, so the island spawns the helper once at
startup instead and again whenever a display changes.

`desktop/notch-geometry.mjs` still derives a width from the menu bar's height, and that derivation
is now the fallback rather than the answer: it runs when the helper is missing (no Swift toolchain
at build time), when it fails, or when it prints something unparseable. It is accurate to a few
points, which is enough that a small error moves the concave fillets rather than exposing the
housing.

Three corrections stack on top, in order of precedence:

| Source | Scope | Use |
| --- | --- | --- |
| `KENNEL_ISLAND_NOTCH_WIDTH` | Operator, width only | Pins an exact width in points on a panel that measures differently from every API describing it. Outranks the measurement. |
| Settings → Notch → Fine tune | User, width and height | Nudges whichever width won, in points per side. This is the control to reach for. |
| Menu-bar derivation | Fallback | Used only when there is no measurement. |

None of them can invent a notch on a display that has none, and a measurement reporting a flat bezel
outranks a tall menu bar — a measurement saying "no housing" is better information than the only
clue the derivation has.

Geometry is recomputed and pushed to the renderer whenever a display is added, removed, or changed.
Turn on **Settings → Notch → Show notch outline** to draw the measured edges over the real housing
with their size in points; it switches itself off when the settings window closes.

## Settings

Detailed settings stay Island-owned: expand the Island and use the gear in the work queue's header.
The main app's Settings can also reveal this same detailed window, but does not duplicate its
notch, hover, media, gesture, or appearance controls. Escape or `Command + W` closes the window.

Preferences live under Kennel's canonical `~/.kennel` state directory and are owned by the unified
main process.
Both windows read the same copy over IPC and the host pushes every change to both, so a slider moving
in the settings window reshapes the notch live — there is no apply button and no restart.

| Tab | What it holds |
| --- | --- |
| Notch | Width and height fine tune, content padding, the calibration outline, demo mode. |
| Hover | The peek's size, dwell, and haptic; open-on-hover; keep-open-on-leave. |
| Media | Album-art fetching, waveform animation. |
| Gestures | Two-finger swipe handling, including inverting the media direction. |

### Adding a preference

A preference is two rows of data, not a component:

1. A field in `SETTINGS_SCHEMA` in `desktop/settings.mjs`, which gives it a type, a default, and a
   range. Reading, clamping, merging, and persisting are all driven off that description.
2. A control in `settingsTabs` in `src/island/settings.ts`, which gives it a label, a slider or a
   checkbox, and the sentence explaining what it is for.

`src/island/settings.test.ts` holds the two sides together: it asserts the renderer's defaults equal
the main process's, that every control names a field the schema actually has, that its kind matches
the stored type, and that no slider can travel outside the range the schema will clamp it to.

Anything crossing the bridge is rebuilt field by field from a shape `desktop/preload.cjs` declares
itself, in both directions — settings arriving, and patches leaving — so neither side can be handed
a key it did not ask for.

## Live behavior

- Reads projects, active sessions, unresolved notifications, and selected chat conversations.
- Shows only server-derived session status and activity.
- Resolves approvals with the exact decision IDs offered by the provider.
- Resolves supported one-field enum/boolean input with the exact schema value.
- Steers only when the authoritative conversation contains a running turn and permits steering.
- Interrupts an in-flight chat turn after an explicit confirmation.
- Loads provider usage from the selected conversation instead of estimating it.
- Opens Kennel for workflows the island cannot safely represent inline.

Double-clicking a session row focuses the unified app directly on that session. The same validated
intent is available externally as `kennel-app://session/<projectId>/<sessionId>`; ordinary nested
approval, input, and steering controls remain single-click actions inside the Island.

## Security boundary

The preload exposes a small typed API. There is no generic HTTP, shell, filesystem, or IPC bridge.
The main process runs two fixed commands for media detection — `pmset -g assertions` and `osascript`
with a literal script — through `execFile` with a fixed argument list and no shell. Neither takes any
renderer input, and neither is reachable from the renderer.
The Electron host accepts only loopback daemon discovery, validates the daemon identity with
`/readyz`, applies response limits and deadlines, and allowlists every route.

Conversation payloads are projected before crossing IPC. Messages, settings, diffs, and raw activity
objects are excluded. Approval context is bounded; if any visible approval field was truncated,
inline decisions are disabled and the request must be reviewed in Kennel.

## Verify and package

```bash
npm test
npm run build
npm run package:mac
```

`package:mac` produces a local unsigned `.app` under `release/`. Distribution outside the local Mac
still requires an Apple Developer signing identity and notarization credentials.

`npm run build:helpers` compiles the two Swift helpers on their own. Both packaging and
`start:desktop` run it first. It is skipped with a note on a machine with no Swift toolchain, and
`desktop/packaging.test.mjs` checks that every helper the script produces is named in the packaged
file allowlist — an unpackaged helper is an app that silently loses haptics and goes back to
guessing the notch.

The peek needs a real pointer, which no unit test can supply, so it has its own walkthrough:

```bash
npm run dev:desktop
```

```bash
node scripts/peek-qa.mjs
```

The script drives the island over CDP — an island started with `--remote-debugging-port=9222` —
moving the pointer onto the housing, measuring the shape from the DOM at each step, and writing a
screenshot per state. It reports rather than fails when the island is awake: the peek belongs to the
quiet island, so a machine playing music has nothing to peek from.

## Figma reference

The implementation follows the visual language of these frames from the Waldo file. Layout is
authored at 1x against real content rather than traced from the 2x frame coordinates, so a long
session title, a long branch name, or an eighth queued session behaves instead of overflowing:

| State | Figma node |
| --- | --- |
| Compact task update | `2738:12575` / `2738:12774` |
| Expanded work queue | `2803:1114` |
| Structured choice | `2738:12728` |
| Permission request | `2795:29749` |
| Usage limits | `2847:2204` |

Stable exported icons and marks are kept under `public/figma`; no runtime view depends on expiring
Figma asset URLs or full-screen image overlays. The frame shells are reference only and live in
`design/figma-shells/`.

import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties } from "react";
import { createDemoIslandAdapter, type DemoScenario } from "./fixtures/island";
import { createMemoryIslandAdapter, useKennelIsland } from "./island/adapter";
import { KennelIsland } from "./island/KennelIsland";
import { createLiveKennelIslandAdapter } from "./island/live-adapter";
import {
  POINTER_LEAVE_GRACE_MS,
  shouldCollapseOnPointerLeave,
} from "./island/stage-rules";
import {
  useHaptics,
  useIslandGestures,
  useKennelSettings,
  useMediaActivity,
  useMediaFocus,
  useMediaLinger,
  useMediaTransport,
  usePointerDwell,
  useStageGeometry,
  useStageInteractivity,
} from "./island/useIslandStage";
import { SettingsApp } from "./settings/SettingsApp";
import type { IslandGesture } from "./island/gestures";
import type { IslandModel, KennelIslandAdapter, QueueIslandModel } from "./island/types";

interface ScenarioLabel {
  id: DemoScenario;
  label: string;
  detail: string;
}

const kennelScenarioLabels: ScenarioLabel[] = [
  { id: "quiet", label: "Quiet", detail: "nothing running" },
  { id: "compact", label: "Resting", detail: "presence rotation" },
  { id: "queue", label: "Work", detail: "session queue" },
  { id: "choice", label: "Choice", detail: "structured input" },
  { id: "permission", label: "Permission", detail: "approval" },
  { id: "usage", label: "Usage", detail: "limits" },
];

const activityScenarioLabels: ScenarioLabel[] = [
  { id: "activity-recording", label: "Voice recording", detail: "elapsed + controls" },
  { id: "activity-delivery", label: "Food delivery", detail: "phase + ETA" },
  { id: "activity-ride", label: "Ride pickup", detail: "arrival + next action" },
  { id: "activity-transit", label: "Transit", detail: "next stop" },
  { id: "activity-flight", label: "Flight", detail: "gate exception" },
  { id: "activity-sports", label: "Sports score", detail: "score + event stream" },
  { id: "activity-focus", label: "Focus timer", detail: "countdown" },
  { id: "activity-workout", label: "Workout", detail: "set + live metric" },
  { id: "activity-charging", label: "EV charging", detail: "progress + ETA" },
  { id: "activity-camera", label: "Camera recording", detail: "elapsed + warning" },
  { id: "activity-weather", label: "Weather", detail: "rain exception" },
  { id: "activity-home", label: "Smart home", detail: "automation beta" },
  { id: "activity-concurrent", label: "Concurrent", detail: "two live activities" },
];

const query = new URLSearchParams(window.location.search);
const forceDesktopPreview = query.has("desktop");
const isNativeDesktopRuntime = window.location.protocol === "kennel:";
// One bundle serves two windows. The host tells them apart with a query, which
// keeps the settings pane out of the island's bundle graph decision entirely:
// same renderer, same preload, different root component.
const isSettingsWindow = query.get("window") === "settings";

const bridgeUnavailableDetail = "Desktop connection unavailable. Restart Kennel Island.";

function bridgeUnavailableQueue(activeTab: QueueIslandModel["activeTab"] = "home"): QueueIslandModel {
  return {
    surface: "queue",
    activeTab,
    pendingCount: 0,
    tasks: [],
    connection: "offline",
    statusMessage: "Kennel Island is unavailable",
    statusDetail: bridgeUnavailableDetail,
    error: bridgeUnavailableDetail,
  };
}

function createBridgeUnavailableAdapter(): KennelIslandAdapter {
  const compact: IslandModel = {
    surface: "compact",
    taskId: "desktop-bridge-unavailable",
    title: "Kennel Island unavailable",
    project: "Kennel",
    branch: "offline",
    agent: "waldo",
    tone: "error",
    phase: "offline",
    presence: [],
    attentionCount: 0,
    connection: "offline",
    detail: bridgeUnavailableDetail,
  };

  return createMemoryIslandAdapter(compact, (model, action) => {
    switch (action.type) {
      case "expand":
        return bridgeUnavailableQueue("work");
      case "set-tab":
        return bridgeUnavailableQueue(action.tab);
      case "collapse":
      case "dismiss":
      case "retry-connection":
        return compact;
      case "open-session":
        return bridgeUnavailableQueue(model.surface === "queue" ? model.activeTab : "home");
      default:
        return model;
    }
  });
}

export function App() {
  const demoAdapter = useMemo(() => createDemoIslandAdapter(), []);
  const bridgeUnavailableAdapter = useMemo(() => createBridgeUnavailableAdapter(), []);
  const liveAdapter = useMemo(() => {
    const desktop = window.kennelDesktop;
    return typeof desktop?.getKennelSnapshot === "function"
      ? createLiveKennelIslandAdapter(desktop)
      : null;
  }, []);
  const isDesktop = forceDesktopPreview || isNativeDesktopRuntime || Boolean(liveAdapter);

  if (isSettingsWindow) return <SettingsApp />;

  if (isDesktop) {
    const desktopAdapter = liveAdapter
      ?? (isNativeDesktopRuntime ? bridgeUnavailableAdapter : demoAdapter);
    return <DesktopIslandApp adapter={desktopAdapter} />;
  }

  return <PrototypeLab adapter={demoAdapter} />;
}

function DesktopIslandApp({ adapter }: { adapter: KennelIslandAdapter }) {
  const { model, dispatch } = useKennelIsland(adapter);
  const islandRef = useRef<HTMLDivElement | null>(null);
  const stage = useStageGeometry();
  // The audio's own truth, and the island's: a pause holds the strip for a few
  // seconds with the bars stopped before anything collapses.
  const liveMedia = useMediaActivity();
  const { media, present: mediaPresent } = useMediaLinger(liveMedia);
  const hovered = useStageInteractivity(islandRef);
  const sendMediaCommand = useMediaTransport();
  const focusMediaApp = useMediaFocus();
  const settings = useKennelSettings();
  const performHaptic = useHaptics();

  // The dwell is what separates "the pointer is here" from "the pointer means
  // it". The notch sits on the route to the menu bar, so without it the island
  // would answer every cursor that passed through on its way somewhere else.
  const settled = usePointerDwell(hovered === true, settings.hover.peekDelayMs);

  // Two-finger swipes over the island, matching what the pointer already does:
  // down opens, up closes, across steps the track.
  //
  // The stage is click-through except while the pointer is on the island, so a
  // wheel event reaching this window is already proof the swipe was made over
  // the island — there is no second hover test to make, and making one against
  // React state only ever dropped gestures that should have landed. The
  // horizontal pair is still gated on something actually playing, so a sideways
  // swipe over a quiet island cannot reach a paused background player.
  //
  // Each recognised gesture taps the trackpad. The hand is on the device by
  // definition — it just made the gesture — so this is the one place feedback
  // is certain to be felt rather than fired at an empty desk.
  const handleGesture = useCallback(
    (gesture: IslandGesture) => {
      if (!settings.gestures.enabled) return;

      switch (gesture) {
        case "open":
          if (!settings.gestures.verticalOpenClose || model.surface !== "compact") return;
          performHaptic("level");
          dispatch({ type: "expand" });
          return;
        case "close":
          if (!settings.gestures.verticalOpenClose || model.surface === "compact") return;
          performHaptic("level");
          dispatch({ type: "dismiss" });
          return;
        case "next-track":
        case "previous-track": {
          if (!settings.gestures.horizontalMedia || !media.playing) return;
          // Left to right takes the next track. `invertMedia` swaps that for
          // someone who reads the gesture as dragging the strip rather than
          // stepping the queue.
          const forward = (gesture === "next-track") !== settings.gestures.invertMedia;
          performHaptic("generic");
          sendMediaCommand(forward ? "next" : "previous");
        }
      }
    },
    [dispatch, media.playing, model.surface, performHaptic, sendMediaCommand, settings.gestures],
  );

  // A tap the moment the pointer arrives, so the island announces that it has
  // taken the cursor before anything on it has had time to move.
  const wasHoveredForHaptic = useRef(false);
  useEffect(() => {
    const isHovered = hovered === true;
    const entered = isHovered && !wasHoveredForHaptic.current;
    wasHoveredForHaptic.current = isHovered;
    if (entered) performHaptic("alignment");
  }, [hovered, performHaptic]);

  useIslandGestures(islandRef, handleGesture);

  const stageStyle = {
    "--stage-width": `${stage.stageWidth}px`,
    "--stage-height": `${stage.stageHeight}px`,
    "--stage-menu-bar-height": `${stage.menuBarHeight}px`,
  } as CSSProperties;

  useEffect(() => {
    document.documentElement.classList.add("is-desktop-island");
    return () => document.documentElement.classList.remove("is-desktop-island");
  }, []);

  // The island is a heads-up display, not an app you switch to. Its window is
  // created non-focusable so clicking a chip never takes the key window away
  // from whatever was being typed in — the caret stays in the chat box, the
  // editor, the terminal. The one surface that needs a caret of its own asks
  // for focus while it is open and gives it straight back on close.
  const needsTextEntry = model.surface === "steer";
  useEffect(() => {
    const desktop = window.kennelDesktop;
    if (typeof desktop?.setFocusable !== "function") return;
    void desktop.setFocusable(needsTextEntry).catch(() => {});
  }, [needsTextEntry]);

  // "Open on hover" skips the peek entirely: the settled pointer opens the
  // panel rather than swelling the housing.
  const openOnHover = settings.hover.openOnHover;
  useEffect(() => {
    if (!openOnHover || !settled || model.surface !== "compact") return;
    dispatch({ type: "expand" });
  }, [dispatch, model.surface, openOnHover, settled]);

  // The grace period covers a pointer that clips a corner on its way to a
  // button; see `shouldCollapseOnPointerLeave` for why absence is not enough.
  const holdOnMouseLeave = settings.hover.holdOnMouseLeave;
  const wasHovered = useRef(false);
  useEffect(() => {
    const collapses = shouldCollapseOnPointerLeave(model.surface, wasHovered.current, hovered);
    wasHovered.current = hovered === true;
    if (!collapses || holdOnMouseLeave) return;

    const timer = window.setTimeout(() => dispatch({ type: "collapse" }), POINTER_LEAVE_GRACE_MS);
    return () => window.clearTimeout(timer);
  }, [dispatch, holdOnMouseLeave, hovered, model.surface]);

  return (
    <main className="island-stage" data-hovered={hovered} style={stageStyle}>
      <div className="island-stage__anchor" ref={islandRef}>
        <KennelIsland
          hovered={hovered === true}
          media={media}
          mediaPresent={mediaPresent}
          model={model}
          settings={settings}
          settled={settled}
          stage={stage}
          onAction={dispatch}
          onFocusMedia={focusMediaApp}
          onHaptic={performHaptic}
        />
      </div>
    </main>
  );
}

// The lab has no desktop host, so the two host-owned inputs are faked here:
// pointer hover comes from the DOM, and media from a switch.
const LAB_TRACK: KennelMediaActivity = {
  playing: true,
  owner: "Music",
  track: {
    title: "Challenge (feat. Juice WRLD)",
    artist: "Young Thug",
    // A real playhead so the scrubber face is reachable in the lab, where there
    // is no player to ask.
    positionSeconds: 92,
    durationSeconds: 205,
    sampledAt: Date.now(),
    seekable: true,
  },
};
// The third media state the lab has to be able to show: something is playing
// and will not say what, which is every Firefox tab and every game.
const LAB_UNKNOWN_AUDIO: KennelMediaActivity = { playing: true, owner: "Firefox", track: null };
const LAB_SILENCE: KennelMediaActivity = { playing: false, owner: null, track: null };

const LAB_MEDIA_STATES = [
  { label: "Play media", media: LAB_SILENCE },
  { label: "Play unattributed audio", media: LAB_TRACK },
  { label: "Stop media", media: LAB_UNKNOWN_AUDIO },
] as const;

function PrototypeLab({ adapter }: { adapter: ReturnType<typeof createDemoIslandAdapter> }) {
  const { model, dispatch } = useKennelIsland(adapter);
  const [hovered, setHovered] = useState(false);
  const [mediaState, setMediaState] = useState<0 | 1 | 2>(0);
  const [selectedScenario, setSelectedScenario] = useState<DemoScenario>("compact");
  // The lab has no host, so `useKennelSettings` returns the defaults — which is
  // exactly what the lab wants, and is what lets the peek be exercised here
  // rather than only on a quiet Mac with a real notch.
  const settings = useKennelSettings();
  const settled = usePointerDwell(hovered, settings.hover.peekDelayMs);

  const selectScenario = (scenario: DemoScenario) => {
    setSelectedScenario(scenario);
    adapter.setScenario(scenario);
  };

  return (
    <main className="prototype-shell">
      <section className="desktop-stage" aria-label="Island preview canvas">
        <div className="desktop-stage__grain" />
        <div
          className="desktop-stage__island"
          onMouseEnter={() => setHovered(true)}
          onMouseLeave={() => setHovered(false)}
        >
          <KennelIsland
            hovered={hovered}
            media={LAB_MEDIA_STATES[mediaState].media}
            mediaPresent={LAB_MEDIA_STATES[mediaState].media.playing}
            model={model}
            settings={settings}
            settled={settled}
            onAction={dispatch}
          />
        </div>
        <button
          className="desktop-stage__media-toggle"
          onClick={() => setMediaState((current) => ((current + 1) % LAB_MEDIA_STATES.length) as 0 | 1 | 2)}
          type="button"
        >
          {LAB_MEDIA_STATES[mediaState].label}
        </button>
        <p className="desktop-stage__hint">⌘~ to summon Kennel</p>
        <div className="desktop-stage__caption">
          <span className="live-dot" />
          frontend-only prototype
        </div>
      </section>

      <section className="state-lab" aria-label="Prototype states">
        <div className="state-lab__intro">
          <div>
            <p className="state-lab__eyebrow">Kennel Island research prototype</p>
            <h1>Dynamic Island activity lab</h1>
          </div>
          <p>
            Compare familiar iPhone ongoing-activity patterns in Kennel&apos;s notch. Every example is
            a fixed local fixture—not a connection to Apple, Uber, Domino&apos;s, or another vendor.
          </p>
        </div>
        <ScenarioGroup
          label="iPhone Live Activity references"
          scenarios={activityScenarioLabels}
          selected={selectedScenario}
          onSelect={selectScenario}
        />
        <ScenarioGroup
          label="Existing Kennel surfaces"
          scenarios={kennelScenarioLabels}
          selected={selectedScenario}
          onSelect={selectScenario}
        />
        <p className="state-lab__tip">
          Click the compact activity to expand it. Hover for the glanceable state; Escape closes an
          open panel. Demo controls only update the local fixture.
        </p>
      </section>
    </main>
  );
}

function ScenarioGroup({
  label,
  scenarios,
  selected,
  onSelect,
}: {
  label: string;
  scenarios: ScenarioLabel[];
  selected: DemoScenario;
  onSelect: (scenario: DemoScenario) => void;
}) {
  return (
    <div className="scenario-group">
      <h2>{label}</h2>
      <div className="scenario-picker">
        {scenarios.map((scenario) => {
          const active = selected === scenario.id;
          return (
            <button
              aria-pressed={active}
              className={active ? "scenario-card is-active" : "scenario-card"}
              key={scenario.id}
              onClick={() => onSelect(scenario.id)}
              type="button"
            >
              <span>{scenario.label}</span>
              <small>{scenario.detail}</small>
            </button>
          );
        })}
      </div>
    </div>
  );
}

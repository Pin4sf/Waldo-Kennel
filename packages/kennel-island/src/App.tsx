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
  useMediaTransport,
  usePointerDwell,
  useStageGeometry,
  useStageInteractivity,
} from "./island/useIslandStage";
import { SettingsApp } from "./settings/SettingsApp";
import type { IslandGesture } from "./island/gestures";
import type { IslandModel, KennelIslandAdapter, QueueIslandModel } from "./island/types";

const scenarioLabels: Array<{ id: DemoScenario; label: string; detail: string }> = [
  { id: "quiet", label: "Quiet", detail: "nothing running" },
  { id: "compact", label: "Resting", detail: "presence rotation" },
  { id: "queue", label: "Work", detail: "session queue" },
  { id: "choice", label: "Choice", detail: "structured input" },
  { id: "permission", label: "Permission", detail: "approval" },
  { id: "usage", label: "Usage", detail: "limits" },
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
  const media = useMediaActivity();
  const hovered = useStageInteractivity(islandRef);
  const sendMediaCommand = useMediaTransport();
  const settings = useKennelSettings();
  const performHaptic = useHaptics();

  // The dwell is what separates "the pointer is here" from "the pointer means
  // it". The notch sits on the route to the menu bar, so without it the island
  // would answer every cursor that passed through on its way somewhere else.
  const settled = usePointerDwell(hovered === true, settings.hover.peekDelayMs);

  // Two-finger swipes over the island, matching what the pointer already does:
  // down opens, up closes, across steps the track. The horizontal pair is gated
  // on something actually playing, so a stray sideways swipe over a quiet
  // island cannot reach a paused player in the background.
  const handleGesture = useCallback(
    (gesture: IslandGesture) => {
      if (!settings.gestures.enabled) return;

      switch (gesture) {
        case "open":
          if (settings.gestures.verticalOpenClose && model.surface === "compact") {
            dispatch({ type: "expand" });
          }
          return;
        case "close":
          if (settings.gestures.verticalOpenClose && model.surface !== "compact") {
            dispatch({ type: "dismiss" });
          }
          return;
        case "next-track":
        case "previous-track": {
          if (!settings.gestures.horizontalMedia || !media.playing) return;
          const forward = gesture === "next-track";
          sendMediaCommand(forward === settings.gestures.invertMedia ? "previous" : "next");
        }
      }
    },
    [dispatch, media.playing, model.surface, sendMediaCommand, settings.gestures],
  );

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
          model={model}
          settings={settings}
          settled={settled}
          stage={stage}
          onAction={dispatch}
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
  track: { title: "Challenge (feat. Juice WRLD)", artist: "Young Thug" },
};
const LAB_SILENCE: KennelMediaActivity = { playing: false, track: null };

function PrototypeLab({ adapter }: { adapter: ReturnType<typeof createDemoIslandAdapter> }) {
  const { model, dispatch } = useKennelIsland(adapter);
  const [hovered, setHovered] = useState(false);
  const [playing, setPlaying] = useState(false);
  // The lab has no host, so `useKennelSettings` returns the defaults — which is
  // exactly what the lab wants, and is what lets the peek be exercised here
  // rather than only on a quiet Mac with a real notch.
  const settings = useKennelSettings();
  const settled = usePointerDwell(hovered, settings.hover.peekDelayMs);

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
            media={playing ? LAB_TRACK : LAB_SILENCE}
            model={model}
            settings={settings}
            settled={settled}
            onAction={dispatch}
          />
        </div>
        <button
          className="desktop-stage__media-toggle"
          onClick={() => setPlaying((current) => !current)}
          type="button"
        >
          {playing ? "Stop media" : "Play media"}
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
            <p className="state-lab__eyebrow">Kennel island</p>
            <h1>Figma state lab</h1>
          </div>
          <p>
            The visuals are isolated from data access. These controls swap mock adapter snapshots;
            the same component can receive daemon-backed snapshots later.
          </p>
        </div>
        <div className="scenario-picker">
          {scenarioLabels.map((scenario) => (
            <button
              aria-pressed={model.surface === scenario.id}
              className={model.surface === scenario.id ? "scenario-card is-active" : "scenario-card"}
              key={scenario.id}
              onClick={() => adapter.setScenario(scenario.id)}
              type="button"
            >
              <span>{scenario.label}</span>
              <small>{scenario.detail}</small>
            </button>
          ))}
        </div>
        <p className="state-lab__tip">
          Tip: click the compact island to expand. Escape closes an open panel; keys 1–4 answer the
          choice prompt.
        </p>
      </section>
    </main>
  );
}

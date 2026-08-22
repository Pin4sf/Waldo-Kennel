import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
  type FormEvent,
  type KeyboardEvent as ReactKeyboardEvent,
  type ReactNode,
  type RefObject,
} from "react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import type {
  ChoiceIslandModel,
  CompactIslandModel,
  IslandPresence,
  IslandPresenceCard,
  IslandAction,
  IslandModel,
  IslandTask,
  PermissionIslandModel,
  QueueIslandModel,
  SteerIslandModel,
  UsageIslandModel,
} from "./types";
import {
  defaultStageGeometry,
  islandRadius,
  islandWidths,
  ISLAND_HEADER_HEIGHT,
} from "./stage-layout";
import {
  ISLAND_CONTENT_BLUR,
  islandContentEnter,
  islandContentVariants,
  islandPanelExitTransition,
  islandPanelTransition,
  islandSizeSpring,
} from "./motion";
import { restingMetricsFor, restingShapeFor, shouldTapFor, type RestingShape } from "./peek";
import {
  peekMaxWidthFor,
  peekWidthFor,
  type PeekSubject,
} from "./peek-layout";
import { orderPresenceCards } from "./presence";
import { providerAccent } from "./providers";
import { defaultKennelSettings } from "./settings";
import { useArtworkAccent, usePeekSubject, usePresenceRotation } from "./useIslandStage";

// Ported from packages/kennel-island (a Vite app, `import.meta.env.BASE_URL`)
// into this Next.js static export, where assets are served from /public.
const FIGMA_ASSET_ROOT = "/figma-island";

interface KennelIslandProps {
  model: IslandModel;
  onAction: (action: IslandAction) => void | Promise<void>;
  stage?: KennelStageGeometry;
  /** Host media state. Drives the waveform and the media ticker. */
  media?: KennelMediaActivity;
  /** The pointer is on the island: reveal the ticker and hold the rotation. */
  hovered?: boolean;
  /**
   * The pointer has dwelled long enough to mean it. Separate from `hovered`
   * because a cursor crossing the notch on its way to the menu bar is not a
   * request for anything.
   */
  settled?: boolean;
  settings?: KennelSettings;
  /** Force Touch feedback, when the shape changes under the pointer. */
  onHaptic?: (pattern: KennelHapticPattern) => void;
  className?: string;
}

const SILENT_MEDIA: KennelMediaActivity = { playing: false, track: null };

const presenceAccent: Record<IslandPresence, string> = {
  blocked: "var(--island-orange)",
  paused: "var(--island-yellow)",
  running: "var(--island-blue)",
};

/**
 * Taps the trackpad the first time the island settles into a shape that
 * deserves it. The ref lags one render behind on purpose: the comparison the
 * tap needs is against the shape being left, not the one just committed.
 */
function useShapeHaptic(shape: RestingShape, onHaptic?: (pattern: KennelHapticPattern) => void) {
  const previous = useRef<RestingShape>(shape);

  useEffect(() => {
    if (shape === previous.current) return;
    const tap = shouldTapFor(previous.current, shape);
    previous.current = shape;
    if (tap) onHaptic?.("alignment");
  }, [onHaptic, shape]);
}

interface FigmaIconProps {
  name: string;
  className?: string;
  alt?: string;
}

function FigmaIcon({ name, className, alt = "" }: FigmaIconProps) {
  return (
    <img
      alt={alt}
      className={className}
      draggable={false}
      src={`${FIGMA_ASSET_ROOT}/${name}`}
    />
  );
}

function runAction(onAction: KennelIslandProps["onAction"], action: IslandAction) {
  void onAction(action);
}

/**
 * A working directory is identified by its tail, not its head, and the island
 * has room for a tail. The full path stays on the control's title and in
 * Kennel, so nothing is lost — only the `/Users/<name>/...` prefix everyone
 * already knows.
 */
function shortenPath(value: string, segments = 3) {
  const parts = value.split("/").filter(Boolean);
  if (parts.length <= segments) return value;
  return `…/${parts.slice(-segments).join("/")}`;
}

function attentionSummary(count: number) {
  return count === 0
    ? "No items need attention"
    : `${count} ${count === 1 ? "item needs" : "items need"} attention`;
}

export function KennelIsland({
  model,
  onAction,
  stage = defaultStageGeometry,
  media = SILENT_MEDIA,
  hovered = false,
  settled = false,
  settings = defaultKennelSettings,
  onHaptic,
  className = "",
}: KennelIslandProps) {
  const reducedMotion = useReducedMotion();

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && model.surface !== "compact") {
        runAction(onAction, { type: "dismiss" });
        return;
      }

      if (model.surface === "choice" && !model.submittingOptionId && /^[1-4]$/.test(event.key)) {
        const option = model.options[Number(event.key) - 1];
        if (option) {
          runAction(onAction, {
            type: "select-choice",
            promptId: model.promptId,
            optionId: option.id,
          });
        }
      }

      if (model.surface === "permission" && event.metaKey && /^[1-9]$/.test(event.key)) {
        if (model.contextTruncated) return;
        const decision = model.decisions?.[Number(event.key) - 1];
        if (!decision || model.submittingDecisionId) return;
        event.preventDefault();
        runAction(onAction, {
          type: "resolve-permission",
          requestId: model.requestId,
          decisionId: decision.id,
        });
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [model, onAction]);

  const resting = model.surface === "compact";
  // Resting sizes itself to its contents, so only the expanded surfaces need
  // the notch clamped to leave room for their clusters.
  const notchWidth = stage.hasNotch
    ? resting
      ? stage.notchWidth
      : Math.min(stage.notchWidth, islandWidths[model.surface] - 160)
    : 96;

  // The resting strip's width is resolved here, not inside `RestingIsland`,
  // because the island has exactly one outer width and it has to be a single
  // animated value across every surface. Sending a strip that widened by one
  // icon and a panel that opened by four hundred points through the same
  // spring is what makes them read as one piece of hardware.
  const cards = orderPresenceCards(resting ? model.presence : []);
  const card = usePresenceRotation(cards, hovered);
  const showsMedia = media.playing;
  const awake = cards.length > 0 || showsMedia || settings.appearance.demoMode;

  // The peek is the dormant island's answer to the pointer. It is gated on the
  // dwell rather than on hover, so a cursor travelling to the menu bar passes
  // straight over a quiet notch without waking anything.
  const peeking = settled && settings.hover.peek && !settings.hover.openOnHover;
  const shape = restingShapeFor({ awake, peeking });
  useShapeHaptic(resting ? shape : "strip", onHaptic);

  // Both clusters are given the same width so the notch cut-out stays centred
  // on the housing even when one side holds more than the other.
  const clusterItems = Math.max(
    (card ? 1 : 0) + (awake ? 1 : 0),
    (showsMedia ? 1 : 0) + (card ? 1 : 0),
  );
  const restingMetrics = restingMetricsFor({
    shape,
    notchWidth,
    notchHeight: stage.notchHeight,
    menuBarHeight: stage.menuBarHeight,
    clusterItems,
    contentPadding: settings.notch.contentPadding,
    peekWidth: settings.hover.peekWidth,
    peekHeight: settings.hover.peekHeight,
  });
  const width = resting ? restingMetrics.width : islandWidths[model.surface];
  const growing = useGrowing(width);

  // The peek's bound comes from the strip above it, so it is resolved here
  // where the strip's width is known rather than inside the bar.
  const peekMaxWidth = peekMaxWidthFor({
    stripWidth: restingMetrics.width,
    contentPadding: settings.notch.contentPadding,
    hasItems: clusterItems > 0,
  });

  const headerRef = useRef<HTMLDivElement | null>(null);
  const { subject } = usePeekSubject({
    headerRef,
    hovered: hovered && resting,
    hasMedia: showsMedia,
    hasSession: Boolean(card),
  });
  const peekSubject = hovered && resting && awake ? subject : null;

  const artwork = useArtworkAccent(media.track?.artwork);

  const style = {
    "--island-radius": `${islandRadius[model.surface]}px`,
    "--island-notch-width": `${Math.max(notchWidth, 0)}px`,
    "--island-notch-height": `${stage.notchHeight}px`,
    "--island-content-padding": `${settings.notch.contentPadding}px`,
    ...(artwork ? { "--island-artwork": artwork } : {}),
  } as CSSProperties;

  return (
    <motion.section
      animate={{ width }}
      aria-label={`Kennel island: ${model.surface}`}
      className={`kennel-island kennel-island--${model.surface} ${className}`}
      data-surface={model.surface}
      initial={false}
      style={style}
      transition={islandSizeSpring(growing, Boolean(reducedMotion))}
    >
      {/* Both surfaces occupy the same grid cell, so the one on its way out
          fades over the one arriving instead of stacking under it and
          doubling the island's height for a frame. */}
      <AnimatePresence initial={false}>
        {resting ? (
          <RestingIsland
            card={card}
            headerRef={headerRef}
            height={restingMetrics.height}
            key="resting"
            media={media}
            model={model}
            peekMaxWidth={peekMaxWidth}
            peekSubject={peekSubject}
            reducedMotion={Boolean(reducedMotion)}
            shape={shape}
            onAction={onAction}
          />
        ) : (
          <ExpandedIsland
            fromHeight={Math.max(ISLAND_HEADER_HEIGHT, stage.menuBarHeight, stage.notchHeight)}
            key="expanded"
            model={model}
            reducedMotion={Boolean(reducedMotion)}
            onAction={onAction}
          />
        )}
      </AnimatePresence>

      {stage.hasNotch && settings.appearance.calibrating ? (
        <NotchCalibrationOutline stage={stage} />
      ) : null}
    </motion.section>
  );
}

/**
 * The measured housing, drawn as an outline over the real one.
 *
 * The fine tune asks the user to match a shape against their own bezel by eye,
 * and a black shape on a black housing gives them nothing to aim at. This is
 * the thing they are actually adjusting, made visible for as long as the
 * settings window is open.
 */
function NotchCalibrationOutline({ stage }: { stage: KennelStageGeometry }) {
  const style = {
    width: `${stage.notchWidth}px`,
    height: `${stage.notchHeight}px`,
  } as CSSProperties;

  return (
    <span aria-hidden="true" className="island-calibration">
      <span className="island-calibration__box" style={style}>
        <span className="island-calibration__label">
          {stage.notchWidth} × {stage.notchHeight}
        </span>
      </span>
    </span>
  );
}

/**
 * Whether a size is on its way up. The ref lags one render behind by design:
 * on the render where the target changes it still holds the size the island is
 * leaving, which is the comparison the spring choice needs.
 */
function useGrowing(size: number) {
  const previous = useRef(size);
  const growing = size >= previous.current;
  useEffect(() => {
    previous.current = size;
  });
  return growing;
}

/**
 * Height of an element as its contents change it. An expanded panel is sized
 * by what is in it, so the only way to spring its height is to watch the
 * content and animate towards what it reports.
 */
function useMeasuredHeight<T extends HTMLElement>() {
  const ref = useRef<T | null>(null);
  const [height, setHeight] = useState<number | null>(null);

  useLayoutEffect(() => {
    const node = ref.current;
    if (!node) return;

    setHeight(node.offsetHeight);
    const observer = new ResizeObserver(() => setHeight(node.offsetHeight));
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  return [ref, height] as const;
}

/* -------------------------------------------------------------------------- */
/* Resting island                                                              */
/* -------------------------------------------------------------------------- */

interface RestingIslandProps {
  model: CompactIslandModel;
  media: KennelMediaActivity;
  /** Resolved by the island, because the outer width has to follow it. */
  card: IslandPresenceCard | null;
  /** Which of the three resting shapes this is. */
  shape: RestingShape;
  /** Outer height for that shape, in points. */
  height: number;
  /** The strip, so the hover zones can be measured off its chips. */
  headerRef: RefObject<HTMLDivElement | null>;
  /** What the peek is talking about, or null when it should not be open. */
  peekSubject: PeekSubject | null;
  /** Midline to midline, the widest the peek may grow. */
  peekMaxWidth: number;
  reducedMotion: boolean;
  onAction: KennelIslandProps["onAction"];
}

/**
 * The island at rest: a header strip straddling the notch and, only while the
 * pointer is on it, a narrower ticker hanging beneath — joined by the same
 * concave fillets that join the strip to the menu bar.
 *
 * With nothing running and nothing playing there is no strip at all. The body
 * collapses to the size of the camera housing, where black on black leaves the
 * notch looking like plain hardware again — and swells by a few points under a
 * settled pointer, which is the only thing that says the hardware is listening.
 */
function RestingIsland({
  model,
  media,
  card,
  shape,
  height,
  headerRef,
  peekSubject,
  peekMaxWidth,
  reducedMotion,
  onAction,
}: RestingIslandProps) {
  const awake = shape === "strip";
  const headerGrowing = useGrowing(height);

  // The accent follows whatever the peek is currently talking about, so the
  // colour under the bar always belongs to the thing named in it.
  const accent = peekSubject === "media"
    ? "var(--island-artwork, var(--island-media))"
    : card
      ? providerAccent(card.provider ?? "unknown").solid
      : presenceAccent.running;

  const restingStyle = {
    "--island-accent": accent,
  } as CSSProperties;

  const expand = () => runAction(onAction, { type: "expand" });

  return (
    <motion.div
      animate="visible"
      className="island-resting"
      data-awake={awake}
      data-shape={shape}
      exit="hidden"
      initial="hidden"
      style={restingStyle}
      transition={reducedMotion ? { duration: 0 } : undefined}
      variants={islandContentVariants}
    >
      {/* Height is animated as a value rather than through a layout
          projection: the strip re-renders on every rotation tick, and a
          projection re-snapshotted mid-flight leaves a scale transform
          behind on whatever it was measuring. */}
      <motion.div
        animate={{ height }}
        className="island-body island-body--header"
        initial={false}
        ref={headerRef}
        transition={islandSizeSpring(headerGrowing, reducedMotion)}
      >
        <span aria-hidden="true" className="island-fillet island-fillet--left" />
        <span aria-hidden="true" className="island-fillet island-fillet--right" />
        {awake ? (
          <RestingHeader card={card} media={media} onActivate={expand} />
        ) : (
          <button
            aria-label="Expand Kennel island"
            className="island-header__notch island-header__notch--bare"
            onClick={expand}
            type="button"
          />
        )}
      </motion.div>

      {/* The bar is one element across every subject, not one per subject. A
          swap changes what is written in it and how wide it has to be; closing
          a bar and opening another one in its place would animate a shape the
          hardware never makes. */}
      <AnimatePresence initial={false}>
        {peekSubject ? (
          <PeekBar
            card={card}
            key="peek"
            maxWidth={peekMaxWidth}
            media={media}
            model={model}
            reducedMotion={reducedMotion}
            subject={peekSubject}
            onActivate={expand}
          />
        ) : null}
      </AnimatePresence>
    </motion.div>
  );
}

function RestingHeader({
  card,
  media,
  onActivate,
}: {
  card: IslandPresenceCard | null;
  media: KennelMediaActivity;
  onActivate: () => void;
}) {
  const countLabel = card
    ? `Open Kennel work queue. ${card.count} ${card.count === 1 ? "session" : "sessions"} ${card.detail.toLowerCase()}`
    : "Open Kennel work queue";

  return (
    <IslandHeader
      notchLabel="Expand Kennel island"
      onNotchActivate={onActivate}
      left={
        <>
          {card ? (
            <button
              aria-label={countLabel}
              className={`island-status island-status--${card.presence}`}
              data-peek="session"
              onClick={onActivate}
              style={providerChipStyle(card)}
              type="button"
            >
              <PresenceGlyph presence={card.presence} />
            </button>
          ) : null}
          <button
            aria-label="Open Kennel"
            className="island-pet"
            data-peek="media"
            onClick={onActivate}
            type="button"
          >
            <MediaArtwork media={media} />
          </button>
        </>
      }
      right={
        <>
          {media.playing ? (
            <span aria-label="Media is playing" className="island-waveform" data-peek="media" role="img">
              <Waveform playing={media.playing} />
            </span>
          ) : null}
          {card ? (
            <button
              aria-label={countLabel}
              className={`island-count island-count--${card.presence}`}
              data-peek="session"
              onClick={onActivate}
              style={providerChipStyle(card)}
              type="button"
            >
              {card.count}
            </button>
          ) : null}
        </>
      }
    />
  );
}

/**
 * The provider's colour on a session chip.
 *
 * The presence colours — orange, yellow, blue — say how urgent a session is.
 * The provider colour says which AI is behind it, and both matter, so the ring
 * carries the presence and the glyph carries the provider rather than one of
 * them being dropped.
 */
function providerChipStyle(card: IslandPresenceCard): CSSProperties {
  const accent = providerAccent(card.provider ?? "unknown");
  return {
    "--island-provider": accent.solid,
    "--island-provider-gradient": accent.gradient,
  } as CSSProperties;
}

/**
 * The album art of whatever is playing.
 *
 * Falls back to the Figma mark when the source will not hand over artwork,
 * which is every browser and every player that is not Music or Spotify. The
 * fallback is deliberately the same size and shape, so the strip does not
 * resize when a track that has art follows one that does not.
 */
function MediaArtwork({ media }: { media: KennelMediaActivity }) {
  const artwork = media.track?.artwork;

  if (!artwork) return <FigmaIcon className="island-pet__art" name="compact-pet.png" />;

  return (
    <img
      alt=""
      className="island-pet__art"
      draggable={false}
      // Artwork arrives as a data URI, so there is no request and no cache to
      // bust; keying on the source is enough to swap it on a track change.
      key={artwork.slice(0, 64)}
      src={artwork}
    />
  );
}

/** Bars in the waveform. Enough to read as a waveform, few enough to stay crisp. */
const WAVEFORM_BARS = 4;

/**
 * The waveform beside the housing.
 *
 * The motion is generated, not measured. Reading the machine's actual output
 * level needs either a virtual audio device the user installs or a
 * ScreenCaptureKit audio tap with its own permission prompt, and neither is
 * worth a prompt for a decoration. So the bars run on staggered loops of
 * different lengths, which never repeat together and read as sound without
 * claiming to be it.
 */
function Waveform({ playing }: { playing: boolean }) {
  return (
    <span className="waveform" data-playing={playing}>
      {Array.from({ length: WAVEFORM_BARS }, (_, index) => (
        <span
          className="waveform__bar"
          key={index}
          style={{ "--bar": String(index) } as CSSProperties}
        />
      ))}
    </span>
  );
}

function PresenceGlyph({ presence }: { presence: IslandPresence }) {
  if (presence === "running") {
    return <FigmaIcon className="island-status__spinner" name="compact-spinner.svg" />;
  }
  if (presence === "paused") {
    return <FigmaIcon className="island-status__pause" name="icon-pause.svg" />;
  }
  return <span>?</span>;
}

/* -------------------------------------------------------------------------- */
/* Peek                                                                        */
/* -------------------------------------------------------------------------- */

interface PeekBarProps {
  subject: PeekSubject;
  card: IslandPresenceCard | null;
  media: KennelMediaActivity;
  model: CompactIslandModel;
  /** Midline to midline. The bar may be narrower; it may never be wider. */
  maxWidth: number;
  reducedMotion: boolean;
  onActivate: () => void;
}

/**
 * The bar that hangs under the strip while the pointer is on the island.
 *
 * It is sized by its content, bounded by the midlines of the chips above it,
 * and only scrolls when the content could not fit that bound — so a track
 * change under a resting cursor resizes the bar rather than starting a marquee
 * that was not needed.
 */
function PeekBar({
  subject,
  card,
  media,
  model,
  maxWidth,
  reducedMotion,
  onActivate,
}: PeekBarProps) {
  const [contentRef, contentWidth] = useContentWidth<HTMLSpanElement>();
  const { width, scrolls } = peekWidthFor({
    // The bound has already been resolved against the strip, so it is passed
    // through rather than recomputed from a width this component cannot see.
    stripWidth: maxWidth,
    contentPadding: 0,
    hasItems: false,
    contentWidth: contentWidth ?? 0,
  });
  const growing = useGrowing(width);

  const label = subject === "media"
    ? media.track
      ? `Now playing. ${media.track.title}`
      : "Now playing"
    : card
      ? `Open work queue. ${card.title}`
      : "Open work queue";

  return (
    <motion.div
      animate={{ width, opacity: 1, y: 0, filter: "blur(0px)" }}
      className="island-peek"
      data-subject={subject}
      exit={{ opacity: 0, y: -8, filter: `blur(${ISLAND_CONTENT_BLUR}px)` }}
      initial={{ width, opacity: 0, y: -8, filter: `blur(${ISLAND_CONTENT_BLUR}px)` }}
      transition={
        reducedMotion
          ? { duration: 0 }
          : { ...islandSizeSpring(growing, false), opacity: islandContentEnter, filter: islandContentEnter }
      }
    >
      <span aria-hidden="true" className="island-fillet island-fillet--left" />
      <span aria-hidden="true" className="island-fillet island-fillet--right" />
      <button aria-label={label} className="peek" onClick={onActivate} type="button">
        {/* `mode="wait"` so the outgoing subject is gone before the incoming
            one is measured: two sets of content mounted at once would report a
            combined width, and the bar would flare to fit both. */}
        <AnimatePresence initial={false} mode="wait">
          <motion.span
            animate="visible"
            className="peek__content"
            exit="hidden"
            initial="hidden"
            key={subject}
            transition={reducedMotion ? { duration: 0 } : undefined}
            variants={islandContentVariants}
          >
            {subject === "session" ? (
              <SessionPeek card={card} contentRef={contentRef} model={model} scrolls={scrolls} />
            ) : (
              <MediaPeek contentRef={contentRef} scrolls={scrolls} track={media.track} />
            )}
          </motion.span>
        </AnimatePresence>
      </button>
    </motion.div>
  );
}

/**
 * Natural width of the peek's content, in points.
 *
 * `scrollWidth` and not `getBoundingClientRect`: the rail clips, so the box the
 * content is drawn in is the bar's width rather than the content's, and the
 * bar's width is the thing being decided here.
 */
function useContentWidth<T extends HTMLElement>() {
  const ref = useRef<T | null>(null);
  const [width, setWidth] = useState<number | null>(null);

  useLayoutEffect(() => {
    const node = ref.current;
    if (!node) return;

    const read = () => setWidth((current) => {
      const next = Math.ceil(node.scrollWidth);
      return current === next ? current : next;
    });

    read();
    const observer = new ResizeObserver(read);
    observer.observe(node);
    return () => observer.disconnect();
  });

  return [ref, width] as const;
}

/**
 * Media in the peek: title and artist, and nothing pinned.
 *
 * Media has no fixed elements because it has no second reading. A session's
 * diff has to stay put — it is a number you glance at — but a track's artist
 * means nothing without the title beside it, so the pair travels together.
 */
function MediaPeek({
  track,
  scrolls,
  contentRef,
}: {
  track: KennelMediaTrack | null;
  scrolls: boolean;
  contentRef: RefObject<HTMLSpanElement | null>;
}) {
  return (
    <PeekRail contentRef={contentRef} scrolls={scrolls}>
      {track ? (
        <>
          <span className="peek__title">{track.title}</span>
          {track.artist ? <span className="peek__artist">{track.artist}</span> : null}
        </>
      ) : (
        // Browser audio identifies nothing about itself, and the island does
        // not invent a title it was never told.
        <span className="peek__title peek__title--muted">Audio playing</span>
      )}
    </PeekRail>
  );
}

/**
 * A session in the peek.
 *
 * Three things are pinned and the rest travels behind them: the agent glyph,
 * the completion tick, and the diff. They are the parts you read as symbols
 * rather than as words — a tick that slid past would have to be waited for, and
 * a diff that slid past would be unreadable at any speed. Each sits over a
 * gradient of the bar's own background so the text passing underneath never
 * competes with it for contrast.
 */
function SessionPeek({
  card,
  model,
  scrolls,
  contentRef,
}: {
  card: IslandPresenceCard | null;
  model: CompactIslandModel;
  scrolls: boolean;
  contentRef: RefObject<HTMLSpanElement | null>;
}) {
  const hasDiff = model.additions !== undefined || model.deletions !== undefined;
  const showsDiff = hasDiff && card?.presence !== "blocked";
  const showsTick = card?.presence === "paused" && model.phase === "complete";

  return (
    <>
      <span aria-hidden="true" className="peek__agent">
        <FigmaIcon name="compact-agent.svg" />
      </span>

      <PeekRail contentRef={contentRef} scrolls={scrolls}>
        <span className="peek__title">{card?.title ?? model.title}</span>
        <span className="branch-chip peek__branch">
          <span>{card?.project ?? model.project}</span>
          <span className="branch-chip__branch">{card?.branch ?? model.branch}</span>
        </span>
        {showsDiff ? null : (
          <span className={`peek__phase peek__phase--${card?.presence ?? "running"}`}>
            {card?.detail ?? model.detail}
          </span>
        )}
      </PeekRail>

      {showsTick ? (
        <span aria-hidden="true" className="peek__pin peek__pin--lead">
          <span className="peek__tick">✓</span>
        </span>
      ) : null}

      {showsDiff ? (
        <span className="peek__pin peek__pin--trail">
          <span className="peek__diff">
            {model.additions !== undefined ? <span className="diff-add">+{model.additions}</span> : null}
            {model.deletions !== undefined ? <span className="diff-delete">-{model.deletions}</span> : null}
          </span>
        </span>
      ) : null}
    </>
  );
}

/**
 * The travelling half of the peek.
 *
 * The track always carries its natural width so it can be measured; whether it
 * moves is decided above, by comparing that width against the bound. A rail
 * that decided for itself would have to measure twice — once to size the bar,
 * once to size itself — and the two measurements would disagree for a frame.
 */
function PeekRail({
  children,
  scrolls,
  contentRef,
}: {
  children: ReactNode;
  scrolls: boolean;
  contentRef: RefObject<HTMLSpanElement | null>;
}) {
  const railRef = useRef<HTMLSpanElement | null>(null);
  const [shift, setShift] = useState(0);

  useLayoutEffect(() => {
    const rail = railRef.current;
    const track = contentRef.current;
    if (!rail || !track || !scrolls) {
      setShift(0);
      return;
    }

    const overflow = Math.round(track.scrollWidth - rail.clientWidth);
    setShift((current) => (current === Math.max(0, overflow) ? current : Math.max(0, overflow)));
  });

  const style = shift
    ? ({
        "--marquee-shift": `${-shift}px`,
        // Roughly 28pt a second: readable at a glance, not a news crawl.
        "--marquee-duration": `${Math.max(4, shift / 28 + 2)}s`,
      } as CSSProperties)
    : undefined;

  return (
    <span className="peek__rail" ref={railRef}>
      <span className="peek__track" data-scrolling={shift > 0} ref={contentRef} style={style}>
        {children}
      </span>
    </span>
  );
}

/* -------------------------------------------------------------------------- */
/* Ticker                                                                      */
/* -------------------------------------------------------------------------- */

/* -------------------------------------------------------------------------- */
/* Expanded island                                                             */
/* -------------------------------------------------------------------------- */

function ExpandedIsland({
  model,
  onAction,
  reducedMotion,
  fromHeight,
}: {
  model: Exclude<IslandModel, CompactIslandModel>;
  onAction: KennelIslandProps["onAction"];
  reducedMotion: boolean;
  /** The resting strip's height, so the panel opens out of it, not over it. */
  fromHeight: number;
}) {
  // A panel is sized by what is in it, so its height is measured rather than
  // declared, and the spring chases the measurement. The panel opens from the
  // height of the strip it replaced, which is what keeps the shape continuous
  // across a swap that replaces the entire subtree.
  const [contentRef, contentHeight] = useMeasuredHeight<HTMLDivElement>();
  const height = contentHeight ?? fromHeight;
  const growing = useGrowing(height);

  return (
    <motion.div
      animate={{ height, opacity: 1, filter: "blur(0px)" }}
      className="island-body island-body--panel"
      exit={{
        height: fromHeight,
        opacity: 0,
        filter: `blur(${ISLAND_CONTENT_BLUR}px)`,
        transition: islandPanelExitTransition(reducedMotion),
      }}
      initial={{ height: fromHeight, opacity: 0, filter: `blur(${ISLAND_CONTENT_BLUR}px)` }}
      transition={islandPanelTransition(growing, reducedMotion)}
    >
      <span aria-hidden="true" className="island-fillet island-fillet--left" />
      <span aria-hidden="true" className="island-fillet island-fillet--right" />

      {/* The panel's height is animated, so the height the animation is
          chasing has to come from something that is not animated. This
          wrapper is that thing: it holds the panel's real content box,
          bottom padding included. */}
      <div className="island-body__measure" ref={contentRef}>
        {model.surface === "queue" ? (
          <QueueHeader model={model} onAction={onAction} />
        ) : (
          <StatusHeader model={model} onAction={onAction} />
        )}

        <motion.div
          animate="visible"
          className="island-content"
          initial="hidden"
          key={model.surface}
          transition={reducedMotion ? { duration: 0 } : undefined}
          variants={islandContentVariants}
        >
          {model.surface === "queue" ? <QueueBody model={model} onAction={onAction} /> : null}
          {model.surface === "choice" ? (
            <PromptBody model={model} onAction={onAction}>
              <ChoiceView model={model} onAction={onAction} />
            </PromptBody>
          ) : null}
          {model.surface === "permission" ? (
            <PromptBody model={model} onAction={onAction}>
              <PermissionView model={model} onAction={onAction} />
            </PromptBody>
          ) : null}
          {model.surface === "steer" ? (
            <PromptBody model={model} onAction={onAction}>
              <SteerView model={model} onAction={onAction} />
            </PromptBody>
          ) : null}
          {model.surface === "usage" ? <UsageView model={model} onAction={onAction} /> : null}
        </motion.div>
      </div>
    </motion.div>
  );
}

/* -------------------------------------------------------------------------- */
/* Header                                                                     */
/* -------------------------------------------------------------------------- */

interface IslandHeaderProps {
  left: ReactNode;
  right: ReactNode;
  notchLabel: string;
  onNotchActivate: () => void;
}

/**
 * The header straddles the hardware notch: a cluster on either side and a
 * transparent-to-the-eye button filling the camera housing in between. Clicking
 * the housing is the gesture people already have from the system, so it is the
 * island's primary toggle.
 */
function IslandHeader({ left, right, notchLabel, onNotchActivate }: IslandHeaderProps) {
  return (
    <div className="island-header">
      <div className="island-header__cluster island-header__cluster--left">{left}</div>
      <button
        aria-label={notchLabel}
        className="island-header__notch"
        onClick={onNotchActivate}
        type="button"
      />
      <div className="island-header__cluster island-header__cluster--right">{right}</div>
    </div>
  );
}

interface StatusHeaderModel {
  tone: CompactIslandModel["tone"];
  phase: CompactIslandModel["phase"];
  count: number;
}

function statusHeaderModel(model: Exclude<IslandModel, QueueIslandModel>): StatusHeaderModel {
  switch (model.surface) {
    case "compact":
      return { tone: model.tone, phase: model.phase, count: model.attentionCount ?? 0 };
    case "steer":
      return { tone: "working", phase: "working", count: 1 };
    case "usage":
      return { tone: "working", phase: "working", count: model.sessionsUsing };
    default:
      return { tone: "action", phase: "needs_input", count: model.questionCount };
  }
}

function StatusHeader({
  model,
  onAction,
}: {
  model: Exclude<IslandModel, QueueIslandModel>;
  onAction: KennelIslandProps["onAction"];
}) {
  const { tone, phase, count } = statusHeaderModel(model);
  const isResting = model.surface === "compact";
  const toggle = () => runAction(onAction, { type: isResting ? "expand" : "dismiss" });
  const countLabel = isResting
    ? `Open Kennel work queue. ${attentionSummary(count)}`
    : `Close this panel. ${count} ${count === 1 ? "item is" : "items are"} open`;

  return (
    <IslandHeader
      notchLabel={isResting ? "Expand Kennel island" : "Collapse Kennel island"}
      onNotchActivate={toggle}
      left={
        <>
          <button
            aria-label={isResting ? "Open Kennel work queue" : "Close this panel"}
            className={`island-status island-status--${tone}`}
            onClick={toggle}
            type="button"
          >
            <StatusGlyph phase={phase} tone={tone} />
          </button>
          <button aria-label="Open Kennel" className="island-pet" onClick={toggle} type="button">
            <FigmaIcon name="compact-pet.png" />
          </button>
        </>
      }
      right={
        <>
          <button aria-label="Kennel activity" className="island-waveform" onClick={toggle} type="button">
            <FigmaIcon name="compact-waveform.svg" />
          </button>
          <button aria-label={countLabel} className="island-count" onClick={toggle} type="button">
            {count}
          </button>
        </>
      }
    />
  );
}

function StatusGlyph({ phase, tone }: { phase: StatusHeaderModel["phase"]; tone: StatusHeaderModel["tone"] }) {
  if (phase === "working") return <FigmaIcon className="island-status__spinner" name="compact-spinner.svg" />;
  if (phase === "needs_input") return <span>?</span>;
  if (phase === "offline") return <span className="island-status__dot island-status__dot--offline" />;
  if (phase === "error") return <span>!</span>;
  if (phase === "complete" && tone === "ready") return <span>✓</span>;
  if (tone === "action" || tone === "error") return <span>!</span>;
  return <span className="island-status__dot" />;
}

function QueueHeader({ model, onAction }: { model: QueueIslandModel; onAction: KennelIslandProps["onAction"] }) {
  return (
    <IslandHeader
      notchLabel="Collapse Kennel island"
      onNotchActivate={() => runAction(onAction, { type: "collapse" })}
      left={
        <div className="island-tabs" role="tablist" aria-label="Island views">
          {(["home", "work"] as const).map((tab) => (
            <button
              aria-selected={model.activeTab === tab}
              className={model.activeTab === tab ? "island-tab is-selected" : "island-tab"}
              key={tab}
              onClick={() => runAction(onAction, { type: "set-tab", tab })}
              role="tab"
              type="button"
            >
              {tab === "home" ? "Home" : "Work"}
            </button>
          ))}
        </div>
      }
      right={
        <>
          <FigmaIcon className="island-tool island-tool--progress" name="progress-header.svg" />
          <button aria-label="Recent activity (coming soon)" className="island-tool-button" disabled type="button">
            <FigmaIcon className="island-tool" name="icon-settings.svg" />
          </button>
          <button aria-label="Inbox (coming soon)" className="island-tool-button" disabled type="button">
            <FigmaIcon className="island-tool" name="icon-tray.svg" />
          </button>
          <button
            aria-label="Island settings"
            className="island-tool-button"
            onClick={() => runAction(onAction, { type: "open-settings" })}
            type="button"
          >
            <FigmaIcon className="island-tool" name="icon-clock.svg" />
          </button>
          <button
            aria-label={`Open Kennel usage. ${attentionSummary(model.pendingCount)}`}
            className="island-count"
            onClick={() => runAction(onAction, { type: "open-usage" })}
            type="button"
          >
            {model.pendingCount}
          </button>
        </>
      }
    />
  );
}

/* -------------------------------------------------------------------------- */
/* Compact                                                                     */
/* -------------------------------------------------------------------------- */

interface CompactTickerProps {
  title: string;
  project: string;
  branch: string;
  phase: CompactIslandModel["phase"];
  tone: CompactIslandModel["tone"];
  detail?: string;
  additions?: number;
  deletions?: number;
  hasDiff: boolean;
  onExpand: () => void;
}

function phaseLabelFor(phase: CompactIslandModel["phase"], tone: CompactIslandModel["tone"]) {
  switch (phase) {
    case "offline":
      return "Offline";
    case "error":
      return "Needs attention";
    case "needs_input":
      return "Needs you";
    case "idle":
      return "Idle";
    case "complete":
      if (tone === "ready") return "Complete";
      if (tone === "review") return "In review";
      if (tone === "action" || tone === "error") return "Needs attention";
      return "Idle";
    default:
      return "Working";
  }
}

function CompactTicker({
  title,
  project,
  branch,
  phase,
  tone,
  detail,
  additions,
  deletions,
  hasDiff,
  onExpand,
}: CompactTickerProps) {
  const showReadyCheck = phase === "complete" && tone === "ready";
  const showDiff = hasDiff && !["needs_input", "idle", "offline", "error"].includes(phase);

  return (
    <button
      aria-label={`Open work queue. ${title}`}
      className="island-ticker"
      onClick={onExpand}
      title={title}
      type="button"
    >
      <FigmaIcon className="island-ticker__agent" name="compact-agent.svg" />
      {showReadyCheck ? <span className="island-ticker__check">✓</span> : null}
      <span className="island-ticker__title">{title}</span>
      <span className="branch-chip island-ticker__branch">
        <span>{project}</span>
        <span className="branch-chip__branch">{branch}</span>
      </span>
      {showDiff ? (
        <span className="island-ticker__diff">
          {additions !== undefined ? <span className="diff-add">+{additions}</span> : null}
          {deletions !== undefined ? <span className="diff-delete">-{deletions}</span> : null}
        </span>
      ) : (
        <span className={`island-ticker__phase island-ticker__phase--${phase} island-ticker__phase--${tone}`}>
          {detail ?? phaseLabelFor(phase, tone)}
        </span>
      )}
    </button>
  );
}

/* -------------------------------------------------------------------------- */
/* Prompt surfaces                                                             */
/* -------------------------------------------------------------------------- */

function PromptBody({
  model,
  onAction,
  children,
}: {
  model: ChoiceIslandModel | PermissionIslandModel | SteerIslandModel;
  onAction: KennelIslandProps["onAction"];
  children: ReactNode;
}) {
  const isSteer = model.surface === "steer";

  return (
    <div className={`island-prompt island-prompt--${model.surface}`}>
      <CompactTicker
        branch={model.branch ?? "main"}
        hasDiff={false}
        phase={isSteer ? "working" : "needs_input"}
        project={model.project ?? "Kennel"}
        title={
          model.surface === "choice"
            ? "Question from Kennel"
            : model.surface === "permission"
              ? "Permission needed"
              : `Steer ${model.title}`
        }
        tone={isSteer ? "working" : "action"}
        onExpand={() => runAction(onAction, { type: "dismiss" })}
      />
      <div className="island-prompt__panel">{children}</div>
    </div>
  );
}

/* -------------------------------------------------------------------------- */
/* Queue                                                                       */
/* -------------------------------------------------------------------------- */

function QueueBody({ model, onAction }: { model: QueueIslandModel; onAction: KennelIslandProps["onAction"] }) {
  if (model.activeTab === "home") return <QueueHome model={model} onAction={onAction} />;

  return (
    <>
      {model.error ? <p className="island-queue__error" role="status">{model.error}</p> : null}
      {model.tasks.length > 0 ? (
        <div className="island-queue" role="list">
          {model.tasks.map((task) => (
            <QueueRow key={task.id} onAction={onAction} task={task} />
          ))}
        </div>
      ) : (
        <QueueEmpty model={model} onAction={onAction} />
      )}
    </>
  );
}

function QueueHome({ model, onAction }: { model: QueueIslandModel; onAction: KennelIslandProps["onAction"] }) {
  const active = model.tasks.filter((task) => task.activity === "active").length;
  const needsYou = model.tasks.filter((task) => task.status === "needs_input").length;

  return (
    <div className="queue-home">
      <div className="queue-home__hero">
        <FigmaIcon className="queue-home__mark" name="agent-waldo.svg" />
        <div className="queue-home__copy">
          <p className="queue-home__eyebrow">{model.statusMessage ?? "Kennel is running"}</p>
          <p className="queue-home__title">
            {model.connection === "offline"
              ? model.statusDetail ?? "Open Kennel to connect"
              : model.tasks.length === 0
                ? model.statusDetail ?? "No active sessions"
                : `${active} agents working · ${needsYou} need you`}
          </p>
        </div>
      </div>
      <div className="queue-home__actions">
        {model.connection === "offline" ? (
          <>
            <button className="queue-home__open" onClick={() => runAction(onAction, { type: "retry-connection" })} type="button">
              Retry
            </button>
            <button className="queue-home__open" onClick={() => runAction(onAction, { type: "open-session" })} type="button">
              Open Kennel
              <FigmaIcon className="micro-icon" name="arrow-right.svg" />
            </button>
          </>
        ) : (
          <button
            className="queue-home__open"
            onClick={() => runAction(onAction, { type: "set-tab", tab: "work" })}
            type="button"
          >
            Open work queue
            <FigmaIcon className="micro-icon" name="arrow-right.svg" />
          </button>
        )}
      </div>
    </div>
  );
}

function QueueEmpty({ model, onAction }: { model: QueueIslandModel; onAction: KennelIslandProps["onAction"] }) {
  return (
    <div className="queue-empty" role="listitem">
      <span className={`queue-row__tone queue-row__tone--${model.connection === "offline" ? "muted" : "ready"}`} />
      <span className="queue-empty__title">{model.statusMessage ?? "No active sessions"}</span>
      <span className="queue-empty__detail">{model.statusDetail ?? "Kennel will surface new work here."}</span>
      <button
        className="queue-row__action"
        onClick={() => runAction(onAction, { type: model.connection === "offline" ? "retry-connection" : "collapse" })}
        type="button"
      >
        {model.connection === "offline" ? "Retry" : "Done"}
      </button>
    </div>
  );
}

function QueueRow({ task, onAction }: { task: IslandTask; onAction: KennelIslandProps["onAction"] }) {
  return (
    <div className={`queue-row ${task.dimmed ? "is-dimmed" : ""}`} role="listitem">
      <span className={`queue-row__tone queue-row__tone--${task.tone}`} />
      <FigmaIcon
        className="queue-row__agent"
        name={task.agent === "waldo" ? "agent-waldo.svg" : "agent-codex.svg"}
      />
      <span className="queue-row__title" title={task.title}>{task.title}</span>
      <span className="branch-chip queue-row__branch">
        <span>{task.project}</span>
        <span className="branch-chip__branch">{task.branch}</span>
      </span>
      <button
        className="queue-row__target"
        onClick={() => runAction(onAction, { type: "open-session", sessionId: task.sessionId, projectId: task.projectId })}
        title={task.target}
        type="button"
      >
        <FigmaIcon className="queue-row__link" name="icon-link.svg" />
        <span>{task.target}</span>
      </button>
      <FigmaIcon
        className="queue-row__progress"
        name={task.activity === "active" || task.activity === "waiting_input" ? "progress-active.svg" : "progress-idle.svg"}
      />
      <span className="queue-row__updated">{task.updatedLabel}</span>
      <button
        className="queue-row__action"
        disabled={task.disabled}
        onClick={() => runAction(onAction, { type: "task-action", taskId: task.id, label: task.actionLabel })}
        type="button"
      >
        {task.actionLabel}
      </button>
    </div>
  );
}

/* -------------------------------------------------------------------------- */
/* Prompt panels                                                               */
/* -------------------------------------------------------------------------- */

interface PromptNavProps {
  index: number;
  count: number;
  onAction: KennelIslandProps["onAction"];
}

function PromptNav({ index, count, onAction }: PromptNavProps) {
  return (
    <div className="prompt-nav">
      <button
        aria-label="Previous prompt"
        className="icon-button"
        disabled={count <= 1}
        onClick={() => runAction(onAction, { type: "navigate-prompt", direction: "previous" })}
        type="button"
      >
        <FigmaIcon className="micro-icon prompt-nav__left" name="chevron-left.svg" />
      </button>
      <span className="prompt-nav__position">{index} / {count}</span>
      <button
        aria-label="Next prompt"
        className="icon-button"
        disabled={count <= 1}
        onClick={() => runAction(onAction, { type: "navigate-prompt", direction: "next" })}
        type="button"
      >
        <FigmaIcon className="micro-icon" name="chevron-right.svg" />
      </button>
      <button
        aria-label="Close prompt"
        className="icon-button"
        onClick={() => runAction(onAction, { type: "dismiss" })}
        type="button"
      >
        <FigmaIcon className="micro-icon" name="icon-close.svg" />
      </button>
    </div>
  );
}

function ChoiceView({ model, onAction }: { model: ChoiceIslandModel; onAction: KennelIslandProps["onAction"] }) {
  return (
    <div className="prompt-panel choice-panel">
      <div className="prompt-panel__head">
        <p className="prompt-panel__question">{model.question}</p>
        <PromptNav count={model.questionCount} index={model.questionIndex} onAction={onAction} />
      </div>
      <div className="choice-list">
        {model.options.map((option, index) => (
          <button
            className={`choice-option ${index === 0 ? "is-selected" : ""} ${option.freeform ? "is-freeform" : ""}`}
            disabled={Boolean(model.submittingOptionId)}
            key={option.id}
            onClick={() => runAction(onAction, { type: "select-choice", promptId: model.promptId, optionId: option.id })}
            type="button"
          >
            <span className="choice-option__number">{index + 1}</span>
            <span className="choice-option__label">{option.label}</span>
            {option.recommended ? <span className="choice-option__badge">Recommended</span> : null}
            {model.submittingOptionId === option.id ? (
              <span className="choice-option__sending">Sending…</span>
            ) : index === 0 ? (
              <FigmaIcon className="choice-option__arrow" name="arrow-right.svg" />
            ) : null}
            {option.freeform ? <span className="choice-option__skip">Skip</span> : null}
          </button>
        ))}
      </div>
      {model.error ? <p className="prompt-panel__error" role="alert">{model.error}</p> : null}
    </div>
  );
}

function PermissionView({ model, onAction }: { model: PermissionIslandModel; onAction: KennelIslandProps["onAction"] }) {
  const [confirmInterrupt, setConfirmInterrupt] = useState(false);
  useEffect(() => setConfirmInterrupt(false), [model.sessionId, model.requestId]);
  const decisions = model.contextTruncated ? [] : model.decisions ?? [];
  const fallbackMessage = model.contextTruncated
    ? "Approval details are incomplete. Review the full request in Kennel before deciding."
    : "Kennel did not provide any approval actions. Review this request in Kennel.";

  return (
    <div className="prompt-panel permission-panel">
      <div className="prompt-panel__head">
        <p className="prompt-panel__question">{model.reason ?? model.question}</p>
        <PromptNav count={model.questionCount} index={model.questionIndex} onAction={onAction} />
      </div>
      <button
        className="permission-context"
        aria-label={[model.command ?? model.question, model.cwd].filter(Boolean).join(" in ")}
        onClick={() => runAction(onAction, { type: "open-session", sessionId: model.sessionId })}
        title={[model.command ?? model.question, model.cwd].filter(Boolean).join(" — ")}
        type="button"
      >
        <span className="branch-chip permission-context__branch">
          <span>{model.project}</span>
          <span className="branch-chip__branch">{model.branch}</span>
        </span>
        <span className="permission-context__files">
          {model.command ? <span className="permission-context__command">{model.command}</span> : null}
          {model.cwd ? <span className="permission-context__cwd">{shortenPath(model.cwd)}</span> : null}
          {!model.command && !model.cwd ? model.contextFiles.map((file) => <span key={file}>{file}</span>) : null}
        </span>
        <FigmaIcon className="permission-context__arrow" name="arrow-right.svg" />
      </button>
      {decisions.length > 0 ? (
        <div className="permission-actions">
          {decisions.map((decision) => (
            <PermissionButton
              disabled={Boolean(model.submittingDecisionId)}
              key={decision.id}
              shortcut={decision.shortcut ?? ""}
              label={model.submittingDecisionId === decision.id ? "Sending…" : decision.label}
              onClick={() => runAction(onAction, { type: "resolve-permission", requestId: model.requestId, decisionId: decision.id })}
            />
          ))}
          {model.canInterrupt !== false ? (
            <button
              aria-label="Interrupt current turn"
              className={`permission-pause ${confirmInterrupt ? "is-confirming" : ""}`}
              disabled={Boolean(model.submittingDecisionId)}
              onClick={() => {
                if (!confirmInterrupt) {
                  setConfirmInterrupt(true);
                  return;
                }
                runAction(onAction, { type: "interrupt-session", requestId: model.requestId });
              }}
              title={confirmInterrupt ? "Click again to interrupt and discard the running turn" : "Interrupt the running turn"}
              type="button"
            >
              {confirmInterrupt ? <span aria-hidden="true">!</span> : <FigmaIcon name="icon-pause.svg" />}
            </button>
          ) : null}
        </div>
      ) : (
        <div className="permission-fallback" role={model.contextTruncated ? "alert" : "status"}>
          <span>{fallbackMessage}</span>
          <button
            aria-label="Open this approval request in Kennel"
            className="permission-fallback__open"
            onClick={() => runAction(onAction, { type: "open-session", sessionId: model.sessionId })}
            type="button"
          >
            Open Kennel
            <FigmaIcon className="micro-icon" name="arrow-right.svg" />
          </button>
        </div>
      )}
      {model.error ? <p className="prompt-panel__error" role="alert">{model.error}</p> : null}
    </div>
  );
}

function PermissionButton({ shortcut, label, onClick, disabled = false }: { shortcut: string; label: string; onClick: () => void; disabled?: boolean }) {
  return (
    <button className="permission-button" disabled={disabled} onClick={onClick} type="button">
      {shortcut ? <span>{shortcut}</span> : null}
      <strong>{label}</strong>
    </button>
  );
}

function SteerView({ model, onAction }: { model: SteerIslandModel; onAction: KennelIslandProps["onAction"] }) {
  const [text, setText] = useState("");

  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (model.submitting) return;
    runAction(onAction, { type: "submit-steer", sessionId: model.sessionId, text });
  };

  const handleKeyDown = (event: ReactKeyboardEvent<HTMLTextAreaElement>) => {
    if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
      event.preventDefault();
      event.currentTarget.form?.requestSubmit();
    }
  };

  return (
    <form className="prompt-panel steer-panel" onSubmit={submit}>
      <div className="prompt-panel__head">
        <p className="prompt-panel__question">Guide the running turn</p>
        <button
          aria-label="Close steer composer"
          className="icon-button"
          onClick={() => runAction(onAction, { type: "dismiss" })}
          type="button"
        >
          <FigmaIcon className="micro-icon" name="icon-close.svg" />
        </button>
      </div>
      <textarea
        aria-label={`Guidance for ${model.title}`}
        autoFocus
        className="steer-panel__input"
        disabled={model.submitting}
        maxLength={8_000}
        onChange={(event) => setText(event.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="What should the agent adjust?"
        value={text}
      />
      <div className="steer-panel__footer">
        <span>⌘↵ to steer</span>
        <button disabled={model.submitting || !text.trim()} type="submit">
          {model.submitting ? "Sending…" : "Steer turn"}
        </button>
      </div>
      {model.error ? <p className="prompt-panel__error" role="alert">{model.error}</p> : null}
    </form>
  );
}

function UsageView({ model, onAction }: { model: UsageIslandModel; onAction: KennelIslandProps["onAction"] }) {
  return (
    <div className="usage-panel">
      <div className="usage-panel__header">
        <button aria-label="Back" className="icon-button" onClick={() => runAction(onAction, { type: "collapse" })} type="button">
          <FigmaIcon className="usage-chevron usage-chevron--left" name="chevron-mini-left.svg" />
        </button>
        <FigmaIcon className="usage-panel__logo" name="icon-pro.svg" />
        <span className="usage-panel__plan">{model.plan}</span>
        {model.account ? <span className="usage-panel__account">{model.account}</span> : null}
        <span className="usage-panel__spacer" />
        <span className="usage-panel__sessions-label">Sessions using:</span>
        <span className="usage-panel__count">{model.sessionsUsing}</span>
        <button aria-label="Close usage" className="icon-button" onClick={() => runAction(onAction, { type: "dismiss" })} type="button">
          <FigmaIcon className="usage-chevron usage-chevron--right" name="chevron-mini-right.svg" />
        </button>
      </div>
      <div className="usage-panel__limits">
        {model.unavailableMessage ? <p className="usage-panel__unavailable">{model.unavailableMessage}</p> : null}
        {model.limits.map((limit) => (
          <div className="usage-limit" key={limit.id}>
            <div className="usage-limit__copy">
              <span className="usage-limit__label">{limit.label}</span>
              <span className="usage-limit__percent">{limit.percent}%</span>
              <span className="usage-limit__reset">{limit.resetLabel}</span>
            </div>
            <div className="usage-limit__track">
              <span style={{ width: `${Math.max(0, Math.min(100, limit.percent))}%` }} />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

import { HEADER_ITEM } from "./peek.ts";

/* --------------------------------------------------------------------------
   Peek layout
   --------------------------------------------------------------------------
   The peek is the bar that hangs under the strip while the pointer is on the
   island. Two rules shape it, and both come from the hardware above it rather
   than from the text inside it.

   Width is bounded by the *midlines* of the outermost item on each side — the
   vertical centre of the leftmost chip to the vertical centre of the rightmost.
   Not the strip's edge: a bar that reached the edge would put its own rounded
   corner directly under a chip's rounded corner, and two curves a few points
   apart read as a mistake. Stopping at the midlines leaves each chip half
   overhanging the bar, which is what makes the pair look milled from one piece.

   Within that bound the bar is sized by its content and nothing else. A short
   label gets a short bar. Only content that cannot fit the bound scrolls, so a
   track change under a resting cursor resizes the bar rather than starting a
   marquee that was not needed.
   -------------------------------------------------------------------------- */

/** Horizontal padding inside the peek, per side. */
export const PEEK_TEXT_PADDING = 14;

/* --------------------------------------------------------------------------
   Peek silhouette
   --------------------------------------------------------------------------
   The bar's outline is not a rounded rectangle with corner fillets any more —
   it is one continuous shape (`public/figma/peek-shape.svg`, the reference
   from Figma): full width where it meets the strip, tapering on both sides
   through a single curve to a narrower flat base where the text sits. Applied
   as a CSS mask stretched to whatever box the bar ends up being
   (`preserveAspectRatio="none"`, `mask-size: 100% 100%`), so the RATIO between
   the shape's top and base is what has to be constant, not any absolute pixel
   width — the reference is one fixed-size export, and the bar is drawn at
   every width a track title happens to need.
   -------------------------------------------------------------------------- */

/** The reference export's own dimensions, for the ratio below. */
const PEEK_SHAPE_TOP_WIDTH = 326.497;
const PEEK_SHAPE_BASE_WIDTH = 270.997 - 55.5001;

/**
 * How much wider the shape's top is than its flat base.
 *
 * The base is where the button's content actually sits, so the OUTER box has
 * to be wider than the content by this factor — an outer box sized to exactly
 * fit the content would mask away its own edges, clipping the very text the
 * bar exists to show.
 */
export const PEEK_SHAPE_FLARE_RATIO = PEEK_SHAPE_TOP_WIDTH / PEEK_SHAPE_BASE_WIDTH;

/**
 * Outer box width for a peek holding content of the given width.
 *
 * The result is guaranteed at least as wide as `contentWidth`: the flare only
 * adds room on top of it, and rounding always rounds up, so the shape's base
 * never lands a fraction of a pixel narrower than what has to fit inside it.
 */
export function peekOuterWidthFor(contentWidth: number): number {
  return Math.ceil(Math.max(0, contentWidth) * PEEK_SHAPE_FLARE_RATIO);
}

/**
 * Narrowest the peek is allowed to get.
 *
 * Below this a bar stops reading as a bar and starts reading as a fragment of
 * one, however little there is to say inside it.
 */
export const PEEK_MIN_WIDTH = 132;

export interface PeekWidthInput {
  /** Outer width of the resting strip, in points. */
  stripWidth: number;
  /** Measured width of the peek's content, in points. */
  contentWidth: number;
  /** Points between the housing and the first chip. */
  contentPadding: number;
  /** Whether the strip has any chips to take midlines from. */
  hasItems: boolean;
}

export interface PeekWidth {
  width: number;
  /** The bound the content was measured against. */
  maxWidth: number;
  /** The content is wider than the bound, so it has to travel to be read. */
  scrolls: boolean;
}

/**
 * The widest the peek may grow: midline of the leftmost chip to midline of the
 * rightmost.
 *
 * Each chip's centre sits one padding plus half an item in from its side of the
 * strip, so the span is the strip minus both of those. A strip with no chips
 * has no midlines to stop at and is bounded by itself.
 */
export function peekMaxWidthFor({
  stripWidth,
  contentPadding,
  hasItems,
}: Pick<PeekWidthInput, "stripWidth" | "contentPadding" | "hasItems">) {
  if (!hasItems) return Math.max(PEEK_MIN_WIDTH, stripWidth);
  return Math.max(PEEK_MIN_WIDTH, stripWidth - 2 * contentPadding - HEADER_ITEM);
}

/** Outer width of the peek for the content it currently holds. */
export function peekWidthFor(input: PeekWidthInput): PeekWidth {
  const maxWidth = peekMaxWidthFor(input);
  const wanted = Math.max(0, input.contentWidth) + 2 * PEEK_TEXT_PADDING;

  return {
    width: Math.round(Math.min(maxWidth, Math.max(PEEK_MIN_WIDTH, wanted))),
    maxWidth: Math.round(maxWidth),
    // Only content that could not fit travels. Measuring first is what keeps a
    // short label still instead of drifting for no reason.
    scrolls: wanted > maxWidth,
  };
}

/* --------------------------------------------------------------------------
   Hover zones
   --------------------------------------------------------------------------
   The strip carries two kinds of chip — session and media — and hovering one
   should say something about that kind. Asking the pointer to hit a 24pt chip
   would make that a game, so the strip is divided into full-height zones
   instead: every point along it belongs to whichever kind is nearest, and the
   boundary falls in the middle of the gap between two chips of different kinds.

   Chips of the same kind on opposite sides of the housing merge into one zone
   spanning the notch, which is why the album art and the waveform — with the
   camera housing between them — are one media zone rather than two.
   -------------------------------------------------------------------------- */

export type PeekSubject = "media" | "session";

export interface HeaderItemBox {
  kind: PeekSubject;
  left: number;
  right: number;
}

export interface PeekZone {
  subject: PeekSubject;
  left: number;
  right: number;
}

/**
 * How far above and below the strip a zone still counts.
 *
 * The strip is at the very top of the display, so there is nothing above it to
 * steal a pointer; extending upward costs nothing and catches a cursor thrown
 * at the menu bar. Below, the overshoot has to stay small or it would swallow
 * the peek the zone just opened.
 */
export const ZONE_OVERSHOOT_TOP = 8;
export const ZONE_OVERSHOOT_BOTTOM = 4;

/**
 * The strip divided into zones, left to right.
 *
 * Adjacent chips of the same kind produce one zone, so the result is the
 * coarsest division the chips allow rather than one zone per chip.
 */
export function peekZonesFor(
  items: readonly HeaderItemBox[],
  bounds: { left: number; right: number },
): PeekZone[] {
  const ordered = [...items].sort((left, right) => left.left - right.left);
  if (ordered.length === 0) return [];

  const zones: PeekZone[] = [];
  for (let index = 0; index < ordered.length; index += 1) {
    const item = ordered[index];
    const previous = zones.at(-1);

    if (previous && previous.subject === item.kind) {
      previous.right = item.right;
      continue;
    }

    // The boundary between two kinds is the middle of the gap between them, so
    // neither chip is easier to reach than the other.
    const start = previous
      ? (ordered[index - 1].right + item.left) / 2
      : bounds.left;
    if (previous) previous.right = start;
    zones.push({ subject: item.kind, left: start, right: item.right });
  }

  zones[0].left = bounds.left;
  zones[zones.length - 1].right = bounds.right;
  return zones;
}

/** The zone a point falls in, or null when it is outside every zone. */
export function zoneAt(zones: readonly PeekZone[], x: number): PeekZone | null {
  return zones.find((zone) => x >= zone.left && x <= zone.right) ?? null;
}

export interface PeekSubjectInput {
  /** What the pointer is over, or null when it is over nothing in particular. */
  zone: PeekSubject | null;
  hasMedia: boolean;
  hasSession: boolean;
  /** The idle rotation's current pick, used only when the pointer is not choosing. */
  rotated?: PeekSubject | null;
}

/**
 * What the peek should be talking about.
 *
 * A zone only gets what it asks for when there is something to say: hovering
 * the waveform with nothing playing falls through to the session rather than
 * opening an empty bar.
 */
export function peekSubjectFor({
  zone,
  hasMedia,
  hasSession,
  rotated = null,
}: PeekSubjectInput): PeekSubject | null {
  const available = (subject: PeekSubject | null) =>
    (subject === "media" && hasMedia) || (subject === "session" && hasSession);

  if (available(zone)) return zone;
  if (available(rotated)) return rotated;
  if (hasSession) return "session";
  return hasMedia ? "media" : null;
}

/* --------------------------------------------------------------------------
   Idle rotation
   -------------------------------------------------------------------------- */

/** How long a pointer has to sit still before the peek offers the other subject. */
export const PEEK_ROTATE_IDLE_MS = 10_000;

/** How long each subject holds the bar once the rotation has started. */
export const PEEK_ROTATE_INTERVAL_MS = 6_000;

/** How far the pointer may drift and still count as parked. */
export const PEEK_IDLE_TOLERANCE = 3;

export function pointerHasMoved(
  from: { x: number; y: number } | null,
  to: { x: number; y: number },
  tolerance = PEEK_IDLE_TOLERANCE,
) {
  if (!from) return true;
  return Math.abs(from.x - to.x) > tolerance || Math.abs(from.y - to.y) > tolerance;
}

/** The other subject, when there is another one worth showing. */
export function otherSubject(subject: PeekSubject): PeekSubject {
  return subject === "media" ? "session" : "media";
}

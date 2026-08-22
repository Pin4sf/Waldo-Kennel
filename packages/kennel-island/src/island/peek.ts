// The explicit extension is what lets `node --experimental-strip-types` run the
// tests beside this file without a bundler in the way.
import { ISLAND_HEADER_HEIGHT } from "./layout.ts";

/* --------------------------------------------------------------------------
   Resting shapes
   --------------------------------------------------------------------------
   At rest the island is one of three shapes, and the whole point of the middle
   one is that it is not a state change — it is an acknowledgement.

     dormant  exactly the camera housing. Black on black, so a quiet machine
              looks like plain hardware and nothing has been added to the bezel.
     peek     the housing plus a few points on every side, with a rounded lower
              edge. It says "this is a thing, and it noticed you" without yet
              committing to a panel, which is what makes an accidental pass over
              the notch cost nothing.
     strip    the awake header, sized by its clusters.

   Sizes are computed here rather than in the component because they are the
   thing worth testing: a peek that is narrower than the housing, or a strip
   shorter than a fine-tuned notch, exposes the hardware the island is
   pretending to be.
   -------------------------------------------------------------------------- */

export type RestingShape = "dormant" | "peek" | "strip";

/** Width of one header item, in points. */
export const HEADER_ITEM = 24;
/** Gap between two header items, in points. */
export const HEADER_GAP = 6;

export function clusterWidth(items: number) {
  return items <= 0 ? 0 : items * HEADER_ITEM + (items - 1) * HEADER_GAP;
}

export interface RestingShapeInput {
  /** Something is running or playing, so the island has content to show. */
  awake: boolean;
  /** The pointer is on the island and has dwelled long enough to mean it. */
  peeking: boolean;
}

/**
 * The shape the resting island should be.
 *
 * An awake island never peeks: it is already showing a strip, and swelling that
 * strip on hover would be a second, competing answer to the same question. The
 * peek exists only for the dormant shape, which has nothing else to say.
 */
export function restingShapeFor({ awake, peeking }: RestingShapeInput): RestingShape {
  if (awake) return "strip";
  return peeking ? "peek" : "dormant";
}

export interface RestingMetricsInput {
  shape: RestingShape;
  notchWidth: number;
  notchHeight: number;
  menuBarHeight: number;
  /** Header items on the busier side; both sides are padded to match. */
  clusterItems: number;
  /** Points between the housing and a cluster. */
  contentPadding: number;
  /** Points the peek adds to each side of the housing. */
  peekWidth: number;
  /** Points the peek adds below the housing. */
  peekHeight: number;
}

export interface RestingMetrics {
  width: number;
  height: number;
}

/**
 * Outer size of the resting island, in points.
 *
 * The strip's height takes the notch height as a floor as well as the menu bar
 * height. They are usually the same, but a user who has fine-tuned the housing
 * taller has told us the hardware is bigger than the menu bar implies, and a
 * strip shorter than that would leave a black step above a black bar.
 */
export function restingMetricsFor(input: RestingMetricsInput): RestingMetrics {
  const {
    shape,
    notchWidth,
    notchHeight,
    menuBarHeight,
    clusterItems,
    contentPadding,
    peekWidth,
    peekHeight,
  } = input;

  if (shape === "strip") {
    const cluster = clusterWidth(clusterItems);
    return {
      width: notchWidth + 2 * cluster + (cluster > 0 ? 2 * contentPadding : 0),
      height: Math.max(ISLAND_HEADER_HEIGHT, menuBarHeight, notchHeight),
    };
  }

  if (shape === "peek") {
    return {
      width: notchWidth + 2 * Math.max(0, peekWidth),
      height: notchHeight + Math.max(0, peekHeight),
    };
  }

  return { width: notchWidth, height: notchHeight };
}

/**
 * Whether entering `next` from `previous` deserves a tap on the trackpad.
 *
 * Only the peek does. Waking, sleeping and rotating happen without the pointer
 * being involved — a tap for those would fire while the hand is somewhere else
 * entirely, which is how a nice detail turns into a machine that twitches.
 */
export function shouldTapFor(previous: RestingShape, next: RestingShape) {
  return next === "peek" && previous !== "peek";
}

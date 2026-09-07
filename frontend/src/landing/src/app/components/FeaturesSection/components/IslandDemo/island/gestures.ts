/* --------------------------------------------------------------------------
   Trackpad gestures over the island
   --------------------------------------------------------------------------
   Three gestures, all of them two-finger swipes made while the pointer is on
   the island:

     down    open the island
     up      close it
     across  step the track, forward or back

   A trackpad reports a swipe as dozens of small wheel events rather than one,
   so a recogniser has to answer three questions the events do not: which axis
   the user meant, when the swipe started, and when it ended. Everything below
   is that bookkeeping, kept pure so it can be tested without a DOM.
   -------------------------------------------------------------------------- */

export type IslandGesture = "open" | "close" | "next-track" | "previous-track";

export interface GestureSample {
  /** Wheel deltas as the DOM reports them. */
  deltaX: number;
  deltaY: number;
  /**
   * macOS natural scrolling, from `WheelEvent.webkitDirectionInvertedFromDevice`.
   * Without it a swipe down would open the island for half the world's Macs and
   * close it for the other half.
   */
  inverted: boolean;
  timeStamp: number;
}

/** A pause this long ends a swipe: the next event starts a fresh one. */
export const GESTURE_IDLE_MS = 140;

/** Travel before an axis is committed, so a slightly diagonal swipe still reads as one. */
export const GESTURE_AXIS_LOCK = 6;

/** Travel that opens or closes the island. */
export const GESTURE_TOGGLE_THRESHOLD = 28;

/**
 * Travel that steps a track. Higher than the toggle, because stepping past a
 * song is more annoying to undo than opening a panel that was not wanted.
 */
export const GESTURE_TRACK_THRESHOLD = 44;

/**
 * Finger movement, in points, with the sign the fingers actually had.
 *
 * Positive y is a finger moving down the trackpad; positive x is a finger
 * moving right. Natural scrolling flips both, which is what `inverted` undoes.
 */
export function fingerTravel(sample: GestureSample) {
  const sign = sample.inverted ? -1 : 1;
  return { x: sample.deltaX * sign, y: sample.deltaY * sign };
}

export interface GestureRecognizer {
  /** Feeds one wheel event. Returns a gesture the moment one is recognised. */
  read: (sample: GestureSample) => IslandGesture | null;
  /** Drops any swipe in progress. */
  reset: () => void;
}

/**
 * Recognises one gesture per swipe.
 *
 * Once a swipe has fired it is latched until the fingers lift, so a long drag
 * cannot walk through an album, and an axis stays locked for the length of a
 * swipe, so a drifting finger cannot turn a close into a skipped track.
 */
export function createGestureRecognizer(): GestureRecognizer {
  let axis: "x" | "y" | null = null;
  let travelX = 0;
  let travelY = 0;
  let latched = false;
  let lastAt: number | null = null;

  const reset = () => {
    axis = null;
    travelX = 0;
    travelY = 0;
    latched = false;
  };

  return {
    reset: () => {
      reset();
      lastAt = null;
    },
    read(sample) {
      if (lastAt !== null && sample.timeStamp - lastAt > GESTURE_IDLE_MS) reset();
      lastAt = sample.timeStamp;

      const travel = fingerTravel(sample);
      travelX += travel.x;
      travelY += travel.y;

      if (axis === null) {
        const decided = Math.max(Math.abs(travelX), Math.abs(travelY));
        if (decided < GESTURE_AXIS_LOCK) return null;
        axis = Math.abs(travelX) > Math.abs(travelY) ? "x" : "y";
      }

      if (latched) return null;

      if (axis === "y") {
        if (travelY >= GESTURE_TOGGLE_THRESHOLD) {
          latched = true;
          return "open";
        }
        if (travelY <= -GESTURE_TOGGLE_THRESHOLD) {
          latched = true;
          return "close";
        }
        return null;
      }

      // A finger moving left pulls the next track into view, the same way it
      // pulls the next photo in, which is the direction the gesture reads as.
      if (travelX <= -GESTURE_TRACK_THRESHOLD) {
        latched = true;
        return "next-track";
      }
      if (travelX >= GESTURE_TRACK_THRESHOLD) {
        latched = true;
        return "previous-track";
      }
      return null;
    },
  };
}

/** Overflow values that make an element a scroll container. */
const SCROLLABLE_OVERFLOW = new Set(["auto", "scroll", "overlay"]);

export interface ScrollMetrics {
  /** Computed `overflow-y`. Content sticking out of a `visible` box scrolls nothing. */
  overflowY: string;
  scrollTop: number;
  scrollHeight: number;
  clientHeight: number;
}

/**
 * Whether an element would use this wheel event to scroll itself.
 *
 * The queue list scrolls, and a list that closes the island instead of
 * scrolling is a list nobody can read past the fold. Deltas are in scroll
 * space, not finger space: positive `deltaY` always means "further down the
 * content", whichever way the trackpad is configured.
 *
 * Overflow is checked first and it is not a formality. A box that merely has
 * content sticking out of it reports `scrollHeight > clientHeight` exactly
 * like a scroll container does — and the island is such a box for the length
 * of every open animation, while its sprung height is still short of the
 * content inside it. Trusting the measurement alone would swallow every
 * gesture made during an animation.
 */
export function scrollCanConsume(metrics: ScrollMetrics, deltaY: number) {
  if (!SCROLLABLE_OVERFLOW.has(metrics.overflowY)) return false;

  const remaining = metrics.scrollHeight - metrics.clientHeight;
  if (remaining <= 0) return false;
  if (deltaY > 0) return metrics.scrollTop < remaining - 1;
  if (deltaY < 0) return metrics.scrollTop > 1;
  return false;
}

import type { Transition, Variants } from "motion/react";

/* --------------------------------------------------------------------------
   Island motion
   --------------------------------------------------------------------------
   Every size change on the island runs on one of the two springs below, so a
   strip that widens by one icon and a panel that opens by four hundred points
   read as the same piece of hardware moving.

   The numbers are fitted to a frame-by-frame trace of the reference recording
   (60 fps capture, panel bounding box measured per frame):

     open   526 -> 1034 px, ~2% overshoot, peak at ~300 ms, settled by ~500 ms
     close  1035 -> 492 px, no overshoot, 90% by ~270 ms, settled by ~430 ms

   Opening overshoots because a panel that arrives dead still reads as a jump
   cut. Closing does not, because a panel that rebounds on its way out reads as
   a mistake. That asymmetry is the whole trick, and it is why there are two
   springs here rather than one shared curve.
   -------------------------------------------------------------------------- */

/** Growing: damping ratio ~0.76, natural frequency ~17 rad/s. */
export const islandOpenSpring: Transition = {
  type: "spring",
  stiffness: 290,
  damping: 26,
  mass: 1,
  restDelta: 0.05,
};

/** Shrinking: damping ratio ~0.99, so it lands without a rebound. */
export const islandCloseSpring: Transition = {
  type: "spring",
  stiffness: 230,
  damping: 30,
  mass: 1,
  restDelta: 0.05,
};

export function islandSizeSpring(growing: boolean, reducedMotion: boolean): Transition {
  if (reducedMotion) return { duration: 0 };
  return growing ? islandOpenSpring : islandCloseSpring;
}

/* --------------------------------------------------------------------------
   Content cross-fade
   --------------------------------------------------------------------------
   The shape and the words inside it are not the same animation. In the
   reference the outgoing content is gone within about five frames, long before
   the shape has finished moving, and the incoming content arrives blurred and
   sharpens only once the shape has nearly landed.

   The blur is not decoration. It is what lets the shape morph without the text
   inside it visibly stretching: a blurred glyph has no edge left to distort.
   -------------------------------------------------------------------------- */

/** How far out of focus content goes while the shape is moving. */
export const ISLAND_CONTENT_BLUR = 8;

const enter: Transition = { duration: 0.22, delay: 0.05, ease: [0.22, 0.61, 0.36, 1] };
const exit: Transition = { duration: 0.09, ease: [0.4, 0, 1, 1] };

export const islandContentEnter = enter;
export const islandContentExit = exit;

export const islandContentVariants: Variants = {
  hidden: { opacity: 0, filter: `blur(${ISLAND_CONTENT_BLUR}px)`, transition: exit },
  visible: { opacity: 1, filter: "blur(0px)", transition: enter },
};

/** The ticker rail hangs beneath the strip, so it leaves the way it arrived. */
export const islandTickerVariants: Variants = {
  hidden: { opacity: 0, y: -8, filter: `blur(${ISLAND_CONTENT_BLUR}px)`, transition: exit },
  visible: { opacity: 1, y: 0, filter: "blur(0px)", transition: enter },
};

/**
 * A panel animates two things at once on different clocks: its height, which
 * is the shape and belongs to the spring, and its content, which belongs to
 * the cross-fade. Motion takes per-key transitions, so both fit in one object.
 */
export function islandPanelTransition(growing: boolean, reducedMotion: boolean): Transition {
  if (reducedMotion) return { duration: 0 };
  return { ...islandSizeSpring(growing, false), opacity: enter, filter: enter };
}

export function islandPanelExitTransition(reducedMotion: boolean): Transition {
  if (reducedMotion) return { duration: 0 };
  return { ...islandCloseSpring, opacity: exit, filter: exit };
}

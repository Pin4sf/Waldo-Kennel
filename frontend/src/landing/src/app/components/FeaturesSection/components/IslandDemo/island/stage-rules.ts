import type { IslandModel } from "./types";

/**
 * Surfaces that exist only while the pointer is on them.
 *
 * A permission or choice prompt is a question an agent is blocked on, so it
 * stays until it is answered or explicitly dismissed. The queue and the usage
 * panel are browsing, and browsing ends when you look away.
 */
export const POINTER_HELD_SURFACES: ReadonlySet<IslandModel["surface"]> = new Set([
  "queue",
  "usage",
]);

/** Grace period between the pointer leaving and the surface collapsing. */
export const POINTER_LEAVE_GRACE_MS = 220;

/**
 * A few points of slack around the body so its edges and concave fillets are
 * easy to hit without demanding pixel-exact aim.
 */
export const POINTER_HIT_TOLERANCE = 3;

export interface PointerRect {
  left: number;
  right: number;
  top: number;
  bottom: number;
}

export function isPointerInside(
  rect: PointerRect,
  x: number,
  y: number,
  tolerance = POINTER_HIT_TOLERANCE,
) {
  return (
    x >= rect.left - tolerance &&
    x <= rect.right + tolerance &&
    y >= rect.top - tolerance &&
    y <= rect.bottom + tolerance
  );
}

/**
 * Whether a hover change should start the collapse countdown.
 *
 * The trigger is the pointer *leaving*, not the pointer being absent. A queue
 * summoned from the keyboard while the pointer sits on the other side of the
 * screen has never been hovered, and closing it a beat later would make the
 * shortcut useless.
 *
 * `hovered` is `null` when no host is tracking the pointer at all — the browser
 * state lab, or a renderer whose bridge failed to attach. That is not the same
 * as "the pointer is elsewhere", and it must never collapse anything.
 */
export function shouldCollapseOnPointerLeave(
  surface: IslandModel["surface"],
  previouslyHovered: boolean,
  hovered: boolean | null,
) {
  return previouslyHovered && hovered === false && POINTER_HELD_SURFACES.has(surface);
}

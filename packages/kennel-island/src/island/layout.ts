import type { IslandModel } from "./types";

export type IslandSurface = IslandModel["surface"];

/**
 * Stage geometry used before the desktop host reports the real display, and in
 * the browser state lab where there is no host at all. The numbers describe a
 * 14-inch MacBook Pro at its default scaled resolution.
 */
export const defaultStageGeometry: KennelStageGeometry = {
  stageWidth: 820,
  stageHeight: 460,
  hasNotch: true,
  notchWidth: 200,
  notchHeight: 37,
  menuBarHeight: 37,
  scaleFactor: 2,
};

/**
 * Island width per surface, in points.
 *
 * Height is deliberately absent: the body is sized by its content so a queue
 * with two sessions is not padded out to the height of a queue with eight, and
 * so a long approval reason cannot be clipped by a hard-coded frame.
 */
export const islandWidths: Record<IslandSurface, number> = {
  compact: 440,
  activity: 440,
  queue: 680,
  choice: 400,
  permission: 400,
  steer: 400,
  usage: 400,
};

/** Corner radius of the island body's bottom edge, in points. */
export const islandRadius: Record<IslandSurface, number> = {
  compact: 20,
  activity: 24,
  queue: 26,
  choice: 24,
  permission: 24,
  steer: 24,
  usage: 24,
};

/**
 * Radius of the concave fillets that join the body's top edge to the menu bar.
 * Large enough to read as a curve at 1x, small enough not to eat the header.
 */
export const ISLAND_FILLET = 12;

/** Header row height. Covers the menu bar strip on every shipping notch. */
export const ISLAND_HEADER_HEIGHT = 44;

export function islandWidthFor(surface: IslandSurface) {
  return islandWidths[surface];
}

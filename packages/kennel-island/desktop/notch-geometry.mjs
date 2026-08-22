// Notch geometry for the island stage.
//
// macOS describes the notch only through `NSScreen.safeAreaInsets`, which
// Electron does not bridge. Everything below is derived from values Electron
// does report, so the island needs no native addon.
//
// The menu bar is the reliable signal. A notched built-in display reserves a
// taller menu bar than notchless hardware, because the bar has to clear the
// camera housing, and both the bar and the housing are laid out in points, so
// the ratio between them holds across every scaled resolution the user picks.
//
// The derived width is an estimate, accurate to a few points. It only has to be
// close: the island body is always at least as wide as the notch, and its
// visible resting shape is wider still, so a small error moves the concave
// fillets rather than exposing the housing.
//
// Two corrections sit on top of the estimate, and they are not the same thing.
// `KENNEL_ISLAND_NOTCH_WIDTH` pins an exact width in points and is an operator
// tool: it replaces the estimate outright. The settings offsets are the user's
// fine tune, applied to whichever width won, because someone dragging a slider
// against their own hardware is correcting the last few points by eye and wants
// the control to mean the same thing on every display they own.

const NOTCH_MENU_BAR_THRESHOLD = 33;
const NOTCH_ASPECT = 5.4;

/** Nothing narrower than this reads as a housing rather than a seam. */
const MIN_NOTCH_WIDTH = 80;
/** The island still has to cover the menu bar strip it straddles. */
const MIN_NOTCH_HEIGHT = 12;

const DEFAULT_GEOMETRY = Object.freeze({
	hasNotch: false,
	notchWidth: 0,
	notchHeight: 0,
	menuBarHeight: 25,
	scaleFactor: 2,
});

function positivePoints(value) {
	const parsed = typeof value === "string" ? Number.parseFloat(value) : value;
	return Number.isFinite(parsed) && parsed > 0 && parsed < 2000 ? Math.round(parsed) : null;
}

/**
 * Height of the macOS menu bar in points, measured as the strip the system
 * reserves between the top of the display and the top of its work area.
 */
export function menuBarHeightFor(display) {
	const bounds = display?.bounds;
	const workArea = display?.workArea;
	if (!bounds || !workArea) return DEFAULT_GEOMETRY.menuBarHeight;

	const reserved = workArea.y - bounds.y;
	return reserved > 0 ? Math.round(reserved) : DEFAULT_GEOMETRY.menuBarHeight;
}

/** A user offset in points, or 0 when the value is not usable as one. */
function offsetPoints(value) {
	const parsed = typeof value === "string" ? Number.parseFloat(value) : value;
	return Number.isFinite(parsed) && Math.abs(parsed) < 500 ? Math.round(parsed) : 0;
}

/**
 * Notch width in points, or 0 when the display has no notch.
 *
 * `overrideWidth` replaces the derived estimate; `widthOffset` then adjusts
 * whichever width won, and is measured per side — a `+2` widens the housing by
 * two points on the left and two on the right, because that is how it looks to
 * someone matching an outline against their own bezel.
 */
export function notchWidthFor(
	display,
	{ overrideWidth = null, widthOffset = 0, measured = null } = {},
) {
	// A measurement from AppKit is the truth about the hardware, including the
	// truth that there is none. Everything below it is inference.
	if (measured && measured.hasNotch === false) return 0;

	const menuBarHeight = menuBarHeightFor(display);
	// The override corrects a measured width. It does not decide that a notch
	// exists: forcing one onto a flat bezel would draw fillets into nothing.
	if (!measured?.hasNotch && menuBarHeight < NOTCH_MENU_BAR_THRESHOLD) return 0;

	const base = positivePoints(overrideWidth)
		?? positivePoints(measured?.notchWidth)
		?? Math.round(menuBarHeight * NOTCH_ASPECT);
	return Math.max(MIN_NOTCH_WIDTH, base + 2 * offsetPoints(widthOffset));
}

/**
 * Notch height in points, or 0 when the display has no notch.
 *
 * The menu bar strip is the floor: that is what the island has to cover to read
 * as the hardware, and a negative offset large enough to expose it would undo
 * the illusion the whole shape depends on.
 */
export function notchHeightFor(display, { heightOffset = 0, ...options } = {}) {
	if (notchWidthFor(display, options) <= 0) return 0;

	const base = positivePoints(options.measured?.notchHeight) ?? menuBarHeightFor(display);
	return Math.max(MIN_NOTCH_HEIGHT, base + offsetPoints(heightOffset));
}

/**
 * Full notch description for a display. `notchHeight` starts at the menu bar
 * height: that is the strip the island has to cover to read as the hardware.
 */
export function notchGeometryFor(display, options = {}) {
	if (!display) return DEFAULT_GEOMETRY;

	const menuBarHeight = menuBarHeightFor(display);
	const notchWidth = notchWidthFor(display, options);
	const scaleFactor = Number.isFinite(display.scaleFactor) && display.scaleFactor > 0
		? display.scaleFactor
		: DEFAULT_GEOMETRY.scaleFactor;

	return Object.freeze({
		hasNotch: notchWidth > 0,
		notchWidth,
		notchHeight: notchWidth > 0 ? notchHeightFor(display, options) : 0,
		menuBarHeight,
		scaleFactor,
	});
}

/** The notch geometry options carried by a settings document. */
export function notchOptionsFromSettings(settings, { overrideWidth = null, measured = null } = {}) {
	return {
		overrideWidth,
		measured,
		widthOffset: settings?.notch?.widthOffset ?? 0,
		heightOffset: settings?.notch?.heightOffset ?? 0,
	};
}

export {
	DEFAULT_GEOMETRY,
	MIN_NOTCH_HEIGHT,
	MIN_NOTCH_WIDTH,
	NOTCH_ASPECT,
	NOTCH_MENU_BAR_THRESHOLD,
};

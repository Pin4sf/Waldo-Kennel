import assert from "node:assert/strict";
import test from "node:test";
import {
	DEFAULT_GEOMETRY,
	MIN_NOTCH_HEIGHT,
	MIN_NOTCH_WIDTH,
	menuBarHeightFor,
	notchGeometryFor,
	notchOptionsFromSettings,
	notchWidthFor,
} from "./notch-geometry.mjs";

function displayWithMenuBar(menuBarHeight, { width = 1512, height = 982, scaleFactor = 2 } = {}) {
	return {
		bounds: { x: 0, y: 0, width, height },
		workArea: { x: 0, y: menuBarHeight, width, height: height - menuBarHeight },
		scaleFactor,
	};
}

test("a notched built-in display reports the menu bar strip as the notch height", () => {
	const geometry = notchGeometryFor(displayWithMenuBar(37));

	assert.equal(geometry.hasNotch, true);
	assert.equal(geometry.menuBarHeight, 37);
	assert.equal(geometry.notchHeight, 37);
	assert.equal(geometry.notchWidth, 200);
});

test("a notchless display reports no notch and keeps its menu bar height", () => {
	const geometry = notchGeometryFor(displayWithMenuBar(25, { width: 1920, height: 1080, scaleFactor: 1 }));

	assert.equal(geometry.hasNotch, false);
	assert.equal(geometry.notchWidth, 0);
	assert.equal(geometry.notchHeight, 0);
	assert.equal(geometry.menuBarHeight, 25);
});

test("the derived notch width tracks the point size of the selected resolution", () => {
	const moreSpace = notchGeometryFor(displayWithMenuBar(33, { width: 1800, height: 1169 }));
	const standard = notchGeometryFor(displayWithMenuBar(37));

	assert.ok(moreSpace.hasNotch);
	assert.ok(moreSpace.notchWidth < standard.notchWidth);
});

test("an operator override pins the notch width in points", () => {
	assert.equal(notchWidthFor(displayWithMenuBar(37), { overrideWidth: "204" }), 204);
	assert.equal(notchWidthFor(displayWithMenuBar(37), { overrideWidth: "not-a-number" }), 200);
	assert.equal(notchWidthFor(displayWithMenuBar(37), { overrideWidth: -12 }), 200);
	assert.equal(notchWidthFor(displayWithMenuBar(37), { overrideWidth: 99999 }), 200);
});

test("an override cannot invent a notch on a notchless display", () => {
	// The override corrects a measured width; it does not decide that a notch
	// exists. A notchless panel with a forced width would draw fillets into a
	// flat bezel.
	const geometry = notchGeometryFor(
		displayWithMenuBar(25, { width: 1920, height: 1080, scaleFactor: 1 }),
		{ overrideWidth: 204 },
	);

	assert.equal(geometry.hasNotch, false);
	assert.equal(geometry.notchWidth, 0);
});

test("the width fine tune widens the housing by its offset on each side", () => {
	const base = notchGeometryFor(displayWithMenuBar(37));
	const wider = notchGeometryFor(displayWithMenuBar(37), { widthOffset: 3 });
	const narrower = notchGeometryFor(displayWithMenuBar(37), { widthOffset: -3 });

	assert.equal(wider.notchWidth, base.notchWidth + 6);
	assert.equal(narrower.notchWidth, base.notchWidth - 6);
});

test("the width fine tune adjusts an operator override rather than replacing it", () => {
	assert.equal(notchWidthFor(displayWithMenuBar(37), { overrideWidth: 220, widthOffset: 2 }), 224);
});

test("the height fine tune moves the island, not the display's menu bar", () => {
	assert.equal(notchGeometryFor(displayWithMenuBar(38), { heightOffset: 4 }).notchHeight, 42);
	assert.equal(notchGeometryFor(displayWithMenuBar(38), { heightOffset: -6 }).notchHeight, 32);
	assert.equal(notchGeometryFor(displayWithMenuBar(38), { heightOffset: 4 }).menuBarHeight, 38);
});

test("a fine tune cannot shrink the housing past the floors", () => {
	// The schema clamps long before this, so these floors only matter to a
	// caller that reached the geometry directly.
	assert.equal(
		notchGeometryFor(displayWithMenuBar(37), { overrideWidth: 90, widthOffset: -40 }).notchWidth,
		MIN_NOTCH_WIDTH,
	);
	assert.equal(notchGeometryFor(displayWithMenuBar(38), { heightOffset: -100 }).notchHeight, MIN_NOTCH_HEIGHT);
});

test("a fine tune cannot invent a notch on a notchless display", () => {
	const geometry = notchGeometryFor(
		displayWithMenuBar(25, { width: 1920, height: 1080, scaleFactor: 1 }),
		{ widthOffset: 40, heightOffset: 20 },
	);

	assert.equal(geometry.hasNotch, false);
	assert.equal(geometry.notchWidth, 0);
	assert.equal(geometry.notchHeight, 0);
});

test("unusable offsets are read as no adjustment at all", () => {
	const base = notchGeometryFor(displayWithMenuBar(37));

	for (const widthOffset of ["", null, Number.NaN, {}, 10_000]) {
		assert.equal(notchGeometryFor(displayWithMenuBar(37), { widthOffset }).notchWidth, base.notchWidth);
	}
	// A range input reports its value as a string.
	assert.equal(notchGeometryFor(displayWithMenuBar(37), { widthOffset: "2" }).notchWidth, base.notchWidth + 4);
});

test("a measurement replaces the derivation it was always standing in for", () => {
	// The menu bar derives 200pt on this display; AppKit says the housing is 220.
	const geometry = notchGeometryFor(displayWithMenuBar(37), {
		measured: { hasNotch: true, notchWidth: 220, notchHeight: 38 },
	});

	assert.equal(geometry.notchWidth, 220);
	assert.equal(geometry.notchHeight, 38);
	// The menu bar is still the display's own, and still what the strip clears.
	assert.equal(geometry.menuBarHeight, 37);
});

test("an operator override still outranks the measurement", () => {
	// The override exists for a panel that measures differently from every API
	// that describes it, so nothing may quietly outrank it.
	assert.equal(
		notchWidthFor(displayWithMenuBar(37), {
			overrideWidth: 210,
			measured: { hasNotch: true, notchWidth: 220, notchHeight: 38 },
		}),
		210,
	);
});

test("the fine tune adjusts a measured housing too", () => {
	const geometry = notchGeometryFor(displayWithMenuBar(38), {
		measured: { hasNotch: true, notchWidth: 220, notchHeight: 38 },
		widthOffset: 2,
		heightOffset: 3,
	});

	assert.equal(geometry.notchWidth, 224);
	assert.equal(geometry.notchHeight, 41);
});

test("a measured flat bezel wins over a tall menu bar", () => {
	// A tall menu bar is the only clue the derivation has. A measurement saying
	// there is no housing is better information than a clue.
	const geometry = notchGeometryFor(displayWithMenuBar(38), { measured: { hasNotch: false } });

	assert.equal(geometry.hasNotch, false);
	assert.equal(geometry.notchWidth, 0);
});

test("a measured notch is honoured on a display whose menu bar looks notchless", () => {
	const geometry = notchGeometryFor(displayWithMenuBar(25, { width: 1920, height: 1080 }), {
		measured: { hasNotch: true, notchWidth: 180, notchHeight: 32 },
	});

	assert.equal(geometry.hasNotch, true);
	assert.equal(geometry.notchWidth, 180);
	assert.equal(geometry.notchHeight, 32);
});

test("settings map onto geometry options without reaching into the schema twice", () => {
	const measured = { hasNotch: true, notchWidth: 220, notchHeight: 38 };
	assert.deepEqual(
		notchOptionsFromSettings(
			{ notch: { widthOffset: 3, heightOffset: -1 } },
			{ overrideWidth: "220", measured },
		),
		{ overrideWidth: "220", measured, widthOffset: 3, heightOffset: -1 },
	);
	assert.deepEqual(notchOptionsFromSettings(undefined), {
		overrideWidth: null,
		measured: null,
		widthOffset: 0,
		heightOffset: 0,
	});
});

test("missing display information falls back instead of throwing", () => {
	assert.deepEqual(notchGeometryFor(null), DEFAULT_GEOMETRY);
	assert.equal(menuBarHeightFor(undefined), DEFAULT_GEOMETRY.menuBarHeight);
	assert.equal(menuBarHeightFor({ bounds: { y: 0 }, workArea: { y: 0 } }), DEFAULT_GEOMETRY.menuBarHeight);
});

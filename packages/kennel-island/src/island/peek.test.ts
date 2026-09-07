import assert from "node:assert/strict";
import test from "node:test";
import { ISLAND_HEADER_HEIGHT } from "./layout.ts";
import {
  clusterWidth,
  restingMetricsFor,
  restingShapeFor,
  shouldTapFor,
  type RestingMetricsInput,
} from "./peek.ts";

const stage = {
  notchWidth: 220,
  notchHeight: 38,
  menuBarHeight: 38,
  contentPadding: 12,
  peekWidth: 14,
  peekHeight: 6,
};

function metrics(overrides: Partial<RestingMetricsInput>) {
  return restingMetricsFor({ ...stage, shape: "dormant", hovered: false, clusterItems: 0, ...overrides });
}

test("a quiet island is exactly the camera housing", () => {
  assert.deepEqual(metrics({ shape: "dormant" }), { width: 220, height: 38 });
});

test("a peek grows on every side, so the housing is never left uncovered", () => {
  const peek = metrics({ shape: "peek" });

  assert.equal(peek.width, 220 + 2 * 14);
  assert.equal(peek.height, 38 + 6);
});

test("a peek with both offsets at zero is the housing, not a flicker", () => {
  assert.deepEqual(metrics({ shape: "peek", peekWidth: 0, peekHeight: 0 }), { width: 220, height: 38 });
});

test("negative peek offsets cannot pull the shape inside the housing", () => {
  assert.deepEqual(metrics({ shape: "peek", peekWidth: -20, peekHeight: -20 }), { width: 220, height: 38 });
});

test("an awake strip pads both sides equally around the housing", () => {
  const strip = metrics({ shape: "strip", clusterItems: 2 });

  assert.equal(strip.width, 220 + 2 * clusterWidth(2) + 2 * 12);
});

test("an untouched strip is exactly as tall as the housing, awake or not", () => {
  // The whole illusion is that nothing was added to the bezel. A strip taller
  // than the housing hangs a black lip below the hardware the moment a song
  // starts, which is the one thing a quiet machine must never do.
  assert.equal(metrics({ shape: "strip", clusterItems: 2 }).height, 38);
  assert.equal(metrics({ shape: "dormant" }).height, 38);
});

test("a hovered strip may take the header's height, because a pointer asked", () => {
  assert.equal(metrics({ shape: "strip", clusterItems: 2, hovered: true }).height, ISLAND_HEADER_HEIGHT);
});

test("a fine-tuned housing taller than the header still wins", () => {
  const tall = metrics({ shape: "strip", clusterItems: 2, hovered: true, notchHeight: 60, menuBarHeight: 38 });
  assert.equal(tall.height, 60);
});

test("an empty strip carries no padding it has nothing to pad", () => {
  assert.equal(metrics({ shape: "strip", clusterItems: 0 }).width, 220);
});

test("a strip is never shorter than the housing it straddles", () => {
  // Someone who fine-tuned the notch taller has told us the hardware is bigger
  // than the menu bar implies.
  const strip = metrics({ shape: "strip", clusterItems: 1, notchHeight: 52 });
  assert.equal(strip.height, 52);
});

test("a strip still clears a short menu bar on a scaled resolution", () => {
  const stripped = { shape: "strip", clusterItems: 1, menuBarHeight: 33, notchHeight: 33 } as const;

  // Untouched it matches the shorter housing exactly — covering the bar is the
  // job, growing past it is not.
  assert.equal(metrics(stripped).height, 33);
  // Hovered, the header's own height is what the chips need to breathe.
  assert.equal(metrics({ ...stripped, hovered: true }).height, ISLAND_HEADER_HEIGHT);
});

test("cluster width counts the gaps between items, not after them", () => {
  assert.equal(clusterWidth(0), 0);
  assert.equal(clusterWidth(1), 24);
  assert.equal(clusterWidth(2), 54);
});

test("an awake island shows its strip rather than peeking over it", () => {
  assert.equal(restingShapeFor({ awake: true, peeking: true }), "strip");
  assert.equal(restingShapeFor({ awake: true, peeking: false }), "strip");
});

test("a quiet island peeks only while the pointer is committed to it", () => {
  assert.equal(restingShapeFor({ awake: false, peeking: true }), "peek");
  assert.equal(restingShapeFor({ awake: false, peeking: false }), "dormant");
});

test("only arriving at a peek is worth a tap on the trackpad", () => {
  assert.equal(shouldTapFor("dormant", "peek"), true);
  assert.equal(shouldTapFor("peek", "peek"), false);
  assert.equal(shouldTapFor("peek", "dormant"), false);
  // Waking happens with the hand somewhere else entirely.
  assert.equal(shouldTapFor("dormant", "strip"), false);
  assert.equal(shouldTapFor("strip", "dormant"), false);
});

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
  return restingMetricsFor({ ...stage, shape: "dormant", clusterItems: 0, ...overrides });
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
  assert.equal(strip.height, ISLAND_HEADER_HEIGHT);
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
  const strip = metrics({ shape: "strip", clusterItems: 1, menuBarHeight: 33, notchHeight: 33 });
  assert.equal(strip.height, ISLAND_HEADER_HEIGHT);
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

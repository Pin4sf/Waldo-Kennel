import assert from "node:assert/strict";
import test from "node:test";
import {
  isPointerInside,
  POINTER_HIT_TOLERANCE,
  shouldCollapseOnPointerLeave,
} from "./stage-rules.ts";

const body = { left: 100, right: 300, top: 0, bottom: 80 };

test("the island hit target extends by the fillet tolerance on every edge", () => {
  assert.equal(isPointerInside(body, 200, 40), true);
  assert.equal(isPointerInside(body, 100 - POINTER_HIT_TOLERANCE, 40), true);
  assert.equal(isPointerInside(body, 300 + POINTER_HIT_TOLERANCE, 40), true);
  assert.equal(isPointerInside(body, 200, 80 + POINTER_HIT_TOLERANCE), true);

  assert.equal(isPointerInside(body, 100 - POINTER_HIT_TOLERANCE - 1, 40), false);
  assert.equal(isPointerInside(body, 200, 80 + POINTER_HIT_TOLERANCE + 1), false);
});

test("a queue the pointer never reached is not collapsed under it", () => {
  // Summoned from the keyboard with the pointer across the screen: hovered has
  // been false the whole time, so there is no leave to react to.
  assert.equal(shouldCollapseOnPointerLeave("queue", false, false), false);
});

test("a queue collapses once the pointer that was on it leaves", () => {
  assert.equal(shouldCollapseOnPointerLeave("queue", true, false), true);
  assert.equal(shouldCollapseOnPointerLeave("usage", true, false), true);
});

test("a pointer still on the island collapses nothing", () => {
  assert.equal(shouldCollapseOnPointerLeave("queue", true, true), false);
});

test("surfaces waiting on an answer are never collapsed by the pointer", () => {
  for (const surface of ["permission", "choice", "steer", "compact"] as const) {
    assert.equal(shouldCollapseOnPointerLeave(surface, true, false), false);
  }
});

test("an untracked pointer never collapses anything", () => {
  // `null` is "no host is reporting the pointer", not "the pointer is away".
  assert.equal(shouldCollapseOnPointerLeave("queue", true, null), false);
  assert.equal(shouldCollapseOnPointerLeave("queue", false, null), false);
});

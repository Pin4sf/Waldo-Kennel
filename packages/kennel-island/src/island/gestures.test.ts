import assert from "node:assert/strict";
import test from "node:test";
import {
  createGestureRecognizer,
  fingerTravel,
  GESTURE_IDLE_MS,
  GESTURE_TOGGLE_THRESHOLD,
  GESTURE_TRACK_THRESHOLD,
  scrollCanConsume,
  type GestureSample,
  type IslandGesture,
} from "./gestures.ts";

function swipe(
  recognizer: ReturnType<typeof createGestureRecognizer>,
  { x = 0, y = 0, steps = 10, inverted = false, from = 0, gap = 10 },
) {
  const fired: IslandGesture[] = [];
  for (let step = 0; step < steps; step++) {
    const sample: GestureSample = {
      // The recogniser is handed DOM deltas, so a natural-scrolling swipe has
      // to arrive here already flipped, exactly as the browser reports it.
      deltaX: (inverted ? -x : x) / steps,
      deltaY: (inverted ? -y : y) / steps,
      inverted,
      timeStamp: from + step * gap,
    };
    const gesture = recognizer.read(sample);
    if (gesture) fired.push(gesture);
  }
  return fired;
}

test("a swipe down opens and a swipe up closes", () => {
  assert.deepEqual(swipe(createGestureRecognizer(), { y: GESTURE_TOGGLE_THRESHOLD + 2 }), ["open"]);
  assert.deepEqual(swipe(createGestureRecognizer(), { y: -(GESTURE_TOGGLE_THRESHOLD + 2) }), ["close"]);
});

test("natural scrolling does not reverse what a finger meant", () => {
  const plain = swipe(createGestureRecognizer(), { y: GESTURE_TOGGLE_THRESHOLD + 2 });
  const natural = swipe(createGestureRecognizer(), {
    y: GESTURE_TOGGLE_THRESHOLD + 2,
    inverted: true,
  });
  assert.deepEqual(natural, plain);
});

test("a finger moving left takes the next track, right the previous", () => {
  assert.deepEqual(swipe(createGestureRecognizer(), { x: -(GESTURE_TRACK_THRESHOLD + 2) }), [
    "next-track",
  ]);
  assert.deepEqual(swipe(createGestureRecognizer(), { x: GESTURE_TRACK_THRESHOLD + 2 }), [
    "previous-track",
  ]);
});

test("a swipe short of the threshold is not a gesture", () => {
  assert.deepEqual(swipe(createGestureRecognizer(), { y: GESTURE_TOGGLE_THRESHOLD - 1 }), []);
  assert.deepEqual(swipe(createGestureRecognizer(), { x: -(GESTURE_TRACK_THRESHOLD - 1) }), []);
});

test("one long swipe fires once, however far it runs", () => {
  const recognizer = createGestureRecognizer();
  assert.deepEqual(swipe(recognizer, { x: -GESTURE_TRACK_THRESHOLD * 6, steps: 60 }), [
    "next-track",
  ]);
});

test("fingers lifting between swipes lets the next one fire", () => {
  const recognizer = createGestureRecognizer();
  assert.deepEqual(swipe(recognizer, { x: -(GESTURE_TRACK_THRESHOLD + 2) }), ["next-track"]);
  const second = swipe(recognizer, {
    x: -(GESTURE_TRACK_THRESHOLD + 2),
    from: 1000 + GESTURE_IDLE_MS,
  });
  assert.deepEqual(second, ["next-track"]);
});

test("a swipe that drifts across keeps the axis it started on", () => {
  const recognizer = createGestureRecognizer();
  // Down first, then a long sideways drift: still one open, never a track step.
  const fired = [
    ...swipe(recognizer, { y: GESTURE_TOGGLE_THRESHOLD + 2, steps: 8 }),
    ...swipe(recognizer, { x: -GESTURE_TRACK_THRESHOLD * 3, steps: 8, from: 80 }),
  ];
  assert.deepEqual(fired, ["open"]);
});

test("a diagonal swipe commits to whichever axis led", () => {
  const recognizer = createGestureRecognizer();
  const fired: IslandGesture[] = [];
  for (let step = 0; step < 20; step++) {
    const gesture = recognizer.read({
      deltaX: -1,
      deltaY: -4,
      inverted: false,
      timeStamp: step * 10,
    });
    if (gesture) fired.push(gesture);
  }
  assert.deepEqual(fired, ["close"]);
});

test("reset drops a swipe that was already under way", () => {
  const recognizer = createGestureRecognizer();
  swipe(recognizer, { y: GESTURE_TOGGLE_THRESHOLD * 0.8, steps: 8 });
  recognizer.reset();
  assert.deepEqual(swipe(recognizer, { y: GESTURE_TOGGLE_THRESHOLD * 0.5, steps: 5, from: 200 }), []);
});

test("finger travel reports the direction the fingers moved", () => {
  const natural = fingerTravel({ deltaX: 3, deltaY: -5, inverted: true, timeStamp: 0 });
  assert.deepEqual(natural, { x: -3, y: 5 });

  const plain = fingerTravel({ deltaX: 3, deltaY: -5, inverted: false, timeStamp: 0 });
  assert.deepEqual(plain, { x: 3, y: -5 });
});

test("a list with room left keeps its own wheel events", () => {
  const list = { overflowY: "auto", scrollTop: 40, scrollHeight: 400, clientHeight: 200 };
  assert.equal(scrollCanConsume(list, 10), true);
  assert.equal(scrollCanConsume(list, -10), true);
});

test("a list scrolled to an end releases the wheel in that direction", () => {
  const top = { overflowY: "auto", scrollTop: 0, scrollHeight: 400, clientHeight: 200 };
  assert.equal(scrollCanConsume(top, -10), false);
  assert.equal(scrollCanConsume(top, 10), true);

  const bottom = { overflowY: "auto", scrollTop: 200, scrollHeight: 400, clientHeight: 200 };
  assert.equal(scrollCanConsume(bottom, 10), false);
  assert.equal(scrollCanConsume(bottom, -10), true);
});

test("a list short enough to fit never takes the wheel", () => {
  const short = { overflowY: "auto", scrollTop: 0, scrollHeight: 120, clientHeight: 200 };
  assert.equal(scrollCanConsume(short, 10), false);
  assert.equal(scrollCanConsume(short, -10), false);
});

test("content sticking out of a visible box scrolls nothing", () => {
  // What the island itself looks like for the length of every open animation:
  // the sprung height is still short of the content, so the box measures as
  // overflowing without being scrollable. Swallowing the gesture here would
  // make a second swipe during an animation do nothing at all.
  const midAnimation = { overflowY: "visible", scrollTop: 0, scrollHeight: 184, clientHeight: 44 };
  assert.equal(scrollCanConsume(midAnimation, 10), false);
  assert.equal(scrollCanConsume(midAnimation, -10), false);

  const clipped = { overflowY: "hidden", scrollTop: 0, scrollHeight: 400, clientHeight: 200 };
  assert.equal(scrollCanConsume(clipped, 10), false);
});

import assert from "node:assert/strict";
import test from "node:test";
import {
  nextPresenceIndex,
  orderPresenceCards,
  presenceRank,
  presenceRotationKey,
  safePresenceIndex,
} from "./presence.ts";
import type { IslandPresence, IslandPresenceCard } from "./types";

function card(presence: IslandPresence, count: number): IslandPresenceCard {
  return {
    presence,
    count,
    title: `${presence} session`,
    project: "Kennel",
    branch: "main",
    agent: "codex",
    detail: presence,
  };
}

test("cards are ordered by urgency, not by arrival", () => {
  const ordered = orderPresenceCards([card("running", 2), card("blocked", 4), card("paused", 1)]);

  assert.deepEqual(ordered.map((entry) => entry.presence), ["blocked", "paused", "running"]);
  assert.deepEqual(ordered.map((entry) => entry.count), [4, 1, 2]);
});

test("a presence with no sessions never takes a turn", () => {
  const ordered = orderPresenceCards([card("running", 0), card("blocked", 3)]);

  assert.deepEqual(ordered.map((entry) => entry.presence), ["blocked"]);
});

test("duplicate presences merge into one card carrying the total", () => {
  const ordered = orderPresenceCards([card("running", 2), card("running", 3)]);

  assert.equal(ordered.length, 1);
  assert.equal(ordered[0].count, 5);
});

test("nothing live produces no cards at all", () => {
  // The island has nothing to say, so it shrinks back onto the notch.
  assert.deepEqual(orderPresenceCards([]), []);
  assert.deepEqual(orderPresenceCards([card("paused", 0)]), []);
});

test("urgency ranks blocked above paused above running", () => {
  assert.ok(presenceRank("blocked") < presenceRank("paused"));
  assert.ok(presenceRank("paused") < presenceRank("running"));
});

test("a new presence restarts the cycle but a changing count does not", () => {
  const running = presenceRotationKey(orderPresenceCards([card("running", 2)]));
  const runningGrew = presenceRotationKey(orderPresenceCards([card("running", 5)]));
  const approvalArrived = presenceRotationKey(
    orderPresenceCards([card("running", 2), card("blocked", 1)]),
  );

  assert.equal(running, runningGrew);
  assert.notEqual(running, approvalArrived);
});

test("the cycle wraps and survives a list shrinking under it", () => {
  assert.equal(nextPresenceIndex(0, 3), 1);
  assert.equal(nextPresenceIndex(2, 3), 0);
  assert.equal(nextPresenceIndex(0, 1), 0);
  assert.equal(nextPresenceIndex(0, 0), 0);

  assert.equal(safePresenceIndex(2, 1), 0);
  assert.equal(safePresenceIndex(5, 0), 0);
  assert.equal(safePresenceIndex(-1, 3), 0);
});

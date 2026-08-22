import assert from "node:assert/strict";
import test from "node:test";
import { HEADER_ITEM } from "./peek.ts";
import {
  otherSubject,
  peekMaxWidthFor,
  peekSubjectFor,
  peekWidthFor,
  peekZonesFor,
  pointerHasMoved,
  PEEK_MIN_WIDTH,
  PEEK_TEXT_PADDING,
  zoneAt,
  type HeaderItemBox,
} from "./peek-layout.ts";

/* The strip in the reference: a 220pt housing, 12pt padding, one chip a side. */
const strip = { stripWidth: 292, contentPadding: 12, hasItems: true };

test("the peek stops at the midlines of the outermost chips", () => {
  // 292 minus a padding and half a chip on each side.
  assert.equal(peekMaxWidthFor(strip), 292 - 24 - HEADER_ITEM);
});

test("a strip with no chips has no midlines to stop at", () => {
  assert.equal(peekMaxWidthFor({ ...strip, hasItems: false }), 292);
});

test("content that fits sizes the bar to itself", () => {
  const peek = peekWidthFor({ ...strip, contentWidth: 180 });

  assert.equal(peek.width, 180 + 2 * PEEK_TEXT_PADDING);
  assert.equal(peek.scrolls, false);
});

test("content that cannot fit fills the bound and travels", () => {
  const peek = peekWidthFor({ ...strip, contentWidth: 900 });

  assert.equal(peek.width, peek.maxWidth);
  assert.equal(peek.scrolls, true);
});

test("content exactly at the bound fills it without travelling", () => {
  const maxWidth = peekMaxWidthFor(strip);
  const peek = peekWidthFor({ ...strip, contentWidth: maxWidth - 2 * PEEK_TEXT_PADDING });

  assert.equal(peek.width, maxWidth);
  assert.equal(peek.scrolls, false);
});

test("a very short label still gets a bar rather than a fragment of one", () => {
  const peek = peekWidthFor({ ...strip, contentWidth: 4 });

  assert.equal(peek.width, PEEK_MIN_WIDTH);
  assert.equal(peek.scrolls, false);
});

test("the bar shrinks when the content does, which is what a track change is", () => {
  const long = peekWidthFor({ ...strip, contentWidth: 200 });
  const short = peekWidthFor({ ...strip, contentWidth: 140 });

  assert.ok(short.width < long.width);
  assert.equal(short.scrolls, false);
});

/* -------------------------------------------------------------------------- */
/* Zones                                                                       */
/* -------------------------------------------------------------------------- */

function box(kind: HeaderItemBox["kind"], left: number): HeaderItemBox {
  return { kind, left, right: left + HEADER_ITEM };
}

const bounds = { left: 0, right: 292 };

test("media chips either side of the housing are one zone spanning it", () => {
  // State 1: album art, housing, waveform.
  const zones = peekZonesFor([box("media", 12), box("media", 256)], bounds);

  assert.equal(zones.length, 1);
  assert.equal(zones[0].subject, "media");
  assert.deepEqual([zones[0].left, zones[0].right], [0, 292]);
});

test("a session chip outside a media chip splits the strip three ways", () => {
  // State 2: provider glyph, album art, housing, waveform, count badge.
  const zones = peekZonesFor(
    [box("session", 12), box("media", 42), box("media", 226), box("session", 256)],
    bounds,
  );

  assert.deepEqual(
    zones.map((zone) => zone.subject),
    ["session", "media", "session"],
  );
  // The boundary falls in the middle of the gap between the two kinds.
  assert.equal(zones[0].right, (36 + 42) / 2);
  assert.equal(zones[2].left, (250 + 256) / 2);
});

test("the outer edges of the strip belong to the zones that reach them", () => {
  const zones = peekZonesFor([box("session", 12), box("media", 226)], bounds);

  assert.equal(zones[0].left, bounds.left);
  assert.equal(zones.at(-1)?.right, bounds.right);
});

test("a strip with no chips has no zones", () => {
  assert.deepEqual(peekZonesFor([], bounds), []);
});

test("a point resolves to the zone containing it, and to nothing outside", () => {
  const zones = peekZonesFor(
    [box("session", 12), box("media", 42), box("media", 226), box("session", 256)],
    bounds,
  );

  assert.equal(zoneAt(zones, 20)?.subject, "session");
  assert.equal(zoneAt(zones, 146)?.subject, "media");
  assert.equal(zoneAt(zones, 280)?.subject, "session");
  assert.equal(zoneAt(zones, -40), null);
  assert.equal(zoneAt(zones, 900), null);
});

/* -------------------------------------------------------------------------- */
/* Subject                                                                     */
/* -------------------------------------------------------------------------- */

test("a zone gets what it asks for when there is something to say", () => {
  assert.equal(
    peekSubjectFor({ zone: "media", hasMedia: true, hasSession: true }),
    "media",
  );
  assert.equal(
    peekSubjectFor({ zone: "session", hasMedia: true, hasSession: true }),
    "session",
  );
});

test("hovering a subject with nothing to say falls through rather than opening empty", () => {
  assert.equal(
    peekSubjectFor({ zone: "media", hasMedia: false, hasSession: true }),
    "session",
  );
  assert.equal(
    peekSubjectFor({ zone: "session", hasMedia: true, hasSession: false }),
    "media",
  );
});

test("with nothing running and nothing playing the peek has no subject", () => {
  assert.equal(peekSubjectFor({ zone: "media", hasMedia: false, hasSession: false }), null);
});

test("the idle rotation only decides when the pointer is not over a zone", () => {
  assert.equal(
    peekSubjectFor({ zone: null, hasMedia: true, hasSession: true, rotated: "media" }),
    "media",
  );
  // A pointer inside a zone outranks the rotation; it is a deliberate choice.
  assert.equal(
    peekSubjectFor({ zone: "session", hasMedia: true, hasSession: true, rotated: "media" }),
    "session",
  );
});

test("a rotation pointing at nothing falls through to what there is", () => {
  assert.equal(
    peekSubjectFor({ zone: null, hasMedia: false, hasSession: true, rotated: "media" }),
    "session",
  );
});

test("the other subject is the other one", () => {
  assert.equal(otherSubject("media"), "session");
  assert.equal(otherSubject("session"), "media");
});

test("a pointer that drifts a point or two is still parked", () => {
  assert.equal(pointerHasMoved({ x: 100, y: 20 }, { x: 101, y: 21 }), false);
  assert.equal(pointerHasMoved({ x: 100, y: 20 }, { x: 120, y: 20 }), true);
  assert.equal(pointerHasMoved({ x: 100, y: 20 }, { x: 100, y: 40 }), true);
  // Nothing to compare against is a move, so the timer starts rather than
  // treating an unknown position as parked.
  assert.equal(pointerHasMoved(null, { x: 100, y: 20 }), true);
});

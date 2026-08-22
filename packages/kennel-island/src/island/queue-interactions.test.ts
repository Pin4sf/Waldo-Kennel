import assert from "node:assert/strict";
import test from "node:test";

import {
  openSessionActionForQueueRow,
  shouldDispatchQueueTaskAction,
} from "./queue-interactions.ts";

test("a queue-row double click carries only its typed session target", () => {
  assert.deepEqual(
    openSessionActionForQueueRow({
      projectId: "project-waldo",
      sessionId: "session-real-42",
    }),
    {
      type: "open-session",
      projectId: "project-waldo",
      sessionId: "session-real-42",
    },
  );
  assert.deepEqual(openSessionActionForQueueRow({}), { type: "open-session" });
});

test("a nested queue action dispatches once even when it is double-clicked", () => {
  assert.equal(shouldDispatchQueueTaskAction(0), true);
  assert.equal(shouldDispatchQueueTaskAction(1), true);
  assert.equal(shouldDispatchQueueTaskAction(2), false);
  assert.equal(shouldDispatchQueueTaskAction(3), false);
});

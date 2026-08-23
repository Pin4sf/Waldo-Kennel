import assert from "node:assert/strict";
import test from "node:test";
import { createDemoIslandAdapter, compactForLiveActivity } from "../fixtures/island.ts";
import {
  applyLiveActivityDemoAction,
  liveActivityReferences,
  liveActivityScenarioIds,
} from "../fixtures/live-activities.ts";

const expectedKinds = [
  "voice-recording",
  "delivery",
  "ride",
  "transit",
  "flight",
  "sports",
  "focus",
  "workout",
  "charging",
  "camera",
  "weather",
  "home",
  "multiple",
];

test("the reference catalog covers each scoped live activity once", () => {
  const activities = liveActivityScenarioIds.map((scenario) => liveActivityReferences[scenario]);

  assert.deepEqual(activities.map((activity) => activity.kind), expectedKinds);
  assert.equal(new Set(activities.map((activity) => activity.id)).size, activities.length);
  assert.equal(new Set(liveActivityScenarioIds).size, liveActivityScenarioIds.length);
});

test("every reference has bounded compact and expanded presentation data", () => {
  for (const scenario of liveActivityScenarioIds) {
    const activity = liveActivityReferences[scenario];

    assert.ok(activity.title.trim(), `${scenario} needs a title`);
    assert.ok(activity.compactValue.trim(), `${scenario} needs a compact value`);
    assert.ok(activity.primaryValue.trim(), `${scenario} needs a primary value`);
    assert.match(activity.source, /pattern reference$/);
    assert.ok((activity.metrics?.length ?? 0) <= 3, `${scenario} has too many metrics`);
    assert.ok((activity.events?.length ?? 0) <= 2, `${scenario} has too many events`);
    assert.ok((activity.companions?.length ?? 0) <= 2, `${scenario} has too many companion activities`);
    assert.ok((activity.actions?.length ?? 0) <= 2, `${scenario} has too many actions`);
    if (activity.progress !== undefined) {
      assert.ok(activity.progress >= 0 && activity.progress <= 1, `${scenario} progress is normalized`);
    }
  }
});

test("each demo opens and closes without losing the selected activity", () => {
  const adapter = createDemoIslandAdapter();

  for (const scenario of liveActivityScenarioIds) {
    adapter.setScenario(scenario);
    const compact = adapter.getSnapshot();
    assert.equal(compact.surface, "compact");
    if (compact.surface !== "compact" || !compact.liveActivity) assert.fail(`${scenario} did not start compact`);
    const id = compact.liveActivity.id;

    adapter.dispatch({ type: "expand" });
    const expanded = adapter.getSnapshot();
    assert.equal(expanded.surface, "activity");
    if (expanded.surface !== "activity") assert.fail(`${scenario} did not expand`);
    assert.equal(expanded.activity.id, id);

    adapter.dispatch({ type: "collapse" });
    const collapsed = adapter.getSnapshot();
    assert.equal(collapsed.surface, "compact");
    if (collapsed.surface !== "compact") assert.fail(`${scenario} did not collapse`);
    assert.equal(collapsed.liveActivity?.id, id);

    adapter.dispatch({ type: "expand" });
    adapter.dispatch({ type: "dismiss" });
    const dismissed = adapter.getSnapshot();
    assert.equal(dismissed.surface, "compact");
    if (dismissed.surface !== "compact") assert.fail(`${scenario} did not dismiss`);
    assert.equal(dismissed.liveActivity?.id, id);
  }
});

test("reference controls update only the local presentation state", () => {
  const original = liveActivityReferences["activity-recording"];
  const paused = applyLiveActivityDemoAction(original, "pause-recording");
  assert.equal(paused.state, "paused");
  assert.match(paused.feedback ?? "", /no external app was contacted/i);

  const resumed = applyLiveActivityDemoAction(paused, "resume-recording");
  assert.equal(resumed.state, "live");

  const complete = applyLiveActivityDemoAction(resumed, "stop-recording");
  assert.equal(complete.state, "complete");
  assert.equal(complete.status, "Recording saved");
  assert.equal(complete.progress, original.progress);
  assert.equal(compactForLiveActivity(complete).liveActivity?.id, original.id);

  const charging = liveActivityReferences["activity-charging"];
  const stopped = applyLiveActivityDemoAction(charging, "stop-charging");
  assert.equal(stopped.state, "ended");
  assert.equal(stopped.status, "Stopped");
  assert.equal(stopped.progress, charging.progress);
  assert.equal(stopped.primaryValue, "68%");
});

test("the demo adapter rejects an activity action aimed at another fixture", () => {
  const adapter = createDemoIslandAdapter();
  adapter.setScenario("activity-ride");
  adapter.dispatch({ type: "expand" });

  const before = adapter.getSnapshot();
  adapter.dispatch({ type: "activity-action", activityId: "another-activity", actionId: "open-ride" });
  assert.equal(adapter.getSnapshot(), before);
});

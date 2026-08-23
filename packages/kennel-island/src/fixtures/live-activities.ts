import type { LiveActivityReference } from "../island/types";

export const liveActivityScenarioIds = [
  "activity-recording",
  "activity-delivery",
  "activity-ride",
  "activity-transit",
  "activity-flight",
  "activity-sports",
  "activity-focus",
  "activity-workout",
  "activity-charging",
  "activity-camera",
  "activity-weather",
  "activity-home",
  "activity-concurrent",
] as const;

export type LiveActivityDemoScenario = (typeof liveActivityScenarioIds)[number];

/**
 * Fixed, reviewable snapshots of common iPhone Live Activity patterns.
 * Vendor names identify the behavior being studied; no vendor data, art, or
 * integration is used by the prototype.
 */
export const liveActivityReferences: Record<LiveActivityDemoScenario, LiveActivityReference> = {
  "activity-recording": {
    id: "voice-memo-recording",
    kind: "voice-recording",
    mechanism: "system",
    source: "Apple Voice Memos pattern reference",
    pattern: "Continuous elapsed metric",
    title: "Voice recording",
    context: "Interview notes",
    status: "Recording",
    state: "live",
    compactValue: "12:48",
    primaryValue: "12:48",
    primaryLabel: "Elapsed recording time",
    timeLabel: "12 minutes, 48 seconds elapsed",
    metrics: [
      { id: "quality", label: "Quality", value: "Lossless" },
      { id: "storage", label: "Available", value: "2h 14m" },
    ],
    events: [
      { id: "started", label: "Recording started", timeLabel: "12:48 ago" },
      { id: "marker", label: "Marker added", detail: "Interview question 4", timeLabel: "03:16" },
    ],
    actions: [
      { id: "pause-recording", label: "Pause", role: "secondary" },
      { id: "stop-recording", label: "Stop", role: "destructive" },
    ],
  },
  "activity-delivery": {
    id: "food-order-delivery",
    kind: "delivery",
    mechanism: "activitykit",
    source: "Domino’s order pattern reference",
    pattern: "Phase progression + ETA",
    title: "Food delivery",
    context: "Order #1842",
    status: "Out for delivery",
    state: "live",
    compactValue: "26 min",
    primaryValue: "26 min",
    primaryLabel: "Estimated arrival",
    timeLabel: "Estimated arrival in 26 minutes",
    progress: 0.76,
    progressLabel: "Driver is on the way",
    steps: [
      { id: "confirmed", label: "Confirmed", state: "complete" },
      { id: "preparing", label: "Preparing", state: "complete" },
      { id: "delivery", label: "On the way", state: "current" },
      { id: "arrived", label: "Delivered", state: "upcoming" },
    ],
    metrics: [
      { id: "driver", label: "Courier", value: "Sam" },
      { id: "dropoff", label: "Drop-off", value: "Front desk" },
    ],
    actions: [
      { id: "contact-courier", label: "Contact", role: "secondary" },
      { id: "open-order", label: "Open order", role: "primary" },
    ],
  },
  "activity-ride": {
    id: "ride-pickup",
    kind: "ride",
    mechanism: "activitykit",
    source: "Uber ride pattern reference",
    pattern: "Countdown + next action",
    title: "Ride pickup",
    context: "Terminal 2 pickup",
    status: "Driver arriving",
    state: "live",
    compactValue: "3 min",
    primaryValue: "3 min",
    primaryLabel: "Driver arrival",
    timeLabel: "Driver arrives in 3 minutes",
    progress: 0.64,
    progressLabel: "Walk to pickup zone B",
    steps: [
      { id: "matched", label: "Matched", state: "complete" },
      { id: "arriving", label: "Arriving", state: "current" },
      { id: "trip", label: "On trip", state: "upcoming" },
    ],
    metrics: [
      { id: "car", label: "Vehicle", value: "White Prius" },
      { id: "plate", label: "Plate", value: "8ABC123" },
      { id: "pickup", label: "Meet at", value: "Zone B" },
    ],
    actions: [
      { id: "share-ride", label: "Share trip", role: "secondary" },
      { id: "open-ride", label: "Open ride", role: "primary" },
    ],
  },
  "activity-transit": {
    id: "transit-go-navigation",
    kind: "transit",
    mechanism: "activitykit",
    source: "Transit GO pattern reference",
    pattern: "Next instruction",
    title: "Transit navigation",
    context: "M15 toward East Harlem",
    status: "Stay on the bus",
    state: "live",
    compactValue: "2 stops",
    primaryValue: "2 stops",
    primaryLabel: "Until your stop",
    timeLabel: "About 6 minutes remaining",
    progress: 0.72,
    progressLabel: "Get off at 34 St",
    steps: [
      { id: "boarded", label: "Boarded", state: "complete" },
      { id: "riding", label: "2 stops", state: "current" },
      { id: "exit", label: "Get off", state: "upcoming" },
    ],
    metrics: [
      { id: "arrival", label: "Arrive", value: "10:42" },
      { id: "walk", label: "Then walk", value: "4 min" },
    ],
    actions: [
      { id: "end-navigation", label: "End GO", role: "destructive" },
      { id: "open-transit", label: "Open route", role: "primary" },
    ],
  },
  "activity-flight": {
    id: "flight-delay",
    kind: "flight",
    mechanism: "activitykit",
    source: "Flighty flight pattern reference",
    pattern: "Exception requiring attention",
    title: "DL 143 to Seattle",
    context: "San Francisco · Terminal 1",
    status: "Gate changed",
    state: "attention",
    compactValue: "B12 · 18m",
    primaryValue: "B12",
    primaryLabel: "New departure gate",
    timeLabel: "Boarding in 18 minutes",
    progress: 0.58,
    progressLabel: "Boarding starts at 7:12 PM",
    alert: {
      title: "Gate changed from B7 to B12",
      detail: "Allow about 6 minutes to walk to the new gate.",
    },
    metrics: [
      { id: "departure", label: "Departure", value: "7:42 PM", detail: "+24 min" },
      { id: "seat", label: "Seat", value: "18A" },
    ],
    events: [
      { id: "delay", label: "Departure delayed", detail: "24 minutes", timeLabel: "6:31 PM" },
      { id: "gate", label: "Gate changed", detail: "B7 → B12", timeLabel: "6:47 PM" },
    ],
    actions: [{ id: "open-flight", label: "Open flight", role: "primary" }],
  },
  "activity-sports": {
    id: "baseball-score",
    kind: "sports",
    mechanism: "activitykit",
    source: "MLB / FotMob score pattern reference",
    pattern: "Event stream + score",
    title: "Yankees 4 · Red Sox 3",
    context: "Fenway Park",
    status: "Top 8th",
    state: "live",
    compactValue: "4–3",
    primaryValue: "4  –  3",
    primaryLabel: "Yankees lead · top of the 8th",
    timeLabel: "Game in progress",
    metrics: [
      { id: "count", label: "Count", value: "2–1" },
      { id: "outs", label: "Outs", value: "1" },
      { id: "runners", label: "On base", value: "1st, 2nd" },
    ],
    events: [
      { id: "single", label: "Volpe singled to left", detail: "Judge to second", timeLabel: "now" },
      { id: "run", label: "Judge homered", detail: "Yankees took the lead", timeLabel: "7th" },
    ],
    actions: [{ id: "open-game", label: "Open game", role: "primary" }],
  },
  "activity-focus": {
    id: "focus-session",
    kind: "focus",
    mechanism: "activitykit",
    source: "Structured / Timery pattern reference",
    pattern: "Countdown",
    title: "Deep work",
    context: "Finish research notes",
    status: "Focus session",
    state: "live",
    compactValue: "18:24",
    primaryValue: "18:24",
    primaryLabel: "Remaining",
    timeLabel: "18 minutes, 24 seconds remaining",
    progress: 0.39,
    progressLabel: "12 of 30 minutes complete",
    metrics: [
      { id: "ends", label: "Ends", value: "11:15" },
      { id: "mode", label: "Focus", value: "Do Not Disturb" },
    ],
    actions: [
      { id: "pause-focus", label: "Pause", role: "secondary" },
      { id: "finish-focus", label: "Finish", role: "primary" },
    ],
  },
  "activity-workout": {
    id: "strength-workout",
    kind: "workout",
    mechanism: "activitykit",
    source: "SmartGym workout pattern reference",
    pattern: "Continuous metric + next action",
    title: "Bench press",
    context: "Upper body · Exercise 3 of 6",
    status: "Set 3 of 4",
    state: "live",
    compactValue: "Set 3/4",
    primaryValue: "0:42",
    primaryLabel: "Rest remaining",
    timeLabel: "42 seconds of rest remaining",
    progress: 0.68,
    progressLabel: "Next: 8 reps at 70 kg",
    metrics: [
      { id: "weight", label: "Weight", value: "70 kg" },
      { id: "reps", label: "Last set", value: "8 reps" },
      { id: "volume", label: "Volume", value: "1,680 kg" },
    ],
    actions: [
      { id: "next-set", label: "Start next set", role: "primary" },
      { id: "end-workout", label: "End", role: "destructive" },
    ],
  },
  "activity-charging": {
    id: "ev-charging",
    kind: "charging",
    mechanism: "activitykit",
    source: "Tesla charging pattern reference",
    pattern: "Continuous progress + ETA",
    title: "Vehicle charging",
    context: "Supercharger · Stall 3A",
    status: "Charging",
    state: "live",
    compactValue: "68%",
    primaryValue: "68%",
    primaryLabel: "Battery charge",
    timeLabel: "24 minutes to the 80 percent limit",
    progress: 0.68,
    progressLabel: "24 min remaining to charge limit",
    metrics: [
      { id: "rate", label: "Rate", value: "142 kW" },
      { id: "added", label: "Added", value: "31 kWh" },
      { id: "cost", label: "Session", value: "$10.54" },
    ],
    actions: [
      { id: "stop-charging", label: "Stop charging", role: "destructive" },
      { id: "open-vehicle", label: "Open vehicle", role: "primary" },
    ],
  },
  "activity-camera": {
    id: "camera-recording",
    kind: "camera",
    mechanism: "activitykit",
    source: "Insta360 recording pattern reference",
    pattern: "Continuous metric + exception",
    title: "Camera recording",
    context: "4K · 60 fps",
    status: "Recording",
    state: "attention",
    compactValue: "04:12",
    primaryValue: "04:12",
    primaryLabel: "Elapsed recording time",
    timeLabel: "4 minutes, 12 seconds elapsed",
    alert: {
      title: "Battery is low",
      detail: "About 7 minutes of recording time remain.",
    },
    metrics: [
      { id: "battery", label: "Battery", value: "18%" },
      { id: "storage", label: "Storage", value: "7 min" },
      { id: "temperature", label: "Camera", value: "Warm" },
    ],
    actions: [
      { id: "pause-camera", label: "Pause", role: "secondary" },
      { id: "stop-camera", label: "Stop", role: "destructive" },
    ],
  },
  "activity-weather": {
    id: "rain-alert",
    kind: "weather",
    mechanism: "activitykit",
    source: "CARROT Weather pattern reference",
    pattern: "Exception + countdown",
    title: "Rain approaching",
    context: "Current location",
    status: "Starts in 12 minutes",
    state: "attention",
    compactValue: "Rain · 12m",
    primaryValue: "12 min",
    primaryLabel: "Until rain starts",
    timeLabel: "Rain starts in 12 minutes",
    progress: 0.62,
    progressLabel: "Rain may last about 35 minutes",
    alert: {
      title: "Heavy rain is approaching",
      detail: "Peak intensity is expected around 4:25 PM.",
    },
    metrics: [
      { id: "chance", label: "Chance", value: "92%" },
      { id: "intensity", label: "Peak", value: "Heavy" },
      { id: "temperature", label: "Now", value: "24°" },
    ],
    actions: [{ id: "open-forecast", label: "Open forecast", role: "primary" }],
  },
  "activity-home": {
    id: "home-automation",
    kind: "home",
    mechanism: "beta",
    source: "Home Assistant beta pattern reference",
    pattern: "Phase progression + immediate action",
    title: "Movie night",
    context: "Living room automation",
    status: "Running step 2 of 4",
    state: "live",
    compactValue: "2 of 4",
    primaryValue: "2 / 4",
    primaryLabel: "Automation steps",
    timeLabel: "Started 18 seconds ago",
    progress: 0.5,
    progressLabel: "Next: dim the living room lights",
    steps: [
      { id: "tv", label: "TV on", state: "complete" },
      { id: "blinds", label: "Close blinds", state: "current" },
      { id: "lights", label: "Dim lights", state: "upcoming" },
      { id: "audio", label: "Set audio", state: "upcoming" },
    ],
    metrics: [
      { id: "devices", label: "Devices", value: "5" },
      { id: "room", label: "Room", value: "Living room" },
    ],
    actions: [
      { id: "stop-automation", label: "Stop", role: "destructive" },
      { id: "open-home", label: "Open home", role: "primary" },
    ],
  },
  "activity-concurrent": {
    id: "concurrent-activities",
    kind: "multiple",
    mechanism: "activitykit",
    source: "Apple concurrency pattern reference",
    pattern: "Multiple concurrent activities",
    title: "2 ongoing activities",
    context: "Delivery + focus timer",
    status: "Two activities are live",
    state: "live",
    compactValue: "2 live",
    primaryValue: "2",
    primaryLabel: "Concurrent activities",
    timeLabel: "Two ongoing activities",
    companions: [
      {
        id: "concurrent-delivery",
        kind: "delivery",
        title: "Food delivery",
        status: "Out for delivery",
        value: "26 min",
      },
      {
        id: "concurrent-focus",
        kind: "focus",
        title: "Deep work",
        status: "Focus session",
        value: "18:24",
      },
    ],
    actions: [{ id: "open-activities", label: "View activities", role: "primary" }],
  },
};

function withFeedback(activity: LiveActivityReference, feedback: string): LiveActivityReference {
  return { ...activity, feedback };
}

/** Keeps the lab's controls visibly interactive without pretending to call a vendor API. */
export function applyLiveActivityDemoAction(
  activity: LiveActivityReference,
  actionId: string,
): LiveActivityReference {
  switch (actionId) {
    case "pause-recording":
    case "pause-camera":
    case "pause-focus":
      return {
        ...activity,
        state: "paused",
        status: "Paused",
        compactValue: "Paused",
        feedback: "Demo paused — no external app was contacted.",
        actions: [
          { id: actionId.replace("pause", "resume"), label: "Resume", role: "primary" },
          ...(activity.actions?.filter((action) => action.id.startsWith("stop") || action.id.startsWith("finish")) ?? []),
        ],
      };
    case "resume-recording":
    case "resume-camera":
    case "resume-focus":
      return {
        ...activity,
        state: "live",
        status: activity.kind === "focus" ? "Focus session" : "Recording",
        compactValue: activity.primaryValue,
        feedback: "Demo resumed — no external app was contacted.",
        actions: [
          { id: actionId.replace("resume", "pause"), label: "Pause", role: "secondary" },
          {
            id: activity.kind === "focus" ? "finish-focus" : `stop-${activity.kind === "camera" ? "camera" : "recording"}`,
            label: activity.kind === "focus" ? "Finish" : "Stop",
            role: activity.kind === "focus" ? "primary" : "destructive",
          },
        ],
      };
    case "stop-recording":
    case "stop-camera":
      return {
        ...activity,
        state: "complete",
        status: "Recording saved",
        compactValue: "Saved",
        feedback: "Recording receipt shown for this local demo.",
        actions: [{ id: "open-reference", label: "View recording", role: "primary" }],
      };
    case "finish-focus":
      return {
        ...activity,
        state: "complete",
        status: "Focus complete",
        compactValue: "Done",
        progressLabel: "Focus session finished",
        feedback: "Completion receipt shown for this local demo.",
        actions: [{ id: "open-reference", label: "View receipt", role: "primary" }],
      };
    case "end-navigation":
    case "end-workout":
    case "stop-charging":
    case "stop-automation":
      return {
        ...activity,
        state: "ended",
        status: "Stopped",
        compactValue: "Stopped",
        progressLabel: activity.progressLabel
          ? `Stopped · ${activity.progressLabel}`
          : "Activity stopped",
        feedback: "Stopped-state receipt shown for this local demo.",
        actions: [{ id: "open-reference", label: "View receipt", role: "primary" }],
      };
    case "next-set":
      return {
        ...activity,
        status: "Set 4 of 4",
        compactValue: "Set 4/4",
        primaryValue: "1:30",
        primaryLabel: "Rest remaining",
        progress: 0.88,
        progressLabel: "Final set: 8 reps at 70 kg",
        feedback: "Advanced to the final set in the demo.",
      };
    default:
      return withFeedback(activity, "Reference action acknowledged — no external app was contacted.");
  }
}

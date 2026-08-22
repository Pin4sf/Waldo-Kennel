export type HomeDestination =
  | "today"
  | "open_loops"
  | "memory"
  | "daily_close"
  | "history";

export type HomeMode = HomeDestination | "catch_up" | "ready_to_close";

export type HomeAvailability =
  | "ready"
  | "partial"
  | "capture_off"
  | "stale"
  | "offline";

export type HomeCaptureState =
  | "off"
  | "active"
  | "partial"
  | "paused"
  | "blocked";

export type HomeAttentionItem = {
  id: string;
  label: string;
  statement: string;
  proposedMeaning: string;
  sourceSummary: string;
  sourceGap: string;
};

export type HomeOpenLoopFixture = {
  id: string;
  label: string;
  meaning: string;
  owner: string;
  recheck: string;
  sourceStrength: string;
};

export type HomeContinuityEvent = {
  id: string;
  time: string;
  title: string;
  detail: string;
};

export type HomeBriefItem = {
  id: string;
  label: string;
};

export type HomeFixtureState = {
  kind: "preview_fixture";
  sourceLabel: "Architecture preview";
  mode: HomeMode;
  availability: HomeAvailability;
  localDateLabel: string;
  captureState: HomeCaptureState;
  reentry: {
    label: string;
    detail: string;
  };
  brief: string[];
  todos: HomeBriefItem[];
  suggestions: HomeBriefItem[];
  attention: HomeAttentionItem[];
  waiting: number;
  readyToClose: number;
  openLoops: HomeOpenLoopFixture[];
  continuity: HomeContinuityEvent[];
};

const deckAttention: HomeAttentionItem = {
  id: "deck-follow-up",
  label: "Deck follow-up",
  statement: "I'll send Ashish the revised deck tomorrow.",
  proposedMeaning: "Prepare the revised deck for Ashish; do not send it.",
  sourceSummary: "Meeting note · user-stated commitment · 3:08 PM",
  sourceGap: "Meeting audio unavailable from 3:10–3:24 PM",
};

const deckOpenLoop: HomeOpenLoopFixture = {
  id: "deck-follow-up",
  label: "Deck follow-up",
  meaning: "Prepare the revised deck for Ashish; do not send it yet.",
  owner: "You",
  recheck: "Recheck tomorrow morning",
  sourceStrength: "User-confirmed",
};

export function homeFixture(
  destination: HomeDestination,
  availability: HomeAvailability = "ready",
): HomeFixtureState {
  return {
    kind: "preview_fixture",
    sourceLabel: "Architecture preview",
    mode: destination,
    availability,
    localDateLabel: "Saturday, 22 August",
    captureState:
      availability === "capture_off"
        ? "off"
        : availability === "offline"
          ? "blocked"
          : availability === "partial"
            ? "partial"
            : "paused",
    reentry: {
      label: "Resume the deck follow-up",
      detail: "The last decision was to prepare the revision without sending it.",
    },
    brief: [
      "One proposed commitment needs your judgment before it becomes a responsibility.",
      "The pitch-deck work remains separate until you explicitly continue it in Work.",
      "Two confirmed items are waiting; neither needs action right now.",
    ],
    todos: [
      { id: "prepare-deck", label: "Prepare the revised deck" },
      { id: "review-meeting-note", label: "Review the meeting decision" },
      { id: "confirm-follow-up", label: "Confirm tomorrow's follow-up" },
    ],
    suggestions: [
      { id: "draft-reply", label: "Draft a reply to Ashish" },
      { id: "collect-evidence", label: "Collect the latest deck evidence" },
      { id: "prepare-work-link", label: "Prepare a Work handoff" },
    ],
    attention: [deckAttention],
    waiting: 2,
    readyToClose: 1,
    openLoops: [deckOpenLoop],
    continuity: [
      {
        id: "correction",
        time: "3:31 PM",
        title: "User correction recorded",
        detail: "Prepare the deck; do not send it yet.",
      },
      {
        id: "work-link",
        time: "3:34 PM",
        title: "Work link proposed",
        detail: "Pitch-deck Project · no Outcome created in this preview.",
      },
      {
        id: "reentry",
        time: "6:12 PM",
        title: "Tomorrow re-entry selected",
        detail: "Resume from the corrected deck follow-up.",
      },
    ],
  };
}

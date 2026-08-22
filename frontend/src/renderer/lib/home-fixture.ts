import {
  defaultHomeContextFlow,
  homePhasePresentation,
  type HomeContextFlow,
  type HomeDayPhase,
  type HomePhasePresentation,
} from "./home-day-phase";

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

export type HomeOpenLoopState = "attention" | "waiting" | "ready_to_close";

export type HomeOpenLoopFixture = {
  id: string;
  label: string;
  meaning: string;
  owner: string;
  state: HomeOpenLoopState;
  trigger: string;
  recheck: string;
  sourceStrength: string;
  sourceSummary: string;
  sourceGap?: string;
  lastConfirmedAt: string;
};

export type HomeContinuityEventKind =
  | "observation"
  | "correction"
  | "work_link_preview"
  | "close_receipt_preview"
  | "reentry";

export type HomeContinuityEvent = {
  id: string;
  kind: HomeContinuityEventKind;
  time: string;
  title: string;
  detail: string;
  sourceSummary: string;
  boundary: string;
};

export type HomeBriefItem = {
  id: string;
  label: string;
};

export type HomeNextThingFixture = {
  title: string;
  startsAt: string;
  framing: string;
  promises: HomeBriefItem[];
  openQuestions: HomeBriefItem[];
  sourceSummary: string;
  sourceGap: string;
};

export type HomePlanChangeFixture = {
  previousAssumption: string;
  newFact: string;
  proposal: string;
  sourceSummary: string;
};

export type HomeClosureFact = {
  id: string;
  label: string;
  evidence: string;
};

export type HomeClosureItem = {
  id: string;
  label: string;
  meaning: string;
};

export type HomeClosureReview = {
  becameTrue: HomeClosureFact[];
  unresolved: HomeClosureItem[];
  sourceGaps: HomeBriefItem[];
  reentryOptions: HomeBriefItem[];
};

export type HomeMemoryCandidate = {
  id: string;
  statement: string;
  sourceSummary: string;
  sourceGap?: string;
  validUntil: string;
  sensitivity: "ordinary" | "sensitive";
  uncertainty: string;
};

export type HomeFixtureOptions = {
  dayPhase?: HomeDayPhase;
  contextFlow?: HomeContextFlow;
};

export type HomeFixtureState = {
  kind: "preview_fixture";
  sourceLabel: "Architecture preview";
  mode: HomeMode;
  availability: HomeAvailability;
  localDateLabel: string;
  dayPhase: HomeDayPhase;
  contextFlow: HomeContextFlow;
  presentation: HomePhasePresentation;
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
  nextThing: HomeNextThingFixture;
  planChange: HomePlanChangeFixture;
  closureReview: HomeClosureReview;
  memoryCandidates: HomeMemoryCandidate[];
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
  state: "attention",
  trigger: "Return when the revised deck is ready for review",
  recheck: "Recheck tomorrow morning",
  sourceStrength: "User-confirmed",
  sourceSummary: "Meeting note · corrected by you · 3:31 PM",
  sourceGap: "Meeting audio unavailable from 3:10–3:24 PM",
  lastConfirmedAt: "Confirmed today at 3:31 PM",
};

const vendorOpenLoop: HomeOpenLoopFixture = {
  id: "vendor-response",
  label: "Vendor response",
  meaning: "Wait for the revised security terms before comparing the shortlist.",
  owner: "Aditi",
  state: "waiting",
  trigger: "Return when the vendor sends the revised terms",
  recheck: "Recheck Monday afternoon",
  sourceStrength: "Confirmed in email",
  sourceSummary: "Email thread · Aditi and vendor · Friday at 5:42 PM",
  lastConfirmedAt: "Confirmed Friday at 5:42 PM",
};

const workshopOpenLoop: HomeOpenLoopFixture = {
  id: "pricing-workshop-note",
  label: "Pricing workshop note",
  meaning: "The packaging decision is ready to be acknowledged and released from Home.",
  owner: "You",
  state: "ready_to_close",
  trigger: "Close after you confirm the decision note reflects the workshop",
  recheck: "Ready to review now",
  sourceStrength: "User-confirmed",
  sourceSummary: "Decision note · confirmed by you · 5:18 PM",
  lastConfirmedAt: "Confirmed today at 5:18 PM",
};

const openLoops: HomeOpenLoopFixture[] = [
  deckOpenLoop,
  vendorOpenLoop,
  workshopOpenLoop,
];

const phaseBriefs: Record<
  HomeDayPhase,
  { brief: string[]; todos: HomeBriefItem[]; suggestions: HomeBriefItem[] }
> = {
  morning: {
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
  },
  afternoon: {
    brief: [
      "The deck instruction is now clear: prepare the revision, but do not send it.",
      "The pricing workshop moved to 4:30 PM, leaving one useful focus block before it.",
      "Two waiting items remain stable; neither earns an interruption this afternoon.",
    ],
    todos: [
      { id: "prepare-deck", label: "Prepare the revised deck" },
      { id: "pricing-workshop", label: "Enter the pricing workshop with a decision" },
      { id: "protect-focus", label: "Protect the next focus block" },
    ],
    suggestions: [
      { id: "prep-workshop", label: "Prepare the unresolved pricing question" },
      { id: "collect-evidence", label: "Collect the latest deck evidence" },
      { id: "defer-admin", label: "Defer the non-urgent admin follow-up" },
    ],
  },
  evening: {
    brief: [
      "The meeting decision was corrected and the revised-deck responsibility remains open.",
      "No Work Outcome or external send was created from today's proposal.",
      "One known capture gap should remain visible if you choose to close the day.",
    ],
    todos: [
      { id: "review-deck-state", label: "Review the deck responsibility" },
      { id: "choose-disposition", label: "Choose what carries forward" },
      { id: "set-reentry", label: "Set tomorrow's re-entry" },
    ],
    suggestions: [
      { id: "resume-deck", label: "Resume from the corrected deck follow-up" },
      { id: "keep-gap", label: "Carry the source gap into tomorrow" },
      { id: "start-closure", label: "Start an explicit Closure review" },
    ],
  },
};

export function homeFixture(
  destination: HomeDestination,
  availability: HomeAvailability = "ready",
  options: HomeFixtureOptions = {},
): HomeFixtureState {
  const dayPhase = options.dayPhase ?? "morning";
  const contextFlow =
    options.contextFlow ?? defaultHomeContextFlow(dayPhase);
  const phase = phaseBriefs[dayPhase];

  return {
    kind: "preview_fixture",
    sourceLabel: "Architecture preview",
    mode: destination,
    availability,
    localDateLabel: "Saturday, 22 August",
    dayPhase,
    contextFlow,
    presentation: homePhasePresentation(dayPhase),
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
    brief: phase.brief,
    todos: phase.todos,
    suggestions: phase.suggestions,
    attention: contextFlow === "quiet_focus" ? [] : [deckAttention],
    waiting: openLoops.filter((loop) => loop.state === "waiting").length,
    readyToClose: openLoops.filter((loop) => loop.state === "ready_to_close").length,
    openLoops,
    continuity: [
      {
        id: "correction",
        kind: "correction",
        time: "3:31 PM",
        title: "User correction recorded",
        detail: "Prepare the deck; do not send it yet.",
        sourceSummary: "Meeting note · corrected by you",
        boundary: "This correction is fixture evidence; no canonical event was written.",
      },
      {
        id: "work-link",
        kind: "work_link_preview",
        time: "3:34 PM",
        title: "Work link proposed",
        detail: "Pitch-deck Project · no Outcome created in this preview.",
        sourceSummary: "Waldo proposal · architecture preview",
        boundary: "No Work Outcome or responsibility link exists.",
      },
      {
        id: "reentry",
        kind: "reentry",
        time: "6:12 PM",
        title: "Tomorrow re-entry selected",
        detail: "Resume from the corrected deck follow-up.",
        sourceSummary: "Closure preview · explicit local choice",
        boundary: "The re-entry is not saved outside this preview.",
      },
    ],
    nextThing: {
      title: "Pricing workshop",
      startsAt: "4:30 PM · in 52 minutes",
      framing: "The team needs one clear decision on packaging before the proposal can move.",
      promises: [
        { id: "pricing-decision", label: "Bring a clear recommendation on the packaging split" },
        { id: "pricing-boundary", label: "Keep implementation scope out of this decision" },
      ],
      openQuestions: [
        { id: "pricing-question", label: "Which customer evidence changes the recommendation?" },
      ],
      sourceSummary: "Calendar · workshop brief · decision note",
      sourceGap: "The latest customer call transcript is unavailable.",
    },
    planChange: {
      previousAssumption: "The pricing workshop would start at 3:30 PM.",
      newFact: "The organizer moved it to 4:30 PM.",
      proposal: "Use the recovered hour for the deck revision; keep the workshop decision bounded.",
      sourceSummary: "Calendar update · received 2:41 PM",
    },
    closureReview: {
      becameTrue: [
        {
          id: "deck-correction",
          label: "The deck instruction was corrected",
          evidence: "Prepare the revision; do not send it yet.",
        },
      ],
      unresolved: [
        {
          id: "deck-follow-up",
          label: "Deck follow-up",
          meaning: "The revision still needs preparation and review.",
        },
      ],
      sourceGaps: [
        {
          id: "meeting-audio-gap",
          label: "Meeting audio was unavailable from 3:10–3:24 PM.",
        },
      ],
      reentryOptions: [
        {
          id: "resume-deck-follow-up",
          label: "Resume from the corrected deck follow-up",
        },
        {
          id: "resume-pricing-note",
          label: "Resume from the pricing workshop decision note",
        },
      ],
    },
    memoryCandidates: [
      {
        id: "deck-send-boundary",
        statement: "Ashish should receive the deck only after the revision is reviewed.",
        sourceSummary: "Meeting note · user correction · today at 3:31 PM",
        sourceGap: "Source contains a 14 minute audio gap",
        validUntil: "Valid until the deck is sent or this instruction is corrected",
        sensitivity: "ordinary",
        uncertainty: "Proposed from one corrected statement; no durable memory exists.",
      },
    ],
  };
}

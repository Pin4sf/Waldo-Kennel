export type HomeDayPhase = "morning" | "afternoon" | "evening";

export type HomeContextFlow =
  | "catch_up"
  | "before_next"
  | "plans_changed"
  | "evening_review"
  | "quiet_focus";

export type HomePhasePresentation = {
  greeting: string;
  briefLabel: string;
  attentionSummary: string;
  primaryListLabel: string;
  suggestionLabel: string;
  finishLabel: string;
  capturePlaceholder: string;
};

const presentations: Record<HomeDayPhase, HomePhasePresentation> = {
  morning: {
    greeting: "Good morning, Shivansh.",
    briefLabel: "Morning brief",
    attentionSummary: "One thing needs you today.",
    primaryListLabel: "To do",
    suggestionLabel: "Waldo suggests",
    finishLabel: "Finish morning brief",
    capturePlaceholder: "Capture a thought…",
  },
  afternoon: {
    greeting: "Good afternoon, Shivansh.",
    briefLabel: "Afternoon brief",
    attentionSummary: "One decision can steady the rest of the day.",
    primaryListLabel: "Still matters",
    suggestionLabel: "Waldo suggests next",
    finishLabel: "Finish afternoon review",
    capturePlaceholder: "Note what changed…",
  },
  evening: {
    greeting: "Good evening, Shivansh.",
    briefLabel: "Evening brief",
    attentionSummary: "One open responsibility is worth carrying deliberately.",
    primaryListLabel: "Before you close",
    suggestionLabel: "For tomorrow",
    finishLabel: "Review the evening brief",
    capturePlaceholder: "Leave context for tomorrow…",
  },
};

const defaultContexts: Record<HomeDayPhase, HomeContextFlow> = {
  morning: "catch_up",
  afternoon: "before_next",
  evening: "evening_review",
};

export function resolveHomeDayPhase(now: Date): HomeDayPhase {
  const hour = now.getHours();
  if (hour >= 5 && hour < 12) return "morning";
  if (hour >= 12 && hour < 17) return "afternoon";
  return "evening";
}

export function homePhasePresentation(
  dayPhase: HomeDayPhase,
): HomePhasePresentation {
  return presentations[dayPhase];
}

export function defaultHomeContextFlow(
  dayPhase: HomeDayPhase,
): HomeContextFlow {
  return defaultContexts[dayPhase];
}

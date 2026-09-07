export type SessionStatus =
  | "working"
  | "pr_open"
  | "draft"
  | "ci_failed"
  | "review_pending"
  | "changes_requested"
  | "approved"
  | "mergeable"
  | "merged"
  | "needs_input"
  | "exited"
  | "no_signal"
  | "idle"
  | "terminated"
  | "unknown";

export type ActivityState =
  | "active"
  | "idle"
  | "waiting_input"
  | "blocked"
  | "exited"
  | "unknown";

import type { KennelConnectionState } from "./kennel-contract";
import type { IslandProvider } from "./providers";

export type IslandTone = "working" | "action" | "review" | "ready" | "muted" | "error";

/** Picks a glyph. There are exactly the two the design ships. */
export type IslandAgent = "waldo" | "codex";

/**
 * Which AI is behind a session. Picks a colour, and has as many values as there
 * are harnesses — which is why it is not folded into `IslandAgent`.
 */
export type { IslandProvider };

export interface IslandTask {
  id: string;
  sessionId?: string;
  projectId?: string;
  title: string;
  project: string;
  branch: string;
  target: string;
  updatedLabel: string;
  actionLabel: "Choose" | "Approve" | "Steer" | "Open";
  agent: IslandAgent;
  provider?: IslandProvider;
  status: SessionStatus;
  activity: ActivityState;
  tone: IslandTone;
  dimmed?: boolean;
  disabled?: boolean;
  error?: string;
}

/**
 * What the island is reporting while it rests, in descending urgency.
 *
 * - `blocked`  a session stopped at a gate: a permission request, a question,
 *              or a choice. Kennel is holding a decision it cannot make.
 * - `paused`   a session finished its turn and is waiting on you. Nothing is
 *              gated; it simply has nothing more to do until you speak.
 * - `running`  a session is working and needs nothing.
 *
 * Media is deliberately absent, and not only from this union: it is host state
 * rather than Kennel state, so it reaches the island as a prop beside the model
 * instead of being folded into a snapshot that is otherwise entirely
 * server-derived. It never competes for the status chip.
 */
export type IslandPresence = "blocked" | "paused" | "running";

/**
 * One resting card. When several presences are live at once the island shows
 * the most urgent first and rotates through the rest, so a queue of four
 * approvals never hides the two sessions still working behind it.
 */
export interface IslandPresenceCard {
  presence: IslandPresence;
  /** Sessions in this presence, which is the number on the badge. */
  count: number;
  /** Ticker content revealed when the pointer rests on this card. */
  title: string;
  project: string;
  branch: string;
  agent: IslandAgent;
  provider?: IslandProvider;
  detail: string;
}

export interface CompactIslandModel {
  surface: "compact";
  taskId: string;
  title: string;
  project: string;
  branch: string;
  agent: IslandAgent;
  provider?: IslandProvider;
  tone: IslandTone;
  phase: "working" | "complete" | "needs_input" | "idle" | "offline" | "error";
  /**
   * Resting cards, most urgent first. Empty means nothing is happening: with
   * no media either, the island shrinks to the notch and disappears.
   */
  presence: IslandPresenceCard[];
  attentionCount?: number;
  connection?: KennelConnectionState;
  detail?: string;
  additions?: number;
  deletions?: number;
  /**
   * A prototype-only reference for an iPhone-style ongoing activity. These
   * fixtures exercise the Island's information hierarchy; they do not imply
   * an integration with the named ecosystem example.
   */
  liveActivity?: LiveActivityReference;
}

export type LiveActivityKind =
  | "voice-recording"
  | "delivery"
  | "ride"
  | "transit"
  | "flight"
  | "sports"
  | "focus"
  | "workout"
  | "charging"
  | "camera"
  | "weather"
  | "home"
  | "multiple";

export type LiveActivityState = "live" | "paused" | "attention" | "complete" | "ended";

export interface LiveActivityStep {
  id: string;
  label: string;
  state: "complete" | "current" | "upcoming" | "attention";
}

export interface LiveActivityMetric {
  id: string;
  label: string;
  value: string;
  detail?: string;
}

export interface LiveActivityEvent {
  id: string;
  label: string;
  detail?: string;
  timeLabel?: string;
}

export interface LiveActivityCompanion {
  id: string;
  kind: Exclude<LiveActivityKind, "multiple">;
  title: string;
  status: string;
  value: string;
}

export interface LiveActivityAction {
  id: string;
  label: string;
  role?: "primary" | "secondary" | "destructive";
}

/**
 * Neutral presentation data shared by every reference activity in the lab.
 * It deliberately describes what is visible, rather than any vendor API.
 */
export interface LiveActivityReference {
  id: string;
  kind: LiveActivityKind;
  mechanism: "system" | "activitykit" | "beta";
  source: string;
  pattern: string;
  title: string;
  context: string;
  status: string;
  state: LiveActivityState;
  compactValue: string;
  primaryValue: string;
  primaryLabel: string;
  timeLabel?: string;
  progress?: number;
  progressLabel?: string;
  steps?: LiveActivityStep[];
  metrics?: LiveActivityMetric[];
  events?: LiveActivityEvent[];
  companions?: LiveActivityCompanion[];
  alert?: {
    title: string;
    detail: string;
  };
  actions?: LiveActivityAction[];
  feedback?: string;
}

export interface LiveActivityIslandModel {
  surface: "activity";
  activity: LiveActivityReference;
}

export interface QueueIslandModel {
  surface: "queue";
  activeTab: "home" | "work";
  pendingCount: number;
  tasks: IslandTask[];
  connection?: KennelConnectionState;
  statusMessage?: string;
  statusDetail?: string;
  error?: string;
  refreshing?: boolean;
}

export interface ChoiceOption {
  id: string;
  label: string;
  recommended?: boolean;
  freeform?: boolean;
  content?: Record<string, unknown>;
  inputAction?: "accept" | "decline" | "cancel";
}

export interface ChoiceIslandModel {
  surface: "choice";
  sessionId?: string;
  promptId: string;
  question: string;
  questionIndex: number;
  questionCount: number;
  options: ChoiceOption[];
  project?: string;
  branch?: string;
  sessionTitle?: string;
  submittingOptionId?: string;
  error?: string;
}

export interface PermissionDecision {
  id: string;
  label: string;
  shortcut?: string;
}

export interface PermissionIslandModel {
  surface: "permission";
  sessionId?: string;
  requestId: string;
  question: string;
  questionIndex: number;
  questionCount: number;
  project: string;
  branch: string;
  contextFiles: string[];
  reason?: string;
  command?: string;
  cwd?: string;
  sessionTitle?: string;
  decisions?: PermissionDecision[];
  contextTruncated?: boolean;
  submittingDecisionId?: string;
  canInterrupt?: boolean;
  error?: string;
}

export interface SteerIslandModel {
  surface: "steer";
  sessionId: string;
  title: string;
  project: string;
  branch: string;
  submitting?: boolean;
  error?: string;
}

export interface UsageLimit {
  id: string;
  label: string;
  percent: number;
  resetLabel: string;
}

export interface UsageIslandModel {
  surface: "usage";
  plan: string;
  account: string;
  sessionsUsing: number;
  limits: UsageLimit[];
  unavailableMessage?: string;
}

export type IslandModel =
  | CompactIslandModel
  | LiveActivityIslandModel
  | QueueIslandModel
  | ChoiceIslandModel
  | PermissionIslandModel
  | SteerIslandModel
  | UsageIslandModel;

export type IslandAction =
  | { type: "expand" }
  | { type: "collapse" }
  | { type: "dismiss" }
  | { type: "set-tab"; tab: "home" | "work" }
  | { type: "task-action"; taskId: string; label: IslandTask["actionLabel"] }
  | { type: "select-choice"; promptId: string; optionId: string }
  | { type: "navigate-prompt"; direction: "previous" | "next" }
  | { type: "resolve-permission"; requestId: string; decisionId: string }
  | { type: "interrupt-session"; requestId: string }
  | { type: "submit-steer"; sessionId: string; text: string }
  | { type: "open-session"; sessionId?: string; projectId?: string }
  | { type: "retry-connection" }
  | { type: "activity-action"; activityId: string; actionId: string }
  | { type: "open-usage" }
  | { type: "open-settings" }
  | { type: "hide-island" };

export interface KennelIslandAdapter {
  getSnapshot: () => IslandModel;
  subscribe: (listener: () => void) => () => void;
  dispatch: (action: IslandAction) => void | Promise<void>;
  dispose?: () => void;
}

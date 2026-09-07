import type {
  ActivityState,
  IslandTask,
  IslandTone,
  QueueIslandModel,
  SessionStatus,
} from "./types";

export type KennelPendingAction =
  | { kind: "approval"; requestId: string; label?: string }
  | { kind: "structured_input"; requestId: string; label?: string }
  | { kind: "open_session"; label?: string };

export interface KennelSessionProjection {
  id: string;
  projectName: string;
  title: string;
  branch?: string;
  kind?: "orchestrator" | "worker";
  harness?: string;
  status: SessionStatus;
  activity: ActivityState;
  updatedLabel: string;
  pending?: KennelPendingAction;
}

export interface KennelQueueProjection {
  sessions: KennelSessionProjection[];
  pendingCount: number;
}

const actionStatuses = new Set<SessionStatus>([
  "needs_input",
  "ci_failed",
  "changes_requested",
  "exited",
  "no_signal",
  "unknown",
]);
const reviewStatuses = new Set<SessionStatus>(["review_pending", "pr_open", "draft"]);
const readyStatuses = new Set<SessionStatus>(["approved", "mergeable", "merged"]);

function toneForStatus(status: SessionStatus): IslandTone {
  if (actionStatuses.has(status)) return "action";
  if (reviewStatuses.has(status)) return "review";
  if (readyStatuses.has(status)) return "ready";
  if (status === "terminated") return "muted";
  return "working";
}

function actionForSession(session: KennelSessionProjection): IslandTask["actionLabel"] {
  if (session.pending?.kind === "structured_input") return "Choose";
  if (session.pending?.kind === "approval") return "Approve";
  if (session.activity === "active") return "Steer";
  return "Open";
}

export function projectKennelQueue({
  sessions,
  pendingCount,
}: KennelQueueProjection): QueueIslandModel {
  return {
    surface: "queue",
    activeTab: "work",
    pendingCount,
    tasks: sessions.slice(0, 5).map((session) => ({
      id: session.id,
      title: session.title,
      project: session.projectName,
      branch: session.branch ?? "main",
      target: session.pending?.label ?? "Open session",
      updatedLabel: session.updatedLabel,
      actionLabel: actionForSession(session),
      agent:
        session.kind === "orchestrator" || session.harness?.toLowerCase().includes("waldo")
          ? "waldo"
          : "codex",
      status: session.status,
      activity: session.activity,
      tone: toneForStatus(session.status),
      dimmed: session.status === "terminated",
    })),
  };
}

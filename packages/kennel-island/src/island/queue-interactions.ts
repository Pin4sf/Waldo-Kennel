import type { IslandAction, IslandTask } from "./types";

type OpenSessionAction = Extract<IslandAction, { type: "open-session" }>;
type SessionTarget = Pick<IslandTask, "projectId" | "sessionId">;

/** The queue row owns navigation; its underlined target is display text. */
export function openSessionActionForQueueRow(task: SessionTarget): OpenSessionAction {
  return {
    type: "open-session",
    ...(task.projectId === undefined ? {} : { projectId: task.projectId }),
    ...(task.sessionId === undefined ? {} : { sessionId: task.sessionId }),
  };
}

/** A double-click on a nested action is still one action, never navigation. */
export function shouldDispatchQueueTaskAction(clickCount: number): boolean {
  return clickCount < 2;
}

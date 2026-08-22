export { createMemoryIslandAdapter, useKennelIsland } from "./adapter";
export { KennelIsland } from "./KennelIsland";
export { defaultStageGeometry, islandRadius, islandWidths } from "./stage-layout";
export {
  POINTER_LEAVE_GRACE_MS,
  shouldCollapseOnPointerLeave,
} from "./stage-rules";
export { useStageGeometry, useStageInteractivity } from "./useIslandStage";
export { projectKennelQueue } from "./kennel-projection";
export type {
  KennelPendingAction,
  KennelQueueProjection,
  KennelSessionProjection,
} from "./kennel-projection";
export type {
  ActivityState,
  IslandAction,
  IslandModel,
  IslandTask,
  KennelIslandAdapter,
  SessionStatus,
} from "./types";

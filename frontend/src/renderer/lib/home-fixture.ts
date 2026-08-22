export type HomeDestination =
  | "today"
  | "open_loops"
  | "memory"
  | "daily_close"
  | "history";

export type HomeMode = HomeDestination | "catch_up" | "ready_to_close";

export type HomeFixtureState = {
  kind: "preview_fixture";
  sourceLabel: "Architecture preview";
  mode: HomeMode;
  availability: "ready" | "partial" | "capture_off" | "offline";
};

export function homeFixture(
  destination: HomeDestination,
  availability: HomeFixtureState["availability"] = "ready",
): HomeFixtureState {
  return {
    kind: "preview_fixture",
    sourceLabel: "Architecture preview",
    mode: destination,
    availability,
  };
}

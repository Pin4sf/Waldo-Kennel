export type HomeMode = "today" | "catch_up" | "open_loop" | "ready_to_close";

export type HomeFixtureState = {
	kind: "preview_fixture";
	sourceLabel: "Architecture preview";
	mode: HomeMode;
};

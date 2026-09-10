import type { components } from "../../api/schema";

export const MISSION_LANES = [
	"define",
	"authorize",
	"observe",
	"needsYou",
	"review",
	"accepted",
	"unavailable",
] as const;
export type MissionLane = (typeof MISSION_LANES)[number];
export type MissionAttention = { lane: MissionLane; reason?: string };

export function runStateAttention(
	state?: components["schemas"]["ControllersOutcomeRunStateResponse"],
	failure?: string,
): MissionAttention {
	if (!state || failure) return { lane: "unavailable", reason: failure };
	const lanes = {
		define: "define",
		ready_to_authorize: "authorize",
		in_progress: "observe",
		needs_you: "needsYou",
		ready_for_review: "review",
		accepted: "accepted",
	} as const;
	return { lane: lanes[state.state], reason: state.blocker?.message ?? state.attentionReason };
}

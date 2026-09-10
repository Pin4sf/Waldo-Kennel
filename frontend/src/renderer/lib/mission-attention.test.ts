import { expect, it } from "vitest";
import { runStateAttention } from "./mission-attention";
import type { OutcomeRunState } from "../hooks/useOutcomeRunState";
it.each([
	["define", "define"],
	["ready_to_authorize", "authorize"],
	["in_progress", "observe"],
	["needs_you", "needsYou"],
	["ready_for_review", "review"],
	["accepted", "accepted"],
] as const)("renders canonical %s as %s", (state, lane) => {
	expect(runStateAttention({ state } as OutcomeRunState).lane).toBe(lane);
});
it("does not present a failed read as lifecycle Needs you or accepted", () => {
	expect(runStateAttention({ state: "accepted" } as OutcomeRunState, "Unavailable")).toEqual({
		lane: "unavailable",
		reason: "Unavailable",
	});
	expect(runStateAttention().lane).toBe("unavailable");
});
it("shows the daemon's blocker message", () => {
	expect(
		runStateAttention({
			state: "needs_you",
			blocker: { code: "blocked", message: "Retained source missing" },
		} as OutcomeRunState).reason,
	).toBe("Retained source missing");
});

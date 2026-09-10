import { expect, it } from "vitest";
import type { components } from "../../api/schema";
import { missionAttention } from "./mission-attention";
const outcome = {
	id: "out",
	currentRevisionNumber: 2,
	latestPlan: { id: "plan", status: "approved", contractRevisionNumber: 2 },
} as components["schemas"]["OutcomeResponse"];
const proof = (status: string, revision = 2) =>
	({
		outcomeId: "out",
		contractRevision: { number: revision },
		status,
		nextAction: "Owner review",
	}) as components["schemas"]["OutcomeProofResponse"];
const schedule = (reason?: components["schemas"]["ScheduleResponse"]["noRunnableReason"]) =>
	({
		outcomeId: "out",
		plan: { id: "plan" },
		workUnits: [],
		noRunnableReason: reason,
	}) as unknown as components["schemas"]["ScheduleResponse"];
it("does not infer execution or acceptance from approval", () => {
	expect(missionAttention(outcome).lane).toBe("ready");
	expect(missionAttention(outcome, undefined, schedule("attempt_executing")).lane).toBe("observe");
	expect(missionAttention(outcome, undefined, schedule("all_units_proven")).lane).toBe("needsYou");
});
it("accepts only the current revision's canonical proof status for the Accepted lane", () => {
	expect(missionAttention(outcome, proof("accepted")).lane).toBe("accepted");
	expect(missionAttention(outcome, proof("accepted", 1)).lane).toBe("ready");
	expect(missionAttention(outcome, { ...proof("accepted"), outcomeId: "other" }).lane).toBe("ready");
});
it("keeps actionable proof and query failure reasons", () => {
	expect(missionAttention(outcome, proof("rework_required"))).toEqual({ lane: "needsYou", reason: "Owner review" });
	expect(missionAttention(outcome, proof("ready_for_acceptance")).lane).toBe("review");
	expect(missionAttention(outcome, undefined, schedule("awaiting_proof"))).toEqual({
		lane: "needsYou",
		reason: "awaiting_proof",
	});
	expect(missionAttention(outcome, proof("accepted"), undefined, "Daemon unavailable")).toEqual({
		lane: "needsYou",
		reason: "Daemon unavailable",
	});
});
it("does not project a different Plan's schedule onto the current Outcome", () => {
	expect(
		missionAttention(outcome, undefined, { ...schedule("attempt_executing"), plan: { ...schedule().plan, id: "old" } })
			.lane,
	).toBe("ready");
});

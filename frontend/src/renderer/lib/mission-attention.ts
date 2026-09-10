import type { components } from "../../api/schema";

type Outcome = components["schemas"]["OutcomeResponse"];
type Proof = components["schemas"]["OutcomeProofResponse"];
type Schedule = components["schemas"]["ScheduleResponse"];
export const MISSION_LANES = ["define", "authorize", "ready", "observe", "needsYou", "review", "accepted"] as const;
export type MissionLane = (typeof MISSION_LANES)[number];
export type MissionAttention = { lane: MissionLane; reason?: string };

/** Read models supply every fact; lane labels do not grant execution authority. */
export function missionAttention(
	outcome: Outcome,
	proof?: Proof,
	schedule?: Schedule,
	failure?: string,
): MissionAttention {
	if (failure) return { lane: "needsYou", reason: failure };
	const currentProof =
		proof?.outcomeId === outcome.id && proof.contractRevision.number === outcome.currentRevisionNumber
			? proof
			: undefined;
	if (currentProof?.status === "accepted") return { lane: "accepted", reason: currentProof.nextAction };
	if (currentProof?.status === "ready_for_acceptance") return { lane: "review", reason: currentProof.nextAction };
	if (currentProof?.status === "rework_required") return { lane: "needsYou", reason: currentProof.nextAction };
	const plan = outcome.latestPlan;
	if (!plan || plan.contractRevisionNumber !== outcome.currentRevisionNumber) return { lane: "define" };
	if (plan.status === "proposed") return { lane: "authorize" };
	const currentSchedule = schedule?.plan.id === plan.id && schedule.outcomeId === outcome.id ? schedule : undefined;
	if (
		currentSchedule?.activeAttempt &&
		["paused", "lost", "failed", "cancelled"].includes(currentSchedule.activeAttempt.status)
	) {
		return { lane: "needsYou", reason: currentSchedule.activeAttempt.status };
	}
	if (currentSchedule?.noRunnableReason === "attempt_paused")
		return { lane: "needsYou", reason: currentSchedule.noRunnableReason };
	if (currentSchedule?.workUnits.some((unit) => unit.state === "retryable" || unit.state === "paused"))
		return { lane: "needsYou", reason: "paused_or_retryable" };
	if (currentSchedule?.noRunnableReason === "attempt_executing") return { lane: "observe" };
	if (currentSchedule?.noRunnableReason) return { lane: "needsYou", reason: currentSchedule.noRunnableReason };
	return { lane: "ready" };
}

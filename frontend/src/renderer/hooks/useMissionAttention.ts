import { useQueries } from "@tanstack/react-query";
import {
	fetchOutcomeProof,
	fetchOutcomeSchedule,
	outcomeProofQueryKey,
	outcomeScheduleQueryKey,
	classifyOutcomeFailure,
	type OutcomeRecord,
} from "./useOutcome";
import { missionAttention } from "../lib/mission-attention";

/** Callers bound this to the visible page, sharing the Mission query cache. */
export function useMissionAttention(outcomes: OutcomeRecord[]) {
	const proofs = useQueries({
		queries: outcomes.map((outcome) => ({
			queryKey: outcomeProofQueryKey(outcome.id),
			queryFn: () => fetchOutcomeProof(outcome.id),
			retry: false,
		})),
	});
	const schedules = useQueries({
		queries: outcomes.map((outcome) => ({
			queryKey: outcomeScheduleQueryKey(outcome.id, outcome.latestPlan?.id),
			queryFn: () => fetchOutcomeSchedule(outcome.id, outcome.latestPlan!.id),
			enabled:
				outcome.latestPlan?.status === "approved" &&
				outcome.latestPlan.contractRevisionNumber === outcome.currentRevisionNumber,
			retry: false,
		})),
	});
	return new Map(
		outcomes.map((outcome, index) => {
			const failure = proofs[index].error ?? schedules[index].error;
			return [
				outcome.id,
				missionAttention(
					outcome,
					proofs[index].data,
					schedules[index].data,
					failure ? classifyOutcomeFailure(failure).message : undefined,
				),
			];
		}),
	);
}

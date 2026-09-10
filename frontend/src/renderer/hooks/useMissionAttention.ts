import { useProjectRunStates } from "./useOutcomeRunState";
import { runStateAttention } from "../lib/mission-attention";
import type { OutcomeRecord } from "./useOutcome";

/** One canonical projection per Project; failed reads never reconstruct lifecycle. */
export function useMissionAttention(outcomes: OutcomeRecord[], projectId: string) {
	const query = useProjectRunStates(projectId);
	return new Map(
		outcomes.map((outcome) => [
			outcome.id,
			runStateAttention(
				query.data?.find((state) => state.outcomeId === outcome.id),
				query.error?.message,
			),
		]),
	);
}

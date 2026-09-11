import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { components } from "../../api/schema";
import { apiClient, apiErrorCode } from "../lib/api-client";
import { classifyOutcomeFailure, type OutcomeFailure } from "./useOutcome";
import { planQueryKey } from "./useOutcome";

export type PlanningCandidate = components["schemas"]["PlanningCandidateResponse"];
export type PlanningResponse = components["schemas"]["PlanningResponse"];
export type PlanningSession = components["schemas"]["PlanningSessionResponse"];
export type PlanningTurn = components["schemas"]["PlanningTurnResponse"];
export type PlanningContextMode = components["schemas"]["PlanningSessionResponse"]["contextMode"];

export function planningCandidatesQueryKey(outcomeId: string | undefined, contractRevision: number | undefined) {
	return ["outcome-planning-candidates", outcomeId ?? "", contractRevision ?? 0] as const;
}

export function planningSessionQueryKey(outcomeId: string | undefined, contractRevision: number | undefined, sessionId?: string) {
	return ["outcome-planning-session", outcomeId ?? "", contractRevision ?? 0, sessionId ?? "current"] as const;
}

type PlanningEnvelope = components["schemas"]["PlanningEnvelope"];
type PlanningCandidatesEnvelope = components["schemas"]["PlanningCandidatesEnvelope"];

export function usePlanningCandidates(outcomeId: string | undefined, contractRevision: number | undefined) {
	const query = useQuery({
		queryKey: planningCandidatesQueryKey(outcomeId, contractRevision),
		enabled: Boolean(outcomeId && contractRevision),
		queryFn: async () => {
			const { data, error } = await apiClient.GET("/api/v1/outcomes/{outcomeId}/planning-candidates", {
				params: { path: { outcomeId: outcomeId as string }, query: { contractRevision } },
			});
			if (error) throw error;
			return (data as PlanningCandidatesEnvelope).candidates;
		},
		retry: false,
	});
	return {
		candidates: query.data ?? [],
		isLoading: query.isLoading,
		failure: query.error ? classifyOutcomeFailure(query.error) : undefined,
		refetch: () => void query.refetch(),
	};
}

export function usePlanningSession(outcomeId: string | undefined, contractRevision: number | undefined, sessionId?: string) {
	const query = useQuery({
		queryKey: planningSessionQueryKey(outcomeId, contractRevision, sessionId),
		enabled: Boolean(outcomeId && contractRevision),
		queryFn: async () => {
			const path = sessionId
				? "/api/v1/outcomes/{outcomeId}/planning-sessions/{planningSessionId}"
				: "/api/v1/outcomes/{outcomeId}/planning-session";
			const { data, error } = sessionId
				? await apiClient.GET(path, { params: { path: { outcomeId: outcomeId as string, planningSessionId: sessionId } } })
				: await apiClient.GET(path, { params: { path: { outcomeId: outcomeId as string } } });
			if (error) {
				if (apiErrorCode(error) === "PLANNING_SESSION_NOT_FOUND" || apiErrorCode(error) === "NOT_FOUND") return null;
				throw error;
			}
			if (!data) return null;
			const planning = (data as PlanningEnvelope).planning;
			// A current session from an older Contract revision is closed to this
			// surface; the next explicit Start creates a fresh revision-scoped one.
			if (planning.session.contractRevisionNumber !== contractRevision) return null;
			return planning;
		},
		retry: false,
		refetchInterval: (query) => (query.state.data?.session.waitingOn === "provider" ? 3000 : false),
	});
	return {
		planning: query.data ?? undefined,
		isLoading: query.isLoading,
		failure: query.error ? classifyOutcomeFailure(query.error) : undefined,
		refetch: () => void query.refetch(),
	};
}

export interface PlanningMutationState<TInput> {
	pending: boolean;
	failure?: OutcomeFailure;
	mutate: (input: TInput) => Promise<PlanningResponse>;
	reset: () => void;
}

function usePlanningMutation<TInput>(outcomeId: string | undefined, contractRevision: number | undefined, mutationFn: (input: TInput) => Promise<PlanningResponse>, allowDifferentSession = false): PlanningMutationState<TInput> {
	const queryClient = useQueryClient();
	const mutation = useMutation({
		mutationFn,
		onSuccess: (planning) => {
			const key = planningSessionQueryKey(outcomeId, contractRevision);
			const current = queryClient.getQueryData<PlanningResponse>(key);
			if (current && current.session.id !== planning.session.id && !allowDifferentSession) return;
			if (current?.session.id === planning.session.id && current.session.revision > planning.session.revision) return;
			if (current?.session.id === planning.session.id && current.session.status === "cancelled" && planning.session.status !== "cancelled") return;
			queryClient.setQueryData(key, planning);
			void queryClient.invalidateQueries({ queryKey: planQueryKey(outcomeId) });
		},
		onError: async () => {
			// A timeout may mean the owner turn committed. Re-read before allowing another action.
			await queryClient.invalidateQueries({ queryKey: planningSessionQueryKey(outcomeId, contractRevision) });
		},
	});
	return {
		pending: mutation.isPending,
		failure: mutation.error ? classifyOutcomeFailure(mutation.error) : undefined,
		mutate: mutation.mutateAsync,
		reset: mutation.reset,
	};
}

export function useStartPlanning(outcomeId: string | undefined, contractRevision: number | undefined) {
	return usePlanningMutation(outcomeId, contractRevision, async (input: { candidateId: string; contextMode: PlanningContextMode; requestKey: string }) => {
		const { data, error } = await apiClient.POST("/api/v1/outcomes/{outcomeId}/planning-sessions", {
			params: { path: { outcomeId: outcomeId as string } },
			body: { expectedContractRevision: contractRevision as number, ...input },
		});
		if (error) throw error;
		return (data as PlanningEnvelope).planning;
	}, true);
}

export function useSendPlanningMessage(outcomeId: string | undefined, contractRevision: number | undefined, sessionId: string | undefined) {
	return usePlanningMutation(outcomeId, contractRevision, async (input: { expectedSessionRevision: number; text: string; requestKey: string }) => {
		const { data, error } = await apiClient.POST("/api/v1/outcomes/{outcomeId}/planning-sessions/{planningSessionId}/messages", {
			params: { path: { outcomeId: outcomeId as string, planningSessionId: sessionId as string } },
			body: input,
		});
		if (error) throw error;
		return (data as PlanningEnvelope).planning;
	});
}

export function useFinalizePlanning(outcomeId: string | undefined, contractRevision: number | undefined, sessionId: string | undefined) {
	return usePlanningMutation(outcomeId, contractRevision, async (input: { expectedSessionRevision: number; requestKey: string }) => {
		const { data, error } = await apiClient.POST("/api/v1/outcomes/{outcomeId}/planning-sessions/{planningSessionId}/proposal", {
			params: { path: { outcomeId: outcomeId as string, planningSessionId: sessionId as string } },
			body: input,
		});
		if (error) throw error;
		return (data as PlanningEnvelope).planning;
	});
}

export function useCancelPlanning(outcomeId: string | undefined, contractRevision: number | undefined, sessionId: string | undefined) {
	return usePlanningMutation(outcomeId, contractRevision, async (input: { expectedSessionRevision: number }) => {
		const { data, error } = await apiClient.POST("/api/v1/outcomes/{outcomeId}/planning-sessions/{planningSessionId}/cancel", {
			params: { path: { outcomeId: outcomeId as string, planningSessionId: sessionId as string } },
			body: input,
		});
		if (error) throw error;
		return (data as PlanningEnvelope).planning;
	});
}

export function planningRequestKey(prefix: string) {
	return `${prefix}-${typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random()}`}`;
}

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef } from "react";
import type { components } from "../../api/schema";
import { apiClient, apiErrorMessage } from "../lib/api-client";

export type OutcomeRunState = components["schemas"]["ControllersOutcomeRunStateResponse"];
type RunCommand = components["schemas"]["ControllersOutcomeRunCommandRequest"];
export const outcomeRunStateKey = (id: string) => ["outcome-run-state", id] as const;
export const projectRunStatesKey = (id: string) => ["project-run-states", id] as const;

export function useOutcomeRunState(id: string) {
	return useQuery({
		queryKey: outcomeRunStateKey(id),
		retry: false,
		queryFn: async () => {
			const { data, error } = await apiClient.GET("/api/v1/outcomes/{outcomeId}/run", {
				params: { path: { outcomeId: id } },
			});
			if (error) throw new Error(apiErrorMessage(error));
			if (!data?.runState) throw new Error("Run state unavailable");
			return data.runState;
		},
	});
}

export function useProjectRunStates(id: string) {
	return useQuery({
		queryKey: projectRunStatesKey(id),
		retry: false,
		queryFn: async () => {
			const { data, error } = await apiClient.GET("/api/v1/projects/{id}/outcome-run-states", {
				params: { path: { id }, query: { scope: "all" } },
			});
			if (error) throw new Error(apiErrorMessage(error));
			if (!data) throw new Error("Run states unavailable");
			return data.runStates;
		},
	});
}

/** A retry reuses the original authority snapshot. Responses only trigger fresh reads. */
export function useOutcomeRunCommand(id: string) {
	const cache = useQueryClient();
	const intent = useRef<{ fingerprint: string; request: RunCommand } | undefined>(undefined);
	const mutation = useMutation({
		retry: false,
		mutationFn: async (request: RunCommand) => {
			const { error } = await apiClient.POST("/api/v1/outcomes/{outcomeId}/run", {
				params: { path: { outcomeId: id } },
				body: request,
			});
			if (error) throw new Error(apiErrorMessage(error));
		},
		onSuccess: () => {
			intent.current = undefined;
		},
		onSettled: async () => {
			await Promise.all(
				["outcome-run-state", "project-run-states", "outcome-schedule", "outcome-attempts"].map((root) =>
					cache.invalidateQueries({ queryKey: [root] }),
				),
			);
		},
	});
	return {
		...mutation,
		command: (action: RunCommand["action"], state: OutcomeRunState) => {
			const body = {
				action,
				planRevisionId: state.freshness.planRevisionId ?? "",
				expectedContractRevision: state.freshness.contractRevisionNumber,
				expectedGeneration: state.intent?.generation ?? 0,
			};
			const fingerprint = JSON.stringify({ id, ...body });
			if (intent.current?.fingerprint !== fingerprint)
				intent.current = { fingerprint, request: { ...body, requestKey: crypto.randomUUID() } };
			return mutation.mutateAsync(intent.current.request);
		},
	};
}

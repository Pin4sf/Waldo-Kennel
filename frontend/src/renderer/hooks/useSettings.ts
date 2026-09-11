/**
 * Daemon-owned user preferences.
 *
 * Read from and written to the daemon rather than held in the renderer, because
 * `kennel spawn`, mobile, and headless spawns resolve the same value. A preference
 * kept in one client would look correct in Settings and disagree with the others.
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient, apiErrorCode, apiErrorMessage } from "../lib/api-client";
import type { components } from "../../api/schema";
import type { SessionMode } from "../types/workspace";

export const settingsQueryKey = ["settings"] as const;

export interface Settings {
	/** Applies to sessions created from now on; never to an existing one. */
	defaultSessionMode: SessionMode;
	/** Agents that can run in chat mode today. Empty means chat is unavailable. */
	chatHarnesses: string[];
	reasoning: {
        mode?: string;
		provider: string;
		model: string;
		effort: string;
		configured: boolean;
		ready: boolean;
		keyConfigured: boolean;
		verified: boolean;
		verifiedAt?: string | null;
		errorCode?: string;
		error?: string;
	};
}

export function useSettings() {
	const query = useQuery({
		queryKey: settingsQueryKey,
		queryFn: async (): Promise<Settings> => {
			const { data, error } = await apiClient.GET("/api/v1/settings");
			if (error) throw error;
			return {
				defaultSessionMode: (data?.defaultSessionMode ?? "tui") as SessionMode,
				chatHarnesses: data?.chatHarnesses ?? [],
				reasoning: data?.reasoning ?? { provider: "", model: "", effort: "", configured: false, ready: false, keyConfigured: false, verified: false },
			};
		},
	});

	return {
		settings: query.data,
		isLoading: query.isLoading,
		error: query.error ? apiErrorMessage(query.error) : undefined,
	};
}

export function useVerifyReasoning() {
	const queryClient = useQueryClient();
	const mutation = useMutation({
		mutationFn: async () => {
			const { data, error } = await apiClient.POST("/api/v1/settings/reasoning/verification");
			if (error) throw error;
			return data;
		},
		onSuccess: () => queryClient.invalidateQueries({ queryKey: settingsQueryKey }),
	});

	return {
		verify: () => mutation.mutateAsync(),
		verifying: mutation.isPending,
		error: mutation.error ? apiErrorMessage(mutation.error) : undefined,
		errorCode: mutation.error ? apiErrorCode(mutation.error) : undefined,
	};
}

export function useUpdateReasoning() {
	const queryClient = useQueryClient();
	const mutation = useMutation({
		mutationFn: async (input: components["schemas"]["UpdateReasoningRequest"]) => {
			const { data, error } = await apiClient.PATCH("/api/v1/settings/reasoning", { body: input });
			if (error) throw error;
			return data;
		},
		onSuccess: () => queryClient.invalidateQueries({ queryKey: settingsQueryKey }),
	});

	return {
		update: (input: components["schemas"]["UpdateReasoningRequest"]) => mutation.mutateAsync(input),
		saving: mutation.isPending,
		error: mutation.error ? apiErrorMessage(mutation.error) : undefined,
	};
}

export function useUpdateSessionInterface() {
	const queryClient = useQueryClient();
	const mutation = useMutation({
		mutationFn: async (mode: SessionMode) => {
			const { data, error } = await apiClient.PATCH("/api/v1/settings/session-interface", {
				body: { defaultSessionMode: mode },
			});
			if (error) throw error;
			return data;
		},
		// Refetch rather than writing the value in locally: the daemon is the source
		// of truth, and the control must not claim a change it did not persist.
		onSuccess: () => queryClient.invalidateQueries({ queryKey: settingsQueryKey }),
	});

	return {
		update: (mode: SessionMode) => mutation.mutate(mode),
		saving: mutation.isPending,
		error: mutation.error ? apiErrorMessage(mutation.error) : undefined,
	};
}

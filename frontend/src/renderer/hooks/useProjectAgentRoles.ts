import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { components } from "../../api/schema";
import { apiClient, apiErrorMessage, hasTrustedApiBaseUrl } from "../lib/api-client";
import { usesPreviewWorkspaceData } from "../lib/preview-mode";
import { workspaceQueryKey } from "./useWorkspaceQuery";

type Project = components["schemas"]["Project"];
type ProjectConfig = components["schemas"]["ProjectConfig"];

export type AgentRole = "worker" | "orchestrator";

/**
 * The v0 locked worker fixture, mirrored from the daemon
 * (`service/outcome/attempt.go`: an empty attempt harness falls back to Codex).
 * Kept as one named constant so the renderer never spells a provider name into
 * a policy decision — it only names the daemon's own documented default so an
 * unset project reads as what will actually run, not as blank.
 */
export const DEFAULT_ROLE_AGENT = "codex";

export const projectQueryKey = (id: string) => ["project", id] as const;

/**
 * One project's durable worker/orchestrator agents — the same
 * `config.worker.agent` / `config.orchestrator.agent` fields Project Settings
 * owns, read and written through the same endpoint.
 *
 * Every writer here does a read-modify-write of the WHOLE `ProjectConfig`:
 * `PUT /api/v1/projects/{id}` replaces the stored config wholesale, so sending
 * a config with only the role field would silently drop the project's branch,
 * intake, reviewer, and env settings. The write is therefore gated on the read
 * having landed — `setRole` refuses rather than guessing at a config it has
 * not seen.
 */
export function useProjectAgentRoles(projectId: string | undefined) {
	const queryClient = useQueryClient();
	// Preview has no daemon project to configure, and an untrusted API base URL
	// means the daemon is not reachable yet; both render as "unavailable"
	// rather than as an empty selection the person could act on.
	const enabled = Boolean(projectId) && !usesPreviewWorkspaceData && hasTrustedApiBaseUrl();

	const query = useQuery({
		queryKey: projectQueryKey(projectId ?? ""),
		enabled,
		queryFn: async () => {
			const { data, error } = await apiClient.GET("/api/v1/projects/{id}", {
				params: { path: { id: projectId as string } },
			});
			if (error) throw new Error(apiErrorMessage(error));
			// A degraded project (unresolvable path) carries no config to edit.
			if (data?.status !== "ok") throw new Error("This project is degraded.");
			return data.project as Project;
		},
	});

	const project = query.data;
	const config: ProjectConfig = project?.config ?? {};

	const mutation = useMutation({
		mutationFn: async ({ role, agentId }: { role: AgentRole; agentId: string }) => {
			if (!project) throw new Error("This project's settings have not loaded yet.");
			const next: ProjectConfig = {
				...config,
				[role]: { ...(config[role] ?? {}), agent: agentId },
			};
			const { error } = await apiClient.PUT("/api/v1/projects/{id}", {
				params: { path: { id: projectId as string } },
				body: { displayName: project.name, config: next },
			});
			if (error) throw new Error(apiErrorMessage(error));
		},
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: projectQueryKey(projectId ?? "") });
			// The sidebar reads orchestratorAgent off the workspace list, so it
			// must not keep showing the previous agent after this write.
			await queryClient.invalidateQueries({ queryKey: workspaceQueryKey });
		},
	});

	return {
		available: enabled && query.isSuccess,
		loading: query.isLoading,
		error: query.error instanceof Error ? query.error.message : undefined,
		/** Unset reads as the daemon's own fallback, not as blank. */
		worker: config.worker?.agent ?? DEFAULT_ROLE_AGENT,
		orchestrator: config.orchestrator?.agent ?? DEFAULT_ROLE_AGENT,
		saving: mutation.isPending,
		saveError: mutation.error instanceof Error ? mutation.error.message : undefined,
		setRole: (role: AgentRole, agentId: string) => mutation.mutateAsync({ role, agentId }),
	};
}

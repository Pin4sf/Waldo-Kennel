import type { components } from "../../api/schema";

type CreateProjectConfigInput = {
	// Empty means the Project is registered now and provider setup is deferred.
	workerAgent: string;
	// Optional provider-scoped model preference. Empty means provider/Kennel
	// default and never manufactures a provider or execution lock.
	workerModel?: string;
	workerMode?: string;
	// Optional: present only after an explicit Advanced Settings choice. Empty
	// means no coordinator is configured; the worker is never reused implicitly.
	orchestratorAgent?: string;
	orchestratorModel?: string;
	orchestratorMode?: string;
	trackerIntake?: components["schemas"]["TrackerIntakeConfig"];
};

function projectRoleAgentConfig(
	model?: string,
	mode?: string,
): components["schemas"]["AgentConfig"] | undefined {
	const cleanModel = model?.trim() ?? "";
	const cleanMode = mode?.trim() ?? "";
	if (!cleanModel && !cleanMode) return undefined;
	return {
		...(cleanModel ? { model: cleanModel } : {}),
		...(cleanMode ? { mode: cleanMode } : {}),
	};
}

export function createProjectConfig(input: CreateProjectConfigInput): components["schemas"]["ProjectConfig"] {
	const workerAgentConfig = input.workerAgent
		? projectRoleAgentConfig(input.workerModel, input.workerMode)
		: undefined;
	const orchestratorAgentConfig = input.orchestratorAgent
		? projectRoleAgentConfig(input.orchestratorModel, input.orchestratorMode)
		: undefined;
	return {
		...(input.workerAgent
			? {
					worker: {
						agent: input.workerAgent as components["schemas"]["RoleOverride"]["agent"],
						...(workerAgentConfig ? { agentConfig: workerAgentConfig } : {}),
					},
					agentPreferences: { defaultWorker: input.workerAgent },
				}
			: {}),
		...(input.orchestratorAgent
			? {
					orchestrator: {
						agent: input.orchestratorAgent as components["schemas"]["RoleOverride"]["agent"],
						...(orchestratorAgentConfig ? { agentConfig: orchestratorAgentConfig } : {}),
					},
				}
			: {}),
		...(input.trackerIntake ? { trackerIntake: input.trackerIntake } : {}),
	};
}


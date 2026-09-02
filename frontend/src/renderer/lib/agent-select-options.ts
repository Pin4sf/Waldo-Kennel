import type { components } from "../../api/schema";

type AgentInfo = components["schemas"]["AgentInfo"];

export const CORE_PROVIDER_IDS = ["codex", "claude-code", "opencode", "cursor", "pi"] as const;
export const DEFAULT_AGENT_PRIORITY: readonly string[] = [];
export const DEFAULT_AGENT_PRIORITY_RANK = new Map<string, number>();

export type AgentStatusTone = "success" | "warning" | "muted";

export type RankedAgentOption = AgentInfo & {
	disabled: boolean;
	priorityRank: number;
	rank: number;
	status: string;
	statusTone: AgentStatusTone;
};

export function agentLabelCompare(a: AgentInfo, b: AgentInfo): number {
	return a.label.localeCompare(b.label) || a.id.localeCompare(b.id);
}

function agentStatus(
	installedAgent: AgentInfo | undefined,
	isAuthorized: boolean,
	isAuthUnknown: boolean,
): Pick<RankedAgentOption, "status" | "statusTone"> {
	if (!installedAgent) return { status: "Needs install", statusTone: "muted" };
	if (isAuthUnknown) return { status: "Auth unknown", statusTone: "warning" };
	if (!isAuthorized) return { status: "Needs auth", statusTone: "warning" };
	return { status: "Ready", statusTone: "success" };
}

// buildRankedAgentOptions is intentionally brand-neutral. Product support comes
// from the daemon catalog; local readiness decides whether an entry is enabled.
// The legacy priorityRank parameter remains in the call shape while callers are
// migrated, but it is ignored so the renderer cannot encode a hidden provider
// preference.
export function buildRankedAgentOptions({
	supported,
	installed,
	authorized,
	fallbackAgents,
	filter,
}: {
	supported?: AgentInfo[];
	installed?: AgentInfo[];
	authorized?: AgentInfo[];
	priorityRank?: Map<string, number>;
	fallbackAgents: AgentInfo[];
	filter?: (agent: AgentInfo) => boolean;
}): RankedAgentOption[] {
	const supportedAgents = (supported ?? fallbackAgents).filter((agent) => (filter ? filter(agent) : true));
	const installedAgents = installed ?? [];
	const authorizedAgents = authorized ?? [];
	const authorizedIds = new Set(authorizedAgents.map((agent) => agent.id));
	const installedById = new Map(installedAgents.map((agent) => [agent.id, agent]));

	return supportedAgents
		.map((agent) => {
			const installedAgent = installedById.get(agent.id);
			const authStatus = installedAgent?.authStatus;
			const isAuthorized = authorizedIds.has(agent.id) || authStatus === "authorized";
			const isAuthUnknown = Boolean(installedAgent) && !isAuthorized && authStatus !== "unauthorized";
			const isSelectable = Boolean(installedAgent) && (isAuthorized || isAuthUnknown);
			const rank = isAuthorized ? 0 : isAuthUnknown ? 1 : installedAgent ? 2 : 3;
			return {
				...agent,
				disabled: !isSelectable,
				priorityRank: Number.MAX_SAFE_INTEGER,
				rank,
				...agentStatus(installedAgent, isAuthorized, isAuthUnknown),
			};
		})
		.sort((a, b) => a.rank - b.rank || agentLabelCompare(a, b));
}

export function singleReadyProvider(options: RankedAgentOption[]): string {
	const ready = options.filter((option) => !option.disabled);
	return ready.length === 1 ? ready[0].id : "";
}

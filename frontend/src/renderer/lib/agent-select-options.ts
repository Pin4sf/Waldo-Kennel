import type { components } from "../../api/schema";

type AgentInfo = components["schemas"]["AgentInfo"];

export const CORE_PROVIDER_IDS = ["codex", "claude-code", "opencode", "cursor", "pi"] as const;

// Kept as compatibility exports while picker call sites migrate. Kennel does
// not encode a provider-brand preference in the renderer.
export const DEFAULT_AGENT_PRIORITY: readonly string[] = [];
export const DEFAULT_AGENT_PRIORITY_RANK = new Map<string, number>();

type AgentRole = keyof NonNullable<AgentInfo["roles"]>;

/**
 * Reads one role admission off a catalog entry, failing CLOSED.
 *
 * `roles` is required by the generated schema, but a daemon on a different
 * version — or a partial catalog entry — can omit it, and a renderer that
 * dereferences it directly takes the whole surface down with a TypeError
 * instead of degrading. An entry that does not state a role is not admitted
 * for it; the daemon refuses the spawn either way, so the honest client
 * behaviour is to hide the option rather than crash or assume yes.
 */
export function admitsRole(agent: AgentInfo, role: AgentRole): boolean {
	return agent.roles?.[role] === true;
}

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

function agentStatus({
	installedAgent,
	isAuthorized,
	isAuthUnknown,
	isProfileReady,
}: {
	installedAgent: AgentInfo | undefined;
	isAuthorized: boolean;
	isAuthUnknown: boolean;
	isProfileReady: boolean;
}): Pick<RankedAgentOption, "status" | "statusTone"> {
	if (!installedAgent) return { status: "Needs install", statusTone: "muted" };
	if (!isProfileReady) {
		return {
			status: installedAgent.readyDetail?.trim() || "Needs setup",
			statusTone: "warning",
		};
	}
	if (!isAuthorized && !isAuthUnknown) return { status: "Needs auth", statusTone: "warning" };
	if (isAuthUnknown) {
		return {
			status: installedAgent.readyDetail?.trim() || "Auth unknown",
			statusTone: "warning",
		};
	}
	// A healthy agent carries no badge. The menu exists to surface PROBLEMS —
	// needs install, needs setup, needs auth — and stamping "Ready" on every
	// other row turns the one signal that should stand out into noise. It also
	// changes each option's accessible name, so a screen reader announces
	// "Codex Ready" for the unremarkable case.
	return { status: "", statusTone: "success" };
}

// buildRankedAgentOptions renders daemon-owned provider facts. Build support,
// local installation, auth/profile readiness and role eligibility are distinct
// concepts; no provider id is special-cased here.
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
	const supportedAgents = (supported ?? fallbackAgents).filter((candidate) => (filter ? filter(candidate) : true));
	const installedAgents = installed ?? [];
	const authorizedAgents = authorized ?? [];
	const authorizedIds = new Set(authorizedAgents.map((candidate) => candidate.id));
	const installedById = new Map(installedAgents.map((candidate) => [candidate.id, candidate]));

	return supportedAgents
		.map((candidate) => {
			const installedAgent = installedById.get(candidate.id);
			const authStatus = installedAgent?.authStatus;
			const isAuthorized = authorizedIds.has(candidate.id) || authStatus === "authorized";
			const isAuthUnknown = Boolean(installedAgent) && !isAuthorized && authStatus !== "unauthorized";
			const isProfileReady = installedAgent?.ready !== false;
			const isSelectable = Boolean(installedAgent) && isProfileReady && (isAuthorized || isAuthUnknown);
			const rank = isSelectable ? 0 : installedAgent ? 1 : 2;
			return {
				...candidate,
				...installedAgent,
				disabled: !isSelectable,
				priorityRank: Number.MAX_SAFE_INTEGER,
				rank,
				...agentStatus({ installedAgent, isAuthorized, isAuthUnknown, isProfileReady }),
			};
		})
		.sort((a, b) => a.rank - b.rank || agentLabelCompare(a, b));
}

export function singleReadyProvider(options: RankedAgentOption[]): string {
	const ready = options.filter((option) => !option.disabled);
	return ready.length === 1 ? ready[0].id : "";
}

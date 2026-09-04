import { describe, expect, it } from "vitest";
import type { components } from "../../api/schema";
import { buildRankedAgentOptions, CORE_PROVIDER_IDS, singleReadyProvider } from "./agent-select-options";

type AgentInfo = components["schemas"]["AgentInfo"];

const roles = (overrides: Partial<AgentInfo["roles"]> = {}): AgentInfo["roles"] => ({
	worker: true,
	coordinator: false,
	switchTarget: false,
	...overrides,
});

const agent = (id: string, label: string, extra: Partial<AgentInfo> = {}): AgentInfo => ({
	id,
	label,
	authStatus: "authorized",
	roles: roles(),
	...extra,
});

describe("buildRankedAgentOptions", () => {
	it("keeps the product surface to the five Kennel providers without brand priority", () => {
		expect(CORE_PROVIDER_IDS).toEqual(["codex", "claude-code", "opencode", "cursor", "pi"]);
		const supported = [
			agent("pi", "Pi"),
			agent("codex", "Codex"),
			agent("cursor", "Cursor"),
			agent("opencode", "OpenCode"),
			agent("claude-code", "Claude Code"),
		];
		const options = buildRankedAgentOptions({ supported, installed: supported, authorized: supported, fallbackAgents: [] });
		expect(options.map((option) => option.label)).toEqual(["Claude Code", "Codex", "Cursor", "OpenCode", "Pi"]);
	});

	it("keeps unavailable providers visible and disables them with daemon readiness detail", () => {
		const supported = [
			agent("codex", "Codex"),
			agent("pi", "Pi", { ready: false, readyDetail: "Select a Pi model profile" }),
		];
		const options = buildRankedAgentOptions({ supported, installed: supported, authorized: supported, fallbackAgents: [] });
		const pi = options.find((option) => option.id === "pi");
		expect(pi?.disabled).toBe(true);
		expect(pi?.status).toBe("Select a Pi model profile");
	});

	it("filters by daemon role capability instead of provider id", () => {
		const supported = [
			agent("claude-code", "Claude Code", { roles: roles({ coordinator: true }) }),
			agent("cursor", "Cursor", { roles: roles({ coordinator: false }) }),
		];
		const options = buildRankedAgentOptions({
			supported,
			installed: supported,
			authorized: supported,
			fallbackAgents: [],
			filter: (candidate) => candidate.roles.coordinator,
		});
		expect(options.map((option) => option.id)).toEqual(["claude-code"]);
	});

	it("auto-selects only when exactly one provider is ready", () => {
		const supported = [agent("codex", "Codex"), agent("cursor", "Cursor")];
		const bothReady = buildRankedAgentOptions({ supported, installed: supported, authorized: supported, fallbackAgents: [] });
		expect(singleReadyProvider(bothReady)).toBe("");

		const oneReady = buildRankedAgentOptions({
			supported,
			installed: [supported[0]],
			authorized: [supported[0]],
			fallbackAgents: [],
		});
		expect(singleReadyProvider(oneReady)).toBe("codex");
	});
});

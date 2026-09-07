import { describe, expect, it } from "vitest";
import { buildAgentInventory } from "./agent-inventory-telemetry";
import type { AgentCatalog } from "../hooks/useAgentsQuery";
import type { components } from "../../api/schema";

type AgentInfo = components["schemas"]["AgentInfo"];

/** Catalog entries carry the daemon-required roles even though telemetry only reads ids. */
const agent = (id: string, label = ""): AgentInfo => ({
	id,
	label,
	roles: { worker: true, coordinator: true, switchTarget: true },
});

const catalog = (over: Partial<AgentCatalog> = {}): AgentCatalog =>
	({ installed: [], authorized: [], supported: [], ...over }) as AgentCatalog;

describe("agent inventory", () => {
	// The open question is whether people ever configure more than one agent.
	// kennel.session.spawned only shows which harness ran, so an install with six
	// authorized agents that always picks one looked identical to one that has
	// only that one.
	it("counts installed, authorized, and supported separately", () => {
		const inventory = buildAgentInventory(
			catalog({
				installed: [agent("claude-code", "Claude Code"), agent("codex", "Codex")],
				authorized: [agent("codex", "Codex")],
				supported: Array.from({ length: 23 }, (_, i) => agent(`a${i}`, `A${i}`)),
			}) as AgentCatalog,
		);
		expect(inventory.installed_count).toBe(2);
		expect(inventory.authorized_count).toBe(1);
		expect(inventory.supported_count).toBe(23);
	});

	// Sorted so the same set of agents always produces the same value and can be
	// grouped in analysis rather than fragmenting by ordering.
	it("reports authorized agent ids sorted and stable", () => {
		const a = buildAgentInventory(catalog({ authorized: [
			agent("codex"), agent("claude-code"), agent("cursor"),
		] }) as AgentCatalog);
		const b = buildAgentInventory(catalog({ authorized: [
			agent("cursor"), agent("codex"), agent("claude-code"),
		] }) as AgentCatalog);
		expect(a.authorized_agents).toBe("claude-code,codex,cursor");
		expect(a.authorized_agents).toBe(b.authorized_agents);
	});

	it("handles an empty or partial catalog without throwing", () => {
		const inventory = buildAgentInventory({} as AgentCatalog);
		expect(inventory).toEqual({
			installed_count: 0, authorized_count: 0, supported_count: 0, authorized_agents: "",
		});
	});

	// Bounded so a registry that grows cannot turn one property into a long string.
	it("bounds the joined id list", () => {
		const many = Array.from({ length: 80 }, (_, i) => agent(`agent-name-${i}`));
		const inventory = buildAgentInventory(catalog({ authorized: many }) as AgentCatalog);
		expect(inventory.authorized_agents.length).toBeLessThanOrEqual(200);
		expect(inventory.authorized_count).toBe(80);
	});

	it("ignores malformed ids rather than reporting empty entries", () => {
		const inventory = buildAgentInventory(catalog({ authorized: [
			agent(""), agent("codex"),
		] }) as AgentCatalog);
		expect(inventory.authorized_agents).toBe("codex");
		expect(inventory.authorized_count).toBe(1);
	});
});

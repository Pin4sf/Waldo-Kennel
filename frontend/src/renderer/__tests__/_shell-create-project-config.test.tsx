import { describe, expect, it } from "vitest";
import { createProjectConfig } from "../routes/_shell";

describe("createProjectConfig", () => {
	it("persists the selected default coding agent as the Mission-role worker preference", () => {
		expect(
			createProjectConfig({
				workerAgent: "codex",
				orchestratorAgent: "claude-code",
			}),
		).toEqual({
			worker: { agent: "codex" },
			orchestrator: { agent: "claude-code" },
			agentPreferences: { defaultWorker: "codex" },
		});
	});

	it("preserves tracker intake and the worker preference alongside selected agent defaults", () => {
		expect(
			createProjectConfig({
				workerAgent: "cursor",
				orchestratorAgent: "opencode",
				trackerIntake: { enabled: true, provider: "github", assignee: "octocat" },
			}),
		).toEqual({
			worker: { agent: "cursor" },
			orchestrator: { agent: "opencode" },
			agentPreferences: { defaultWorker: "cursor" },
			trackerIntake: { enabled: true, provider: "github", assignee: "octocat" },
		});
	});
});

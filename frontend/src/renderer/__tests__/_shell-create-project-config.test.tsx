import { describe, expect, it } from "vitest";
import { createProjectConfig } from "../routes/_shell";

describe("createProjectConfig", () => {
	it("keeps a provider-less Project provider-less", () => {
		expect(createProjectConfig({ workerAgent: "" })).toEqual({});
	});

	it("persists an explicit worker without manufacturing a coordinator", () => {
		expect(
			createProjectConfig({
				workerAgent: "opencode",
			}),
		).toEqual({
			worker: { agent: "opencode" },
			agentPreferences: { defaultWorker: "opencode" },
		});
	});

	it("persists worker and coordinator only when both were explicitly selected", () => {
		expect(
			createProjectConfig({
				workerAgent: "claude-code",
				orchestratorAgent: "opencode",
				trackerIntake: { enabled: true, provider: "github", assignee: "octocat" },
			}),
		).toEqual({
			worker: { agent: "claude-code" },
			orchestrator: { agent: "opencode" },
			agentPreferences: { defaultWorker: "claude-code" },
			trackerIntake: { enabled: true, provider: "github", assignee: "octocat" },
		});
	});
});

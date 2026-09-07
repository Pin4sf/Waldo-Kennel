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

	it("persists an explicit worker model as a preference on that role", () => {
		expect(
			createProjectConfig({
				workerAgent: "claude-code",
				workerModel: "claude-sonnet",
			}),
		).toEqual({
			worker: {
				agent: "claude-code",
				agentConfig: { model: "claude-sonnet" },
			},
			agentPreferences: { defaultWorker: "claude-code" },
		});
	});

	it("does not persist an orphan model when no worker provider was selected", () => {
		expect(
			createProjectConfig({
				workerAgent: "",
				workerModel: "claude-sonnet",
			}),
		).toEqual({});
	});

	it("persists worker and coordinator preferences independently", () => {
		expect(
			createProjectConfig({
				workerAgent: "claude-code",
				workerModel: "claude-sonnet",
				orchestratorAgent: "opencode",
				orchestratorMode: "balanced",
				trackerIntake: { enabled: true, provider: "github", assignee: "octocat" },
			}),
		).toEqual({
			worker: {
				agent: "claude-code",
				agentConfig: { model: "claude-sonnet" },
			},
			orchestrator: {
				agent: "opencode",
				agentConfig: { mode: "balanced" },
			},
			agentPreferences: { defaultWorker: "claude-code" },
			trackerIntake: { enabled: true, provider: "github", assignee: "octocat" },
		});
	});
});

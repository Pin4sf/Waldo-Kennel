import { describe, expect, it } from "vitest";
import { createProjectConfig } from "../routes/_shell";

describe("createProjectConfig", () => {
	it("omits the coordinator override on the default path so the daemon applies its canonical default", () => {
		expect(
			createProjectConfig({
				workerAgent: "codex",
			}),
		).toEqual({
			worker: { agent: "codex" },
			agentPreferences: { defaultWorker: "codex" },
		});
	});

	it("persists the worker preference and an explicit Advanced Settings coordinator override together", () => {
		expect(
			createProjectConfig({
				workerAgent: "codex",
				orchestratorAgent: "claude-code",
				trackerIntake: { enabled: true, provider: "github", assignee: "octocat" },
			}),
		).toEqual({
			worker: { agent: "codex" },
			orchestrator: { agent: "claude-code" },
			agentPreferences: { defaultWorker: "codex" },
			trackerIntake: { enabled: true, provider: "github", assignee: "octocat" },
		});
	});
});

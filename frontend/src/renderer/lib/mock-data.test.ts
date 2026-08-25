import { describe, expect, it } from "vitest";

import { mockSessionScmSummaries, mockShellTerminals, mockWorkspaces } from "./mock-data";

describe("preview workspace identity", () => {
	it("uses Kennel product naming throughout the public demo fixtures", () => {
		expect(mockWorkspaces[0]).toMatchObject({
			id: "kennel-design",
			name: "kennel-design",
			path: "/demo/kennel-design",
		});
		expect(mockShellTerminals[0]).toMatchObject({
			projectId: "kennel-design",
			title: "kennel-design",
			workingDir: "/Users/demo/Projects/kennel-design",
		});

		const publicFixtureText = JSON.stringify({ mockSessionScmSummaries, mockShellTerminals, mockWorkspaces });
		expect(publicFixtureText).not.toMatch(/ao-demo|Agent Orchestrator|AO preview/i);
	});
});

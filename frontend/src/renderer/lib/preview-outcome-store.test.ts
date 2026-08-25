import { beforeEach, describe, expect, it } from "vitest";

import {
	approvePreviewPlan,
	createPreviewOutcome,
	getPreviewOutcome,
	listPreviewOutcomes,
	proposePreviewPlan,
	resetPreviewOutcomeStore,
} from "./preview-outcome-store";

describe("preview outcome lifecycle", () => {
	beforeEach(() => resetPreviewOutcomeStore());

	it("demonstrates contract confirmation and plan authorization without claiming acceptance", () => {
		const outcome = createPreviewOutcome("kennel-design", {
			title: "Ship the Work tab demo",
			goal: "Make the approved Work experience demonstrable.",
			successCriteria: ["List and Board views show the same work."],
			review: "Owner walkthrough in Electron.",
			constraints: ["Use preview data only."],
			nonGoals: ["Do not claim backend acceptance."],
			clarification: "Use the current local calendar day.",
			requestKey: "demo-request-1",
		});

		expect(listPreviewOutcomes("kennel-design")).toEqual([outcome]);
		const proposed = proposePreviewPlan(outcome.id, 1);
		expect(proposed).toMatchObject({ outcomeId: outcome.id, status: "proposed" });
		expect(proposed.workUnits[0]?.title).toContain("Ship the Work tab demo");

		const approved = approvePreviewPlan(outcome.id, proposed.id, 1);
		expect(approved.status).toBe("approved");
		expect(getPreviewOutcome(outcome.id)?.latestPlan?.status).toBe("approved");
		expect(JSON.stringify(getPreviewOutcome(outcome.id))).not.toMatch(/accepted|complete|closed/i);
	});
});

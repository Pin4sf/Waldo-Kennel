import { beforeEach, describe, expect, it } from "vitest";

import {
	approvePreviewPlan,
	createPreviewOutcome,
	getPreviewOutcome,
	getPreviewPlan,
	listPreviewOutcomes,
	proposePreviewPlan,
	resetPreviewOutcomeStore,
	revisePreviewOutcome,
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

	it("requires a fresh plan after the contract is revised", () => {
		const outcome = createPreviewOutcome("kennel-design", {
			title: "Ship the Work tab demo",
			goal: "Make the approved Work experience demonstrable.",
			successCriteria: ["List and Board views show the same work."],
			review: "Owner walkthrough in Electron.",
			requestKey: "demo-request-revise",
		});
		const plan = proposePreviewPlan(outcome.id, 1);
		approvePreviewPlan(outcome.id, plan.id, 1);

		const revised = revisePreviewOutcome(outcome.id, {
			expectedRevision: 1,
			goal: "Make the revised Work experience demonstrable.",
			successCriteria: ["List and Board views show the same revised work."],
			review: "Owner walkthrough in Electron.",
		});

		expect(revised.latestPlan).toBeUndefined();
		expect(getPreviewPlan(outcome.id)).toBeUndefined();
		expect(proposePreviewPlan(outcome.id, 2)).toMatchObject({
			contractRevisionNumber: 2,
			number: 2,
		});
	});
});

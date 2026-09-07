import { describe, expect, it } from "vitest";

import { deriveOutcomeDashboardPresentation } from "./outcome-dashboard-presentation";

describe("deriveOutcomeDashboardPresentation", () => {
	it("re-enters an Outcome without a plan at Decide & Authorize", () => {
		expect(deriveOutcomeDashboardPresentation({})).toEqual({
			nextActionKey: "outcome.dashboard.reviewPlan",
			stageKey: "outcome.stage.decideAuthorize",
			stateKey: "outcome.dashboard.contractSaved",
		});
	});

	it("re-enters a proposed plan at its authority decision", () => {
		expect(deriveOutcomeDashboardPresentation({
			currentRevisionNumber: 1,
			latestPlan: { contractRevisionNumber: 1, status: "proposed" },
		})).toEqual({
			nextActionKey: "outcome.dashboard.reviewAuthorization",
			stageKey: "outcome.stage.decideAuthorize",
			stateKey: "outcome.dashboard.planProposed",
		});
	});

	it("keeps an approved Plan inspectable through its exact Outcome", () => {
		expect(deriveOutcomeDashboardPresentation({
			currentRevisionNumber: 1,
			latestPlan: { contractRevisionNumber: 1, status: "approved" },
		})).toEqual({
			nextActionKey: "outcome.dashboard.reviewApprovedPlan",
			stageKey: "outcome.dashboard.authorizedStage",
			stateKey: "outcome.dashboard.executionNotConnected",
		});
	});

	it("does not treat a plan bound to an older contract as current authority", () => {
		expect(deriveOutcomeDashboardPresentation({
			currentRevisionNumber: 2,
			latestPlan: { contractRevisionNumber: 1, status: "approved" },
		}).stateKey).toBe("outcome.dashboard.contractSaved");
	});
});

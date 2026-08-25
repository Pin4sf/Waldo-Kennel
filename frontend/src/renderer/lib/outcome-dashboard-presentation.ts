type OutcomeDashboardFacts = {
	currentRevisionNumber?: number;
	latestPlan?: { contractRevisionNumber?: number; status: string };
};

export type OutcomeDashboardPresentation = {
	stageKey: "outcome.stage.decideAuthorize" | "outcome.dashboard.authorizedStage";
	stateKey:
		| "outcome.dashboard.contractSaved"
		| "outcome.dashboard.planProposed"
		| "outcome.dashboard.executionNotConnected";
	nextActionKey:
		| "outcome.dashboard.reviewPlan"
		| "outcome.dashboard.reviewAuthorization"
		| "outcome.dashboard.reviewApprovedPlan";
};

/**
 * Derive dashboard re-entry from canonical Outcome/Plan facts only.
 *
 * Act & Observe is not yet a dedicated route on beta. An approved Plan remains
 * inspectable through its exact Outcome identity; project sessions stay an
 * adjacent projection until Attempt linkage can join them honestly.
 */
export function deriveOutcomeDashboardPresentation(
	outcome: OutcomeDashboardFacts,
): OutcomeDashboardPresentation {
	const planBindsCurrentContract =
		outcome.latestPlan !== undefined &&
		outcome.latestPlan.contractRevisionNumber === outcome.currentRevisionNumber;
	if (outcome.latestPlan?.status === "approved" && planBindsCurrentContract) {
		return {
			stageKey: "outcome.dashboard.authorizedStage",
			stateKey: "outcome.dashboard.executionNotConnected",
			nextActionKey: "outcome.dashboard.reviewApprovedPlan",
		};
	}
	if (outcome.latestPlan?.status === "proposed" && planBindsCurrentContract) {
		return {
			stageKey: "outcome.stage.decideAuthorize",
			stateKey: "outcome.dashboard.planProposed",
			nextActionKey: "outcome.dashboard.reviewAuthorization",
		};
	}
	return {
		stageKey: "outcome.stage.decideAuthorize",
		stateKey: "outcome.dashboard.contractSaved",
		nextActionKey: "outcome.dashboard.reviewPlan",
	};
}

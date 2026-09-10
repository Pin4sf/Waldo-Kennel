type OutcomeDashboardFacts = {
	currentRevisionNumber?: number;
	latestPlan?: { contractRevisionNumber?: number; status: string };
};

export type OutcomeDashboardPresentation = {
	stageKey: "outcome.stage.decideAuthorize" | "outcome.dashboard.authorizedStage";
	stateKey:
		| "outcome.dashboard.contractSaved"
		| "outcome.dashboard.planProposed"
		| "mission.authorizationRecorded";
	nextActionKey:
		| "outcome.dashboard.reviewPlan"
		| "outcome.dashboard.reviewAuthorization"
		| "outcome.dashboard.reviewApprovedPlan";
};

/** Portfolio labels use recorded Contract and Plan authority, never inferred execution. */
export function deriveOutcomeDashboardPresentation(
	outcome: OutcomeDashboardFacts,
): OutcomeDashboardPresentation {
	const planBindsCurrentContract =
		outcome.latestPlan !== undefined &&
		outcome.latestPlan.contractRevisionNumber === outcome.currentRevisionNumber;
	if (outcome.latestPlan?.status === "approved" && planBindsCurrentContract) {
		return {
			stageKey: "outcome.dashboard.authorizedStage",
			stateKey: "mission.authorizationRecorded",
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

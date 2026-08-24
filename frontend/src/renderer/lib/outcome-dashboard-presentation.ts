type OutcomeDashboardFacts = {
	currentRevisionNumber?: number;
	latestPlan?: { contractRevisionNumber?: number; status: string };
};

export type OutcomeDashboardPresentation = {
	destination: "outcome" | "project";
	stageKey: "outcome.stage.decideAuthorize" | "outcome.dashboard.authorizedStage";
	stateKey:
		| "outcome.dashboard.contractSaved"
		| "outcome.dashboard.planProposed"
		| "outcome.dashboard.executionNotConnected";
	nextActionKey:
		| "outcome.dashboard.reviewPlan"
		| "outcome.dashboard.reviewAuthorization"
		| "outcome.dashboard.openProjectSessions";
};

/**
 * Derive dashboard re-entry from canonical Outcome/Plan facts only.
 *
 * Act & Observe is not yet a dedicated route on beta. An approved plan opens
 * the project work projection, where its daemon-backed sessions are visible,
 * instead of pretending it still awaits authorization.
 */
export function deriveOutcomeDashboardPresentation(
	outcome: OutcomeDashboardFacts,
): OutcomeDashboardPresentation {
	const planBindsCurrentContract =
		outcome.latestPlan !== undefined &&
		outcome.latestPlan.contractRevisionNumber === outcome.currentRevisionNumber;
	if (outcome.latestPlan?.status === "approved" && planBindsCurrentContract) {
		return {
			destination: "project",
			stageKey: "outcome.dashboard.authorizedStage",
			stateKey: "outcome.dashboard.executionNotConnected",
			nextActionKey: "outcome.dashboard.openProjectSessions",
		};
	}
	if (outcome.latestPlan?.status === "proposed" && planBindsCurrentContract) {
		return {
			destination: "outcome",
			stageKey: "outcome.stage.decideAuthorize",
			stateKey: "outcome.dashboard.planProposed",
			nextActionKey: "outcome.dashboard.reviewAuthorization",
		};
	}
	return {
		destination: "outcome",
		stageKey: "outcome.stage.decideAuthorize",
		stateKey: "outcome.dashboard.contractSaved",
		nextActionKey: "outcome.dashboard.reviewPlan",
	};
}

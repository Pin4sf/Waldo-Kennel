import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { OutcomeDecideAuthorizeSurface } from "../components/outcome/OutcomeDecideAuthorizeSurface";
import { OutcomeLifecycleShell } from "../components/outcome/OutcomeLifecycleShell";
import { OutcomeUnderstandSurface } from "../components/outcome/OutcomeUnderstandSurface";
import { WorkEnterSurface } from "../components/outcome/WorkEnterSurface";

type WorkSearch = {
	/** Selected project. Absent renders the Enter surface (stage: enter). */
	project?: string;
	/** Lifecycle stage within the Work destination. Defaults to understand. */
	stage?: "understand" | "decide_authorize";
	/** The Outcome a saved contract produced; required from decide onward. */
	outcome?: string;
};

function validateSearch(search: Record<string, unknown>): WorkSearch {
	const stage = search.stage === "decide_authorize" ? search.stage : undefined;
	return {
		project: typeof search.project === "string" && search.project !== "" ? search.project : undefined,
		stage,
		outcome: typeof search.outcome === "string" && search.outcome !== "" ? search.outcome : undefined,
	};
}

// Work is a destination, not a lifecycle stage. Enter, Understand, and
// Decide & Authorize share this one route so the spine stays one surface set;
// context rides in search params so refreshes keep it, and Home/Work mode
// memory (pathname-only) keeps working.
export const Route = createFileRoute("/_shell/work")({
	validateSearch,
	component: WorkRoute,
});

function WorkRoute() {
	const { project, stage, outcome } = Route.useSearch();
	const navigate = useNavigate();

	if (!project) {
		return <WorkEnterSurface />;
	}

	if (stage === "decide_authorize") {
		if (!outcome) {
			// A deep link without its Outcome falls back to Understand.
			return (
				<OutcomeLifecycleShell projectId={project} stage="understand">
					<OutcomeUnderstandSurface projectId={project} />
				</OutcomeLifecycleShell>
			);
		}
		return (
			<OutcomeLifecycleShell outcomeId={outcome} projectId={project} stage="decide_authorize">
				<OutcomeDecideAuthorizeSurface outcomeId={outcome} />
			</OutcomeLifecycleShell>
		);
	}

	return (
		<OutcomeLifecycleShell projectId={project} stage="understand">
			<OutcomeUnderstandSurface
				onContractSaved={(saved) => {
					void navigate({ to: "/work", search: { project, stage: "decide_authorize", outcome: saved.id } });
				}}
				projectId={project}
			/>
		</OutcomeLifecycleShell>
	);
}

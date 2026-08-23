import { createFileRoute } from "@tanstack/react-router";

import { OutcomeLifecycleShell } from "../components/outcome/OutcomeLifecycleShell";
import { OutcomeUnderstandSurface } from "../components/outcome/OutcomeUnderstandSurface";
import { WorkEnterSurface } from "../components/outcome/WorkEnterSurface";

type WorkSearch = {
	/** Selected project. Absent renders the Enter surface (stage: enter). */
	project?: string;
};

function validateSearch(search: Record<string, unknown>): WorkSearch {
	return {
		project: typeof search.project === "string" && search.project !== "" ? search.project : undefined,
	};
}

// Work is a destination, not a lifecycle stage. Enter and Understand share this
// one route so the spine stays one surface set rather than sibling pages, and
// the project rides in the search param so a refresh keeps its context.
// Home/Work mode memory tracks the pathname only and is unaffected.
export const Route = createFileRoute("/_shell/work")({
	validateSearch,
	component: WorkRoute,
});

function WorkRoute() {
	const { project } = Route.useSearch();

	if (!project) {
		return <WorkEnterSurface />;
	}

	return (
		<OutcomeLifecycleShell projectId={project} stage="understand">
			<OutcomeUnderstandSurface projectId={project} />
		</OutcomeLifecycleShell>
	);
}

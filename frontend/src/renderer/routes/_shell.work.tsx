import { createFileRoute } from "@tanstack/react-router";

import { WorkEnterSurface } from "../components/outcome/WorkEnterSurface";

// Work is a destination, not a lifecycle stage. It currently opens on Enter;
// later stages mount inside the same OutcomeLifecycleShell rather than adding
// sibling routes, so the spine stays one surface set instead of five pages.
export const Route = createFileRoute("/_shell/work")({
	component: WorkRoute,
});

function WorkRoute() {
	return <WorkEnterSurface />;
}

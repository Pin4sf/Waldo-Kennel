import { usesWorkLaunchMode } from "../lib/preview-mode";
import { createFileRoute, redirect } from "@tanstack/react-router";
import { SessionsBoard } from "../components/SessionsBoard";

export function enterProjectOutcomes({ params }: { params: { projectId: string } }) {
	if (usesWorkLaunchMode)
		throw redirect({
			to: "/work",
			search: { view: "outcomes", portfolio: params.projectId, project: params.projectId },
			replace: true,
		});
}

export const Route = createFileRoute("/_shell/projects/$projectId")({
	beforeLoad: enterProjectOutcomes,
	component: ProjectBoardRoute,
});

function ProjectBoardRoute() {
	const { projectId } = Route.useParams();
	return <SessionsBoard projectId={projectId} />;
}
